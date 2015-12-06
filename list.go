package maillist

type List struct {
	ID        int64  `db:"id"`
	AccountID int64  `db:"account_id" validate:"required"`
	Name      string `db:"name" validate:"required"`
	EventID   int64  `db:"event_id" validate:"required"`
	Status    string `db:"status" validate:"eq=active|eq=deleted"`
}

func (s *Session) GetLists(accountID int64) ([]*List, error) {
	var ls []*List
	if err := s.selectMany(&ls, "account_id", accountID); err != nil {
		return nil, err
	}

	return ls, nil
}

func (s *Session) InsertList(l *List) error {
	if l.Status == "" {
		l.Status = "active"
	}
	return s.insert(l)
}

func (s *Session) GetList(listID int64) (*List, error) {
	var l List
	err := s.selectOne(&l, "id", listID)
	return &l, err
}

func (s *Session) UpdateList(l *List) error {
	return s.update(l)
}

func (s *Session) DeleteList(listID int64) error {
	return s.delete(List{}, listID)
}
