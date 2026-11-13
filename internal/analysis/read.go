package analysis

import (
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// The reading view of doc and the Kinds of list: the spec as one text, and
// what of each Kind it holds.

// ReadEntry is one Definition of the reading view.
type ReadEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
	// Folder is the folder it is read under: its own folder, and for a thing
	// with parts the folder its own folder lies in. "" is the spec folder.
	Folder string `json:"folder"`
	// Whole is the Definition it is a part of, if any, and Depth how many
	// wholes lie above it: 0 for what is no part.
	Whole string `json:"whole,omitempty"`
	Depth int    `json:"depth"`
	// Text is its text without front matter, and Shown the same with each
	// link shown by its label or by the title of what it names.
	Text  string `json:"text"`
	Shown string `json:"-"`
}

// Read returns the Definitions at or below a folder of the spec in reading
// order, the whole spec for the spec folder itself. In each folder the
// Definition that carries the folder's name comes first, then its files and
// its things with parts by name, each thing followed by its parts, then the
// folders below. It is empty for a folder that holds none.
func Read(p *project.Project, folder string) []ReadEntry {
	var ids []string
	for _, id := range p.Spec.OwnDefinitions() {
		if folder == spec.RootPackage || strings.HasPrefix(id, folder+"/") {
			ids = append(ids, id)
		}
	}
	return readEntries(p.Spec, ids, 0)
}

// ReadBound returns what doc --read --bound prints around the Definitions of
// a folder, entries as Read returns them, so that one text holds all an
// audit of the folder reads. Bound are the Definitions that bind one of them
// from outside the folder without a link, what doc lists under Bound by;
// Linked are the Definitions outside the folder that one of them links, but
// Kinds and what Libraries hold. Each comes with its parts, as doc shows it,
// and in reading order; no Definition comes twice, nor one of the folder.
// Depth counts the wholes between a part and the Definition it comes with.
func ReadBound(p *project.Project, entries []ReadEntry) (bound, linked []ReadEntry) {
	sp := p.Spec
	printed, listed := map[string]bool{}, map[string]bool{}
	for _, e := range entries {
		printed[e.ID], listed[e.ID] = true, true
	}
	binding := sp.Binding()
	var binds, links []string
	for _, e := range entries {
		for _, id := range boundBy(sp, binding, sp.Definitions[e.ID]) {
			if !listed[id] {
				listed[id] = true
				binds = append(binds, id)
			}
		}
	}
	for _, e := range entries {
		for _, l := range sp.Definitions[e.ID].Links {
			t := sp.Definitions[l.Target]
			if l.Kind || l.Package || t == nil || !t.Own() || t.Kind == spec.CoreKind || listed[l.Target] {
				continue
			}
			listed[l.Target] = true
			links = append(links, l.Target)
		}
	}
	return withParts(sp, binds, printed), withParts(sp, links, printed)
}

// withParts returns Definitions in reading order, each followed by its
// parts, but none that printed holds already and none that is a part of
// another of them, since that one brings it; it adds what it returns to
// printed.
func withParts(sp *spec.Spec, ids []string, printed map[string]bool) []ReadEntry {
	out := []ReadEntry{}
	for _, root := range readOrder(sp, ids) {
		inOther := printed[root]
		for _, other := range ids {
			inOther = inOther || sp.IsPartOf(root, other)
		}
		if inOther {
			continue
		}
		printed[root] = true
		unit := []string{root}
		for _, part := range sp.Parts(root) {
			if !printed[part] {
				printed[part] = true
				unit = append(unit, part)
			}
		}
		base := 0
		for w := sp.Definitions[sp.Definitions[root].Whole]; w != nil; w = sp.Definitions[w.Whole] {
			base++
		}
		out = append(out, readEntries(sp, unit, base)...)
	}
	return out
}

// readOrder sorts Definitions in reading order, by readKey.
func readOrder(sp *spec.Spec, ids []string) []string {
	keys := map[string][]string{}
	for _, id := range ids {
		keys[id] = readKey(sp, id)
	}
	out := append([]string{}, ids...)
	sort.Slice(out, func(i, j int) bool {
		a, b := keys[out[i]], keys[out[j]]
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
	return out
}

// readEntries returns the entries of Definitions in reading order, each
// Depth less base.
func readEntries(sp *spec.Spec, ids []string, base int) []ReadEntry {
	out := []ReadEntry{}
	for _, id := range readOrder(sp, ids) {
		d := sp.Definitions[id]
		e := ReadEntry{ID: id, Title: d.Title, Kind: d.Kind, Folder: spec.FolderOf(id), Whole: d.Whole, Text: body(d.Data), Shown: shown(sp, d.Data), Depth: -base}
		if _, thing := sp.Thing(e.Folder); thing && id == spec.AnchorOf(e.Folder) {
			e.Folder = spec.FolderOf(e.Folder)
		}
		for w := sp.Definitions[d.Whole]; w != nil; w = sp.Definitions[w.Whole] {
			e.Depth++
		}
		out = append(out, e)
	}
	return out
}

// readKey is what a Definition sorts by for reading: its path, each part
// of it ranked. The file that carries its folder's name comes first in the
// folder, then the other files together with the folders that are a thing
// with parts, since such a thing reads like a file of the folder it lies in,
// and last the folders that are a namespace or a Package.
func readKey(sp *spec.Spec, id string) []string {
	parts := strings.Split(id, "/")
	key := make([]string, len(parts))
	for i, part := range parts {
		rank := "2"
		switch {
		case i == len(parts)-1 && i > 0 && part == parts[i-1]:
			rank = "0"
		case i == len(parts)-1:
			rank = "1"
		default:
			if _, thing := sp.Thing(strings.Join(parts[:i+1], "/")); thing {
				rank = "1"
			}
		}
		key[i] = rank + part
	}
	return key
}

// KindCount is one Kind with the Definitions of the spec that are of it.
type KindCount struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	// Definitions counts the project's Definitions of the Kind, and Folders
	// names the top-level folders of the spec they lie in, "." for the spec
	// folder itself.
	Definitions int      `json:"definitions"`
	Folders     []string `json:"folders"`
}

// Kinds returns every Kind the spec can use, its own, those of its Libraries
// and Package, each with how many Definitions the spec has of it and where:
// what the spec describes so far, and what it has nothing of.
func Kinds(p *project.Project) []KindCount {
	sp := p.Spec
	counts := map[string]*KindCount{}
	for id, d := range sp.Definitions {
		if d.Kind == spec.CoreKind || id == spec.CorePackage {
			counts[id] = &KindCount{Kind: id, Title: d.Title, Folders: []string{}}
		}
	}
	for _, id := range sp.OwnDefinitions() {
		c := counts[sp.Definitions[id].Kind]
		if c == nil {
			continue
		}
		c.Definitions++
		top, _, below := strings.Cut(id, "/")
		if !below {
			top = "."
		}
		found := false
		for _, f := range c.Folders {
			found = found || f == top
		}
		if !found {
			c.Folders = append(c.Folders, top)
		}
	}
	out := make([]KindCount, 0, len(counts))
	for _, c := range counts {
		sort.Strings(c.Folders)
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out
}
