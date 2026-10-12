package projectrun

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"go.yaml.in/yaml/v3"
)

func TestImportExternalGitCandidatePreservesBinaryModesAndDeletes(t *testing.T) {
	root, baseCommit, finalCommit := newExternalGitCandidateRepo(t)
	base, err := projectwork.Load(root, baseCommit)
	if err != nil {
		t.Fatal(err)
	}
	baseDigest := base.Snapshot.Digest()
	identity, err := source.IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}
	request := projectworkspace.Request{
		RepositoryRoot: root, RepositoryIdentity: identity.Digest, BaseSHA: baseCommit,
		OverlayDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		TaskID:        "external-packet-1", AllowedPaths: []string{"src"},
	}
	handle := projectworkspace.Handle{
		ID: "external-git-fixture", CWD: root, RepositoryRoot: request.RepositoryRoot,
		RepositoryIdentity: request.RepositoryIdentity, BaseSHA: request.BaseSHA,
		OverlayDigest: request.OverlayDigest, TaskID: request.TaskID,
		BaseDigest: "sha256:" + baseDigest,
	}
	delta, err := ImportExternalGitCandidate(root, baseCommit, finalCommit, baseDigest, request, handle,
		projectworkspace.Limits{MaxFiles: 20, MaxFileBytes: 1 << 20, MaxTotalBytes: 2 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if delta.BaseSHA != request.BaseSHA || delta.BaseDigest != handle.BaseDigest || delta.TaskID != request.TaskID || delta.Digest == "" {
		t.Fatalf("imported delta lost request/handle binding: %+v", delta)
	}
	got := map[string]projectworkspace.Change{}
	for _, change := range delta.Changes {
		got[change.Path] = change
	}
	if len(got) != 4 {
		t.Fatalf("got %d changes, want modify, executable-mode change, add, and delete: %+v", len(got), got)
	}
	binary := got["src/changed.bin"]
	wantBinary := []byte{0x00, 0xff, 0x7f, 0x00, 0x80, 0x42}
	if binary.Kind != projectworkspace.ChangeModify || binary.Mode != "100644" || !bytes.Equal(binary.Content, wantBinary) {
		t.Fatalf("binary modification was not preserved: %+v", binary)
	}
	executable := got["src/script.sh"]
	if executable.Kind != projectworkspace.ChangeModify || executable.Mode != "100755" || !bytes.Equal(executable.Content, []byte("#!/bin/sh\necho final\n")) {
		t.Fatalf("mode/content change was not preserved: %+v", executable)
	}
	added := got["src/new.bin"]
	if added.Kind != projectworkspace.ChangeAdd || added.Mode != "100755" || !bytes.Equal(added.Content, []byte{0xfe, 0x00, 0x81}) {
		t.Fatalf("binary executable addition was not preserved: %+v", added)
	}
	deleted := got["src/removed.txt"]
	if deleted.Kind != projectworkspace.ChangeDelete || deleted.Content != nil || deleted.Mode != "" {
		t.Fatalf("deletion was not preserved as an empty delete: %+v", deleted)
	}
	if _, exists := got["src/unchanged.txt"]; exists {
		t.Fatal("unchanged path appeared in the imported delta")
	}
}

func TestImportExternalGitCandidateBindsBaseDigestAndAllowedScope(t *testing.T) {
	root, baseCommit, finalCommit := newExternalGitCandidateRepo(t)
	base, err := projectwork.Load(root, baseCommit)
	if err != nil {
		t.Fatal(err)
	}
	baseDigest := base.Snapshot.Digest()
	identity, err := source.IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}
	request := projectworkspace.Request{
		RepositoryRoot: root, RepositoryIdentity: identity.Digest, BaseSHA: baseCommit,
		OverlayDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		TaskID:        "external-packet-2", AllowedPaths: []string{"src/only"},
	}
	handle := projectworkspace.Handle{ID: "external-git-fixture", CWD: root, RepositoryRoot: root,
		RepositoryIdentity: identity.Digest, BaseSHA: baseCommit, OverlayDigest: request.OverlayDigest,
		TaskID: request.TaskID, BaseDigest: "sha256:" + baseDigest}
	limits := projectworkspace.Limits{MaxFiles: 20, MaxFileBytes: 1 << 20, MaxTotalBytes: 2 << 20}
	if _, err := ImportExternalGitCandidate(root, baseCommit, finalCommit, baseDigest, request, handle, limits); err == nil {
		t.Fatal("out-of-scope Git change was imported")
	}
	request.AllowedPaths = []string{"src"}
	if _, err := ImportExternalGitCandidate(root, baseCommit, finalCommit, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", request, handle, limits); err == nil {
		t.Fatal("a Git base unrelated to the packet candidate digest was imported")
	}
	if _, err := ImportExternalGitCandidate(root, baseCommit[:12], finalCommit, baseDigest, request, handle, limits); err == nil {
		t.Fatal("mutable/abbreviated base revision was accepted")
	}
}

func newExternalGitCandidateRepo(t *testing.T) (root, baseCommit, finalCommit string) {
	t.Helper()
	root = t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=NUL",
			"GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid",
			"GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	run("init", "-q")
	run("config", "user.name", "Fixture")
	run("config", "user.email", "fixture@example.invalid")
	run("switch", "-c", "dev/external-git-fixture")
	if _, err := projectwork.Init(root, "external-git-fixture", true); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest projectwork.Config
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.CoverageMode = "selected"
	manifest.WorkflowMode = projectwork.WorkflowModeEmpty
	manifest.InventoryRoots = []string{"src"}
	manifestBytes, err = yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	write := func(path string, data []byte, mode os.FileMode) {
		t.Helper()
		target := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, mode); err != nil {
			t.Fatal(err)
		}
	}
	write("src/changed.bin", []byte{0x00, 0x11, 0x7f, 0x00, 0x80, 0x42}, 0o644)
	write("src/removed.txt", []byte("removed\n"), 0o644)
	write("src/unchanged.txt", []byte("unchanged\n"), 0o644)
	write("src/script.sh", []byte("#!/bin/sh\necho base\n"), 0o644)
	run("add", "-A")
	run("commit", "-qm", "fixed external base")
	baseCommit = gitExternalOutput(t, root, "rev-parse", "HEAD")

	write("src/changed.bin", []byte{0x00, 0xff, 0x7f, 0x00, 0x80, 0x42}, 0o644)
	if err := os.Remove(filepath.Join(root, "src", "removed.txt")); err != nil {
		t.Fatal(err)
	}
	write("src/script.sh", []byte("#!/bin/sh\necho final\n"), 0o755)
	write("src/new.bin", []byte{0xfe, 0x00, 0x81}, 0o644)
	run("add", "-A", "--", "src")
	run("update-index", "--chmod=+x", "--", "src/new.bin", "src/script.sh")
	run("commit", "-qm", "external candidate")
	finalCommit = gitExternalOutput(t, root, "rev-parse", "HEAD")
	// Mutable checkout bytes must not affect an import bound to finalCommit.
	write("src/changed.bin", []byte("uncommitted checkout data"), 0o644)
	return root, baseCommit, finalCommit
}

func gitExternalOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=NUL")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(bytes.TrimSpace(output))
}
