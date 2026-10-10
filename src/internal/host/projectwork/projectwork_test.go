package projectwork

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestInitPreviewAndGuardedWriteCreateControlPlaneAndDocumentation(t *testing.T) {
	root := testGitRoot(t)
	preview, err := Init(root, "New project", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Files) != 5 {
		t.Fatalf("preview planned %d files, want 5", len(preview.Files))
	}
	for _, file := range preview.Files {
		if !strings.HasPrefix(file.Path, ".markitect/") && file.Path != DefaultDocumentPath {
			t.Fatalf("Init planned a path outside .markitect: %s", file.Path)
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file.Path))); !os.IsNotExist(err) {
			t.Fatalf("preview wrote target %s: %v", file.Path, err)
		}
	}
	written, err := Init(root, "New project", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(written.Written) != 5 {
		t.Fatalf("Init wrote %d paths, want 5: %v", len(written.Written), written.Written)
	}
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Report.Managers) != 1 || project.Report.Managers[0].Namespace != "" || len(project.Report.Statements) != 0 {
		t.Fatalf("Init invented domain definitions: %+v", project.Report)
	}
	if project.Config.WorkflowMode != WorkflowModeGuided || project.Config.AcceptancePolicy != AcceptancePolicyCommittedModel {
		t.Fatalf("Init defaults = workflow %q, acceptance %q", project.Config.WorkflowMode, project.Config.AcceptancePolicy)
	}
	document, err := Document(project, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(document, "does not establish that repository code") {
		t.Fatal("generated document omitted the code-conformance limitation")
	}
	if _, err := os.Stat(filepath.Join(root, ".artifacts")); !os.IsNotExist(err) {
		t.Fatalf("project initialization or document write created legacy Host state: %v", err)
	}
}

func TestWorkflowAndAcceptanceConfigModesAreClosedAndLegacyCompatible(t *testing.T) {
	legacy, err := DecodeConfig([]byte(projectConfig(initManagerPath)))
	if err != nil {
		t.Fatal(err)
	}
	if legacy.WorkflowMode != "" || legacy.AcceptancePolicy != "" {
		t.Fatalf("legacy omitted fields were not preserved as compatibility defaults: %+v", legacy)
	}
	for _, mode := range []string{WorkflowModeGuided, WorkflowModeEmpty} {
		data := strings.Replace(projectConfig(initManagerPath), "name: Fixture\n", "name: Fixture\nworkflowMode: "+mode+"\n", 1)
		config, err := DecodeConfig([]byte(data))
		if err != nil || config.WorkflowMode != mode {
			t.Errorf("workflowMode %q decode = %q, %v", mode, config.WorkflowMode, err)
		}
	}
	committed := strings.Replace(projectConfig(initManagerPath), "name: Fixture\n", "name: Fixture\nacceptancePolicy: committed-model\n", 1)
	if config, err := DecodeConfig([]byte(committed)); err != nil || config.AcceptancePolicy != AcceptancePolicyCommittedModel {
		t.Fatalf("committed-model acceptance policy decode = %+v, %v", config, err)
	}
	for field, value := range map[string]string{"workflowMode": "automatic", "acceptancePolicy": "drafts-accepted"} {
		data := strings.Replace(projectConfig(initManagerPath), "name: Fixture\n", "name: Fixture\n"+field+": "+value+"\n", 1)
		if _, err := DecodeConfig([]byte(data)); err == nil {
			t.Errorf("invalid %s %q was accepted", field, value)
		}
	}
}

func TestTransitionalExclusionsRequireSafeNonoverlappingSelectors(t *testing.T) {
	base := projectConfig(initManagerPath)
	for name, list := range map[string]string{
		"repository root":   "- path: .\n  reason: legacy\n",
		"traversal":         "- path: ../outside\n  reason: legacy\n",
		"Markitect state":   "- path: .markitect/state/\n  reason: legacy\n",
		"reserved Git path": "- path: .git/config\n  reason: legacy\n",
		"overlap":           "- path: legacy/\n  reason: parent\n- path: legacy/child.txt\n  reason: child\n",
		"ordinary overlap":  "- path: src/file.go\n  reason: migration\n",
		"model overlap":     "- path: .markitect/model/manager.yaml\n  reason: migration\n",
	} {
		data := base
		if name == "ordinary overlap" {
			data = strings.Replace(data, "exclusions: []\n", "exclusions:\n  - path: src/file.go\n    reason: excluded\n", 1)
		} else if name == "model overlap" {
			list = "- path: .markitect/model/\n  reason: migration\n"
		}
		data += "transitionalExclusions:\n" + list
		if _, err := DecodeConfig([]byte(data)); err == nil {
			t.Errorf("%s transitional selector was accepted", name)
		}
	}
	valid := base + "transitionalExclusions:\n  - path: legacy/generated/\n    reason: Migration is incomplete\n"
	config, err := DecodeConfig([]byte(valid))
	if err != nil || len(config.TransitionalExclusions) != 1 || config.TransitionalExclusions[0].Path != "legacy/generated/" {
		t.Fatalf("valid transitional directory selector decode = %+v, %v", config, err)
	}
}

