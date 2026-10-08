package source

import (
	"reflect"
	"strings"
	"testing"
)

func TestRevisionInventorySelectsMetadataAtExactCommit(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "src/orders.py", "original")
	writeTestFile(t, root, "src/nested/stock.py", "stock")
	writeTestFile(t, root, "outside.txt", "unselected")
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-qm", "inventory")
	commit := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	writeTestFile(t, root, "src/orders.py", "changed working bytes")
	writeTestFile(t, root, "src/untracked.py", "untracked")
	queries := 0
	run := func(repo string, args ...string) ([]byte, error) {
		for _, arg := range args {
			if arg == "cat-file" {
				t.Fatal("metadata inventory requested content")
			}
		}
		if len(args) > 1 && args[1] == "ls-tree" {
			queries++
			if !reflect.DeepEqual(args[len(args)-2:], []string{"missing", "src"}) {
				t.Fatalf("unselected pathspecs: %v", args)
			}
		}
		return selectiveGitOutput(repo, args...)
	}
	got, err := inventoryRevisionRoots(root, commit, []string{"src/", "missing"}, run)
	if err != nil {
		t.Fatal(err)
	}
	want := []WorkingFileMetadata{{Path: "src/nested/stock.py", Mode: "100644", Size: 5}, {Path: "src/orders.py", Mode: "100644", Size: 8}}
	if !reflect.DeepEqual(got.Entries, want) || !reflect.DeepEqual(got.MissingPrefixes, []string{"missing"}) || queries != 1 {
		t.Fatalf("inventory=%+v queries=%d", got, queries)
	}
	if got.Revision != commit || !strings.HasPrefix(got.MetadataDigest, "sha256:") {
		t.Fatalf("bindings=%+v", got)
	}
	working, err := InventoryWorkingRoots(root, []string{"src"})
	if err != nil || len(working.Entries) != 3 {
		t.Fatalf("working inventory=%+v err=%v", working, err)
	}
}

func TestRevisionInventoryRejectsUnfixedOrUnsafeScope(t *testing.T) {
	root, commit := selectiveGitFixture(t)
	for _, revision := range []string{"HEAD", commit[:12], "-bad", strings.ToUpper(commit)} {
		if _, err := InventoryRevisionRoots(root, revision, []string{"src"}); err == nil {
			t.Errorf("accepted revision %q", revision)
		}
	}
	for _, roots := range [][]string{{"src", "src/nested"}, {"../outside"}, {".git"}, {"src/*"}, {"Src", "src"}} {
		if _, err := InventoryRevisionRoots(root, commit, roots); err == nil {
			t.Errorf("accepted roots %v", roots)
		}
	}
}

func TestRevisionInventoryRejectsSymlinkMetadataWithoutOpeningContent(t *testing.T) {
	root, _ := selectiveGitFixture(t)
	writeTestFile(t, root, "target.txt", "outside")
	gitTest(t, root, "add", "target.txt")
	gitTest(t, root, "commit", "-qm", "target")
	oid := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD:target.txt"))
	gitTest(t, root, "update-index", "--add", "--cacheinfo", "120000,"+oid+",selected/link")
	gitTest(t, root, "commit", "-qm", "symlink metadata")
	commit := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	if _, err := InventoryRevisionRoots(root, commit, []string{"selected"}); err == nil {
		t.Fatal("accepted symlink")
	}
}
