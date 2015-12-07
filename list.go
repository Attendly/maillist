package maillist

type List struct {
	ID        int64  `db:"id"`
	AccountID int64  `db:"account_id" validate:"required"`
	Name      string `db:"name" validate:"required"`
	EventID   int64  `db:"event_id" validate:"required"`
	Status    string `db:"status" validate:"eq=active|eq=deleted"`
}

// ListSubscriber represents a joining table for list and subscribers
type ListSubscriber struct {
	ID           int64  `db:"id"`
	ListID       int64  `db:"list_id" validate:"required"`
	SubscriberID int64  `db:"subscriber_id" validate:"required"`
	Status       string `db:"status" validate:"eq=active|eq=deleted"`
}

// GetLists retrieves all the mailing lists associated with an account.
func (s *Session) GetLists(accountID int64) ([]*List, error) {
	var ls []*List
	if err := s.selectMany(&ls, "account_id", accountID); err != nil {
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
	var l List
	err := s.selectOne(&l, "id", listID)
	return &l, err
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

func (s *Session) AddSubscriberToList(listID, subscriberID int64) error {
	ls := ListSubscriber{
		ListID:       listID,
		SubscriberID: subscriberID,
		Status:       "active",
	}
	return s.insert(&ls)
}
