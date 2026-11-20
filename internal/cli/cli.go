// Package cli is the command line of the rebuilt tool, markitect-next: it
// parses the arguments, reads the spec folder they name and prints what the
// language finds, as text or as JSON.
package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
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

const usage = `usage: markitect-next COMMAND ARGUMENTS

  markitect-next check SPECFOLDER [--json]   check the spec in SPECFOLDER

Diagnostics read path:line: code: message, with the path below the spec
folder. --json prints one document with format_version and kind. Exit codes:
0 when all is well, 1 when the command found something, 2 when it could not run.
`

// Run runs the command line args, without the program name, and returns the
// exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return ExitFailed
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, usage)
		return ExitOK
	case "check":
		code, err := check(args[1:], stdout)
		if err != nil {
			fmt.Fprintf(stderr, "markitect-next check: %v\n", err)
		}
		return code
	}
	fmt.Fprintf(stderr, "markitect-next: unknown command %q\n\n%s", args[0], usage)
	return ExitFailed
}

// options are a command's flags and positional arguments. Flags may stand
// anywhere; "--" ends them.
type options struct {
	json       bool
	positional []string
}

func parse(args []string) (options, error) {
	var o options
	for i, a := range args {
		switch {
		case a == "--":
			o.positional = append(o.positional, args[i+1:]...)
			return o, nil
		case a == "--json" || a == "-json":
			o.json = true
		case strings.HasPrefix(a, "-") && a != "-":
			return o, fmt.Errorf("unknown flag %s", a)
		default:
			o.positional = append(o.positional, a)
		}
	}
	return o, nil
}

func check(args []string, stdout io.Writer) (int, error) {
	o, err := parse(args)
	if err != nil {
		return ExitFailed, err
	}
	if len(o.positional) != 1 {
		return ExitFailed, errors.New("takes one spec folder: markitect-next check SPECFOLDER [--json]")
	}
	folder := o.positional[0]
	if info, err := os.Stat(folder); err != nil || !info.IsDir() {
		return ExitFailed, fmt.Errorf("not a folder: %s", folder)
	}
	spec, err := lang.Load(os.DirFS(folder))
	if err != nil {
		return ExitFailed, err
	}
	found := lang.Check(spec)
	if o.json {
		doc := checkDocument{FormatVersion: FormatVersion, Kind: "check", Documents: len(spec.Documents), Diagnostics: []diagnostic{}}
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

// checkDocument is the JSON document of check.
type checkDocument struct {
	FormatVersion int          `json:"format_version"`
	Kind          string       `json:"kind"`
	Documents     int          `json:"documents"`
	Diagnostics   []diagnostic `json:"diagnostics"`
}

// diagnostic is one diagnostic in a JSON document; file is the path below the
// spec folder.
type diagnostic struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Code    string `json:"code"`
	Message string `json:"message"`
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
