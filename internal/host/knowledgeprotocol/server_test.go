package knowledgeprotocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectgraph"
)

func TestServeNegotiatesListsAndCallsClosedReadOnlyTools(t *testing.T) {
	var input strings.Builder
	writeMessage(t, &input, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"protocolVersion": latestProtocolVersion, "capabilities": map[string]any{"roots": map[string]any{"listChanged": true}, "sampling": map[string]any{}}, "clientInfo": map[string]string{"name": "fixture", "version": "1"}},
	})
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	writeMessage(t, &input, map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "tools/call",
		"params": map[string]any{"name": "knowledge_trace", "arguments": map[string]any{
			"managerId": "manager/orders", "targetId": "statement/order-cancel", "bidirectional": true,
			"maxDepth": 3, "maxSteps": 120, "maxResults": 40,
			"runId": "run-1", "explorationId": "explore-1", "sessionId": "session-1", "briefingHistory": true,
		}},
	})
	var output bytes.Buffer
	called := false
	err := Serve(strings.NewReader(input.String()), &output, func(request ToolRequest) (any, error) {
		called = true
		if request.Scope != (projectgraph.Selection{ManagerID: "manager/orders"}) {
			t.Fatalf("scope = %+v", request.Scope)
		}
		if request.Query.Action != projectgraph.ActionTrace || request.Query.TargetID != "statement/order-cancel" || request.Query.Reverse || !request.Query.Bidirectional || request.Query.MaxDepth != 3 || request.Query.MaxSteps != 120 || request.Query.MaxResults != 40 {
			t.Fatalf("query = %+v", request.Query)
		}
		if len(request.Records.RunIDs) != 1 || request.Records.RunIDs[0] != "run-1" || len(request.Records.ExplorationIDs) != 1 || request.Records.ExplorationIDs[0] != "explore-1" || len(request.Records.BrownfieldSessionIDs) != 1 || request.Records.BrownfieldSessionIDs[0] != "session-1" || !request.Records.IncludeBriefingHistory {
			t.Fatalf("records = %+v", request.Records)
		}
		return map[string]any{"ok": true}, nil
	})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	if !called {
		t.Fatal("tool handler was not called")
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("response count = %d, want initialize/list/call: %s", len(lines), output.String())
	}
	var initialized map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initialized); err != nil {
		t.Fatal(err)
	}
	if got := initialized["result"].(map[string]any)["protocolVersion"]; got != latestProtocolVersion {
		t.Fatalf("protocol version = %v", got)
	}
	var listed struct {
		Result struct {
			Tools []struct {
				Name        string         `json:"name"`
				InputSchema map[string]any `json:"inputSchema"`
				Annotations map[string]any `json:"annotations"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Result.Tools) != 6 {
		t.Fatalf("listed %d tools, want six", len(listed.Result.Tools))
	}
	for _, tool := range listed.Result.Tools {
		if tool.Annotations["readOnlyHint"] != true || tool.Annotations["destructiveHint"] != false {
			t.Fatalf("tool %q lacks read-only annotations: %+v", tool.Name, tool.Annotations)
		}
		if tool.InputSchema["additionalProperties"] != false {
			t.Fatalf("tool %q schema is not closed", tool.Name)
		}
	}
	var calledResponse struct {
		Result struct {
			Structured map[string]any `json:"structuredContent"`
			IsError    bool           `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[2]), &calledResponse); err != nil {
		t.Fatal(err)
	}
	if calledResponse.Result.IsError || calledResponse.Result.Structured["ok"] != true {
		t.Fatalf("tool result = %+v", calledResponse.Result)
	}
}

func TestToolArgumentsRejectUnknownPrivateAndConflictingScope(t *testing.T) {
	for _, args := range []map[string]any{
		{"managerId": "manager/orders", "repo": "C:/private"},
		{"managerId": "manager/orders", "projectScope": true},
		{"projectScope": false},
		{"projectScope": true, "bidirectional": true, "reverse": true, "targetId": "statement/a"},
		{"projectScope": true, "maxDepth": 33, "targetId": "statement/a"},
	} {
		data, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parseToolRequest("knowledge_trace", data); err == nil {
			t.Errorf("accepted invalid arguments: %s", data)
		}
	}
	if _, err := parseToolRequest("knowledge_graph", []byte(`{"projectScope":true,"targetId":"statement/a"}`)); err == nil {
		t.Fatal("accepted targetId for graph tool")
	}
	if _, err := parseToolRequest("knowledge_graph", []byte(`{"projectScope":true,"maxDepth":-1}`)); err == nil {
		t.Fatal("silently ignored an invalid traversal bound on graph tool")
	}
}

