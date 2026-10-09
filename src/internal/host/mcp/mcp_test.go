package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
)

func TestClosedSchemasAndAuthority(t *testing.T) {
	s, _ := New(t.TempDir(), projectapp.Operations{})
	if _, err := New("", projectapp.Operations{}); err == nil {
		t.Fatal("implicit root accepted")
	}
	tools := s.Tools()
	if len(tools) != 10 {
		t.Fatalf("tools: %d", len(tools))
	}
	for _, tool := range tools {
		if tool.InputSchema.(map[string]any)["additionalProperties"] != false {
			t.Fatalf("open %s", tool.Name)
		}
	}
	for _, args := range []string{`{"runId":"r","root":"elsewhere"}`, `{"runId":"r","repo":"elsewhere"}`, `{"runId":"r","runId":"other"}`, `{"RunID":"r"}`, `{"runId":null}`, `{}`, `[]`} {
		if _, err := s.Call(context.Background(), "project_status", []byte(args)); err == nil {
			t.Fatalf("accepted %s", args)
		}
	}
	for _, name := range []string{"project_setup", "project_explore", "project_brownfield", "unknown"} {
		if _, err := s.Call(context.Background(), name, []byte(`{}`)); err == nil {
			t.Fatalf("invented %s", name)
		}
	}
	for _, args := range []string{`{"goal":"g","executeAuthorized":false,"modelEdit":{"unknown":true}}`, `{"goal":"g","executeAuthorized":false,"executeAuthorized":true}`} {
		if _, err := s.Call(context.Background(), "project_plan", []byte(args)); err == nil {
			t.Fatalf("accepted %s", args)
		}
	}
	for _, args := range []string{`{"runId":"r"}`, `{"runId":"r","planId":"p","candidateId":"c","expectedVerificationDigest":null,"targetBranch":"b","expectedHead":"h","expectedWorktree":"w"}`} {
		if _, err := s.Call(context.Background(), "project_apply", []byte(args)); err == nil {
			t.Fatalf("Apply guards omitted: %s", args)
		}
	}
}

func TestSetupSchemaExposesClosedWindowsSandboxBackendEnum(t *testing.T) {
	setupSchema := schema(reflect.TypeFor[projectsetup.Options]())
	properties := setupSchema["properties"].(map[string]any)
	backendSchema := properties["windowsSandboxBackend"].(map[string]any)
	if backendSchema["type"] != "string" || !reflect.DeepEqual(backendSchema["enum"], []string{"mxc"}) {
		t.Fatalf("Windows sandbox backend schema = %#v, want optional closed enum mxc", backendSchema)
	}
	if err := validate("mxc", backendSchema); err != nil {
		t.Fatalf("valid mxc backend rejected by schema: %v", err)
	}
	if err := validate("unsupported", backendSchema); err == nil {
		t.Fatal("unsupported Windows sandbox backend passed the MCP schema")
	}
}

func TestAppServerEnvironmentModeSchemaIsClosed(t *testing.T) {
	modeSchema := schema(reflect.TypeFor[projectrun.AppServerEnvironmentMode]())
	if modeSchema["type"] != "string" || !reflect.DeepEqual(modeSchema["enum"], []string{"inherit"}) {
		t.Fatalf("App Server environment mode schema = %#v, want closed inherit enum", modeSchema)
	}
	if err := validate("inherit", modeSchema); err != nil {
		t.Fatalf("valid inherited environment mode rejected: %v", err)
	}
	if err := validate("all", modeSchema); err == nil {
		t.Fatal("unsupported environment mode passed the schema")
	}
}

