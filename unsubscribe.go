package maillist

import "time"

type Unsubscribe struct {
	ID      int64     `db:"id"`
	Email   string    `db:"email" validate:"required,email"`
	Created time.Time `db:"created" validate:"eq(0)"`
}
