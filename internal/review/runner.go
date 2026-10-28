package review

import (
	"context"
	"strings"

	"github.com/rejisterjack/scrute/internal/gitdiff"
)

// Completer sends one prompt to a model and returns the review JSON text.
type Completer interface {
	Complete(ctx context.Context, model, system, user string) (raw string, inputTokens, outputTokens int, err error)
}

// Runner reviews a staged diff, retrying once when the JSON is invalid.
type Runner struct {
	Model        Completer
	ProviderName string
	ModelName    string
	Strictness   string
	Patterns     []string
	ContextLines int
	MaxDiffBytes int
}

// Outcome is a finished review plus what was not sent.
type Outcome struct {
	Review     Review
	Omitted    []string
	Binaries   []string
	Submodules []string
	Sent       []string
	Input      int
	Output     int
	Reviewed   bool
}

// Run packs the diff and reviews each chunk.
func (r Runner) Run(ctx context.Context, files []gitdiff.FileChange) (Outcome, error) {
	chunks, omitted, binaries, submodules := Pack(files, r.Patterns, r.ContextLines, r.MaxDiffBytes)
	out := Outcome{
		Review:     Review{Issues: []Issue{}},
		Omitted:    omitted,
		Binaries:   binaries,
		Submodules: submodules,
	}
	if len(chunks) == 0 {
		return out, nil
	}
	system := SystemPrompt(r.Strictness, r.ProviderName)
	var parts []Review
	for _, chunk := range chunks {
		user := UserPrompt(chunk)
		for _, section := range chunk {
			out.Sent = append(out.Sent, section.Path)
		}
		rev, inTok, outTok, err := r.complete(ctx, system, user)
		out.Input += inTok
		out.Output += outTok
		if err != nil {
			return Outcome{}, err
		}
		parts = append(parts, rev)
	}
	out.Review = Merge(parts)
	out.Reviewed = true
	return out, nil
}

func (r Runner) complete(ctx context.Context, system, user string) (Review, int, int, error) {
	raw, inTok, outTok, err := r.Model.Complete(ctx, r.ModelName, system, user)
	if err != nil {
		return Review{}, inTok, outTok, err
	}
	rev, err := Parse(raw)
	if err == nil {
		return rev, inTok, outTok, nil
	}
	retryUser := user + "\n\nThe previous response was invalid: " + err.Error() + "\nReturn one review JSON object with a summary and an issues array."
	raw, in2, out2, err2 := r.Model.Complete(ctx, r.ModelName, system, retryUser)
	inTok += in2
	outTok += out2
	if err2 != nil {
		return Review{}, inTok, outTok, err2
	}
	rev, err = Parse(raw)
	if err != nil {
		return Review{}, inTok, outTok, &UnparsedError{Raw: strings.TrimSpace(raw)}
	}
	return rev, inTok, outTok, nil
}
