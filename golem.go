package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	irc "github.com/fluffle/goirc/client"
)

type Golem struct {
	conn *irc.Conn
}

// FromConfig creates a new Golem instance from the provided Config.
// A client connection will be configured with the provided server, user and channel information.
// A command handler is registered for dispatching commands received as PRIVMSGs.
func FromConfig(cfg *Config) *Golem {
	ircCfg := irc.NewConfig(cfg.irc.nick)
	ircCfg.SSL = true
	ircCfg.SSLConfig = &tls.Config{ServerName: cfg.irc.host}
	ircCfg.Server = fmt.Sprintf("%s:%v",
		cfg.irc.host, cfg.irc.port)
	ircCfg.Pass = cfg.irc.pass

	client := irc.Client(ircCfg)
	client.HandleFunc(irc.CONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			log.Println("Connection established")

			log.Println("Requesting capabilities...")
			conn.Cap("REQ", "twitch.tv/tags")

			log.Println("Joining channels...")
			for _, channel := range cfg.irc.channels {
				conn.Join(channel)
			}
		})

	commandHandler := CommandHandler{cfg.cmd.triggers}
	client.HandleFunc(irc.PRIVMSG, commandHandler.HandleCommand)

	return &Golem{conn: client}
}

// Run a fully configured IRC golem.
// The method blocks as long as the IRC connection remains established.
// Both SIGINT and SIGTERM are handled and will terminate the IRC connection.
func (g *Golem) Run() {
	quit := make(chan bool, 1)

	g.conn.HandleFunc(irc.DISCONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			log.Println("Connection closed. Exiting...")
			quit <- true
		})

	if err := g.conn.Connect(); err != nil {
		log.Fatalln("Connection error:", err.Error())
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigChan:
			log.Println("Got signal:", sig)

			// if we're not connected for some reason, quit immediately
			if g.conn.Connected() {
				log.Println("Closing connection...")
				g.conn.Quit()
			} else {
				log.Println("Exiting...")
				quit <- true
			}

		case <-quit:
			return
		}
	}
}
