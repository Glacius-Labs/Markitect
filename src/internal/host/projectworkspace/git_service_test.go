package projectworkspace

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var gitServiceTestLimits = Limits{MaxFiles: 32, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20}

type gitFixture struct {
	root string
	base string
}

func newGitFixture(t *testing.T) gitFixture {
	t.Helper()
	root := filepath.Join(t.TempDir(), "adopter")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	testGit(t, root, "init", "--quiet")
	testGit(t, root, "config", "user.name", "Workspace Test")
	testGit(t, root, "config", "user.email", "workspace-test@example.invalid")
	writeFixtureFile(t, root, "src/app.go", []byte("package app\n\n// Original context.\nconst Version = \"one\"\n"), 0o644)
	writeFixtureFile(t, root, "src/runner.sh", []byte("#!/bin/sh\nprintf original\n"), 0o755)
	writeFixtureFile(t, root, "docs/guide.md", []byte("# Guide\n\nStable documentation.\n"), 0o644)
	writeFixtureFile(t, root, "assets/original.bin", []byte{0, 0xff, 0x10, 0x80, 0x00}, 0o644)
	writeFixtureFile(t, root, "assets/changed.bin", []byte{0, 0x80, 0x11, 0x80, 0x00}, 0o644)
	testGit(t, root, "add", "src/app.go", "src/runner.sh", "docs/guide.md", "assets/original.bin", "assets/changed.bin")
	testGit(t, root, "update-index", "--chmod=+x", "src/runner.sh")
	testGit(t, root, "commit", "--quiet", "-m", "initial project")
	base := strings.TrimSpace(testGit(t, root, "rev-parse", "HEAD"))
	return gitFixture{root: root, base: base}
}

func newGitServiceRequest(t *testing.T, fixture gitFixture, storageRoot, task string, allowed, excluded []string) (*GitService, Request) {
	t.Helper()
	service, err := NewGitService(storageRoot, gitServiceTestLimits)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := InspectRepository(context.Background(), fixture.root, fixture.base)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{
		RepositoryRoot:     fixture.root,
		RepositoryIdentity: "repo:workspace-fixture",
		BaseSHA:            fixture.base,
		OverlayDigest:      binding.OverlayDigest,
		TaskID:             task,
		AllowedPaths:       allowed,
		ExcludedPaths:      excluded,
	}
	return service, request
}

func TestGitServiceCandidateHasRealHistoryAndPreservesAdopterWIP(t *testing.T) {
	fixture := newGitFixture(t)
	writeFixtureFile(t, fixture.root, "src/app.go", []byte("package app\n\n// Original context.\nconst Version = \"adopter-wip\"\n"), 0o644)
	writeFixtureFile(t, fixture.root, "src/local-only.go", []byte("package app\nconst Local = true\n"), 0o644)
	testGit(t, fixture.root, "add", "src/app.go") // Preserve a staged adopter edit too.
	storageRoot := filepath.Join(t.TempDir(), "owned-workspaces")
	service, request := newGitServiceRequest(t, fixture, storageRoot, "manager:history", []string{"src"}, nil)
	handle, err := service.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close(context.Background(), handle) })

	if got := strings.TrimSpace(testGit(t, handle.CWD, "log", "-1", "--format=%s")); got != "initial project" {
		t.Fatalf("candidate lacks the repository's real commit history: %q", got)
	}
	if got := strings.TrimSpace(testGit(t, handle.CWD, "blame", "-L", "4,4", "--porcelain", "HEAD", "--", "src/app.go")); !strings.HasPrefix(got, fixture.base) {
		t.Fatalf("candidate blame lacks committed source attribution: %q", got)
	}
	if got, err := os.ReadFile(filepath.Join(handle.CWD, "src/app.go")); err != nil || !bytes.Contains(got, []byte("adopter-wip")) {
		t.Fatalf("candidate did not receive adopter's WIP: content=%q err=%v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(handle.CWD, "src/local-only.go")); err != nil || !bytes.Contains(got, []byte("Local = true")) {
		t.Fatalf("candidate did not receive adopter's untracked WIP: content=%q err=%v", got, err)
	}

	writeFixtureFile(t, handle.CWD, "src/feature.go", []byte("package app\nconst Feature = true\n"), 0o644)
	delta, err := service.Harvest(context.Background(), handle)
	if err != nil {
		t.Fatal(err)
	}
	if len(delta.Changes) != 1 || delta.Changes[0].Kind != ChangeAdd || delta.Changes[0].Path != "src/feature.go" {
		t.Fatalf("harvest did not isolate candidate change from inherited WIP: %#v", delta.Changes)
	}
	if got, err := os.ReadFile(filepath.Join(fixture.root, "src/app.go")); err != nil || !bytes.Contains(got, []byte("adopter-wip")) {
		t.Fatalf("candidate work changed adopter WIP: content=%q err=%v", got, err)
	}
	if got := testGit(t, fixture.root, "status", "--porcelain"); !strings.Contains(got, "src/app.go") || !strings.Contains(got, "src/local-only.go") {
		t.Fatalf("adopter worktree status was not preserved: %q", got)
	}
}

