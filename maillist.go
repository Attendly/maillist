// Package maillist sends bulk e-mail to lists of addresses using
// the Sendgrid API.
//
// All functionality is implemented as methods on a session object,
// which should be closed when finished with it.
//
// Usage:
// - Open a new session
// - Create or retrieve an Account
// - Create or retrive a List (which contains subscribers)
// - Optionally add more subscribers to the list
// - Create and schedule a campaign for that list
// - This package will ensure the emails are sent out when the scheduled
//   time is reached

package maillist
