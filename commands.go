package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	irc "github.com/fluffle/goirc/client"
)

const BINGO_URL = "https://pump19.eu/bingo"
const CODEFALL_URL = "https://pump19.eu/codefall"
const HELP_URL = "https://pump19.eu/commands"

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

	go cmd.handler(context, parts[1:])

	return true
}

type CommandHandler struct {
	triggers  []string
	codefall  *Codefall
	ircClient *irc.Conn

	commands []Command
	channels []string
}

func newCommandHandler(triggers []string, dsn string, ircClient *irc.Conn) *CommandHandler {
	codefall := newCodefall(dsn)
	if codefall == nil {
		log.Println("Failed to setup codefall handler")
		return nil
	}

	ch := &CommandHandler{
		triggers:  triggers,
		codefall:  codefall,
		ircClient: ircClient,
	}

	go codefall.listen(ch.announceCodefall)

	ch.commands = append(ch.commands, Command{
		regexp.MustCompile(`^(?:codefall)(?: ([123]))?$`),
		ch.handleCodefall})

	ch.commands = append(ch.commands, Command{
		regexp.MustCompile(`^(?:mult(?:i(?:pl(?:y|es?))?)?) (?:\$)?([0-9]+(?:\.[0-9]{1,2})?)$`),
		ch.handleMultiples})

	ch.commands = append(ch.commands, Command{
		regexp.MustCompile(`^(?:help)$`),
		ch.handleHelp})

	ch.commands = append(ch.commands, Command{
		regexp.MustCompile(`^(?:bingo)$`),
		ch.handleBingo})

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

func (ch *CommandHandler) announceCodefall(code Code) {
	if len(ch.channels) == 0 || !ch.ircClient.Connected() {
		return
	}

	log.Println("Announcing codefall on", ch.channels)
	codeMsg := fmt.Sprintf("Codefall | %v (%v) %v/%v", code.description, code.codeType, CODEFALL_URL, code.key)

	for _, channel := range ch.channels {
		go ch.ircClient.Privmsg(channel, codeMsg)
	}
}

func (ch *CommandHandler) handleCodefall(context IrcData, args []string) {
	limit, err := strconv.Atoi(args[0])
	if err != nil || limit <= 0 || limit > 3 {
		limit = 1
	}

	userName := context.source
	codes := ch.codefall.getRandomEntries(userName, limit)

	if len(codes) <= 0 {
		noCodesMsg := fmt.Sprintf("Could not find any unclaimed codes. Visit %v to add new entries.", CODEFALL_URL)
		context.conn.Privmsg(context.target, noCodesMsg)
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

func (ch *CommandHandler) handleHelp(context IrcData, args []string) {
	helpMsg := fmt.Sprintf("Pump19 is run by Twisted Pear. Check %v for a list of supported commands.", HELP_URL)
	context.conn.Privmsg(context.target, helpMsg)
}

func (ch *CommandHandler) handleBingo(context IrcData, args []string) {
	helpMsg := fmt.Sprintf("Check out %v for our interactive Trope Bingo cards.", BINGO_URL)
	context.conn.Privmsg(context.target, helpMsg)
}

// HandleCommand checks PRIVMSG for commands and dispatches them if found.
// It accepts only commands on joined channels.
func (ch *CommandHandler) handleCommand(conn *irc.Conn, line *irc.Line) {
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

		logLine := fmt.Sprintf("Got command '%v' from %v in %v",
			command, name, channel)

		log.Println(logLine)
	}
}

// JoinedChannel keeps track of JOIN messages (i.e. joined channels).
// These are used for announcing certain events on all known channels.
func (ch *CommandHandler) joinedChannel(conn *irc.Conn, line *irc.Line) {
	channel := line.Target()
	log.Println("Joined channel", channel)

	// conn.Privmsg(channel, "I Am Just Clay, And I Listen")

	ch.channels = append(ch.channels, channel)
}