func TestLoadSelectsConfiguredInventoryAndExcludesCache(t *testing.T) {
	root, commit := testProject(t)
	cache := filepath.Join(root, filepath.FromSlash(".markitect/cache/compiled.json"))
	if err := os.MkdirAll(filepath.Dir(cache), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, []byte("cache"), 0644); err != nil {
		t.Fatal(err)
	}
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if project.Snapshot.Files["src/unmodeled.txt"] == nil || len(project.Report.Files) == 0 || project.Report.Files[0].Path != "src/unmodeled.txt" {
		t.Fatalf("selected unknown inventory was not surfaced: %+v", project.Report)
	}
	if _, exists := project.Snapshot.Files[".markitect/cache/compiled.json"]; exists {
		t.Fatal("cache entered selected input snapshot")
	}
	fixed, err := Load(root, commit)
	if err != nil {
		t.Fatal(err)
	}
	if fixed.Provisional || fixed.Revision != commit {
		t.Fatalf("fixed snapshot identity lost: %+v", fixed)
	}
}

func TestLoadModelWithoutRuntimeForWorkingAndFixedSnapshots(t *testing.T) {
	root, initialRevision := testProject(t)
	withRuntime, err := Load(root, initialRevision)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := withRuntime.Snapshot.Files[RuntimePath]; !ok {
		t.Fatal("initial fixed snapshot did not include the present runtime file")
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(RuntimePath))); err != nil {
		t.Fatal(err)
	}
	working, err := Load(root, "")
	if err != nil {
		t.Fatalf("working model load without runtime failed: %v", err)
	}
	if _, ok := working.Snapshot.Files[RuntimePath]; ok {
		t.Fatal("working snapshot bound a runtime file that is absent")
	}
	if working.Digest == withRuntime.Digest {
		t.Fatal("runtime presence change did not alter the selected model digest")
	}
	if _, err := projectmodel.Context(working.Report, working.Report.Managers[0].ID); err != nil {
		t.Fatalf("model context required runtime configuration: %v", err)
	}
	gitTest(t, root, "add", "-u", RuntimePath)
	gitTest(t, root, "commit", "-m", "remove optional runtime configuration")
	withoutRuntimeRevision := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	fixed, err := Load(root, withoutRuntimeRevision)
	if err != nil {
		t.Fatalf("fixed model load without runtime failed: %v", err)
	}
	if fixed.Provisional || fixed.Revision != withoutRuntimeRevision {
		t.Fatalf("fixed model identity changed: %+v", fixed)
	}
	if _, ok := fixed.Snapshot.Files[RuntimePath]; ok {
		t.Fatal("fixed snapshot bound a runtime file that is absent")
	}
	if _, err := projectmodel.Context(fixed.Report, fixed.Report.Managers[0].ID); err != nil {
		t.Fatalf("fixed model context required runtime configuration: %v", err)
	}
	repeated, err := Load(root, withoutRuntimeRevision)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Digest != fixed.Digest || repeated.Snapshot.Digest() != fixed.Snapshot.Digest() {
		t.Fatalf("fixed absence binding was nondeterministic: %s/%s versus %s/%s", fixed.Digest, fixed.Snapshot.Digest(), repeated.Digest, repeated.Snapshot.Digest())
	}
}

