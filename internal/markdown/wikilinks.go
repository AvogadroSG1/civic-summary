package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

		// Check if the target file exists (including sequence-suffixed variants).
		dateFolder := parsed.Format("20060102")
		resolved, ok := resolveTarget(cfg.BaseDir, dateFolder, targetName)
		if !ok {
			return match
		}

		// Create wikilink preserving original text as display name.
		return fmt.Sprintf("[[%s|%s]]", resolved, match)
	})
}

// resolveTarget finds a summary file in the date folder, handling sequence suffixes.
// It first checks for an exact match (solo meeting, Sequence=0), then falls back
// to globbing for sequenced files (e.g., "Name-1.md", "Name-2.md").
// Returns the resolved filename (without .md) and true if found.
func resolveTarget(baseDir, dateFolder, targetName string) (string, bool) {
	folderPath := filepath.Join(baseDir, dateFolder)

	// Check exact match first (solo meeting).
	exactPath := filepath.Join(folderPath, targetName+".md")
	if _, err := os.Stat(exactPath); err == nil {
		return targetName, true
	}

	// Fall back to glob for sequenced files.
	pattern := filepath.Join(folderPath, targetName+"-*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", false
	}

	// Sort alphabetically so "-1" comes before "-2".
	sort.Strings(matches)
	resolved := strings.TrimSuffix(filepath.Base(matches[0]), ".md")
	return resolved, true
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
