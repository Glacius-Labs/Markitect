package run

import (
	"fmt"
	"strings"
)

// The tasks say what one start must achieve, in the host's words; how to
// work, the tools and the trailer format are the harness's (concept 1.0
// §3.7: tasks, not process).

func (s *schedule) makerTask(n *node) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Realize the change for Manager %s on branch %s, starting from %s, and push one submission that gives each of your %d checklist items a verdict with evidence.", n.name, n.branch, short(s.plan.Base), len(n.items))
	if len(n.children) > 0 && s.childrenApproved(n) {
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
	return fmt.Sprintf("Review submission %s of Manager %s on branch %s against the model and the diff from %s: confirm or dispute every verdict, then push your review as one commit without file changes on the submission.", n.submission, n.name, n.branch, short(s.plan.Base))
}

func (s *schedule) verifyTask(n *node) string {
	return fmt.Sprintf("Check the state at %s for Manager %s against each of your %d checklist items, and push your verdicts as one commit without file changes on branch %s.", short(s.plan.Base), n.name, len(n.items), n.branch)
}
