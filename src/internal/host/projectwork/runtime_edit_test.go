package projectwork

import (
	"strings"
	"testing"
)

func TestRuntimeConfigEditUserPlanApplyAndModelIsolation(t *testing.T) {
	root, _ := testProject(t)
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	modelDigest := project.Model.Digest
	content := "mode: intentionally-not-a-runtime-mode\n"
	mutation := Mutation{APIVersion: APIVersion, BaseDigest: project.Digest, Actor: HumanActor, Goal: "Configure the project runtime", Files: []FileChange{{Path: RuntimePath, Content: content}}}
	plan, err := PlanEdit(project, mutation)
	if err != nil {
		t.Fatalf("syntactically valid runtime edit should be reviewable: %v", err)
	}
	if len(plan.Mutation.Files) != 1 || plan.Mutation.Files[0].Path != RuntimePath {
		t.Fatalf("runtime preview changed unexpected paths: %+v", plan.Mutation.Files)
	}
	if plan.Report.Digest != project.Report.Digest || plan.CandidateDigest == project.Digest {
		t.Fatalf("runtime edit should bind a new Project digest without changing model report: report %s -> %s, digest %s -> %s", project.Report.Digest, plan.Report.Digest, project.Digest, plan.CandidateDigest)
	}
	if _, err := ApplyEdit(root, plan, plan.BaseDigest); err != nil {
		t.Fatalf("apply runtime edit: %v", err)
	}
	updated, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(updated.Snapshot.Files[RuntimePath]); got != content {
		t.Fatalf("runtime bytes = %q, want %q", got, content)
	}
	if updated.Model.Digest != modelDigest {
		t.Fatalf("runtime-only edit changed model digest: %s -> %s", modelDigest, updated.Model.Digest)
	}
}

func TestRuntimeConfigEditAllowsOnlyUserOrRootManager(t *testing.T) {
	root, _ := testProject(t)
	childPath := ".markitect/model/orders/manager.yaml"
	child := "apiVersion: " + APIVersion + "\nkind: Manager\nmetadata:\n  name: orders\n  namespace: orders\npurpose: Owns order responsibilities.\nspec:\n  parent:\n    apiVersion: " + APIVersion + "\n    kind: Manager\n    namespace: \"\"\n    name: project-owner\n  owns: [src/]\n"
	writeFile(t, root, childPath, child)
	writeFile(t, root, ManifestPath, projectConfig(initManagerPath, ".markitect/model/statement.yaml", childPath))
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	var childID string
	for _, manager := range project.Report.Managers {
		if manager.Namespace == "orders" {
			childID = manager.ID
		}
	}
	if childID == "" {
		t.Fatalf("child manager missing from report: %+v", project.Report.Managers)
	}
	for _, actor := range []string{rootManagerID(), childID} {
		_, err := PlanEdit(project, Mutation{APIVersion: APIVersion, BaseDigest: project.Digest, Actor: actor, Goal: "Configure runtime", Files: []FileChange{{Path: RuntimePath, Content: "mode: controlled-local\n"}}})
		if actor == childID {
			if err == nil || !strings.Contains(err.Error(), "active root Manager") {
				t.Fatalf("child Manager runtime edit error = %v", err)
			}
		} else if err != nil {
			t.Fatalf("root Manager runtime edit rejected: %v", err)
		}
	}
}

func TestRuntimeConfigEditRejectsDeleteStaleAndAmbiguousYAML(t *testing.T) {
	root, _ := testProject(t)
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	base := Mutation{APIVersion: APIVersion, BaseDigest: project.Digest, Actor: HumanActor, Goal: "Configure runtime"}
	deleted := base
	deleted.Files = []FileChange{{Path: RuntimePath, Delete: true}}
	if _, err := PlanEdit(project, deleted); err == nil || !strings.Contains(err.Error(), "only model files may be deleted") {
		t.Fatalf("runtime deletion error = %v", err)
	}
	for name, content := range map[string]string{
		"scalar root":      "runtime\n",
		"sequence root":    "[one, two]\n",
		"duplicate key":    "mode: one\nmode: two\n",
		"alias":            "base: &base {mode: one}\ncopy: *base\n",
		"anchor":           "base: &base {mode: one}\n",
		"extra document":   "mode: one\n---\nmode: two\n",
		"invalid syntax":   "mode: [\n",
		"oversize content": "value: " + strings.Repeat("x", (1<<20)+1) + "\n",
	} {
		mutation := base
		mutation.Files = []FileChange{{Path: RuntimePath, Content: content}}
		if _, err := PlanEdit(project, mutation); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
	mutation := base
	mutation.Files = []FileChange{{Path: RuntimePath, Content: "mode: controlled-local\n"}}
	plan, err := PlanEdit(project, mutation)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, RuntimePath, "mode: changed-after-review\n")
	if _, err := ApplyEdit(root, plan, plan.BaseDigest); err == nil || !strings.Contains(err.Error(), "changed after planning") {
		t.Fatalf("stale runtime edit error = %v", err)
	}
}
