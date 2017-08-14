package maillist

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"strings"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Message is a single email. It keeps track of whether the message has been
// sent or not.
type Message struct {
	SubscriberID int64  `db:"subscriber_id" validate:"required"`
	CampaignID   int64  `db:"campaign_id" validate:"required"`
	Status       string `db:"status" validate:"eq=pending|eq=sent|eq=failed|eq=cancelled"`
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

	if err == sql.ErrNoRows {
		return nil, ErrNotFound

	} else if err != nil {
		return nil, err
	}
	return &m, nil
}

// sendMessage sends a single message to it's destination
func (s *Session) sendMessage(m *Message) error {
	var email *mail.SGMailV3
	var err error
	var spam bool

	if email, err = buildEmail(s, m); err != nil {
		return err
	}

	if spam, err = s.HasReportedSpam(email.Personalizations[0].To[0].Address); err != nil {
		return err

	} else if spam {
		return nil
	}

	if s.config.JustPrint {
		s.info(string(printEmail(email)))

	} else if err = s.send(email); err != nil {
		return err
	}

	if _, err = s.dbmap.Exec("update message set status='sent' where subscriber_id=? and campaign_id=?",
		m.SubscriberID, m.CampaignID); err != nil {
		return fmt.Errorf("couldn't update message status: %v", err)
	}

	if err = s.updateCampaignStatus(m.CampaignID); err != nil {
		return fmt.Errorf("couldn't update campaign status: %v", err)
	}
	return nil
}

func (s *Session) send(m *mail.SGMailV3) error {
	request := sendgrid.GetRequest(s.config.SendGridAPIKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)
	response, err := sendgrid.API(request)
	if err != nil {
		s.error(err)
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		err := errors.New(response.Body)
		s.error(err)
		return err
	}

	return nil
}

// buildEmail creates a new email in the format expected by sendgrid
func buildEmail(s *Session, m *Message) (*mail.SGMailV3, error) {
	sub, err := s.GetSubscriber(m.SubscriberID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get subscriber: %v", err)
	}

	campaign, err := s.GetCampaign(m.CampaignID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get campaign %d: %v", m.CampaignID, err)
	}

	account, err := s.GetAccount(campaign.AccountID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get account: %v", err)
	}

	to := mail.NewEmail(sub.FirstName+" "+sub.LastName,
		sub.Email)
	subject := campaign.Subject
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
	contentType := "text/plain"
	if strings.Contains(campaign.Body, "DOCTYPE") {
		contentType = "text/html"
	}
	content := mail.NewContent(contentType, buf.String())
	from := mail.NewEmail(account.FirstName+" "+account.LastName,
		account.Email)
	return mail.NewV3MailInit(from, subject, to, content), nil
}

// printEmail just prints an email to stderr. It is useful for
// debugging/logging
func printEmail(m *mail.SGMailV3) []byte {
	s := fmt.Sprintln("Email to send")
	s += fmt.Sprintf("To: %s (%s)\n",
		m.Personalizations[0].To[0].Address,
		m.Personalizations[0].To[0].Name)
	s += fmt.Sprintf("From: %s (%s)\n", m.From.Address, m.From.Name)
	s += fmt.Sprintf("Subject: %s\nBody: %s\n", m.Subject, m.Content[0].Value)
	return []byte(s)
}