func TestLoadFullCoverageModelWithoutRuntimeForWorkingAndFixedSnapshots(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "No runtime fixture", true); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "initialize full-coverage model")
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(RuntimePath))); err != nil {
		t.Fatal(err)
	}
	working, err := Load(root, "")
	if err != nil {
		t.Fatalf("working full-coverage model load without runtime failed: %v", err)
	}
	if _, ok := working.Snapshot.Files[RuntimePath]; ok {
		t.Fatal("working full-coverage snapshot bound an absent runtime")
	}
	if _, err := projectmodel.Context(working.Report, working.Report.Managers[0].ID); err != nil {
		t.Fatalf("working full-coverage context required runtime configuration: %v", err)
	}
	gitTest(t, root, "add", "-u", RuntimePath)
	gitTest(t, root, "commit", "-m", "remove runtime from full-coverage project")
	revision := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	fixed, err := Load(root, revision)
	if err != nil {
		t.Fatalf("fixed full-coverage model load without runtime failed: %v", err)
	}
	if _, ok := fixed.Snapshot.Files[RuntimePath]; ok {
		t.Fatal("fixed full-coverage snapshot bound an absent runtime")
	}
	if _, err := projectmodel.Context(fixed.Report, fixed.Report.Managers[0].ID); err != nil {
		t.Fatalf("fixed full-coverage context required runtime configuration: %v", err)
	}
}

func TestPlanAndApplyEditBindCandidateAndRejectStaleOrSelfAuthority(t *testing.T) {
	root, _ := testProject(t)
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	path := ".markitect/model/statement.yaml"
	mutation := Mutation{APIVersion: APIVersion, BaseDigest: project.Digest, Actor: rootManagerID(), Goal: "Clarify the project goal", Files: []FileChange{{Path: path, Content: statementDefinition("A clarified project goal.")}}}
	plan, err := PlanEdit(project, mutation)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Digest == "" || plan.CandidateDigest == "" || plan.Impact.Digest == "" {
		t.Fatalf("plan lacks digest bindings: %+v", plan)
	}
	if _, err := ApplyEdit(root, plan, project.Digest); err != nil {
		t.Fatal(err)
	}
	updated, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Digest == project.Digest || !strings.Contains(string(updated.Snapshot.Files[path]), "A clarified project goal.") {
		t.Fatal("guarded edit did not produce the planned candidate")
	}

	root, _ = testProject(t)
	project, err = Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	manager := strings.Replace(managerDefinition(), "    - .", "    - src/", 1)
	_, err = PlanEdit(project, Mutation{APIVersion: APIVersion, BaseDigest: project.Digest, Actor: rootManagerID(), Goal: "Expand own authority", Files: []FileChange{{Path: initManagerPath, Content: manager}}})
	if err == nil || !strings.Contains(err.Error(), "cannot change the active mandate") {
		t.Fatalf("self-authorized manager change error = %v", err)
	}

	plan, err = PlanEdit(project, mutationFor(project, path, "Planned change."))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "src/unmodeled.txt", "manual selected inventory edit\n")
	if _, err := ApplyEdit(root, plan, project.Digest); err == nil || !strings.Contains(err.Error(), "changed after planning") {
		t.Fatalf("stale plan error = %v", err)
	}
}

func TestFixedHEADPlanCanApplyOnlyWhileWorkingInputsMatch(t *testing.T) {
	root, commit := testProject(t)
	project, err := Load(root, commit)
	if err != nil {
		t.Fatal(err)
	}
	path := ".markitect/model/statement.yaml"
	plan, err := PlanEdit(project, mutationFor(project, path, "Fixed revision edit."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyEdit(root, plan, project.Digest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Fixed revision edit.") {
		t.Fatalf("fixed revision plan was not applied: %s", got)
	}
}

func TestUserCanSelectNewInventoryThroughReviewedManifestEdit(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "New project", true); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "src/entry.go", "package src\n")
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	config := strings.Replace(projectConfig(initManagerPath), "name: Fixture", "name: New project", 1)
	mutation := Mutation{APIVersion: APIVersion, BaseDigest: project.Digest, Actor: HumanActor, Goal: "Select the source inventory", Files: []FileChange{{Path: ManifestPath, Content: config}}}
	plan, err := PlanEdit(project, mutation)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Report.Files) != 1 || plan.Report.Files[0].Path != "src/entry.go" {
		t.Fatalf("candidate report did not bind newly selected inventory: %+v", plan.Report.Files)
	}
	writeFile(t, root, "src/entry.go", "package src // changed after preview\n")
	if _, err := ApplyEdit(root, plan, project.Digest); err == nil {
		t.Fatalf("changed newly selected file should stale its plan: %v", err)
	}
	project, err = Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	mutation.BaseDigest = project.Digest
	plan, err = PlanEdit(project, mutation)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "src/added.go", "package src\n")
	if _, err := ApplyEdit(root, plan, project.Digest); err == nil {
		t.Fatalf("new member of selected inventory should stale its plan: %v", err)
	}
	project, err = Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	mutation.BaseDigest = project.Digest
	plan, err = PlanEdit(project, mutation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyEdit(root, plan, project.Digest); err != nil {
		t.Fatal(err)
	}
	updated, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Report.Files) != 2 || updated.Report.Files[0].Path != "src/added.go" || updated.Report.Files[1].Path != "src/entry.go" {
		t.Fatalf("selected inventory was not installed: %+v", updated.Report.Files)
	}
	rendered, err := Document(updated, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "unmodeled: no Artifact path declared") {
		t.Fatal("document did not distinguish observed files from Artifact coverage")
	}
}

