package projectcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/mcp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// The verb table is the single source of the command surface: CLI parsing,
// help and MCP registration all derive from it (docs/design/verb-table.md).

type effect string

const (
	effectRead         effect = "read"
	effectWrite        effect = "write"
	effectExecute      effect = "execute"
	effectExecuteWrite effect = "execute and write"
	effectServer       effect = "server"
	effectInfo         effect = "information"
)

type argKind int

const (
	kindString argKind = iota
	kindBool
	kindList
	kindInt
	kindRecord
)

// arg is one verb argument. The CLI flag is its kebab-case name; the MCP field
// and the JSON input field are its camelCase form.
type arg struct {
	name         string
	kind         argKind
	help         string
	value        string // CLI placeholder, such as ID or FILE
	operand      bool   // positional in the CLI
	required     bool   // required by the CLI synopsis; the handler validates combinations
	cliOnly      bool   // not an MCP field, such as --repo
	readOnlyOmit bool   // dropped from the read-only MCP tool
}

// subVerb is a sub-verb such as an adopt stage. MCP takes it as the field
// "action"; readOnly marks sub-verbs that a read-only server keeps.
type subVerb struct {
	name     string
	summary  string
	effect   effect
	readOnly bool
}

type verb struct {
	name       string
	group      string
	summary    string
	synopsis   string
	effect     effect
	args       []arg
	subs       []subVerb
	defaultSub string
	cliOnly    bool
	inspect    string // the read-only verb to inspect before an execution
	recovery   string // MCP diagnostic recovery guidance
	interrupt  bool   // the CLI cancels on interrupt
	notes      string // extra help text

	input    reflect.Type
	invoke   func(context.Context, env, json.RawMessage) (any, error)
	register func(*mcp.Server, env, bool)
}

// env carries what a composition fixes for every call: the project root and,
// for adoption, the source root. MCP never takes either from tool arguments.
type env struct {
	root       string
	sourceRoot string
	readOnly   bool
	ops        projectapp.Operations
}

// usageError is an invalid invocation. The CLI exits with code 2 and prints no
// report.
type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }

func usagef(format string, args ...any) error { return usageError{fmt.Errorf(format, args...)} }

// outcomeError is a completed operation with a non-conforming or blocked
// result. Its report is still returned; the CLI exits with code 1.
type outcomeError struct{ message string }

func (e outcomeError) Error() string { return e.message }

func outcomef(format string, args ...any) error { return outcomeError{fmt.Sprintf(format, args...)} }

// define binds a typed handler to its table entry. CLI and MCP decode the same
// JSON input with the same closed schema, then share the effect checks.
func define[In, Out any](v verb, handler func(context.Context, env, In) (Out, error)) verb {
	v.input = reflect.TypeFor[In]()
	call := func(ctx context.Context, e env, in In) (Out, error) {
		if err := checkEffect(v, e, reflect.ValueOf(in)); err != nil {
			var zero Out
			return zero, err
		}
		if err := resolveRevisions(e.root, reflect.ValueOf(&in).Elem()); err != nil {
			var zero Out
			return zero, err
		}
		return handler(ctx, e, in)
	}
	v.invoke = func(ctx context.Context, e env, raw json.RawMessage) (any, error) {
		in, err := mcp.DecodeArguments[In](raw)
		if err != nil {
			return nil, usageError{err}
		}
		return call(ctx, e, in)
	}
	v.register = func(s *mcp.Server, e env, readOnly bool) {
		options := []mcp.Option{mcp.WithRecovery(v.recoveryText()), mcp.WithPublicError(publicError)}
		if readOnly {
			if omit := v.readOnlyOmits(); len(omit) != 0 {
				options = append(options, mcp.WithOmit(omit...))
			}
			if len(v.subs) != 0 {
				options = append(options, mcp.WithEnum("action", v.readOnlySubs()...))
			}
		} else if len(v.subs) != 0 {
			options = append(options, mcp.WithEnum("action", v.subNames()...))
		}
		mutation := !readOnly && v.effect != effectRead
		mcp.Register(s, v.name, v.description(), mutation, func(ctx context.Context, in In) (Out, error) { return call(ctx, e, in) }, options...)
	}
	return v
}

var fullCommitID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// resolveRevisions turns each revision argument into the full commit ID it
// names, so a caller may write HEAD, a branch or a short ID while the Host
// still binds one fixed, full commit and records it.
func resolveRevisions(root string, in reflect.Value) error {
	for _, field := range []string{"Revision", "Since"} {
		f := in.FieldByName(field)
		if !f.IsValid() || f.Kind() != reflect.String || f.String() == "" {
			continue
		}
		name := "--revision"
		if field == "Since" {
			name = "--since"
		}
		// Git peels tags to their commit, whatever the case of the ID.
		out, err := source.GitOutput(root, "rev-parse", "--verify", "--quiet", "--end-of-options", f.String()+"^{commit}")
		full := strings.ToLower(strings.TrimSpace(string(out)))
		if err == nil && fullCommitID.MatchString(full) {
			f.SetString(full)
			continue
		}
		return usagef("%s %q does not name a commit in this repository", name, f.String())
	}
	return nil
}

