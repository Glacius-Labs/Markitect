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
// not have. In Verify every Finding of a reject must name one: the reject hands
// its Findings to a new Implement, which receives no other.
func (x *execution) findingsErr(tool string, decision Decision, findings []Finding) error {
	for i, f := range findings {
		switch {
		case f.Definition != "" && !defines(x.request.Prepared, f.Definition):
			return fmt.Errorf("Finding %d names %s, which is not a Definition of the checked model; name an existing Definition as Package/ID from your selected context or leave definition empty, then %s again", i+1, f.Definition, tool)
		case f.Definition == "" && x.request.Mandate == Verify && decision == Reject:
			return fmt.Errorf("Finding %d of your reject names no Definition; a Verify reject hands every Finding to a new Implement, which receives only Findings that name a Definition of the checked model. Name the Definition it affects: for a change, one that the changed artifact's entry in .markitect/registry/ realizes; for a failing declared check, the requiredBy Definition of that check in .markitect/execution.yaml. Then %s again", i+1, tool)
		}
	}
	return nil
}

// noteUnbound keeps the Findings of a closed Assessment or Verify that name no
// Definition as Shared Notes: noted, not implemented unasked. A Finding longer
// than one Note continues in further Notes; nothing is cut.
func (x *execution) noteUnbound(ctx context.Context) error {
	if x.request.Mandate == Implement {
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

// ApprovalKey indexes the stored Verify approvals of one input under one basis.
func ApprovalKey(basis string, input project.CandidateRef) string {
	sum := sha256.Sum256([]byte(basis + "\x00" + input.Commit + "\x00" + input.Tree))
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
// approval of exactly input under p's basis, or empty: the host then states no
// fact. Whether that approval was right is not part of the fact.
func Verified(p project.Prepared, input project.CandidateRef, stored []Result) string {
	var managers []string
	for _, m := range p.Topology().Managers() {
		managers = append(managers, m.Package)
	}
	for _, r := range stored {
		if r.Basis != p.Basis() || r.Input.Commit != input.Commit || r.Input.Tree != input.Tree || r.Input.Format != input.Format || r.Decision != Approve {
			continue
		}
		if validateClosed(r, Request{ID: r.ID, Project: input.Project, Mandate: Verify, Input: r.Input}, r.Binding, managers) == nil {
			return r.ID
		}
	}
	return ""
}
