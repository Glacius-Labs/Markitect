package snapshot

import "strings"

// ChangeKind classifies an exact-path difference. Renames are Delete plus Add.
type ChangeKind uint8

const (
	Add ChangeKind = iota + 1
	Modify
	Delete
)

// Change is an immutable difference between two fixed trees. A modification
// includes mode-only changes. Absence is explicit through Before and After.
type Change struct {
	kind   ChangeKind
	path   Path
	before Entry
	after  Entry
}

func (c Change) Kind() ChangeKind { return c.kind }
func (c Change) Path() Path       { return c.path }

func (c Change) Before() (Entry, bool) { return c.before, c.kind == Modify || c.kind == Delete }
func (c Change) After() (Entry, bool)  { return c.after, c.kind == Modify || c.kind == Add }

// Compare returns caller-owned changes in exact-path byte order. It compares
// actual bytes and modes, not only digests. Equal trees return no changes.
func Compare(before, after Tree) []Change {
	var changes []Change
	i, j := 0, 0
	for i < len(before.entries) || j < len(after.entries) {
		switch {
		case j == len(after.entries) || (i < len(before.entries) && strings.Compare(before.entries[i].path.value, after.entries[j].path.value) < 0):
			entry := before.entries[i]
			changes = append(changes, Change{kind: Delete, path: entry.path, before: entry})
			i++
		case i == len(before.entries) || strings.Compare(before.entries[i].path.value, after.entries[j].path.value) > 0:
			entry := after.entries[j]
			changes = append(changes, Change{kind: Add, path: entry.path, after: entry})
			j++
		default:
			old, next := before.entries[i], after.entries[j]
			if old != next {
				changes = append(changes, Change{kind: Modify, path: old.path, before: old, after: next})
			}
			i++
			j++
		}
	}
	return changes
}
