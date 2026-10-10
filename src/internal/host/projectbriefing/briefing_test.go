package projectbriefing

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestBuildProducesStructuredBeforeAfterAndManagerNeighborContract(t *testing.T) {
	before, after := fixtureProjects()
	bundle, err := Build(before, after, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(bundle.Events))
	}
	event := bundle.Events[0]
	if event.Change != "modified" || event.Before == nil || event.After == nil {
		t.Fatalf("event change payload incomplete: %#v", event)
	}
	if event.Before.Spec["description"] != "old rule" || event.After.Spec["description"] != "new rule" {
		t.Fatalf("declared before/after content lost: %#v", event)
	}
	if len(event.AffectedArtifacts) != 1 || event.AffectedArtifacts[0] != artifactID() {
		t.Fatalf("affected artifacts = %v", event.AffectedArtifacts)
	}
	rootID, childID := managerIDs()
	if !contains(event.AffectedManagers, childID) || !contains(event.AffectedManagers, rootID) {
		t.Fatalf("affected managers = %v", event.AffectedManagers)
	}
	var rootBriefing *Briefing
	for i := range bundle.Managers {
		if bundle.Managers[i].ManagerID == rootID {
			rootBriefing = &bundle.Managers[i]
		}
	}
	if rootBriefing == nil || len(rootBriefing.Contracts) != 1 || rootBriefing.Contracts[0].ID != changedStatementID() {
		t.Fatalf("parent briefing did not retain the relevant public neighbor contract: %#v", rootBriefing)
	}
	if bundle.Global.ID == "" || bundle.Global.ModelDigest != after.Model.Digest {
		t.Fatalf("global briefing is not bound to accepted model: %#v", bundle.Global)
	}
}

