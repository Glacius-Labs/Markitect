package examples

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const recursiveAssuranceConfigPath = "examples/recursive-assurance/canonical.yaml"

func TestRecursiveAssuranceScopesAreNestedAndIndependentlyChecked(t *testing.T) {
	root, revision := publicExampleFixture(t, "examples/recursive-assurance")
	fixed, err := host.LoadCanonicalSource(root, revision, recursiveAssuranceConfigPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if fixed.Snapshot.Provisional || fixed.Snapshot.ID != revision || fixed.Model.Revision != revision {
		t.Fatalf("source is not fixed to HEAD %q", revision)
	}
	if len(fixed.Diagnostics) != 0 {
		t.Fatalf("canonical fixture has diagnostics: %#v", fixed.Diagnostics)
	}
	if len(fixed.Pins) != 6 || len(fixed.Activation.Projectors) != 4 {
		t.Fatalf("want six exact package pins and four bound Projection Modules; pins=%d bindings=%d", len(fixed.Pins), len(fixed.Activation.Projectors))
	}

	observed, err := source.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	observed = cloneSnapshot(observed)
	observed.ID = ""
	observed.Provisional = true

	cases := []struct {
		name, target, check string
		definitions         []string
		policies            []string
	}{
		{"orders-dotnet", "src/Orders", "recursive-assurance-orders", []string{"OrderRules/orders/calculate-line-total"}, []string{"orders-dotnet"}},
		{"billing-dotnet", "src/Billing", "recursive-assurance-billing", []string{"BillingQuery/billing/sum-outstanding"}, []string{"billing-dotnet"}},
		{"checkout-dotnet", "src/Checkout", "recursive-assurance-checkout", []string{"CheckoutComposition/checkout/add-order-to-outstanding", "OrderRules/orders/calculate-line-total", "BillingQuery/billing/sum-outstanding"}, []string{"checkout-dotnet", "orders-dotnet", "billing-dotnet"}},
		{"commerce-dotnet", "src/Commerce", "recursive-assurance-commerce", []string{"CommerceComposition/commerce/combine-two-checkouts", "CheckoutComposition/checkout/add-order-to-outstanding", "OrderRules/orders/calculate-line-total", "BillingQuery/billing/sum-outstanding"}, []string{"commerce-dotnet", "checkout-dotnet", "orders-dotnet", "billing-dotnet"}},
	}
	seenTargets := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "recursive-assurance", Name: tc.name}
			var check authoring.Check
			for _, configured := range fixed.Config.Checks {
				if configured.Name == tc.check {
					check = configured
					break
				}
			}
			if check.Name == "" {
				t.Fatalf("missing check %q", tc.check)
			}
			prepared, err := host.PrepareCanonicalProjection(fixed, observed, id, "recursive-assurance-shape/1", host.Hash([]byte("recursive-assurance-shape")), nil, check)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSuffix(prepared.Request.TargetPath, "/"); got != tc.target {
				t.Fatalf("target %q, want %q", got, tc.target)
			}
			if seenTargets[tc.target] {
				t.Errorf("target %q is shared between Projection scopes", tc.target)
			}
			seenTargets[tc.target] = true
			if len(prepared.Request.Definitions) != len(tc.definitions) {
				t.Fatalf("selected %d Definitions, want %d", len(prepared.Request.Definitions), len(tc.definitions))
			}
			wantDefs := map[string]bool{}
			for _, v := range tc.definitions {
				wantDefs[v] = true
			}
			for _, def := range prepared.Request.Definitions {
				identity := def.Identity()
				key := def.Kind + "/" + identity.Namespace + "/" + identity.Name
				if !wantDefs[key] {
					t.Errorf("unexpected selected Definition %s", key)
				}
				delete(wantDefs, key)
			}
			if len(wantDefs) > 0 {
				t.Errorf("missing selected Definitions: %#v", wantDefs)
			}
			if len(prepared.Request.Policies) != len(tc.policies) {
				t.Fatalf("selected %d policies, want %d", len(prepared.Request.Policies), len(tc.policies))
			}
			wantPolicies := map[string]bool{}
			for _, v := range tc.policies {
				wantPolicies[v] = true
			}
			for _, policy := range prepared.Request.Policies {
				delete(wantPolicies, policy.Identity().Name)
			}
			if len(wantPolicies) > 0 {
				t.Errorf("missing policies: %#v", wantPolicies)
			}
			if len(prepared.Request.Projector.RequiredChecks) != 1 || prepared.Request.Projector.RequiredChecks[0] != tc.check {
				t.Errorf("required checks = %#v, want exactly %q", prepared.Request.Projector.RequiredChecks, tc.check)
			}
			if prepared.Plan != nil || !hasRecursiveEscalation(prepared.Escalations, "projection.candidate-required") {
				t.Errorf("missing candidate should remain incomplete, plan=%t escalations=%#v", prepared.Plan != nil, prepared.Escalations)
			}
		})
	}
	if len(seenTargets) != 4 {
		t.Fatalf("tested %d independent roots, want 4", len(seenTargets))
	}
}

func hasRecursiveEscalation(escalations []host.CanonicalProjectionEscalation, code string) bool {
	for _, e := range escalations {
		if e.Code == code {
			return true
		}
	}
	return false
}
