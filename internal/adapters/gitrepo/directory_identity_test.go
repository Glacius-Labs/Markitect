package gitrepo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestRealDirectoryRetainsObservedIdentityAfterPathReplacement(t *testing.T) {
	parent := testkit.TempDir(t)
	selected := filepath.Join(parent, "selected")
	replacement := filepath.Join(parent, "replacement")
	for _, dir := range []string{selected, replacement} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	before, err := realDirectory(selected)
	if err != nil {
		t.Fatal(err)
	}
	// Do not call SameFile before the replacement: identity must already be
	// captured by realDirectory, rather than loaded later from its old path.
	if err := os.Rename(selected, selected+".original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, selected); err != nil {
		t.Fatal(err)
	}
	after, err := realDirectory(selected)
	if err != nil {
		t.Fatal(err)
	}
	original, err := realDirectory(selected + ".original")
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("replacement at the same path retained the observed directory identity")
	}
	if !os.SameFile(before, original) {
		t.Fatal("renaming the original directory lost its observed identity")
	}
}
