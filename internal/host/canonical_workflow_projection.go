package host

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/agentrules"
	"github.com/Glacius-Labs/Markitect/internal/modules/azurepipelines"
	"github.com/Glacius-Labs/Markitect/internal/modules/githooks"
	"github.com/Glacius-Labs/Markitect/internal/modules/markdown"
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
		content, present := observed.Files[entry.Path]
		if !present {
			return githooks.Input{}, nil, fmt.Errorf("Git Hooks inventory entry %q has no observed bytes", entry.Path)
		}
		input.ObservedArtifacts = append(input.ObservedArtifacts, githooks.ArtifactObservation{
			Path: entry.Path, Bytes: append([]byte(nil), content...), Mode: entry.Mode,
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

func prepareAzurePipelinesProjectionInput(
	request canonical.ProjectionRequest,
	suppliedChecks []authoring.Check,
	inventory []source.WorkingFileMetadata,
	observed *snapshot.Snapshot,
	previous *records.ProjectionRecord,
	canonicalAffected bool,
) (azurepipelines.Input, []CanonicalProjectionEscalation, error) {
	if observed == nil {
		return azurepipelines.Input{}, nil, fmt.Errorf("Azure Pipelines projection requires an observed target snapshot")
	}
	names, escalations, err := selectedCanonicalChecks(request.Projector, true, suppliedChecks)
	if err != nil {
		return azurepipelines.Input{}, nil, err
	}
	checks := selectedAuthoringChecks(names, suppliedChecks)
	input := azurepipelines.Input{
		Definitions: request.Definitions, Schemas: request.Schemas, Edges: request.Edges, Policies: request.Policies,
		TargetPrefix: request.TargetPrefix, AllowedRoots: request.Projector.AllowedRoots, RequestDigest: request.RequestDigest,
		CanonicalAffected: canonicalAffected, InventoryComplete: true,
	}
	for _, check := range checks {
		input.Checks = append(input.Checks, azurepipelines.NamedCheck{Name: check.Name, Argv: append([]string(nil), check.Run...)})
	}
	for _, entry := range inventory {
		if !stringsHasPathPrefix(entry.Path, request.TargetPrefix) {
			continue
		}
		content, present := observed.Files[entry.Path]
		if !present {
			return azurepipelines.Input{}, nil, fmt.Errorf("Azure Pipelines inventory entry %q has no observed bytes", entry.Path)
		}
		input.ObservedArtifacts = append(input.ObservedArtifacts, azurepipelines.ArtifactObservation{
			Path: entry.Path, Bytes: append([]byte(nil), content...), Mode: entry.Mode,
		})
	}
	if previous != nil {
		input.Previous = &azurepipelines.PriorProjection{RequestDigest: previous.RequestDigest, Complete: previous.State == records.StateMaterializedUnverified}
		for _, artifact := range previous.Artifacts {
			input.Previous.Artifacts = append(input.Previous.Artifacts, azurepipelines.ArtifactBinding{Path: artifact.Path, Digest: artifact.Digest, Mode: artifact.Mode})
		}
	}
	// Scoped reconciliation does not load independent, freshness-validated
	// VerificationResults, so the adapter cannot claim current evidence.
	return input, escalations, nil
}

func proposeAzurePipelinesProjection(request canonical.ProjectionRequest, checks []authoring.Check, inventory []source.WorkingFileMetadata, observed *snapshot.Snapshot, previous *records.ProjectionRecord, canonicalAffected bool) (CanonicalScopedProposal, error) {
	p := CanonicalScopedProposal{ProjectionID: request.Projection.Identity().Key(), Module: request.ModulePin, Request: request}
	if request.Projector.ID != azurepipelines.ProjectorID || request.Projector.Target != azurepipelines.TargetTechnology || request.Projector.Version != azurepipelines.ModuleVersion {
		p.Decision = "escalate"
		p.Escalations = []CanonicalProjectionEscalation{{Code: "projection.entrypoint-unsupported", Identity: p.ProjectionID, Message: "unsupported static Azure Pipelines Projector entrypoint"}}
		return p, nil
	}
	input, checkEscalations, err := prepareAzurePipelinesProjectionInput(request, checks, inventory, observed, previous, canonicalAffected)
	if err != nil {
		return p, err
	}
	proposal := azurepipelines.Propose(input)
	p.Decision = string(proposal.Decision)
	p.Reasons = proposal.Reasons
	p.EvidenceRefreshRequired = proposal.EvidenceRefreshRequired
	p.Outputs = make(map[string][]byte, len(proposal.Files))
	p.OutputModes = make(map[string]string, len(proposal.Files))
	for path, content := range proposal.Files {
		p.Outputs[path] = append([]byte(nil), content...)
		p.OutputModes[path] = proposal.Mode
	}
	p.Escalations = append(p.Escalations, checkEscalations...)
	for _, escalation := range proposal.Escalations {
		p.Escalations = append(p.Escalations, CanonicalProjectionEscalation{Code: escalation.Code, Identity: p.ProjectionID, Message: escalation.Message})
	}
	return p, nil
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
	case request.Projector.ID == azurepipelines.ProjectorID && request.Projector.Target == azurepipelines.TargetTechnology:
		return "azurepipelines", nil
	default:
		return "", fmt.Errorf("unsupported static Projector entrypoint %q targeting %q", request.Projector.ID, request.Projector.Target)
	}
}

func selectedCanonicalTargetInventory(entries []source.WorkingFileMetadata, prefix string, excluded map[string]struct{}) []source.WorkingFileMetadata {
	selected := make([]source.WorkingFileMetadata, 0, len(entries))
	for _, entry := range entries {
		if _, omit := excluded[entry.Path]; omit || !stringsHasPathPrefix(entry.Path, prefix) {
			continue
		}
		selected = append(selected, entry)
	}
	return selected
}

// renderCanonicalDeterministicProjection composes installed static capabilities.
// Ownership is evaluated by each Module's Propose before controller execution;
// this stage derives only the same exact candidate bytes and declared modes.
func renderCanonicalDeterministicProjection(fixed *CanonicalSource, observed *snapshot.Snapshot, request canonical.ProjectionRequest, checks []authoring.Check) (map[string][]byte, map[string]string, []CanonicalProjectionEscalation, error) {
	files, modes := map[string][]byte{}, map[string]string{}
	entrypoint, err := canonicalWorkflowEntrypoint(request)
	if err != nil {
		return nil, nil, nil, err
	}
	escalations := []CanonicalProjectionEscalation{}
	switch entrypoint {
	case "agent-rules":
		input, err := canonicalProviderProjectionInput(fixed.Model, request)
		if err != nil {
			return nil, nil, nil, err
		}
		file, err := agentrules.Render(input)
		if err != nil {
			escalations = append(escalations, CanonicalProjectionEscalation{Code: "agent-rules.guidance-invalid", Message: err.Error()})
		} else {
			files[file.Path], modes[file.Path] = file.Content, snapshot.RegularMode
		}
	case "markdown":
		rendered := markdown.RenderProjection(markdown.Input{Definitions: request.Definitions, Schemas: request.Schemas, Policies: request.Policies, TargetPrefix: request.TargetPrefix})
		for _, d := range rendered.Diagnostics {
			escalations = append(escalations, CanonicalProjectionEscalation{Code: d.Code, Identity: d.Identity, Message: d.Message})
		}
		if len(escalations) == 0 {
			files = rendered.Files
			for name := range files {
				modes[name] = snapshot.RegularMode
			}
		}
	case "githooks":
		input, selectedEscalations, err := prepareGitHooksProjectionInput(request, checks, nil, observed, nil)
		if err != nil {
			return nil, nil, nil, err
		}
		escalations = append(escalations, selectedEscalations...)
		if len(escalations) == 0 {
			input.InventoryComplete = false // Rendering makes no inventory or ownership claim.
			rendered := githooks.Render(input)
			for _, e := range rendered.Escalations {
				escalations = append(escalations, CanonicalProjectionEscalation{Code: e.Code, Identity: request.Projection.Identity().Key(), Message: e.Message})
			}
			if len(escalations) == 0 {
				for _, file := range rendered.Files {
					files[file.Path], modes[file.Path] = file.Content, file.Mode
				}
			}
		}
	case "azurepipelines":
		input, selectedEscalations, err := prepareAzurePipelinesProjectionInput(request, checks, nil, observed, nil, true)
		if err != nil {
			return nil, nil, nil, err
		}
		escalations = append(escalations, selectedEscalations...)
		if len(escalations) == 0 {
			input.InventoryComplete = false // Rendering makes no inventory or ownership claim.
			rendered := azurepipelines.Render(input)
			for _, d := range rendered.Diagnostics {
				escalations = append(escalations, CanonicalProjectionEscalation{Code: d.Code, Identity: d.Identity, Message: d.Message})
			}
			if len(escalations) == 0 {
				files = rendered.Files
				for name := range files {
					modes[name] = rendered.Mode
				}
			}
		}
	default:
		return nil, nil, nil, fmt.Errorf("static Projector %q requires an explicit candidate", request.Projector.ID)
	}
	return files, modes, escalations, nil
}
