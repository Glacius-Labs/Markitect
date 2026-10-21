package run

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// helperRun is one delegated task of a Manager: a helper role instantiated in
// its own container for one turn. paths is its write subset; empty means
// read-only. It starts after the delegating turn ended, so the Manager and the
// helper never hold the same writable files at once.
type helperRun struct {
	id      string
	m       *member
	role    Helper
	task    string
	paths   []PathMount
	running bool
}

// delegate serves the tool delegate: it validates a bounded task for one of
// the Manager's helper roles and its write subset, which must lie in the
// Manager's own writable area, outside .markitect, existing in the shared tree
// and disjoint from the subsets of its other helpers.
func (e *engine) delegate(ctx context.Context, m *member, ts *turnState, args json.RawMessage) ToolAnswer {
	var in struct {
		Role  string   `json:"role"`
		Task  string   `json:"task"`
		Paths []string `json:"paths"`
	}
	if err := decodeStrict(args, &in); err != nil {
		return refuse("%v", err)
	}
	task, err := toolText("task", in.Task)
	if err != nil {
		return refuse("%v", err)
	}
	i := slices.IndexFunc(m.Helpers, func(h Helper) bool { return h.Role == in.Role })
	if i < 0 {
		var roles []string
		for _, h := range m.Helpers {
			roles = append(roles, h.Role)
		}
		return refuse("role must be one of %s.", strings.Join(roles, ", "))
	}
	role := m.Helpers[i]
	var tree snapshot.Tree
	if len(in.Paths) > 0 {
		e.mu.Lock()
		stage, writes := m.stage, m.p.spec.Writes
		e.mu.Unlock()
		switch {
		case !role.Writes:
			return refuse("The role %s works read-only; delegate it a task without paths.", role.Role)
		case !writes || stage != stageOwn && stage != stageIntegrate:
			return refuse("You can delegate writes only while you work on your own contribution in a Pass that changes files.")
		}
		h, err := e.peek()
		if err != nil {
			return refuse("The working tree could not be observed: %v. Try again.", err)
		}
		tree = h.Tree
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if m.p != e.cur || m.stage != stageOwn && m.stage != stageIntegrate {
		return refuse("You delegate while you work on your basis or your complete result.")
	}
	if ts.proposal != nil {
		return refuse("You submitted %s in this turn; delegate before you submit.", ts.proposal.id)
	}
	active := len(m.helpers) + len(ts.delegations)
	if active >= m.MaxHelpers {
		return refuse("You may have at most %d helpers at a time; wait for the results of the running ones.", m.MaxHelpers)
	}
	var mounts []PathMount
	for _, raw := range in.Paths {
		mount, err := e.delegatedPathLocked(m, ts, tree, raw)
		if err != nil {
			return refuse("%v", err)
		}
		for _, other := range mounts {
			if overlaps(other.Path, mount.Path) {
				return refuse("paths %q and %q overlap.", other.Path, mount.Path)
			}
		}
		mounts = append(mounts, mount)
	}
	m.nextHelper++
	h := &helperRun{id: fmt.Sprintf("h%d", m.nextHelper), m: m, role: role, task: task, paths: mounts}
	ts.delegations = append(ts.delegations, h)
	var paths []string
	for _, p := range mounts {
		paths = append(paths, p.Path)
	}
	_ = e.record(Event{Kind: "helper.delegated", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"helper": h.id, "role": role.Role, "task": task, "paths": paths}})
	return ToolAnswer{Text: fmt.Sprintf("Delegated %s to %s %s; it starts when this turn ends and its result reaches you. Until then its paths are read-only for you.", task, role.Role, h.id)}
}

// delegatedPathLocked validates one path of a delegated write subset.
func (e *engine) delegatedPathLocked(m *member, ts *turnState, tree snapshot.Tree, raw string) (PathMount, error) {
	name := strings.TrimSuffix(strings.TrimSpace(raw), "/")
	path, err := snapshot.NewPath(name)
	if err != nil {
		return PathMount{}, fmt.Errorf("path %q is not a clean repository-relative path", raw)
	}
	if name == ".markitect" || strings.HasPrefix(name, ".markitect/") {
		return PathMount{}, fmt.Errorf("path %q lies in .markitect: protected inputs and your registry stay with you", raw)
	}
	_, isFile := tree.Lookup(path)
	isDir := !isFile && slices.ContainsFunc(tree.Entries(), func(en snapshot.Entry) bool { return strings.HasPrefix(en.Path().String(), name+"/") }) ||
		slices.ContainsFunc(m.Write.Mounts, func(x PathMount) bool { return !x.File && x.Path == name })
	if !isFile && !isDir {
		return PathMount{}, fmt.Errorf("path %q does not exist in the shared tree; delegate an existing file or a directory that holds files", raw)
	}
	if !writableIn(m.Write, name) || isFile && e.cfg.OwnerOf(path) != m.Path {
		return PathMount{}, fmt.Errorf("path %q is not in your own writable area", raw)
	}
	for _, h := range append(slices.Clone(m.helpers), ts.delegations...) {
		for _, other := range h.paths {
			if overlaps(other.Path, name) {
				return PathMount{}, fmt.Errorf("path %q overlaps %q of your helper %s", raw, other.Path, h.id)
			}
		}
	}
	return PathMount{Path: name, Writable: true, File: isFile}, nil
}

