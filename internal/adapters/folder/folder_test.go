package folder

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
		".gitignore":               "# build output\n*.log\n!keep.log\nbuild/\n/top.txt\ndocs/**/draft-*.md\n\\#hash.txt\nfoo/**\n!foo/keep.txt\n*.[!o]x\n",
		"foo/keep.txt":             "kept by the negation inside foo/**\n",
		"foo/other.txt":            "ignored by foo/**\n",
		"a.ox":                     "kept: [!o] is not o\n",
		"a.cx":                     "ignored by [!o]\n",
		"fake/.git":                "not a repository\n",
		"fake/x.go":                "walked, since fake/.git names no repository\n",
		"real/.git/HEAD":           "ref: refs/heads/main\n",
		"real/x.go":                "inside another repository\n",
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
		"nested/file.txt":          "another worktree's\n",
		"src/main.go":              "package main\n",
	})
	w, err := Walk(dir)
	if err != nil {
		t.Fatal(err)
	}
	files, _ := w.Files()
	got := strings.Join(files, ",")
	want := ".gitignore,.markitect/manifest.yaml,a.ox,docs/a/final.md,fake/x.go,foo/keep.txt,keep.log,local.txt,nested,real,src/main.go,sub/.gitignore,sub/top.txt"
	if got != want {
		t.Fatalf("the walk found\n%s\nnot\n%s", got, want)
	}
	// Another repository is one entry, as in Git, so that check names it.
	if data, err := w.Read("real"); err != nil || !strings.Contains(string(data), "repository") {
		t.Fatalf("%q %v", data, err)
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
	root, err := Disk{}.Root(filepath.Join(dir, "src", "a"))
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := filepath.Abs(dir); !strings.EqualFold(root, want) {
		t.Fatalf("root %s, not %s", root, want)
	}
	if _, err := (Disk{}).Root(testkit.TempDir(t)); err == nil || !strings.Contains(err.Error(), "markitect init") {
		t.Fatalf("outside a project: %v", err)
	}
	// A folder that does not exist is not taken for the project around it.
	if _, err := (Disk{}).Root(filepath.Join(dir, "src", "nosuch")); err == nil {
		t.Fatal("a missing folder found the project above it")
	}
}

// A project in a folder of a repository honours the .gitignore files above it, up
// to the top of the repository, each from its own folder, as Git does.
func TestFolderInARepository(t *testing.T) {
	top := testkit.TempDir(t)
	put(t, top, map[string]string{
		".git/HEAD":                             "ref: refs/heads/main\n",
		".gitignore":                            "node_modules/\n/build\n*.log\n",
		"services/.gitignore":                   "!keep.log\n",
		"services/api/.markitect/manifest.yaml": "libraries: {}\n",
		"services/api/node_modules/x.js":        "ignored from the top\n",
		"services/api/build/out.txt":            "kept: /build is anchored at the top\n",
		"services/api/a.log":                    "ignored from the top\n",
		"services/api/keep.log":                 "brought back from services\n",
		"services/api/src/main.go":              "package main\n",
	})
	root, err := Disk{}.Root(filepath.Join(top, "services", "api", "src"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := Walk(root)
	if err != nil {
		t.Fatal(err)
	}
	files, _ := w.Files()
	if got, want := strings.Join(files, ","), ".markitect/manifest.yaml,build/out.txt,keep.log,src/main.go"; got != want {
		t.Fatalf("the walk found\n%s\nnot\n%s", got, want)
	}
	ws, err := Disk{}.Project(root)
	if err != nil {
		t.Fatal(err)
	}
	if !ws.GitIgnored("node_modules/x.js") || ws.GitIgnored("src/main.go") {
		t.Fatal("the workspace reads the .gitignore files above differently")
	}
}

// A symbolic link is a file whose content is where it points, as Git stores it.
func TestFolderLinks(t *testing.T) {
	dir := testkit.TempDir(t)
	put(t, dir, map[string]string{".markitect/manifest.yaml": "libraries: {}\n", "a.txt": "a\n"})
	if err := os.Symlink("a.txt", filepath.Join(dir, "link")); err != nil {
		t.Skipf("this system makes no symbolic link here: %v", err)
	}
	files, links, err := Disk{}.Open(dir)
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
