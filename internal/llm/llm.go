// Package llm provides a provider-agnostic client for large language model
// completions. Two wire protocols are supported: Anthropic-compatible
// (POST /v1/messages) and OpenAI-compatible (POST /v1/chat/completions).
//
// Because both are reachable at an arbitrary base URL, the same two
// implementations cover first-party APIs, gateways such as OpenRouter or Groq,
// and self-hosted servers such as Ollama, vLLM, and LM Studio.
package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
)

// Client generates a summary from a rendered prompt.
type Client interface {
	// Complete sends the rendered prompt as the user message and returns the
	// model's text response. The system message, if any, comes from the
	// client's configuration.
	Complete(ctx context.Context, prompt string) (string, error)

	// Ping issues a minimal completion to verify that the credentials, model,
	// and base URL are usable. It makes one real (but negligibly small)
	// request.
	Ping(ctx context.Context) error

	// Describe returns a short "provider/model" label for logs and status
	// output. It never includes the API key.
	Describe() string
}

// New builds a Client for cfg, reading the API key from the environment
// variable named by cfg.APIKeyEnv. The key is read here rather than during
// configuration loading so that commands which never call a model — listing
// bodies, inspecting quarantine — keep working without credentials.
func New(cfg domain.LLMConfig) (Client, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("llm: model is required")
	}
	if cfg.MaxTokens <= 0 {
		return nil, fmt.Errorf("llm: max_tokens must be positive, got %d", cfg.MaxTokens)
	}
	if cfg.APIKeyEnv == "" {
		return nil, fmt.Errorf("llm: api_key_env is required")
	}

	apiKey := os.Getenv(cfg.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("llm: %s is not set; export it with your %s API key",
			cfg.APIKeyEnv, cfg.Provider)
	}

	switch cfg.Provider {
	case domain.ProviderAnthropic:
		return newAnthropicClient(cfg, apiKey), nil
	case domain.ProviderOpenAI:
		return newOpenAIClient(cfg, apiKey), nil
	default:
		return nil, fmt.Errorf("llm: unknown provider %q; supported: %v", cfg.Provider, domain.Providers())
	}
}
