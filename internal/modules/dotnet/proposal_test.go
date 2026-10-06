package dotnet

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func moduleProposalFixture() Input {
	api := "proof.example/v1"
	schema := core.Schema{
		APIVersion: api,
		Purpose:    "Project-owned mission vocabulary.",
		Kinds: map[string]core.Kind{
			"Mission": {
				Purpose: "A bounded operational objective.",
				Properties: map[string]core.Property{
					"effectBoundary": {Purpose: "The application boundary for visible effects.", Type: core.TypeString, MinCount: 1, MaxCount: 1},
				},
			},
			"Capability": {Purpose: "A unit of owned operational ability."},
		},
	}
	definitions := []core.Definition{
		{APIVersion: api, Kind: "Mission", Metadata: core.Metadata{Namespace: "ops", Name: "renewal"}, Purpose: "Renew expiring assets.", Spec: map[string]any{"effectBoundary": "application"}},
		{APIVersion: api, Kind: "Capability", Metadata: core.Metadata{Namespace: "ops", Name: "dispatch"}, Purpose: "Schedule renewal actions."},
	}
	policy := func(kind, name, guidance string) core.Definition {
		return core.Definition{
			APIVersion: "markitect.foundation/v1", Kind: "ProjectionPolicy",
			Metadata: core.Metadata{Namespace: "ops", Name: name},
			Purpose:  "Guide one target representation.",
			Spec: map[string]any{
				"sourceKind":       map[string]any{"apiVersion": api, "kind": kind},
				"targetTechnology": "dotnet",
				"guidance":         guidance,
			},
		}
	}
	return Input{
		Definitions: definitions, Schemas: []core.Schema{schema},
		Policies: []core.Definition{
			policy("Mission", "mission-to-dotnet", "Keep visible effects behind the application boundary."),
			policy("Capability", "capability-to-dotnet", "Keep ownership explicit in the application layer."),
		},
		TargetPrefix: "src/", AllowedRoots: []string{"src/"},
		RequestDigest: "sha256:" + strings.Repeat("1", 64), InventoryComplete: true,
	}
}

func TestProposeBuildsUnknownKindTaskFromSchemaAndPoliciesWithoutNamingFiles(t *testing.T) {
	input := moduleProposalFixture()
	proposal := Propose(input)
	if proposal.Decision != DecisionWork || proposal.Task == nil || proposal.Task.RequestDigest != input.RequestDigest {
		t.Fatalf("new scope proposal = %#v", proposal)
	}
	if proposal.Task.TargetPrefix != "src" || !reflect.DeepEqual(proposal.Task.AllowedExtensions, []string{".cs", ".csproj"}) {
		t.Fatalf("task target bounds = %#v", proposal.Task)
	}
	if len(proposal.Task.Kinds) != 2 || proposal.Task.Kinds[0].Identity.Kind != "Capability" || proposal.Task.Kinds[1].Identity.Kind != "Mission" {
		t.Fatalf("unknown source Kinds were not retained in stable order: %#v", proposal.Task.Kinds)
	}
	mission := proposal.Task.Kinds[1]
	if mission.SchemaPurpose != "Project-owned mission vocabulary." || mission.Kind.Purpose != "A bounded operational objective." || mission.Kind.Properties["effectBoundary"].Purpose != "The application boundary for visible effects." {
		t.Fatalf("Core semantics were dropped from the Executor task: %#v", mission)
	}
	if len(mission.Policies) != 1 || !strings.Contains(mission.Policies[0].Spec["guidance"].(string), "application boundary") {
		t.Fatalf("matching ProjectionPolicy was not included: %#v", mission.Policies)
	}
	if proposal.Task.Objective == "" || len(proposal.Task.ExistingOwnedArtifacts) != 0 {
		t.Fatalf("task must state its objective and leave new filenames to Executor: %#v", proposal.Task)
	}
}

func TestProposeRequiresAdequateUnambiguousTargetGuidance(t *testing.T) {
	input := moduleProposalFixture()
	input.Policies = input.Policies[:1]
	if got := Propose(input); got.Decision != DecisionEscalate || !reflect.DeepEqual(got.Reasons, []string{"policy.guidance-missing"}) {
		t.Fatalf("missing selected Kind policy did not escalate: %#v", got)
	}

	input = moduleProposalFixture()
	input.Policies = append(input.Policies, input.Policies[0])
	if got := Propose(input); got.Decision != DecisionEscalate || !reflect.DeepEqual(got.Reasons, []string{"policy.ambiguous"}) {
		t.Fatalf("duplicate policy was not treated as ambiguous: %#v", got)
	}

	input = moduleProposalFixture()
	input.Policies[0].Spec["targetTechnology"] = "markdown"
	if got := Propose(input); got.Decision != DecisionEscalate || !containsCode(got.Reasons, "policy.target-mismatch") {
		t.Fatalf("wrong-target policy did not escalate: %#v", got)
	}

	input = moduleProposalFixture()
	input.Policies[0].Spec["guidance"] = "   "
	if got := Propose(input); got.Decision != DecisionEscalate || !containsCode(got.Reasons, "policy.guidance-empty") {
		t.Fatalf("empty guidance did not escalate: %#v", got)
	}
}

