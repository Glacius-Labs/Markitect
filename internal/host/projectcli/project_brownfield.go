package projectcli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

// runBrownfield exposes the durable, provider-free Brownfield session ledger.
// A stage is preview-only unless --write is explicit; writes compare-and-swap
// against the session digest shown by the preview.
func runBrownfield(opts options, out io.Writer) error {
	if opts.brownfieldAction == "run" {
		return runBrownfieldManagerStage(opts, out, projectadoption.AgentExecManagerRunInvoker{})
	}
	sourceRoot := opts.sourceRepo
	if sourceRoot == "" {
		sourceRoot = opts.repo
	}
	if strings.TrimSpace(opts.repo) == "" || strings.TrimSpace(sourceRoot) == "" {
		return errors.New("Brownfield requires target --repo and a source repository")
	}
	if opts.write && strings.TrimSpace(opts.expect) == "" {
		return errors.New("Brownfield --write requires --expect with the session digest")
	}
	if !opts.write && opts.expect != "" && opts.brownfieldAction == "resume" {
		return errors.New("Brownfield resume is read-only and does not accept --expect")
	}

	switch opts.brownfieldAction {
	case "start":
		if opts.sessionID != "" || opts.input == "" || opts.revision == "" {
			return errors.New("Brownfield start requires --revision and --input, and does not accept --session")
		}
		data, err := readRecord(sourceRoot, opts.input)
		if err != nil {
			return err
		}
		var request brownfieldStartInput
		if err := decodeClosedProjectJSON(data, &request); err != nil {
			return fmt.Errorf("decode Brownfield start input: %w", err)
		}
		if request.ScopeStatuses == nil {
			return errors.New("Brownfield start input must include scopeStatuses, using an empty array when none apply")
		}
		target, err := projectwork.Load(opts.repo, opts.revision)
		if err != nil {
			return fmt.Errorf("load fixed target project: %w", err)
		}
		session, err := projectadoption.StartBrownfieldSession(sourceRoot, target, request.Discovery, request.ScopeStatuses)
		if err != nil {
			return err
		}
		return emitBrownfieldSession(opts, sourceRoot, "start", "", session, nil, nil, out)

	case "record-adoption":
		return errors.New("caller-supplied adoption receipts are not accepted; use plan, then apply-adoption with the reviewed plan digest")

	case "begin", "iterate", "propose", "integrate", "resolve", "plan", "apply-adoption", "resume", "context":
		if strings.TrimSpace(opts.sessionID) == "" {
			return fmt.Errorf("Brownfield %s requires --session", opts.brownfieldAction)
		}
		if opts.revision != "" {
			return errors.New("Brownfield stages use the fixed target revision recorded by the session; start a new session to select another revision")
		}
		if opts.brownfieldAction == "resume" {
			if opts.input != "" || opts.write || opts.expect != "" || opts.revision != "" {
				return errors.New("Brownfield resume accepts only --repo, optional --source-repo, and --session")
			}
			session, readiness, err := projectadoption.ResumeBrownfieldSession(sourceRoot, opts.repo, opts.sessionID)
			if err != nil {
				return err
			}
			overview := overviewBrownfieldSession(session)
			return writeJSON(out, brownfieldResult{Status: "resumed", Action: "resume", SessionDigest: session.Digest, Session: &overview, Readiness: &readiness})
		}
		if opts.brownfieldAction == "context" {
			if opts.input == "" || opts.write || opts.expect != "" {
				return errors.New("Brownfield context requires --input and is read-only")
			}
			data, err := readRecord(sourceRoot, opts.input)
			if err != nil {
				return err
			}
			var request brownfieldContextInput
			if err := decodeClosedProjectJSON(data, &request); err != nil {
				return fmt.Errorf("decode Brownfield context input: %w", err)
			}
			session, _, err := projectadoption.ResumeBrownfieldSession(sourceRoot, opts.repo, opts.sessionID)
			if err != nil {
				return err
			}
			result := brownfieldResult{Status: "context", Action: "context", SessionDigest: session.Digest}
			switch request.Phase {
			case "", "propose":
				managerContext, buildErr := projectadoption.BuildManagerReverseContext(session, request.IterationID)
				if buildErr != nil {
					return buildErr
				}
				result.ManagerContext = &managerContext
			case "integrate":
				integrationContext, buildErr := projectadoption.BuildManagerIntegrationContext(session, request.IterationID)
				if buildErr != nil {
					return buildErr
				}
				result.IntegrationContext = &integrationContext
			default:
				return errors.New("Brownfield context phase must be propose or integrate")
			}
			return writeJSON(out, result)
		}
		if opts.input == "" {
			return fmt.Errorf("Brownfield %s requires --input", opts.brownfieldAction)
		}
		prior, readiness, err := projectadoption.ResumeBrownfieldSession(sourceRoot, opts.repo, opts.sessionID)
		if err != nil {
			return err
		}
		if opts.expect != "" && opts.expect != prior.Digest {
			return errors.New("--expect does not match the current Brownfield session digest")
		}
		if opts.write && opts.expect != prior.Digest {
			return errors.New("Brownfield stage write requires --expect with the current session digest")
		}
		data, err := readRecord(sourceRoot, opts.input)
		if err != nil {
			return err
		}
		target, err := projectwork.Load(opts.repo, prior.Target.Revision)
		if err != nil {
			return fmt.Errorf("reload fixed target project: %w", err)
		}
		if opts.brownfieldAction == "apply-adoption" {
			if !opts.write {
				return errors.New("apply-adoption requires --write; use the plan action for preview")
			}
			var request brownfieldApplyAdoptionInput
			if err := decodeClosedProjectJSON(data, &request); err != nil {
				return fmt.Errorf("decode Brownfield apply-adoption input: %w", err)
			}
			schemaDigest, buildDigest, bindingErr := projectadoption.CurrentBindings(projectmodel.Schema())
			if bindingErr != nil {
				return bindingErr
			}
			next, plan, receipt, applyErr := projectadoption.ApplyAndRecordSessionAdoption(sourceRoot, opts.repo, target, prior, request.IterationID, request.ExpectedPlanDigest, schemaDigest, buildDigest)
			if applyErr != nil {
				if receipt.CandidateDigest != "" {
					return fmt.Errorf("model adoption was applied (plan %s, receipt candidate %s), but the Brownfield receipt could not be recorded: %w; do not rerun apply-adoption automatically; inspect the target model and session ledger first", plan.PlanDigest, receipt.CandidateDigest, applyErr)
				}
				return applyErr
			}
			if _, err := projectadoption.WriteBrownfieldSession(sourceRoot, next, prior.Digest); err != nil {
				return fmt.Errorf("model adoption was applied (plan %s, receipt candidate %s), but the Brownfield session receipt could not be recorded because its ledger compare-and-swap failed: %w; do not rerun apply-adoption automatically; inspect the target model and session ledger first", plan.PlanDigest, receipt.CandidateDigest, err)
			}
			overview := overviewBrownfieldSession(next)
			return writeJSON(out, brownfieldResult{Status: "recorded", Action: "apply-adoption", PriorSessionDigest: prior.Digest, SessionDigest: next.Digest, Session: &overview, Plan: &plan, Receipt: &receipt})
		}
		var next projectadoption.BrownfieldSession
		var plan *projectadoption.AdoptionPlan
		switch opts.brownfieldAction {
		case "begin":
			var request projectadoption.ReverseIterationRequest
			if err := decodeClosedProjectJSON(data, &request); err != nil {
				return fmt.Errorf("decode Brownfield begin input: %w", err)
			}
			next, err = projectadoption.BeginReverseIteration(sourceRoot, target, prior, request)
		case "propose":
			var request brownfieldProposalInput
			if err := decodeClosedProjectJSON(data, &request); err != nil {
				return fmt.Errorf("decode Brownfield proposal input: %w", err)
			}
			next, err = projectadoption.RecordManagerProposal(prior, request.IterationID, request.Proposal)
		case "integrate":
			var request brownfieldIntegrationInput
			if err := decodeClosedProjectJSON(data, &request); err != nil {
				return fmt.Errorf("decode Brownfield integration input: %w", err)
			}
			next, err = projectadoption.IntegrateManagerProposal(prior, request.IterationID, request.Integration.ManagerID, request.Integration)
		case "iterate":
			var request brownfieldIterateInput
			if err := decodeClosedProjectJSON(data, &request); err != nil {
				return fmt.Errorf("decode Brownfield iteration input: %w", err)
			}
			next, err = projectadoption.BeginReverseIteration(sourceRoot, target, prior, request.Request)
			if err == nil {
				next, err = projectadoption.RecordManagerProposal(next, request.Request.ID, request.Proposal)
			}
			if err == nil && request.Integration != nil {
				next, err = projectadoption.IntegrateManagerProposal(next, request.Request.ID, request.Integration.ManagerID, *request.Integration)
			}
		case "resolve":
			var request brownfieldResolveInput
			if err := decodeClosedProjectJSON(data, &request); err != nil {
				return fmt.Errorf("decode Brownfield resolution input: %w", err)
			}
			next, err = projectadoption.RecordSessionResolution(prior, request.IterationID, request.Resolution)
		case "plan":
			var request brownfieldPlanInput
			if err := decodeClosedProjectJSON(data, &request); err != nil {
				return fmt.Errorf("decode Brownfield plan input: %w", err)
			}
			if opts.write || opts.expect != "" {
				return errors.New("Brownfield plan is a preview and does not write a session stage")
			}
			schemaDigest, buildDigest, bindingErr := projectadoption.CurrentBindings(projectmodel.Schema())
			if bindingErr != nil {
				return bindingErr
			}
			planValue, planErr := projectadoption.PlanSessionAdoption(sourceRoot, target, prior, request.IterationID, schemaDigest, buildDigest)
			if planErr != nil {
				return planErr
			}
			plan = &planValue
			overview := overviewBrownfieldSession(prior)
			return writeJSON(out, brownfieldResult{Status: "preview", Action: "plan", SessionDigest: prior.Digest, Session: &overview, Readiness: &readiness, Plan: plan})
		default:
			return fmt.Errorf("unsupported Brownfield action %q", opts.brownfieldAction)
		}
		if err != nil {
			return err
		}
		return emitBrownfieldSession(opts, sourceRoot, opts.brownfieldAction, prior.Digest, next, &readiness, plan, out)
	default:
		return errors.New("Brownfield action must be start, begin, context, propose, integrate, iterate, resolve, plan, apply-adoption, or resume")
	}
}

