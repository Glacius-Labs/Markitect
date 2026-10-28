package analysis

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

// Change kinds of Definitions and files.
const (
	Added    = "added"
	Modified = "modified"
	Removed  = "removed"
)

// Change is one changed Definition or file between the base and the current
// state.
type Change struct {
	Path   string `json:"path"` // Definition ID or repository file
	Change string `json:"change"`
}

// Impact is what a change reaches: Definitions, files and Packages, and each
// working Manager's checklist.
type Impact struct {
	Full        bool     `json:"full"`
	Definitions []Change `json:"changedDefinitions"` // model and Library Definitions
	Files       []Change `json:"changedFiles"`       // repository files outside Markitect's folder
	// Reached are the model Definitions the change reaches, each with why.
	Reached []Item `json:"reached"`
	// ReachedFiles are the files the change touches or that realize a reached
	// Definition, with their owners.
	ReachedFiles []ReachedFile `json:"reachedFiles"`
	// Packages are the Managers that work: those with checklist items or
	// reached files, and their parents up to Root, which integrate.
	Packages   []string    `json:"packages"`
	Checklists []Checklist `json:"checklists"`
	// Notes are side notes, such as Findings whose Definition no longer exists.
	Notes []string `json:"notes,omitempty"`
}

// Item is one Definition with the reasons it is reached.
type Item struct {
	Definition string   `json:"definition"`
	Reasons    []string `json:"reasons"`
}

// ReachedFile is one reached repository file.
type ReachedFile struct {
	File   string   `json:"file"`
	Owner  string   `json:"owner"`
	Change string   `json:"change,omitempty"` // empty when the file is reached through a Definition
	For    []string `json:"for,omitempty"`    // the reached Definitions it realizes
}

// Checklist is one Manager's checklist (concept §6.2): its own Definitions the
// change reaches, the inherited Definitions the change reaches, every Finding
// assigned to it, and the stale realization it must clean up. A verdict on an
// item means conforming to it and to the Rules above it.
type Checklist struct {
	Package   string              `json:"package"`
	Own       []Item              `json:"own"`
	Inherited []Item              `json:"inherited"`
	Findings  []Finding           `json:"findings"`
	Stale     []realization.Stale `json:"stale"`
}

// Empty reports whether the checklist has no item.
func (c Checklist) Empty() bool {
	return len(c.Own)+len(c.Inherited)+len(c.Findings)+len(c.Stale) == 0
}

// ImpactInput names the two states and the options of one impact question.
type ImpactInput struct {
	Base     *project.Project // the state at the base commit
	Current  *project.Project // the working tree
	Changed  []Change         // repository files changed between them, outside Markitect's folder
	Findings []Finding
	Full     bool
}

