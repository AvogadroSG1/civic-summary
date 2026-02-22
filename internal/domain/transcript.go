package domain

import "strings"

// TranscriptSource indicates how the transcript was obtained.
type TranscriptSource string

const (
	TranscriptSourceCaptions TranscriptSource = "captions"
	TranscriptSourceWhisper  TranscriptSource = "whisper"
)

// Transcript is a value object containing the SRT-format text for a meeting.
type Transcript struct {
	Content string
	Path    string
	Source  TranscriptSource
}

// WordCount returns the number of words in the transcript content.
func (t Transcript) WordCount() int {
	if t.Content == "" {
		return 0
	}
	return len(strings.Fields(t.Content))
}

// IsEmpty returns true if the transcript has no content.
func (t Transcript) IsEmpty() bool {
	return strings.TrimSpace(t.Content) == ""
}
