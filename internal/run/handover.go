package run

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Hand-over between Runs (DEC-073): a Finding reaches the next Implement only
// when it names a Definition of the current checked model; Assessment and
// Verify keep their other Findings as Notes. The host states that the
// command's bound input was already verified only from a stored closed Verify
// approval of exactly that input under the same Basis.

// defines reports whether address names a Definition of p's checked model as
// Package/ID, the form registry entries use.
func defines(p project.Prepared, address string) bool {
	i := strings.LastIndexByte(address, '/')
	if i <= 0 || i == len(address)-1 {
		return false
	}
	m, _ := p.Topology().Manager(address[:i])
	for _, d := range m.Definitions {
		if d.Package == address[:i] && d.ID == address[i+1:] {
			return true
		}
	}
	return false
}

// Initial is what the next Implement receives of a closed result's Findings:
// those that name a Definition of p's checked model.
func Initial(p project.Prepared, findings []Finding) []Finding {
	var initial []Finding
	for _, f := range findings {
		if defines(p, f.Definition) {
			initial = append(initial, f)
		}
	}
	return initial
}

// findingsErr refuses a Finding that names a Definition the checked model does
// not have. In Verify a Definition marks what the correction Implement
// receives: every Finding of a reject names one, a Finding of an approval none.
func (x *execution) findingsErr(s *managerState, tool string, decision Decision, findings []Finding) error {
	reject := x.request.Mandate == Verify && decision == Reject
	for i, f := range findings {
		switch {
		case f.Definition != "" && x.request.Mandate == Verify && decision == Approve:
			return fmt.Errorf("Finding %d of your approval names %s; a Verify approval hands no Finding to Implement, so an optional addition is a Note: leave definition empty, or reject if a change must be made, then %s again", i+1, f.Definition, tool)
		case f.Definition != "" && !defines(x.request.Prepared, f.Definition):
			next := "name an existing Definition as Package/ID or leave definition empty"
			if reject {
				next = x.definitionStep(s)
			}
			return fmt.Errorf("Finding %d names %s, which is not a Definition of the checked model; %s, then %s again", i+1, f.Definition, next, tool)
		case f.Definition == "" && reject:
			return fmt.Errorf("Finding %d of your reject names no Definition; a Verify reject hands its Findings to a new Implement, which receives only Findings that name a Definition of the checked model. %s, then %s again", i+1, x.definitionStep(s), tool)
		}
	}
	return nil
}

// definitionStep names how to find the Definition a reject Finding affects and
// lists the current Definitions of s's area, its subtree and its parents.
func (x *execution) definitionStep(s *managerState) string {
	var near []string
	for _, m := range x.request.Prepared.Topology().Managers() {
		if !related(m.Package, s.manager.Package) {
			continue
		}
		for _, d := range m.Definitions {
			near = append(near, d.Package+"/"+d.ID)
		}
	}
	step := "Name the current Definition whose obligation it affects as Package/ID: for a change, one that the changed artifact's entry in .markitect/registry/ realizes; for a failing declared check, its requiredBy Definition in .markitect/execution.yaml; otherwise a Definition of the affected area. Any Definition of the checked model is accepted"
	if len(near) != 0 {
		step += "; those of your area and its parents are: " + strings.Join(near, ", ")
	}
	return step
}

