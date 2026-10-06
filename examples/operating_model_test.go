package examples

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const operatingModelConfigPath = "examples/operating-model/canonical.yaml"

func TestOperatingModelProjectionScopesAreIndependentAndComposed(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the operating-model example test file")
	}
	root := filepath.Dir(filepath.Dir(testFile))

	fixed, err := host.LoadCanonicalSource(root, "", operatingModelConfigPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixed.Diagnostics) != 0 {
		t.Fatalf("operating-model canonical source has structural diagnostics: %#v", fixed.Diagnostics)
	}
	if len(fixed.Pins) != 4 || len(fixed.Activation.Projectors) != 2 {
		t.Fatalf("expected four exact Module pins and two activated Projection Modules; pins=%d projectors=%d", len(fixed.Pins), len(fixed.Activation.Projectors))
	}
	if len(fixed.Model.Edges) != 7 {
		t.Fatalf("expected two ProductComposition child edges and five Projection-to-policy edges, got %d", len(fixed.Model.Edges))
	}

	observed, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	selected := []struct {
		name        string
		target      string
		definition  map[string]bool
		policyCount int
	}{
		{
			name:        "orders-dotnet",
			target:      "src/Orders",
			definition:  map[string]bool{"[\"commerce.operating-model/v1\",\"OrderRules\",\"orders\",\"accepted-order-total\"]": true},
			policyCount: 1,
		},
		{
			name:        "billing-dotnet",
			target:      "src/Billing",
			definition:  map[string]bool{"[\"commerce.operating-model/v1\",\"BillingQuery\",\"billing\",\"outstanding-order-total\"]": true},
			policyCount: 1,
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
			prepared, err := host.PrepareCanonicalProjection(
				fixed,
				observed,
				identity,
				"operating-model-shape-test/1",
				host.Hash([]byte("operating-model-shape-test")),
				nil,
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
		})
	}
}
