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
			return writeJSON(out, brownfieldResult{Status: "resumed", Action: "resume", SessionDigest: session.Digest, Session: &session, Readiness: &readiness})
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
			managerContext, err := projectadoption.BuildManagerReverseContext(session, request.IterationID)
			if err != nil {
				return err
			}
			return writeJSON(out, brownfieldResult{Status: "context", Action: "context", SessionDigest: session.Digest, ManagerContext: &managerContext})
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
			return writeJSON(out, brownfieldResult{Status: "recorded", Action: "apply-adoption", PriorSessionDigest: prior.Digest, SessionDigest: next.Digest, Session: &next, Plan: &plan, Receipt: &receipt})
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
			return writeJSON(out, brownfieldResult{Status: "preview", Action: "plan", SessionDigest: prior.Digest, Session: &prior, Readiness: &readiness, Plan: plan})
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
	return writeJSON(out, brownfieldResult{Status: status, Action: action, PriorSessionDigest: priorDigest, SessionDigest: session.Digest, Session: &session, Plan: plan})
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
}

type brownfieldApplyAdoptionInput struct {
	IterationID        string `json:"iterationId"`
	ExpectedPlanDigest string `json:"expectedPlanDigest"`
}

type brownfieldResult struct {
	Status             string                                 `json:"status"`
	Action             string                                 `json:"action"`
	PriorSessionDigest string                                 `json:"priorSessionDigest,omitempty"`
	SessionDigest      string                                 `json:"sessionDigest"`
	Session            *projectadoption.BrownfieldSession     `json:"session,omitempty"`
	Readiness          *projectadoption.Readiness             `json:"readiness,omitempty"`
	Plan               *projectadoption.AdoptionPlan          `json:"plan,omitempty"`
	ManagerContext     *projectadoption.ManagerReverseContext `json:"managerContext,omitempty"`
	Receipt            *projectadoption.AdoptionReceipt       `json:"receipt,omitempty"`
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
