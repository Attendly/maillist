package webapi

import (
	"log"
	"os"

	"github.com/Attendly/maillist"
)

func main() {
	var err error
	// doSendGrid()
	// db.Init()
	// srv := api.InitAPI()

	// err := http.ListenAndServe(":8080", srv)
	// if err != nil {
	// log.Printf("error: %v", err)
	// }

	config := maillist.Config{
		DatabaseAddress: os.Args[1],
		SendGridAPIKey:  os.Args[2],
	}

	s, err := maillist.OpenSession(&config)
	if err != nil {
		log.Printf("error: %v\n", err)
	}

	m := maillist.Message{
		SubscriberID: 54,
		CampaignID:   10,
		Status:       "pending",
	}

	err = s.InsertMessage(&m)
	if err != nil {
		log.Printf("error: %v\n", err)
	}

	// a := maillist.Account{
	// Email: "sendgrid@eventarc.com",
	// }
	// s.InsertAccount(&a)

	// l := maillist.List{
	// AccountID: a.ID,
	// Name:      "My Awesome Mailing List",
	// EventID:   5,
	// }
	// err = s.InsertList(&l)
	// if err != nil {
	// log.Printf("error: %v\n", err)
	// }

	// sub := maillist.Subscriber{
	// ListID:    l.ID,
	// FirstName: "Tommy",
	// LastName:  "Barker",
	// Email:     "tom@attendly.com",
	// }
	// err = s.InsertSubscriber(&sub)
	// if err != nil {
	// log.Printf("error: %v\n", err)
	// }

	// c := maillist.Campaign{
	// Subject: "Awesome Event 2016",
	// Body:    "This is a test of attendly email list service",
	// }
	// s.SendCampaign(&c, &a, &l)
}
