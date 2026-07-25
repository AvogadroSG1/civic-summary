package cmd

import (
	"fmt"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/llm"
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
func buildExecutors(cfg *config.Config) (*executor.YtDlpExecutor, *executor.WhisperExecutor) {
	commander := executor.NewOsCommander()

	ytdlp := executor.NewYtDlpExecutor(commander, cfg.Tools.YtDlp)

	var whisper *executor.WhisperExecutor
	if cfg.Tools.Whisper != "" && cfg.Tools.WhisperModel != "" {
		whisper = executor.NewWhisperExecutor(commander, cfg.Tools.Whisper, cfg.Tools.WhisperModel)
	}

	return ytdlp, whisper
}

// buildLLMClientFor returns a resolver that builds the language-model client for
// a body, honouring any per-body override of the global llm block. The client is
// built lazily so that commands which never analyse a meeting do not require an
// API key.
func buildLLMClientFor(cfg *config.Config) service.LLMClientFor {
	return func(body domain.Body) (llm.Client, error) {
		return llm.New(cfg.ResolveLLM(body))
	}
}

// buildAnalysisService creates an AnalysisService wired to the configured
// provider. Both the full pipeline and the standalone analyze command use it.
func buildAnalysisService(cfg *config.Config) *service.AnalysisService {
	return service.NewAnalysisService(buildLLMClientFor(cfg), cfg.TemplateDir())
}

// buildPipeline creates a fully-wired PipelineOrchestrator.
func buildPipeline(cfg *config.Config) *service.PipelineOrchestrator {
	ytdlp, whisper := buildExecutors(cfg)

	discovery := service.NewDiscoveryService(ytdlp, cfg)
	transcription := service.NewTranscriptionService(ytdlp, whisper)
	analysis := buildAnalysisService(cfg)
	crossref := service.NewCrossReferenceService(cfg)
	validation := service.NewValidationService()
	quarantine := service.NewQuarantineService(cfg)
	index := service.NewIndexService(cfg)

	return service.NewPipelineOrchestrator(
		discovery, transcription, analysis, crossref,
		validation, quarantine, index, cfg,
	)
}
