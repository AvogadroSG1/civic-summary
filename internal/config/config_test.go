package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureConfig(t *testing.T) string {
	t.Helper()
	// Find fixtures relative to this test file.
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(wd, "..", "..", "testdata", "fixtures", "config.yaml")
}

// validLLM returns an llm block that passes validation, for the tests that
// build a Config literal instead of loading one (where viper defaults do not
// apply).
func validLLM() domain.LLMConfig {
	return domain.LLMConfig{
		Provider:       domain.ProviderAnthropic,
		Model:          "claude-opus-5",
		MaxTokens:      16000,
		MaxTokensField: domain.MaxTokensFieldModern,
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	assert.Equal(t, "/tmp/civic-summary-test", cfg.OutputDir)
	assert.Equal(t, 90, cfg.LogRetentionDays)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, []int{1, 2, 3}, cfg.BackoffDelays)
	assert.Equal(t, "yt-dlp", cfg.Tools.YtDlp)
}

func TestLoad_LLMBlock(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	assert.Equal(t, domain.ProviderAnthropic, cfg.LLM.Provider)
	assert.Equal(t, "claude-opus-5", cfg.LLM.Model)
	assert.Equal(t, "CIVIC_SUMMARY_TEST_API_KEY", cfg.LLM.APIKeyEnv)
	assert.Equal(t, 16000, cfg.LLM.MaxTokens)
	assert.Equal(t, 600, cfg.LLM.TimeoutSeconds)
	assert.Equal(t, "Global system prompt.", cfg.LLM.SystemPrompt)
	// Unset keys fall back to viper defaults.
	assert.Equal(t, domain.MaxTokensFieldModern, cfg.LLM.MaxTokensField)
	assert.Equal(t, 2, cfg.LLM.MaxRetries)
	assert.True(t, cfg.LLM.Stream)
	assert.Nil(t, cfg.LLM.Temperature, "temperature must stay unset: current Claude models reject it")
}

func TestLoad_LLMDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
output_dir: /tmp/defaults-test
bodies:
  test:
    playlist_id: PLtest
    output_subdir: Out
    filename_pattern: "Test-{{.MeetingDate}}"
    title_date_regex: '^(\d{4}-\d{2}-\d{2})'
    prompt_template: test.prompt.tmpl
    tags: [Test]
