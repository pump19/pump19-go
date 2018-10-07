package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type IrcConfig struct {
	host string
	port uint16

	nick string
	pass string

	channels []string
}

type CmdConfig struct {
	triggers []string
}

type Config struct {
	irc IrcConfig
	cmd CmdConfig
}

type LoadError struct {
	variable string
	what     string
}

func (e *LoadError) Error() string {
	return fmt.Sprintf("Environment variable %s %s",
		e.variable, e.what)
}

// LoadConfig assembles the configuration used by the pump19 irc go-lem.
// It retrieves that information from environment variables.
// It returns a fully assembled Config or an error indicated a missing setting.
func LoadConfig() (*Config, error) {
	cfg := Config{}

	// IrcConfig {{{
	if hostStr, exists := os.LookupEnv("PUMP19_IRC_HOSTNAME"); !exists {
		return nil, &LoadError{"PUMP19_IRC_HOSTNAME", "is not set"}
	} else {
		cfg.irc.host = hostStr
	}

	if portStr, exists := os.LookupEnv("PUMP19_IRC_PORT"); !exists {
		return nil, &LoadError{"PUMP19_IRC_PORT", "is not set"}
	} else if port, err := strconv.ParseUint(portStr, 10, 16); err != nil {
		return nil, &LoadError{"PUMP19_IRC_PORT", "cannot be parsed"}
	} else {
		cfg.irc.port = uint16(port)
	}

	if nickStr, exists := os.LookupEnv("PUMP19_IRC_NICKNAME"); !exists {
		return nil, &LoadError{"PUMP19_IRC_NICKNAME", "is not set"}
	} else {
		cfg.irc.nick = nickStr
	}

	if passStr, exists := os.LookupEnv("PUMP19_IRC_PASSWORD"); !exists {
		return nil, &LoadError{"PUMP19_IRC_PASSWORD", "is not set"}
	} else {
		cfg.irc.pass = passStr
	}

	if chanStr, exists := os.LookupEnv("PUMP19_IRC_CHANNELS"); !exists {
		return nil, &LoadError{"PUMP19_IRC_CHANNELS", "is not set"}
	} else {
		cfg.irc.channels = strings.Split(chanStr, ",")
	}
	// IrcConfig }}}

	// CmdConfig {{{
	if triggerStr, exists := os.LookupEnv("PUMP19_CMD_TRIGGER"); !exists {
		return nil, &LoadError{"PUMP19_CMD_TRIGGER", "is not set"}
	} else {
		cfg.cmd.triggers = strings.Split(triggerStr, "")
	}
	// CmdConfig }}}

	return &cfg, nil
}
