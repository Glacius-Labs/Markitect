package state

import (
	"fmt"
	"sort"
	"testing/fstest"

	"github.com/Glacius-Labs/Markitect/internal/edit"
	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// moveCheck is what the move events a step added say, checked against the
// files: the events no move made, and the rows with source move that the
// moves carry, each by its line as the tool writes it.
type moveCheck struct {
	problems []string
	rows     map[string]bool
}

// checkMoves replays the move events a step added, in their order, on the
// spec before the step, as move computes them, and carries the rows of the
// state before through each, as move carries them. A move event is made by
// a move when its document is one before it and its new path none, the two
// differ, the document is gone after the step unless a later move brought
// it back, and every document it names as rewritten, its new path
// included, has after the step what the replay makes of it, line ends as
// hashing reads them. Any other change of the spec in the step is an
// ordinary change of the spec, and gets no row with source move. When an
// event is no move made, no row with source move stands.
func checkMoves(role string, before, after *Project) (moveCheck, error) {
	mc := moveCheck{rows: map[string]bool{}}
	events, err := addedEvents(before.FS, after.FS)
	if err != nil {
		return mc, err
	}
	old, _, err := canonicalRows(before.FS)
	if err != nil {
		return mc, err
	}
	var rows []Row
	for _, r := range old {
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return string(rows[i].line()) < string(rows[j].line()) })
	texts := map[string][]byte{}
	for _, d := range before.Spec.Documents {
		texts[d.Path] = d.Source
	}
	named := map[string]bool{}
	var froms []string
	deny := func(e Event, format string, args ...any) {
		mc.problems = append(mc.problems, fmt.Sprintf("%s may not add a move event that no move made: %s -> %s: %s", role, e.From, e.To, fmt.Sprintf(format, args...)))
	}
	for _, e := range events {
		if e.Type != EventMove {
			continue
		}
		_, had := texts[e.From]
		_, taken := texts[e.To]
		switch {
		case e.From == e.To:
			deny(e, "it moves a document onto itself")
			continue
		case !had:
			deny(e, "%s is no document there", e.From)
			continue
		case taken:
			deny(e, "%s is a document already", e.To)
			continue
		}
		fsys := fstest.MapFS{}
		var docs []*lang.Document
		for p, text := range texts {
			fsys[p] = &fstest.MapFile{Data: text}
			docs = append(docs, lang.Parse(p, text))
		}
		m, err := edit.MoveDocument(lang.NewSpec(docs), fsys, e.From, e.To)
		if err != nil {
			deny(e, "it cannot be replayed: %v", err)
			continue
		}
		rewritten := map[string]Text{}
		for _, w := range m.Changes.Writes {
			was := w.Path
			if was == m.To {
				was = m.From
			}
			rewritten[was] = Text{Before: texts[was], After: w.Data}
		}
		texts[m.To] = texts[m.From]
		delete(texts, m.From)
		for _, w := range m.Changes.Writes {
			texts[w.Path] = w.Data
		}
		rows = MoveRows(rows, m.From, m.To, rewritten)
		froms = append(froms, e.From)
		named[e.To] = true
		for _, p := range e.Rewritten {
			named[p] = true
		}
	}
	var paths []string
	for p := range named {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		d, ok := after.Spec.Document(p)
		text, replayed := texts[p]
		switch {
		case !ok || !replayed:
			mc.problems = append(mc.problems, fmt.Sprintf("%s may not add a move event that no move made: %s is not where the moves lead", role, p))
		case Hash(d.Source) != Hash(text):
			mc.problems = append(mc.problems, fmt.Sprintf("%s may not add a move event that no move made: %s is not what the moves make of it", role, p))
		}
	}
	for _, p := range froms {
		if _, back := texts[p]; !back {
			if _, ok := after.Spec.Document(p); ok {
				mc.problems = append(mc.problems, fmt.Sprintf("%s may not add a move event that no move made: %s is still there, and a copy is no move", role, p))
			}
		}
	}
	if len(mc.problems) > 0 {
		mc.rows = map[string]bool{}
		return mc, nil
	}
	for _, r := range rows {
		if r.Source == "move" {
			mc.rows[string(r.line())] = true
		}
	}
	return mc, nil
}
