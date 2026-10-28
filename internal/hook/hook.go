// Package hook installs an opt-in pre-push hook that runs scrute.
package hook

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejisterjack/scrute/internal/gitdiff"
)

const marker = "# scrute-managed-pre-push"

var (
	// ErrExists means a pre-push hook is present and was not installed by scrute.
	ErrExists = errors.New("a pre-push hook already exists")
	// ErrNotInstalled means there is no scrute-managed hook to remove.
	ErrNotInstalled = errors.New("no scrute pre-push hook is installed")
)

// Install writes a pre-push hook that aborts the push when scrute exits non-zero.
func Install(root, executable string) error {
	if executable == "" {
		path, err := os.Executable()
		if err != nil {
			return err
		}
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		executable = path
	}
	path, err := hookPath(root)
	if err != nil {
		return err
	}
	if body, err := os.ReadFile(path); err == nil {
		if !strings.Contains(string(body), marker) {
			return ErrExists
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	script := fmt.Sprintf(`#!/bin/sh
%s
%s
status=$?
if [ "$status" -ne 0 ]; then
  echo "scrute: push aborted (exit $status)" >&2
  exit "$status"
fi
exit 0
`, marker, shellQuote(executable))
	return os.WriteFile(path, []byte(script), 0o755)
}

// Uninstall removes a hook previously written by Install.
func Uninstall(root string) error {
	path, err := hookPath(root)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotInstalled
	}
	if err != nil {
		return err
	}
	if !strings.Contains(string(body), marker) {
		return ErrNotInstalled
	}
	return os.Remove(path)
}

func hookPath(root string) (string, error) {
	dir, err := gitdiff.GitDir(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hooks", "pre-push"), nil
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
