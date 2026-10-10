package host

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/records"
)

func TestCanonicalVerificationRejectsOutOfScopeAndCanonicalArtifactClaims(t *testing.T) {
	for _, path := range []string{"scratch/unrelated.txt", "examples/canonical-projection/canonical.yaml"} {
		t.Run(path, func(t *testing.T) {
			fixed, observed := reconciliationFixture(t)
			active := reconciliationRecords(t, fixed, observed)
			var record records.ProjectionRecord
			for _, candidate := range active {
				if candidate.Module.Name == "markitect-dotnet" {
					record = candidate
				}
			}
			target := *observed
			target.Provisional = false
			target.ID = strings.Repeat("b", 40)
			data, exists := target.Files[path]
			if !exists {
				data = []byte("unrelated")
				target.Files[path] = data
			}
			mode := target.Modes[path]
			if mode == "" {
				mode = snapshot.RegularMode
			}
			target.Modes[path] = mode
			facts := []records.ArtifactFact{{Path: path, Digest: sha256Prefix(sha256Hex(data)), Mode: mode}}
			digest, err := records.TargetSnapshotDigest(facts)
			if err != nil {
				t.Fatal(err)
			}
			record.Artifacts = []records.Artifact{{Path: path, Digest: facts[0].Digest, Mode: mode, Change: records.ChangeRetained}}
			record.TargetSnapshotDigest = digest
			record, err = records.NewProjectionRecord(record)
			if err != nil {
				t.Fatal(err)
			}
			_, err = VerifyCanonicalProjection(fixed, &target, record, records.VerifierIdentity{})
			if err == nil || (!strings.Contains(err.Error(), "outside") && !strings.Contains(err.Error(), "canonical source")) {
				t.Fatalf("misowned artifact accepted or wrong refusal: %v", err)
			}
		})
	}
}
func TestCanonicalVerificationRequiresModelRevisionToMatchSourceSnapshot(t *testing.T) {
	fixed, observed := reconciliationFixture(t)
	active := reconciliationRecords(t, fixed, observed)
	target := *observed
	target.Provisional = false
	target.ID = strings.Repeat("b", 40)
	fixed.Model, fixed.Diagnostics = core.Compile(fixed.Model.Schemas, fixed.Model.Definitions, strings.Repeat("c", 40))
	if _, err := VerifyCanonicalProjection(fixed, &target, active[0], records.VerifierIdentity{}); err == nil || !strings.Contains(err.Error(), "fixed source revision") {
		t.Fatalf("mismatched model revision accepted: %v", err)
	}
}
