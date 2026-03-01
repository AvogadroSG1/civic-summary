package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	tmpDir := t.TempDir()
	return &config.Config{
		OutputDir:     tmpDir,
		MaxRetries:    1,
		BackoffDelays: []int{1},
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

func TestDiscoveryService_DiscoverNewMeetings(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "abc123|February 04, 2025 | Mayor & Council Regular Session\ndef456|January 21, 2025 | Mayor & Council Work Session\n",
	}

	cfg := testConfig(t)
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)

	body, _ := cfg.GetBody("hagerstown")
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	assert.Len(t, meetings, 2)
	assert.Equal(t, "abc123", meetings[0].VideoID)
	assert.Equal(t, "2025-02-04", meetings[0].ISODate())
	assert.Equal(t, "hagerstown", meetings[0].BodySlug)
}

func TestDiscoveryService_SkipsProcessed(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "abc123|February 04, 2025 | Mayor & Council Regular Session\n",
	}

	cfg := testConfig(t)
	body, _ := cfg.GetBody("hagerstown")

	// Create the finalized summary file to simulate already-processed.
	summaryDir := filepath.Join(cfg.FinalizedDir(body), "20250204")
	require.NoError(t, os.MkdirAll(summaryDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(summaryDir, "Hagerstown-City-Council-2025-02-04-Citizen-Summary.md"),
		[]byte("# Existing"), 0o644,
	))

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	assert.Empty(t, meetings)
}

func TestDiscoveryService_SkipsUnparsableTitles(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "abc123|Public Hearing for Comprehensive Plan 2040\n",
	}

	cfg := testConfig(t)
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)

	body, _ := cfg.GetBody("hagerstown")
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	assert.Empty(t, meetings) // Should be skipped due to unparsable date.
}

func TestDiscoveryService_MeetingType_Detection(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "id1|February 04, 2025 | Mayor & Council Work Session\n",
	}

	cfg := testConfig(t)
	cfg.Bodies["hagerstown"] = domain.Body{
		Slug:            "hagerstown",
		Name:            "Hagerstown City Council",
		PlaylistID:      "TEST",
		OutputSubdir:    "Test",
		FilenamePattern: "Test-{{.MeetingDate}}",
		TitleDateRegex:  `^([A-Z][a-z]+ \d{1,2},? \d{4})`,
		Tags:            []string{"test"},
		PromptTemplate:  "test.tmpl",
		MeetingTypes:    []string{"Regular Session", "Work Session"},
	}

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)
	body, _ := cfg.GetBody("hagerstown")
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	assert.Len(t, meetings, 1)
	assert.Equal(t, "Work Session", meetings[0].MeetingType)
}

// boccConfig returns a config with a BOCC body for BOCC-specific tests.
func boccConfig(t *testing.T) *config.Config {
	t.Helper()
	tmpDir := t.TempDir()
	return &config.Config{
		OutputDir:     tmpDir,
		MaxRetries:    1,
		BackoffDelays: []int{1},
		Bodies: map[string]domain.Body{
			"bocc": {
				Slug:            "bocc",
				Name:            "Washington County Board of County Commissioners",
				PlaylistID:      "PL7X-j0EwreAd_6kV3IjxO-_XNwDNn0esS",
				OutputSubdir:    "Washington County BOCC - Citizen Summary",
				FilenamePattern: "BOCC-{{.MeetingDate}}-Citizen-Summary",
				TitleDateRegex:  `- ([A-Z][a-z]+ \d{1,2}, \d{4})`,
				Tags:            []string{"BOCC", "Washington-County"},
				PromptTemplate:  "bocc.prompt.tmpl",
				MeetingTypes:    []string{"Regular Meeting", "Work Session", "Public Hearing"},
				Author:          "Peter O'Connor",
			},
		},
	}
}

func TestDiscoveryService_BOCC_DateAtEnd(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "vid1|Board of County Commissioners Regular Meeting - January 7, 2025\n",
	}

	cfg := boccConfig(t)
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)

	body, _ := cfg.GetBody("bocc")
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	require.Len(t, meetings, 1)
	assert.Equal(t, "vid1", meetings[0].VideoID)
	assert.Equal(t, "2025-01-07", meetings[0].ISODate())
	assert.Equal(t, "Regular Meeting", meetings[0].MeetingType)
	assert.Equal(t, "bocc", meetings[0].BodySlug)
}

func TestDiscoveryService_BOCC_SkipsNonDateTitles(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		// These titles don't match the BOCC regex (no "- Month Day, Year" suffix).
		Stdout: "vid1|Public Hearing for Comprehensive Plan 2040\nvid2|Board Work Session Overview\n",
	}

	cfg := boccConfig(t)
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)

	body, _ := cfg.GetBody("bocc")
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	assert.Empty(t, meetings)
}

