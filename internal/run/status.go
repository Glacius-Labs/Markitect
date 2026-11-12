package run

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// States of an item of status.
const (
	// StateNew: the Definition has no record.
	StateNew = "new"
	// StateChanged: an input changed since its record.
	StateChanged = "changed"
	// StateFail and StateSpecProblem: recorded so at the inputs as they are.
	StateFail        = VerdictFail
	StateSpecProblem = VerdictSpecProblem
	// StateGone: the Definition is gone from the spec; its line names the files
	// that realized it.
	StateGone = "gone"
	// StateConflict: two lines match the inputs with different verdicts, as after a
	// merge.
	StateConflict = "conflict"
	// StateOK and StateSkip: verified at the inputs as they are; only status --all
	// lists them.
	StateOK   = VerdictOK
	StateSkip = VerdictSkip
)

// Item is one unit to verify, with what changed and where to look: a
// Definition that is no part of another, with the parts it covers.
type Item struct {
	Definition string `json:"definition"`
	Title      string `json:"title,omitempty"`
	// Parts are the parts the item covers, verified with it as one.
	Parts       []string `json:"parts,omitempty"`
	State       string   `json:"state"`
	Fingerprint string   `json:"fingerprint"`
	Changes     []Change `json:"changes,omitempty"`
	// Verdict and Why are those of the line the item rests on, if any.
	Verdict string `json:"verdict,omitempty"`
	Why     string `json:"why,omitempty"`
	// Verdicts are the different verdicts of a conflict.
	Verdicts []string `json:"verdicts,omitempty"`
	Look     []string `json:"look,omitempty"`
	// Last says that the item is built and verified last, on the whole
	// project: its Kind declares so, or that no file realizes it.
	Last bool `json:"last,omitempty"`
}

// Status is what is to verify in a project folder.
type Status struct {
	// SpecErrors stop the list: there is none until the spec compiles.
	SpecErrors spec.Diagnostics
	Items      []Item
	// Errors are the errors in the realization: an unlisted file, a stale registry
	// row, an unused Definition.
	Errors spec.Diagnostics
	// Skipped are the Definitions recorded skip at the inputs as they are: notes,
	// which do not count.
	Skipped []Item
	// Unreadable are the lines of the record markitect cannot read.
	Unreadable []int
	// Questions says that the architect's questions for the owner wait in
	// project.QuestionsFile (DEC-100). They are a note: they do not count, and a
	// list limited to a Package leaves them out, since they belong to none.
	Questions bool
	// Definitions counts the Definitions of the spec, and Units what is
	// verified as one: a thing with its parts is one unit.
	Definitions int
	Units       int
	// Packages are the spec's Packages with what each has on the list and the
	// Packages it depends on, sorted by name: what a driver needs to plan work
	// per Package in the order of their dependencies (DEC-103).
	Packages []PackageItems
	// Package names the one Package the list is limited to, if any: Items,
	// Skipped and Definitions are then that Package's, and Errors and Unreadable
	// are empty.
	Package string
	// ErrorsElsewhere and UnreadableElsewhere count what a list limited to a
	// Package leaves out, since it belongs to no Package: the errors in the
	// realization and the lines of the record markitect cannot read.
	ErrorsElsewhere, UnreadableElsewhere int
	// Slices are the slices a build of the list goes in, in order (DEC-112);
	// empty when no builder can settle anything.
	Slices []Slice
	// Slice names the Packages of the one slice the list is limited to, if
	// any, and ItemsElsewhere counts the items that list leaves out.
	Slice          []string
	ItemsElsewhere int
	// Binding names, for the items as listed, what binds them without a link:
	// for each Package from which at least one listed item is bound, the
	// Definitions that bind it, Root first, then by name. Empty when no listed
	// item is bound.
	Binding []Binding
	// boundBy holds the Packages that bind each item, and binds the Definitions
	// that bind each Package, whatever is listed; Binding is cut from them for
	// the list as it is printed.
	boundBy map[string][]string
	binds   map[string][]string
}

