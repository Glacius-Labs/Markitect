package state

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// The folders of the work in .markitect/, by paths below the project folder.
const (
	ContextDir  = ".markitect/context"
	FindingsDir = ".markitect/findings"
	PlanDir     = ".markitect/plan"
	BriefingDir = ".markitect/briefing"
)

// WorkRoots are the folders whose Markdown files are the documents of the
// work.
var WorkRoots = []string{ContextDir, FindingsDir, PlanDir, BriefingDir}

// WorkFolders are the folders init lays out for the work, each of which must
// survive a Git commit.
var WorkFolders = []string{ContextDir, FindingsDir + "/pending", FindingsDir + "/ready", FindingsDir + "/closed", PlanDir, BriefingDir}

// Work is the documents of the work, the reports, findings, plan items and
// briefings in .markitect/, with what check finds in them. They are no part
// of the spec: they have no row in the state, and a link from them into the
// spec makes nothing suspect.
type Work struct {
	// Documents are sorted by path below the project folder.
	Documents   []*lang.Document
	Diagnostics []lang.Diagnostic
	// Unreadable are the files ReadWorkLenient could not read, by path.
	Unreadable []string
}

// ReadWork reads the documents of the work of a project and checks them: a
// title; a type line, when there is one, that names a document of the spec,
// and then the sections that type requires; and every relative link, which
// must lead to an existing file or folder of the project. A document without
// a type line is checked for its title and its links only. A folder whose
// name starts with a dot, every file that is no Markdown document and
// everything the project's .gitignore files exclude, matched as the walk
// matches, are passed over, and a missing folder of the work is no error.
// .markitect/ignore does not apply: it names files that realize nothing. A
// file that cannot be read fails.
func ReadWork(p *Project) (*Work, error) { return readWork(p, false) }

// ReadWorkLenient reads the documents of the work as ReadWork does, without
// checking them, and leaves a file that cannot be read out, naming it in
// Unreadable.
func ReadWorkLenient(p *Project) (*Work, error) { return readWork(p, true) }

func readWork(p *Project, lenient bool) (*Work, error) {
	w := &Work{}
	git := &walker{fsys: p.FS, git: map[string][]pattern{}}
	for _, root := range WorkRoots {
		// A folder that .gitignore excludes is not read, nor is one inside
		// an excluded folder.
		skip := false
		for _, folder := range []string{Dir, root} {
			ex, err := git.excluded(folder, true)
			if err != nil {
				return nil, err
			}
			skip = skip || ex
		}
		if skip {
			continue
		}
		err := fs.WalkDir(p.FS, root, func(file string, e fs.DirEntry, err error) error {
			switch {
			case errors.Is(err, fs.ErrNotExist) && file == root:
				return fs.SkipDir
			case err != nil:
				return err
			}
			if file != root {
				ex, err := git.excluded(file, e.IsDir())
				if err != nil {
					return err
				}
				if ex && e.IsDir() {
					return fs.SkipDir
				} else if ex {
					return nil
				}
			}
			switch {
			case e.IsDir():
				if file != root && strings.HasPrefix(e.Name(), ".") {
					return fs.SkipDir
				}
				return nil
			case !lang.IsDocumentName(e.Name()):
				return nil
			}
			if !e.Type().IsRegular() {
				if info, err := fs.Stat(p.FS, file); err != nil || !info.Mode().IsRegular() {
					return nil
				}
			}
			data, err := fs.ReadFile(p.FS, file)
			if err != nil && lenient {
				w.Unreadable = append(w.Unreadable, file)
				return nil
			} else if err != nil {
				return err
			}
			w.Documents = append(w.Documents, lang.Parse(file, data))
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(w.Documents, func(i, j int) bool { return w.Documents[i].Path < w.Documents[j].Path })
	if lenient {
		return w, nil
	}
	for _, d := range w.Documents {
		w.Diagnostics = append(w.Diagnostics, p.checkWork(d)...)
	}
	sort.SliceStable(w.Diagnostics, func(i, j int) bool {
		if w.Diagnostics[i].Path != w.Diagnostics[j].Path {
			return w.Diagnostics[i].Path < w.Diagnostics[j].Path
		}
		return w.Diagnostics[i].Line < w.Diagnostics[j].Line
	})
	return w, nil
}

// checkWork checks one document of the work as lang.Check checks one of the
// spec: its title, its type line, the sections its type requires and its
// links; its type is a document of the spec, and a link leads to any existing
// file or folder of the project.
func (p *Project) checkWork(d *lang.Document) []lang.Diagnostic {
	out := lang.CheckTitle(d)
	add := func(line int, code, format string, args ...any) {
		out = append(out, lang.Diagnostic{Path: d.Path, Line: line, Code: code, Message: fmt.Sprintf(format, args...)})
	}
	switch {
	case d.TypeLine != 0 && !d.HasType():
		add(d.TypeLine, lang.CodeTypeLine, "type line must be exactly one link to a document")
	case d.HasType():
		var t *lang.Document
		if rel, ok := p.specFile(d.TypePath); ok {
			t, _ = p.Spec.Document(rel)
		}
		if t == nil {
			add(d.TypeLine, lang.CodeTypeUnresolved, "no document at \"%s\"", d.TypeRaw)
			break
		}
		out = append(out, lang.CheckSections(d, t, d.TypeLine, t.Path)...)
	}
	for _, line := range d.StrayTypeLines {
		add(line, lang.CodeTypeLine, "a type line stands only on the first line after the title")
	}
	seen := map[string]bool{}
	for _, r := range d.Paths {
		if seen[r.Target] || d.TypeLine != 0 && r.Line == d.TypeLine {
			continue
		}
		seen[r.Target] = true
		if !p.hasPath(r.Target) {
			add(r.Line, lang.CodeLinkUnresolved, "no file at \"%s\"", r.Raw)
		}
	}
	return out
}

// hasPath says whether a path below the project folder names a file or a
// folder there. A path that leaves the project folder names none.
func (p *Project) hasPath(target string) bool {
	return target == "." || target != ".." && !strings.HasPrefix(target, "../") && !strings.HasPrefix(target, "/") && exists(p.FS, target)
}

// exists says whether a path names a file or a folder of the project, each
// of its parts in the case it is written, so that a link reads the same on a
// file system that ignores case as on one that does not.
func exists(fsys fs.FS, p string) bool {
	dir := "."
	for _, part := range strings.Split(p, "/") {
		entries, err := fs.ReadDir(fsys, dir)
		if err != nil {
			return false
		}
		found := false
		for _, e := range entries {
			if e.Name() == part {
				found = true
				break
			}
		}
		if !found {
			return false
		}
		dir = path.Join(dir, part)
	}
	return true
}
