package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// The commands on the spec besides check: vet analyses its design, list shows
// what it holds and what depends on what, and doc shows one Definition or
// Package. They run over a spec with errors too, and say so on standard error.

// vet prints the warnings of the spec's design as diagnostics, and one line that
// counts its notes; with --strict the notes too. It exits 1 only with --strict and
// a warning: notes never change the exit code (DEC-102). --json always carries
// both.
func (c CLI) vet(_ context.Context, args []string) (int, error) {
	var o options
	var strict bool
	fs := c.flags("vet", &o)
	fs.BoolVar(&strict, "strict", false, "list the notes too, and exit 1 when there is a warning")
	if positional, err := parse(fs, args); err != nil {
		return Misused, err
	} else if len(positional) != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	ws, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	c.warnErrors(p)
	r := analysis.Vet(p, historyOf(ws, p))
	if o.json {
		ds := diagnostics(severityWarning, r.Warnings)
		for _, n := range r.Notes {
			ds = append(ds, diagnostic{Severity: severityNote, Code: n.Code, Message: n.Message, File: n.File, Line: n.Line, Targets: n.Targets, Structure: n.Structure})
		}
		if err := c.printJSON(struct {
			envelope
			Diagnostics []diagnostic `json:"diagnostics"`
		}{newEnvelope("vet"), ds}); err != nil {
			return Misused, err
		}
	} else {
		for _, d := range r.Warnings {
			fmt.Fprintln(c.Stdout, d.String())
		}
		switch {
		case strict:
			for _, n := range r.Notes {
				fmt.Fprintln(c.Stdout, n.String())
			}
		case len(r.Notes) > 0:
			fmt.Fprintln(c.Stdout, notesLine(r.Notes))
		}
	}
	if strict && len(r.Warnings) > 0 {
		return Failed, nil
	}
	return OK, nil
}

// historyOf reads from the project's record what vet knows of the spec's past:
// how often each Definition's own text changed between verdicts, and what the
// reviewer counted with an ok or a skip on its text as it is. Nothing when there
// is no record. A line it cannot read counts nothing here: status names it.
func historyOf(ws Workspace, p *project.Project) analysis.History {
	data, err := ws.Read(project.ReviewFile)
	if err != nil {
		return analysis.History{}
	}
	rec := run.ParseRecord(data)
	h := analysis.History{Revisions: rec.Revisions(), Counts: map[string]analysis.Count{}}
	self := map[string]string{}
	if p.Spec != nil {
		for id, d := range p.Spec.Definitions {
			if d.Library == "" {
				self[id] = run.Hash(d.Data)
			}
		}
	}
	// Of several lines at one text, as a merge leaves them, vet judges the one
	// with the highest risk, whatever their order.
	for id, counts := range rec.Counts(self) {
		for i, c := range counts {
			count := analysis.Count{Cases: c.Cases, Covered: c.Covered, Skip: c.Skip}
			if i == 0 || count.Risk() > h.Counts[id].Risk() {
				h.Counts[id] = count
			}
		}
	}
	return h
}

// notesLine is the one line plain vet prints for its notes: how many unlinked
// mentions it found within the Package structure and in how many Definitions, as
// the measurement record counts them, how many more go against or beyond the
// structure, and how many other notes there are.
func notesLine(notes []analysis.Note) string {
	var mentions, sources, apart, other int
	for _, n := range notes {
		switch {
		case n.Code != "vet.unlinked_mention":
			other++
		case n.Structure == "":
			mentions += len(n.Targets)
			sources++
		default:
			apart += len(n.Targets)
		}
	}
	var parts []string
	if mentions > 0 {
		parts = append(parts, fmt.Sprintf("%d unlinked %s in %d %s", mentions, plural(mentions, "mention", "mentions"), sources, plural(sources, "Definition", "Definitions")))
	}
	switch {
	case apart > 0 && mentions > 0:
		parts = append(parts, fmt.Sprintf("%d more against or beyond the Package structure", apart))
	case apart > 0:
		parts = append(parts, fmt.Sprintf("%d unlinked %s against or beyond the Package structure", apart, plural(apart, "mention", "mentions")))
	}
	switch {
	case other > 0 && len(parts) > 0:
		parts = append(parts, fmt.Sprintf("%d other %s", other, plural(other, "note", "notes")))
	case other > 0:
		parts = append(parts, fmt.Sprint(other))
	}
	return "notes: " + strings.Join(parts, ", ") + "; markitect vet --strict lists them"
}

