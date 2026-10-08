package host

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

func TestPerResourcePolicyImpactIsBoundedToSubjectAndDependents(t *testing.T) {
	constraints := []string{
		"- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}",
		"- name: module-count\n  select: {kind: Module}\n  assert: {op: count, scope: resource, relation: dependsOn, min: 1}",
		"- name: module-targets\n  select: {kind: Module}\n  assert: {op: allowed-targets, relation: dependsOn, values: [Core]}",
	}
	for i, constraint := range constraints {
		t.Run([]string{"equal", "resource-count", "allowed-targets"}[i], func(t *testing.T) {
			beforeSnapshot := policyImpactSnapshot(t, constraint, true, true, true)
			afterSnapshot := clonePolicySnapshot(beforeSnapshot)
			path := "resources/orders.yaml"
			data := string(afterSnapshot.Files[path])
			if i == 0 {
				data = strings.Replace(data, "intent: expected", "intent: changed", 1)
			} else if i == 1 {
				data = strings.Replace(data, "dependsOn: [{kind: Core, name: platform, namespace: engineering}]", "dependsOn: []", 1)
			} else {
				data = strings.Replace(data, "kind: Core", "kind: Module", 1)
				data = strings.Replace(data, "name: platform", "name: other", 1)
			}
			afterSnapshot.Files[path] = []byte(data)
			before, after := parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot)
			if len(before.Diagnostics) != 0 || len(after.Diagnostics) == 0 {
				t.Fatalf("expected per-resource violation after edit: before=%+v after=%+v", before.Diagnostics, after.Diagnostics)
			}
			impact := Changes(before, after)
			if !contains(impact.Affected, "engineering/engineering.markitect.org/v1alpha1/Module/orders") || !contains(impact.Affected, "engineering/Skill/add-order") {
				t.Fatalf("changed subject or its declared dependent omitted: %+v", impact)
			}
			if contains(impact.Affected, "engineering/Workflow/unrelated") || hasImpactCause(impact, "constraint-selection") {
				t.Fatalf("per-resource policy unexpectedly caused global impact: %+v", impact)
			}
		})
	}
}

func TestPerResourcePolicySelectorEntryAndExitStayBounded(t *testing.T) {
	constraint := "- name: governed-intent\n  select: {kind: Module, labels: {governed: yes}}\n  assert: {op: equal, field: intent, value: expected}"
	for _, entering := range []bool{false, true} {
		name, beforeLabel, afterLabel := "exit", "  labels: {governed: yes}\n", ""
		if entering {
			name, beforeLabel, afterLabel = "entry", "", "  labels: {governed: yes}\n"
		}
		t.Run(name, func(t *testing.T) {
			beforeSnapshot := policyImpactSnapshot(t, constraint, false, true, !entering)
			path := "resources/orders.yaml"
			afterSnapshot := clonePolicySnapshot(beforeSnapshot)
			data := string(afterSnapshot.Files[path])
			if beforeLabel != "" {
				data = strings.Replace(data, beforeLabel, "", 1)
			}
			if afterLabel != "" {
				data = strings.Replace(data, "  namespace: engineering\n", "  namespace: engineering\n"+afterLabel, 1)
			}
			afterSnapshot.Files[path] = []byte(data)
			impact := Changes(parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot))
			if !contains(impact.Affected, "engineering/engineering.markitect.org/v1alpha1/Module/orders") || contains(impact.Affected, "engineering/Workflow/unrelated") {
				t.Fatalf("per-resource selector change not bounded: %+v", impact)
			}
			if hasImpactCause(impact, "constraint-selection") {
				t.Fatalf("per-resource selector change mislabeled collection-wide: %+v", impact.Causes)
			}
		})
	}
}

func TestCollectionPolicySelectorChangeRemainsGlobal(t *testing.T) {
	constraint := "- name: module-inventory-count\n  select: {kind: Module, labels: {governed: yes}}\n  assert: {op: count, min: 1, max: 1}"
	beforeSnapshot := policyImpactSnapshot(t, constraint, false, false, false)
	afterSnapshot := clonePolicySnapshot(beforeSnapshot)
	path := "resources/orders.yaml"
	afterSnapshot.Files[path] = []byte(strings.Replace(string(afterSnapshot.Files[path]), "  namespace: engineering\n", "  namespace: engineering\n  labels: {governed: yes}\n", 1))
	impact := Changes(parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot))
	if !contains(impact.Affected, "engineering/Workflow/unrelated") || !hasImpactCause(impact, "constraint-selection") {
		t.Fatalf("selection-level count change should remain global and explain why: %+v", impact)
	}
	for _, cause := range impact.Causes {
		if cause.Kind == "constraint-selection" && cause.Resource != "" {
			t.Fatalf("global cause should have no singular resource: %+v", cause)
		}
	}
}

