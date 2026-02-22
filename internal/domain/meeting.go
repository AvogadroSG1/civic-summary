package domain

import "time"

// Meeting is the aggregate root representing a single government meeting.
// It tracks a video through the pipeline from discovery to finalized summary.
type Meeting struct {
	VideoID     string
	Title       string
	MeetingDate time.Time
	MeetingType string
	BodySlug    string
}

// DateFolder returns the folder name for this meeting (YYYYMMDD format).
func (m Meeting) DateFolder() string {
	return m.MeetingDate.Format("20060102")
}

// ISODate returns the meeting date in ISO 8601 format (YYYY-MM-DD).
func (m Meeting) ISODate() string {
	return m.MeetingDate.Format("2006-01-02")
}

// HumanDate returns the meeting date in human-readable format (January 02, 2006).
func (m Meeting) HumanDate() string {
	return m.MeetingDate.Format("January 02, 2006")
}
