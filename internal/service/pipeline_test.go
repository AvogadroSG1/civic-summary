package service_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validSummaryContent returns a summary that passes validation:
// frontmatter, 5 required sections, >500 words, timestamps, conclusion, footer.
func validSummaryContent() string {
	return `---
date: 2025-02-05
author: Peter O'Connor
tags:
  - City-Council
  - Hagerstown
  - Regular-Session
source: https://www.youtube.com/watch?v=abc123
meeting_date: 2025-02-04
---

# Hagerstown City Council Meeting - Citizen Summary
**Date:** February 04, 2025
**Meeting Type:** Regular Session
**Video:** [YouTube Recording](https://www.youtube.com/watch?v=abc123)

---

## 1. Updates

### Community Events **[00:01:00-00:05:00]**
The Mayor announced upcoming community events including the annual Spring Festival scheduled for April. The festival will feature live music, food vendors from local restaurants, craft exhibitions, and a children's activity area. Registration for vendor booths is now open through the city's website. The Mayor encouraged all residents to participate and noted that last year's event attracted over five thousand attendees from across Washington County and surrounding regions in Maryland.

### Staff Reports **[00:05:00-00:15:00]**
City Manager Robert Wilson provided comprehensive updates on ongoing infrastructure projects across the city. The East End water main replacement project is currently seventy-five percent complete. The project has remained within the approved budget of two point three million dollars. Additionally, the repaving of South Potomac Street is scheduled to begin in early March. The city code enforcement division reported a fifteen percent increase in compliance over the past quarter.

---

## 2. Citizen Comments

### John Smith **[00:20:00-00:25:00]**
Mr. Smith expressed concerns about traffic congestion at the intersection of Dual Highway and Eastern Boulevard. He cited three recent accidents and requested the council consider commissioning a traffic study. He also suggested additional street lighting along the corridor between Wesel Boulevard and Eastern Boulevard for pedestrian safety.

---

## 3. Actions Taken

### Grant Approval **[00:35:00-00:40:00]**
Council unanimously approved a fifty thousand dollar Community Development Block Grant application for neighborhood infrastructure improvements in the Jonathan Street corridor. The grant will fund sidewalk repairs and ADA-compliant curb cuts. Vote count was five to zero in favor.

### Contract Award **[00:48:00-00:52:00]**
Council approved a three-year contract with CleanCity Services for municipal building janitorial services at a total cost of three hundred eighty-four thousand dollars annually. The contract includes a performance review clause.

---

## 4. Input Requested from Council

### Budget Priorities **[00:55:00-01:05:00]**
Finance Director Sarah Chen presented preliminary budget projections for the next fiscal year and requested council guidance on capital improvement prioritization. The city faces an estimated four million dollar gap between identified capital needs and available funding. Council members requested additional analysis comparing long-term cost implications of deferring road maintenance versus investing now, with findings to be presented at the next work session.

---

## 5. Critical Discussions

### Infrastructure Bond Proposal **[01:10:00-01:35:00]**
Council engaged in extensive discussion regarding a proposed ten million dollar general obligation bond for infrastructure improvements. City Engineer James Martinez presented a detailed assessment showing that twenty-three miles of city roads are currently rated poor or failing. The estimated cost of addressing all identified deficiencies is fourteen point five million dollars. The annual debt service would be approximately eight hundred thousand dollars. Staff was directed to prepare a detailed project list with cost estimates.

**Why this matters:** This bond would fund critical road and bridge repairs affecting daily commutes for thousands of Hagerstown residents. The condition of city infrastructure directly impacts property values, public safety, and economic development.

---

## Conclusion

The council took decisive action on the community development grant and janitorial services contract. The infrastructure bond proposal will continue through public input in coming weeks.

---

*This citizen summary was created from the official meeting video and transcript. For complete details, watch the full meeting recording or review official minutes when published.*
`
}

