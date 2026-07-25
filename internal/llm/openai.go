package llm

import (
	"context"
	"errors"
	"strings"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// openaiClient talks to any OpenAI-compatible POST /v1/chat/completions
// endpoint, which covers OpenAI itself as well as gateways and self-hosted
// servers that implement the same route.
type openaiClient struct {
	cfg    domain.LLMConfig
	client openai.Client
}

// newOpenAIClient builds a client for the Chat Completions API.
func newOpenAIClient(cfg domain.LLMConfig, apiKey string) *openaiClient {
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
	return &openaiClient{cfg: cfg, client: openai.NewClient(opts...)}
}

// Describe returns a "provider/model" label.
func (c *openaiClient) Describe() string {
	return c.cfg.Describe()
}

// Complete sends the prompt and returns the first choice's message content.
func (c *openaiClient) Complete(ctx context.Context, prompt string) (string, error) {
	return c.complete(ctx, c.params(prompt, c.cfg.MaxTokens))
}

// Ping issues a one-token completion; see anthropicClient.Ping for why an empty
// response counts as success.
func (c *openaiClient) Ping(ctx context.Context) error {
	_, err := c.complete(ctx, c.params("ping", 1))
	var llmErr *Error
	if errors.As(err, &llmErr) && llmErr.Kind == KindEmptyResponse {
		return nil
	}
	return err
}

// params builds the request. The output limit goes in whichever field the
// endpoint understands: OpenAI's reasoning models require
// max_completion_tokens, while some self-hosted servers only accept max_tokens.
func (c *openaiClient) params(prompt string, maxTokens int) openai.ChatCompletionNewParams {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, 2)
	if c.cfg.SystemPrompt != "" {
		messages = append(messages, openai.SystemMessage(c.cfg.SystemPrompt))
	}
	messages = append(messages, openai.UserMessage(prompt))

	params := openai.ChatCompletionNewParams{
		Model:    c.cfg.Model,
		Messages: messages,
	}
	if c.cfg.MaxTokensField == domain.MaxTokensFieldLegacy {
		params.MaxTokens = openai.Int(int64(maxTokens))
	} else {
		params.MaxCompletionTokens = openai.Int(int64(maxTokens))
	}
	if c.cfg.Temperature != nil {
		params.Temperature = openai.Float(*c.cfg.Temperature)
	}
	return params
}

// complete runs the request, streaming unless disabled.
func (c *openaiClient) complete(ctx context.Context, params openai.ChatCompletionNewParams) (string, error) {
	if !c.cfg.Stream {
		completion, err := c.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return "", classify(c.cfg, err)
		}
		return c.text(*completion)
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)
	var acc openai.ChatCompletionAccumulator
	for stream.Next() {
		acc.AddChunk(stream.Current())
	}
	if err := stream.Err(); err != nil {
		return "", classify(c.cfg, err)
	}
	return c.text(acc.ChatCompletion)
}

// text extracts the first choice's content.
func (c *openaiClient) text(completion openai.ChatCompletion) (string, error) {
	if len(completion.Choices) == 0 {
		return "", emptyResponseError(c.cfg)
	}
	trimmed := strings.TrimSpace(completion.Choices[0].Message.Content)
	if trimmed == "" {
		return "", emptyResponseError(c.cfg)
	}
	return trimmed, nil
}
