package projectapp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

// BrownfieldOperation selects one concrete stage. Input contains exactly the
// payload for that stage; preview/write guards and ledger CAS are shared by CLI/MCP.
type BrownfieldOperation struct {
	Root           string          `json:"root"`
	SourceRoot     string          `json:"sourceRoot,omitempty"`
	Revision       string          `json:"revision,omitempty"`
	SessionID      string          `json:"sessionId,omitempty"`
	Action         string          `json:"action"`
	Write          bool            `json:"write"`
	ExpectedDigest string          `json:"expectedDigest,omitempty"`
	Input          BrownfieldInput `json:"input"`
}

type BrownfieldInput struct {
	Start       *BrownfieldStartInput                    `json:"start,omitempty"`
	Begin       *projectadoption.ReverseIterationRequest `json:"begin,omitempty"`
	Context     *BrownfieldContextInput                  `json:"context,omitempty"`
	Proposal    *BrownfieldProposalInput                 `json:"proposal,omitempty"`
	Integration *BrownfieldIntegrationInput              `json:"integration,omitempty"`
	Iterate     *BrownfieldIterateInput                  `json:"iterate,omitempty"`
	Resolve     *BrownfieldResolveInput                  `json:"resolve,omitempty"`
	Plan        *BrownfieldPlanInput                     `json:"plan,omitempty"`
	Apply       *BrownfieldApplyAdoptionInput            `json:"apply,omitempty"`
}

func (i BrownfieldInput) present() bool { return i.count() != 0 }
func (i BrownfieldInput) count() int {
	n := 0
	for _, set := range []bool{i.Start != nil, i.Begin != nil, i.Context != nil, i.Proposal != nil, i.Integration != nil, i.Iterate != nil, i.Resolve != nil, i.Plan != nil, i.Apply != nil} {
		if set {
			n++
		}
	}
	return n
}
func (r BrownfieldOperation) validate() error {
	if r.Action == "record-adoption" {
		return errors.New("caller-supplied adoption receipts are not accepted; use plan, then apply-adoption with the reviewed plan digest")
	}
	if err := requireRoot(r.Root); err != nil {
		return err
	}
	valid := map[string]bool{"start": r.Input.Start != nil, "begin": r.Input.Begin != nil, "context": r.Input.Context != nil, "propose": r.Input.Proposal != nil, "integrate": r.Input.Integration != nil, "iterate": r.Input.Iterate != nil, "resolve": r.Input.Resolve != nil, "plan": r.Input.Plan != nil, "apply-adoption": r.Input.Apply != nil, "resume": r.Input.count() == 0}
	match, known := valid[r.Action]
	if !known || !match || (r.Action != "resume" && r.Input.count() != 1) {
		return errors.New("Brownfield input must contain exactly the typed payload for the selected action")
	}
	return nil
}

