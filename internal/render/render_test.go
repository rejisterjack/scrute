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
			Summary: "The change drops a nil check before the map lookup.",
			Issues: []review.Issue{
				{Severity: review.Warning, File: "main.go", Line: 10, Title: "Ignored error", Detail: "WriteTo error is discarded."},
				{Severity: review.Critical, File: "internal/pay.go", Line: 42, Title: "Nil map lookup", Detail: "m[k] panics when m is nil."},
			},
		},
		Omitted:  []string{"vendor/huge.go"},
		Binaries: []string{"assets/logo.png"},
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
