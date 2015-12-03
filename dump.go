package maillist

import "time"

type Session struct {
	database
	config Config
}

type Unsubscribe struct {
	ID      int64     `db:"id"`
	Email   string    `db:"email" validate:"required,email"`
	Created time.Time `db:"created" validate:"eq(0)"`
}

type Message struct {
	ID           int64  `db:"id"`
	SubscriberID int64  `db:"subscriber_id" validate:"required"`
	CampaignID   int64  `db:"campaign_id" validate:"required"`
	Status       string `db:"status" validate:"eq=pending|eq=sent"`
}

type Config struct {
	DatabaseAddress string
	SendGridAPIKey  string
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

	return &s, err
}

func (s *Session) InsertMessage(m *Message) error {
	return s.insert(m)
}
