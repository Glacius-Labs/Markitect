package projections

import (
	"testing"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

// A waiver may permit evidence-relative convergence, but the plan must retain
// the distinct waived state and its governance provenance rather than
// relabeling it as policy-passed.
func TestWaivedPolicyRemainsVisibleWhenProjectionConverges(t *testing.T) {
	input := inputFor(config(ModeDeterministic))
	input.Model.PolicyStatus = core.PolicyWaived
	input.Model.PolicyResults = []core.PolicyResult{{
		APIVersion:       "example.org/domain/v1",
		Constraint:       "commands-need-validator",
		Subject:          "example/UseCase/Create",
		Status:           core.PolicyWaived,
		ConstraintDigest: "constraint-digest",
		SubjectDigest:    "subject-digest",
		ExceptionName:    "temporary-validator-waiver",
		Rationale:        "migration in progress",
		Owner:            "architecture-owner",
		Decision:         "approved",
	}}

	plan, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusConverged {
		t.Fatalf("valid explicit waiver should permit evidence-relative convergence: %#v", plan)
	}
	if plan.PolicyStatus != core.PolicyWaived || len(plan.PolicyResults) != 1 || plan.PolicyResults[0].Status != core.PolicyWaived || plan.PolicyResults[0].ExceptionName != "temporary-validator-waiver" {
		t.Fatalf("waiver governance provenance was not preserved: %#v", plan)
	}
	result := plan.PolicyResults[0]
	if result.Rationale != "migration in progress" || result.Owner != "architecture-owner" || result.Decision != "approved" || result.ConstraintDigest != "constraint-digest" || result.SubjectDigest != "subject-digest" {
		t.Fatalf("waiver decision and digest binding were lost: %#v", result)
	}
}

func TestSourceIndexIncludesExactDomainDescriptorsAndDeduplicatesIdenticalSources(t *testing.T) {
	model := testModel()
	domain := model.DomainInputs[0]
	domain.Package = "engineering-constitution"
	domain.PackageVersion = "2.1.0"
	model.DomainInputs = []core.ModelDomainInput{domain, domain}
	model.Resources = append(model.Resources, model.Resources[0])

	index, err := SourceIndex(model)
	if err != nil {
		t.Fatal(err)
	}
	if len(index) != 2 {
		t.Fatalf("identical canonical source identities should deduplicate, got %#v", index)
	}
	got, exists := index["domain:example.org/domain/v1/architecture"]
	if !exists || got.Kind != "Domain" || got.Source.Path != domain.Path || got.Source.Digest != domain.Digest || got.Package != domain.Package || got.PackageVersion != domain.PackageVersion {
		t.Fatalf("Domain descriptor provenance incomplete: %#v", got)
	}
	if got, exists := index["example/UseCase/Create"]; !exists || got.Kind != "UseCase" || got.Source.Path != "architecture/resources.yaml" {
		t.Fatalf("resource GraphKey provenance changed: %#v", got)
	}
}

func TestSourceIndexRejectsConflictingProvenanceForOneIdentity(t *testing.T) {
	model := testModel()
	conflict := model.DomainInputs[0]
	conflict.Path = "other/Domain.yaml"
	model.DomainInputs = append(model.DomainInputs, conflict)
	if _, err := SourceIndex(model); err == nil {
		t.Fatal("conflicting provenance for one Domain descriptor must be rejected")
	}
}

func TestBuildAcceptsDomainDescriptorAsExactContractSource(t *testing.T) {
	input := inputFor(config(ModeDeterministic))
	input.Config.Contracts[0].Sources = []string{"domain:example.org/domain/v1/architecture"}
	plan, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	source := plan.Contracts[0].Sources
	if len(source) != 1 || source[0].Kind != "Domain" || source[0].Source.Path != "architecture/Domain.yaml" {
		t.Fatalf("contract did not retain Domain owner provenance: %#v", source)
	}
}

func TestDependencyStatusPropagatesDriftAndIncompleteEvidence(t *testing.T) {
	for _, test := range []struct {
		name         string
		prerequisite string
		wantStatus   string
		wantCode     string
		wantMessage  string
	}{
		{name: "drift", prerequisite: StatusDrift, wantStatus: StatusDrift, wantCode: "projection.dependency-drift", wantMessage: "prerequisite contract \"producer\" has drift"},
		{name: "incomplete", prerequisite: StatusIncomplete, wantStatus: StatusIncomplete, wantCode: "projection.dependency-incomplete", wantMessage: "prerequisite contract \"producer\" is incomplete"},
	} {
		t.Run(test.name, func(t *testing.T) {
			contracts := []ContractPlan{
				{ID: "consumer", DependsOn: []string{"producer"}, Status: StatusConverged},
				{ID: "producer", Status: test.prerequisite},
			}
			diagnostics := propagateDependencyStatus(contracts)
			if contracts[0].Status != test.wantStatus {
				t.Fatalf("dependent status = %q, want %q", contracts[0].Status, test.wantStatus)
			}
			if len(diagnostics) != 1 || diagnostics[0].Code != test.wantCode || diagnostics[0].ContractID != "consumer" || diagnostics[0].Message != test.wantMessage {
				t.Fatalf("dependency cause was not explained: %#v", diagnostics)
			}
		})
	}
}

func TestBuildPropagatesDeclaredPrerequisiteStatus(t *testing.T) {
	input := inputFor(config(ModeDeterministic))
	base := input.Config.Contracts[0]
	consumer := base
	consumer.ID = "consumer"
	consumer.DependsOn = []string{"producer"}
	consumer.Targets = []TargetPath{{Path: "generated/consumer.md"}}
	producer := base
	producer.ID = "producer"
	producer.Targets = []TargetPath{{Path: "generated/producer.md"}}
	input.Config.Contracts = []Contract{consumer, producer}
	input.Files = map[string][]byte{
		"generated/consumer.md": []byte("consumer"),
		"generated/producer.md": []byte("old"),
	}
	input.Desired = map[string][]byte{
		"generated/consumer.md": []byte("consumer"),
		"generated/producer.md": []byte("new"),
	}

	plan, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusDrift || len(plan.Contracts) != 2 {
		t.Fatalf("prerequisite drift should keep the overall plan in drift: %#v", plan)
	}
	if plan.Contracts[0].ID != "consumer" || plan.Contracts[0].Status != StatusDrift {
		t.Fatalf("consumer status did not inherit its producer prerequisite: %#v", plan.Contracts[0])
	}
	if len(plan.Diagnostics) != 2 || plan.Diagnostics[0].Code != "projection.dependency-drift" || plan.Diagnostics[0].ContractID != "consumer" || plan.Diagnostics[0].Message != "prerequisite contract \"producer\" has drift" {
		t.Fatalf("plan omitted dependency provenance: %#v", plan.Diagnostics)
	}
}
