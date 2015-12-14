package maillist

import (
	"database/sql"
	"fmt"
)

// List represents a user defined mailing list, these are seperate from
// event-associated lists
type List struct {
	ID         int64  `db:"id"`
	AccountID  int64  `db:"account_id" validate:"required"`
	Name       string `db:"name" validate:"required"`
	Status     string `db:"status" validate:"eq=active|eq=deleted"`
	CreateTime int64  `db:"create_time" validate:"required"`
}

// ListSubscriber represents a joining table for list and subscribers
type ListSubscriber struct {
	ListID       int64 `db:"list_id" validate:"required"`
	SubscriberID int64 `db:"subscriber_id" validate:"required"`
	CreateTime   int64 `db:"create_time" validate:"required"`
}

// GetLists retrieves all the mailing lists associated with an account.
func (s *Session) GetLists(accountID int64) ([]*List, error) {

	selectSQL := fmt.Sprintf(`
SELECT %s
	FROM list

WHERE status!='deleted'
	AND account_id=?`,
		s.selectString(&List{}))

	var ls []*List
	if _, err := s.dbmap.Select(&ls, selectSQL, accountID); err != nil {
		return nil, err
	}

	return ls, nil
}

// InsertList adds a new mailing list to the database.
func (s *Session) InsertList(l *List) error {
	if l.Status == "" {
		l.Status = "active"
	}
	return s.insert(l)
}

// GetList retrieves a mailing list with a given ID
func (s *Session) GetList(listID int64) (*List, error) {

	query := fmt.Sprintf(`
SELECT %s
	FROM list

WHERE status!='deleted'
	AND id=?`,
		s.selectString(List{}))

	var l List
	if err := s.dbmap.SelectOne(&l, query, listID); err == sql.ErrNoRows {
		return nil, nil

	} else if err != nil {
		return nil, err
	}
	return &l, nil
}

// UpdateList updates a mailing list in the database, identified by it's ID
func (s *Session) UpdateList(l *List) error {
	return s.update(l)
}

// DeleteList removes a mailing list from the database (actually just marks it
// as `deleted` so we can a log of it)
func (s *Session) DeleteList(listID int64) error {
	return s.delete(List{}, listID)
}

// AddSubscriberToList adds a subscriber to a mailing list. Internally it is
// added to the list_subscriber joining table
func (s *Session) AddSubscriberToList(listID, subscriberID int64) error {

	query := `
SELECT account_id
	FROM list

WHERE status!='deleted'
	AND id=?`

	listAccountID, err := s.dbmap.SelectInt(query, listID)
	if err != nil {
		return err
	}
	if listAccountID == 0 {
		return fmt.Errorf("could not find associated account of list id:%d",
			listID)
	}

	subscriberAccountID, err := s.dbmap.SelectInt(`
SELECT account_id
	FROM subscriber

WHERE status!='deleted'
	AND id=?`,
		subscriberID)

	if err != nil {
		return err
	}
	if subscriberAccountID == 0 {
		return fmt.Errorf("could not find associated account of subscriber id:%d",
			subscriberID)
	}

	if listAccountID != subscriberAccountID {
		return fmt.Errorf("list and subscriber must be in the same account")
	}

	ls := ListSubscriber{
		ListID:       listID,
		SubscriberID: subscriberID,
	}

	return s.insert(&ls)
}

// RemoveSubscriberFromList removes a subscriber from a list. Note this is
// distinct from unsubscribing which is done on an account basis
func (s *Session) RemoveSubscriberFromList(listID, subscriberID int64) error {

	query := `
DELETE FROM list_subscriber

WHERE list_id=?
	AND subscriber_id=?`

	_, err := s.dbmap.Exec(query, listID, subscriberID)
	return err
}
