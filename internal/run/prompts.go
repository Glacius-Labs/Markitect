package run

import (
	"fmt"
	"slices"
	"strings"
)

// modelAuthority is the quality duty every Manager and reviewer carries: the
// bound model is binding intent, never theirs to repair.
const modelAuthority = "The bound model always holds. An implementation choice it leaves open inside your responsibility is yours to make and document. What you cannot settle within the model goes to your parent; no answer overrides the model."

func labelOf(m *mgr) string {
	if m.Parent == "" {
		return "Root (.)"
	}
	return "Manager " + m.Path
}

func contactOf(m *mgr) string {
	if m.Role == "" {
		return m.Path
	}
	return fmt.Sprintf("%s (%s)", m.Path, m.Role)
}

func renderFinding(f *Finding) string {
	kind := "note"
	if f.Blocking {
		kind = "genuine"
	}
	text := fmt.Sprintf("Finding %s [%s], owner %s, raised by %s in %s", f.ID, kind, f.Owner, f.RaisedBy, f.Pass)
	if f.Kind != "" {
		text = fmt.Sprintf("Finding %s [%s], kind %s, owner %s, raised by %s in %s", f.ID, kind, f.Kind, f.Owner, f.RaisedBy, f.Pass)
	}
	if f.Blocking && f.Correction != "" {
		text += ", correction " + string(f.Correction)
	}
	if f.Subject != "" {
		text += ", subject " + f.Subject
		if f.Definition != "" {
			text += " " + f.Definition
		}
	}
	text = fmt.Sprintf("%s.\n  Obligation: %s", text, f.Obligation)
	if f.Expected != "" {
		text += "\n  Expected: " + f.Expected
	}
	text += "\n  Observed: " + f.Observed
	for _, d := range f.Disputes {
		text += fmt.Sprintf("\n  Disputed by %s in %s: %s\n    Evidence: %s", d.By, d.Pass, d.Reason, d.Evidence)
	}
	return text
}

func list(b *strings.Builder, title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	if title != "" {
		fmt.Fprintf(b, "\n%s\n", title)
	} else {
		b.WriteString("\n")
	}
	for _, l := range lines {
		fmt.Fprintf(b, "- %s\n", strings.ReplaceAll(l, "\n", "\n  "))
	}
}

// writeEnvelope renders the content of a submission.
func writeEnvelope(b *strings.Builder, env envelope, findings func(string) *Finding) {
	fmt.Fprintf(b, "  Summary: %s\n", env.Summary)
	if len(env.Scope) > 0 {
		fmt.Fprintf(b, "  Bound scope (not a coverage verdict): %s\n", strings.Join(env.Scope, ", "))
	}
	writeEvidence(b, env.Support)
	for _, cmd := range env.Commands {
		exit := "unknown"
		if cmd.ExitCode != nil {
			exit = fmt.Sprint(*cmd.ExitCode)
		}
		fmt.Fprintf(b, "  Observed command %s: %s; status %s; exit %s (not a candidate-bound conformance receipt)\n", cmd.ID, cmd.Command, cmd.Status, exit)
	}
	for _, check := range env.Checks {
		fmt.Fprintf(b, "  Required candidate check %s (%s): passed %t\n", check.ID, check.Name, check.Passed)
	}
	list(b, "Unresolved:", env.Unresolved)
	if pl := env.Plan; pl != nil {
		writePlan(b, pl)
	}
	if len(env.Artifacts) > 0 {
		fmt.Fprintf(b, "  Artifacts: %s\n", strings.Join(env.Artifacts, ", "))
	}
	if env.Verdict != "" {
		fmt.Fprintf(b, "  Verdict: %s\n", env.Verdict)
	}
	for _, id := range env.Findings {
		if f := findings(id); f != nil {
			fmt.Fprintf(b, "  %s\n", strings.ReplaceAll(renderFinding(f), "\n", "\n  "))
		}
	}
	if len(env.Closed) > 0 {
		fmt.Fprintf(b, "  Closed findings: %s\n", strings.Join(env.Closed, ", "))
	}
	if len(env.Disputed) > 0 {
		fmt.Fprintf(b, "  Disputed findings: %s\n", strings.Join(env.Disputed, ", "))
	}
	if len(env.Children) > 0 {
		fmt.Fprintf(b, "  Integrates the accepted child results %s\n", strings.Join(env.Children, ", "))
	}
}

