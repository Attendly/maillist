package maillist

type Account struct {
	ID        int64  `db:"id"`
	FirstName string `db:"first_name" validate:"required"`
	LastName  string `db:"last_name" validate:"required"`
	Email     string `db:"email" validate:"required"`
	Status    string `db:"status" validate:"eq=active|eq=deleted"`
}

func (s *Session) InsertAccount(a *Account) error {
	if a.Status == "" {
		a.Status = "active"
	}
	return s.insert(a)
}

func (s *Session) GetAccount(userID int64) (*Account, error) {
	var a Account
	err := s.selectOne(&a, "id", userID)
	return &a, err
}

func (s *Session) UpdateAccount(a *Account) error {
	return s.update(a)
}

func (s *Session) DeleteAccount(accountID int64) error {
	return s.delete(Account{}, accountID)
}
