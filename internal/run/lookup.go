package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Look-up topics.
var lookUpTopics = []string{"brief", "definition", "owner", "registry", "state", "plan", "basis", "findings", "checks", "questions", "probe", "method"}

// lookUp serves the tool look_up: structural facts from the Orchestrator's
// state with their source and stand, so Managers ask deterministic tools for
// known facts and the tree for decisions.
func (e *engine) lookUp(m *member, ts *turnState, args json.RawMessage) ToolAnswer {
	var in struct {
		Topic   string `json:"topic"`
		ID      string `json:"id"`
		Path    string `json:"path"`
		Manager string `json:"manager"`
	}
	if err := decodeStrict(args, &in); err != nil {
		return refuse("%v", err)
	}
	if !slices.Contains(lookUpTopics, in.Topic) {
		return refuse("topic must be one of %s.", strings.Join(lookUpTopics, ", "))
	}
	switch in.Topic {
	case "method":
		text, ok := methods[strings.TrimSpace(in.ID)]
		if !ok {
			return refuse("id names a method: one of %s.", strings.Join(methodNames(), ", "))
		}
		return ToolAnswer{Text: text}
	case "brief":
		var b strings.Builder
		e.standingLocked(&b, m)
		return ToolAnswer{Text: "Your standing context of this run, from its bound inputs:\n" + b.String()}
	case "definition":
		d, ok := e.cfg.Definition(strings.TrimSpace(in.ID))
		if !ok {
			return refuse("%q is no Definition of the bound model.", in.ID)
		}
		label := "context owned by " + d.Owner
		if d.Owner == m.Path {
			label = "your own obligation"
		}
		text := fmt.Sprintf("%s (%s, bound model):\n%s", d.ID, label, d.Text)
		turn := ""
		if ts != nil {
			turn = ts.id
		}
		_ = e.record(Event{Kind: "lookup.definition", Pass: m.p.id, Manager: m.Path, Turn: turn, Data: map[string]any{"id": d.ID, "owner": d.Owner, "answer": textDigest(text), "bytes": len(text)}})
		return ToolAnswer{Text: text}
	case "owner":
		path, err := snapshot.NewPath(strings.TrimSuffix(strings.TrimSpace(in.Path), "/"))
		if err != nil {
			return refuse("path must be a repository-relative path.")
		}
		owner := e.cfg.OwnerOf(path)
		if owner == "" {
			return ToolAnswer{Text: fmt.Sprintf("%s has no owner under the declared ownership: nobody may create it.", path)}
		}
		return ToolAnswer{Text: fmt.Sprintf("%s is owned by %s (declared ownership of the bound run).", path, owner)}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	p := m.p
	switch in.Topic {
	case "probe":
		r := e.receiptLocked(strings.TrimSpace(in.ID))
		if r == nil {
			return refuse("id names a probe receipt (R...) of this run.")
		}
		var b strings.Builder
		writeReceipt(&b, r, true)
		return ToolAnswer{Text: strings.TrimRight(b.String(), "\n")}
	case "registry":
		view := e.cfg.Registry(p.accepted)
		path := strings.TrimSpace(in.Path)
		if path == "" {
			counts := map[string]int{}
			for _, r := range view.Registrations {
				counts[r.Owner]++
			}
			var lines []string
			for _, shard := range slices.Sorted(mapKeys(view.Shards)) {
				lines = append(lines, fmt.Sprintf("%s at version %s", shard, view.Shards[shard][:min(12, len(view.Shards[shard]))]))
			}
			for _, owner := range slices.Sorted(mapKeys(counts)) {
				lines = append(lines, fmt.Sprintf("%s: %d registered artifacts", owner, counts[owner]))
			}
			var b strings.Builder
			fmt.Fprintf(&b, "Derived registry view of the accepted stand %s (read-only, assembled by the Orchestrator from checked local versions).", p.accepted.Digest().String()[:12])
			list(&b, "Included:", lines)
			return ToolAnswer{Text: b.String()}
		}
		for _, r := range view.Registrations {
			if r.Path == path {
				return ToolAnswer{Text: fmt.Sprintf("%s is registered by %s (entry %s) as %s realizing %s, in the accepted stand.", r.Path, r.Owner, r.Entry, r.Form, strings.Join(r.Realizes, ", "))}
			}
		}
		return ToolAnswer{Text: fmt.Sprintf("%s has no registration in the accepted stand; a new file is registered by its owner before its basis or result is submitted.", path)}
	case "state":
		var lines []string
		for _, x := range p.members {
			line := fmt.Sprintf("%s: %s (%s)", x.Path, x.state, x.stage)
			if x.state == Waiting && x.waiting != "" {
				line += ", waiting for " + x.waiting
			}
			lines = append(lines, line)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Pass %s (%s):", p.id, p.spec.Purpose)
		list(&b, "", lines)
		return ToolAnswer{Text: b.String()}
	case "plan":
		target := m.Path
		if in.Manager != "" {
			target = in.Manager
		}
		pr := e.plans[target]
		if pr == nil {
			if x := p.byPath[target]; x != nil && p.spec.Plans {
				pr = x.basis
				if x.result != nil {
					pr = x.result
				}
			}
		}
		if pr == nil || pr.env.Plan == nil {
			return ToolAnswer{Text: fmt.Sprintf("No accepted plan of %s is available.", target)}
		}
		return ToolAnswer{Text: renderPlan(target, pr)}
	case "basis":
		target := m.Parent
		if in.Manager != "" {
			target = in.Manager
		}
		x := p.byPath[target]
		if x == nil || x.basis == nil {
			return ToolAnswer{Text: fmt.Sprintf("No accepted basis of %q is available in %s.", target, p.id)}
		}
		return ToolAnswer{Text: e.renderAcceptedLocked(x.basis)}
	case "findings":
		var lines []string
		for _, f := range e.findings {
			if !f.Closed && (e.within(f.Owner, m.Path) || f.RaisedBy == m.Path) {
				lines = append(lines, renderFinding(f))
			}
		}
		if len(lines) == 0 {
			return ToolAnswer{Text: "No open finding concerns you or your subtree."}
		}
		var b strings.Builder
		list(&b, "Open findings of your responsibility:", lines)
		return ToolAnswer{Text: b.String()}
	case "checks":
		var lines []string
		for _, c := range e.cfg.Checks {
			line := fmt.Sprintf("%s (required by %s)", c.Name, c.RequiredBy)
			if r := e.lastCheckLocked(c.Name); r != nil {
				line += fmt.Sprintf(": last result passed=%t", r.Passed)
			}
			if e.probesAvailable() {
				prefix := strings.Join(append([]string{c.Request.Executable}, c.Request.Args[:c.ProfileArgs]...), " ")
				line += fmt.Sprintf("\n  A probe with check %q runs `%s ARGS` in %s with your probe file placed there, without network, on a disposable copy.", c.Name, prefix, c.Request.Directory)
			}
			lines = append(lines, line)
		}
		if len(lines) == 0 {
			return ToolAnswer{Text: "The project declares no required checks."}
		}
		var b strings.Builder
		list(&b, "Required checks:", lines)
		return ToolAnswer{Text: b.String()}
	case "questions":
		var lines []string
		for _, q := range e.questions {
			if q.From == m.Path || q.To == m.Path {
				line := fmt.Sprintf("%s %s -> %s: %s", q.ID, q.From, q.To, q.Text)
				switch {
				case q.Answered:
					line += "\n  Answer: " + q.Answer
				case q.Lapsed:
					line += " (lapsed)"
				default:
					line += " (open)"
				}
				lines = append(lines, line)
			}
		}
		if len(lines) == 0 {
			return ToolAnswer{Text: "No questions to or from you."}
		}
		var b strings.Builder
		list(&b, "Questions:", lines)
		return ToolAnswer{Text: b.String()}
	}
	return refuse("unknown topic")
}

func (e *engine) lastCheckLocked(name string) *CheckResult {
	for i := len(e.passes) - 1; i >= 0; i-- {
		for _, c := range e.passes[i].checks {
			if c.Name == name {
				return &c
			}
		}
	}
	return nil
}

func mapKeys[V any](m map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// check serves the tool check. With a probe it runs one behavioral probe and
// answers with its receipt. Otherwise it observes the shared work volume while
// other writers keep running, File checks the composition of the accepted stand
// with the caller's own live area and answers with the findings. The File check
// is advice: only the Orchestrator's check after the caller's writers stopped
// gates a submission.
func (e *engine) check(ctx context.Context, m *member, ts *turnState, args json.RawMessage) ToolAnswer {
	var in struct {
		Scope string   `json:"scope"`
		Probe *probeIn `json:"probe"`
	}
	if trimmed := bytes.TrimSpace(args); len(trimmed) > 0 && string(trimmed) != "null" {
		if err := decodeStrict(args, &in); err != nil {
			return refuse("%v", err)
		}
	}
	switch {
	case in.Probe != nil && in.Scope != "":
		return refuse("check runs either a probe or the File check of a scope, not both.")
	case in.Probe != nil && ts.h != nil:
		return refuse("A helper runs no probes; report what to probe to your Manager.")
	case in.Probe != nil:
		return e.probe(ctx, m, ts, *in.Probe)
	case !m.p.spec.Writes:
		return refuse("This Pass has no File check of your area; give a probe.")
	case in.Scope != "" && in.Scope != "own" && in.Scope != "all":
		return refuse("scope must be \"own\" or \"all\".")
	}
	h, err := e.cfg.Workspaces.Peek(ctx, HarvestRequest{ID: e.workID, MaxEntries: e.cfg.MaxEntries, MaxBytes: e.cfg.MaxBytes})
	switch {
	case err != nil:
		return e.refuseCheck(m, ts, fmt.Sprintf("The working tree could not be observed: %v.", err))
	case h.Uncertain:
		return e.refuseCheck(m, ts, fmt.Sprintf("The working tree could not be observed completely: %s. Try again later.", h.Reason))
	case len(h.Unsupported) > 0:
		return e.refuseCheck(m, ts, fmt.Sprintf("The working tree holds %d unsupported entries (symbolic links, devices or names that are not valid paths), so it cannot be checked completely: %s. Remove them and try again.", len(h.Unsupported), listNames(h.Unsupported)))
	}
	e.mu.Lock()
	accepted := m.p.accepted
	e.mu.Unlock()
	composite, err := overlay(accepted, h.Tree, m.Path, e.cfg.OwnerOf)
	if err != nil {
		return e.refuseCheck(m, ts, err.Error())
	}
	var own, others []Finding
	for _, f := range e.cfg.FileCheck(composite) {
		if f.Owner == m.Path {
			own = append(own, f)
		} else {
			others = append(others, f)
		}
	}
	_ = e.record(Event{Kind: "check.requested", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"scope": in.Scope, "own": len(own), "others": len(others)}})
	var b strings.Builder
	b.WriteString("File check of your current area composed with the accepted stand. This is advice; your submission is checked after your turn ends.")
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
func (e *engine) refuseCheck(m *member, ts *turnState, reason string) ToolAnswer {
	_ = e.record(Event{Kind: "check.requested", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"own": 0, "others": 0, "refused": reason}})
	return refuse("%s", reason)
}

// listFindings writes at most maxListed findings, each as one entry.
func listFindings(b *strings.Builder, title string, findings []Finding, withOwner bool) {
	var lines []string
	for _, f := range findings[:min(len(findings), maxListed)] {
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
