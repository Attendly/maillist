package maillist

import (
	"html/template"
	"log"
	"time"

	"github.com/sendgrid/sendgrid-go"
)

type getAttendeeFunc func(eventID int64) []*Subscriber

// Session is an opaque type holding database connections and other
// implementation details
type Session struct {
	database
	config    Config
	wake      chan bool
	templates map[int64]*template.Template
	sgClient  *sendgrid.SGClient
}

// Config stores application defined options
type Config struct {
	DatabaseAddress      string
	JustPrint            bool
	GetAttendeesCallback getAttendeeFunc

	SendGridAPIKey   string
	SendGridUsername string
	SendGridPassword string
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
	s.addTable(Message{}, "message")
	s.addTable(ListSubscriber{}, "list_subscriber")

	// err = s.dbmap.CreateTablesIfNotExists()

	s.templates = make(map[int64]*template.Template)
	s.sgClient = sendgrid.NewSendGridClientWithApiKey(s.config.SendGridAPIKey)

	s.wake = make(chan bool)
	go service(&s)
	s.wake <- true

	return &s, err
}

// Close closes the session. It blocks until the session is cleanly exited
func (s *Session) Close() error {
	close(s.wake)
	return s.db.Close()
}

// listens for commands from the API. This is intended to be run asynchronously
// and mainly exists to prevent the API from blocking.
func service(s *Session) {
	ticker := time.NewTicker(time.Minute)

next:
	select {
	case _, ok := <-s.wake:
		if !ok {
			ticker.Stop()
			return
		}
	case <-ticker.C:
	}

	for {
		c, err := getDueCampaign(s)
		if err != nil {
			log.Printf("error: %v\n", err)
			break
		}
		if c == nil {
			break
		}

		if err = s.sendCampaign(c.ID); err != nil {
			log.Printf("error: %v\n", err)
			break
		}
	}

	for {
		m, err := pendingMessage(s)
		if err != nil {
			log.Printf("error: %v\n", err)
			break
		}
		if m == nil {
			break
		}

		if err = s.sendMessage(m); err != nil {
			log.Print(err)
			break
		}
	}
	goto next
}
