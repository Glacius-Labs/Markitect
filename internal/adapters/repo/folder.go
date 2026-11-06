package repo

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Folder reads a project folder without Git (DEC-097): it walks the folder and
// skips what the project's .gitignore files exclude, read as plain text, nested
// ones included. Global Git settings are out of scope; where this differs from
// Git's view, check names the file. A folder with its own .git is another
// repository and is not walked, as Git does not.
type Folder struct{}

// Root is the project folder that holds dir: the nearest folder, dir or above,
// with Markitect's manifest in it.
func (Folder) Root(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for at := abs; ; {
		if info, err := os.Stat(filepath.Join(at, filepath.FromSlash(project.ManifestFile))); err == nil && !info.IsDir() {
			return at, nil
		}
		up := filepath.Dir(at)
		if up == at {
			return "", fmt.Errorf("%s is in no Markitect project: no folder above it holds %s; markitect init makes one", dir, project.ManifestFile)
		}
		at = up
	}
}

// Open walks the project folder at root, and returns its files and the symbolic
// links among them with where each points.
func (Folder) Open(root string) (project.Repository, map[string]string, error) {
	w, err := Walk(root)
	if err != nil {
		return nil, nil, err
	}
	return w, w.links, nil
}

// Walk walks the project folder at root.
func Walk(root string) (*Walked, error) {
	w := &Walked{Dir: root, ignore: newIgnores(root), links: map[string]string{}, repos: map[string]bool{}}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.Name() == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if w.ignore.ignored(rel, true) {
				return filepath.SkipDir
			}
			if nestedRepository(p) {
				// Git keeps another repository as one entry, and so does the
				// walk, so that check names it instead of losing its files.
				w.files = append(w.files, rel)
				w.repos[rel] = true
				return filepath.SkipDir
			}
			return nil
		}
		if w.ignore.ignored(rel, false) {
			return nil
		}
		w.files = append(w.files, rel)
		if d.Type()&fs.ModeSymlink != 0 {
			to, err := os.Readlink(p)
			if err != nil {
				return err
			}
			w.links[rel] = filepath.ToSlash(to)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(w.files)
	return w, nil
}

// nestedRepository says whether a folder is a repository or worktree of its own:
// its .git is a folder with a HEAD, or a file that names one. Any other .git is
// no repository, and the folder is walked.
func nestedRepository(dir string) bool {
	git := filepath.Join(dir, ".git")
	info, err := os.Lstat(git)
	switch {
	case err != nil:
		return false
	case info.IsDir():
		_, err := os.Stat(filepath.Join(git, "HEAD"))
		return err == nil
	}
	data, err := os.ReadFile(git)
	return err == nil && strings.HasPrefix(string(data), "gitdir:")
}

// repositoryEntry is what the entry of another repository reads as.
const repositoryEntry = "a repository of its own\n"

// Walked is the files of a project folder as one walk found them.
type Walked struct {
	Dir    string
	files  []string
	links  map[string]string
	repos  map[string]bool
	ignore *ignores
}

// Files returns every file the walk found, folder-relative with forward slashes.
func (w *Walked) Files() ([]string, error) { return append([]string(nil), w.files...), nil }

// Read reads a file. A symbolic link reads as where it points, as Git stores it.
func (w *Walked) Read(file string) ([]byte, error) {
	if w.repos[file] {
		return []byte(repositoryEntry), nil
	}
	p := filepath.Join(w.Dir, filepath.FromSlash(file))
	if info, err := os.Lstat(p); err == nil && info.Mode()&os.ModeSymlink != 0 {
		to, err := os.Readlink(p)
		return []byte(to), err
	}
	return os.ReadFile(p)
}

// GitIgnored says whether the project's .gitignore files exclude a file.
func (w *Walked) GitIgnored(file string) bool {
	parts := strings.Split(file, "/")
	for i := 1; i < len(parts); i++ {
		if w.ignore.ignored(strings.Join(parts[:i], "/"), true) {
			return true
		}
	}
	return w.ignore.ignored(file, false)
}

