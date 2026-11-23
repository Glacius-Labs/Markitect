// Package cli is the command line of markitect: it parses the arguments,
// finds the project they work in, reads its spec and prints what the language
// finds, as text or as JSON. It writes nothing.
package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// FormatVersion is the version of every JSON document the command prints. A
// program reads format_version and kind, not the wording of the text output.
const FormatVersion = 1

// Exit codes.
const (
	ExitOK     = 0 // all is well
	ExitFound  = 1 // the command found something
	ExitFailed = 2 // the command could not run
)

const usage = `usage: markitect COMMAND ARGUMENTS [-C PROJECT] [--json]

Look
  show DOCUMENT               a document with its type, sections and links
  table [TYPE]                every document of a type as a Markdown table, or
                              every document with its type and title
  topic [TOPIC] [--all]       the signals of the spec, or everything about a
                              document, a type or a folder
  questions                   the questions of the packages the spec has
  diff OLD [NEW]              what is new, gone or changed in the spec of NEW,
                              by default this project, against the project OLD

Check
  check                       check the spec

Tool
  version                     the tool's version and build
  licenses                    the license notices of the modules the tool is
                              built from

The project is the nearest folder, from the working directory upward, that
holds the file .markitect/spec, or the folder -C names; the file's one line
names the spec folder. A DOCUMENT, TYPE or TOPIC is a path below the spec
folder, with or without .md. Diagnostics read path:line: code: message. --json
prints one document with format_version and kind. Exit codes: 0 when all is
well, 1 when the command found something, 2 when it could not run.
`

// A command takes its positional arguments, between min and max of them.
type command struct {
	min, max int
	run      func(o options, stdout io.Writer) (int, error)
}

var commands = map[string]command{
	"check":     {0, 0, check},
	"show":      {1, 1, show},
	"table":     {0, 1, table},
	"topic":     {0, 1, topic},
	"questions": {0, 0, questions},
	"version":   {0, 0, version},
	"licenses":  {0, 0, licenses},
	"diff":      {1, 2, diff},
}

// Run runs the command line args, without the program name, and returns the
// exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return ExitFailed
	}
	// --help, -h and the command help print the usage, wherever they stand.
	if wantsHelp(args) {
		fmt.Fprint(stdout, usage)
		return ExitOK
	}
	// The flags may stand before the command, as in markitect -C PROJECT
	// check.
	name, rest, err := splitCommand(args)
	if err != nil {
		fmt.Fprintf(stderr, "markitect: %v; markitect --help lists the commands and flags\n", err)
		return ExitFailed
	}
	if name == "help" {
		fmt.Fprint(stdout, usage)
		return ExitOK
	}
	c, ok := commands[name]
	if !ok {
		fmt.Fprintf(stderr, "markitect: unknown command %q; markitect --help lists the commands\n", name)
		return ExitFailed
	}
	o, err := parse(rest)
	if err == nil {
		err = validate(name, c, o)
	}
	code := ExitFailed
	if err == nil {
		code, err = c.run(o, stdout)
	}
	if err != nil {
		fmt.Fprintf(stderr, "markitect %s: %v\n", name, err)
	}
	return code
}

// wantsHelp says whether the arguments ask for the usage: help, -h, -help or
// --help, not as the value of a flag and not after --.
func wantsHelp(args []string) bool {
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--":
			return false
		case a == "help" && i == 0 || a == "-h" || a == "-help" || a == "--help":
			return true
		case flagWithValue[a]:
			i++
		}
	}
	return false
}

// splitCommand finds the command among the arguments: the first that is no
// flag and no value of a flag, whatever flags stand before it. It returns the
// command and the other arguments in their order. It fails for a flag that no
// command has, for a flag without its value and when there is no command.
func splitCommand(args []string) (string, []string, error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return "", nil, errors.New("no command given")
		case flagWithValue[a]:
			if i++; i >= len(args) {
				return "", nil, fmt.Errorf("flag %s needs a value", a)
			}
		case flagWithoutValue[a]:
		case strings.HasPrefix(a, "-") && a != "-":
			return "", nil, fmt.Errorf("unknown flag %s", a)
		default:
			return a, append(append([]string{}, args[:i]...), args[i+1:]...), nil
		}
	}
	return "", nil, errors.New("no command given")
}

// flagWithoutValue are the flags that stand alone.
var flagWithoutValue = map[string]bool{"--json": true, "-json": true, "--all": true, "-all": true}

// flagWithValue are the flags that take the argument after them as a value.
var flagWithValue = map[string]bool{"-C": true}

// synopsis is the one line of usage of each command.
var synopsis = map[string]string{
	"check": "check", "show": "show DOCUMENT", "table": "table [TYPE]", "topic": "topic [TOPIC] [--all]", "questions": "questions",
	"version": "version", "licenses": "licenses", "diff": "diff OLD [NEW]",
}

