package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rejisterjack/scrute/internal/review"
)

func TestVersion(t *testing.T) {
	var out, errBuf strings.Builder
	if code := Run([]string{"version"}, &out, &errBuf); code != 0 {
		t.Fatalf("code %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "scrute ") {
		t.Fatalf("out %q", out.String())
	}
}

func TestInitAndReviewNothingStaged(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := initRepo(t)
	t.Chdir(dir)

	var out, errBuf strings.Builder
	if code := Run([]string{"init"}, &out, &errBuf); code != 0 {
		t.Fatalf("init %d %s", code, errBuf.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".scrute.yaml")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errBuf.Reset()
	if code := Run([]string{"init"}, &out, &errBuf); code != 2 {
		t.Fatalf("second init code %d", code)
	}

	out.Reset()
	errBuf.Reset()
	if code := Run(nil, &out, &errBuf); code != 0 {
		t.Fatalf("review %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Nothing staged.") {
		t.Fatalf("out %q", out.String())
	}
}

func TestMissingKey(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "main.go")
	t.Chdir(dir)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	var out, errBuf strings.Builder
	code := Run(nil, &out, &errBuf)
	if code != 2 {
		t.Fatalf("code %d out %s err %s", code, out.String(), errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "ANTHROPIC_API_KEY") {
		t.Fatalf("err %s", errBuf.String())
	}
	if strings.Contains(errBuf.String(), "sk-") {
		t.Fatal("unexpected key material")
	}
}

func TestHookCommand(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := initRepo(t)
	t.Chdir(dir)
	var out, errBuf strings.Builder
	if code := Run([]string{"hook", "install"}, &out, &errBuf); code != 0 {
		t.Fatalf("install %d %s", code, errBuf.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "hooks", "pre-push")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := Run([]string{"hook", "uninstall"}, &out, &errBuf); code != 0 {
		t.Fatalf("uninstall %d %s", code, errBuf.String())
	}
}

func TestBadFormat(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := initRepo(t)
	t.Chdir(dir)
	var out, errBuf strings.Builder
	if code := Run([]string{"--format", "xml"}, &out, &errBuf); code != 2 {
		t.Fatalf("code %d %s", code, errBuf.String())
	}
}

func TestReviewExit(t *testing.T) {
	clean := review.Outcome{Reviewed: true, Review: review.Review{Summary: "Fine.", Issues: []review.Issue{}}}
	if reviewExit(clean, "warning") != 0 {
		t.Fatal("clean review")
	}
	warned := clean
	warned.Review.Issues = []review.Issue{{Severity: review.Warning, File: "a.go", Line: 1, Title: "t", Detail: "d"}}
	if reviewExit(warned, "warning") != 1 {
		t.Fatal("warning should exit 1")
	}
	if reviewExit(warned, "critical") != 0 {
		t.Fatal("warning is below critical")
	}
	omitted := clean
	omitted.Omitted = []string{"big.go"}
	if reviewExit(omitted, "warning") != 2 {
		t.Fatal("omitted files should exit 2 when the review is otherwise clean")
	}
	if reviewExit(warned, "none") != 0 {
		t.Fatal("fail_on none")
	}
}

func TestNotARepository(t *testing.T) {
	t.Chdir(t.TempDir())
	var out, errBuf strings.Builder
	if code := Run(nil, &out, &errBuf); code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestMarkdownOutSkipsBinaryWithoutKey(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "a.txt")
	git(t, dir, "commit", "-m", "base")
	if err := os.WriteFile(filepath.Join(dir, "pic.bin"), []byte("ok\x00no"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "pic.bin")
	t.Chdir(dir)
	t.Setenv("ANTHROPIC_API_KEY", "")

	var out, errBuf strings.Builder
	code := Run([]string{"--out", "REVIEW.md"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("code %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "pic.bin") {
		t.Fatalf("stdout %s", out.String())
	}
	body, err := os.ReadFile("REVIEW.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Skipped binary files") || !strings.Contains(string(body), "pic.bin") {
		t.Fatalf("markdown:\n%s", body)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	git(t, dir, "config", "user.email", "scrute@example.com")
	git(t, dir, "config", "user.name", "Scrute")
	return dir
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
