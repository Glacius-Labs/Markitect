package projectcli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectapp"
	"io"
)

func runBrownfield(opts options, out io.Writer) error {
	if opts.brownfieldAction == "run" {
		return runBrownfieldManagerStage(opts, out, projectadoption.AgentExecManagerRunInvoker{})
	}
	operation := projectapp.BrownfieldOperation{Root: opts.repo, SourceRoot: opts.sourceRepo, Revision: opts.revision, SessionID: opts.sessionID, Action: opts.brownfieldAction, Write: opts.write, ExpectedDigest: opts.expect}
	var target any
	switch opts.brownfieldAction {
	case "start":
		operation.Input.Start = &projectapp.BrownfieldStartInput{}
		target = operation.Input.Start
	case "begin":
		operation.Input.Begin = &projectadoption.ReverseIterationRequest{}
		target = operation.Input.Begin
	case "context":
		operation.Input.Context = &projectapp.BrownfieldContextInput{}
		target = operation.Input.Context
	case "propose":
		operation.Input.Proposal = &projectapp.BrownfieldProposalInput{}
		target = operation.Input.Proposal
	case "integrate":
		operation.Input.Integration = &projectapp.BrownfieldIntegrationInput{}
		target = operation.Input.Integration
	case "iterate":
		operation.Input.Iterate = &projectapp.BrownfieldIterateInput{}
		target = operation.Input.Iterate
	case "resolve":
		operation.Input.Resolve = &projectapp.BrownfieldResolveInput{}
		target = operation.Input.Resolve
	case "plan":
		operation.Input.Plan = &projectapp.BrownfieldPlanInput{}
		target = operation.Input.Plan
	case "apply-adoption":
		operation.Input.Apply = &projectapp.BrownfieldApplyAdoptionInput{}
		target = operation.Input.Apply
	case "resume":
		if opts.input != "" {
			return errors.New("Brownfield resume does not accept --input")
		}
	case "record-adoption":
		return errors.New("caller-supplied adoption receipts are not accepted; use plan, then apply-adoption with the reviewed plan digest")
	default:
		return errors.New("unsupported Brownfield action")
	}
	if target != nil {
		if opts.input == "" {
			return fmt.Errorf("Brownfield %s requires --input", opts.brownfieldAction)
		}
		sourceRoot := opts.sourceRepo
		if sourceRoot == "" {
			sourceRoot = opts.repo
		}
		data, err := readRecord(sourceRoot, opts.input)
		if err != nil {
			return err
		}
		if err := decodeClosedProjectJSON(data, target); err != nil {
			return fmt.Errorf("decode Brownfield %s input: %w", opts.brownfieldAction, err)
		}
	}
	result, err := (projectapp.Operations{}).Brownfield(operation)
	if err != nil {
		return err
	}
	return writeJSON(out, result)
}

type brownfieldStartInput = projectapp.BrownfieldStartInput
type brownfieldIterateInput = projectapp.BrownfieldIterateInput
type brownfieldProposalInput = projectapp.BrownfieldProposalInput
type brownfieldIntegrationInput = projectapp.BrownfieldIntegrationInput
type brownfieldResolveInput = projectapp.BrownfieldResolveInput
type brownfieldPlanInput = projectapp.BrownfieldPlanInput
type brownfieldContextInput = projectapp.BrownfieldContextInput
type brownfieldApplyAdoptionInput = projectapp.BrownfieldApplyAdoptionInput
type brownfieldResult = projectapp.BrownfieldResult
type brownfieldSessionOverview = projectapp.BrownfieldSessionOverview
type brownfieldSourceOverview = projectapp.BrownfieldSourceOverview
type brownfieldTargetOverview = projectapp.BrownfieldTargetOverview
type brownfieldScopeOverview = projectapp.BrownfieldScopeOverview
type brownfieldIterationOverview = projectapp.BrownfieldIterationOverview
type brownfieldAdoptionOverview = projectapp.BrownfieldAdoptionOverview

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