// Binding is a Package with the Definitions that bind it and what lies in it
// and below it (DEC-113).
type Binding struct {
	Package     string   `json:"package"`
	Definitions []string `json:"definitions"`
}

// PackageItems is one Package with the items of status whose Definition
// belongs to it, not to a Package below it.
type PackageItems struct {
	// Package is the Package's name: its path, or Root.
	Package string `json:"package"`
	// Definitions counts the Definitions that belong to it, and Units what
	// of them is verified as one.
	Definitions int `json:"definitions"`
	Units       int `json:"units"`
	// Items counts its items, and SpecProblems those of them recorded
	// spec-problem, which wait for the owner.
	Items        int `json:"items"`
	SpecProblems int `json:"spec_problems"`
	// DependsOn names the Packages it depends on, as list --deps has them: a
	// link to an ancestor is implicit and counts not, so no Package depends on
	// Root.
	DependsOn []string `json:"depends_on"`
}

// Clean says whether nothing is to verify: the project compiles and every
// Definition is verified at its current inputs. Notes do not count. For a list
// limited to a Package it says so of that Package's Definitions.
func (s Status) Clean() bool {
	return len(s.SpecErrors) == 0 && len(s.Items) == 0 && len(s.Errors) == 0 && len(s.Unreadable) == 0
}

// ForTheOwner says that something is to verify and all of it waits for the
// owner: every item is a spec-problem recorded at the inputs as they are, and
// nothing else is left, no error, no unreadable line and no item that changed,
// is gone or is in conflict. No builder settles that, so a driver can bring
// the work back. For a slice it says so of the slice: the errors in the
// realization and the lines of the record it leaves out are the last
// slice's, and do not count.
func (s Status) ForTheOwner() bool {
	if len(s.Items) == 0 || len(s.SpecErrors) > 0 || len(s.Errors) > 0 || len(s.Unreadable) > 0 {
		return false
	}
	if len(s.Slice) == 0 && s.ErrorsElsewhere+s.UnreadableElsewhere > 0 {
		return false
	}
	for _, it := range s.Items {
		if it.State != StateSpecProblem {
			return false
		}
	}
	return true
}

// StatusRequest is one call of status.
type StatusRequest struct {
	Dir string
	// All lists every Definition, to verify everything again.
	All bool
	// Package limits the list to the items whose Definition belongs to the
	// Package of that name, as a strand of a parallel run works on them.
	Package string
	// Slice limits the list to one slice of a build: to the items of the
	// named Packages that are not verified last.
	Slice []string
	// SliceSize is how many items a slice holds at most; 0 means SliceSize.
	SliceSize int
}

// Status says what is to verify in the folder, and changes nothing.
func (e *Engine) Status(_ context.Context, req StatusRequest) (Status, error) {
	root, err := e.Folder.Root(req.Dir)
	if err != nil {
		return Status{}, err
	}
	// Under the lock, so that status does not see the record half written by a
	// mark beside it. Without a cache folder it reads without one.
	if dir, err := e.lockDir(); err == nil {
		release, err := lock(dir, recordKey(root), lockWait)
		if err != nil {
			return Status{}, fmt.Errorf("the record of %s is locked: %w", root, err)
		}
		defer release()
	}
	f, err := e.readFolder(root)
	if err != nil {
		return Status{}, err
	}
	data, err := readRecordFile(f.root)
	if err != nil {
		return Status{}, err
	}
	st, err := f.status(ParseRecord(data), req.All)
	if err != nil {
		return st, err
	}
	if len(st.SpecErrors) == 0 {
		st.Slices = planSlices(f.cur.Spec, st.Items, len(st.Errors), req.SliceSize)
	}
	switch {
	case len(req.Slice) > 0:
		st, err = st.inSlice(req.Slice)
	case req.Package != "":
		st, err = st.only(req.Package)
	default:
		st.Questions = f.set[project.QuestionsFile]
	}
	if err != nil {
		return st, err
	}
	st.Binding = st.bindingOfItems()
	return st, nil
}

