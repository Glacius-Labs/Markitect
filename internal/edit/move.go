package edit

import (
	"bytes"
	"io/fs"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// Moved is what MoveDocument computed: the document's old and new path, the
// documents it changes, at their final paths in the order of their old paths,
// and the change set.
type Moved struct {
	From, To string
	Changed  []string
	Changes  Changes
}

// MoveDocument moves the document oldArg to newArg, both document names as on
// the command line. Every link to it, type lines and link reference
// definitions included, points to the new path. When it changes folder, every
// relative destination in it, of links, images, definitions and HTML href and
// src values, is rewritten to reach the same file; an HTML value it cannot
// read refuses the move. Link text, the #section and every other byte stay as
// they are.
func MoveDocument(spec *lang.Spec, fsys fs.FS, oldArg, newArg string) (Moved, error) {
	old, ok := DocumentPath(oldArg)
	if _, known := spec.Document(old); !ok || !known {
		return Moved{}, refuse("unknown path: " + oldArg)
	}
	to, ok := DocumentPath(newArg)
	if !ok {
		return Moved{}, refuse("invalid path: " + strconv.Quote(newArg))
	}
	if _, taken := spec.Document(to); taken || exists(fsys, to) {
		return Moved{}, refuse(to + ": already exists")
	}
	m := Moved{From: old, To: to, Changes: Changes{Moves: []Move{{From: old, To: to}}}}
	for _, d := range spec.Documents {
		src, dst := d.Path, d.Path
		if src == old {
			dst = to
		}
		target := func(r lang.Ref) string {
			if r.Target == old {
				return to
			}
			return r.Target
		}
		// The moved document rewrites every relative destination, to any file,
		// when it changes folder; the others only their links to it.
		refs, pick := d.Refs, func(d *lang.Document) []lang.Ref { return d.Refs }
		if src == old {
			refs, pick = d.Paths, func(d *lang.Document) []lang.Ref { return d.Paths }
			if dir(src) != dir(dst) && len(d.UnplacedHTML) > 0 {
				return Moved{}, refuse(src + ": cannot read the HTML link at line " + strconv.Itoa(d.UnplacedHTML[0]))
			}
		}
		fresh, err := rewrite(d, refs, func(r lang.Ref) (string, bool) {
			if r.Target != old && dir(src) == dir(dst) || strings.HasPrefix(r.Target, "/") {
				return "", false
			}
			return rel(target(r), dir(dst)), true
		})
		if err != nil {
			return Moved{}, err
		}
		if bytes.Equal(fresh, d.Source) {
			continue
		}
		if err := verify(dst, fresh, refs, pick, target); err != nil {
			return Moved{}, err
		}
		m.Changed = append(m.Changed, dst)
		m.Changes.Writes = append(m.Changes.Writes, Write{Path: dst, Data: fresh})
	}
	return m, nil
}

// verify reads the rewritten document again and checks that each of its
// links reaches the target it should.
func verify(p string, fresh []byte, before []lang.Ref, pick func(*lang.Document) []lang.Ref, target func(lang.Ref) string) error {
	after := pick(lang.Parse(p, fresh))
	if len(after) != len(before) {
		return refuse(p + ": rewriting its links changed how many there are")
	}
	for i, r := range before {
		if want := target(r); !strings.HasPrefix(want, "/") && after[i].Target != want {
			return refuse(p + ": the link at line " + strconv.Itoa(r.Line) + " would not reach " + want)
		}
	}
	return nil
}