func TestNonRootManagerCannotChangeInventoryScope(t *testing.T) {
	root, _ := testProject(t)
	childPath := ".markitect/model/orders/manager.yaml"
	child := "apiVersion: " + APIVersion + "\nkind: Manager\nmetadata:\n  name: orders\n  namespace: orders\npurpose: Owns order responsibilities.\nspec:\n  parent:\n    apiVersion: " + APIVersion + "\n    kind: Manager\n    namespace: \"\"\n    name: project-owner\n  owns: [src/]\n"
	writeFile(t, root, childPath, child)
	config := strings.Replace(projectConfig(".markitect/model/manager.yaml", ".markitect/model/statement.yaml", childPath), "inventoryRoots:\n  - src\n", "inventoryRoots:\n  - src\n", 1)
	writeFile(t, root, ManifestPath, config)
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
		t.Fatalf("child Manager missing from report: %+v", project.Report.Managers)
	}
	changedScope := strings.Replace(config, "inventoryRoots:\n  - src\n", "inventoryRoots:\n  - docs\n", 1)
	_, err = PlanEdit(project, Mutation{APIVersion: APIVersion, BaseDigest: project.Digest, Actor: childID, Goal: "Change inventory boundary", Files: []FileChange{{Path: ManifestPath, Content: changedScope}}})
	if err == nil || !strings.Contains(err.Error(), "active root Manager") {
		t.Fatalf("non-root Manager inventory scope error = %v", err)
	}
}

func TestPlanEditRejectsNamespaceMismatchPathEscapeAndClosedInputAliases(t *testing.T) {
	root, _ := testProject(t)
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = PlanEdit(project, Mutation{APIVersion: APIVersion, BaseDigest: project.Digest, Actor: HumanActor, Goal: "Escape", Files: []FileChange{{Path: ".markitect/model/../outside.yaml", Content: "x"}}})
	if err == nil || !strings.Contains(err.Error(), "normalized repository-relative") {
		t.Fatalf("path escape error = %v", err)
	}
	s := mapSnapshot(map[string]string{
		ManifestPath:                             "apiVersion: " + APIVersion + "\nname: Fixture\nmodelFiles: [.markitect/model/orders/statement.yaml]\ninventoryRoots: [src]\nexclusions: []\n",
		RuntimePath:                              "{}\n",
		initManagerPath:                          managerDefinition(),
		".markitect/model/orders/statement.yaml": statementDefinition("Wrong namespace."),
	})
	if _, err := FromSnapshot(root, s); err == nil || !strings.Contains(err.Error(), "requires \"orders\"") {
		t.Fatalf("namespace mismatch error = %v", err)
	}
	alias := []byte("apiVersion: " + APIVersion + "\nname: &project Fixture\nmodelFiles: [.markitect/model/manager.yaml]\ninventoryRoots: []\nexclusions: []\n")
	if _, err := DecodeConfig(alias); err == nil || !strings.Contains(err.Error(), "aliases and anchors") {
		t.Fatalf("YAML alias error = %v", err)
	}
	duplicate := []byte("apiVersion: " + APIVersion + "\nname: Fixture\nname: Duplicate\nmodelFiles: [.markitect/model/manager.yaml]\ninventoryRoots: []\nexclusions: []\n")
	if _, err := DecodeConfig(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("duplicate YAML key error = %v", err)
	}
}

