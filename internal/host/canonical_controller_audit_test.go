package host

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

func TestAuditCanonicalControllerRequiresAuditAll(t *testing.T) {
	root, revision, cfg := canonicalControllerFixture(t)
	_, err := AuditCanonicalController(root, revision, revision, "examples/canonical-projection/canonical.yaml", cfg)
	if err == nil || err.Error() != "controller audit requires auditAll=true; runtime configuration is part of verification evidence and is not changed by audit" {
		t.Fatalf("audit without auditAll did not fail explicitly: %v", err)
	}
}

func TestAuditCanonicalControllerMissingEvidenceIsIncompleteAndReadOnly(t *testing.T) {
	root, revision, cfg := canonicalControllerFixture(t)
	cfg.AuditAll = true
	marker := filepath.Join(filepath.Dir(cfg.RecordStore), "audit-must-not-invoke-agent")
	t.Setenv(canonicalControllerActorEnv, "1")
	t.Setenv(canonicalControllerMarkerEnv, marker)

	report, err := AuditCanonicalController(root, revision, revision, "examples/canonical-projection/canonical.yaml", cfg)
	if err != nil {
		t.Fatalf("audit missing materializations: %v", err)
	}
	if report.Status != "incomplete" || report.Digest == "" || report.Coverage != canonicalControllerAuditCoverage {
		t.Fatalf("audit incorrectly closed an empty evidence ledger: %+v", report)
	}
	if len(report.Projections) == 0 {
		t.Fatal("audit omitted canonical Projections with no active records")
	}
	missingRecords, missingScopes := 0, 0
	for _, finding := range report.Findings {
		switch finding.Code {
		case "record.missing-active":
			missingRecords++
		case "assurance.scope-missing":
			missingScopes++
		}
	}
	if missingRecords != len(report.Projections) || missingScopes != len(report.Projections) {
		t.Fatalf("missing canonical ownership and assurance scope were not accounted: projections=%d records=%d scopes=%d", len(report.Projections), missingRecords, missingScopes)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("read-only audit invoked a configured actor, marker stat err=%v", err)
	}
	if _, err := os.Stat(cfg.RecordStore); !os.IsNotExist(err) {
		t.Fatalf("read-only audit initialized the external record store: %v", err)
	}
}

func TestAuditCanonicalControllerRetainsExactExclusionWithoutClosing(t *testing.T) {
	root, revision, cfg := canonicalControllerFixture(t)
	cfg.AuditAll = true
	cfg.TargetExclusions = []CanonicalTargetExclusion{{Path: "docs/represented/deleted.bin", Reason: "outside selected evidence boundary"}}

	report, err := AuditCanonicalController(root, revision, revision, "examples/canonical-projection/canonical.yaml", cfg)
	if err != nil {
		t.Fatalf("audit with an exact target exclusion: %v", err)
	}
	if report.Status != "incomplete" || len(report.Exclusions) != 1 {
		t.Fatalf("exact exclusion disappeared or falsely closed declared scope: status=%s exclusions=%+v", report.Status, report.Exclusions)
	}
	exclusion := report.Exclusions[0]
	if exclusion.Path != cfg.TargetExclusions[0].Path || exclusion.Reason != cfg.TargetExclusions[0].Reason || exclusion.State != "missing" {
		t.Fatalf("exact exclusion is not fully visible in audit: %+v", exclusion)
	}
	for _, finding := range report.Findings {
		if finding.Code == "artifact.unowned" && finding.ArtifactPath == exclusion.Path {
			t.Fatalf("an intentional exact exclusion blocked declared-scope completion: %+v", finding)
		}
	}
}

