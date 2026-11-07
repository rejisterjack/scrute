package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rejisterjack/scrute/internal/review"
)

func TestGolden(t *testing.T) {
	view := View{
		Provider: "anthropic",
		Model:    "claude-sonnet-5",
		Reviewed: true,
		Review: review.Review{
			Summary: "The change drops a nil check before the map lookup and ignores a write error on the happy path.",
			Issues: []review.Issue{
				{Severity: review.Critical, File: "internal/pay.go", Line: 42, Title: "Nil map lookup", Detail: "m[k] panics when m is nil."},
				{Severity: review.Critical, File: "internal/pay.go", Line: 58, Title: "Unchecked type assert", Detail: "user.(map[string]any) panics when the session value is not a map."},
				{Severity: review.Warning, File: "main.go", Line: 10, Title: "Ignored error", Detail: "WriteTo error is discarded."},
				{Severity: review.Warning, File: "internal/pay.go", Line: 71, Title: "Missing context cancel", Detail: "The HTTP client call does not honor ctx; shutdown can hang."},
				{Severity: review.Suggestion, File: "main.go", Line: 18, Title: "Prefer fmt.Errorf wrapping", Detail: "return err hides the call site; wrap with %w for diagnosis."},
			},
		},
		Omitted:  []string{"vendor/huge.go", "generated/zz_api.go"},
		Binaries: []string{"assets/logo.png", "assets/icon.ico"},
	}
	for _, tc := range []struct {
		format string
		file   string
	}{
		{"term", "review.txt"},
		{"markdown", "review.md"},
		{"json", "review.json"},
	} {
		got, err := Format(tc.format, view)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("testdata", tc.file)
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Fatalf("%s\n got:\n%s\nwant:\n%s", tc.format, got, want)
		}
	}
}