func TestRealHostPlanSelectionAndRedactedFailure(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git: %s %v", out, err)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "mcp-fixture")
	git("config", "user.email", "mcp@example.test")
	git("config", "user.name", "MCP Fixture")
	os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture"), 0600)
	git("add", "README.md")
	git("commit", "-m", "fixture")
	base := git("rev-parse", "HEAD")
	var gotRoot, gotBase string
	calls := 0
	o := projectapp.Operations{Host: projectrun.Host{FromSnapshot: func(string, *projectrun.Snapshot) (*projectrun.Project, error) {
		return nil, errors.New("unexpected snapshot")
	}, PlanEdit: func(*projectrun.Project, projectrun.Mutation) (projectrun.EditPlan, error) {
		return projectrun.EditPlan{}, errors.New("unexpected edit")
	}, Load: func(r, b string) (*projectrun.Project, error) {
		calls++
		gotRoot, gotBase = r, b
		return nil, errors.New("secret credential value at private path")
	}}}
	s, _ := New(root, o)
	args, _ := json.Marshal(projectrun.PlanRequest{Goal: "bounded", BaseRevision: base, ExecuteAuthorized: false})
	result, err := s.Call(context.Background(), "project_plan", args)
	if err != nil || !result.IsError || calls != 1 || gotRoot != root || gotBase != base {
		t.Fatalf("mapping: %+v %v %d %s %s", result, err, calls, gotRoot, gotBase)
	}
	encoded, _ := json.Marshal(result)
	if bytes.Contains(encoded, []byte("secret")) {
		t.Fatal("Host error leaked")
	}
	_, err = o.Plan(projectapp.PlanOperation{Selection: projectapp.Selection{Root: root, Revision: base}, Request: projectrun.PlanRequest{Goal: "bounded", BaseRevision: base}})
	if err == nil || calls != 2 {
		t.Fatal("shared Host parity failed")
	}
	result, err = s.Call(context.Background(), "project_status", []byte(`{"runId":"missing"}`))
	if err != nil || !result.IsError {
		t.Fatal("missing durable run was success")
	}
}

func TestStdioHandshakeCallCancellationAndStatus(t *testing.T) {
	s, _ := New(t.TempDir(), projectapp.Operations{})
	started := make(chan struct{})
	cancelled := make(chan struct{})
	type input struct {
		RunID string `json:"runId"`
	}
	type report struct {
		RunID string `json:"runId"`
	}
	Register(s, "fixture_wait", "protocol fixture", true, func(ctx context.Context, r input) (report, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return report{}, ctx.Err()
	})
	Register(s, "fixture_status", "protocol fixture status", false, func(ctx context.Context, r input) (report, error) { return report{r.RunID}, nil })
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	done := make(chan error, 1)
	go func() { done <- s.Serve(context.Background(), inR, outW); outW.Close() }()
	defer inW.Close()
	defer outR.Close()
	reader := bufio.NewReader(outR)
	send := func(line string) {
		t.Helper()
		if _, err := io.WriteString(inW, line+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	receive := func() map[string]any {
		t.Helper()
		line, err := reader.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		var r map[string]any
		if json.Unmarshal(line, &r) != nil {
			t.Fatal(string(line))
		}
		return r
	}
	send(`{"jsonrpc":"2.0","id":0,"method":"tools/list"}`)
	if receive()["error"].(map[string]any)["code"] != float64(-32600) {
		t.Fatal("preinit tools allowed")
	}
	for _, params := range []string{`{"protocolVersion":"2025-11-25","clientInfo":{"name":"fixture","version":"1"}}`, `{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"fixture"}}`} {
		send(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":` + params + `}`)
		if receive()["error"] == nil {
			t.Fatal("missing initialize fields accepted")
		}
	}
	send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"fixture","version":"1"}}}`)
	r := receive()
	result := r["result"].(map[string]any)
	if result["protocolVersion"] != ProtocolVersion || result["capabilities"].(map[string]any)["tasks"] != nil {
		t.Fatal("false capabilities")
	}
	send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	send(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	if len(receive()["result"].(map[string]any)["tools"].([]any)) != 12 {
		t.Fatal("discovery")
	}
	send(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"fixture_wait","arguments":{"runId":"durable-1"}}}`)
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("not started")
	}
	send(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"fixture_status","arguments":{"runId":"durable-1"}}}`)
	if receive()["id"] != float64(4) {
		t.Fatal("status blocked")
	}
	send(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"fixture_wait","arguments":{"runId":"durable-2"}}}`)
	if !receive()["result"].(map[string]any)["isError"].(bool) {
		t.Fatal("concurrent mutation allowed")
	}
	send(`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":3,"reason":"stop"}}`)
	r = receive()
	if r["id"] != float64(3) || !r["result"].(map[string]any)["isError"].(bool) {
		t.Fatal("cancellation result")
	}
	<-cancelled
	send(`{"jsonrpc":"2.0","id":6,"method":"resources/list"}`)
	if receive()["error"] == nil {
		t.Fatal("invented resources")
	}
	inW.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown hung")
	}
}

func TestOutputSchemaMatchesStructuredJSON(t *testing.T) {
	s, _ := New(t.TempDir(), projectapp.Operations{})
	result, err := s.Call(context.Background(), "project_status", []byte(`{"runId":"missing"}`))
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	d := json.NewDecoder(strings.NewReader(result.Content[0].Text))
	d.UseNumber()
	if err = d.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if err = validate(decoded, s.tools["project_status"].tool.OutputSchema.(map[string]any)); err != nil {
		t.Fatal(err)
	}
}

