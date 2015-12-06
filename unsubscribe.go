package maillist

import "time"

// Unsubscribe records that an emailing address has unsubscribed from a given
// account and should never be sent a marketing email
type Unsubscribe struct {
	ID        int64     `db:"id"`
	AccountID int64     `db:"account_id" validate:"required"`
	Email     string    `db:"email" validate:"required,email"`
	Created   time.Time `db:"created" validate:"eq(0)"`
}
