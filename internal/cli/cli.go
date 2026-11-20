// Package cli is the command line of the rebuilt tool, markitect-next: it
// parses the arguments, finds the project they work in, sets one up with
// init, reads its spec, prints what the language finds, as text or as JSON,
// and applies the changes of the commands that write.
package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/edit"
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

const usage = `usage: markitect-next COMMAND ARGUMENTS [-C PROJECT] [--json]

  init [--spec PATH]          set up a project: .markitect/ and the spec folder,
                              by default docs/spec
  check                       check the spec
  show DOCUMENT               a document with its type, sections and links
  table TYPE                  every document of a type as a Markdown table
  new TYPE PATH [TITLE]       create a document of a type
  move OLD NEW                move a document and rewrite the links to it
  apply TEMPLATE [TARGET]     copy a template of the library into the spec
  topic [TOPIC] [--all]       the signals of the spec, or everything about a
                              document, a type or a folder
  questions                   the questions of the packages the spec has
  version                     the tool's version, and whether the project pins it

The project is the nearest folder, from the working directory upward, that
holds .markitect/version, or the folder -C names. A DOCUMENT, TYPE or PATH is a path
below the spec folder, with or without .md. Diagnostics read
path:line: code: message. --json prints one document with format_version and
kind. Exit codes: 0 when all is well, 1 when the command found something, 2
when it could not run.
`

// A command takes its positional arguments, between min and max of them.
type command struct {
	min, max int
	run      func(o options, stdout io.Writer) (int, error)
}

var commands = map[string]command{
	"init":      {0, 0, initProject},
	"check":     {0, 0, check},
	"show":      {1, 1, show},
	"table":     {1, 1, table},
	"new":       {2, 3, newDocument},
	"move":      {2, 2, move},
	"apply":     {1, 2, apply},
	"topic":     {0, 1, topic},
	"questions": {0, 0, questions},
	"version":   {0, 0, version},
}

// Run runs the command line args, without the program name, and returns the
// exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return ExitFailed
	}
	name := args[0]
	switch name {
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, usage)
		return ExitOK
	}
	c, ok := commands[name]
	if !ok {
		fmt.Fprintf(stderr, "markitect-next: unknown command %q\n\n%s", name, usage)
		return ExitFailed
	}
	o, err := parse(args[1:])
	switch {
	case err != nil:
	case len(o.positional) < c.min || len(o.positional) > c.max:
		err = fmt.Errorf("wrong number of arguments\n\n%s", usage)
	case o.specSet && name != "init":
		err = errors.New("--spec is a flag of init")
	}
	code := ExitFailed
	if err == nil {
		code, err = c.run(o, stdout)
	}
	if err != nil {
		var r *edit.Refusal
		if errors.As(err, &r) {
			for _, p := range r.Problems {
				fmt.Fprintf(stderr, "markitect-next %s: %s\n", name, p)
			}
		} else {
			fmt.Fprintf(stderr, "markitect-next %s: %v\n", name, err)
		}
	}
	return code
}

// options are a command's flags and positional arguments. Flags may stand
// anywhere; "--" ends them.
type options struct {
	json bool
	all  bool
	// dir is the project folder -C names, "" without the flag.
	dir string
	// spec is the spec folder --spec names, specSet whether it was given.
	spec       string
	specSet    bool
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
		case a == "--spec" || a == "-spec":
			o.spec, err = value(a)
			o.specSet = true
		case strings.HasPrefix(a, "--spec=") || strings.HasPrefix(a, "-spec="):
			_, o.spec, _ = strings.Cut(a, "=")
			o.specSet = true
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
	if p, ok := edit.DocumentPath(arg); ok {
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

// checkDocument is the JSON document of check.
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
