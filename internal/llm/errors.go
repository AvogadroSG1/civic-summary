package llm

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
)

// Kind classifies an LLM failure independently of the provider that produced
// it, so callers can distinguish a misconfiguration from a transient outage.
type Kind int

// Failure kinds, ordered from configuration problems to transport problems.
const (
	// KindUnknown is a failure that could not be classified.
	KindUnknown Kind = iota
	// KindAuth is a rejected or missing credential.
	KindAuth
	// KindModelNotFound is an unknown or unavailable model identifier.
	KindModelNotFound
	// KindContextWindow means the prompt exceeded the model's context window.
	KindContextWindow
	// KindInvalidRequest is a request the provider rejected as malformed, such
	// as sending temperature to a model that no longer accepts it.
	KindInvalidRequest
	// KindRateLimit is a throttled request.
	KindRateLimit
	// KindServer is a provider-side failure or overload.
	KindServer
	// KindTransport is a network, TLS, or timeout failure with no HTTP response.
	KindTransport
	// KindEmptyResponse means the request succeeded but returned no text.
	KindEmptyResponse
)

// String returns a short lowercase label for the kind.
func (k Kind) String() string {
	switch k {
	case KindAuth:
		return "authentication"
	case KindModelNotFound:
		return "model not found"
	case KindContextWindow:
		return "context window exceeded"
	case KindInvalidRequest:
		return "invalid request"
	case KindRateLimit:
		return "rate limited"
	case KindServer:
		return "provider error"
	case KindTransport:
		return "transport error"
	case KindEmptyResponse:
		return "empty response"
	default:
		return "unknown error"
	}
}

// Error describes a failed LLM request in provider-independent terms.
type Error struct {
	// Kind is the classification used to decide whether a retry is worthwhile.
	Kind Kind
	// Provider and Model identify which endpoint failed.
	Provider string
	Model    string
	// Status is the HTTP status code, or 0 when no response was received.
	Status int
	// Hint is an actionable next step for the operator, when one is known.
	Hint string
	// Err is the underlying SDK or transport error.
	Err error
}

// Error implements the error interface.
func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s/%s: %s", e.Provider, e.Model, e.Kind)
	if e.Status != 0 {
		fmt.Fprintf(&b, " (HTTP %d)", e.Status)
	}
	if e.Hint != "" {
		fmt.Fprintf(&b, ": %s", e.Hint)
	}
	if e.Err != nil {
		fmt.Fprintf(&b, ": %v", e.Err)
	}
	return b.String()
}

// Unwrap exposes the underlying SDK or transport error.
func (e *Error) Unwrap() error { return e.Err }

// Permanent reports whether retrying the identical request could ever succeed.
// It satisfies the interface that retry.Do uses to skip pointless retries: a
// bad API key or a misspelled model will fail the same way every time, and in
// this pipeline each retry re-downloads and re-transcribes the video.
func (e *Error) Permanent() bool {
	switch e.Kind {
	case KindAuth, KindModelNotFound, KindContextWindow, KindInvalidRequest:
		return true
	default:
		return false
	}
}

// classify converts a provider SDK error into an *Error. The status code and
// response body are read from whichever SDK type is present; anything without
// an HTTP response is treated as a transport failure.
func classify(cfg domain.LLMConfig, err error) error {
	if err == nil {
		return nil
	}

	var status int
	var body string

	var anthropicErr *anthropic.Error
	var openaiErr *openai.Error
	switch {
	case errors.As(err, &anthropicErr):
		status = anthropicErr.StatusCode
		body = anthropicErr.RawJSON()
	case errors.As(err, &openaiErr):
		status = openaiErr.StatusCode
		body = openaiErr.Message
		if body == "" {
			body = openaiErr.Code
		}
	default:
		return &Error{
			Kind:     KindTransport,
			Provider: cfg.Provider,
			Model:    cfg.Model,
			Hint:     transportHint(cfg),
			Err:      err,
		}
	}

	e := &Error{
		Provider: cfg.Provider,
		Model:    cfg.Model,
		Status:   status,
		Err:      err,
	}

	// Some compatible servers report a bad model or an oversized prompt as a
	// generic 400, so the body is consulted before falling back to the status.
	switch {
	case isContextWindow(body):
		e.Kind = KindContextWindow
		e.Hint = "the transcript is too long for this model; use a larger-context model or lower max_tokens"
	case isModelNotFound(body), status == http.StatusNotFound:
		e.Kind = KindModelNotFound
		e.Hint = fmt.Sprintf("model %q is not available at this endpoint; check llm.model and llm.base_url", cfg.Model)
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		e.Kind = KindAuth
		e.Hint = fmt.Sprintf("check the API key in $%s", cfg.APIKeyEnv)
	case status == http.StatusTooManyRequests:
		e.Kind = KindRateLimit
	case status == http.StatusBadRequest, status == http.StatusUnprocessableEntity:
		e.Kind = KindInvalidRequest
		e.Hint = invalidRequestHint(cfg, body)
	case status >= 500:
		e.Kind = KindServer
	default:
		e.Kind = KindUnknown
	}

	return e
}

// transportHint points at the most likely cause of a connection failure.
func transportHint(cfg domain.LLMConfig) string {
	if cfg.BaseURL != "" {
		return fmt.Sprintf("could not reach %s; check llm.base_url and that the server is running", cfg.BaseURL)
	}
	return "could not reach the provider; check network connectivity"
}

// invalidRequestHint recognizes the rejections that are most likely to be
// configuration mistakes rather than genuine prompt problems.
func invalidRequestHint(cfg domain.LLMConfig, body string) string {
	lower := strings.ToLower(body)
	switch {
	case strings.Contains(lower, "temperature"),
		strings.Contains(lower, "top_p"),
		strings.Contains(lower, "top_k"):
		return "this model rejects sampling parameters; remove llm.temperature from your config"
	case strings.Contains(lower, "max_completion_tokens"), strings.Contains(lower, "max_tokens"):
		return fmt.Sprintf("this endpoint rejected the output-limit field; try llm.max_tokens_field: %v",
			domain.MaxTokensFields())
	case cfg.BaseURL != "":
		return fmt.Sprintf("%s rejected the request; confirm it is %s-compatible", cfg.BaseURL, cfg.Provider)
	default:
		return ""
	}
}

// isContextWindow reports whether an error body describes an oversized prompt.
func isContextWindow(body string) bool {
	lower := strings.ToLower(body)
	for _, marker := range []string{
		"context_length_exceeded",
		"context length",
		"context window",
		"prompt is too long",
		"too many tokens",
		"reduce the length",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// isModelNotFound reports whether an error body describes an unknown model.
// Compatible servers word this many ways ("model 'x' not found, try pulling it
// first", "The model 'x' does not exist"), so beyond the two machine-readable
// codes it looks for the word "model" alongside an absence marker.
func isModelNotFound(body string) bool {
	lower := strings.ToLower(body)
	for _, code := range []string{"model_not_found", "not_found_error"} {
		if strings.Contains(lower, code) {
			return true
		}
	}
	if !strings.Contains(lower, "model") {
		return false
	}
	for _, marker := range []string{"not found", "does not exist", "unknown", "unavailable"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// emptyResponseError reports a successful request that produced no text.
func emptyResponseError(cfg domain.LLMConfig) error {
	return &Error{
		Kind:     KindEmptyResponse,
		Provider: cfg.Provider,
		Model:    cfg.Model,
		Hint:     "the model returned no content; max_tokens may be too small to fit any output",
	}
}
