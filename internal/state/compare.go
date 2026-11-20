package state

import (
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// Snapshot is a project as compare reads it: the project with its spec, the
// state as status takes it, and the history.
type Snapshot struct {
	Project *Project
	Status  *Status
	History []Event
}

// Comparison is what differs between two projects, each list sorted by
// path. Documents are named by their path below the spec folder, files by
// their path below the project folder.
type Comparison struct {
	New, Gone []string
	// Moved are the documents the newer project's history moved from a path
	// of the older one, with the diff of their text.
	Moved []MovedDoc
	// Rewritten are the documents at the same path whose text differs only
	// by the links the newer project's moves rewrote.
	Rewritten []string
	// Changed are the documents at the same path whose text differs.
	Changed []ChangedDoc
	// Suspect are the documents whose own text is the same, while a document
	// they link to, one that links to them, or their type changed, is new,
	// is gone or moved, or a file of the spec folder they link changed.
	Suspect []SuspectDoc
	// Verdicts are the documents of the newer project whose verdict or
	// artifacts differ from the older one's.
	Verdicts []VerdictChange
	// Unexplained are the files unexplained in the newer project that were
	// not in the older one.
	Unexplained []string
}

// MovedDoc is a document the history moved, and the diff of its text.
// Mechanical says that its text differs only by the links the moves
// rewrote; it then has no diff.
type MovedDoc struct {
	From, To, Diff string
	Mechanical     bool
}

// ChangedDoc is a document whose text changed, and the diff.
type ChangedDoc struct{ Doc, Diff string }

// SuspectDoc is a document that did not change itself, and the documents
// around it that did, as "link:PATH" or "type:PATH".
type SuspectDoc struct {
	Doc     string
	Because []string
}

// VerdictChange is a document's verdict and artifacts in both projects; a
// verdict is "" without a row.
type VerdictChange struct {
	Doc                        string
	OldVerdict, NewVerdict     string
	OldArtifacts, NewArtifacts []string
}

// Differs says whether the comparison found anything.
func (c *Comparison) Differs() bool {
	return len(c.New)+len(c.Gone)+len(c.Moved)+len(c.Rewritten)+len(c.Changed)+len(c.Suspect)+len(c.Verdicts)+len(c.Unexplained) > 0
}

// Compare compares an older project with a newer one. replayed holds, by
// their paths in the newer project, the texts the newer project's moves
// alone make of the older project's documents they rewrote: a document
// whose text is that is rewritten, not changed, and makes no neighbour
// suspect.
func Compare(older, newer *Snapshot, replayed map[string][]byte) *Comparison {
	c := &Comparison{}
	oldSpec, newSpec := older.Project.Spec, newer.Project.Spec
	// Where each document of the older project went: its own path, or where
	// the newer history's moves led it.
	moves := map[string]string{}
	for _, e := range newer.History {
		if e.Type == EventMove {
			moves[e.From] = e.To
		}
	}
	movedTo := map[string]string{}
	for _, d := range oldSpec.Documents {
		if _, still := newSpec.Document(d.Path); still {
			continue
		}
		at, seen := d.Path, map[string]bool{}
		for !seen[at] {
			seen[at] = true
			next, ok := moves[at]
			if !ok {
				break
			}
			at = next
		}
		if _, there := newSpec.Document(at); there && at != d.Path {
			if _, before := oldSpec.Document(at); !before {
				movedTo[d.Path] = at
			}
		}
	}
	movedFrom := map[string]string{}
	for from, to := range movedTo {
		movedFrom[to] = from
	}
	mechanical := func(n *lang.Document) bool {
		text, ok := replayed[n.Path]
		return ok && Hash(text) == Hash(n.Source)
	}
	touched := map[string]bool{}
	for _, d := range oldSpec.Documents {
		to, moved := movedTo[d.Path]
		n, same := newSpec.Document(d.Path)
		switch {
		case moved:
			nd, _ := newSpec.Document(to)
			if mechanical(nd) {
				c.Moved = append(c.Moved, MovedDoc{From: d.Path, To: to, Mechanical: true})
				continue
			}
			c.Moved = append(c.Moved, MovedDoc{From: d.Path, To: to, Diff: textDiff(older, newer, d, nd)})
			touched[d.Path], touched[to] = true, true
		case !same:
			c.Gone = append(c.Gone, d.Path)
			touched[d.Path] = true
		case Hash(d.Source) == Hash(n.Source):
		case mechanical(n):
			c.Rewritten = append(c.Rewritten, d.Path)
		default:
			c.Changed = append(c.Changed, ChangedDoc{Doc: d.Path, Diff: textDiff(older, newer, d, n)})
			touched[d.Path] = true
		}
	}
	for _, d := range newSpec.Documents {
		if _, before := oldSpec.Document(d.Path); !before && movedFrom[d.Path] == "" {
			c.New = append(c.New, d.Path)
			touched[d.Path] = true
		}
	}
	// Suspect: unchanged documents next to a touched one, in either project.
	oldNear, newNear := neighbours(oldSpec), neighbours(newSpec)
	for _, d := range newSpec.Documents {
		if touched[d.Path] {
			continue
		}
		// A document moved without a change of its own is compared at the
		// path it had.
		from := d.Path
		if f, ok := movedFrom[d.Path]; ok {
			from = f
		}
		if _, before := oldSpec.Document(from); !before {
			continue
		}
		because := map[string]bool{}
		for _, near := range []struct {
			links map[string]map[string]string
			at    string
		}{{oldNear, from}, {newNear, d.Path}} {
			for other, kind := range near.links[near.at] {
				if touched[other] {
					because[kind+other] = true
				}
			}
		}
		if len(because) > 0 {
			c.Suspect = append(c.Suspect, SuspectDoc{Doc: d.Path, Because: sortedKeys(because)})
		}
	}
	// Linked files of the spec folder that changed make their documents
	// suspect.
	for _, d := range newSpec.Documents {
		if touched[d.Path] {
			continue
		}
		var because []string
		from := d.Path
		if f, ok := movedFrom[d.Path]; ok {
			from = f
		}
		was, is := older.Status.assets(from), newer.Status.assets(d.Path)
		for k, h := range is {
			if was[k] != h {
				because = append(because, k)
			}
		}
		for k := range was {
			if _, ok := is[k]; !ok {
				because = append(because, k)
			}
		}
		if len(because) == 0 {
			continue
		}
		found := false
		for i := range c.Suspect {
			if c.Suspect[i].Doc == d.Path {
				c.Suspect[i].Because = append(c.Suspect[i].Because, because...)
				sort.Strings(c.Suspect[i].Because)
				found = true
			}
		}
		if !found {
			sort.Strings(because)
			c.Suspect = append(c.Suspect, SuspectDoc{Doc: d.Path, Because: because})
		}
	}
	// Verdicts, for every document of the newer project.
	for _, d := range newSpec.Documents {
		from := d.Path
		if f, ok := movedFrom[d.Path]; ok {
			from = f
		}
		oldRow, _, hadRow := older.Status.Row(from)
		newRow, _, hasRow := newer.Status.Row(d.Path)
		v := VerdictChange{Doc: d.Path, OldArtifacts: []string{}, NewArtifacts: []string{}}
		if hadRow {
			v.OldVerdict, v.OldArtifacts = oldRow.Verdict, append(v.OldArtifacts, oldRow.Files()...)
		}
		if hasRow {
			v.NewVerdict, v.NewArtifacts = newRow.Verdict, append(v.NewArtifacts, newRow.Files()...)
		}
		if v.OldVerdict != v.NewVerdict || strings.Join(v.OldArtifacts, "\n") != strings.Join(v.NewArtifacts, "\n") {
			c.Verdicts = append(c.Verdicts, v)
		}
	}
	was := map[string]bool{}
	for _, u := range older.Status.Unexplained {
		was[u] = true
	}
	for _, u := range newer.Status.Unexplained {
		if !was[u] {
			c.Unexplained = append(c.Unexplained, u)
		}
	}
	sort.Strings(c.New)
	sort.Strings(c.Gone)
	sort.Strings(c.Rewritten)
	sort.Slice(c.Moved, func(i, j int) bool { return c.Moved[i].To < c.Moved[j].To })
	sort.Slice(c.Changed, func(i, j int) bool { return c.Changed[i].Doc < c.Changed[j].Doc })
	sort.Slice(c.Suspect, func(i, j int) bool { return c.Suspect[i].Doc < c.Suspect[j].Doc })
	sort.Slice(c.Verdicts, func(i, j int) bool { return c.Verdicts[i].Doc < c.Verdicts[j].Doc })
	return c
}

// assets are a document's linked files of the spec folder with their
// hashes, as status read them.
func (s *Status) assets(doc string) map[string]string {
	out := map[string]string{}
	if d, ok := s.docs[doc]; ok {
		for k, h := range d.current {
			if strings.HasPrefix(k, InputAsset) {
				out[k] = h
			}
		}
	}
	return out
}

// textDiff is the diff of a document's text in two projects, with its file
// below each project folder as the headers.
func textDiff(older, newer *Snapshot, o, n *lang.Document) string {
	return Diff(path.Join(older.Project.SpecFolder, o.Path), path.Join(newer.Project.SpecFolder, n.Path), o.Source, n.Source)
}

// neighbours maps each document to the documents it links to, those that
// link to it, as "link:", and its type, as "type:".
func neighbours(spec *lang.Spec) map[string]map[string]string {
	out := map[string]map[string]string{}
	add := func(a, b, kind string) {
		if out[a] == nil {
			out[a] = map[string]string{}
		}
		if _, ok := out[a][b]; !ok || kind == "type:" {
			out[a][b] = kind
		}
	}
	for _, d := range spec.Documents {
		if d.HasType() {
			add(d.Path, d.TypePath, "type:")
		}
		for _, l := range d.Links {
			if l.Target != d.Path {
				add(d.Path, l.Target, "link:")
				add(l.Target, d.Path, "link:")
			}
		}
	}
	return out
}

func sortedKeys(set map[string]bool) []string {
	var out []string
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
