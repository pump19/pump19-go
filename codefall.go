package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/lib/pq"
)

type Codefall struct {
	database *sql.DB
	listener *pq.Listener
}

type Code struct {
	description string
	codeType    string
	key         string
}

func newCodefall(dsn string) *Codefall {
	database, err := sql.Open("postgres", dsn)

	if err != nil {
		log.Fatalln("Cannot open database", err)
	} else if err = database.Ping(); err != nil {
		log.Fatalln("Cannot open database", err)
	}

	// we need a second connection in order to use the pubsub mechanism
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Println("Detected error on codefall listener", err)
		}
	}

	listener := pq.NewListener(dsn, time.Second, time.Minute, reportProblem)
	if listener == nil {
		log.Println("Cannot open listener:", err)
	} else if err = listener.Listen("codefall"); err != nil {
		log.Println("Cannot listen on codefall channel", err)
		listener.Close()
	}

	return &Codefall{database, listener}
}

func (cf *Codefall) listen(callback func(Code)) {
	listener := cf.listener

	for {
		select {
		case notification := <-listener.Notify:
			if notification.Channel != "codefall" || len(notification.Extra) == 0 {
				continue
			}

			log.Printf("Got notification on %v: %v\n", notification.Channel, notification.Extra)
			secret := notification.Extra

			go func() {
				if code := cf.getSpecificEntry(secret); code != nil {
					callback(*code)
				}
			}()

		case <-time.After(90 * time.Second):
			go listener.Ping()
		}
	}
}

func (cf *Codefall) getSpecificEntry(secret string) *Code {
	row := cf.database.QueryRow(`
		SELECT description, code_type, key
			FROM codefall_unclaimed
			WHERE key = $1
			LIMIT 1`, secret)

	var code Code
	if err := row.Scan(&code.description, &code.codeType, &code.key); err != nil {
		log.Println("Could not query unclaimed code for secret", secret, ":", err)
		return nil
	}

	return &code
}

func (cf *Codefall) getRandomEntries(userName string, limit int) []Code {
	rows, err := cf.database.Query(`
		SELECT description, code_type, key
			FROM codefall_unclaimed
			WHERE user_name = $1
			ORDER BY random()
			LIMIT $2`, userName, limit)

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

	return codes
}