// checkEffect applies the effect rules shared by every adapter: --expect needs
// --write or --execute, a write needs --expect where the verb takes one, an
// execution needs --execute, and a read-only server writes and starts nothing.
func checkEffect(v verb, e env, in reflect.Value) error {
	write, hasWrite := boolField(in, "Write")
	execute, hasExecute := boolField(in, "Execute")
	expect, hasExpect := stringField(in, "Expect")
	if e.readOnly && (write || execute) {
		return usagef("this MCP server is read-only")
	}
	if hasExpect && expect != "" && !write && !execute {
		return usagef("--expect is valid only together with --write or --execute")
	}
	if hasWrite && hasExpect && write && expect == "" {
		return usagef("--write requires --expect with the digest of the reviewed preview")
	}
	if hasExecute && !execute && v.effect != effectRead && v.requiresExecute(in) {
		return usagef("%s starts agents or configured checks and requires --execute; inspect first with `markitect %s`", v.name, v.inspect)
	}
	return nil
}

// requiresExecute is true for execute verbs and for adopt run.
func (v verb) requiresExecute(in reflect.Value) bool {
	if v.effect == effectExecute || v.effect == effectExecuteWrite {
		return true
	}
	if action, ok := stringField(in, "Action"); ok {
		for _, sub := range v.subs {
			if sub.name == action && (sub.effect == effectExecute || sub.effect == effectExecuteWrite) {
				write, _ := boolField(in, "Write")
				return write
			}
		}
	}
	return false
}

func boolField(v reflect.Value, name string) (bool, bool) {
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.Bool {
		return false, false
	}
	return f.Bool(), true
}

func stringField(v reflect.Value, name string) (string, bool) {
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.String {
		return "", false
	}
	return f.String(), true
}

func (v verb) description() string {
	if v.notes == "" {
		return v.summary
	}
	return v.summary + " " + v.notes
}

func (v verb) recoveryText() string {
	if v.recovery != "" {
		return v.recovery
	}
	return "Inspect the returned structured report, correct the selected model, configuration or inputs, then refresh this operation's preview and expected digest before writing."
}

func (v verb) readOnlyOmits() []string {
	omit := []string{}
	for _, a := range v.args {
		if a.cliOnly {
			continue
		}
		switch {
		case a.readOnlyOmit, a.name == "write", a.name == "expect", a.name == "execute":
			omit = append(omit, camel(a.name))
		}
	}
	return omit
}

func (v verb) subNames() []string {
	names := make([]string, 0, len(v.subs))
	for _, sub := range v.subs {
		names = append(names, sub.name)
	}
	return names
}

func (v verb) readOnlySubs() []string {
	names := []string{}
	for _, sub := range v.subs {
		if sub.readOnly {
			names = append(names, sub.name)
		}
	}
	return names
}

// mcpVerb reports whether a verb is an MCP tool, and in read-only mode whether
// the read-only server keeps it.
func (v verb) mcpVerb(readOnly bool) bool {
	if v.cliOnly || v.register == nil {
		return false
	}
	if !readOnly {
		return true
	}
	return v.effect == effectRead || v.effect == effectWrite
}

// camel converts a kebab-case flag name to its camelCase field name.
func camel(name string) string {
	parts := strings.Split(name, "-")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// publicError exposes product-authored usage and outcome messages to MCP
// clients. Other errors keep the adapter's safe classification.
func publicError(err error) *mcp.Diagnostic {
	var usage usageError
	if errors.As(err, &usage) {
		return &mcp.Diagnostic{Code: "invalid_arguments", Message: truncate(usage.Error(), 512), Recovery: "Correct the arguments as the message states; tools/list shows each tool's fields."}
	}
	var name projectapp.ManagerNameError
	if errors.As(err, &name) {
		return &mcp.Diagnostic{Code: "invalid_arguments", Message: truncate(name.Error(), 512), Recovery: "Use the Manager's full ID or a short name that is unique; `model` lists both."}
	}
	var outcome outcomeError
	if errors.As(err, &outcome) {
		return &mcp.Diagnostic{Code: "nonconforming", Message: truncate(outcome.Error(), 512), Recovery: "The operation completed; inspect the returned report and correct its findings before continuing."}
	}
	return nil
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}
