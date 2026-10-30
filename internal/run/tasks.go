package run

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// The tasks say what one start must achieve, in the host's words; how to
// work, the tools and the trailer format are the harness's (concept 1.0
// §3.7: tasks, not process).

func (s *schedule) makerTask(n *node) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Realize the change for Manager %s on branch %s, starting from %s, and push one submission that gives each of your %d checklist items a verdict with evidence.", n.name, n.branch, short(n.from), len(n.items))
	if held := s.held(n); held != "" {
		fmt.Fprintf(&b, " %s escalated to you and continue from your approved submission when you settle the matter with a change of your own, so this one covers your own area: judge each item by your own files; their work comes when you integrate it.", held)
	} else if s.ownFirst(n) {
		b.WriteString(" Your working children start from your approved submission, so this one covers your own area: judge each item by your own files; the parts your children realize come when you integrate their work.")
	} else if len(n.children) > 0 && s.childrenApproved(n) {
		fmt.Fprintf(&b, " Your working children's approved work: %s. Merge it, review it and integrate it before you submit.", s.approvals(n))
	}
	if len(n.inbox) > 0 {
		b.WriteString(" The messages delivered with this start need your action first.")
	}
	return b.String()
}

func (s *schedule) approvals(n *node) string {
	var parts []string
	for _, c := range n.children {
		parts = append(parts, fmt.Sprintf("%s at %s (branch %s)", c.name, c.approval, c.branch))
	}
	return strings.Join(parts, ", ")
}

func (s *schedule) reviewTask(n *node) string {
	task := fmt.Sprintf("Review submission %s of Manager %s on branch %s against the model and the diff from %s: confirm or dispute every verdict, then push your review as one commit without file changes on the submission.", n.submission, n.name, n.branch, short(n.from))
	if held := s.held(n); held != "" {
		task += " It covers the Manager's own area and settles an escalation of " + held + "; what they realize comes with a later submission."
	} else if s.ownFirst(n) {
		task += " It covers the Manager's own area before its working children start; what they realize comes with a later submission."
	}
	if n.evidence != "" {
		task += "\n\n" + n.evidence
	}
	return task
}

// evidenceTail bounds the output of a failing check in a checker's task.
const evidenceTail = 4 << 10

// checkEvidence says what the required checks found on a submission, from
// the push that brought it. They refuse only a merged state, so a partial
// tree may fail them; its checker weighs the result (DEC-087).
func (s *schedule) checkEvidence(pushes []Push, logErr error, submission string) string {
	if len(s.plan.Checks) == 0 {
		return ""
	}
	if logErr != nil {
		return fmt.Sprintf("The host could not read the required checks' results on this submission: %v. Run them yourself if your verdicts depend on them.", logErr)
	}
	var results []runremote.CheckResult
	for _, p := range pushes {
		for _, u := range p.Updates {
			if p.Decision.Accepted && u.New == submission {
				results = p.Decision.Checks
			}
		}
	}
	if len(results) == 0 {
		return "The decision log holds no required checks' results on this submission. Run them yourself if your verdicts depend on them."
	}
	var b strings.Builder
	b.WriteString("The required checks on the branch's full state at this submission:")
	for _, r := range results {
		switch {
		case r.Passed:
			fmt.Fprintf(&b, "\n- %q passed.", r.Name)
		case r.TimedOut:
			fmt.Fprintf(&b, "\n- %q ran out of time.", r.Name)
		default:
			fmt.Fprintf(&b, "\n- %q exited with %d.", r.Name, r.Exit)
		}
		if out := r.Output; !r.Passed && out != "" {
			if len(out) > evidenceTail {
				out = "…" + out[len(out)-evidenceTail:]
			}
			fmt.Fprintf(&b, " Its output ends:\n\n%s\n", strings.TrimRight(out, "\n"))
		}
	}
	if !allPassed(results) {
		b.WriteString("\nA submission that holds no child's approved work is accepted even when a check fails, since its tree may be partial; the merged state must pass them. Judge whether the failure is a defect of this submission.")
	}
	return b.String()
}

// held names the working children that wait for the Manager to settle their
// escalation.
func (s *schedule) held(n *node) string {
	var names []string
	for _, c := range n.children {
		if c.held {
			names = append(names, c.name)
		}
	}
	return strings.Join(names, ", ")
}

// ownFirst reports whether the Manager's working children wait for its
// approval (concept 2.0 §6.2).
func (s *schedule) ownFirst(n *node) bool {
	for _, c := range n.children {
		if !c.released {
			return true
		}
	}
	return false
}

func (s *schedule) verifyTask(n *node) string {
	return fmt.Sprintf("Check the state at %s for Manager %s against each of your %d checklist items, and push your verdicts as one commit without file changes on branch %s.", short(s.plan.Base), n.name, len(n.items), n.branch)
}
