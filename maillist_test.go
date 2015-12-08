package maillist_test

import (
	"log"
	"os"
	"time"

	"github.com/Attendly/maillist"
	_ "github.com/go-sql-driver/mysql"
)

// Example session of sending a single test email. DatabaseAddress,
// SendGridAPIKey would have to be set appropriately. JustPrint should be
// false, and Subscriber.Email changed to send a real message.
func Example() {
	var err error
	var s *maillist.Session
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config := maillist.Config{
		DatabaseAddress: os.Getenv("ATTENDLY_EMAIL_DATABASE"),
		JustPrint:       true,

		SendGridUsername: os.Getenv("ATTENDLY_EMAIL_USERNAME"),
		SendGridPassword: os.Getenv("ATTENDLY_EMAIL_PASSWORD"),
		SendGridAPIKey:   os.Getenv("ATTENDLY_EMAIL_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	a := maillist.Account{
		FirstName: "Joe",
		LastName:  "Bloggs",
		Email:     "sendgrid@eventarc.com",
	}
	if err := s.UpsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	l := maillist.List{
		AccountID: a.ID,
		Name:      "My Awesome Mailing List",
	}
	if err = s.InsertList(&l); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	sub := maillist.Subscriber{
		AccountID: a.ID,
		FirstName: "Tommy",
		LastName:  "Barker",
		Email:     "tom@attendly.com",
	}
	if err = s.GetOrInsertSubscriber(&sub); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	if err = s.AddSubscriberToList(l.ID, sub.ID); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	c := maillist.Campaign{
		AccountID: a.ID,
		Subject:   "Awesome Event 2016",
		Body:      "Hi {{.FirstName}} {{.LastName}},\nThis is a test of attendly email list service",
		Scheduled: time.Now(),
	}
	if err = s.InsertCampaign(&c, []int64{l.ID}, nil); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	time.Sleep(5 * time.Second)
	if err := s.Close(); err != nil {
		log.Fatalf("could not close session: %v", err)
	}

	// Output:
	// Email to send
	// To: tom@attendly.com (Tommy Barker)
	// From: sendgrid@eventarc.com (Joe Bloggs)
	// Subject: Awesome Event 2016
	// Body: Hi Tommy Barker,
	// This is a test of attendly email list service
}

func ExampleWithEvent() {
	var err error
	var s *maillist.Session
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var accountID int64

	getAttendees := func(eventID int64) []*maillist.Subscriber {
		return []*maillist.Subscriber{{
			AccountID: accountID,
			FirstName: "Freddy",
			LastName:  "Example",
			Email:     "fred@example.com",
		}}
	}

	config := maillist.Config{
		DatabaseAddress:      os.Getenv("ATTENDLY_EMAIL_DATABASE"),
		GetAttendeesCallback: getAttendees,
		JustPrint:            true,

		SendGridUsername: os.Getenv("ATTENDLY_EMAIL_USERNAME"),
		SendGridPassword: os.Getenv("ATTENDLY_EMAIL_PASSWORD"),
		SendGridAPIKey:   os.Getenv("ATTENDLY_EMAIL_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	a := maillist.Account{
		FirstName: "Joe",
		LastName:  "Bloggs",
		Email:     "sendgrid@eventarc.com",
	}
	if err := s.UpsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	accountID = a.ID

	l := maillist.List{
		AccountID: a.ID,
		Name:      "My Awesome Mailing List",
	}
	if err = s.InsertList(&l); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	c := maillist.Campaign{
		AccountID: a.ID,
		Subject:   "Awesome Event 2016",
		Body:      "Hi {{.FirstName}} {{.LastName}},\nThis is a test of attendly email list service",
		Scheduled: time.Now(),
	}
	if err = s.InsertCampaign(&c, nil, []int64{5}); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	time.Sleep(5 * time.Second)
	if err := s.Close(); err != nil {
		log.Fatalf("could not close session: %v", err)
	}

	// Output:
	// Email to send
	// To: fred@example.com (Freddy Example)
	// From: sendgrid@eventarc.com (Joe Bloggs)
	// Subject: Awesome Event 2016
	// Body: Hi Freddy Example,
	// This is a test of attendly email list service
}

// func TestGetSpamReports(t *testing.T) {

// config := maillist.Config{
// DatabaseAddress: os.Getenv("ATTENDLY_EMAIL_DATABASE"),
// JustPrint:       true,

// SendGridUsername: os.Getenv("ATTENDLY_EMAIL_USERNAME"),
// SendGridPassword: os.Getenv("ATTENDLY_EMAIL_PASSWORD"),
// SendGridAPIKey:   os.Getenv("ATTENDLY_EMAIL_APIKEY"),
// }

// s, err := maillist.OpenSession(&config)
// if err != nil {
// t.Errorf("%v", err)
// }

// reports, err := s.GetSpamReports()

// t.Errorf("%+v\n%+v\n\n", reports, err)
// }