// related reports whether one Package contains the other or both are the same.
func related(a, b string) bool {
	return a == b || a == "." || b == "." || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

// noteUnbound keeps as Shared Notes the Findings that name no Definition of a
// closed Assessment approval or Verify judgment: noted, not implemented
// unasked. A rejected Assessment and a model blocker hand nothing over and keep
// their Findings in the result only. A Finding longer than one Note continues
// in further Notes; nothing is cut.
func (x *execution) noteUnbound(ctx context.Context) error {
	if x.request.Mandate == Implement || x.result.Decision == ModelBlocker || x.request.Mandate == Assessment && x.result.Decision != Approve {
		return nil
	}
	for _, f := range x.result.Findings {
		if defines(x.request.Prepared, f.Definition) || strings.TrimSpace(f.Text) == "" {
			continue
		}
		parts := noteParts(f.Text)
		for i, part := range parts {
			scope := fmt.Sprintf("Finding of %s Run %s that names no Definition of the checked model: noted, not implemented", x.request.Mandate, x.request.ID)
			if len(parts) > 1 {
				scope += fmt.Sprintf(" (part %d of %d)", i+1, len(parts))
			}
			if _, err := x.executor.Records.Append(ctx, Note{Project: x.request.Project, Run: x.request.ID, Manager: f.Manager, Context: scope, Text: part}); err != nil {
				return err
			}
		}
	}
	return nil
}

// MaxChanges bounds the diff that Verify Root's assignment carries in full, far
// below one provider message even after escaping. A larger diff arrives as its
// complete file summary; the Managers read the diff itself in their copies.
const MaxChanges = 256 << 10

// changes is what Verify Root's assignment carries of diff from the bound
// input commit: the diff itself, or above MaxChanges every changed file with
// its added and removed lines and how to read the complete diff.
func changes(diff, bound string) string {
	if len(diff) <= MaxChanges {
		return diff
	}
	var b strings.Builder
	fmt.Fprintf(&b, "The complete diff from the bound input %s is %d bytes, more than one assignment carries (%d); read it with git diff %s in your copy. Changed files with their added and removed lines:\n", bound, len(diff), MaxChanges, bound)
	file, added, removed := "", 0, 0
	flush := func() {
		if file != "" {
			fmt.Fprintf(&b, "%s +%d -%d\n", file, added, removed)
		}
	}
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			file, added, removed = strings.TrimPrefix(line, "diff --git "), 0, 0
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
		case strings.HasPrefix(line, "+"):
			added++
		case strings.HasPrefix(line, "-"):
			removed++
		}
	}
	flush()
	return b.String()
}

// noteParts splits text at rune boundaries into parts of at most MaxNoteText bytes.
func noteParts(text string) []string {
	var parts []string
	for len(text) > MaxNoteText {
		cut := MaxNoteText
		for !utf8.RuneStart(text[cut]) {
			cut--
		}
		parts = append(parts, text[:cut])
		text = text[cut:]
	}
	return append(parts, text)
}

// ApprovalKey indexes the stored Verify approvals of one judged content, the
// candidate's tree in its object format, under one basis. Commit metadata
// (time, parents, message) is no content: the same captured state committed
// again keeps its key.
func ApprovalKey(basis string, input project.CandidateRef) string {
	sum := sha256.Sum256([]byte(basis + "\x00" + input.Format + "\x00" + input.Tree))
	return hex.EncodeToString(sum[:])
}

// Approval is r's ApprovalKey when r is a completed Verify approval with a basis.
func (r Result) Approval() (string, bool) {
	if r.Mandate != Verify || r.Decision != Approve || r.Technical != Completed || len(r.Errors) != 0 || r.Basis == "" {
		return "", false
	}
	return ApprovalKey(r.Basis, r.Input), true
}

// Verified is the ID of the first stored result that is a closed Verify
// approval of a candidate with exactly input's content under p's basis, or
// empty: the host then states no fact. input is the command's captured
// snapshot, including accounted untracked files; it is compared with the
// candidate the approval judged, never with another candidate. Whether that
// approval was right is not part of the fact.
func Verified(p project.Prepared, input project.CandidateRef, stored []Result) string {
	var managers []string
	for _, m := range p.Topology().Managers() {
		managers = append(managers, m.Package)
	}
	for _, r := range stored {
		if r.Basis != p.Basis() || r.Input.Tree != input.Tree || r.Input.Format != input.Format || r.Decision != Approve {
			continue
		}
		if validateClosed(r, Request{ID: r.ID, Project: input.Project, Mandate: Verify, Input: r.Input}, r.Binding, managers) == nil {
			return r.ID
		}
	}
	return ""
}
