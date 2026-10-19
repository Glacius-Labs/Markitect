package run

import (
	"fmt"
	"strings"
)

const mandate = "Compare your artifacts with the bound model. Preserve correct realization and report concrete deviations. In Execute, correct your files and registry and coordinate dependencies through your parent and children. In fresh Verify, assess the immutable stand and the interaction of your children. Report scope, findings, checks and unresolved questions. Do not change intent, write another owner's files or contact another branch directly."

func label(m *mgr) string {
	if m.Parent == "" {
		return "Root (.)"
	}
	return fmt.Sprintf("Manager %s", m.Path)
}

func contact(m *mgr) string {
	if m.Role == "" {
		return m.Path
	}
	return fmt.Sprintf("%s (%s)", m.Path, m.Role)
}

func renderFinding(f *Finding) string {
	kind := "advisory"
	if f.Blocking {
		kind = "blocking"
	}
	return fmt.Sprintf("Finding %s [%s], owner %s, raised by %s in round %d.\n  Obligation: %s\n  Observed: %s", f.ID, kind, f.Owner, f.RaisedBy, f.Round, f.Obligation, f.Observed)
}

func list(b *strings.Builder, title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(b, "\n%s\n", title)
	for _, l := range lines {
		fmt.Fprintf(b, "- %s\n", strings.ReplaceAll(l, "\n", "\n  "))
	}
}

func (e *engine) contextLocked(b *strings.Builder, m *mgr) {
	if m.Brief != "" {
		fmt.Fprintf(b, "\n%s\n", strings.TrimSpace(m.Brief))
	}
	if e.cfg.OwnerRequest != "" {
		fmt.Fprintf(b, "\nOwner request: %s\n", strings.TrimSpace(e.cfg.OwnerRequest))
	}
	list(b, "Owner decisions (fixed for this run):", e.cfg.OwnerDecisions)
}

// buildNote tells a Manager of a build-from-the-model run that its area is
// empty.
func buildNote(m *mgr) string {
	if m.Parent == "" {
		return "Your area holds no artifacts yet: you build it from the model, starting with the project skeleton and the shared prerequisites your children build on."
	}
	return "Your area holds no artifacts yet: you build it from the model, on the foundations your parent supplies."
}

// buildSeed is the initial Execute item of a building Manager with building
// children, which wait for its first turn.
func buildSeed(m *mgr) string {
	foundations := "the foundations your children need"
	if m.Parent == "" {
		foundations = "the project skeleton and the shared prerequisites your children need"
	}
	return "Build start: your children wait until this turn ends. Within your own area prepare " + foundations + ", then coordinate your children with send. Their areas are theirs to build, not yours."
}

// writeNote states what an enforced write plan makes read-only in /work. An
// unenforced plan, or one that restricts nothing, needs no note.
func writeNote(plan WritePlan) string {
	var writable, readOnly []string
	for _, mount := range plan.Mounts {
		if mount.Writable {
			writable = append(writable, mount.Path)
		} else {
			readOnly = append(readOnly, mount.Path)
		}
	}
	switch {
	case !plan.Enforced:
		return ""
	case !plan.Base && len(writable) == 0:
		return "Nothing in /work is writable for you: it is read-only and a write there fails. "
	case !plan.Base:
		note := "Outside your writable areas /work is read-only and a write there fails. Your writable areas: " + strings.Join(writable, ", ") + ". "
		if len(readOnly) > 0 {
			note += "Read-only even within them: " + strings.Join(readOnly, ", ") + ". "
		}
		return note
	case len(readOnly) > 0:
		note := "These areas of /work are read-only and a write there fails: " + strings.Join(readOnly, ", ") + ". "
		if len(writable) > 0 {
			note += "Writable again within them: " + strings.Join(writable, ", ") + ". "
		}
		return note
	}
	return ""
}

