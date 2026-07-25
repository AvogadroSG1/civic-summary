package service_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/llm"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubClient is an llm.Client that records the prompts it receives, which is
// what makes the rendered prompt assertable.
type stubClient struct {
	response string
	err      error
	prompts  []string
}

func (s *stubClient) Complete(_ context.Context, prompt string) (string, error) {
	s.prompts = append(s.prompts, prompt)
	if s.err != nil {
		return "", s.err
	}
	return s.response, nil
}

func (s *stubClient) Ping(context.Context) error { return s.err }

func (s *stubClient) Describe() string { return "stub/test-model" }

// lastPrompt returns the most recent prompt, failing if none was sent.
func (s *stubClient) lastPrompt(t *testing.T) string {
	t.Helper()
	require.NotEmpty(t, s.prompts, "no prompt was sent to the model")
	return s.prompts[len(s.prompts)-1]
}

// stubClientFor returns a resolver that always yields stub.
func stubClientFor(stub *stubClient) service.LLMClientFor {
	return func(domain.Body) (llm.Client, error) { return stub, nil }
}

// newAnalysisService wires a service around a stub returning response.
func newAnalysisService(t *testing.T, response string) (*service.AnalysisService, *stubClient) {
	t.Helper()
	stub := &stubClient{response: response}
	return service.NewAnalysisService(stubClientFor(stub), setupTemplateDir(t)), stub
}

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
	svc, _ := newAnalysisService(t,
		"Here's the summary:\n---\ndate: 2025-02-05\nauthor: Peter O'Connor\ntags:\n  - City-Council\n---\n# Meeting Summary\nContent here.")

	summary, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())
	require.NoError(t, err)

	// Sanitize should strip the "Here's the summary:" preamble.
	assert.NotEmpty(t, summary.Content)
	assert.Contains(t, summary.Content, "---")
	assert.NotContains(t, summary.Content, "Here's the summary")
}

func TestAnalysisService_Analyze_CleanOutput(t *testing.T) {
	svc, _ := newAnalysisService(t, "---\ndate: 2025-02-05\n---\n# Summary\nBody content.")

	summary, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())
	require.NoError(t, err)

	// Already clean — should pass through unchanged.
	assert.True(t, strings.HasPrefix(summary.Content, "---"), "should start with frontmatter")
}

func TestAnalysisService_Analyze_ModelError(t *testing.T) {
	stub := &stubClient{err: assert.AnError}
	svc := service.NewAnalysisService(stubClientFor(stub), setupTemplateDir(t))

	_, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "analysis")
}

// TestAnalysisService_Analyze_ClientError covers a provider that cannot even be
// constructed, such as a missing API key.
func TestAnalysisService_Analyze_ClientError(t *testing.T) {
	failing := func(domain.Body) (llm.Client, error) { return nil, assert.AnError }
	svc := service.NewAnalysisService(failing, setupTemplateDir(t))

	_, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "building llm client")
}

func TestAnalysisService_BuildPrompt_Hagerstown(t *testing.T) {
	svc, stub := newAnalysisService(t, "---\ndate: 2025-02-05\n---\n# Summary")
	meeting := testMeeting()
	body := testHagerstownBody()

	_, err := svc.Analyze(context.Background(), meeting, testTranscript(), body)
	require.NoError(t, err)

	prompt := stub.lastPrompt(t)
	assert.Contains(t, prompt, body.Name)
	assert.Contains(t, prompt, "February 04, 2025", "human-readable meeting date")
	assert.Contains(t, prompt, "2025-02-04", "ISO meeting date")
	assert.Contains(t, prompt, "Regular Session", "meeting type")
	assert.Contains(t, prompt, "https://www.youtube.com/watch?v=abc123", "video URL")
	assert.Contains(t, prompt, body.Author)
	assert.Contains(t, prompt, testTranscript().Content, "the transcript itself")
	assert.Contains(t, prompt, "- City-Council")
	assert.Contains(t, prompt, "- Hagerstown")
	// Body-specific wording confirms the right template was rendered.
	assert.Contains(t, prompt, "Citizen Comments")
	assert.Contains(t, prompt, "Input Requested from Council")
}

func TestAnalysisService_BuildPrompt_BOCC(t *testing.T) {
	svc, stub := newAnalysisService(t, "---\ndate: 2025-02-05\n---\n# Summary")
	boccMeeting := domain.Meeting{
		VideoID:     "xyz789",
		Title:       "Board of County Commissioners Regular Meeting - January 7, 2025",
		MeetingDate: time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC),
		MeetingType: "Regular Meeting",
		BodySlug:    "bocc",
	}
	body := testBOCCBody()

	_, err := svc.Analyze(context.Background(), boccMeeting, testTranscript(), body)
	require.NoError(t, err)

	prompt := stub.lastPrompt(t)
	assert.Contains(t, prompt, body.Name)
	assert.Contains(t, prompt, "January 07, 2025")
	assert.Contains(t, prompt, "https://www.youtube.com/watch?v=xyz789")
	assert.Contains(t, prompt, "- BOCC")
	assert.Contains(t, prompt, "- Washington-County")
	// The BOCC template addresses commissioners, not a council.
	assert.Contains(t, prompt, "Commissioners")
}

// TestAnalysisService_MeetingTypeTag checks the derived tag that gets appended
// to the body's configured tags.
func TestAnalysisService_MeetingTypeTag(t *testing.T) {
	tests := []struct {
		name        string
		meetingType string
		wantTag     string
	}{
		{"work session", "Work Session", "- Work-Session"},
		{"special meeting", "Special Meeting", "- Special-Meeting"},
		{"evening meeting", "Evening Meeting", "- Evening-Meeting"},
		{"regular session", "Regular Session", "- Regular-Session"},
		{"unknown defaults to regular session", "Board Meeting", "- Regular-Session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, stub := newAnalysisService(t, "---\ndate: 2025-02-05\n---\n# Summary")
			meeting := testMeeting()
			meeting.MeetingType = tt.meetingType

			_, err := svc.Analyze(context.Background(), meeting, testTranscript(), testHagerstownBody())
			require.NoError(t, err)

			assert.Contains(t, stub.lastPrompt(t), tt.wantTag)
		})
	}
}

func TestAnalysisService_TemplateMissing(t *testing.T) {
	// Point to an empty temp dir — no templates.
	stub := &stubClient{response: "output"}
	svc := service.NewAnalysisService(stubClientFor(stub), t.TempDir())

	_, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "building prompt")
	assert.Empty(t, stub.prompts, "no request should be made when the template is missing")
}

func TestAnalysisService_Sanitize_MetaCommentary(t *testing.T) {
	tests := []struct {
		name           string
		modelOutput    string
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
			svc, _ := newAnalysisService(t, tt.modelOutput)

			summary, err := svc.Analyze(context.Background(), testMeeting(), testTranscript(), testHagerstownBody())
			require.NoError(t, err)

			assert.Contains(t, summary.Content, tt.expectContains)
			if tt.expectMissing != "" {
				assert.NotContains(t, summary.Content, tt.expectMissing)
			}
		})
	}
}