// only limits status to the Package of a name: to the items and notes whose
// Definition belongs to it. The errors in the realization and the lines
// of the record markitect cannot read belong to no Package, so it leaves them
// out and counts them. A spec with errors has no list to limit.
func (s Status) only(name string) (Status, error) {
	s.Package = name
	if len(s.SpecErrors) > 0 {
		return s, nil
	}
	known := false
	for _, p := range s.Packages {
		if p.Package == name {
			known, s.Definitions, s.Units = true, p.Definitions, p.Units
		}
	}
	if !known {
		return Status{}, fmt.Errorf("%w: the spec has no Package %q; markitect list names them, and the one at the top is %s", ErrUsage, name, spec.PackageName(spec.RootPackage))
	}
	names := map[string]bool{}
	for _, p := range s.Packages {
		names[p.Package] = true
	}
	in := func(items []Item) []Item {
		var out []Item
		for _, it := range items {
			if packageNamed(names, it.Definition) == name {
				out = append(out, it)
			}
		}
		return out
	}
	s.Items, s.Skipped = in(s.Items), in(s.Skipped)
	s.ErrorsElsewhere, s.Errors = len(s.Errors), nil
	s.UnreadableElsewhere, s.Unreadable = len(s.Unreadable), nil
	return s, nil
}

// view is the folder's inputs and its record, settled per Definition.
type view struct {
	now map[string]Inputs
	// dirs holds the directory of each Package that something binds, as it
	// is now, by the Package's name.
	dirs  map[string]Directory
	items map[string]Item
	// goneInputs holds the inputs of a gone Definition: the files its lines name,
	// as they are now.
	goneInputs map[string]Inputs
	// evidence hashes a file of the folder as evidence.
	evidence func(file string) string
	// covered holds the Definitions of the spec that have no item of their
	// own: the parts a whole is verified with.
	covered map[string]bool
}

// see settles the record against the folder.
func (f *folder) see(rec Record) (view, error) {
	now, dirs, err := state{p: f.cur, hash: f.hash}.inputs()
	if err != nil {
		return view{}, err
	}
	evidence := func(file string) string {
		h, err := f.hash(file)
		if err != nil {
			return gone
		}
		return hashOrGone(h)
	}
	v := view{now: now, dirs: dirs, items: map[string]Item{}, goneInputs: map[string]Inputs{}, evidence: evidence, covered: map[string]bool{}}
	recorded := map[string]Directory{}
	for _, d := range rec.Directories {
		recorded[d.key()] = d
	}
	for _, id := range f.cur.Spec.OwnDefinitions() {
		if _, unit := now[id]; !unit {
			v.covered[id] = true
		}
	}
	lines := rec.byDefinition()
	for _, id := range sortedKeys(now) {
		d := f.cur.Spec.Definitions[id]
		it := Item{Definition: id, Title: d.Title, Parts: now[id].Files(InputPart), Fingerprint: now[id].Fingerprint(), Look: append([]string{d.File}, now[id].Files(InputFile)...), Last: f.cur.Spec.VerifiedLast(id)}
		s := settle(lines[id], now[id], evidence)
		switch {
		case len(lines[id]) == 0:
			it.State = StateNew
		case len(s.conflict) > 0:
			it.State = StateConflict
			for _, l := range s.conflict {
				it.Verdicts = appendNew(it.Verdicts, l.Verdict)
			}
		case s.match != nil:
			it.State, it.Verdict, it.Why = s.match.Verdict, s.match.Verdict, s.match.Why
			it.Look = appendNew(it.Look, s.match.Inputs.Files(InputEvidence)...)
		default:
			it.State, it.Changes, it.Verdict, it.Why = StateChanged, v.binding(id, s.closest.Inputs, s.changes, recorded), s.closest.Verdict, s.closest.Why
			it.Look = appendNew(it.Look, s.closest.Inputs.Files(InputEvidence)...)
		}
		v.items[id] = it
	}
	for _, id := range sortedKeys(lines) {
		if _, ok := now[id]; ok {
			continue
		}
		// A line on a part is of a record written before its whole was verified
		// as one: the part is not gone, and its whole has its own item. The
		// next verdict drops the line.
		if v.covered[id] {
			continue
		}
		in := Inputs{}
		it := Item{Definition: id, State: StateGone}
		for _, l := range lines[id] {
			for _, file := range l.Inputs.Files(InputFile) {
				in[InputFile+file] = evidence(file)
			}
			it.Verdict, it.Why = l.Verdict, l.Why
		}
		it.Fingerprint, it.Look = in.Fingerprint(), in.Files(InputFile)
		v.goneInputs[id] = in
		v.items[id] = it
	}
	return v, nil
}

