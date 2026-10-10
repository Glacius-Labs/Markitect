package host

import (
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/consumers/projections"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

func TestMapProjectionInclusionsSortsAndMatchesOnlyExplicitSources(t *testing.T) {
	config := projections.Config{Contracts: []projections.Contract{
		{ID: "z-contract", Sources: []string{"source/z", "source/a"}, Representation: "readme", Targets: []projections.TargetPath{{Path: "z/output.txt"}, {Path: "a/output.txt"}}, Materializer: projections.Materializer{Name: "agent", Version: "2", Mode: projections.ModeAI}, VerificationChecks: []string{"z-check", "a-check"}, Freedom: []string{"layout", "wording"}},
		{ID: "unrelated", Sources: []string{"source/unrelated"}, Targets: []projections.TargetPath{{Path: "unrelated/out"}}},
		{ID: "a-contract", Sources: []string{"source/a"}, Targets: []projections.TargetPath{{Path: "a/out"}}, Materializer: projections.Materializer{Name: "renderer", Version: "1", Mode: projections.ModeDeterministic}},
	}}
	want := []ProjectionInclusion{
		{ConfigPath: "contracts.yaml", ConfigDigest: "sha256:test", ContractID: "a-contract", Sources: []string{"source/a"}, MatchingSourceKeys: []string{"source/a"}, Targets: []string{"a/out"}, Materializer: projections.Materializer{Name: "renderer", Version: "1", Mode: projections.ModeDeterministic}},
		{ConfigPath: "contracts.yaml", ConfigDigest: "sha256:test", ContractID: "z-contract", Sources: []string{"source/a", "source/z"}, MatchingSourceKeys: []string{"source/a", "source/z"}, Representation: "readme", Targets: []string{"a/output.txt", "z/output.txt"}, Materializer: projections.Materializer{Name: "agent", Version: "2", Mode: projections.ModeAI}, Freedom: []string{"layout", "wording"}, VerificationChecks: []string{"a-check", "z-check"}},
	}
	for _, keys := range [][]string{{"source/z", "source/a", "source/z"}, {"source/a", "source/z"}} {
		got := mapProjectionInclusions(config, keys, "contracts.yaml", "sha256:test")
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("projection inclusion mapping = %#v, want %#v", got, want)
		}
	}
	if got := mapProjectionInclusions(config, []string{"a/output.txt"}, "contracts.yaml", "sha256:test"); len(got) != 0 {
		t.Fatalf("output path was inferred as a semantic source: %#v", got)
	}
}

func TestProjectionContextAndImpactUseTheirExplicitKeySets(t *testing.T) {
	config := projections.Config{Contracts: []projections.Contract{
		{ID: "context-contract", Sources: []string{"context/key"}, Targets: []projections.TargetPath{{Path: "context.txt"}}},
		{ID: "impact-contract", Sources: []string{"impact/key"}, Targets: []projections.TargetPath{{Path: "impact.txt"}}},
	}}
	context := mapProjectionInclusions(config, []string{"context/key"}, "contracts.yaml", "sha256:test")
	impact := mapProjectionInclusions(config, []string{"impact/key"}, "contracts.yaml", "sha256:test")
	if len(context) != 1 || context[0].ContractID != "context-contract" {
		t.Fatalf("context mapping = %#v", context)
	}
	if len(impact) != 1 || impact[0].ContractID != "impact-contract" {
		t.Fatalf("impact mapping = %#v", impact)
	}
}

