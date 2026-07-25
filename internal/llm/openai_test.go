package llm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openaiStreamBody renders text as a Chat Completions SSE stream. An empty text
// produces a stream with no content delta, as happens when the output limit is
// exhausted before any token is emitted.
func openaiStreamBody(text string) string {
	var body strings.Builder
	write := func(data string) {
		fmt.Fprintf(&body, "data: %s\n\n", data)
	}

	if text != "" {
		write(fmt.Sprintf(`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"test-model","choices":[{"index":0,"delta":{"role":"assistant","content":%s},"finish_reason":null}]}`, mustJSON(text)))
		write(`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"test-model","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
	} else {
		write(`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"test-model","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":"length"}]}`)
	}
	write("[DONE]")

	return body.String()
}

// openaiCompletionBody renders text as a non-streaming Chat Completions response.
func openaiCompletionBody(text string) string {
	return fmt.Sprintf(`{"id":"c1","object":"chat.completion","created":1,"model":"test-model","choices":[{"index":0,"message":{"role":"assistant","content":%s},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`, mustJSON(text))
}

// openaiServer serves a fixed body and records the last request body.
func openaiServer(t *testing.T, status int, body string, stream bool) (*httptest.Server, *map[string]any) {
	t.Helper()
	captured := map[string]any{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("decoding request body: %v", err)
		}

		if stream && status == http.StatusOK {
			w.Header().Set("content-type", "text/event-stream")
		} else {
			w.Header().Set("content-type", "application/json")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv, &captured
}

// openaiBaseURL returns the base URL a user would configure for a compatible
// server. The trailing slash is deliberately omitted: the SDK adds it, which is
// what stops the last path segment from being dropped during URL resolution.
func openaiBaseURL(srv *httptest.Server) string {
	return srv.URL + "/v1"
}

func TestOpenAIComplete_Streaming(t *testing.T) {
	srv, captured := openaiServer(t, http.StatusOK, openaiStreamBody("# Summary\n\nBody text."), true)
	client := newTestClient(t, baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv)))

	out, err := client.Complete(context.Background(), "the rendered template")

	require.NoError(t, err)
	assert.Equal(t, "# Summary\n\nBody text.", out)
	assert.Equal(t, "test-model", (*captured)["model"])
	assert.Equal(t, true, (*captured)["stream"])
}

func TestOpenAIComplete_NonStreaming(t *testing.T) {
	srv, captured := openaiServer(t, http.StatusOK, openaiCompletionBody("summary text"), false)
	cfg := baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv))
	cfg.Stream = false
	client := newTestClient(t, cfg)

	out, err := client.Complete(context.Background(), "prompt")

	require.NoError(t, err)
	assert.Equal(t, "summary text", out)
	assert.NotContains(t, *captured, "stream")
}

func TestOpenAIComplete_SendsPromptAsUserMessage(t *testing.T) {
	srv, captured := openaiServer(t, http.StatusOK, openaiStreamBody("ok"), true)
	client := newTestClient(t, baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv)))

	_, err := client.Complete(context.Background(), "TRANSCRIPT MARKER")
	require.NoError(t, err)

	messages, ok := (*captured)["messages"].([]any)
	require.True(t, ok, "messages should be an array")
	require.Len(t, messages, 1)

	message := messages[0].(map[string]any)
	assert.Equal(t, "user", message["role"])
	assert.Contains(t, mustJSON(message["content"]), "TRANSCRIPT MARKER")
}

func TestOpenAIComplete_SendsSystemMessageFirst(t *testing.T) {
	srv, captured := openaiServer(t, http.StatusOK, openaiStreamBody("ok"), true)
	cfg := baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv))
	cfg.SystemPrompt = "You are terse."
	client := newTestClient(t, cfg)

	_, err := client.Complete(context.Background(), "prompt")
	require.NoError(t, err)

	messages := (*captured)["messages"].([]any)
	require.Len(t, messages, 2)
	assert.Equal(t, "system", messages[0].(map[string]any)["role"])
	assert.Equal(t, "user", messages[1].(map[string]any)["role"])
}

// TestOpenAIComplete_MaxTokensField covers both output-limit spellings, since
// OpenAI's reasoning models require the modern field while some self-hosted
// servers only understand the legacy one.
func TestOpenAIComplete_MaxTokensField(t *testing.T) {
	tests := []struct {
		field    string
		wantKey  string
		otherKey string
	}{
		{domain.MaxTokensFieldModern, "max_completion_tokens", "max_tokens"},
		{domain.MaxTokensFieldLegacy, "max_tokens", "max_completion_tokens"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			srv, captured := openaiServer(t, http.StatusOK, openaiStreamBody("ok"), true)
			cfg := baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv))
			cfg.MaxTokensField = tt.field
			client := newTestClient(t, cfg)

			_, err := client.Complete(context.Background(), "prompt")

			require.NoError(t, err)
			assert.Equal(t, float64(1024), (*captured)[tt.wantKey])
			assert.NotContains(t, *captured, tt.otherKey)
		})
	}
}

func TestOpenAIComplete_OmitsTemperatureByDefault(t *testing.T) {
	srv, captured := openaiServer(t, http.StatusOK, openaiStreamBody("ok"), true)
	client := newTestClient(t, baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv)))

	_, err := client.Complete(context.Background(), "prompt")

	require.NoError(t, err)
	assert.NotContains(t, *captured, "temperature")
}

func TestOpenAIComplete_SendsConfiguredTemperature(t *testing.T) {
	srv, captured := openaiServer(t, http.StatusOK, openaiStreamBody("ok"), true)
	cfg := baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv))
	temperature := 0.7
	cfg.Temperature = &temperature
	client := newTestClient(t, cfg)

	_, err := client.Complete(context.Background(), "prompt")

	require.NoError(t, err)
	assert.Equal(t, 0.7, (*captured)["temperature"])
}

func TestOpenAIComplete_EmptyResponse(t *testing.T) {
	srv, _ := openaiServer(t, http.StatusOK, openaiStreamBody(""), true)
	client := newTestClient(t, baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv)))

	_, err := client.Complete(context.Background(), "prompt")

	llmErr := requireKind(t, err, llm.KindEmptyResponse)
	assert.False(t, llmErr.Permanent(), "an empty response is worth retrying")
	assert.Contains(t, llmErr.Hint, "max_tokens")
}

func TestOpenAIComplete_ErrorClassification(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantKind llm.Kind
		wantPerm bool
		wantHint string
	}{
		{
			name:     "unauthorized",
			status:   http.StatusUnauthorized,
			body:     `{"error":{"message":"Incorrect API key provided","type":"invalid_request_error","code":"invalid_api_key"}}`,
			wantKind: llm.KindAuth,
			wantPerm: true,
			wantHint: "$" + testKeyEnv,
		},
		{
			name:     "model not found",
			status:   http.StatusNotFound,
			body:     `{"error":{"message":"The model 'test-model' does not exist","type":"invalid_request_error","code":"model_not_found"}}`,
			wantKind: llm.KindModelNotFound,
			wantPerm: true,
			wantHint: "llm.base_url",
		},
		{
			name:     "model not found reported as 400",
			status:   http.StatusBadRequest,
			body:     `{"error":{"message":"model 'test-model' not found, try pulling it first","type":"api_error"}}`,
			wantKind: llm.KindModelNotFound,
			wantPerm: true,
		},
		{
			name:     "context length exceeded",
			status:   http.StatusBadRequest,
			body:     `{"error":{"message":"This model's maximum context length is 8192 tokens","type":"invalid_request_error","code":"context_length_exceeded"}}`,
			wantKind: llm.KindContextWindow,
			wantPerm: true,
			wantHint: "larger-context model",
		},
		{
			name:     "legacy max_tokens rejected",
			status:   http.StatusBadRequest,
			body:     `{"error":{"message":"Unsupported parameter: 'max_tokens' is not supported with this model","type":"invalid_request_error","code":"unsupported_parameter"}}`,
			wantKind: llm.KindInvalidRequest,
			wantPerm: true,
			wantHint: "max_tokens_field",
		},
		{
			name:     "rate limited",
			status:   http.StatusTooManyRequests,
			body:     `{"error":{"message":"Rate limit reached","type":"requests","code":"rate_limit_exceeded"}}`,
			wantKind: llm.KindRateLimit,
			wantPerm: false,
		},
		{
			name:     "server error",
			status:   http.StatusInternalServerError,
			body:     `{"error":{"message":"internal error","type":"server_error"}}`,
			wantKind: llm.KindServer,
			wantPerm: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := openaiServer(t, tt.status, tt.body, false)
			client := newTestClient(t, baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv)))

			_, err := client.Complete(context.Background(), "prompt")

			llmErr := requireKind(t, err, tt.wantKind)
			assert.Equal(t, tt.wantPerm, llmErr.Permanent())
			assert.Equal(t, tt.status, llmErr.Status)
			if tt.wantHint != "" {
				assert.Contains(t, llmErr.Hint, tt.wantHint)
			}
		})
	}
}

func TestOpenAIComplete_TransportError(t *testing.T) {
	client := newTestClient(t, baseConfig(domain.ProviderOpenAI, "http://127.0.0.1:1/v1/"))

	_, err := client.Complete(context.Background(), "prompt")

	llmErr := requireKind(t, err, llm.KindTransport)
	assert.False(t, llmErr.Permanent())
	assert.Contains(t, llmErr.Hint, "llm.base_url")
}

func TestOpenAIPing_UsesOneToken(t *testing.T) {
	srv, captured := openaiServer(t, http.StatusOK, openaiStreamBody(""), true)
	client := newTestClient(t, baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv)))

	err := client.Ping(context.Background())

	require.NoError(t, err)
	assert.Equal(t, float64(1), (*captured)["max_completion_tokens"])
}

func TestOpenAIPing_ReportsAuthFailure(t *testing.T) {
	srv, _ := openaiServer(t, http.StatusUnauthorized,
		`{"error":{"message":"Incorrect API key provided","type":"invalid_request_error","code":"invalid_api_key"}}`, false)
	client := newTestClient(t, baseConfig(domain.ProviderOpenAI, openaiBaseURL(srv)))

	err := client.Ping(context.Background())

	llmErr := requireKind(t, err, llm.KindAuth)
	assert.Contains(t, llmErr.Hint, "$"+testKeyEnv)
}
