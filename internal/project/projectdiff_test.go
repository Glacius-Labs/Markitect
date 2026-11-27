package project

import (
	"fmt"
	"strings"
	"testing"
)

// gone, in the changes of with, removes a file.
const gone = "\x00gone"

// shop is a project of three modules: shop, with cart below it, and users, with
// the files that build them, and two files no module names.
func shop() map[string]string {
	return map[string]string{
		".markitect/model":               "docs/model\n",
		"docs/model/shop/README.md":      "# Shop\n\nThe shop.\n\n## Realized in\n- `src/shop/`\n- `cmd/shop.go`\n\n## Contents\n- `*.md`\n",
		"docs/model/shop/rules.md":       "# Rules\n\nOne rule.\n",
		"docs/model/shop/cart/README.md": "# Cart\n\nThe cart.\n\n## Realized in\n- `src/shop/cart/*.go`\n",
		"docs/model/shop/cart/add.md":    "# Add\n",
		"docs/model/users/README.md":     "# Users\n\nThe users.\n\n## Realized in\n- `src/users/*.go`\n",
		"docs/model/users/user.md":       "# User\n",
		"src/shop/a.go":                  "package shop\n",
		"src/shop/cart/c.go":             "package cart\n",
		"src/shop/cart/c_test.go":        "package cart\n",
		"src/users/u.go":                 "package users\n",
		"cmd/shop.go":                    "package main\n",
		"tools/build.sh":                 "#!/bin/sh\n",
		"LICENSE":                        "MIT\n",
	}
}

func with(files map[string]string, change map[string]string) map[string]string {
	out := map[string]string{}
	for p, s := range files {
		out[p] = s
	}
	for p, s := range change {
		if s == gone {
			delete(out, p)
		} else {
			out[p] = s
		}
	}
	return out
}

