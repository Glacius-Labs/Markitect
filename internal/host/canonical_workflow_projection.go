package host

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/githooks"
)

// prepareGitHooksProjectionInput is a pure adapter from the Host's already
// selected Projection request and observed ledger facts to the Module contract.
// It does not infer checks from policy prose or inspect the repository.
func prepareGitHooksProjectionInput(
	request canonical.ProjectionRequest,
	suppliedChecks []authoring.Check,
	inventory []source.WorkingFileMetadata,
	observed *snapshot.Snapshot,
	previous *records.ProjectionRecord,
) (githooks.Input, []CanonicalProjectionEscalation, error) {
	if observed == nil {
		return githooks.Input{}, nil, fmt.Errorf("Git Hooks projection requires an observed target snapshot")
	}
	names, escalations, err := selectedCanonicalChecks(request.Projector, true, suppliedChecks)
	if err != nil {
		return githooks.Input{}, nil, err
	}
	checks := selectedAuthoringChecks(names, suppliedChecks)
	input := githooks.Input{
		Definitions:       request.Definitions,
		Schemas:           request.Schemas,
		Edges:             request.Edges,
		Policies:          request.Policies,
		TargetPrefix:      request.TargetPrefix,
		AllowedRoots:      request.Projector.AllowedRoots,
		RequestDigest:     request.RequestDigest,
		InventoryComplete: true,
	}
	for _, check := range checks {
		input.RequiredChecks = append(input.RequiredChecks, githooks.NamedCheck{Name: check.Name, Argv: append([]string(nil), check.Run...)})
	}
	for _, entry := range inventory {
		if !stringsHasPathPrefix(entry.Path, request.TargetPrefix) {
			continue
		}
		input.ObservedArtifacts = append(input.ObservedArtifacts, githooks.ArtifactObservation{
			Path: entry.Path, Bytes: append([]byte(nil), observed.Files[entry.Path]...), Mode: entry.Mode,
		})
	}
	if previous != nil {
		input.Previous = &githooks.PriorProjection{RequestDigest: previous.RequestDigest, Complete: previous.State == records.StateMaterializedUnverified}
		for _, artifact := range previous.Artifacts {
			input.Previous.Artifacts = append(input.Previous.Artifacts, githooks.ArtifactBinding{Path: artifact.Path, Digest: artifact.Digest, Mode: artifact.Mode})
		}
	}
	// The scoped reconciler has ownership records but does not load or
	// freshness-validate independent VerificationResults. Never synthesize one.
	return input, escalations, nil
}

func proposeGitHooksProjection(request canonical.ProjectionRequest, checks []authoring.Check, inventory []source.WorkingFileMetadata, observed *snapshot.Snapshot, previous *records.ProjectionRecord) (CanonicalScopedProposal, error) {
	p := CanonicalScopedProposal{ProjectionID: request.Projection.Identity().Key(), Module: request.ModulePin, Request: request}
	if request.Projector.ID != githooks.ProjectorID || request.Projector.Target != githooks.TargetTechnology || request.Projector.Version != "1.0.0" {
		p.Decision = "escalate"
		p.Escalations = []CanonicalProjectionEscalation{{Code: "projection.entrypoint-unsupported", Identity: p.ProjectionID, Message: "unsupported static Git Hooks Projector entrypoint"}}
		return p, nil
	}
	input, checkEscalations, err := prepareGitHooksProjectionInput(request, checks, inventory, observed, previous)
	if err != nil {
		return p, err
	}
	proposal := githooks.Propose(input)
	p.Decision = string(proposal.Decision)
	p.Reasons = proposal.Reasons
	p.EvidenceRefreshRequired = proposal.EvidenceRefreshRequired
	p.Outputs = make(map[string][]byte, len(proposal.Files))
	p.OutputModes = make(map[string]string, len(proposal.Files))
	for _, file := range proposal.Files {
		p.Outputs[file.Path] = append([]byte(nil), file.Content...)
		p.OutputModes[file.Path] = file.Mode
	}
	p.Escalations = append(p.Escalations, checkEscalations...)
	for _, escalation := range proposal.Escalations {
		p.Escalations = append(p.Escalations, CanonicalProjectionEscalation{Code: escalation.Code, Identity: p.ProjectionID, Message: escalation.Message})
	}
	return p, nil
}

func canonicalWorkflowEntrypoint(request canonical.ProjectionRequest) (string, error) {
	if request.Projector.Version != "1.0.0" {
		return "", fmt.Errorf("unsupported static Projector entrypoint version %q", request.Projector.Version)
	}
	switch {
	case request.Projector.ID == "dotnet-source" && request.Projector.Target == "dotnet":
		return "dotnet", nil
	case (request.Projector.ID == "agent-rules-codex" && request.Projector.Target == "codex") || (request.Projector.ID == "agent-rules-claude" && request.Projector.Target == "claude"):
		return "agent-rules", nil
	case request.Projector.ID == "markdown-documentation" && request.Projector.Target == "markdown":
		return "markdown", nil
	case request.Projector.ID == githooks.ProjectorID && request.Projector.Target == githooks.TargetTechnology:
		return "githooks", nil
	default:
		return "", fmt.Errorf("unsupported static Projector entrypoint %q targeting %q", request.Projector.ID, request.Projector.Target)
	}
}
