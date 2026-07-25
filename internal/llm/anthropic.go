package llm

import (
	"context"
	"errors"
	"strings"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicClient talks to any Anthropic-compatible POST /v1/messages endpoint.
type anthropicClient struct {
	cfg    domain.LLMConfig
	client anthropic.Client
}

// newAnthropicClient builds a client for the Anthropic Messages API.
func newAnthropicClient(cfg domain.LLMConfig, apiKey string) *anthropicClient {
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithMaxRetries(cfg.MaxRetries),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.Timeout() > 0 {
		opts = append(opts, option.WithRequestTimeout(cfg.Timeout()))
	}
	return &anthropicClient{cfg: cfg, client: anthropic.NewClient(opts...)}
}

// Describe returns a "provider/model" label.
func (c *anthropicClient) Describe() string {
	return c.cfg.Describe()
}

// Complete sends the prompt and returns the concatenated text blocks.
func (c *anthropicClient) Complete(ctx context.Context, prompt string) (string, error) {
	return c.complete(ctx, c.params(prompt, c.cfg.MaxTokens))
}

// Ping issues a one-token completion. Such a small budget is exhausted before
// any text is produced, so an empty response still proves that the credentials,
// model, and endpoint are usable.
func (c *anthropicClient) Ping(ctx context.Context) error {
	_, err := c.complete(ctx, c.params("ping", 1))
	var llmErr *Error
	if errors.As(err, &llmErr) && llmErr.Kind == KindEmptyResponse {
		return nil
	}
	return err
}

// params builds the request. Temperature is included only when configured,
// because current Claude models reject sampling parameters with HTTP 400.
func (c *anthropicClient) params(prompt string, maxTokens int) anthropic.MessageNewParams {
	params := anthropic.MessageNewParams{
		Model:     c.cfg.Model,
		MaxTokens: int64(maxTokens),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	}
	if c.cfg.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{{Text: c.cfg.SystemPrompt}}
	}
	if c.cfg.Temperature != nil {
		params.Temperature = anthropic.Float(*c.cfg.Temperature)
	}
	return params
}

// complete runs the request, streaming unless disabled.
func (c *anthropicClient) complete(ctx context.Context, params anthropic.MessageNewParams) (string, error) {
	if !c.cfg.Stream {
		msg, err := c.client.Messages.New(ctx, params)
		if err != nil {
			return "", classify(c.cfg, err)
		}
		return c.text(*msg)
	}

	stream := c.client.Messages.NewStreaming(ctx, params)
	var msg anthropic.Message
	for stream.Next() {
		if err := msg.Accumulate(stream.Current()); err != nil {
			return "", classify(c.cfg, err)
		}
	}
	if err := stream.Err(); err != nil {
		return "", classify(c.cfg, err)
	}
	return c.text(msg)
}

// text concatenates the text blocks of a message, ignoring thinking and tool
// blocks.
func (c *anthropicClient) text(msg anthropic.Message) (string, error) {
	var out strings.Builder
	for _, block := range msg.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			out.WriteString(text.Text)
		}
	}
	trimmed := strings.TrimSpace(out.String())
	if trimmed == "" {
		return "", emptyResponseError(c.cfg)
	}
	return trimmed, nil
}
