package gitdiff

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadStagedMatchesGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "scrute@example.com")
	runGit(t, dir, "config", "user.name", "Scrute")
	runGit(t, dir, "config", "core.autocrlf", "false")

	mustWrite(t, filepath.Join(dir, "lib", "note.txt"), "alpha\nbeta\ngamma\n")
	mustWrite(t, filepath.Join(dir, "keep.go"), "package keep\n\nfunc Keep() int { return 1 }\n")
	mustWrite(t, filepath.Join(dir, "oldname.txt"), "same bytes\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "base")

	// Staged edit plus a later unstaged edit. Only the staged line belongs in the review.
	mustWrite(t, filepath.Join(dir, "lib", "note.txt"), "alpha\nbeta staged\ngamma\n")
	runGit(t, dir, "add", "lib/note.txt")
	mustWrite(t, filepath.Join(dir, "lib", "note.txt"), "alpha\nbeta staged\ngamma\nuntracked edit\n")

	mustWrite(t, filepath.Join(dir, "new.go"), "package new\n")
	runGit(t, dir, "add", "new.go")

	runGit(t, dir, "rm", "keep.go")
	runGit(t, dir, "mv", "oldname.txt", "newname.txt")

	mustWrite(t, filepath.Join(dir, "blob.bin"), "ok\x00no\n")
	runGit(t, dir, "add", "blob.bin")

	_, got, err := ReadStaged(context.Background(), filepath.Join(dir, "lib"))
	if err != nil {
		t.Fatalf("ReadStaged: %v", err)
	}

	wantStatus := gitNameStatus(t, dir)
	gotStatus := map[string]string{}
	for _, file := range got {
		key := file.Path
		switch file.Status {
		case Added:
			gotStatus[key] = "A"
		case Modified:
			gotStatus[key] = "M"
		case Deleted:
			gotStatus[key] = "D"
		case Renamed:
			gotStatus[file.OldPath+" -> "+file.Path] = "R"
		default:
			t.Fatalf("unknown status %q for %s", file.Status, file.Path)
		}
		if file.Status != Renamed && strings.Contains(file.Patch, "untracked edit") {
			t.Fatalf("unstaged line leaked into %s", file.Path)
		}
	}
	if len(gotStatus) != len(wantStatus) {
		t.Fatalf("status set\n got %#v\nwant %#v", gotStatus, wantStatus)
	}
	for key, status := range wantStatus {
		if gotStatus[key] != status {
			t.Fatalf("status %s: got %q want %q\nfull got %#v\nfull want %#v", key, gotStatus[key], status, gotStatus, wantStatus)
		}
	}

	gotAdd, gotDel := changedLines(joinPatches(got))
	wantAdd, wantDel := changedLines(gitDiff(t, dir))
	if strings.Join(gotAdd, "\n") != strings.Join(wantAdd, "\n") || strings.Join(gotDel, "\n") != strings.Join(wantDel, "\n") {
		t.Fatalf("line parity\nadded got:\n%s\nadded want:\n%s\nremoved got:\n%s\nremoved want:\n%s",
			strings.Join(gotAdd, "\n"), strings.Join(wantAdd, "\n"), strings.Join(gotDel, "\n"), strings.Join(wantDel, "\n"))
	}

	var binary bool
	for _, file := range got {
		if file.Path == "blob.bin" {
			binary = file.Binary && file.Text == "" && file.Patch == ""
		}
		if file.Path == "lib/note.txt" && !strings.Contains(file.Patch, "beta staged") {
			t.Fatalf("staged line missing:\n%s", file.Patch)
		}
	}
	if !binary {
		t.Fatal("expected blob.bin to be treated as binary with no content")
	}
}

func TestReadStagedEmpty(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "scrute@example.com")
	runGit(t, dir, "config", "user.name", "Scrute")
	mustWrite(t, filepath.Join(dir, "a.txt"), "hello\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "base")

	root, files, err := ReadStaged(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	gotRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	wantRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if gotRoot != wantRoot {
		t.Fatalf("root %s", root)
	}
	if len(files) != 0 {
		t.Fatalf("files %#v", files)
	}
}

func TestNotRepository(t *testing.T) {
	dir := t.TempDir()
	_, _, err := ReadStaged(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), ErrNotRepository.Error()) {
		t.Fatalf("err %v", err)
	}
}

func gitNameStatus(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := runGit(t, dir, "diff", "--cached", "--name-status", "-M")
	status := map[string]string{}
	if strings.TrimSpace(out) == "" {
		return status
	}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		fields := strings.Split(line, "\t")
		code := fields[0]
		switch {
		case strings.HasPrefix(code, "R"):
			status[fields[1]+" -> "+fields[2]] = "R"
		case strings.HasPrefix(code, "A"):
			status[fields[1]] = "A"
		case strings.HasPrefix(code, "D"):
			status[fields[1]] = "D"
		case strings.HasPrefix(code, "M"):
			status[fields[1]] = "M"
		default:
			t.Fatalf("unhandled name-status %q", line)
		}
	}
	return status
}

func gitDiff(t *testing.T, dir string) string {
	t.Helper()
	return runGit(t, dir, "diff", "--cached", "--no-ext-diff", "--unified=3", "-M")
}

func joinPatches(files []FileChange) string {
	var b strings.Builder
	for _, file := range files {
		b.WriteString(file.Patch)
	}
	return b.String()
}

func changedLines(diff string) (added, removed []string) {
	for _, line := range strings.Split(diff, "\n") {
		if line == "" || strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "new file") ||
			strings.HasPrefix(line, "deleted file") || strings.HasPrefix(line, "similarity ") ||
			strings.HasPrefix(line, "rename ") || strings.HasPrefix(line, "old mode") ||
			strings.HasPrefix(line, "new mode") || strings.HasPrefix(line, `\`) ||
			strings.HasPrefix(line, "Binary files") {
			continue
		}
		if strings.HasPrefix(line, "+") {
			added = append(added, strings.TrimRight(line[1:], "\r"))
		} else if strings.HasPrefix(line, "-") {
			removed = append(removed, strings.TrimRight(line[1:], "\r"))
		}
	}
	return added, removed
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Scrute",
		"GIT_AUTHOR_EMAIL=scrute@example.com",
		"GIT_COMMITTER_NAME=Scrute",
		"GIT_COMMITTER_EMAIL=scrute@example.com",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
