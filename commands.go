package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	irc "github.com/fluffle/goirc/client"
	_ "github.com/lib/pq"
)

const CODEFALL_URL = "https://pump19.eu/codefall"

type IrcData struct {
	conn   *irc.Conn
	source string
	target string
}

type Command struct {
	regex   *regexp.Regexp
	handler func(context IrcData, args []string)
}

// Run checks a raw command against the stored regex.
// If there is a match, the command arguments are dispatched to the registered handler.
func (cmd *Command) Run(context IrcData, raw string) bool {
	parts := cmd.regex.FindStringSubmatch(raw)
	if len(parts) <= 0 {
		return false
	}

	cmd.handler(context, parts[1:])
	return true
}

type CommandHandler struct {
	triggers []string
	database *sql.DB

	commands []Command
}

func NewCommandHandler(triggers []string, dsn string) *CommandHandler {
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalln("Cannot open database", err)
	} else if err = database.Ping(); err != nil {
		log.Fatalln("Cannot open database", err)
	}

	ch := &CommandHandler{
		triggers: triggers,
		database: database,
	}

	ch.commands = append(ch.commands, Command{
		regexp.MustCompile(`^(?:codefall)(?: ([123]))?$`),
		ch.handleCodefall})

	ch.commands = append(ch.commands, Command{
		regexp.MustCompile(`^(?:mult(?:i(?:pl(?:y|es?)?)?)?️) (?:\$)?([0-9]+(?:\.[0-9]{1,2})?)$`),
		ch.handleMultiples})

	return ch
}

// checkTrigger looks for one of several prefixes in the provided string.
// If there is a match, the prefix is removed and the command is returned.
func (ch *CommandHandler) checkTrigger(raw string) (bool, string) {
	for _, trigger := range ch.triggers {
		if strings.HasPrefix(raw, trigger) {
			return true, strings.TrimPrefix(raw, trigger)
		}
	}

	// no trigger found, this isn't for us
	return false, ""
}

func (ch *CommandHandler) handleCodefall(context IrcData, args []string) {
	limit, err := strconv.Atoi(args[0])
	if err != nil || limit <= 0 || limit > 3 {
		limit = 1
	}

	rows, err := ch.database.Query(`
		SELECT description, code_type, key
			FROM codefall_unclaimed
			WHERE user_name = $1
			ORDER BY random()
			LIMIT $2`, context.source, limit)

	type Code struct {
		description string
		codeType    string
		key         string
	}

	var codes []Code

	if err != nil {
		log.Println("Could not query unclaimed codes", err)
	} else {
		for rows.Next() {
			var code Code

			err = rows.Scan(&code.description, &code.codeType, &code.key)

			if err != nil {
				log.Println("Failed to parse result line")
				continue
			}

			codes = append(codes, code)
		}
	}

	if len(codes) <= 0 {
		noCodesMsg := fmt.Sprintf("Could not find any unclaimed codes. Visit %v to add new entries.", CODEFALL_URL)

		context.conn.Privmsg(context.target, noCodesMsg)
		return
	}

	var b strings.Builder
	b.WriteString("Codefall")
	for _, code := range codes {
		codeStr := fmt.Sprintf(" | %v (%v) %v/%v", code.description, code.codeType, CODEFALL_URL, code.key)
		b.WriteString(codeStr)
	}

	codesMsg := b.String()
	context.conn.Privmsg(context.target, codesMsg)
}

func (ch *CommandHandler) handleMultiples(context IrcData, args []string) {
	value, err := strconv.ParseFloat(args[0], 64)
	if err != nil || value > 1000 {
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Multiples of $%.2f", value))

	for _, multiplier := range []float64{2, 3, 4, 5, 6, 7, 8, 9, 10, 25, 50, 100} {
		b.WriteString(fmt.Sprintf(" | $%.2f", value*multiplier))
	}

	multMsg := b.String()
	context.conn.Privmsg(context.target, multMsg)
}

// HandleCommand checks PRIVMSG for commands and dispatches them if found.
// It accepts only commands on joined channels.
func (ch *CommandHandler) HandleCommand(conn *irc.Conn, line *irc.Line) {
	// we only care for public commands
	if !line.Public() {
		return
	}

	isCommand, command := ch.checkTrigger(line.Text())
	if !isCommand {
		return
	}

	nick := line.Nick

	// if available, log the display name
	name, found := line.Tags["display-name"]
	if !found {
		name = nick
	}

	channel := line.Target()

	context := IrcData{conn, nick, channel}
	for _, handler := range ch.commands {
		handled := handler.Run(context, command)

		if !handled {
			continue
		}

		logline := fmt.Sprintf("Got command '%v' from %v in %v",
			command, name, channel)

		log.Println(logline)
	}
}
