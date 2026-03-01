package domain

import "time"

// QuarantineEntry represents a failed meeting that has been quarantined
// for later retry.
type QuarantineEntry struct {
	VideoID       string    `json:"video_id"`
	MeetingDate   string    `json:"meeting_date"`
	BodySlug      string    `json:"body_slug"`
	Sequence      int       `json:"sequence"`
	Error         string    `json:"error"`
	QuarantinedAt time.Time `json:"quarantined_at"`
	RetryCount    int       `json:"retry_count"`
}

// QuarantineManifest is the master index of all quarantined items.
type QuarantineManifest struct {
	Items   map[string]QuarantineManifestEntry `json:"items"`
	Created time.Time                          `json:"created"`
}

// QuarantineManifestEntry is a lightweight entry in the manifest.
type QuarantineManifestEntry struct {
	MeetingDate string    `json:"meeting_date"`
	BodySlug    string    `json:"body_slug"`
	Sequence    int       `json:"sequence"`
	Added       time.Time `json:"added"`
}

// NewQuarantineManifest creates an empty manifest.
func NewQuarantineManifest() *QuarantineManifest {
	return &QuarantineManifest{
		Items:   make(map[string]QuarantineManifestEntry),
		Created: time.Now(),
	}
}
