package project

import "path"

// SpecDiff is what differs between the specs of two projects, each list
// sorted by path. Documents are named by their path below the spec folder.
type SpecDiff struct {
	New, Gone []string
	// Changed are the documents at the same path whose text differs, line
	// ends and a byte order mark apart.
	Changed []ChangedDoc
}

// ChangedDoc is a document whose text changed, and the diff, with its file
// below each project folder as the headers.
type ChangedDoc struct{ Doc, Diff string }

// Differs says whether the specs differ.
func (c *SpecDiff) Differs() bool {
	return len(c.New)+len(c.Gone)+len(c.Changed) > 0
}

// DiffSpecs compares the spec of an older project with that of a newer one. The
// documents of a spec are sorted by path, and so is each list.
func DiffSpecs(older, newer *Project) *SpecDiff {
	c := &SpecDiff{}
	for _, d := range newer.Spec.Documents {
		o, ok := older.Spec.Document(d.Path)
		if !ok {
			c.New = append(c.New, d.Path)
			continue
		}
		if diff := Diff(path.Join(older.SpecFolder, o.Path), path.Join(newer.SpecFolder, d.Path), o.Source, d.Source); diff != "" {
			c.Changed = append(c.Changed, ChangedDoc{d.Path, diff})
		}
	}
	for _, d := range older.Spec.Documents {
		if _, ok := newer.Spec.Document(d.Path); !ok {
			c.Gone = append(c.Gone, d.Path)
		}
	}
	return c
}