// writableIn reports whether a write plan lets its turns write path: the base
// first, then every mount in order overrides it for its directory or file.
func writableIn(plan WritePlan, path string) bool {
	ok := plan.Base
	for _, m := range plan.Mounts {
		if path == m.Path || strings.HasPrefix(path, m.Path+"/") {
			ok = m.Writable
		}
	}
	return ok
}

// overlaps reports whether one path is the other or lies below it.
func overlaps(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

// startHelpersLocked starts the delegated helpers of every Manager whose
// delegating turn ended.
func (e *engine) startHelpersLocked(p *pass) {
	for _, m := range p.members {
		if m.running {
			continue
		}
		for _, h := range m.helpers {
			if h.running {
				continue
			}
			if !e.reserveTurnLocked() {
				e.addProblemLocked(Problem{Manager: m.Path, Category: "technical", Text: "no Agent turn is left in the budget for a delegated helper"})
				e.halt(fmt.Sprintf("the agent turn budget of %d ended", e.cfg.Budget.AgentTurns))
				return
			}
			h.running = true
			p.running++
			kind := ExecuteTurn
			if p.spec.Review {
				kind = VerifyTurn
			}
			id := fmt.Sprintf("%s-%s-%s-%s", e.cfg.RunID, p.id, m.slug, h.id)
			home := id + "-home"
			ts := &turnState{m: m, h: h, id: id, role: h.role.Role, kind: kind, mode: "helper"}
			turn := Turn{Kind: kind, Profile: h.role.Profile, Workspace: p.volume, ReadOnly: true, Mounts: helperMounts(m.Write, h.paths), Home: home,
				Prompt: e.helperPromptLocked(h), Tools: helperTools(len(h.paths) > 0)}
			e.useHomeLocked(home)
			_ = e.record(Event{Kind: "helper.start", Pass: p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"helper": h.id, "role": h.role.Role}})
			e.wg.Add(1)
			go e.helperTurn(h, ts, turn)
		}
	}
}

// helperMounts is the write plan of a helper: the base is read-only, each
// delegated path writable, and every read-only area of its Manager's plan
// inside a delegated directory stays read-only.
func helperMounts(plan WritePlan, paths []PathMount) []PathMount {
	mounts := slices.Clone(paths)
	for _, m := range plan.Mounts {
		if m.Writable {
			continue
		}
		for _, d := range paths {
			if !d.File && strings.HasPrefix(m.Path, d.Path+"/") {
				mounts = append(mounts, m)
				break
			}
		}
	}
	slices.SortFunc(mounts, func(a, b PathMount) int { return strings.Compare(a.Path, b.Path) })
	return mounts
}

func (e *engine) helperTurn(h *helperRun, ts *turnState, turn Turn) {
	defer e.wg.Done()
	res, err := e.runTurn(ts, turn, e.helperHandler(h, ts))
	e.mu.Lock()
	defer e.mu.Unlock()
	defer e.cond.Broadcast()
	m := h.m
	m.p.running--
	m.helpers = slices.DeleteFunc(m.helpers, func(x *helperRun) bool { return x == h })
	outcome := "reported: " + ts.helperSummary
	switch {
	case err != nil:
		outcome = fmt.Sprintf("failed: %v", err)
	case ts.helperSummary == "":
		outcome = "ended without a report"
	}
	_ = e.record(Event{Kind: "helper.end", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"helper": h.id, "role": h.role.Role, "outcome": outcome}})
	if e.ctx.Err() != nil || !res.Stopped {
		return
	}
	e.deliverLocked(m, item{text: fmt.Sprintf("Your %s %s (task: %s) %s", h.role.Role, h.id, h.task, outcome)})
	e.scheduleLocked()
}

// helperHandler serves the tools of a helper's turn: its result report, the
// look-up and, with a write subset, the check of its accounting.
func (e *engine) helperHandler(h *helperRun, ts *turnState) ToolHandler {
	return func(ctx context.Context, name string, args json.RawMessage) ToolAnswer {
		switch name {
		case toolReport:
			var in struct {
				Summary string `json:"summary"`
			}
			if err := decodeStrict(args, &in); err != nil {
				return refuse("%v", err)
			}
			summary, err := toolText("summary", in.Summary)
			if err != nil {
				return refuse("%v", err)
			}
			e.mu.Lock()
			ts.helperSummary = summary
			ts.reports = append(ts.reports, "result")
			e.mu.Unlock()
			return ToolAnswer{Text: "Recorded for your Manager. A later report in this turn replaces it."}
		case toolLookUp:
			return e.lookUp(h.m, nil, args)
		case toolCheck:
			if len(h.paths) > 0 {
				return e.check(ctx, h.m, ts, args)
			}
		}
		var names []string
		for _, t := range helperTools(len(h.paths) > 0) {
			names = append(names, fmt.Sprintf("%q", t.Name))
		}
		return refuse("Unknown tool %q; you have %s.", name, strings.Join(names, ", "))
	}
}
