// Package cli is the scrute command line.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rejisterjack/scrute/internal/buildinfo"
	"github.com/rejisterjack/scrute/internal/config"
	"github.com/rejisterjack/scrute/internal/gitdiff"
	"github.com/rejisterjack/scrute/internal/hook"
	"github.com/rejisterjack/scrute/internal/provider"
	"github.com/rejisterjack/scrute/internal/provider/anthropic"
	"github.com/rejisterjack/scrute/internal/provider/openai"
	"github.com/rejisterjack/scrute/internal/render"
	"github.com/rejisterjack/scrute/internal/review"
	"github.com/spf13/cobra"
)

const (
	exitOK       = 0
	exitIssues   = 1
	exitUsage    = 2
	exitProvider = 3
)

type failure struct {
	code int
	err  error
}

func (f *failure) Error() string {
	if f == nil || f.err == nil {
		return ""
	}
	return f.err.Error()
}

func (f *failure) Unwrap() error { return f.err }

func say(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func fail(code int, err error) error {
	if err == nil {
		return &failure{code: code}
	}
	return &failure{code: code, err: err}
}

// Main runs scrute with the process standard streams.
func Main(args []string) int {
	return Run(args, os.Stdout, os.Stderr)
}

// Run executes scrute and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	cmd := newRoot(stdout, stderr)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	err := cmd.Execute()
	var failed *failure
	if errors.As(err, &failed) {
		if failed.err != nil {
			say(stderr, "scrute: %s\n", failed.err)
		}
		return failed.code
	}
	if err != nil {
		say(stderr, "scrute: %s\n", err)
		return exitUsage
	}
	return exitOK
}

func newRoot(stdout, stderr io.Writer) *cobra.Command {
	var (
		providerName string
		model        string
		strictness   string
		format       string
		outPath      string
		verbose      bool
	)
	cmd := &cobra.Command{
		Use:           "scrute",
		Short:         "Review your staged diff before you push",
		Long:          "Scrute reads the staged git diff, sends it to Claude or GPT with your own API key, and prints a structured review. Source stays on your machine.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buildinfo.Version,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReview(cmd.Context(), stdout, stderr, reviewFlags{
				provider: providerName, model: model, strictness: strictness,
				format: format, out: outPath, verbose: verbose,
			})
		},
	}
	cmd.SetVersionTemplate("scrute {{.Version}}\n")
	cmd.CompletionOptions.DisableDefaultCmd = true
	flags := cmd.Flags()
	flags.StringVar(&providerName, "provider", "", "model provider: anthropic or openai")
	flags.StringVar(&model, "model", "", "model id (default depends on the provider)")
	flags.StringVar(&strictness, "strictness", "", "lenient, balanced, or strict")
	flags.StringVar(&format, "format", "", "output format: term, markdown, or json")
	flags.StringVar(&outPath, "out", "", "write the review to this file")
	flags.BoolVar(&verbose, "verbose", false, "print files sent and token counts on stderr")

	cmd.AddCommand(newInit(stdout), newVersion(stdout), newHook(stdout))
	return cmd
}

type reviewFlags struct {
	provider   string
	model      string
	strictness string
	format     string
	out        string
	verbose    bool
}

