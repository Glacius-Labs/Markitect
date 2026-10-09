package projectworkspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGitServiceRenameRetainsEditedContentAndMode(t *testing.T) {
	fixture := newGitFixture(t)
	content := strings.Repeat("stable contextual line remains\n", 20) + "original final line\n"
	writeFixtureFile(t, fixture.root, "src/move.txt", []byte(content), 0644)
	testGit(t, fixture.root, "add", "src/move.txt")
	testGit(t, fixture.root, "commit", "--quiet", "-m", "rename baseline")
	fixture.base = strings.TrimSpace(testGit(t, fixture.root, "rev-parse", "HEAD"))
	service, r := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "rename-edit", []string{"src"}, nil)
	h, err := service.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close(context.Background(), h)
	if err := os.Rename(filepath.Join(h.CWD, "src/move.txt"), filepath.Join(h.CWD, "src/moved.txt")); err != nil {
		t.Fatal(err)
	}
	edited := strings.Repeat("stable contextual line remains\n", 20) + "edited final line\n"
	writeFixtureFile(t, h.CWD, "src/moved.txt", []byte(edited), 0755)
	if runtime.GOOS == "windows" {
		testGit(t, h.CWD, "add", "src/moved.txt")
		testGit(t, h.CWD, "update-index", "--chmod=+x", "src/moved.txt")
	}
	d, err := service.Harvest(context.Background(), h)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Changes) != 1 || d.Changes[0].Kind != ChangeRename || d.Changes[0].OldPath != "src/move.txt" || d.Changes[0].Path != "src/moved.txt" || d.Changes[0].Mode != "100755" || string(d.Changes[0].Content) != edited {
		t.Fatalf("edited rename lost provenance/mode/bytes: %+v", d.Changes)
	}
}

func TestGitServiceRejectsGitlinksBeforeSilentInventoryLoss(t *testing.T) {
	fixture := newGitFixture(t)
	testGit(t, fixture.root, "update-index", "--add", "--cacheinfo", "160000,"+fixture.base+",vendor/submodule")
	if err := os.MkdirAll(filepath.Join(fixture.root, "vendor/submodule"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectRepository(context.Background(), fixture.root, fixture.base); err == nil || !strings.Contains(err.Error(), "submodule") {
		t.Fatalf("gitlink omitted silently: %v", err)
	}
}

func TestGitServiceOverlayUsesConfiguredFiniteLimitsBeforeClone(t *testing.T) {
	fixture := newGitFixture(t)
	storage := filepath.Join(t.TempDir(), "storage")
	service, err := NewGitService(storage, Limits{MaxFiles: 2, MaxFileBytes: 2, MaxTotalBytes: 3})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := InspectRepository(context.Background(), fixture.root, fixture.base)
	if err != nil {
		t.Fatal(err)
	}
	r := Request{RepositoryRoot: fixture.root, RepositoryIdentity: "fixture", BaseSHA: fixture.base, OverlayDigest: binding.OverlayDigest, TaskID: "bounded", AllowedPaths: []string{"src"}}
	overlay := []Change{{Kind: ChangeAdd, Path: "src/huge.bin", Mode: "100644", Content: []byte{1, 2, 3}}}
	digest, err := CandidateOverlayDigest(overlay)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.PrepareCandidate(context.Background(), r, overlay, digest); err == nil {
		t.Fatal("oversized overlay bypassed service limit")
	}
	entries, err := os.ReadDir(storage)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed prepare left owned clone: %v %v", entries, err)
	}
}

func TestGitServiceRejectsLinkInventory(t *testing.T) {
	fixture := newGitFixture(t)
	target := filepath.Join(t.TempDir(), "foreign")
	if err := os.WriteFile(target, []byte("outside bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(fixture.root, "src/link")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("Windows symlink privilege unavailable")
		}
		t.Fatal(err)
	}
	if _, err := InspectRepository(context.Background(), fixture.root, fixture.base); err == nil {
		t.Fatal("outside symlink read as regular context")
	}
}

func TestGitServiceRunsNativeBuildWithCrossFileContext(t *testing.T) {
	fixture := newGitFixture(t)
	writeFixtureFile(t, fixture.root, "go.mod", []byte("module example.invalid/workspacefixture\n\ngo 1.27.1\n"), 0644)
	writeFixtureFile(t, fixture.root, "src/helper.go", []byte("package app\nfunc Current() string { return Version }\n"), 0644)
	testGit(t, fixture.root, "add", "go.mod", "src/helper.go")
	testGit(t, fixture.root, "commit", "--quiet", "-m", "cross-file build context")
	fixture.base = strings.TrimSpace(testGit(t, fixture.root, "rev-parse", "HEAD"))
	service, r := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "build", []string{"src"}, nil)
	h, err := service.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close(context.Background(), h)
	writeFixtureFile(t, h.CWD, "src/app_test.go", []byte("package app\nimport \"testing\"\nfunc TestCurrent(t *testing.T) { if Current()!=\"one\" { t.Fatal(Current()) } }\n"), 0644)
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Dir = h.CWD
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native candidate build failed: %v\n%s", err, output)
	}
	d, err := service.Harvest(context.Background(), h)
	if err != nil || len(d.Changes) != 1 || d.Changes[0].Path != "src/app_test.go" {
		t.Fatalf("native test delta wrong: %+v %v", d, err)
	}
}

