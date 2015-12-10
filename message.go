package maillist

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"

	"github.com/sendgrid/sendgrid-go"
)

// Message is a single email. It keeps track of whether the message has been
// sent or not.
type Message struct {
	SubscriberID int64  `db:"subscriber_id" validate:"required"`
	CampaignID   int64  `db:"campaign_id" validate:"required"`
	Status       string `db:"status" validate:"eq=pending|eq=sent"`
	CreateTime   int64  `db:"create_time" validate:"required"`
}

// InsertMessage inserts a message into the database. It's ID field will be
// updated.
func (s *Session) InsertMessage(m *Message) error {
	return s.insert(m)
}

// pendingMessage retrieves a single message that is waiting to be sent
func pendingMessage(s *Session) (*Message, error) {
	var m Message
	query := fmt.Sprintf("select %s from message where status='pending' limit 1",
		s.selectString(&m))
	err := s.dbmap.SelectOne(&m, query)
	if err == nil {
		return &m, nil
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

// sendMessage sends a single message to it's destination
func (s *Session) sendMessage(m *Message) error {
	var email *sendgrid.SGMail
	var err error

	if email, err = buildEmail(s, m); err != nil {
		return err
	}

	if spam, err := s.HasReportedSpam(email.To[0]); err != nil {
		return err
	} else if spam {
		return nil
	}

	if s.config.JustPrint {
		s.config.Logger.Write(printEmail(email))
	} else if err := s.sgClient.Send(email); err != nil {
		return err
	}

	if _, err := s.dbmap.Exec("update message set status='sent' where subscriber_id=? and campaign_id=?",
		m.SubscriberID, m.CampaignID); err != nil {
		return fmt.Errorf("couldn't update message status: %v\n", err)
	}

	if err = s.updateCampaignStatus(m.CampaignID); err != nil {
		return fmt.Errorf("couldn't update campaign status: %v\n", err)
	}
	return nil
}

// buildEmail creates a new email in the format expected by sendgrid
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

	email.To = []string{sub.Email}
	email.ToName = []string{sub.FirstName + " " + sub.LastName}
	email.Subject = campaign.Subject
	if s.templates[m.CampaignID] == nil {
		t, err := template.New("").Parse(campaign.Body)
		if err != nil {
			return nil, err
		}
		s.templates[m.CampaignID] = t
	}
	var buf bytes.Buffer
	token, _ := s.UnsubscribeToken(sub)
	bodyStruct := struct {
		FirstName, LastName, UnsubscribeURL string
	}{sub.FirstName, sub.LastName, s.config.UnsubscribeURL + "/" + token}

	if err := s.templates[m.CampaignID].Execute(&buf, &bodyStruct); err != nil {
		return nil, err
	}
	email.HTML = buf.String()
	email.From = account.Email
	email.FromName = account.FirstName + " " + account.LastName
	return email, nil
}

// printEmail just prints an email to stderr. It is useful for
// debugging/logging
func printEmail(m *sendgrid.SGMail) []byte {
	s := fmt.Sprintln("Email to send")
	s += fmt.Sprintf("To: %s (%s)\n", m.To[0], m.ToName[0])
	s += fmt.Sprintf("From: %s (%s)\n", m.From, m.FromName)
	s += fmt.Sprintf("Subject: %s\nBody: %s\n", m.Subject, m.HTML)
	return []byte(s)
}