func (e *engine) executePromptLocked(m *mgr, items []string, first bool) string {
	var b strings.Builder
	if !first {
		b.WriteString("You continue your Execute work. These items arrived since your last turn.\n")
		list(&b, "Items:", items)
		b.WriteString("\nWork on them, then call finish.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%s\n\nYou are %s in an Execute turn.\n", mandate, label(m))
	e.contextLocked(&b, m)
	var contacts []string
	if m.Parent != "" {
		contacts = append(contacts, "parent: "+contact(e.byPath[m.Parent]))
	}
	for _, c := range m.Children {
		contacts = append(contacts, "child: "+contact(e.byPath[c]))
	}
	if m.Parent == "" {
		contacts = append(contacts, "owner (the human; ask only what the owner decisions do not settle)")
	}
	list(&b, "Your contacts for the tool send:", contacts)
	if m.Building {
		b.WriteString("\n" + buildNote(m) + "\n")
	}
	b.WriteString("\nThe shared working tree is /work. Other Managers work in it at the same time: change only your own area and your registry shard. " + writeNote(m.Write) + "Run the project's build and tests yourself. Write binaries, coverage files and other scratch output outside /work, for example under /tmp: everything left under /work is captured and must be an accounted artifact.\n")
	list(&b, "Items:", items)
	b.WriteString("\nWork on them. Use send to coordinate through your parent and children. Call finish when you are done or waiting.\n")
	return b.String()
}

func (e *engine) verifyPromptLocked(m *mgr, rs *roundState, retry bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\nYou are %s in a fresh Verify turn of round %d.\n", mandate, label(m), rs.n)
	if retry {
		b.WriteString("Your previous Verify attempt ended without a valid assess call. Assess again.\n")
	}
	e.contextLocked(&b, m)
	b.WriteString("\nThe candidate is mounted read-only at /work; write build or test output only under /tmp. Blocking means a concrete model violation, absent evidence the model explicitly requires, or a failed required check, citing the obligation and the observed behavior or source evidence. Preferred extra test depth is advisory: set blocking to false.\n")

	var checks []string
	for _, c := range rs.checks {
		switch {
		case c.Passed:
			checks = append(checks, c.Name+": passed")
		default:
			line := fmt.Sprintf("%s: FAILED, exit code %d", c.Name, c.ExitCode)
			if c.Reason != "" {
				line += ", " + c.Reason
			}
			if c.Output != "" {
				line += "\n" + c.Output
			}
			checks = append(checks, line)
		}
	}
	list(&b, "Required checks on this candidate:", checks)

	var answered []string
	for _, q := range e.questions {
		if q.Answered {
			answered = append(answered, fmt.Sprintf("%s: %s\n  Answer: %s", q.ID, q.Text, q.Answer))
		}
	}
	list(&b, "Answered owner questions:", answered)

	var structural, mine, addressed []string
	for _, f := range rs.structural {
		if f.Owner == m.Path {
			structural = append(structural, renderFinding(f))
		}
	}
	for _, f := range e.findings {
		switch {
		case f.Closed || f.RaisedBy == MarkitectFinding:
		case f.RaisedBy == m.Path:
			mine = append(mine, renderFinding(f))
		case f.Owner == m.Path:
			addressed = append(addressed, renderFinding(f))
		}
	}
	list(&b, "Structural findings of your area on this candidate (they hold for this candidate only):", structural)
	var children []string
	for _, c := range m.Children {
		if a, ok := rs.assessments[c]; ok {
			children = append(children, fmt.Sprintf("%s, satisfied %t: %s", c, a.Satisfied, a.Summary))
		}
	}
	list(&b, "Assessments of your children in this round:", children)
	list(&b, "Open findings you raised earlier; close one with a reason only when it is resolved on this candidate:", mine)
	list(&b, "Open findings addressed to you:", addressed)
	if len(m.changed) > 0 {
		shown := m.changed
		more := ""
		if len(shown) > maxListedPaths {
			shown, more = shown[:maxListedPaths], fmt.Sprintf("\n- and %d more", len(m.changed)-maxListedPaths)
		}
		fmt.Fprintf(&b, "\nFiles of your area that changed since the previous candidate although you ran no Execute turn:\n- %s%s\n", strings.Join(shown, "\n- "), more)
	}
	b.WriteString("\nReport with the tool assess exactly once: summary, satisfied, findings (owner is you or one of your descendants) and closed. satisfied is true only when none of your own findings stays open and blocking.\n")
	return b.String()
}
