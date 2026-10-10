package projectcli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

const runRecovery = "Inspect the run with `status RUN` and its blockers before resuming or repairing the same run."

// --- Work items -----------------------------------------------------------

type exploreInput struct {
	Exploration string         `json:"exploration,omitempty"`
	Input       *exploreRecord `json:"input,omitempty"`
	Revision    string         `json:"revision,omitempty"`
	Expect      string         `json:"expect,omitempty"`
	Write       bool           `json:"write,omitempty"`
}

// exploreRecord is an exploration record as a caller writes it: for a new
// record the Host fills createdAgainstBindingDigest and digest.
type exploreRecord struct {
	APIVersion       string                                    `json:"apiVersion"`
	ID               string                                    `json:"id"`
	Status           string                                    `json:"status"`
	Request          string                                    `json:"request"`
	CreatedAgainst   string                                    `json:"createdAgainstBindingDigest,omitempty"`
	Scopes           []projectexplore.Scope                    `json:"scopes"`
	Decisions        []projectexplore.Decision                 `json:"decisions"`
	Drafts           []projectexplore.DraftProposal            `json:"drafts"`
	Acknowledgements []projectexplore.StructureAcknowledgement `json:"structureAcknowledgements"`
	Completions      []projectexplore.ApplyReceipt             `json:"completions"`
	Digest           string                                    `json:"digest,omitempty"`
}

// record validates the input with the exploration package's own input rules.
func (r *exploreRecord) record() (*projectexplore.Record, error) {
	if r == nil {
		return nil, nil
	}
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	record, err := projectexplore.DecodeRecordInput(data)
	if err != nil {
		return nil, usageError{err}
	}
	return &record, nil
}

var exploreVerb = define(verb{
	name: "explore", group: "Work items", effect: effectWrite,
	summary:  "List or read work-item explorations, or preview and write one record.",
	synopsis: "explore [--exploration ID | --input FILE] [--revision R] [--expect DIGEST --write]",
	recovery: "Inspect the exploration and its structure findings; correct the record and refresh the preview and its digest before writing.",
	args: []arg{argRepo,
		{name: "exploration", value: "ID", help: "Exploration to read."},
		{name: "input", kind: kindRecord, value: "FILE", help: "Exploration record to preview or write."},
		argRevision, argExpect, argWrite},
}, func(ctx context.Context, e env, in exploreInput) (projectapp.ExploreResult, error) {
	if in.Exploration != "" && in.Input != nil {
		return projectapp.ExploreResult{}, usagef("use either --exploration or --input")
	}
	record, err := in.Input.record()
	if err != nil {
		return projectapp.ExploreResult{}, err
	}
	return staleOnly(e.ops.Explore(projectapp.ExploreOperation{
		Selection: projectapp.Selection{Root: e.root, Revision: in.Revision}, ExplorationID: in.Exploration,
		Record: record, Write: in.Write, ExpectedDigest: in.Expect,
	}))
})

type readyInput struct {
	Exploration    string `json:"exploration"`
	Scope          string `json:"scope"`
	Acknowledge    bool   `json:"acknowledge,omitempty"`
	Actor          string `json:"actor,omitempty"`
	Authority      string `json:"authority,omitempty"`
	DecisionRef    string `json:"decisionRef,omitempty"`
	AcknowledgedAt string `json:"acknowledgedAt,omitempty"`
	Revision       string `json:"revision,omitempty"`
	Expect         string `json:"expect,omitempty"`
	Write          bool   `json:"write,omitempty"`
}