func emitBrownfieldSession(opts options, sourceRoot, action, priorDigest string, session projectadoption.BrownfieldSession, readiness *projectadoption.Readiness, plan *projectadoption.AdoptionPlan, out io.Writer) error {
	status := "preview"
	if opts.write {
		expected := priorDigest
		if action == "start" {
			// A new ledger has no prior digest; the preview digest is the
			// creation token and WriteBrownfieldSession verifies it.
			expected = opts.expect
			if expected != session.Digest {
				return errors.New("--expect does not match the Brownfield session creation preview digest")
			}
		}
		if action != "start" && expected != opts.expect {
			return errors.New("--expect does not match the prior Brownfield session digest")
		}
		if _, err := projectadoption.WriteBrownfieldSession(sourceRoot, session, expected); err != nil {
			return err
		}
		status = "recorded"
	}
	overview := overviewBrownfieldSession(session)
	return writeJSON(out, brownfieldResult{Status: status, Action: action, PriorSessionDigest: priorDigest, SessionDigest: session.Digest, Session: &overview, Plan: plan})
}

type brownfieldStartInput struct {
	Discovery     projectadoption.Discovery     `json:"discovery"`
	ScopeStatuses []projectadoption.ScopeStatus `json:"scopeStatuses"`
}

type brownfieldIterateInput struct {
	Request     projectadoption.ReverseIterationRequest `json:"request"`
	Proposal    projectadoption.ManagerProposal         `json:"proposal"`
	Integration *projectadoption.ManagerIntegration     `json:"integration,omitempty"`
}

