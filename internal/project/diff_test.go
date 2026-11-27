package project

import (
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	old := []byte("# Return\r\n\r\nA book comes back within three weeks.\r\nLate books cost a fee.\r\n")
	new := []byte("# Return\n\nA book comes back within four weeks.\nLate books cost a fee.\nA lost book is paid.\n")
	want := "--- docs/model/return.md\n+++ docs/model/return.md\n@@ -1,4 +1,5 @@\n # Return\n \n-A book comes back within three weeks.\n+A book comes back within four weeks.\n Late books cost a fee.\n+A lost book is paid.\n"
	if got := Diff("docs/model/return.md", "docs/model/return.md", old, new); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if Diff("a", "b", old, []byte(strings.ReplaceAll(string(old), "\r\n", "\n"))) != "" {
		t.Error("line ends made a diff")
	}
	var long []string
	for i := 0; i < 20; i++ {
		long = append(long, "line")
	}
	a := strings.Join(append(append([]string{"start"}, long...), "end"), "\n") + "\n"
	b := strings.Join(append(append([]string{"begin"}, long...), "finish"), "\n") + "\n"
	got := Diff("a", "b", []byte(a), []byte(b))
	if strings.Count(got, "@@ ") != 2 || !strings.Contains(got, "@@ -1,4 +1,4 @@\n-start\n+begin\n line\n line\n line\n") ||
		!strings.Contains(got, "@@ -19,4 +19,4 @@\n line\n line\n line\n-end\n+finish\n") {
		t.Error(got)
	}
	if got := Diff("a", "b", nil, []byte("one\n")); got != "--- a\n+++ b\n@@ -0,0 +1 @@\n+one\n" {
		t.Error(got)
	}
}

func TestDiffOfALineEnd(t *testing.T) {
	got := Diff("a", "b", []byte("one\ntwo"), []byte("one\ntwo\n"))
	if got != "--- a\n+++ b\n@@ -1,2 +1,2 @@\n one\n-two\n\\ No newline at end of file\n+two\n" {
		t.Errorf("%q", got)
	}
	if got := Diff("a", "b", []byte("one\ntwo"), []byte("one\r\ntwo")); got != "" {
		t.Errorf("%q", got)
	}
}
