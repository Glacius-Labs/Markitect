package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	AcceptModelTool       = "accept_model"
	ModelProblemTool      = "model_problem"
	AcceptResultTool      = "accept_result"
	RequestCorrectionTool = "request_correction"
)

// ReviewIssue is a concrete required matter, rather than a receipt or a
// per-Definition evidence census. Ordinary observations remain with the Agent.
type ReviewIssue struct {
	Owner      string
	Obligation string
	Expected   string
	Observed   string
}

// AssignmentReview commissions exactly one focused assessment or independent
// Verify. Source must be quiescent. A fresh Git copy starts at Commit, with a
// fresh provider home and conversation; reviewer edits never become the result.
type AssignmentReview struct {
	ID         WorkspaceID
	Purpose    Purpose
	Source     AssignmentWorkspace
	Commit     string
	Profile    string
	Context    string
	Timeout    time.Duration
	Transcript string
}

type ReviewWorkspaces interface {
	AssignmentWorkspaces
	AssignmentForks
}

// AssignmentReviewResult retains the work and native observation even when no
// substantive decision was reached. AcceptedCommit names the source reviewed,
// never the reviewer's own result. No legacy Outcome or publication permit is
// manufactured from a provider completion event.
type AssignmentReviewResult struct {
	Workspace      AssignmentWorkspace
	Handoff        AssignmentHandoff
	Observation    TurnResult
	Closure        TurnResult
	AcceptedCommit string
	Issues         []ReviewIssue
	ModelProblem   bool
}

const modelAssessmentTask = `Assess only the fixed model for material contradictions and missing decisions. Do not implement, infer policy from existing artifacts, or search for optional improvements. A problem is material when conforming realizations could give incompatible public outcomes and the model neither settles nor delegates the choice. Cite the actual model statements and concrete alternatives. Ordinary implementation freedom is no blocker. Use accept_model when no required matter remains, or model_problem for each grounded blocker. Leave your reasoning in your ordinary final summary; no final JSON or status word is required.`

const finalVerifyTask = `Independently verify the named integrated Root commit against the fixed model, across code, tests, documentation and configuration. Derive required outcomes from the model, inspect the actual assertions and public interfaces, and challenge weak green tests, unsupported claims and integration inconsistencies. Use normal tools for focused tests and retain actual observations; earlier tests remain earlier evidence. Do not implement repairs or demand optional improvements. Use accept_result only for this exact commit when no required matter remains, request_correction for concrete defects or unresolved required matters, or model_problem for a material intent defect. A broken test script is ordinary review work: correct your script and reassess its claim, preserving failed observations. Summarize reasoning and open matters in normal prose; no final JSON or status word is required.`

