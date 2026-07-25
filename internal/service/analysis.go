package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/llm"
	"github.com/AvogadroSG1/civic-summary/internal/markdown"
)

// LLMClientFor returns the language-model client to use for a body. Bodies can
// override the global provider and model, so the client is resolved per body
// rather than once per run.
type LLMClientFor func(body domain.Body) (llm.Client, error)

// AnalysisService generates meeting summaries with a language model.
type AnalysisService struct {
	clientFor   LLMClientFor
	templateDir string
}

// NewAnalysisService creates a new AnalysisService.
func NewAnalysisService(clientFor LLMClientFor, templateDir string) *AnalysisService {
	return &AnalysisService{clientFor: clientFor, templateDir: templateDir}
}

// PromptData holds all data injected into a prompt template.
type PromptData struct {
	MeetingDateHuman string
	MeetingDateISO   string
	MeetingType      string
	VideoID          string
	VideoURL         string
	AgendaURL        string
	TodayDate        string
	Author           string
	Tags             []string
	Transcript       string
	BodyName         string
	FooterText       string
}

// Analyze sends the meeting transcript to the configured model and returns the
// generated summary.
func (s *AnalysisService) Analyze(ctx context.Context, meeting domain.Meeting, transcript domain.Transcript, body domain.Body) (domain.Summary, error) {
	client, err := s.clientFor(body)
	if err != nil {
		return domain.Summary{}, fmt.Errorf("building llm client: %w", err)
	}

	prompt, err := s.buildPrompt(meeting, transcript, body)
	if err != nil {
		return domain.Summary{}, fmt.Errorf("building prompt: %w", err)
	}

	slog.Info("analyzing meeting",
		"video_id", meeting.VideoID,
		"body", body.Slug,
		"model", client.Describe(),
		"transcript_words", transcript.WordCount(),
	)

	rawOutput, err := client.Complete(ctx, prompt)
	if err != nil {
		return domain.Summary{}, fmt.Errorf("analysis: %w", err)
	}

	// Models sometimes prefix the document with meta-commentary; strip it.
	content := markdown.Sanitize(rawOutput)

	return domain.Summary{
		Content: content,
	}, nil
}

// buildPrompt renders the body-specific prompt template with meeting data.
func (s *AnalysisService) buildPrompt(meeting domain.Meeting, transcript domain.Transcript, body domain.Body) (string, error) {
	tmplPath := filepath.Join(s.templateDir, body.PromptTemplate)
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("reading template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(body.PromptTemplate).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	// Determine meeting type tag.
	tags := make([]string, len(body.Tags))
	copy(tags, body.Tags)
	tags = append(tags, meetingTypeTag(meeting.MeetingType))

	data := PromptData{
		MeetingDateHuman: meeting.HumanDate(),
		MeetingDateISO:   meeting.ISODate(),
		MeetingType:      meeting.MeetingType,
		VideoID:          meeting.VideoID,
		VideoURL:         body.VideoURL(meeting.VideoID),
		TodayDate:        time.Now().Format("2006-01-02"),
		Author:           body.Author,
		Tags:             tags,
		Transcript:       transcript.Content,
		BodyName:         body.Name,
		FooterText:       body.FooterText,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

// meetingTypeTag converts a meeting type to a tag-friendly format.
func meetingTypeTag(meetingType string) string {
	switch meetingType {
	case "Work Session":
		return "Work-Session"
	case "Special Meeting":
		return "Special-Meeting"
	case "Evening Meeting":
		return "Evening-Meeting"
	default:
		return "Regular-Session"
	}
}
