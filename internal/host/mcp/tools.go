// Package mcp implements a local MCP adapter over typed project Host operations.
// It owns no workflow, provider invocation, or authorization policy.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
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
func Register[T, R any](s *Server, name, description string, mutation bool, call func(context.Context, T) (R, error)) {
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
			code := "host_rejected"
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				code = "cancelled"
			}
			// Partial Host reports retain durable run/candidate handles after failure.
			return outcome(name, &data, code), nil
		}
		return outcome(name, &data, ""), nil
	}
	s.tools[name] = b
}
func outcome[R any](name string, data *R, code string) CallResult {
	out := Outcome[R]{Operation: name, Data: data}
	if code != "" {
		out.Diagnostic = &Diagnostic{Code: code, Message: "The shared Host operation did not complete successfully.", Recovery: "Inspect project_status using the existing runId; check repository, base, review and freshness inputs before resume or repair. Host details remain local."}
	}
	raw, err := json.Marshal(out)
	if err != nil {
		out = Outcome[R]{Operation: name, Diagnostic: &Diagnostic{Code: "encoding_failed", Message: "Host result could not be encoded.", Recovery: "Inspect durable Host status before retrying."}}
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
