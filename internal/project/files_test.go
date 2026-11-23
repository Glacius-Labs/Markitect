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
