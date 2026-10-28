// Package openai reviews a diff with the caller's OpenAI API key.
package openai

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/rejisterjack/scrute/internal/provider"
	"github.com/rejisterjack/scrute/internal/review"
)

// Client calls the OpenAI Chat Completions API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
	retries *int
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL points the client at a non-default endpoint.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient sets the HTTP client. Tests use this to reach a local server.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.http = httpClient }
}

// WithMaxRetries overrides the SDK retry count.
func WithMaxRetries(n int) Option {
	return func(c *Client) { c.retries = &n }
}

// New returns a client that sends apiKey on each request.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{apiKey: apiKey}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Complete requests a strict JSON schema review.
func (c *Client) Complete(ctx context.Context, model, system, user string) (raw string, inputTokens, outputTokens int, err error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", 0, 0, errors.New("API key is empty")
	}
	if model == "" {
		model = provider.DefaultOpenAIModel
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	sdk := openai.NewClient(c.options()...)
	resp, err := sdk.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage(user),
		},
		MaxCompletionTokens: openai.Int(8192),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   "scrute_review",
					Strict: openai.Bool(true),
					Schema: review.Schema(),
				},
			},
		},
	})
	if err != nil {
		return "", 0, 0, provider.Redact(err, c.apiKey)
	}
	if len(resp.Choices) == 0 {
		return "", int(resp.Usage.PromptTokens), int(resp.Usage.CompletionTokens), errors.New("model response did not include a review")
	}
	raw = resp.Choices[0].Message.Content
	if strings.TrimSpace(raw) == "" {
		return "", int(resp.Usage.PromptTokens), int(resp.Usage.CompletionTokens), errors.New("model response did not include a review")
	}
	return raw, int(resp.Usage.PromptTokens), int(resp.Usage.CompletionTokens), nil
}

func (c *Client) options() []option.RequestOption {
	opts := []option.RequestOption{option.WithAPIKey(c.apiKey)}
	if c.baseURL != "" {
		opts = append(opts, option.WithBaseURL(c.baseURL))
	}
	if c.http != nil {
		opts = append(opts, option.WithHTTPClient(c.http))
	}
	if c.retries != nil {
		opts = append(opts, option.WithMaxRetries(*c.retries))
	}
	return opts
}
