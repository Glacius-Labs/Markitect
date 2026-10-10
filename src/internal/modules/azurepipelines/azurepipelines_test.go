package azurepipelines

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"go.yaml.in/yaml/v3"
)

func fixture() Input {
	const api = "workflow.example/v1"
	const ruleText = "Require the canonical workflow check.\nscript: echo must-remain-inert\n$(touch injected)"
	const policyGuidance = "Keep the selected Rule explicit.\nsteps:\n  - script: echo must-remain-inert"
	definition := core.Definition{
		APIVersion: api,
		Kind:       "Rule",
		Metadata:   core.Metadata{Namespace: "delivery", Name: "canonical-checks"},
		Purpose:    "Project-owned requirement for checked workflows.",
		Spec:       map[string]any{"text": ruleText},
		Source:     core.Source{Path: ".markitect/areas/delivery/workflow.yaml", Digest: "sha256:" + strings.Repeat("c", 64), Line: 12},
	}
	policy := core.Definition{
		APIVersion: "markitect.foundation/v1",
		Kind:       "ProjectionPolicy",
		Metadata:   core.Metadata{Namespace: "delivery", Name: "workflow-to-azure"},
		Purpose:    "Select the Azure Pipelines representation.",
		Spec: map[string]any{
			"sourceKind":       map[string]any{"apiVersion": api, "kind": "Rule"},
			"targetTechnology": TargetTechnology,
			"guidance":         policyGuidance,
		},
	}
	return Input{
		Definitions:       []core.Definition{definition},
		Schemas:           []core.Schema{{APIVersion: api, Purpose: "Project engineering policy schema.", Kinds: map[string]core.Kind{"Rule": {Purpose: "A project-owned normative statement."}}}},
		Policies:          []core.Definition{policy},
		TargetPrefix:      "ci/azure/",
		AllowedRoots:      []string{"ci"},
		Checks:            []NamedCheck{{Name: RequiredCheck, Argv: []string{"go", "test", "./internal/modules/azurepipelines"}}},
		RequestDigest:     "sha256:" + strings.Repeat("a", 64),
		InventoryComplete: true,
	}
}

