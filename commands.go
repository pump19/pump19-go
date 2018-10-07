package main

import (
	"fmt"
	"log"
	"strings"

	irc "github.com/fluffle/goirc/client"
)

type CommandHandler struct {
	triggers []string
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

	// if available, log the display name
	name, found := line.Tags["display-name"]
	if !found {
		name = line.Nick
	}

	logline := fmt.Sprintf("Got command '%v' from %v in %v",
		command, name, line.Target())

	log.Println(logline)
}
