package maillist

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Campaign is a message template sent at a particular time to one or more
// mailing lists
type Campaign struct {
	ID         int64  `db:"id"`
	AccountID  int64  `db:"account_id" validate:"required"`
	Subject    string `db:"subject" validate:"required"`
	Body       string `db:"body" validate:"required"`
	Address    string `db:"address" validate:"required"`
	Status     string `db:"status" validate:"eq=scheduled|eq=pending|eq=sent|eq=cancelled|eq=failed|eq=draft"`
	ListIDs    string `db:"list_ids" validate:"-"`
	EventIDs   string `db:"event_ids" validate:"-"`
	Scheduled  int64  `db:"scheduled" validate:"required"`
	CreateTime int64  `db:"create_time" validate:"required"`
}

// InsertCampaign adds the campaign to the scheduler to be sent to all its
// subscribers
func (s *Session) InsertCampaign(c *Campaign, listIDs []int64, eventIDs []int64) error {
	if c.ListIDs != "" || c.EventIDs != "" {
		return errors.New("Events and Mailing-lists should be passed in InsertCampaign's parameters, not as part of the structure")
	}

	if len(listIDs) == 0 && len(eventIDs) == 0 {
		return fmt.Errorf(
			"not scheduling campaign '%s' without attached mailing lists or events",
			c.Subject)
	}

	for _, id := range listIDs {
		list, err := s.GetList(id)
		if err != nil {
			return err
		}

		if list == nil {
			return fmt.Errorf("could not find list with ID '%d'", id)
		}

		if list.AccountID != c.AccountID {
			return fmt.Errorf("campaign with account ID '%d' contains list '%d' with account ID '%d'", c.AccountID, list.ID, list.AccountID)
		}
	}

	c.ListIDs = intsToString(listIDs)
	c.EventIDs = intsToString(eventIDs)

	if c.Status == "" {
		c.Status = "scheduled"
	}

	if c.Status != "scheduled" && c.Status != "draft" {
		return fmt.Errorf("campaign status must be 'scheduled' or 'draft'")
	}

	err := s.insert(c)
	if err != nil {
		return err
	}

	s.wake <- true
	return nil
}

// GetCampaignsInAccount returns the campaigns for the given account
func (s *Session) GetCampaignsInAccount(accountID int64) ([]*Campaign, error) {

	selectSQL := fmt.Sprintf(`
SELECT %s
	FROM campaign

WHERE account_id=?`,
		s.selectString(Campaign{}))

	var cs []*Campaign
	if _, err := s.dbmap.Select(&cs, selectSQL, accountID); err != nil {
		return nil, err

	} else if len(cs) == 0 {
		return nil, ErrNotFound
	}

	return cs, nil
}

// CancelCampaign will cancel the given campaign from sending
func (s *Session) CancelCampaign(campaignID int64) error {
	campaignSQL := `
UPDATE campaign
	SET status='deleted'

WHERE id=?`

	if _, err := s.dbmap.Exec(campaignSQL, campaignID); err != nil {
		return err
	}

	messageSQL := `
UPDATE message
	SET status='cancelled'

WHERE status='pending'
	AND campaign_id=?`
	if _, err := s.dbmap.Exec(messageSQL, campaignID); err != nil {
		return err
	}

	return nil
}

// GetCampaign retrieves a campaign with a given ID
func (s *Session) GetCampaign(campaignID int64) (*Campaign, error) {

	selectSQL := fmt.Sprintf(`
SELECT %s
	FROM campaign

WHERE id=?
	AND status!='deleted'`,
		s.selectString(Campaign{}))

	var c Campaign
	if err := s.dbmap.SelectOne(&c, selectSQL, campaignID); err == sql.ErrNoRows {
		return nil, ErrNotFound

	} else if err != nil {
		return nil, err
	}
	return &c, nil
}

// sendCampaign takes a scheduled campaign and adds it's messages to the queue
// of pending messages
func (s *Session) sendCampaign(campaignID int64) error {

	updateSQL := `
UPDATE campaign
	SET status='pending'

WHERE status='scheduled'
	AND id=?`

	if r, err := s.dbmap.Exec(updateSQL, campaignID); err != nil {
		return err

	} else if r2, err := r.RowsAffected(); r2 != 1 {
		return err
	}

	c, err := s.GetCampaign(campaignID)
	if err != nil {
		return err
	}

	listIDs := stringToInts(c.ListIDs)
	eventIDs := stringToInts(c.EventIDs)

	subsToSend := make(map[string]*Subscriber)

	// Add all the subscribers campaign events to subsToSend
	for _, eventID := range eventIDs {
		var sub2 *Subscriber
		subs := s.config.GetAttendeesCallback(eventID)
		for _, sub := range subs {
			if subsToSend[sub.Email] != nil {
				continue
			}

			sub2, err = s.GetSubscriberByEmail(sub.Email, c.AccountID)
			if err == ErrNotFound {
				sub.AccountID = c.AccountID
				if err = s.InsertSubscriber(sub); err != nil {
					return err
				}
				subsToSend[sub.Email] = sub
				continue

			} else if err != nil {
				return err
			}

			subsToSend[sub.Email] = sub2
		}
	}

	// Add all the subscribers in the campaign lists to subsToSend
	for _, listID := range listIDs {
		var subs []*Subscriber
		subs, err = s.GetSubscribers(listID)
		if err != nil && err != ErrNotFound {
			return err
		}
		for _, sub := range subs {
			if subsToSend[sub.Email] == nil {
				subsToSend[sub.Email] = sub
			}
		}
	}

	for _, sub := range subsToSend {
		if sub.Status != statusActive {
			continue
		}

		m := Message{
			SubscriberID: sub.ID,
			CampaignID:   campaignID,
			Status:       "pending",
		}

		if err = s.InsertMessage(&m); err != nil {
			break
		}
	}

	return err
}

// getDueCampaign retrieves a campaign that is due to be sent. It returns
// nil,nil if none are due
func getDueCampaign(s *Session) (*Campaign, error) {
	var c Campaign
	selectSQL := fmt.Sprintf(`
SELECT %s
	FROM campaign

WHERE status='scheduled'
	AND scheduled<=?

LIMIT 1`,
		s.selectString(&c))

	err := s.dbmap.SelectOne(&c, selectSQL, time.Now().Unix())
	if err == sql.ErrNoRows {
		return nil, ErrNotFound

	} else if err != nil {
		return nil, err
	}

	return &c, nil
}

// UpdateCampaignStatus checks if all a campaigns messages have been sent, and
// updates status from `pending` to `sent`.
func (s *Session) updateCampaignStatus(campaignID int64) error {

	selectSQL := `
SELECT count(*)
	FROM message

WHERE status='pending'
	AND campaign_id=?`

	if count, err := s.dbmap.SelectInt(selectSQL, campaignID); err != nil {
		return err

	} else if count > 0 {
		return nil
	}

	updateSQL := `
UPDATE campaign
	SET status='sent'

WHERE id=?`

	_, err := s.dbmap.Exec(updateSQL, campaignID)
	return err
}

// intsToString creates a space delimitted string of integers of a list
func intsToString(xs []int64) string {
	ss := make([]string, len(xs))
	for i := range xs {
		ss[i] = strconv.FormatInt(xs[i], 10)
	}
	return strings.Join(ss, " ")
}

// stringToInts creates a list of integers from a space-delimitted string
func stringToInts(s string) []int64 {
	ss := strings.Fields(s)
	xs := make([]int64, len(ss))
	for i := range ss {
		xs[i], _ = strconv.ParseInt(ss[i], 10, 64)
	}
	return xs
}
