package projectcli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/tooling/licenses"
)

// Main runs one product CLI invocation from the verb table and returns its exit
// code: 0 success, 1 a completed non-conforming or blocked outcome, 2 an
// invalid invocation or an operation that failed without a report.
func Main(args []string, out, errout io.Writer, version string) int {
	if len(args) == 0 {
		printVerbList(errout)
		return 2
	}
	switch args[0] {
	case "--help", "-h":
		printVerbList(out)
		return 0
	case "help":
		switch len(args) {
		case 1:
			printVerbList(out)
			return 0
		case 2:
			v, ok := lookupVerb(args[1])
			if !ok {
				fmt.Fprintf(errout, "markitect help: unknown verb %q\n", args[1])
				return 2
			}
			printVerbHelp(out, v)
			return 0
		}
		fmt.Fprintln(errout, "markitect help: accepts at most one verb")
		return 2
	}
	v, ok := lookupVerb(args[0])
	if !ok {
		fmt.Fprintf(errout, "markitect: unknown verb %q; run `markitect help`\n", args[0])
		return 2
	}
	switch v.name {
	case "version", "licenses":
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			printVerbHelp(out, v)
			return 0
		}
		if len(args) != 1 {
			fmt.Fprintf(errout, "markitect %s: accepts no arguments\n", v.name)
			return 2
		}
		if v.name == "version" {
			fmt.Fprintf(out, "Markitect %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
			return 0
		}
		if _, err := io.WriteString(out, licenses.Text); err != nil {
			return 2
		}
		return 0
	}
	inv, help, err := parseInvocation(v, args[1:])
	if help {
		printVerbHelp(out, v)
		return 0
	}
	if err != nil {
		fmt.Fprintf(errout, "markitect %s: %v\n", v.name, err)
		return 2
	}
	if v.name == "mcp" {
		if err := serveMCP(inv.env, out); err != nil {
			fmt.Fprintf(errout, "markitect mcp: %v\n", err)
			return 2
		}
		return 0
	}
	ctx := context.Background()
	if v.interrupt {
		bounded, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()
		ctx = bounded
	}
	result, err := v.invoke(ctx, inv.env, inv.raw)
	return finish(v, result, err, out, errout)
}

// finish prints the report whenever one exists, so a partial report with
// durable handles is never lost, and maps the outcome to an exit code.
func finish(v verb, result any, err error, out, errout io.Writer) int {
	var usage usageError
	if errors.As(err, &usage) {
		fmt.Fprintf(errout, "markitect %s: %v\n", v.name, err)
		return 2
	}
	hasReport := result != nil && !reflect.ValueOf(result).IsZero()
	if err == nil || hasReport {
		if writeErr := writeJSON(out, result); writeErr != nil {
			fmt.Fprintf(errout, "markitect %s: %v\n", v.name, writeErr)
			return 2
		}
	}
	if err == nil {
		return 0
	}
	fmt.Fprintf(errout, "markitect %s: %v\n", v.name, err)
	if hasReport {
		return 1
	}
	return 2
}

type invocation struct {
	env env
	raw json.RawMessage
}

// parseInvocation turns CLI arguments into the verb's JSON input, so the CLI
// decodes exactly what an MCP client would send.
func parseInvocation(v verb, args []string) (invocation, bool, error) {
	fs := flag.NewFlagSet(v.name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	texts := map[string]*singleValue{}
	bools := map[string]*bool{}
	lists := map[string]*stringList{}
	for _, a := range v.args {
		if a.operand {
			continue
		}
		switch a.kind {
		case kindBool:
			bools[a.name] = fs.Bool(a.name, false, a.help)
		case kindList:
			list := &stringList{}
			fs.Var(list, a.name, a.help)
			lists[a.name] = list
		default:
			text := &singleValue{}
			fs.Var(text, a.name, a.help)
			texts[a.name] = text
		}
	}
	help := false
	fs.BoolVar(&help, "help", false, "show help")
	fs.BoolVar(&help, "h", false, "show help")
	positionals := []string{}
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return invocation{}, true, nil
			}
			return invocation{}, false, err
		}
		if fs.NArg() == 0 {
			break
		}
		positionals = append(positionals, fs.Arg(0))
		rest = fs.Args()[1:]
	}
	if help {
		return invocation{}, true, nil
	}
	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	fields := map[string]any{}
	if len(v.subs) != 0 {
		action := v.defaultSub
		if len(positionals) != 0 {
			if !v.hasSub(positionals[0]) {
				return invocation{}, false, fmt.Errorf("unknown %s %q; use one of %s", v.subLabel(), positionals[0], strings.Join(v.subNames(), ", "))
			}
			action, positionals = positionals[0], positionals[1:]
		}
		if action == "" {
			return invocation{}, false, fmt.Errorf("requires a %s: %s", v.subLabel(), strings.Join(v.subNames(), ", "))
		}
		fields["action"] = action
	}
	for _, a := range v.args {
		if !a.operand {
			continue
		}
		if len(positionals) == 0 {
			if a.required {
				return invocation{}, false, fmt.Errorf("requires the %s operand", a.value)
			}
			continue
		}
		fields[camel(a.name)], positionals = positionals[0], positionals[1:]
	}
	if len(positionals) != 0 {
		return invocation{}, false, fmt.Errorf("unexpected argument %q", positionals[0])
	}
	e := env{ops: projectOperations()}
	root, err := absoluteRoot(textValue(texts, "repo"))
	if err != nil {
		return invocation{}, false, err
	}
	e.root, e.sourceRoot = root, root
	if source := textValue(texts, "source-repo"); source != "" {
		if e.sourceRoot, err = absoluteRoot(source); err != nil {
			return invocation{}, false, err
		}
	}
	if readOnly, ok := bools["read-only"]; ok {
		e.readOnly = *readOnly
	}
	recordRoot := e.root
	if v.name == "adopt" {
		recordRoot = e.sourceRoot
	}
	for _, a := range v.args {
		if a.operand || a.cliOnly {
			continue
		}
		field := camel(a.name)
		if a.required && !seen[a.name] {
			return invocation{}, false, fmt.Errorf("requires --%s", a.name)
		}
		if !seen[a.name] {
			continue
		}
		switch a.kind {
		case kindBool:
			fields[field] = *bools[a.name]
		case kindList:
			fields[field] = []string(*lists[a.name])
		case kindInt:
			text := texts[a.name].value
			if _, err := strconv.ParseInt(text, 10, 64); err != nil {
				return invocation{}, false, fmt.Errorf("--%s must be an integer", a.name)
			}
			fields[field] = json.Number(text)
		case kindRecord:
			data, err := readRecord(recordRoot, texts[a.name].value)
			if err != nil {
				return invocation{}, false, err
			}
			if !json.Valid(data) {
				return invocation{}, false, fmt.Errorf("--%s %s is not valid JSON", a.name, texts[a.name].value)
			}
			fields[field] = json.RawMessage(data)
		default:
			fields[field] = texts[a.name].value
		}
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return invocation{}, false, err
	}
	return invocation{env: e, raw: raw}, false, nil
}

