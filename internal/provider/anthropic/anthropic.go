// Package anthropic reviews a diff with the caller's Anthropic API key.
package anthropic

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/rejisterjack/scrute/internal/provider"
	"github.com/rejisterjack/scrute/internal/review"
)

// Client calls the Anthropic Messages API.
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

// Complete requests a JSON schema review. Current Claude models reject a forced
// tool_choice, so the schema travels in output_config.format and the review is
// the text block of the response.
func (c *Client) Complete(ctx context.Context, model, system, user string) (raw string, inputTokens, outputTokens int, err error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", 0, 0, errors.New("API key is empty")
	}
	if model == "" {
		model = provider.DefaultAnthropicModel
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	sdk := anthropic.NewClient(c.options()...)
	message, err := sdk.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 16000,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{Schema: review.Schema()},
		},
	})
	if err != nil {
		return "", 0, 0, provider.Redact(err, c.apiKey)
	}
	raw, err = reviewJSON(message)
	if err != nil {
		return "", int(message.Usage.InputTokens), int(message.Usage.OutputTokens), provider.Redact(err, c.apiKey)
	}
	return raw, int(message.Usage.InputTokens), int(message.Usage.OutputTokens), nil
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

func reviewJSON(message *anthropic.Message) (string, error) {
	if message == nil {
		return "", errors.New("empty model response")
	}
	for _, block := range message.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return block.Text, nil
		}
	}
	return "", errors.New("model response did not include a review")
}
