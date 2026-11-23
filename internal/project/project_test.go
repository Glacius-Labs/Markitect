package project

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestSpecPath(t *testing.T) {
	for arg, want := range map[string]string{
		"docs/spec": "docs/spec", `docs\spec\`: "docs/spec", " ./docs//spec ": "docs/spec",
		"spec": "spec", "a/../spec": "spec", "docs/.markitect": "docs/.markitect",
	} {
		if got, err := SpecPath(arg); err != nil || got != want {
			t.Errorf("%q: got %q %v, want %q", arg, got, err, want)
		}
	}
	for _, arg := range []string{"", " ", ".", "./", "..", "../spec", "/spec", `C:\spec`, "c:spec", ".markitect", ".markitect/spec", ".Markitect/x"} {
		if got, err := SpecPath(arg); err == nil {
			t.Errorf("%q: got %q, want a refusal", arg, got)
		}
	}
}

func TestReadSpecFolder(t *testing.T) {
	fsys := fstest.MapFS{".markitect/spec": {Data: []byte("\xef\xbb\xbf docs\\spec\r\n")}}
	if got, err := ReadSpecFolder(fsys); err != nil || got != "docs/spec" {
		t.Errorf("got %q %v", got, err)
	}
	// Whatever else lies in .markitect/ is ignored.
	fsys[".markitect/version"] = &fstest.MapFile{Data: []byte("not a version")}
	fsys[".markitect/state.jsonl"] = &fstest.MapFile{Data: []byte("{")}
	if got, err := ReadSpecFolder(fsys); err != nil || got != "docs/spec" {
		t.Errorf("got %q %v", got, err)
	}
	for name, files := range map[string]fstest.MapFS{
		"missing":   {".markitect/version": {Data: []byte("1\n")}},
		"empty":     {".markitect/spec": {Data: []byte(" \n")}},
		"two lines": {".markitect/spec": {Data: []byte("docs\nspec\n")}},
		"outside":   {".markitect/spec": {Data: []byte("../spec\n")}},
	} {
		if _, err := ReadSpecFolder(files); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestLoad(t *testing.T) {
	fsys := fstest.MapFS{
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
	if p.SpecFolder != "docs/spec" || p.Spec == nil || strings.Join(paths, " ") != "lending.md loans/loan.md" {
		t.Errorf("got %q %v", p.SpecFolder, paths)
	}
	delete(fsys, "docs/spec/lending.md")
	delete(fsys, "docs/spec/loans/loan.md")
	if _, err := Load(fsys); err == nil || !strings.Contains(err.Error(), "docs/spec") {
		t.Errorf("a missing spec folder: %v", err)
	}
}
