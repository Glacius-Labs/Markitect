package run

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// assessment is the short semantic content left in the provider's final answer.
// Technical bindings and execution observations belong to the Host.
type assessment struct {
	Status       string          `json:"status"`
	Summary      string          `json:"summary"`
	Findings     []findingIn     `json:"findings"`
	Open         []string        `json:"open"`
	Contract     bool            `json:"contract,omitempty"`
	Plan         *plan           `json:"plan,omitempty"`
	Closed       []closeIn       `json:"closed,omitempty"`
	Disputed     []disputeIn     `json:"disputed,omitempty"`
	Dispositions []dispositionIn `json:"dispositions,omitempty"`
	Problem      string          `json:"problem,omitempty"`
}

const finalAssessmentInstruction = `Your observed native Turn end submits your work automatically; no report tool exists. Leave one short final JSON assessment: {"status":"satisfied","summary":"what is established and why","findings":[],"open":[]}. Status satisfied is your explicit judgment that the current Pass mandate is met, not Verified. Use needs_correction with concrete findings, continue for unfinished work, or blocked with the actual wait/problem. Keep required gaps in open; never hide them behind satisfied. A finding names the obligation, expected outcome and observed mismatch; optional suggestions have kind suggestion. The Host captures the exact work, changed paths, actual command/check observations and identities: do not reproduce those fields or invent receipts. Parents still inspect and decide child submissions with decide; use wait only for actual prerequisites. Only when needed, add contract:true with continue for an early contract, plan in a Plan Pass, or specific closed/disputed/dispositions for existing findings or adverse probes. Root states a confirmed material model problem in problem; it ends without publication.`

func readAssessment(final string) (assessment, error) {
	var a assessment
	text := strings.TrimSpace(final)
	if strings.HasPrefix(text, "```") {
		_, text, _ = strings.Cut(text, "\n")
		text = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "```"))
	}
	if err := json.Unmarshal([]byte(text), &a); err != nil {
		return a, fmt.Errorf("the final answer needs a short assessment object: status, summary, findings, open")
	}
	if _, err := toolText("summary", a.Summary); err != nil {
		return a, err
	}
	if a.Findings == nil || a.Open == nil {
		return a, fmt.Errorf("findings and open must be explicit arrays (empty when none)")
	}
	if a.Contract && a.Status != "continue" {
		return a, fmt.Errorf("a needed contract accompanies continuing work, not a completion verdict")
	}
	switch a.Status {
	case string(Satisfied), string(NeedsCorrection), "continue", "blocked":
	default:
		return a, fmt.Errorf("status is satisfied, needs_correction, continue or blocked; no status is inferred from a completion announcement")
	}
	return a, nil
}

// captureTurn records exactly one Host submission per observed Turn end. A
// submission is an observation, including unknown/failed semantic content; only
// the normal acceptance and Verify process can establish a result.
func (e *engine) captureTurn(ts *turnState, res TurnResult, runErr error) {
	ts.commands = res.Commands
	var reason string
	if res.Stopped {
		h, err := e.peek()
		if err != nil {
			reason = err.Error()
		} else {
			ts.harvest = &h
		}
	}
	ts.captureError = reason
	e.mu.Lock()
	defer e.mu.Unlock()
	data := map[string]any{"completed": res.Completed, "stopped": res.Stopped, "final": res.Final, "captureError": reason, "commands": res.Commands}
	data["checks"] = ts.m.p.checks
	data["declaredStatus"] = "unknown"
	if a, err := readAssessment(res.Final); err == nil {
		data["declaredStatus"] = a.Status
	}
	if runErr != nil {
		data["error"] = runErr.Error()
	}
	if ts.harvest != nil {
		stand := ts.m.p.accepted
		if tree, err := overlay(stand, ts.harvest.Tree, ts.m.Path, e.cfg.OwnerOf); err == nil {
			data["candidate"] = tree.Digest().String()
			var paths []string
			for _, change := range snapshot.Compare(stand, tree) {
				paths = append(paths, change.Path().String())
			}
			data["changedPaths"] = paths
		} else {
			data["captureError"] = err.Error()
		}
	}
	var receipts []string
	for _, r := range e.receipts {
		if r.Turn == ts.id {
			receipts = append(receipts, r.ID)
		}
	}
	data["probeReceipts"] = receipts
	_ = e.record(Event{Kind: "turn.submitted", Pass: ts.m.p.id, Manager: ts.m.Path, Turn: ts.id, Data: data})
}