func TestSelectedLocalDomainChangeHasExplicitGlobalCause(t *testing.T) {
	beforeSnapshot := domainInputSnapshot()
	afterSnapshot := clonePolicySnapshot(beforeSnapshot)
	path := "domains/software.yaml"
	afterSnapshot.Files[path] = append([]byte("# selected language definition changed\n"), afterSnapshot.Files[path]...)
	impact := Changes(parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot))
	if !contains(impact.Affected, "engineering/software.markitect.io/v1alpha1/Module/survey") {
		t.Fatalf("selected Domain change should invalidate all project evidence: %+v", impact)
	}
	if !hasImpactCause(impact, "domain-definition") {
		t.Fatalf("selected Domain change lacks an explicit cause: %+v", impact.Causes)
	}
	for _, cause := range impact.Causes {
		if cause.Kind == "unowned-input" && cause.Path == path {
			t.Fatalf("selected Domain was mislabeled as unowned: %+v", impact.Causes)
		}
	}
}

func TestCollectionUniqueConstraintWithoutScopeRemainsGlobal(t *testing.T) {
	constraint := "- name: unique-governed-intent\n  select: {kind: Module, labels: {governed: yes}}\n  assert: {op: unique, field: intent}"
	beforeSnapshot := policyImpactSnapshot(t, constraint, false, false, false)
	otherPath := "resources/other.yaml"
	beforeSnapshot.Files[otherPath] = []byte(strings.Replace(string(beforeSnapshot.Files[otherPath]), "metadata: {name: other, namespace: engineering}", "metadata:\n  name: other\n  namespace: engineering\n  labels: {governed: yes}", 1))
	afterSnapshot := clonePolicySnapshot(beforeSnapshot)
	ordersPath := "resources/orders.yaml"
	afterSnapshot.Files[ordersPath] = []byte(strings.Replace(string(afterSnapshot.Files[ordersPath]), "  namespace: engineering\n", "  namespace: engineering\n  labels: {governed: yes}\n", 1))
	impact := Changes(parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot))
	if !contains(impact.Affected, "engineering/Workflow/unrelated") || !hasImpactCause(impact, "constraint-selection") {
		t.Fatalf("selection-level unique assertion should remain global: %+v", impact)
	}
}

func TestContextWithoutInvalidationGuardsPerResourcePolicyImpact(t *testing.T) {
	constraint := "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}"
	beforeSnapshot := policyImpactSnapshot(t, constraint, true, false, false)
	afterSnapshot := clonePolicySnapshot(beforeSnapshot)
	path := "resources/orders.yaml"
	afterSnapshot.Files[path] = []byte(strings.Replace(string(afterSnapshot.Files[path]), "intent: expected", "intent: changed", 1))
	impact := Changes(parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot))
	if !contains(impact.Affected, "engineering/Workflow/unrelated") || !hasImpactCause(impact, "context-policy-effect") {
		t.Fatalf("context-visible result without invalidation was not kept conservative: %+v", impact)
	}
}

func TestImpactCausesNameReverseEdgeAndUnknownInputFallback(t *testing.T) {
	constraint := "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}"
	beforeSnapshot := policyImpactSnapshot(t, constraint, true, true, false)
	afterSnapshot := clonePolicySnapshot(beforeSnapshot)
	path := "resources/orders.yaml"
	afterSnapshot.Files[path] = []byte(strings.Replace(string(afterSnapshot.Files[path]), "intent: expected", "intent: changed", 1))
	impact := Changes(parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot))
	found := false
	for _, cause := range impact.Causes {
		if cause.Kind == "invalidation" && cause.From == "engineering/Skill/add-order" && cause.To == "engineering/engineering.markitect.org/v1alpha1/Module/orders" && cause.Relation == "uses" && (cause.Snapshot == "base" || cause.Snapshot == "candidate") {
			found = true
		}
	}
	if !found {
		t.Fatalf("named reverse invalidation edge missing: %+v", impact.Causes)
	}

	unknownBefore, unknownAfter := clonePolicySnapshot(beforeSnapshot), clonePolicySnapshot(beforeSnapshot)
	unknownBefore.Files["README.md"], unknownAfter.Files["README.md"] = []byte("old"), []byte("new")
	global := Changes(parsePolicyImpactProject(t, unknownBefore), parsePolicyImpactProject(t, unknownAfter))
	if !contains(global.Affected, "engineering/Workflow/unrelated") || !hasImpactCause(global, "unowned-input") {
		t.Fatalf("unowned input no longer causes explained global impact: %+v", global)
	}
	for _, cause := range global.Causes {
		if cause.Kind == "unowned-input" && cause.Resource != "" {
			t.Fatalf("global cause should not name one resource: %+v", cause)
		}
	}
}

