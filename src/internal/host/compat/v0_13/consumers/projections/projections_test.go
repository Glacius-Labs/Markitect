package projections

import (
	"strings"
	"testing"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

func testModel() core.SemanticModel {
	return core.SemanticModel{
		APIVersion:  core.SemanticModelVersion,
		Snapshot:    core.ModelSnapshot{ID: "snapshot-1", Digest: "snapshot-digest"},
		ModelDigest: "model-digest", StructuralStatus: "passed", ValidationStatus: "passed", PolicyStatus: "passed",
		Resources: []core.ModelResource{{
			Identity: core.ModelIdentity{Kind: "UseCase", Name: "Create", Key: "example/UseCase/Create"},
			Source:   core.ModelSource{Path: "architecture/resources.yaml", Line: 8, Digest: "resource-digest"},
		}},
		DomainInputs: []core.ModelDomainInput{{APIVersion: "example.org/domain/v1", Name: "architecture", Path: "architecture/Domain.yaml", Digest: "domain-digest"}},
	}
}

func config(mode string, checks ...string) Config {
	return Config{APIVersion: ConfigAPIVersion, Version: ConfigVersion, Contracts: []Contract{{
		ID: "readme", Sources: []string{"example/UseCase/Create"}, Representation: "agent-context",
		Materializer: Materializer{Name: "markdown", Version: "1", Mode: mode},
		Targets:      []TargetPath{{Path: "generated/README.md"}}, VerificationChecks: checks,
	}}}
}

func inputFor(cfg Config) Input {
	return Input{Model: testModel(), Config: cfg, ConfigDigest: "config-digest", IntentDigest: "intent-digest",
		ToolName: "markitect", ToolVersion: "1.0.0", ToolDigest: "tool-digest",
		Files:   map[string][]byte{"generated/README.md": []byte("same")},
		Desired: map[string][]byte{"generated/README.md": []byte("same")},
	}
}

func TestDeterministicProjectionConvergesAndExplainsSource(t *testing.T) {
	input := inputFor(config(ModeDeterministic))
	plan, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusConverged || plan.Contracts[0].Status != StatusConverged || plan.Contracts[0].Targets[0].Status != TargetMatched {
		t.Fatalf("unexpected plan state: %#v", plan)
	}
	if len(plan.Contracts[0].Sources) != 1 || plan.Contracts[0].Sources[0].Source.Path != "architecture/resources.yaml" {
		t.Fatalf("source provenance missing: %#v", plan.Contracts[0].Sources)
	}
	first := plan.PlanDigest
	planAgain, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if first != planAgain.PlanDigest {
		t.Fatalf("same fixed input produced different digest: %s != %s", first, planAgain.PlanDigest)
	}

	encoded, err := yaml.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "apiVersion:") || !strings.Contains(string(encoded), "planDigest:") || !strings.Contains(string(encoded), "policyResults:") {
		t.Fatalf("YAML consumer shape lost lower camel-case fields:\n%s", encoded)
	}
}

func TestFailedPolicyBlocksWithoutDroppingResult(t *testing.T) {
	input := inputFor(config(ModeDeterministic))
	input.Model.PolicyStatus = core.PolicyFailed
	input.Model.PolicyResults = []core.PolicyResult{{APIVersion: "example.org/domain/v1", Constraint: "commands-need-validator", Subject: "example/UseCase/Create", Status: core.PolicyFailed, ConstraintDigest: "constraint-digest", SubjectDigest: "subject-digest", Message: "Validator is missing"}}
	plan, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || len(plan.PolicyResults) != 1 || plan.PolicyResults[0].Message != "Validator is missing" {
		t.Fatalf("failed policy was not preserved and blocked: %#v", plan)
	}
}

func TestAIProjectionRequiresFreshBoundPassingEvidence(t *testing.T) {
	input := inputFor(config(ModeAI, "architecture-test"))
	input.Desired = nil
	plan, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusIncomplete || plan.Contracts[0].Targets[0].Status != TargetIncomplete {
		t.Fatalf("candidate presence should remain incomplete without evidence: %#v", plan.Contracts[0])
	}

	contractDigest := plan.Contracts[0].ContractDigest
	input.Evidence = []CheckEvidence{{ContractID: "readme", Check: "architecture-test", Status: EvidencePassed,
		SnapshotDigest: input.Model.Snapshot.Digest, IntentDigest: input.IntentDigest, ContractDigest: contractDigest,
		TargetDigests: map[string]string{"generated/README.md": plan.Contracts[0].Targets[0].ObservedDigest},
	}}
	plan, err = Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusConverged || plan.Contracts[0].Targets[0].Status != TargetVerified || plan.Contracts[0].VerificationChecks[0].Status != EvidencePassed {
		t.Fatalf("fresh passing evidence did not verify candidate: %#v", plan.Contracts[0])
	}

	input.Evidence[0].SnapshotDigest = "stale-snapshot"
	plan, err = Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusIncomplete || plan.Contracts[0].Targets[0].Status != TargetIncomplete || plan.Contracts[0].VerificationChecks[0].Status != EvidenceIncomplete {
		t.Fatalf("stale evidence was accepted: %#v", plan.Contracts[0])
	}
}

func TestStructuralAndPathFailuresAreErrors(t *testing.T) {
	input := inputFor(config(ModeDeterministic))
	input.Model.StructuralStatus = "failed"
	if _, err := Build(input); err == nil {
		t.Fatal("structurally invalid model must be rejected")
	}

	for _, target := range []string{"../outside.md", "CON.md", ".git/config", "Generated/README.md"} {
		t.Run(target, func(t *testing.T) {
			bad := inputFor(config(ModeDeterministic))
			bad.Config.Contracts[0].Targets[0].Path = target
			if target == "Generated/README.md" {
				bad.Files = map[string][]byte{"generated/README.md": []byte("same"), target: []byte("same")}
				bad.Desired = map[string][]byte{target: []byte("same")}
			}
			if _, err := Build(bad); err == nil {
				t.Fatalf("unsafe or aliased target %q must be rejected", target)
			}
		})
	}
}

func TestProtectedOwnerDuplicatesAreAllowedButTargetsCannotOverlap(t *testing.T) {
	input := inputFor(config(ModeDeterministic))
	input.ProtectedPaths = []string{"architecture", "architecture/resources.yaml", "architecture/resources.yaml"}
	if _, err := Build(input); err != nil {
		t.Fatalf("overlapping protected owner facts should be allowed: %v", err)
	}
	input.Config.Contracts[0].Targets[0].Path = "architecture/generated.md"
	input.Desired = map[string][]byte{"architecture/generated.md": []byte("x")}
	if _, err := Build(input); err == nil {
		t.Fatal("target overlapping protected directory must be rejected")
	}
}

func TestConfigIsClosedAndSingleDocument(t *testing.T) {
	base := `apiVersion: markitect.example.org/projections/v1alpha1
version: "1"
contracts:
  - id: docs
    sources: [example/UseCase/Create]
    representation: docs
    materializer: {name: renderer, version: "1", mode: deterministic}
    targets: [{path: docs/out.md}]
`
	if _, err := ParseConfig([]byte(base)); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseConfig([]byte(base + "---\n{}\n")); err == nil {
		t.Fatal("multiple YAML documents should be rejected")
	}
	if _, err := ParseConfig([]byte(strings.Replace(base, "contracts:", "surprise: true\ncontracts:", 1))); err == nil {
		t.Fatal("unknown YAML fields should be rejected")
	}
}
