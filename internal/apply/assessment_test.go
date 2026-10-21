package apply_test

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const assessmentMandate = "Assess the bound model before any realization."

func assessmentSpec() run.PassSpec {
	return run.PassSpec{Purpose: "assess", Mandate: assessmentMandate, Review: true, Guidance: "A genuine finding is a material model issue.", Corrections: []run.Purpose{"model_error", "model_gap"}}
}

// assessingAgents follows the Orchestrator's protocol as read-only assessors:
// each decides its children's proposals, submits its basis and then its
// complete result, covering every own obligation with an inspected model
// source. Root can try to ask the user before its basis and end with a model
// error or a confirmed finding. A refused tool call fails the test, except the
// question to the user, whose refusal is kept.
type assessingAgents struct {
	t           *testing.T
	obligations map[string][]string
	source      string
	ask, abort  string
	finding     string
	// raiser raises a genuine finding in the first Pass that Root disputes and
	// the raiser's fresh reviewer disproves in the reassessment.
	raiser  string
	mu      sync.Mutex
	turns   []run.Turn
	refusal string
}

var findingRE = regexp.MustCompile(`Finding (F\d+) \[genuine\]`)

func newAssessingAgents(t *testing.T, req apply.ApplyRequest) *assessingAgents {
	t.Helper()
	cfg, err := apply.BuildApplyConfig(req, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	a := &assessingAgents{t: t, obligations: map[string][]string{}}
	for _, m := range cfg.Team {
		a.obligations[m.Path] = m.Obligations
	}
	for _, e := range cfg.Start.Entries() {
		if name := e.Path().String(); a.source == "" && strings.HasPrefix(name, ".markitect/model/") {
			a.source = name
		}
	}
	if a.source == "" {
		t.Fatal("the start stand has no model file")
	}
	return a
}

func (a *assessingAgents) Run(ctx context.Context, turn run.Turn, tools run.ToolHandler) (run.TurnResult, error) {
	a.mu.Lock()
	a.turns = append(a.turns, turn)
	a.mu.Unlock()
	call := func(name string, args map[string]any) run.ToolAnswer {
		data, _ := json.Marshal(args)
		answer := tools(ctx, name, data)
		if answer.Refused {
			a.t.Errorf("turn %s: %s %v refused: %s", turn.ID, name, args, answer.Text)
		}
		return answer
	}
	for _, m := range decideRE.FindAllStringSubmatch(turn.Prompt, -1) {
		call("report", map[string]any{"kind": "accept", "proposal": m[1]})
	}
	_, rest, _ := strings.Cut(turn.Prompt, "\nNow: ")
	now, _, _ := strings.Cut(rest, "\n")
	root := turn.Manager == "."
	switch {
	case strings.HasPrefix(now, "work on your own contribution"):
		if root && a.ask != "" && a.refusal == "" {
			// A closed Run has no channel to the user: the question is refused.
			data, _ := json.Marshal(map[string]any{"to": "user", "kind": "question", "text": a.ask})
			answer := tools(ctx, "send", data)
			a.refusal = "not refused: " + answer.Text
			if answer.Refused {
				a.refusal = answer.Text
			}
		}
		call("report", map[string]any{"kind": "basis", "summary": "Assessment context of " + turn.Manager + "."})
	case strings.HasPrefix(now, "submit your complete result"), strings.HasPrefix(now, "integrate your children"):
		if root && a.abort != "" {
			call("report", map[string]any{"kind": "problem", "category": "model", "text": a.abort, "abort": true})
			break
		}
		var coverage []map[string]any
		for _, o := range a.obligations[turn.Manager] {
			coverage = append(coverage, map[string]any{"definition": o, "status": "done", "evidence": []map[string]any{{
				"claim": "The statements of " + o + " are resolved.", "source": a.source, "location": o, "reasoning": "The Definition states its public behavior.",
			}}})
		}
		result := map[string]any{"kind": "result", "summary": "Assessed " + turn.Manager + ".", "obligations": coverage, "verdict": "satisfied"}
		first := strings.Contains(turn.ID, "-p1-")
		switch {
		case root && a.finding != "":
			result["verdict"] = "needs_correction"
			result["findings"] = []map[string]any{{"owner": ".", "obligation": a.source, "observed": a.finding, "blocking": true, "correction": "model_gap"}}
		case turn.Manager == a.raiser && first:
			result["verdict"] = "needs_correction"
			result["findings"] = []map[string]any{{"owner": a.raiser, "obligation": a.source, "observed": "The limit is unstated.", "blocking": true}}
		case turn.Manager == a.raiser:
			if id := findingRE.FindStringSubmatch(call("look_up", map[string]any{"topic": "findings"}).Text); id != nil {
				result["closed"] = []map[string]any{{"id": id[1], "as": "disproved", "reason": "The Definition states the limit.", "evidence": a.source}}
			}
		case root && a.raiser != "" && first:
			if id := findingRE.FindStringSubmatch(call("look_up", map[string]any{"topic": "findings"}).Text); id != nil {
				result["verdict"] = "needs_correction"
				result["disputed"] = []map[string]any{{"id": id[1], "reason": "The limit is stated.", "evidence": a.source}}
			}
		}
		call("report", result)
	}
	return run.TurnResult{Conversation: "conv-" + turn.ID, Usage: run.Usage{Model: "m", InputTokens: 10, OutputTokens: 5, ProviderTurns: 1}, Stopped: true}, nil
}

// checkCounter fails every check and counts the requests: an assessment runs none.
type checkCounter struct {
	mu    sync.Mutex
	calls int
}

func (c *checkCounter) Run(ctx context.Context, r run.CheckRequest) (run.CheckReceipt, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return failingRunner{}.Run(ctx, r)
}

func assessmentRequest(t *testing.T) (apply.ApplyRequest, *assessingAgents, *memoryWorkspaces, *checkCounter, *[]string) {
	t.Helper()
	req, _ := layoutRequest(t)
	agents, workspaces, checks := newAssessingAgents(t, req), newMemoryWorkspaces(), &checkCounter{}
	var removed []string
	req.Agents, req.Workspaces, req.Runner, req.Evidence, req.Publisher = agents, workspaces, checks, &memoryEvidence{}, &fakePublisher{}
	req.RemoveHome = func(_ context.Context, home string) error {
		removed = append(removed, home)
		return nil
	}
	return req, agents, workspaces, checks, &removed
}

// The assessment is one review Pass of the captured stand: fresh read-only
// reviewers with the caller's mandate, no required check, no publication, and
// the work volume and the homes removed afterwards.
func TestAssessCapturedRunsOneReadOnlyReviewWithoutChecksOrPublication(t *testing.T) {
	req, agents, workspaces, checks, removed := assessmentRequest(t)
	result, err := apply.AssessCaptured(context.Background(), req, assessmentSpec())
	if err != nil {
		t.Fatal(err)
	}
	outcome := result.Outcome
	if outcome.Status != run.Verified || result.Receipt != nil || len(req.Publisher.(*fakePublisher).requests) != 0 {
		t.Fatalf("status %s (%s), receipt %+v", outcome.Status, outcome.Reason, result.Receipt)
	}
	if len(outcome.Passes) != 1 || outcome.Passes[0].Purpose != "assess" || !outcome.Passes[0].Review || outcome.Passes[0].Verdict != run.Satisfied || len(outcome.Passes[0].Checks) != 0 {
		t.Fatalf("passes %+v", outcome.Passes)
	}
	if checks.calls != 0 {
		t.Fatalf("%d checks ran", checks.calls)
	}
	assigned := map[string]bool{}
	for _, turn := range agents.turns {
		if turn.Kind != run.VerifyTurn || !turn.ReadOnly || slices.ContainsFunc(turn.Mounts, func(m run.PathMount) bool { return m.Writable }) {
			t.Errorf("turn %s is no read-only review turn: %+v", turn.ID, turn)
		}
		if strings.Contains(turn.Prompt, "Pass p1: assess\n"+assessmentMandate) {
			assigned[turn.Manager] = true
		}
	}
	if len(assigned) != len(agents.obligations) {
		t.Fatalf("the mandate reached %v of the %d Managers", assigned, len(agents.obligations))
	}
	if !slices.Contains(workspaces.closed, "run-1-work") || len(*removed) == 0 || !slices.Equal(*removed, outcome.Homes) {
		t.Fatalf("closed %v, removed %v of %v", workspaces.closed, *removed, outcome.Homes)
	}
}

// Root's model-error decision ends the assessment Failed with the model-error
// reason, as in Apply.
func TestAssessCapturedEndsFailedOnRootsModelError(t *testing.T) {
	req, agents, _, _, _ := assessmentRequest(t)
	agents.abort = "catalog/item and lending/borrow state contradicting limits"
	result, err := apply.AssessCaptured(context.Background(), req, assessmentSpec())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome.Status != run.Failed || !strings.HasPrefix(result.Outcome.Reason, run.ModelErrorReason) {
		t.Fatalf("status %s, reason %q", result.Outcome.Status, result.Outcome.Reason)
	}
	if !slices.ContainsFunc(result.Outcome.Problems, func(p run.Problem) bool { return p.Manager == "." && p.Abort && p.Category == "model" }) {
		t.Fatalf("problems %+v", result.Outcome.Problems)
	}
}

// The assessment is a closed Run: Root's question to the user is refused even
// when the caller supplies an owner channel, the caller's owner decisions never
// reach a prompt, and the finding Root confirms seals the Pass as needing
// correction with its class and the assessment's reason, not a model error.
func TestAssessCapturedIsAClosedRunAndKeepsTheConfirmedFinding(t *testing.T) {
	req, agents, _, _, _ := assessmentRequest(t)
	owner := &answeringOwner{}
	req.Owner, req.OwnerDecisions = owner, []string{"The maximum is 60."}
	agents.ask = "Which maximum applies to the public value?"
	agents.finding = "The public maximum is referenced but never stated; 40 and 100 both conform."
	result, err := apply.AssessCaptured(context.Background(), req, assessmentSpec())
	if err != nil {
		t.Fatal(err)
	}
	outcome := result.Outcome
	if outcome.Status != run.Incomplete || outcome.Reason != apply.AssessmentProblemReason {
		t.Fatalf("status %s, reason %q", outcome.Status, outcome.Reason)
	}
	if !strings.Contains(agents.refusal, "Nobody answers questions in this run") || len(owner.asked) != 0 || len(outcome.Questions) != 0 {
		t.Fatalf("refusal %q, asked %+v, questions %+v", agents.refusal, owner.asked, outcome.Questions)
	}
	for _, turn := range agents.turns {
		if strings.Contains(turn.Prompt, "The maximum is 60.") || strings.Contains(turn.Prompt, "\nuser: ") {
			t.Fatalf("turn %s sees the owner channel or a decision:\n%s", turn.ID, turn.Prompt)
		}
	}
	if !slices.ContainsFunc(outcome.Findings, func(f run.Finding) bool {
		return f.RaisedBy == "." && f.Blocking && !f.Closed && f.Observed == agents.finding && f.Correction == "model_gap"
	}) {
		t.Fatalf("findings %+v", outcome.Findings)
	}
	if len(outcome.Passes) != 1 || !outcome.Passes[0].Sealed || outcome.Passes[0].Verdict != run.NeedsCorrection {
		t.Fatalf("passes %+v", outcome.Passes)
	}
}

// A finding that only Root's fresh dispute keeps open is reassessed by a new
// complete review of the unchanged stand, where its raiser disproves it.
func TestAssessCapturedReassessesADisputedFinding(t *testing.T) {
	req, agents, _, _, _ := assessmentRequest(t)
	cfg, err := apply.BuildApplyConfig(req, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range cfg.Team {
		if m.Parent == "." && len(m.Children) == 0 && len(m.Obligations) > 0 {
			agents.raiser = m.Path
			break
		}
	}
	if agents.raiser == "" {
		t.Fatal("no leaf below Root")
	}
	result, err := apply.AssessCaptured(context.Background(), req, assessmentSpec())
	if err != nil {
		t.Fatal(err)
	}
	outcome := result.Outcome
	if outcome.Status != run.Verified || len(outcome.Passes) != 2 || outcome.Passes[0].Verdict != run.NeedsCorrection || outcome.Passes[1].Verdict != run.Satisfied {
		t.Fatalf("status %s (%s), passes %+v", outcome.Status, outcome.Reason, outcome.Passes)
	}
	if len(outcome.Findings) != 1 || !outcome.Findings[0].Closed || outcome.Findings[0].ClosedAs != run.ClosedDisproved || len(outcome.Findings[0].Disputes) != 1 {
		t.Fatalf("findings %+v", outcome.Findings)
	}
}

func TestAssessCapturedRefusesAPassThatIsNoReadOnlyReview(t *testing.T) {
	for name, spec := range map[string]run.PassSpec{
		"writes":    {Purpose: "assess", Mandate: assessmentMandate, Writes: true, Corrections: []run.Purpose{"model_gap"}},
		"no review": {Purpose: "assess", Mandate: assessmentMandate, Corrections: []run.Purpose{"model_gap"}},
		"plans":     {Purpose: "assess", Mandate: assessmentMandate, Plans: true, Corrections: []run.Purpose{"model_gap"}},
	} {
		t.Run(name, func(t *testing.T) {
			req, agents, workspaces, _, _ := assessmentRequest(t)
			if _, err := apply.AssessCaptured(context.Background(), req, spec); err == nil {
				t.Fatal("the Pass was accepted")
			}
			if len(agents.turns) != 0 || len(workspaces.ids) != 0 {
				t.Fatalf("the refused assessment started work: %d turns, volumes %v", len(agents.turns), workspaces.ids)
			}
		})
	}
}
