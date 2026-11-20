package state

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestInitFiles(t *testing.T) {
	got := map[string]string{}
	for _, f := range InitFiles("1.2.3", "docs/spec") {
		got[f.Path] = string(f.Data)
	}
	want := map[string]string{
		".markitect/version":        "1.2.3\n",
		".markitect/spec":           "docs/spec\n",
		".markitect/.gitattributes": "state.jsonl merge=union\nhistory.jsonl merge=union\n",
		".markitect/ignore":         "",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for p, w := range want {
		if got[p] != w {
			t.Errorf("%s: got %q, want %q", p, got[p], w)
		}
	}
}

func TestSpecPath(t *testing.T) {
	for arg, want := range map[string]string{
		"docs/spec": "docs/spec", `docs\spec\`: "docs/spec", " ./docs//spec ": "docs/spec",
		"spec": "spec", "a/../spec": "spec", "docs/.markitect": "docs/.markitect",
	} {
		if got, err := SpecPath(arg); err != nil || got != want {
			t.Errorf("%q: got %q %v, want %q", arg, got, err, want)
		}
	}
	for _, arg := range []string{"", " ", ".", "./", "..", "../spec", "/spec", `C:\spec`, "c:spec", ".markitect", ".markitect/spec", ".Markitect/x", "a/.git/spec", ".git", "a/.GIT/b", ".Git"} {
		if got, err := SpecPath(arg); err == nil {
			t.Errorf("%q: got %q, want a refusal", arg, got)
		}
	}
}

func TestReadLayout(t *testing.T) {
	fsys := fstest.MapFS{
		".markitect/version": {Data: []byte("\xef\xbb\xbf 0.35.0\r\n")},
		".markitect/spec":    {Data: []byte("docs\\spec\r\n")},
	}
	l, err := ReadLayout(fsys)
	if err != nil || l.Version != "0.35.0" || l.SpecFolder != "docs/spec" {
		t.Errorf("got %+v %v", l, err)
	}
	for name, files := range map[string]fstest.MapFS{
		"version missing":   {".markitect/spec": {Data: []byte("docs/spec\n")}},
		"version empty":     {".markitect/version": {Data: []byte(" \n")}, ".markitect/spec": {Data: []byte("docs/spec\n")}},
		"two version lines": {".markitect/version": {Data: []byte("1\n2\n")}, ".markitect/spec": {Data: []byte("docs/spec\n")}},
		"spec missing":      {".markitect/version": {Data: []byte("1\n")}},
		"spec outside":      {".markitect/version": {Data: []byte("1\n")}, ".markitect/spec": {Data: []byte("../spec\n")}},
	} {
		if _, err := ReadLayout(files); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestLoad(t *testing.T) {
	fsys := fstest.MapFS{
		".markitect/version":      {Data: []byte("1.0.0\n")},
		".markitect/spec":         {Data: []byte("docs/spec\n")},
		"docs/spec/lending.md":    {Data: []byte("# Lending\n\nSee [loans](loans/loan.md).\n")},
		"docs/spec/loans/loan.md": {Data: []byte("# Loan\n")},
		"docs/other.md":           {Data: []byte("# Not in the spec\n")},
	}
	p, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, d := range p.Spec.Documents {
		paths = append(paths, d.Path)
	}
	if p.Version != "1.0.0" || p.Spec == nil || strings.Join(paths, " ") != "lending.md loans/loan.md" {
		t.Errorf("got %+v %v", p.Layout, paths)
	}
	delete(fsys, "docs/spec/lending.md")
	delete(fsys, "docs/spec/loans/loan.md")
	if _, err := Load(fsys); err == nil || !strings.Contains(err.Error(), "docs/spec") {
		t.Errorf("a missing spec folder: %v", err)
	}
}
