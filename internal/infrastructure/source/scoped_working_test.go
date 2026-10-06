package source

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestObserveSelectedWorkingReadsExactPathsAndReportsMissing(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "selected/data.txt", "selected bytes")
	writeTestFile(t, root, "selected/extra.txt", "unselected bytes")
	writeTestFile(t, root, "outside.txt", "outside bytes")
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

func TestObserveSelectedWorkingRejectsUnsafeAndAliasedPaths(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	for _, paths := range [][]string{{"../escape"}, {".git/config"}, {"A.txt", "a.txt"}} {
		if _, err := ObserveSelectedWorking(root, paths); err == nil {
			t.Errorf("accepted unsafe selected paths %#v", paths)
		}
	}
}

func TestLoadSelectedDoesNotRequireUnselectedBlobContent(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "selected.txt", "selected bytes")
	writeTestFile(t, root, "unselected.txt", "unselected bytes")
	gitTest(t, root, "add", "selected.txt", "unselected.txt")
	gitTest(t, root, "commit", "-qm", "selected and unrelated content")
	commit := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	unselectedOID := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD:unselected.txt"))
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
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "target/known.txt", "known bytes")
	writeTestFile(t, root, "target/nested/untracked.bin", "untracked bytes")
	writeTestFile(t, root, "other/ignored.txt", "outside exact prefix")
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
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "outputs/generated.txt", "generated")
	got, err := InventoryWorkingRoots(root, []string{"outputs", "absent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Path != "outputs/generated.txt" || !reflect.DeepEqual(got.MissingPrefixes, []string{"absent"}) {
		t.Fatalf("inventory = %#v", got)
	}
}

func TestScopedWorkingAPIsSupportLinkedWorktreeIdentity(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "--detach", linked, strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD")))
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
	root, _ := selectiveGitFixture(t)
	if _, err := InventoryWorkingRoots(root, []string{"target", "target/nested"}); err == nil {
		t.Fatal("accepted overlapping inventory prefixes")
	}
	target := filepath.Join(root, "target-link")
	if err := os.Symlink(filepath.Join(root, "seed.txt"), target); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := InventoryWorkingRoots(root, []string{"target-link"}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink inventory error = %v", err)
	}
}
