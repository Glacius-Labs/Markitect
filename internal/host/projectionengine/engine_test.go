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