// binding names, in place of each changed input bound:P of an item, the
// binding Definitions that differ between the directory of P the line names
// and the directory of P now: bound:ID, (new) for one the line's directory
// lacks and (gone) for one the directory now lacks. The item itself is left
// out, since its own inputs say how it changed. Where the line has no bound:P,
// or the record no longer holds the directory it names, the Package stays
// named, as any other input.
func (v view) binding(id string, line Inputs, changes []Change, recorded map[string]Directory) []Change {
	var out []Change
	for _, c := range changes {
		name, ok := strings.CutPrefix(c.Key, InputBound)
		var was Directory
		if ok && c.How != "new" {
			was, ok = recorded[Directory{Bound: name, Digest: line[c.Key]}.key()]
		}
		if !ok || c.How == "new" {
			out = append(out, c)
			continue
		}
		is := v.dirs[name].Binds
		ids := map[string]bool{}
		for b := range was.Binds {
			ids[b] = true
		}
		for b := range is {
			ids[b] = true
		}
		for _, b := range sortedKeys(ids) {
			before, after := was.Binds[b], is[b]
			switch {
			case b == id || before == after:
			case before == "":
				out = append(out, Change{InputBound + b, "new"})
			case after == "":
				out = append(out, Change{InputBound + b, "gone"})
			default:
				out = append(out, Change{InputBound + b, "changed"})
			}
		}
	}
	// What changed in a directory is never the item alone without its own
	// inputs saying so; should it be, the Packages stay named.
	if len(out) == 0 {
		return changes
	}
	return out
}

func (f *folder) status(rec Record, all bool) (Status, error) {
	st := Status{Unreadable: rec.Unreadable}
	if errs := f.cur.SpecErrors(); len(errs) > 0 {
		st.SpecErrors = errs
		return st, nil
	}
	st.Definitions = len(f.cur.Spec.OwnDefinitions())
	v, err := f.see(rec)
	if err != nil {
		return Status{}, err
	}
	st.Units = len(v.now)
	for _, id := range sortedKeys(v.items) {
		it := v.items[id]
		switch {
		case it.State == StateSkip && !all:
			st.Skipped = append(st.Skipped, it)
		case it.State == StateOK && !all:
		default:
			st.Items = append(st.Items, it)
		}
	}
	st.Errors = f.cur.RealizationErrors()
	st.Packages = packageItems(f.cur.Spec, st.Items)
	st.boundBy, st.binds = map[string][]string{}, map[string][]string{}
	for _, it := range st.Items {
		if packages := v.now[it.Definition].Files(InputBound); len(packages) > 0 {
			st.boundBy[it.Definition] = packages
		}
	}
	for name, dir := range v.dirs {
		st.binds[name] = sortedKeys(dir.Binds)
	}
	return st, nil
}

// bindingOfItems names what binds the items as they are listed, by Package:
// Root first, then by name.
func (s Status) bindingOfItems() []Binding {
	names := map[string]bool{}
	for _, it := range s.Items {
		for _, name := range s.boundBy[it.Definition] {
			names[name] = true
		}
	}
	var out []Binding
	for _, name := range sortedKeys(names) {
		if len(s.binds[name]) > 0 {
			out = append(out, Binding{Package: name, Definitions: s.binds[name]})
		}
	}
	root := spec.PackageName(spec.RootPackage)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Package == root && out[j].Package != root })
	return out
}