func (e *engine) assessTurnLocked(m *member, ts *turnState, final string) {
	if m.stage == stageStart || m.stale != "" {
		return
	}
	a, err := readAssessment(final)
	if err == nil && ts.captureError != "" {
		err = fmt.Errorf("the work capture is unresolved: %s", ts.captureError)
	}
	// An acknowledgement after acceptance or an idle coordination Turn cannot
	// create another result. Its captured unknown assessment does not undo an
	// earlier independently accepted result either.
	if err != nil && (ts.startedStage != stageOwn && ts.startedStage != stageIntegrate || ts.blocked || m.stage == stageStart || m.stage == stageBlocked || m.stage == stageDone || m.stage == stageDelegated) {
		return
	}
	if err == nil && (ts.blocked || m.stage == stageBlocked) {
		if a.Status == string(Satisfied) || a.Status == string(NeedsCorrection) || a.Contract {
			err = fmt.Errorf("this Turn began blocked and cannot complete productive work")
		} else {
			return
		}
	}
	if err == nil && a.Problem != "" {
		answer := e.problem(m, ts, "model", a.Problem, m.Parent == "")
		if answer.Refused {
			err = fmt.Errorf("%s", answer.Text)
		}
		if e.ctx.Err() != nil {
			return
		}
	}
	if err == nil && a.Status == "blocked" {
		if ts.wait == "" && len(e.demandsLocked(m)) == 0 && e.awaitedLocked(m) == "" {
			err = fmt.Errorf("blocked work needs its actual prerequisite or concrete problem, not an inferred success")
		} else {
			return
		}
	}
	if err == nil && a.Status == "continue" && !a.Contract {
		ts.progress = a.Summary
		return
	}
	if err == nil {
		kind := kindResult
		if a.Contract {
			kind = kindBasis
		}
		in := contribution{Kind: kind, Summary: a.Summary, Findings: a.Findings, Unresolved: a.Open, Plan: a.Plan, Closed: a.Closed, Disputed: a.Disputed, Dispositions: a.Dispositions}
		if kind == kindResult {
			in.Verdict = a.Status
		}
		// Only executed observations of this exact submitted stand are support.
		// Their assertions and semantic relevance remain the reviewers' duty.
		if ts.harvest != nil {
			stand, composeErr := overlay(m.p.accepted, ts.harvest.Tree, m.Path, e.cfg.OwnerOf)
			if composeErr != nil {
				err = composeErr
			} else {
				for _, r := range e.receipts {
					if r.Outcome == ProbePassed && r.Stand == stand.Digest().String() && e.answersForLocked(r, m.Path) {
						kind := "execution"
						if r.Boundary != "" {
							kind = "public_behavior"
						}
						in.Support = append(in.Support, evidenceItem{Kind: kind, Authority: r.Obligation, Boundary: r.Boundary, Claim: r.Claim, Receipt: r.ID, Stand: r.Stand})
					}
				}
			}
		}
		if err == nil {
			answer := e.submit(m, ts, in)
			if !answer.Refused {
				return
			}
			err = fmt.Errorf("%s", answer.Text)
		}
	}
	// One specific semantic correction is allowed. No missing-report loop, and
	// no invented satisfied verdict for an absent or ambiguous final answer.
	m.empty++
	_ = e.record(Event{Kind: "assessment.unresolved", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"reason": err.Error()}})
	if m.empty > 1 {
		e.halt(fmt.Sprintf("the final assessment of %s remains unresolved: %v", m.Path, err))
		return
	}
	m.items = append(m.items, item{text: "Your work was captured. Correct only its final semantic assessment: " + err.Error() + ". Leave a short final JSON object with status, summary, findings and open. Do not call a report tool."})
	ts.semanticMissing = true
}

type actionIn struct {
	Kind     string `json:"kind"`
	Proposal string `json:"proposal"`
	Child    string `json:"child"`
	Reason   string `json:"reason"`
	For      string `json:"for"`
	Stage    string `json:"stage"`
}

var waitSchema = object([]string{"for"}, schema{"for": str("Question, child or Manager needed first."), "stage": str("basis or result, when naming a Manager contract."), "reason": str("Why this is an actual prerequisite.")})
var decisionSchema = object([]string{"kind"}, schema{"kind": schema{"type": "string", "enum": []string{kindAccept, kindReturn, kindReopen, kindRevise}}, "proposal": str("Child proposal for accept or return."), "child": str("Child whose accepted result must reopen."), "reason": str("Concrete reason for return, reopen or revising your contract.")})

func (e *engine) act(m *member, ts *turnState, name string, args json.RawMessage) ToolAnswer {
	var in actionIn
	if err := decodeStrict(args, &in); err != nil {
		return refuse("%v", err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if m.p != e.cur || m.p.sealed {
		return refuse("This Pass no longer accepts actions.")
	}
	if m.stage == stageStart {
		return refuse("This work was reset; its actual prerequisites must be re-established.")
	}
	if name == toolWait {
		answer, kind := e.declareWait(m, ts, in)
		if !answer.Refused {
			ts.actions = append(ts.actions, kind)
		}
		return answer
	}
	if ts.blocked || m.stage == stageBlocked || m.stage == stageStart {
		return refuse("This Turn has no productive decision rights.")
	}
	var answer ToolAnswer
	switch in.Kind {
	case kindAccept, kindReturn:
		answer = e.decide(m, ts, in)
	case kindReopen:
		answer = e.reopen(m, ts, in)
	case kindRevise:
		answer = e.revise(m, ts, in)
	default:
		return refuse("Only a necessary accept, return, reopen or revise decision is available.")
	}
	if !answer.Refused {
		ts.actions = append(ts.actions, in.Kind)
	}
	return answer
}