func TestPartialDurableReportSanitization(t *testing.T) {
	s, _ := New(t.TempDir(), projectapp.Operations{})
	Register(s, "fixture_partial", "partial shared report", true, func(context.Context, runInput) (projectrun.VerifyReport, error) {
		return projectrun.VerifyReport{RunID: "durable-id", CandidateID: "candidate", Checks: []projectrun.CheckResult{{Stdout: "secret-out", Stderr: "secret-err", Error: "secret-error", Command: []string{"token=secret"}, ExecutablePath: "private-path"}}}, errors.New("secret-host")
	})
	result, err := s.Call(context.Background(), "fixture_partial", []byte(`{"runId":"durable-id"}`))
	if err != nil || !result.IsError {
		t.Fatal("partial failure lost")
	}
	raw, _ := json.Marshal(result)
	if bytes.Contains(raw, []byte("secret")) || bytes.Contains(raw, []byte("private-path")) || !bytes.Contains(raw, []byte("durable-id")) {
		t.Fatalf("sanitization/handle failure: %s", raw)
	}
	var decoded any
	d := json.NewDecoder(strings.NewReader(result.Content[0].Text))
	d.UseNumber()
	d.Decode(&decoded)
	if err := validate(decoded, s.tools["fixture_partial"].tool.OutputSchema.(map[string]any)); err != nil {
		t.Fatal(err)
	}
}

func TestAllLifecycleOutputSchemas(t *testing.T) {
	for _, v := range []any{projectrun.PlanRecord{}, projectrun.RunReport{}, projectrun.StatusReport{}, projectrun.VerifyReport{}, projectrun.FullVerifyReport{}, projectrun.ApplyPreflight{}, projectrun.ApplyReport{}, projectrun.DeliverReport{}} {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var decoded any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		d.Decode(&decoded)
		if err := validate(decoded, schema(reflect.TypeOf(v))); err != nil {
			t.Fatalf("%T schema: %v", v, err)
		}
	}
}

func TestUnsignedOutputSchema(t *testing.T) {
	if err := validate(json.Number("18446744073709551615"), schema(reflect.TypeFor[uint64]())); err != nil {
		t.Fatal(err)
	}
	if err := validate(json.Number("18446744073709551616"), schema(reflect.TypeFor[uint64]())); err == nil {
		t.Fatal("out of range integer accepted")
	}
}

func TestRawMessageJSONValueAndByteBase64Schemas(t *testing.T) {
	type dto struct {
		Options json.RawMessage `json:"options"`
		Bytes   []byte          `json:"bytes"`
	}
	for _, value := range []string{`{"model":"fixture","settings":{"rights":true}}`, `["fixture",1]`, `"fixture"`, `42`, `true`, `null`} {
		original := dto{Options: json.RawMessage(value), Bytes: []byte{0, 255, 1}}
		raw, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}
		var decoded dto
		if err := decodeTyped(raw, schema(reflect.TypeFor[dto]()), &decoded); err != nil {
			t.Fatalf("RawMessage %s rejected: %v", value, err)
		}
		if !bytes.Equal(decoded.Options, original.Options) || !bytes.Equal(decoded.Bytes, original.Bytes) {
			t.Fatalf("value changed: %+v", decoded)
		}
		var output any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if err := d.Decode(&output); err != nil {
			t.Fatal(err)
		}
		if err := validate(output, schema(reflect.TypeFor[dto]())); err != nil {
			t.Fatalf("encoded output schema mismatch: %v", err)
		}
		if !bytes.Contains(raw, []byte(`"bytes":"AP8B"`)) {
			t.Fatalf("byte encoding changed: %s", raw)
		}
	}
	if schema(reflect.TypeFor[[]byte]())["type"] != "string" {
		t.Fatal("byte schema changed")
	}
	if len(schema(reflect.TypeFor[json.RawMessage]())) != 0 {
		t.Fatal("RawMessage is not a JSON-value schema")
	}
	var bad dto
	if err := decodeTyped([]byte(`{"options":{},"bytes":[0,255,1]}`), schema(reflect.TypeFor[dto]()), &bad); err == nil {
		t.Fatal("array accepted for base64 bytes")
	}
}

