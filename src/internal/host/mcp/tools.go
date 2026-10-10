// Package mcp implements a local MCP adapter over typed project Host operations.
// It owns no workflow, provider invocation, or authorization policy.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
)

type Tool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  any             `json:"inputSchema"`
	OutputSchema any             `json:"outputSchema"`
	Annotations  map[string]bool `json:"annotations"`
}
type Diagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Recovery string `json:"recovery"`
}
type Outcome[R any] struct {
	Operation  string      `json:"operation"`
	Data       *R          `json:"data,omitempty"`
	Diagnostic *Diagnostic `json:"diagnostic,omitempty"`
}
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type CallResult struct {
	Content           []Content `json:"content"`
	StructuredContent any       `json:"structuredContent"`
	IsError           bool      `json:"isError"`
}
type binding struct {
	tool     Tool
	mutation bool
	call     func(context.Context, json.RawMessage) (CallResult, error)
}
type Server struct {
	root     string
	tools    map[string]binding
	mutation sync.Mutex
}

// New binds a server to one explicit root. Tool arguments cannot redirect Host authority.
// Register and Serve must not execute concurrently; composition owns registration.
func New(root string, o projectapp.Operations) (*Server, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("MCP requires an explicit project root")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	s := &Server{root: filepath.Clean(root), tools: map[string]binding{}}
	Register(s, "project_plan", "Plan against an explicit base; executeAuthorized records existing caller authority.", true, func(ctx context.Context, r projectrun.PlanRequest) (projectrun.PlanRecord, error) {
		return o.Plan(projectapp.PlanOperation{Selection: projectapp.Selection{Root: s.root, Revision: r.BaseRevision}, Request: r})
	})
	Register(s, "project_run", "Run an existing durable plan. Use its runId with status/resume/repair after interruption.", true, func(ctx context.Context, r runInput) (projectrun.RunReport, error) { return o.Run(ctx, s.run(r)) })
	Register(s, "project_resume", "Resume the same durable run without creating a replacement plan.", true, func(ctx context.Context, r runInput) (projectrun.RunReport, error) { return o.Resume(ctx, s.run(r)) })
	Register(s, "project_repair", "Repair the same durable run using existing Host repair semantics.", true, func(ctx context.Context, r runInput) (projectrun.RunReport, error) { return o.Repair(ctx, s.run(r)) })
	Register(s, "project_status", "Read the durable state of a selected run.", false, func(ctx context.Context, r runInput) (projectrun.StatusReport, error) { return o.Status(s.run(r)) })
	Register(s, "project_verify", "Verify the selected run and persist actual Host evidence.", true, func(ctx context.Context, r runInput) (projectrun.VerifyReport, error) { return o.Verify(ctx, s.run(r)) })
	Register(s, "project_full_verify", "Verify a selected revision; write selects persistence of evidence.", true, func(ctx context.Context, r projectrun.FullVerifyRequest) (projectrun.FullVerifyReport, error) {
		return o.FullVerify(ctx, projectapp.FullVerifyOperation{Root: s.root, Request: r})
	})
	Register(s, "project_preflight", "Read exact guarded Apply inputs for a selected run and candidate.", false, func(ctx context.Context, r preflightInput) (projectrun.ApplyPreflight, error) {
		return o.PreflightApply(projectapp.PreflightOperation{Root: s.root, RunID: r.RunID, CandidateID: r.CandidateID})
	})
	Register(s, "project_apply", "Apply only the exact reviewed, verified candidate with all freshness guards.", true, func(ctx context.Context, r projectrun.ApplyRequest) (projectrun.ApplyReport, error) {
		return o.Apply(projectapp.ApplyOperation{Root: s.root, Request: r})
	})
	Register(s, "project_deliver", "Advance acknowledged scope through existing durable delivery operations.", true, func(ctx context.Context, r projectrun.DeliverRequest) (projectrun.DeliverReport, error) {
		return o.Deliver(ctx, projectapp.DeliverOperation{Root: s.root, Request: r})
	})
	return s, nil
}

type runInput struct {
	RunID string `json:"runId"`
}
type preflightInput struct {
	RunID       string `json:"runId"`
	CandidateID string `json:"candidateId"`
}

func (s *Server) run(r runInput) projectapp.RunOperation {
	return projectapp.RunOperation{Root: s.root, RunID: r.RunID}
}

// Register adds a typed shared operation at composition time, for example setup
// once its application seam exists. It never executes a CLI subprocess.
// PublicErrorMapper is a composition-owned translation of known application
// errors into intentionally public diagnostics. Never return err.Error(), raw
// provider/check output, credentials or local machine paths from this callback.
// Return nil for unknown errors to retain the adapter's safe fallback.
type PublicErrorMapper func(error) *Diagnostic