func TestGitServiceHarvestCapturesAddModifyDeleteRenameModeAndBinary(t *testing.T) {
	fixture := newGitFixture(t)
	storageRoot := filepath.Join(t.TempDir(), "owned-workspaces")
	service, request := newGitServiceRequest(t, fixture, storageRoot, "manager:complete-delta", []string{"src", "docs", "assets"}, nil)
	handle, err := service.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close(context.Background(), handle) })

	writeFixtureFile(t, handle.CWD, "src/app.go", []byte("package app\n\n// Original context.\nconst Version = \"two\"\n"), 0o644)
	writeFixtureFile(t, handle.CWD, "src/new-tool.sh", []byte("#!/bin/sh\nprintf tool\n"), 0o755)
	writeFixtureFile(t, handle.CWD, "src/runner.sh", []byte("#!/bin/sh\nprintf updated\n"), 0o755)
	if err := os.Remove(filepath.Join(handle.CWD, "docs/guide.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(handle.CWD, "assets/original.bin"), filepath.Join(handle.CWD, "assets/renamed.bin")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(handle.CWD, "assets/changed.bin"), []byte{0, 0x80, 0x11, 0x81, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, handle.CWD, "assets/new.bin", []byte{0x00, 0xfe, 0x91, 0x00}, 0o644)

	delta, err := service.Harvest(context.Background(), handle)
	if err != nil {
		t.Fatal(err)
	}
	byPath := make(map[string]Change)
	for _, change := range delta.Changes {
		byPath[change.Path] = change
	}
	if got := byPath["src/app.go"]; got.Kind != ChangeModify || !bytes.Contains(got.Content, []byte(`Version = "two"`)) {
		t.Fatalf("modify missing or incorrect: %#v", got)
	}
	expectedAddMode := "100755"
	if runtime.GOOS == "windows" {
		expectedAddMode = "100644" // Windows cannot express an executable bit on an untracked file.
	}
	if got := byPath["src/new-tool.sh"]; got.Kind != ChangeAdd || got.Mode != expectedAddMode {
		t.Fatalf("add or executable mode missing: %#v", got)
	}
	if got := byPath["src/runner.sh"]; got.Kind != ChangeModify || got.Mode != "100755" {
		t.Fatalf("tracked executable mode was not retained: %#v", got)
	}
	if got := byPath["docs/guide.md"]; got.Kind != ChangeDelete {
		t.Fatalf("delete missing: %#v", got)
	}
	if got := byPath["assets/renamed.bin"]; got.Kind != ChangeRename || got.OldPath != "assets/original.bin" || !bytes.Equal(got.Content, []byte{0, 0xff, 0x10, 0x80, 0x00}) {
		t.Fatalf("rename or binary bytes missing: %#v", got)
	}
	if got := byPath["assets/changed.bin"]; got.Kind != ChangeModify || !bytes.Equal(got.Content, []byte{0, 0x80, 0x11, 0x81, 0x00}) {
		t.Fatalf("binary modification bytes missing: %#v", got)
	}
	if got := byPath["assets/new.bin"]; got.Kind != ChangeAdd || !bytes.Equal(got.Content, []byte{0x00, 0xfe, 0x91, 0x00}) {
		t.Fatalf("binary add bytes missing: %#v", got)
	}
}

func TestGitServiceEmptyOwnershipRejectsWritesAndAllowsEmptyCandidate(t *testing.T) {
	fixture := newGitFixture(t)
	service, request := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "workspaces"), "manager:read-only", nil, nil)
	handle, err := service.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Close(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
	handle, err = service.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close(context.Background(), handle)
	writeFixtureFile(t, handle.CWD, "src/app.go", []byte("candidate write\n"), 0o644)
	if _, err := service.Harvest(context.Background(), handle); err == nil {
		t.Fatal("empty ownership scope accepted candidate writes")
	}
}

