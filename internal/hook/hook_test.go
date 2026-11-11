package hook

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallAndUninstall(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	if err := Install(dir, "/usr/local/bin/scrute"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".git", "hooks", "pre-push")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, marker) || !strings.Contains(text, "'/usr/local/bin/scrute'") || !strings.Contains(text, "--range") {
		t.Fatalf("hook:\n%s", text)
	}
	if !strings.Contains(text, "refs/heads/*") {
		t.Fatalf("hook does not limit itself to branches:\n%s", text)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("mode %v", info.Mode())
	}

	if err := Install(dir, "/opt/scrute"); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "'/opt/scrute'") {
		t.Fatalf("hook was not updated:\n%s", body)
	}

	if err := Uninstall(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("hook still present: %v", err)
	}
	if err := Uninstall(dir); err != ErrNotInstalled {
		t.Fatalf("uninstall twice: %v", err)
	}
}

func TestInstallRefusesForeignHook(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	path := filepath.Join(dir, ".git", "hooks", "pre-push")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho other\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Install(dir, "/usr/local/bin/scrute"); err != ErrExists {
		t.Fatalf("err %v", err)
	}
}

func TestHookRunsRangeForBranchUpdates(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	logPath := filepath.Join(dir, "calls")
	fake := filepath.Join(dir, "fake-scrute")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + logPath + "'\ncase \"$2\" in\nbad*) exit 4 ;;\nesac\nexit 0\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Install(dir, fake); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-push")
	zeros := strings.Repeat("0", 64)
	stdin := strings.Join([]string{
		"refs/heads/main abcdef refs/heads/main 111111",
		"refs/tags/v1 tagsha refs/tags/v1 " + zeros,
		"refs/heads/gone " + zeros + " refs/heads/gone 222222",
		"",
	}, "\n")
	run := exec.Command(hookPath, "origin", "file://example")
	run.Dir = dir
	run.Stdin = strings.NewReader(stdin)
	if out, err := run.CombinedOutput(); err != nil {
		t.Fatalf("hook: %v\n%s", err, out)
	}
	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if text != "--range 111111..abcdef\n" {
		t.Fatalf("calls:\n%s", text)
	}

	bad := exec.Command(hookPath, "origin", "file://example")
	bad.Dir = dir
	bad.Stdin = strings.NewReader("refs/heads/main abcdef refs/heads/main badbase\n")
	out, err := bad.CombinedOutput()
	if err == nil {
		t.Fatal("expected the hook to abort")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 4 {
		t.Fatalf("exit %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "push aborted (exit 4)") {
		t.Fatalf("stderr %s", out)
	}
}