func Register[T, R any](s *Server, name, description string, mutation bool, call func(context.Context, T) (R, error), publicError ...PublicErrorMapper) {
	if len(publicError) > 1 {
		panic("only one MCP public error mapper is supported")
	}
	if _, exists := s.tools[name]; exists {
		panic("duplicate MCP tool: " + name)
	}
	input := schema(reflect.TypeFor[T]())
	b := binding{tool: Tool{Name: name, Description: description, InputSchema: input, OutputSchema: schema(reflect.TypeFor[Outcome[R]]()), Annotations: map[string]bool{"readOnlyHint": !mutation, "destructiveHint": mutation, "openWorldHint": false}}, mutation: mutation}
	b.call = func(ctx context.Context, raw json.RawMessage) (CallResult, error) {
		var r T
		if err := decodeTyped(raw, input, &r); err != nil {
			return CallResult{}, err
		}
		if err := ctx.Err(); err != nil {
			return outcome[R](name, nil, "cancelled"), nil
		}
		data, err := call(ctx, r)
		if err != nil {
			// Partial Host reports retain durable run/candidate handles after failure.
			diagnostic := classifyError(name, err)
			// Cancellation and shared sentinel classifications cannot be hidden by
			// a custom mapper; it supplies additional application classifications.
			if diagnostic.Code == "host_rejected" && len(publicError) == 1 && publicError[0] != nil {
				if mapped := publicError[0](err); validPublicDiagnostic(mapped) {
					diagnostic = *mapped
				}
			}
			return diagnosticOutcome(name, &data, &diagnostic), nil
		}
		return outcome(name, &data, ""), nil
	}
	s.tools[name] = b
}
func outcome[R any](name string, data *R, code string) CallResult {
	var diagnostic *Diagnostic
	if code != "" {
		d := diagnosticFor(name, code)
		diagnostic = &d
	}
	return diagnosticOutcome(name, data, diagnostic)
}
func diagnosticOutcome[R any](name string, data *R, diagnostic *Diagnostic) CallResult {
	out := Outcome[R]{Operation: name, Data: data, Diagnostic: diagnostic}
	raw, err := json.Marshal(out)
	if err != nil {
		d := diagnosticFor(name, "encoding_failed")
		out = Outcome[R]{Operation: name, Diagnostic: &d}
		raw, _ = json.Marshal(out)
	}
	var normalized any
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	_ = d.Decode(&normalized)
	sanitize(normalized)
	raw, _ = json.Marshal(normalized)
	return CallResult{Content: []Content{{Type: "text", Text: string(raw)}}, StructuredContent: normalized, IsError: out.Diagnostic != nil}
}

// Raw check streams, command arguments and local errors stay in durable Host
// storage. Status, digests and product-relative ownership paths remain usable.
func sanitize(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, value := range x {
			switch k {
			case "stdout", "stderr", "error", "executablePath", "persistedPath", "root":
				if _, ok := value.(string); ok {
					x[k] = ""
				}
			case "command":
				switch value.(type) {
				case []any:
					x[k] = []any{}
				case string:
					x[k] = ""
				}
			default:
				sanitize(value)
			}
		}
	case []any:
		for _, item := range x {
			sanitize(item)
		}
	}
}
func (s *Server) Tools() []Tool {
	names := make([]string, 0, len(s.tools))
	for n := range s.tools {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Tool, 0, len(names))
	for _, n := range names {
		out = append(out, s.tools[n].tool)
	}
	return out
}

// Call delegates validation and execution. Mutations are serialized by rejection
// while an operation is active; read-only status remains available.
func (s *Server) Call(ctx context.Context, name string, args json.RawMessage) (CallResult, error) {
	b, ok := s.tools[name]
	if !ok {
		return CallResult{}, errors.New("unknown or unavailable tool")
	}
	if b.mutation {
		if !s.mutation.TryLock() {
			return outcome[struct{}](name, nil, "busy"), nil
		}
		defer s.mutation.Unlock()
	}
	return b.call(ctx, args)
}

// Classifications inspect known sentinels and a small set of product-owned
// boundary messages. None forwards the cause's text. They guide repair; Host
// validation and guards remain the authority for any later operation.
func classifyError(operation string, err error) Diagnostic {
	code := "host_rejected"
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		code = "cancelled"
	case errors.Is(err, projectrun.ErrStale):
		code = "stale"
	case errors.Is(err, projectrun.ErrLocked):
		code = "locked"
	case errors.Is(err, projectrun.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		code = "notfound"
	case errors.Is(err, projectrun.ErrNotRunnable):
		code = "notrunnable"
	default:
		// Check wrapper layers individually so generic provider text cannot become
		// the public message. Additional typed application errors use the mapper.
		for cause := err; cause != nil; cause = errors.Unwrap(cause) {
			text := cause.Error()
			switch text {
			case "project Host frontend is incomplete", "apply requires Host and runtime invoker", "Host load is required":
				code = "config_invalid"
			case "project model has structural error findings", "candidate model analysis did not succeed", "validated model edit is missing fixed digests or has structural findings", "whole-repository coverage is unaccounted; classify unknown paths before planning implementation":
				code = "model_invalid"
			case "a bounded project goal is required", "selected revision conflicts with plan request baseRevision", "selected project root is required", "project run requires a non-provisional snapshot", "apply requires exact verification digest, target branch, HEAD and working-file digest", "candidate has no matching successful verification", "candidate lacks matching successful verification", "apply run and plan IDs do not match", "requested candidate is not the integrated candidate for this run", "requested candidate is not the latest verified integrated candidate":
				code = "precondition_failed"
			default:
				if strings.HasPrefix(text, "load selected project revision: ") || strings.HasPrefix(text, "resolve fixed default base revision: ") {
					code = "selection_failed"
				}
			}
			if code != "host_rejected" {
				break
			}
		}
	}
	return diagnosticFor(operation, code)
}

