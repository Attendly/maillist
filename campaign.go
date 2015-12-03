package maillist

import (
	"fmt"
	"os"
)

type Campaign struct {
	ID      int64  `db:"id" validate="required"`
	Subject string `db:"subject" validate="required"`
	Body    string `db:"body" validate="required"`
	Status  string `db:"status" validate:"eq=pending|eq=sent|eq=cancelled|eq=failed"`
}

func (s *Session) SendCampaign(campaign *Campaign, account *Account, lists ...*List) error {
	if len(lists) != 1 {
		return fmt.Errorf("multiple lists not implemented")
	}

	l := lists[0]

	if l.AccountID != account.ID {
		return fmt.Errorf("List account ID doesn't match accountID")
	}

	_, err := s.GetSubscribers(l.ID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Session) StartService() {
	for {
		var ms []*Message
		err := s.selectMany(&ms, "status", "pending")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}
}

func (s *Session) SendMessage(m *Message) {

	// sg := sendgrid.NewSendGridClientWithApiKey(s.config.SendGridAPIKey)
	// message := sendgrid.NewMail()

	// message.AddTo(s.Email)
	// message.AddToName(s.FirstName + " " + s.LastName)
	// message.SetSubject(campaign.Subject)
	// message.SetText(campaign.Body)
	// message.SetFrom(account.Email)

	// return sg.Send(message)
}
