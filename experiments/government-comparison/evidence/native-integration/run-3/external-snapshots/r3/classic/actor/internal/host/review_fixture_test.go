package host

import (
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
)

func reviewResources(withRule, omitUnrelated, configureCodex bool) []*authoring.Resource {
	policy := &authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "review-test"}, Path: projectPath}}
	policy.Spec.Areas = []authoring.Area{{Name: "general", Path: "docs/general"}}
	if withRule {
		policy.Spec.Areas[0].Rules = []core.Ref{{Name: "shared"}}
	}
	if configureCodex {
		policy.Spec.Targets = []string{"codex"}
	}
	entry := &authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "review", Namespace: "general"}, Path: "docs/general/workflows/review.yaml"}, Spec: authoring.Spec{Text: "Review the change.", Uses: []core.Ref{{Kind: "Text", Name: "guide"}}}}
	guide := &authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Name: "guide", Namespace: "general"}, Path: "docs/general/text/guide.yaml"}, Spec: authoring.Spec{Text: "Stable guide."}}
	unrelated := &authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Name: "unrelated", Namespace: "general"}, Path: "docs/general/text/unrelated.yaml"}, Spec: authoring.Spec{Text: "Unrelated reference."}}
	resources := []*authoring.Resource{policy, guide, entry}
	if withRule {
		rule := &authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "shared", Namespace: "general"}, Path: "docs/general/rules/shared.yaml"}, Spec: authoring.Spec{Text: "Shared requirement."}}
		resources = append(resources, rule)
	}
	if !omitUnrelated {
		resources = append(resources, unrelated)
	}
	return resources
}

func reviewFixture(t *testing.T, revision string, provisional bool, resources []*authoring.Resource, extra map[string]string) *Project {
	t.Helper()
	snapshot := &snapshot.Snapshot{ID: revision, Provisional: provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, resource := range resources {
		snapshot.Files[resource.Path] = encodeResource(t, *resource)
		snapshot.Modes[resource.Path] = "100644"
	}
	for name, content := range extra {
		snapshot.Files[name] = []byte(content)
		snapshot.Modes[name] = "100644"
	}
	parsed, err := Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("review fixture has diagnostics: %#v", parsed.Diagnostics)
	}
	outputs, err := GenerateOutputs(parsed)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range outputs {
		snapshot.Files[name] = data
		snapshot.Modes[name] = "100644"
	}
	parsed, err = Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("rendered review fixture has diagnostics: %#v", parsed.Diagnostics)
	}
	return parsed
}

func mustRecordReview(t *testing.T, p *Project, config ReviewConfig) *ReviewRecord {
	t.Helper()
	record, err := RecordReview(p, "general/Workflow/review", "v1", "tool-1", config, "Original review text; no verdict parsing.")
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func renderOutputs(t *testing.T, p *Project) map[string][]byte {
	t.Helper()
	outputs, err := GenerateOutputs(p)
	if err != nil {
		t.Fatal(err)
	}
	return outputs
}
