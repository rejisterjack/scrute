// Package render prints a review to the terminal, markdown, or JSON.
package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rejisterjack/scrute/internal/review"
)

// View is the finished review prepared for display.
type View struct {
	Provider   string
	Model      string
	Review     review.Review
	Omitted    []string
	Binaries   []string
	Submodules []string
	Reviewed   bool
	Color      bool
}

// Format renders view as term, markdown, or json.
func Format(format string, view View) (string, error) {
	switch normalize(format) {
	case "term", "":
		return terminal(view), nil
	case "markdown":
		return markdown(view), nil
	case "json":
		return asJSON(view)
	default:
		return "", fmt.Errorf("format must be term, markdown, or json, got %q", format)
	}
}

func normalize(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "terminal", "text":
		return "term"
	case "md":
		return "markdown"
	default:
		return strings.ToLower(strings.TrimSpace(format))
	}
}

func terminal(view View) string {
	var b strings.Builder
	fmt.Fprintf(&b, "scrute  %s/%s\n", view.Provider, view.Model)
	b.WriteByte('\n')
	if strings.TrimSpace(view.Review.Summary) != "" {
		b.WriteString(paint(view.Color, "1", "Summary"))
		b.WriteByte('\n')
		b.WriteString(indent(view.Review.Summary, "  "))
		b.WriteString("\n\n")
	}
	issues := sorted(view.Review.Issues)
	if len(issues) == 0 {
		if view.Reviewed {
			b.WriteString("No issues.\n")
		} else {
			b.WriteString("Nothing to review.\n")
		}
	} else {
		var last string
		for _, issue := range issues {
			if issue.Severity != last {
				if last != "" {
					b.WriteByte('\n')
				}
				b.WriteString(paint(view.Color, severityColor(issue.Severity), severityTitle(issue.Severity)))
				b.WriteByte('\n')
				last = issue.Severity
			}
			fmt.Fprintf(&b, "  %s:%d  %s\n", issue.File, issue.Line, issue.Title)
			b.WriteString(indent(issue.Detail, "    "))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
		b.WriteString(counts(issues))
		b.WriteByte('\n')
	}
	writeList(&b, "Omitted, over the diff budget:", view.Omitted)
	writeList(&b, "Skipped binary files:", view.Binaries)
	writeList(&b, "Skipped submodules:", view.Submodules)
	return b.String()
}

func markdown(view View) string {
	var b strings.Builder
	b.WriteString("# Scrute review\n\n")
	fmt.Fprintf(&b, "Provider: %s\n\nModel: %s\n\n", view.Provider, view.Model)
	if strings.TrimSpace(view.Review.Summary) != "" {
		b.WriteString("## Summary\n\n")
		b.WriteString(strings.TrimSpace(view.Review.Summary))
		b.WriteString("\n\n")
	}
	issues := sorted(view.Review.Issues)
	if len(issues) == 0 {
		if view.Reviewed {
			b.WriteString("No issues.\n\n")
		} else {
			b.WriteString("Nothing to review.\n\n")
		}
	}
	var last string
	for _, issue := range issues {
		if issue.Severity != last {
			fmt.Fprintf(&b, "## %s\n\n", severityTitle(issue.Severity))
			last = issue.Severity
		}
		fmt.Fprintf(&b, "### %s:%d %s\n\n%s\n\n", issue.File, issue.Line, issue.Title, strings.TrimSpace(issue.Detail))
	}
	writeMDList(&b, "Omitted", view.Omitted)
	writeMDList(&b, "Skipped binary files", view.Binaries)
	writeMDList(&b, "Skipped submodules", view.Submodules)
	return b.String()
}

func asJSON(view View) (string, error) {
	issues := sorted(view.Review.Issues)
	if issues == nil {
		issues = []review.Issue{}
	}
	doc := struct {
		Provider   string         `json:"provider"`
		Model      string         `json:"model"`
		Summary    string         `json:"summary"`
		Issues     []review.Issue `json:"issues"`
		Omitted    []string       `json:"omitted,omitempty"`
		Binaries   []string       `json:"binaries,omitempty"`
		Submodules []string       `json:"submodules,omitempty"`
	}{
		Provider:   view.Provider,
		Model:      view.Model,
		Summary:    view.Review.Summary,
		Issues:     issues,
		Omitted:    view.Omitted,
		Binaries:   view.Binaries,
		Submodules: view.Submodules,
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(body) + "\n", nil
}

func sorted(issues []review.Issue) []review.Issue {
	out := append([]review.Issue(nil), issues...)
	// review.Merge already sorts. Keep that order if the caller did too,
	// but sort again so every format is stable.
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && lessIssue(out[j], out[j-1]) {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
	return out
}

func lessIssue(a, b review.Issue) bool {
	if review.Rank(a.Severity) != review.Rank(b.Severity) {
		return review.Rank(a.Severity) < review.Rank(b.Severity)
	}
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Title < b.Title
}

func counts(issues []review.Issue) string {
	var critical, warning, suggestion int
	for _, issue := range issues {
		switch issue.Severity {
		case review.Critical:
			critical++
		case review.Warning:
			warning++
		case review.Suggestion:
			suggestion++
		}
	}
	var parts []string
	if critical > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", critical))
	}
	if warning > 0 {
		parts = append(parts, fmt.Sprintf("%d warning", warning))
	}
	if suggestion > 0 {
		parts = append(parts, fmt.Sprintf("%d suggestion", suggestion))
	}
	return strings.Join(parts, ", ")
}

func severityTitle(severity string) string {
	switch severity {
	case review.Critical:
		return "Critical"
	case review.Warning:
		return "Warning"
	default:
		return "Suggestion"
	}
}

func severityColor(severity string) string {
	switch severity {
	case review.Critical:
		return "31"
	case review.Warning:
		return "33"
	default:
		return "36"
	}
}

func paint(enabled bool, code, text string) string {
	if !enabled {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func indent(text, pad string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, line := range lines {
		lines[i] = pad + line
	}
	return strings.Join(lines, "\n")
}

func writeList(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteByte('\n')
	b.WriteString(title)
	b.WriteByte('\n')
	for _, item := range items {
		b.WriteString("  - ")
		b.WriteString(item)
		b.WriteByte('\n')
	}
}

func writeMDList(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
}

// Write sends the rendered review to w.
func Write(w io.Writer, format string, view View) error {
	text, err := Format(format, view)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, bytes.NewReader([]byte(text)))
	return err
}