func writePlan(b *strings.Builder, pl *plan) {
	fmt.Fprintf(b, "  Approach: %s\n", pl.Approach)
	if len(pl.Artifacts) > 0 {
		fmt.Fprintf(b, "  Planned artifacts: %s\n", strings.Join(pl.Artifacts, ", "))
	}
	for _, in := range pl.Inputs {
		fmt.Fprintf(b, "  Needs from %s: %s\n", in.Owner, in.Need)
	}
	for _, out := range pl.Outputs {
		fmt.Fprintf(b, "  Provides to %s: %s\n", strings.Join(out.Consumers, ", "), out.Provides)
	}
	for _, d := range pl.Dependencies {
		fmt.Fprintf(b, "  In %s, starts after the accepted %s of %s: %s\n", d.Pass, d.Stage, d.Owner, d.Need)
	}
	if pl.Order != "" {
		fmt.Fprintf(b, "  Order: %s\n", pl.Order)
	}
	for _, x := range pl.Blockers {
		fmt.Fprintf(b, "  Blocker: %s\n", x)
	}
	for _, x := range pl.Assumptions {
		fmt.Fprintf(b, "  Assumption: %s\n", x)
	}
}

// renderProposal is the item a parent receives to decide a child's checked
// submission.
func (e *engine) renderProposalLocked(pr *proposal) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Your child %s submitted its %s %s for your decision: use decide kind accept or return with proposal %s.\n", contactOf(pr.member.mgr), pr.kind, pr.id, pr.id)
	writeEnvelope(&b, pr.env, e.findingLocked)
	// Its finding effects take effect only with your acceptance.
	for i := range pr.raised {
		f := pr.raised[i]
		f.ID, f.Pass = "(proposed)", pr.member.p.id
		fmt.Fprintf(&b, "  %s\n", strings.ReplaceAll(renderFinding(&f), "\n", "\n  "))
	}
	for _, id := range slices.Sorted(mapKeys(pr.closing)) {
		c := pr.closing[id]
		fmt.Fprintf(&b, "  Closes %s as %s: %s\n    Evidence: %s\n", id, c.as, c.reason, c.evidence)
	}
	for _, id := range slices.Sorted(mapKeys(pr.disputes)) {
		d := pr.disputes[id]
		fmt.Fprintf(&b, "  Disputes %s: %s\n    Evidence: %s\n", id, d.Reason, d.Evidence)
	}
	for _, id := range slices.Sorted(mapKeys(pr.dispositions)) {
		d := pr.dispositions[id]
		if d.Setup != "" {
			fmt.Fprintf(&b, "  Assessed script setup failure %s, corrected by %s: %s\n    Reviewer evidence before candidate execution (not a Runtime certification): %s\n", id, d.Passing, d.Reason, d.Setup)
			continue
		}
		if d.As == DisposedRerun {
			fmt.Fprintf(&b, "  Disposes of the adverse receipt %s: replaced by the passing rerun %s\n", id, d.Rerun)
			continue
		}
		if d.Passing != "" {
			var erred, passing string
			if r := e.receiptLocked(id); r != nil {
				erred = r.Claim
			}
			if r := e.receiptLocked(d.Passing); r != nil {
				passing = r.Claim
			}
			fmt.Fprintf(&b, "  Disposes of the error receipt %s (claim: %s) as caused by the probe, not the candidate; a timeout or a kill can also be a hang or crash of the candidate, so judge whether the passing receipt %s (claim: %s) establishes that claim: %s\n    Evidence: %s\n", id, erred, d.Passing, passing, d.Reason, d.Evidence)
			continue
		}
		fmt.Fprintf(&b, "  Disposes of the adverse receipt %s as an erroneous expectation: %s\n    Evidence: %s\n", id, d.Reason, d.Evidence)
	}
	// Every probe the child ran in this Pass, whatever its report cites, so
	// that the parent judges the actual assertions and the adverse ones too.
	probes := e.probesOfLocked(pr.member.Path, pr.member.p.id)
	support := slices.Clone(pr.env.Support)
	for _, ev := range support {
		if r := e.receiptLocked(ev.Receipt); r != nil && !slices.Contains(probes, r) {
			probes = append(probes, r)
		}
	}
	if len(probes) > 0 {
		b.WriteString("  Probes it ran in this Pass: judge the model authority for the expected outcome and which material case could still pass each script. Relied-on probes are shown in full within a 32 KiB budget; retrieve any cut receipt with look_up topic probe before relying on it.\n")
		relied := map[string]bool{}
		for _, ev := range support {
			relied[ev.Receipt] = true
		}
		remaining := 32 << 10
		for _, r := range probes {
			var rb strings.Builder
			bytes := len(r.Content) + len(r.Output)
			full := relied[r.ID] && bytes <= remaining
			if full {
				remaining -= bytes
			}
			writeReceipt(&rb, r, full)
			fmt.Fprintf(&b, "    %s\n", strings.ReplaceAll(strings.TrimRight(rb.String(), "\n"), "\n", "\n    "))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderAccepted renders an accepted submission for its consumers.
func (e *engine) renderAcceptedLocked(pr *proposal) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Accepted %s %s of %s:\n", pr.kind, pr.id, pr.member.Path)
	writeEnvelope(&b, pr.env, e.findingLocked)
	return strings.TrimRight(b.String(), "\n")
}

func renderPlan(manager string, pr *proposal) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Accepted plan %s of %s (%s):\n", pr.id, manager, pr.member.p.id)
	writePlan(&b, pr.env.Plan)
	return strings.TrimRight(b.String(), "\n")
}

