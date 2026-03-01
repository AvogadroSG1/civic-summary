package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/AvogadroSG1/civic-summary/internal/retry"
)

// PipelineOrchestrator coordinates all pipeline stages for processing meetings.
type PipelineOrchestrator struct {
	discovery     *DiscoveryService
	transcription *TranscriptionService
	analysis      *AnalysisService
	crossref      *CrossReferenceService
	validation    *ValidationService
	quarantine    *QuarantineService
	index         *IndexService
	cfg           *config.Config
	retryCfg      retry.Config
}

// NewPipelineOrchestrator creates a fully-wired pipeline orchestrator.
func NewPipelineOrchestrator(
	discovery *DiscoveryService,
	transcription *TranscriptionService,
	analysis *AnalysisService,
	crossref *CrossReferenceService,
	validation *ValidationService,
	quarantine *QuarantineService,
	index *IndexService,
	cfg *config.Config,
) *PipelineOrchestrator {
	return &PipelineOrchestrator{
		discovery:     discovery,
		transcription: transcription,
		analysis:      analysis,
		crossref:      crossref,
		validation:    validation,
		quarantine:    quarantine,
		index:         index,
		cfg:           cfg,
		retryCfg:      retry.NewConfig(cfg.MaxRetries, cfg.BackoffDelays),
	}
}

// ProcessBody runs the full pipeline for a single government body.
func (p *PipelineOrchestrator) ProcessBody(ctx context.Context, body domain.Body, dryRun bool) (*domain.ProcessingStats, error) {
	stats := &domain.ProcessingStats{}

	output.Banner(fmt.Sprintf("Processing: %s", body.Name))

	// Phase 1: Discovery
	meetings, err := p.discovery.DiscoverNewMeetings(ctx, body)
	if err != nil {
		return stats, fmt.Errorf("discovery failed: %w", err)
	}

	stats.Discovered = len(meetings)

	if len(meetings) == 0 {
		output.Success("No new videos to process for %s", body.Name)
	}

	if dryRun {
		for _, m := range meetings {
			output.Info("Would process: %s (%s) - %s", m.ISODate(), m.VideoID, m.Title)
		}
		return stats, nil
	}

	// Phase 2-5: Process each meeting with retry
	for _, meeting := range meetings {
		output.Info("Processing: %s (%s)", meeting.ISODate(), meeting.Title)

		err := retry.Do(ctx, p.retryCfg, meeting.VideoID, func() error {
			return p.processSingleMeeting(ctx, meeting, body)
		})

		if err != nil {
			output.Failure("Failed: %s - %s", meeting.ISODate(), err)
			stats.Failed++

			// Quarantine on failure.
			qErr := p.quarantine.Quarantine(body, meeting, err.Error(), "", "")
			if qErr != nil {
				slog.Error("quarantine failed", "error", qErr)
			}
			stats.Quarantined++
		} else {
			output.Success("Completed: %s", meeting.ISODate())
			stats.Processed++
		}
	}

	// Retry quarantined items.
	p.retryQuarantined(ctx, body, stats)

	// Update index.
	if err := p.index.UpdateIndex(body); err != nil {
		slog.Warn("index update failed", "error", err)
	}

	return stats, nil
}

// ProcessAll runs the pipeline for all configured bodies.
func (p *PipelineOrchestrator) ProcessAll(ctx context.Context, dryRun bool) (map[string]*domain.ProcessingStats, error) {
	allStats := make(map[string]*domain.ProcessingStats)

	for slug, body := range p.cfg.Bodies {
		stats, err := p.ProcessBody(ctx, body, dryRun)
		allStats[slug] = stats
		if err != nil {
			slog.Error("body processing failed",
				"body", slug,
				"error", err,
			)
		}
	}

	return allStats, nil
}

// processSingleMeeting runs phases 2-5 for a single meeting.
func (p *PipelineOrchestrator) processSingleMeeting(ctx context.Context, meeting domain.Meeting, body domain.Body) error {
	// Ensure output directory exists.
	dateDir := filepath.Join(p.cfg.FinalizedDir(body), meeting.DateFolder())
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		return fmt.Errorf("creating date directory: %w", err)
	}

	// Phase 2: Transcription
	transcript, err := p.transcription.Transcribe(ctx, meeting, dateDir)
	if err != nil {
		return fmt.Errorf("transcription: %w", err)
	}

	if err := p.transcription.ValidateTranscript(transcript); err != nil {
		return fmt.Errorf("transcript validation: %w", err)
	}

	// Phase 3: Analysis
	summary, err := p.analysis.Analyze(ctx, meeting, transcript, body)
	if err != nil {
		return fmt.Errorf("analysis: %w", err)
	}

	// Phase 4: Cross-reference (non-critical)
	content := p.crossref.AddCrossReferences(summary.Content, meeting, body)

	// Phase 5: Validation
	result := p.validation.Validate(content, body)
	if result.HasErrors() {
		for _, issue := range result.Errors() {
			slog.Error("validation error", "issue", issue.String())
		}
		return fmt.Errorf("validation failed with %d errors", len(result.Errors()))
	}

	for _, issue := range result.Warnings() {
		slog.Warn("validation warning", "issue", issue.String())
	}

	// Write final summary.
	summaryPath := p.discovery.SummaryPath(meeting, body)
	if err := os.WriteFile(summaryPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing summary: %w", err)
	}

	slog.Info("summary finalized",
		"path", summaryPath,
		"words", len(content),
	)

	return nil
}

// retryQuarantined attempts to reprocess previously quarantined meetings.
func (p *PipelineOrchestrator) retryQuarantined(ctx context.Context, body domain.Body, stats *domain.ProcessingStats) {
	entries, err := p.quarantine.ListQuarantined(body)
	if err != nil || len(entries) == 0 {
		return
	}

	output.Banner("Retrying Quarantined Items")
	output.Info("Found %d quarantined item(s)", len(entries))

	for _, entry := range entries {
		output.Info("Retrying: %s (date: %s, retries: %d)",
			entry.VideoID, entry.MeetingDate, entry.RetryCount)

		meeting := domain.Meeting{
			VideoID:  entry.VideoID,
			BodySlug: body.Slug,
			Sequence: entry.Sequence,
		}
		// Parse the meeting date.
		if date, err := parseFlexibleDate(entry.MeetingDate); err == nil {
			meeting.MeetingDate = date
		}

		if err := p.quarantine.IncrementRetry(body, entry.VideoID); err != nil {
			slog.Warn("failed to increment retry count", "error", err)
		}

		if err := p.processSingleMeeting(ctx, meeting, body); err != nil {
			output.Failure("Retry failed: %s - %s", entry.VideoID, err)
		} else {
			output.Success("Retry succeeded: %s", entry.VideoID)
			if err := p.quarantine.Remove(body, entry.VideoID); err != nil {
				slog.Warn("failed to remove from quarantine", "error", err)
			}
			stats.Processed++
		}
	}
}
