package source

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

func TestObserveSelectedWorkingReadsExactPathsAndReportsMissing(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	repo.Write("selected/data.txt", "selected bytes")
	repo.Write("selected/extra.txt", "unselected bytes")
	repo.Write("outside.txt", "outside bytes")
	got, err := ObserveSelectedWorking(root, []string{"selected/data.txt", "missing.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Requested, []string{"missing.txt", "selected/data.txt"}) || !reflect.DeepEqual(got.MissingPaths, []string{"missing.txt"}) {
		t.Fatalf("requested/missing = %#v/%#v", got.Requested, got.MissingPaths)
	}
	if got.Snapshot == nil || !got.Snapshot.Provisional || len(got.Snapshot.Files) != 1 || string(got.Snapshot.Files["selected/data.txt"]) != "selected bytes" {
		t.Fatalf("selected working snapshot = %#v", got.Snapshot)
	}
	if _, ok := got.Snapshot.Files["selected/extra.txt"]; ok {
		t.Fatal("unselected sibling was included")
	}
	if got.Snapshot.Modes["selected/data.txt"] != "100644" {
		t.Fatalf("mode = %q, want regular-file mode", got.Snapshot.Modes["selected/data.txt"])
	}
}

func TestObserveSelectedWorkingUsesIndexModeWhenGitDisablesFileMode(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	repo.Write("selected/tool.sh", "#!/bin/sh\n")
	repo.Git("add", "selected/tool.sh")
	repo.Git("update-index", "--chmod=+x", "selected/tool.sh")
	repo.Git("config", "core.filemode", "false")
	indexModes, err := selectedIndexModes(root, []string{"selected/tool.sh"})
	if err != nil || indexModes["selected/tool.sh"] != "100755" {
		t.Fatalf("selected Git index modes = %#v, err=%v", indexModes, err)
	}
	got, err := ObserveSelectedWorking(root, []string{"selected/tool.sh"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Snapshot.Modes["selected/tool.sh"] != "100755" {
		t.Fatalf("tracked worktree mode = %q, want Git index mode 100755", got.Snapshot.Modes["selected/tool.sh"])
	}

	repo.Write("selected/untracked.sh", "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(root, "selected", "untracked.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	untracked, err := ObserveSelectedWorking(root, []string{"selected/untracked.sh"})
	if err != nil {
		t.Fatal(err)
	}
	want := "100644"
	if runtime.GOOS != "windows" {
		want = "100755"
	}
	if untracked.Snapshot.Modes["selected/untracked.sh"] != want {
		t.Fatalf("untracked mode = %q, want filesystem mode %q", untracked.Snapshot.Modes["selected/untracked.sh"], want)
	}
}

func TestObserveSelectedWorkingDefaultsToFilesystemModeWhenFileModeIsUnset(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	repo.Write("selected/tool.sh", "#!/bin/sh\n")
	repo.Git("add", "selected/tool.sh")
	repo.Git("update-index", "--chmod=+x", "selected/tool.sh")
	// Ensure the key exists before removing it, then verify no system/global
	// setting shadows Git's documented default in this test environment.
	repo.Git("config", "--local", "core.filemode", "true")
	repo.Git("config", "--local", "--unset-all", "core.filemode")
	cmd := exec.Command("git", "--no-replace-objects", "-C", root, "config", "--show-origin", "--get", "core.filemode")
	cmd.Env = CleanGitEnv()
	if output, err := cmd.CombinedOutput(); err == nil {
		t.Skipf("core.filemode is set outside the local test repository: %s", strings.TrimSpace(string(output)))
	} else {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
			t.Fatalf("inspect inherited core.filemode: %v\n%s", err, output)
		}
	}

	enabled, err := GitFileModeEnabled(root)
	if err != nil || !enabled {
		t.Fatalf("unset core.filemode = %v, err=%v; want Git default true", enabled, err)
	}
	got, err := ObserveSelectedWorking(root, []string{"selected/tool.sh"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Snapshot.Modes["selected/tool.sh"] != "100644" {
		t.Fatalf("tracked worktree mode = %q, want observed filesystem mode 100644 when core.filemode is unset", got.Snapshot.Modes["selected/tool.sh"])
	}
}

func TestObserveSelectedWorkingPreservesFilesystemModeWhenGitEnablesFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows filesystems do not expose POSIX executable mode bits")
	}
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	repo.Write("tool.sh", "#!/bin/sh\n")
	repo.Git("add", "tool.sh")
	repo.Git("update-index", "--chmod=+x", "tool.sh")
	repo.Git("config", "core.filemode", "true")
	if err := os.Chmod(filepath.Join(root, "tool.sh"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ObserveSelectedWorking(root, []string{"tool.sh"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Snapshot.Modes["tool.sh"] != "100644" {
		t.Fatalf("tracked worktree mode = %q, want filesystem mode 100644", got.Snapshot.Modes["tool.sh"])
	}
}

func TestObserveSelectedWorkingRejectsUnsafeAndAliasedPaths(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	for _, paths := range [][]string{{"../escape"}, {".git/config"}, {".GIT/config"}, {"A.txt", "a.txt"}} {
		if _, err := ObserveSelectedWorking(root, paths); err == nil {
			t.Errorf("accepted unsafe selected paths %#v", paths)
		}
	}
}

func TestObserveSelectedWorkingReportsCaseAliasesMissing(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	writeTestFile(t, root, "docs/readme.md", "lower")
	repo.Git("add", "docs/readme.md")
	repo.Git("commit", "-qm", "lower-case readme")
	// A case-insensitive filesystem opens docs/readme.md for these spellings,
	// but Git tracks only the on-disk spelling.
	for _, alias := range []string{"docs/README.md", "Docs/readme.md"} {
		got, err := ObserveSelectedWorking(root, []string{alias})
		if err != nil {
			t.Fatalf("observe %q: %v", alias, err)
		}
		if len(got.Snapshot.Files) != 0 || !reflect.DeepEqual(got.MissingPaths, []string{alias}) {
			t.Fatalf("%q observed as %v (missing=%v); Git tracks only docs/readme.md", alias, mapKeys(got.Snapshot.Files), got.MissingPaths)
		}
	}
	exact, err := ObserveSelectedWorking(root, []string{"docs/readme.md"})
	if err != nil || len(exact.MissingPaths) != 0 || string(exact.Snapshot.Files["docs/readme.md"]) != "lower" {
		t.Fatalf("exact spelling observation = %#v, %v", exact, err)
	}
}

func TestLoadSelectedDoesNotRequireUnselectedBlobContent(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	repo.Write("selected.txt", "selected bytes")
	repo.Write("unselected.txt", "unselected bytes")
	commit := repo.Commit("selected and unrelated content")
	unselectedOID := repo.Git("rev-parse", "HEAD:unselected.txt")
	objectPath := filepath.Join(root, ".git", "objects", unselectedOID[:2], unselectedOID[2:])
	if err := os.Remove(objectPath); err != nil {
		t.Fatalf("remove unselected loose blob %q: %v", objectPath, err)
	}
	loaded, err := LoadSelected(root, commit, []string{"selected.txt"})
	if err != nil {
		t.Fatalf("selected load depended on unselected blob content: %v", err)
	}
	if len(loaded.Snapshot.Files) != 1 || string(loaded.Snapshot.Files["selected.txt"]) != "selected bytes" {
		t.Fatalf("selected snapshot = %#v", loaded.Snapshot.Files)
	}
}

func TestInventoryWorkingRootsIsMetadataOnlyAndBoundedToExactRoots(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	repo.Write("target/known.txt", "known bytes")
	repo.Write("target/nested/untracked.bin", "untracked bytes")
	repo.Write("other/ignored.txt", "outside exact prefix")
	got, err := InventoryWorkingRoots(root, []string{"target"})
	if err != nil {
		t.Fatal(err)
	}
	want := []WorkingFileMetadata{
		{Path: "target/known.txt", Mode: "100644", Size: int64(len("known bytes"))},
		{Path: "target/nested/untracked.bin", Mode: "100644", Size: int64(len("untracked bytes"))},
	}
	if !reflect.DeepEqual(got.Entries, want) {
		t.Fatalf("inventory entries = %#v, want %#v", got.Entries, want)
	}
	if len(got.MissingPrefixes) != 0 || !strings.HasPrefix(got.MetadataDigest, "sha256:") {
		t.Fatalf("missing/digest = %#v/%q", got.MissingPrefixes, got.MetadataDigest)
	}
	for _, entry := range got.Entries {
		if strings.HasPrefix(entry.Path, "other/") {
			t.Fatalf("out-of-scope path entered inventory: %q", entry.Path)
		}
	}
	digest := got.MetadataDigest
	gotAgain, err := InventoryWorkingRoots(root, []string{"target"})
	if err != nil || gotAgain.MetadataDigest != digest {
		t.Fatalf("repeat inventory digest = %#v, %v", gotAgain, err)
	}
}

func TestInventoryWorkingRootsReportsUnknownUntrackedAndMissingRoots(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	repo.Write("outputs/generated.txt", "generated")
	got, err := InventoryWorkingRoots(repo.Dir, []string{"outputs", "absent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Path != "outputs/generated.txt" || !reflect.DeepEqual(got.MissingPrefixes, []string{"absent"}) {
		t.Fatalf("inventory = %#v", got)
	}
}

func TestInventoryWorkingRootsReportsCaseAliasPrefixMissing(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	writeTestFile(t, root, "docs/readme.md", "lower")
	got, err := InventoryWorkingRoots(root, []string{"Docs"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 0 || !reflect.DeepEqual(got.MissingPrefixes, []string{"Docs"}) {
		t.Fatalf("prefix Docs listed %#v (missing=%v); on-disk directory is docs", got.Entries, got.MissingPrefixes)
	}
	exact, err := InventoryWorkingRoots(root, []string{"docs"})
	if err != nil || len(exact.Entries) != 1 || exact.Entries[0].Path != "docs/readme.md" {
		t.Fatalf("exact prefix inventory = %#v, %v", exact, err)
	}
}

func TestScopedWorkingAPIsReportShortNameAliasesMissing(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("8.3 short names are Windows-only")
	}
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	writeTestFile(t, root, "LongDirectoryName/LongFileName.md", "x")
	if _, err := os.Lstat(filepath.Join(root, "LONGDI~1", "LONGFI~1.MD")); err != nil {
		t.Skipf("8.3 short names unavailable on this volume: %v", err)
	}
	inventory, err := InventoryWorkingRoots(root, []string{"LONGDI~1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Entries) != 0 || !reflect.DeepEqual(inventory.MissingPrefixes, []string{"LONGDI~1"}) {
		t.Fatalf("8.3 prefix LONGDI~1 listed %#v (missing=%v)", inventory.Entries, inventory.MissingPrefixes)
	}
	for _, alias := range []string{"LONGDI~1/LongFileName.md", "LongDirectoryName/LONGFI~1.MD"} {
		observed, err := ObserveSelectedWorking(root, []string{alias})
		if err != nil {
			t.Fatalf("observe %q: %v", alias, err)
		}
		if len(observed.Snapshot.Files) != 0 || !reflect.DeepEqual(observed.MissingPaths, []string{alias}) {
			t.Fatalf("8.3 path %q observed as %v (missing=%v)", alias, mapKeys(observed.Snapshot.Files), observed.MissingPaths)
		}
	}
}

func TestScopedWorkingAPIsSupportLinkedWorktreeIdentity(t *testing.T) {
	repo, head := selectiveGitFixture(t)
	linked := filepath.Join(testkit.TempDir(t), "linked")
	repo.Git("worktree", "add", "--detach", linked, head)
	writeTestFile(t, linked, "observed.txt", "working copy")
	got, err := ObserveSelectedWorking(linked, []string{"observed.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(got.Identity.Root, linked) || got.Identity.GitDir == got.Identity.CommonDir {
		t.Fatalf("linked worktree identity = %#v", got.Identity)
	}
}

func TestInventoryWorkingRootsRejectsOverlapsAndSymlinks(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	if _, err := InventoryWorkingRoots(root, []string{"target", "target/nested"}); err == nil {
		t.Fatal("accepted overlapping inventory prefixes")
	}
	if _, err := InventoryWorkingRoots(root, []string{".GIT"}); err == nil {
		t.Fatal("accepted case-aliased Git metadata prefix")
	}
	target := filepath.Join(root, "target-link")
	if err := os.Symlink(filepath.Join(root, "seed.txt"), target); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := InventoryWorkingRoots(root, []string{"target-link"}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink inventory error = %v", err)
	}
}

func TestWalkScopedMetadataStopsAtEntryBound(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "one.txt"), []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "two.txt"), []byte("2"), 0o600); err != nil {
		t.Fatal(err)
	}
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer rootFS.Close()
	var entries []WorkingFileMetadata
	var total int64
	visited := 0
	err = walkScopedMetadata(rootFS, "target", &entries, &total, &visited, 1)
	if err == nil || !strings.Contains(err.Error(), "entry-count limit of 1") {
		t.Fatalf("walk error = %v, want entry-count bound", err)
	}
	if visited != 2 || len(entries) > 1 {
		t.Fatalf("walk visited %d entries and recorded %d files after crossing bound", visited, len(entries))
	}
}
