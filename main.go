package main

import (
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile)

	log.Println("Loading configuration...")
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalln("Failed to load config:", err)
	}

	log.Println("Building go-lem...")
	golem := FromConfig(cfg)
	if golem == nil {
		log.Fatalln("Failed to build go-lem")
	}

	log.Println("Starting go-lem...")
	golem.Run()
}
