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

// GetAccount retrieves an account with a given ID. Returns nil,nil if that ID
// does not exist (or has been deleted)
func (s *Session) GetAccount(accountID int64) (*Account, error) {

	selectSQL := fmt.Sprintf(`
SELECT
	%s
FROM
	account

WHERE
	status!='deleted'
	AND id=?`,
		s.selectString(Account{}))

	var a Account
	if err := s.dbmap.SelectOne(&a, selectSQL, accountID); err == sql.ErrNoRows {
		return nil, nil

	} else if err != nil {
		return nil, err
	}

	return &a, nil
}

// GetAccountByEmail retrieves an account with a given email address. Returns
// nil,nil if that email address does not exist (or has been deleted)
func (s *Session) GetAccountByEmail(email string) (*Account, error) {

	selectSQL := fmt.Sprintf(`
SELECT
	%s
FROM
	account

WHERE
	status!='deleted'
	AND email=?`,
		s.selectString(Account{}))

	var a Account
	if err := s.dbmap.SelectOne(&a, selectSQL, email); err == sql.ErrNoRows {
		return nil, nil

	} else if err != nil {
		return nil, err
	}
	return &a, nil
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
