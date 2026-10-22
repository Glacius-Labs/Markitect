package apply

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type RealizationWorkspaces interface {
	run.ReviewWorkspaces
	run.AssignmentForks
	run.AssignmentUpdates
	run.AssignmentCommitReader
}

// AssignmentRealizationRequest is the explicit caller boundary for DEC-067.
// One finite context and message allowance cover assessment, implementation,
// questions, corrections and Verify together. It has no Publisher and does not
// reinterpret the legacy invocation's Pass/Turn counters or report format.
type AssignmentRealizationRequest struct {
	ID             string
	Build          string
	Prepared       CapturedSettingsAttempt
	OwnerRequest   string
	MaxMessages    int
	MessageTimeout time.Duration
	MaxEntries     int
	MaxBytes       int64
	TranscriptDir  string
	Workspaces     RealizationWorkspaces
	Agents         run.AssignmentAgents
	Integration    run.AssignmentIntegration
	Checks         run.Checks
	CheckRequest   func(CheckSelection) (run.CheckRequest, error)
	CreateHome     func(context.Context, string) error
}

type AssignmentCheckObservation struct {
	Commit  string
	Name    string
	Request run.CheckRequest
	Receipt run.CheckReceipt
	Error   string
}

// AssignmentRealizationResult deliberately has no legacy Outcome or publication
// permit. AcceptedCommit is set only after independent Verify and observed
// assignment shutdown. Work, homes, prior reviews/checks and partial errors stay
// inspectable; the caller owns eventual workspace/home cleanup.
type AssignmentRealizationResult struct {
	ID                string
	Build             string
	ModelHead         string
	StartingWorkspace run.AssignmentWorkspace
	Assessment        run.AssignmentReviewResult
	Work              *run.AssignmentTree
	Root              run.AssignmentHandoff
	Tree              snapshot.Tree
	TreeCommit        string
	ModelProblem      *run.ReviewIssue
	Reviews           []run.AssignmentReviewResult
	Checks            []AssignmentCheckObservation
	AcceptedCommit    string
	NoChange          bool
	Messages          int
}