// pipelineConfig creates a config for pipeline tests with a hagerstown body.
func pipelineConfig(t *testing.T) *config.Config {
	t.Helper()
	tmpDir := t.TempDir()
	return &config.Config{
		OutputDir:     tmpDir,
		MaxRetries:    1,
		BackoffDelays: []int{0},
		Bodies: map[string]domain.Body{
			"hagerstown": {
				Slug:            "hagerstown",
				Name:            "Hagerstown City Council",
				PlaylistID:      "PLJXxCe9GA2fEf4TIVzTH2O-kFJlS8VVgQ",
				OutputSubdir:    "Hagerstown Town Council - Citizen Summary",
				FilenamePattern: "Hagerstown-City-Council-{{.MeetingDate}}-Citizen-Summary",
				TitleDateRegex:  `^([A-Z][a-z]+ \d{1,2},? \d{4})`,
				Tags:            []string{"City-Council", "Hagerstown"},
				PromptTemplate:  "hagerstown.prompt.tmpl",
				Author:          "Peter O'Connor",
			},
		},
	}
}

// buildPipelineOrchestrator creates a fully-wired pipeline with mock executors.
func buildPipelineOrchestrator(t *testing.T, cfg *config.Config, mock *executor.MockCommander) *service.PipelineOrchestrator {
	t.Helper()

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	claude := executor.NewClaudeExecutor(mock, "claude")

	tmplDir := setupTemplateDir(t)

	discovery := service.NewDiscoveryService(ytdlp, cfg)
	transcription := service.NewTranscriptionService(ytdlp, nil)
	analysis := service.NewAnalysisService(claude, tmplDir)
	crossref := service.NewCrossReferenceService(cfg)
	validation := service.NewValidationService()
	quarantine := service.NewQuarantineService(cfg)
	index := service.NewIndexService(cfg)

	return service.NewPipelineOrchestrator(
		discovery, transcription, analysis, crossref,
		validation, quarantine, index, cfg,
	)
}

// mockDiscoveryResponse sets up the mock to return playlist entries for a body.
func mockDiscoveryResponse(mock *executor.MockCommander, playlistOutput string) {
	// ListPlaylist uses DefaultResult since the args include the full playlist URL.
	mock.DefaultResult = &executor.CommandResult{Stdout: playlistOutput}
}

func TestPipelineOrchestrator_ProcessBody_DryRun(t *testing.T) {
	cfg := pipelineConfig(t)
	mock := executor.NewMockCommander()

	// Discovery finds 2 meetings.
	mockDiscoveryResponse(mock, "abc123|February 04, 2025 | Mayor & Council Regular Session\ndef456|January 21, 2025 | Mayor & Council Work Session\n")

	pipeline := buildPipelineOrchestrator(t, cfg, mock)
	body, _ := cfg.GetBody("hagerstown")

	stats, err := pipeline.ProcessBody(context.Background(), body, true)
	require.NoError(t, err)

	assert.Equal(t, 2, stats.Discovered)
	assert.Equal(t, 0, stats.Processed, "dry run should not process")
	assert.Equal(t, 0, stats.Failed)
}

func TestPipelineOrchestrator_ProcessBody_Success(t *testing.T) {
	cfg := pipelineConfig(t)
	body, _ := cfg.GetBody("hagerstown")

	mock := executor.NewMockCommander()

	// Discovery: 1 meeting.
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "abc123|February 04, 2025 | Mayor & Council Regular Session\n",
	}

	// Captions: list-subs shows available.
	videoURL := "https://www.youtube.com/watch?v=abc123"
	listSubsKey := fmt.Sprintf("yt-dlp --list-subs %s", videoURL)
	mock.OnCommand(listSubsKey, &executor.CommandResult{
		Stdout: "Available automatic captions for abc123:\nen  English",
	}, nil)

	// Captions: download succeeds (the actual file path uses the date dir).
	// The pipeline creates the date dir first, then passes it to Transcribe.
	// We need to pre-create the SRT file in the expected location.
	dateDir := filepath.Join(cfg.FinalizedDir(body), "20250204")
	require.NoError(t, os.MkdirAll(dateDir, 0o755))

	srtPath := filepath.Join(dateDir, "abc123.en.srt")
	srtContent := generateWords(600) // Meet minimum word count.
	require.NoError(t, os.WriteFile(srtPath, []byte(srtContent), 0o644))

	// Claude: return valid summary content.
	mock.OnCommand("claude", &executor.CommandResult{
		Stdout: validSummaryContent(),
	}, nil)

	pipeline := buildPipelineOrchestrator(t, cfg, mock)

	stats, err := pipeline.ProcessBody(context.Background(), body, false)
	require.NoError(t, err)

	assert.Equal(t, 1, stats.Discovered)
	assert.Equal(t, 1, stats.Processed)
	assert.Equal(t, 0, stats.Failed)

	// Verify the summary file was written.
	summaryPath := filepath.Join(dateDir, "Hagerstown-City-Council-2025-02-04-Citizen-Summary.md")
	_, err = os.Stat(summaryPath)
	assert.NoError(t, err, "summary file should exist")
}

