package run

import (
	"bytes"
	"fmt"
	"strings"
)

// Page renders the Result page (concept 2.0 §6.4) as Markdown.
func Page(r *Result) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# Run %s: %s\n\n", r.Run, r.State)
	fmt.Fprintf(&b, "%s.\n\n", capitalize(r.Why))
	b.WriteString("| | |\n|---|---|\n")
	row := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "| %s | %s |\n", k, cell(v))
		}
	}
	row("Command", r.Command)
	row("Outcome", r.Outcome)
	row("Target", r.Target)
	row("Base", r.Base)
	row("Start", r.Start)
	row("Run branch", r.Branch)
	if r.Full {
		row("Scope", "full")
	}
	row("Began", r.Began.UTC().Format("2006-01-02 15:04:05 UTC"))
	row("Ended", r.Ended.UTC().Format("2006-01-02 15:04:05 UTC"))
	row("Tokens", fmt.Sprintf("%d in, %d out", r.Usage.InputTokens, r.Usage.OutputTokens))

	b.WriteString("\n## Checklist\n\n")
	if len(r.Items) == 0 {
		b.WriteString("No checklist items.\n")
	} else {
		b.WriteString("| Manager | Item | Part | Verdict | Evidence | Approved by | Checker's evidence |\n|---|---|---|---|---|---|---|\n")
		for _, it := range r.Items {
			approved := "not approved"
			if it.Approved {
				approved = it.Checker
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n", cell(it.Manager), cell(it.Item), it.Part, orNone(it.Verdict), cell(it.Evidence), cell(approved), cell(it.Check))
		}
	}

	b.WriteString("\n## Required checks\n\n")
	rc := r.RequiredChecks
	switch {
	case rc.Ran && rc.WholeRepository:
		fmt.Fprintf(&b, "Ran and passed on the whole repository at %s.\n", rc.Commit)
	case len(rc.Checks) > 0:
		fmt.Fprintf(&b, "Ran on the whole repository at %s and did not pass.\n", rc.Commit)
	case r.Command == "verify":
		b.WriteString("Did not run on the whole repository.\n")
	default:
		b.WriteString("Did not run on a state that would land.\n")
	}
	if rc.Note != "" {
		fmt.Fprintf(&b, "\n%s.\n", capitalize(rc.Note))
	}
	if len(rc.Checks) > 0 {
		b.WriteString("\n| Check | Passed | Exit | Seconds |\n|---|---|---|---|\n")
		for _, c := range rc.Checks {
			fmt.Fprintf(&b, "| %s | %t | %d | %.1f |\n", cell(c.Name), c.Passed, c.Exit, c.Seconds)
		}
	}

	if len(r.OwnerDecisions) > 0 {
		b.WriteString("\n## Decisions for the owner\n\n")
		for _, d := range r.OwnerDecisions {
			fmt.Fprintf(&b, "- **%s:** %s Recommendation: %s\n", d.Manager, oneLine(d.Matter), oneLine(d.Recommendation))
		}
	}
	if len(r.Findings) > 0 {
		b.WriteString("\n## Findings\n\n")
		for _, f := range r.Findings {
			fmt.Fprintf(&b, "- [[%s]] (%s): observed %s; expected %s\n", f.Definition, f.Manager, oneLine(f.Observed), oneLine(f.Expected))
		}
	}

	b.WriteString("\n## Not taking part\n\n")
	if len(r.NotParticipating) == 0 {
		b.WriteString("Every Package took part.\n")
	}
	for _, p := range r.NotParticipating {
		fmt.Fprintf(&b, "- %s: %s\n", p.Package, p.Reason)
	}

	if len(r.Queue) > 0 {
		b.WriteString("\n## Queue\n\n")
		for _, q := range r.Queue {
			fmt.Fprintf(&b, "- %s\n", oneLine(q))
		}
	}
	if len(r.Replaced) > 0 {
		b.WriteString("\n## Replaced Run branches\n\nThis landing covers the impact of these kept Runs, so their branches are deleted with it:\n\n")
		for _, branch := range r.Replaced {
			fmt.Fprintf(&b, "- %s\n", branch)
		}
	}

	if len(r.Starts) > 0 {
		b.WriteString("\n## Agent starts\n\n| Manager | Role | Round | Ended | Agent | Seconds | Detail |\n|---|---|---|---|---|---|---|\n")
		for _, st := range r.Starts {
			ended := string(st.Reason)
			if st.Class != "" {
				ended += " (" + string(st.Class) + ")"
			}
			fmt.Fprintf(&b, "| %s | %s | %d | %s | %s | %.0f | %s |\n", cell(st.Manager), st.Role, st.Round, ended, cell(st.Agent), st.Seconds, cell(st.Detail))
		}
	}
	if len(r.Messages) > 0 {
		b.WriteString("\n## Messages\n\n")
		for _, m := range r.Messages {
			from := m.From
			if from == "" {
				from = "host"
			}
			to := m.To
			if to == "" {
				to = "owner"
			}
			fmt.Fprintf(&b, "- %s %s, %s to %s: %s\n", m.ID, m.Kind, from, to, oneLine(m.Text))
		}
	}
	if len(r.Notes) > 0 || len(r.ImpactNotes) > 0 {
		b.WriteString("\n## Side observations\n\n")
		for _, n := range r.ImpactNotes {
			fmt.Fprintf(&b, "- impact: %s\n", oneLine(n))
		}
		for _, n := range r.Notes {
			fmt.Fprintf(&b, "- %s (%s %s): %s\n", n.ID, n.Manager, n.Role, oneLine(n.Text))
		}
	}
	if len(r.Unkept) > 0 {
		b.WriteString("\n## Not kept\n\n")
		for _, u := range r.Unkept {
			fmt.Fprintf(&b, "- %s\n", oneLine(u))
		}
	}
	if len(r.Cleanup) > 0 || r.State == StateLanded {
		b.WriteString("\n## Cleanup\n\n")
		for _, c := range r.Cleanup {
			fmt.Fprintf(&b, "- %s\n", oneLine(c))
		}
		if r.State == StateLanded {
			b.WriteString("- After landing, the host deletes this Run's branch and the replaced Run branches, removes the Run's remote and leaves the target's lock; this page is part of the landing, so what that leaves is reported only by the command: on its standard error and as `afterLanding` in its JSON.\n")
		}
	}
	return b.Bytes()
}

// FindingsPage renders verify's Findings as a Findings page, the input of
// apply --findings.
func FindingsPage(r *Result) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# Findings of Run %s\n", r.Run)
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "\n## [[%s]]\n\nObserved: %s\n\nExpected: %s\n", f.Definition, oneLine(f.Observed), oneLine(f.Expected))
	}
	return b.Bytes()
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
