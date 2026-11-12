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
	sp := p.Spec
	var ids []string
	keys := map[string][]string{}
	for _, id := range sp.OwnDefinitions() {
		if folder == spec.RootPackage || strings.HasPrefix(id, folder+"/") {
			ids = append(ids, id)
			keys[id] = readKey(sp, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := keys[ids[i]], keys[ids[j]]
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
	out := []ReadEntry{}
	for _, id := range ids {
		d := sp.Definitions[id]
		e := ReadEntry{ID: id, Title: d.Title, Kind: d.Kind, Folder: spec.FolderOf(id), Whole: d.Whole, Text: body(d.Data), Shown: shown(sp, d.Data)}
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