func TestGitServiceRejectsOutOfScopeAndControlPlaneChanges(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "outside ownership", path: "docs/guide.md"},
		{name: "control plane", path: ".markitect/policy.yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newGitFixture(t)
			service, request := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "workspaces"), "manager:scope-"+strings.ReplaceAll(tc.name, " ", "-"), []string{"src"}, nil)
			handle, err := service.Prepare(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			defer service.Close(context.Background(), handle)
			writeFixtureFile(t, handle.CWD, tc.path, []byte("unauthorized\n"), 0o644)
			if _, err := service.Harvest(context.Background(), handle); err == nil {
				t.Fatalf("unauthorized path %q was accepted", tc.path)
			}
		})
	}
}

func TestGitServiceRejectsStaleAdopterAndForgedHandle(t *testing.T) {
	fixture := newGitFixture(t)
	service, request := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "workspaces"), "manager:stale", []string{"src"}, nil)
	handle, err := service.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	forged := handle
	forged.ID = forged.ID + "-forged"
	if _, err := service.Harvest(context.Background(), forged); err == nil {
		t.Fatal("forged workspace handle was accepted")
	}
	if err := os.WriteFile(filepath.Join(fixture.root, "src/app.go"), []byte("new adopter state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Harvest(context.Background(), handle); err == nil {
		t.Fatal("harvest accepted a source repository changed after prepare")
	}
	if err := service.Close(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
}

func TestGitServiceConcurrentCandidatesAndCloseOwnWorkspaceOnly(t *testing.T) {
	fixture := newGitFixture(t)
	storageRoot := filepath.Join(t.TempDir(), "owned-workspaces")
	service, firstRequest := newGitServiceRequest(t, fixture, storageRoot, "manager:first", []string{"src"}, nil)
	secondRequest := firstRequest
	secondRequest.TaskID = "manager:second"
	first, err := service.Prepare(context.Background(), firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Prepare(context.Background(), secondRequest)
	if err != nil {
		_ = service.Close(context.Background(), first)
		t.Fatal(err)
	}
	if first.CWD == second.CWD {
		t.Fatalf("concurrent candidates share a worktree: %q", first.CWD)
	}
	if err := service.Close(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(second.CWD); err != nil {
		t.Fatalf("closing first candidate removed second candidate: %v", err)
	}
	if _, err := service.Harvest(context.Background(), second); err != nil {
		t.Fatalf("second candidate was unusable after closing first: %v", err)
	}
	if err := service.Close(context.Background(), second); err != nil {
		t.Fatal(err)
	}
}

func TestGitServiceRequiresSelectedBaseToEqualSourceHead(t *testing.T) {
	fixture := newGitFixture(t)
	storageRoot := filepath.Join(t.TempDir(), "workspaces")
	_, err := NewGitService(storageRoot, gitServiceTestLimits)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InspectRepository(context.Background(), fixture.root, strings.Repeat("0", 40)); err == nil {
		t.Fatal("repository inspection accepted a selected base other than source HEAD")
	}
}

func writeFixtureFile(t *testing.T, root, relative string, content []byte, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && mode&0o111 != 0 {
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
	}
}

func testGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), directory, err, output)
	}
	return string(output)
}
