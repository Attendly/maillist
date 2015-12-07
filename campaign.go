package maillist

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
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
	ListIDs   string    `db:"list_ids" validate:"-"`
	EventIDs  string    `db:"event_ids" validate:"-"`
}

// SendCampaign sends an email to everyone in the provided lists. Duplicate
// addresses are ignored
func (s *Session) InsertCampaign(c *Campaign, listIDs []int64, eventIDs []int64) error {
	if c.ListIDs != "" || c.EventIDs != "" {
		return errors.New("Events and Mailing-lists should be passed in InsertCampaign's parameters, not as part of the structure")
	}

	c.ListIDs = intsToString(listIDs)
	c.EventIDs = intsToString(eventIDs)

	// if l.AccountID != c.AccountID {
	// return fmt.Errorf("List account ID doesn't match accountID")
	// }

	c.Status = "scheduled"
	err := s.insert(c)
	if err != nil {
		return err
	}

	s.wake <- true
	return nil
}

// sendCampaign takes a scheduled campaign and adds it's messages to the queue
// of pending messages
func (s *Session) sendCampaign(campaignID int64) error {

	if r, err := s.dbmap.Exec("update campaign set status='pending' where status='scheduled' and id=?",
		campaignID); err != nil {
		return err
	} else if r2, err := r.RowsAffected(); r2 != 1 {
		return err
	}

	var c Campaign
	if err := s.selectOne(&c, "id", campaignID); err != nil {
		log.Printf("%+v\n", err)
		return err
	}

	listIDs := stringToInts(c.ListIDs)
	eventIDs := stringToInts(c.EventIDs)

	subs2 := make(map[string]*Subscriber)

	for _, eventID := range eventIDs {
		subs := s.config.GetAttendeesCallback(eventID)
		for _, sub := range subs {
			if subs2[sub.Email] == nil {
				err := s.GetOrInsertSubscriber(sub)
				if err != nil {
					return err
				}
				subs2[sub.Email] = sub
			}
		}
	}

	for _, listID := range listIDs {
		subs, err := s.GetSubscribers(listID)
		if err != nil {
			log.Printf("%+v\n", err)
			return err
		}
		for _, sub := range subs {
			if subs2[sub.Email] == nil {
				subs2[sub.Email] = sub
			}
		}
	}

	var err error
	for _, sub := range subs2 {
		m := Message{
			SubscriberID: sub.ID,
			CampaignID:   campaignID,
			Status:       "pending",
		}

		if err = s.InsertMessage(&m); err != nil {
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

func intsToString(xs []int64) string {
	ss := make([]string, len(xs))
	for i := range xs {
		ss[i] = strconv.FormatInt(xs[i], 10)
	}
	return strings.Join(ss, " ")
}

func stringToInts(s string) []int64 {
	ss := strings.Fields(s)
	xs := make([]int64, len(ss))
	for i := range ss {
		xs[i], _ = strconv.ParseInt(ss[i], 10, 64)
	}
	return xs
}
