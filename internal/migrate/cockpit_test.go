package migrate

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/render"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func TestCockpitPlansScopedResourcesWithoutProviderOwnership(t *testing.T) {
	snapshot := &source.Snapshot{Files: map[string][]byte{
		"docs/README.md":                                                    []byte("# Cockpit\n"),
		"docs/general/README.md":                                            []byte("# General\n"),
		"docs/general/rules/documentation.md":                               []byte("# Documentation\nKeep source ownership explicit.\n"),
		"docs/general/rules/mechanisms.md":                                  []byte("# Mechanisms\nUse canonical sources.\n"),
		"docs/general/workflows/author-mechanism.md":                        []byte("# Authoring\nFollow the source workflow.\n"),
		"docs/general/workflows/other.md":                                   []byte("Another workflow.\n"),
		"docs/general/skills/author-mechanism.md":                           []byte("---\nname: author-mechanism\ndescription: Author mechanisms\n---\n\n# Author\nFollow the [Authoring Workflow](../workflows/author-mechanism.md). Also see [another workflow](../workflows/other.md).\n"),
		"docs/general/agents/documentation-auditor.md":                      []byte("---\n{\"name\":\"documentation-auditor\",\"description\":\"Review docs.\",\"codex\":{\"model\":\"gpt-6-sol\",\"model_reasoning_effort\":\"high\",\"sandbox_mode\":\"read-only\"},\"claude\":{\"model\":\"sonnet\",\"effort\":\"high\",\"permissionMode\":\"plan\",\"tools\":[\"Read\"],\"disallowedTools\":[\"Write\"],\"maxTurns\":3}}\n---\n\nRead [documentation ownership](../rules/documentation.md) and [mechanism ownership](../rules/mechanisms.md).\n"),
		"docs/customers/README.md":                                          []byte("# Customers\n"),
		"docs/customers/septeo/README.md":                                   []byte("# Septeo\n"),
		"docs/customers/septeo/projects/README.md":                          []byte("# Projects\n"),
		"docs/customers/septeo/projects/wz-assist/README.md":                []byte("# Wz.Assist\n"),
		"docs/customers/septeo/projects/wz-assist/rules/customer-only.md":   []byte("Only for Wz.Assist.\n"),
		"docs/customers/septeo/projects/wz-assist/workflows/delivery.md":    []byte("Delivery workflow.\n"),
		"docs/customers/septeo/projects/wz-assist/skills/wz-assist-work.md": []byte("---\nname: wz-assist-work\ndescription: Work for Wz.Assist\n---\n\n# Work\nFollow the [Delivery Workflow](../workflows/delivery.md).\n"),
		"docs/customers/vortecx/README.md":                                  []byte("# Vortecx\n"),
		"docs/customers/vortecx/projects/engine/README.md":                  []byte("# Engine\n"),
	}}

	first, err := Cockpit(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Cockpit(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated plans differ")
	}
	if len(first) != 10 {
		t.Fatalf("got %d outputs, want 9 typed resources plus project: %v", len(first), outputNames(first))
	}
	project, err := format.Parse(cockpitProjectPath, first[cockpitProjectPath])
	if err != nil {
		t.Fatal(err)
	}
	if project.Spec.Profile != "cockpit" || len(project.Spec.Targets) != 0 || len(project.Spec.RuleAdapters) != 0 {
		t.Fatalf("migration transferred provider ownership: %#v", project.Spec)
	}
	resources := []*core.Resource{project,
		mustResource(t, first, "docs/general/rules/documentation.yaml"),
		mustResource(t, first, "docs/general/rules/mechanisms.yaml"),
		mustResource(t, first, "docs/general/workflows/author-mechanism.yaml"),
		mustResource(t, first, "docs/general/workflows/other.yaml"),
		mustResource(t, first, "docs/general/skills/author-mechanism.yaml"),
		mustResource(t, first, "docs/general/agents/documentation-auditor.yaml"),
		mustResource(t, first, "docs/customers/septeo/projects/wz-assist/rules/customer-only.yaml"),
		mustResource(t, first, "docs/customers/septeo/projects/wz-assist/workflows/delivery.yaml"),
		mustResource(t, first, "docs/customers/septeo/projects/wz-assist/skills/wz-assist-work.yaml"),
	}
	graph := core.Build(resources)
	if len(graph.Diagnostics) != 0 {
		t.Fatalf("planned graph has diagnostics: %#v", graph.Diagnostics)
	}
	agent := graph.Resources["cockpit-general/Agent/documentation-auditor"]
	if agent == nil || agent.Spec.Providers.Claude == nil || strings.Join(agent.Spec.Providers.Claude.DisallowedTools, ",") != "Write" || agent.Spec.Providers.Claude.MaxTurns != 3 || agent.Spec.Providers.Codex == nil || agent.Spec.Providers.Codex.Effort != "high" {
		t.Fatalf("agent provider metadata was not retained: %#v", agent)
	}
	if agent.Spec.Text != "\nRead [documentation ownership](../rules/documentation.md) and [mechanism ownership](../rules/mechanisms.md).\n" {
		t.Fatalf("agent body was not retained after metadata: %q", agent.Spec.Text)
	}
	if len(agent.Spec.Rules) != 2 || agent.Spec.Rules[0].Name != "documentation" || agent.Spec.Rules[1].Name != "mechanisms" {
		t.Fatalf("agent's explicit rules were not modeled as typed relationships: %#v", agent.Spec.Rules)
	}
	authorSkill := graph.Resources["cockpit-general/Skill/author-mechanism"]
	if authorSkill == nil || len(authorSkill.Spec.Uses) != 1 || authorSkill.Spec.Uses[0].Kind != "Workflow" || authorSkill.Spec.Uses[0].Name != "author-mechanism" {
		t.Fatalf("skill's named workflow was not modeled as a typed relationship: %#v", authorSkill)
	}
	if !strings.Contains(authorSkill.Spec.Text, "[another workflow](../workflows/other.md)") {
		t.Fatalf("unreviewed prose navigation link was not preserved: %q", authorSkill.Spec.Text)
	}
	wzSkill := graph.Resources["project-septeo-wz-assist/Skill/wz-assist-work"]
	if wzSkill == nil || len(wzSkill.Spec.Uses) != 1 || wzSkill.Spec.Uses[0].Namespace != "project-septeo-wz-assist" || wzSkill.Spec.Uses[0].Name != "delivery" {
		t.Fatalf("Wz.Assist skill's workflow was not kept in project scope: %#v", wzSkill)
	}
	wzArea := findArea(t, project.Spec.Areas, "project-septeo-wz-assist")
	engineArea := findArea(t, project.Spec.Areas, "project-vortecx-engine")
	if !hasRule(wzArea.Rules, "project-septeo-wz-assist", "customer-only") {
		t.Fatalf("Wz.Assist rule was not assigned to its owning scope: %#v", wzArea.Rules)
	}
	if hasRule(engineArea.Rules, "project-septeo-wz-assist", "customer-only") {
		t.Fatalf("Wz.Assist rule leaked into another customer project: %#v", engineArea.Rules)
	}
	for _, imported := range engineArea.Imports {
		if imported == "customer-septeo" || imported == "project-septeo-wz-assist" {
			t.Fatalf("Vortecx project imports another customer's scope %q", imported)
		}
	}
	if !hasRule(engineArea.Rules, "cockpit-general", "documentation") {
		t.Fatalf("General rules were not inherited into project scope: %#v", engineArea.Rules)
	}
	project.Spec.Targets = []string{"claude"}
	providerGraph := core.Build(resources)
	outputs, err := render.Generate(providerGraph)
	if err != nil {
		t.Fatal(err)
	}
	var globalRules []string
	for name := range outputs {
		if strings.HasPrefix(name, ".claude/rules/") {
			globalRules = append(globalRules, name)
		}
	}
	sort.Strings(globalRules)
	if strings.Join(globalRules, ",") != ".claude/rules/general-documentation.md,.claude/rules/general-mechanisms.md" {
		t.Fatalf("customer/project rules or inherited General rules were mis-scoped globally: %v", globalRules)
	}
}