func TestProposeNoopRequiresCurrentRequestBytesAndVerification(t *testing.T) {
	input := moduleProposalFixture()
	const ownedPath = "src/mission.cs"
	current := []byte("class Mission {}")
	binding := ArtifactBinding{Path: ownedPath, Digest: digest(current), Mode: "regular"}
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, Artifacts: []ArtifactBinding{binding}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: ownedPath, Bytes: current, Mode: "regular"}}
	input.Verification = &VerificationBinding{Passed: true, RequestDigest: input.RequestDigest, Artifacts: []ArtifactBinding{binding}}
	got := Propose(input)
	if got.Decision != DecisionNoop || got.EvidenceRefreshRequired || got.Task != nil {
		t.Fatalf("current verified state should be no-op: %#v", got)
	}

	input.RequestDigest = "sha256:" + strings.Repeat("2", 64)
	stale := Propose(input)
	if stale.Decision != DecisionNoop || !stale.EvidenceRefreshRequired || stale.Task != nil {
		t.Fatalf("unaffected stale binding should require evidence refresh without materialization: %#v", stale)
	}
	input.CanonicalAffected = true
	affected := Propose(input)
	if affected.Decision != DecisionWork || affected.Task == nil || !reflect.DeepEqual(affected.Reasons, []string{"canonical-or-binding-change"}) {
		t.Fatalf("affected canonical scope did not propose bounded work: %#v", affected)
	}
	if len(affected.Task.ExistingOwnedArtifacts) != 1 || affected.Task.ExistingOwnedArtifacts[0].Path != ownedPath {
		t.Fatalf("task omitted current owned artifact state: %#v", affected.Task.ExistingOwnedArtifacts)
	}
}

func TestProposeDriftUnobservedUnknownAndRetiredArtifacts(t *testing.T) {
	input := moduleProposalFixture()
	const ownedPath = "src/existing.cs"
	original := []byte("class Existing {}")
	binding := ArtifactBinding{Path: ownedPath, Digest: digest(original), Mode: "regular"}
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, Artifacts: []ArtifactBinding{binding}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: ownedPath, Bytes: []byte("drift"), Mode: "regular"}}
	if got := Propose(input); got.Decision != DecisionWork || !reflect.DeepEqual(got.Reasons, []string{"projection-drift"}) {
		t.Fatalf("owned byte drift did not propose repair: %#v", got)
	}

	missing := input
	missing.ObservedArtifacts = nil
	missing.CanonicalAffected = false
	if got := Propose(missing); got.Decision != DecisionWork || !reflect.DeepEqual(got.Reasons, []string{"projection-drift"}) {
		t.Fatalf("complete inventory did not expose missing owned artifact: %#v", got)
	}

	incomplete := input
	incomplete.InventoryComplete = false
	if got := Propose(incomplete); got.Decision != DecisionEscalate || !reflect.DeepEqual(got.Reasons, []string{"target.inventory.incomplete"}) {
		t.Fatalf("incomplete inventory was treated as clean: %#v", got)
	}

	unknown := input
	unknown.ObservedArtifacts = append(append([]ArtifactObservation(nil), input.ObservedArtifacts...), ArtifactObservation{Path: "src/manual.cs", Bytes: []byte("unowned"), Mode: "regular"})
	if got := Propose(unknown); got.Decision != DecisionEscalate || !reflect.DeepEqual(got.Reasons, []string{"artifact.unknown"}) {
		t.Fatalf("unowned target artifact did not escalate: %#v", got)
	}

	retired := input
	retired.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, RetiredArtifacts: []string{"src/retired.cs"}}
	if got := Propose(retired); got.Decision != DecisionEscalate || !reflect.DeepEqual(got.Reasons, []string{"artifact.retired"}) {
		t.Fatalf("retired artifact was treated as deletable: %#v", got)
	}
}

func TestProposeIsDeterministicAcrossInputOrdering(t *testing.T) {
	input := moduleProposalFixture()
	first := Propose(input)
	reversed := moduleProposalFixture()
	reversed.Definitions[0], reversed.Definitions[1] = reversed.Definitions[1], reversed.Definitions[0]
	reversed.Policies[0], reversed.Policies[1] = reversed.Policies[1], reversed.Policies[0]
	again := Propose(reversed)
	if !reflect.DeepEqual(first, again) {
		t.Fatalf("proposal depends on input ordering:\nfirst=%#v\nagain=%#v", first, again)
	}
}

func containsCode(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestProposeDoesNotTreatEmptyPriorArtifactSetAsCurrent(t *testing.T) {
	input := moduleProposalFixture()
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true}
	if got := Propose(input); got.Decision != DecisionWork || got.Task == nil || !containsCode(got.Reasons, "incomplete-materialization") {
		t.Fatalf("empty prior artifact set was treated as a clean no-op: %#v", got)
	}
}

func TestProposeCurrentVerificationClearsOldMaterializationRequest(t *testing.T) {
	input := moduleProposalFixture()
	const ownedPath = "src/mission.cs"
	current := []byte("class Mission {}")
	binding := ArtifactBinding{Path: ownedPath, Digest: digest(current), Mode: "regular"}
	input.Previous = &PriorProjection{RequestDigest: "sha256:" + strings.Repeat("2", 64), Complete: true, Artifacts: []ArtifactBinding{binding}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: ownedPath, Bytes: current, Mode: "regular"}}
	input.Verification = &VerificationBinding{Passed: true, RequestDigest: input.RequestDigest, Artifacts: []ArtifactBinding{binding}}
	if got := Propose(input); got.Decision != DecisionNoop || got.EvidenceRefreshRequired {
		t.Fatalf("current verification should clear old materialization request digest: %#v", got)
	}
}
