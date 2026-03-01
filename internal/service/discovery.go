// Package service contains the business logic for each pipeline stage.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
)

// DiscoveryService finds unprocessed meetings from YouTube video sources.
type DiscoveryService struct {
	ytdlp *executor.YtDlpExecutor
	cfg   *config.Config
}

// NewDiscoveryService creates a new DiscoveryService.
func NewDiscoveryService(ytdlp *executor.YtDlpExecutor, cfg *config.Config) *DiscoveryService {
	return &DiscoveryService{ytdlp: ytdlp, cfg: cfg}
}

// DiscoverNewMeetings finds all unprocessed meetings for a body.
// It lists the playlist, parses dates from titles, assigns sequence numbers
// for same-date disambiguation, and filters out already-processed meetings.
func (s *DiscoveryService) DiscoverNewMeetings(ctx context.Context, body domain.Body) ([]domain.Meeting, error) {
	yearFilter := fmt.Sprintf("%d", time.Now().Year())

	slog.Info("discovering new videos",
		"body", body.Slug,
		"source", body.DiscoveryURL(),
		"year_filter", yearFilter,
	)

	entries, err := s.ytdlp.ListPlaylist(ctx, body.DiscoveryURL(), yearFilter)
	if err != nil {
		return nil, fmt.Errorf("listing videos for %s: %w", body.Slug, err)
	}

	slog.Info("found videos in playlist",
		"body", body.Slug,
		"count", len(entries),
	)

	dateRegex, err := regexp.Compile(body.TitleDateRegex)
	if err != nil {
		return nil, fmt.Errorf("compiling date regex for %s: %w", body.Slug, err)
	}

	// Phase 1: Parse all entries into meetings.
	var allParsed []domain.Meeting
	for _, entry := range entries {
		meeting, err := s.parseMeeting(entry, body, dateRegex)
		if err != nil {
			slog.Warn("skipping video: unable to parse",
				"video_id", entry.VideoID,
				"title", entry.Title,
				"error", err,
			)
			continue
		}
		allParsed = append(allParsed, meeting)
	}

	// Phase 2: Assign sequence numbers for same-date disambiguation.
	assignSequences(allParsed)

	// Phase 3: Filter out already-processed meetings.
	var meetings []domain.Meeting
	for _, meeting := range allParsed {
		if s.isProcessed(meeting, body) {
			slog.Info("already processed",
				"body", body.Slug,
				"date", meeting.ISODate(),
				"sequence", meeting.Sequence,
			)
			continue
		}
		meetings = append(meetings, meeting)
	}

	slog.Info("new meetings discovered",
		"body", body.Slug,
		"count", len(meetings),
	)

	return meetings, nil
}

// parseMeeting extracts meeting metadata from a playlist entry.
func (s *DiscoveryService) parseMeeting(entry executor.PlaylistEntry, body domain.Body, dateRegex *regexp.Regexp) (domain.Meeting, error) {
	matches := dateRegex.FindStringSubmatch(entry.Title)
	if len(matches) < 2 {
		return domain.Meeting{}, fmt.Errorf("no date found in title %q", entry.Title)
	}

	dateStr := matches[1]
	meetingDate, err := parseFlexibleDate(dateStr)
	if err != nil {
		return domain.Meeting{}, fmt.Errorf("parsing date %q from title: %w", dateStr, err)
	}

	meetingType := detectMeetingType(entry.Title, body.MeetingTypes)

	return domain.Meeting{
		VideoID:     entry.VideoID,
		Title:       entry.Title,
		MeetingDate: meetingDate,
		MeetingType: meetingType,
		BodySlug:    body.Slug,
	}, nil
}

// isProcessed checks if a summary file already exists for this meeting.
func (s *DiscoveryService) isProcessed(meeting domain.Meeting, body domain.Body) bool {
	summaryPath := s.SummaryPath(meeting, body)
	_, err := os.Stat(summaryPath)
	return err == nil
}

// SummaryPath returns the expected file path for a meeting's summary.
func (s *DiscoveryService) SummaryPath(meeting domain.Meeting, body domain.Body) string {
	filename := buildFilename(meeting, body)
	return fmt.Sprintf("%s/%s/%s.md",
		s.cfg.FinalizedDir(body),
		meeting.DateFolder(),
		filename,
	)
}

// buildFilename generates the output filename from the body's pattern.
// Appends sequence suffix for same-date disambiguation (e.g., "-1", "-2").
func buildFilename(meeting domain.Meeting, body domain.Body) string {
	name := body.FilenamePattern
	name = strings.ReplaceAll(name, "{{.MeetingDate}}", meeting.ISODate())
	return name + meeting.SequenceSuffix()
}

// assignSequences sets Sequence on each meeting for same-date disambiguation.
// Solo meetings get Sequence=0. Multiple meetings on the same date are sorted
// by VideoID for determinism and assigned 1, 2, 3, etc.
func assignSequences(meetings []domain.Meeting) {
	// Group by ISO date.
	groups := make(map[string][]int) // date -> indices
	for i := range meetings {
		date := meetings[i].ISODate()
		groups[date] = append(groups[date], i)
	}

	for _, indices := range groups {
		if len(indices) == 1 {
			meetings[indices[0]].Sequence = 0
			continue
		}

		// Sort indices by VideoID for deterministic ordering.
		sort.Slice(indices, func(a, b int) bool {
			return meetings[indices[a]].VideoID < meetings[indices[b]].VideoID
		})

		for seq, idx := range indices {
			meetings[idx].Sequence = seq + 1
		}
	}
}

// parseFlexibleDate tries multiple date formats.
func parseFlexibleDate(s string) (time.Time, error) {
	// Clean ordinal suffixes.
	cleaned := s
	for _, suffix := range []string{"st,", "nd,", "rd,", "th,"} {
		cleaned = strings.Replace(cleaned, suffix, ",", 1)
	}
	for _, suffix := range []string{"st ", "nd ", "rd ", "th "} {
		cleaned = strings.Replace(cleaned, suffix, " ", 1)
	}
	cleaned = strings.TrimSpace(cleaned)

	formats := []string{
		"2006-01-02",
		"January 2, 2006",
		"January 2 2006",
		"January 02, 2006",
		"January 02 2006",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, cleaned); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %q", s)
}

// detectMeetingType determines the meeting type from the video title.
func detectMeetingType(title string, configuredTypes []string) string {
	titleLower := strings.ToLower(title)
	for _, mt := range configuredTypes {
		if strings.Contains(titleLower, strings.ToLower(mt)) {
			return mt
		}
	}

	// Common fallback patterns.
	switch {
	case strings.Contains(titleLower, "work session"):
		return "Work Session"
	case strings.Contains(titleLower, "special"):
		return "Special Meeting"
	case strings.Contains(titleLower, "evening"):
		return "Evening Meeting"
	default:
		return "Regular Session"
	}
}