// Brownfield exposes the durable, provider-free Brownfield session ledger.
// A stage is preview-only unless --write is explicit; writes compare-and-swap
// against the session digest shown by the preview.
func (o Operations) Brownfield(operation BrownfieldOperation) (BrownfieldResult, error) {
	if err := operation.validate(); err != nil {
		return BrownfieldResult{}, err
	}
	sourceRoot := operation.SourceRoot
	if sourceRoot == "" {
		sourceRoot = operation.Root
	}
	if strings.TrimSpace(operation.Root) == "" || strings.TrimSpace(sourceRoot) == "" {
		return BrownfieldResult{}, errors.New("Brownfield requires target --repo and a source repository")
	}
	if operation.Write && strings.TrimSpace(operation.ExpectedDigest) == "" {
		return BrownfieldResult{}, errors.New("Brownfield --write requires --expect with the session digest")
	}
	if !operation.Write && operation.ExpectedDigest != "" && operation.Action == "resume" {
		return BrownfieldResult{}, errors.New("Brownfield resume is read-only and does not accept --expect")
	}

	switch operation.Action {
	case "start":
		if operation.SessionID != "" || !operation.Input.present() || operation.Revision == "" {
			return BrownfieldResult{}, errors.New("Brownfield start requires --revision and --input, and does not accept --session")
		}
		request := *operation.Input.Start
		if request.ScopeStatuses == nil {
			return BrownfieldResult{}, errors.New("Brownfield start input must include scopeStatuses, using an empty array when none apply")
		}
		target, err := projectwork.Load(operation.Root, operation.Revision)
		if err != nil {
			return BrownfieldResult{}, fmt.Errorf("load fixed target project: %w", err)
		}
		session, err := projectadoption.StartBrownfieldSession(sourceRoot, target, request.Discovery, request.ScopeStatuses)
		if err != nil {
			return BrownfieldResult{}, err
		}
		return finishBrownfield(operation, sourceRoot, "start", "", session, nil)

	case "record-adoption":
		return BrownfieldResult{}, errors.New("caller-supplied adoption receipts are not accepted; use plan, then apply-adoption with the reviewed plan digest")

	case "begin", "iterate", "propose", "integrate", "resolve", "plan", "apply-adoption", "resume", "context":
		if strings.TrimSpace(operation.SessionID) == "" {
			return BrownfieldResult{}, fmt.Errorf("Brownfield %s requires --session", operation.Action)
		}
		if operation.Revision != "" {
			return BrownfieldResult{}, errors.New("Brownfield stages use the fixed target revision recorded by the session; start a new session to select another revision")
		}
		if operation.Action == "resume" {
			if operation.Input.present() || operation.Write || operation.ExpectedDigest != "" || operation.Revision != "" {
				return BrownfieldResult{}, errors.New("Brownfield resume accepts only --repo, optional --source-repo, and --session")
			}
			session, readiness, err := projectadoption.ResumeBrownfieldSession(sourceRoot, operation.Root, operation.SessionID)
			if err != nil {
				return BrownfieldResult{}, err
			}
			overview := overviewBrownfieldSession(session)
			return BrownfieldResult{Status: "resumed", Action: "resume", SessionDigest: session.Digest, Session: &overview, Readiness: &readiness}, nil
		}
		if operation.Action == "context" {
			if !operation.Input.present() || operation.Write || operation.ExpectedDigest != "" {
				return BrownfieldResult{}, errors.New("Brownfield context requires --input and is read-only")
			}
			request := *operation.Input.Context
			session, _, err := projectadoption.ResumeBrownfieldSession(sourceRoot, operation.Root, operation.SessionID)
			if err != nil {
				return BrownfieldResult{}, err
			}
			result := BrownfieldResult{Status: "context", Action: "context", SessionDigest: session.Digest}
			switch request.Phase {
			case "", "propose":
				managerContext, buildErr := projectadoption.BuildManagerReverseContext(session, request.IterationID)
				if buildErr != nil {
					return BrownfieldResult{}, buildErr
				}
				result.ManagerContext = &managerContext
			case "integrate":
				integrationContext, buildErr := projectadoption.BuildManagerIntegrationContext(session, request.IterationID)
				if buildErr != nil {
					return BrownfieldResult{}, buildErr
				}
				result.IntegrationContext = &integrationContext
			default:
				return BrownfieldResult{}, errors.New("Brownfield context phase must be propose or integrate")
			}
			return result, nil
		}
		if !operation.Input.present() {
			return BrownfieldResult{}, fmt.Errorf("Brownfield %s requires --input", operation.Action)
		}
		prior, readiness, err := projectadoption.ResumeBrownfieldSession(sourceRoot, operation.Root, operation.SessionID)
		if err != nil {
			return BrownfieldResult{}, err
		}
		if operation.ExpectedDigest != "" && operation.ExpectedDigest != prior.Digest {
			return BrownfieldResult{}, errors.New("--expect does not match the current Brownfield session digest")
		}
		if operation.Write && operation.ExpectedDigest != prior.Digest {
			return BrownfieldResult{}, errors.New("Brownfield stage write requires --expect with the current session digest")
		}
		target, err := projectwork.Load(operation.Root, prior.Target.Revision)
		if err != nil {
			return BrownfieldResult{}, fmt.Errorf("reload fixed target project: %w", err)
		}
		if operation.Action == "apply-adoption" {
			if !operation.Write {
				return BrownfieldResult{}, errors.New("apply-adoption requires --write; use the plan action for preview")
			}
			request := *operation.Input.Apply
			schemaDigest, buildDigest, bindingErr := projectadoption.CurrentBindings(projectmodel.Schema())
			if bindingErr != nil {
				return BrownfieldResult{}, bindingErr
			}
			next, plan, receipt, applyErr := projectadoption.ApplyAndRecordSessionAdoption(sourceRoot, operation.Root, target, prior, request.IterationID, request.ExpectedPlanDigest, schemaDigest, buildDigest)
			if applyErr != nil {
				if receipt.CandidateDigest != "" {
					return BrownfieldResult{}, fmt.Errorf("model adoption was applied (plan %s, receipt candidate %s), but the Brownfield receipt could not be recorded: %w; do not rerun apply-adoption automatically; inspect the target model and session ledger first", plan.PlanDigest, receipt.CandidateDigest, applyErr)
				}
				return BrownfieldResult{}, applyErr
			}
			if _, err := projectadoption.WriteBrownfieldSession(sourceRoot, next, prior.Digest); err != nil {
				return BrownfieldResult{}, fmt.Errorf("model adoption was applied (plan %s, receipt candidate %s), but the Brownfield session receipt could not be recorded because its ledger compare-and-swap failed: %w; do not rerun apply-adoption automatically; inspect the target model and session ledger first", plan.PlanDigest, receipt.CandidateDigest, err)
			}
			overview := overviewBrownfieldSession(next)
			return BrownfieldResult{Status: "recorded", Action: "apply-adoption", PriorSessionDigest: prior.Digest, SessionDigest: next.Digest, Session: &overview, Plan: &plan, Receipt: &receipt}, nil
		}
		var next projectadoption.BrownfieldSession
		var plan *projectadoption.AdoptionPlan
		switch operation.Action {
		case "begin":
			request := *operation.Input.Begin
			next, err = projectadoption.BeginReverseIteration(sourceRoot, target, prior, request)
		case "propose":
			request := *operation.Input.Proposal
			next, err = projectadoption.RecordManagerProposal(prior, request.IterationID, request.Proposal)
		case "integrate":
			request := *operation.Input.Integration
			next, err = projectadoption.IntegrateManagerProposal(prior, request.IterationID, request.Integration.ManagerID, request.Integration)
		case "iterate":
			request := *operation.Input.Iterate
			next, err = projectadoption.BeginReverseIteration(sourceRoot, target, prior, request.Request)
			if err == nil {
				next, err = projectadoption.RecordManagerProposal(next, request.Request.ID, request.Proposal)
			}
			if err == nil && request.Integration != nil {
				next, err = projectadoption.IntegrateManagerProposal(next, request.Request.ID, request.Integration.ManagerID, *request.Integration)
			}
		case "resolve":
			request := *operation.Input.Resolve
			next, err = projectadoption.RecordSessionResolution(prior, request.IterationID, request.Resolution)
		case "plan":
			request := *operation.Input.Plan
			if operation.Write || operation.ExpectedDigest != "" {
				return BrownfieldResult{}, errors.New("Brownfield plan is a preview and does not write a session stage")
			}
			schemaDigest, buildDigest, bindingErr := projectadoption.CurrentBindings(projectmodel.Schema())
			if bindingErr != nil {
				return BrownfieldResult{}, bindingErr
			}
			planValue, planErr := projectadoption.PlanSessionAdoption(sourceRoot, target, prior, request.IterationID, schemaDigest, buildDigest)
			if planErr != nil {
				return BrownfieldResult{}, planErr
			}
			plan = &planValue
			overview := overviewBrownfieldSession(prior)
			return BrownfieldResult{Status: "preview", Action: "plan", SessionDigest: prior.Digest, Session: &overview, Readiness: &readiness, Plan: plan}, nil
		default:
			return BrownfieldResult{}, fmt.Errorf("unsupported Brownfield action %q", operation.Action)
		}
		if err != nil {
			return BrownfieldResult{}, err
		}
		return finishBrownfield(operation, sourceRoot, operation.Action, prior.Digest, next, plan)
	default:
		return BrownfieldResult{}, errors.New("Brownfield action must be start, begin, context, propose, integrate, iterate, resolve, plan, apply-adoption, or resume")
	}
}

