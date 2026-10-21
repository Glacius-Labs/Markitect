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
	for _, c := range env.Obligations {
		line := fmt.Sprintf("  Obligation %s: %s", c.Definition, c.Status)
		if c.Note != "" {
			line += " (" + c.Note + ")"
		}
		b.WriteString(strings.ReplaceAll(line, "\n", "\n    ") + "\n")
		if c.Conditions != "" {
			fmt.Fprintf(b, "    Conditions and coverage: %s\n", c.Conditions)
		}
		writeEvidence(b, c.Evidence)
	}
	if c := env.Integration; c != nil {
		fmt.Fprintf(b, "  Parent integration: %s; conditions: %s; reasoning: %s\n", c.Status, c.Conditions, c.Note)
		writeEvidence(b, c.Evidence)
	}
	if pl := env.Plan; pl != nil {
		writePlan(b, pl)
	}
	if len(env.Artifacts) > 0 {
		fmt.Fprintf(b, "  Artifacts: %s\n", strings.Join(env.Artifacts, ", "))
	}
	for _, ev := range env.Evidence {
		fmt.Fprintf(b, "  Evidence: %s\n", ev)
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
	fmt.Fprintf(&b, "Your child %s submitted its %s %s for your decision: report kind accept or return with proposal %s.\n", contactOf(pr.member.mgr), pr.kind, pr.id, pr.id)
	if pr.member.p.spec.ModelBasis && pr.kind == kindResult && pr.member.basis != nil {
		fmt.Fprintf(&b, "Its model-first basis %s was accepted before candidate access. Compare these original expectations with the result; new findings may add cases but do not erase the original conditions:\n", pr.member.basis.id)
		writeEnvelope(&b, pr.member.basis.env, e.findingLocked)
	}
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
	if probes := e.probesOfLocked(pr.member.Path, pr.member.p.id); len(probes) > 0 {
		b.WriteString("  Probes it ran in this Pass: judge the model authority for the expected outcome and which material case could still pass each script. Relied-on probes are shown in full within a 32 KiB budget; retrieve any cut receipt with look_up topic probe before relying on it.\n")
		relied := map[string]bool{}
		for _, c := range pr.env.Obligations {
			for _, ev := range c.Evidence {
				relied[ev.Receipt] = true
			}
		}
		if c := pr.env.Integration; c != nil {
			for _, ev := range c.Evidence {
				relied[ev.Receipt] = true
			}
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
	fmt.Fprintf(&b, "This Agent Turn has a finite limit of %s. End after submitting basis/result; if work remains, report concise progress before the limit and continue in a later Turn. Use wait for a missing named submission or answer instead of polling it.\n", e.cfg.Budget.TurnTimeout)
	b.WriteString("Each Turn uses a fresh container: conversation and the provider home persist, but in-memory tool/REPL variables and /tmp do not. Save recipes in provider-home files or resend their complete content; never rely on an earlier Turn's variables. A probe is a separate execution and must include all its own setup and helpers.\n")
	if p.spec.Review && !m.candidateHidden() {
		b.WriteString("Plan one highest-value bounded probe for this Turn, then leave time to report and end. Start a second only when the observed tool latency still leaves a conservative reserve for a valid report and any repair; a short script can take minutes including Runtime setup. If a check refuses for remaining time, do not try another: report progress with only kind and summary, then end. For result evidence, cite each passing receipt once, only under its recorded obligation. Keep unsupported conditions open with a note; cite failed receipts in findings, not coverage evidence. Report a demonstrated failure of your own public contract under that own obligation, even when its implementation cause is a dependency; coordinate required work outside your ownership through your parent. In parent integration, independently assess such cross-area failures and record the necessary finding for the actual correction owner within your subtree; accepting the caller's contract finding alone does not route that correction.\n")
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
	if e.cfg.OwnerRequest != "" && !m.candidateHidden() {
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
	if m.candidateHidden() {
		if m.modelBasisPending() {
			b.WriteString("MODEL-FIRST BASIS: /work is an empty read-only authority view. The immutable candidate, author coverage, check results and prior receipts are withheld until every required reviewer's model-first basis is accepted after normal Turn completion. Use your delivered bound model and look_up definition to derive material valid/invalid classes, exact interface forms, expected outcomes, exceptions and composed cases. Record every own obligation as open with these conditions and a note that candidate assessment is pending; parents also record open integration. Supply no candidate evidence, artifacts or findings in this basis. Coordinate model questions through the tree; retain unresolved model questions explicitly in your conditions. End the Turn immediately after submission.\n")
		} else {
			b.WriteString("MODEL-ONLY COORDINATION: Your model-first basis is accepted. Every reviewer remains blind to the candidate until all required bases are accepted and a fresh candidate Turn starts. Decide child model-first bases and answer model questions; candidate results and probes are unavailable.\n")
			b.WriteString(e.renderAcceptedLocked(m.basis) + "\n")
		}
		b.WriteString(semanticAssessment + "\n")
		if m.Parent != "" {
			if parent := p.byPath[m.Parent]; parent.basis != nil {
				b.WriteString("\nYour parent's model-first " + e.renderAcceptedLocked(parent.basis) + "\n")
			}
		}
		return
	}
	if p.spec.ModelBasis && m.basis != nil {
		b.WriteString("Candidate access is now open on the immutable review stand. Your accepted precommitment follows; compare each original condition with the actual setup, public action and assertions, execute your own discriminating cases, and account for every original condition in your result or leave its missing evidence open. Add discoveries without replacing the precommitment.\n")
		b.WriteString(e.renderAcceptedLocked(m.basis) + "\n")
		if e.cfg.OwnerRequest != "" {
			b.WriteString("Bound owner request: " + e.cfg.OwnerRequest + "\n")
		}
	}
	if evidenceRequired(p.spec) {
		b.WriteString(semanticAssessment + "\n")
		b.WriteString("Derive and record material conditions, exceptions and interactions from the authority this Pass binds and required contracts in coverage.conditions (Apply: bound Definitions; Extract: fixed artifacts and documented intent, with the proposed model as work product); justify which evidence covers them and what remains open. For executable public behavior, done/preserved requires candidate-bound public_behavior receipts through the real declared entry, not private validators, source reassurance or suite counts. Static/documentary claims use reasoned source inspection. Missing required execution remains open. Every parent, including Root with no own Definitions, reports its own integration conditions and evidence; green children alone cannot establish global invariants. Implement satisfied means implementation submitted to independent Verify, not verified conformance.\n")
		if p.spec.Review {
			b.WriteString("Before assessing author claims or coverage, derive your own conditions and counterexamples from that authority in your basis. Challenge omissions inside each Definition: valid inputs wrongly refused, invalid inputs wrongly accepted, boundaries, exceptions, artifact contradictions and child interactions. Then compare the candidate and author coverage against those expectations. Never use author claims as the complete checklist.\n")
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
	if m.candidateHidden() && !m.modelBasisPending() {
		return "coordinate model questions and decisions on child bases; await all required model-first bases before candidate assessment."
	}
	switch m.stage {
	case stageBlocked:
		return fmt.Sprintf("your own work waits for %s. Until then answer what is asked of you; your project access is read-only.", m.waiting)
	case stageOwn:
		if m.modelBasisPending() {
			return "work on your own contribution: derive the model-first review basis before candidate access, keep coverage open and submit report kind basis."
		}
		if len(m.Children) == 0 {
			return "work on your own contribution and submit it as your basis (report kind basis)."
		}
		return "work on your own contribution and submit your basis (report kind basis): what your children need from you before they start. Do not do their work."
	case stageIntegrate:
		var results []string
		for _, c := range m.Children {
			if r := m.p.byPath[c].result; r != nil {
				results = append(results, fmt.Sprintf("%s by %s: %s (verdict %s)", r.id, c, r.env.Summary, r.env.Verdict))
			}
		}
		text := fmt.Sprintf("submit your complete result of this %s Pass (report kind result) as its mandate defines it, with every own obligation covered and your verdict.", m.p.spec.Purpose)
		if len(results) > 0 {
			text = "integrate your children's accepted results with your own contribution and the interactions between them, then " + text + " Accepted child results: " + strings.Join(results, "; ") + "."
		}
		return text
	case stageBasis, stageResult:
		return fmt.Sprintf("your %s %s awaits its decision; answer what is asked of you.", m.pending.kind, m.pending.id)
	case stageDelegated:
		if m.p.spec.ModelBasis {
			return "assess your own candidate conditions while your children review; decide their proposals and answer their questions. Report progress to continue later, or wait for a named child after your own assessment; integrate only after their results are accepted."
		}
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
	b.WriteString(modelAuthority + " " + methodIndex() + "\nWhen your task is done, report your result with the tool report.\n")
	return b.String()
}