// packageNamed returns the name of the Package a Definition ID belongs to,
// among the names status lists: the nearest folder above it that is one of
// them, or Root.
func packageNamed(names map[string]bool, id string) string {
	for at := spec.FolderOf(id); at != spec.RootPackage; at = spec.FolderOf(at) {
		if names[at] {
			return at
		}
	}
	return spec.PackageName(spec.RootPackage)
}

// packageItems counts the items per Package of the spec. A Definition that is
// gone counts for the nearest Package above its path that exists.
func packageItems(sp *spec.Spec, items []Item) []PackageItems {
	byName := map[string]*PackageItems{}
	for _, path := range sp.SortedPackages() {
		p := &PackageItems{Package: spec.PackageName(path), Definitions: len(sp.Packages[path].Definitions), DependsOn: []string{}}
		for _, to := range sortedKeys(sp.Edges[path]) {
			if to != path {
				p.DependsOn = append(p.DependsOn, spec.PackageName(to))
			}
		}
		byName[p.Package] = p
	}
	for unit := range sp.Units() {
		byName[spec.PackageName(sp.PackageOf(unit))].Units++
	}
	for _, it := range items {
		name := spec.PackageName(sp.PackageOf(it.Definition))
		p := byName[name]
		p.Items++
		if it.State == StateSpecProblem {
			p.SpecProblems++
		}
	}
	out := make([]PackageItems, 0, len(byName))
	for _, name := range sortedKeys(byName) {
		out = append(out, *byName[name])
	}
	return out
}

// CodeUnreadable marks a line of the record markitect cannot read.
const CodeUnreadable = "record.unreadable"

// RecordErrors are the lines of the record markitect cannot read, as
// diagnostics.
func (s Status) RecordErrors() spec.Diagnostics {
	var ds spec.Diagnostics
	for _, n := range s.Unreadable {
		ds.Add(CodeUnreadable, project.ReviewFile, n, "markitect cannot read this line; the owner fixes or removes it")
	}
	return ds
}

// Text is status as people and agents read it.
func (s Status) Text() string {
	var b strings.Builder
	if len(s.SpecErrors) > 0 {
		fmt.Fprintf(&b, "The spec has %d %s, so there is no list until it compiles; markitect check groups them:\n\n", len(s.SpecErrors), plural(len(s.SpecErrors), "error", "errors"))
		for _, d := range s.SpecErrors {
			fmt.Fprintf(&b, "%s\n", d.String())
		}
	}
	if len(s.Items) > 0 {
		fmt.Fprintf(&b, "To verify%s (%d of %d %s):\n\n", s.inPackage(), len(s.Items), s.Units, plural(s.Units, "item", "items"))
		for _, it := range s.Items {
			b.WriteString(it.text())
		}
	}
	if len(s.Binding) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("What binds these items without a link; each is verified against it, and markitect doc ID prints one:\n\n")
		for _, bd := range s.Binding {
			fmt.Fprintf(&b, "%s: %s\n", bd.Package, strings.Join(bd.Definitions, ", "))
		}
	}
	if len(s.Errors) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Errors in the realization (%d), for whoever builds:\n\n", len(s.Errors))
		for _, d := range s.Errors {
			fmt.Fprintf(&b, "%s\n", d.String())
		}
	}
	if errs := s.RecordErrors(); len(errs) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		for _, d := range errs {
			fmt.Fprintf(&b, "%s\n", d.String())
		}
	}
	if len(s.Skipped) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Notes: skipped, verified as far as the project can show:\n\n")
		for _, it := range s.Skipped {
			fmt.Fprintf(&b, "%s: %s\n", it.Definition, it.Why)
		}
	}
	if s.Clean() {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		if len(s.Slice) > 0 {
			fmt.Fprintf(&b, "Nothing to verify%s: every item of it is verified at its current inputs.\n", s.inPackage())
		} else {
			fmt.Fprintf(&b, "Nothing to verify%s: the spec compiles, and %s verified at %s current inputs.\n", s.inPackage(), every(s.Definitions), plural(s.Definitions, "its", "their"))
		}
	}
	if s.ForTheOwner() {
		fmt.Fprintf(&b, "\nOnly spec-problems are left%s: they wait for the owner, and no builder settles them.\n", s.inPackage())
	}
	if s.Questions {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "The architect's questions for the owner wait in %s.\n", project.QuestionsFile)
	}
	if len(s.Slice) > 0 {
		var parts []string
		if n := s.ItemsElsewhere; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s of other slices", n, plural(n, "item", "items")))
		}
		if n := s.ErrorsElsewhere; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s that no file realizes yet", n, plural(n, "Definition", "Definitions")))
		}
		if n := s.UnreadableElsewhere; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s of the record that markitect cannot read", n, plural(n, "line", "lines")))
		}
		if len(parts) > 0 {
			fmt.Fprintf(&b, "\nNot listed: %s; markitect status lists all of it.\n", strings.Join(parts, ", "))
		}
		return b.String()
	}
	if n := s.ErrorsElsewhere + s.UnreadableElsewhere; n > 0 {
		var parts []string
		if e := s.ErrorsElsewhere; e > 0 {
			parts = append(parts, fmt.Sprintf("%d %s in the realization", e, plural(e, "error", "errors")))
		}
		if u := s.UnreadableElsewhere; u > 0 {
			parts = append(parts, fmt.Sprintf("%d %s of the record that markitect cannot read", u, plural(u, "line", "lines")))
		}
		fmt.Fprintf(&b, "\nNot listed: %s, which %s to no Package; markitect status lists %s.\n", strings.Join(parts, " and "), plural(n, "belongs", "belong"), plural(n, "it", "them"))
	}
	return b.String()
}

