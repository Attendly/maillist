package maillist

import (
	"database/sql"
	"fmt"
)

// Subscriber stores a single email address and some associated parameters.
// Each subscriber must have an associated account, and a given email address
// will have one subscriber for each account
type Subscriber struct {
	ID        int64  `db:"id"`
	AccountID int64  `db:"account_id" validate:"required"`
	FirstName string `db:"first_name" validate:"required"`
	LastName  string `db:"last_name" validate:"required"`
	Email     string `db:"email" validate:"required,email"`
	Status    string `db:"status" validate:"eq=active|eq=deleted"`
}

// GetSubscribers retrieves all the subscribers in a mailing list
func (s *Session) GetSubscribers(listID int64) ([]*Subscriber, error) {
	var subs []*Subscriber

	sql := fmt.Sprintf("select %s from subscriber inner join list_subscriber on subscriber.id = subscriber_id where list_id=?", s.selectString(&Subscriber{}))
	if _, err := s.dbmap.Select(&subs, sql, listID); err != nil {
		return nil, err
	}
	return subs, nil
}

// GetSubscriber retrieves a subscriber with a given ID
func (s *Session) GetSubscriber(subscriberID int64) (*Subscriber, error) {
	var sub Subscriber
	sql := fmt.Sprintf("select %s from subscriber where and id=?",
		s.selectString(&sub))
	err := s.dbmap.SelectOne(&sub, sql, subscriberID)
	return &sub, err
}

// GetOrInsertSubscriber retrieves a subscriber from the database if it cannot
// be found. Otherwise adds a new entry. This is mostly used to prevent
// duplicate subscribers.
func (s *Session) GetOrInsertSubscriber(sub *Subscriber) error {
	if sub.ID != 0 {
		return nil
	}
	query := fmt.Sprintf("select %s from subscriber where email=?", s.selectString(sub))
	err := s.dbmap.SelectOne(&sub, query, sub.Email)
	if err != sql.ErrNoRows {
		return err
	}
	if sub.Status == "" {
		sub.Status = "active"
	}
	return s.insert(sub)
}

// Unsubscribe marks a subscriber as not wanting to recieve any more marketting
// emails
func (s *Session) Unsubscribe(subID int64) error {
	_, err := s.dbmap.Exec("update subscriber set status='deleted' where id=?", subID)
	return err
}