`), 0o644))

	cfg, err := config.Load(path)
	require.NoError(t, err)

	body, err := cfg.GetBody("test")
	require.NoError(t, err)
	resolved := cfg.ResolveLLM(body)

	assert.Equal(t, domain.ProviderAnthropic, resolved.Provider)
	assert.Equal(t, "claude-opus-5", resolved.Model)
	assert.Equal(t, "ANTHROPIC_API_KEY", resolved.APIKeyEnv,
		"api_key_env should default from the provider")
	assert.Equal(t, 16000, resolved.MaxTokens)
	assert.Equal(t, domain.MaxTokensFieldModern, resolved.MaxTokensField)
	assert.Equal(t, 900, resolved.TimeoutSeconds)
	assert.Equal(t, 15*time.Minute, resolved.Timeout())
	assert.Equal(t, 2, resolved.MaxRetries)
	assert.True(t, resolved.Stream)
	assert.Empty(t, resolved.BaseURL)
	assert.Empty(t, resolved.SystemPrompt)
}

func TestResolveLLM_InheritsGlobalBlock(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	body, err := cfg.GetBody("hagerstown")
	require.NoError(t, err)
	resolved := cfg.ResolveLLM(body)

	assert.Equal(t, "claude-opus-5", resolved.Model)
	assert.Equal(t, 16000, resolved.MaxTokens)
	assert.Equal(t, "Global system prompt.", resolved.SystemPrompt)
}

func TestResolveLLM_AppliesBodyOverride(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	body, err := cfg.GetBody("bocc")
	require.NoError(t, err)
	resolved := cfg.ResolveLLM(body)

	assert.Equal(t, "claude-sonnet-5", resolved.Model)
	assert.Equal(t, 32000, resolved.MaxTokens)
	// An explicitly empty override clears the inherited value.
	assert.Empty(t, resolved.SystemPrompt)
	// Unmentioned keys still come from the global block.
	assert.Equal(t, domain.ProviderAnthropic, resolved.Provider)
	assert.Equal(t, "CIVIC_SUMMARY_TEST_API_KEY", resolved.APIKeyEnv)
	assert.Equal(t, 600, resolved.TimeoutSeconds)
}

func TestResolveLLM_DefaultAPIKeyEnvFollowsProvider(t *testing.T) {
	cfg := &config.Config{LLM: domain.LLMConfig{Provider: domain.ProviderOpenAI}}

	resolved := cfg.ResolveLLM(domain.Body{})

	assert.Equal(t, "OPENAI_API_KEY", resolved.APIKeyEnv)
}

func TestLoad_LLMEnvOverride(t *testing.T) {
	t.Setenv("CIVIC_SUMMARY_LLM_MODEL", "claude-sonnet-5")
	t.Setenv("CIVIC_SUMMARY_LLM_BASE_URL", "http://localhost:11434/v1")

	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	assert.Equal(t, "claude-sonnet-5", cfg.LLM.Model)
	assert.Equal(t, "http://localhost:11434/v1", cfg.LLM.BaseURL)
}

func TestLoad_Bodies(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	assert.Len(t, cfg.Bodies, 2)

	hagerstown, err := cfg.GetBody("hagerstown")
	require.NoError(t, err)
	assert.Equal(t, "hagerstown", hagerstown.Slug)
	assert.Equal(t, "Hagerstown City Council", hagerstown.Name)
	assert.Equal(t, "PLJXxCe9GA2fEf4TIVzTH2O-kFJlS8VVgQ", hagerstown.PlaylistID)
	assert.Contains(t, hagerstown.Tags, "City-Council")

	bocc, err := cfg.GetBody("bocc")
	require.NoError(t, err)
	assert.Equal(t, "bocc", bocc.Slug)
	assert.Equal(t, "Washington County Board of County Commissioners", bocc.Name)
}

func TestLoad_UnknownBody(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	_, err = cfg.GetBody("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown body")
}

func TestLoad_BodySlugs(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	slugs := cfg.BodySlugs()
	assert.Len(t, slugs, 2)
	assert.Contains(t, slugs, "hagerstown")
	assert.Contains(t, slugs, "bocc")
}

func TestLoad_Paths(t *testing.T) {
	cfg, err := config.Load(fixtureConfig(t))
	require.NoError(t, err)

	body, _ := cfg.GetBody("hagerstown")
	assert.Equal(t,
		"/tmp/civic-summary-test/Hagerstown Town Council - Citizen Summary",
		cfg.BodyOutputDir(body),
	)
	assert.Equal(t,
		"/tmp/civic-summary-test/Hagerstown Town Council - Citizen Summary/Finalized Meeting Summaries",
		cfg.FinalizedDir(body),
	)
}

func TestValidate_MissingOutputDir(t *testing.T) {
	cfg := &config.Config{}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output_dir is required")
}

func TestValidate_NoBodies(t *testing.T) {
	cfg := &config.Config{OutputDir: "/tmp"}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one body")
}

func TestValidate_VideoSourceURLOnly(t *testing.T) {
	cfg := &config.Config{
		OutputDir: "/tmp",
		LLM:       validLLM(),
		Bodies: map[string]domain.Body{
			"test": {
				VideoSourceURL:  "https://www.youtube.com/@example/streams",
				OutputSubdir:    "Test Output",
				FilenamePattern: "Test-{{.MeetingDate}}",
				TitleDateRegex:  `^(\d{4}-\d{2}-\d{2})`,
				PromptTemplate:  "test.prompt.tmpl",
				Tags:            []string{"Test"},
			},
		},
	}
	err := cfg.Validate()
	assert.NoError(t, err)
}

// TestValidate_LLM checks the resolved llm block per body, so a bad override on
// one body is reported against that body.
func TestValidate_LLM(t *testing.T) {
	tests := []struct {
		name     string
		override *domain.LLMOverride
		wantErr  string
	}{
		{
			name:     "unsupported provider",
			override: &domain.LLMOverride{Provider: ptr("gemini")},
			wantErr:  `llm.provider "gemini" is not supported`,
		},
		{
			name:     "empty model",
			override: &domain.LLMOverride{Model: ptr("")},
			wantErr:  "llm.model is required",
		},
		{
			name:     "zero max_tokens",
			override: &domain.LLMOverride{MaxTokens: ptr(0)},
			wantErr:  "llm.max_tokens must be positive",
		},
		{
			name:     "unsupported max_tokens_field",
			override: &domain.LLMOverride{MaxTokensField: ptr("output_tokens")},
			wantErr:  `llm.max_tokens_field "output_tokens" is not supported`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				OutputDir: "/tmp",
				LLM:       validLLM(),
				Bodies: map[string]domain.Body{
					"test": {
						PlaylistID:      "PLtest",
						OutputSubdir:    "Test Output",
						FilenamePattern: "Test-{{.MeetingDate}}",
						TitleDateRegex:  `^(\d{4}-\d{2}-\d{2})`,
						PromptTemplate:  "test.prompt.tmpl",
						Tags:            []string{"Test"},
						LLM:             tt.override,
					},
				},
			}

			err := cfg.Validate()

			require.Error(t, err)
			assert.Contains(t, err.Error(), `body "test"`)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// ptr returns a pointer to v, for building LLMOverride literals.
func ptr[T any](v T) *T { return &v }

func TestValidate_NeitherPlaylistIDNorVideoSourceURL(t *testing.T) {
	cfg := &config.Config{
		OutputDir: "/tmp",
		Bodies: map[string]domain.Body{
			"test": {
				OutputSubdir:    "Test Output",
				FilenamePattern: "Test-{{.MeetingDate}}",
				TitleDateRegex:  `^(\d{4}-\d{2}-\d{2})`,
				PromptTemplate:  "test.prompt.tmpl",
				Tags:            []string{"Test"},
			},
		},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "playlist_id or video_source_url is required")
}

func TestValidate_BothPlaylistIDAndVideoSourceURL(t *testing.T) {
	cfg := &config.Config{
		OutputDir: "/tmp",
		LLM:       validLLM(),
		Bodies: map[string]domain.Body{
			"test": {
				PlaylistID:      "PLtest123",
				VideoSourceURL:  "https://www.youtube.com/@example/streams",
				OutputSubdir:    "Test Output",
				FilenamePattern: "Test-{{.MeetingDate}}",
				TitleDateRegex:  `^(\d{4}-\d{2}-\d{2})`,
				PromptTemplate:  "test.prompt.tmpl",
				Tags:            []string{"Test"},
			},
		},
	}
	err := cfg.Validate()
	assert.NoError(t, err)
}