func TestDuplicateKeysAndCaseAliasesFailBeforeTheHandler(t *testing.T) {
	input := handshakeLines(t) +
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"knowledge_graph","arguments":{"managerId":"one","managerId":"two"}}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"knowledge_graph","arguments":{"managerId":"one","ManagerId":"two"}}}` + "\n" +
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"knowledge_graph","name":"knowledge_trace","arguments":{"projectScope":true}}}` + "\n"
	var output bytes.Buffer
	called := false
	if err := Serve(strings.NewReader(input), &output, func(ToolRequest) (any, error) { called = true; return map[string]any{"unexpected": true}, nil }); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("duplicate/alias keys reached the knowledge handler")
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 { // initialize response plus one error for each malformed call
		t.Fatalf("response count=%d output=%s", len(lines), output.String())
	}
	for _, line := range lines[1:] {
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatal(err)
		}
		if response["error"] == nil {
			t.Fatalf("invalid duplicate-key request did not fail: %s", line)
		}
	}
}

func TestDuplicateKeysInOpaqueCapabilitiesAreRejected(t *testing.T) {
	line := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{"roots":{"listChanged":true,"listChanged":false}},"clientInfo":{"name":"fixture","version":"1"}}}` + "\n"
	var output bytes.Buffer
	if err := Serve(strings.NewReader(line), &output, func(ToolRequest) (any, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &response); err != nil {
		t.Fatal(err)
	}
	if response["error"] == nil {
		t.Fatalf("accepted duplicate nested capability keys: %s", output.String())
	}
}

func TestToolHandlerFailuresAreRedacted(t *testing.T) {
	line := handshakeLines(t) + `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"knowledge_relations","arguments":{"managerId":"manager/orders","targetId":"private/statement/secret"}}}` + "\n"
	var output bytes.Buffer
	err := Serve(strings.NewReader(line), &output, func(ToolRequest) (any, error) {
		return nil, errors.New("private/statement/secret internal path C:/secret")
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "secret") || !strings.Contains(output.String(), "knowledge query failed") {
		t.Fatalf("handler error was not redacted: %s", output.String())
	}
}

func TestVersionFallbackAndConnectionLifecycle(t *testing.T) {
	var input strings.Builder
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "knowledge_graph", "arguments": map[string]any{"projectScope": true}}})
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "initialize", "params": map[string]any{"protocolVersion": "2026-01-01", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "future", "version": "1"}}})
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "initialize", "params": map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "fixture", "version": "1"}}})
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "ping"})
	var output bytes.Buffer
	called := false
	if err := Serve(strings.NewReader(input.String()), &output, func(ToolRequest) (any, error) { called = true; return nil, nil }); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 || called {
		t.Fatalf("lifecycle responses=%d handler called=%t output=%s", len(lines), called, output.String())
	}
	var before, initialized, duplicate, ping map[string]any
	for i, target := range []*map[string]any{&before, &initialized, &duplicate, &ping} {
		if err := json.Unmarshal([]byte(lines[i]), target); err != nil {
			t.Fatal(err)
		}
	}
	if before["error"].(map[string]any)["code"] != float64(-32002) {
		t.Fatalf("pre-initialize call was accepted: %s", lines[0])
	}
	if got := initialized["result"].(map[string]any)["protocolVersion"]; got != latestProtocolVersion {
		t.Fatalf("fallback protocol = %v", got)
	}
	if duplicate["error"].(map[string]any)["code"] != float64(-32600) {
		t.Fatalf("duplicate initialize was accepted: %s", lines[2])
	}
	if ping["result"] == nil {
		t.Fatalf("ping failed: %s", lines[3])
	}
}

func TestLegacySupportedVersionOmitsStructuredContent(t *testing.T) {
	var input strings.Builder
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "fixture", "version": "1"}}})
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	writeMessage(t, &input, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "knowledge_graph", "arguments": map[string]any{"projectScope": true}}})
	var output bytes.Buffer
	if err := Serve(strings.NewReader(input.String()), &output, func(ToolRequest) (any, error) { return map[string]any{"ok": true}, nil }); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("response count=%d output=%s", len(lines), output.String())
	}
	var init struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &init); err != nil || init.Result.ProtocolVersion != "2024-11-05" {
		t.Fatalf("legacy version negotiation: %+v err=%v", init, err)
	}
	var call struct {
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &call); err != nil {
		t.Fatal(err)
	}
	if _, exists := call.Result["structuredContent"]; exists {
		t.Fatalf("legacy response used unsupported structuredContent: %s", lines[1])
	}
}

func TestServeRejectsOversizedMessage(t *testing.T) {
	input := strings.Repeat("x", maxMessageBytes+1) + "\n"
	var output bytes.Buffer
	if err := Serve(strings.NewReader(input), &output, func(ToolRequest) (any, error) { return nil, nil }); err == nil {
		t.Fatal("accepted a message exceeding 1 MiB")
	}
}

func writeMessage(t *testing.T, b *strings.Builder, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(b, string(data))
}

func handshakeLines(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	writeMessage(t, &b, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "fixture", "version": "1"}}})
	writeMessage(t, &b, map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	return b.String()
}
