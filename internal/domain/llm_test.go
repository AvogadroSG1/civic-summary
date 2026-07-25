package domain_test

import (
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestProviders(t *testing.T) {
	assert.Equal(t, []string{"anthropic", "openai"}, domain.Providers())
}

func TestMaxTokensFields(t *testing.T) {
	assert.Equal(t, []string{"max_completion_tokens", "max_tokens"}, domain.MaxTokensFields())
}

func TestDefaultAPIKeyEnv(t *testing.T) {
	assert.Equal(t, "ANTHROPIC_API_KEY", domain.DefaultAPIKeyEnv(domain.ProviderAnthropic))
	assert.Equal(t, "OPENAI_API_KEY", domain.DefaultAPIKeyEnv(domain.ProviderOpenAI))
	assert.Equal(t, "ANTHROPIC_API_KEY", domain.DefaultAPIKeyEnv(""))
}

func TestLLMConfig_Timeout(t *testing.T) {
	assert.Equal(t, 15*time.Minute, domain.LLMConfig{TimeoutSeconds: 900}.Timeout())
	assert.Zero(t, domain.LLMConfig{TimeoutSeconds: 0}.Timeout())
	assert.Zero(t, domain.LLMConfig{TimeoutSeconds: -1}.Timeout())
}

func TestLLMConfig_Describe(t *testing.T) {
	cfg := domain.LLMConfig{Provider: "anthropic", Model: "claude-opus-5"}

	assert.Equal(t, "anthropic/claude-opus-5", cfg.Describe())
}

// base returns a fully populated global LLM block for merge tests.
func base() domain.LLMConfig {
	temperature := 0.2
	return domain.LLMConfig{
		Provider:       domain.ProviderAnthropic,
		Model:          "claude-opus-5",
		BaseURL:        "https://api.anthropic.com",
		APIKeyEnv:      "ANTHROPIC_API_KEY",
		MaxTokens:      16000,
		MaxTokensField: domain.MaxTokensFieldModern,
		Temperature:    &temperature,
		TimeoutSeconds: 900,
		MaxRetries:     2,
		Stream:         true,
		SystemPrompt:   "global system",
	}
}

func TestLLMOverride_Apply_NilLeavesBaseUnchanged(t *testing.T) {
	var override *domain.LLMOverride

	assert.Equal(t, base(), override.Apply(base()))
}

func TestLLMOverride_Apply_EmptyLeavesBaseUnchanged(t *testing.T) {
	assert.Equal(t, base(), (&domain.LLMOverride{}).Apply(base()))
}

func TestLLMOverride_Apply_ReplacesSetFieldsOnly(t *testing.T) {
	model := "claude-sonnet-5"
	maxTokens := 32000
	override := &domain.LLMOverride{Model: &model, MaxTokens: &maxTokens}

	merged := override.Apply(base())

	assert.Equal(t, "claude-sonnet-5", merged.Model)
	assert.Equal(t, 32000, merged.MaxTokens)
	// Everything else is inherited.
	assert.Equal(t, domain.ProviderAnthropic, merged.Provider)
	assert.Equal(t, "https://api.anthropic.com", merged.BaseURL)
	assert.Equal(t, 900, merged.TimeoutSeconds)
	assert.Equal(t, "global system", merged.SystemPrompt)
}

// TestLLMOverride_Apply_HonoursExplicitZeroValues is the reason the override
// type uses pointers: a body must be able to clear an inherited base_url or
// turn streaming off, which a plain-value merge could not express.
func TestLLMOverride_Apply_HonoursExplicitZeroValues(t *testing.T) {
	emptyURL := ""
	streamOff := false
	noSystem := ""
	override := &domain.LLMOverride{
		BaseURL:      &emptyURL,
		Stream:       &streamOff,
		SystemPrompt: &noSystem,
	}

	merged := override.Apply(base())

	assert.Empty(t, merged.BaseURL)
	assert.False(t, merged.Stream)
	assert.Empty(t, merged.SystemPrompt)
}

func TestLLMOverride_Apply_ReplacesTemperature(t *testing.T) {
	warmer := 0.9
	override := &domain.LLMOverride{Temperature: &warmer}

	merged := override.Apply(base())

	assert.NotNil(t, merged.Temperature)
	assert.InDelta(t, 0.9, *merged.Temperature, 1e-9)
}

func TestLLMOverride_Apply_InheritsTemperatureWhenUnset(t *testing.T) {
	merged := (&domain.LLMOverride{}).Apply(base())

	assert.NotNil(t, merged.Temperature)
	assert.InDelta(t, 0.2, *merged.Temperature, 1e-9)
}

func TestLLMOverride_Apply_SwitchesProvider(t *testing.T) {
	provider := domain.ProviderOpenAI
	model := "gpt-5"
	keyEnv := "OPENAI_API_KEY"
	baseURL := "http://localhost:11434/v1"
	field := domain.MaxTokensFieldLegacy
	override := &domain.LLMOverride{
		Provider:       &provider,
		Model:          &model,
		APIKeyEnv:      &keyEnv,
		BaseURL:        &baseURL,
		MaxTokensField: &field,
	}

	merged := override.Apply(base())

	assert.Equal(t, domain.ProviderOpenAI, merged.Provider)
	assert.Equal(t, "gpt-5", merged.Model)
	assert.Equal(t, "OPENAI_API_KEY", merged.APIKeyEnv)
	assert.Equal(t, "http://localhost:11434/v1", merged.BaseURL)
	assert.Equal(t, domain.MaxTokensFieldLegacy, merged.MaxTokensField)
}

// TestLLMOverride_Apply_DoesNotMutateBase guards against the merge aliasing the
// global block, which would let one body's override leak into another's.
func TestLLMOverride_Apply_DoesNotMutateBase(t *testing.T) {
	original := base()
	model := "claude-sonnet-5"

	_ = (&domain.LLMOverride{Model: &model}).Apply(original)

	assert.Equal(t, "claude-opus-5", original.Model)
}
