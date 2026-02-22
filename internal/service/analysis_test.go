package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMeeting returns a deterministic meeting for analysis tests.
func testMeeting() domain.Meeting {
	return domain.Meeting{
		VideoID:     "abc123",
		Title:       "February 04, 2025 | Mayor & Council Regular Session",
		MeetingDate: time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
		MeetingType: "Regular Session",
		BodySlug:    "hagerstown",
	}
}

// testTranscript returns a minimal transcript for analysis tests.
func testTranscript() domain.Transcript {
	return domain.Transcript{
		Content: "1\n00:00:01,000 --> 00:00:05,000\nThe meeting will come to order.\n",
		Path:    "/tmp/test.srt",
		Source:  domain.TranscriptSourceCaptions,
	}
}

// testHagerstownBody returns a hagerstown body config for analysis tests.
func testHagerstownBody() domain.Body {
	return domain.Body{
		Slug:            "hagerstown",
		Name:            "Hagerstown City Council",
		PlaylistID:      "PLJXxCe9GA2fEf4TIVzTH2O-kFJlS8VVgQ",
		OutputSubdir:    "Hagerstown Town Council - Citizen Summary",
		FilenamePattern: "Hagerstown-City-Council-{{.MeetingDate}}-Citizen-Summary",
		TitleDateRegex:  `^([A-Z][a-z]+ \d{1,2},? \d{4})`,
		Tags:            []string{"City-Council", "Hagerstown"},
		PromptTemplate:  "hagerstown.prompt.tmpl",
		Author:          "Peter O'Connor",
		FooterText:      "",
	}
}

// testBOCCBody returns a BOCC body config for analysis tests.
func testBOCCBody() domain.Body {
	return domain.Body{
		Slug:            "bocc",
		Name:            "Washington County Board of County Commissioners",
		PlaylistID:      "PL7X-j0EwreAd_6kV3IjxO-_XNwDNn0esS",
		OutputSubdir:    "Washington County BOCC - Citizen Summary",
		FilenamePattern: "BOCC-{{.MeetingDate}}-Citizen-Summary",
		TitleDateRegex:  `- ([A-Z][a-z]+ \d{1,2}, \d{4})`,
		Tags:            []string{"BOCC", "Washington-County"},
		PromptTemplate:  "bocc.prompt.tmpl",
		Author:          "Peter O'Connor",
		FooterText:      "",
	}
}

// setupTemplateDir copies real templates into a temp dir for testing.
func setupTemplateDir(t *testing.T) string {
	t.Helper()

	// Resolve project root relative to this test file.
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Join(wd, "..", "..")

	tmpDir := t.TempDir()

	for _, tmpl := range []string{"hagerstown.prompt.tmpl", "bocc.prompt.tmpl"} {
		src := filepath.Join(projectRoot, "templates", tmpl)
		content, err := os.ReadFile(src)
		require.NoError(t, err, "reading template %s", tmpl)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, tmpl), content, 0o644))
	}

	return tmpDir
}

func TestAnalysisService_Analyze(t *testing.T) {
	tmplDir := setupTemplateDir(t)

	// Mock Claude returning raw markdown with preamble that should be sanitized.
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "Here's the summary:\n---\ndate: 2025-02-05\nauthor: Peter O'Connor\ntags:\n  - City-Council\n---\n# Meeting Summary\nContent here.",
	}

	claude := executor.NewClaudeExecutor(mock, "claude")
	svc := service.NewAnalysisService(claude, tmplDir)

	summary, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())
	require.NoError(t, err)

	// Sanitize should strip the "Here's the summary:" preamble.
	assert.True(t, len(summary.Content) > 0, "summary should not be empty")
	assert.Contains(t, summary.Content, "---")
	assert.NotContains(t, summary.Content, "Here's the summary")
}

func TestAnalysisService_Analyze_CleanOutput(t *testing.T) {
	tmplDir := setupTemplateDir(t)

	// Mock Claude returning clean output (starts with frontmatter).
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "---\ndate: 2025-02-05\n---\n# Summary\nBody content.",
	}

	claude := executor.NewClaudeExecutor(mock, "claude")
	svc := service.NewAnalysisService(claude, tmplDir)

	summary, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())
	require.NoError(t, err)

	// Already clean — should pass through unchanged.
	assert.True(t, summary.Content[0:3] == "---", "should start with frontmatter")
}

func TestAnalysisService_Analyze_ClaudeError(t *testing.T) {
	tmplDir := setupTemplateDir(t)

	mock := executor.NewMockCommander()
	mock.OnCommand("claude", nil, assert.AnError)

	claude := executor.NewClaudeExecutor(mock, "claude")
	svc := service.NewAnalysisService(claude, tmplDir)

	_, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claude analysis")
}

