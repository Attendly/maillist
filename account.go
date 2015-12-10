package maillist

import (
	"database/sql"
	"fmt"
)

// Account is equivalent to a user. All lists, messages, and subscribers must
// have an associated account
type Account struct {
	ID         int64  `db:"id"`
	FirstName  string `db:"first_name" validate:"required"`
	LastName   string `db:"last_name" validate:"required"`
	Email      string `db:"email" validate:"required"`
	Status     string `db:"status" validate:"eq=active|eq=deleted"`
	CreateTime int64  `db:"create_time" validate:"required"`
}

// InsertAccount adds the database to the account. The ID field will be
// updated. It is an error to have duplicate email addresses for the account
// table
func (s *Session) InsertAccount(a *Account) error {
	if a.Status == "" {
		a.Status = "active"
	}
	return s.insert(a)
}

// UpsertAccount updates an account if the associated email address already
// exists. Otherwise it inserts a new account
func (s *Session) UpsertAccount(a *Account) error {
	existing, err := s.GetAccountByEmail(a.Email)
	if err != nil {
		return err
	}

	if existing == nil {
		return s.InsertAccount(a)
	}

	a.ID = existing.ID
	a.Status = "active"
	a.CreateTime = existing.CreateTime
	return s.UpdateAccount(a)
}

// GetAccount retrieves an account with a given ID
func (s *Session) GetAccount(accountID int64) (*Account, error) {
	var a Account
	query := fmt.Sprintf("select %s from account where status!='deleted' and id=?", s.selectString(&a))
	err := s.dbmap.SelectOne(&a, query, accountID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

// GetAccount retrieves an account with a given ID
func (s *Session) GetAccountByEmail(email string) (*Account, error) {
	var a Account
	query := fmt.Sprintf("select %s from account where status!='deleted' and email=?",
		s.selectString(&a))
	err := s.dbmap.SelectOne(&a, query, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

// UpdateAccount updates an account (identified by it's ID)
func (s *Session) UpdateAccount(a *Account) error {
	if a.Status == "" {
		a.Status = "active"
	}
	return s.update(a)
}

// DeleteAccount removes an account
func (s *Session) DeleteAccount(accountID int64) error {
	return s.delete(Account{}, accountID)
}
