package maillist

import (
	"database/sql"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
)

type Message struct {
	ID           int64  `db:"id"`
	SubscriberID int64  `db:"subscriber_id" validate:"required"`
	CampaignID   int64  `db:"campaign_id" validate:"required"`
	Status       string `db:"status" validate:"eq=pending|eq=sent"`
}

func (s *Session) InsertMessage(m *Message) error {
	return s.insert(m)
}

func pendingMessage(s *Session) (*Message, error) {
	var m Message
	err := s.selectOne(&m, "status", "pending")
	if err == nil {
		return &m, nil
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

func (s *Session) sendMessage(m *Message) error {
	email, err := buildEmail(s, m)
	if err != nil {
		return err
	}

	if s.config.JustPrint {
		printEmail(email)
	} else if err := s.sgClient.Send(email); err != nil {
		return err
	}

	m.Status = "sent"
	if err = s.update(&m); err != nil {
		return fmt.Errorf("couldn't update message status: %v\n", err)
	}
	if err = s.updateCampaignStatus(m.CampaignID); err != nil {
		return fmt.Errorf("couldn't update campaign status: %v\n", err)
	}
	return nil
}

func buildEmail(s *Session, m *Message) (*sendgrid.SGMail, error) {
	email := sendgrid.NewMail()

	sub, err := s.GetSubscriber(m.SubscriberID)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get subscriber: %v", err)
	}
	campaign, err := s.GetCampaign(m.CampaignID)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get campaign %d: %v", m.CampaignID, err)
	}
	account, err := s.GetAccount(campaign.AccountID)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get account: %v", err)
	}

	email.AddTo(sub.Email)
	email.AddToName(sub.FirstName + " " + sub.LastName)
	email.SetSubject(campaign.Subject)
	email.SetHTML(campaign.Body)
	email.SetFrom(account.Email)
	return email, nil
}

func printEmail(m *sendgrid.SGMail) {
	fmt.Println("Email to send")
	fmt.Printf("To: %s (%s)\n", m.To[0], m.ToName[0])
	fmt.Printf("From: %s (%s)\n", m.From, m.FromName)
	fmt.Printf("Subject: %s\nBody: %s\n", m.Subject, m.HTML)
}
