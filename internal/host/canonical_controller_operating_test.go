package host

import (
	"context"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
)

func TestCanonicalControllerOperatingBaselineClosure(t *testing.T) {
	root, sourceRevision, evidenceRevision, cfg, marker := canonicalControllerVerificationFixture(t)
	cfg.AuditAll = true
	const configPath = "examples/canonical-projection/canonical.yaml"

	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomePassed)
	verified, err := VerifyCanonicalController(context.Background(), root, sourceRevision, evidenceRevision, configPath, cfg, true)
	if err != nil || verified.Outcome != "passed" {
		t.Fatalf("verify materialized projections: outcome=%q err=%v", verified.Outcome, err)
	}

	audit, err := AuditCanonicalController(root, sourceRevision, sourceRevision, configPath, cfg)
	if err != nil {
		t.Fatalf("audit verified projections: %v", err)
	}
	if audit.Status != "complete" {
		t.Fatalf("freshly verified declared scope did not close: status=%q findings=%+v", audit.Status, audit.Findings)
	}
	if len(audit.Projections) != 2 {
		t.Fatalf("audit projections=%d, want the two declared canonical projections: %+v", len(audit.Projections), audit.Projections)
	}
	for _, projection := range audit.Projections {
		if len(projection.ActiveRecordIDs) != 1 || len(projection.FreshVerificationResultIDs) != 1 || projection.Outcome != "passed" {
			t.Errorf("projection %s lacks exactly one active record and fresh passing verification: %+v", projection.ProjectionID, projection)
		}
	}
}
