package project

import (
	"strings"
	"testing"
	"testing/fstest"
)

func projectOf(t *testing.T, files map[string]string) *Project {
	t.Helper()
	fsys := fstest.MapFS{
		".markitect/model": {Data: []byte("docs/model\n")},
	}
	for p, text := range files {
		fsys["docs/model/"+p] = &fstest.MapFile{Data: []byte(text)}
	}
	p, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDiffModels(t *testing.T) {
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
	c := DiffModels(older, newer)
	if strings.Join(c.New, " ") != "new.md" || strings.Join(c.Gone, " ") != "gone.md" || len(c.Changed) != 1 || c.Changed[0].Doc != "b.md" || !c.Differs() {
		t.Fatalf("got %+v", c)
	}
	if !strings.HasPrefix(c.Changed[0].Diff, "--- docs/model/b.md\n+++ docs/model/b.md\n") || !strings.Contains(c.Changed[0].Diff, "-Two.\n+Two, changed.\n") {
		t.Errorf("diff %q", c.Changed[0].Diff)
	}
	if DiffModels(older, older).Differs() {
		t.Error("a project differs from itself")
	}
}
