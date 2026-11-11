package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rejisterjack/scrute/internal/hook"
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

func TestRangeReviewsCommitsNotTheIndex(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "main.go")
	git(t, dir, "commit", "-m", "base")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "main.go")
	git(t, dir, "commit", "-m", "change")
	t.Chdir(dir)
	t.Setenv("ANTHROPIC_API_KEY", "")

	var out, errBuf strings.Builder
	code := Run([]string{"--range", "HEAD~1..HEAD"}, &out, &errBuf)
	if code != 2 || !strings.Contains(errBuf.String(), "ANTHROPIC_API_KEY") {
		t.Fatalf("code %d out %q err %q", code, out.String(), errBuf.String())
	}
	if strings.Contains(out.String(), "Nothing staged") {
		t.Fatal("a committed range was treated as an empty index")
	}
	out.Reset()
	errBuf.Reset()
	if code := Run([]string{"--range", "HEAD...HEAD~1"}, &out, &errBuf); code != 2 {
		t.Fatalf("three-dot code %d %s", code, errBuf.String())
	}
}

func TestReviewAgainstStub(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("TOKEN=secret-value-should-not-leak\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { println(\"STAGED\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", ".env", "main.go")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { println(\"STAGED\"); println(\"WORKTREE ONLY\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	const key = "sk-ant-test-secret-value"
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		if r.Header.Get("x-api-key") != key {
			t.Errorf("missing api key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
		  "id": "msg_test",
		  "type": "message",
		  "role": "assistant",
		  "model": "claude-sonnet-5-5",
		  "stop_reason": "end_turn",
		  "content": [{"type":"text","text":"{\"summary\":\"Nil risk.\",\"issues\":[{\"severity\":\"warning\",\"file\":\"main.go\",\"line\":3,\"title\":\"Unchecked call\",\"detail\":\"println ignores nothing.\"}]}"}],
		  "usage": {"input_tokens": 9, "output_tokens": 8, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`)
	}))
	defer server.Close()

	t.Setenv("ANTHROPIC_API_KEY", key)
	t.Setenv("SCRUTE_ANTHROPIC_BASE_URL", server.URL)
	t.Setenv("OPENAI_API_KEY", "")

	var out, errBuf strings.Builder
	code := Run([]string{"--verbose", "--format", "json", "--out", "REVIEW.md"}, &out, &errBuf)
	if code != 1 {
		t.Fatalf("code %d\nstdout %s\nstderr %s", code, out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), "Unchecked call") || !strings.Contains(out.String(), `"summary"`) {
		t.Fatalf("stdout %s", out.String())
	}
	if !strings.Contains(errBuf.String(), "input_tokens=9") || !strings.Contains(errBuf.String(), "sent main.go") {
		t.Fatalf("stderr %s", errBuf.String())
	}
	if strings.Contains(errBuf.String(), ".env") || strings.Contains(body, "secret-value-should-not-leak") || strings.Contains(body, "WORKTREE ONLY") {
		t.Fatal("request included an excluded file or an unstaged line")
	}
	if !strings.Contains(body, "STAGED") || !strings.Contains(body, "json_schema") {
		t.Fatalf("request did not carry the staged review:\n%s", body)
	}
	if strings.Contains(body, key) {
		t.Fatal("API key was copied into the request body")
	}
	markdown, err := os.ReadFile(filepath.Join(dir, "REVIEW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "Unchecked call") {
		t.Fatalf("review file %s", markdown)
	}
}

func TestOpenAIStubExitZero(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "main.go")
	t.Chdir(dir)

	const key = "sk-openai-test-secret-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+key {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
		  "id": "chatcmpl-test",
		  "object": "chat.completion",
		  "created": 1,
		  "model": "gpt-5.4",
		  "choices": [{"index": 0, "finish_reason": "stop", "message": {"role": "assistant", "content": "{\"summary\":\"Clean.\",\"issues\":[]}"}}],
		  "usage": {"prompt_tokens": 4, "completion_tokens": 2, "total_tokens": 6}
		}`)
	}))
	defer server.Close()
	t.Setenv("OPENAI_API_KEY", key)
	t.Setenv("SCRUTE_OPENAI_BASE_URL", server.URL)

	var out, errBuf strings.Builder
	code := Run([]string{"--provider", "openai"}, &out, &errBuf)
	if code != 0 || !strings.Contains(out.String(), "No issues.") {
		t.Fatalf("code %d out %s err %s", code, out.String(), errBuf.String())
	}
}

func TestModelAndStrictnessAreSent(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := stageGoFile(t, "package main\n")
	const key = "sk-ant-test-secret-value"
	var body string
	server := anthropicServer(t, key, &body, 200, `{"summary":"Clean.","issues":[]}`)
	defer server.Close()
	t.Setenv("ANTHROPIC_API_KEY", key)
	t.Setenv("SCRUTE_ANTHROPIC_BASE_URL", server.URL)

	var out, errBuf strings.Builder
	code := Run([]string{"--model", "claude-test-model", "--strictness", "strict", "--format", "md", "--out", "REVIEW.md"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("code %d %s", code, errBuf.String())
	}
	if !strings.Contains(body, "claude-test-model") || !strings.Contains(body, "Strictness is strict") {
		t.Fatalf("request:\n%s", body)
	}
	file, err := os.ReadFile(filepath.Join(dir, "REVIEW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(file), "# Scrute review") {
		t.Fatalf("markdown alias wrote:\n%s", file)
	}
}

func TestFailOnThreshold(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := stageGoFile(t, "package main\n")
	if err := os.WriteFile(filepath.Join(dir, ".scrute.yaml"), []byte("fail_on: critical\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const key = "sk-ant-test-secret-value"
	server := anthropicServer(t, key, nil, 200, `{"summary":"A warning.","issues":[{"severity":"warning","file":"main.go","line":1,"title":"Note","detail":"Worth a look."}]}`)
	defer server.Close()
	t.Setenv("ANTHROPIC_API_KEY", key)
	t.Setenv("SCRUTE_ANTHROPIC_BASE_URL", server.URL)

	var out, errBuf strings.Builder
	if code := Run(nil, &out, &errBuf); code != 0 {
		t.Fatalf("critical threshold code %d %s", code, errBuf.String())
	}
	if err := os.WriteFile(filepath.Join(dir, ".scrute.yaml"), []byte("fail_on: warning\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errBuf.Reset()
	if code := Run(nil, &out, &errBuf); code != 1 {
		t.Fatalf("warning threshold code %d out %s err %s", code, out.String(), errBuf.String())
	}
}

func TestProviderFailureExits3(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	stageGoFile(t, "package main\n")
	const key = "sk-ant-test-secret-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"rejected `+key+`"}}`)
	}))
	defer server.Close()
	t.Setenv("ANTHROPIC_API_KEY", key)
	t.Setenv("SCRUTE_ANTHROPIC_BASE_URL", server.URL)

	var out, errBuf strings.Builder
	code := Run(nil, &out, &errBuf)
	if code != 3 {
		t.Fatalf("code %d out %s err %s", code, out.String(), errBuf.String())
	}
	if strings.Contains(errBuf.String(), key) {
		t.Fatalf("key leaked: %s", errBuf.String())
	}
}

func TestOmittedFileExits2(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := stageGoFile(t, strings.Repeat("package main\n", 20))
	if err := os.WriteFile(filepath.Join(dir, ".scrute.yaml"), []byte("max_diff_bytes: 30\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errBuf strings.Builder
	code := Run(nil, &out, &errBuf)
	if code != 2 || !strings.Contains(out.String(), "Omitted") || !strings.Contains(out.String(), "main.go") {
		t.Fatalf("code %d out %s err %s", code, out.String(), errBuf.String())
	}
}

func TestCancelledContext(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	stageGoFile(t, "package main\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, errBuf strings.Builder
	err := runReview(ctx, &out, &errBuf, reviewFlags{})
	var failed *failure
	if !errors.As(err, &failed) || failed.code != 2 {
		t.Fatalf("err %v out %s errbuf %s", err, out.String(), errBuf.String())
	}
}

func TestHookReviewsRealRange(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	bin := filepath.Join(t.TempDir(), "scrute")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = moduleRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("ROOT_SENTINEL\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "README")
	git(t, dir, "commit", "-m", "base")
	git(t, dir, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(dir, "feature.go"), []byte("package feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "feature.go")
	git(t, dir, "commit", "-m", "feature")
	head := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD"))
	parent := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD~1"))

	const key = "sk-ant-test-secret-value"
	var body string
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		hits++
		writeAnthropicReview(w, `{"summary":"Range reviewed.","issues":[]}`)
	}))
	defer server.Close()
	t.Setenv("ANTHROPIC_API_KEY", key)
	t.Setenv("SCRUTE_ANTHROPIC_BASE_URL", server.URL)
	if err := hook.Install(dir, bin); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-push")

	runHook := func(stdin string) (string, error) {
		t.Helper()
		cmd := exec.Command(hookPath, "origin", "file://example")
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader(stdin)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	out, err := runHook(fmt.Sprintf("refs/heads/feature %s refs/heads/feature %s\n", head, parent))
	if err != nil {
		t.Fatalf("range hook: %v\n%s", err, out)
	}
	if hits != 1 || !strings.Contains(body, "package feature") || strings.Contains(body, "ROOT_SENTINEL") {
		t.Fatalf("hits %d body:\n%s\nhook:\n%s", hits, body, out)
	}
	if !strings.Contains(out, "Range reviewed.") {
		t.Fatalf("hook output %s", out)
	}

	zeros := strings.Repeat("0", 40)
	out, err = runHook(fmt.Sprintf("refs/heads/feature %s refs/heads/feature %s\n", head, zeros))
	if err != nil {
		t.Fatalf("new branch hook: %v\n%s", err, out)
	}
	if hits != 2 || !strings.Contains(body, "package feature") || strings.Contains(body, "ROOT_SENTINEL") {
		t.Fatalf("hits %d body:\n%s", hits, body)
	}

	before := hits
	out, err = runHook("")
	if err != nil {
		t.Fatalf("empty push: %v\n%s", err, out)
	}
	if hits != before {
		t.Fatalf("empty push called the model %d times", hits-before)
	}
}

func stageGoFile(t *testing.T, body string) string {
	t.Helper()
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "main.go")
	t.Chdir(dir)
	return dir
}

func anthropicServer(t *testing.T, key string, body *string, status int, reviewJSON string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if body != nil {
			*body = string(raw)
		}
		if r.Header.Get("x-api-key") != key {
			t.Errorf("missing api key")
		}
		if status != 200 {
			w.WriteHeader(status)
		}
		writeAnthropicReview(w, reviewJSON)
	}))
}

func writeAnthropicReview(w http.ResponseWriter, reviewJSON string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{
	  "id": "msg_test",
	  "type": "message",
	  "role": "assistant",
	  "model": "claude-sonnet-5-5",
	  "stop_reason": "end_turn",
	  "content": [{"type":"text","text":`+fmt.Sprintf("%q", reviewJSON)+`}],
	  "usage": {"input_tokens": 3, "output_tokens": 2, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
	}`)
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
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
	_ = gitOutput(t, dir, args...)
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}
