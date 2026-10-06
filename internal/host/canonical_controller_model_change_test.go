package host

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// This exercises the real controller with deterministic protocol actors;
// their passing responses do not establish semantic conformity.
func TestCanonicalControllerModelChangeReconcilesBothRepresentationsAndCloses(t *testing.T) {
	const (
		configPath  = "examples/canonical-projection/canonical.yaml"
		useCasePath = "examples/canonical-projection/definitions/create-order.use-case.yaml"
		oldPurpose  = "Allows a customer to create an order."
		newPurpose  = "A customer may submit a validated order."
	)
	root, initialSource, initialEvidence, cfg, marker := canonicalControllerVerificationFixture(t)
	cfg.AuditAll = true
	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomePassed)
	initialVerification, err := VerifyCanonicalController(context.Background(), root, initialSource, initialEvidence, configPath, cfg, true)
	if err != nil || initialVerification.Outcome != records.OutcomePassed || len(initialVerification.Results) != 2 {
		t.Fatalf("persist initial passing verification: outcome=%s results=%d err=%v", initialVerification.Outcome, len(initialVerification.Results), err)
	}
	initialAudit, err := AuditCanonicalController(root, initialSource, initialSource, configPath, cfg)
	if err != nil || initialAudit.Status != "complete" {
		t.Fatalf("initial verified source did not close: status=%s findings=%+v err=%v", initialAudit.Status, initialAudit.Findings, err)
	}

	store, _, oldActive, err := readCanonicalControllerLedger(root, cfg)
	if err != nil || store == nil || len(oldActive) != 2 {
		t.Fatalf("read initial active materializations: count=%d err=%v", len(oldActive), err)
	}
	oldByProjection := map[string]records.ProjectionRecord{}
	oldArtifacts := []string{}
	for _, record := range oldActive {
		oldByProjection[record.ProjectionID] = record
		for _, artifact := range record.Artifacts {
			oldArtifacts = append(oldArtifacts, artifact.Path)
		}
	}
	oldArtifacts = sortedUniquePaths(oldArtifacts)
	beforeSourceChange, err := source.ObserveSelectedWorking(root, oldArtifacts)
	if err != nil {
		t.Fatal(err)
	}

	definitionPath := filepath.Join(root, filepath.FromSlash(useCasePath))
	definitionBytes, err := os.ReadFile(definitionPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(definitionBytes, []byte(oldPurpose)) {
		t.Fatalf("fixture UseCase purpose not found in %s", useCasePath)
	}
	updatedDefinition := bytes.Replace(definitionBytes, []byte(oldPurpose), []byte(newPurpose), 1)
	if bytes.Equal(updatedDefinition, definitionBytes) {
		t.Fatal("UseCase purpose was not changed")
	}
	if err := os.WriteFile(definitionPath, updatedDefinition, 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "add", useCasePath)
	scopedTestGit(t, root, "commit", "-m", "change create-order canonical purpose")
	updatedSource := scopedTestGit(t, root, "rev-parse", "HEAD")
	if updatedSource == initialSource {
		t.Fatal("canonical source revision did not change")
	}
	afterSourceChange, err := source.ObserveSelectedWorking(root, oldArtifacts)
	if err != nil {
		t.Fatal(err)
	}
	if !equalCanonicalValue(beforeSourceChange.Snapshot.Files, afterSourceChange.Snapshot.Files) || !equalCanonicalValue(beforeSourceChange.Snapshot.Modes, afterSourceChange.Snapshot.Modes) {
		t.Fatal("editing canonical intent changed already materialized target bytes")
	}

	staleAudit, err := AuditCanonicalController(root, initialSource, updatedSource, configPath, cfg)
	if err != nil {
		t.Fatalf("audit changed source against old materializations: %v", err)
	}
	if staleAudit.Status != "incomplete" || staleAudit.SourceRevision != updatedSource {
		t.Fatalf("old verification closed a changed canonical model: status=%s source=%s findings=%+v", staleAudit.Status, staleAudit.SourceRevision, staleAudit.Findings)
	}
	if len(staleAudit.Projections) != 2 {
		t.Fatalf("changed source audit did not account for both declared representations: %+v", staleAudit.Projections)
	}
	for _, projection := range staleAudit.Projections {
		if projection.Status == records.OutcomePassed || len(projection.FreshVerificationResultIDs) != 0 {
			t.Fatalf("old verification was accepted for changed source Projection %s: %+v", projection.ProjectionID, projection)
		}
	}

	proposal, err := ProposeCanonicalController(root, initialSource, updatedSource, configPath, cfg)
	if err != nil {
		t.Fatalf("plan changed canonical source: %v", err)
	}
	proposals := map[string]CanonicalScopedProposal{}
	for _, item := range proposal.Plan.Proposals {
		proposals[item.ProjectionID] = item
	}
	if len(proposals) != 2 {
		t.Fatalf("change impact omitted a declared representation: got %d proposals: %+v", len(proposals), proposals)
	}
	for _, record := range oldActive {
		item, ok := proposals[record.ProjectionID]
		if !ok || item.Decision != "work" {
			t.Fatalf("canonical UseCase change did not route work to Projection %s: %+v", record.ProjectionID, item)
		}
	}

	run, err := ExecuteCanonicalController(context.Background(), root, initialSource, updatedSource, configPath, cfg, "test-tool/1", "sha256:"+strings.Repeat("a", 64))
	if err != nil || run.Status != "planned" || len(run.Work) != 2 {
		t.Fatalf("execute changed-model work: status=%s work=%d err=%v", run.Status, len(run.Work), err)
	}
	apply, err := ApplyCanonicalController(root, configPath, cfg, run, run.Digest, true)
	if err != nil || apply.Status != records.StateMaterializedUnverified || len(apply.Records) != 2 {
		t.Fatalf("apply changed-model candidates: status=%s records=%d err=%v", apply.Status, len(apply.Records), err)
	}
	if apply.EvidenceRevision == "" || apply.EvidenceRevision == updatedSource {
		t.Fatalf("Apply did not return an exact evidence revision: %s", apply.EvidenceRevision)
	}

	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomePassed)
	freshVerification, err := VerifyCanonicalController(context.Background(), root, updatedSource, apply.EvidenceRevision, configPath, cfg, true)
	if err != nil || freshVerification.Outcome != records.OutcomePassed || len(freshVerification.Results) != 2 {
		t.Fatalf("verify changed-model materializations: outcome=%s results=%d err=%v", freshVerification.Outcome, len(freshVerification.Results), err)
	}
	finalAudit, err := AuditCanonicalController(root, initialSource, updatedSource, configPath, cfg)
	if err != nil || finalAudit.Status != "complete" {
		t.Fatalf("freshly reconciled source did not close: status=%s findings=%+v err=%v", finalAudit.Status, finalAudit.Findings, err)
	}
	if finalAudit.SourceRevision != updatedSource || finalAudit.Digest == initialAudit.Digest {
		t.Fatal("final audit did not bind the changed source and current evidence")
	}

	_, _, newActive, err := readCanonicalControllerLedger(root, cfg)
	if err != nil || len(newActive) != 2 {
		t.Fatalf("read changed-model active records: count=%d err=%v", len(newActive), err)
	}
	for _, record := range newActive {
		old, ok := oldByProjection[record.ProjectionID]
		if !ok || record.ModelDigest == old.ModelDigest || record.Revision != updatedSource {
			t.Fatalf("materialized record did not bind changed source model: old=%+v new=%+v", old, record)
		}
	}
	if len(freshVerification.Results) != 2 || freshVerification.Results[0].Revision != updatedSource {
		t.Fatalf("fresh results do not bind the changed source: %+v", freshVerification.Results)
	}
	oldResultIDs, newResultIDs := map[string]bool{}, map[string]bool{}
	for _, result := range initialVerification.Results {
		oldResultIDs[result.ID] = true
	}
	for _, result := range freshVerification.Results {
		newResultIDs[result.ID] = true
		if oldResultIDs[result.ID] {
			t.Fatalf("new model reused prior verification result ID %s", result.ID)
		}
	}
	for _, projection := range finalAudit.Projections {
		if len(projection.FreshVerificationResultIDs) != 1 || !newResultIDs[projection.FreshVerificationResultIDs[0]] {
			t.Fatalf("final audit did not bind the fresh result for %s: %+v", projection.ProjectionID, projection)
		}
	}

	markdownRecordFound := false
	for _, record := range newActive {
		if record.Module.Name != "markitect-markdown" {
			continue
		}
		markdownRecordFound = true
		for _, artifact := range record.Artifacts {
			contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(contents), newPurpose) {
				t.Fatalf("Markdown representation %s omits changed UseCase purpose %q:\n%s", artifact.Path, newPurpose, contents)
			}
		}
	}
	if !markdownRecordFound {
		t.Fatal("changed-model active set omitted the declared Markdown representation")
	}
}
