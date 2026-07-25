// Package config handles loading and validating application configuration
// from YAML files and environment variables using Viper.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/spf13/viper"
)

// Defaults for the llm block. The model is a current, widely-available Claude
// model; max_tokens leaves headroom above a typical summary because models that
// reason before answering spend part of the same budget on reasoning.
const (
	defaultModel          = "claude-opus-5"
	defaultMaxTokens      = 16000
	defaultTimeoutSeconds = 900
	defaultLLMRetries     = 2
)

// Config holds all application configuration.
type Config struct {
	OutputDir        string                 `mapstructure:"output_dir"`
	LogRetentionDays int                    `mapstructure:"log_retention_days"`
	MaxRetries       int                    `mapstructure:"max_retries"`
	BackoffDelays    []int                  `mapstructure:"backoff_delays"`
	Tools            ToolsConfig            `mapstructure:"tools"`
	LLM              domain.LLMConfig       `mapstructure:"llm"`
	Bodies           map[string]domain.Body `mapstructure:"bodies"`
}

// ToolsConfig holds paths to external tool binaries.
type ToolsConfig struct {
	YtDlp        string `mapstructure:"ytdlp"`
	Whisper      string `mapstructure:"whisper"`
	WhisperModel string `mapstructure:"whisper_model"`
}