// inPackage names the Package or the slice a list is limited to, for a
// heading.
func (s Status) inPackage() string {
	switch {
	case len(s.Slice) > 0:
		return " in the slice of " + packagesText(s.Slice)
	case s.Package == "":
		return ""
	}
	return " in Package " + s.Package
}

func every(n int) string {
	if n == 1 {
		return "its one Definition is"
	}
	return fmt.Sprintf("all %d Definitions are", n)
}

// text is one item as status prints it.
func (it Item) text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s@%s\n", it.Definition, it.Fingerprint)
	if n := len(it.Parts); n > 0 {
		fmt.Fprintf(&b, "  with its %d %s, verified as one; markitect doc %s shows it whole\n", n, plural(n, "part", "parts"), path.Dir(it.Definition))
	}
	switch it.State {
	case StateNew:
		b.WriteString("  no record yet\n")
	case StateChanged:
		var changes []string
		for _, c := range it.Changes {
			changes = append(changes, c.String())
		}
		fmt.Fprintf(&b, "  changed: %s\n", strings.Join(changes, ", "))
		if it.Verdict != VerdictOK {
			fmt.Fprintf(&b, "  recorded before: %s", it.Verdict)
			if it.Why != "" {
				fmt.Fprintf(&b, ": %s", it.Why)
			}
			b.WriteString("\n")
		}
	case StateFail, StateSpecProblem, StateSkip:
		fmt.Fprintf(&b, "  %s: %s\n", it.State, it.Why)
	case StateOK:
		b.WriteString("  ok\n")
	case StateGone:
		b.WriteString("  gone from the spec: are the files that realized it removed, or do they have a new basis?\n")
		if it.Verdict != VerdictOK {
			fmt.Fprintf(&b, "  recorded: %s: %s\n", it.Verdict, it.Why)
		}
	case StateConflict:
		fmt.Fprintf(&b, "  conflict: lines with %s match the inputs as they are\n", strings.Join(it.Verdicts, " and "))
	}
	if len(it.Look) > 0 {
		fmt.Fprintf(&b, "  look at: %s\n", strings.Join(it.Look, ", "))
	}
	return b.String()
}

// appendNew appends what list does not hold yet.
func appendNew(list []string, values ...string) []string {
	for _, v := range values {
		found := false
		for _, w := range list {
			found = found || v == w
		}
		if !found {
			list = append(list, v)
		}
	}
	return list
}
