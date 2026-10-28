// Package gitdiff reads the staged diff (index versus HEAD) with go-git.
// Object writes stay in memory so a review never adds objects to the repository.
package gitdiff

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage/memory"
)

const (
	Added    = "added"
	Modified = "modified"
	Deleted  = "deleted"
	Renamed  = "renamed"
)

var (
	// ErrNotRepository means dir is not inside a work tree.
	ErrNotRepository = errors.New("not a git repository")
	// ErrConflict means the index has unmerged stages.
	ErrConflict = errors.New("resolve merge conflicts before review")
	// ErrBare means the repository has no work tree.
	ErrBare = errors.New("bare repositories are not supported")
)

// Span is a 1-based inclusive line range in the new file.
type Span struct {
	Start int
	End   int
}

// FileChange is one staged path.
type FileChange struct {
	Path      string
	OldPath   string
	Status    string
	Binary    bool
	Submodule bool
	Patch     string
	Text      string
	Hunks     []Span
}

// FindRoot returns the work tree root that contains dir.
func FindRoot(dir string) (string, error) {
	repo, err := open(dir)
	if err != nil {
		return "", err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", ErrBare
	}
	root := wt.Filesystem.Root()
	if root == "" {
		return "", ErrNotRepository
	}
	return root, nil
}

// GitDir returns the .git directory for the work tree, including linked work trees.
func GitDir(root string) (string, error) {
	gitPath := filepath.Join(root, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", ErrNotRepository)
	}
	if info.IsDir() {
		return gitPath, nil
	}
	body, err := os.ReadFile(gitPath)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(body))
	const prefix = "gitdir:"
	if !strings.HasPrefix(line, prefix) {
		return "", fmt.Errorf("cannot read git dir from %s", gitPath)
	}
	dir := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(root, dir)
	}
	return filepath.Clean(dir), nil
}

// ReadStaged returns staged changes at dir. An empty slice means nothing is staged.
func ReadStaged(ctx context.Context, dir string) (string, []FileChange, error) {
	if ctx == nil {
		return "", nil, errors.New("nil context")
	}
	repo, err := open(dir)
	if err != nil {
		return "", nil, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", nil, ErrBare
	}
	root := wt.Filesystem.Root()

	headTree, err := headTree(repo)
	if err != nil {
		return "", nil, err
	}
	indexTree, err := indexTree(repo)
	if err != nil {
		return "", nil, err
	}

	changes, err := object.DiffTreeWithOptions(ctx, headTree, indexTree, object.DefaultDiffTreeOptions)
	if err != nil {
		return "", nil, fmt.Errorf("diff staged changes: %w", err)
	}
	patch, err := changes.PatchContext(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("build staged patch: %w", err)
	}

	var files []FileChange
	patches := patch.FilePatches()
	for i, fp := range patches {
		var src *object.Change
		if i < len(changes) {
			src = changes[i]
		}
		file, err := fileChange(repo, fp, src)
		if err != nil {
			return "", nil, err
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return root, files, nil
}

func open(dir string) (*git.Repository, error) {
	repo, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", ErrNotRepository)
	}
	return repo, nil
}

func headTree(repo *git.Repository) (*object.Tree, error) {
	head, err := repo.Head()
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("read HEAD: %w", err)
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("read HEAD commit: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("read HEAD tree: %w", err)
	}
	return tree, nil
}

func indexTree(repo *git.Repository) (*object.Tree, error) {
	idx, err := repo.Storer.Index()
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	// go-git's Merged constant is 1, but the decoder stores git's raw stage.
	// Stage 0 is a normal index entry. Any other stage is an unmerged path.
	for _, entry := range idx.Entries {
		if entry.Stage != 0 {
			return nil, fmt.Errorf("%s: %w", entry.Name, ErrConflict)
		}
	}

	mem := memory.NewStorage()
	root := &inode{dir: map[string]*inode{}}
	for _, entry := range idx.Entries {
		mode := canonMode(entry.Mode)
		hash := entry.Hash
		if mode == filemode.Submodule {
			if err := insert(root, entry.Name, mode, hash); err != nil {
				return nil, err
			}
			continue
		}
		var blob []byte
		if hash == plumbing.ZeroHash {
			blob = nil
		} else {
			blob, err = readBlob(repo.Storer, hash)
			if err != nil {
				return nil, fmt.Errorf("read staged blob %s: %w", entry.Name, err)
			}
		}
		stored, err := storeBlob(mem, blob)
		if err != nil {
			return nil, err
		}
		if err := insert(root, entry.Name, mode, stored); err != nil {
			return nil, err
		}
	}
	hash, err := writeTree(mem, root)
	if err != nil {
		return nil, err
	}
	tree, err := object.GetTree(mem, hash)
	if err != nil {
		return nil, fmt.Errorf("load index tree: %w", err)
	}
	return tree, nil
}

type inode struct {
	mode filemode.FileMode
	hash plumbing.Hash
	dir  map[string]*inode
}

func insert(root *inode, path string, mode filemode.FileMode, hash plumbing.Hash) error {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")
	cur := root
	for i, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("invalid index path %q", path)
		}
		if cur.dir == nil {
			return fmt.Errorf("path conflict at %s", path)
		}
		last := i == len(parts)-1
		child := cur.dir[part]
		if last {
			if child != nil && child.dir != nil {
				return fmt.Errorf("path conflict at %s", path)
			}
			cur.dir[part] = &inode{mode: mode, hash: hash}
			return nil
		}
		if child == nil {
			child = &inode{dir: map[string]*inode{}}
			cur.dir[part] = child
		}
		if child.dir == nil {
			return fmt.Errorf("path conflict at %s", path)
		}
		cur = child
	}
	return fmt.Errorf("invalid index path %q", path)
}

