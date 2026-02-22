package cmd

import (
	"fmt"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/spf13/cobra"
)

// loadConfig loads and validates the application configuration.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

// getBody resolves a body slug from command flags.
func getBody(cmd *cobra.Command, cfg *config.Config) (domain.Body, error) {
	slug, err := cmd.Flags().GetString("body")
	if err != nil {
		return domain.Body{}, fmt.Errorf("reading --body flag: %w", err)
	}
	if slug == "" {
		return domain.Body{}, fmt.Errorf("--body flag is required")
	}
	return cfg.GetBody(slug)
}

// buildExecutors creates all executor instances from config.
func buildExecutors(cfg *config.Config) (*executor.YtDlpExecutor, *executor.WhisperExecutor, *executor.ClaudeExecutor) {
	commander := executor.NewOsCommander()

	ytdlp := executor.NewYtDlpExecutor(commander, cfg.Tools.YtDlp)

	var whisper *executor.WhisperExecutor
	if cfg.Tools.Whisper != "" && cfg.Tools.WhisperModel != "" {
		whisper = executor.NewWhisperExecutor(commander, cfg.Tools.Whisper, cfg.Tools.WhisperModel)
	}

	claude := executor.NewClaudeExecutor(commander, cfg.Tools.Claude)

	return ytdlp, whisper, claude
}

// buildPipeline creates a fully-wired PipelineOrchestrator.
func buildPipeline(cfg *config.Config) *service.PipelineOrchestrator {
	ytdlp, whisper, claude := buildExecutors(cfg)

	discovery := service.NewDiscoveryService(ytdlp, cfg)
	transcription := service.NewTranscriptionService(ytdlp, whisper)
	analysis := service.NewAnalysisService(claude, cfg.TemplateDir())
	crossref := service.NewCrossReferenceService(cfg)
	validation := service.NewValidationService()
	quarantine := service.NewQuarantineService(cfg)
	index := service.NewIndexService(cfg)

	return service.NewPipelineOrchestrator(
		discovery, transcription, analysis, crossref,
		validation, quarantine, index, cfg,
	)
}
