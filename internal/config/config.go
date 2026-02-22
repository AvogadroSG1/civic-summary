// Package config handles loading and validating application configuration
// from YAML files and environment variables using Viper.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	OutputDir        string                 `mapstructure:"output_dir"`
	LogRetentionDays int                    `mapstructure:"log_retention_days"`
	MaxRetries       int                    `mapstructure:"max_retries"`
	BackoffDelays    []int                  `mapstructure:"backoff_delays"`
	Tools            ToolsConfig            `mapstructure:"tools"`
	Bodies           map[string]domain.Body `mapstructure:"bodies"`
}

// ToolsConfig holds paths to external tool binaries.
type ToolsConfig struct {
	YtDlp        string `mapstructure:"ytdlp"`
	Whisper      string `mapstructure:"whisper"`
	WhisperModel string `mapstructure:"whisper_model"`
	Claude       string `mapstructure:"claude"`
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
	v.SetDefault("tools.claude", "claude")

	// Environment variable binding (12-Factor: config in env)
	v.SetEnvPrefix("CIVIC_SUMMARY")
	v.AutomaticEnv()
	_ = v.BindEnv("output_dir")
	_ = v.BindEnv("log_retention_days")
	_ = v.BindEnv("max_retries")
	_ = v.BindEnv("tools.ytdlp", "CIVIC_SUMMARY_YTDLP")
	_ = v.BindEnv("tools.whisper", "CIVIC_SUMMARY_WHISPER")
	_ = v.BindEnv("tools.whisper_model", "CIVIC_SUMMARY_WHISPER_MODEL")
	_ = v.BindEnv("tools.claude", "CIVIC_SUMMARY_CLAUDE")

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
