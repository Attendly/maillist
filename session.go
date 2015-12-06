package maillist

import (
	"html/template"
	"log"

	"github.com/sendgrid/sendgrid-go"
)

// Session is an opaque type holding database connections and other
// implementation details
type Session struct {
	database
	config    Config
	messages  chan string
	templates *template.Template
	sgClient  *sendgrid.SGClient
}

// Config stores application defined options
type Config struct {
	DatabaseAddress string
	SendGridAPIKey  string
	JustPrint       bool
}

// OpenSession initialises a connection with the mailing list system. A call to
// Session.Close() should follow to ensure a clean exit.
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

// Close closes the session. It blocks until the session is cleanly exited
func (s *Session) Close() error {
	s.messages <- "close"
	return s.db.Close()
}

// listens for commands from the API. This is intended to be run asynchronously
// and mainly exists to prevent the API from blocking.
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
