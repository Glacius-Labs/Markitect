package project

import "path"

// Comparison is what differs between the specs of two projects, each list
// sorted by path. Documents are named by their path below the spec folder.
type Comparison struct {
	New, Gone []string
	// Changed are the documents at the same path whose text differs, line
	// ends and a byte order mark apart.
	Changed []ChangedDoc
}

// ChangedDoc is a document whose text changed, and the diff, with its file
// below each project folder as the headers.
type ChangedDoc struct{ Doc, Diff string }

// Differs says whether the comparison found anything.
func (c *Comparison) Differs() bool {
	return len(c.New)+len(c.Gone)+len(c.Changed) > 0
}

// Compare compares the spec of an older project with that of a newer one. The
// documents of a spec are sorted by path, and so is each list.
func Compare(older, newer *Project) *Comparison {
	c := &Comparison{}
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