// Load reads configuration from the config file and environment variables.
// Config file search order:
//  1. --config flag (if provided)
//  2. ~/.civic-summary/config.yaml
//  3. ./config.yaml (for development)
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	// Defaults
	v.SetDefault("log_retention_days", 90)
	v.SetDefault("max_retries", 3)
	v.SetDefault("backoff_delays", []int{5, 20, 60})
	v.SetDefault("tools.ytdlp", "yt-dlp")
	v.SetDefault("llm.provider", domain.ProviderAnthropic)
	v.SetDefault("llm.model", defaultModel)
	v.SetDefault("llm.max_tokens", defaultMaxTokens)
	v.SetDefault("llm.max_tokens_field", domain.MaxTokensFieldModern)
	v.SetDefault("llm.timeout_seconds", defaultTimeoutSeconds)
	v.SetDefault("llm.max_retries", defaultLLMRetries)
	v.SetDefault("llm.stream", true)

	// Environment variable binding (12-Factor: config in env)
	v.SetEnvPrefix("CIVIC_SUMMARY")
	v.AutomaticEnv()
	_ = v.BindEnv("output_dir")
	_ = v.BindEnv("log_retention_days")
	_ = v.BindEnv("max_retries")
	_ = v.BindEnv("tools.ytdlp", "CIVIC_SUMMARY_YTDLP")
	_ = v.BindEnv("tools.whisper", "CIVIC_SUMMARY_WHISPER")
	_ = v.BindEnv("tools.whisper_model", "CIVIC_SUMMARY_WHISPER_MODEL")
	_ = v.BindEnv("llm.provider", "CIVIC_SUMMARY_LLM_PROVIDER")
	_ = v.BindEnv("llm.model", "CIVIC_SUMMARY_LLM_MODEL")
	_ = v.BindEnv("llm.base_url", "CIVIC_SUMMARY_LLM_BASE_URL")
	_ = v.BindEnv("llm.api_key_env", "CIVIC_SUMMARY_LLM_API_KEY_ENV")
	_ = v.BindEnv("llm.max_tokens", "CIVIC_SUMMARY_LLM_MAX_TOKENS")
	_ = v.BindEnv("llm.max_tokens_field", "CIVIC_SUMMARY_LLM_MAX_TOKENS_FIELD")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("finding home directory: %w", err)
		}
		v.AddConfigPath(filepath.Join(home, ".civic-summary"))
		v.AddConfigPath(".")
		v.SetConfigName("config")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Inject slugs from map keys into each body.
	for slug, body := range cfg.Bodies {
		body.Slug = slug
		cfg.Bodies[slug] = body
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// ResolveLLM returns the language-model configuration for a body: the global
// llm block with the body's override applied, plus the defaults that depend on
// other values and so cannot be expressed as static viper defaults.
func (c *Config) ResolveLLM(body domain.Body) domain.LLMConfig {
	resolved := body.LLM.Apply(c.LLM)

	if resolved.APIKeyEnv == "" {
		resolved.APIKeyEnv = domain.DefaultAPIKeyEnv(resolved.Provider)
	}

	return resolved
}

// Validate checks that required configuration fields are present.
func (c *Config) Validate() error {
	if c.OutputDir == "" {
		return fmt.Errorf("output_dir is required")
	}
	if len(c.Bodies) == 0 {
		return fmt.Errorf("at least one body must be configured")
	}
	for slug, body := range c.Bodies {
		if body.PlaylistID == "" && body.VideoSourceURL == "" {
			return fmt.Errorf("body %q: playlist_id or video_source_url is required", slug)
		}
		if body.OutputSubdir == "" {
			return fmt.Errorf("body %q: output_subdir is required", slug)
		}
		if body.FilenamePattern == "" {
			return fmt.Errorf("body %q: filename_pattern is required", slug)
		}
		if body.TitleDateRegex == "" {
			return fmt.Errorf("body %q: title_date_regex is required", slug)
		}
		if body.PromptTemplate == "" {
			return fmt.Errorf("body %q: prompt_template is required", slug)
		}
		if len(body.Tags) == 0 {
			return fmt.Errorf("body %q: at least one tag is required", slug)
		}
		if err := validateLLM(c.ResolveLLM(body)); err != nil {
			return fmt.Errorf("body %q: %w", slug, err)
		}
	}
	return nil
}

// validateLLM checks a body's resolved llm block. The API key is deliberately
// not checked here: Validate runs on every command, and commands that never
// reach a model must keep working without credentials.
func validateLLM(cfg domain.LLMConfig) error {
	if !slices.Contains(domain.Providers(), cfg.Provider) {
		return fmt.Errorf("llm.provider %q is not supported; supported: %v", cfg.Provider, domain.Providers())
	}
	if cfg.Model == "" {
		return fmt.Errorf("llm.model is required")
	}
	if cfg.MaxTokens <= 0 {
		return fmt.Errorf("llm.max_tokens must be positive, got %d", cfg.MaxTokens)
	}
	if !slices.Contains(domain.MaxTokensFields(), cfg.MaxTokensField) {
		return fmt.Errorf("llm.max_tokens_field %q is not supported; supported: %v",
			cfg.MaxTokensField, domain.MaxTokensFields())
	}
	return nil
}

// GetBody returns the body configuration for the given slug, or an error if not found.
func (c *Config) GetBody(slug string) (domain.Body, error) {
	body, ok := c.Bodies[slug]
	if !ok {
		return domain.Body{}, fmt.Errorf("unknown body %q; available: %v", slug, c.BodySlugs())
	}
	return body, nil
}

// BodySlugs returns a sorted list of configured body slugs.
func (c *Config) BodySlugs() []string {
	slugs := make([]string, 0, len(c.Bodies))
	for slug := range c.Bodies {
		slugs = append(slugs, slug)
	}
	return slugs
}

// BodyOutputDir returns the full output directory for a body's summaries.
func (c *Config) BodyOutputDir(body domain.Body) string {
	return filepath.Join(c.OutputDir, body.OutputSubdir)
}

// FinalizedDir returns the finalized meeting summaries directory for a body.
func (c *Config) FinalizedDir(body domain.Body) string {
	return filepath.Join(c.BodyOutputDir(body), "Finalized Meeting Summaries")
}

// QuarantineDir returns the quarantine directory for a body.
func (c *Config) QuarantineDir(body domain.Body) string {
	return filepath.Join(c.BodyOutputDir(body), "Automation", "quarantine")
}

// LogDir returns the log directory for a body.
func (c *Config) LogDir(body domain.Body) string {
	return filepath.Join(c.BodyOutputDir(body), "Automation", "logs")
}

// TemplateDir returns the directory containing prompt templates.
// Searches: ~/.civic-summary/templates, then ./templates
func (c *Config) TemplateDir() string {
	home, err := os.UserHomeDir()
	if err == nil {
		dir := filepath.Join(home, ".civic-summary", "templates")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return "templates"
}