func TestDiscoveryService_SameDateDisambiguation_TwoMeetings(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "vid_b|Board of County Commissioners Regular Meeting - February 24, 2025\nvid_a|Board of County Commissioners Work Session - February 24, 2025\n",
	}

	cfg := boccConfig(t)
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)

	body, _ := cfg.GetBody("bocc")
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	require.Len(t, meetings, 2)

	// Both on same date, sorted by VideoID: vid_a=1, vid_b=2.
	byID := make(map[string]domain.Meeting)
	for _, m := range meetings {
		byID[m.VideoID] = m
	}
	assert.Equal(t, 1, byID["vid_a"].Sequence)
	assert.Equal(t, "-1", byID["vid_a"].SequenceSuffix())
	assert.Equal(t, 2, byID["vid_b"].Sequence)
	assert.Equal(t, "-2", byID["vid_b"].SequenceSuffix())
}

func TestDiscoveryService_SameDateDisambiguation_ThreeMeetings(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "vid_c|BOCC Meeting - February 24, 2025\nvid_a|BOCC Session - February 24, 2025\nvid_b|BOCC Hearing - February 24, 2025\n",
	}

	cfg := boccConfig(t)
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)

	body, _ := cfg.GetBody("bocc")
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	require.Len(t, meetings, 3)

	byID := make(map[string]domain.Meeting)
	for _, m := range meetings {
		byID[m.VideoID] = m
	}
	// Sorted by VideoID: vid_a=1, vid_b=2, vid_c=3.
	assert.Equal(t, 1, byID["vid_a"].Sequence)
	assert.Equal(t, 2, byID["vid_b"].Sequence)
	assert.Equal(t, 3, byID["vid_c"].Sequence)
}

func TestDiscoveryService_SameDateDisambiguation_SoloMeeting(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "vid1|Board of County Commissioners Regular Meeting - January 7, 2025\nvid_a|BOCC Session - February 24, 2025\nvid_b|BOCC Hearing - February 24, 2025\n",
	}

	cfg := boccConfig(t)
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)

	body, _ := cfg.GetBody("bocc")
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	require.Len(t, meetings, 3)

	byID := make(map[string]domain.Meeting)
	for _, m := range meetings {
		byID[m.VideoID] = m
	}
	// Solo meeting on Jan 7 → sequence 0, no suffix.
	assert.Equal(t, 0, byID["vid1"].Sequence)
	assert.Equal(t, "", byID["vid1"].SequenceSuffix())
	// Pair on Feb 24 → sequences 1 and 2.
	assert.Equal(t, 1, byID["vid_a"].Sequence)
	assert.Equal(t, 2, byID["vid_b"].Sequence)
}

func TestDiscoveryService_SameDateDisambiguation_OneProcessed(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "vid_a|BOCC Session - February 24, 2025\nvid_b|BOCC Hearing - February 24, 2025\n",
	}

	cfg := boccConfig(t)
	body, _ := cfg.GetBody("bocc")

	// Mark vid_a (sequence 1) as already processed.
	summaryDir := filepath.Join(cfg.FinalizedDir(body), "20250224")
	require.NoError(t, os.MkdirAll(summaryDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(summaryDir, "BOCC-2025-02-24-Citizen-Summary-1.md"),
		[]byte("# Existing"), 0o644,
	))

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	// Only vid_b (sequence 2) should remain.
	require.Len(t, meetings, 1)
	assert.Equal(t, "vid_b", meetings[0].VideoID)
	assert.Equal(t, 2, meetings[0].Sequence)
}

func TestDiscoveryService_BOCC_MeetingTypeVariants(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		// Test that "Commissioner" (singular) and "Commissioners" (plural) both parse.
		Stdout: "vid1|Board of County Commissioners Regular Meeting - February 4, 2025\nvid2|Board of County Commissioner Work Session - January 21, 2025\nvid3|BOCC Public Hearing - January 14, 2025\n",
	}

	cfg := boccConfig(t)
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	discovery := service.NewDiscoveryService(ytdlp, cfg)

	body, _ := cfg.GetBody("bocc")
	meetings, err := discovery.DiscoverNewMeetings(context.Background(), body)
	require.NoError(t, err)

	require.Len(t, meetings, 3)

	// All should parse dates correctly.
	assert.Equal(t, "2025-02-04", meetings[0].ISODate())
	assert.Equal(t, "2025-01-21", meetings[1].ISODate())
	assert.Equal(t, "2025-01-14", meetings[2].ISODate())

	// Meeting type detection.
	assert.Equal(t, "Regular Meeting", meetings[0].MeetingType)
	assert.Equal(t, "Work Session", meetings[1].MeetingType)
	assert.Equal(t, "Public Hearing", meetings[2].MeetingType)
}
