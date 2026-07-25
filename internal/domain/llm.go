package domain

import "time"

// Supported LLM provider identifiers, as they appear in configuration. Each
// names a wire protocol rather than a vendor: any endpoint implementing that
// protocol works, including gateways and self-hosted servers.
const (
	// ProviderAnthropic speaks the Anthropic Messages API (POST /v1/messages).
	ProviderAnthropic = "anthropic"
	// ProviderOpenAI speaks the OpenAI Chat Completions API
	// (POST /v1/chat/completions).
	ProviderOpenAI = "openai"
)

// Supported values for LLMConfig.MaxTokensField.
const (
	// MaxTokensFieldModern is the current OpenAI output-limit field. Reasoning
	// models reject the legacy field, so this is the default.
	MaxTokensFieldModern = "max_completion_tokens"
	// MaxTokensFieldLegacy is the deprecated OpenAI output-limit field, still
	// the only one some self-hosted servers understand.
	MaxTokensFieldLegacy = "max_tokens"
)

// Providers returns the supported provider identifiers, for validation messages
// and help text.
func Providers() []string {
	return []string{ProviderAnthropic, ProviderOpenAI}
}

// MaxTokensFields returns the supported LLMConfig.MaxTokensField values.
func MaxTokensFields() []string {
	return []string{MaxTokensFieldModern, MaxTokensFieldLegacy}
}

// DefaultAPIKeyEnv returns the conventional environment variable holding the
// API key for a provider.
func DefaultAPIKeyEnv(provider string) string {
	if provider == ProviderOpenAI {
		return "OPENAI_API_KEY"
	}
	return "ANTHROPIC_API_KEY"
}

// LLMConfig is the resolved language-model configuration for one body: the
// global block with any per-body override already applied.
type LLMConfig struct {
	// Provider selects the wire protocol. See ProviderAnthropic/ProviderOpenAI.
	Provider string `yaml:"provider" mapstructure:"provider"`
	// Model is the model identifier, passed to the provider verbatim.
	Model string `yaml:"model" mapstructure:"model"`
	// BaseURL overrides the provider's default endpoint. Empty means use the
	// provider default.
	BaseURL string `yaml:"base_url" mapstructure:"base_url"`
	// APIKeyEnv names the environment variable holding the API key. The key
	// itself is never stored in configuration.
	APIKeyEnv string `yaml:"api_key_env" mapstructure:"api_key_env"`
	// MaxTokens caps the response. On models that reason before answering it
	// also covers reasoning tokens, so it needs headroom beyond the summary.
	MaxTokens int `yaml:"max_tokens" mapstructure:"max_tokens"`
	// MaxTokensField selects which OpenAI output-limit field to send. Ignored
	// by the Anthropic provider.
	MaxTokensField string `yaml:"max_tokens_field" mapstructure:"max_tokens_field"`
	// Temperature is sent only when non-nil. Current Claude models reject
	// sampling parameters with HTTP 400, so it stays unset unless configured.
	Temperature *float64 `yaml:"temperature" mapstructure:"temperature"`
	// TimeoutSeconds bounds a single request, including SDK-level retries.
	TimeoutSeconds int `yaml:"timeout_seconds" mapstructure:"timeout_seconds"`
	// MaxRetries is how many times the provider SDK retries transient failures.
	MaxRetries int `yaml:"max_retries" mapstructure:"max_retries"`
	// Stream requests a streamed response, which avoids HTTP timeouts on large
	// outputs. Disable it for compatible servers with unreliable SSE support.
	Stream bool `yaml:"stream" mapstructure:"stream"`
	// SystemPrompt is sent as the system message. Empty means send none, which
	// matches the historical behaviour of piping the whole prompt as one turn.
	SystemPrompt string `yaml:"system_prompt" mapstructure:"system_prompt"`
}

// Timeout returns TimeoutSeconds as a duration. A non-positive value means no
// client-side timeout.
func (c LLMConfig) Timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return 0
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// Describe returns a short "provider/model" label for logs and status output.
// It never includes the API key.
func (c LLMConfig) Describe() string {
	return c.Provider + "/" + c.Model
}

// LLMOverride is a per-body override of the global LLM block. Every field is a
// pointer so that omitting one inherits the global value while setting one to
// its zero value — an empty base_url, or stream: false — is still honoured.
type LLMOverride struct {
	Provider       *string  `yaml:"provider" mapstructure:"provider"`
	Model          *string  `yaml:"model" mapstructure:"model"`
	BaseURL        *string  `yaml:"base_url" mapstructure:"base_url"`
	APIKeyEnv      *string  `yaml:"api_key_env" mapstructure:"api_key_env"`
	MaxTokens      *int     `yaml:"max_tokens" mapstructure:"max_tokens"`
	MaxTokensField *string  `yaml:"max_tokens_field" mapstructure:"max_tokens_field"`
	Temperature    *float64 `yaml:"temperature" mapstructure:"temperature"`
	TimeoutSeconds *int     `yaml:"timeout_seconds" mapstructure:"timeout_seconds"`
	MaxRetries     *int     `yaml:"max_retries" mapstructure:"max_retries"`
	Stream         *bool    `yaml:"stream" mapstructure:"stream"`
	SystemPrompt   *string  `yaml:"system_prompt" mapstructure:"system_prompt"`
}

// Apply returns base with every field set on o overriding it. A nil override
// returns base unchanged, which is the common case of a body with no llm block.
func (o *LLMOverride) Apply(base LLMConfig) LLMConfig {
	if o == nil {
		return base
	}

	merged := base
	override(&merged.Provider, o.Provider)
	override(&merged.Model, o.Model)
	override(&merged.BaseURL, o.BaseURL)
	override(&merged.APIKeyEnv, o.APIKeyEnv)
	override(&merged.MaxTokens, o.MaxTokens)
	override(&merged.MaxTokensField, o.MaxTokensField)
	override(&merged.TimeoutSeconds, o.TimeoutSeconds)
	override(&merged.MaxRetries, o.MaxRetries)
	override(&merged.Stream, o.Stream)
	override(&merged.SystemPrompt, o.SystemPrompt)

	// Temperature is itself optional, so an override replaces the pointer.
	if o.Temperature != nil {
		merged.Temperature = o.Temperature
	}

	return merged
}

// override assigns value to target when value is set.
func override[T any](target *T, value *T) {
	if value != nil {
		*target = *value
	}
}