var readyVerb = define(verb{
	name: "ready", group: "Work items", effect: effectWrite,
	summary:  "Show a scope's readiness, or acknowledge its exact structure under caller authority.",
	synopsis: "ready --exploration ID --scope ID [--acknowledge --actor A --authority TEXT --decision-ref REF --acknowledged-at TIME] [--revision R] [--expect DIGEST --write]",
	recovery: "Inspect the scope's readiness and structure findings; acknowledge or correct them and refresh the preview and its digest before writing.",
	args: []arg{argRepo,
		{name: "exploration", value: "ID", required: true, help: "Exploration."},
		{name: "scope", value: "ID", required: true, help: "Scope within the exploration."},
		{name: "acknowledge", kind: kindBool, help: "Acknowledge the exact proposed structure."},
		{name: "actor", value: "A", help: "Acknowledging actor."},
		{name: "authority", value: "TEXT", help: "Authority under which the actor acknowledges."},
		{name: "decision-ref", value: "REF", help: "Decision reference."},
		{name: "acknowledged-at", value: "TIME", help: "Explicit RFC 3339 timestamp."},
		argRevision, argExpect, argWrite},
}, func(ctx context.Context, e env, in readyInput) (projectapp.ReadinessResult, error) {
	var acknowledgement *projectapp.StructureAcknowledgementInput
	if in.Acknowledge {
		if strings.TrimSpace(in.Actor) == "" || strings.TrimSpace(in.Authority) == "" || strings.TrimSpace(in.DecisionRef) == "" {
			return projectapp.ReadinessResult{}, usagef("--acknowledge requires --actor, --authority and --decision-ref")
		}
		recordedAt, err := time.Parse(time.RFC3339Nano, in.AcknowledgedAt)
		if err != nil {
			return projectapp.ReadinessResult{}, usagef("--acknowledged-at must be an explicit RFC 3339 timestamp")
		}
		acknowledgement = &projectapp.StructureAcknowledgementInput{Actor: in.Actor, Authority: in.Authority, Provenance: in.DecisionRef, RecordedAt: recordedAt}
	} else if in.Actor != "" || in.Authority != "" || in.DecisionRef != "" || in.AcknowledgedAt != "" {
		return projectapp.ReadinessResult{}, usagef("--actor, --authority, --decision-ref and --acknowledged-at need --acknowledge")
	}
	return staleOnly(e.ops.Readiness(projectapp.ReadinessOperation{
		Selection: projectapp.Selection{Root: e.root, Revision: in.Revision}, ExplorationID: in.Exploration, ScopeID: in.Scope,
		Acknowledgement: acknowledgement, Write: in.Write, ExpectedDigest: in.Expect,
	}))
})

