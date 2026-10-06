package host

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
)

func TestSameRefreshSelectedContractNormalizesOnlyGlobalFreshnessFields(t *testing.T) {
	base := canonical.ProjectionRequest{
		Revision: "old-revision", ModelDigest: "old-model", RequestDigest: "old-request",
		Projection:       core.Definition{APIVersion: "markitect.foundation/v1", Kind: "Projection", Metadata: core.Metadata{Name: "billing", Namespace: "example"}, Purpose: "same", Source: core.Source{Path: "projection.yaml", Digest: "file-digest"}},
		Binding:          canonical.ProjectionBinding{Module: "billing-module"},
		ModulePin:        canonical.Pin{Name: "billing-module", Version: "1.0.0", Digest: "module-digest"},
		Projector:        canonical.ProjectorRegistration{ID: "dotnet", Version: "v1", Target: "dotnet", AllowedRoots: []string{"src/billing"}},
		Definitions:      []core.Definition{{APIVersion: "example/v1", Kind: "UseCase", Metadata: core.Metadata{Name: "invoice", Namespace: "billing"}, Purpose: "issue invoice", Spec: map[string]any{"status": "open"}, Source: core.Source{Path: "billing.yaml", Digest: "billing-file"}}},
		Schemas:          []core.Schema{{APIVersion: "example/v1", Purpose: "billing", Source: core.Source{Path: "schema.yaml", Digest: "schema-file"}}},
		Policies:         []core.Definition{{APIVersion: "markitect.foundation/v1", Kind: "ProjectionPolicy", Metadata: core.Metadata{Name: "billing-policy", Namespace: "example"}, Purpose: "same policy", Source: core.Source{Path: "policy.yaml", Digest: "policy-file"}}},
		TargetRepository: ".", TargetPath: "src/billing", TargetPrefix: "src/billing",
		TargetFiles: map[string][]byte{"src/billing/invoice.cs": []byte("old")}, TargetDigests: map[string]string{"src/billing/invoice.cs": "old-digest"},
	}
	current := base
	current.Revision, current.ModelDigest, current.RequestDigest = "new-revision", "new-model", "new-request"
	current.TargetFiles = map[string][]byte{"src/billing/invoice.cs": []byte("new")}
	current.TargetDigests = map[string]string{"src/billing/invoice.cs": "new-digest"}
	if !sameRefreshSelectedContract(base, current) {
		t.Fatal("global freshness and target evidence changes should not alter selected canonical contract")
	}
	changed := current
	changed.Definitions = append([]core.Definition(nil), current.Definitions...)
	changed.Definitions[0].Purpose = "different meaning"
	if sameRefreshSelectedContract(base, changed) {
		t.Fatal("selected Definition change was normalized away")
	}
	changed = current
	changed.Policies = append([]core.Definition(nil), current.Policies...)
	changed.Policies[0].Purpose = "different rule"
	if sameRefreshSelectedContract(base, changed) {
		t.Fatal("selected Policy change was normalized away")
	}
	changed = current
	changed.ModulePin.Digest = "other-module-digest"
	if sameRefreshSelectedContract(base, changed) {
		t.Fatal("Module binding change was normalized away")
	}
}

func TestCanonicalEvidenceRefreshCheckInputsAreProjectionScoped(t *testing.T) {
	cfg := CanonicalControllerConfig{
		CheckInputs: []string{"checks/common.go"},
		AssuranceScopes: []CanonicalAssuranceScope{
			{ProjectionID: "billing", CheckInputs: []string{"checks/billing.go"}},
			{ProjectionID: "orders", CheckInputs: []string{"checks/orders.go"}},
		},
	}
	got := canonicalEvidenceRefreshCheckInputs(cfg, []string{"billing"})
	if !equalStringSets(got, []string{"checks/common.go", "checks/billing.go"}) {
		t.Fatalf("selected refresh check inputs = %#v", got)
	}
}