func policyImpactSnapshot(t *testing.T, constraints string, relationContext, relationInvalidate, labelOrders bool) *snapshot.Snapshot {
	t.Helper()
	contextValue, invalidateValue, labels := "false", "false", ""
	if relationContext {
		contextValue = "true"
	}
	if relationInvalidate {
		invalidateValue = "true"
	}
	if labelOrders {
		labels = "  labels: {governed: yes}\n"
	}
	domain := "apiVersion: markitect.example.org/v1alpha1\nkind: Domain\nmetadata: {name: engineering}\nspec:\n  apiVersion: engineering.markitect.org/v1alpha1\n  kinds:\n    Core:\n      required: [purpose]\n      properties: {purpose: {type: string}}\n    Module:\n      required: [intent]\n      properties:\n        intent: {type: string}\n        dependsOn: {type: array, items: {type: ref}}\n  relations:\n    dependsOn:\n      field: dependsOn\n      sourceKinds: [Module]\n      targetKinds: [Core, Module]\n      context: " + contextValue + "\n      invalidate: " + invalidateValue + "\n  constraints:\n    " + strings.ReplaceAll(strings.TrimSpace(constraints), "\n", "\n    ") + "\n"
	project := "apiVersion: markitect.example.org/v1alpha1\nkind: Project\nmetadata: {name: policy-impact}\nspec:\n  domains: [domains/engineering.yaml]\n  areas: [{name: engineering, path: resources}]\n"
	module := "apiVersion: engineering.markitect.org/v1alpha1\nkind: Module\nmetadata:\n  name: orders\n  namespace: engineering\n" + labels + "spec:\n  intent: expected\n  dependsOn: [{kind: Core, name: platform, namespace: engineering}]\n"
	coreResource := "apiVersion: engineering.markitect.org/v1alpha1\nkind: Core\nmetadata: {name: platform, namespace: engineering}\nspec: {purpose: Shared services.}\n"
	otherModule := "apiVersion: engineering.markitect.org/v1alpha1\nkind: Module\nmetadata: {name: other, namespace: engineering}\nspec:\n  intent: expected\n  dependsOn: [{kind: Core, name: platform, namespace: engineering}]\n"
	skill := "apiVersion: markitect.example.org/v1alpha1\nkind: Skill\nmetadata: {name: add-order, namespace: engineering}\nspec:\n  text: Add an order.\n  uses: [{apiVersion: engineering.markitect.org/v1alpha1, kind: Module, name: orders, namespace: engineering}]\n"
	unrelated := "apiVersion: markitect.example.org/v1alpha1\nkind: Workflow\nmetadata: {name: unrelated, namespace: engineering}\nspec: {text: Independent task.}\n"
	return &snapshot.Snapshot{ID: "policy-impact", Provisional: true, Files: map[string][]byte{
		"markitect.yaml": []byte(project), "domains/engineering.yaml": []byte(domain),
		"resources/orders.yaml": []byte(module), "resources/platform.yaml": []byte(coreResource),
		"resources/other.yaml": []byte(otherModule), "resources/add-order.skill.yaml": []byte(skill),
		"resources/unrelated.workflow.yaml": []byte(unrelated),
	}, Modes: map[string]string{}}
}

func parsePolicyImpactProject(t *testing.T, s *snapshot.Snapshot) *Project {
	t.Helper()
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func clonePolicySnapshot(s *snapshot.Snapshot) *snapshot.Snapshot {
	clone := &snapshot.Snapshot{ID: s.ID, Provisional: s.Provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	for name, data := range s.Files {
		clone.Files[name] = append([]byte(nil), data...)
	}
	for name, mode := range s.Modes {
		clone.Modes[name] = mode
	}
	return clone
}

func hasImpactCause(impact *Impact, kind string) bool {
	for _, cause := range impact.Causes {
		if cause.Kind == kind {
			return true
		}
	}
	return false
}
