package edit

import (
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// NewDocument computes a new document of the type at typePath at pathArg, a
// document name as on the command line: its title, by default the file name,
// the type line with a relative link to the type, and an empty section for
// each section the type requires. It returns the document's path.
func NewDocument(spec *lang.Spec, fsys fs.FS, typePath, pathArg, title string) (string, Changes, error) {
	t, ok := spec.Document(typePath)
	if !ok {
		return "", Changes{}, refuse("unknown path: " + typePath)
	}
	p, ok := DocumentPath(pathArg)
	if !ok {
		return "", Changes{}, refuse("invalid path: " + strconv.Quote(pathArg))
	}
	if _, taken := spec.Document(p); taken || exists(fsys, p) {
		return "", Changes{}, refuse(p + ": already exists")
	}
	if title == "" {
		base := path.Base(p)
		title = base[:len(base)-len(".md")]
	}
	name := t.Title
	if name == "" {
		name = t.Path
	}
	var b strings.Builder
	b.WriteString("# " + title + "\n\nType: [" + name + "](" + quote(rel(t.Path, dir(p))) + ")\n\n")
	for _, s := range t.Declared() {
		if !s.Optional {
			b.WriteString("## " + s.Name + "\n\n")
		}
	}
	return p, Changes{Writes: []Write{{Path: p, Data: []byte(b.String()), Create: true}}}, nil
}