func finishBrownfield(operation BrownfieldOperation, sourceRoot, action, priorDigest string, session projectadoption.BrownfieldSession, plan *projectadoption.AdoptionPlan) (BrownfieldResult, error) {
	status := "preview"
	if operation.Write {
		expected := priorDigest
		if action == "start" {
			// A new ledger has no prior digest; the preview digest is the
			// creation token and WriteBrownfieldSession verifies it.
			expected = operation.ExpectedDigest
			if expected != session.Digest {
				return BrownfieldResult{}, errors.New("--expect does not match the Brownfield session creation preview digest")
			}
		}
		if action != "start" && expected != operation.ExpectedDigest {
			return BrownfieldResult{}, errors.New("--expect does not match the prior Brownfield session digest")
		}
		if _, err := projectadoption.WriteBrownfieldSession(sourceRoot, session, expected); err != nil {
			return BrownfieldResult{}, err
		}
		status = "recorded"
	}
	overview := overviewBrownfieldSession(session)
	return BrownfieldResult{Status: status, Action: action, PriorSessionDigest: priorDigest, SessionDigest: session.Digest, Session: &overview, Plan: plan}, nil
}

type BrownfieldStartInput struct {
	Discovery     projectadoption.Discovery     `json:"discovery"`
	ScopeStatuses []projectadoption.ScopeStatus `json:"scopeStatuses"`
}

