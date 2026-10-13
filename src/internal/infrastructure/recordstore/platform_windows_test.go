//go:build windows

package recordstore

import (
	"os"
	"testing"
)

func TestNativeRelativeDirectoryCreation(t *testing.T) {
	s, root := openFixture(t)
	f, err := s.root.Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := platformWritableParent(f)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := platformMkdir(s.root, p, "child"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("actual entries: %v", entries)
	if _, err := s.root.Lstat("child"); err != nil {
		t.Fatal(err)
	}
}