func TestConfigRejectsCaseAliasedScopesAndReservedRuntimeRoots(t *testing.T) {
	for _, roots := range []string{"[src, SRC/generated]", "[.MARKITECT/cache]"} {
		data := []byte("apiVersion: " + APIVersion + "\nname: Fixture\nmodelFiles: [.markitect/model/manager.yaml]\ninventoryRoots: " + roots + "\nexclusions: []\n")
		if _, err := DecodeConfig(data); err == nil {
			t.Errorf("inventory roots %s should be rejected", roots)
		}
	}
	wrongCase := []byte("ApiVersion: " + APIVersion + "\nname: Fixture\nmodelFiles: [.markitect/model/manager.yaml]\ninventoryRoots: []\nexclusions: []\n")
	if _, err := DecodeConfig(wrongCase); err == nil {
		t.Fatal("incorrectly cased YAML field should be rejected")
	}
}

func TestConfigRejectsDocumentControlPathCollisionsAndLinksFromCustomPath(t *testing.T) {
	for _, destination := range []string{
		"../outside.md",
		"AGENTS.md",
		"CLAUDE.md",
		".markitect/runtime.yaml",
		".markitect/ignore.yaml",
		".markitect/workflows/model-first.md",
	} {
		data := []byte("apiVersion: " + APIVersion + "\nname: Fixture\ndocumentPath: " + destination + "\nmodelFiles: [.markitect/model/manager.yaml]\ninventoryRoots: []\nexclusions: []\n")
		if _, err := DecodeConfig(data); err == nil {
			t.Errorf("documentPath %q should be rejected", destination)
		}
	}
	config, err := DecodeConfig([]byte("apiVersion: " + APIVersion + "\nname: Fixture\ndocumentPath: docs/generated/overview.md\nmodelFiles: [.markitect/model/manager.yaml]\ninventoryRoots: []\nexclusions: []\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sourceLinkAt(DocumentPath(config), "src/feature/file.go"), "[src/feature/file.go](../../src/feature/file.go)"; got != want {
		t.Fatalf("custom document source link = %q, want %q", got, want)
	}
}

func TestArtifactPathLinksAreRelativeToConfiguredDocumentPath(t *testing.T) {
	project := &Project{
		Config: Config{Name: "Fixture", CoverageMode: "full", DocumentPath: "ARCHITECTURE.md"},
		Report: projectmodel.Report{
			Statements: []projectmodel.Statement{{ID: "s", Name: "Goal", Source: ".markitect/model/goal.yaml", Description: "Goal."}},
			Artifacts:  []projectmodel.Artifact{{ID: "a", Name: "Main", Paths: []string{"src/main.go"}}},
		},
	}
	text := documentText(project)
	if !strings.Contains(text, "- Source: [.markitect/model/goal.yaml](.markitect/model/goal.yaml)") {
		t.Fatalf("statement source link is not relative to ARCHITECTURE.md:\n%s", text)
	}
	const want = "- Expected paths: [src/main.go](src/main.go)"
	if !strings.Contains(text, want+"\n") {
		got := "missing"
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "- Expected paths:") {
				got = line
			}
		}
		t.Fatalf("artifact path line = %q, want %q", got, want)
	}
}

func TestSourceLinksAreCaseSensitiveOnEveryPlatform(t *testing.T) {
	for _, test := range []struct{ destination, value, want string }{
		{"docs/markitect/project.md", "Docs/guide.md", "[Docs/guide.md](../../Docs/guide.md)"},
		{"docs/markitect/project.md", "docs/Markitect/guide.md", "[docs/Markitect/guide.md](../Markitect/guide.md)"},
		{"docs/markitect/project.md", "docs/markitect/guide.md", "[docs/markitect/guide.md](guide.md)"},
		{"docs/markitect/project.md", "docs", "[docs](..)"},
		{"ARCHITECTURE.md", "src/my file.go", "[src/my file.go](src/my%20file.go)"},
		{"docs/markitect/project.md", "../outside.md", "../outside.md"},
	} {
		if got := sourceLinkAt(test.destination, test.value); got != test.want {
			t.Errorf("sourceLinkAt(%q, %q) = %q, want %q", test.destination, test.value, got, test.want)
		}
	}
}

