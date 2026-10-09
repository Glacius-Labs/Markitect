package markdownreference

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

func sampleInput() Input {
	const api = "workflow.example.org/v1"
	schema := core.Schema{
		APIVersion: api,
		Purpose:    "Project-owned workflow meaning.",
		Source:     core.Source{Path: "canonical/schema.yaml", Digest: "schema-digest", Line: 1},
		Kinds: map[string]core.Kind{
			"Rule": {Purpose: "A required policy statement.", Properties: map[string]core.Property{
				"statement": {Purpose: "The rule in human-readable form.", Type: core.TypeString, MinCount: 1, MaxCount: 1},
			}},
			"Process": {Purpose: "An ordered workflow.", Properties: map[string]core.Property{
				"steps": {Purpose: "The ordered actions.", Type: core.TypeString, MinCount: 1, MaxCount: core.Unbounded},
			}},
		},
	}
	rule := core.Definition{
		APIVersion: api, Kind: "Rule", Metadata: core.Metadata{Namespace: "example", Name: "retain-source-authority"},
		Purpose: "Keep canonical source authoritative.", Spec: map[string]any{"statement": "Review the canonical source."},
		Source: core.Source{Path: "canonical/rule.yaml", Digest: "rule-digest", Line: 1},
	}
	process := core.Definition{
		APIVersion: api, Kind: "Process", Metadata: core.Metadata{Namespace: "example", Name: "review-change"},
		Purpose: "Review one proposed change.", Spec: map[string]any{"steps": []any{"inspect", "review", "record"}},
		Source: core.Source{Path: "canonical/process.yaml", Digest: "process-digest", Line: 1},
	}
	policy := func(kind, name, guidance string) core.Definition {
		return core.Definition{
			APIVersion: "markitect.foundation/v1", Kind: "ProjectionPolicy", Metadata: core.Metadata{Namespace: "example", Name: name},
			Purpose: "Bounded presentation guidance.", Spec: map[string]any{
				"sourceKind": map[string]any{"apiVersion": api, "kind": kind}, "targetTechnology": "markdown", "guidance": guidance,
			},
		}
	}
	defs := []core.Definition{rule, process}
	return Input{
		Schemas:     []core.Schema{schema},
		Selected:    []core.DefinitionIdentity{rule.Identity(), process.Identity()},
		Definitions: defs,
		Policies:    []core.Definition{policy("Rule", "rule-to-markdown", "Keep the statement concise."), policy("Process", "process-to-markdown", "Preserve the order of steps.")},
		Target:      TargetConfig{Technology: Target, Prefix: "docs/represented", AllowedRoots: []string{"docs"}},
		Check:       Check{ScopeComplete: true, InventoryComplete: true, RequestDigest: "sha256:" + strings.Repeat("a", 64)},
	}
}

func TestProposeProducesDeterministicIndependentReferenceBundle(t *testing.T) {
	input := sampleInput()
	first := Propose(input)
	reversed := input
	reversed.Schemas = append([]core.Schema(nil), input.Schemas...)
	reversed.Definitions = []core.Definition{input.Definitions[1], input.Definitions[0]}
	reversed.Selected = []core.DefinitionIdentity{input.Selected[1], input.Selected[0]}
	reversed.Policies = []core.Definition{input.Policies[1], input.Policies[0]}
	second := Propose(reversed)
	if first.Decision != DecisionWork || second.Decision != DecisionWork || len(first.Files) != 1 || len(second.Files) != 1 {
		t.Fatalf("unexpected proposals: first=%#v second=%#v", first, second)
	}
	path := "docs/represented/canonical-projection.md"
	one, ok := first.Files[path]
	if RegularFile != "100644" {
		t.Fatalf("Module mode contract changed: %q", RegularFile)
	}
	if !ok || one.Mode != RegularFile {
		t.Fatalf("Module did not own its canonical regular file: %#v", first.Files)
	}
	if !bytes.Equal(one.Bytes, second.Files[path].Bytes) {
		t.Fatal("reference bundle changed with input ordering")
	}
	text := string(one.Bytes)
	for _, marker := range []string{"# Canonical Reference Bundle", "## Schema contracts", "## Selected canonical Definitions", "## Selected ProjectionPolicies", "markitect-reference:definition:"} {
		if !strings.Contains(text, marker) {
			t.Errorf("bundle is missing %q", marker)
		}
	}
	if strings.Contains(text, "# Canonical projection\n") {
		t.Fatal("reference bundle retained the existing prose renderer layout")
	}
	independentlyCheckJSONBlocks(t, one.Bytes, input)
}

func TestRenderProducesCandidateWithoutClaimingInventoryOrPriorState(t *testing.T) {
	input := sampleInput()
	input.Check = Check{}
	input.Prior = nil
	input.Observed = nil
	got := Render(input)
	if len(got.Issues) != 0 || len(got.Files) != 1 {
		t.Fatalf("pure render required lifecycle facts: %#v", got)
	}
	if got.Files["docs/represented/canonical-projection.md"].Mode != RegularFile {
		t.Fatalf("pure render omitted output mode: %#v", got.Files)
	}
}

