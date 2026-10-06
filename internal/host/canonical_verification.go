package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

type CanonicalVerification struct {
	EvidenceRevision       string                     `json:"evidenceRevision"`
	EvidenceSnapshotDigest string                     `json:"evidenceSnapshotDigest"`
	Result                 records.VerificationResult `json:"result"`
	Gates                  []GateResult               `json:"gates"`
}

// VerifyCanonicalProjection verifies a supplied materialization record against
// one immutable target snapshot. Each check identity binds its exact command
// and the complete evidence snapshot, including checker source bytes. This is
// bounded command evidence, not independent semantic judgment or authorization.
func VerifyCanonicalProjection(fixed *CanonicalSource, target *snapshot.Snapshot, record records.ProjectionRecord, verifier records.VerifierIdentity) (CanonicalVerification, error) {
	report := CanonicalVerification{}
	if fixed == nil || fixed.Snapshot == nil || target == nil || fixed.Snapshot.Provisional || target.Provisional || !canonicalRevisionPattern.MatchString(fixed.Snapshot.ID) || !canonicalRevisionPattern.MatchString(target.ID) {
		return report, errors.New("canonical verification requires fixed full source and evidence revisions")
	}
	if fixed.Model.Revision != fixed.Snapshot.ID {
		return report, errors.New("canonical Model does not bind the fixed source revision")
	}
	if len(fixed.Diagnostics) != 0 {
		return report, errors.New("canonical source has structural diagnostics")
	}
	if err := validateCanonicalSourceUnchanged(fixed, target); err != nil {
		return report, err
	}
	if err := records.ValidateProjectionRecord(record); err != nil {
		return report, err
	}
	if record.State != records.StateMaterializedUnverified {
		return report, errors.New("only a complete materialization record can yield verification")
	}
	identity, err := projectionIdentityFromKey(record.ProjectionID)
	if err != nil {
		return report, err
	}
	projection, ok := fixed.Model.Definition(identity)
	if !ok {
		return report, errors.New("record Projection is absent from the fixed canonical model")
	}
	targetFiles, err := canonicalProjectionTargetFiles(projection, target.Files)
	if err != nil {
		return report, err
	}
	request, err := canonical.BindProjection(fixed.Model, fixed.Activation, fixed.Config.ProjectionBindings, identity, targetFiles)
	if err != nil {
		return report, err
	}
	if record.Revision != fixed.Snapshot.ID || record.ModelDigest != request.ModelDigest || record.Module.Name != request.ModulePin.Name || record.Module.Version != request.ModulePin.Version || record.Module.Digest != request.ModulePin.Digest || record.Projector.ID != request.Projector.ID || record.Projector.Version != request.Projector.Version {
		return report, errors.New("record canonical source or runtime capability binding is stale")
	}
	scope, policies := []string{}, []string{}
	for _, d := range request.Definitions {
		scope = append(scope, d.Identity().Key())
	}
	for _, d := range request.Policies {
		policies = append(policies, d.Identity().Key())
	}
	sort.Strings(scope)
	sort.Strings(policies)
	if !reflect.DeepEqual(scope, record.ScopeIDs) || !equalStringSets(policies, record.PolicyIDs) {
		return report, errors.New("record scope or selected policies differ from canonical intent")
	}
	protected := map[string]bool{}
	for _, name := range canonicalSourcePaths(fixed) {
		protected[name] = true
	}
	facts := []records.ArtifactFact{}
	for _, artifact := range record.Artifacts {
		if protected[artifact.Path] {
			return report, fmt.Errorf("recorded target %q overlaps canonical source", artifact.Path)
		}
		if _, allowed := targetFiles[artifact.Path]; !allowed {
			return report, fmt.Errorf("recorded target %q is outside the selected representation scope", artifact.Path)
		}
		data, exists := target.Files[artifact.Path]
		if !exists {
			return report, fmt.Errorf("recorded target %q is missing", artifact.Path)
		}
		mode := target.Modes[artifact.Path]
		if mode == "" {
			mode = snapshot.RegularMode
		}
		facts = append(facts, records.ArtifactFact{Path: artifact.Path, Role: records.RoleProjectionTarget, Digest: sha256Prefix(sha256Hex(data)), Mode: mode})
	}
	targetDigest, err := records.TargetSnapshotDigest(facts)
	if err != nil {
		return report, err
	}
	if targetDigest != record.TargetSnapshotDigest {
		return report, errors.New("recorded target bytes or modes have drifted")
	}
	names, escalations, err := selectedCanonicalChecks(request.Projector, true, fixed.Config.Checks)
	if err != nil {
		return report, err
	}
	if len(escalations) != 0 {
		return report, &VerifyError{Kind: "incomplete-evidence", Err: errors.New(escalations[0].Message)}
	}
	checks := selectedAuthoringChecks(names, fixed.Config.Checks)
	report.EvidenceRevision = target.ID
	report.EvidenceSnapshotDigest = sha256Prefix(target.Digest())
	gates, verifyErr := VerifySnapshotChecks(target, checks)
	report.Gates = gates
	byName := map[string]GateResult{}
	for _, gate := range gates {
		byName[gate.Name] = gate
	}
	results := []records.CheckResult{}
	for _, check := range checks {
		outcome := records.CheckIncomplete
		if gate, exists := byName[check.Name]; exists {
			outcome = records.CheckPassed
			if gate.ExitCode != 0 {
				outcome = records.CheckFailed
			}
		}
		results = append(results, records.CheckResult{ID: check.Name, Version: "fixed-command-input/v1", Digest: CanonicalCheckEvidenceDigest(check, target), Outcome: outcome})
	}
	outcome := records.OutcomePassed
	reason := "Only declared commands passed at the stated fixed evidence snapshot; semantic sufficiency and verifier independence remain unproven."
	if verifyErr != nil {
		outcome = records.OutcomeIncomplete
		var classified *VerifyError
		if errors.As(verifyErr, &classified) && classified.Kind == "gate-failure" {
			outcome = records.OutcomeFailed
		}
		reason = verifyErr.Error()
	}
	report.Result, err = records.NewVerificationResult(records.VerificationResult{
		RecordID: record.ID, Revision: record.Revision, ModelDigest: record.ModelDigest, TargetSnapshotDigest: targetDigest,
		EvidenceRevision: target.ID, EvidenceSnapshotDigest: sha256Prefix(target.Digest()),
		Verifier: verifier, Checks: results, Outcome: outcome, Reason: reason,
	})
	if err != nil {
		return report, err
	}
	return report, verifyErr
}

// CanonicalCheckEvidenceDigest binds command definition plus exact complete
// immutable evidence input. It does not claim the PATH executable is hashed.
func CanonicalCheckEvidenceDigest(check authoring.Check, evidence *snapshot.Snapshot) string {
	if evidence == nil {
		return ""
	}
	value := struct {
		Check                    authoring.Check
		Revision, SnapshotDigest string
	}{check, evidence.ID, evidence.Digest()}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return sha256Prefix(sha256Hex(encoded))
}
func projectionIdentityFromKey(key string) (core.DefinitionIdentity, error) {
	var values []string
	if err := json.Unmarshal([]byte(key), &values); err != nil || len(values) != 4 {
		return core.DefinitionIdentity{}, errors.New("record Projection identity must be an exact four-component canonical key")
	}
	identity := core.DefinitionIdentity{APIVersion: values[0], Kind: values[1], Namespace: values[2], Name: values[3]}
	if identity.Key() != key {
		return identity, errors.New("record Projection key is not normalized")
	}
	return identity, nil
}
func equalStringSets(a, b []string) bool {
	left, right := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(left)
	sort.Strings(right)
	return reflect.DeepEqual(left, right)
}
