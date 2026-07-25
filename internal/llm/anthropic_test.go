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

// anthropicStreamBody renders text as a well-formed Messages API SSE stream.
// The decoder dispatches on blank lines and switches on the event name, so both
// are required.
func anthropicStreamBody(text string) string {
	var body strings.Builder
	write := func(event, data string) {
		fmt.Fprintf(&body, "event: %s\ndata: %s\n\n", event, data)
	}

	write("message_start", `{"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","model":"test-model","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`)
	if text != "" {
		write("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		write("content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%s}}`, mustJSON(text)))
		write("content_block_stop", `{"type":"content_block_stop","index":0}`)
	}
	write("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":20}}`)
	write("message_stop", `{"type":"message_stop"}`)

	return body.String()
}

// anthropicMessageBody renders text as a non-streaming Messages API response.
func anthropicMessageBody(text string) string {
	content := "[]"
	if text != "" {
		content = fmt.Sprintf(`[{"type":"text","text":%s}]`, mustJSON(text))
	}
	return fmt.Sprintf(`{"id":"msg_test","type":"message","role":"assistant","model":"test-model","content":%s,"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":20}}`, content)
}

// mustJSON encodes v as JSON for embedding in a fixture body.
func mustJSON(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

// anthropicServer serves a fixed body and records the last request body.
func anthropicServer(t *testing.T, status int, body string, stream bool) (*httptest.Server, *map[string]any) {
	t.Helper()
	captured := map[string]any{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))

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

func TestAnthropicComplete_Streaming(t *testing.T) {
	srv, captured := anthropicServer(t, http.StatusOK, anthropicStreamBody("# Summary\n\nBody text."), true)
	client := newTestClient(t, baseConfig(domain.ProviderAnthropic, srv.URL))

	out, err := client.Complete(context.Background(), "the rendered template")

	require.NoError(t, err)
	assert.Equal(t, "# Summary\n\nBody text.", out)
	assert.Equal(t, "test-model", (*captured)["model"])
	assert.Equal(t, float64(1024), (*captured)["max_tokens"])
	assert.Equal(t, true, (*captured)["stream"])
}

func TestAnthropicComplete_NonStreaming(t *testing.T) {
	srv, captured := anthropicServer(t, http.StatusOK, anthropicMessageBody("summary text"), false)
	cfg := baseConfig(domain.ProviderAnthropic, srv.URL)
	cfg.Stream = false
	client := newTestClient(t, cfg)

	out, err := client.Complete(context.Background(), "prompt")

	require.NoError(t, err)
	assert.Equal(t, "summary text", out)
	assert.NotContains(t, *captured, "stream")
}

// TestAnthropicComplete_SendsPromptAsUserMessage locks in the mapping that used
// to be untestable: the whole rendered template goes in one user message.
func TestAnthropicComplete_SendsPromptAsUserMessage(t *testing.T) {
	srv, captured := anthropicServer(t, http.StatusOK, anthropicStreamBody("ok"), true)
	client := newTestClient(t, baseConfig(domain.ProviderAnthropic, srv.URL))

	_, err := client.Complete(context.Background(), "TRANSCRIPT MARKER")
	require.NoError(t, err)

	messages, ok := (*captured)["messages"].([]any)
	require.True(t, ok, "messages should be an array")
	require.Len(t, messages, 1)

	message := messages[0].(map[string]any)
	assert.Equal(t, "user", message["role"])
	assert.Contains(t, mustJSON(message["content"]), "TRANSCRIPT MARKER")
	assert.NotContains(t, *captured, "system")
}

func TestAnthropicComplete_SendsSystemPrompt(t *testing.T) {
	srv, captured := anthropicServer(t, http.StatusOK, anthropicStreamBody("ok"), true)
	cfg := baseConfig(domain.ProviderAnthropic, srv.URL)
	cfg.SystemPrompt = "You are terse."
	client := newTestClient(t, cfg)

	_, err := client.Complete(context.Background(), "prompt")
	require.NoError(t, err)

	assert.Contains(t, mustJSON((*captured)["system"]), "You are terse.")
}

// TestAnthropicComplete_OmitsTemperatureByDefault guards the constraint that
// current Claude models reject sampling parameters with HTTP 400.
func TestAnthropicComplete_OmitsTemperatureByDefault(t *testing.T) {
	srv, captured := anthropicServer(t, http.StatusOK, anthropicStreamBody("ok"), true)
	client := newTestClient(t, baseConfig(domain.ProviderAnthropic, srv.URL))

	_, err := client.Complete(context.Background(), "prompt")

	require.NoError(t, err)
	assert.NotContains(t, *captured, "temperature")
}

func TestAnthropicComplete_SendsConfiguredTemperature(t *testing.T) {
	srv, captured := anthropicServer(t, http.StatusOK, anthropicStreamBody("ok"), true)
	cfg := baseConfig(domain.ProviderAnthropic, srv.URL)
	temperature := 0.4
	cfg.Temperature = &temperature
	client := newTestClient(t, cfg)

	_, err := client.Complete(context.Background(), "prompt")

	require.NoError(t, err)
	assert.Equal(t, 0.4, (*captured)["temperature"])
}

func TestAnthropicComplete_EmptyResponse(t *testing.T) {
	srv, _ := anthropicServer(t, http.StatusOK, anthropicStreamBody(""), true)
	client := newTestClient(t, baseConfig(domain.ProviderAnthropic, srv.URL))

	_, err := client.Complete(context.Background(), "prompt")

	llmErr := requireKind(t, err, llm.KindEmptyResponse)
	assert.False(t, llmErr.Permanent(), "an empty response is worth retrying")
	assert.Contains(t, llmErr.Hint, "max_tokens")
}

func TestAnthropicComplete_ErrorClassification(t *testing.T) {
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
			body:     `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`,
			wantKind: llm.KindAuth,
			wantPerm: true,
			wantHint: "$" + testKeyEnv,
		},
		{
			name:     "model not found",
			status:   http.StatusNotFound,
			body:     `{"type":"error","error":{"type":"not_found_error","message":"model: test-model"}}`,
			wantKind: llm.KindModelNotFound,
			wantPerm: true,
			wantHint: "llm.model",
		},
		{
			name:     "prompt too long",
			status:   http.StatusBadRequest,
			body:     `{"type":"error","error":{"type":"invalid_request_error","message":"prompt is too long: 300000 tokens > 200000 maximum"}}`,
			wantKind: llm.KindContextWindow,
			wantPerm: true,
			wantHint: "larger-context model",
		},
		{
			name:     "temperature rejected",
			status:   http.StatusBadRequest,
			body:     `{"type":"error","error":{"type":"invalid_request_error","message":"temperature: Extra inputs are not permitted"}}`,
			wantKind: llm.KindInvalidRequest,
			wantPerm: true,
			wantHint: "remove llm.temperature",
		},
		{
			name:     "rate limited",
			status:   http.StatusTooManyRequests,
			body:     `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`,
			wantKind: llm.KindRateLimit,
			wantPerm: false,
		},
		{
			name:     "overloaded",
			status:   http.StatusServiceUnavailable,
			body:     `{"type":"error","error":{"type":"overloaded_error","message":"overloaded"}}`,
			wantKind: llm.KindServer,
			wantPerm: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := anthropicServer(t, tt.status, tt.body, false)
			client := newTestClient(t, baseConfig(domain.ProviderAnthropic, srv.URL))

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

func TestAnthropicComplete_TransportError(t *testing.T) {
	cfg := baseConfig(domain.ProviderAnthropic, "http://127.0.0.1:1")
	client := newTestClient(t, cfg)

	_, err := client.Complete(context.Background(), "prompt")

	llmErr := requireKind(t, err, llm.KindTransport)
	assert.False(t, llmErr.Permanent())
	assert.Contains(t, llmErr.Hint, "llm.base_url")
}

// TestAnthropicPing_UsesOneToken checks the probe stays cheap and treats a
// truncated (therefore empty) response as success.
func TestAnthropicPing_UsesOneToken(t *testing.T) {
	srv, captured := anthropicServer(t, http.StatusOK, anthropicStreamBody(""), true)
	client := newTestClient(t, baseConfig(domain.ProviderAnthropic, srv.URL))

	err := client.Ping(context.Background())

	require.NoError(t, err)
	assert.Equal(t, float64(1), (*captured)["max_tokens"])
}

func TestAnthropicPing_ReportsAuthFailure(t *testing.T) {
	srv, _ := anthropicServer(t, http.StatusUnauthorized,
		`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`, false)
	client := newTestClient(t, baseConfig(domain.ProviderAnthropic, srv.URL))

	err := client.Ping(context.Background())

	llmErr := requireKind(t, err, llm.KindAuth)
	assert.Contains(t, llmErr.Hint, "$"+testKeyEnv)
}

// requireKind asserts that err is an *llm.Error of the given kind and returns it.
func requireKind(t *testing.T, err error, want llm.Kind) *llm.Error {
	t.Helper()
	require.Error(t, err)

	var llmErr *llm.Error
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, want, llmErr.Kind, "unexpected kind: %v", llmErr)
	return llmErr
}