// ComputeImpact derives what the change from Base to Current reaches. With
// Full every Definition is reached and every Manager works.
//
// Reach follows one step: a Definition is reached when it changed, when it
// links to (or is of) a Definition that changed, or when a file the change
// touches realizes it. Inherited items are only the Definitions above a
// Package that the change itself changed, so a changed Rule at Root becomes an
// item for every Manager below it, while a Root Definition reached only
// through a file or a link stays Root's own item.
func ComputeImpact(in ImpactInput) Impact {
	cur := in.Current.Model
	im := Impact{Full: in.Full, Files: in.Changed}
	im.Definitions = changedDefinitions(in.Base, in.Current)

	reasons := map[string][]string{}
	reach := func(id, why string) {
		if d := cur.Definitions[id]; d != nil && d.InModel() {
			reasons[id] = append(reasons[id], why)
		}
	}
	changed := map[string]bool{}
	for _, c := range im.Definitions {
		changed[c.Path] = true
		if c.Change != Removed {
			reach(c.Path, c.Change)
		}
	}
	for _, id := range cur.ModelDefinitions() {
		for _, l := range cur.Definitions[id].Links {
			if l.Target != "" && changed[l.Target] {
				verb := "links [[%s]], which changed"
				if l.Kind {
					verb = "is of Kind [[%s]], which changed"
				}
				reach(id, fmt.Sprintf(verb, l.Target))
			}
		}
	}
	for _, f := range in.Changed {
		entries := []*realization.Entry{in.Current.Registry.Entries[f.Path]}
		if f.Change == Removed && in.Base.Registry != nil {
			entries = []*realization.Entry{in.Base.Registry.Entries[f.Path]}
		}
		for _, e := range entries {
			if e == nil {
				continue
			}
			for _, id := range e.Realizes {
				reach(id, fmt.Sprintf("realized by %s, which was %s", f.Path, f.Change))
			}
		}
	}
	if in.Full {
		for _, id := range cur.ModelDefinitions() {
			reach(id, "full")
		}
	}
	for _, id := range sortedKeys(reasons) {
		im.Reached = append(im.Reached, Item{Definition: id, Reasons: dedupe(reasons[id])})
	}

	im.ReachedFiles = reachedFiles(in, reasons)

	checklists := map[string]*Checklist{}
	list := func(pkg string) *Checklist {
		if checklists[pkg] == nil {
			checklists[pkg] = &Checklist{Package: pkg, Own: []Item{}, Inherited: []Item{}, Findings: []Finding{}, Stale: []realization.Stale{}}
		}
		return checklists[pkg]
	}
	for _, item := range im.Reached {
		pkg := cur.Definitions[item.Definition].Package
		list(pkg).Own = append(list(pkg).Own, item)
	}
	for _, pkg := range cur.SortedPackages() {
		for _, above := range model.Ancestors(pkg) {
			for _, id := range cur.Packages[above].Definitions {
				switch {
				case in.Full:
					list(pkg).Inherited = append(list(pkg).Inherited, Item{Definition: id, Reasons: []string{"full"}})
				case changed[id]:
					list(pkg).Inherited = append(list(pkg).Inherited, Item{Definition: id, Reasons: dedupe(reasons[id])})
				}
			}
		}
	}
	for _, f := range in.Findings {
		d := cur.Definitions[f.Definition]
		if d == nil || !d.InModel() {
			im.Notes = append(im.Notes, fmt.Sprintf("Finding for [[%s]] (line %d): the Definition no longer exists; kept as a side note. Observed: %s Expected: %s", f.Definition, f.Line, f.Observed, f.Expected))
			continue
		}
		list(d.Package).Findings = append(list(d.Package).Findings, f)
	}
	for _, s := range in.Current.Registry.Stale {
		list(s.Package).Stale = append(list(s.Package).Stale, s)
	}

	working := map[string]bool{}
	for pkg, c := range checklists {
		if !c.Empty() {
			working[pkg] = true
		}
	}
	for _, f := range im.ReachedFiles {
		working[f.Owner] = true
	}
	if in.Full {
		for pkg := range cur.Packages {
			working[pkg] = true
		}
	}
	for pkg := range working {
		for _, above := range model.Ancestors(pkg) {
			working[above] = true
		}
	}
	for _, pkg := range sortedKeys(working) {
		if cur.Packages[pkg] == nil {
			continue
		}
		im.Packages = append(im.Packages, pkg)
		im.Checklists = append(im.Checklists, *list(pkg))
	}
	return im
}

func reachedFiles(in ImpactInput, reasons map[string][]string) []ReachedFile {
	reg := in.Current.Registry
	files := map[string]*ReachedFile{}
	for _, c := range in.Changed {
		owner, ok := reg.Owner[c.Path]
		if !ok && c.Change == Removed && in.Base.Registry != nil {
			owner, ok = in.Base.Registry.Owner[c.Path]
			if in.Current.Model.Packages[owner] == nil {
				owner = model.RootPackage
			}
		}
		if !ok {
			continue // ignored files
		}
		files[c.Path] = &ReachedFile{File: c.Path, Owner: owner, Change: c.Change}
	}
	for _, file := range sortedKeys(reg.Entries) {
		e := reg.Entries[file]
		for _, id := range e.Realizes {
			if len(reasons[id]) == 0 {
				continue
			}
			if files[file] == nil {
				files[file] = &ReachedFile{File: file, Owner: reg.Owner[file]}
			}
			files[file].For = append(files[file].For, id)
		}
	}
	var out []ReachedFile
	for _, f := range sortedKeys(files) {
		out = append(out, *files[f])
	}
	return out
}

// changedDefinitions compares the Definitions of two states, the project's
// and the Libraries'. The base state reads the standard Libraries from this
// binary, so a standard Library also counts as changed when its content
// differs from the digest the base commit's lock pinned: then all its Kinds
// changed.
func changedDefinitions(base, cur *project.Project) []Change {
	changedLibrary := map[string]bool{}
	for _, lib := range cur.Libraries {
		// Committed Libraries are compared file by file below; only a standard
		// Library is read from this binary on both sides.
		if locked, ok := base.Lock.Libraries[lib.Name]; ok && lib.Standard && locked.Digest != model.LibraryDigest(lib.Files) {
			changedLibrary[lib.Name] = true
		}
	}
	var out []Change
	for _, id := range sortedIDs(cur.Model) {
		d := cur.Model.Definitions[id]
		if d.Library == "core" {
			continue
		}
		old := base.Model.Definitions[id]
		switch {
		case old == nil:
			out = append(out, Change{Path: id, Change: Added})
		case !bytes.Equal(old.Data, d.Data) || changedLibrary[d.Library]:
			out = append(out, Change{Path: id, Change: Modified})
		}
	}
	for _, id := range sortedIDs(base.Model) {
		if base.Model.Definitions[id].Library != "core" && cur.Model.Definitions[id] == nil {
			out = append(out, Change{Path: id, Change: Removed})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
