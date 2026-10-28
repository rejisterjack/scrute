package review

import (
	"fmt"
	"strings"

	"github.com/rejisterjack/scrute/internal/gitdiff"
)

// SystemPrompt is the instruction block for one review request.
func SystemPrompt(strictness, providerName string) string {
	var focus string
	switch strictness {
	case "lenient":
		focus = "Report only critical issues: likely bugs and security mistakes."
	case "strict":
		focus = "Report critical issues, warnings about correctness and API misuse, and suggestions about maintainability. Skip formatting nits that a formatter would fix."
	default:
		focus = "Report critical issues and warnings about correctness and API misuse. Skip style."
	}
	closing := "Respond with the review JSON object and nothing else."
	if providerName == "anthropic" {
		closing = "Submit the review by calling the submit_review tool exactly once."
	}
	return strings.TrimSpace(fmt.Sprintf(`You are Scrute, a pre-push code reviewer. Review only the staged diff you are given. Report issues that are justified by that diff. Do not invent files or line numbers. Do not restate the diff. Line numbers refer to the new file. Use line 0 only when the issue applies to the whole file.

Each issue has severity, file, line, title, and detail.
Severity:
- critical: a likely bug, security mistake, or data loss
- warning: a correctness or API misuse problem that may happen
- suggestion: a smaller maintainability improvement

Strictness is %s.
%s

If nothing is worth reporting, return an empty issues list and a one-sentence summary. Prefer fewer, sharper issues over a long list.

%s`, strictness, focus, closing))
}

// UserPrompt concatenates file sections for one request.
func UserPrompt(sections []Section) string {
	var b strings.Builder
	b.WriteString("Review these staged changes.\n")
	for _, section := range sections {
		b.WriteByte('\n')
		b.WriteString(section.Body)
		if !strings.HasSuffix(section.Body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// Section is the prompt text for one file.
type Section struct {
	Path string
	Body string
}

func sectionBody(file gitdiff.FileChange, contextLines int) string {
	var b strings.Builder
	if file.Status == gitdiff.Renamed {
		fmt.Fprintf(&b, "## %s (renamed from %s)\n", file.Path, file.OldPath)
	} else {
		fmt.Fprintf(&b, "## %s (%s)\n", file.Path, file.Status)
	}
	if strings.TrimSpace(file.Patch) != "" {
		b.WriteString("### diff\n")
		b.WriteString(file.Patch)
		if !strings.HasSuffix(file.Patch, "\n") {
			b.WriteByte('\n')
		}
	}
	if context := numberedContext(file.Text, file.Hunks, contextLines); context != "" {
		b.WriteString("### context\n")
		b.WriteString(context)
		if !strings.HasSuffix(context, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func numberedContext(text string, hunks []gitdiff.Span, radius int) string {
	if text == "" || radius < 0 {
		return ""
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(hunks) == 0 {
		hunks = []gitdiff.Span{{Start: 1, End: min(len(lines), 1)}}
	}
	type window struct{ start, end int }
	var windows []window
	for _, hunk := range hunks {
		start := hunk.Start - radius
		end := hunk.End + radius
		if start < 1 {
			start = 1
		}
		if end > len(lines) {
			end = len(lines)
		}
		if start > len(lines) || end < 1 {
			continue
		}
		if len(windows) > 0 && start <= windows[len(windows)-1].end+1 {
			windows[len(windows)-1].end = end
			continue
		}
		windows = append(windows, window{start, end})
	}
	if len(windows) == 0 {
		return ""
	}
	width := len(fmt.Sprintf("%d", len(lines)))
	var b strings.Builder
	for i, w := range windows {
		if i > 0 {
			b.WriteString("...\n")
		}
		for n := w.start; n <= w.end; n++ {
			fmt.Fprintf(&b, "%*d | %s\n", width, n, lines[n-1])
		}
	}
	return b.String()
}
