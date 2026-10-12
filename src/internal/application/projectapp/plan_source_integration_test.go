package projectapp

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

func TestPlanWithoutRoutingSelectsAllManagers(t *testing.T) {
	root := makePlanFixture(t)
	operations := NewOperations()
	selection := Selection{Root: root}
	report, err := operations.Index(selection)
	if err != nil {
		t.Fatalf("load selected project through Application: %v", err)
	}
	plan, err := operations.Plan(PlanOperation{Selection: selection, Request: projectrun.PlanRequest{Goal: "Improve the bounded project behavior."}})
	if err != nil {
		t.Fatalf("plan default manager selection: %v", err)
	}
	if len(plan.Managers) != len(report.Managers) {
		t.Fatalf("goal-only request selected %d of %d declared managers", len(plan.Managers), len(report.Managers))
	}
}

func TestGuidedPlanPreviewReadsAcceptedHistoryWithoutStalingSelectedBasis(t *testing.T) {
	root := makePlanFixture(t)
	commitGuidedWorkflow(t, root)
	revision := gitOutput(t, root, "rev-parse", "HEAD")
	plan, err := NewOperations().Plan(PlanOperation{Selection: Selection{Root: root, Revision: revision}, Request: projectrun.PlanRequest{Goal: "Improve the bounded project behavior."}})
	if err != nil {
		t.Fatalf("guided plan after accepted-history reconciliation: %v", err)
	}
	if plan.Status != projectrun.StatusPlanned || plan.BaseRevision != revision || plan.WorkingSnapshot == "" {
		t.Fatalf("guided plan did not retain its fixed and raw working bindings: %+v", plan)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect", "state", "briefings", "history.json")); !os.IsNotExist(err) {
		t.Fatalf("guided plan preview wrote accepted history: %v", err)
	}
}

func commitGuidedWorkflow(t *testing.T, root string) {
	t.Helper()
	manifestPath := filepath.Join(root, filepath.FromSlash(projectvalue.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := bytes.Replace(manifest, []byte("workflowMode: guided\n"), nil, 1)
	if bytes.Equal(updated, manifest) {
		t.Fatal("fixture did not start without the guided workflow")
	}
	if err := os.WriteFile(manifestPath, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", projectvalue.ManifestPath)
	runGit(t, root, "commit", "-m", "record the initial structured workflow")
	updated = bytes.Replace(updated, []byte("name: Facade fixture\n"), []byte("name: Facade fixture\nworkflowMode: guided\n"), 1)
	if bytes.Equal(updated, manifest) || bytes.Contains(updated, []byte("workflowMode: guided")) == false {
		t.Fatal("could not enable guided workflow in fixture")
	}
	if err := os.WriteFile(manifestPath, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", projectvalue.ManifestPath)
	runGit(t, root, "commit", "-m", "enable guided workflow")
}

func TestSelectedBasisAcceptsOperationalStateAndGitNormalizedCheckout(t *testing.T) {
	root := makePlanFixture(t)
	commitGuidedWorkflow(t, root)
	runGit(t, root, "config", "core.autocrlf", "true")
	manifestPath := filepath.Join(root, filepath.FromSlash(projectvalue.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	selectedManifest := bytes.Replace(manifest, []byte("inventoryRoots: []\n"), []byte("inventoryRoots:\n  - docs\n"), 1)
	if bytes.Equal(selectedManifest, manifest) {
		t.Fatal("fixture did not start with empty selected inventory roots")
	}
	if err := os.WriteFile(manifestPath, selectedManifest, 0o600); err != nil {
		t.Fatal(err)
	}
	docsRoot := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsRoot, ".gitattributes"), []byte("binary.dat -text\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsRoot, "binary.dat"), []byte("binary\ncontent\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsRoot, "README.md"), []byte("selected source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", projectvalue.ManifestPath, "docs")
	runGit(t, root, "commit", "-m", "record source normalization fixture")
	revision := gitOutput(t, root, "rev-parse", "HEAD")
	operations := NewOperations()

	readmePath := filepath.Join(docsRoot, "README.md")
	readmeBytes, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readmePath, bytes.ReplaceAll(readmeBytes, []byte("\n"), []byte("\r\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".markitect", "state", "explorations"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".markitect", "state", "explorations", "fixture.json"), []byte("operational exploration state\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := projectrun.PlanRequest{Goal: "Improve the bounded project behavior."}
	plan, err := operations.Plan(PlanOperation{Selection: Selection{Root: root, Revision: revision}, Request: request})
	if err != nil {
		t.Fatalf("Git-clean CRLF checkout and operational state should not stale selected inputs: %v", err)
	}
	if plan.BaseSnapshot == "" || plan.WorkingSnapshot == "" || plan.BaseSnapshot == plan.WorkingSnapshot {
		t.Fatalf("fixture did not retain distinct fixed and raw working bindings for the normalized checkout: base=%q working=%q", plan.BaseSnapshot, plan.WorkingSnapshot)
	}

	binaryPath := filepath.Join(docsRoot, "binary.dat")
	binaryBytes, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, bytes.ReplaceAll(binaryBytes, []byte("\n"), []byte("\r\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.Plan(PlanOperation{Selection: Selection{Root: root, Revision: revision}, Request: request}); err == nil || !strings.Contains(err.Error(), "commit accepted selected changes") {
		t.Fatalf("Git -text content change must remain blocked, got %v", err)
	}

	if err := os.WriteFile(readmePath, append(readmeBytes, []byte("manual source change\r\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "update-index", "--assume-unchanged", "docs/README.md")
	if _, err := operations.Plan(PlanOperation{Selection: Selection{Root: root, Revision: revision}, Request: request}); err == nil || !strings.Contains(err.Error(), "assume-unchanged or skip-worktree") {
		t.Fatalf("substantive source change hidden by Git's assume-unchanged flag should remain blocked, got %v", err)
	}
	runGit(t, root, "update-index", "--no-assume-unchanged", "docs/README.md")
}
