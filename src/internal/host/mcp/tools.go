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
	recovery string
	call     func(context.Context, json.RawMessage) (CallResult, error)
}
type Server struct {
	root         string
	instructions string
	tools        map[string]binding
	mutation     sync.Mutex
}

const defaultInstructions = "Tools use the explicitly selected repository and existing caller authority. Preserve durable run IDs; protocol request IDs are not product run IDs. Only listed shared operations are available."

// New binds a server to one explicit root. Tool arguments cannot redirect Host
// authority. The server registers no tools itself: composition registers each
// operation, and Register and Serve must not execute concurrently.
func New(root string) (*Server, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("MCP requires an explicit project root")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Server{root: filepath.Clean(root), instructions: defaultInstructions, tools: map[string]binding{}}, nil
}

// Root is the server's fixed project root.
func (s *Server) Root() string { return s.root }

// SetInstructions replaces the initialize instructions, for example to state a
// read-only tool set.
func (s *Server) SetInstructions(text string) { s.instructions = text }

// PublicErrorMapper is a composition-owned translation of known application
// errors into intentionally public diagnostics. Never return err.Error(), raw
// provider/check output, credentials or local machine paths from this callback.
// Return nil for unknown errors to retain the adapter's safe fallback.
type PublicErrorMapper func(error) *Diagnostic

// Option adjusts one registration.
type Option func(*registration)

type registration struct {
	publicError PublicErrorMapper
	omit        []string
	enums       map[string][]string
	recovery    string
}

// WithPublicError adds application error classifications.
func WithPublicError(mapper PublicErrorMapper) Option {
	return func(r *registration) { r.publicError = mapper }
}

// WithOmit removes top-level input fields from the closed schema, so calls that
// send them are rejected.
func WithOmit(fields ...string) Option {
	return func(r *registration) { r.omit = append(r.omit, fields...) }
}

// WithEnum restricts a top-level string field to the given values.
func WithEnum(field string, values ...string) Option {
	return func(r *registration) {
		if r.enums == nil {
			r.enums = map[string][]string{}
		}
		r.enums[field] = append([]string(nil), values...)
	}
}

// WithRecovery sets the operation-specific recovery guidance of diagnostics.
func WithRecovery(text string) Option {
	return func(r *registration) { r.recovery = text }
}

// Register adds a typed shared operation at composition time. It never
// executes a CLI subprocess.
func Register[T, R any](s *Server, name, description string, mutation bool, call func(context.Context, T) (R, error), options ...Option) {
	if _, exists := s.tools[name]; exists {
		panic("duplicate MCP tool: " + name)
	}
	var reg registration
	for _, option := range options {
		option(&reg)
	}
	if reg.recovery == "" {
		reg.recovery = genericRecovery
	}
	input := restrict(schema(reflect.TypeFor[T]()), reg.omit, reg.enums)
	b := binding{tool: Tool{Name: name, Description: description, InputSchema: input, OutputSchema: outputSchema(reflect.TypeFor[Outcome[R]]()), Annotations: map[string]bool{"readOnlyHint": !mutation, "destructiveHint": mutation, "openWorldHint": false}}, mutation: mutation, recovery: reg.recovery}
	b.call = func(ctx context.Context, raw json.RawMessage) (CallResult, error) {
		var r T
		if err := decodeTyped(raw, input, &r); err != nil {
			return CallResult{}, err
		}
		if err := ctx.Err(); err != nil {
			return outcome[R](name, nil, "cancelled", reg.recovery), nil
		}
		data, err := call(ctx, r)
		if err != nil {
			// Partial Host reports retain durable run/candidate handles after failure.
			diagnostic := classifyError(err, reg.recovery)
			// Cancellation and shared sentinel classifications cannot be hidden by
			// a custom mapper; it supplies additional application classifications.
			if diagnostic.Code == "host_rejected" && reg.publicError != nil {
				if mapped := reg.publicError(err); validPublicDiagnostic(mapped) {
					diagnostic = *mapped
				}
			}
			// A failure without a report carries no data, like the CLI's exit 2.
			if reflect.ValueOf(&data).Elem().IsZero() {
				return diagnosticOutcome[R](name, nil, &diagnostic), nil
			}
			return diagnosticOutcome(name, &data, &diagnostic), nil
		}
		return outcome(name, &data, "", reg.recovery), nil
	}
	s.tools[name] = b
}

// DecodeArguments decodes tool arguments with the same closed-schema rules as
// a registered tool, so another adapter can accept exactly the same inputs.
func DecodeArguments[T any](raw json.RawMessage) (T, error) {
	var out T
	err := decodeTyped(raw, schema(reflect.TypeFor[T]()), &out)
	return out, err
}

func restrict(input map[string]any, omit []string, enums map[string][]string) map[string]any {
	if len(omit) == 0 && len(enums) == 0 {
		return input
	}
	properties, _ := input["properties"].(map[string]any)
	for _, field := range omit {
		if _, ok := properties[field]; !ok {
			panic("MCP omit names an unknown field: " + field)
		}
		delete(properties, field)
		required := []string{}
		for _, name := range toStrings(input["required"]) {
			if name != field {
				required = append(required, name)
			}
		}
		input["required"] = required
	}
	for field, values := range enums {
		if _, ok := properties[field]; !ok {
			panic("MCP enum names an unknown field: " + field)
		}
		properties[field] = map[string]any{"type": "string", "enum": values}
	}
	return input
}
func outcome[R any](name string, data *R, code, recovery string) CallResult {
	var diagnostic *Diagnostic
	if code != "" {
		d := diagnosticFor(code, recovery)
		diagnostic = &d
	}
	return diagnosticOutcome(name, data, diagnostic)
}
func diagnosticOutcome[R any](name string, data *R, diagnostic *Diagnostic) CallResult {
	out := Outcome[R]{Operation: name, Data: data, Diagnostic: diagnostic}
	raw, err := json.Marshal(out)
	if err != nil {
		d := diagnosticFor("encoding_failed", genericRecovery)
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

// ValidateResult checks a call's structured result against the tool's
// published output schema.
func (s *Server) ValidateResult(name string, result CallResult) error {
	b, ok := s.tools[name]
	if !ok {
		return errors.New("unknown tool")
	}
	return validate(result.StructuredContent, b.tool.OutputSchema.(map[string]any), "")
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
			return outcome[struct{}](name, nil, "busy", b.recovery), nil
		}
		defer s.mutation.Unlock()
	}
	return b.call(ctx, args)
}

// Classifications inspect known sentinels and a small set of product-owned
// boundary messages. None forwards the cause's text. They guide repair; Host
// validation and guards remain the authority for any later operation.
func classifyError(err error, recovery string) Diagnostic {
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
	return diagnosticFor(code, recovery)
}

func diagnosticFor(code, recovery string) Diagnostic {
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
	switch code {
	case "locked", "busy":
		recovery = "Let the active writer finish or cancel that request, then inspect current state before retrying. Do not delete a Host lock to bypass it. " + recovery
	case "stale":
		recovery = "Refresh the operation preview and its expected digest against current inputs; preserve completed work. " + recovery
	case "notfound":
		recovery = "Check the selected repository and the supplied existing record IDs or file inputs. " + recovery
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

// genericRecovery applies when a registration names no operation-specific
// recovery.
const genericRecovery = "Inspect the returned structured validation report, correct the selected model/configuration and required inputs, then refresh this operation's preview and expected digest before writing."

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