type BrownfieldIterateInput struct {
	Request     projectadoption.ReverseIterationRequest `json:"request"`
	Proposal    projectadoption.ManagerProposal         `json:"proposal"`
	Integration *projectadoption.ManagerIntegration     `json:"integration,omitempty"`
}

type BrownfieldProposalInput struct {
	IterationID string                          `json:"iterationId"`
	Proposal    projectadoption.ManagerProposal `json:"proposal"`
}

type BrownfieldIntegrationInput struct {
	IterationID string                             `json:"iterationId"`
	Integration projectadoption.ManagerIntegration `json:"integration"`
}

type BrownfieldResolveInput struct {
	IterationID string                     `json:"iterationId"`
	Resolution  projectadoption.Resolution `json:"resolution"`
}

type BrownfieldPlanInput struct {
	IterationID string `json:"iterationId"`
}

type BrownfieldContextInput struct {
	IterationID string `json:"iterationId"`
	Phase       string `json:"phase,omitempty"`
}

type BrownfieldApplyAdoptionInput struct {
	IterationID        string `json:"iterationId"`
	ExpectedPlanDigest string `json:"expectedPlanDigest"`
}

type BrownfieldResult struct {
	Status             string                                     `json:"status"`
	Action             string                                     `json:"action"`
	PriorSessionDigest string                                     `json:"priorSessionDigest,omitempty"`
	SessionDigest      string                                     `json:"sessionDigest"`
	Session            *BrownfieldSessionOverview                 `json:"session,omitempty"`
	Readiness          *projectadoption.Readiness                 `json:"readiness,omitempty"`
	Plan               *projectadoption.AdoptionPlan              `json:"plan,omitempty"`
	ManagerContext     *projectadoption.ManagerReverseContext     `json:"managerContext,omitempty"`
	IntegrationContext *projectadoption.ManagerIntegrationContext `json:"integrationContext,omitempty"`
	Receipt            *projectadoption.AdoptionReceipt           `json:"receipt,omitempty"`
}

// BrownfieldSessionOverview exposes only fixed bases and workflow metadata.
// The durable ledger remains the full validator input, but CLI responses never
// return unassigned source bodies or private Manager report content.
type BrownfieldSessionOverview struct {
	APIVersion          string                        `json:"apiVersion"`
	ID                  string                        `json:"id"`
	Digest              string                        `json:"digest"`
	Source              BrownfieldSourceOverview      `json:"source"`
	Target              BrownfieldTargetOverview      `json:"target"`
	TargetContextDigest string                        `json:"targetContextDigest"`
	Scopes              []BrownfieldScopeOverview     `json:"scopes"`
	Iterations          []BrownfieldIterationOverview `json:"iterations"`
	Adoptions           []BrownfieldAdoptionOverview  `json:"adoptions"`
}