func TestGitServiceObservedChangesBoundedBeforeRenameWork(t *testing.T) {
	fixture := newGitFixture(t)
	service, r := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "bounds", []string{"src"}, nil)
	service.limits = Limits{MaxFiles: 2, MaxFileBytes: 2, MaxTotalBytes: 3}
	h, err := service.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close(context.Background(), h)
	writeFixtureFile(t, h.CWD, "src/oversize.bin", []byte{0, 1, 2}, 0644)
	if _, err := service.Harvest(context.Background(), h); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("oversized new file accepted: %v", err)
	}
	if err := os.Remove(filepath.Join(h.CWD, "src/oversize.bin")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b", "c"} {
		writeFixtureFile(t, h.CWD, "src/"+name, []byte{1}, 0644)
	}
	if _, err := service.Harvest(context.Background(), h); err == nil {
		t.Fatal("too many observed additions accepted")
	}
}

func TestGitServiceRetainsSelectedBranchHistoryWithoutPushRemote(t *testing.T) {
	fixture := newGitFixture(t)
	first := fixture.base
	testGit(t, fixture.root, "branch", "context-branch", first)
	writeFixtureFile(t, fixture.root, "src/second.go", []byte("package app\nconst Second=true\n"), 0644)
	testGit(t, fixture.root, "add", "src/second.go")
	testGit(t, fixture.root, "commit", "--quiet", "-m", "second history commit")
	fixture.base = strings.TrimSpace(testGit(t, fixture.root, "rev-parse", "HEAD"))
	service, r := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "history", nil, nil)
	h, err := service.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close(context.Background(), h)
	if got := strings.TrimSpace(testGit(t, h.CWD, "rev-parse", "refs/remotes/origin/context-branch")); got != first {
		t.Fatalf("branch context lost: %s", got)
	}
	if got := strings.TrimSpace(testGit(t, h.CWD, "rev-parse", "HEAD~1")); got != first {
		t.Fatalf("earlier history fabricated or absent: %s", got)
	}
	if got := strings.TrimSpace(testGit(t, h.CWD, "remote")); got != "" {
		t.Fatalf("candidate has a push remote: %s", got)
	}
}

func TestGitServiceHostOperationalWritesDoNotStaleSourceOrLeakIntoContext(t *testing.T) {
	fixture := newGitFixture(t)
	writeFixtureFile(t, fixture.root, ".markitect/model.yaml", []byte("canonical: stable\n"), 0644)
	writeFixtureFile(t, fixture.root, ".markitect/runs/private/log.json", []byte("private host log\n"), 0644)
	service, r := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "host-records", []string{"src"}, nil)
	h, err := service.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close(context.Background(), h)
	if _, err := os.Stat(filepath.Join(h.CWD, ".markitect/runs/private/log.json")); !os.IsNotExist(err) {
		t.Fatalf("Host private record copied into context: %v", err)
	}
	writeFixtureFile(t, fixture.root, ".markitect/runs/private/log.json", []byte("updated host receipt\n"), 0644)
	if _, err := service.Harvest(context.Background(), h); err != nil {
		t.Fatalf("Host record staled task: %v", err)
	}
	writeFixtureFile(t, h.CWD, ".markitect/runs/fake.json", []byte("unauthorized task record"), 0644)
	if _, err := service.Harvest(context.Background(), h); err == nil {
		t.Fatal("task control write was silently excluded")
	}
	if err := os.Remove(filepath.Join(h.CWD, ".markitect/runs/fake.json")); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, fixture.root, ".markitect/model.yaml", []byte("canonical: changed\n"), 0644)
	if _, err := service.Harvest(context.Background(), h); err == nil {
		t.Fatal("canonical model edit did not stale task")
	}
}
