package maillist

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	sql := fmt.Sprintf("select %s from subscriber where id=?",
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
	query := fmt.Sprintf("select %s from subscriber where account_id=? and email=?", s.selectString(sub))
	err := s.dbmap.SelectOne(&sub, query, sub.AccountID, sub.Email)
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
func (s *Session) Unsubscribe(sub *Subscriber) error {
	_, err := s.dbmap.Exec("update subscriber set status='deleted' where id=?", sub.ID)
	return err
}

// getUnsubscribeSalt gets a random string unique to this installation to salt
// unsubscribe tokens with as part of the hashing process.
func getUnsubscribeSalt(s *Session) (string, error) {
	salt, err := s.dbmap.SelectStr("select value from variable where name='unsubscribe-salt'")
	if err != nil {
		return "", err
	}
	if salt != "" {
		return salt, nil
	}

	var buf [64]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	salt = base64.StdEncoding.EncodeToString(buf[:])

	_, err = s.dbmap.Exec("insert into variable (name, value) values ('unsubscribe-salt', ?)", salt)
	if err != nil {
		return "", err
	}
	return salt, nil
}

// UnsubscribeToken gets a crypographically secure token which represents a
// subscriber. Using such a token means that only the recepiant of an email can
// unsubscribe from that mailing list.
func (s *Session) UnsubscribeToken(sub *Subscriber) (string, error) {
	salt, err := getUnsubscribeSalt(s)
	if err != nil {
		return "", err
	}

	buf := sha256.Sum256([]byte(salt + sub.Email))
	hash := base64.URLEncoding.EncodeToString(buf[:])

	return fmt.Sprintf("%d~%s", sub.ID, hash), nil
}

// GetSubscriberByToken retrieves the subscriber associated with a token.
// Returns an error if the token doesn't match any in the database
func (s *Session) GetSubscriberByToken(token string) (*Subscriber, error) {
	ss := strings.Split(token, "~")
	if len(ss) != 2 {
		return nil, errors.New("Unsubscribe token could not be parsed")
	}

	id, err := strconv.ParseInt(ss[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Unsubscribe token could not be parsed: %v", err)
	}

	sub, err := s.GetSubscriber(id)
	if err != nil {
		return nil, err
	}

	wantedToken, err := s.UnsubscribeToken(sub)
	if err != nil {
		return nil, err
	}
	if token != wantedToken {
		return nil, errors.New("Invalid unsubscribe token")
	}
	return sub, nil
}