func writeTree(mem storer.EncodedObjectStorer, node *inode) (plumbing.Hash, error) {
	names := make([]string, 0, len(node.dir))
	for name := range node.dir {
		names = append(names, name)
	}
	entries := make([]object.TreeEntry, 0, len(names))
	for _, name := range names {
		child := node.dir[name]
		entry := object.TreeEntry{Name: name}
		if child.dir != nil {
			hash, err := writeTree(mem, child)
			if err != nil {
				return plumbing.ZeroHash, err
			}
			entry.Mode = filemode.Dir
			entry.Hash = hash
		} else {
			entry.Mode = child.mode
			entry.Hash = child.hash
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return treeSortKey(entries[i]) < treeSortKey(entries[j])
	})
	tree := &object.Tree{Entries: entries}
	obj := mem.NewEncodedObject()
	if err := tree.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encode tree: %w", err)
	}
	hash, err := mem.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("store tree: %w", err)
	}
	return hash, nil
}

func treeSortKey(entry object.TreeEntry) string {
	if entry.Mode == filemode.Dir {
		return entry.Name + "/"
	}
	return entry.Name
}

func canonMode(mode filemode.FileMode) filemode.FileMode {
	switch mode & 0170000 {
	case 0040000:
		return filemode.Dir
	case 0120000:
		return filemode.Symlink
	case 0160000:
		return filemode.Submodule
	case 0100000:
		if mode&0111 != 0 {
			return filemode.Executable
		}
		return filemode.Regular
	default:
		if mode.IsFile() {
			return mode
		}
		return filemode.Regular
	}
}

func readBlob(src storer.EncodedObjectStorer, hash plumbing.Hash) ([]byte, error) {
	obj, err := src.EncodedObject(plumbing.BlobObject, hash)
	if err != nil {
		return nil, err
	}
	reader, err := obj.Reader()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return io.ReadAll(reader)
}

func storeBlob(mem storer.EncodedObjectStorer, body []byte) (plumbing.Hash, error) {
	obj := mem.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	writer, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return plumbing.ZeroHash, err
	}
	if err := writer.Close(); err != nil {
		return plumbing.ZeroHash, err
	}
	hash, err := mem.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return hash, nil
}

type singlePatch struct {
	fp fdiff.FilePatch
}

func (s singlePatch) FilePatches() []fdiff.FilePatch { return []fdiff.FilePatch{s.fp} }
func (s singlePatch) Message() string                { return "" }

