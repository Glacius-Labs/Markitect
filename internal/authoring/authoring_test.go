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
		"internal/authoring/resources/rule-canonical-ownership.yaml":  false,
		"internal/authoring/resources/text-resource-modelling.yaml":   false,
		"internal/authoring/resources/workflow-authoring-change.yaml": false,
		"internal/authoring/resources/skill-authoring.yaml":           false,
	}
	if len(got.Inputs) != len(want) {
		t.Fatalf("compiled %d inputs, want %d: %#v", len(got.Inputs), len(want), got.Inputs)
	}
	var skillText string
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
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("compiled context omitted %s", name)
		}
	}
	for _, required := range []string{"find --query TEXT", "explain --kind KIND", "canonical YAML", "human acceptance"} {
		if !strings.Contains(skillText, required) {
			t.Errorf("authoring Skill omits %q", required)
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
