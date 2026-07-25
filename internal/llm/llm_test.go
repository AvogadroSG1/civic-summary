package llm_test

import (
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testKeyEnv is the environment variable tests use to supply an API key.
const testKeyEnv = "CIVIC_SUMMARY_TEST_API_KEY"

// baseConfig returns a valid config for provider, pointed at baseURL.
func baseConfig(provider, baseURL string) domain.LLMConfig {
	return domain.LLMConfig{
		Provider:       provider,
		Model:          "test-model",
		BaseURL:        baseURL,
		APIKeyEnv:      testKeyEnv,
		MaxTokens:      1024,
		MaxTokensField: domain.MaxTokensFieldModern,
		Stream:         true,
	}
}

// newTestClient builds a client against a test server, with the API key set for
// the duration of the test.
func newTestClient(t *testing.T, cfg domain.LLMConfig) llm.Client {
	t.Helper()
	t.Setenv(testKeyEnv, "test-key")
	client, err := llm.New(cfg)
	require.NoError(t, err)
	return client
}

func TestNew_UnknownProvider(t *testing.T) {
	t.Setenv(testKeyEnv, "test-key")

	_, err := llm.New(baseConfig("gemini", ""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown provider "gemini"`)
	assert.Contains(t, err.Error(), domain.ProviderAnthropic)
	assert.Contains(t, err.Error(), domain.ProviderOpenAI)
}

func TestNew_MissingAPIKey(t *testing.T) {
	t.Setenv(testKeyEnv, "")

	_, err := llm.New(baseConfig(domain.ProviderAnthropic, ""))

	require.Error(t, err)
	assert.Contains(t, err.Error(), testKeyEnv)
	assert.Contains(t, err.Error(), "is not set")
}

func TestNew_MissingModel(t *testing.T) {
	t.Setenv(testKeyEnv, "test-key")
	cfg := baseConfig(domain.ProviderAnthropic, "")
	cfg.Model = ""

	_, err := llm.New(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")
}

func TestNew_MissingAPIKeyEnv(t *testing.T) {
	cfg := baseConfig(domain.ProviderAnthropic, "")
	cfg.APIKeyEnv = ""

	_, err := llm.New(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key_env is required")
}

func TestNew_NonPositiveMaxTokens(t *testing.T) {
	t.Setenv(testKeyEnv, "test-key")
	cfg := baseConfig(domain.ProviderAnthropic, "")
	cfg.MaxTokens = 0

	_, err := llm.New(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_tokens must be positive")
}

func TestNew_Describe(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{domain.ProviderAnthropic, "anthropic/test-model"},
		{domain.ProviderOpenAI, "openai/test-model"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			client := newTestClient(t, baseConfig(tt.provider, ""))

			assert.Equal(t, tt.want, client.Describe())
		})
	}
}

func TestKind_String(t *testing.T) {
	tests := []struct {
		kind llm.Kind
		want string
	}{
		{llm.KindAuth, "authentication"},
		{llm.KindModelNotFound, "model not found"},
		{llm.KindContextWindow, "context window exceeded"},
		{llm.KindInvalidRequest, "invalid request"},
		{llm.KindRateLimit, "rate limited"},
		{llm.KindServer, "provider error"},
		{llm.KindTransport, "transport error"},
		{llm.KindEmptyResponse, "empty response"},
		{llm.KindUnknown, "unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.kind.String())
		})
	}
}

func TestError_Permanent(t *testing.T) {
	permanent := []llm.Kind{
		llm.KindAuth, llm.KindModelNotFound, llm.KindContextWindow, llm.KindInvalidRequest,
	}
	transient := []llm.Kind{
		llm.KindRateLimit, llm.KindServer, llm.KindTransport, llm.KindEmptyResponse, llm.KindUnknown,
	}

	for _, kind := range permanent {
		assert.True(t, (&llm.Error{Kind: kind}).Permanent(), "%s should be permanent", kind)
	}
	for _, kind := range transient {
		assert.False(t, (&llm.Error{Kind: kind}).Permanent(), "%s should be retryable", kind)
	}
}

func TestError_Message(t *testing.T) {
	err := &llm.Error{
		Kind:     llm.KindAuth,
		Provider: "anthropic",
		Model:    "claude-opus-5",
		Status:   401,
		Hint:     "check the API key in $ANTHROPIC_API_KEY",
	}

	msg := err.Error()

	assert.Contains(t, msg, "anthropic/claude-opus-5")
	assert.Contains(t, msg, "authentication")
	assert.Contains(t, msg, "HTTP 401")
	assert.Contains(t, msg, "$ANTHROPIC_API_KEY")
}
