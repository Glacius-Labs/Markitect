package run

import (
	"fmt"
	"strings"
)

const mandate = "In Execute, compare your artifacts with the bound model, preserve correct realization, create missing artifacts and correct deviations within your files and registry; change nothing that already conforms, and leave out improvements the model does not require. Coordinate dependencies and integration through your parent and children. The model always holds and is never yours to repair: a contradiction in it, or a gap that changes behavior at the project boundary, goes to your parent instead of being decided by you; a choice it leaves open inside the project is yours. In the subsequent fresh Verify, assess the immutable stand and the interaction of your children. Report scope, findings, checks and unresolved questions. Do not change intent, write another owner's files or contact another branch directly."

// blockingGuidance defines what blocks in Verify for Apply: the part of the
// Verify prompt that Config.VerifyGuidance replaces.
const blockingGuidance = "Blocking means a concrete model violation, absent evidence the model explicitly requires, or a failed required check, citing the obligation and the observed behavior or source evidence. A Definition realized outside its owner's artifacts (rebuilt in another area, including a parent's own area, instead of using the owner's realization) and a change outside a Manager's own area are model violations; as a parent, check how your children actually use each other's realizations."

// extraDepthNote closes Apply's Verify guidance: it belongs to that text and is
// not appended to Config.VerifyGuidance.
const extraDepthNote = "Preferred extra test depth is advisory: set blocking to false."

// checkNote tells an Execute turn about the check tool. It is part of the
// prompt only when Config.CheckTool offers the tool.
const checkNote = " The tool check reports the findings of the project's accounting on the current /work tree, yours first (scope \"all\" lists those of other Managers too). It observes while other Managers may still write, so use it as advice before you finish; the candidate is judged only after every Execute turn ended."

func (e *engine) mandateText() string {
	if e.cfg.Mandate != "" {
		return e.cfg.Mandate
	}
	return mandate
}

func (e *engine) guidanceText() string {
	if e.cfg.VerifyGuidance != "" {
		return e.cfg.VerifyGuidance
	}
	return blockingGuidance + " " + extraDepthNote
}

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

// initialItem is the first Execute item of every Manager: it starts the
// Manager's own reconciliation work without a finding. A parent first supplies
// the foundation its children need, because they start when its turn ends; a
// child builds on the foundation its parent supplied.
func initialItem(m *mgr) string {
	var b strings.Builder
	b.WriteString("Carry out your responsibility now, as the mandate above describes. No change is a legitimate result when your area already conforms.")
	if m.Parent != "" {
		b.WriteString(" Your parent's foundation is in place; its messages to you tell you what is available. Build on it.")
	}
	if len(m.Children) > 0 {
		b.WriteString(" Your children start when this turn ends. First establish, within your own area, the foundation they need: project scaffolding such as the module and build files, never a realization of their Definitions. Where none exists, create it; where one exists, assess it against your obligations and your children's known needs, preserve it when it fits, and correct or supplement it within your own area (or coordinate its owner) before they begin. When several children need a rule or type that one of them owns, decide that its owner provides it in a part the others can use without a dependency cycle. Then tell them with send the concrete basis they can use. Their areas are theirs to realize, not yours.")
	}
	return b.String()
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

// answeredLocked lists the owner's answers known so far, earlier ones first.
func (e *engine) answeredLocked() []string {
	var answered []string
	for _, q := range e.questions {
		if q.Answered {
			answered = append(answered, fmt.Sprintf("%s: %s\n  Answer: %s", q.ID, q.Text, q.Answer))
		}
	}
	return answered
}

func (e *engine) executePromptLocked(m *mgr, items []string, first bool) string {
	var b strings.Builder
	if !first {
		b.WriteString("You continue your Execute work. These items arrived since your last turn.\n")
		list(&b, "Items:", items)
		b.WriteString("\nWork on them, then call finish.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%s\n\nYou are %s in an Execute turn.\n", e.mandateText(), label(m))
	e.contextLocked(&b, m)
	list(&b, "Answered owner questions:", e.answeredLocked())
	var contacts []string
	if m.Parent != "" {
		contacts = append(contacts, "parent: "+contact(e.byPath[m.Parent]))
	}
	for _, c := range m.Children {
		contacts = append(contacts, "child: "+contact(e.byPath[c]))
	}
	if m.Parent == "" && e.cfg.Owner != nil {
		contacts = append(contacts, "owner (the human; ask only what the owner decisions do not settle)")
	}
	list(&b, "Your contacts for the tool send:", contacts)
	if m.Parent == "" && e.cfg.DefectTool {
		b.WriteString("\nWhat the model and the owner decisions settle, you decide within the tree. A defect of the model itself goes to the owner only through the tool model_defect, which ends the run.\n")
	}
	check := ""
	if e.cfg.CheckTool {
		check = checkNote
	}
	b.WriteString("\nThe shared working tree is /work. Other Managers work in it at the same time: change only your own area and your registry shard. " + writeNote(m.Write) + "Run the project's build and tests yourself. Write binaries, coverage files and other scratch output outside /work, for example under /tmp: everything left under /work is captured and must be an accounted artifact." + check + "\n")
	list(&b, "Items:", items)
	b.WriteString("\nWork on them. Use send to coordinate through your parent and children. Call finish when you are done or waiting.\n")
	return b.String()
}

func (e *engine) verifyPromptLocked(m *mgr, rs *roundState, retry bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\nYou are %s in a fresh Verify turn of round %d.\n", e.mandateText(), label(m), rs.n)
	if retry {
		b.WriteString("Your previous Verify attempt ended without a valid assess call. Assess again.\n")
	}
	e.contextLocked(&b, m)
	b.WriteString("\nThe candidate is mounted read-only at /work; write build or test output only under /tmp. " + e.guidanceText() + "\n")

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

	list(&b, "Answered owner questions:", e.answeredLocked())

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
	b.WriteString("\nWhen your assessment is complete, report it with the tool assess: summary, satisfied, findings (owner is you or one of your descendants) and closed. satisfied is true only when none of your own findings stays open and blocking. The last valid assess call of your turn counts.\n")
	return b.String()
}
