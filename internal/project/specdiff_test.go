package project

import (
	"strings"
	"testing"
	"testing/fstest"
)

func projectOf(t *testing.T, files map[string]string) *Project {
	t.Helper()
	fsys := fstest.MapFS{
		".markitect/spec": {Data: []byte("docs/spec\n")},
	}
	for p, text := range files {
		fsys["docs/spec/"+p] = &fstest.MapFile{Data: []byte(text)}
	}
	p, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDiffSpecs(t *testing.T) {
	older := projectOf(t, map[string]string{
		"a.md":     "# A\r\n\r\nOne.\r\n",
		"b.md":     "# B\n\nTwo.\n",
		"gone.md":  "# Gone\n",
		"sub/c.md": "# C\n",
	})
	newer := projectOf(t, map[string]string{
		"a.md":     "# A\n\nOne.\n", // only the line ends differ
		"b.md":     "# B\n\nTwo, changed.\n",
		"new.md":   "# New\n",
		"sub/c.md": "# C\n",
	})
	c := DiffSpecs(older, newer)
	if strings.Join(c.New, " ") != "new.md" || strings.Join(c.Gone, " ") != "gone.md" || len(c.Changed) != 1 || c.Changed[0].Doc != "b.md" || !c.Differs() {
		t.Fatalf("got %+v", c)
	}
	if !strings.HasPrefix(c.Changed[0].Diff, "--- docs/spec/b.md\n+++ docs/spec/b.md\n") || !strings.Contains(c.Changed[0].Diff, "-Two.\n+Two, changed.\n") {
		t.Errorf("diff %q", c.Changed[0].Diff)
	}
	if DiffSpecs(older, older).Differs() {
		t.Error("a project differs from itself")
	}
}