func diagnosticFor(operation, code string) Diagnostic {
	messages := map[string]string{
		"host_rejected":       "The shared Host rejected this operation; inspect its structured validation report and selected inputs.",
		"stale":               "The saved operation no longer matches the selected project inputs or target state.",
		"locked":              "Another Host writer holds the project runtime lock.",
		"notfound":            "A required selected record or file was not found.",
		"notrunnable":         "The selected run does not satisfy the Host execution preconditions.",
		"cancelled":           "The operation was cancelled or its deadline expired; completion must be checked before retrying.",
		"busy":                "Another mutation is active on this MCP server.",
		"selection_failed":    "The Host could not load the selected repository revision.",
		"model_invalid":       "The selected model or coverage has unresolved structural findings.",
		"config_invalid":      "The operation's Host or runtime configuration is incomplete.",
		"precondition_failed": "Required operation inputs, snapshot, verification or candidate bindings do not match.",
		"encoding_failed":     "The Host result could not be encoded; completion must be checked before retrying.",
	}
	recovery := operationRecovery(operation)
	switch code {
	case "locked", "busy":
		recovery = "Let the active writer finish or cancel that request, then inspect current state before retrying. Do not delete a Host lock to bypass it. " + recovery
	case "stale":
		recovery = "Refresh the operation preview and its expected digest against current inputs; preserve completed work. " + recovery
	case "notfound":
		recovery = "Check the selected repository and the supplied existing record IDs or file inputs. " + recovery
		if operation == "project_status" {
			recovery = "Check the selected repository and supplied runId against existing durable records; use a confirmed existing ID before requesting status again."
		}
	case "notrunnable":
		recovery = "Resolve the returned execution preconditions before retrying. " + recovery
	case "selection_failed":
		recovery = "Check the server's selected repository and the explicit base/revision; load an existing fixed revision before retrying."
	case "model_invalid":
		recovery = "Inspect the returned model/coverage validation report, correct canonical inputs and recheck the selected model before planning or writing."
	case "config_invalid":
		recovery = "Check the selected project runtime and configured Host operations; correct missing configuration and rerun the operation's preview or doctor check."
	}
	return Diagnostic{Code: code, Message: messages[code], Recovery: recovery}
}

func operationRecovery(operation string) string {
	switch operation {
	case "project_run", "project_resume", "project_repair", "project_status", "project_verify":
		return "Inspect project_status with the existing runId and its blockers before resuming or repairing the same run."
	case "project_apply", "project_preflight":
		return "Inspect project_status with the existing runId; obtain current preflight inputs and matching successful verification/review before Apply."
	case "project_deliver":
		return "If the partial report contains a runId, inspect that existing run with project_status. Otherwise check the acknowledged exploration/scope and refresh delivery inputs before retrying."
	case "project_plan":
		return "Check the selected base, bounded goal, model/runtime and any exploration/scope bindings; refresh the plan preview before authorized execution."
	case "project_full_verify":
		return "Check the selected revision, model and configured checks; correct their validation findings and repeat verification for the intended fixed snapshot."
	}
	if strings.Contains(operation, "setup") || strings.Contains(operation, "doctor") {
		return "Inspect setup/doctor findings and selected configuration; correct required inputs, refresh the setup preview and its digest before writing."
	}
	if strings.Contains(operation, "brownfield") {
		return "Inspect the existing session and stage report; correct source/base and stage preconditions, then refresh that stage preview and expected digest."
	}
	if strings.Contains(operation, "explor") || strings.Contains(operation, "readiness") {
		return "Inspect the selected exploration/scope and structure findings; correct or acknowledge required inputs and refresh the preview/digest before writing."
	}
	return "Inspect the returned structured validation report, correct the selected model/configuration and required inputs, then refresh this operation's preview and expected digest before writing."
}

func validPublicDiagnostic(d *Diagnostic) bool {
	if d == nil || len(d.Code) == 0 || len(d.Code) > 64 || len(d.Message) == 0 || len(d.Message) > 512 || len(d.Recovery) == 0 || len(d.Recovery) > 1024 {
		return false
	}
	for _, r := range d.Code {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	for _, text := range []string{d.Message, d.Recovery} {
		for _, r := range text {
			if unicode.IsControl(r) {
				return false
			}
		}
	}
	return true
}