type BrownfieldSourceOverview struct {
	Root                string `json:"root"`
	DiscoveryID         string `json:"discoveryId"`
	Commit              string `json:"commit"`
	Digest              string `json:"digest"`
	ScopeRootCount      int    `json:"scopeRootCount"`
	SelectedPathCount   int    `json:"selectedPathCount"`
	ExclusionCount      int    `json:"exclusionCount"`
	UnselectedPathCount int    `json:"unselectedPathCount"`
	EvidenceCount       int    `json:"evidenceCount"`
}

type BrownfieldTargetOverview struct {
	Root          string `json:"root"`
	Revision      string `json:"revision"`
	ProjectDigest string `json:"projectDigest"`
	ModelDigest   string `json:"modelDigest"`
}

type BrownfieldScopeOverview struct {
	ScopeID string `json:"scopeId"`
	Status  string `json:"status"`
}

type BrownfieldIterationOverview struct {
	ID                  string `json:"id"`
	ParentIterationID   string `json:"parentIterationId,omitempty"`
	ManagerID           string `json:"managerId"`
	TargetContextDigest string `json:"targetContextDigest"`
	ProposalDigest      string `json:"proposalDigest,omitempty"`
	IntegrationDigest   string `json:"integrationDigest,omitempty"`
	ResolutionDigest    string `json:"resolutionDigest,omitempty"`
}

type BrownfieldAdoptionOverview struct {
	IterationID     string `json:"iterationId"`
	PlanDigest      string `json:"planDigest"`
	ReceiptStatus   string `json:"receiptStatus"`
	CandidateDigest string `json:"candidateDigest"`
}

func overviewBrownfieldSession(session projectadoption.BrownfieldSession) BrownfieldSessionOverview {
	result := BrownfieldSessionOverview{
		APIVersion: session.APIVersion, ID: session.ID, Digest: session.Digest,
		Source: BrownfieldSourceOverview{Root: session.Source.Identity.Root, DiscoveryID: session.Source.ID, Commit: session.Source.Commit,
			Digest: session.Source.Digest, ScopeRootCount: len(session.Source.ScopeRoots), SelectedPathCount: len(session.Source.Selected),
			ExclusionCount: len(session.Source.Exclusions), UnselectedPathCount: len(session.Source.Unselected), EvidenceCount: len(session.Source.Evidence)},
		Target:              BrownfieldTargetOverview{Root: session.Target.Root, Revision: session.Target.Revision, ProjectDigest: session.Target.ProjectDigest, ModelDigest: session.Target.ModelDigest},
		TargetContextDigest: session.TargetContext.Digest,
		Scopes:              make([]BrownfieldScopeOverview, 0, len(session.Scopes)),
		Iterations:          make([]BrownfieldIterationOverview, 0, len(session.Iterations)),
		Adoptions:           make([]BrownfieldAdoptionOverview, 0, len(session.Adoptions)),
	}
	for _, scope := range session.Scopes {
		result.Scopes = append(result.Scopes, BrownfieldScopeOverview{ScopeID: scope.ScopeID, Status: scope.Status})
	}
	for _, iteration := range session.Iterations {
		item := BrownfieldIterationOverview{ID: iteration.ID, ParentIterationID: iteration.ParentIterationID, ManagerID: iteration.ManagerID, TargetContextDigest: iteration.TargetContextDigest}
		if iteration.Proposal != nil {
			item.ProposalDigest = iteration.Proposal.Digest
		}
		if iteration.Integration != nil {
			item.IntegrationDigest = iteration.Integration.Digest
		}
		if iteration.Resolution != nil {
			item.ResolutionDigest = iteration.Resolution.Digest
		}
		result.Iterations = append(result.Iterations, item)
	}
	for _, adoption := range session.Adoptions {
		result.Adoptions = append(result.Adoptions, BrownfieldAdoptionOverview{IterationID: adoption.IterationID, PlanDigest: adoption.Plan.PlanDigest,
			ReceiptStatus: adoption.Receipt.Status, CandidateDigest: adoption.Receipt.CandidateDigest})
	}
	return result
}