var realizationID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,49}$`)

const assignmentImplementTask = `Implement the fixed model in your own responsibility. Planning and foundations belong to this work. Preserve conforming artifacts; no change is a valid result. Change only an observed model violation or a missing required realization, including the necessary tests, documentation and configuration. Delegate every required child, inspect actual child diffs and assertions, and integrate their accepted results. Check changed interfaces and shared invariants with relevant integration tests; do not routinely rerun every child test. Responsibility and fixed intent are working rules, not per-file ACLs. Do not invent policy or optional improvements. Escalate material model problems through your parent; as Root, use model_problem when you confirm one, ending affected work so intent can be corrected outside execution. Summarize changes, actual observations and open matters in ordinary prose; the current Git stand is your handoff.`

func RealizeAssignments(ctx context.Context, req AssignmentRealizationRequest) (result AssignmentRealizationResult, err error) {
	result.ID, result.Build = req.ID, req.Build
	if _, bounded := ctx.Deadline(); !bounded || ctx.Err() != nil || !realizationID.MatchString(req.ID) || strings.TrimSpace(req.Build) == "" || req.MaxMessages < 1 || req.MessageTimeout <= 0 || req.MaxEntries < 1 || req.MaxEntries > 1<<20 || req.MaxBytes < 1 || req.MaxBytes > 1<<32 || !filepath.IsAbs(req.TranscriptDir) || req.Workspaces == nil || req.Agents == nil || req.Integration == nil || req.CreateHome == nil {
		return result, errors.New("assignment realization needs prepared inputs, concrete ports and one finite invocation allowance")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	team, modelContext, candidates, checks, err := prepareAssignments(req)
	if err != nil {
		return result, err
	}
	result.ModelHead = req.Prepared.Structural().Model().Inputs().Head
	allowance := &realizationAgents{agents: req.Agents, max: req.MaxMessages}
	defer func() {
		if result.Work != nil {
			err = errors.Join(err, result.Work.Close())
			if problem := result.Work.ModelProblem(); problem != nil {
				result.ModelProblem = problem
				err = errors.Join(err, run.ErrAssignmentModelProblem)
			}
		}
		result.Messages = allowance.messages()
		if err != nil {
			result.AcceptedCommit = ""
		}
	}()
	start := req.Prepared.Structural().Model().Inputs().Raw.Census
	result.StartingWorkspace, err = req.Workspaces.PrepareAssignment(ctx, run.WorkspaceSpec{ID: run.WorkspaceID(req.ID + "-work-0"), Tree: start})
	if err != nil {
		return result, err
	}
	if result.StartingWorkspace == nil {
		return result, errors.New("the assignment starting copy was not established")
	}
	rootProfile := team[0].VerifyProfile
	result.Assessment, err = run.ReviewAssignment(ctx, req.Workspaces, allowance, run.AssignmentReview{ID: run.WorkspaceID(req.ID + "-assess"), Purpose: run.Assess, Source: result.StartingWorkspace, Commit: result.StartingWorkspace.StartingCommit(), Profile: rootProfile, Context: modelContext, Timeout: req.MessageTimeout, Transcript: filepath.Join(req.TranscriptDir, req.ID+"-assess.jsonl"), CreateHome: req.CreateHome})
	if err != nil || result.Assessment.AcceptedCommit == "" {
		if result.Assessment.ModelProblem && len(result.Assessment.Issues) > 0 {
			issue := result.Assessment.Issues[0]
			result.ModelProblem = &issue
		}
		return result, err
	}
	result.Work, err = run.OpenAssignmentTree(ctx, team, req.Workspaces, allowance, req.Integration, run.AssignmentSchedule{IDPrefix: req.ID + "-work", PreparedRoot: result.StartingWorkspace, Task: assignmentImplementTask + "\nOwner request: " + req.OwnerRequest, MaxMessages: req.MaxMessages, Timeout: req.MessageTimeout, TranscriptDir: req.TranscriptDir, CreateHome: req.CreateHome})
	if err != nil {
		return result, err
	}
	result.Root, err = result.Work.Implement(ctx)
	if err != nil {
		return result, err
	}
	for {
		tree, readErr := req.Workspaces.ReadAssignmentCommit(ctx, run.AssignmentCommitRequest{Source: result.Work.Flow.Workspace("."), Commit: result.Root.ResultCommit, MaxEntries: req.MaxEntries, MaxBytes: req.MaxBytes})
		err = readErr
		if err != nil {
			return result, err
		}
		result.Tree, result.TreeCommit = tree, result.Root.ResultCommit
		var issues []run.ReviewIssue
		for _, finding := range append(candidates.ModelCheck(result.Tree), candidates.Account(result.Tree)...) {
			issues = append(issues, run.ReviewIssue{Owner: finding.Owner, Obligation: finding.Code, Expected: "The fixed model and complete artifact accounting must hold.", Observed: finding.Path + ": " + finding.Message})
		}
		if len(issues) == 0 {
			issues, err = checkAssignmentResult(ctx, req, &result, checks)
			if err != nil {
				return result, err
			}
		}
		if len(issues) == 0 {
			n := len(result.Reviews) + 1
			context := assignmentVerifyContext(modelContext, result)
			verification, reviewErr := run.ReviewAssignment(ctx, req.Workspaces, allowance, run.AssignmentReview{ID: run.WorkspaceID(fmt.Sprintf("%s-verify-%d", req.ID, n)), Purpose: run.Verify, Source: result.Work.Flow.Workspace("."), Commit: result.Root.ResultCommit, Profile: rootProfile, Context: context, Timeout: req.MessageTimeout, Transcript: filepath.Join(req.TranscriptDir, fmt.Sprintf("%s-verify-%d.jsonl", req.ID, n)), CreateHome: req.CreateHome})
			result.Reviews = append(result.Reviews, verification)
			if reviewErr != nil {
				return result, reviewErr
			}
			if verification.ModelProblem {
				if len(verification.Issues) > 0 {
					issue := verification.Issues[0]
					result.ModelProblem = &issue
				}
				return result, nil
			}
			if verification.AcceptedCommit == result.Root.ResultCommit {
				result.AcceptedCommit = result.Root.ResultCommit
				result.NoChange = result.Tree.Equal(start)
				return result, nil
			}
			issues = verification.Issues
		}
		if len(issues) == 0 {
			return result, errors.New("the named result has neither acceptance nor concrete required matters")
		}
		var correction strings.Builder
		for _, issue := range issues {
			fmt.Fprintf(&correction, "Owner %s; obligation %s; expected %s; observed %s\n", issue.Owner, issue.Obligation, issue.Expected, issue.Observed)
		}
		corrected, correctErr := result.Work.Correct(ctx, correction.String())
		err = correctErr
		if corrected.ResultCommit != "" {
			result.Root = corrected
		}
		if err != nil {
			return result, err
		}
	}
}

func checkAssignmentResult(ctx context.Context, req AssignmentRealizationRequest, result *AssignmentRealizationResult, checks []run.Check) ([]run.ReviewIssue, error) {
	var issues []run.ReviewIssue
	for _, check := range checks {
		request := check.Request
		request.ID = run.CheckID(fmt.Sprintf("%s-check-%d", req.ID, len(result.Checks)+1))
		request.Tree = result.Tree
		receipt, err := req.Checks.Run(ctx, request)
		observation := AssignmentCheckObservation{Commit: result.Root.ResultCommit, Name: check.Name, Request: request, Receipt: receipt}
		if err != nil {
			observation.Error = err.Error()
		}
		result.Checks = append(result.Checks, observation)
		if err != nil {
			return issues, err
		}
		if !receipt.Passed(request) {
			if !receipt.Started || !receipt.Exited || receipt.TimedOut || receipt.Uncertain || receipt.ID != request.ID || receipt.Tree != request.Tree.Digest() || receipt.ImageID != request.ImageID {
				return issues, errors.New("required check did not establish an exact, completed result")
			}
			issues = append(issues, run.ReviewIssue{Owner: check.Owner, Obligation: check.RequiredBy, Expected: "Required check " + check.Name + " passes on " + result.Root.ResultCommit, Observed: fmt.Sprintf("Exit %d: %s", receipt.ExitCode, receipt.Output)})
		}
	}
	return issues, nil
}

// Every provider Message, including parent consultations, shares this one
// invocation counter. Separate purposes and corrections never reset it.
type realizationAgents struct {
	mu        sync.Mutex
	agents    run.AssignmentAgents
	max, used int
}

func (a *realizationAgents) OpenAssignment(ctx context.Context, spec run.AssignmentSpec, h run.ToolHandler) (run.AssignmentSession, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if a.messages() >= a.max {
		return nil, errors.New("invocation assignment message allowance exhausted")
	}
	s, err := a.agents.OpenAssignment(ctx, spec, h)
	if err != nil {
		return s, err
	}
	return &realizationSession{AssignmentSession: s, allowance: a}, nil
}
func (a *realizationAgents) messages() int { a.mu.Lock(); defer a.mu.Unlock(); return a.used }

type realizationSession struct {
	run.AssignmentSession
	allowance *realizationAgents
}

func (s *realizationSession) Message(ctx context.Context, text string, timeout time.Duration) (run.TurnResult, error) {
	if ctx.Err() != nil {
		return run.TurnResult{}, ctx.Err()
	}
	a := s.allowance
	a.mu.Lock()
	if a.used >= a.max {
		a.mu.Unlock()
		return run.TurnResult{}, errors.New("invocation assignment message allowance exhausted")
	}
	a.used++
	a.mu.Unlock()
	return s.AssignmentSession.Message(ctx, text, timeout)
}
