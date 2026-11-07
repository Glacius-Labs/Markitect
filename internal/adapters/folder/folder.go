// Package folder reads and writes a project folder without Git (DEC-097): it
// walks the folder and skips what the project's .gitignore files exclude, read as
// plain text, nested ones included. Global Git settings are out of scope; where
// this differs from Git's view, check names the file. A folder with its own .git
// is another repository and is not walked, as Git does not.
package folder

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Disk opens project folders on the local disk: for the record (Root, Open) and
// for the command line (Project, At).
type Disk struct{}

// Root is the project folder that holds dir: the nearest folder, dir or above,
// with Markitect's manifest in it. dir must exist, so that a mistyped folder is
// not taken for the project around it.
func (Disk) Root(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err != nil {
		return "", err
	} else if !info.IsDir() {
		return "", fmt.Errorf("%s is no folder", dir)
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

// Open walks the project folder at root, and returns its files.
func (Disk) Open(root string) (project.Repository, error) {
	w, err := Walk(root)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Project opens the project folder that holds dir.
func (d Disk) Project(dir string) (*Workspace, error) {
	root, err := d.Root(dir)
	if err != nil {
		return nil, err
	}
	return &Workspace{Dir: root, ignore: newIgnores(root)}, nil
}

// At opens dir itself, which need not be a project yet: init makes it one.
func (Disk) At(dir string) (*Workspace, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(abs); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%s is no folder", dir)
	}
	return &Workspace{Dir: abs, ignore: newIgnores(abs)}, nil
}

// Workspace is a project folder that a command reads and writes. Files walks it
// at each call, so what the command wrote is seen at once.
type Workspace struct {
	Dir    string
	ignore *ignores
}

// Root returns the project folder.
func (w *Workspace) Root() string { return w.Dir }

// Files returns every file a walk of the folder finds.
func (w *Workspace) Files() ([]string, error) {
	walked, err := Walk(w.Dir)
	if err != nil {
		return nil, err
	}
	return walked.Files()
}

// Read reads a file, as the walk does.
func (w *Workspace) Read(file string) ([]byte, error) {
	return readFile(w.Dir, file)
}

// GitIgnored says whether the project's .gitignore files exclude a file.
func (w *Workspace) GitIgnored(file string) bool { return w.ignore.excludes(file) }

// Write writes a file of the folder, creating its folders.
func (w *Workspace) Write(file string, data []byte) error {
	full := filepath.Join(w.Dir, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}

// Exists reports whether a path exists in the folder.
func (w *Workspace) Exists(file string) bool {
	_, err := os.Lstat(filepath.Join(w.Dir, filepath.FromSlash(file)))
	return err == nil
}

// Remove deletes a file of the folder.
func (w *Workspace) Remove(file string) error {
	return os.Remove(filepath.Join(w.Dir, filepath.FromSlash(file)))
}

// List returns the files below a folder of the project, relative to the project
// folder, whether or not the walk would take them.
func (w *Workspace) List(dir string) ([]string, error) {
	var out []string
	root := filepath.Join(w.Dir, filepath.FromSlash(dir))
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return filepath.SkipDir
			}
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(w.Dir, p)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out, err
}

// Walk walks the project folder at root.
func Walk(root string) (*Walked, error) {
	w := &Walked{Dir: root, ignore: newIgnores(root), repos: map[string]bool{}}
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
	return readFile(w.Dir, file)
}

// readFile reads a file of the folder at root: another repository as its entry,
// and a symbolic link as where it points, as Git stores them.
func readFile(root, file string) ([]byte, error) {
	p := filepath.Join(root, filepath.FromSlash(file))
	info, err := os.Lstat(p)
	switch {
	case err != nil:
		return nil, err
	case info.Mode()&os.ModeSymlink != 0:
		to, err := os.Readlink(p)
		return []byte(to), err
	case info.IsDir() && nestedRepository(p):
		return []byte(repositoryEntry), nil
	}
	return os.ReadFile(p)
}

// GitIgnored says whether the project's .gitignore files exclude a file.
func (w *Walked) GitIgnored(file string) bool { return w.ignore.excludes(file) }

// excludes says whether the .gitignore files exclude a file, itself or a folder
// it lies in.
func (g *ignores) excludes(file string) bool {
	parts := strings.Split(file, "/")
	for i := 1; i < len(parts); i++ {
		if g.ignored(strings.Join(parts[:i], "/"), true) {
			return true
		}
	}
	return g.ignored(file, false)
}

// ignores is the .gitignore files of a folder, read as they are needed, and of the
// folders above it up to the top of the repository it lies in, if any.
type ignores struct {
	root  string
	files map[string][]pattern
	// above holds the .gitignore files above root, the outermost first.
	above []ancestor
}

// ancestor is the .gitignore file of a folder above the project folder: its
// patterns, and the path from that folder down to the project folder.
type ancestor struct {
	down     string
	patterns []pattern
}

func newIgnores(root string) *ignores {
	return &ignores{root: root, files: map[string][]pattern{}, above: ancestors(root)}
}

// ancestors reads the .gitignore files of the folders above root, up to the
// nearest one with a .git, which is the top of the repository the project lies
// in, as Git applies them. A project in no repository has none.
func ancestors(root string) []ancestor {
	if _, err := os.Lstat(filepath.Join(root, ".git")); err == nil {
		return nil
	}
	var out []ancestor
	down := ""
	for at := root; ; {
		up := filepath.Dir(at)
		if up == at {
			// No repository above: no folder above counts.
			return nil
		}
		down = path.Join(filepath.Base(at), down)
		if data, err := os.ReadFile(filepath.Join(up, ".gitignore")); err == nil {
			out = append([]ancestor{{down: down, patterns: parseIgnore("", string(data))}}, out...)
		}
		if _, err := os.Lstat(filepath.Join(up, ".git")); err == nil {
			return out
		}
		at = up
	}
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
	// The files above the project folder match the path from their own folder.
	for _, a := range g.above {
		for _, p := range a.patterns {
			if p.matches(path.Join(a.down, rel), dir) {
				ignored = !p.negate
			}
		}
	}
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