// list prints the Packages and their Definitions; with --deps, what each
// depends on.
func (c CLI) list(_ context.Context, args []string) (int, error) {
	var o options
	var deps bool
	fs := c.flags("list", &o)
	fs.BoolVar(&deps, "deps", false, "add what each Package and Definition depends on, and each Package's coupling, abstractness and distance")
	if positional, err := parse(fs, args); err != nil {
		return Misused, err
	} else if len(positional) != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	_, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	c.warnErrors(p)
	pkgs := analysis.ListPackages(p, deps)
	if o.json {
		if err := c.printJSON(struct {
			envelope
			Packages []analysis.ListedPackage `json:"packages"`
		}{newEnvelope("list"), pkgs}); err != nil {
			return Misused, err
		}
		return OK, nil
	}
	fmt.Fprint(c.Stdout, renderList(pkgs))
	return OK, nil
}

// doc prints a Definition with its text, what links to it, what it links to,
// the files that realize it and its record; or a Package with its public and
// its internal Definitions apart, its registry page and its files. A target
// that is neither exits 1.
func (c CLI) doc(ctx context.Context, args []string) (int, error) {
	var o options
	fs := c.flags("doc", &o)
	positional, err := parse(fs, args)
	if err != nil {
		return Misused, err
	}
	if len(positional) != 1 {
		return Misused, fmt.Errorf("names one Definition or Package, such as members/member or members")
	}
	_, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	c.warnErrors(p)
	target := strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(positional[0], "[["), "]]"), ".md")
	if d := p.Spec.Definitions[target]; d != nil {
		dd := c.docDefinition(ctx, o, p, d)
		if o.json {
			return OK, c.printJSON(struct {
				envelope
				Definition analysis.DocDefinition `json:"definition"`
			}{newEnvelope("doc"), dd})
		}
		fmt.Fprint(c.Stdout, renderDocDefinition(dd))
		return OK, nil
	}
	if target == "Root" || target == "." {
		target = spec.RootPackage
	}
	if pkg := p.Spec.Packages[target]; pkg != nil {
		dp := analysis.DocPackageOf(p, pkg)
		if o.json {
			return OK, c.printJSON(struct {
				envelope
				Package analysis.DocPackage `json:"package"`
			}{newEnvelope("doc"), dp})
		}
		fmt.Fprint(c.Stdout, renderDocPackage(dp))
		return OK, nil
	}
	var ds spec.Diagnostics
	if p.Spec.Folders[target] {
		ds.Add("doc.unknown", "", 0, "%s is a folder and no Package: it holds no Definition of kind Package, so it is neither a Definition nor a Package of the spec", target)
	} else {
		ds.Add("doc.unknown", "", 0, "%s is neither a Definition nor a Package of the spec", target)
	}
	if o.json {
		return Failed, c.printJSON(struct {
			envelope
			Diagnostics []diagnostic `json:"diagnostics"`
		}{newEnvelope("doc"), diagnostics(severityError, ds)})
	}
	fmt.Fprintf(c.Stderr, "markitect doc: %s\n", ds[0].String())
	return Failed, nil
}

// docDefinition gathers what doc shows of a Definition, with its record.
func (c CLI) docDefinition(ctx context.Context, o options, p *project.Project, d *spec.Definition) analysis.DocDefinition {
	dd := analysis.DocDefinitionOf(p, d)
	if d.Own() {
		if st, err := c.Record.Status(ctx, run.StatusRequest{Dir: o.repo, All: true}); err == nil {
			for _, it := range st.Items {
				if it.Definition != d.ID {
					continue
				}
				r := &analysis.DocRecord{Item: it.Definition + "@" + it.Fingerprint, State: it.State, Verdict: it.Verdict, Why: it.Why}
				for _, ch := range it.Changes {
					r.Changes = append(r.Changes, ch.String())
				}
				dd.Record = r
			}
		}
	}
	return dd
}
