package maillist_test

import (
	"log"
	"time"

	"github.com/Attendly/maillist"
)

// Example session of sending a single test email. DatabaseAddress,
// SendGridAPIKey would have to be set appropriately. JustPrint should be
// false, and Subscriber.Email changed to send a real message.
func Example() {
	var err error
	var s *maillist.Session
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config := maillist.Config{
		DatabaseAddress: "tt:tt@unix(/run/mysqld/mysqld.sock)/attendly_email_service",
		SendGridAPIKey:  "",
		JustPrint:       true,
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	a := maillist.Account{
		FirstName: "Joe",
		LastName:  "Bloggs",
		Email:     "sendgrid@eventarc.com",
	}
	if err := s.InsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	l := maillist.List{
		AccountID: a.ID,
		Name:      "My Awesome Mailing List",
		EventID:   5,
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
	if err = s.InsertSubscriber(&sub); err != nil {
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
	if err = s.InsertCampaign(&c, &l); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	time.Sleep(5 * time.Second)
	s.Close()

	// Output:
	// Email to send
	// To: tom@attendly.com (Tommy Barker)
	// From: sendgrid@eventarc.com (Joe Bloggs)
	// Subject: Awesome Event 2016
	// Body: Hi Tommy Barker,
	// This is a test of attendly email list service
}