func TestCanonicalControllerAuditRecheckDetectsInventoryAndByteChanges(t *testing.T) {
	root, revision, cfg := canonicalControllerFixture(t)
	cfg.AuditAll = true
	proposal, err := ProposeCanonicalController(root, revision, revision, "examples/canonical-projection/canonical.yaml", cfg)
	if err != nil {
		t.Fatalf("propose full-scope audit inputs: %v", err)
	}

	newTarget := filepath.Join(root, "docs", "represented", "created-after-plan.md")
	if err := os.MkdirAll(filepath.Dir(newTarget), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newTarget, []byte("new target after inventory\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(newTarget) })
	inventoryChanged, _, err := canonicalControllerAuditRecheckInputs(root, proposal)
	if err != nil || !inventoryChanged {
		t.Fatalf("audit did not detect target created after planning: inventoryChanged=%t err=%v", inventoryChanged, err)
	}

	if len(proposal.InputPaths) == 0 {
		t.Fatal("proposal has no selected canonical inputs for byte recheck")
	}
	changedPath := filepath.Join(root, filepath.FromSlash(proposal.InputPaths[0]))
	original, err := os.ReadFile(changedPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.WriteFile(changedPath, original, 0644) })
	if err := os.WriteFile(changedPath, append(original, []byte("\nchanged after plan\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	_, inputsChanged, err := canonicalControllerAuditRecheckInputs(root, proposal)
	if err != nil || !inputsChanged {
		t.Fatalf("audit did not detect selected input bytes changed after planning: inputsChanged=%t err=%v", inputsChanged, err)
	}
}

func TestAuditCanonicalControllerReportsConfiguredProjectionMissingFromPartialActiveSet(t *testing.T) {
	if !runCanonicalControllerScenarioInChild(t) {
		return
	}
	root, sourceRevision, evidenceRevision, cfg, marker := canonicalControllerVerificationFixture(t)
	cfg.AuditAll = true
	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomePassed)
	verified, err := VerifyCanonicalController(context.Background(), root, sourceRevision, evidenceRevision, "examples/canonical-projection/canonical.yaml", cfg, true)
	if err != nil || verified.Outcome != records.OutcomePassed || len(verified.Results) != 2 {
		t.Fatalf("prepare current two-Projection verification evidence: outcome=%s results=%d err=%v", verified.Outcome, len(verified.Results), err)
	}

	store, _, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil || store == nil || len(active) != 2 {
		t.Fatalf("read verified active records: count=%d err=%v", len(active), err)
	}
	state, err := store.Read()
	if err != nil || len(state.Verifications) < 2 {
		t.Fatalf("fixture did not retain verification evidence before reducing active selection: results=%d err=%v", len(state.Verifications), err)
	}
	keep := state.ActiveSelection.RecordIDs[0]
	state, err = store.SelectActive(state.Head, []string{keep})
	if err != nil || len(state.ActiveSelection.RecordIDs) != 1 {
		t.Fatalf("select one of two verified Projection records: selection=%v err=%v", state.ActiveSelection.RecordIDs, err)
	}
	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomeFailed)
	beforeInvocations := countCanonicalVerifierInvocations(t, marker)

	report, err := AuditCanonicalController(root, sourceRevision, sourceRevision, "examples/canonical-projection/canonical.yaml", cfg)
	if err != nil {
		t.Fatalf("audit one missing active Projection alongside a fresh verification ledger: %v", err)
	}
	if report.Status != "incomplete" || len(report.Projections) != 2 {
		t.Fatalf("partial active selection incorrectly closed or omitted canonical Projections: status=%s projections=%d", report.Status, len(report.Projections))
	}
	missingProjection := ""
	activeCount := 0
	for _, projection := range report.Projections {
		if len(projection.ActiveRecordIDs) == 0 {
			missingProjection = projection.ProjectionID
		} else {
			activeCount++
		}
	}
	if missingProjection == "" || activeCount != 1 {
		t.Fatalf("audit did not preserve the exact mixed active/missing state: active=%d missing=%q", activeCount, missingProjection)
	}
	foundMissing := false
	for _, finding := range report.Findings {
		if finding.Code == "record.missing-active" && finding.ProjectionID == missingProjection {
			foundMissing = true
		}
	}
	if !foundMissing {
		t.Fatalf("audit omitted the configured Projection missing from active selection %s: %+v", missingProjection, report.Findings)
	}
	if got := countCanonicalVerifierInvocations(t, marker); got != beforeInvocations {
		t.Fatalf("read-only audit invoked a Verifier: before=%d after=%d", beforeInvocations, got)
	}
}