func TestBuildUnchangedModelHasNoEventsAndStableBriefingIDs(t *testing.T) {
	before, after := fixtureProjects()
	unchanged := cloneProject(after)
	unchanged.Revision = strings.Repeat("c", 40)
	first, err := Build(after, unchanged, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(after, unchanged, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Events) != 0 {
		t.Fatalf("unchanged model produced %d events", len(first.Events))
	}
	if first.Global.ID != second.Global.ID {
		t.Fatalf("briefing id changed: %s vs %s", first.Global.ID, second.Global.ID)
	}
	_ = before
}

func TestBuildRejectsProvisionalRevision(t *testing.T) {
	before, after := fixtureProjects()
	after.Provisional = true
	if _, err := Build(before, after, testProvenance()); !errors.Is(err, ErrUncommittedModel) {
		t.Fatalf("error = %v, want ErrUncommittedModel", err)
	}
}

func TestDismissIsSeparateFromImmutableEventAndLoadRequiresExactModelDigest(t *testing.T) {
	root, base, revision := committedModelFixture(t)
	bundle, err := Generate(root, base, revision, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	_, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	newDigest, err := Write(root, bundle, digest)
	if err != nil {
		t.Fatal(err)
	}
	managerID := bundle.Events[0].AffectedManagers[0]
	_, events, _, err := LoadForManager(root, project.Model.Digest, managerID, revision)
	if err != nil || len(events) != 1 {
		t.Fatalf("load manager events = %d, err = %v", len(events), err)
	}
	storedEvent := events[0]
	newDigest, err = Dismiss(root, storedEvent.ID, managerID, newDigest)
	if err != nil {
		t.Fatal(err)
	}
	_, eventsAfterDismiss, _, err := LoadForManager(root, project.Model.Digest, managerID, revision)
	if err != nil || len(eventsAfterDismiss) != 1 || hash(eventsAfterDismiss[0]) != hash(storedEvent) {
		t.Fatalf("dismissal changed or resolved the event: %#v err=%v", eventsAfterDismiss, err)
	}
	if _, _, _, err := LoadForManager(root, "old-model-digest", managerID); !errors.Is(err, ErrStaleModel) {
		t.Fatalf("stale model error = %v", err)
	}
	stateAfter, _, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(stateAfter.Dismissals) != 1 || len(stateAfter.Briefings) != 1 {
		t.Fatalf("dismissal did not remain a separate visibility record: %#v", stateAfter)
	}
	path := filepath.Join(root, filepath.FromSlash(storePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var tampered Store
	if err := json.Unmarshal(data, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered.Briefings[0].Events[0].After.Spec["description"] = "edited on disk"
	data, err = json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Read(root); !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("tampered nested event passed store read: %v", err)
	}
}

func TestGenerateChecksAncestryAndNoOpWriteDoesNotAppend(t *testing.T) {
	root, base, revision := committedModelFixture(t)
	if _, err := Generate(root, revision, base, testProvenance()); !errors.Is(err, ErrRevisionNotAncestor) {
		t.Fatalf("reverse revision pair error = %v", err)
	}
	baseProject, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "checkout", "--", "README.md")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("Operational documentation change.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitCommitTest(t, root, "documentation only")
	noOpRevision := gitOutputTest(t, root, "rev-parse", "HEAD")
	noOp, err := Generate(root, revision, noOpRevision, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	if len(noOp.Events) != 0 || noOp.SinceModelDigest != noOp.ModelDigest || baseProject.Model.Digest != noOp.ModelDigest {
		t.Fatalf("documentation-only revision generated model events: %#v", noOp)
	}
	_, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Write(root, noOp, digest)
	if err != nil || result != digest {
		t.Fatalf("no-op write digest=%q err=%v want current digest %q", result, err, digest)
	}
	state, _, err := Read(root)
	if err != nil || len(state.Briefings) != 0 {
		t.Fatalf("no-op write appended history: %#v err=%v", state, err)
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
		t.Fatalf("no-op write created history file: %v", err)
	}
}

func committedModelFixture(t *testing.T) (root, base, revision string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate project briefing fixture")
	}
	repository := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	sourceRoot := filepath.Join(repository, "examples", "project-world")
	root = t.TempDir()
	if err := filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil || relative == "." {
			return err
		}
		target := filepath.Join(root, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "init", "--initial-branch=feature-briefing")
	gitTest(t, root, "add", ".")
	gitCommitTest(t, root, "briefing fixture baseline")
	base = gitOutputTest(t, root, "rev-parse", "HEAD")
	modelPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	old := "Cancellation is valid only while the order is confirmed; a shipped order cannot be cancelled."
	updated := strings.Replace(string(content), old, "Cancellation is valid only before shipment; a shipped order cannot be cancelled.", 1)
	if updated == string(content) {
		t.Fatal("fixture model declaration was not found")
	}
	if err := os.WriteFile(modelPath, []byte(updated), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "accepted model change")
	revision = gitOutputTest(t, root, "rev-parse", "HEAD")
	return root, base, revision
}

func gitTest(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func gitCommitTest(t *testing.T, root, message string) {
	t.Helper()
	gitTest(t, root, "commit", "-m", message)
}

func gitOutputTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output))
}

func sourceFileAtRevision(t *testing.T, root, revision, path string) ([]byte, error) {
	t.Helper()
	command := exec.Command("git", "show", revision+":"+path)
	command.Dir = root
	return command.Output()
}

func TestManagerHistoryUsesRevisionAncestryAcrossModelRevert(t *testing.T) {
	root, base, changedRevision := committedModelFixture(t)
	baseProject, err := projectwork.Load(root, base)
	if err != nil {
		t.Fatal(err)
	}
	changedProject, err := projectwork.Load(root, changedRevision)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Generate(root, base, changedRevision, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	_, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	digest, err = Write(root, first, digest)
	if err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	baseBytes, err := sourceFileAtRevision(t, root, base, ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, baseBytes, 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "revert accepted rule")
	revertedRevision := gitOutputTest(t, root, "rev-parse", "HEAD")
	second, err := Generate(root, changedRevision, revertedRevision, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	digest, err = Write(root, second, digest)
	if err != nil {
		t.Fatal(err)
	}
	revertedProject, err := projectwork.Load(root, revertedRevision)
	if err != nil {
		t.Fatal(err)
	}
	if revertedProject.Model.Digest != baseProject.Model.Digest || changedProject.Model.Digest == baseProject.Model.Digest {
		t.Fatalf("fixture did not form A→B→A model history: base=%s changed=%s reverted=%s", baseProject.Model.Digest, changedProject.Model.Digest, revertedProject.Model.Digest)
	}
	manager := first.Events[0].AffectedManagers[0]
	briefings, events, _, err := LoadForManager(root, revertedProject.Model.Digest, manager, revertedRevision)
	if err != nil {
		t.Fatal(err)
	}
	if len(briefings) != 2 || len(events) != 2 || briefings[0].Revision != changedRevision || briefings[1].Revision != revertedRevision {
		t.Fatalf("revision-ordered revert history briefings=%#v events=%#v", briefings, events)
	}
}

func TestManagerHistoryRejectsUnbriefedModelGapIncludingRevert(t *testing.T) {
	root, base, briefedRevision := committedModelFixture(t)
	briefed, err := Generate(root, base, briefedRevision, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	_, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Write(root, briefed, digest)
	if err != nil {
		t.Fatal(err)
	}
	manager := briefed.Events[0].AffectedManagers[0]
	modelPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-before-shipped.yaml")
	changedBytes, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	thirdValue := strings.Replace(string(changedBytes), "before shipment", "prior to fulfillment", 1)
	if thirdValue == string(changedBytes) {
		t.Fatal("could not create unbriefed third model value")
	}
	if err := os.WriteFile(modelPath, []byte(thirdValue), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "unbriefed semantic change")
	unbriefedRevision := gitOutputTest(t, root, "rev-parse", "HEAD")
	unbriefedProject, err := projectwork.Load(root, unbriefedRevision)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := LoadForManager(root, unbriefedProject.Model.Digest, manager, unbriefedRevision); !errors.Is(err, ErrStaleModel) {
		t.Fatalf("unbriefed target model error = %v", err)
	}
	if err := os.WriteFile(modelPath, changedBytes, 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml")
	gitCommitTest(t, root, "unbriefed model revert")
	revertedRevision := gitOutputTest(t, root, "rev-parse", "HEAD")
	briefedProject, err := projectwork.Load(root, briefedRevision)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := LoadForManager(root, briefedProject.Model.Digest, manager, revertedRevision); !errors.Is(err, ErrStaleModel) {
		t.Fatalf("unbriefed intermediate change/revert error = %v", err)
	}
}

func TestWriteRejectsTamperedPreviewBundle(t *testing.T) {
	root, base, revision := committedModelFixture(t)
	bundle, err := Generate(root, base, revision, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	bundle.Events[0].After.Spec["description"] = "tampered preview"
	_, digest, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, bundle, digest); !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("tampered bundle write error = %v", err)
	}
}

func TestReadMissingStoreDoesNotCreateOperationalDirectories(t *testing.T) {
	root := t.TempDir()
	if _, _, err := Read(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".markitect")); !os.IsNotExist(err) {
		t.Fatalf("Read created .markitect: err=%v", err)
	}
}

func fixtureProjects() (*projectwork.Project, *projectwork.Project) {
	rootID, childID := managerIDs()
	changedID, neighborID, artID := changedStatementID(), neighborStatementID(), artifactID()
	managerRoot := projectmodel.Manager{ID: rootID, Name: "root", Namespace: "demo", Owns: []string{"."}}
	managerChild := projectmodel.Manager{ID: childID, Name: "orders", Namespace: "demo", Parent: rootID, Owns: []string{"orders/"}}
	changed := projectmodel.Statement{ID: changedID, Name: "order-rule", Namespace: "demo", Owner: childID, Category: "rule", Description: "old rule", Public: true, Uses: []string{}, Requires: []string{}}
	neighbor := projectmodel.Statement{ID: neighborID, Name: "integration-contract", Namespace: "demo", Owner: rootID, Category: "architecture", Description: "uses order rule", Public: true, Uses: []string{changedID}, Requires: []string{}}
	artifact := projectmodel.Artifact{ID: artID, Name: "orders-api", Owner: childID, Role: "implementation", Realizes: []string{changedID}, Paths: []string{"orders/api.go"}, Checks: []string{}, Required: true}
	oldDef := core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Statement", Metadata: core.Metadata{Name: "order-rule", Namespace: "demo"}, Purpose: "Rule", Spec: map[string]any{"category": "rule", "description": "old rule", "public": true, "uses": []any{}, "requires": []any{}}, Source: core.Source{Path: ".markitect/model/demo.yaml", Digest: "old"}}
	newDef := oldDef
	newDef.Spec = map[string]any{"category": "rule", "description": "new rule", "public": true, "uses": []any{}, "requires": []any{}}
	newDef.Source = core.Source{Path: ".markitect/model/demo.yaml", Digest: "new"}
	rootDef := core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Manager", Metadata: core.Metadata{Name: "root", Namespace: "demo"}, Purpose: "Root", Spec: map[string]any{"owns": []any{"."}}, Source: core.Source{Path: ".markitect/model/demo.yaml"}}
	childDef := core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Manager", Metadata: core.Metadata{Name: "orders", Namespace: "demo"}, Purpose: "Orders", Spec: map[string]any{"parent": rootID, "owns": []any{"orders/"}}, Source: core.Source{Path: ".markitect/model/demo.yaml"}}
	old := fixtureProject(strings.Repeat("a", 40), []core.Definition{rootDef, childDef, oldDef}, []projectmodel.Manager{managerRoot, managerChild}, []projectmodel.Statement{changed, neighbor}, []projectmodel.Artifact{artifact})
	changed.Description = "new rule"
	newer := fixtureProject(strings.Repeat("b", 40), []core.Definition{rootDef, childDef, newDef}, []projectmodel.Manager{managerRoot, managerChild}, []projectmodel.Statement{changed, neighbor}, []projectmodel.Artifact{artifact})
	return old, newer
}

func fixtureProject(revision string, defs []core.Definition, managers []projectmodel.Manager, statements []projectmodel.Statement, artifacts []projectmodel.Artifact) *projectwork.Project {
	return &projectwork.Project{Revision: revision, Model: core.Model{Digest: "model-" + revision[:1], Definitions: defs}, Report: projectmodel.Report{Digest: "report-" + revision[:1], Managers: managers, Statements: statements, Artifacts: artifacts, Checks: []projectmodel.Check{}}}
}
func cloneProject(p *projectwork.Project) *projectwork.Project {
	cp := *p
	cp.Model = core.Model{Digest: p.Model.Digest, Definitions: append([]core.Definition(nil), p.Model.Definitions...)}
	cp.Report = p.Report
	return &cp
}
func testProvenance() Provenance {
	return Provenance{DecisionReference: "ADR-17", Actor: "project-owner", Authority: "accepted model decision"}
}
func managerIDs() (string, string) {
	root := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Namespace: "demo", Name: "root"}.Key()
	child := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Namespace: "demo", Name: "orders"}.Key()
	return root, child
}
func changedStatementID() string {
	return core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "demo", Name: "order-rule"}.Key()
}
func neighborStatementID() string {
	return core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Statement", Namespace: "demo", Name: "integration-contract"}.Key()
}
func artifactID() string {
	return core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Artifact", Namespace: "demo", Name: "orders-api"}.Key()
}
