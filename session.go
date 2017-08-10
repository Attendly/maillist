package maillist

import (
	"errors"
	"fmt"
	"html/template"
	"time"
)

type getAttendeeFunc func(eventID int64) []*Subscriber

// Session is an opaque type holding database connections and other
// implementation details
type Session struct {
	database
	config    Config
	wake      chan bool
	templates map[int64]*template.Template
}

// Config stores application defined options
type Config struct {
	DatabaseAddress      string
	JustPrint            bool
	Logger               Logger
	GetAttendeesCallback getAttendeeFunc
	UnsubscribeURL       string

	SendGridAPIKey   string
	SendGridUsername string
	SendGridPassword string
}

// Logger interface
type Logger interface {
	Error(a ...interface{})
	Info(a ...interface{})
}

func (c Session) error(a ...interface{}) {
	if c.config.Logger != nil {
		c.config.Logger.Error(a...)
	} else {
		s := append([]interface{}{"[error]"}, a...)
		fmt.Println(s...)
	}
}

func (c Session) info(a ...interface{}) {
	if c.config.Logger != nil {
		c.config.Logger.Info(a...)
	} else {
		fmt.Println(a...)
	}
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

	if !config.JustPrint {
		if config.SendGridAPIKey == "" {
			return nil, errors.New("maillist: SendGridAPIKey must be set")
		}
		if config.SendGridUsername == "" {
			return nil, errors.New("maillist: SendGridUsername must be set")
		}
		if config.SendGridPassword == "" {
			return nil, errors.New("maillist: SendGridPassword must be set")
		}
	}

	if s.config.GetAttendeesCallback == nil {
		s.config.GetAttendeesCallback = func(eventID int64) []*Subscriber {
			s.error("maillist: GetAttendeesCallback not set -- sending to events disabled")
			return nil
		}
	}

	s.addTable(Account{}, "account")
	s.addTable(List{}, "list")
	s.addTable(Campaign{}, "campaign")
	s.addTable(Subscriber{}, "subscriber")
	s.addTable(Message{}, "message")
	s.addTable(ListSubscriber{}, "list_subscriber")

	s.templates = make(map[int64]*template.Template)

	s.wake = make(chan bool)
	go service(&s)
	s.wake <- true

	return &s, err
}

// Close closes the session. It blocks until the session is cleanly exited
func (c *Session) Close() error {
	close(c.wake)
	return c.db.Close()
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
		if err == ErrNotFound {
			break

		} else if err != nil {
			s.error("couldn't retrieve due campaign:", err)
			break
		}

		if err = s.sendCampaign(c.ID); err != nil {
			s.error("couldn't send campaign:", err)
			break
		}
	}

	for {
		m, err := pendingMessage(s)
		if err == ErrNotFound {
			break

		} else if err != nil {
			s.error("couldn't retrieve pending message:", err)
			break
		}

		if err = s.sendMessage(m); err != nil {
			s.error(err)
			break
		}
		time.Sleep(time.Second)
	}
	goto next
}
