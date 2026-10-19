package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// check serves the Execute tool "check": it peeks the shared work volume while
// the writers keep running, accounts the peeked tree and answers with the
// findings, those of the calling Manager first. The answer is advice: a
// candidate is only ever harvested after every Execute turn ended. A peek that
// is uncertain or incomplete is refused with the reason, which is not a failure
// of the run.
func (e *engine) check(ctx context.Context, m *mgr, ts *turnState, args json.RawMessage) ToolAnswer {
	var in struct {
		Scope string `json:"scope"`
	}
	if trimmed := bytes.TrimSpace(args); len(trimmed) > 0 && string(trimmed) != "null" {
		if err := decodeStrict(args, &in); err != nil {
			return refuse("%v", err)
		}
	}
	if in.Scope != "" && in.Scope != "own" && in.Scope != "all" {
		return refuse("scope must be \"own\" or \"all\".")
	}
	e.mu.Lock()
	finished := ts.finished
	e.mu.Unlock()
	if finished {
		return refuse("This turn is already finished; no further calls are accepted.")
	}
	h, err := e.cfg.Workspaces.Peek(ctx, HarvestRequest{ID: e.volumeID("work"), MaxEntries: e.cfg.MaxEntries, MaxBytes: e.cfg.MaxBytes})
	switch {
	case err != nil:
		return e.refuseCheck(m, ts, fmt.Sprintf("The working tree could not be observed: %v.", err))
	case h.Uncertain:
		return e.refuseCheck(m, ts, fmt.Sprintf("The working tree could not be observed completely: %s. Try again later.", h.Reason))
	case len(h.Unsupported) > 0:
		shown := h.Unsupported[:min(len(h.Unsupported), maxListedNames)]
		listing := fmt.Sprintf("%q", shown)
		if len(h.Unsupported) > len(shown) {
			listing += fmt.Sprintf(" and %d more", len(h.Unsupported)-len(shown))
		}
		return e.refuseCheck(m, ts, fmt.Sprintf("The working tree holds %d unsupported entries (symbolic links, devices or other special entries, or names that are not valid paths), so it cannot be checked completely: %s. Remove them and try again.", len(h.Unsupported), listing))
	}

	var own, others []Finding
	for _, f := range e.cfg.Account(h.Tree) {
		if e.byPath[f.Owner] == nil {
			f.Observed = fmt.Sprintf("%s (reported for unknown owner %q)", f.Observed, f.Owner)
			f.Owner = "."
		}
		if f.Owner == m.Path {
			own = append(own, f)
		} else {
			others = append(others, f)
		}
	}
	_ = e.record(Event{Kind: "check.requested", Manager: m.Path, Turn: ts.id, Round: ts.round, Data: map[string]any{"scope": in.Scope, "own": len(own), "others": len(others)}})

	var b strings.Builder
	b.WriteString("Findings on the current /work tree. Other Managers may still be writing; this is advice, the candidate is judged later.")
	if len(own) == 0 {
		b.WriteString("\nNone for you.")
	} else {
		listFindings(&b, "Yours:", own, false)
	}
	switch {
	case len(others) == 0:
	case in.Scope == "all":
		listFindings(&b, "Of other Managers:", others, true)
	default:
		fmt.Fprintf(&b, "\n%d finding(s) of other Managers (call check with scope \"all\" to list them).", len(others))
	}
	return ToolAnswer{Text: b.String()}
}

// refuseCheck records a refused use and answers with the reason.
func (e *engine) refuseCheck(m *mgr, ts *turnState, reason string) ToolAnswer {
	_ = e.record(Event{Kind: "check.requested", Manager: m.Path, Turn: ts.id, Round: ts.round, Data: map[string]any{"own": 0, "others": 0, "refused": reason}})
	return refuse("%s", reason)
}

// listFindings writes at most maxListedPaths findings, each as one entry.
func listFindings(b *strings.Builder, title string, findings []Finding, withOwner bool) {
	var lines []string
	for _, f := range findings[:min(len(findings), maxListedPaths)] {
		line := strings.TrimSpace(f.Observed)
		if obligation := strings.TrimSpace(f.Obligation); obligation != "" {
			line = obligation + ": " + line
		}
		if withOwner {
			line = fmt.Sprintf("[%s] %s", f.Owner, line)
		}
		lines = append(lines, line)
	}
	if len(findings) > len(lines) {
		lines = append(lines, fmt.Sprintf("and %d more", len(findings)-len(lines)))
	}
	list(b, title, lines)
}
