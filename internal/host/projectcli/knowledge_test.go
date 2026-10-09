package projectcli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/knowledgeprotocol"
	"github.com/Glacius-Labs/Markitect/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

func TestParseKnowledgeRequiresExplicitScopeAndActionShape(t *testing.T) {
	valid := [][]string{
		{"knowledge", "--repo", ".", "--manager", "manager/orders", "--knowledge-action", "graph"},
		{"knowledge", "--repo", ".", "--knowledge-scope", "project", "--knowledge-action", "trace", "--node-id", "file:src/orders.py", "--max-depth", "3", "--bidirectional"},
		{"knowledge", "--repo", ".", "--manager", "manager/orders", "--knowledge-action", "history", "--node-id", "statement/orders/cancel", "--run", "run-1", "--briefing-history"},
		{"knowledge-mcp", "--repo", ".", "--revision", strings.Repeat("a", 40)},
	}
	for _, args := range valid {
		if _, help, err := parse(args, nil); err != nil || help {
			t.Errorf("parse valid knowledge args %v: help=%t err=%v", args, help, err)
		}
	}
	parsed, _, err := parse(valid[1], nil)
	if err != nil {
		t.Fatal(err)
	}
	query, err := knowledgeQuery(parsed)
	if err != nil || !query.Bidirectional || query.Reverse {
		t.Fatalf("bidirectional traversal was not mapped to the service request: %+v err=%v", query, err)
	}
	invalid := [][]string{
		{"knowledge", "--repo", ".", "--knowledge-action", "graph"},
		{"knowledge", "--repo", ".", "--manager", "manager/orders", "--knowledge-scope", "project", "--knowledge-action", "graph"},
		{"knowledge", "--repo", ".", "--knowledge-scope", "other", "--knowledge-action", "graph"},
		{"knowledge", "--repo", ".", "--manager", "manager/orders", "--knowledge-action", "trace"},
		{"knowledge", "--repo", ".", "--manager", "manager/orders", "--knowledge-action", "graph", "--node-id", "some-node"},
		{"knowledge", "--repo", ".", "--manager", "one", "--manager", "two", "--knowledge-action", "graph"},
		{"knowledge", "--repo", ".", "--manager", "manager/orders", "--knowledge-action", "trace", "--node-id", "some-node", "--reverse", "--bidirectional"},
		{"knowledge-mcp", "--repo", ".", "--manager", "manager/orders"},
	}
	for _, args := range invalid {
		if _, _, err := parse(args, nil); err == nil {
			t.Errorf("accepted invalid knowledge args %v", args)
		}
	}
}

