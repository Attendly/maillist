package maillist_test

import (
	"bytes"
	"log"
	"os"
	"testing"
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
		DatabaseAddress: os.Getenv("MAILLIST_DATABASE"),
		JustPrint:       true,
		Logger:          os.Stdout,
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",

		SendGridUsername: os.Getenv("SENDGRID_USERNAME"),
		SendGridPassword: os.Getenv("SENDGRID_PASSWORD"),
		SendGridAPIKey:   os.Getenv("SENDGRID_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	defer s.Close()

	a := maillist.Account{
		FirstName: "Joe",
		LastName:  "Bloggs",
		Email:     "sendgrid@example.com",
	}
	if err := s.InsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	defer s.DeleteAccount(a.ID)

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
	if err = s.InsertSubscriber(&sub); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	defer s.DeleteSubscriber(sub.ID)

	if err = s.AddSubscriberToList(l.ID, sub.ID); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	c := maillist.Campaign{
		AccountID: a.ID,
		Subject:   "Awesome Event 2016",
		Body:      "Hi {{.FirstName}} {{.LastName}},\nThis is a test of attendly email list service",
		Scheduled: time.Now().Unix(),
	}
	if err = s.InsertCampaign(&c, []int64{l.ID}, nil); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	time.Sleep(5 * time.Second)

	// Output:
	// Email to send
	// To: tom@attendly.com (Tommy Barker)
	// From: sendgrid@example.com (Joe Bloggs)
	// Subject: Awesome Event 2016
	// Body: Hi Tommy Barker,
	// This is a test of attendly email list service
}

func TestGetAttendeesCallback(t *testing.T) {
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

	var buf bytes.Buffer

	config := maillist.Config{
		DatabaseAddress:      os.Getenv("MAILLIST_DATABASE"),
		GetAttendeesCallback: getAttendees,
		JustPrint:            true,
		Logger:               &buf,
		UnsubscribeURL:       "https://myeventarc.localhost/unsubscribe",

		SendGridUsername: os.Getenv("SENDGRID_USERNAME"),
		SendGridPassword: os.Getenv("SENDGRID_PASSWORD"),
		SendGridAPIKey:   os.Getenv("SENDGRID_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	defer s.Close()

	a := maillist.Account{
		FirstName: "Spamface",
		LastName:  "The Bold",
		Email:     "example@example.com",
	}
	if err := s.InsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	defer s.DeleteAccount(a.ID)
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
		Scheduled: time.Now().Unix(),
	}
	if err = s.InsertCampaign(&c, nil, []int64{5}); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	time.Sleep(5 * time.Second)

	out := buf.String()
	want := `Email to send
To: fred@example.com (Freddy Example)
From: example@example.com (Spamface The Bold)
Subject: Awesome Event 2016
Body: Hi Freddy Example,
This is a test of attendly email list service
`
	if out != want {
		log.Fatalf("got: '%s'\n\nwant: '%s'\n\n", out, want)
	}
}

func TestGetSpamReports(t *testing.T) {
	config := maillist.Config{
		DatabaseAddress: os.Getenv("MAILLIST_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("SENDGRID_USERNAME"),
		SendGridPassword: os.Getenv("SENDGRID_PASSWORD"),
		SendGridAPIKey:   os.Getenv("SENDGRID_APIKEY"),
	}

	s, err := maillist.OpenSession(&config)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer s.Close()

	if b, err := s.HasReportedSpam("example@example.com"); err != nil {
		t.Fatalf("error: %v\n", err)
	} else if b {
		t.Fatalf("Example incorrectly has reported spam\n")
	}

	if b, err := s.HasReportedSpam("jorgen@hotmail.com"); err != nil {
		t.Fatalf("error: %v\n", err)
	} else if !b {
		t.Fatalf("Example incorrectly has not reported spam\n")
	}
}

func TestUnsubscribeToken(t *testing.T) {
	var (
		err       error
		token     string
		sub, sub2 *maillist.Subscriber
		s         *maillist.Session
	)

	config := maillist.Config{
		DatabaseAddress: os.Getenv("MAILLIST_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("SENDGRID_USERNAME"),
		SendGridPassword: os.Getenv("SENDGRID_PASSWORD"),
		SendGridAPIKey:   os.Getenv("SENDGRID_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		t.Fatalf("%v", err)
	}
	defer s.Close()

	a := maillist.Account{
		FirstName: "Ray",
		LastName:  "Charles",
		Email:     "raycharles@example.com",
	}
	if err := s.InsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	defer s.DeleteAccount(a.ID)

	sub = &maillist.Subscriber{
		AccountID: a.ID,
		FirstName: "Johnny",
		LastName:  "Knoxville",
		Email:     "johnny.k@example.com",
	}

	if err = s.InsertSubscriber(sub); err != nil {
		t.Fatalf("error: %v", err)
	}
	defer s.DeleteSubscriber(sub.ID)

	if token, err = s.UnsubscribeToken(sub); err != nil {
		t.Fatalf("error: %v", err)
	}

	if sub2, err = s.GetSubscriberByToken(token); err != nil {
		t.Fatalf("error:%v", err)
	}
	if sub.ID != sub2.ID {
		t.Fatalf("GetSubscriberByToken result incorrect\n")
	}
}

func TestGetLists(t *testing.T) {
	config := maillist.Config{
		DatabaseAddress: os.Getenv("MAILLIST_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("SENDGRID_USERNAME"),
		SendGridPassword: os.Getenv("SENDGRID_PASSWORD"),
		SendGridAPIKey:   os.Getenv("SENDGRID_APIKEY"),
	}

	s, err := maillist.OpenSession(&config)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer s.Close()

	a := maillist.Account{
		FirstName: "Brian",
		LastName:  "Cohen",
		Email:     "briancohen@example.com",
	}
	if err := s.InsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	defer s.DeleteAccount(a.ID)

	l1 := maillist.List{
		AccountID: a.ID,
		Name:      "TestGetLists 1",
	}
	if err = s.InsertList(&l1); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	l2 := maillist.List{
		AccountID: a.ID,
		Name:      "TestGetLists 2",
	}
	if err = s.InsertList(&l2); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	lists, err := s.GetLists(a.ID)
	if err != nil {
		log.Fatalf("Could not GetLists: %v", err)
	}

	if len(lists) != 2 {
		log.Fatalf("Error in GetLists: length is %d, want %d\n", len(lists), 2)
	}

	if (lists[0].ID != l1.ID || lists[1].ID != l2.ID) &&
		(lists[0].ID != l2.ID || lists[1].ID != l1.ID) {
		log.Fatalf("error in GetLists: didn't get list\n")
	}

	if err := s.DeleteList(l1.ID); err != nil {
		log.Fatalf("Could not delete mailing lists: %v", err)
	}

	if err := s.DeleteList(l2.ID); err != nil {
		log.Fatalf("Could not delete mailing lists: %v", err)
	}
}

func TestMultipleAccounts(t *testing.T) {
	var (
		err error
		s   *maillist.Session
	)

	config := maillist.Config{
		DatabaseAddress: os.Getenv("MAILLIST_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("SENDGRID_USERNAME"),
		SendGridPassword: os.Getenv("SENDGRID_PASSWORD"),
		SendGridAPIKey:   os.Getenv("SENDGRID_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		t.Fatalf("Could not open session: %v", err)
	}
	defer s.Close()

	a1 := maillist.Account{
		FirstName: "Test",
		LastName:  "MultipleAccounts1",
		Email:     "testmultipleaccounts1@example.com",
	}
	a2 := maillist.Account{
		FirstName: "Test",
		LastName:  "MultipleAccounts2",
		Email:     "testmultipleaccounts2@example.com",
	}
	if err := s.InsertAccount(&a1); err != nil {
		t.Fatalf("Could not insert account: %v\n", err)
	}
	defer s.DeleteAccount(a1.ID)
	if err := s.InsertAccount(&a2); err != nil {
		t.Fatalf("Could not insert account: %v\n", err)
	}
	defer s.DeleteAccount(a2.ID)

	l1 := maillist.List{
		AccountID: a1.ID,
		Name:      "Testmultipleaccounts1",
	}

	l2 := maillist.List{
		AccountID: a2.ID,
		Name:      "Testmultipleaccounts2",
	}

	if err := s.InsertList(&l1); err != nil {
		t.Fatalf("Could not insert list: %v\n", err)
	}
	defer s.DeleteList(l1.ID)
	if err := s.InsertList(&l2); err != nil {
		t.Fatalf("Could not insert list: %v\n", err)
	}
	defer s.DeleteList(l2.ID)

	var ls1 []*maillist.List
	if ls1, err = s.GetLists(a1.ID); err != nil {
		t.Fatalf("Could not retrieve lists: %v\n", err)
	}
	var ls2 []*maillist.List
	if ls2, err = s.GetLists(a2.ID); err != nil {
		t.Fatalf("Could not retrieve lists: %v\n", err)
	}
	if len(ls1) != 1 || ls1[0].ID != l1.ID || ls1[0].AccountID != a1.ID {
		t.Fatalf("Get list incorrect result\ngot:%+vwanted:%+v\n", ls1, l1)
	}
	if len(ls2) != 1 || ls2[0].ID != l2.ID || ls2[0].AccountID != a2.ID {
		t.Fatalf("Get list incorrect result\ngot:%+vwanted:%+v\n", ls2, l2)
	}

	s1 := maillist.Subscriber{
		AccountID: a1.ID,
		FirstName: "TestMultipleAccounts",
		LastName:  "1",
		Email:     "testingmultipleaccounts@example.com",
	}
	s2 := maillist.Subscriber{
		AccountID: a2.ID,
		FirstName: "TestMultipleAccounts",
		LastName:  "1",
		Email:     "testingmultipleaccounts@example.com",
	}
	if err = s.InsertSubscriber(&s1); err != nil {
		t.Fatalf("Could not insert subscriber: %v\n", err)
	}
	defer s.DeleteSubscriber(s1.ID)

	if err = s.InsertSubscriber(&s2); err != nil {
		t.Fatalf("Could not insert subscriber: %v\n", err)
	}
	defer s.DeleteSubscriber(s2.ID)

	if s1.ID == s2.ID {
		t.Fatalf("Subscribers should have been added to different accounts")
	}

	if err = s.AddSubscriberToList(l1.ID, s1.ID); err != nil {
		t.Fatalf("Could not add subscriber to list: %v\n", err)
	}
	defer s.RemoveSubscriberFromList(l1.ID, s1.ID)

	if err = s.AddSubscriberToList(l2.ID, s2.ID); err != nil {
		t.Fatalf("Could not add subscriber to list: %v\n", err)
	}
	defer s.RemoveSubscriberFromList(l2.ID, s2.ID)

	if err = s.AddSubscriberToList(l1.ID, s2.ID); err == nil {
		t.Fatal("Expected error when adding subscriber to list with different account")
	}
	if err = s.AddSubscriberToList(l2.ID, s1.ID); err == nil {
		t.Fatal("Expected error when adding subscriber to list with different account")
	}
}

func TestDuplicateAccountEmail(t *testing.T) {
	var (
		s   *maillist.Session
		err error
	)
	config := maillist.Config{
		DatabaseAddress: os.Getenv("MAILLIST_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("SENDGRID_USERNAME"),
		SendGridPassword: os.Getenv("SENDGRID_PASSWORD"),
		SendGridAPIKey:   os.Getenv("SENDGRID_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		t.Fatalf("Could not open session: %v", err)
	}
	defer s.Close()

	a1 := maillist.Account{
		FirstName: "Test",
		LastName:  "Duplicate account email 1",
		Email:     "testduplicateaccountemail@example.com",
	}
	a2 := maillist.Account{
		FirstName: "Test",
		LastName:  "Duplicate account email 2",
		Email:     "testduplicateaccountemail@example.com",
	}

	if err := s.InsertAccount(&a1); err != nil {
		log.Fatalf("Could not insert account: %v", err)
	}
	if err := s.InsertAccount(&a2); err == nil {
		log.Fatalf("Expected error: duplicate email addresses")
	}
	s.DeleteAccount(a1.ID)

	if err := s.InsertAccount(&a2); err != nil {
		log.Fatalf("Could not insert account: %v", err)
	}
	defer s.DeleteAccount(a2.ID)
}

func TestDuplicateSubscriberInList(t *testing.T) {

	var (
		err error
		s   *maillist.Session
	)

	config := maillist.Config{
		DatabaseAddress: os.Getenv("MAILLIST_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("SENDGRID_USERNAME"),
		SendGridPassword: os.Getenv("SENDGRID_PASSWORD"),
		SendGridAPIKey:   os.Getenv("SENDGRID_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		t.Fatalf("Could not open session: %v", err)
	}
	defer s.Close()

	a1 := maillist.Account{
		FirstName: "Test",
		LastName:  "DuplicateSubscriberInList",
		Email:     "testduplicatesubscriberinlist@example.com",
	}
	if err := s.InsertAccount(&a1); err != nil {
		t.Fatalf("Could not insert account: %v\n", err)
	}
	defer s.DeleteAccount(a1.ID)

	l1 := maillist.List{
		AccountID: a1.ID,
		Name:      "testduplicatesubscriberinlist",
	}

	if err := s.InsertList(&l1); err != nil {
		t.Fatalf("Could not insert list: %v\n", err)
	}
	defer s.DeleteList(l1.ID)

	s1 := maillist.Subscriber{
		AccountID: a1.ID,
		FirstName: "Test",
		LastName:  "DuplicateSubscriberInList",
		Email:     "testduplicatesubscriberinlist@example.com",
	}

	if err = s.InsertSubscriber(&s1); err != nil {
		t.Fatalf("Could not insert subscriber: %v\n", err)
	}

	if err = s.InsertSubscriber(&s1); err == nil {
		t.Errorf("should have gotten an error for inserting duplicate subscriber")
	}

	if err = s.AddSubscriberToList(l1.ID, s1.ID); err != nil {
		t.Errorf("could not add subscriber to list: %v\n", err)
	}

	if err = s.AddSubscriberToList(l1.ID, s1.ID); err == nil {
		t.Error("expected error: subscriber already in list")
	}

	if err = s.RemoveSubscriberFromList(l1.ID, s1.ID); err != nil {
		t.Errorf("could not remove subscriber from list: %v\n", err)
	}

	if err = s.AddSubscriberToList(l1.ID, s1.ID); err != nil {
		t.Errorf("could not add subscriber to list: %v\n", err)
	}

	if err = s.DeleteSubscriber(s1.ID); err != nil {
		t.Fatalf("Could not delete subscriber: %v\n", err)
	}
}

func TestAccounts(t *testing.T) {

	var (
		err error
		s   *maillist.Session
	)

	config := maillist.Config{
		DatabaseAddress: os.Getenv("MAILLIST_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("SENDGRID_USERNAME"),
		SendGridPassword: os.Getenv("SENDGRID_PASSWORD"),
		SendGridAPIKey:   os.Getenv("SENDGRID_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		t.Fatalf("Could not open session: %v", err)
	}
	defer s.Close()

	a := maillist.Account{
		FirstName: "Test",
		LastName:  "Accounts",
		Email:     "testaccounts@example.com",
	}
	if err = s.InsertAccount(&a); err != nil {
		t.Fatalf("Could not insert account: %v\n", err)
	}
	if err = s.InsertAccount(&a); err == nil {
		t.Error("expected error when inserting duplicate account email\n")
	}

	if a2, err := s.GetAccountByEmail("testaccounts@example.com"); err != nil || a2 == nil || a2.ID != a.ID {
		t.Errorf("could not retrieve account: %v\n", err)
	}

	if a2, err := s.GetAccountByEmail("notexists@example.com"); err != nil || a2 != nil {
		t.Errorf("got %v %v, expected nil,nil\n", a2, err)
	}

	if a2, err := s.GetAccount(a.ID); err != nil || a2 == nil {
		t.Errorf("could not retrieve account: %v\n", err)
	}

	if a2, err := s.GetAccount(0x777d6afae21b698b); err != nil || a2 != nil {
		t.Errorf("got %v %v, expected nil,nil\n", a2, err)
	}

	if err = s.DeleteAccount(a.ID); err != nil {
		t.Fatalf("Could not delete account: %v\n", err)
	}

	if a2, err := s.GetAccount(a.ID); err != nil || a2 != nil {
		t.Errorf("got %v %v, expected nil,nil\n", a2, err)
	}

	if a2, err := s.GetAccountByEmail(a.Email); err != nil || a2 != nil {
		t.Errorf("got %v %v, expected nil,nil\n", a2, err)
	}

	if err = s.InsertAccount(&a); err != nil {
		t.Fatalf("Could not insert account: %v\n", err)
	}

	if err = s.DeleteAccount(a.ID); err != nil {
		t.Fatalf("Could not delete account: %v\n", err)
	}
}
