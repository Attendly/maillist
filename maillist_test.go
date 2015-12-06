package maillist_test

import (
	"log"
	"os"
	"time"

	"github.com/Attendly/maillist"
)

// Example session of sending a single test email. DatabaseAddress,
// SendGridAPIKey would have to be set appropriately. JustPrint should be
// false, and Subscriber.Email changed to send a real message.
func Example() {
	var err error
	var s *maillist.Session

	config := maillist.Config{
		DatabaseAddress: "tt:tt@unix(/run/mysqld/mysqld.sock)/attendly_email_service",
		SendGridAPIKey:  "",
		JustPrint:       true,
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}

	a := maillist.Account{
		FirstName: "Joe",
		LastName:  "Bloggs",
		Email:     "sendgrid@eventarc.com",
	}
	if err := s.InsertAccount(&a); err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}

	l := maillist.List{
		AccountID: a.ID,
		Name:      "My Awesome Mailing List",
		EventID:   5,
	}
	if err = s.InsertList(&l); err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}

	sub := maillist.Subscriber{
		ListID:    l.ID,
		FirstName: "Tommy",
		LastName:  "Barker",
		Email:     "tom@attendly.com",
	}
	if err = s.InsertSubscriber(&sub); err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}

	c := maillist.Campaign{
		AccountID: a.ID,
		Subject:   "Awesome Event 2016",
		Body:      "This is a test of attendly email list service",
	}
	if err = s.SendCampaign(&c, &l); err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}
	time.Sleep(time.Second)
	s.Close()

	// Output:
	// Email to send
	// To: tom@attendly.com (Tommy Barker)
	// From: sendgrid@eventarc.com ()
	// Subject: Awesome Event 2016
	// Body: This is a test of attendly email list service
}
