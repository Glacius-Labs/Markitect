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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
)

type fixtureRun struct {
	Run string `json:"run"`
}

type fixtureReport struct {
	Run string `json:"run"`
}

type fixtureWrite struct {
	Action string `json:"action,omitempty"`
	Name   string `json:"name"`
	Expect string `json:"expect,omitempty"`
	Write  bool   `json:"write,omitempty"`
}

func fixtureServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	Register(s, "status", "fixture status", false, func(ctx context.Context, r fixtureRun) (fixtureReport, error) {
		if r.Run == "missing" {
			return fixtureReport{}, projectrun.ErrNotFound
		}
		return fixtureReport{r.Run}, nil
	})
	return s
}

func TestClosedSchemasAndAuthority(t *testing.T) {
	s := fixtureServer(t)
	if _, err := New(""); err == nil {
		t.Fatal("implicit root accepted")
	}
	if len(s.Tools()) != 1 {
		t.Fatalf("New registered tools of its own: %+v", s.Tools())
	}
	for _, tool := range s.Tools() {
		if tool.InputSchema.(map[string]any)["additionalProperties"] != false {
			t.Fatalf("open %s", tool.Name)
		}
	}
	for _, args := range []string{`{"run":"r","root":"elsewhere"}`, `{"run":"r","repo":"elsewhere"}`, `{"run":"r","run":"other"}`, `{"Run":"r"}`, `{"run":null}`, `{}`, `[]`} {
		if _, err := s.Call(context.Background(), "status", []byte(args)); err == nil {
			t.Fatalf("accepted %s", args)
		}
	}
	if _, err := s.Call(context.Background(), "unknown", []byte(`{}`)); err == nil {
		t.Fatal("invented a tool")
	}
	if result, err := s.Call(context.Background(), "status", []byte(`{"run":"r"}`)); err != nil || result.IsError {
		t.Fatalf("valid call failed: %+v %v", result, err)
	}
}

func TestOmitAndEnumRestrictTheClosedSchema(t *testing.T) {
	s, _ := New(t.TempDir())
	calls := 0
	Register(s, "fixture", "write fixture", false, func(ctx context.Context, r fixtureWrite) (fixtureReport, error) {
		calls++
		return fixtureReport{r.Name}, nil
	}, WithOmit("write", "expect"), WithEnum("action", "create", "list"))
	properties := s.Tools()[0].InputSchema.(map[string]any)["properties"].(map[string]any)
	if _, ok := properties["write"]; ok {
		t.Fatalf("omitted field still in schema: %+v", properties)
	}
	if !reflect.DeepEqual(properties["action"].(map[string]any)["enum"], []string{"create", "list"}) {
		t.Fatalf("enum not applied: %+v", properties["action"])
	}
	for _, args := range []string{`{"name":"n","write":true}`, `{"name":"n","expect":"d"}`, `{"name":"n","action":"dismiss"}`} {
		if _, err := s.Call(context.Background(), "fixture", []byte(args)); err == nil {
			t.Fatalf("accepted %s", args)
		}
	}
	if _, err := s.Call(context.Background(), "fixture", []byte(`{"name":"n","action":"list"}`)); err != nil || calls != 1 {
		t.Fatalf("restricted call failed: %v (%d calls)", err, calls)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("omitting an unknown field did not panic")
		}
	}()
	Register(s, "fixture_bad_omit", "bad omit", false, func(ctx context.Context, r fixtureWrite) (fixtureReport, error) { return fixtureReport{}, nil }, WithOmit("missing"))
}

func TestDecodeArgumentsMatchesToolDecodingAndNamesTheField(t *testing.T) {
	in, err := DecodeArguments[fixtureWrite](json.RawMessage(`{"name":"n","write":true,"expect":"d"}`))
	if err != nil || in.Name != "n" || !in.Write || in.Expect != "d" {
		t.Fatalf("decode: %+v %v", in, err)
	}
	for args, want := range map[string]string{
		`{"write":true}`:          "name is required",
		`{"name":"n","extra":1}`:  "extra is not a known field",
		`{"name":1}`:              "name must be a string",
		`{"name":"n","name":"m"}`: "invalid JSON arguments",
	} {
		if _, err := DecodeArguments[fixtureWrite](json.RawMessage(args)); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: error %v, want %q", args, err, want)
		}
	}
}

func TestEmbeddedStructFieldsArePromotedInSchemas(t *testing.T) {
	type report struct {
		projectrun.RunSummary
		Next string `json:"next"`
	}
	raw, _ := json.Marshal(report{RunSummary: projectrun.RunSummary{ID: "r", Status: "planned"}, Next: "run"})
	var decoded any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if err := validate(decoded, schema(reflect.TypeFor[report]()), ""); err != nil {
		t.Fatalf("embedded fields not promoted: %v", err)
	}
}

