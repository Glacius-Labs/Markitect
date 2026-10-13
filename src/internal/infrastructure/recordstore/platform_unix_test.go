//go:build !windows

package recordstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateModesReplaceAndBroadExistingRefusal(t *testing.T) {
	s, root := openFixture(t)
	n := fixtureName()
	if err := s.Create(n, []byte("old")); err != nil {
		t.Fatal(err)
	}
	rel, _ := n.Path()
	p := filepath.Join(root, filepath.FromSlash(rel))
	old, err := s.Read(n, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Replace(n, old, []byte("new"), nil); err != nil {
		t.Fatal(err)
	}
	i, err := os.Stat(p)
	if err != nil || i.Mode().Perm() != 0600 {
		t.Fatalf("replacement mode: %v %v", i, err)
	}
	for d := filepath.Dir(p); d != filepath.Join(root, ".markitect"); d = filepath.Dir(d) {
		i, err := os.Stat(d)
		if err != nil || i.Mode().Perm() != 0700 {
			t.Fatalf("private ancestor: %s %v %v", d, i, err)
		}
	}
	if err := os.Chmod(p, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(n, 0); err == nil {
		t.Fatal("broad existing record accepted")
	}
	i, _ = os.Stat(p)
	if i.Mode().Perm() != 0644 {
		t.Fatal("record permissions silently repaired")
	}
}
