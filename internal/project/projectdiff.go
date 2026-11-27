package project

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// How a file differs between two states of a project.
const (
	FileNew     = "new"
	FileGone    = "gone"
	FileChanged = "changed"
)

// FileChange is a file, by its path below the project folder, and how it
// differs.
type FileChange struct{ Path, Kind string }

// ModuleChange is a module whose model, its files, or both changed. Module is
// the folder below the model folder, lang.ModelFolderModule for the model folder itself.
type ModuleChange struct {
	Module string
	// Model is whether a document directly in the module is new, gone or changed.
	Model bool
	// Files is whether a file the module names is new, gone or changed; the lists
	// hold those files, each sorted by path.
	Files              bool
	New, Gone, Changed []string
}

// Changes lists the changed files of the module in the order of their paths.
func (m ModuleChange) Changes() []FileChange {
	var out []FileChange
	for kind, list := range map[string][]string{FileNew: m.New, FileGone: m.Gone, FileChanged: m.Changed} {
		for _, p := range list {
			out = append(out, FileChange{p, kind})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// ProjectDiff is what differs between two projects: their models, each module
// whose model or files changed, and the changed files that no module names.
type ProjectDiff struct {
	Model *ModelDiff
	// Modules are the modules of either project where something changed, the
	// model folder first and then by path.
	Modules []ModuleChange
	// Unassigned are the files that no module names, in the state they belong to,
	// that are new, gone or changed, sorted by path.
	Unassigned []FileChange
	// UnassignedTotal is how many files of the newer project no module names.
	UnassignedTotal int
}

// Differs says whether the projects differ, in a document or in a file.
func (d *ProjectDiff) Differs() bool {
	return d.Model.Differs() || len(d.Modules) > 0 || len(d.Unassigned) > 0
}

// side is one state of a project: its files that are looked at, and the
// modules that name each of them.
type side struct {
	files fileTable
	// owners maps a file to the modules that name it; a module that has a module
	// below it that names the file too does not count. A file in no module is
	// absent.
	owners map[string][]string
}

// within reports whether the module child lies within the module parent.
func within(child, parent string) bool {
	if parent == lang.ModelFolderModule {
		return child != lang.ModelFolderModule
	}
	return strings.HasPrefix(child, parent+"/")
}

func newSide(p *Project, ig *Ignore, inside Folder) (*side, error) {
	patterns := map[string][]lang.PathPattern{}
	var modules []string
	for _, m := range p.Model.Modules() {
		if list := p.Model.RealizedIn(m); len(list) > 0 {
			patterns[m] = list
			modules = append(modules, m)
		}
	}
	files, err := listFiles(p.FS, ig, Folder{Path: p.ModelFolder}, inside)
	if err != nil {
		return nil, err
	}
	s := &side{files: files, owners: map[string][]string{}}
	for file := range files {
		var naming []string
		for _, m := range modules {
			for _, pat := range patterns[m] {
				if pat.Names(file) {
					naming = append(naming, m)
					break
				}
			}
		}
		for _, m := range naming {
			lowest := true
			for _, other := range naming {
				if within(other, m) {
					lowest = false
					break
				}
			}
			if lowest {
				s.owners[file] = append(s.owners[file], m)
			}
		}
	}
	return s, nil
}

// DiffProjects compares an older project with a newer one: their models, and,
// from the files of both that the ignore file does not leave out, which of them
// is new, gone or changed, byte for byte and without loading either in whole.
// A file is new, gone or changed by itself, whatever the READMEs say. A new or
// changed file belongs to the modules that name it in the newer project, a gone
// file to those that named it in the older one, a module below another taking
// the file from the one above in that state. A module whose model or whose files
// changed is a change of the project; a changed file that no module names, in
// the state it belongs to, is listed apart. oldInside is the folder of the older
// project below the newer one, newInside that of the newer below the older, the
// zero Folder where the project does not lie inside the other: a copy kept inside a project
// is left out of that project's files.
func DiffProjects(older, newer *Project, ig *Ignore, oldInside, newInside Folder) (*ProjectDiff, error) {
	d := &ProjectDiff{Model: DiffModels(older, newer)}
	before, err := newSide(older, ig, newInside)
	if err != nil {
		return nil, fmt.Errorf("the old project: %w", err)
	}
	after, err := newSide(newer, ig, oldInside)
	if err != nil {
		return nil, fmt.Errorf("the new project: %w", err)
	}

	cmp := newComparer()
	// What happened to each file, by itself.
	paths := make([]string, 0, len(before.files)+len(after.files))
	for file := range before.files {
		paths = append(paths, file)
	}
	for file := range after.files {
		if _, ok := before.files[file]; !ok {
			paths = append(paths, file)
		}
	}
	sort.Strings(paths)
	byModule := map[string]*ModuleChange{}
	assign := func(module, file, kind string) {
		c := byModule[module]
		if c == nil {
			c = &ModuleChange{Module: module}
			byModule[module] = c
		}
		switch kind {
		case FileNew:
			c.New = append(c.New, file)
		case FileGone:
			c.Gone = append(c.Gone, file)
		default:
			c.Changed = append(c.Changed, file)
		}
	}
	for _, file := range paths {
		oldSize, inOld := before.files[file]
		newSize, inNew := after.files[file]
		kind, owners := FileNew, after.owners[file]
		switch {
		case !inNew:
			kind, owners = FileGone, before.owners[file]
		case inOld:
			same, err := cmp.same(older.FS, newer.FS, file, oldSize, newSize)
			if err != nil {
				return nil, err
			}
			if same {
				continue
			}
			kind = FileChanged
		}
		if len(owners) == 0 {
			d.Unassigned = append(d.Unassigned, FileChange{file, kind})
		}
		for _, m := range owners {
			assign(m, file, kind)
		}
	}
	for file := range after.files {
		if len(after.owners[file]) == 0 {
			d.UnassignedTotal++
		}
	}

	modelDirs := map[string]bool{}
	for _, list := range [][]string{d.Model.New, d.Model.Gone} {
		for _, p := range list {
			modelDirs[path.Dir(p)] = true
		}
	}
	for _, c := range d.Model.Changed {
		modelDirs[path.Dir(c.Doc)] = true
	}
	moduleSet := map[string]bool{}
	for m := range modelDirs {
		moduleSet[m] = true
	}
	for m := range byModule {
		moduleSet[m] = true
	}
	modules := make([]string, 0, len(moduleSet))
	for m := range moduleSet {
		modules = append(modules, m)
	}
	lang.SortModules(modules)
	for _, m := range modules {
		c := byModule[m]
		if c == nil {
			c = &ModuleChange{Module: m}
		}
		c.Model = modelDirs[m]
		c.Files = len(c.New)+len(c.Gone)+len(c.Changed) > 0
		d.Modules = append(d.Modules, *c)
	}
	return d, nil
}