type brownfieldProposalInput struct {
	IterationID string                          `json:"iterationId"`
	Proposal    projectadoption.ManagerProposal `json:"proposal"`
}

type brownfieldIntegrationInput struct {
	IterationID string                             `json:"iterationId"`
	Integration projectadoption.ManagerIntegration `json:"integration"`
}

type brownfieldResolveInput struct {
	IterationID string                     `json:"iterationId"`
	Resolution  projectadoption.Resolution `json:"resolution"`
}

type brownfieldPlanInput struct {
	IterationID string `json:"iterationId"`
}

type brownfieldContextInput struct {
	IterationID string `json:"iterationId"`
	Phase       string `json:"phase,omitempty"`
}

type brownfieldApplyAdoptionInput struct {
	IterationID        string `json:"iterationId"`
	ExpectedPlanDigest string `json:"expectedPlanDigest"`
}

type brownfieldResult struct {
	Status             string                                     `json:"status"`
	Action             string                                     `json:"action"`
	PriorSessionDigest string                                     `json:"priorSessionDigest,omitempty"`
	SessionDigest      string                                     `json:"sessionDigest"`
	Session            *brownfieldSessionOverview                 `json:"session,omitempty"`
	Readiness          *projectadoption.Readiness                 `json:"readiness,omitempty"`
	Plan               *projectadoption.AdoptionPlan              `json:"plan,omitempty"`
	ManagerContext     *projectadoption.ManagerReverseContext     `json:"managerContext,omitempty"`
	IntegrationContext *projectadoption.ManagerIntegrationContext `json:"integrationContext,omitempty"`
	Receipt            *projectadoption.AdoptionReceipt           `json:"receipt,omitempty"`
}