// validate checks the parsed arguments against the command: the number of
// positional arguments, and that each flag given is one of the command's.
func validate(name string, c command, o options) error {
	switch {
	case len(o.positional) < c.min || len(o.positional) > c.max:
		return fmt.Errorf("wrong number of arguments\nusage: markitect %s [-C PROJECT] [--json]\nmarkitect --help lists every command", synopsis[name])
	case o.all && name != "topic":
		return errors.New("--all is a flag of topic")
	}
	return nil
}

// options are a command's flags and positional arguments. Flags may stand
// anywhere; "--" ends them.
type options struct {
	json bool
	all  bool
	// dir is the project folder -C names, "" without the flag.
	dir        string
	positional []string
}

func parse(args []string) (options, error) {
	var o options
	for i := 0; i < len(args); i++ {
		a := args[i]
		value := func(flag string) (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag %s needs a value", flag)
			}
			i++
			return args[i], nil
		}
		var err error
		switch {
		case a == "--":
			o.positional = append(o.positional, args[i+1:]...)
			return o, nil
		case a == "--json" || a == "-json":
			o.json = true
		case a == "--all" || a == "-all":
			o.all = true
		case a == "-C":
			if o.dir, err = value(a); err == nil && o.dir == "" {
				err = errors.New("flag -C needs a folder")
			}
		case strings.HasPrefix(a, "-") && a != "-":
			err = fmt.Errorf("unknown flag %s", a)
		default:
			o.positional = append(o.positional, a)
		}
		if err != nil {
			return o, err
		}
	}
	return o, nil
}

// document finds a document named on the command line.
func document(spec *lang.Spec, arg string) (*lang.Document, error) {
	if p, ok := documentPath(arg); ok {
		if d, ok := spec.Document(p); ok {
			return d, nil
		}
	}
	return nil, fmt.Errorf("unknown path: %s", arg)
}

func check(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	spec := w.Spec
	found := lang.Check(spec)
	if o.json {
		doc := checkDocument{header: newHeader("check"), Documents: len(spec.Documents), Diagnostics: []diagnostic{}}
		for _, d := range found {
			doc.Diagnostics = append(doc.Diagnostics, diagnostic{File: d.Path, Line: d.Line, Code: d.Code, Message: d.Message})
		}
		if err := printJSON(stdout, doc); err != nil {
			return ExitFailed, err
		}
	} else {
		var b strings.Builder
		for _, d := range found {
			b.WriteString(d.String() + "\n")
		}
		fmt.Fprintf(&b, "%s in %s\n", plural(len(found), "error"), plural(len(spec.Documents), "document"))
		if _, err := io.WriteString(stdout, b.String()); err != nil {
			return ExitFailed, err
		}
	}
	if len(found) > 0 {
		return ExitFound, nil
	}
	return ExitOK, nil
}

// header starts every JSON document.
type header struct {
	FormatVersion int    `json:"format_version"`
	Kind          string `json:"kind"`
}

func newHeader(kind string) header { return header{FormatVersion: FormatVersion, Kind: kind} }

// checkDocument is the JSON document of check: the number of documents of
// the spec, and the diagnostics.
type checkDocument struct {
	header
	Documents   int          `json:"documents"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

// diagnostic is one diagnostic in a JSON document; file is the path below the
// spec folder.
type diagnostic struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// output prints a JSON document with --json and the text otherwise.
func output(o options, stdout io.Writer, doc any, text string) (int, error) {
	if o.json {
		if err := printJSON(stdout, doc); err != nil {
			return ExitFailed, err
		}
		return ExitOK, nil
	}
	if _, err := io.WriteString(stdout, text); err != nil {
		return ExitFailed, err
	}
	return ExitOK, nil
}

// printJSON prints a document, indented, without escaping <, > and &.
func printJSON(w io.Writer, v any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	_, err := w.Write(b.Bytes())
	return err
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// documentPath turns a document's name on the command line into its path
// below the spec folder: backslashes become slashes and ".md" is added when it
// is missing. ok is false for a path that leaves the spec folder, is absolute,
// has a scheme or names no file.
func documentPath(arg string) (string, bool) {
	p := strings.TrimSpace(strings.ReplaceAll(arg, "\\", "/"))
	if !strings.HasSuffix(strings.ToLower(p), ".md") {
		p += ".md"
	}
	if strings.HasPrefix(p, "/") {
		return "", false
	}
	p = path.Clean(p)
	if p == ".." || strings.HasPrefix(p, "../") || hasScheme(p) || strings.EqualFold(path.Base(p), ".md") {
		return "", false
	}
	return p, true
}

func hasScheme(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z':
		case i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '.' || c == '-'):
		case i > 0 && c == ':':
			return true
		default:
			return false
		}
	}
	return false
}
