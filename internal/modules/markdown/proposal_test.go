package markdown

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
		Purpose:    "Mission planning for a local project.",
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
	policy := core.Definition{
		APIVersion: "markitect.foundation/v1", Kind: "ProjectionPolicy",
		Metadata: core.Metadata{Namespace: "ops", Name: "mission-to-markdown"},
		Purpose:  "Keep operational decisions visible.",
		Spec: map[string]any{
			"sourceKind":       map[string]any{"apiVersion": api, "kind": "Mission"},
			"targetTechnology": "markdown",
			"guidance":         "Preserve the explicit application boundary.",
		},
	}
	return Input{
		Definitions: definitions, Schemas: []core.Schema{schema}, Policies: []core.Definition{policy, {
			APIVersion: "markitect.foundation/v1", Kind: "ProjectionPolicy",
			Metadata: core.Metadata{Namespace: "ops", Name: "capability-to-markdown"},
			Purpose:  "Describe owned operational ability.",
			Spec:     map[string]any{"sourceKind": map[string]any{"apiVersion": api, "kind": "Capability"}, "targetTechnology": "markdown", "guidance": "Keep ownership and responsibilities explicit."},
		}},
		TargetPrefix: "docs/represented/", AllowedRoots: []string{"docs/"},
		RequestDigest: "sha256:" + strings.Repeat("1", 64), InventoryComplete: true,
	}
}

func TestProposeChoosesModuleOwnedPathAndComparesRenderedBytes(t *testing.T) {
	input := moduleProposalFixture()
	first := Propose(input)
	if first.Decision != DecisionWork || len(first.Files) != 1 {
		t.Fatalf("new projection proposal = %#v", first)
	}
	const wantPath = "docs/represented/index.md"
	content, exists := first.Files[wantPath]
	if !exists || !strings.Contains(string(content), "A bounded operational objective.") || !strings.Contains(string(content), "Preserve the explicit application boundary.") {
		t.Fatalf("Module did not choose/render its own target and selected policy: %#v", first)
	}

	digestValue := digest(content)
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, Artifacts: []ArtifactBinding{{Path: wantPath, Digest: digestValue, Mode: "regular"}}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: wantPath, Bytes: content, Mode: "regular"}}
	input.Verification = &VerificationBinding{Passed: true, RequestDigest: input.RequestDigest, Artifacts: []ArtifactBinding{{Path: wantPath, Digest: digestValue, Mode: "regular"}}}
	current := Propose(input)
	if current.Decision != DecisionNoop || current.EvidenceRefreshRequired || len(current.Files) != 0 {
		t.Fatalf("current verified projection proposal = %#v", current)
	}
}

func TestProposeSeparatesEvidenceRefreshAndEscalatesUnsafeInventory(t *testing.T) {
	input := moduleProposalFixture()
	first := Propose(input)
	pathName := "docs/represented/index.md"
	content := first.Files[pathName]
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, Artifacts: []ArtifactBinding{{Path: pathName, Digest: digest(content), Mode: "regular"}}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: pathName, Bytes: content, Mode: "regular"}}
	input.RequestDigest = "sha256:" + strings.Repeat("2", 64)
	input.CanonicalAffected = false
	stale := Propose(input)
	if stale.Decision != DecisionNoop || !stale.EvidenceRefreshRequired {
		t.Fatalf("unaffected representation should request evidence refresh without materialization: %#v", stale)
	}

	incomplete := input
	incomplete.InventoryComplete = false
	if got := Propose(incomplete); got.Decision != DecisionEscalate || got.Reasons[0] != "target.inventory.incomplete" {
		t.Fatalf("incomplete inventory was treated as clean: %#v", got)
	}
	unknown := input
	unknown.ObservedArtifacts = append(append([]ArtifactObservation(nil), input.ObservedArtifacts...), ArtifactObservation{Path: "docs/represented/manual.md", Bytes: []byte("unowned"), Mode: "regular"})
	if got := Propose(unknown); got.Decision != DecisionEscalate || got.Reasons[0] != "artifact.unknown" {
		t.Fatalf("unknown target artifact was not escalated: %#v", got)
	}
	retired := input
	retired.Previous = &PriorProjection{RequestDigest: input.Previous.RequestDigest, Complete: true, RetiredArtifacts: []string{"docs/represented/old.md"}}
	if got := Propose(retired); got.Decision != DecisionEscalate || got.Reasons[0] != "artifact.retired" {
		t.Fatalf("retired ownership was not escalated: %#v", got)
	}
}

func TestProposeRepairsObservedDriftAndIsOrderDeterministic(t *testing.T) {
	input := moduleProposalFixture()
	first := Propose(input)
	pathName := "docs/represented/index.md"
	desired := first.Files[pathName]
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, Artifacts: []ArtifactBinding{{Path: pathName, Digest: digest(desired), Mode: "regular"}}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: pathName, Bytes: []byte("drift"), Mode: "regular"}}
	drift := Propose(input)
	if drift.Decision != DecisionWork || !reflect.DeepEqual(drift.Reasons, []string{"projection-drift"}) {
		t.Fatalf("owned drift did not produce repair work: %#v", drift)
	}

	reversed := moduleProposalFixture()
	reversed.Definitions[0], reversed.Definitions[1] = reversed.Definitions[1], reversed.Definitions[0]
	reversed.Policies[0], reversed.Policies[1] = reversed.Policies[1], reversed.Policies[0]
	again := Propose(reversed)
	if !reflect.DeepEqual(first, again) {
		t.Fatalf("proposal depends on input order:\nfirst=%#v\nagain=%#v", first, again)
	}
}

func TestProposeCurrentVerificationClearsOldMaterializationRequest(t *testing.T) {
	input := moduleProposalFixture()
	first := Propose(input)
	pathName := "docs/represented/index.md"
	content := first.Files[pathName]
	binding := ArtifactBinding{Path: pathName, Digest: digest(content), Mode: "regular"}
	input.Previous = &PriorProjection{RequestDigest: "sha256:" + strings.Repeat("2", 64), Complete: true, Artifacts: []ArtifactBinding{binding}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: pathName, Bytes: content, Mode: "regular"}}
	input.Verification = &VerificationBinding{Passed: true, RequestDigest: input.RequestDigest, Artifacts: []ArtifactBinding{binding}}
	if got := Propose(input); got.Decision != DecisionNoop || got.EvidenceRefreshRequired {
		t.Fatalf("current verification should clear old materialization request digest: %#v", got)
	}
}
