package maillist

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"
)

// Campaign is a message template sent at a particular time to one or more
// mailing lists
type Campaign struct {
	ID        int64     `db:"id"`
	AccountID int64     `db:"account_id" validate:"required"`
	Subject   string    `db:"subject" validate:"required"`
	Body      string    `db:"body" validate:"required"`
	Status    string    `db:"status" validate:"eq=scheduled|eq=pending|eq=sent|eq=cancelled|eq=failed"`
	Scheduled time.Time `db:"scheduled" validate:"required"`
	Lists     string    `db:"lists" validate:"-"`
}

// SendCampaign sends an email to everyone in the provided lists. Duplicate
// addresses are ignored
func (s *Session) InsertCampaign(campaign *Campaign, lists ...*List) error {
	if len(lists) != 1 {
		return fmt.Errorf("multiple lists not implemented")
	}

	l := lists[0]

	if l.AccountID != campaign.AccountID {
		return fmt.Errorf("List account ID doesn't match accountID")
	}

	campaign.Status = "scheduled"
	campaign.Lists = strconv.FormatInt(l.ID, 10)
	err := s.insert(campaign)
	if err != nil {
		return err
	}

	s.wake <- true
	return nil
}

// sendCampaign takes a scheduled campaign and adds it's messages to the queue
// of pending messages
func (s *Session) sendCampaign(campaignID int64) error {
	var c Campaign
	if err := s.selectOne(&c, "id", campaignID); err != nil {
		log.Printf("%+v\n", err)
		return err
	}
	if c.Status != "scheduled" {
		return nil
	}

	c.Status = "pending"
	err := s.update(&c)
	if err != nil {
		log.Printf("%+v\n", err)
		return err
	}

	listID, _ := strconv.ParseInt(c.Lists, 10, 64)

	subs, err := s.GetSubscribers(listID)
	if err != nil {
		log.Printf("%+v\n", err)
		return err
	}

	for _, sub := range subs {
		m := Message{
			SubscriberID: sub.ID,
			CampaignID:   campaignID,
			Status:       "pending",
		}

		err = s.InsertMessage(&m)
		if err != nil {
			log.Printf("%+v\n", err)
			break
		}
	}

	return err
}

// getDueCampaign retrieves a campaign that is due to be sent. It returns
// nil,nil if none are due
func getDueCampaign(s *Session) (*Campaign, error) {
	var c Campaign
	query := fmt.Sprintf(
		`select %s from campaign where status='scheduled' and scheduled<=? limit 1`,
		s.selectString(&c))

	err := s.dbmap.SelectOne(&c, query, time.Now())
	if err == nil {
		return &c, nil
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return nil, err
}

// GetCampaign retrieves a campaign with a given ID
func (s *Session) GetCampaign(campaignID int64) (*Campaign, error) {
	var c Campaign
	err := s.selectOne(&c, "id", campaignID)
	return &c, err
}

// UpdateCampaignStatus checks if all a campaigns messages have been sent, and
// updates status from `pending` to `sent`.
func (s *Session) updateCampaignStatus(campaignID int64) error {
	count, err := s.dbmap.SelectInt("select count(*) from message where status='pending' and campaign_id=?", campaignID)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err = s.dbmap.Exec("update campaign set status='sent' where id=?",
		campaignID)
	return err
}
