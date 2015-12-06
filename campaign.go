package maillist

import "fmt"

type Campaign struct {
	ID        int64  `db:"id"`
	AccountID int64  `db:"account_id" validate:"required"`
	Subject   string `db:"subject" validate:"required"`
	Body      string `db:"body" validate:"required"`
	Status    string `db:"status" validate:"eq=pending|eq=sent|eq=cancelled|eq=failed"`
}

func (s *Session) SendCampaign(campaign *Campaign, lists ...*List) error {
	if len(lists) != 1 {
		return fmt.Errorf("multiple lists not implemented")
	}

	l := lists[0]

	if l.AccountID != campaign.AccountID {
		return fmt.Errorf("List account ID doesn't match accountID")
	}

	subs, err := s.GetSubscribers(l.ID)
	if err != nil {
		return err
	}

	campaign.Status = "pending"
	err = s.insert(campaign)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		m := Message{
			SubscriberID: sub.ID,
			CampaignID:   campaign.ID,
			Status:       "pending",
		}

		err = s.InsertMessage(&m)
		if err != nil {
			break
		}
	}

	s.messages <- "wake"

	return err
}

func (s *Session) GetCampaign(campaignID int64) (*Campaign, error) {
	var c Campaign
	err := s.selectOne(&c, "id", campaignID)
	return &c, err
}

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