// ignores is the .gitignore files of a folder, read as they are needed.
type ignores struct {
	root  string
	files map[string][]pattern
}

func newIgnores(root string) *ignores {
	return &ignores{root: root, files: map[string][]pattern{}}
}

// pattern is one line of a .gitignore file.
type pattern struct {
	// base is the folder of the .gitignore file, "" at the root.
	base    string
	glob    string
	negate  bool
	dirOnly bool
	// anchored patterns match the path below base; the others match a name at any
	// depth.
	anchored bool
}

// patterns returns the patterns of the .gitignore file in a folder.
func (g *ignores) patterns(dir string) []pattern {
	if ps, ok := g.files[dir]; ok {
		return ps
	}
	// A folder without a .gitignore file has no patterns of its own.
	var ps []pattern
	if data, err := os.ReadFile(filepath.Join(g.root, filepath.FromSlash(dir), ".gitignore")); err == nil {
		ps = parseIgnore(dir, string(data))
	}
	g.files[dir] = ps
	return ps
}

// parseIgnore reads a .gitignore file in the folder base.
func parseIgnore(base, text string) []pattern {
	var out []pattern
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSuffix(line, "\r")
		// Trailing spaces go, unless a backslash keeps one.
		for strings.HasSuffix(line, " ") && !strings.HasSuffix(line, `\ `) {
			line = line[:len(line)-1]
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := pattern{base: base}
		if strings.HasPrefix(line, "!") {
			p.negate, line = true, line[1:]
		} else if strings.HasPrefix(line, `\!`) || strings.HasPrefix(line, `\#`) {
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			p.dirOnly, line = true, strings.TrimSuffix(line, "/")
		}
		p.anchored = strings.Contains(line, "/")
		p.glob = strings.TrimPrefix(line, "/")
		if p.glob != "" {
			out = append(out, p)
		}
	}
	return out
}

// ignored says whether the .gitignore files exclude a path, a folder with dir.
// Files further down win over those above, and in one file the last matching
// line wins.
func (g *ignores) ignored(rel string, dir bool) bool {
	parent := path.Dir(rel)
	if parent == "." {
		parent = ""
	}
	var folders []string
	for at := parent; ; at = path.Dir(at) {
		if at == "." {
			at = ""
		}
		folders = append(folders, at)
		if at == "" {
			break
		}
	}
	ignored := false
	for i := len(folders) - 1; i >= 0; i-- {
		for _, p := range g.patterns(folders[i]) {
			if p.matches(rel, dir) {
				ignored = !p.negate
			}
		}
	}
	return ignored
}

func (p pattern) matches(rel string, dir bool) bool {
	if p.dirOnly && !dir {
		return false
	}
	below := rel
	if p.base != "" {
		var ok bool
		if below, ok = strings.CutPrefix(rel, p.base+"/"); !ok {
			return false
		}
	}
	if !p.anchored {
		return globMatch(p.glob, path.Base(below))
	}
	return matchSegments(strings.Split(p.glob, "/"), strings.Split(below, "/"))
}

// matchSegments matches a path against a pattern part by part; "**" stands for
// any number of parts, none included.
func matchSegments(pat, name []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			if len(pat) == 1 {
				// A trailing "/**" holds everything inside, not the folder.
				return len(name) > 0
			}
			for i := 0; i <= len(name); i++ {
				if matchSegments(pat[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if !globMatch(pat[0], name[0]) {
			return false
		}
		pat, name = pat[1:], name[1:]
	}
	return len(name) == 0
}

// globMatch matches one part of a path as Git does: "[!x]" is "not x", which Go
// writes "[^x]".
func globMatch(glob, name string) bool {
	var b strings.Builder
	for i := 0; i < len(glob); i++ {
		switch {
		case glob[i] == '\\' && i+1 < len(glob):
			b.WriteString(glob[i : i+2])
			i++
		case glob[i] == '[' && i+1 < len(glob) && glob[i+1] == '!':
			b.WriteString("[^")
			i++
		default:
			b.WriteByte(glob[i])
		}
	}
	ok, _ := path.Match(b.String(), name)
	return ok
}
