package host

import (
	"encoding/json"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"reflect"
	"testing"
)

func impactSource(t *testing.T, revision string, purpose string) *CanonicalSource {
	t.Helper()
	target := core.KindIdentity{APIVersion: "test.example/v1", Kind: "Node"}
	schema := core.Schema{APIVersion: target.APIVersion, Purpose: "An explicit graph", Kinds: map[string]core.Kind{"Node": {Purpose: "One graph member", Properties: map[string]core.Property{"next": {Purpose: "Canonical link", Type: core.TypeReference, MinCount: 0, MaxCount: 1, Target: &target}}}}}
	defs := []core.Definition{
		{APIVersion: target.APIVersion, Kind: target.Kind, Metadata: core.Metadata{Name: "a"}, Purpose: purpose, Spec: map[string]any{"next": map[string]any{"name": "b", "namespace": ""}}},
		{APIVersion: target.APIVersion, Kind: target.Kind, Metadata: core.Metadata{Name: "b"}, Purpose: "B", Spec: map[string]any{"next": map[string]any{"name": "a", "namespace": ""}}},
		{APIVersion: target.APIVersion, Kind: target.Kind, Metadata: core.Metadata{Name: "unrelated"}, Purpose: "Unrelated", Spec: map[string]any{}},
	}
	model, diags := core.Compile([]core.Schema{schema}, defs, revision)
	if len(diags) != 0 {
		t.Fatal(diags)
	}
	return &CanonicalSource{Model: model}
}
func TestCanonicalImpactUsesExplicitReverseEdgesAndTerminatesCycles(t *testing.T) {
	base := impactSource(t, "base", "A")
	candidate := impactSource(t, "candidate", "Changed A")
	report, err := AnalyzeCanonicalImpact(base, candidate, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := base.Model.Definitions[0].Identity().Key()
	b := base.Model.Definitions[1].Identity().Key()
	if !reflect.DeepEqual(report.ChangedDefinitions, []string{a}) || !reflect.DeepEqual(report.AffectedDefinitions, []string{a, b}) {
		t.Fatalf("impact=%#v", report)
	}
	cause := report.DefinitionCauses[b]
	if cause.Code != "referenced-definition-changed" || cause.Via == nil || cause.Via.From != b || cause.Via.To != a || cause.Via.Property != "next" {
		t.Fatalf("lost explicit cause: %#v", cause)
	}
	encoded, _ := json.Marshal(report)
	again, err := AnalyzeCanonicalImpact(base, candidate, nil)
	if err != nil {
		t.Fatal(err)
	}
	repeated, _ := json.Marshal(again)
	if string(encoded) != string(repeated) {
		t.Fatal("nondeterministic report")
	}
}
func TestCanonicalImpactReportsSourceOnlyChangeSeparatelyFromModelMeaning(t *testing.T) {
	base := impactSource(t, "same", "A")
	candidate := impactSource(t, "same", "A")
	candidate.Model.Definitions[0].Source = core.Source{Path: "defs/a.yaml", Digest: "new bytes"}
	candidate.Model, candidate.Diagnostics = core.Compile(candidate.Model.Schemas, candidate.Model.Definitions, candidate.Model.Revision)
	if base.Model.Digest != candidate.Model.Digest {
		t.Fatal("provenance unexpectedly changed semantic digest")
	}
	report, err := AnalyzeCanonicalImpact(base, candidate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.ChangedDefinitions) != 1 || len(report.AffectedDefinitions) != 2 {
		t.Fatalf("source bytes disappeared: %#v", report)
	}
}
func TestCanonicalImpactBlocksUncompiledAndStructurallyInvalidModels(t *testing.T) {
	base := impactSource(t, "base", "A")
	candidate := impactSource(t, "candidate", "A")
	candidate.Model.Digest = "tampered"
	if _, err := AnalyzeCanonicalImpact(base, candidate, nil); err == nil {
		t.Fatal("uncompiled model accepted")
	}
	candidate = impactSource(t, "candidate", "A")
	candidate.Diagnostics = []core.Diagnostic{{Code: "reference.unresolved"}}
	if _, err := AnalyzeCanonicalImpact(base, candidate, nil); err == nil {
		t.Fatal("structural diagnostics accepted")
	}
}

func TestCanonicalImpactRejectsTamperedResolvedEdges(t *testing.T) {
	base := impactSource(t, "base", "A")
	candidate := impactSource(t, "candidate", "A")
	candidate.Model.Edges = nil
	if _, err := AnalyzeCanonicalImpact(base, candidate, nil); err == nil {
		t.Fatal("tampered normalized edges accepted")
	}
}