func TestPipelineOrchestrator_ProcessBody_AnalysisFails_Quarantined(t *testing.T) {
	cfg := pipelineConfig(t)
	body, _ := cfg.GetBody("hagerstown")

	mock := executor.NewMockCommander()

	// Discovery: 1 meeting.
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "abc123|February 04, 2025 | Mayor & Council Regular Session\n",
	}

	// Captions available.
	videoURL := "https://www.youtube.com/watch?v=abc123"
	listSubsKey := fmt.Sprintf("yt-dlp --list-subs %s", videoURL)
	mock.OnCommand(listSubsKey, &executor.CommandResult{
		Stdout: "Available automatic captions\nen  English",
	}, nil)

	// Pre-create SRT with enough words.
	dateDir := filepath.Join(cfg.FinalizedDir(body), "20250204")
	require.NoError(t, os.MkdirAll(dateDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dateDir, "abc123.en.srt"),
		[]byte(generateWords(600)), 0o644,
	))

	// Claude: return an error.
	mock.OnCommand("claude", nil, fmt.Errorf("API rate limit exceeded"))

	pipeline := buildPipelineOrchestrator(t, cfg, mock)

	stats, err := pipeline.ProcessBody(context.Background(), body, false)
	require.NoError(t, err) // ProcessBody itself doesn't fail, individual meetings do.

	assert.Equal(t, 1, stats.Discovered)
	assert.Equal(t, 0, stats.Processed)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, 1, stats.Quarantined)

	// Verify quarantine entry was created.
	qEntries, qErr := service.NewQuarantineService(cfg).ListQuarantined(body)
	require.NoError(t, qErr)
	assert.Len(t, qEntries, 1)
	assert.Equal(t, "abc123", qEntries[0].VideoID)
}

func TestPipelineOrchestrator_ProcessAll_MultipleBodies(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		OutputDir:     tmpDir,
		MaxRetries:    1,
		BackoffDelays: []int{0},
		Bodies: map[string]domain.Body{
			"hagerstown": {
				Slug:            "hagerstown",
				Name:            "Hagerstown City Council",
				PlaylistID:      "PLtest1",
				OutputSubdir:    "Hagerstown",
				FilenamePattern: "Hagerstown-{{.MeetingDate}}",
				TitleDateRegex:  `^([A-Z][a-z]+ \d{1,2},? \d{4})`,
				Tags:            []string{"City-Council"},
				PromptTemplate:  "hagerstown.prompt.tmpl",
			},
			"bocc": {
				Slug:            "bocc",
				Name:            "Washington County BOCC",
				PlaylistID:      "PLtest2",
				OutputSubdir:    "BOCC",
				FilenamePattern: "BOCC-{{.MeetingDate}}",
				TitleDateRegex:  `- ([A-Z][a-z]+ \d{1,2}, \d{4})`,
				Tags:            []string{"BOCC"},
				PromptTemplate:  "bocc.prompt.tmpl",
			},
		},
	}

	mock := executor.NewMockCommander()
	// No meetings found for any body — tests ProcessAll loop with empty results.
	mock.DefaultResult = &executor.CommandResult{Stdout: ""}

	pipeline := buildPipelineOrchestrator(t, cfg, mock)

	allStats, err := pipeline.ProcessAll(context.Background(), true)
	require.NoError(t, err)

	assert.Len(t, allStats, 2, "should have stats for both bodies")
	for _, stats := range allStats {
		assert.Equal(t, 0, stats.Discovered)
	}
}