// brownfieldSessionOverview exposes only fixed bases and workflow metadata.
// The durable ledger remains the full validator input, but CLI responses never
// return unassigned source bodies or private Manager report content.
type brownfieldSessionOverview struct {
	APIVersion          string                        `json:"apiVersion"`
	ID                  string                        `json:"id"`
	Digest              string                        `json:"digest"`
	Source              brownfieldSourceOverview      `json:"source"`
	Target              brownfieldTargetOverview      `json:"target"`
	TargetContextDigest string                        `json:"targetContextDigest"`
	Scopes              []brownfieldScopeOverview     `json:"scopes"`
	Iterations          []brownfieldIterationOverview `json:"iterations"`
	Adoptions           []brownfieldAdoptionOverview  `json:"adoptions"`
}

type brownfieldSourceOverview struct {
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

type brownfieldTargetOverview struct {
	Root          string `json:"root"`
	Revision      string `json:"revision"`
	ProjectDigest string `json:"projectDigest"`
	ModelDigest   string `json:"modelDigest"`
}

type brownfieldScopeOverview struct {
	ScopeID string `json:"scopeId"`
	Status  string `json:"status"`
}

type brownfieldIterationOverview struct {
	ID                  string `json:"id"`
	ParentIterationID   string `json:"parentIterationId,omitempty"`
	ManagerID           string `json:"managerId"`
	TargetContextDigest string `json:"targetContextDigest"`
	ProposalDigest      string `json:"proposalDigest,omitempty"`
	IntegrationDigest   string `json:"integrationDigest,omitempty"`
	ResolutionDigest    string `json:"resolutionDigest,omitempty"`
}

type brownfieldAdoptionOverview struct {
	IterationID     string `json:"iterationId"`
	PlanDigest      string `json:"planDigest"`
	ReceiptStatus   string `json:"receiptStatus"`
	CandidateDigest string `json:"candidateDigest"`
}

func overviewBrownfieldSession(session projectadoption.BrownfieldSession) brownfieldSessionOverview {
	result := brownfieldSessionOverview{
		APIVersion: session.APIVersion, ID: session.ID, Digest: session.Digest,
		Source: brownfieldSourceOverview{Root: session.Source.Identity.Root, DiscoveryID: session.Source.ID, Commit: session.Source.Commit,
			Digest: session.Source.Digest, ScopeRootCount: len(session.Source.ScopeRoots), SelectedPathCount: len(session.Source.Selected),
			ExclusionCount: len(session.Source.Exclusions), UnselectedPathCount: len(session.Source.Unselected), EvidenceCount: len(session.Source.Evidence)},
		Target:              brownfieldTargetOverview{Root: session.Target.Root, Revision: session.Target.Revision, ProjectDigest: session.Target.ProjectDigest, ModelDigest: session.Target.ModelDigest},
		TargetContextDigest: session.TargetContext.Digest,
		Scopes:              make([]brownfieldScopeOverview, 0, len(session.Scopes)),
		Iterations:          make([]brownfieldIterationOverview, 0, len(session.Iterations)),
		Adoptions:           make([]brownfieldAdoptionOverview, 0, len(session.Adoptions)),
	}
	for _, scope := range session.Scopes {
		result.Scopes = append(result.Scopes, brownfieldScopeOverview{ScopeID: scope.ScopeID, Status: scope.Status})
	}
	for _, iteration := range session.Iterations {
		item := brownfieldIterationOverview{ID: iteration.ID, ParentIterationID: iteration.ParentIterationID, ManagerID: iteration.ManagerID, TargetContextDigest: iteration.TargetContextDigest}
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
		result.Adoptions = append(result.Adoptions, brownfieldAdoptionOverview{IterationID: adoption.IterationID, PlanDigest: adoption.Plan.PlanDigest,
			ReceiptStatus: adoption.Receipt.Status, CandidateDigest: adoption.Receipt.CandidateDigest})
	}
	return result
}

func decodeClosedProjectJSON(data []byte, target any) error {
	if len(data) == 0 || len(data) > 32<<20 {
		return errors.New("Brownfield input must contain one JSON object no larger than 32 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("Brownfield input must contain exactly one JSON value")
		}
		return err
	}
	return nil
}