// writeNote states where a turn may write in /work.
func writeNote(readOnly bool, mounts []PathMount) string {
	scratch := " Write build output and other scratch files under /tmp: everything left under /work is captured and must be accounted."
	var writable, protected []string
	for _, mount := range mounts {
		if mount.Writable {
			writable = append(writable, mount.Path)
		} else {
			protected = append(protected, mount.Path)
		}
	}
	switch {
	case readOnly && len(writable) == 0:
		return "In this turn /work is read-only." + scratch
	case readOnly:
		note := "Outside these paths /work is read-only for you and a write there fails: " + strings.Join(writable, ", ") + "."
		if len(protected) > 0 {
			note += " Read-only even within them: " + strings.Join(protected, ", ") + "."
		}
		return note + scratch
	case len(protected) > 0:
		note := "These paths of /work are read-only for you and a write there fails: " + strings.Join(protected, ", ") + "."
		if len(writable) > 0 {
			note += " Writable again within them: " + strings.Join(writable, ", ") + "."
		}
		return note + scratch
	}
	return "You may write in /work." + scratch
}

// promptLocked renders one turn of a Manager or reviewer: its standing context
// when the conversation is fresh, its Pass assignment when delivered, the
// current task, what waits for its decision and its new items.
func (e *engine) promptLocked(m *member, items []item, standing, readOnly bool, mounts []PathMount) string {
	p := m.p
	var b strings.Builder
	if standing {
		e.standingLocked(&b, m)
	}
	var texts []string
	for _, it := range items {
		if it.kind == itemAssignment {
			e.assignmentLocked(&b, m)
			continue
		}
		texts = append(texts, it.text)
	}
	b.WriteString("\nNow: " + e.taskLocked(m) + "\n")
	b.WriteString(finalAssessmentInstruction + "\n")
	fmt.Fprintf(&b, "This Agent Turn has a finite limit of %s. End with a short final assessment; use status continue when work remains. Use wait for a missing named submission or answer instead of polling it.\n", e.cfg.Budget.TurnTimeout)
	b.WriteString("Each Turn uses a fresh container: conversation and the provider home persist, but in-memory tool/REPL variables and /tmp do not. Save recipes in provider-home files or resend their complete content; never rely on an earlier Turn's variables. A probe is a separate execution and must include all its own setup and helpers.\n")
	if p.spec.Review {
		b.WriteString("The Runtime bounds checks and reserves time for a report. If it refuses further execution for this Turn, end with status continue. For result evidence, one passing execution may support several conclusions when its actual assertions support them; explain support and gaps. Keep unsupported conditions open with a note; cite failed receipts in findings, not coverage evidence. Report a demonstrated failure of your own public contract under that own obligation, even when its implementation cause is a dependency; coordinate required work outside your ownership through your parent. In parent integration, independently assess such cross-area failures and record the necessary finding for the actual correction owner within your subtree; accepting the caller's contract finding alone does not route that correction.\n")
	}
	switch {
	case !p.spec.Review && !p.spec.Writes:
		fmt.Fprintf(&b, "In the %s Pass /work is read-only for every Manager: nobody writes a project file in it, and your result is complete without one. Write scratch files under /tmp.\n", p.spec.Purpose)
	case !p.spec.Review:
		b.WriteString(writeNote(readOnly, mounts) + "\n")
	}
	list(&b, "Waiting for your decision or answer:", e.demandsLocked(m))
	list(&b, "Items:", texts)
	return b.String()
}