func TestCockpitRejectsExistingOutputAndUnownedMechanisms(t *testing.T) {
	for name, files := range map[string]map[string][]byte{
		"existing resource": {
			"docs/general/README.md":          []byte("# General\n"),
			"docs/general/rules/writing.md":   []byte("Text.\n"),
			"docs/general/rules/writing.yaml": []byte("existing\n"),
		},
		"unsupported owner": {
			"docs/unmapped/rules/writing.md": []byte("Text.\n"),
		},
		"case insensitive output collision": {
			"docs/general/README.md":        []byte("# General\n"),
			"docs/general/rules/Writing.md": []byte("First.\n"),
			"docs/general/rules/writing.md": []byte("Second.\n"),
		},
		"unsafe source path": {
			"docs/../docs/general/rules/writing.md": []byte("Text.\n"),
		},
		"case insensitive project config collision": {
			"MARKITECT.YAML": []byte("existing project configuration\n"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Cockpit(&source.Snapshot{Files: files}); err == nil {
				t.Fatal("expected migration to reject unsafe plan")
			}
		})
	}
}

func TestCockpitDoesNotInferReviewedRelationshipWhenLinkIsRemoved(t *testing.T) {
	snapshot := &source.Snapshot{Files: map[string][]byte{
		"docs/general/workflows/author-mechanism.md": []byte("Workflow.\n"),
		"docs/general/skills/author-mechanism.md":    []byte("---\nname: author-mechanism\ndescription: Author mechanisms\n---\n\nThe workflow exists, but is not named here.\n"),
	}}
	definitions, err := collectCockpitDefinitions(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range definitions {
		if item.resource.Kind == "Skill" && len(item.resource.Spec.Uses) != 0 {
			t.Fatalf("migration inferred a dependency without a direct link: %#v", item.resource.Spec.Uses)
		}
	}
}

func mustResource(t *testing.T, outputs map[string][]byte, path string) *core.Resource {
	t.Helper()
	resource, err := format.Parse(path, outputs[path])
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return resource
}

func findArea(t *testing.T, areas []core.Area, name string) core.Area {
	t.Helper()
	for _, area := range areas {
		if area.Name == name {
			return area
		}
	}
	t.Fatalf("area %q not found", name)
	return core.Area{}
}

func hasRule(refs []core.Ref, namespace, name string) bool {
	for _, ref := range refs {
		if ref.Namespace == namespace && ref.Name == name {
			return true
		}
	}
	return false
}
