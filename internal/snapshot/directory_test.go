package snapshot_test

import (
	"errors"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func directory(t *testing.T, name string) snapshot.DirectoryBoundary {
	t.Helper()
	d, err := snapshot.DirectoryAt(path(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDirectoryBoundaryRequiresExplicitRoot(t *testing.T) {
	root := snapshot.RootDirectory()
	var zero snapshot.DirectoryBoundary
	if !root.IsRoot() || root.String() != "." || zero.IsRoot() || zero.String() != "" {
		t.Fatal("root and invalid boundary are not distinct")
	}
	if got, err := snapshot.DirectoryAt(snapshot.Path{}); !errors.Is(err, snapshot.ErrInvalidPath) || got != zero {
		t.Fatalf("zero Path became a directory: %+v %v", got, err)
	}
	if _, err := snapshot.NewPath("."); !errors.Is(err, snapshot.ErrInvalidPath) {
		t.Fatal("directory root weakened the file Path contract")
	}
	for _, boundary := range []snapshot.DirectoryBoundary{root, zero, directory(t, "real")} {
		if boundary.Contains(snapshot.Path{}) || boundary.ContainsDescendant(snapshot.Path{}) || boundary.ContainsDirectory(zero) {
			t.Fatal("invalid path or boundary matched an area")
		}
	}
	if zero.Contains(path(t, "file")) || zero.ContainsDirectory(root) || zero.ContainsDirectory(directory(t, "real")) {
		t.Fatal("an invalid boundary contained a valid value")
	}
}

func TestDirectoryBoundaryDistinguishesProtectionAndFileCreation(t *testing.T) {
	area := directory(t, "real")
	for _, tt := range []struct {
		name       string
		inclusive  bool
		descendant bool
	}{
		{"real", true, false},
		{"real/file.go", true, true},
		{"real/sub/file.go", true, true},
		{"real-other/file.go", false, false},
		{"really/file.go", false, false},
		{"Real/file.go", false, false},
		{"outside/real/file.go", false, false},
		{".markitect/state/control", false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := path(t, tt.name)
			if area.Contains(p) != tt.inclusive || area.ContainsDescendant(p) != tt.descendant {
				t.Fatal("inclusive and strict directory relationships differ")
			}
			if !snapshot.RootDirectory().Contains(p) || !snapshot.RootDirectory().ContainsDescendant(p) {
				t.Fatal("explicit repository area lost a valid relative path")
			}
		})
	}
	state := directory(t, ".markitect/state")
	if !state.Contains(path(t, ".markitect/state")) || !state.Contains(path(t, ".markitect/state/control")) || state.Contains(path(t, ".markitect/statex/control")) {
		t.Fatal("a protected subtree did not retain its exact boundary")
	}
}

func TestDirectoryBoundaryCeilingsAndExactSpelling(t *testing.T) {
	root, area, child, sibling := snapshot.RootDirectory(), directory(t, "real"), directory(t, "real/sub"), directory(t, "real-other")
	if !root.ContainsDirectory(root) || !root.ContainsDirectory(area) || !area.ContainsDirectory(area) || !area.ContainsDirectory(child) {
		t.Fatal("an equal or narrower directory was excluded")
	}
	if area.ContainsDirectory(root) || child.ContainsDirectory(area) || area.ContainsDirectory(sibling) {
		t.Fatal("a wider or unrelated directory crossed a ceiling")
	}
	for _, name := range []string{"模型/Änderung", "a/e\u0301", "a/é", "a./b", "dir/a:b", "CON"} {
		got := directory(t, name)
		if got.String() != name || got.IsRoot() || !got.ContainsDirectory(directory(t, name+"/child")) {
			t.Fatalf("directory spelling or relationship changed: %q", name)
		}
	}
	composed, decomposed := directory(t, "a/é"), directory(t, "a/e\u0301")
	if composed.ContainsDirectory(decomposed) || decomposed.ContainsDirectory(composed) {
		t.Fatal("normalization-distinct boundaries were folded together")
	}
}
