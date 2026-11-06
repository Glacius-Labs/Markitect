package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func put(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for file, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// The walk applies the project's .gitignore files as Git does, nested ones and
// negation included, and skips Git's own folder and other repositories.
func TestFolderWalksWithoutGit(t *testing.T) {
	dir := testkit.TempDir(t)
	put(t, dir, map[string]string{
		".markitect/manifest.yaml": "libraries: {}\n",
		".gitignore":               "# build output\n*.log\n!keep.log\nbuild/\n/top.txt\ndocs/**/draft-*.md\n\\#hash.txt\n",
		"keep.log":                 "kept\n",
		"run.log":                  "ignored\n",
		"top.txt":                  "ignored at the top\n",
		"sub/top.txt":              "kept below\n",
		"sub/.gitignore":           "local.txt\r\n",
		"sub/local.txt":            "ignored by the nested file\n",
		"sub/x/local.txt":          "ignored at any depth below\n",
		"sub/build/out.bin":        "ignored folder\n",
		"local.txt":                "kept above the nested file\n",
		"docs/a/b/draft-1.md":      "ignored by **\n",
		"docs/a/final.md":          "kept\n",
		"#hash.txt":                "ignored by an escaped #\n",
		".git/config":              "Git's own folder\n",
		"nested/.git":              "gitdir: elsewhere\n",
		"nested/file.txt":          "another repository's\n",
		"src/main.go":              "package main\n",
	})
	w, err := Walk(dir)
	if err != nil {
		t.Fatal(err)
	}
	files, _ := w.Files()
	got := strings.Join(files, ",")
	want := ".gitignore,.markitect/manifest.yaml,docs/a/final.md,keep.log,local.txt,src/main.go,sub/.gitignore,sub/top.txt"
	if got != want {
		t.Fatalf("the walk found\n%s\nnot\n%s", got, want)
	}
	for file, ignored := range map[string]bool{"run.log": true, "sub/build/out.bin": true, "build/x": true, "keep.log": false, "src/main.go": false, "sub/local.txt": true} {
		if w.GitIgnored(file) != ignored {
			t.Errorf("GitIgnored(%s) is %v", file, !ignored)
		}
	}
	if data, err := w.Read("src/main.go"); err != nil || string(data) != "package main\n" {
		t.Fatalf("%q %v", data, err)
	}
}

// Root finds the project folder above a subfolder, and says when there is none.
func TestFolderRoot(t *testing.T) {
	dir := testkit.TempDir(t)
	put(t, dir, map[string]string{".markitect/manifest.yaml": "libraries: {}\n", "src/a/x.go": "package a\n"})
	root, err := Folder{}.Root(filepath.Join(dir, "src", "a"))
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := filepath.Abs(dir); !strings.EqualFold(root, want) {
		t.Fatalf("root %s, not %s", root, want)
	}
	if _, err := (Folder{}).Root(testkit.TempDir(t)); err == nil || !strings.Contains(err.Error(), "markitect init") {
		t.Fatalf("outside a project: %v", err)
	}
}

// A symbolic link is a file whose content is where it points, as Git stores it.
func TestFolderLinks(t *testing.T) {
	dir := testkit.TempDir(t)
	put(t, dir, map[string]string{".markitect/manifest.yaml": "libraries: {}\n", "a.txt": "a\n"})
	if err := os.Symlink("a.txt", filepath.Join(dir, "link")); err != nil {
		t.Skipf("this system makes no symbolic link here: %v", err)
	}
	files, links, err := Folder{}.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if links["link"] != "a.txt" {
		t.Fatalf("links %v", links)
	}
	if data, err := files.Read("link"); err != nil || string(data) != "a.txt" {
		t.Fatalf("%q %v", data, err)
	}
}