func TestRenderProducesExecutableAzureStepWithExactConfiguredArgvAndProvenance(t *testing.T) {
	input := fixture()
	result := Render(input)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Render diagnostics = %#v", result.Diagnostics)
	}
	const wantPath = "ci/azure/azure-pipelines.yml"
	if result.Mode != "100644" || len(result.Files) != 1 {
		t.Fatalf("render result must own one regular output: mode=%q files=%#v", result.Mode, result.Files)
	}
	data, exists := result.Files[wantPath]
	if !exists || len(data) == 0 {
		t.Fatalf("rendered output missing at %q: %#v", wantPath, result.Files)
	}
	var document struct {
		Steps []struct {
			Script      string `yaml:"script"`
			DisplayName string `yaml:"displayName"`
		} `yaml:"steps"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("rendered Azure YAML is invalid: %v\n%s", err, data)
	}
	if len(document.Steps) != 1 || document.Steps[0].Script != strings.Join(input.Checks[0].Argv, " ") || document.Steps[0].DisplayName != RequiredCheck {
		t.Fatalf("pipeline must contain exactly the supplied executable Project check: %#v", document.Steps)
	}
	var yamlTree map[string]any
	if err := yaml.Unmarshal(data, &yamlTree); err != nil || len(yamlTree) != 1 || yamlTree["steps"] == nil {
		t.Fatalf("canonical prose or policy guidance escaped YAML comments: tree=%#v error=%v", yamlTree, err)
	}
	text := string(data)
	for _, evidence := range []string{ModuleName + "@" + ModuleVersion, ProjectorID, TargetTechnology, input.Definitions[0].Identity().Key(), input.Definitions[0].Source.Path, input.Definitions[0].Source.Digest, input.Policies[0].Identity().Key()} {
		if !strings.Contains(text, evidence) {
			t.Errorf("pipeline provenance omits %q", evidence)
		}
	}
	if strings.TrimSpace(document.Steps[0].Script) == "" {
		t.Fatal("comment-only or no-op-only YAML is not an executable projection")
	}
	var recordedDefinitions []core.Definition
	var recordedPolicies []core.Definition
	for _, line := range strings.Split(text, "\n") {
		for prefix, destination := range map[string]*[]core.Definition{
			"# Markitect Definition JSON: ":       &recordedDefinitions,
			"# Markitect ProjectionPolicy JSON: ": &recordedPolicies,
		} {
			if strings.HasPrefix(line, prefix) {
				var value core.Definition
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, prefix)), &value); err != nil {
					t.Fatalf("provenance comment is not safely encoded JSON: %v", err)
				}
				*destination = append(*destination, value)
			}
		}
	}
	if !reflect.DeepEqual(recordedDefinitions, input.Definitions) || !reflect.DeepEqual(recordedPolicies, input.Policies) {
		t.Fatalf("inert JSON comments must preserve exact selected meaning and policies:\ndefinitions=%#v\npolicies=%#v", recordedDefinitions, recordedPolicies)
	}
}

func TestProposeRequiresCurrentOwnedBytesForNoop(t *testing.T) {
	input := fixture()
	target := "ci/azure/azure-pipelines.yml"
	want := Render(input).Files[target]
	first := Propose(input)
	if first.Decision != DecisionWork || first.Files[target] == nil || first.Mode != "100644" {
		t.Fatalf("unmaterialized output should propose bounded work: %#v", first)
	}
	binding := ArtifactBinding{Path: target, Digest: digest(want), Mode: OutputMode}
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, Artifacts: []ArtifactBinding{binding}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: target, Bytes: want, Mode: OutputMode}}
	withoutVerification := Propose(input)
	if withoutVerification.Decision != DecisionNoop || !withoutVerification.EvidenceRefreshRequired || len(withoutVerification.Files) != 0 {
		t.Fatalf("byte-current representation without verification must be no-op plus evidence refresh: %#v", withoutVerification)
	}
	input.Verification = &VerificationBinding{Passed: true, RequestDigest: input.RequestDigest, Artifacts: []ArtifactBinding{binding}}
	verified := Propose(input)
	if verified.Decision != DecisionNoop || verified.EvidenceRefreshRequired || !reflect.DeepEqual(verified.Reasons, []string{"representation-current"}) {
		t.Fatalf("exact current verification should clear refresh: %#v", verified)
	}
	input.Verification.RequestDigest = "sha256:" + strings.Repeat("b", 64)
	stale := Propose(input)
	if stale.Decision != DecisionNoop || !stale.EvidenceRefreshRequired {
		t.Fatalf("stale verification must request refresh without rewriting matching bytes: %#v", stale)
	}
}

func TestProposeEscalatesMissingAmbiguousAndUnsafeInputs(t *testing.T) {
	input := fixture()
	input.Policies = nil
	if got := Propose(input); got.Decision != DecisionEscalate {
		t.Fatalf("missing policy did not escalate: %#v", got)
	}

	input = fixture()
	input.Policies = append(input.Policies, input.Policies[0])
	if got := Propose(input); got.Decision != DecisionEscalate {
		t.Fatalf("duplicate policy did not escalate: %#v", got)
	}

	input = fixture()
	input.Checks[0].Argv = []string{"go", "test", "./...", "&&", "echo", "unexpected"}
	if got := Propose(input); got.Decision != DecisionEscalate {
		t.Fatalf("shell metacharacter argv did not escalate: %#v", got)
	}

	input = fixture()
	input.Checks = nil
	if got := Propose(input); got.Decision != DecisionEscalate {
		t.Fatalf("missing required Project check did not escalate: %#v", got)
	}
}

func TestProposeEscalatesUnknownAndPortableCaseAliasArtifacts(t *testing.T) {
	input := fixture()
	target := "ci/azure/azure-pipelines.yml"
	want := Render(input).Files[target]
	binding := ArtifactBinding{Path: target, Digest: digest(want), Mode: OutputMode}
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, Artifacts: []ArtifactBinding{binding}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: target, Bytes: want, Mode: OutputMode}}
	input.ObservedArtifacts = append(input.ObservedArtifacts, ArtifactObservation{Path: "ci/azure/manual.yml", Bytes: []byte("unowned"), Mode: OutputMode})
	if got := Propose(input); got.Decision != DecisionEscalate || got.Reasons[0] != "artifact.unknown" {
		t.Fatalf("unknown target file did not escalate: %#v", got)
	}
	input.ObservedArtifacts = []ArtifactObservation{{Path: "ci/azure/Azure-Pipelines.yml", Bytes: want, Mode: OutputMode}}
	if got := Propose(input); got.Decision != DecisionEscalate || got.Reasons[0] != "artifact.path.alias" {
		t.Fatalf("case-fold alias could be mistaken for an absent target: %#v", got)
	}
}

func TestRenderRejectsUnsafePortableTargetComponents(t *testing.T) {
	for _, prefix := range []string{".Git/secret", "ci/CON.txt", "ci/name./nested", "ci/name /nested", "ci/COM1.yml", "ci/LPT³.txt"} {
		input := fixture()
		input.TargetPrefix = prefix
		if got := Render(input); len(got.Files) != 0 || len(got.Diagnostics) == 0 {
			t.Errorf("unsafe target prefix %q was not rejected: %#v", prefix, got)
		}
	}
}

func TestRenderRejectsInvalidOrUnboundedUTF8TargetPaths(t *testing.T) {
	for _, prefix := range []string{
		string([]byte{'c', 'i', '/', 0xff}),
		"ci/" + strings.Repeat("a", maxPathPartBytes+1),
		strings.Repeat("a", maxPathBytes+1),
		strings.Repeat("a/", maxPathParts) + "a",
	} {
		input := fixture()
		input.TargetPrefix = prefix
		if got := Render(input); len(got.Files) != 0 || len(got.Diagnostics) == 0 {
			t.Errorf("invalid or unbounded target prefix was not rejected: len=%d result=%#v", len(prefix), got)
		}
	}
}

func TestProposeRequiresLiteralSHA256Prefix(t *testing.T) {
	input := fixture()
	input.RequestDigest = strings.Repeat("a", 64)
	if got := Propose(input); got.Decision != DecisionEscalate || got.Reasons[0] != "request.digest.invalid" {
		t.Fatalf("unprefixed request digest was accepted: %#v", got)
	}

	input = fixture()
	input.Previous = &PriorProjection{RequestDigest: strings.Repeat("b", 64), Complete: true}
	if got := Propose(input); got.Decision != DecisionEscalate || got.Reasons[0] != "record.request.digest.invalid" {
		t.Fatalf("unprefixed prior request digest was accepted: %#v", got)
	}

	input = fixture()
	target := "ci/azure/azure-pipelines.yml"
	want := Render(input).Files[target]
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, Artifacts: []ArtifactBinding{{Path: target, Digest: strings.Repeat("c", 64), Mode: OutputMode}}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: target, Bytes: want, Mode: OutputMode}}
	if got := Propose(input); got.Decision != DecisionEscalate || got.Reasons[0] != "record.artifact.invalid" {
		t.Fatalf("unprefixed prior artifact digest was accepted: %#v", got)
	}
}

func TestRenderIsDeterministicAcrossInputOrder(t *testing.T) {
	firstInput := fixture()
	firstInput.Definitions = append(firstInput.Definitions, core.Definition{
		APIVersion: "workflow.example/v1",
		Kind:       "Rule",
		Metadata:   core.Metadata{Namespace: "delivery", Name: "secondary"},
		Purpose:    "Another selected workflow.",
		Spec:       map[string]any{},
	})
	first := Render(firstInput)
	reversed := firstInput
	reversed.Definitions = append([]core.Definition(nil), firstInput.Definitions...)
	reversed.Definitions[0], reversed.Definitions[1] = reversed.Definitions[1], reversed.Definitions[0]
	if !bytes.Equal(first.Files["ci/azure/azure-pipelines.yml"], Render(reversed).Files["ci/azure/azure-pipelines.yml"]) {
		t.Fatal("rendered Azure pipeline depends on input Definition ordering")
	}
}
