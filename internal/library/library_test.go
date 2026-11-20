package library

import (
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
)

func TestEmbedsPackagesTemplatesAndTheReportType(t *testing.T) {
	folders := func(dir string) []string {
		entries, err := fs.ReadDir(FS(), dir)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, e := range entries {
			if e.IsDir() {
				out = append(out, e.Name())
			}
		}
		sort.Strings(out)
		return out
	}
	if got := strings.Join(folders("packages"), " "); got != "company core software" {
		t.Errorf("packages: %s", got)
	}
	if got := strings.Join(folders("templates"), " "); got != "company software" {
		t.Errorf("templates: %s", got)
	}
	report, err := fs.ReadFile(FS(), ReportType)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(report), "# Report\n") {
		t.Errorf("report type starts %q", strings.SplitN(string(report), "\n", 2)[0])
	}
	if _, err := fs.Stat(FS(), path.Join("packages", ReportType)); err == nil {
		t.Error("the report type is inside packages/")
	}
}

func TestEveryFileIsADocumentWithATitleAndLineFeeds(t *testing.T) {
	n := 0
	err := fs.WalkDir(FS(), ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		n++
		data, err := fs.ReadFile(FS(), p)
		if err != nil {
			return err
		}
		text := string(data)
		if path.Ext(p) != ".md" || !strings.HasPrefix(text, "# ") || strings.Contains(text, "\r") || !strings.HasSuffix(text, "\n") {
			t.Errorf("%s is not a document with a title, LF line ends and a final line break", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 40 {
		t.Errorf("only %d files", n)
	}
}

func TestEveryPackageHasReferencesAndQuestions(t *testing.T) {
	entries, err := fs.ReadDir(FS(), "packages")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		refs, err := fs.ReadFile(FS(), "references/"+e.Name()+".md")
		if err != nil {
			t.Fatal(err)
		}
		types, _ := fs.Glob(FS(), "packages/"+e.Name()+"/*.md")
		if rows := strings.Count(string(refs), "\n| ") - 1; rows != len(types) {
			t.Errorf("%s: %d reference rows for %d types", e.Name(), rows, len(types))
		}
		questions, err := fs.ReadFile(FS(), "questions/"+e.Name()+".md")
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(questions), "\n- "); n < 5 || n > 10 {
			t.Errorf("%s: %d questions", e.Name(), n)
		}
	}
}
