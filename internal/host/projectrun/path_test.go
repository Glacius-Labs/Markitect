package projectrun

import (
	"io/fs"
	"runtime"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestProjectProposalPathsStayInSelectedInventory(t *testing.T) {
	config := Config{InventoryRoots: []string{"src/"}, Exclusions: []Exclusion{{Path: "src/private/", Reason: "private"}}}
	for _, path := range []string{"outside/new.go", "src/private/secret.go", "src/Private/new.go", ".markitect/model/new.yaml", ".markitect/drafts/agent-fixtures/x.md", ".markitect/new-control-file"} {
		if projectPathAllowed(config, path) {
			t.Errorf("proposal path %q should be rejected", path)
		}
	}
	for _, path := range []string{"src/orders/order.go", "src/new.go"} {
		if !projectPathAllowed(config, path) {
			t.Errorf("proposal path %q should be allowed", path)
		}
	}
}

func TestCaptureModeMatchesGitUsesWindowsPermissionSemantics(t *testing.T) {
	if runtime.GOOS == "windows" {
		if !captureModeMatchesGit(fs.FileMode(0o666), "100644") {
			t.Fatal("Windows writable file permissions should match Git 100644")
		}
		if !captureModeMatchesGit(fs.FileMode(0o666), "100755") {
			t.Fatal("Windows permissions do not encode the Git executable bit")
		}
		return
	}
	if captureModeMatchesGit(fs.FileMode(0o666), "100644") {
		t.Fatal("POSIX mode comparison must remain exact")
	}
	if !captureModeMatchesGit(fs.FileMode(0o644), "100644") {
		t.Fatal("POSIX 0644 should match Git 100644")
	}
}

func TestApplyProposalRejectsUninventoriedAndControlPlanePaths(t *testing.T) {
	config := Config{InventoryRoots: []string{"src/"}}
	report := projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root", Owns: []string{"."}}}}
	task := ManagerTask{ManagerID: "root", Owns: []string{"."}}
	for _, path := range []string{"outside/new.go", ".markitect/drafts/agent-fixtures/test.md"} {
		_, err := applyProposal(candidateData{Files: map[string]File{}}, []agentexec.CandidateFile{{Path: path, Mode: "0644", Content: "x"}}, config, report, task, "work", nil, Limits{MaxCandidateFileBytes: 1024, MaxCandidateBytes: 2048})
		if err == nil {
			t.Errorf("proposal to %q unexpectedly passed", path)
		}
	}
}