type briefInput struct {
	Action     string `json:"action,omitempty"`
	Since      string `json:"since,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Provenance string `json:"provenance,omitempty"`
	Manager    string `json:"manager,omitempty"`
	Event      string `json:"event,omitempty"`
	Expect     string `json:"expect,omitempty"`
	Write      bool   `json:"write,omitempty"`
}

var briefVerb = define(verb{
	name: "brief", group: "Work items", effect: effectWrite,
	summary:  "Create, list or dismiss Manager briefings about accepted model changes.",
	synopsis: "brief --since R1 --revision R2 --provenance TEXT [--expect DIGEST --write] | brief list [--manager ID] | brief dismiss --event ID --manager ID --expect DIGEST --write",
	notes:    "brief dismiss takes the stateDigest from brief list as --expect.",
	subs: []subVerb{
		{name: "create", summary: "Preview or record the briefing for a revision range.", effect: effectWrite, readOnly: true},
		{name: "list", summary: "List visible briefings, or one Manager's briefings.", effect: effectRead, readOnly: true},
		{name: "dismiss", summary: "Hide one event from one Manager's view.", effect: effectWrite},
	},
	defaultSub: "create",
	args: []arg{argRepo,
		{name: "since", value: "R1", help: "Older revision (create)."},
		{name: "revision", value: "R2", help: "Newer revision (create)."},
		{name: "provenance", value: "TEXT", help: "Decision reference for the briefing (create)."},
		{name: "manager", value: "ID", help: "Manager (list, dismiss)."},
		{name: "event", value: "ID", help: "Event to dismiss (dismiss)."},
		argExpect, argWrite},
}, func(ctx context.Context, e env, in briefInput) (any, error) {
	switch in.Action {
	case "", "create":
		if in.Since == "" || in.Revision == "" || in.Provenance == "" {
			return nil, usagef("brief requires --since, --revision and --provenance")
		}
		return briefCreate(e.root, in)
	case "list":
		if in.Since != "" || in.Revision != "" || in.Provenance != "" || in.Event != "" || in.Write {
			return nil, usagef("brief list takes only --manager")
		}
		return briefList(e.root, in.Manager)
	case "dismiss":
		if in.Event == "" || in.Manager == "" || !in.Write {
			return nil, usagef("brief dismiss requires --event, --manager, --expect and --write")
		}
		binding, err := projectbriefing.Dismiss(e.root, in.Event, in.Manager, in.Expect)
		if err != nil {
			return nil, err
		}
		return struct {
			StateDigest string `json:"stateDigest"`
			Note        string `json:"note"`
		}{binding, "Dismissal affects visibility only; the event remains unresolved and available in manager briefings."}, nil
	}
	return nil, usagef("unknown brief action %q", in.Action)
})

func briefCreate(root string, in briefInput) (any, error) {
	_, stateDigest, err := projectbriefing.Read(root)
	if err != nil {
		return nil, err
	}
	bundle, err := projectbriefing.Generate(root, in.Since, in.Revision, projectbriefing.Provenance{
		DecisionReference: in.Provenance,
		Actor:             "cli-caller",
		Authority:         "explicit caller declaration; identity is not authenticated",
	})
	if err != nil {
		return nil, err
	}
	if in.Write {
		if stateDigest, err = projectbriefing.Write(root, bundle, in.Expect); err != nil {
			return nil, err
		}
	}
	return struct {
		Bundle      projectbriefing.Bundle `json:"bundle"`
		StateDigest string                 `json:"stateDigest"`
	}{bundle, stateDigest}, nil
}

func briefList(root, manager string) (any, error) {
	revision, err := gitHeadRevision(root)
	if err != nil {
		return nil, fmt.Errorf("resolve the committed model revision for briefings: %w", err)
	}
	project, err := projectwork.Load(root, revision)
	if err != nil {
		return nil, err
	}
	if project.Config.WorkflowMode == "guided" {
		if _, err := projectbriefing.EnsureAcceptedHistory(root, revision); err != nil {
			return nil, err
		}
	}
	state, stateDigest, err := projectbriefing.Read(root)
	if err != nil {
		return nil, err
	}
	if manager == "" {
		notifications, hidden, err := visibleBriefings(root, state)
		if err != nil {
			return nil, err
		}
		return struct {
			Notifications        []visibleNotification `json:"notifications"`
			HiddenDismissedCount int                   `json:"hiddenDismissedCount"`
			StateDigest          string                `json:"stateDigest"`
		}{notifications, hidden, stateDigest}, nil
	}
	briefings, events, binding, err := projectbriefing.LoadForManager(root, project.Report.ModelDigest, manager, project.Revision)
	if err != nil {
		return nil, err
	}
	withStatus := make([]managerEvent, 0, len(events))
	for _, event := range events {
		resolution := projectbriefing.EventResolutionStatus(state, event.ID)
		withStatus = append(withStatus, managerEvent{Event: event, ResolutionStatus: resolution.Status, Resolution: resolution.Resolution})
	}
	return struct {
		Briefings     []projectbriefing.Briefing `json:"briefings"`
		Events        []managerEvent             `json:"events"`
		ContextDigest string                     `json:"contextDigest"`
		StateDigest   string                     `json:"stateDigest"`
	}{briefings, withStatus, binding, stateDigest}, nil
}

// --- Delivery -------------------------------------------------------------

type planInput struct {
	Goal        string                `json:"goal"`
	Operation   string                `json:"operation,omitempty"`
	Manager     []string              `json:"manager,omitempty"`
	Revision    string                `json:"revision,omitempty"`
	Since       string                `json:"since,omitempty"`
	Exploration string                `json:"exploration,omitempty"`
	Scope       string                `json:"scope,omitempty"`
	Input       *projectwork.Mutation `json:"input,omitempty"`
	Expect      string                `json:"expect,omitempty"`
	Write       bool                  `json:"write,omitempty"`
}

// planReport is the plan record plus the digest that `plan --write --expect`
// takes; the record's own digest also covers the persisted run identity.
type planReport struct {
	projectrun.PlanRecord
	PreviewDigest string `json:"previewDigest"`
}

var planVerb = define(verb{
	name: "plan", group: "Delivery", effect: effectWrite,
	summary:  "Preview or persist a bounded plan against a fixed revision.",
	synopsis: "plan --goal TEXT [--operation apply|cleanup|reconcile] [--manager ID ...] [--revision R] [--since R0] [--exploration ID --scope ID] [--input FILE] [--expect DIGEST --write]",
	notes:    "--expect takes the preview's previewDigest. --input takes a model mutation that the run delivers together with its implementation.",
	recovery: "Check the selected revision, goal, model, runtime and any exploration or scope; refresh the plan preview and its previewDigest before writing.",
	args: []arg{argRepo,
		{name: "goal", value: "TEXT", required: true, help: "Bounded plan goal."},
		{name: "operation", value: "KIND", help: "apply (default), cleanup or reconcile."},
		{name: "manager", kind: kindList, value: "ID", help: "Manager to plan for (repeatable)."},
		argRevision,
		{name: "since", value: "R0", help: "Older revision for change impact."},
		{name: "exploration", value: "ID", help: "Acknowledged exploration."},
		{name: "scope", value: "ID", help: "Acknowledged scope."},
		{name: "input", kind: kindRecord, value: "FILE", help: "Draft model mutation delivered with the run."},
		argExpect, argWrite},
}, func(ctx context.Context, e env, in planInput) (planReport, error) {
	switch in.Operation {
	case "", projectrun.OperationApply, projectrun.OperationCleanup, projectrun.OperationReconcile:
	default:
		return planReport{}, usagef("--operation must be apply, cleanup or reconcile")
	}
	request := projectrun.PlanRequest{
		Operation: in.Operation, Goal: in.Goal, Managers: append([]string(nil), in.Manager...), BaseRevision: in.Revision,
		SinceRevision: in.Since, ExplorationID: in.Exploration, ScopeID: in.Scope, ModelEdit: in.Input, ExecuteAuthorized: in.Write,
	}
	plan, err := e.ops.Plan(projectapp.PlanOperation{Selection: projectapp.Selection{Root: e.root, Revision: in.Revision}, Request: request, ExpectedPreviewDigest: in.Expect})
	if err != nil {
		return staleOnly(planReport{PlanRecord: plan}, err)
	}
	preview, err := projectrun.PlanPreviewDigest(plan)
	if err != nil {
		return planReport{}, err
	}
	return planReport{PlanRecord: plan, PreviewDigest: preview}, nil
})

type runPlanInput struct {
	Plan    string `json:"plan"`
	Execute bool   `json:"execute,omitempty"`
}

var runVerb = define(verb{
	name: "run", group: "Delivery", effect: effectExecute, inspect: "status", interrupt: true,
	summary:  "Run a persisted plan with its agents.",
	synopsis: "run PLAN --execute",
	recovery: runRecovery,
	args:     []arg{{name: "plan", value: "PLAN", operand: true, required: true, help: "Plan ID from `plan --write`."}, argRepo, argExecute},
}, func(ctx context.Context, e env, in runPlanInput) (projectrun.RunReport, error) {
	return e.ops.Run(ctx, projectapp.RunOperation{Root: e.root, RunID: in.Plan})
})

type runIDInput struct {
	Run     string `json:"run"`
	Execute bool   `json:"execute,omitempty"`
}

var resumeVerb = define(verb{
	name: "resume", group: "Delivery", effect: effectExecute, inspect: "status", interrupt: true,
	summary:  "Resume the same durable run after an interruption.",
	synopsis: "resume RUN --execute",
	recovery: runRecovery,
	args:     []arg{{name: "run", value: "RUN", operand: true, required: true, help: "Run ID."}, argRepo, argExecute},
}, func(ctx context.Context, e env, in runIDInput) (projectrun.RunReport, error) {
	return e.ops.Resume(ctx, projectapp.RunOperation{Root: e.root, RunID: in.Run})
})

var repairVerb = define(verb{
	name: "repair", group: "Delivery", effect: effectExecute, inspect: "status", interrupt: true,
	summary:  "Repair the same durable run.",
	synopsis: "repair RUN --execute",
	notes:    "Inspect the run with `status RUN` first.",
	recovery: runRecovery,
	args:     []arg{{name: "run", value: "RUN", operand: true, required: true, help: "Run ID."}, argRepo, argExecute},
}, func(ctx context.Context, e env, in runIDInput) (projectrun.RunReport, error) {
	return e.ops.Repair(ctx, projectapp.RunOperation{Root: e.root, RunID: in.Run})
})

type statusInput struct {
	Run string `json:"run,omitempty"`
}

var statusVerb = define(verb{
	name: "status", group: "Delivery", effect: effectRead,
	summary:  "Show the project overview, or one run's durable state.",
	synopsis: "status [RUN]",
	recovery: "Check the run ID against existing durable runs; `status` without a run lists them.",
	args:     []arg{{name: "run", value: "RUN", operand: true, help: "Run ID; without it the project overview."}, argRepo},
}, func(ctx context.Context, e env, in statusInput) (any, error) {
	if in.Run == "" {
		return projectOverview(e)
	}
	return e.ops.Status(projectapp.RunOperation{Root: e.root, RunID: in.Run})
})

type verifyInput struct {
	Run      string `json:"run,omitempty"`
	Revision string `json:"revision,omitempty"`
	Execute  bool   `json:"execute,omitempty"`
	Write    bool   `json:"write,omitempty"`
}

var verifyVerb = define(verb{
	name: "verify", group: "Delivery", effect: effectExecute, inspect: "status", interrupt: true,
	summary:  "Verify a run's candidate, or a fixed revision, with the configured checks and verifier.",
	synopsis: "verify RUN --execute | verify --revision R --execute [--write]",
	notes:    "Verifying a run records its evidence in the run. --write persists a revision's standalone verification record.",
	recovery: "Check the selected run or revision, model and configured checks; correct their findings and repeat verification.",
	args: []arg{{name: "run", value: "RUN", operand: true, help: "Run to verify."}, argRepo,
		{name: "revision", value: "R", help: "Fixed revision to verify instead of a run."},
		argExecute,
		{name: "write", kind: kindBool, help: "Persist the standalone verification record (with --revision)."}},
}, func(ctx context.Context, e env, in verifyInput) (any, error) {
	if (in.Run == "") == (in.Revision == "") {
		return nil, usagef("requires exactly one of RUN or --revision")
	}
	if in.Run != "" {
		if in.Write {
			return nil, usagef("--write applies only to --revision; a run records its own evidence")
		}
		report, err := e.ops.Verify(ctx, projectapp.RunOperation{Root: e.root, RunID: in.Run})
		return report, err
	}
	report, err := e.ops.FullVerify(ctx, projectapp.FullVerifyOperation{Root: e.root, Request: projectrun.FullVerifyRequest{Revision: in.Revision, Write: in.Write}})
	if err != nil {
		return report, err
	}
	if report.Status != "passed" {
		return report, outcomef("full verification is %s", report.Status)
	}
	return report, nil
})

type applyInput struct {
	Plan      string `json:"plan"`
	Run       string `json:"run"`
	Candidate string `json:"candidate"`
	Branch    string `json:"branch,omitempty"`
	Head      string `json:"head,omitempty"`
	Worktree  string `json:"worktree,omitempty"`
	Expect    string `json:"expect,omitempty"`
	Write     bool   `json:"write,omitempty"`
}

var applyVerb = define(verb{
	name: "apply", group: "Delivery", effect: effectWrite,
	summary:  "Show the apply preflight, or apply the exact reviewed and verified candidate.",
	synopsis: "apply --plan ID --run ID --candidate ID [--branch B --head C --worktree DIGEST --expect DIGEST --write]",
	notes:    "Without --write it returns the preflight: the verification digest, branch, head and worktree digest that the write repeats.",
	recovery: "Inspect the run with `status RUN`; take current preflight values and a matching successful verification before applying.",
	args: []arg{argRepo,
		{name: "plan", value: "ID", required: true, help: "Plan ID."},
		{name: "run", value: "ID", required: true, help: "Run ID."},
		{name: "candidate", value: "ID", required: true, help: "Integrated candidate."},
		{name: "branch", value: "B", help: "Target branch from the preflight."},
		{name: "head", value: "C", help: "Expected target HEAD from the preflight."},
		{name: "worktree", value: "DIGEST", help: "Expected worktree digest from the preflight."},
		{name: "expect", value: "DIGEST", help: "Verification digest from the preflight."},
		argWrite},
}, func(ctx context.Context, e env, in applyInput) (any, error) {
	if !in.Write {
		preflight, err := e.ops.PreflightApply(projectapp.PreflightOperation{Root: e.root, RunID: in.Run, CandidateID: in.Candidate})
		if err != nil {
			return preflight, err
		}
		if preflight.PlanID != in.Plan {
			return nil, usagef("--plan does not match the verified run")
		}
		return preflight, nil
	}
	if in.Branch == "" || in.Head == "" || in.Worktree == "" {
		return nil, usagef("--write requires --branch, --head and --worktree from the preflight")
	}
	report, err := e.ops.Apply(projectapp.ApplyOperation{Root: e.root, Request: projectrun.ApplyRequest{
		RunID: in.Run, PlanID: in.Plan, CandidateID: in.Candidate, TargetBranch: in.Branch, ExpectedHead: in.Head,
		ExpectedWorktree: in.Worktree, ExpectedVerificationDigest: in.Expect,
	}})
	if err != nil {
		return report, err
	}
	if report.Status != projectrun.StatusApplied {
		return report, outcomef("apply status is %s", report.Status)
	}
	return report, nil
})

type deliverInput struct {
	Exploration string `json:"exploration"`
	Scope       string `json:"scope"`
	Run         string `json:"run,omitempty"`
	Execute     bool   `json:"execute,omitempty"`
	Write       bool   `json:"write,omitempty"`
}

var deliverVerb = define(verb{
	name: "deliver", group: "Delivery", effect: effectExecuteWrite, inspect: "ready", interrupt: true,
	summary:  "Plan, run, verify and apply an acknowledged scope in one step.",
	synopsis: "deliver --exploration ID --scope ID [--run ID] --execute --write",
	notes:    "It applies the result to the target branch.",
	recovery: "If the partial report has a runId, inspect that run with `status RUN`; otherwise check the acknowledged exploration and scope before retrying.",
	args: []arg{argRepo,
		{name: "exploration", value: "ID", required: true, help: "Acknowledged exploration."},
		{name: "scope", value: "ID", required: true, help: "Acknowledged scope."},
		{name: "run", value: "ID", help: "Existing run to continue."},
		argExecute,
		{name: "write", kind: kindBool, help: "Apply the verified result to the target branch."}},
}, func(ctx context.Context, e env, in deliverInput) (projectrun.DeliverReport, error) {
	if !in.Write {
		return projectrun.DeliverReport{}, usagef("deliver applies its result and requires --write as well as --execute")
	}
	report, err := e.ops.Deliver(ctx, projectapp.DeliverOperation{Root: e.root, Request: projectrun.DeliverRequest{
		ExplorationID: in.Exploration, ScopeID: in.Scope, RunID: in.Run, ExecuteAuthorized: true,
	}})
	if err == nil && report.Status != projectrun.StatusApplied {
		return report, outcomef("delivery status is %s", report.Status)
	}
	return report, err
})
