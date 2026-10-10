package examples

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host/canonical"
)

var capabilitySchemaPaths = []string{
	"examples/capability-ontologies/software-architecture/schema.yaml",
	"examples/capability-ontologies/delivery/schema.yaml",
	"examples/capability-ontologies/workflow-responsibility/schema.yaml",
	"examples/capability-ontologies/trading-card-combat/schema.yaml",
	"examples/capability-ontologies/filing-review/schema.yaml",
}

var capabilityDefinitionPaths = []string{
	"examples/capability-ontologies/software-architecture/components.yaml",
	"examples/capability-ontologies/software-architecture/dependency.yaml",
	"examples/capability-ontologies/software-architecture/storage.yaml",
	"examples/capability-ontologies/delivery/deployment.yaml",
	"examples/capability-ontologies/delivery/rollout.yaml",
	"examples/capability-ontologies/workflow-responsibility/owner.yaml",
	"examples/capability-ontologies/workflow-responsibility/approver.yaml",
	"examples/capability-ontologies/workflow-responsibility/work-item.yaml",
	"examples/capability-ontologies/trading-card-combat/card.yaml",
	"examples/capability-ontologies/trading-card-combat/action.yaml",
	"examples/capability-ontologies/filing-review/filing.yaml",
	"examples/capability-ontologies/filing-review/review.yaml",
}

type capabilityMutation func(path string, data []byte) []byte

