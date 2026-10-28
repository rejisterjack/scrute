package hook

import (
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
	if !strings.Contains(text, marker) || !strings.Contains(text, "'/usr/local/bin/scrute'") {
		t.Fatalf("hook:\n%s", text)
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
