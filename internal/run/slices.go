package run

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// SliceSize is how many items a slice of a build holds at most, unless one
// Package alone has more: about what one builder and its reviewer have been
// shown to settle in one go.
const SliceSize = 40

// Slice is one step of a build that goes slice by slice (DEC-112): a long
// list is built and verified in parts, what others rely on first.
type Slice struct {
	// Packages names the Packages whose items the slice holds, in the order of
	// their dependencies. It is empty for the last slice, which holds
	// whatever is left then.
	Packages []string `json:"packages"`
	// Through names the Packages its command lists: its own and every
	// Package before them in that order, so that what an earlier slice left,
	// or what this one puts back on the list there, is its work too.
	Through []string `json:"through"`
	// Items counts the items of its own Packages that a builder can settle.
	Items int `json:"items"`
	// Rest marks the last slice: the items that are verified last, on the
	// whole project, the errors in the realization and what an earlier slice
	// left.
	Rest bool `json:"rest,omitempty"`
	// Command is the command that lists the slice.
	Command string `json:"command"`
}

// early says whether an item belongs to the slice of its Package: it is
// not verified last, a builder can settle it, and its Definition exists.
func (it Item) early() bool {
	return !it.Last && it.State != StateGone && it.State != StateSpecProblem
}

// buildable says whether a builder can settle the item: a spec-problem
// waits for the owner.
func (it Item) buildable() bool { return it.State != StateSpecProblem }

// packageOrder returns the Packages in the order a build takes them: a
// Package after the Package above it and after every Package it depends on,
// and by name where that leaves a choice. Root comes first.
func packageOrder(sp *spec.Spec) []string {
	paths := sp.SortedPackages()
	placed := map[string]bool{}
	var order []string
	ready := func(p string) bool {
		if p != spec.RootPackage && !placed[sp.Packages[p].Parent] {
			return false
		}
		for to := range sp.Edges[p] {
			if to != p && !placed[to] {
				return false
			}
		}
		return true
	}
	for len(order) < len(paths) {
		next := ""
		found := false
		for _, p := range paths {
			if !placed[p] && ready(p) {
				next, found = p, true
				break
			}
		}
		if !found {
			// A cycle is an error of the spec, and status has no list then;
			// should one reach here, the rest follows by name.
			for _, p := range paths {
				if !placed[p] {
					next = p
					break
				}
			}
		}
		placed[next] = true
		order = append(order, next)
	}
	return order
}

// planSlices cuts the items into the slices of a build. A list of at most
// size items is one slice. A longer one goes Package by Package in the
// order of their dependencies, neighbours joined while they stay within
// size, and ends with a slice for the rest.
func planSlices(sp *spec.Spec, items []Item, size int) []Slice {
	if size <= 0 {
		size = SliceSize
	}
	total := 0
	perPackage := map[string]int{}
	for _, it := range items {
		if !it.buildable() {
			continue
		}
		total++
		if it.early() {
			perPackage[sp.PackageOf(it.Definition)]++
		}
	}
	if total == 0 {
		return nil
	}
	rest := Slice{Packages: []string{}, Through: []string{}, Items: total, Rest: true, Command: "markitect status"}
	if total <= size {
		return []Slice{rest}
	}
	var out []Slice
	// passed holds the Packages taken so far, in order: a slice's command
	// names them all.
	var passed []string
	cur := Slice{}
	flush := func() {
		if len(cur.Packages) > 0 {
			cur.Through = append([]string{}, passed...)
			cur.Command = "markitect status --slice " + strings.Join(cur.Through, ",")
			out = append(out, cur)
		}
		cur = Slice{}
	}
	for _, p := range packageOrder(sp) {
		n := perPackage[p]
		if n > 0 && cur.Items > 0 && cur.Items+n > size {
			flush()
		}
		passed = append(passed, spec.PackageName(p))
		if n == 0 {
			continue
		}
		cur.Packages = append(cur.Packages, spec.PackageName(p))
		cur.Items += n
		rest.Items -= n
	}
	flush()
	if len(out) == 0 {
		return []Slice{rest}
	}
	return append(out, rest)
}

// inSlice limits status to one slice: to the items of the named Packages
// that are not verified last, with the errors in the realization that
// whoever builds now has to settle. The other items, the Definitions that
// no file realizes yet and the lines of the record markitect cannot read
// it leaves out and counts.
func (s Status) inSlice(names []string) (Status, error) {
	if len(s.SpecErrors) > 0 {
		s.Slice = names
		return s, nil
	}
	known := map[string]bool{}
	for _, p := range s.Packages {
		known[p.Package] = true
	}
	want := map[string]bool{}
	for _, name := range names {
		if !known[name] {
			return Status{}, fmt.Errorf("%w: the spec has no Package %q; markitect status --slices names the slices, each with its command", ErrUsage, name)
		}
		want[name] = true
	}
	s.Slice = names
	var items []Item
	for _, it := range s.Items {
		if want[packageNamed(known, it.Definition)] && !it.Last && it.State != StateGone {
			items = append(items, it)
		}
	}
	s.ItemsElsewhere = len(s.Items) - len(items)
	s.Items, s.Skipped = items, nil
	// An error in the realization is the work of whoever builds now, such as
	// a file no registry page lists, and a slice is not settled with it. A
	// Definition that no file realizes yet is the one error that a later
	// slice settles by building it.
	var errs spec.Diagnostics
	for _, d := range s.Errors {
		if d.Code != project.CodeUnused {
			errs = append(errs, d)
		}
	}
	s.ErrorsElsewhere, s.Errors = len(s.Errors)-len(errs), errs
	s.UnreadableElsewhere, s.Unreadable = len(s.Unreadable), nil
	return s, nil
}

// Plan is the slices of a build as people and agents read them: each with
// the command that lists it. A spec with errors has no list to cut, so it
// prints what Text prints.
func (s Status) Plan() string {
	if len(s.SpecErrors) > 0 || len(s.Slices) == 0 {
		return s.Text()
	}
	var b strings.Builder
	if len(s.Slices) == 1 {
		n := s.Slices[0].Items
		fmt.Fprintf(&b, "The list is built in one slice (%d %s):\n\n1. %s\n", n, plural(n, "item", "items"), s.Slices[0].Command)
		return b.String()
	}
	fmt.Fprintf(&b, "The list is built in %d slices, one after the other, each built and verified before the next:\n\n", len(s.Slices))
	for i, sl := range s.Slices {
		fmt.Fprintf(&b, "%d. %s\n", i+1, sl.Command)
		if sl.Rest {
			fmt.Fprintf(&b, "   what is left then: %d %s verified last, on the whole project, with the errors in the realization and whatever an earlier slice left\n", sl.Items, plural(sl.Items, "item", "items"))
			continue
		}
		fmt.Fprintf(&b, "   %d %s of %s", sl.Items, plural(sl.Items, "item", "items"), packagesText(sl.Packages))
		if i > 0 {
			b.WriteString(", and whatever of the Packages before is on the list again")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// packagesText names Packages in a sentence.
func packagesText(names []string) string {
	switch len(names) {
	case 1:
		return "Package " + names[0]
	case 2:
		return "the Packages " + names[0] + " and " + names[1]
	}
	return "the Packages " + strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// SliceNames reads the value of --slice: Package names separated by commas.
func SliceNames(value string) []string {
	var out []string
	for _, name := range strings.Split(value, ",") {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, name)
		}
	}
	return out
}
