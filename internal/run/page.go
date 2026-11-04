package run

import (
	"bytes"
	"fmt"
	"strings"
)

// Page renders the Result page (concept 3.0 §10) as Markdown: the one page a
// Command writes at the end.
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
	tokens := fmt.Sprintf("%d in, %d out", r.Usage.InputTokens, r.Usage.OutputTokens)
	if r.Usage.Partial {
		tokens = "at least " + tokens + ": a session reported no total, for example one the Command ended"
	}
	row("Tokens", tokens)
	row("Full check", fullCheckLine(r))

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

	b.WriteString("\n## How much checking happened\n\n")
	writeDepth(&b, r)

	b.WriteString("\n## What could not be tested\n\n")
	notShown := 0
	for _, it := range r.Items {
		if it.Answer == AnswerNotShown {
			fmt.Fprintf(&b, "- %s: %s\n", it.Item, oneLine(it.Explanation))
			notShown++
		}
	}
	switch {
	case r.NotTested != "":
		if notShown > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "The reviewer's closing words: %s\n", oneLine(r.NotTested))
	case notShown == 0:
		b.WriteString("No reviewer named anything.\n")
	}

	b.WriteString("\n## Required checks\n\n")
	rc := r.RequiredChecks
	switch {
	case rc.Ran:
		fmt.Fprintf(&b, "Ran and passed on the clean state of %s.\n", rc.Commit)
	case len(rc.Checks) > 0:
		fmt.Fprintf(&b, "Ran on the clean state of %s and did not pass.\n", rc.Commit)
	default:
		b.WriteString("Did not run.\n")
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
	if len(r.Sessions) > 0 {
		b.WriteString("\n## Sessions\n\n| Role | Round | Ended | Agent | Seconds | Detail |\n|---|---|---|---|---|---|\n")
		for _, s := range r.Sessions {
			ended := string(s.Reason)
			if s.Class != "" {
				ended += " (" + string(s.Class) + ")"
			}
			agent := s.Agent
			if len(s.Models) > 1 {
				agent += " (" + strings.Join(s.Models, ", ") + ")"
			}
			fmt.Fprintf(&b, "| %s | %d | %s | %s | %.0f | %s |\n", s.Role, s.Round, ended, cell(agent), s.Seconds, cell(s.Detail))
		}
	}
	if len(r.Cleanup) > 0 || r.State == StateLanded {
		b.WriteString("\n## Cleanup\n\n")
		for _, c := range r.Cleanup {
			fmt.Fprintf(&b, "- %s\n", oneLine(c))
		}
		if r.State == StateLanded {
			b.WriteString("- After landing, the Command deletes this Run's branch and the replaced Run branches and removes its container; this page is part of the landing, so what that leaves is reported only by the command: on its standard error and as `afterLanding` in its JSON.\n")
		}
	}
	return b.Bytes()
}

func writeDepth(b *bytes.Buffer, r *Result) {
	d := r.Depth
	if d.Pointers == 0 && d.FilesOpened == 0 && d.Commands == 0 {
		b.WriteString("No review ran.\n")
		return
	}
	if d.Measured {
		fmt.Fprintf(b, "The transcripts show %d files opened, %d commands run and %d subagents started.", d.FilesOpened, d.Commands, d.Subagents)
	} else {
		b.WriteString("This provider's transcript shows no reads: not measured.")
	}
	fmt.Fprintf(b, " The answers name %d locations: %d shown opened, %d not measured.", d.Pointers, d.PointersOpened, d.PointersNotMeasured)
	if d.RepeatedAnswers > 0 {
		fmt.Fprintf(b, " %d answers repeat another's words and pointers.", d.RepeatedAnswers)
	}
	b.WriteString("\n")
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
