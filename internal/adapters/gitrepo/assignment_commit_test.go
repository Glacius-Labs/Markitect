package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestReadAssignmentCommitUsesNamedObjectsAndIgnoresArchiveAttributes(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "source")
	source := copies["source"]
	integrationWrite(t, roots["source"], ".gitattributes", "common.txt export-ignore\ntemplate.txt export-subst\n")
	integrationWrite(t, roots["source"], "template.txt", "$Format:%H$\n")
	commit, err := source.Commit(ctx, "Named check input")
	if err != nil {
		t.Fatal(err)
	}
	integrationWrite(t, roots["source"], "common.txt", "later committed result\n")
	latest, err := source.Commit(ctx, "Later result")
	if err != nil {
		t.Fatal(err)
	}
	integrationWrite(t, roots["source"], "common.txt", "dirty current work\n")
	before, _ := source.git(ctx, "status", "--porcelain")
	request := run.AssignmentCommitRequest{Source: source, Commit: commit, MaxEntries: 20, MaxBytes: 4096}
	tree, err := w.ReadAssignmentCommit(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"common.txt": "shared baseline\n", "template.txt": "$Format:%H$\n"} {
		path, _ := snapshot.NewPath(name)
		entry, ok := tree.Lookup(path)
		if !ok || string(entry.Bytes()) != want {
			t.Fatalf("named bytes for %s: %q", name, entry.Bytes())
		}
	}
	after, _ := source.git(ctx, "status", "--porcelain")
	head, _ := source.git(ctx, "rev-parse", "HEAD")
	current, _ := os.ReadFile(filepath.Join(roots["source"], "common.txt"))
	if before != after || head != latest || string(current) != "dirty current work\n" {
		t.Fatal("reading a named commit changed the live source")
	}
	request.MaxEntries = 1
	if _, err = w.ReadAssignmentCommit(ctx, request); !errors.Is(err, ErrLimit) {
		t.Fatalf("entry budget: %v", err)
	}
	request.MaxEntries, request.MaxBytes = 20, 1
	if _, err = w.ReadAssignmentCommit(ctx, request); !errors.Is(err, ErrLimit) {
		t.Fatalf("byte budget: %v", err)
	}
	request.MaxBytes, request.Commit = 4096, strings.Repeat("f", 40)
	if _, err = w.ReadAssignmentCommit(ctx, request); err == nil {
		t.Fatal("read an unavailable commit")
	}
	request.Commit, err = source.git(ctx, "rev-parse", commit+"^{tree}")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.ReadAssignmentCommit(ctx, request); err == nil {
		t.Fatal("accepted a raw tree object as a submitted commit")
	}
}

type damagedCommitRuntime struct {
	WorkspaceRuntime
	mode string
}

func (f damagedCommitRuntime) RunHelper(ctx context.Context, spec container.ContainerSpec, args []string, in io.Reader, out io.Writer) error {
	var data bytes.Buffer
	if err := f.WorkspaceRuntime.RunHelper(ctx, spec, args, in, &data); err != nil {
		return err
	}
	content := data.Bytes()
	if strings.Contains(strings.Join(args, " "), "cat-file") {
		switch f.mode {
		case "truncated":
			content = content[:len(content)-1]
		case "corrupt":
			if pos := bytes.IndexByte(content, '\n'); pos >= 0 && pos+1 < len(content) {
				content[pos+1] ^= 1
			}
		case "helper failure":
			_, _ = out.Write(content)
			return errors.New("helper failed after output")
		}
	}
	_, err := out.Write(content)
	return err
}

func TestReadAssignmentCommitRefusesCorruptPartialOrFailedOutput(t *testing.T) {
	w, copies, _ := integrationFixture(t, "source")
	runtime := w.runtime
	for _, mode := range []string{"truncated", "corrupt", "helper failure"} {
		t.Run(mode, func(t *testing.T) {
			w.runtime = damagedCommitRuntime{WorkspaceRuntime: runtime, mode: mode}
			tree, err := w.ReadAssignmentCommit(context.Background(), run.AssignmentCommitRequest{Source: copies["source"], Commit: copies["source"].StartingCommit(), MaxEntries: 20, MaxBytes: 4096})
			if err == nil || tree.EntryCount() != 0 {
				t.Fatalf("accepted damaged output: %v", err)
			}
		})
	}
}

func TestReadAssignmentCommitPreservesModeAndRefusesSymlink(t *testing.T) {
	ctx := context.Background()
	w, copies, _ := integrationFixture(t, "source")
	source := copies["source"]
	if _, err := source.git(ctx, "update-index", "--chmod=+x", "common.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.git(ctx, "commit", "-m", "Executable result"); err != nil {
		t.Fatal(err)
	}
	commit, _ := source.git(ctx, "rev-parse", "HEAD")
	request := run.AssignmentCommitRequest{Source: source, Commit: commit, MaxEntries: 20, MaxBytes: 4096}
	tree, err := w.ReadAssignmentCommit(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := snapshot.NewPath("common.txt")
	entry, _ := tree.Lookup(path)
	if entry.Mode() != snapshot.Executable {
		t.Fatal("named Git mode was lost")
	}
	oid, _ := source.git(ctx, "rev-parse", commit+":common.txt")
	if _, err := source.git(ctx, "update-index", "--add", "--cacheinfo", "120000,"+oid+",link"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.git(ctx, "commit", "-m", "Unsupported symbolic entry"); err != nil {
		t.Fatal(err)
	}
	request.Commit, _ = source.git(ctx, "rev-parse", "HEAD")
	if _, err = w.ReadAssignmentCommit(ctx, request); err == nil {
		t.Fatal("accepted a symlink as ordinary content")
	}
}