func load(t *testing.T, files map[string]string) *Project {
	t.Helper()
	p, err := Load(tree(files))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func mustDiff(t *testing.T, older, newer map[string]string, ig *Ignore) *ProjectDiff {
	t.Helper()
	d, err := DiffProjects(load(t, older), load(t, newer), ig, Folder{}, Folder{})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// diffOf compares two projects with an ignore file and renders the result, one
// line per module and file, so that a case reads at a glance.
func diffOf(t *testing.T, older, newer map[string]string, ignore string) string {
	t.Helper()
	ig, err := ParseIgnore([]byte(ignore))
	if err != nil {
		t.Fatal(err)
	}
	return render(mustDiff(t, older, newer, ig))
}

func render(d *ProjectDiff) string {
	var b strings.Builder
	for _, m := range d.Modules {
		fmt.Fprintf(&b, "%s model=%v files=%v\n", m.Module, m.Model, m.Files)
		for _, f := range m.Changes() {
			fmt.Fprintf(&b, "  %s %s\n", f.Kind, f.Path)
		}
	}
	b.WriteString("unassigned:")
	for _, f := range d.Unassigned {
		fmt.Fprintf(&b, " %s %s;", f.Kind, f.Path)
	}
	fmt.Fprintf(&b, "\ntotal %d differs %v\n", d.UnassignedTotal, d.Differs())
	return b.String()
}

func TestDiffProjectsNothingChanged(t *testing.T) {
	// Two files are in no module: tools/build.sh and LICENSE.
	want := "unassigned:\ntotal 2 differs false\n"
	if got := diffOf(t, shop(), shop(), ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

func TestDiffProjectsModelOnly(t *testing.T) {
	newer := with(shop(), map[string]string{"docs/model/shop/rules.md": "# Rules\n\nTwo rules.\n"})
	want := "shop model=true files=false\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
	// A document of the module below is that module's, not the module above.
	newer = with(shop(), map[string]string{"docs/model/shop/cart/add.md": "# Add\n\nMore.\n", "docs/model/users/more.md": "# More\n"})
	want = "shop/cart model=true files=false\nusers model=true files=false\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

func TestDiffProjectsFilesOnly(t *testing.T) {
	newer := with(shop(), map[string]string{"src/users/u.go": "package users // changed\n"})
	want := "users model=false files=true\n  changed src/users/u.go\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

func TestDiffProjectsModelAndFiles(t *testing.T) {
	newer := with(shop(), map[string]string{
		"docs/model/users/user.md": "# User\n\nA person.\n",
		"src/users/u.go":           "package users // changed\n",
		"src/users/v.go":           "package users\n",
	})
	want := "users model=true files=true\n  changed src/users/u.go\n  new src/users/v.go\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

func TestDiffProjectsFileNewGoneChanged(t *testing.T) {
	newer := with(shop(), map[string]string{
		"src/shop/a.go":         "package shop // changed\n",
		"src/shop/b.go":         "package shop\n",
		"src/shop/sub/deep.go":  "package sub\n",
		"cmd/shop.go":           gone,
		"src/shop/cart/c.go":    gone,
		"src/users/u.go":        "package users\n", // equal
		"src/shop/cart/new.txt": "not a .go file, but below src/shop/, which the shop names",
		"tools/new.sh":          "new\n",
	})
	// The files of a module are in path order, whatever their kind.
	want := "shop model=false files=true\n" +
		"  gone cmd/shop.go\n  changed src/shop/a.go\n  new src/shop/b.go\n  new src/shop/cart/new.txt\n  new src/shop/sub/deep.go\n" +
		"shop/cart model=false files=true\n  gone src/shop/cart/c.go\n" +
		"unassigned: new tools/new.sh;\ntotal 3 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// A file that a module and a module below it both name counts for the lower
// one only.
func TestDiffProjectsFileNamedByParentAndChild(t *testing.T) {
	newer := with(shop(), map[string]string{"src/shop/cart/c.go": "package cart // changed\n"})
	want := "shop/cart model=false files=true\n  changed src/shop/cart/c.go\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
	// With the README of the child without a pattern, the file is the parent's again.
	older := with(shop(), map[string]string{"docs/model/shop/cart/README.md": "# Cart\n\nThe cart.\n"})
	newer = with(older, map[string]string{"src/shop/cart/c.go": "package cart // changed\n"})
	want = "shop model=false files=true\n  changed src/shop/cart/c.go\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, older, newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

// A file that two unrelated modules name counts for both.
func TestDiffProjectsFileNamedByUnrelatedModules(t *testing.T) {
	older := with(shop(), map[string]string{
		"docs/model/users/README.md": "# Users\n\nThe users.\n\n## Realized in\n- `src/users/*.go`\n- `cmd/shop.go`\n",
	})
	newer := with(older, map[string]string{"cmd/shop.go": "package main // changed\n"})
	want := "shop model=false files=true\n  changed cmd/shop.go\nusers model=false files=true\n  changed cmd/shop.go\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, older, newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

// The module of the model folder lies above every other module, and comes first.
func TestDiffProjectsModelFolderModule(t *testing.T) {
	older := with(shop(), map[string]string{
		"docs/model/README.md": "# Project\n\nAll of it.\n\n## Realized in\n- `src/`\n- `LICENSE`\n",
	})
	newer := with(older, map[string]string{
		"src/users/u.go":       "package users // changed\n",
		"LICENSE":              "BSD\n",
		"docs/model/README.md": "# Project\n\nAll of it, again.\n\n## Realized in\n- `src/`\n- `LICENSE`\n",
	})
	want := ". model=true files=true\n  changed LICENSE\nusers model=false files=true\n  changed src/users/u.go\nunassigned:\ntotal 1 differs true\n"
	if got := diffOf(t, older, newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

func TestDiffProjectsModuleOnlyInOneState(t *testing.T) {
	newer := with(shop(), map[string]string{
		"docs/model/orders/README.md": "# Orders\n\nThe orders.\n\n## Realized in\n- `src/orders/`\n",
		"src/orders/o.go":             "package orders\n",
	})
	// The module is only in NEW: its documents are new, and its file.
	want := "orders model=true files=true\n  new src/orders/o.go\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("new module\n%s", got)
	}
	// The module is only in OLD: its documents are gone, and its file.
	want = "orders model=true files=true\n  gone src/orders/o.go\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, newer, shop(), ""); got != want {
		t.Errorf("gone module\n%s", got)
	}
	// The module is gone, but its file stays as it was, now in no module: the file
	// did not change, so only the model of the module did.
	older := newer
	newer = with(shop(), map[string]string{"src/orders/o.go": "package orders\n"})
	want = "orders model=true files=false\nunassigned:\ntotal 3 differs true\n"
	if got := diffOf(t, older, newer, ""); got != want {
		t.Errorf("module gone, file stays\n%s", got)
	}
}

// Only the Realized in of two READMEs changes between the states, the files are
// the same: a file is new, gone or changed by itself, so the modules show their
// model as changed, the README being a document of theirs, and no file.
func TestDiffProjectsRealizedInChangedFilesIdentical(t *testing.T) {
	newer := with(shop(), map[string]string{
		"docs/model/users/README.md": "# Users\n\nThe users.\n\n## Realized in\n- `src/users/*.go`\n- `tools/`\n",
		"docs/model/shop/README.md":  "# Shop\n\nThe shop.\n\n## Realized in\n- `src/shop/`\n\n## Contents\n- `*.md`\n",
	})
	want := "shop model=true files=false\nusers model=true files=false\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

// A new or changed file goes to the modules that name it in NEW, a gone file to
// those that named it in OLD, whatever the READMEs say in the other state.
func TestDiffProjectsFilesFollowTheStateTheyBelongTo(t *testing.T) {
	newer := with(shop(), map[string]string{
		"docs/model/users/README.md": "# Users\n\nThe users.\n\n## Realized in\n- `src/users/*.go`\n- `tools/`\n",
		"docs/model/shop/README.md":  "# Shop\n\nThe shop.\n\n## Realized in\n- `src/shop/`\n\n## Contents\n- `*.md`\n",
		"tools/build.sh":             "#!/bin/sh\n# changed\n", // changed: users names it in NEW
		"cmd/shop.go":                gone,                     // gone: shop named it in OLD
		"tools/new.sh":               "new\n",                  // new: users names it in NEW
	})
	want := "shop model=true files=true\n  gone cmd/shop.go\nusers model=true files=true\n  changed tools/build.sh\n  new tools/new.sh\nunassigned:\ntotal 1 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
	// A changed file that was named in OLD only, and by no module in NEW, is one no
	// module names, in the state it belongs to.
	newer = with(shop(), map[string]string{
		"docs/model/shop/README.md": "# Shop\n\nThe shop.\n\n## Realized in\n- `src/shop/`\n\n## Contents\n- `*.md`\n",
		"cmd/shop.go":               "package main // changed\n",
	})
	want = "shop model=true files=false\nunassigned: changed cmd/shop.go;\ntotal 3 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

// A changed file that no module names is listed, an unchanged one is only counted.
func TestDiffProjectsFilesNoModuleNames(t *testing.T) {
	newer := with(shop(), map[string]string{
		"LICENSE":        "BSD\n",
		"tools/new.sh":   "new\n",
		"tools/build.sh": gone,
		"Makefile":       "all:\n",
	})
	want := "unassigned: changed LICENSE; new Makefile; gone tools/build.sh; new tools/new.sh;\ntotal 3 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

// A model without any Realized in: no module has files, so every changed file
// that is not left out shows under the files no module names.
func TestDiffProjectsWithoutRealizedIn(t *testing.T) {
	plain := map[string]string{
		".markitect/model":     "model\n",
		"model/README.md":      "# Model\n",
		"model/img/diagram.md": "# Diagram\n",
		"src/a.go":             "a",
		"src/b.go":             "b",
		"src/gone.go":          "g",
		"build/out.bin":        "o",
		".hidden/x":            "x",
	}
	newer := with(plain, map[string]string{"src/a.go": "A", "src/c.go": "c", "src/gone.go": gone, "build/out.bin": "O", ".hidden/x": "X", "model/img/diagram.md": "# Diagram\n\nChanged.\n"})
	// The one changed document makes its module a change; its files are none.
	want := "img model=true files=false\n" +
		"unassigned: changed .hidden/x; changed build/out.bin; changed src/a.go; new src/c.go; gone src/gone.go;\ntotal 5 differs true\n"
	if got := render(mustDiff(t, plain, newer, nil)); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	// The ignore file leaves out what it names.
	ig, err := ParseIgnore([]byte("build/\n*.go\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := render(mustDiff(t, plain, newer, ig)), "img model=true files=false\nunassigned: changed .hidden/x;\ntotal 1 differs true\n"; got != want {
		t.Errorf("with ignore\n%s", got)
	}
}

// Binary files, empty files and a file that differs only in its line ends.
func TestDiffProjectsBytesAreCompared(t *testing.T) {
	older := with(shop(), map[string]string{
		"src/users/bin.dat":   "\x00\x01\x02\xff",
		"src/users/empty.go":  "",
		"src/users/lf.go":     "a\nb\n",
		"src/users/bom.go":    "\xef\xbb\xbfpackage users\n",
		"src/users/same.go":   "same\n",
		"src/users/grows.go":  "",
		"src/users/shrink.go": "x",
	})
	newer := with(older, map[string]string{
		"src/users/bin.dat":   "\x00\x01\x03\xff",
		"src/users/lf.go":     "a\r\nb\r\n",
		"src/users/bom.go":    "package users\n",
		"src/users/grows.go":  "x",
		"src/users/shrink.go": "",
	})
	// bin.dat is no .go file, so no module names it. A line end, a byte order
	// mark, and an empty file against one of a byte all count.
	want := "users model=false files=true\n  changed src/users/bom.go\n  changed src/users/grows.go\n  changed src/users/lf.go\n  changed src/users/shrink.go\n" +
		"unassigned: changed src/users/bin.dat;\ntotal 3 differs true\n"
	if got := diffOf(t, older, newer, ""); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// Large files differ at their last byte, and are compared in chunks.
func TestDiffProjectsLargeFiles(t *testing.T) {
	big := strings.Repeat("x", 3<<20)
	older := with(shop(), map[string]string{"src/users/big.go": big, "src/users/big2.go": big})
	newer := with(older, map[string]string{"src/users/big2.go": big[:len(big)-1] + "y"})
	want := "users model=false files=true\n  changed src/users/big2.go\nunassigned:\ntotal 2 differs true\n"
	if got := diffOf(t, older, newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

func TestDiffProjectsIgnoreAppliesToBothSides(t *testing.T) {
	older := with(shop(), map[string]string{"src/users/gen/g.go": "old", "tools/cache/old.txt": "o"})
	newer := with(shop(), map[string]string{"src/users/gen/g.go": "new", "tools/cache/new.txt": "n", "tools/real.sh": "r"})
	if got := diffOf(t, older, newer, "gen/\ncache\n"); got != "unassigned: new tools/real.sh;\ntotal 3 differs true\n" {
		t.Errorf("got\n%s", got)
	}
	// Without it the differences show, on both sides.
	want := "unassigned: changed src/users/gen/g.go; new tools/cache/new.txt; gone tools/cache/old.txt; new tools/real.sh;\ntotal 5 differs true\n"
	if got := diffOf(t, older, newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

// Files of the model folder and the folder .markitect/ are no files of the project;
// a name that starts with a dot is a name like any other.
func TestDiffProjectsLeavesOutTheModelFolderAndMarkitect(t *testing.T) {
	newer := with(shop(), map[string]string{
		"docs/model/shop/image.png": "png",
		".github/ci.yml":            "ci",
		"src/shop/.secret":          "s",
		".markitect/ignore":         "# nothing\n",
	})
	want := "shop model=false files=true\n  new src/shop/.secret\nunassigned: new .github/ci.yml;\ntotal 3 differs true\n"
	if got := diffOf(t, shop(), newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

func TestDiffProjectsUnreadableFile(t *testing.T) {
	older := tree(shop())
	// Same size, so that the bytes must be read.
	newer := tree(with(shop(), map[string]string{"src/users/u.go": "package USERS\n"}))
	o, n := load(t, shop()), load(t, shop())
	o.FS, n.FS = failing{FS: older, open: "src/users/u.go"}, newer
	if _, err := DiffProjects(o, n, nil, Folder{}, Folder{}); err == nil || err.Error() != "cannot read src/users/u.go in the old project: permission denied" {
		t.Errorf("got %v", err)
	}
	o.FS, n.FS = older, failing{FS: newer, read: "src/users/u.go"}
	if _, err := DiffProjects(o, n, nil, Folder{}, Folder{}); err == nil || err.Error() != "cannot read src/users/u.go in the new project: disk error" {
		t.Errorf("got %v", err)
	}
	// A file that only one side has is never opened.
	n.FS = failing{FS: tree(with(shop(), map[string]string{"tools/new.sh": "x"})), open: "tools/new.sh"}
	if d, err := DiffProjects(o, n, nil, Folder{}, Folder{}); err != nil || len(d.Unassigned) != 1 {
		t.Errorf("got %v %v", d, err)
	}
}

// The output does not depend on the order of the file system.
func TestDiffProjectsIsDeterministic(t *testing.T) {
	newer := with(shop(), map[string]string{"src/shop/b.go": "b", "src/shop/a.go": "a", "tools/z": "z", "tools/a": "a", "src/users/q.go": "q", "docs/model/users/q.md": "# Q\n"})
	first := diffOf(t, shop(), newer, "")
	for i := 0; i < 20; i++ {
		if got := diffOf(t, shop(), newer, ""); got != first {
			t.Fatalf("run %d differs:\n%s\n%s", i, got, first)
		}
	}
}

// prefixed is the files of a project kept in a folder inside another project.
func prefixed(files map[string]string, folder string) map[string]string {
	out := map[string]string{}
	for p, s := range files {
		out[folder+"/"+p] = s
	}
	return out
}

// A copy kept inside a project is left out of that project's files, in either direction.
func TestDiffProjectsLeavesOutAProjectInside(t *testing.T) {
	outer := with(shop(), prefixed(shop(), "backup/old"))
	// OLD lies inside NEW.
	d, err := DiffProjects(load(t, shop()), load(t, outer), nil, Folder{Path: "backup/old"}, Folder{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := render(d), "unassigned:\ntotal 2 differs false\n"; got != want {
		t.Errorf("OLD inside NEW\n%s", got)
	}
	// Without it the copy shows as new files.
	d, err = DiffProjects(load(t, shop()), load(t, outer), nil, Folder{}, Folder{})
	if err != nil {
		t.Fatal(err)
	}
	if !d.Differs() || len(d.Modules) != 0 || len(d.Unassigned) == 0 {
		t.Errorf("OLD inside NEW, not left out\n%s", render(d))
	}
	// NEW lies inside OLD; the files of OLD do not count the copy.
	d, err = DiffProjects(load(t, outer), load(t, shop()), nil, Folder{}, Folder{Path: "backup/old"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := render(d), "unassigned:\ntotal 2 differs false\n"; got != want {
		t.Errorf("NEW inside OLD\n%s", got)
	}
	// What else changed in the outer project still shows.
	outer = with(outer, map[string]string{"src/users/u.go": "package users // changed\n"})
	d, err = DiffProjects(load(t, shop()), load(t, outer), nil, Folder{Path: "backup/old"}, Folder{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := render(d), "users model=false files=true\n  changed src/users/u.go\nunassigned:\ntotal 2 differs true\n"; got != want {
		t.Errorf("a change beside the copy\n%s", got)
	}
}
