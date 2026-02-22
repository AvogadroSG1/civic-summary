package service_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func crossrefConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		OutputDir: t.TempDir(),
		Bodies: map[string]domain.Body{
			"hagerstown": {
				Slug:            "hagerstown",
				Name:            "Hagerstown City Council",
				PlaylistID:      "TEST",
				OutputSubdir:    "Hagerstown Town Council - Citizen Summary",
				FilenamePattern: "Hagerstown-City-Council-{{.MeetingDate}}-Citizen-Summary",
				TitleDateRegex:  `^([A-Z][a-z]+ \d{1,2},? \d{4})`,
				Tags:            []string{"City-Council"},
				PromptTemplate:  "test.tmpl",
			},
		},
	}
}

func TestCrossReferenceService_AddCrossReferences(t *testing.T) {
	cfg := crossrefConfig(t)
	body, _ := cfg.GetBody("hagerstown")

	// Create a finalized summary for January 7, 2025 that we can link to.
	targetDir := filepath.Join(cfg.FinalizedDir(body), "20250107")
	require.NoError(t, os.MkdirAll(targetDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(targetDir, "Hagerstown-City-Council-2025-01-07-Citizen-Summary.md"),
		[]byte("# Existing Summary"), 0o644,
	))

	svc := service.NewCrossReferenceService(cfg)

	meeting := domain.Meeting{
		VideoID:     "abc123",
		MeetingDate: time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
		BodySlug:    "hagerstown",
	}

	content := "During the meeting, the council referenced the January 7, 2025 session."

	result := svc.AddCrossReferences(content, meeting, body)

	// The date "January 7, 2025" should become a wikilink.
	assert.Contains(t, result, "[[Hagerstown-City-Council-2025-01-07-Citizen-Summary|January 7, 2025]]")
}

func TestCrossReferenceService_NoMatchingDates(t *testing.T) {
	cfg := crossrefConfig(t)
	body, _ := cfg.GetBody("hagerstown")

	svc := service.NewCrossReferenceService(cfg)

	meeting := domain.Meeting{
		VideoID:     "abc123",
		MeetingDate: time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
		BodySlug:    "hagerstown",
	}

	// Content with a date, but no matching finalized file on disk.
	content := "The council discussed the March 15, 2025 budget deadline."

	result := svc.AddCrossReferences(content, meeting, body)

	// No wikilinks should be added since no matching file exists.
	assert.Equal(t, content, result)
	assert.NotContains(t, result, "[[")
}

func TestCrossReferenceService_SelfDateExcluded(t *testing.T) {
	cfg := crossrefConfig(t)
	body, _ := cfg.GetBody("hagerstown")

	meetingDate := time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC)

	// Create a finalized summary for Feb 4, 2025 (the meeting's own date).
	selfDir := filepath.Join(cfg.FinalizedDir(body), "20250204")
	require.NoError(t, os.MkdirAll(selfDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(selfDir, "Hagerstown-City-Council-2025-02-04-Citizen-Summary.md"),
		[]byte("# This Meeting"), 0o644,
	))

	svc := service.NewCrossReferenceService(cfg)

	meeting := domain.Meeting{
		VideoID:     "abc123",
		MeetingDate: meetingDate,
		BodySlug:    "hagerstown",
	}

	content := "This February 4, 2025 meeting covered important topics."

	result := svc.AddCrossReferences(content, meeting, body)

	// The meeting's own date should NOT be linked (self-link prevention).
	assert.NotContains(t, result, "[[")
	assert.Contains(t, result, "February 4, 2025")
}