func textValue(texts map[string]*singleValue, name string) string {
	if value, ok := texts[name]; ok {
		return value.value
	}
	return ""
}

// singleValue is a single-valued flag. A second occurrence is an error, so a
// repeated flag never silently replaces the first value.
type singleValue struct {
	value string
	set   bool
}

func (s *singleValue) String() string {
	if s == nil {
		return ""
	}
	return s.value
}

func (s *singleValue) Set(value string) error {
	if s.set {
		return errors.New("may be given only once")
	}
	s.value, s.set = value, true
	return nil
}

// absoluteRoot defaults the project root to the current directory.
func absoluteRoot(path string) (string, error) {
	if path == "" {
		path = "."
	}
	return filepath.Abs(path)
}

func (v verb) hasSub(name string) bool {
	for _, sub := range v.subs {
		if sub.name == name {
			return true
		}
	}
	return false
}

func (v verb) subLabel() string {
	if v.name == "adopt" {
		return "stage"
	}
	return "action"
}

func lookupVerb(name string) (verb, bool) {
	for _, v := range verbTable() {
		if v.name == name {
			return v, true
		}
	}
	return verb{}, false
}

func printVerbList(out io.Writer) {
	var b strings.Builder
	b.WriteString("Usage: markitect <verb> [arguments]\n")
	group := ""
	for _, v := range verbTable() {
		if v.group != group {
			group = v.group
			fmt.Fprintf(&b, "\n%s\n", group)
		}
		fmt.Fprintf(&b, "  %-10s %s\n", v.name, v.summary)
	}
	b.WriteString("\nRun `markitect help VERB` for a verb's arguments. --repo defaults to the current directory.\n")
	_, _ = io.WriteString(out, b.String())
}

func printVerbHelp(out io.Writer, v verb) {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage: markitect %s\n\n%s\n", v.synopsis, v.summary)
	if v.notes != "" {
		fmt.Fprintf(&b, "%s\n", v.notes)
	}
	tool := "no (CLI only)"
	if v.mcpVerb(false) {
		tool = v.name
	}
	fmt.Fprintf(&b, "\nEffect: %s. MCP tool: %s.\n", v.effect, tool)
	if len(v.subs) != 0 {
		title := "Actions"
		if v.name == "adopt" {
			title = "Stages"
		}
		fmt.Fprintf(&b, "\n%s:\n", title)
		for _, sub := range v.subs {
			fmt.Fprintf(&b, "  %-10s %s (%s)\n", sub.name, sub.summary, sub.effect)
		}
	}
	if len(v.args) != 0 {
		b.WriteString("\nArguments:\n")
		for _, a := range v.args {
			name := "--" + a.name
			if a.operand {
				name = a.value
			} else if a.value != "" {
				name += " " + a.value
			}
			fmt.Fprintf(&b, "  %-28s %s\n", name, a.help)
		}
	}
	_, _ = io.WriteString(out, b.String())
}

type stringList []string

func (s *stringList) String() string { return fmt.Sprint([]string(*s)) }
func (s *stringList) Set(value string) error {
	if value == "" {
		return errors.New("value must not be empty")
	}
	*s = append(*s, value)
	return nil
}
