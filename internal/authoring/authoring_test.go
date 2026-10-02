package authoring

import (
	"strings"
	"testing"
)

func TestContextCompilesPortableCoreAuthoringClosure(t *testing.T) {
	const version = "0.1.0-rc.3-dev"
	const toolDigest = "sha256:tool-bytes"
	got, err := Context(version, toolDigest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != version || got.ToolDigest != toolDigest || got.Entry != entry {
		t.Fatalf("unexpected authoring context identity: %#v", got)
	}
	if got.Provisional || got.Revision != "embedded" {
		t.Fatalf("embedded immutable guidance lacks an explicit source marker: %#v", got)
	}
	want := map[string]bool{
		"markitect.yaml": false,
		"internal/authoring/resources/rule-canonical-ownership.yaml":       false,
		"internal/authoring/resources/text-resource-modelling.yaml":        false,
		"internal/authoring/resources/workflow-authoring-change.yaml":      false,
		"internal/authoring/resources/workflow-engineering-discovery.yaml": false,
		"internal/authoring/resources/workflow-constitution-change.yaml":   false,
		"internal/authoring/resources/skill-authoring.yaml":                false,
	}
	if len(got.Inputs) != len(want) {
		t.Fatalf("compiled %d inputs, want %d: %#v", len(got.Inputs), len(want), got.Inputs)
	}
	var skillText, workflowText, discoveryText, constitutionText string
	discoveryWorkflow, constitutionWorkflow := false, false
	for _, input := range got.Inputs {
		if _, ok := want[input.Path]; !ok {
			t.Errorf("unexpected context input %q", input.Path)
			continue
		}
		want[input.Path] = true
		if !strings.HasPrefix(input.Hash, "sha256:") || input.Reason == "" {
			t.Errorf("input lacks stable provenance: %#v", input)
		}
		if input.Path == "internal/authoring/resources/skill-authoring.yaml" {
			skillText = input.Resource.Spec.Text
		}
		if input.Path == "internal/authoring/resources/workflow-authoring-change.yaml" {
			workflowText = input.Resource.Spec.Text
		}
		if input.Path == "internal/authoring/resources/workflow-engineering-discovery.yaml" {
			discoveryText = input.Resource.Spec.Text
		}
		if input.Path == "internal/authoring/resources/workflow-constitution-change.yaml" {
			constitutionText = input.Resource.Spec.Text
		}
		if input.Resource != nil && input.Resource.Kind == "Workflow" && input.Resource.Metadata.Namespace == "core" {
			discoveryWorkflow = discoveryWorkflow || input.Resource.Metadata.Name == "engineering-discovery"
			constitutionWorkflow = constitutionWorkflow || input.Resource.Metadata.Name == "constitution-change"
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("compiled context omitted %s", name)
		}
	}
	for _, required := range []string{"find --query TEXT", "explain --kind KIND", "canonical YAML", "human acceptance", ".markitect/areas/<responsibility>/", "YAML `kind` field is authoritative", "configured Area paths", "human-owned navigation", "Configure Areas to cover canonical resources and exact file inputs"} {
		if !strings.Contains(skillText, required) {
			t.Errorf("authoring Skill omits %q", required)
		}
	}
	for _, required := range []string{"init --repo PATH --name PROJECT --namespace OWNER`", "`.markitect/areas/OWNER`", "`.markitect/areas/OWNER/README.md`", "`--path AREA`", "`.markitect/packages/`", "other committed snapshot-included paths remain valid"} {
		if !strings.Contains(workflowText, required) && !strings.Contains(skillText, required) {
			t.Errorf("authoring guidance omits %q", required)
		}
	}
	if !discoveryWorkflow || !constitutionWorkflow {
		t.Errorf("authoring context does not reach both optional workflows (discovery=%t constitution=%t)", discoveryWorkflow, constitutionWorkflow)
	}
	for _, required := range []string{"optional provider-neutral Engineering Discovery workflow", "outside configured Areas", "separate human decision"} {
		if !strings.Contains(skillText, required) {
			t.Errorf("authoring Skill omits optional discovery boundary %q", required)
		}
	}
	for _, required := range []string{"immutable full commit IDs", "counterexamples", "numerator, denominator", "candidate hash", "evidence-ledger hash", "do not authenticate", "No step in discovery writes"} {
		if !strings.Contains(discoveryText, required) {
			t.Errorf("Engineering Discovery workflow omits %q", required)
		}
	}
	for _, required := range []string{"exact candidate and evidence hashes", "Domain definition before", "normal Project branch", "does not prove code conforms"} {
		if !strings.Contains(constitutionText, required) {
			t.Errorf("Constitution Change workflow omits %q", required)
		}
	}
}

func TestContextIdentityTracksToolAndVersion(t *testing.T) {
	base, err := Context("1.2.3", "sha256:one")
	if err != nil {
		t.Fatal(err)
	}
	otherTool, err := Context("1.2.3", "sha256:two")
	if err != nil {
		t.Fatal(err)
	}
	otherVersion, err := Context("1.2.4", "sha256:one")
	if err != nil {
		t.Fatal(err)
	}
	if base.Digest == otherTool.Digest || base.Digest == otherVersion.Digest {
		t.Fatal("authoring context digest ignored tool identity or version")
	}
	if _, err := Context(" ", ""); err == nil {
		t.Fatal("empty tool version accepted")
	}
}