// ReviewAssignment receives work through the normal Git/summary handoff. The
// separate decision tools express the review judgment, not work submission.
// Missing judgment ends incomplete without a reminder or a hidden retry.
func ReviewAssignment(ctx context.Context, workspaces ReviewWorkspaces, agents AssignmentAgents, options AssignmentReview) (result AssignmentReviewResult, err error) {
	if _, bounded := ctx.Deadline(); !bounded || workspaces == nil || agents == nil || options.Source == nil || options.Commit == "" || options.Source.ID() == options.ID || options.Timeout <= 0 || !filepath.IsAbs(options.Transcript) || strings.TrimSpace(options.Context) == "" || (options.Purpose != Assess && options.Purpose != Verify) {
		return result, errors.New("focused review needs a fixed source, distinct identity, model context and finite bounds")
	}
	current, err := options.Source.Commit(ctx, "Stand supplied for independent review")
	if err != nil || current != options.Commit {
		return result, errors.Join(err, errors.New("source no longer names the commissioned review commit"))
	}
	result.Workspace, err = workspaces.ForkAssignment(ctx, options.Source, options.ID)
	if err != nil {
		return result, err
	}
	if result.Workspace == nil || result.Workspace.StartingCommit() != options.Commit {
		return result, errors.New("independent review copy did not start at the commissioned commit")
	}
	accept := AcceptResultTool
	task := finalVerifyTask
	if options.Purpose == Assess {
		accept, task = AcceptModelTool, modelAssessmentTask
	}
	accepted, decided := false, false
	tools := []ToolSpec{
		{Name: accept, Description: "Accept this exact commissioned stand, with your substantive reason.", InputSchema: raw(object([]string{"commit", "reason"}, schema{"commit": str("The commissioned commit."), "reason": str("Why the mandate is met.")}))},
		{Name: ModelProblemTool, Description: "Report a material problem of the fixed model, with its statements and incompatible outcomes.", InputSchema: reviewIssueSchema()},
	}
	if options.Purpose == Verify {
		tools = append(tools, ToolSpec{Name: RequestCorrectionTool, Description: "Return a concrete defect or unresolved required matter to its responsible owner.", InputSchema: reviewIssueSchema()})
	}
	handler := func(_ context.Context, name string, args json.RawMessage) ToolAnswer {
		if name == accept {
			var a struct{ Commit, Reason string }
			if json.Unmarshal(args, &a) != nil || a.Commit != options.Commit || strings.TrimSpace(a.Reason) == "" || len(result.Issues) != 0 {
				return ToolAnswer{Text: "acceptance needs this exact commit, a reason and no open required matters", Refused: true}
			}
			accepted, decided = true, true
			return ToolAnswer{Text: "Review judgment recorded for " + options.Commit}
		}
		if name != ModelProblemTool && (name != RequestCorrectionTool || options.Purpose != Verify) {
			return ToolAnswer{Text: "unknown focused review decision", Refused: true}
		}
		var issue ReviewIssue
		if json.Unmarshal(args, &issue) != nil || strings.TrimSpace(issue.Owner) == "" || strings.TrimSpace(issue.Obligation) == "" || strings.TrimSpace(issue.Expected) == "" || strings.TrimSpace(issue.Observed) == "" {
			return ToolAnswer{Text: "name the owner, obligation, expected outcome and observed problem", Refused: true}
		}
		result.Issues = append(result.Issues, issue)
		result.ModelProblem = result.ModelProblem || name == ModelProblemTool
		accepted, decided = false, true
		return ToolAnswer{Text: "Required matter retained for its owner."}
	}
	a, err := StartAssignment(ctx, workspaces, agents, AssignmentRequest{ID: options.ID, Prepared: result.Workspace, Profile: options.Profile, Home: string(options.ID) + "-home", Transcript: options.Transcript, Tools: tools, Handler: handler})
	if err != nil {
		return result, err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), min(options.Timeout, time.Minute))
		defer cancel()
		var closeErr error
		result.Closure, closeErr = a.Close(closeCtx)
		if !result.Closure.Stopped {
			closeErr = errors.Join(closeErr, errors.New("independent reviewer stop was not observed"))
		}
		err = errors.Join(err, closeErr)
		if err == nil && accepted {
			result.AcceptedCommit = options.Commit
		}
	}()
	result.Observation, err = a.Message(ctx, fmt.Sprintf("%s\nCommissioned commit: %s\n%s", task, options.Commit, options.Context), options.Timeout)
	result.Observation = cloneTurnResult(result.Observation)
	if err != nil {
		return result, err
	}
	if !result.Observation.Completed {
		return result, errors.New("independent review response did not complete")
	}
	result.Handoff, err = a.Submit(ctx)
	if err == nil && !decided {
		err = errors.New("review work was received without an explicit substantive decision")
	}
	return result, err
}

func reviewIssueSchema() json.RawMessage {
	return raw(object([]string{"owner", "obligation", "expected", "observed"}, schema{
		"owner": str("Responsible Manager path."), "obligation": str("The model Definition and relevant statement."),
		"expected": str("What the model requires, or the materially unresolved alternatives."), "observed": str("Actual source, behavior or unresolved required matter and why it blocks."),
	}))
}
