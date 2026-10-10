package host

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/src/internal/host/records"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/markdownreference"
)

// The Host supplies already selected semantic and observed facts. The Module
// owns its representation, lifecycle decision and explicit ownership refusals.
func prepareMarkdownReferenceInput(request canonical.ProjectionRequest, inventory []source.WorkingFileMetadata, observed *snapshot.Snapshot, previous *records.ProjectionRecord, affected bool, verificationValues ...*markdownreference.Verification) (markdownreference.Input, error) {
	var verification *markdownreference.Verification
	if len(verificationValues) > 0 {
		verification = verificationValues[0]
	}
	input := markdownreference.Input{
		Schemas: request.Schemas, Definitions: request.Definitions, Policies: request.Policies,
		Target:   markdownreference.TargetConfig{Technology: request.Projector.Target, Prefix: request.TargetPrefix, AllowedRoots: request.Projector.AllowedRoots},
		Check:    markdownreference.Check{ScopeComplete: true, InventoryComplete: true, CanonicalAffected: affected, RequestDigest: request.RequestDigest},
		Verified: verification,
	}
	for _, definition := range request.Definitions {
		input.Selected = append(input.Selected, definition.Identity())
	}
	if observed == nil {
		return input, fmt.Errorf("Markdown Reference requires an observed snapshot")
	}
	for _, entry := range inventory {
		if !stringsHasPathPrefix(entry.Path, request.TargetPrefix) {
			continue
		}
		content, present := observed.Files[entry.Path]
		if !present {
			return input, fmt.Errorf("Markdown Reference inventory entry %q has no observed bytes", entry.Path)
		}
		input.Observed = append(input.Observed, markdownreference.ArtifactObservation{Path: entry.Path, Bytes: append([]byte(nil), content...), Mode: entry.Mode})
	}
	if previous != nil {
		input.Prior = &markdownreference.PriorRecord{Complete: previous.State == records.StateMaterializedUnverified, RequestDigest: previous.RequestDigest}
		for _, artifact := range previous.Artifacts {
			input.Prior.Artifacts = append(input.Prior.Artifacts, markdownreference.ArtifactBinding{Path: artifact.Path, Digest: artifact.Digest, Mode: artifact.Mode})
		}
	}
	// No independent VerificationResult is loaded here. Never synthesize one.
	return input, nil
}

func proposeMarkdownReferenceProjection(request canonical.ProjectionRequest, inventory []source.WorkingFileMetadata, observed *snapshot.Snapshot, previous *records.ProjectionRecord, affected bool, verificationValues ...*markdownreference.Verification) (CanonicalScopedProposal, error) {
	result := CanonicalScopedProposal{ProjectionID: request.Projection.Identity().Key(), Module: request.ModulePin, Request: request}
	input, err := prepareMarkdownReferenceInput(request, inventory, observed, previous, affected, verificationValues...)
	if err != nil {
		return result, err
	}
	proposal := markdownreference.Propose(input)
	result.Decision, result.Reasons = string(proposal.Decision), proposal.Reasons
	result.Outputs, result.OutputModes = map[string][]byte{}, map[string]string{}
	for name, artifact := range proposal.Files {
		result.Outputs[name], result.OutputModes[name] = append([]byte(nil), artifact.Bytes...), artifact.Mode
	}
	for _, reason := range proposal.Reasons {
		if reason == "evidence-refresh-required" {
			result.EvidenceRefreshRequired = true
		}
	}
	for _, issue := range proposal.Issues {
		result.Escalations = append(result.Escalations, CanonicalProjectionEscalation{Code: issue.Code, Identity: result.ProjectionID, Message: issue.Message})
	}
	return result, nil
}