func (e *engine) standingLocked(b *strings.Builder, m *member) {
	switch {
	case m.p.spec.Assesses:
		fmt.Fprintf(b, "You assess the bound model of the responsibility of %s read-only, in a fresh context, before anything is realized.\n", labelOf(m.mgr))
	case m.p.spec.Review:
		fmt.Fprintf(b, "You are the independent reviewer of the responsibility of %s. You assess an immutable candidate read-only, in a fresh context without the authors' conversation.\n", labelOf(m.mgr))
	default:
		fmt.Fprintf(b, "You are %s.\n", labelOf(m.mgr))
	}
	if m.Brief != "" {
		fmt.Fprintf(b, "\n%s\n", strings.TrimSpace(m.Brief))
	}
	if e.cfg.OwnerRequest != "" {
		fmt.Fprintf(b, "\nOwner request: %s\n", strings.TrimSpace(e.cfg.OwnerRequest))
	}
	var contacts []string
	if m.Parent != "" {
		contacts = append(contacts, "parent: "+contactOf(e.byPath[m.Parent]))
	}
	for _, c := range m.Children {
		contacts = append(contacts, "child: "+contactOf(e.byPath[c]))
	}
	list(b, "Your contacts for send:", contacts)
	if len(m.Helpers) > 0 {
		var roles []string
		for _, h := range m.Helpers {
			mode := "read-only"
			if h.Writes {
				mode = "may change a delegated subset of your area"
			}
			roles = append(roles, fmt.Sprintf("%s (%s)", h.Role, mode))
		}
		list(b, fmt.Sprintf("Your helper roles for delegate (at most %d at a time):", m.MaxHelpers), roles)
	}
	b.WriteString("\n" + modelAuthority + " " + methodIndex() + "\n")
	if m.Parent == "" && !m.p.spec.Review && e.cfg.RootGuidance != "" {
		b.WriteString(strings.TrimSpace(e.cfg.RootGuidance) + "\n")
	}
}

