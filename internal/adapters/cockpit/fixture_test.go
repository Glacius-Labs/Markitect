package cockpit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Synthetic records for the tests. They follow the shapes the run engine
// records (run.Event lines with its data keys) but describe no real run.

var t0 = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

type scenario struct {
	t      *testing.T
	events []run.Event
	clock  time.Duration
}

func newScenario(t *testing.T) *scenario {
	return &scenario{t: t}
}

func (s *scenario) add(kind, pass, manager, turn string, data any) int {
	s.clock += time.Second
	s.events = append(s.events, run.Event{At: t0.Add(s.clock), Kind: kind, Pass: pass, Manager: manager, Turn: turn, Data: data})
	return len(s.events)
}

func (s *scenario) lines() []byte {
	var out []byte
	for _, e := range s.events {
		line, err := json.Marshal(e)
		if err != nil {
			s.t.Fatal(err)
		}
		out = append(append(out, line...), '\n')
	}
	return out
}

func (s *scenario) records() []Record {
	var out []Record
	for i, line := range splitLines(s.lines()) {
		out = append(out, decodeRecord(i+1, line))
	}
	return out
}

func splitLines(data []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			out = append(out, data[start:i])
			start = i + 1
		}
	}
	return out
}

// writeRun writes the scenario as a run directory under a new state directory.
func (s *scenario) writeRun(id string, report any) string {
	state := s.t.TempDir()
	dir := filepath.Join(state, "runs", id)
	if err := os.MkdirAll(filepath.Join(dir, "transcripts"), 0o700); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), s.lines(), 0o600); err != nil {
		s.t.Fatal(err)
	}
	if report != nil {
		data, err := json.Marshal(report)
		if err != nil {
			s.t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "report.json"), data, 0o600); err != nil {
			s.t.Fatal(err)
		}
	}
	return state
}

func (s *scenario) state(pass, manager, from, to, stage, why, waitingFor string) int {
	data := map[string]any{"from": from, "to": to, "stage": stage, "why": why}
	if to == "waiting" {
		data["waitingFor"] = waitingFor
	}
	return s.add("manager.state_changed", pass, manager, "", data)
}

func (s *scenario) turn(pass, manager, id, role string, readOnly bool) {
	s.add("turn.start", pass, manager, id, map[string]any{"role": role, "kind": "execute", "profile": "project-agent", "readOnly": readOnly, "mounts": nil, "home": "h", "resume": "", "workspace": "w"})
}

func (s *scenario) turnEnd(pass, manager, id, role string, reports ...string) {
	s.add("turn.end", pass, manager, id, map[string]any{"role": role, "kind": "execute", "stopped": true, "reports": reports, "reason": "provider result",
		"usage": run.Usage{Model: "test-model", OutputTokens: 10, ProviderTurns: 2}})
}

func (s *scenario) submit(pass, manager, turn, id, kind, purpose, summary string, plan any) {
	env := map[string]any{"summary": summary, "obligations": []any{}}
	if plan != nil {
		env["plan"] = plan
	}
	s.add("proposal.submitted", pass, manager, turn, map[string]any{"proposal": id, "kind": kind, "purpose": purpose, "envelope": env, "findings": nil, "closed": []string{}, "disputed": []string{}})
	s.add("proposal.checked", pass, manager, "", map[string]any{"proposal": id, "passed": true})
}

func (s *scenario) accept(pass, manager, id, kind, by string) int {
	return s.add(kind+".accepted", pass, manager, "", map[string]any{"proposal": id, "by": by, "stand": "stand", "registryShards": map[string]string{}, "registrations": 0})
}

