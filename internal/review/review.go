// Package review turns a staged diff into a structured model review.
package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	Critical   = "critical"
	Warning    = "warning"
	Suggestion = "suggestion"
)

// Review is the structured result returned by a model.
type Review struct {
	Summary string  `json:"summary"`
	Issues  []Issue `json:"issues"`
}

// Issue is one finding anchored to a new-file line.
type Issue struct {
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

// UnparsedError is returned when the model response is still invalid after one retry.
type UnparsedError struct {
	Raw string
}

func (e *UnparsedError) Error() string {
	return "unparsed model response:\n" + e.Raw
}

// Schema is the JSON schema sent to the provider.
func Schema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"summary", "issues"},
		"properties": map[string]any{
			"summary": map[string]any{"type": "string"},
			"issues": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"severity", "file", "line", "title", "detail"},
					"properties": map[string]any{
						"severity": map[string]any{
							"type": "string",
							"enum": []string{Critical, Warning, Suggestion},
						},
						"file":   map[string]any{"type": "string"},
						"line":   map[string]any{"type": "integer"},
						"title":  map[string]any{"type": "string"},
						"detail": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

// Parse validates a model JSON payload.
func Parse(raw string) (Review, error) {
	raw = stripFence(strings.TrimSpace(raw))
	var rev Review
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rev); err != nil {
		return Review{}, fmt.Errorf("invalid review JSON: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Review{}, fmt.Errorf("invalid review JSON: extra data")
		}
		return Review{}, fmt.Errorf("invalid review JSON: %w", err)
	}
	if strings.TrimSpace(rev.Summary) == "" {
		return Review{}, fmt.Errorf("review summary is empty")
	}
	if rev.Issues == nil {
		rev.Issues = []Issue{}
	}
	for i, issue := range rev.Issues {
		switch issue.Severity {
		case Critical, Warning, Suggestion:
		default:
			return Review{}, fmt.Errorf("issue %d: severity must be critical, warning, or suggestion", i)
		}
		if strings.TrimSpace(issue.File) == "" {
			return Review{}, fmt.Errorf("issue %d: file is empty", i)
		}
		if issue.Line < 0 {
			return Review{}, fmt.Errorf("issue %d: line must be zero or greater", i)
		}
		if strings.TrimSpace(issue.Title) == "" || strings.TrimSpace(issue.Detail) == "" {
			return Review{}, fmt.Errorf("issue %d: title and detail are required", i)
		}
	}
	return rev, nil
}

func stripFence(raw string) string {
	if !strings.HasPrefix(raw, "```") {
		return raw
	}
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimPrefix(raw, "json")
	raw = strings.TrimPrefix(raw, "\n")
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
	return strings.TrimSpace(raw)
}

// Rank orders severities. Lower is more severe.
func Rank(severity string) int {
	switch severity {
	case Critical:
		return 0
	case Warning:
		return 1
	case Suggestion:
		return 2
	default:
		return 3
	}
}

// Failed reports whether the review should exit 1 for failOn.
func Failed(rev Review, failOn string) bool {
	if failOn == "none" || failOn == "" {
		return false
	}
	limit := Rank(failOn)
	for _, issue := range rev.Issues {
		if Rank(issue.Severity) <= limit {
			return true
		}
	}
	return false
}