func TestConfigDoesNotReserveRemovedRouterDocumentPath(t *testing.T) {
	for _, destination := range []string{
		".agents/skills/markitect-model-first/SKILL.md",
		".claude/skills/markitect-model-first/SKILL.md",
	} {
		data := []byte("apiVersion: " + APIVersion + "\nname: Fixture\ndocumentPath: " + destination + "\nmodelFiles: [.markitect/model/manager.yaml]\ninventoryRoots: []\nexclusions: []\n")
		if _, err := DecodeConfig(data); err != nil {
			t.Errorf("removed router path %q remains reserved: %v", destination, err)
		}
	}
}

func TestDecodeMutationRejectsDuplicateAndUnknownJSONFields(t *testing.T) {
	duplicate := []byte("{\"apiVersion\":\"" + APIVersion + "\",\"apiVersion\":\"" + APIVersion + "\",\"baseDigest\":\"x\",\"actor\":\"user\",\"goal\":\"x\",\"files\":[]}")
	if _, err := DecodeMutation(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("duplicate JSON field error = %v", err)
	}
	unknown := map[string]any{"apiVersion": APIVersion, "baseDigest": "x", "actor": "user", "goal": "x", "files": []any{}, "authority": "root"}
	data, _ := json.Marshal(unknown)
	if _, err := DecodeMutation(data); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown JSON field error = %v", err)
	}
	for name, raw := range map[string]string{
		"case alias":            `{"apiVersion":"` + APIVersion + `","baseDigest":"x","actor":"user","Actor":"user","goal":"x","files":[]}`,
		"wrong root spelling":   `{"ApiVersion":"` + APIVersion + `","baseDigest":"x","actor":"user","goal":"x","files":[]}`,
		"wrong nested spelling": `{"apiVersion":"` + APIVersion + `","baseDigest":"x","actor":"user","goal":"x","files":[{"Path":".markitect/model/statement.yaml","content":"x"}]}`,
	} {
		if _, err := DecodeMutation([]byte(raw)); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func testProject(t *testing.T) (string, string) {
	t.Helper()
	root := testGitRoot(t)
	if err := os.MkdirAll(filepath.Join(root, ".markitect/model"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ManifestPath, projectConfig(initManagerPath, ".markitect/model/statement.yaml"))
	writeFile(t, root, RuntimePath, "{}\n")
	writeFile(t, root, initManagerPath, managerDefinition())
	writeFile(t, root, ".markitect/model/statement.yaml", statementDefinition("The project has a goal."))
	writeFile(t, root, "src/unmodeled.txt", "observed but unassigned\n")
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "project fixture")
	return root, strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
}

func testGitRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitTest(t, root, "init", "-b", "codex/projectwork-test")
	gitTest(t, root, "config", "user.email", "projectwork@example.test")
	gitTest(t, root, "config", "user.name", "Projectwork Test")
	writeFile(t, root, "README.md", "fixture\n")
	gitTest(t, root, "add", "README.md")
	gitTest(t, root, "commit", "-m", "initial")
	return root
}

func managerDefinition() string {
	return "apiVersion: " + APIVersion + "\nkind: Manager\nmetadata:\n  name: project-owner\n  namespace: \"\"\npurpose: Owns the project.\nspec:\n  owns:\n    - .\n"
}

func statementDefinition(description string) string {
	encoded, _ := json.Marshal(description)
	return "apiVersion: " + APIVersion + "\nkind: Statement\nmetadata:\n  name: project-goal\n  namespace: \"\"\npurpose: Project goal.\nspec:\n  category: concept\n  description: " + string(encoded) + "\n"
}

func projectConfig(files ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: %s\nname: Fixture\nmodelFiles:\n", APIVersion)
	for _, file := range files {
		fmt.Fprintf(&b, "  - %s\n", file)
	}
	b.WriteString("inventoryRoots:\n  - src\nexclusions: []\n")
	return b.String()
}

func rootManagerID() string {
	encoded, _ := json.Marshal([]string{APIVersion, "Manager", "", "project-owner"})
	return string(encoded)
}

func mutationFor(project *Project, file, description string) Mutation {
	return Mutation{APIVersion: APIVersion, BaseDigest: project.Digest, Actor: rootManagerID(), Goal: "Edit statement", Files: []FileChange{{Path: file, Content: statementDefinition(description)}}}
}

func mapSnapshot(files map[string]string) *snapshot.Snapshot {
	result := &snapshot.Snapshot{Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, content := range files {
		result.Files[path] = []byte(content)
		result.Modes[path] = snapshot.RegularMode
	}
	return result
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func gitTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}