func TestKnowledgeCLIAndMCPShareTheSameProjectService(t *testing.T) {
	repo := copyProjectWorld(t)
	var cliOut, cliErr bytes.Buffer
	args := []string{"project", "knowledge", "--repo", repo, "--knowledge-scope", "project", "--knowledge-action", "graph"}
	if code := Run(args, &cliOut, &cliErr); code != 0 {
		t.Fatalf("knowledge CLI exit=%d stderr=%s", code, cliErr.String())
	}
	var cliResult projectapp.KnowledgeResult
	if err := json.Unmarshal(cliOut.Bytes(), &cliResult); err != nil {
		t.Fatalf("decode CLI result: %v", err)
	}
	if len(cliResult.Graph.Nodes) == 0 || cliResult.Graph.Binding.SnapshotDigest == "" {
		t.Fatalf("CLI result lacks selected graph/binding: nodes=%d binding=%+v", len(cliResult.Graph.Nodes), cliResult.Graph.Binding)
	}
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": "knowledge_graph", "arguments": map[string]any{"projectScope": true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	request = append(append(knowledgeMCPHandshake(t), request...), '\n')
	var protocolOut bytes.Buffer
	operations := projectOperations()
	selection := projectapp.Selection{Root: repo}
	err = knowledgeprotocol.Serve(bytes.NewReader(request), &protocolOut, func(call knowledgeprotocol.ToolRequest) (any, error) {
		return operations.Knowledge(projectapp.KnowledgeOperation{
			Selection: selection, Scope: call.Scope, Query: call.Query, Records: call.Records,
		})
	})
	if err != nil {
		t.Fatalf("serve MCP request: %v", err)
	}
	var envelope struct {
		Result struct {
			Structured json.RawMessage `json:"structuredContent"`
			IsError    bool            `json:"isError"`
		} `json:"result"`
	}
	responses := bytes.Split(bytes.TrimSpace(protocolOut.Bytes()), []byte{'\n'})
	if len(responses) != 2 {
		t.Fatalf("unexpected MCP response sequence: %s", protocolOut.String())
	}
	if err := json.Unmarshal(responses[1], &envelope); err != nil {
		t.Fatalf("decode MCP response: %v", err)
	}
	if envelope.Result.IsError || len(envelope.Result.Structured) == 0 {
		t.Fatalf("MCP call failed: %s", protocolOut.String())
	}
	var mcpResult projectapp.KnowledgeResult
	if err := json.Unmarshal(envelope.Result.Structured, &mcpResult); err != nil {
		t.Fatalf("decode structured MCP result: %v", err)
	}
	if !bytes.Equal(mustJSON(t, cliResult.Graph), mustJSON(t, mcpResult.Graph)) {
		t.Fatal("CLI and MCP knowledge graph results differ")
	}

	loaded, err := projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	ordersManager, inventoryManager := "", ""
	privateInventoryArtifact := ""
	for _, manager := range loaded.Report.Managers {
		if manager.Name == "orders" {
			ordersManager = manager.ID
		}
		if manager.Name == "inventory" {
			inventoryManager = manager.ID
		}
	}
	for _, artifact := range loaded.Report.Artifacts {
		if artifact.Owner == inventoryManager {
			privateInventoryArtifact = artifact.ID
			break
		}
	}
	if ordersManager == "" || inventoryManager == "" || privateInventoryArtifact == "" {
		t.Fatalf("fixture lacks Orders, Inventory, or a private Inventory artifact: orders=%q inventory=%q artifact=%q", ordersManager, inventoryManager, privateInventoryArtifact)
	}
	managerArgs := []string{"project", "knowledge", "--repo", repo, "--manager", ordersManager, "--knowledge-action", "graph"}
	var managerOut, managerErr bytes.Buffer
	if code := Run(managerArgs, &managerOut, &managerErr); code != 0 {
		t.Fatalf("Manager knowledge CLI exit=%d stderr=%s", code, managerErr.String())
	}
	var managerResult projectapp.KnowledgeResult
	if err := json.Unmarshal(managerOut.Bytes(), &managerResult); err != nil {
		t.Fatalf("decode Manager CLI result: %v", err)
	}
	identityOnlyOwner := false
	for _, node := range managerResult.Graph.Nodes {
		if node.ID == inventoryManager {
			identityOnlyOwner = true
			if len(node.Properties) != 0 || node.Purpose != "" || node.Source != nil {
				t.Fatalf("public contract owner identity exposed private manager details: %+v", node)
			}
		}
	}
	if !identityOnlyOwner {
		t.Fatal("public contract owner identity was not admitted to the Manager graph")
	}
	privateCall, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": map[string]any{"name": "knowledge_explain", "arguments": map[string]any{"managerId": ordersManager, "targetId": privateInventoryArtifact}},
	})
	if err != nil {
		t.Fatal(err)
	}
	privateCall = append(append(knowledgeMCPHandshake(t), privateCall...), '\n')
	protocolOut.Reset()
	err = knowledgeprotocol.Serve(bytes.NewReader(privateCall), &protocolOut, func(call knowledgeprotocol.ToolRequest) (any, error) {
		return operations.Knowledge(projectapp.KnowledgeOperation{Selection: selection, Scope: call.Scope, Query: call.Query, Records: call.Records})
	})
	if err != nil {
		t.Fatal(err)
	}
	responses = bytes.Split(bytes.TrimSpace(protocolOut.Bytes()), []byte{'\n'})
	if len(responses) != 2 {
		t.Fatalf("unexpected private MCP response sequence: %s", protocolOut.String())
	}
	var privateEnvelope struct {
		Result struct {
			Structured json.RawMessage `json:"structuredContent"`
			IsError    bool            `json:"isError"`
			Content    []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(responses[1], &privateEnvelope); err != nil {
		t.Fatalf("decode private MCP response: %v", err)
	}
	if !privateEnvelope.Result.IsError || len(privateEnvelope.Result.Structured) != 0 || len(privateEnvelope.Result.Content) != 1 || privateEnvelope.Result.Content[0].Type != "text" || privateEnvelope.Result.Content[0].Text != "knowledge query failed; check the explicit scope, target and bounded selectors" {
		t.Fatalf("private-scope failure was not generic and content-only: %s", protocolOut.String())
	}
	if bytes.Contains(protocolOut.Bytes(), mustJSON(t, privateInventoryArtifact)) {
		t.Fatalf("private-scope failure exposed target identity %q: %s", privateInventoryArtifact, protocolOut.String())
	}
}

func knowledgeMCPHandshake(t *testing.T) []byte {
	t.Helper()
	initialize, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 99, "method": "initialize",
		"params": map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "fixture", "version": "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	notification, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	if err != nil {
		t.Fatal(err)
	}
	return append(append(append(initialize, '\n'), notification...), '\n')
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