// assignmentLocked renders the Pass assignment: purpose and mandate, the sealed
// result of the previous Pass, the accepted plans and bases this Manager builds
// on and, in a review, the candidate's checks and the findings to reassess.
func (e *engine) assignmentLocked(b *strings.Builder, m *member) {
	p := m.p
	fmt.Fprintf(b, "\nPass %s: %s\n%s\n", p.id, p.spec.Purpose, strings.TrimSpace(p.spec.Mandate))
	if evidenceRequired(p.spec) {
		b.WriteString(semanticAssessment + "\n")
		b.WriteString("Assess material conditions, exceptions and interactions from the authority this Pass binds and required contracts (Apply: bound Definitions; Extract: fixed artifacts and documented intent, with the proposed model as work product); justify which evidence covers them and what remains open. For executable public behavior, done/preserved requires candidate-bound public_behavior receipts through the real declared entry, not private validators, source reassurance or suite counts. Static/documentary claims use reasoned source inspection. Missing required execution remains open. Every parent, including Root with no own Definitions, assesses integration using relevant child evidence and concrete challenges; green child labels alone cannot establish global invariants. No per-obligation mapping or separate integration form is required. Implement satisfied means implementation submitted to independent Verify, not verified conformance.\n")
		if p.spec.Review {
			b.WriteString("Before assessing author claims or coverage, derive your own conditions and counterexamples from that authority. Challenge omissions inside each Definition: valid inputs wrongly refused, invalid inputs wrongly accepted, boundaries, exceptions, artifact contradictions and child interactions. Then compare the candidate and author coverage against those expectations. Never use author claims as the complete checklist.\n")
		}
	}
	if p.spec.Review {
		fmt.Fprintf(b, "%s\n", strings.TrimSpace(p.spec.Guidance))
	}
	if p.number > 1 {
		prev := e.passes[p.number-2]
		fmt.Fprintf(b, "The previous Pass %s (%s) was sealed with the verdict %s.\n", prev.id, prev.spec.Purpose, prev.verdict)
	}
	if !p.spec.Plans {
		if pr := e.plans[m.Path]; pr != nil && pr.env.Plan != nil && !p.spec.Review {
			b.WriteString("\n" + renderPlan(m.Path, pr) + "\n")
		}
	}
	if m.Parent != "" {
		if parent := p.byPath[m.Parent]; parent.basis != nil {
			b.WriteString("\nYour parent's " + e.renderAcceptedLocked(parent.basis) + "\n")
		}
	}
	for _, d := range e.dependenciesLocked(m) {
		y := p.byPath[d.Owner]
		pr := y.basis
		if d.Stage == stageNameResult {
			pr = y.result
		}
		if pr != nil {
			b.WriteString("\nYour dependency: " + e.renderAcceptedLocked(pr) + "\n")
		}
	}
	if p.spec.Review {
		var checks []string
		for _, c := range p.checks {
			line := c.Name + ": passed"
			if !c.Passed {
				line = fmt.Sprintf("%s: FAILED, exit code %d", c.Name, c.ExitCode)
				if c.Reason != "" {
					line += ", " + c.Reason
				}
				if c.Output != "" {
					line += "\n" + c.Output
				}
			}
			checks = append(checks, line)
		}
		b.WriteString("\nThe candidate passed the Model check and the File check.\n")
		if p.spec.Assesses {
			b.WriteString("No required check runs and no probe is offered in this Pass: a check judges artifacts, not the model.\n")
		}
		list(b, "Required checks on this candidate:", checks)
		var mine, within, inherited []string
		for _, f := range e.findings {
			switch {
			case f.Closed:
			case f.RaisedBy == m.Path && f.Review:
				mine = append(mine, renderFinding(f))
			case !e.within(f.Owner, m.Path):
			case f.RaisedBy == MarkitectFinding || e.within(f.RaisedBy, m.Path):
				within = append(within, renderFinding(f))
			default:
				inherited = append(inherited, renderFinding(f))
			}
		}
		list(b, "Open findings you raised in earlier Passes: reassess each; close one as resolved when this candidate corrects it, or as disproved with evidence when it was erroneous:", mine)
		list(b, "Other open findings raised within your responsibility; they keep your result from being satisfied until they are closed:", within)
		list(b, "Open findings an ancestor raised within your responsibility; that ancestor closes them after your result. Report satisfied only when this candidate corrects each of them; otherwise report needs_correction, or dispute a review finding you judge erroneous:", inherited)
		var current, earlier []string
		candidate := p.tree.Digest().String()
		for _, r := range e.receipts {
			if !r.adverse() || r.disposed() || !e.answersForLocked(r, m.Path) {
				continue
			}
			var rb strings.Builder
			writeReceipt(&rb, r, false)
			if r.Stand == candidate && len(current) < maxListed {
				current = append(current, strings.TrimRight(rb.String(), "\n"))
			} else if r.Requester == m.Path && len(earlier) < maxListed {
				earlier = append(earlier, strings.TrimRight(rb.String(), "\n"))
			}
		}
		list(b, "Adverse probe receipts on this candidate within your responsibility, or by an ancestor about your obligations, that still need an independent disposition: cite one in a genuine finding when it shows a violation, or dispose of it with reason and evidence:", current)
		list(b, "Adverse probe receipts of your responsibility on earlier candidates (context: assess their claims again on this candidate):", earlier)
	} else {
		var mine []string
		for _, f := range e.findings {
			if !f.Closed && !f.Review && f.RaisedBy == m.Path && f.Pass != p.id {
				mine = append(mine, renderFinding(f))
			}
		}
		list(b, "Open findings you raised in earlier work Passes: close each with reason and evidence in your basis or result once it is resolved:", mine)
	}
}

