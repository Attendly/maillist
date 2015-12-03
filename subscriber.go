package maillist

type Subscriber struct {
	ID        int64  `db:"id"`
	ListID    int64  `db:"list_id" validate:"required"`
	FirstName string `db:"first_name" validate:"required"`
	LastName  string `db:"last_name" validate:"required"`
	Email     string `db:"email" validate:"required,email"`
	Status    string `db:"status" validate:"eq=active|eq=deleted"`
}

func (s *Session) GetSubscribers(listID int64) ([]*Subscriber, error) {
	var subs []*Subscriber
	if err := s.database.selectMany(&subs, "list_id", listID); err != nil {
		return nil, err
	}
	return subs, nil
}

func (s *Session) InsertSubscriber(sub *Subscriber) error {
	return s.insert(sub)
}

func (s *Session) GetSubscriber(subscriberID int64) (*Subscriber, error) {
	var sub Subscriber
	err := s.selectOne(&sub, "id", subscriberID)
	return &sub, err
}
