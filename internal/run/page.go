package run

import (
	"bytes"
	"fmt"
	"strings"
)

// Page renders the Result page as Markdown: the one page a Run ends with, or the
// page gate writes.
func Page(r *Result) []byte {
	var b bytes.Buffer
	gate := isGate(r)
	if gate {
		fmt.Fprintf(&b, "# Gate %s: %s\n\n", r.Run, r.State)
	} else {
		fmt.Fprintf(&b, "# Run %s: %s\n\n", r.Run, r.State)
	}
	fmt.Fprintf(&b, "%s.\n\n", capitalize(r.Why))
	b.WriteString("| | |\n|---|---|\n")
	row := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "| %s | %s |\n", k, cell(v))
		}
	}
	row("Command", r.Command)
	if gate {
		row("State", r.State)
	}
	row("Outcome", r.Outcome)
	row("Target", r.Target)
	row("Base", r.Base)
	row("Start", r.Start)
	row("Certificate", r.Certificate)
	row("Work", r.Work)
	row("Run branch", r.Branch)
	if r.Full {
		row("Scope", "full")
	}
	if r.Repairs > 0 {
		row("Repair rounds", fmt.Sprintf("%d", r.Repairs))
	}
	row("Began", r.Began.UTC().Format("2006-01-02 15:04:05 UTC"))
	row("Ended", r.Ended.UTC().Format("2006-01-02 15:04:05 UTC"))
	if !gate {
		row("Full check", fullCheckLine(r))
	}
	row("Session failure", r.Failure)

	if len(r.OwnerDecisions) > 0 {
		b.WriteString("\n## Decisions for the owner\n\n")
		for _, d := range r.OwnerDecisions {
			fmt.Fprintf(&b, "- %s", oneLine(d.Matter))
			if d.Recommendation != "" {
				fmt.Fprintf(&b, " Recommendation: %s", oneLine(d.Recommendation))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n## Checklist\n\n")
	if len(r.Items) == 0 {
		b.WriteString("No checklist items.\n")
	} else {
		b.WriteString("| Item | Answer | Where | Explanation |\n|---|---|---|---|\n")
		for _, it := range r.Items {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", cell(it.Item), orNone(it.Answer), cell(pointers(it.Pointers)), cell(it.Explanation))
		}
	}

	b.WriteString("\n## What could not be tested\n\n")
	notShown := 0
	for _, it := range r.Items {
		if it.Answer == AnswerNotShown {
			fmt.Fprintf(&b, "- %s: %s\n", it.Item, oneLine(it.Explanation))
			notShown++
		}
	}
	if notShown == 0 {
		b.WriteString("No reviewer named anything. How much the reviewer really read is not measured.\n")
	} else {
		b.WriteString("\nHow much the reviewer really read is not measured.\n")
	}

	if len(r.SentBack) > 0 {
		b.WriteString("\n## Repair rounds\n\n")
		for i, why := range r.SentBack {
			fmt.Fprintf(&b, "%d. %s\n", i+1, oneLine(why))
		}
	}
	if len(r.Findings) > 0 {
		b.WriteString("\n## Findings\n\n")
		for _, f := range r.Findings {
			fmt.Fprintf(&b, "- [[%s]]: observed %s; expected %s\n", f.Definition, oneLine(f.Observed), oneLine(f.Expected))
		}
	}
	if len(r.Unsettled) > 0 {
		b.WriteString("\n## What does not hold\n\n")
		for _, u := range r.Unsettled {
			fmt.Fprintf(&b, "- %s\n", oneLine(u))
		}
	}
	if len(r.MissingLinks) > 0 {
		b.WriteString("\n## Possible missing links\n\nA reviewer named these, and the registry does not list them for that item:\n\n")
		for _, l := range r.MissingLinks {
			fmt.Fprintf(&b, "- %s\n", oneLine(l))
		}
	}
	if len(r.ImpactNotes) > 0 {
		b.WriteString("\n## Side observations\n\n")
		for _, n := range r.ImpactNotes {
			fmt.Fprintf(&b, "- %s\n", oneLine(n))
		}
	}
	if len(r.Replaced) > 0 {
		b.WriteString("\n## Replaced Run branches\n\nThis landing covers the impact of these kept Runs, so their branches are deleted with it:\n\n")
		for _, branch := range r.Replaced {
			fmt.Fprintf(&b, "- %s\n", branch)
		}
	}
	return b.Bytes()
}

func pointers(ps []Pointer) string {
	var parts []string
	for _, p := range ps {
		s := p.File
		if p.Line > 0 {
			s += fmt.Sprintf(":%d", p.Line)
		}
		if len(p.Tests) > 0 {
			s += " (" + strings.Join(p.Tests, ", ") + ")"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
}

// FindingsPage renders verify's Findings as a Findings page, the input of
// apply --findings.
func FindingsPage(r *Result) []byte {
	var b bytes.Buffer
	if isGate(r) {
		fmt.Fprintf(&b, "# Findings of gate %s\n", r.Run)
	} else {
		fmt.Fprintf(&b, "# Findings of Run %s\n", r.Run)
	}
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "\n## [[%s]]\n\nObserved: %s\n\nExpected: %s\n", f.Definition, oneLine(f.Observed), oneLine(f.Expected))
	}
	return b.Bytes()
}

func isGate(r *Result) bool {
	return r.Command == commandGate || r.Command == commandGateFindings
}

func cell(s string) string {
	return strings.ReplaceAll(oneLine(s), "|", `\|`)
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