func fileChange(repo *git.Repository, fp fdiff.FilePatch, src *object.Change) (FileChange, error) {
	if src != nil && (entrySubmodule(src.From) || entrySubmodule(src.To)) {
		return submoduleChange(src), nil
	}
	from, to := fp.Files()
	change := FileChange{}
	switch {
	case from == nil && to != nil:
		change.Status = Added
		change.Path = to.Path()
	case from != nil && to == nil:
		change.Status = Deleted
		change.Path = from.Path()
	case from != nil && to != nil && from.Path() != to.Path():
		change.Status = Renamed
		change.OldPath = from.Path()
		change.Path = to.Path()
	case to != nil:
		change.Status = Modified
		change.Path = to.Path()
	default:
		change.Status = Modified
	}

	if (from != nil && from.Mode() == filemode.Submodule) || (to != nil && to.Mode() == filemode.Submodule) {
		change.Submodule = true
		change.Binary = true
		return change, nil
	}

	var fromBytes, toBytes []byte
	var err error
	if from != nil {
		fromBytes, err = readBlob(repo.Storer, from.Hash())
		if err != nil {
			return FileChange{}, fmt.Errorf("read %s: %w", from.Path(), err)
		}
	}
	if to != nil {
		toBytes, err = readBlob(repo.Storer, to.Hash())
		if err != nil {
			return FileChange{}, fmt.Errorf("read %s: %w", to.Path(), err)
		}
		if !isBinary(toBytes) {
			change.Text = string(toBytes)
		}
	}
	// A deletion has no index blob. Numbered context comes from HEAD, which is
	// the file that the staged delete removes.
	if to == nil && !isBinary(fromBytes) {
		change.Text = string(fromBytes)
	}
	// A NUL in the blob is the binary signal. go-git also treats an empty chunk
	// list as binary, which would mislabel an empty text file.
	if isBinary(fromBytes) || isBinary(toBytes) {
		change.Binary = true
		change.Text = ""
		return change, nil
	}

	var buf bytes.Buffer
	enc := fdiff.NewUnifiedEncoder(&buf, fdiff.DefaultContextLines)
	if err := enc.Encode(singlePatch{fp}); err != nil {
		return FileChange{}, fmt.Errorf("encode patch %s: %w", change.Path, err)
	}
	change.Patch = buf.String()
	deleted := change.Status == Deleted
	change.Hunks = hunks(fp, len(strings.Split(strings.TrimSuffix(change.Text, "\n"), "\n")), deleted)
	return change, nil
}

func entrySubmodule(entry object.ChangeEntry) bool {
	return entry.Name != "" && entry.TreeEntry.Mode == filemode.Submodule
}

func submoduleChange(src *object.Change) FileChange {
	file := FileChange{Submodule: true, Binary: true}
	switch {
	case src.From.Name == "":
		file.Status = Added
		file.Path = src.To.Name
	case src.To.Name == "":
		file.Status = Deleted
		file.Path = src.From.Name
	case src.From.Name != src.To.Name:
		file.Status = Renamed
		file.OldPath = src.From.Name
		file.Path = src.To.Name
	default:
		file.Status = Modified
		file.Path = src.To.Name
	}
	return file
}

func isBinary(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	n := len(body)
	if n > 8000 {
		n = 8000
	}
	return bytes.IndexByte(body[:n], 0) >= 0
}

func hunks(fp fdiff.FilePatch, fileLines int, deleted bool) []Span {
	oldLine := 1
	newLine := 1
	var spans []Span
	for _, chunk := range fp.Chunks() {
		n := countLines(chunk.Content())
		switch chunk.Type() {
		case fdiff.Equal:
			oldLine += n
			newLine += n
		case fdiff.Delete:
			if deleted {
				start := oldLine
				end := oldLine
				if n > 0 {
					end = oldLine + n - 1
				}
				spans = append(spans, Span{Start: start, End: end})
				oldLine += n
				continue
			}
			start := newLine
			if start < 1 {
				start = 1
			}
			spans = append(spans, Span{Start: start, End: start})
		case fdiff.Add:
			if n == 0 {
				continue
			}
			if !deleted {
				spans = append(spans, Span{Start: newLine, End: newLine + n - 1})
			}
			newLine += n
		}
	}
	if fileLines > 0 {
		for i := range spans {
			if spans[i].Start > fileLines {
				spans[i].Start = fileLines
			}
			if spans[i].End > fileLines {
				spans[i].End = fileLines
			}
			if spans[i].End < spans[i].Start {
				spans[i].End = spans[i].Start
			}
		}
	}
	return spans
}

func countLines(content string) int {
	if content == "" {
		return 0
	}
	n := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		n++
	}
	return n
}