func TestRecursiveTypesHaveFiniteSchemas(t *testing.T) {
	type node struct {
		Name     string `json:"name"`
		Children []node `json:"children"`
		Parent   *node  `json:"parent,omitempty"`
	}
	raw, _ := json.Marshal(node{Name: "root", Children: []node{{Name: "leaf", Children: []node{}}}})
	var decoded any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if err := validate(decoded, schema(reflect.TypeFor[node]()), ""); err != nil {
		t.Fatalf("recursive value rejected: %v", err)
	}
}

func TestSetupSchemaExposesClosedWindowsSandboxBackendEnum(t *testing.T) {
	setupSchema := schema(reflect.TypeFor[projectsetup.Options]())
	properties := setupSchema["properties"].(map[string]any)
	backendSchema := properties["windowsSandboxBackend"].(map[string]any)
	if backendSchema["type"] != "string" || !reflect.DeepEqual(backendSchema["enum"], []string{"mxc"}) {
		t.Fatalf("Windows sandbox backend schema = %#v, want optional closed enum mxc", backendSchema)
	}
	if err := validate("mxc", backendSchema, ""); err != nil {
		t.Fatalf("valid mxc backend rejected by schema: %v", err)
	}
	if err := validate("unsupported", backendSchema, ""); err == nil {
		t.Fatal("unsupported Windows sandbox backend passed the MCP schema")
	}
}

func TestAppServerEnvironmentModeSchemaIsClosed(t *testing.T) {
	modeSchema := schema(reflect.TypeFor[projectrun.AppServerEnvironmentMode]())
	if modeSchema["type"] != "string" || !reflect.DeepEqual(modeSchema["enum"], []string{"inherit"}) {
		t.Fatalf("App Server environment mode schema = %#v, want closed inherit enum", modeSchema)
	}
	if err := validate("inherit", modeSchema, ""); err != nil {
		t.Fatalf("valid inherited environment mode rejected: %v", err)
	}
	if err := validate("all", modeSchema, ""); err == nil {
		t.Fatal("unsupported environment mode passed the schema")
	}
}

func TestHostErrorTextIsNotForwarded(t *testing.T) {
	s, _ := New(t.TempDir())
	Register(s, "fixture_plan", "fixture plan", true, func(ctx context.Context, r fixtureRun) (fixtureReport, error) {
		return fixtureReport{}, errors.New("secret credential value at private path")
	})
	result, err := s.Call(context.Background(), "fixture_plan", []byte(`{"run":"r"}`))
	if err != nil || !result.IsError {
		t.Fatalf("failure was success: %+v %v", result, err)
	}
	encoded, _ := json.Marshal(result)
	if bytes.Contains(encoded, []byte("secret")) || bytes.Contains(encoded, []byte("private")) {
		t.Fatalf("Host error leaked: %s", encoded)
	}
	result, err = fixtureServer(t).Call(context.Background(), "status", []byte(`{"run":"missing"}`))
	if err != nil || !result.IsError || !strings.Contains(result.Content[0].Text, `"code":"notfound"`) {
		t.Fatalf("missing durable run was success: %+v %v", result, err)
	}
}

