package snapshot_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func path(t *testing.T, value string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(value)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func entry(t *testing.T, name string, mode snapshot.Mode, content string) snapshot.Entry {
	t.Helper()
	e, err := snapshot.NewEntry(path(t, name), mode, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func tree(t *testing.T, entries ...snapshot.Entry) snapshot.Tree {
	t.Helper()
	result, err := snapshot.NewTree(entries)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestPathValidationPreservesSpelling(t *testing.T) {
	for _, value := range []string{
		"README.md", "Src/Hello World.go", "模型/Änderung.txt", ".config/é", "a/e\u0301",
		"dir/a:b", "a?b", "a*b", `a"b`, "a<b", "a>b", "a|b", "a.", "a ", "a./b", "a /b",
		"CON", "con.txt", "dir/NUL", "aux.go", "PRN", "COM1", "LPT².txt",
	} {
		if got := path(t, value).String(); got != value {
			t.Fatalf("path changed from %q to %q", value, got)
		}
	}
	for _, value := range []string{
		"", ".", "..", "./a", "a/../b", "a/./b", "/a", "//server/share", "a/", "a//b",
		`a\b`, `C:/a`, `C:relative`, `a:b`, "a\x00b", "a\nb", "a\u007fb", "a\u0085b", "a\xffb",
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := snapshot.NewPath(value); !errors.Is(err, snapshot.ErrInvalidPath) {
				t.Fatalf("NewPath(%q): %v", value, err)
			}
		})
	}
}

func TestEntryModesAndZeroValues(t *testing.T) {
	p := path(t, "file")
	for _, mode := range []snapshot.Mode{0, 3, 255} {
		if _, err := snapshot.NewEntry(p, mode, nil); !errors.Is(err, snapshot.ErrInvalidMode) {
			t.Fatalf("mode %d: %v", mode, err)
		}
	}
	if _, err := snapshot.NewEntry(snapshot.Path{}, snapshot.Regular, nil); !errors.Is(err, snapshot.ErrInvalidPath) {
		t.Fatalf("zero path: %v", err)
	}
	if got, err := snapshot.NewTree([]snapshot.Entry{{}}); !errors.Is(err, snapshot.ErrInvalidEntry) || !got.Equal(snapshot.Tree{}) {
		t.Fatalf("zero entry: %v, %v", got, err)
	}
	empty, err := snapshot.NewEntry(p, snapshot.Regular, nil)
	if err != nil || empty != entry(t, "file", snapshot.Regular, "") {
		t.Fatalf("nil content: %v", err)
	}
	zero := snapshot.Tree{}
	if !zero.Equal(tree(t)) || zero.Digest() != tree(t).Digest() || len(snapshot.Compare(zero, tree(t))) != 0 {
		t.Fatal("zero tree differs from constructed empty tree")
	}
	if _, found := zero.Lookup(snapshot.Path{}); found {
		t.Fatal("zero path found")
	}
}

func TestTreeOrderingAndIdentity(t *testing.T) {
	a := entry(t, "a", snapshot.Regular, "hello")
	b := entry(t, "z/file", snapshot.Executable, "\x00\r\n\xff")
	before := tree(t, b, a)
	after := tree(t, a, b)
	if !before.Equal(after) || before.Digest() != after.Digest() || len(snapshot.Compare(before, after)) != 0 {
		t.Fatal("input order changed fixed tree")
	}
	if !reflect.DeepEqual(before.Entries(), []snapshot.Entry{a, b}) {
		t.Fatal("entries are not sorted")
	}
	// Fixed v1 vector, independently calculated from the documented framing.
	if before.Digest().String() != "10833e1afccf9a165fcfa1aafaee3ff7fe2182bb8e8e9e49e82f007f74393473" {
		t.Fatal("tree identity encoding changed")
	}
	if a.ContentID().String() != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatal("content identity is not SHA-256 of exact bytes")
	}
	base := tree(t, a)
	for _, changed := range []snapshot.Entry{
		entry(t, "A", snapshot.Regular, "hello"),
		entry(t, "a", snapshot.Executable, "hello"),
		entry(t, "a", snapshot.Regular, "hello\n"),
	} {
		candidate := tree(t, changed)
		if base.Equal(candidate) || base.Digest() == candidate.Digest() {
			t.Fatalf("identity ignored path, mode or bytes: %v", changed)
		}
		if changed.Path() != a.Path() || changed.Mode() != a.Mode() {
			if changed.ContentID() != a.ContentID() {
				t.Fatal("content identity included path or mode")
			}
		}
	}
	if _, found := base.Lookup(path(t, "A")); found {
		t.Fatal("lookup folded case")
	}
}

func TestTreeRejectsCollisionsDeterministically(t *testing.T) {
	for _, names := range [][]string{
		{"a", "a"}, {"a", "a/b"}, {"a", "a-else", "a/b"}, {"a/b", "a/b/c"},
	} {
		t.Run(names[0]+":"+names[len(names)-1], func(t *testing.T) {
			var entries []snapshot.Entry
			for _, name := range names {
				entries = append(entries, entry(t, name, snapshot.Regular, name))
			}
			result, first := snapshot.NewTree(entries)
			slices.Reverse(entries)
			_, second := snapshot.NewTree(entries)
			if !errors.Is(first, snapshot.ErrPathCollision) || second == nil || first.Error() != second.Error() {
				t.Fatalf("errors differ or collision accepted: %v / %v", first, second)
			}
			if !result.Equal(snapshot.Tree{}) {
				t.Fatal("failed construction returned a partial tree")
			}
		})
	}
}

func TestGenericTreeKeepsCaseAndUnicodeAliasesDistinct(t *testing.T) {
	// These names can conflict on particular filesystems. The generic producer
	// preserves them; an adapter must refuse incompatible physical materialization.
	names := []string{"A", "a", "Dir/file", "dir/file", "Σ/x", "ς/y", "K/x", "K/y", "é", "e\u0301"}
	var entries []snapshot.Entry
	for _, name := range names {
		entries = append(entries, entry(t, name, snapshot.Regular, name))
	}
	fixed := tree(t, entries...)
	if len(fixed.Entries()) != len(names) {
		t.Fatal("tree collapsed distinct spellings")
	}
	for _, e := range entries {
		if got, found := fixed.Lookup(e.Path()); !found || got != e {
			t.Fatalf("exact path %q changed", e.Path())
		}
	}
}

func TestFixedValuesDoNotExposeAliases(t *testing.T) {
	input := []byte("original")
	e, err := snapshot.NewEntry(path(t, "a"), snapshot.Regular, input)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []snapshot.Entry{e}
	fixed, err := snapshot.NewTree(inputs)
	if err != nil {
		t.Fatal(err)
	}
	id := fixed.Digest()
	input[0] = 'X'
	inputs[0] = entry(t, "b", snapshot.Executable, "replacement")
	output := fixed.Entries()
	output[0] = inputs[0]
	bytes := e.Bytes()
	bytes[0] = 'Y'
	found, ok := fixed.Lookup(e.Path())
	if !ok {
		t.Fatal("lost entry")
	}
	found.Bytes()[0] = 'Z'
	changes := snapshot.Compare(snapshot.Tree{}, fixed)
	next, ok := changes[0].After()
	if !ok {
		t.Fatal("addition has no content")
	}
	next.Bytes()[0] = 'Q'
	changes[0] = snapshot.Change{}
	if fixed.Digest() != id || string(e.Bytes()) != "original" || !fixed.Equal(tree(t, entry(t, "a", snapshot.Regular, "original"))) {
		t.Fatal("mutation escaped into a fixed value")
	}
}

func TestCompareCarriesExactOldAndNewContent(t *testing.T) {
	a := entry(t, "a", snapshot.Regular, "deleted")
	b := entry(t, "b", snapshot.Regular, "old")
	c := entry(t, "c", snapshot.Regular, "unchanged bytes")
	d := entry(t, "d", snapshot.Regular, "added")
	unchanged := entry(t, "e", snapshot.Regular, "same")
	bNext := entry(t, "b", snapshot.Regular, "new")
	cNext := entry(t, "c", snapshot.Executable, "unchanged bytes")
	before := tree(t, c, unchanged, a, b)
	after := tree(t, unchanged, d, cNext, bNext)
	changes := snapshot.Compare(before, after)
	if len(changes) != 4 {
		t.Fatalf("changes: %v", changes)
	}
	for i, want := range []struct {
		kind            snapshot.ChangeKind
		old, next       snapshot.Entry
		hasOld, hasNext bool
	}{
		{snapshot.Delete, a, snapshot.Entry{}, true, false},
		{snapshot.Modify, b, bNext, true, true},
		{snapshot.Modify, c, cNext, true, true},
		{snapshot.Add, snapshot.Entry{}, d, false, true},
	} {
		got := changes[i]
		old, hasOld := got.Before()
		next, hasNext := got.After()
		if got.Kind() != want.kind || old != want.old || next != want.next || hasOld != want.hasOld || hasNext != want.hasNext {
			t.Fatalf("change %d: %v", i, got)
		}
		expectedPath := want.old.Path()
		if !want.hasOld {
			expectedPath = want.next.Path()
		}
		if got.Path() != expectedPath {
			t.Fatalf("change %d path: %s", i, got.Path())
		}
	}
	if len(snapshot.Compare(before, snapshot.Tree{})) != 4 || len(snapshot.Compare(snapshot.Tree{}, after)) != 4 {
		t.Fatal("comparison with empty tree lost entries")
	}
	rename := snapshot.Compare(tree(t, a), tree(t, entry(t, "A", snapshot.Regular, "deleted")))
	if len(rename) != 2 || rename[0].Kind() != snapshot.Add || rename[1].Kind() != snapshot.Delete {
		t.Fatal("case-only rename lost exact spelling")
	}
}