// taskLocked states what the Manager is to do in its current stage.
func (e *engine) taskLocked(m *member) string {
	switch m.stage {
	case stageBlocked:
		return fmt.Sprintf("your own work waits for %s. Until then answer what is asked of you; your project access is read-only.", m.waiting)
	case stageOwn:
		if len(m.Children) == 0 {
			return "work on your own contribution and submit its complete result in your final assessment. A basis is optional only when another Manager needs a contract before your result."
		}
		return "work on your own contribution, supply contracts or decisions your children actually need, accept their results and integrate them into your result. A separate basis is optional; it never gates unrelated work. Do not do their work."
	case stageIntegrate:
		var results []string
		for _, c := range m.Children {
			if r := m.p.byPath[c].result; r != nil {
				results = append(results, fmt.Sprintf("%s by %s: %s (verdict %s)", r.id, c, r.env.Summary, r.env.Verdict))
			}
		}
		text := fmt.Sprintf("submit your complete result of this %s Pass in your final assessment as its mandate defines it, with every own obligation covered and your verdict.", m.p.spec.Purpose)
		if len(results) > 0 {
			text = "integrate your children's accepted results with your own contribution and the interactions between them, then " + text + " Accepted child results: " + strings.Join(results, "; ") + "."
		}
		return text
	case stageBasis, stageResult:
		return fmt.Sprintf("your %s %s awaits its decision; answer what is asked of you.", m.pending.kind, m.pending.id)
	case stageDelegated:
		return "your children work on your accepted basis; decide their proposals and answer their questions."
	case stageDone:
		return "your complete result is accepted; answer what is asked of you."
	}
	return "wait until your work starts."
}

// helperPromptLocked renders the single turn of a helper.
func (e *engine) helperPromptLocked(h *helperRun) string {
	m := h.m
	var b strings.Builder
	fmt.Fprintf(&b, "You are a %s helper of %s in Pass %s (%s). You work for your Manager only and report your result to it.\n", h.role.Role, labelOf(m.mgr), m.p.id, m.p.spec.Purpose)
	if strings.TrimSpace(h.role.Instructions) != "" {
		fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(h.role.Instructions))
	}
	if m.Brief != "" {
		fmt.Fprintf(&b, "\nYour Manager's responsibility:\n%s\n", strings.TrimSpace(m.Brief))
	}
	fmt.Fprintf(&b, "\nTask: %s\n", h.task)
	var paths []string
	for _, p := range h.paths {
		paths = append(paths, p.Path)
	}
	if len(paths) == 0 {
		b.WriteString("You may read the whole project; /work is read-only for you.\n")
	} else {
		fmt.Fprintf(&b, "You may read the whole project and change only %s; the registry stays with your Manager.\n", strings.Join(paths, ", "))
	}
	b.WriteString(modelAuthority + " " + methodIndex() + "\n" + finalAssessmentInstruction + "\n")
	return b.String()
}
