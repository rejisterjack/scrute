// Package provider selects a model API and keeps API keys out of error text.
package provider

import (
	"errors"
	"strings"
)

const (
	Anthropic             = "anthropic"
	OpenAI                = "openai"
	DefaultAnthropicModel = "claude-sonnet-5"
	DefaultOpenAIModel    = "gpt-5.4"
)

// DefaultModel is the built-in model for a provider when config leaves model empty.
func DefaultModel(name string) string {
	switch name {
	case OpenAI:
		return DefaultOpenAIModel
	default:
		return DefaultAnthropicModel
	}
}

// EnvVar is the environment variable that holds the caller's key.
func EnvVar(name string) string {
	switch name {
	case OpenAI:
		return "OPENAI_API_KEY"
	default:
		return "ANTHROPIC_API_KEY"
	}
}

// Redact removes a secret from an error string.
func Redact(err error, secret string) error {
	if err == nil || len(secret) < 8 {
		return err
	}
	msg := strings.ReplaceAll(err.Error(), secret, "[REDACTED]")
	if msg == err.Error() {
		return err
	}
	return errors.New(msg)
}
