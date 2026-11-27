package project

import "path"

// ModelDiff is what differs between the models of two projects, each list
// sorted by path. Documents are named by their path below the model folder.
type ModelDiff struct {
	New, Gone []string
	// Changed are the documents at the same path whose text differs, line
	// ends and a byte order mark apart.
	Changed []ChangedDoc
}

// ChangedDoc is a document whose text changed, and the diff, with its file
// below each project folder as the headers.
type ChangedDoc struct{ Doc, Diff string }

// Differs says whether the models differ.
func (c *ModelDiff) Differs() bool {
	return len(c.New)+len(c.Gone)+len(c.Changed) > 0
}

// DiffModels compares the model of an older project with that of a newer one. The
// documents of a model are sorted by path, and so is each list.
func DiffModels(older, newer *Project) *ModelDiff {
	c := &ModelDiff{}
	for _, d := range newer.Model.Documents {
		o, ok := older.Model.Document(d.Path)
		if !ok {
			c.New = append(c.New, d.Path)
			continue
		}
		if diff := Diff(path.Join(older.ModelFolder, o.Path), path.Join(newer.ModelFolder, d.Path), o.Source, d.Source); diff != "" {
			c.Changed = append(c.Changed, ChangedDoc{d.Path, diff})
		}
	}
	for _, d := range older.Model.Documents {
		if _, ok := newer.Model.Document(d.Path); !ok {
			c.Gone = append(c.Gone, d.Path)
		}
	}
	return c
}
