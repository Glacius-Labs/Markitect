package host

import (
	"context"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

func TestCanonicalControllerCachedVerificationClearsRefreshForUnchangedEvidence(t *testing.T) {
	root, sourceRevision, evidenceRevision, cfg, marker := canonicalControllerVerificationFixture(t)
	cfg.AuditAll = true
	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomePassed)

	verified, err := VerifyCanonicalController(context.Background(), root, sourceRevision, evidenceRevision, "examples/canonical-projection/canonical.yaml", cfg, true)
	if err != nil || verified.Outcome != records.OutcomePassed || len(verified.Results) != 2 {
		t.Fatalf("persist passing verification for both scopes: outcome=%q results=%d err=%v", verified.Outcome, len(verified.Results), err)
	}
	if got := countCanonicalVerifierInvocations(t, marker); got != 2 {
		t.Fatalf("verification invoked %d times, want child and parent exactly once", got)
	}
	for _, result := range verified.Results {
		if result.EvidenceRevision != evidenceRevision || result.EvidenceSnapshotDigest == "" || result.ControllerConfigDigest != verified.ConfigDigest || result.ControllerVerifierInputDigest == "" {
			t.Fatalf("controller result lacks complete cache provenance: %#v", result)
		}
	}

	proposal, err := ProposeCanonicalController(root, sourceRevision, sourceRevision, "examples/canonical-projection/canonical.yaml", cfg)
	if err != nil {
		t.Fatalf("read-only proposal with unchanged cached evidence: %v", err)
	}
	if len(proposal.Plan.Proposals) != 2 || len(proposal.Plan.EvidenceRefreshRequired) != 0 {
		t.Fatalf("unchanged cached evidence did not clear refresh for the complete child/parent cohort: refresh=%v proposals=%#v", proposal.Plan.EvidenceRefreshRequired, proposal.Plan.Proposals)
	}
	for _, item := range proposal.Plan.Proposals {
		if item.Decision != "no-op" || item.EvidenceRefreshRequired {
			t.Fatalf("unchanged passing verifier result did not authorize a current no-op for %s: %#v", item.ProjectionID, item)
		}
	}
	if got := countCanonicalVerifierInvocations(t, marker); got != 2 {
		t.Fatalf("planning invoked verifier again; invocation count=%d", got)
	}

	// A changed runtime verifier identity invalidates the cached PASS while
	// leaving the canonical source and target bytes untouched.
	staleConfig := cfg
	staleConfig.Verifier.ProviderVersion += "/changed"
	stale, err := ProposeCanonicalController(root, sourceRevision, sourceRevision, "examples/canonical-projection/canonical.yaml", staleConfig)
	if err != nil {
		t.Fatalf("proposal after verifier configuration drift: %v", err)
	}
	if len(stale.Plan.EvidenceRefreshRequired) != 2 {
		t.Fatalf("changed verifier configuration reused stale PASS evidence: refresh=%v", stale.Plan.EvidenceRefreshRequired)
	}
	if got := countCanonicalVerifierInvocations(t, marker); got != 2 {
		t.Fatalf("read-only stale-evidence proposal invoked verifier; invocation count=%d", got)
	}

	// A later failed event for the leaf supersedes its earlier PASS. The parent
	// must also refresh because its exact direct-child result changed.
	store, state, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil || store == nil {
		t.Fatalf("read ledger before appending failed latest result: %v", err)
	}
	var leaf records.ProjectionRecord
	for _, record := range active {
		if record.Module.Name == "markitect-dotnet" {
			leaf = record
		}
	}
	var previous records.VerificationResult
	for _, result := range verified.Results {
		if result.RecordID == leaf.ID {
			previous = result
		}
	}
	if leaf.ID == "" || previous.ID == "" {
		t.Fatal("fixture did not provide the leaf record and its passing result")
	}
	failedInput := previous
	failedInput.ID = ""
	failedInput.Outcome = records.OutcomeFailed
	failedInput.Reason = "later failed verifier result"
	for i := range failedInput.Checks {
		if failedInput.Checks[i].ID == canonicalControllerAgentCheckID {
			failedInput.Checks[i].Outcome = records.CheckFailed
		}
	}
	failed, err := records.NewVerificationResult(failedInput)
	if err != nil {
		t.Fatalf("construct later failed leaf result: %v", err)
	}
	if _, err := store.AppendVerification(state.Head, failed); err != nil {
		t.Fatalf("append later failed leaf result: %v", err)
	}
	failedProposal, err := ProposeCanonicalController(root, sourceRevision, sourceRevision, "examples/canonical-projection/canonical.yaml", cfg)
	if err != nil {
		t.Fatalf("proposal after failed latest child result: %v", err)
	}
	if len(failedProposal.Plan.EvidenceRefreshRequired) != 2 {
		t.Fatalf("failed child PASS was reused or did not invalidate its parent: refresh=%v", failedProposal.Plan.EvidenceRefreshRequired)
	}
	if got := countCanonicalVerifierInvocations(t, marker); got != 2 {
		t.Fatalf("read-only failed-evidence proposal invoked verifier; invocation count=%d", got)
	}
}