func decodeAndCompileCapabilityFixtures(t *testing.T, mutate capabilityMutation) ([]core.Schema, []core.Definition, core.Model, []core.Diagnostic, []string) {
	t.Helper()
	schemas := make([]core.Schema, 0, len(capabilitySchemaPaths))
	definitions := make([]core.Definition, 0, len(capabilityDefinitionPaths))
	var manifest []string
	for _, path := range capabilitySchemaPaths {
		data, err := os.ReadFile(filepath.Join(harnessRepositoryRoot(t), filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if mutate != nil {
			data = mutate(path, bytes.Clone(data))
		}
		sum := sha256.Sum256(data)
		manifest = append(manifest, fmt.Sprintf("%s sha256:%x", path, sum))
		schema, err := canonical.DecodeSchema(path, data)
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		schemas = append(schemas, schema)
	}
	for _, path := range capabilityDefinitionPaths {
		data, err := os.ReadFile(filepath.Join(harnessRepositoryRoot(t), filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if mutate != nil {
			data = mutate(path, bytes.Clone(data))
		}
		sum := sha256.Sum256(data)
		manifest = append(manifest, fmt.Sprintf("%s sha256:%x", path, sum))
		definition, err := canonical.DecodeDefinition(path, data)
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		definitions = append(definitions, definition)
	}
	sort.Strings(manifest)
	model, diagnostics := core.Compile(schemas, definitions, "c1-fixture-v1")
	return schemas, definitions, model, diagnostics, manifest
}

func TestC1FiveDistinctTypedOntologiesCompileThroughCanonicalPipeline(t *testing.T) {
	schemas, definitions, model, diagnostics, manifest := decodeAndCompileCapabilityFixtures(t, nil)
	for _, entry := range manifest {
		t.Logf("C1_INPUT %s", entry)
	}
	if len(diagnostics) != 0 {
		encoded, _ := json.Marshal(diagnostics)
		t.Fatalf("frozen positive fixtures should compile: %s", encoded)
	}
	if len(model.Schemas) != 5 || len(model.Definitions) != 12 || len(model.Edges) != 7 {
		t.Fatalf("compiled counts schemas=%d definitions=%d edges=%d; want 5, 12, 7", len(model.Schemas), len(model.Definitions), len(model.Edges))
	}
	t.Logf("C1_COMBINED digest=%s schemas=%d definitions=%d edges=%d", model.Digest, len(model.Schemas), len(model.Definitions), len(model.Edges))

	for i := range schemas {
		api := schemas[i].APIVersion
		var selected []core.Definition
		for _, definition := range definitions {
			if definition.APIVersion == api {
				selected = append(selected, definition)
			}
		}
		perModel, perDiagnostics := core.Compile([]core.Schema{schemas[i]}, selected, "c1-fixture-v1")
		if len(perDiagnostics) != 0 {
			encoded, _ := json.Marshal(perDiagnostics)
			t.Fatalf("ontology %s should compile independently: %s", api, encoded)
		}
		wantDefinitions, wantEdges := ontologyCounts(api)
		if len(perModel.Schemas) != 1 || len(perModel.Definitions) != wantDefinitions || len(perModel.Edges) != wantEdges {
			t.Fatalf("ontology %s compiled schemas=%d definitions=%d edges=%d; want 1, %d, %d", api, len(perModel.Schemas), len(perModel.Definitions), len(perModel.Edges), wantDefinitions, wantEdges)
		}
		t.Logf("C1_ONTOLOGY apiVersion=%s digest=%s definitions=%d edges=%d", api, perModel.Digest, len(perModel.Definitions), len(perModel.Edges))
	}

	expectedPurposes := []struct {
		api, schemaPurpose, kind, kindPurpose, property, propertyPurpose, definition, definitionPurpose string
	}{
		{"architecture.example.org/v1", "Describes an application component map and dependencies.", "Component", "Identifies one owned software unit and its layer.", "name", "Names the component.", "catalog", "Publishes product catalog behavior."},
		{"delivery.example.org/v1", "Describes deployments, environments and rollout intent.", "Deployment", "Selects a service deployment and its capacity.", "service", "Names the deployed service.", "production", "Describes the production payments service."},
		{"responsibility.example.org/v1", "Records work ownership and workflow decision authority.", "Role", "Names a workflow role and its authority.", "authority", "States the role's authority.", "owner", "Owns completion of the release task."},
		{"cardgame.example.org/v1", "Describes cards and combat actions in a turn-based duel.", "Card", "Defines a playable card and printed combat values.", "cost", "States the resource cost.", "ember-fox", "Defines a low-cost creature that can attack."},
		{"filing.example.org/v1", "Tracks a submitted filing and its human review outcome.", "Filing", "Records the state of a submitted dossier.", "status", "Records the filing lifecycle state.", "dossier-17", "Records the dossier submitted by vendor A."},
	}
	for _, expected := range expectedPurposes {
		schemaFound := false
		for _, schema := range model.Schemas {
			if schema.APIVersion == expected.api {
				schemaFound = schema.Purpose == expected.schemaPurpose
			}
		}
		if !schemaFound {
			t.Errorf("Schema purpose for %s did not survive normalization", expected.api)
		}
		kind, ok := model.Kind(core.KindIdentity{APIVersion: expected.api, Kind: expected.kind})
		if !ok || kind.Purpose != expected.kindPurpose || kind.Properties[expected.property].Purpose != expected.propertyPurpose {
			t.Errorf("Kind/Property purpose for %s/%s did not survive normalization: %#v", expected.api, expected.kind, kind)
		}
		definition, ok := model.Definition(core.DefinitionIdentity{APIVersion: expected.api, Kind: expected.kind, Namespace: namespaceForAPI(expected.api), Name: expected.definition})
		if !ok || definition.Purpose != expected.definitionPurpose {
			t.Errorf("Definition purpose for %s/%s did not survive normalization: %#v", expected.api, expected.definition, definition)
		}
	}
	review, ok := model.Definition(core.DefinitionIdentity{APIVersion: "filing.example.org/v1", Kind: "Review", Namespace: "vendor-a", Name: "initial-review"})
	if !ok {
		t.Fatal("compiled model is missing the filing review Definition")
	}
	filingKind, ok := review.Spec["filingKind"].(map[string]any)
	if !ok || filingKind["apiVersion"] != "filing.example.org/v1" || filingKind["kind"] != "Filing" {
		t.Fatalf("kindReference did not survive normalization: %#v", review.Spec["filingKind"])
	}

	permutedSchemas := reverseSchemas(schemas)
	permutedDefinitions := reverseDefinitions(definitions)
	permuted, permutedDiagnostics := core.Compile(permutedSchemas, permutedDefinitions, "c1-fixture-v1")
	if len(permutedDiagnostics) != 0 {
		t.Fatalf("reordered inputs should compile: %#v", permutedDiagnostics)
	}
	if model.Digest != permuted.Digest || !equalEdges(model.Edges, permuted.Edges) {
		t.Fatalf("input order changed normalized output: digest %s / %s; edges equal %v", model.Digest, permuted.Digest, equalEdges(model.Edges, permuted.Edges))
	}
	t.Logf("C1_PERMUTED digest=%s edges=%d", permuted.Digest, len(permuted.Edges))
}

func TestC1TypedOntologyNegativeCasesAreRejected(t *testing.T) {
	cases := []struct {
		name   string
		want   string
		mutate capabilityMutation
	}{
		{"unresolved-reference", "reference.unresolved", func(path string, data []byte) []byte {
			if path == "examples/capability-ontologies/delivery/rollout.yaml" {
				return bytes.Replace(data, []byte("name: production}"), []byte("name: missing}"), 1)
			}
			return data
		}},
		{"wrong-kind-reference", "reference.target-kind", func(path string, data []byte) []byte {
			if path == "examples/capability-ontologies/delivery/rollout.yaml" {
				return bytes.Replace(data, []byte("deployment: {namespace: payments, name: production}"), []byte("deployment: {apiVersion: delivery.example.org/v1, kind: Rollout, namespace: payments, name: production}"), 1)
			}
			return data
		}},
		{"closed-unknown-property", "spec.unknown-property", func(path string, data []byte) []byte {
			if path == "examples/capability-ontologies/delivery/deployment.yaml" {
				return bytes.Replace(data,
					[]byte("spec: {service: payments-api, environment: production, replicas: 3}"),
					[]byte("spec:\n  service: payments-api\n  environment: production\n  replicas: 3\n  surprise: true"), 1)
			}
			return data
		}},
		{"cardinality", "property.cardinality", func(path string, data []byte) []byte {
			if path == "examples/capability-ontologies/delivery/schema.yaml" {
				return bytes.Replace(data, []byte("replicas: {purpose: Declares the requested instance count., type: integer, minCount: 1, maxCount: 1}"), []byte("replicas: {purpose: Declares the requested instance count., type: integer, minCount: 1, maxCount: 2}"), 1)
			}
			if path == "examples/capability-ontologies/delivery/deployment.yaml" {
				return bytes.Replace(data, []byte("replicas: 3"), []byte("replicas: [3, 4, 5]"), 1)
			}
			return data
		}},
		{"enum", "property.enum-value", func(path string, data []byte) []byte {
			if path == "examples/capability-ontologies/delivery/deployment.yaml" {
				return bytes.Replace(data, []byte("environment: production"), []byte("environment: qa"), 1)
			}
			return data
		}},
		{"unresolved-kind-reference", "kind-reference.unresolved", func(path string, data []byte) []byte {
			if path == "examples/capability-ontologies/filing-review/review.yaml" {
				return bytes.Replace(data, []byte("kind: Filing"), []byte("kind: MissingKind"), 1)
			}
			return data
		}},
		{"closed-kind-reference", "kind-reference.value", func(path string, data []byte) []byte {
			if path == "examples/capability-ontologies/filing-review/review.yaml" {
				return bytes.Replace(data, []byte("kind: Filing}"), []byte("kind: Filing, name: ignored}"), 1)
			}
			return data
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, model, diagnostics, _ := decodeAndCompileCapabilityFixtures(t, tc.mutate)
			if len(diagnostics) == 0 {
				t.Fatalf("negative fixture compiled to model %s; wanted %s", model.Digest, tc.want)
			}
			found := false
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == tc.want {
					found = true
				}
			}
			encoded, _ := json.Marshal(diagnostics)
			t.Logf("C1_NEGATIVE name=%s diagnostics=%s", tc.name, encoded)
			if !found {
				t.Fatalf("diagnostics did not contain %s: %s", tc.want, encoded)
			}
		})
	}
}

func namespaceForAPI(api string) string {
	switch api {
	case "architecture.example.org/v1":
		return "shop"
	case "delivery.example.org/v1":
		return "payments"
	case "responsibility.example.org/v1":
		return "release"
	case "cardgame.example.org/v1":
		return "starter-deck"
	case "filing.example.org/v1":
		return "vendor-a"
	default:
		return ""
	}
}

func ontologyCounts(api string) (definitions, edges int) {
	switch api {
	case "architecture.example.org/v1":
		return 3, 2
	case "delivery.example.org/v1":
		return 2, 1
	case "responsibility.example.org/v1":
		return 3, 2
	case "cardgame.example.org/v1":
		return 2, 1
	case "filing.example.org/v1":
		return 2, 1
	default:
		return 0, 0
	}
}

func reverseSchemas(values []core.Schema) []core.Schema {
	result := append([]core.Schema(nil), values...)
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return result
}

func reverseDefinitions(values []core.Definition) []core.Definition {
	result := append([]core.Definition(nil), values...)
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return result
}

func equalEdges(left, right []core.Edge) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