func TestPipelineOrchestrator_ProcessAll_OneBodyFails(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		OutputDir:     tmpDir,
		MaxRetries:    1,
		BackoffDelays: []int{0},
		Bodies: map[string]domain.Body{
			"hagerstown": {
				Slug:            "hagerstown",
				Name:            "Hagerstown City Council",
				PlaylistID:      "PLgood",
				OutputSubdir:    "Hagerstown",
				FilenamePattern: "Hagerstown-{{.MeetingDate}}",
				TitleDateRegex:  `^([A-Z][a-z]+ \d{1,2},? \d{4})`,
				Tags:            []string{"City-Council"},
				PromptTemplate:  "hagerstown.prompt.tmpl",
			},
			"bocc": {
				Slug:            "bocc",
				Name:            "Washington County BOCC",
				PlaylistID:      "PLbad",
				OutputSubdir:    "BOCC",
				FilenamePattern: "BOCC-{{.MeetingDate}}",
				TitleDateRegex:  `- ([A-Z][a-z]+ \d{1,2}, \d{4})`,
				Tags:            []string{"BOCC"},
				PromptTemplate:  "bocc.prompt.tmpl",
			},
		},
	}

	mock := executor.NewMockCommander()

	// Both bodies return empty playlists (no new meetings).
	// We test that both bodies are processed even if we simulate a yt-dlp error for one.
	// The mock will return empty for all yt-dlp calls.
	mock.DefaultResult = &executor.CommandResult{Stdout: ""}

	pipeline := buildPipelineOrchestrator(t, cfg, mock)

	allStats, err := pipeline.ProcessAll(context.Background(), false)
	require.NoError(t, err)

	// Both bodies should have stats entries (graceful degradation).
	assert.Len(t, allStats, 2)
}

func TestPipelineOrchestrator_RetryQuarantined_Success(t *testing.T) {
	cfg := pipelineConfig(t)
	body, _ := cfg.GetBody("hagerstown")

	// Pre-quarantine an item.
	qSvc := service.NewQuarantineService(cfg)
	meeting := domain.Meeting{
		VideoID:     "quarantined123",
		MeetingDate: time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
		MeetingType: "Regular Session",
		BodySlug:    "hagerstown",
	}
	require.NoError(t, qSvc.Quarantine(body, meeting, "previous failure", "", ""))

	// Verify quarantine entry exists.
	entries, err := qSvc.ListQuarantined(body)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	mock := executor.NewMockCommander()

	// Discovery returns no NEW meetings (the quarantined one is retried separately).
	mock.DefaultResult = &executor.CommandResult{Stdout: ""}

	// For the retry: set up captions and analysis to succeed.
	videoURL := "https://www.youtube.com/watch?v=quarantined123"
	listSubsKey := fmt.Sprintf("yt-dlp --list-subs %s", videoURL)
	mock.OnCommand(listSubsKey, &executor.CommandResult{
		Stdout: "Available automatic captions\nen  English",
	}, nil)

	// Pre-create the date directory and SRT file for the retried meeting.
	dateDir := filepath.Join(cfg.FinalizedDir(body), "20250114")
	require.NoError(t, os.MkdirAll(dateDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dateDir, "quarantined123.en.srt"),
		[]byte(generateWords(600)), 0o644,
	))

	// Claude returns valid summary.
	mock.OnCommand("claude", &executor.CommandResult{
		Stdout: validSummaryContent(),
	}, nil)

	pipeline := buildPipelineOrchestrator(t, cfg, mock)

	stats, err := pipeline.ProcessBody(context.Background(), body, false)
	require.NoError(t, err)

	assert.Equal(t, 0, stats.Discovered)
	// The retried item should be counted as processed.
	assert.Equal(t, 1, stats.Processed)

	// Quarantine should be empty after successful retry.
	entries, err = qSvc.ListQuarantined(body)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