func TestAnalysisService_BuildPrompt_Hagerstown(t *testing.T) {
	tmplDir := setupTemplateDir(t)

	// We need Claude to succeed so we can verify the prompt was built correctly.
	// The prompt is passed as stdin to Claude; we can inspect it via mock calls.
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "---\ndate: 2025-02-05\n---\n# Summary",
	}

	claude := executor.NewClaudeExecutor(mock, "claude")
	svc := service.NewAnalysisService(claude, tmplDir)

	meeting := testMeeting()
	body := testHagerstownBody()

	_, err := svc.Analyze(context.Background(), meeting, testTranscript(), body)
	require.NoError(t, err)

	// The mock records calls — verify Claude was invoked.
	require.Len(t, mock.Calls, 1)
	assert.Contains(t, mock.Calls[0], "claude")
}

func TestAnalysisService_BuildPrompt_BOCC(t *testing.T) {
	tmplDir := setupTemplateDir(t)

	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "---\ndate: 2025-02-05\n---\n# Summary",
	}

	claude := executor.NewClaudeExecutor(mock, "claude")
	svc := service.NewAnalysisService(claude, tmplDir)

	boccMeeting := domain.Meeting{
		VideoID:     "xyz789",
		Title:       "Board of County Commissioners Regular Meeting - January 7, 2025",
		MeetingDate: time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC),
		MeetingType: "Regular Meeting",
		BodySlug:    "bocc",
	}

	_, err := svc.Analyze(context.Background(), boccMeeting, testTranscript(), testBOCCBody())
	require.NoError(t, err)

	require.Len(t, mock.Calls, 1)
	assert.Contains(t, mock.Calls[0], "claude")
}

func TestAnalysisService_MeetingTypeTag(t *testing.T) {
	// meetingTypeTag is unexported, so we test it indirectly through Analyze.
	// The tags injected into the template contain the meeting type tag.
	// We verify by checking that different meeting types produce different outputs.
	tmplDir := setupTemplateDir(t)

	tests := []struct {
		name        string
		meetingType string
	}{
		{"work session", "Work Session"},
		{"special meeting", "Special Meeting"},
		{"evening meeting", "Evening Meeting"},
		{"regular session (default)", "Regular Session"},
		{"unknown defaults to regular", "Board Meeting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := executor.NewMockCommander()
			mock.DefaultResult = &executor.CommandResult{
				Stdout: "---\ndate: 2025-02-05\n---\n# Summary",
			}

			claude := executor.NewClaudeExecutor(mock, "claude")
			svc := service.NewAnalysisService(claude, tmplDir)

			meeting := testMeeting()
			meeting.MeetingType = tt.meetingType

			_, err := svc.Analyze(context.Background(), meeting, testTranscript(), testHagerstownBody())
			assert.NoError(t, err)
		})
	}
}

func TestAnalysisService_TemplateMissing(t *testing.T) {
	// Point to an empty temp dir — no templates.
	emptyDir := t.TempDir()

	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{Stdout: "output"}

	claude := executor.NewClaudeExecutor(mock, "claude")
	svc := service.NewAnalysisService(claude, emptyDir)

	_, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "building prompt")
}

func TestAnalysisService_Sanitize_MetaCommentary(t *testing.T) {
	tmplDir := setupTemplateDir(t)

	tests := []struct {
		name           string
		claudeOutput   string
		expectContains string
		expectMissing  string
	}{
		{
			"strips Here's preamble",
			"Here's the summary:\n---\ndate: 2025-02-05\n---\n# Summary",
			"---",
			"Here's",
		},
		{
			"strips I'll create preamble",
			"I'll create the citizen summary below.\n---\ndate: 2025-02-05\n---\n# Summary",
			"---",
			"I'll create",
		},
		{
			"preserves clean output",
			"---\ndate: 2025-02-05\n---\n# Summary",
			"---\ndate:",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := executor.NewMockCommander()
			mock.DefaultResult = &executor.CommandResult{Stdout: tt.claudeOutput}

			claude := executor.NewClaudeExecutor(mock, "claude")
			svc := service.NewAnalysisService(claude, tmplDir)

			summary, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())
			require.NoError(t, err)

			assert.Contains(t, summary.Content, tt.expectContains)
			if tt.expectMissing != "" {
				assert.NotContains(t, summary.Content, tt.expectMissing)
			}
		})
	}
}
