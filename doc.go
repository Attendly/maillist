/*
Package maillist sends bulk e-mail to lists of addresses using
the Sendgrid API.

All functionality is implemented as methods on a session object,
which should be closed when finished with it.

The script 'init-db.sh' should be run to initialize the database for this
package.

Usage

Open a new session.

	config := maillist.Config{
		DatabaseAddress: "username:password@unix(/run/mysqld/mysqld.sock)/attendly_email_service"
		JustPrint:       true,
		Logger:          os.Stdout,
		UnsubscribeURL:  "https://localhost/unsubscribe",

		SendGridUsername: "sendgrid@example.com"
		SendGridPassword: "asdf1234"
		SendGridAPIKey:   "SG.0x145597e70x13313b210x49fd869f"
	}
	s, _ := maillist.OpenSession(&config)
	defer s.Close()

Create or retrieve an Account.
	a := maillist.Account{
		FirstName: "Joe",
		LastName:  "Bloggs",
		Email:     "sendgrid@example.com",
	}
	s.InsertAccount(&a)

Create or retrive a List (which contains subscribers).
	l := maillist.List{
		AccountID: a.ID,
		Name:      "My Awesome Mailing List",
	}
	s.InsertList(&l)

Optionally add more subscribers to the list.
	sub := maillist.Subscriber{
		AccountID: a.ID,
		FirstName: "Tommy",
		LastName:  "Barker",
		Email:     "tom@example.com",
	}
	s.InsertSubscriber(&sub)
	s.AddSubscriberToList(l.ID, sub.ID)

Create and schedule a campaign for that list.
	c := maillist.Campaign{
		AccountID: a.ID,
		Subject:   "Awesome Event 2016",
		Body:      "Hi {{.FirstName}} {{.LastName}},\nThis is a test of attendly email list service",
		Scheduled: time.Now().Unix(),
	}
	s.InsertCampaign(&c, []int64{l.ID}, nil)

This package will ensure the emails are sent out when the scheduled
time is reached as long as at least one session remains open.
*/
package maillist
