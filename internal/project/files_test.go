package project

import (
	"errors"
	"io/fs"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

func tree(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for p, text := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(text)}
	}
	return fsys
}

func names(t fileTable) string {
	var out []string
	for p := range t {
		out = append(out, p)
	}
	sort.Strings(out)
	return strings.Join(out, " ")
}

func TestParseIgnoreErrors(t *testing.T) {
	for text, want := range map[string]string{
		"a\n/b\n":        `.markitect/ignore:2: bad pattern "/b": a pattern is relative to the project folder, not absolute`,
		"# c\n\na/../b":  `.markitect/ignore:3: bad pattern "a/../b": a pattern stays below the project folder, without ..`,
		"x[\n":           `.markitect/ignore:1: bad pattern "x[": a pattern uses the wildcards *, ? and [a-z]`,
		"a\\b\n":         `.markitect/ignore:1: bad pattern "a\b": a pattern separates folders with /, not \`,
		"a//b\r\n":       `.markitect/ignore:1: bad pattern "a//b": a pattern has no empty folder name and no .`,
		"/\n":            `.markitect/ignore:1: bad pattern "/": a pattern is relative to the project folder, not absolute`,
		"C:/x\n":         `.markitect/ignore:1: bad pattern "C:/x": a pattern is relative to the project folder and has no drive`,
		"a\n  \t\n./b\n": `.markitect/ignore:3: bad pattern "./b": a pattern has no empty folder name and no .`,
	} {
		if _, err := ParseIgnore([]byte(text)); err == nil || err.Error() != want {
			t.Errorf("%q: got %v, want %s", text, err, want)
		}
	}
}

func TestIgnoreRules(t *testing.T) {
	const ignore = "\xef\xbb\xbf# build output\r\n\r\nnode_modules\r\nbuild/   \r\n*.log\t\r\ndocs/generated\r\ngen/*/\r\nmy file.txt \r\n# not-a-comment-because-of-the-indent\r\n  # indented is a pattern\r\n"
	ig, err := ParseIgnore([]byte(ignore))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		rel    string
		folder bool
		left   bool
	}{
		// A name without a slash matches at any depth, file or folder.
		{"node_modules", true, true},
		{"a/b/node_modules", true, true},
		{"a/node_modules", false, true},
		{"node_modules_x", true, false},
		// A trailing slash matches only folders, at any depth when it is the only slash.
		{"build", true, true},
		{"src/build", true, true},
		{"build", false, false},
		{"src/build", false, false},
		// Wildcards stay within a name.
		{"x.log", false, true},
		{"deep/er/x.log", false, true},
		{"x.log.txt", false, false},
		// A pattern with a slash is a path from the project folder, file or folder.
		{"docs/generated", true, true},
		{"docs/generated", false, true},
		{"src/docs/generated", true, false},
		{"docs/generated/x", false, false},
		// A path pattern with a trailing slash matches only folders.
		{"gen/a", true, true},
		{"gen/a", false, false},
		{"gen/a/b", true, false},
		{"x/gen/a", true, false},
		// Spaces are part of the name, and trailing white space is dropped.
		{"my file.txt", false, true},
		{"sub/my file.txt", false, true},
		{"my file.txt ", false, false},
		// The line with a comment mark after indentation is a pattern.
		{"  # indented is a pattern", false, true},
		{"# build output", false, false},
		{"src/main.go", false, false},
	} {
		if got := ig.leaves(tc.rel, tc.folder); got != tc.left {
			t.Errorf("%q (folder %v): left out %v, want %v", tc.rel, tc.folder, got, tc.left)
		}
	}
	var none *Ignore
	if none.leaves("a", false) || (&Ignore{}).leaves("a", true) {
		t.Error("an empty ignore leaves something out")
	}
}

func TestReadIgnore(t *testing.T) {
	ig, err := ReadIgnore(tree(map[string]string{"a": ""}))
	if err != nil || ig.leaves("x", false) {
		t.Errorf("no file: %v", err)
	}
	ig, err = ReadIgnore(tree(map[string]string{".markitect/ignore": "x\n"}))
	if err != nil || !ig.leaves("x", false) {
		t.Errorf("file: %v", err)
	}
	if _, err = ReadIgnore(tree(map[string]string{".markitect/ignore": "x[\n"})); err == nil || !strings.HasPrefix(err.Error(), ".markitect/ignore:1:") {
		t.Errorf("bad file: %v", err)
	}
	// A folder of that name cannot be read as the file.
	if _, err = ReadIgnore(tree(map[string]string{".markitect/ignore/x": "x\n"})); err == nil || !strings.HasPrefix(err.Error(), ".markitect/ignore:") {
		t.Errorf("folder: %v", err)
	}
}

func TestListFiles(t *testing.T) {
	fsys := tree(map[string]string{
		"main.go":                  "",
		"docs/spec/a.md":           "# A\n",
		"docs/spec/img/a.png":      "png",
		"docs/notes.txt":           "",
		"docs/spec-old/b.md":       "",
		".git/config":              "",
		".github/workflows/ci.yml": "",
		".hidden":                  "",
		"src/.env":                 "",
		"src/code/x.go":            "",
		"src/.cache/y.go":          "",
		"vendor/lib/z.go":          "",
		"src/vendor/q.go":          "",
		"a b/c d.txt":              "",
		"out/x.o":                  "",
		"keep/out":                 "",
		".markitect/spec":          "docs/spec\n",
		".markitect/ignore":        "vendor\n/out\n",
	})
	// An invalid ignore file never reaches listFiles; the valid one first.
	ig, err := ParseIgnore([]byte("vendor\nout/\n"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := listFiles(fsys, ig, Folder{Path: "docs/spec"})
	if err != nil {
		t.Fatal(err)
	}
	// What starts with a dot, the spec folder, a nested ignored folder and an ignored folder
	// are left out; a file of the name of an ignored folder is not.
	want := "a b/c d.txt docs/notes.txt docs/spec-old/b.md keep/out main.go src/code/x.go"
	if names(got) != want {
		t.Errorf("got  %s\nwant %s", names(got), want)
	}
	if got["docs/notes.txt"] != 0 || got["main.go"] != 0 {
		t.Errorf("sizes %v", got)
	}
	// Without an ignore file, only the fixed rules apply.
	got, err = listFiles(fsys, nil, Folder{Path: "docs/spec"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "a b/c d.txt docs/notes.txt docs/spec-old/b.md keep/out main.go out/x.o src/code/x.go src/vendor/q.go vendor/lib/z.go"; names(got) != want {
		t.Errorf("got  %s\nwant %s", names(got), want)
	}
}

// failing makes a file that cannot be opened or read.
type failing struct {
	fs.FS
	open, read string
}

type brokenFile struct{ fs.File }

func (brokenFile) Read([]byte) (int, error) { return 0, errors.New("disk error") }

func (f failing) Open(name string) (fs.File, error) {
	if name == f.open {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	file, err := f.FS.Open(name)
	if err == nil && name == f.read {
		return brokenFile{file}, nil
	}
	return file, err
}

func TestSameBytes(t *testing.T) {
	cmp := newComparer()
	big := strings.Repeat("0123456789abcdef", 64<<10/16*3) // three chunks
	flip := func(s string, at int) string { return s[:at] + "X" + s[at+1:] }
	for name, tc := range map[string]struct {
		a, b string
		same bool
	}{
		"equal":            {"a\nb", "a\nb", true},
		"empty":            {"", "", true},
		"line ends differ": {"a\nb", "a\r\nb", false},
		"one byte":         {"abc", "abd", false},
		"binary":           {"\x00\x01\x02", "\x00\x01\x03", false},
		"long equal":       {big, big, true},
		"long first":       {big, flip(big, 0), false},
		"long at a chunk":  {big, flip(big, 64<<10), false},
		"long last":        {big, flip(big, len(big)-1), false},
		"exact chunk":      {big[:64<<10], big[:64<<10], true},
		"exact chunk last": {big[:64<<10], flip(big[:64<<10], 64<<10-1), false},
		"size differs":     {"abc", "abcd", false},
	} {
		a, b := tree(map[string]string{"f": tc.a}), tree(map[string]string{"f": tc.b})
		got, err := cmp.same(a, b, "f", int64(len(tc.a)), int64(len(tc.b)))
		if err != nil || got != tc.same {
			t.Errorf("%s: got %v %v, want %v", name, got, err, tc.same)
		}
	}
	a := tree(map[string]string{"dir/f.bin": "same"})
	for name, tc := range map[string]struct {
		old, new fs.FS
		want     string
	}{
		"open old":  {failing{FS: a, open: "dir/f.bin"}, a, "cannot read dir/f.bin in the old project: permission denied"},
		"open new":  {a, failing{FS: a, open: "dir/f.bin"}, "cannot read dir/f.bin in the new project: permission denied"},
		"read new":  {a, failing{FS: a, read: "dir/f.bin"}, "cannot read dir/f.bin in the new project: disk error"},
		"read both": {failing{FS: a, read: "dir/f.bin"}, failing{FS: a, read: "dir/f.bin"}, "cannot read dir/f.bin in the old project: disk error"},
	} {
		if _, err := cmp.same(tc.old, tc.new, "dir/f.bin", 4, 4); err == nil || err.Error() != tc.want {
			t.Errorf("%s: %v, want %s", name, err, tc.want)
		}
	}
}
