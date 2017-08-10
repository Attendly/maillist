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
	ID         int64  `db:"id"`
	AccountID  int64  `db:"account_id" validate:"required"`
	FirstName  string `db:"first_name" validate:"required"`
	LastName   string `db:"last_name" validate:"required"`
	Email      string `db:"email" validate:"required,email"`
	Status     string `db:"status" validate:"eq=active|eq=deleted|eq=unsubscribed"`
	CreateTime int64  `db:"create_time" validate:"required"`
}

// GetSubscribers retrieves all the subscribers in a mailing list
func (s *Session) GetSubscribers(listID int64) ([]*Subscriber, error) {
	var subs []*Subscriber

	selectSQL := fmt.Sprintf(`
SELECT
	%s
FROM
	subscriber

INNER JOIN
	list_subscriber
ON
	subscriber.id=subscriber_id

WHERE
	subscriber.status='active'
	AND list_id=?`,
		s.selectString(&Subscriber{}))

	if _, err := s.dbmap.Select(&subs, selectSQL, listID); err != nil {
		return nil, err

	} else if len(subs) == 0 {
		return nil, ErrNotFound
	}
	return subs, nil
}

// GetSubscriber retrieves a subscriber with a given ID. Returns nil,nil if no
// such subscriber exists
func (s *Session) GetSubscriber(subscriberID int64) (*Subscriber, error) {
	var sub Subscriber
	query := fmt.Sprintf("select %s from subscriber where id=? and status!='deleted'",
		s.selectString(&sub))

	err := s.dbmap.SelectOne(&sub, query, subscriberID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound

	} else if err != nil {
		return nil, err
	}
	return &sub, nil
}

// GetSubscriberByEmail retrieves a subscriber with a given email address.
// Returns nil,nil if no such subscriber exists
func (s *Session) GetSubscriberByEmail(email string, accountID int64) (*Subscriber, error) {

	selectSQL := fmt.Sprintf(`
SELECT
	%s
FROM
	subscriber

WHERE
	status!='deleted'
	AND email=?
	AND account_id=?`,
		s.selectString(Subscriber{}))

	var sub Subscriber
	err := s.dbmap.SelectOne(&sub, selectSQL, email, accountID)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound

	} else if err != nil {
		return nil, err
	}
	return &sub, nil
}

// InsertSubscriber into the db
func (s *Session) InsertSubscriber(sub *Subscriber) error {
	if sub.Status == "" {
		sub.Status = statusActive
	}
	return s.insert(sub)
}

// DeleteSubscriber from the db
func (s *Session) DeleteSubscriber(id int64) error {
	return s.delete(Subscriber{}, id)
}

// Unsubscribe marks a subscriber as not wanting to recieve any more marketting
// emails
func (s *Session) Unsubscribe(sub *Subscriber) error {

	updateSQL := `
UPDATE
	subscriber

SET
	status='unsubscribed'

WHERE
	id=?`

	_, err := s.dbmap.Exec(updateSQL, sub.ID)

	return err
}

// getUnsubscribeSalt gets a random string unique to this installation to salt
// unsubscribe tokens with as part of the hashing process.
func getUnsubscribeSalt(s *Session) (string, error) {

	selectSQL := `
SELECT
	value
FROM
	variable

WHERE
	name='unsubscribe-salt'`

	salt, err := s.dbmap.SelectStr(selectSQL)
	if err != nil {
		return "", err
	}
	if salt != "" {
		return salt, nil
	}

	var buf [64]byte
	if _, err = rand.Read(buf[:]); err != nil {
		return "", err
	}
	salt = base64.StdEncoding.EncodeToString(buf[:])

	insertSQL := `
INSERT INTO
	variable
	(name, value)

VALUES
	('unsubscribe-salt', ?)`

	_, err = s.dbmap.Exec(insertSQL, salt)
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

	buf := sha256.Sum256([]byte(salt + sub.Email + strconv.FormatInt(sub.ID, 10)))
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
		return nil, fmt.Errorf("unsubscribe token could not be parsed: %v", err)
	}

	sub, err := s.GetSubscriber(id)
	if err != nil {
		return nil, err

	} else if sub == nil {
		err = fmt.Errorf("subscriber with ID '%d' not found", id)
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