// library builds a synthetic Library run: Root with catalog and lending, a
// Plan whose lending plan depends on catalog's Implement result, a routed
// cross-branch question, parallel Implement turns, a Verify finding, a
// correction Implement and a fresh Verify on a changed candidate.
func library(t *testing.T) (*scenario, map[string]int) {
	s := newScenario(t)
	at := map[string]int{}
	s.add("run.start", "", "", "", map[string]any{"run": "synthetic-library", "managers": []string{".", "catalog", "lending"}, "start": "tree-start", "agentTurns": 60, "passes": 12, "deadline": t0.Add(time.Hour)})
	// Plan p1.
	s.add("pass.start", "p1", "", "", map[string]any{"purpose": "plan", "review": false, "writes": false, "tree": "tree-start"})
	s.state("p1", ".", "waiting", "waiting", "start", "the Pass started", "")
	s.state("p1", "catalog", "waiting", "waiting", "start", "the Pass started", "the accepted basis of Root (.)")
	s.state("p1", "lending", "waiting", "waiting", "start", "the Pass started", "the accepted basis of Root (.)")
	s.add("manager.release", "p1", ".", "", map[string]any{"why": "the Pass started"})
	s.turn("p1", ".", "r-p1-root-m1", "manager", true)
	s.submit("p1", ".", "r-p1-root-m1", "P1", "basis", "plan", "Root's plan basis.", map[string]any{"approach": "coordinate", "dependencies": []any{}})
	s.turnEnd("p1", ".", "r-p1-root-m1", "manager", "basis")
	s.accept("p1", ".", "P1", "basis", "the Orchestrator")
	s.add("manager.release", "p1", "catalog", "", map[string]any{"why": "the basis of its parent was accepted", "parent": "."})
	s.add("manager.release", "p1", "lending", "", map[string]any{"why": "the basis of its parent was accepted", "parent": "."})
	s.submit("p1", "catalog", "", "P2", "result", "plan", "Catalog plan.", map[string]any{"approach": "catalog", "dependencies": []any{}})
	s.submit("p1", "lending", "", "P3", "result", "plan", "Lending plan.", map[string]any{"approach": "lending",
		"dependencies": []any{map[string]any{"owner": "catalog", "need": "the lookup interface", "pass": "implement", "stage": "result"}}})
	s.accept("p1", "catalog", "P2", "result", ".")
	s.accept("p1", "lending", "P3", "result", ".")
	s.add("gate", "p1", "", "", map[string]any{"at": "at the end", "result": map[string]any{"modelCheck": 0, "fileCheck": 0, "tree": "tree-start"}})
	s.add("pass.sealed", "p1", "", "", map[string]any{"purpose": "plan", "verdict": "satisfied", "tree": "tree-start", "managers": []run.ManagerRecord{
		{Manager: ".", State: run.Done, Basis: "P1", Result: "P1", Verdict: run.Satisfied},
		{Manager: "catalog", State: run.Done, Result: "P2", Verdict: run.Satisfied},
		{Manager: "lending", State: run.Done, Result: "P3", Verdict: run.Satisfied}}})
	// Implement p2: parallel work, a cross-branch question routed through Root.
	s.add("pass.start", "p2", "", "", map[string]any{"purpose": "implement", "review": false, "writes": true, "tree": "tree-start"})
	s.state("p2", "lending", "waiting", "waiting", "start", "the Pass started", "the accepted basis of Root (.)")
	s.add("manager.release", "p2", ".", "", map[string]any{"why": "the Pass started"})
	s.submit("p2", ".", "", "P4", "basis", "implement", "Root basis.", nil)
	s.accept("p2", ".", "P4", "basis", "the Orchestrator")
	s.add("manager.release", "p2", "catalog", "", map[string]any{"why": "the basis of its parent was accepted", "parent": "."})
	s.add("manager.release", "p2", "lending", "", map[string]any{"why": "the basis of its parent was accepted", "parent": "."})
	at["released"] = len(s.events)
	s.turn("p2", "catalog", "r-p2-catalog-m1", "manager", false)
	s.state("p2", "catalog", "ready", "working", "own", "a turn started", "")
	s.turn("p2", "lending", "r-p2-lending-m1", "manager", true)
	s.state("p2", "lending", "waiting", "working", "blocked", "a turn started", "")
	at["parallel"] = len(s.events)
	s.add("question.asked", "p2", "lending", "r-p2-lending-m1", run.Question{ID: "Q1", From: "lending", To: ".", Pass: "p2", Text: "Is the catalog lookup case-sensitive?", Asked: t0})
	s.add("question.asked", "p2", ".", "r-p2-root-m2", run.Question{ID: "Q2", From: ".", To: "catalog", About: "Q1", Pass: "p2", Text: "Lending asks: is lookup case-sensitive?", Asked: t0})
	at["asked"] = len(s.events)
	s.add("question.answered", "p2", "catalog", "r-p2-catalog-m1", run.Question{ID: "Q2", From: ".", To: "catalog", About: "Q1", Pass: "p2", Text: "Lending asks: is lookup case-sensitive?", Asked: t0, Answer: "No, keys are normalized.", Answered: true, AnsweredAt: t0})
	s.add("question.answered", "p2", ".", "r-p2-root-m2", run.Question{ID: "Q1", From: "lending", To: ".", Pass: "p2", Text: "Is the catalog lookup case-sensitive?", Asked: t0, Answer: "No, keys are normalized.", Answered: true, AnsweredAt: t0})
	at["answered"] = len(s.events)
	s.turnEnd("p2", "lending", "r-p2-lending-m1", "manager", "wait")
	s.state("p2", "lending", "working", "waiting", "blocked", "its turn ended", "the accepted result of Manager catalog (the lookup interface)")
	at["dependency"] = len(s.events)
	s.turnEnd("p2", "catalog", "r-p2-catalog-m1", "manager", "result")
	s.submit("p2", "catalog", "r-p2-catalog-m1", "P5", "result", "implement", "Catalog done.", nil)
	at["accepted"] = s.accept("p2", "catalog", "P5", "result", ".")
	s.state("p2", "catalog", "waiting", "done", "done", "result accepted", "")
	at["stale"] = len(s.events)
	s.submit("p2", "lending", "", "P6", "result", "implement", "Lending done.", nil)
	s.accept("p2", "lending", "P6", "result", ".")
	s.add("pass.sealed", "p2", "", "", map[string]any{"purpose": "implement", "verdict": "satisfied", "tree": "tree-a", "managers": []run.ManagerRecord{}})
	// Verify p3 on candidate A: a genuine finding for lending.
	s.add("candidate", "p3", "", "", map[string]any{"tree": "tree-a", "workspace": "w", "volume": "v", "entries": 9})
	s.add("check", "p3", "", "", run.CheckResult{Name: "go-test", Passed: true, ExitCode: 0, Output: "ok"})
	s.add("pass.start", "p3", "", "", map[string]any{"purpose": "verify", "review": true, "writes": false, "tree": "tree-a"})
	s.turn("p3", "lending", "r-p3-lending-r1", "reviewer", true)
	finding := run.Finding{ID: "F1", Owner: "lending", RaisedBy: "lending", Obligation: "lending/limit: at most 3 open loans", Observed: "A fourth loan is accepted.", Blocking: true, Correction: run.Implement, Pass: "p3", Review: true, Candidate: "tree-a"}
	at["finding"] = s.add("finding.recorded", "p3", "lending", "", finding)
	s.add("pass.sealed", "p3", "", "", map[string]any{"purpose": "verify", "verdict": "needs_correction", "tree": "tree-a", "managers": []run.ManagerRecord{}})
	// Correction Implement p4 and fresh Verify p5 on candidate B.
	at["correction"] = s.add("pass.start", "p4", "", "", map[string]any{"purpose": "implement", "review": false, "writes": true, "tree": "tree-a"})
	s.add("pass.sealed", "p4", "", "", map[string]any{"purpose": "implement", "verdict": "satisfied", "tree": "tree-b", "managers": []run.ManagerRecord{}})
	s.add("candidate", "p5", "", "", map[string]any{"tree": "tree-b", "workspace": "w", "volume": "v", "entries": 9})
	s.add("pass.start", "p5", "", "", map[string]any{"purpose": "verify", "review": true, "writes": false, "tree": "tree-b"})
	s.add("question.asked", "p5", ".", "r-p5-root-r1", run.Question{ID: "Q3", From: ".", To: "user", Pass: "p5", Text: "May the receipt mention the limit?", Asked: t0})
	closed := finding
	closed.Closed, closed.ClosedPass, closed.ClosedAs, closed.ClosedReason = true, "p5", run.ClosedResolved, "The limit holds now."
	at["closed"] = s.add("finding.closed", "p5", "lending", "", closed)
	s.add("question.lapsed", "p5", ".", "", run.Question{ID: "Q3", From: ".", To: "user", Pass: "p5", Text: "May the receipt mention the limit?", Asked: t0, Lapsed: true})
	s.add("pass.sealed", "p5", "", "", map[string]any{"purpose": "verify", "verdict": "satisfied", "tree": "tree-b", "managers": []run.ManagerRecord{}})
	at["outcome"] = s.add("outcome", "", "", "", map[string]any{"status": "verified", "reason": ""})
	return s, at
}

// applyReport is a report.json of the shape the CLI writes.
func applyReport(outcome, publication string) map[string]any {
	r := map[string]any{"formatVersion": 1, "operation": "apply", "status": "passed", "outcome": outcome,
		"run": map[string]any{"id": "synthetic-library", "status": "verified"}}
	if publication != "" {
		paths := []map[string]any{{"Path": "lending/limit.go", "State": "after"}}
		if publication == "no_change" {
			paths = []map[string]any{}
		}
		r["publication"] = map[string]any{"ID": "synthetic-library", "Outcome": publication, "Paths": paths, "Reason": ""}
	}
	return r
}
