package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/state"
)

// reportsDocument is the JSON document of reports: what check finds in the
// reports, and the reports with the values of their sections. A value that is
// not of its form is empty.
type reportsDocument struct {
	header
	Errors  []diagnostic  `json:"errors"`
	Reports []reportEntry `json:"reports"`
}

type reportEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// Source is owner, intake or job, with Job and Role for a job; Day is
	// the day it was raised.
	Source      string   `json:"source"`
	Job         string   `json:"job"`
	Role        string   `json:"role"`
	Day         string   `json:"day"`
	Place       string   `json:"place"`
	PlaceGone   bool     `json:"place_gone"`
	Occurred    string   `json:"occurred"`
	Property    string   `json:"property"`
	Impact      string   `json:"impact"`
	Likelihood  string   `json:"likelihood"`
	Branches    []string `json:"branches"`
	Escalated   bool     `json:"escalated"`
	Proposals   []string `json:"proposals"`
	Open        bool     `json:"open"`
	ClosedOn    string   `json:"closed_on"`
	Outcome     string   `json:"outcome"`
	DuplicateOf string   `json:"duplicate_of"`
}

// reports lists the reports of the project, or with --open only the open
// ones, so that a job or a briefing reads them without reading Markdown. It
// exits 1 while check finds errors in them.
func reports(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	f, err := factoryOf(w)
	if err != nil {
		return ExitFailed, err
	}
	doc := reportsDocument{header: newHeader("reports"), Errors: []diagnostic{}, Reports: []reportEntry{}}
	var b strings.Builder
	for _, d := range f.Diagnostics {
		if strings.HasPrefix(d.Path, state.ReportsDir+"/") {
			doc.Errors = append(doc.Errors, diagnostic{File: d.Path, Line: d.Line, Code: d.Code, Message: d.Message})
			b.WriteString(d.String() + "\n")
		}
	}
	open, escalated := 0, 0
	for _, r := range f.Reports {
		if o.open && !r.Open() {
			continue
		}
		doc.Reports = append(doc.Reports, reportEntry{Name: r.Name, Path: r.Path, Source: r.By, Job: r.Job, Role: r.Role, Day: r.Raised,
			Place: r.Place, PlaceGone: r.PlaceGone, Occurred: r.Occurred, Property: r.Property, Impact: r.Impact, Likelihood: r.Likelihood,
			Branches: nonNil(r.Branches), Escalated: r.Escalated, Proposals: nonNil(r.Proposals), Open: r.Open(),
			ClosedOn: r.ClosedOn, Outcome: r.Outcome, DuplicateOf: r.DuplicateOf})
		b.WriteString(reportText(r))
		if r.Open() {
			open++
			if r.Escalated {
				escalated++
			}
		}
	}
	fmt.Fprintf(&b, "%s, %d open, %d of them escalated\n", plural(len(doc.Reports), "report"), open, escalated)
	code, err := output(o, stdout, doc, b.String())
	if err == nil && len(doc.Errors) > 0 {
		code = ExitFound
	}
	return code, err
}

// reportText is one report as text: its name and whether it is open, then a
// line per value, "(none)" for a value it does not hold.
func reportText(r *state.Report) string {
	var b strings.Builder
	or := func(v string) string {
		if v == "" {
			return "(none)"
		}
		return v
	}
	status := "open"
	if !r.Open() {
		status = "closed"
	}
	if r.Escalated {
		status += ", escalated"
	}
	source := r.By
	if r.By == "job" {
		source = "job " + r.Job + " " + r.Role
	}
	if r.Raised != "" {
		source += ", " + r.Raised
	}
	place := or(r.Place)
	if r.PlaceGone {
		place += " (gone)"
	}
	rating := "(none)"
	if r.Impact != "" {
		rating = "impact " + r.Impact + ", likelihood " + r.Likelihood
	}
	closing := "(none)"
	switch {
	case r.Outcome == state.ClosedDuplicate:
		closing = r.ClosedOn + ": duplicate of " + r.DuplicateOf
	case r.Outcome != "":
		closing = r.ClosedOn + ": " + r.Outcome
	}
	fmt.Fprintf(&b, "%s: %s\n  source: %s\n  place: %s\n  occurred or risk: %s\n  property: %s\n  rating: %s\n  branches: %s\n",
		r.Name, status, or(source), place, or(r.Occurred), or(r.Property), rating, or(strings.Join(r.Branches, ", ")))
	for _, p := range r.Proposals {
		b.WriteString("  proposal: " + p + "\n")
	}
	b.WriteString("  closing: " + closing + "\n")
	return b.String()
}