func runReview(ctx context.Context, stdout, stderr io.Writer, flags reviewFlags) error {
	if !validFormat(flags.format) {
		return fail(exitUsage, fmt.Errorf("format must be term, markdown, or json, got %q", flags.format))
	}
	wd, err := os.Getwd()
	if err != nil {
		return fail(exitUsage, err)
	}
	root, err := gitdiff.FindRoot(wd)
	if err != nil {
		return fail(exitUsage, err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fail(exitUsage, err)
	}
	if err := cfg.ApplyFlags(flags.provider, flags.model, flags.strictness); err != nil {
		return fail(exitUsage, err)
	}
	if cfg.Model == "" {
		cfg.Model = provider.DefaultModel(cfg.Provider)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	_, files, err := gitdiff.ReadStaged(readCtx, root)
	cancel()
	if err != nil {
		return fail(exitUsage, err)
	}
	if len(files) == 0 {
		say(stdout, "Nothing staged.\n")
		return nil
	}

	chunks, _, _, _ := review.Pack(files, cfg.Exclude, cfg.ContextLines, cfg.MaxDiffBytes)
	var model review.Completer
	if len(chunks) > 0 {
		key := os.Getenv(provider.EnvVar(cfg.Provider))
		if strings.TrimSpace(key) == "" {
			return fail(exitUsage, fmt.Errorf("set %s", provider.EnvVar(cfg.Provider)))
		}
		switch cfg.Provider {
		case provider.OpenAI:
			model = openai.New(key)
		default:
			model = anthropic.New(key)
		}
	}

	outcome, err := (review.Runner{
		Model:        model,
		ProviderName: cfg.Provider,
		ModelName:    cfg.Model,
		Strictness:   cfg.Strictness,
		Patterns:     cfg.Exclude,
		ContextLines: cfg.ContextLines,
		MaxDiffBytes: cfg.MaxDiffBytes,
	}).Run(ctx, files)
	if err != nil {
		return fail(exitProvider, err)
	}

	if flags.verbose {
		say(stderr, "scrute: %s/%s files=%d input_tokens=%d output_tokens=%d\n",
			cfg.Provider, cfg.Model, len(outcome.Sent), outcome.Input, outcome.Output)
		for _, path := range outcome.Sent {
			say(stderr, "scrute: sent %s\n", path)
		}
	}

	view := render.View{
		Provider:   cfg.Provider,
		Model:      cfg.Model,
		Review:     outcome.Review,
		Omitted:    outcome.Omitted,
		Binaries:   outcome.Binaries,
		Submodules: outcome.Submodules,
		Reviewed:   outcome.Reviewed,
		Color:      flags.format == "" && useColor(stdout),
	}
	stdoutFormat := flags.format
	if stdoutFormat == "" {
		stdoutFormat = "term"
	}
	if err := render.Write(stdout, stdoutFormat, view); err != nil {
		return fail(exitUsage, err)
	}
	if flags.out != "" {
		fileFormat := stdoutFormat
		if flags.format == "" {
			fileFormat = "markdown"
			view.Color = false
		}
		text, err := render.Format(fileFormat, view)
		if err != nil {
			return fail(exitUsage, err)
		}
		if err := os.WriteFile(flags.out, []byte(text), 0o644); err != nil {
			return fail(exitUsage, err)
		}
	}

	code := reviewExit(outcome, cfg.FailOn)
	if code == exitOK {
		return nil
	}
	return fail(code, nil)
}

func reviewExit(out review.Outcome, failOn string) int {
	switch {
	case review.Failed(out.Review, failOn):
		return exitIssues
	case len(out.Omitted) > 0:
		return exitUsage
	default:
		return exitOK
	}
}

func validFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "term", "terminal", "text", "markdown", "md", "json":
		return true
	default:
		return false
	}
}

func useColor(w io.Writer) bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func newInit(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:           "init",
		Short:         "Write a .scrute.yaml in the repository root",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return fail(exitUsage, err)
			}
			root, err := gitdiff.FindRoot(wd)
			if err != nil {
				return fail(exitUsage, err)
			}
			path := filepath.Join(root, config.FileName)
			if _, err := os.Stat(path); err == nil {
				return fail(exitUsage, fmt.Errorf("%s already exists", path))
			} else if !errors.Is(err, os.ErrNotExist) {
				return fail(exitUsage, err)
			}
			if err := os.WriteFile(path, []byte(config.Template()), 0o644); err != nil {
				return fail(exitUsage, err)
			}
			say(stdout, "Wrote %s\n", path)
			return nil
		},
	}
}

func newVersion(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print the scrute version",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			say(stdout, "scrute %s\n", buildinfo.Version)
			return nil
		},
	}
}

func newHook(stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "hook",
		Short:         "Install or remove a pre-push hook",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:           "install",
			Short:         "Abort git push when scrute finds issues",
			SilenceUsage:  true,
			SilenceErrors: true,
			Args:          cobra.NoArgs,
			RunE: func(*cobra.Command, []string) error {
				root, err := repoRoot()
				if err != nil {
					return fail(exitUsage, err)
				}
				if err := hook.Install(root, ""); err != nil {
					return fail(exitUsage, err)
				}
				say(stdout, "Installed pre-push hook. A non-zero scrute exit aborts the push.\n")
				return nil
			},
		},
		&cobra.Command{
			Use:           "uninstall",
			Short:         "Remove the scrute pre-push hook",
			SilenceUsage:  true,
			SilenceErrors: true,
			Args:          cobra.NoArgs,
			RunE: func(*cobra.Command, []string) error {
				root, err := repoRoot()
				if err != nil {
					return fail(exitUsage, err)
				}
				if err := hook.Uninstall(root); err != nil {
					return fail(exitUsage, err)
				}
				say(stdout, "Removed pre-push hook.\n")
				return nil
			},
		},
	)
	return cmd
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return gitdiff.FindRoot(wd)
}
