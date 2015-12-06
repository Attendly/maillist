package maillist

import (
	"html/template"
	"log"

	"github.com/sendgrid/sendgrid-go"
)

type Session struct {
	database
	config    Config
	messages  chan string
	templates *template.Template
	sgClient  *sendgrid.SGClient
}

type Config struct {
	DatabaseAddress string
	SendGridAPIKey  string
	JustPrint       bool
}

func OpenSession(config *Config) (*Session, error) {
	var s Session
	var err error

	s.database, err = openDatabase(config.DatabaseAddress)
	if err != nil {
		return nil, err
	}

	s.config = *config

	s.addTable(Account{}, "account")
	s.addTable(List{}, "list")
	s.addTable(Campaign{}, "campaign")
	s.addTable(Subscriber{}, "subscriber")
	s.addTable(Unsubscribe{}, "unsubscribe")
	s.addTable(Message{}, "message")

	err = s.dbmap.CreateTablesIfNotExists()

	s.sgClient = sendgrid.NewSendGridClientWithApiKey(s.config.SendGridAPIKey)

	s.messages = make(chan string)
	go service(&s)
	s.messages <- "wake"

	return &s, err
}

func (s *Session) Close() error {
	s.messages <- "close"
	return s.db.Close()
}

func service(s *Session) {
next:
	message := <-s.messages
	switch message {
	case "wake":
		for {
			m, err := pendingMessage(s)
			if err != nil {
				log.Printf("error: %v\n", err)
				goto next
			}
			if m == nil {
				goto next
			}

			if err = s.sendMessage(m); err != nil {
				log.Print(err)
				goto next
			}
		}
	case "close":
		return
	default:
		log.Printf("Message not understood: %s", message)
		goto next
	}
}
