package projectcli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/internal/host/projectexplore"
)

func TestMCPSharedExplorePreviewAndCASParity(t *testing.T) {
	root := copyProjectWorld(t)
	operations := projectOperations()
	record := &projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: "mcp-work", Status: projectexplore.StatusActive, Request: "Preview the same selected scope through both transports", Scopes: []projectexplore.Scope{{ID: "scope", Name: "Scope", Goal: "Inspect owned artifacts", Operation: "apply", ManagerIDs: []string{}}}, Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}
	preview, err := operations.Explore(projectapp.ExploreOperation{Selection: projectapp.Selection{Root: root}, Record: record})
	if err != nil {
		t.Fatal(err)
	}
	server, err := projectMCP(root, operations)
	if err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(mcpExploreInput{Record: record, Write: false})
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.Call(context.Background(), "project_explore", args)
	if err != nil || result.IsError {
		t.Fatalf("MCP preview: %+v %v", result, err)
	}
	data := result.StructuredContent.(map[string]any)["data"].(map[string]any)
	plan := data["plan"].(map[string]any)
	if plan["digest"] != preview.Plan.Digest {
		t.Fatalf("transport changed guarded preview: got=%v want=%s", plan["digest"], preview.Plan.Digest)
	}
	args, _ = json.Marshal(mcpExploreInput{Record: record, Write: true, ExpectedDigest: "sha256:stale"})
	failed, err := server.Call(context.Background(), "project_explore", args)
	if err != nil || !failed.IsError {
		t.Fatalf("wrong digest accepted: %+v %v", failed, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect", "state", "explorations", record.ID+".json")); !os.IsNotExist(err) {
		t.Fatalf("wrong digest wrote exploration: %v", err)
	}
	args, _ = json.Marshal(mcpExploreInput{Record: record, Write: true, ExpectedDigest: preview.Plan.Digest})
	written, err := server.Call(context.Background(), "project_explore", args)
	if err != nil || written.IsError {
		t.Fatalf("MCP write: %+v %v", written, err)
	}
	stored, err := projectexplore.Load(root, record.ID)
	if err != nil || stored.Digest != preview.Plan.Next.Digest {
		t.Fatalf("MCP stored different record: %+v %v", stored, err)
	}
}

func TestMCPCompositionHasTypedBootstrapAndRejectsRootRedirection(t *testing.T) {
	root := copyProjectWorld(t)
	server, err := projectMCP(root, projectOperations())
	if err != nil {
		t.Fatal(err)
	}
	present := map[string]bool{}
	for _, tool := range server.Tools() {
		present[tool.Name] = true
	}
	for _, name := range []string{"project_init", "project_check", "project_edit", "project_onboard", "project_setup", "project_explore", "project_readiness", "project_brownfield", "project_brownfield_run", "project_run", "project_status", "project_verify", "project_apply"} {
		if !present[name] {
			t.Fatalf("missing product tool %s", name)
		}
	}
	for _, name := range []string{"project_check", "project_explore", "project_setup"} {
		if _, err := server.Call(context.Background(), name, json.RawMessage(`{"root":"elsewhere","write":false}`)); err == nil {
			t.Fatalf("%s accepted caller root redirection", name)
		}
	}
	opts, help, err := parse([]string{"project", "mcp", "--repo", root}, os.Stderr)
	if err != nil || help || opts.action != "mcp" {
		t.Fatalf("CLI MCP composition missing: %+v %v", opts, err)
	}
}
