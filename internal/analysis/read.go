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
	// Folder is the folder it lies in, "" for the spec folder itself.
	Folder string `json:"folder"`
	// Whole is the Definition it is a part of, if any.
	Whole string `json:"whole,omitempty"`
	// Text is its text without front matter, and Shown the same with each
	// link shown by its label or by the title of what it names.
	Text  string `json:"text"`
	Shown string `json:"-"`
}

// Read returns the Definitions at or below a folder of the spec in reading
// order, the whole spec for the spec folder itself: in each folder the
// Definition that carries the folder's name first, then the others by name,
// then the folders below. It is empty for a folder that holds none.
func Read(p *project.Project, folder string) []ReadEntry {
	var ids []string
	for _, id := range p.Spec.OwnDefinitions() {
		if folder == spec.RootPackage || strings.HasPrefix(id, folder+"/") {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return readsBefore(ids[i], ids[j]) })
	out := []ReadEntry{}
	for _, id := range ids {
		d := p.Spec.Definitions[id]
		out = append(out, ReadEntry{ID: id, Title: d.Title, Kind: d.Kind, Folder: spec.FolderOf(id), Whole: d.Whole, Text: body(d.Data), Shown: shown(p.Spec, d.Data)})
	}
	return out
}

// readsBefore orders two Definition IDs for reading: where their paths part,
// a file comes before a folder, the file that carries its folder's name
// before the other files, and names otherwise by the alphabet.
func readsBefore(a, b string) bool {
	as, bs := strings.Split(a, "/"), strings.Split(b, "/")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] == bs[i] {
			continue
		}
		aFile, bFile := i == len(as)-1, i == len(bs)-1
		if aFile != bFile {
			return aFile
		}
		if aFile && i > 0 {
			if as[i] == as[i-1] {
				return true
			}
			if bs[i] == bs[i-1] {
				return false
			}
		}
		return as[i] < bs[i]
	}
	return len(as) < len(bs)
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
