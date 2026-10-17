package source

import (
	"reflect"
	"strings"
	"testing"
)

func TestRevisionInventorySelectsMetadataAtExactCommit(t *testing.T) {
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	repo.Write("src/orders.py", "original")
	repo.Write("src/nested/stock.py", "stock")
	repo.Write("outside.txt", "unselected")
	commit := repo.Commit("inventory")
	repo.Write("src/orders.py", "changed working bytes")
	repo.Write("src/untracked.py", "untracked")
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
	repo, commit := selectiveGitFixture(t)
	root := repo.Dir
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
	repo, _ := selectiveGitFixture(t)
	root := repo.Dir
	repo.Write("target.txt", "outside")
	repo.Commit("target")
	oid := repo.Git("rev-parse", "HEAD:target.txt")
	repo.Git("update-index", "--add", "--cacheinfo", "120000,"+oid+",selected/link")
	commit := commitIndex(repo, "symlink metadata")
	if _, err := InventoryRevisionRoots(root, commit, []string{"selected"}); err == nil {
		t.Fatal("accepted symlink")
	}
}
