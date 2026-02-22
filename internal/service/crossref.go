package service

import (
	"log/slog"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/markdown"
)

// CrossReferenceService adds Obsidian wikilinks to meeting summaries
// by finding date references that match other processed meetings.
type CrossReferenceService struct {
	cfg *config.Config
}

// NewCrossReferenceService creates a new CrossReferenceService.
func NewCrossReferenceService(cfg *config.Config) *CrossReferenceService {
	return &CrossReferenceService{cfg: cfg}
}

// AddCrossReferences scans content for date references and adds wikilinks
// to matching summaries. Returns the updated content.
func (s *CrossReferenceService) AddCrossReferences(content string, meeting domain.Meeting, body domain.Body) string {
	baseDir := s.cfg.FinalizedDir(body)

	// Build the filename pattern for wikilink targets.
	// Strip the {{.MeetingDate}} portion to create a template with {{.ISODate}}.
	filenamePattern := body.FilenamePattern

	wikicfg := markdown.CrossReferenceConfig{
		BaseDir:         baseDir,
		FilenamePattern: filenamePattern,
		SelfDate:        meeting.MeetingDate,
	}

	result := markdown.AddWikilinks(content, wikicfg)

	slog.Info("cross-references added",
		"body", body.Slug,
		"meeting_date", meeting.ISODate(),
	)

	return result
}
