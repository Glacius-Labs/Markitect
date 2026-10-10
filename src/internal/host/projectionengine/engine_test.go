package projectionengine

import "testing"

func TestBuildAcceptsTupleIdentityAsExactSource(t *testing.T) {
	key := `["ddd.example.org/v1","Aggregate","sales","Order"]`
	input := Input{
		ModelDigest: "model", SnapshotDigest: "snapshot",
		Config: Config{APIVersion: ConfigAPIVersion, Version: ConfigVersion, Contracts: []Contract{{
			ID: "order-source", Sources: []string{key}, Representation: "go",
			Materializer: Materializer{Name: "test", Version: "1", Mode: ModeDeterministic},
			Targets:      []TargetPath{{Path: "src/order.go"}},
		}}},
		ConfigDigest: "config", IntentDigest: "intent", ToolName: "tool", ToolVersion: "1", ToolDigest: "tool-digest",
		Sources:    []SourceProvenance{{Key: key, Kind: "Aggregate", Source: SourceInfo{Path: "architecture/order.yaml", Digest: "source-digest"}}},
		Files:      map[string][]byte{"src/order.go": []byte("same")},
		Desired:    map[string][]byte{"src/order.go": []byte("same")},
		Governance: Governance{Status: "passed"},
	}
	plan, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusConverged || plan.Contracts[0].Sources[0].Key != key {
		t.Fatalf("tuple identity did not bind exactly: %+v", plan)
	}
}

func TestBuildBlocksFromExplicitGovernanceFailure(t *testing.T) {
	key := `["api/v1","Kind","","item"]`
	input := Input{
		ModelDigest: "model", SnapshotDigest: "snapshot",
		Config:       Config{APIVersion: ConfigAPIVersion, Version: ConfigVersion, Contracts: []Contract{{ID: "one", Sources: []string{key}, Representation: "text", Materializer: Materializer{Name: "test", Version: "1", Mode: ModeDeterministic}, Targets: []TargetPath{{Path: "out.txt"}}}}},
		ConfigDigest: "config", IntentDigest: "intent", ToolName: "tool", ToolVersion: "1", ToolDigest: "tool-digest",
		Sources: []SourceProvenance{{Key: key, Kind: "Kind"}}, Files: map[string][]byte{"out.txt": []byte("x")}, Desired: map[string][]byte{"out.txt": []byte("x")},
		Governance: Governance{Status: "failed"},
	}
	plan, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked {
		t.Fatalf("failed host governance did not block: %+v", plan)
	}
}

func TestBuildTreatsArtifactModeAsObservedAndDesiredProjectionState(t *testing.T) {
	key := `["api/v1","Kind","","item"]`
	input := Input{
		ModelDigest: "model", SnapshotDigest: "snapshot",
		Config: Config{APIVersion: ConfigAPIVersion, Version: ConfigVersion, Contracts: []Contract{{
			ID: "hook", Sources: []string{key}, Representation: "git-hook",
			Materializer: Materializer{Name: "test", Version: "1", Mode: ModeDeterministic},
			Targets:      []TargetPath{{Path: "hooks/pre-commit"}},
		}}},
		ConfigDigest: "config", IntentDigest: "intent", ToolName: "tool", ToolVersion: "1", ToolDigest: "tool-digest",
		Sources:      []SourceProvenance{{Key: key, Kind: "Kind"}},
		Files:        map[string][]byte{"hooks/pre-commit": []byte("#!/bin/sh\n")},
		FileModes:    map[string]string{"hooks/pre-commit": "100644"},
		Desired:      map[string][]byte{"hooks/pre-commit": []byte("#!/bin/sh\n")},
		DesiredModes: map[string]string{"hooks/pre-commit": "100755"},
		Governance:   Governance{Status: GovernancePassed},
	}
	plan, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	target := plan.Contracts[0].Targets[0]
	if target.Status != TargetDrifted || target.ObservedMode != "100644" || target.DesiredMode != "100755" {
		t.Fatalf("mode-only drift was not represented: %+v", target)
	}
	regular := input
	regular.DesiredModes = map[string]string{"hooks/pre-commit": "100644"}
	regularPlan, err := Build(regular)
	if err != nil {
		t.Fatal(err)
	}
	if regularPlan.Contracts[0].Targets[0].Status != TargetMatched || regularPlan.PlanDigest == plan.PlanDigest {
		t.Fatalf("desired mode was not bound by plan: regular=%+v executable=%+v", regularPlan.Contracts[0].Targets[0], target)
	}
}

func TestBuildRejectsArtifactModesOutsideExactFilesAndTargets(t *testing.T) {
	key := `["api/v1","Kind","","item"]`
	input := Input{
		ModelDigest: "model", SnapshotDigest: "snapshot",
		Config:       Config{APIVersion: ConfigAPIVersion, Version: ConfigVersion, Contracts: []Contract{{ID: "one", Sources: []string{key}, Representation: "text", Materializer: Materializer{Name: "test", Version: "1", Mode: ModeDeterministic}, Targets: []TargetPath{{Path: "out.txt"}}}}},
		ConfigDigest: "config", IntentDigest: "intent", ToolName: "tool", ToolVersion: "1", ToolDigest: "tool-digest",
		Sources: []SourceProvenance{{Key: key, Kind: "Kind"}}, Files: map[string][]byte{"out.txt": []byte("x")}, Desired: map[string][]byte{"out.txt": []byte("x")},
		Governance: Governance{Status: GovernancePassed},
	}
	for _, tc := range []struct {
		name string
		edit func(*Input)
	}{
		{"unsupported observed mode", func(in *Input) { in.FileModes = map[string]string{"out.txt": "100664"} }},
		{"mode for absent observed file", func(in *Input) { in.FileModes = map[string]string{"missing": "100755"} }},
		{"desired mode outside target", func(in *Input) { in.DesiredModes = map[string]string{"other": "100755"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copy := input
			tc.edit(&copy)
			if _, err := Build(copy); err == nil {
				t.Fatal("invalid mode binding was accepted")
			}
		})
	}
}