func TestSafeHostClassificationAndOperationRecovery(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{projectrun.ErrStale, "stale"}, {projectrun.ErrLocked, "locked"}, {projectrun.ErrNotFound, "notfound"}, {projectrun.ErrNotRunnable, "notrunnable"}, {fs.ErrNotExist, "notfound"}, {context.Canceled, "cancelled"}, {context.DeadlineExceeded, "cancelled"},
		{errors.New("project model has structural error findings"), "model_invalid"},
		{errors.New("project Host frontend is incomplete"), "config_invalid"},
		{errors.New("apply requires exact verification digest, target branch, HEAD and working-file digest"), "precondition_failed"},
		{fmt.Errorf("load selected project revision: %w", errors.New("provider secret")), "selection_failed"},
		{errors.New("provider secret"), "host_rejected"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			d := classifyError("project_run", fmt.Errorf("outer private detail: %w", tc.err))
			if d.Code != tc.code || strings.Contains(d.Message, "secret") || strings.Contains(d.Recovery, "private") {
				t.Fatalf("unsafe or flattened diagnostic: %+v", d)
			}
		})
	}
	for _, operation := range []string{"project_setup", "project_doctor", "project_edit", "project_explore", "project_readiness", "project_brownfield_accept", "project_plan", "project_full_verify"} {
		for _, code := range []string{"host_rejected", "cancelled", "busy", "encoding_failed", "stale", "notfound", "notrunnable"} {
			d := diagnosticFor(operation, code)
			if strings.Contains(d.Recovery, "project_status") || strings.Contains(d.Recovery, "runId") {
				t.Fatalf("pre-run action invented handle: %s %+v", operation, d)
			}
		}
	}
	if !strings.Contains(diagnosticFor("project_apply", "stale").Recovery, "preflight") {
		t.Fatal("Apply freshness recovery omitted")
	}
	if strings.Contains(diagnosticFor("project_status", "notfound").Recovery, "Inspect project_status") {
		t.Fatal("notfound status recovery loops")
	}
	if !strings.Contains(diagnosticFor("project_deliver", "host_rejected").Recovery, "If the partial report contains a runId") {
		t.Fatal("delivery invented run")
	}
}

func TestPublicErrorMapperAndRetainedStructuredValidation(t *testing.T) {
	s, _ := New(t.TempDir(), projectapp.Operations{})
	type report struct {
		SessionID string   `json:"sessionId"`
		Findings  []string `json:"findings"`
		Error     string   `json:"error,omitempty"`
	}
	sentinel := errors.New("private selected-model detail")
	Register(s, "project_brownfield_fixture", "typed application error fixture", true, func(context.Context, runInput) (report, error) {
		return report{SessionID: "existing-session", Findings: []string{"missing owner for declared artifact"}, Error: "provider secret"}, sentinel
	}, func(err error) *Diagnostic {
		if errors.Is(err, sentinel) {
			return &Diagnostic{Code: "stage_precondition", Message: "The acceptance stage needs reviewed ownership.", Recovery: "Correct the session's ownership findings, refresh its preview and expected digest, then accept the same session."}
		}
		return nil
	})
	result, err := s.Call(context.Background(), "project_brownfield_fixture", []byte(`{"runId":"fixture-input"}`))
	if err != nil || !result.IsError {
		t.Fatal("mapped failure lost")
	}
	raw := result.Content[0].Text
	for _, want := range []string{"stage_precondition", "existing-session", "missing owner"} {
		if !strings.Contains(raw, want) {
			t.Fatalf("missing %s: %s", want, raw)
		}
	}
	for _, bad := range []string{"private", "provider secret", "project_status", "runId"} {
		if strings.Contains(raw, bad) {
			t.Fatalf("leaked or invented %s: %s", bad, raw)
		}
	}
	Register(s, "fixture_bad_mapper", "unsafe shape rejected", true, func(context.Context, runInput) (report, error) { return report{}, sentinel }, func(error) *Diagnostic { return &Diagnostic{Code: "bad code", Message: "message", Recovery: "repair"} })
	result, _ = s.Call(context.Background(), "fixture_bad_mapper", []byte(`{"runId":"fixture"}`))
	if !strings.Contains(result.Content[0].Text, "host_rejected") || strings.Contains(result.Content[0].Text, "bad code") {
		t.Fatal("invalid callback escaped")
	}
	Register(s, "fixture_sentinel_mapper", "sentinel cannot be hidden", true, func(context.Context, runInput) (report, error) { return report{}, projectrun.ErrStale }, func(error) *Diagnostic { t.Error("known sentinel reached custom mapper"); return nil })
	result, _ = s.Call(context.Background(), "fixture_sentinel_mapper", []byte(`{"runId":"fixture"}`))
	if !strings.Contains(result.Content[0].Text, `"code":"stale"`) {
		t.Fatal("stale classification lost")
	}
}
