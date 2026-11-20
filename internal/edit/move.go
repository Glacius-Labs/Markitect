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
// definitions included, points to the new path; when it changes folder, its
// own relative links are rewritten to reach the same documents. Link text, the
// #section and every other byte stay as they are.
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
		fresh, err := rewrite(d, func(r lang.Ref) (string, bool) {
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
		if err := verify(dst, fresh, d.Refs, target); err != nil {
			return Moved{}, err
		}
		m.Changed = append(m.Changed, dst)
		m.Changes.Writes = append(m.Changes.Writes, Write{Path: dst, Data: fresh})
	}
	return m, nil
}

// verify reads the rewritten document again and checks that each of its
// links reaches the target it should.
func verify(p string, fresh []byte, before []lang.Ref, target func(lang.Ref) string) error {
	after := lang.Parse(p, fresh).Refs
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
