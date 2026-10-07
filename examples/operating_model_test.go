package examples

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const operatingModelConfigPath = "examples/operating-model/canonical.yaml"

func TestOperatingModelProjectionScopesAreIndependentAndComposed(t *testing.T) {
	root, revision := publicExampleFixture(t, "examples/operating-model")
	fixed, err := host.LoadCanonicalSource(root, revision, operatingModelConfigPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if fixed.Snapshot.Provisional || fixed.Snapshot.ID != revision || fixed.Model.Revision != revision {
		t.Fatalf("canonical fixture is not bound to full Git HEAD %q: snapshot=%q model=%q provisional=%t", revision, fixed.Snapshot.ID, fixed.Model.Revision, fixed.Snapshot.Provisional)
	}
	if len(fixed.Diagnostics) != 0 {
		t.Fatalf("operating-model canonical source has structural diagnostics: %#v", fixed.Diagnostics)
	}
	if len(fixed.Pins) != 5 || len(fixed.Activation.Projectors) != 3 {
		t.Fatalf("expected five exact Module pins and three activated Projection Modules; pins=%d projectors=%d", len(fixed.Pins), len(fixed.Activation.Projectors))
	}
	if len(fixed.Model.Edges) != 7 {
		t.Fatalf("expected two ProductComposition child edges and five Projection-to-policy edges, got %d", len(fixed.Model.Edges))
	}

	observed, err := source.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	// Use committed bytes for the empty target observation so checkout newline
	// conversion cannot appear as canonical source drift on Windows.
	observed = cloneSnapshot(observed)
	observed.ID = ""
	observed.Provisional = true
	selected := []struct {
		name        string
		target      string
		definition  map[string]bool
		policyCount int
		checkNames  []string
	}{
		{
			name:        "orders-dotnet",
			target:      "src/Orders",
			definition:  map[string]bool{"[\"commerce.operating-model/v1\",\"OrderRules\",\"orders\",\"accepted-order-total\"]": true},
			policyCount: 1,
			checkNames:  []string{"operating-model-orders"},
		},
		{
			name:        "billing-dotnet",
			target:      "src/Billing",
			definition:  map[string]bool{"[\"commerce.operating-model/v1\",\"BillingQuery\",\"billing\",\"outstanding-order-total\"]": true},
			policyCount: 1,
			checkNames:  []string{"operating-model-billing"},
		},
		{
			name:   "product-markdown",
			target: "docs/represented",
			definition: map[string]bool{
				"[\"commerce.operating-model/v1\",\"ProductComposition\",\"commerce\",\"orders-and-billing\"]": true,
				"[\"commerce.operating-model/v1\",\"OrderRules\",\"orders\",\"accepted-order-total\"]":         true,
				"[\"commerce.operating-model/v1\",\"BillingQuery\",\"billing\",\"outstanding-order-total\"]":   true,
			},
			policyCount: 3,
			checkNames:  []string{"operating-model-composition", "operating-model-documentation"},
		},
	}

	for _, projection := range selected {
		t.Run(projection.name, func(t *testing.T) {
			identity := core.DefinitionIdentity{
				APIVersion: "markitect.foundation/v1",
				Kind:       "Projection",
				Namespace:  "operating-model",
				Name:       projection.name,
			}
			checks := fixed.Config.Checks[:0:0]
			for _, name := range projection.checkNames {
				for _, configured := range fixed.Config.Checks {
					if configured.Name == name {
						checks = append(checks, configured)
					}
				}
			}
			if len(checks) != len(projection.checkNames) {
				t.Fatalf("found %d configured checks, want %d", len(checks), len(projection.checkNames))
			}
			prepared, err := host.PrepareCanonicalProjection(
				fixed,
				observed,
				identity,
				"operating-model-shape-test/1",
				host.Hash([]byte("operating-model-shape-test")),
				nil,
				checks...,
			)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSuffix(prepared.Request.TargetPath, "/"); got != projection.target {
				t.Fatalf("Projection target = %q, want %q", got, projection.target)
			}
			if len(prepared.Request.Definitions) != len(projection.definition) {
				t.Fatalf("selected %d Definitions, want %d", len(prepared.Request.Definitions), len(projection.definition))
			}
			for _, definition := range prepared.Request.Definitions {
				key := definition.Identity().Key()
				if !projection.definition[key] {
					t.Errorf("unexpected selected Definition %s", key)
				}
				delete(projection.definition, key)
			}
			if len(projection.definition) != 0 {
				t.Errorf("Projection omitted selected Definitions: %#v", projection.definition)
			}
			if len(prepared.Request.Policies) != projection.policyCount {
				t.Errorf("selected %d ProjectionPolicies, want %d", len(prepared.Request.Policies), projection.policyCount)
			}
			if len(prepared.Request.Projector.RequiredChecks) != len(projection.checkNames) {
				t.Errorf("Projector requires %d checks, want %d", len(prepared.Request.Projector.RequiredChecks), len(projection.checkNames))
			}
			for _, name := range projection.checkNames {
				found := false
				for _, required := range prepared.Request.Projector.RequiredChecks {
					if required == name {
						found = true
					}
				}
				if !found {
					t.Errorf("Projector does not require its owner-selected check %q", name)
				}
			}
			if projection.name == "product-markdown" {
				if prepared.Plan == nil || len(prepared.Outputs) != 1 || len(prepared.Escalations) != 0 {
					t.Fatalf("parent Markdown Prepare did not produce one bounded plan/output: plan=%t outputs=%d escalations=%#v", prepared.Plan != nil, len(prepared.Outputs), prepared.Escalations)
				}
				if len(prepared.Plan.Contracts) != 1 {
					t.Fatalf("parent Markdown plan has %d contracts, want one", len(prepared.Plan.Contracts))
				}
				plannedChecks := map[string]bool{}
				for _, check := range prepared.Plan.Contracts[0].VerificationChecks {
					plannedChecks[check.Check] = true
				}
				for _, name := range projection.checkNames {
					if !plannedChecks[name] {
						t.Errorf("parent Markdown plan omitted required check %q", name)
					}
				}
			} else if prepared.Plan != nil || !hasOperatingModelEscalation(prepared.Escalations, "projection.candidate-required") {
				t.Errorf(".NET leaf Prepare must await a request-bound candidate: plan=%t escalations=%#v", prepared.Plan != nil, prepared.Escalations)
			}
		})
	}
}

func hasOperatingModelEscalation(escalations []host.CanonicalProjectionEscalation, code string) bool {
	for _, escalation := range escalations {
		if escalation.Code == code {
			return true
		}
	}
	return false
}
