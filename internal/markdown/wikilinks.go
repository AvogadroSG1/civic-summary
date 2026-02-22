package markdown

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// datePatterns matches various date formats found in meeting summaries.
// Examples: "September 16, 2025", "September 16th, 2025", "August 8, 2025"
var datePattern = regexp.MustCompile(`([A-Z][a-z]+ \d{1,2}(?:st|nd|rd|th)?,? \d{4})`)

// CrossReferenceConfig holds the settings for cross-reference wikilink generation.
type CrossReferenceConfig struct {
	// BaseDir is the directory containing finalized meeting summaries.
	BaseDir string
	// FilenamePattern is a Go template string for the target filename.
	// Example: "Hagerstown-City-Council-{{.ISODate}}-Citizen-Summary"
	FilenamePattern string
	// SelfDate is the meeting date of the current document (to avoid self-links).
	SelfDate time.Time
}

// AddWikilinks scans content for date references and converts them to
// Obsidian wikilinks if a corresponding summary file exists.
func AddWikilinks(content string, cfg CrossReferenceConfig) string {
	return datePattern.ReplaceAllStringFunc(content, func(match string) string {
		// Clean ordinal suffixes for parsing.
		cleaned := match
		for _, suffix := range []string{"st,", "nd,", "rd,", "th,"} {
			cleaned = strings.Replace(cleaned, suffix, ",", 1)
		}
		for _, suffix := range []string{"st ", "nd ", "rd ", "th "} {
			cleaned = strings.Replace(cleaned, suffix, " ", 1)
		}

		// Parse the date.
		parsed, err := parseHumanDate(cleaned)
		if err != nil {
			return match
		}

		// Skip self-links.
		if parsed.Equal(cfg.SelfDate) {
			return match
		}

		// Build the target filename.
		isoDate := parsed.Format("2006-01-02")
		targetName := strings.ReplaceAll(cfg.FilenamePattern, "{{.ISODate}}", isoDate)
		targetName = strings.ReplaceAll(targetName, "{{.MeetingDate}}", isoDate)

		// Check if the target file exists.
		dateFolder := parsed.Format("20060102")
		targetPath := fmt.Sprintf("%s/%s/%s.md", cfg.BaseDir, dateFolder, targetName)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return match
		}

		// Create wikilink preserving original text as display name.
		return fmt.Sprintf("[[%s|%s]]", targetName, match)
	})
}

// parseHumanDate attempts to parse a human-readable date string.
func parseHumanDate(s string) (time.Time, error) {
	formats := []string{
		"January 2, 2006",
		"January 2 2006",
		"January 02, 2006",
		"January 02 2006",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %q", s)
}