func TestProjectionContextCanReadPolicyFailingSource(t *testing.T) {
	_, fixture := representationFixture(t)
	files := make(map[string][]byte, len(fixture.Snapshot.Files))
	for name, content := range fixture.Snapshot.Files {
		files[name] = append([]byte(nil), content...)
	}
	files["projections.yaml"] = append([]byte(nil), files["projections.config"]...)
	files["markitect.yaml"] = append(files["markitect.yaml"], []byte("  adapters:\n    - name: projections\n      type: local-projection\n      version: v1alpha1\n      config:\n        contracts: projections.yaml\n        coverage: markitect-artifacts.yaml\n")...)
	failed, err := Parse(&snapshot.Snapshot{Provisional: true, Files: files})
	if err != nil {
		t.Fatal(err)
	}
	// Diagnostic analysis must not call compileAdapterModel/strict validation.
	failed.Diagnostics = []core.Diagnostic{{Code: "policy.failed", Message: "declared constraint failed"}}

	got, err := ProjectionContext(failed, []string{"proof/Rule/desired"})
	if err != nil {
		t.Fatalf("policy-failing source was blocked from projection analysis: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("projection context = %#v", got)
	}
	if got[0].ConfigPath != "projections.yaml" || got[0].ConfigDigest != hashBytes(files["projections.yaml"]) {
		t.Fatalf("projection config provenance = %#v", got[0])
	}
	config, err := projections.ParseConfig(files["projections.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	config.Contracts[0].Sources = append(config.Contracts[0].Sources, "proof/Rule/unknown")
	malformed, err := YAML(config)
	if err != nil {
		t.Fatal(err)
	}
	failed.Snapshot.Files["projections.yaml"] = malformed
	if _, err := ProjectionContext(failed, []string{"proof/Rule/desired"}); err == nil {
		t.Fatal("projection contract with an unknown semantic source was accepted")
	}
}

func TestProjectionImpactPropagatesExplicitContractPrerequisitesOnly(t *testing.T) {
	config := projections.Config{Contracts: []projections.Contract{
		{ID: "producer", Sources: []string{"source/changed"}, Targets: []projections.TargetPath{{Path: "producer.txt"}}},
		{ID: "consumer", Sources: []string{"source/other"}, DependsOn: []string{"producer"}, Targets: []projections.TargetPath{{Path: "consumer.txt"}}},
		{ID: "integration", Sources: []string{"source/third"}, DependsOn: []string{"consumer"}, Targets: []projections.TargetPath{{Path: "integration.txt"}}},
		{ID: "unrelated", Sources: []string{"source/unrelated"}, Targets: []projections.TargetPath{{Path: "unrelated.txt"}}},
	}}
	context := mapProjectionInclusions(config, []string{"source/changed"}, "config", "digest")
	if len(context) != 1 || context[0].ContractID != "producer" {
		t.Fatalf("prerequisites became implicit context edges: %#v", context)
	}
	impact := mapProjectionInclusions(config, []string{"source/changed"}, "config", "digest", projectionInclusionOptions{IncludeDependents: true})
	if len(impact) != 3 {
		t.Fatalf("contract review fanout incomplete: %#v", impact)
	}
	for _, inclusion := range impact {
		switch inclusion.ContractID {
		case "consumer":
			if !reflect.DeepEqual(inclusion.ViaContracts, []string{"producer"}) || len(inclusion.MatchingSourceKeys) != 0 {
				t.Fatalf("consumer cause invented a source match: %#v", inclusion)
			}
		case "integration":
			if !reflect.DeepEqual(inclusion.ViaContracts, []string{"consumer"}) {
				t.Fatalf("transitive prerequisite cause missing: %#v", inclusion)
			}
		case "producer":
			if len(inclusion.ViaContracts) != 0 {
				t.Fatalf("producer cause wrong: %#v", inclusion)
			}
		default:
			t.Fatalf("unrelated contract included: %#v", inclusion)
		}
	}
}

func TestProjectionImpactShowsChangedMappingAndCoverageWithoutInventedSources(t *testing.T) {
	files := registeredRepresentationFiles(t)
	p, err := Parse(&snapshot.Snapshot{Provisional: true, Files: files})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"projections.config", "markitect-artifacts.yaml", "markitect.yaml"} {
		got, err := ProjectionImpact(p, nil, []string{path})
		if err != nil || len(got) != 2 {
			t.Fatalf("changed registered input %s lost contract fanout: %v %#v", path, err, got)
		}
		for _, inclusion := range got {
			if len(inclusion.MatchingSourceKeys) != 0 || !reflect.DeepEqual(inclusion.ViaInputs, []string{path}) {
				t.Fatalf("config cause invented semantic match: %#v", inclusion)
			}
		}
	}
	if got, err := ProjectionImpact(p, nil, []string{"unrelated.txt"}); err != nil || len(got) != 0 {
		t.Fatalf("unrelated input broadened explicit contract review: %v %#v", err, got)
	}
}
