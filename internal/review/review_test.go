package review

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rejisterjack/scrute/internal/gitdiff"
)

func TestParseAndFailOn(t *testing.T) {
	raw := `{
	  "summary": "SQL is built with string concatenation.",
	  "issues": [
	    {"severity": "critical", "file": "store.go", "line": 4, "title": "SQL injection", "detail": "id is concatenated into the query."}
	  ]
	}`
	rev, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !Failed(rev, "warning") || !Failed(rev, "critical") {
		t.Fatal("expected critical finding to fail warning and critical thresholds")
	}
	if Failed(rev, "none") {
		t.Fatal("none should not fail")
	}

	style := `{"summary":"Comment spelling only.","issues":[{"severity":"suggestion","file":"doc.go","line":1,"title":"Spelling","detail":"recieve is misspelled."}]}`
	rev, err = Parse(style)
	if err != nil {
		t.Fatal(err)
	}
	if Failed(rev, "warning") {
		t.Fatal("suggestion must not fail a warning threshold")
	}
	if !Failed(rev, "suggestion") {
		t.Fatal("suggestion should fail a suggestion threshold")
	}
	if hasSeverity(rev, Critical) {
		t.Fatal("comment fixture must not be critical")
	}
}

func TestCorpusPromptContainsBug(t *testing.T) {
	file := gitdiff.FileChange{
		Path:   "store.go",
		Status: gitdiff.Modified,
		Text:   "package store\n\nfunc q(id string) string {\n\treturn \"select * from users where id=\" + id\n}\n",
		Patch:  "@@ -1,4 +1,4 @@\n func q(id string) string {\n-\treturn query\n+\treturn \"select * from users where id=\" + id\n }\n",
		Hunks:  []gitdiff.Span{{Start: 4, End: 4}},
	}
	body := sectionBody(file, 2)
	if !strings.Contains(body, "select * from users where id=") {
		t.Fatalf("prompt dropped the buggy line:\n%s", body)
	}
	if !strings.Contains(body, "4 |") {
		t.Fatalf("prompt dropped the line number:\n%s", body)
	}
	raw := `{"summary":"The query concatenates input.","issues":[{"severity":"critical","file":"store.go","line":4,"title":"Injection","detail":"User input is concatenated into SQL."}]}`
	rev, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSeverity(rev, Critical) || hasSeverity(rev, Suggestion) {
		t.Fatalf("severities %#v", rev.Issues)
	}
}

func TestPackSplitsAndOmits(t *testing.T) {
	small := []gitdiff.FileChange{
		{Path: "a.go", Status: gitdiff.Modified, Text: "package a\n", Patch: "+package a\n"},
	}
	packed, _, _, _ := Pack(small, nil, 0, 1<<20)
	if len(packed) != 1 || len(packed[0]) != 1 {
		t.Fatalf("probe %#v", packed)
	}
	one := len(packed[0][0].Body)
	files := []gitdiff.FileChange{
		{Path: ".env", Status: gitdiff.Added, Text: "TOKEN=1\n", Patch: "+TOKEN=1\n"},
		small[0],
		{Path: "b.go", Status: gitdiff.Modified, Text: "package b\n", Patch: "+package b\n"},
		{Path: "huge.go", Status: gitdiff.Added, Text: strings.Repeat("x", one*4), Patch: "+" + strings.Repeat("x", one*4)},
		{Path: "pic.bin", Status: gitdiff.Added, Binary: true},
	}
	chunks, omitted, binaries, _ := Pack(files, []string{".env"}, 0, one+8)
	if len(binaries) != 1 || binaries[0] != "pic.bin" {
		t.Fatalf("binaries %#v", binaries)
	}
	if len(omitted) != 1 || omitted[0] != "huge.go" {
		t.Fatalf("omitted %#v", omitted)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks %d", len(chunks))
	}
	for _, chunk := range chunks {
		for _, section := range chunk {
			if strings.Contains(section.Body, "TOKEN") {
				t.Fatal("excluded file was packed")
			}
		}
	}
}

func TestDeletedFileContextInPrompt(t *testing.T) {
	body := sectionBody(gitdiff.FileChange{
		Path:   "note.txt",
		Status: gitdiff.Deleted,
		Text:   "alpha\nbeta\ngamma\ndelta\n",
		Patch:  "@@ -1,4 +0,0 @@\n-alpha\n-beta\n-gamma\n-delta\n",
		Hunks:  []gitdiff.Span{{Start: 1, End: 4}},
	}, 1)
	if !strings.Contains(body, "3 | gamma") {
		t.Fatalf("prompt dropped deleted context:\n%s", body)
	}
}

func TestMergeDedupes(t *testing.T) {
	merged := Merge([]Review{
		{Summary: "First.", Issues: []Issue{{Severity: Warning, File: "a.go", Line: 2, Title: "Err", Detail: "ignored"}}},
		{Summary: "Second.", Issues: []Issue{
			{Severity: Warning, File: "a.go", Line: 2, Title: "Err", Detail: "ignored"},
			{Severity: Critical, File: "a.go", Line: 9, Title: "Panic", Detail: "nil"},
		}},
	})
	if merged.Summary != "First.\n\nSecond." {
		t.Fatalf("summary %q", merged.Summary)
	}
	if len(merged.Issues) != 2 || merged.Issues[0].Severity != Critical {
		t.Fatalf("%#v", merged.Issues)
	}
}

type scripted struct {
	calls []string
	raw   []string
}

func (s *scripted) Complete(_ context.Context, _, _, user string) (string, int, int, error) {
	s.calls = append(s.calls, user)
	raw := s.raw[0]
	s.raw = s.raw[1:]
	return raw, 3, 4, nil
}

func TestRunnerRetriesInvalidJSON(t *testing.T) {
	model := &scripted{raw: []string{
		"not json",
		`{"summary":"Clean enough.","issues":[]}`,
	}}
	out, err := (Runner{
		Model:        model,
		ProviderName: "openai",
		ModelName:    "gpt-5.4",
		Strictness:   "balanced",
		MaxDiffBytes: 1000,
	}).Run(context.Background(), []gitdiff.FileChange{{
		Path: "a.go", Status: gitdiff.Modified, Text: "package a\n", Patch: "+package a\n",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Reviewed || out.Review.Summary != "Clean enough." || len(model.calls) != 2 {
		t.Fatalf("out %#v calls %d", out.Review, len(model.calls))
	}
	if out.Input != 6 || out.Output != 8 {
		t.Fatalf("tokens %d %d", out.Input, out.Output)
	}
}

func TestRunnerSecondParseIsUnparsed(t *testing.T) {
	model := &scripted{raw: []string{"not json", "still not json"}}
	_, err := (Runner{
		Model:        model,
		ProviderName: "openai",
		ModelName:    "gpt-5.4",
		Strictness:   "balanced",
		MaxDiffBytes: 1000,
	}).Run(context.Background(), []gitdiff.FileChange{{
		Path: "a.go", Status: gitdiff.Modified, Text: "package a\n", Patch: "+package a\n",
	}})
	var unparsed *UnparsedError
	if err == nil || !errors.As(err, &unparsed) || len(model.calls) != 2 {
		t.Fatalf("err %v calls %d", err, len(model.calls))
	}
	if strings.Contains(err.Error(), "sk-") {
		t.Fatal("unexpected secret")
	}
}

func hasSeverity(rev Review, severity string) bool {
	for _, issue := range rev.Issues {
		if issue.Severity == severity {
			return true
		}
	}
	return false
}