func TestStdioHandshakeCallCancellationAndStatus(t *testing.T) {
	s, _ := New(t.TempDir())
	s.SetInstructions("fixture instructions")
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
	if result["instructions"] != "fixture instructions" {
		t.Fatalf("instructions: %v", result["instructions"])
	}
	send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	send(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	if len(receive()["result"].(map[string]any)["tools"].([]any)) != 2 {
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
	s := fixtureServer(t)
	result, err := s.Call(context.Background(), "status", []byte(`{"run":"missing"}`))
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	d := json.NewDecoder(strings.NewReader(result.Content[0].Text))
	d.UseNumber()
	if err = d.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if err = validate(decoded, s.tools["status"].tool.OutputSchema.(map[string]any), ""); err != nil {
		t.Fatal(err)
	}
}

func TestPartialDurableReportSanitization(t *testing.T) {
	s, _ := New(t.TempDir())
	Register(s, "fixture_partial", "partial shared report", true, func(context.Context, fixtureRun) (projectrun.VerifyReport, error) {
		return projectrun.VerifyReport{RunID: "durable-id", CandidateID: "candidate", Checks: []projectrun.CheckResult{{Stdout: "secret-out", Stderr: "secret-err", Error: "secret-error", Command: []string{"token=secret"}, ExecutablePath: "private-path"}}}, errors.New("secret-host")
	})
	result, err := s.Call(context.Background(), "fixture_partial", []byte(`{"run":"durable-id"}`))
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
	if err := validate(decoded, s.tools["fixture_partial"].tool.OutputSchema.(map[string]any), ""); err != nil {
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
		if err := validate(decoded, schema(reflect.TypeOf(v)), ""); err != nil {
			t.Fatalf("%T schema: %v", v, err)
		}
	}
}

func TestUnsignedOutputSchema(t *testing.T) {
	if err := validate(json.Number("18446744073709551615"), schema(reflect.TypeFor[uint64]()), ""); err != nil {
		t.Fatal(err)
	}
	if err := validate(json.Number("18446744073709551616"), schema(reflect.TypeFor[uint64]()), ""); err == nil {
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
		if err := validate(output, schema(reflect.TypeFor[dto]()), ""); err != nil {
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
			d := classifyError(fmt.Errorf("outer private detail: %w", tc.err), "Fixture recovery.")
			if d.Code != tc.code || strings.Contains(d.Message, "secret") || strings.Contains(d.Recovery, "private") {
				t.Fatalf("unsafe or flattened diagnostic: %+v", d)
			}
		})
	}
	for _, code := range []string{"host_rejected", "cancelled", "busy", "encoding_failed", "stale", "notfound", "notrunnable", "locked"} {
		if d := diagnosticFor(code, "Fixture recovery."); !strings.Contains(d.Recovery, "Fixture recovery.") {
			t.Fatalf("%s lost the operation recovery: %+v", code, d)
		}
	}
	s, _ := New(t.TempDir())
	Register(s, "fixture_recovery", "recovery fixture", true, func(context.Context, fixtureRun) (fixtureReport, error) {
		return fixtureReport{}, errors.New("private")
	}, WithRecovery("Inspect the fixture with status."))
	Register(s, "fixture_default", "default recovery", true, func(context.Context, fixtureRun) (fixtureReport, error) {
		return fixtureReport{}, errors.New("private")
	})
	result, _ := s.Call(context.Background(), "fixture_recovery", []byte(`{"run":"r"}`))
	if !strings.Contains(result.Content[0].Text, "Inspect the fixture with status.") {
		t.Fatalf("registered recovery missing: %s", result.Content[0].Text)
	}
	result, _ = s.Call(context.Background(), "fixture_default", []byte(`{"run":"r"}`))
	if !strings.Contains(result.Content[0].Text, "refresh this operation") || strings.Contains(result.Content[0].Text, "project_") {
		t.Fatalf("generic recovery missing or names a removed tool: %s", result.Content[0].Text)
	}
}

func TestPublicErrorMapperAndRetainedStructuredValidation(t *testing.T) {
	s, _ := New(t.TempDir())
	type report struct {
		SessionID string   `json:"sessionId"`
		Findings  []string `json:"findings"`
		Error     string   `json:"error,omitempty"`
	}
	sentinel := errors.New("private selected-model detail")
	Register(s, "adopt_fixture", "typed application error fixture", true, func(context.Context, fixtureRun) (report, error) {
		return report{SessionID: "existing-session", Findings: []string{"missing owner for declared artifact"}, Error: "provider secret"}, sentinel
	}, WithPublicError(func(err error) *Diagnostic {
		if errors.Is(err, sentinel) {
			return &Diagnostic{Code: "stage_precondition", Message: "The acceptance stage needs reviewed ownership.", Recovery: "Correct the session's ownership findings, refresh its preview and expected digest, then accept the same session."}
		}
		return nil
	}))
	result, err := s.Call(context.Background(), "adopt_fixture", []byte(`{"run":"fixture-input"}`))
	if err != nil || !result.IsError {
		t.Fatal("mapped failure lost")
	}
	raw := result.Content[0].Text
	for _, want := range []string{"stage_precondition", "existing-session", "missing owner"} {
		if !strings.Contains(raw, want) {
			t.Fatalf("missing %s: %s", want, raw)
		}
	}
	for _, bad := range []string{"private", "provider secret"} {
		if strings.Contains(raw, bad) {
			t.Fatalf("leaked %s: %s", bad, raw)
		}
	}
	Register(s, "fixture_bad_mapper", "unsafe shape rejected", true, func(context.Context, fixtureRun) (report, error) { return report{}, sentinel }, WithPublicError(func(error) *Diagnostic { return &Diagnostic{Code: "bad code", Message: "message", Recovery: "repair"} }))
	result, _ = s.Call(context.Background(), "fixture_bad_mapper", []byte(`{"run":"fixture"}`))
	if !strings.Contains(result.Content[0].Text, "host_rejected") || strings.Contains(result.Content[0].Text, "bad code") {
		t.Fatal("invalid callback escaped")
	}
	Register(s, "fixture_sentinel_mapper", "sentinel cannot be hidden", true, func(context.Context, fixtureRun) (report, error) { return report{}, projectrun.ErrStale }, WithPublicError(func(error) *Diagnostic { t.Error("known sentinel reached custom mapper"); return nil }))
	result, _ = s.Call(context.Background(), "fixture_sentinel_mapper", []byte(`{"run":"fixture"}`))
	if !strings.Contains(result.Content[0].Text, `"code":"stale"`) {
		t.Fatal("stale classification lost")
	}
}
