package projectcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/application/mcp"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
	"go.yaml.in/yaml/v3"
)

func externalContractCases() map[string]contractCase {
	return map[string]contractCase{
		"packet-export": {externalPacketExportHappy, externalPacketExportNoWrite},
		"packet-list":   {externalPacketListHappy, externalPacketListNoWrite},
		"packet-submit": {externalPacketSubmitHappy, externalPacketSubmitNoWrite},
	}
}

func newExternalContract(t *testing.T, surface string) (*contractCaller, projectapp.Operations, *testkit.Repo, projectrun.PlanRecord, *contractRuntimeInvoker) {
	t.Helper()
	caller, ops, repo, invoker := newRuntimeContract(t, surface, false)
	plan := contractPersistPlan(t, caller, ops)
	return caller, ops, repo, plan, invoker
}

func externalPendingPacket(t *testing.T, ops projectapp.Operations, repo *testkit.Repo, plan projectrun.PlanRecord) projectrun.ExternalPacket {
	t.Helper()
	report, err := ops.ExternalStartOrResume(context.Background(), projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if (err != nil && !errors.Is(err, projectrun.ErrAwaitingExternal)) || report.ID != plan.ID || report.Status != projectrun.StatusAwaitingExternal || !report.ExternalExecution {
		t.Fatalf("normal scheduler did not stop at its external packet: report=%+v err=%v", report, err)
	}
	packets, err := ops.ExternalPackets(projectapp.RunOperation{Root: repo.Dir, RunID: plan.ID})
	if err != nil || len(packets) != 1 || packets[0].RunID != plan.ID || packets[0].Digest == "" || len(packets[0].InvocationBytes) == 0 {
		t.Fatalf("external scheduler produced no complete pending packet: packets=%+v err=%v", packets, err)
	}
	return packets[0]
}

func externalPacketExportHappy(t *testing.T, surface string) {
	caller, _, _, plan, invoker := newExternalContract(t, surface)
	data := caller.Success("packet-export", []string{plan.ID, "--execute"}, map[string]any{"run": plan.ID, "execute": true})
	result := decodeOutput[struct {
		Run     projectrun.RunReport        `json:"run"`
		Packets []projectrun.ExternalPacket `json:"packets"`
	}](t, data)
	if result.Run.ID != plan.ID || result.Run.Status != projectrun.StatusAwaitingExternal || !result.Run.ExternalExecution || len(result.Packets) != 1 {
		t.Fatalf("packet-export did not start the normal scheduler in external mode: %+v", result)
	}
	if result.Packets[0].RunID != plan.ID || result.Packets[0].PlanDigest != plan.Digest || result.Packets[0].Digest == "" || len(result.Packets[0].InvocationBytes) == 0 {
		t.Fatalf("packet-export returned an unbound packet: %+v", result.Packets[0])
	}
	if invoker.calls != 0 {
		t.Fatalf("external export invoked the configured runtime: calls=%d", invoker.calls)
	}
	assertExternalPending(t, caller, plan.ID, result.Packets[0])
}

func externalPacketExportNoWrite(t *testing.T, surface string) {
	caller, _, _, plan, invoker := newExternalContract(t, surface)
	caller.NoWrite(func() {
		contractUsageRejected(t, caller, "packet-export", []string{plan.ID}, map[string]any{"run": plan.ID})
	})
	if invoker.calls != 0 {
		t.Fatalf("packet-export without --execute invoked the configured runtime: calls=%d", invoker.calls)
	}
}

func externalPacketListHappy(t *testing.T, surface string) {
	caller, ops, repo, plan, invoker := newExternalContract(t, surface)
	packet := externalPendingPacket(t, ops, repo, plan)
	assertExternalPending(t, caller, plan.ID, packet)
	if invoker.calls != 0 {
		t.Fatalf("packet-list invoked the configured runtime: calls=%d", invoker.calls)
	}
}

func externalPacketListNoWrite(t *testing.T, surface string) {
	caller, ops, repo, plan, invoker := newExternalContract(t, surface)
	packet := externalPendingPacket(t, ops, repo, plan)
	caller.NoWrite(func() { assertExternalPending(t, caller, plan.ID, packet) })
	if invoker.calls != 0 {
		t.Fatalf("packet-list invoked the configured runtime: calls=%d", invoker.calls)
	}
}

func TestExternalPacketListRejectsStaleSourceAndMethodPolicy(t *testing.T) {
	caller, ops, repo, plan, invoker := newExternalContract(t, "cli")
	packet := externalPendingPacket(t, ops, repo, plan)

	// A changed selected source must stop the CLI from listing the packet.
	repo.Write("src/value.txt", "changed after packet\n")
	beforeSourceList := contractInventory(t, repo.Dir)
	code, output, stderr := runCLIWith(t, caller.ops, "packet-list", "--repo", caller.root, plan.ID)
	if code == 0 || strings.TrimSpace(output) != "" || !strings.Contains(strings.ToLower(stderr), "stale") {
		t.Fatalf("packet-list did not reject the changed source with a stale diagnostic: code=%d output=%s stderr=%s", code, output, stderr)
	}
	if !reflect.DeepEqual(beforeSourceList, contractInventory(t, repo.Dir)) {
		t.Fatal("stale-source packet-list wrote project files or directories")
	}
	if invoker.calls != 0 {
		t.Fatalf("stale-source packet-list invoked the configured runtime: calls=%d", invoker.calls)
	}

	// Restore the exact source bytes, then change only the durable method policy
	// so this second listing independently exercises its policy binding.
	repo.Write("src/value.txt", "initial\n")
	runtime, err := projectconfig.LoadRuntime(repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	runtime.Limits.MaxParallel = 2
	runtimeBytes, err := yaml.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	repo.Write(projectconfig.RuntimePath, string(runtimeBytes))
	server, err := newMCPServer(env{root: repo.Dir, ops: ops, readOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	beforePolicyList := contractInventory(t, repo.Dir)
	result, err := server.Call(context.Background(), "packet-list", mustRaw(t, map[string]any{"run": plan.ID}))
	if err != nil {
		t.Fatalf("read-only packet-list arguments failed: %v", err)
	}
	if err := server.ValidateResult("packet-list", result); err != nil {
		t.Fatalf("stale read-only packet-list result violated its schema: %v", err)
	}
	outcome := decodeOutput[mcp.Outcome[json.RawMessage]](t, mustRaw(t, result.StructuredContent))
	diagnostic := diagnosticText(outcome.Diagnostic)
	if !result.IsError || outcome.Data != nil || outcome.Diagnostic == nil || outcome.Diagnostic.Code == "" ||
		!strings.Contains(strings.ToLower(diagnostic), "stale") {
		t.Fatalf("read-only packet-list did not reject changed method policy with a stale diagnostic and no packet data: result=%+v outcome=%+v", result, outcome)
	}
	if !reflect.DeepEqual(beforePolicyList, contractInventory(t, repo.Dir)) {
		t.Fatal("stale-policy read-only packet-list wrote project files or directories")
	}
	if invoker.calls != 0 {
		t.Fatalf("stale-policy packet-list invoked the configured runtime: calls=%d", invoker.calls)
	}
	if packet.ID == "" {
		t.Fatal("fixture did not create a pending packet")
	}
}

func assertExternalPending(t *testing.T, caller *contractCaller, runID string, expected projectrun.ExternalPacket) {
	t.Helper()
	data := caller.Success("packet-list", []string{runID}, map[string]any{"run": runID})
	packets := decodeOutput[[]projectrun.ExternalPacket](t, data)
	if len(packets) != 1 || !externalPacketMatches(packets[0], expected) {
		t.Fatalf("packet-list did not return the unchanged pending packet: got=%+v want=%+v", packets, expected)
	}
}

func externalPacketSubmitHappy(t *testing.T, surface string) {
	caller, ops, repo, plan, invoker := newExternalContract(t, surface)
	packet := externalPendingPacket(t, ops, repo, plan)
	response := externalResponseBytes(t, packet)
	responsePath := writeExternalResponseFile(t, repo.Dir, response)
	var responseInput any
	if err := json.Unmarshal(response, &responseInput); err != nil {
		t.Fatal(err)
	}
	expectedStoredResponse := response
	if surface == "mcp" {
		// MCP transports a JSON value, so the request contains the compact
		// serialization of the decoded map rather than the fixture's original
		// struct field order. The product stores the exact bytes it receives.
		wireResponse, err := json.Marshal(responseInput)
		if err != nil {
			t.Fatal(err)
		}
		expectedStoredResponse = wireResponse
	}
	data := caller.Success("packet-submit", []string{plan.ID, "--packet", packet.ID, "--expect", packet.Digest, "--response", responsePath, "--execute"}, map[string]any{
		"run": plan.ID, "packet": packet.ID, "expect": packet.Digest, "response": responseInput, "execute": true,
	})
	result := decodeOutput[struct {
		PacketID      string `json:"packetId"`
		Digest        string `json:"digest"`
		Stored        bool   `json:"stored"`
		NewSubmission bool   `json:"newSubmission"`
	}](t, data)
	if result.PacketID != packet.ID || result.Digest != packet.Digest || !result.Stored || !result.NewSubmission {
		t.Fatalf("packet-submit did not durably accept the exact packet response: %+v", result)
	}
	caller.NoWrite(func() {
		repeat := decodeOutput[struct {
			PacketID      string `json:"packetId"`
			Digest        string `json:"digest"`
			Stored        bool   `json:"stored"`
			NewSubmission bool   `json:"newSubmission"`
		}](t, caller.Success("packet-submit", []string{plan.ID, "--packet", packet.ID, "--expect", packet.Digest, "--response", responsePath, "--execute"}, map[string]any{
			"run": plan.ID, "packet": packet.ID, "expect": packet.Digest, "response": responseInput, "execute": true,
		}))
		if repeat.PacketID != packet.ID || repeat.Digest != packet.Digest || !repeat.Stored || repeat.NewSubmission {
			t.Fatalf("packet-submit did not report an idempotent stored duplicate: %+v", repeat)
		}
	})
	storedPath := filepath.Join(repo.Dir, ".markitect", "records", "run", plan.ID, "external", "responses", packet.ID+".json")
	stored, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("packet-submit did not persist its raw response: %v", err)
	}
	var submission struct {
		PacketID       string `json:"packetId"`
		PacketDigest   string `json:"packetDigest"`
		ResponseDigest string `json:"responseDigest"`
		Response       []byte `json:"response"`
	}
	if err := json.Unmarshal(stored, &submission); err != nil || submission.PacketID != packet.ID || submission.PacketDigest != packet.Digest || !reflect.DeepEqual(submission.Response, expectedStoredResponse) {
		t.Fatalf("stored raw response is not bound to the submitted packet: submission=%+v err=%v", submission, err)
	}
	if invoker.calls != 0 {
		t.Fatalf("packet-submit invoked the configured runtime: calls=%d", invoker.calls)
	}
}

func externalPacketSubmitNoWrite(t *testing.T, surface string) {
	caller, ops, repo, plan, invoker := newExternalContract(t, surface)
	packet := externalPendingPacket(t, ops, repo, plan)
	response := externalResponseBytes(t, packet)
	responsePath := writeExternalResponseFile(t, repo.Dir, response)
	var responseInput any
	if err := json.Unmarshal(response, &responseInput); err != nil {
		t.Fatal(err)
	}
	caller.NoWrite(func() {
		contractUsageRejected(t, caller, "packet-submit", []string{plan.ID, "--packet", packet.ID, "--expect", packet.Digest, "--response", responsePath}, map[string]any{
			"run": plan.ID, "packet": packet.ID, "expect": packet.Digest, "response": responseInput,
		})
	})
	if invoker.calls != 0 {
		t.Fatalf("packet-submit without --execute invoked the configured runtime: calls=%d", invoker.calls)
	}
}

func externalResponseBytes(t *testing.T, packet projectrun.ExternalPacket) []byte {
	t.Helper()
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: packet.Invocation.RunID, Nonce: packet.Invocation.Nonce,
		Role: packet.Invocation.Request.Role, InputDigest: packet.Invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: json.RawMessage(`{"status":"complete","summary":"external contract response"}`), Uncertainty: []string{},
	}
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestExternalPacketContractFailures(t *testing.T) {
	for _, surface := range []string{"cli", "mcp"} {
		t.Run(surface, func(t *testing.T) {
			caller, ops, repo, plan, invoker := newExternalContract(t, surface)
			packet := externalPendingPacket(t, ops, repo, plan)
			response := externalResponseBytes(t, packet)
			responsePath := writeExternalResponseFile(t, repo.Dir, response)
			var responseInput any
			if err := json.Unmarshal(response, &responseInput); err != nil {
				t.Fatal(err)
			}
			staleDigest := "sha256:" + strings.Repeat("0", 64)
			caller.NoWrite(func() {
				argv := []string{plan.ID, "--packet", packet.ID, "--expect", staleDigest, "--response", responsePath, "--execute"}
				input := map[string]any{
					"run": plan.ID, "packet": packet.ID, "expect": staleDigest, "response": responseInput, "execute": true,
				}
				result := contractResult{}
				detail := ""
				if surface == "cli" {
					code, output, stderr := runCLIWith(t, caller.ops, append([]string{"packet-submit", "--repo", caller.root}, argv...)...)
					result.Code = code
					detail = output + stderr
				} else {
					result = caller.Invoke("packet-submit", argv, input)
					detail = string(result.Data) + diagnosticText(result.Diagnostic)
				}
				if result.Code == 0 || !strings.Contains(strings.ToLower(detail), "digest") {
					t.Fatalf("stale packet digest was not rejected explicitly: code=%d details=%s diagnostic=%+v", result.Code, detail, result.Diagnostic)
				}
			})
			assertExternalPending(t, caller, plan.ID, packet)
			if invoker.calls != 0 {
				t.Fatalf("stale packet submission invoked the configured runtime: calls=%d", invoker.calls)
			}
		})
	}
}

func TestExternalPacketReadOnlyMCPContract(t *testing.T) {
	_, ops, repo, plan, invoker := newExternalContract(t, "cli")
	packet := externalPendingPacket(t, ops, repo, plan)
	server, err := newMCPServer(env{root: repo.Dir, ops: ops, readOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, tool := range server.Tools() {
		names = append(names, tool.Name)
	}
	assertContractNames(t, "read-only packet tools", []string{"packet-list"}, externalToolNames(names))
	before := contractInventory(t, repo.Dir)
	result, err := server.Call(context.Background(), "packet-list", mustRaw(t, map[string]any{"run": plan.ID}))
	if err != nil || result.IsError {
		t.Fatalf("read-only packet-list failed: result=%+v err=%v", result, err)
	}
	if err := server.ValidateResult("packet-list", result); err != nil {
		t.Fatal(err)
	}
	outcome := decodeOutput[mcp.Outcome[json.RawMessage]](t, mustRaw(t, result.StructuredContent))
	if outcome.Data == nil {
		t.Fatalf("read-only packet-list returned no data: %+v", outcome)
	}
	var packets []projectrun.ExternalPacket
	if err := json.Unmarshal(*outcome.Data, &packets); err != nil || len(packets) != 1 || !externalPacketMatches(packets[0], packet) {
		t.Fatalf("read-only packet-list returned different pending packets: packets=%+v err=%v", packets, err)
	}
	for _, name := range []string{"packet-export", "packet-submit"} {
		if _, err := server.Call(context.Background(), name, mustRaw(t, map[string]any{})); err == nil {
			t.Fatalf("read-only MCP unexpectedly registered %s", name)
		}
	}
	if !reflect.DeepEqual(before, contractInventory(t, repo.Dir)) {
		t.Fatal("read-only MCP packet operations wrote project files")
	}
	if invoker.calls != 0 {
		t.Fatalf("read-only packet operations invoked the configured runtime: calls=%d", invoker.calls)
	}
}

func externalPacketMatches(actual, expected projectrun.ExternalPacket) bool {
	// The digest binds the complete packet envelope; compare its transport
	// identity and exact protocol bytes explicitly across JSON round trips.
	return actual.ID == expected.ID && actual.RunID == expected.RunID && actual.Digest == expected.Digest &&
		actual.PlanDigest == expected.PlanDigest && actual.RequestDigest == expected.RequestDigest &&
		bytes.Equal(actual.InvocationBytes, expected.InvocationBytes)
}

func writeExternalResponseFile(t *testing.T, root string, response []byte) string {
	t.Helper()
	const relative = ".markitect/drafts/external-response.json"
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, response, 0o600); err != nil {
		t.Fatal(err)
	}
	return relative
}

func externalToolNames(names []string) []string {
	result := []string{}
	for _, name := range names {
		if strings.HasPrefix(name, "packet-") {
			result = append(result, name)
		}
	}
	return result
}

func diagnosticText(diagnostic *mcp.Diagnostic) string {
	if diagnostic == nil {
		return ""
	}
	return diagnostic.Message + " " + diagnostic.Code
}
