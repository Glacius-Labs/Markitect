package project

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestModelPath(t *testing.T) {
	for arg, want := range map[string]string{
		"docs/model": "docs/model", `docs\model\`: "docs/model", " ./docs//model ": "docs/model",
		"model": "model", "a/../model": "model", "docs/.markitect": "docs/.markitect",
	} {
		if got, err := ModelPath(arg); err != nil || got != want {
			t.Errorf("%q: got %q %v, want %q", arg, got, err, want)
		}
	}
	for _, arg := range []string{"", " ", ".", "./", "..", "../model", "/model", `C:\model`, "c:model", ".markitect", ".markitect/model", ".Markitect/x"} {
		if got, err := ModelPath(arg); err == nil {
			t.Errorf("%q: got %q, want a refusal", arg, got)
		}
	}
}

func TestReadModelFolder(t *testing.T) {
	fsys := fstest.MapFS{".markitect/model": {Data: []byte("\xef\xbb\xbf docs\\model\r\n")}}
	if got, err := ReadModelFolder(fsys); err != nil || got != "docs/model" {
		t.Errorf("got %q %v", got, err)
	}
	// Whatever else lies in .markitect/ is ignored.
	fsys[".markitect/version"] = &fstest.MapFile{Data: []byte("not a version")}
	fsys[".markitect/state.jsonl"] = &fstest.MapFile{Data: []byte("{")}
	if got, err := ReadModelFolder(fsys); err != nil || got != "docs/model" {
		t.Errorf("got %q %v", got, err)
	}
	for name, files := range map[string]fstest.MapFS{
		"missing":   {".markitect/version": {Data: []byte("1\n")}},
		"empty":     {".markitect/model": {Data: []byte(" \n")}},
		"two lines": {".markitect/model": {Data: []byte("docs\nmodel\n")}},
		"outside":   {".markitect/model": {Data: []byte("../model\n")}},
	} {
		if _, err := ReadModelFolder(files); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestLoad(t *testing.T) {
	fsys := fstest.MapFS{
		".markitect/model":         {Data: []byte("docs/model\n")},
		"docs/model/lending.md":    {Data: []byte("# Lending\n\nSee [loans](loans/loan.md).\n")},
		"docs/model/loans/loan.md": {Data: []byte("# Loan\n")},
		"docs/other.md":            {Data: []byte("# Not in the model\n")},
	}
	p, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, d := range p.Model.Documents {
		paths = append(paths, d.Path)
	}
	if p.ModelFolder != "docs/model" || p.Model == nil || strings.Join(paths, " ") != "lending.md loans/loan.md" {
		t.Errorf("got %q %v", p.ModelFolder, paths)
	}
	delete(fsys, "docs/model/lending.md")
	delete(fsys, "docs/model/loans/loan.md")
	if _, err := Load(fsys); err == nil || !strings.Contains(err.Error(), "docs/model") {
		t.Errorf("a missing model folder: %v", err)
	}
}