// independentlyCheckJSONBlocks checks only the public JSON payloads and their
// resource identities; it does not use renderer internals or provider output.
func independentlyCheckJSONBlocks(t *testing.T, markdown []byte, input Input) {
	t.Helper()
	blocks := strings.Split(string(markdown), "~~~json\n")
	if len(blocks) < 2 {
		t.Fatal("reference bundle has no JSON payload blocks")
	}
	decodedSchemas := map[string]core.Schema{}
	decodedDefinitions := map[string]core.Definition{}
	decodedPolicies := map[string]core.Definition{}
	for _, tail := range blocks[1:] {
		jsonText, _, found := strings.Cut(tail, "\n~~~")
		if !found {
			t.Fatal("unterminated JSON payload")
		}
		var resource struct {
			APIVersion string        `json:"apiVersion"`
			Kind       string        `json:"kind"`
			Metadata   core.Metadata `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(jsonText), &resource); err != nil {
			t.Fatalf("invalid JSON payload: %v", err)
		}
		if resource.Kind == "" {
			var schema core.Schema
			if err := json.Unmarshal([]byte(jsonText), &schema); err != nil {
				t.Fatal(err)
			}
			decodedSchemas[schema.APIVersion] = schema
			continue
		}
		var definition core.Definition
		if err := json.Unmarshal([]byte(jsonText), &definition); err != nil {
			t.Fatal(err)
		}
		if definition.Kind == "ProjectionPolicy" {
			decodedPolicies[definition.Identity().Key()] = definition
		} else {
			decodedDefinitions[definition.Identity().Key()] = definition
		}
	}
	if len(decodedSchemas) != len(input.Schemas) || len(decodedDefinitions) != len(input.Definitions) || len(decodedPolicies) != len(input.Policies) {
		t.Fatalf("reference bundle payload counts differ from selected public inputs: schemas=%d definitions=%d policies=%d", len(decodedSchemas), len(decodedDefinitions), len(decodedPolicies))
	}
	for _, schema := range input.Schemas {
		if !reflect.DeepEqual(decodedSchemas[schema.APIVersion], schema) {
			t.Errorf("Schema %s payload changed canonical fields", schema.APIVersion)
		}
	}
	for _, definition := range input.Definitions {
		if !reflect.DeepEqual(decodedDefinitions[definition.Identity().Key()], definition) {
			t.Errorf("Definition %s payload changed canonical fields", definition.Identity().Key())
		}
	}
	for _, policy := range input.Policies {
		if !reflect.DeepEqual(decodedPolicies[policy.Identity().Key()], policy) {
			t.Errorf("ProjectionPolicy %s payload changed canonical fields", policy.Identity().Key())
		}
	}
}

func TestProposeRefusesUnsupportedTargetInvalidOwnershipAndIncompleteScope(t *testing.T) {
	for name, mutate := range map[string]func(*Input){
		"non-markdown target":       func(in *Input) { in.Target.Technology = "codex" },
		"path outside allowed root": func(in *Input) { in.Target.Prefix = "outside/docs" },
		"traversal target":          func(in *Input) { in.Target.Prefix = "docs/../outside" },
		"missing selected identity": func(in *Input) { in.Selected = in.Selected[:1] },
		"incomplete selection":      func(in *Input) { in.Check.ScopeComplete = false },
	} {
		t.Run(name, func(t *testing.T) {
			input := sampleInput()
			mutate(&input)
			got := Propose(input)
			if got.Decision != DecisionEscalate || len(got.Files) != 0 || len(got.Issues) == 0 {
				t.Fatalf("invalid input was not refused: %#v", got)
			}
		})
	}
}

func TestProposeAcceptsTrailingSlashOnExplicitCapabilityRoot(t *testing.T) {
	input := sampleInput()
	input.Target.Prefix = "docs/represented/"
	input.Target.AllowedRoots = []string{"docs/represented/"}
	got := Propose(input)
	if got.Decision != DecisionWork || len(got.Files) != 1 {
		t.Fatalf("canonical target root spelling was refused: %#v", got)
	}
	if _, ok := got.Files["docs/represented/canonical-projection.md"]; !ok {
		t.Fatalf("trailing slash changed the Module-owned path: %#v", got.Files)
	}
}

func TestProposeRequiresPriorOwnershipForObservedOutputAndEscalatesPrefixCollisions(t *testing.T) {
	input := sampleInput()
	input.Observed = []ArtifactObservation{{Path: "docs/represented/canonical-projection.md", Bytes: []byte("unowned"), Mode: RegularFile}}
	got := Propose(input)
	if got.Decision != DecisionEscalate || got.Issues[0].Code != "reference.owner.unknown" {
		t.Fatalf("existing unowned output was not escalated: %#v", got)
	}
	input = sampleInput()
	input.Prior = &PriorRecord{Complete: true, RequestDigest: input.Check.RequestDigest}
	input.Observed = []ArtifactObservation{{Path: "docs/represented/other.md", Bytes: []byte("other"), Mode: RegularFile}}
	got = Propose(input)
	if got.Decision != DecisionEscalate || got.Issues[0].Code != "reference.target.ambiguous" {
		t.Fatalf("unowned prefix collision was not escalated: %#v", got)
	}
}

func TestProposeNoopRequiresCurrentIndependentVerification(t *testing.T) {
	input := sampleInput()
	initial := Propose(input)
	path := "docs/represented/canonical-projection.md"
	artifact := initial.Files[path]
	binding := ArtifactBinding{Path: path, Digest: digest(artifact.Bytes), Mode: RegularFile}
	input.Prior = &PriorRecord{Complete: true, RequestDigest: input.Check.RequestDigest, Artifacts: []ArtifactBinding{binding}}
	input.Observed = []ArtifactObservation{{Path: path, Bytes: artifact.Bytes, Mode: RegularFile}}
	got := Propose(input)
	if got.Decision != DecisionNoop || len(got.Files) != 0 || len(got.Reasons) != 1 || got.Reasons[0] != "evidence-refresh-required" {
		t.Fatalf("missing verification did not request refresh: %#v", got)
	}
	input.Verified = &Verification{Passed: true, RequestDigest: input.Check.RequestDigest, Artifacts: []ArtifactBinding{binding}}
	got = Propose(input)
	if got.Decision != DecisionNoop || got.Reasons[0] != "representation-current" {
		t.Fatalf("current independent verification did not establish no-op: %#v", got)
	}
}
