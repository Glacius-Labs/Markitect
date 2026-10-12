package projectrun

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

// resumeCrashProject provides only the model/report/file facts consumed by
// private Run lifecycle gates; it does not acquire or compile repository YAML.
func resumeCrashProject(t *testing.T, root, revision string) *projectvalue.Project {
	t.Helper()
	const (
		ordersPath    = "src/orders/implementation.txt"
		inventoryPath = "src/inventory/implementation.txt"
	)
	rootID := e2eManagerID("", "project-owner")
	ordersID := e2eManagerID("orders", "orders")
	inventoryID := e2eManagerID("inventory", "inventory")
	ordersStatement := e2eFixtureIdentity("Statement", "orders", "orders-work")
	inventoryStatement := e2eFixtureIdentity("Statement", "inventory", "inventory-work")
	ordersArtifact := e2eFixtureIdentity("Artifact", "orders", "orders-code")
	inventoryArtifact := e2eFixtureIdentity("Artifact", "inventory", "inventory-code")
	ordersCheck := e2eFixtureIdentity("Check", "orders", "orders-check")
	inventoryCheck := e2eFixtureIdentity("Check", "inventory", "inventory-check")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	checkCommand := append([]string{filepath.Base(executable)}, runtimeFixtureArgs(t, []string{"-test.run=^TestProjectRunCheckProcess$"})...)
	files := map[string][]byte{
		"README.md":   []byte("project-run process fixture\n"),
		ordersPath:    []byte("orders implementation v1\n"),
		inventoryPath: []byte("inventory implementation v1\n"),
	}
	modes := map[string]string{"README.md": "100644", ordersPath: "100644", inventoryPath: "100644"}
	report := projectmodel.Report{
		APIVersion: projectmodel.APIVersion, ModelDigest: "resume-crash-fixture-model", Digest: "resume-crash-fixture-report", Status: "succeeded",
		Managers: []projectmodel.Manager{
			{ID: rootID, Name: "project-owner", Purpose: "Owns integration and project-wide delivery.", Owns: []string{"."}},
			{ID: ordersID, Name: "orders", Namespace: "orders", Purpose: "Owns the orders implementation.", Parent: rootID, Owns: []string{"src/orders/"}},
			{ID: inventoryID, Name: "inventory", Namespace: "inventory", Purpose: "Owns the inventory implementation.", Parent: rootID, Owns: []string{"src/inventory/"}},
		},
		Statements: []projectmodel.Statement{
			{ID: ordersStatement, Name: "orders-work", Namespace: "orders", Owner: ordersID, Category: "concept", Description: "Implement the orders artifact."},
			{ID: inventoryStatement, Name: "inventory-work", Namespace: "inventory", Owner: inventoryID, Category: "concept", Description: "Implement the inventory artifact."},
		},
		Artifacts: []projectmodel.Artifact{
			{ID: ordersArtifact, Name: "orders-code", Owner: ordersID, Role: "implementation", Realizes: []string{ordersStatement}, Paths: []string{ordersPath}, Checks: []string{ordersCheck}, Required: true},
			{ID: inventoryArtifact, Name: "inventory-code", Owner: inventoryID, Role: "implementation", Realizes: []string{inventoryStatement}, Paths: []string{inventoryPath}, Checks: []string{inventoryCheck}, Required: true},
		},
		Checks: []projectmodel.Check{
			{ID: ordersCheck, Name: "orders-check", Owner: ordersID, Command: checkCommand, Uses: []string{ordersStatement}, Limitation: "Validates the explicit Runtime fixture bytes."},
			{ID: inventoryCheck, Name: "inventory-check", Owner: inventoryID, Command: checkCommand, Uses: []string{inventoryStatement}, Limitation: "Validates the explicit Runtime fixture bytes."},
		},
		Files: []projectmodel.FileEntry{
			{Path: ordersPath, Owner: ordersID, Class: "source", Statements: []string{ordersStatement}, Artifacts: []string{ordersArtifact}, Checks: []string{ordersCheck}, Exists: true},
			{Path: inventoryPath, Owner: inventoryID, Class: "source", Statements: []string{inventoryStatement}, Artifacts: []string{inventoryArtifact}, Checks: []string{inventoryCheck}, Exists: true},
		},
	}
	project := runtimeProjectFixture(revision, files, modes, core.Model{}, report)
	project.Root = root
	project.Digest = "resume-crash-fixture-project"
	project.Config = projectvalue.Config{APIVersion: projectvalue.APIVersion, Name: "Resume crash fixture", WorkflowMode: projectvalue.WorkflowModeEmpty, InventoryRoots: []string{"src"}}
	return project
}

func resumeCrashHost(root string, project *projectvalue.Project) Host {
	managerIDs := map[string]bool{
		e2eManagerID("", "project-owner"):      true,
		e2eManagerID("orders", "orders"):       true,
		e2eManagerID("inventory", "inventory"): true,
	}
	cloneProject := func(source *snapshot.Snapshot) *projectvalue.Project {
		loaded := *project
		if source == nil {
			source = project.Snapshot
		}
		copySnapshot := &snapshot.Snapshot{ID: source.ID, Provisional: source.Provisional, Files: make(map[string][]byte, len(source.Files)), Modes: make(map[string]string, len(source.Modes))}
		for path, content := range source.Files {
			copySnapshot.Files[path] = append([]byte(nil), content...)
		}
		for path, mode := range source.Modes {
			copySnapshot.Modes[path] = mode
		}
		loaded.Snapshot = copySnapshot
		return &loaded
	}
	return Host{
		Load: func(loadRoot, revision string) (*projectvalue.Project, error) {
			if loadRoot != root || (revision != "" && revision != project.Revision) {
				return nil, fmt.Errorf("resume crash fixture received an unbound project load")
			}
			return cloneProject(project.Snapshot), nil
		},
		FromSnapshot: func(loadRoot string, source *snapshot.Snapshot) (*projectvalue.Project, error) {
			if loadRoot != root || source == nil {
				return nil, fmt.Errorf("resume crash fixture received an unbound snapshot")
			}
			return cloneProject(source), nil
		},
		PlanEdit: func(*projectvalue.Project, projectvalue.Mutation) (projectvalue.EditPlan, error) {
			return projectvalue.EditPlan{}, fmt.Errorf("resume crash fixture has no model-edit capability")
		},
		ClassifyCandidate: func(base, candidate *projectvalue.Project, _ ...string) (*projectvalue.Project, error) {
			if base == nil || base.Revision != project.Revision || candidate == nil || candidate.Revision != project.Revision || candidate.Snapshot == nil {
				return nil, fmt.Errorf("resume crash fixture candidate is missing")
			}
			return candidate, nil
		},
		BriefingForManager: func(loadRoot, revision, modelDigest, managerID string, _ *projectbriefing.Store) (BriefingContext, error) {
			if loadRoot != root || (revision != "" && revision != project.Revision) || modelDigest != project.Report.ModelDigest || !managerIDs[managerID] {
				return BriefingContext{}, fmt.Errorf("resume crash fixture received an unbound briefing request")
			}
			return BriefingContext{Digest: "resume-crash-fixture-empty-briefing"}, nil
		},
	}
}

func e2eFixtureIdentity(kind, namespace, name string) string {
	data, _ := json.Marshal([]string{projectmodel.APIVersion, kind, namespace, name})
	return string(data)
}
