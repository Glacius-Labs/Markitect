package projecthistory

import (
	"encoding/json"
	"errors"
	projectbriefing "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectbriefing"
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
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

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
	_, digest, err := readState(root)
	if err != nil {
		t.Fatal(err)
	}
	newDigest, err := Write(root, bundle, digest)
	if err != nil {
		t.Fatal(err)
	}
	managerID := managerExposingEvent(t, project, bundle.Events[0])
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
	if _, _, _, err := LoadForManager(root, "old-model-digest", managerID); !errors.Is(err, projectbriefing.ErrStaleModel) {
		t.Fatalf("stale model error = %v", err)
	}
	stateAfter, _, err := readState(root)
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
	var tampered projectbriefing.Store
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
	if _, _, err := readState(root); !errors.Is(err, projectbriefing.ErrInvalidBundle) {
		t.Fatalf("tampered nested event passed store read: %v", err)
	}
}

func TestGenerateChecksAncestryAndNoOpWriteDoesNotAppend(t *testing.T) {
	root, base, revision := committedModelFixture(t)
	if _, err := Generate(root, revision, base, testProvenance()); !errors.Is(err, projectbriefing.ErrRevisionNotAncestor) {
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
	_, digest, err := readState(root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Write(root, noOp, digest)
	if err != nil || result != digest {
		t.Fatalf("no-op write digest=%q err=%v want current digest %q", result, err, digest)
	}
	state, _, err := readState(root)
	if err != nil || len(state.Briefings) != 0 {
		t.Fatalf("no-op write appended history: %#v err=%v", state, err)
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(storePath))); !os.IsNotExist(err) {
		t.Fatalf("no-op write created history file: %v", err)
	}
}

func committedModelFixture(t *testing.T) (root, base, revision string) {
	t.Helper()
	root = t.TempDir()
	copyProjectWorldTest(t, root)
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

func managerExposingEvent(t *testing.T, project *projectvalue.Project, event projectbriefing.Event) string {
	t.Helper()
	for _, managerID := range event.AffectedManagers {
		context, err := projectmodel.Context(project.Report, managerID)
		if err != nil {
			continue
		}
		for _, statement := range context.Statements {
			if statement.ID == event.DefinitionID.Key() {
				return managerID
			}
		}
		for _, contract := range context.Contracts {
			if contract.ID == event.DefinitionID.Key() {
				return managerID
			}
		}
	}
	t.Fatalf("no affected ManagerContext exposes Statement %s", event.DefinitionID.Key())
	return ""
}

func managerExposingEventInHistory(t *testing.T, before, after *projectvalue.Project, event projectbriefing.Event) string {
	t.Helper()
	for _, managerID := range event.AffectedManagers {
		for _, project := range []*projectvalue.Project{before, after} {
			if project == nil {
				continue
			}
			view, err := projectmodel.Context(project.Report, managerID)
			if err != nil {
				continue
			}
			for _, definition := range []*core.Definition{event.Before, event.After} {
				if definition == nil || definition.Kind != "Statement" {
					continue
				}
				for _, statement := range view.Statements {
					if statement.ID == definition.Identity().Key() {
						return managerID
					}
				}
			}
		}
	}
	t.Fatalf("no affected manager exposes an event side in its historical contexts: %s", event.ID)
	return ""
}

// copyProjectWorldTest copies the project-world example into root.
func copyProjectWorldTest(t *testing.T, root string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate project briefing fixture")
	}
	repository := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	sourceRoot := filepath.Join(repository, "examples", "project-world")
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
	_, digest, err := readState(root)
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
	manager := managerExposingEvent(t, revertedProject, first.Events[0])
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
	_, digest, err := readState(root)
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
	if _, _, _, err := LoadForManager(root, unbriefedProject.Model.Digest, manager, unbriefedRevision); !errors.Is(err, projectbriefing.ErrStaleModel) {
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
	if _, _, _, err := LoadForManager(root, briefedProject.Model.Digest, manager, revertedRevision); !errors.Is(err, projectbriefing.ErrStaleModel) {
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
	_, digest, err := readState(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, bundle, digest); !errors.Is(err, projectbriefing.ErrInvalidBundle) {
		t.Fatalf("tampered bundle write error = %v", err)
	}
}

func TestLoadForManagerHidesPrivateChildStatementFromAncestor(t *testing.T) {
	root := t.TempDir()
	copyProjectWorldTest(t, root)
	privatePath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "private-internal-guard.yaml")
	projectConfigPath := filepath.Join(root, ".markitect", "project.yaml")
	projectConfig, err := os.ReadFile(projectConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	projectConfigText := strings.Replace(string(projectConfig),
		"  - .markitect/model/commerce/sales/orders/order.yaml\n",
		"  - .markitect/model/commerce/sales/orders/order.yaml\n  - .markitect/model/commerce/sales/orders/private-internal-guard.yaml\n", 1)
	if projectConfigText == string(projectConfig) {
		t.Fatal("could not add the private Statement to the model inputs")
	}
	if err := os.WriteFile(projectConfigPath, []byte(projectConfigText), 0644); err != nil {
		t.Fatal(err)
	}
	privateModel := `apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  name: private-internal-guard
  namespace: commerce.sales.orders
purpose: Internal order handling detail.
spec:
  category: rule
  description: Internal guard before change.
  public: false
  uses: []
  requires: []
`
	if err := os.WriteFile(privatePath, []byte(privateModel), 0644); err != nil {
		t.Fatal(err)
	}
	publicPath := filepath.Join(root, ".markitect", "model", "commerce", "sales", "orders", "cancel-order.yaml")
	publicBytes, err := os.ReadFile(publicPath)
	if err != nil {
		t.Fatal(err)
	}
	publicModel := strings.Replace(string(publicBytes),
		"    - namespace: commerce.sales.inventory.reservations\n      name: release-reservation\n  requires:",
		"    - namespace: commerce.sales.inventory.reservations\n      name: release-reservation\n    - namespace: commerce.sales.orders\n      name: private-internal-guard\n  requires:", 1)
	if publicModel == string(publicBytes) {
		t.Fatal("could not add a private relation to the public contract")
	}
	if err := os.WriteFile(publicPath, []byte(publicModel), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "init", "--initial-branch=private-briefing")
	gitTest(t, root, "add", ".")
	gitCommitTest(t, root, "private child Statement baseline")

	privateUpdated := strings.Replace(privateModel, "Internal guard before change.", "Internal guard after change.", 1)
	if err := os.WriteFile(privatePath, []byte(privateUpdated), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/private-internal-guard.yaml")
	gitCommitTest(t, root, "change private child Statement")
	publicBytes, err = os.ReadFile(publicPath)
	if err != nil {
		t.Fatal(err)
	}
	publicDescription := "A confirmed order may be cancelled before shipment only when exactly one active inventory reservation exists; cancellation releases it in the same transaction or leaves both records unchanged on failure."
	publicUpdated := "Cancel a confirmed order before shipment only when exactly one active reservation exists; release it in the same transaction or leave both records unchanged on failure."
	publicModel = strings.Replace(string(publicBytes), publicDescription, publicUpdated, 1)
	if publicModel == string(publicBytes) {
		t.Fatal("could not update the public ancestor contract")
	}
	if err := os.WriteFile(publicPath, []byte(publicModel), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".markitect/model/commerce/sales/orders/cancel-order.yaml")
	gitCommitTest(t, root, "change public ancestor contract")
	revision := gitOutputTest(t, root, "rev-parse", "HEAD")
	project, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	privateStatement := projectmodel.Statement{}
	for _, statement := range project.Report.Statements {
		if statement.Name == "private-internal-guard" {
			privateStatement = statement
		}
	}
	if privateStatement.ID == "" || privateStatement.Public {
		t.Fatalf("fixture's child Statement is not private: %+v", privateStatement)
	}
	parentByManager := make(map[string]string, len(project.Report.Managers))
	for _, manager := range project.Report.Managers {
		parentByManager[manager.ID] = manager.Parent
	}
	ancestorID := parentByManager[privateStatement.Owner]
	if ancestorID == "" {
		t.Fatalf("child Statement owner %s has no ancestor", privateStatement.Owner)
	}
	if _, err := EnsureAcceptedHistory(root, revision); err != nil {
		t.Fatal(err)
	}
	state, _, err := readState(root)
	if err != nil {
		t.Fatal(err)
	}
	rootBriefings, rootEvents, rootBinding, err := LoadForManagerFrom(root, state, project.Model.Digest, ancestorID, revision)
	if err != nil {
		t.Fatal(err)
	}
	briefedEventCount := 0
	for _, briefing := range rootBriefings {
		briefedEventCount += len(briefing.EventIDs)
	}
	if len(rootBriefings) != 2 || briefedEventCount != 2 {
		t.Fatalf("ancestor briefing lost authoritative event count: briefings=%+v", rootBriefings)
	}
	for _, event := range rootEvents {
		if event.DefinitionID.Key() == privateStatement.ID {
			t.Fatalf("ancestor received private child Statement event: %+v", event)
		}
		if event.DefinitionID.Name == "cancel-order" {
			for _, definition := range []*core.Definition{event.Before, event.After} {
				if definition == nil {
					continue
				}
				for _, relationField := range []string{"uses", "requires"} {
					relations, ok := definition.Spec[relationField].([]string)
					if !ok {
						t.Fatalf("projected %s relation has unexpected representation %T", relationField, definition.Spec[relationField])
					}
					for _, relatedID := range relations {
						if relatedID == privateStatement.ID {
							t.Fatalf("ancestor public contract event leaked private %s relation", relationField)
						}
					}
				}
			}
		}
	}
	if len(rootEvents) != 1 || rootEvents[0].DefinitionID.Name != "cancel-order" {
		t.Fatalf("ancestor should retain its public contract event: %+v", rootEvents)
	}
	publicContractID := ""
	for _, statement := range project.Report.Statements {
		if statement.Name == "cancel-order" && statement.Public {
			publicContractID = statement.ID
		}
	}
	contractRetained := false
	for _, briefing := range rootBriefings {
		for _, contract := range briefing.Contracts {
			if contract.ID == publicContractID {
				contractRetained = true
			}
			for _, relatedID := range append(append([]string(nil), contract.Uses...), contract.Requires...) {
				if relatedID == privateStatement.ID {
					t.Fatalf("projectbriefing.Briefing.Contracts leaked private Statement relation %s", relatedID)
				}
			}
		}
	}
	if publicContractID == "" || !contractRetained {
		t.Fatalf("ancestor briefing lost its public contract obligation %s: %+v", publicContractID, rootBriefings)
	}
	_, _, repeatedRootBinding, err := LoadForManagerFrom(root, state, project.Model.Digest, ancestorID, revision)
	if err != nil {
		t.Fatal(err)
	}
	if repeatedRootBinding != rootBinding {
		t.Fatalf("same ManagerContext produced unstable binding: first=%s repeated=%s", rootBinding, repeatedRootBinding)
	}

	_, childEvents, childBinding, err := LoadForManagerFrom(root, state, project.Model.Digest, privateStatement.Owner, revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(childEvents) != 2 {
		t.Fatalf("owning Manager lost its private Statement or public contract events: %+v", childEvents)
	}
	if childBinding == rootBinding {
		t.Fatalf("distinct ManagerContexts produced identical bindings: %s", childBinding)
	}
	storedPrivateEvent := false
	for _, bundle := range state.Briefings {
		if projectbriefing.FullBundleDigest(bundle) != bundle.Digest {
			t.Fatalf("manager projection changed the stored bundle digest for %s", bundle.Revision)
		}
		for _, event := range bundle.Events {
			if event.DefinitionID.Key() == privateStatement.ID {
				storedPrivateEvent = event.Before != nil && event.After != nil
			}
		}
	}
	if !storedPrivateEvent {
		t.Fatal("manager projection mutated or removed the canonical private event from stored history")
	}
}

func TestReadMissingStoreDoesNotCreateOperationalDirectories(t *testing.T) {
	root := t.TempDir()
	if _, _, err := readState(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".markitect")); !os.IsNotExist(err) {
		t.Fatalf("Read created .markitect: err=%v", err)
	}
}

func testProvenance() projectbriefing.Provenance {
	return projectbriefing.Provenance{DecisionReference: "ADR-17", Actor: "project-owner", Authority: "accepted model decision"}
}
