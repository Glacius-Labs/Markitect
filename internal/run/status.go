package run

import (
	"context"
	"fmt"
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

// Item is one Definition to verify, with what changed and where to look.
type Item struct {
	Definition  string   `json:"definition"`
	Title       string   `json:"title,omitempty"`
	State       string   `json:"state"`
	Fingerprint string   `json:"fingerprint"`
	Changes     []Change `json:"changes,omitempty"`
	// Verdict and Why are those of the line the item rests on, if any.
	Verdict string `json:"verdict,omitempty"`
	Why     string `json:"why,omitempty"`
	// Verdicts are the different verdicts of a conflict.
	Verdicts []string `json:"verdicts,omitempty"`
	Look     []string `json:"look,omitempty"`
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
	// Definitions counts the Definitions of the spec.
	Definitions int
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
}

// PackageItems is one Package with the items of status whose Definition lies
// directly in it.
type PackageItems struct {
	// Package is the Package's name: its path, or Root.
	Package string `json:"package"`
	// Definitions counts the Definitions that lie directly in it.
	Definitions int `json:"definitions"`
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

// StatusRequest is one call of status.
type StatusRequest struct {
	Dir string
	// All lists every Definition, to verify everything again.
	All bool
	// Package limits the list to the items whose Definition lies directly in the
	// Package of that name, as a strand of a parallel run works on them.
	Package string
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
	if req.Package == "" {
		st.Questions = f.set[project.QuestionsFile]
		return st, nil
	}
	return st.only(req.Package)
}

// only limits status to the Package of a name: to the items and notes whose
// Definition lies directly in it. The errors in the realization and the lines
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
			known, s.Definitions = true, p.Definitions
		}
	}
	if !known {
		return Status{}, fmt.Errorf("%w: the spec has no Package %q; markitect list names them, and the one at the top is %s", ErrUsage, name, spec.PackageName(spec.RootPackage))
	}
	in := func(items []Item) []Item {
		var out []Item
		for _, it := range items {
			if spec.PackageName(spec.ParentOf(it.Definition)) == name {
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
	now   map[string]Inputs
	items map[string]Item
	// goneInputs holds the inputs of a gone Definition: the files its lines name,
	// as they are now.
	goneInputs map[string]Inputs
	// evidence hashes a file of the folder as evidence.
	evidence func(file string) string
}

// see settles the record against the folder.
func (f *folder) see(rec Record) (view, error) {
	now, err := state{p: f.cur, hash: f.hash}.inputs()
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
	v := view{now: now, items: map[string]Item{}, goneInputs: map[string]Inputs{}, evidence: evidence}
	lines := rec.byDefinition()
	for _, id := range sortedKeys(now) {
		d := f.cur.Spec.Definitions[id]
		it := Item{Definition: id, Title: d.Title, Fingerprint: now[id].Fingerprint(), Look: append([]string{d.File}, now[id].Files(InputFile)...)}
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
			it.State, it.Changes, it.Verdict, it.Why = StateChanged, s.changes, s.closest.Verdict, s.closest.Why
			it.Look = appendNew(it.Look, s.closest.Inputs.Files(InputEvidence)...)
		}
		v.items[id] = it
	}
	for _, id := range sortedKeys(lines) {
		if _, ok := now[id]; ok {
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
	return st, nil
}

// packageItems counts the items per Package, for every Package of the spec and
// for a Package that is gone but still has a gone Definition on the list.
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
	for _, it := range items {
		name := spec.PackageName(spec.ParentOf(it.Definition))
		p := byName[name]
		if p == nil {
			p = &PackageItems{Package: name, DependsOn: []string{}}
			byName[name] = p
		}
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
		fmt.Fprintf(&b, "To verify%s (%d of %d %s):\n\n", s.inPackage(), len(s.Items), s.Definitions, plural(s.Definitions, "Definition", "Definitions"))
		for _, it := range s.Items {
			b.WriteString(it.text())
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
		fmt.Fprintf(&b, "Nothing to verify%s: the spec compiles, and %s verified at %s current inputs.\n", s.inPackage(), every(s.Definitions), plural(s.Definitions, "its", "their"))
	}
	if s.Questions {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Questions for the owner wait in %s.\n", project.QuestionsFile)
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

// inPackage names the Package a list is limited to, for a heading.
func (s Status) inPackage() string {
	if s.Package == "" {
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
