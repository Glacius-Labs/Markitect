// Package cli is the markitect command line: install, update, init and tidy; check,
// vet, list and doc over the spec; status and mark over the record of what was
// verified (DEC-099); config over the harness configuration (DEC-103); the hook;
// version and licenses. The commands are flat, print
// diagnostics as Go's tools do and, with --json, one envelope. None of them starts
// an agent, steers the work or uses Git (DEC-096, DEC-097).
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/skills"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// Workspace is a project folder, as a walk of it finds its files.
type Workspace interface {
	project.Repository
	Root() string
	Write(file string, data []byte) error
	Exists(file string) bool
	List(dir string) ([]string, error)
	Remove(file string) error
}

// Folders opens project folders.
type Folders interface {
	// Project opens the project folder that holds dir: the nearest one, dir or
	// above, with Markitect's manifest.
	Project(dir string) (Workspace, error)
	// At opens dir itself, which init makes a project.
	At(dir string) (Workspace, error)
}

// Exit codes, as Go's tools have them.
const (
	OK       = 0
	Failed   = 1 // the command did its job and found something
	Misused  = 2 // the command could not run, such as on a usage error
	usageTop = skills.About + `
No command starts an agent or uses Git: the /markitect-build and /markitect-review
skills have your Claude Code session start the agents, and the session commits.

Usage:
  markitect install                               tell Claude Code what Markitect is, in every folder
  markitect update [--repo DIR] [--check] [--files] [--force]
                                                  install the newest release, then refresh the personal skill and the
                                                  project's skills, agents, hooks and lock
  markitect init   [--repo DIR] [--name TITLE]   make a folder a Markitect project and install the harness
  markitect tidy   [--repo DIR]                   bring the lock in line with the manifest
  markitect check  [--repo DIR] [--json] [--spec] compile the spec and the registry; --spec looks only at the spec
  markitect vet    [--repo DIR] [--json] [--strict]
                                                  analyse the spec's design: coupling, size, how far a change reaches;
                                                  it warns, and exits 1 only with --strict
  markitect list   [--repo DIR] [--json] [--deps] [--packages]  the Packages and their Definitions; --deps adds what each depends on, --packages prints the Packages alone
  markitect doc    [--repo DIR] [--json] TARGET   a Definition or Package: its text, its links both ways, its files
                                                  and its record
  markitect status [--repo DIR] [--json] [--all]  what is to verify: each Definition with a changed input or no record,
                                                  with what changed, where to look and its fingerprint; --all lists
                                                  every Definition, to verify everything again
  markitect mark   [--repo DIR] [--json] DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]... [-m "WHY"] [--cases N --covered M]
                                                  record a verdict at the state status printed: ok, fail, skip or
                                                  spec-problem; evidence is path:line or path; only the
                                                  markitect-reviewer agent runs it
  markitect config [--repo DIR] KEY | [--user] KEY VALUE | [--user] --unset KEY | --list [--json]
                                                  read or change the harness configuration: the agents' model and
                                                  effort per role, the strategy per skill and custom workflows; the
                                                  project's .markitect/harness.yaml beats your ~/.markitect/harness.yaml
  markitect version [--repo DIR]                  which markitect this is, and whether the personal skill and the project match it
  markitect licenses

Diagnostics read file:line: code: message. --json prints one document with
format_version and kind. Exit codes: 0 when all is well, 1 when the command found
something (an error, an item to verify, a refused verdict, a warning of vet
--strict), 2 when it could not run.
`
)

// Releases finds and installs newer binaries of markitect.
type Releases interface {
	// Latest returns the version of the newest release, such as "0.15.0".
	Latest(ctx context.Context) (string, error)
	// Install puts the binary of that release in the place of the executable exe.
	Install(ctx context.Context, version, exe string) error
	// Run starts the executable exe and returns its exit code.
	Run(ctx context.Context, exe string, args []string, stdout, stderr io.Writer) (int, error)
	// Tidy removes what an earlier Install left beside the executable exe.
	Tidy(exe string)
}

// CLI runs one command.
type CLI struct {
	Folders Folders
	// Record is the engine of status and mark.
	Record *run.Engine
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// PathWarning says what is wrong when the PATH does not lead to this
	// markitect, or "" when nothing is. Nil means no check.
	PathWarning func() string
	// Releases is where update looks for a newer binary. Nil means update never
	// replaces the binary.
	Releases Releases
	// Executable names the file of this binary, which update replaces. Nil
	// means os.Executable.
	Executable func() (string, error)
	// Version is this binary's version. Nil means buildinfo.Version.
	Version func() string
}

// Run runs the command line args and returns the exit code.
func (c CLI) Run(ctx context.Context, args []string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(c.Stdout, usageTop)
		return OK
	}
	cmd, rest := args[0], args[1:]
	var err error
	code := OK
	switch cmd {
	case "install":
		code, err = c.install(rest)
	case "update":
		code, err = c.update(ctx, rest)
	case "init":
		code, err = c.init(ctx, rest)
	case "tidy":
		code, err = c.tidy(ctx, rest)
	case "check":
		code, err = c.check(ctx, rest)
	case "vet":
		code, err = c.vet(ctx, rest)
	case "list":
		code, err = c.list(ctx, rest)
	case "doc":
		code, err = c.doc(ctx, rest)
	case "status":
		code, err = c.status(ctx, rest)
	case "mark":
		code, err = c.mark(ctx, rest)
	case "config":
		code, err = c.config(ctx, rest)
	case "hook":
		code, err = c.hook(ctx, rest)
	case "version":
		code, err = c.version(ctx, rest)
	case "licenses":
		fmt.Fprint(c.Stdout, buildinfo.Notices)
	default:
		fmt.Fprintf(c.Stderr, "markitect: unknown command %q\n\n%s", cmd, usageTop)
		return Misused
	}
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(c.Stderr, "markitect %s: %v\n", cmd, err)
		}
		return Misused
	}
	switch cmd {
	case "check", "tidy", "vet", "list", "doc", "status":
		c.staleNotice(rest)
	}
	return code
}

// options are the flags every command shares.
type options struct {
	repo string
	json bool
}

func (c CLI) flags(name string, o *options) *flag.FlagSet {
	fs := flag.NewFlagSet("markitect "+name, flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the project")
	switch name {
	case "check", "vet", "list", "doc", "status", "config":
		fs.BoolVar(&o.json, "json", false, "print one JSON document instead of text")
	}
	return fs
}

// parse parses a command's flags wherever they stand, before or after its
// arguments, as agents write them, such as markitect doc x --json. It returns
// the arguments.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// load opens the project folder and loads the project.
func (c CLI) load(o options) (Workspace, *project.Project, error) {
	ws, err := c.Folders.Project(o.repo)
	if err != nil {
		return nil, nil, err
	}
	p, err := project.Load(ws)
	if err != nil {
		return nil, nil, err
	}
	return ws, p, nil
}

// warnErrors tells the user that an analysis ran over a spec with errors.
func (c CLI) warnErrors(p *project.Project) {
	if n := len(p.SpecErrors()); n > 0 {
		fmt.Fprintf(c.Stderr, "markitect: the spec has %d errors; run markitect check. The result below may be incomplete.\n", n)
	}
}

// check compiles the spec and the registry and prints every error, the spec's
// before the realization's. It produces nothing. Exit 1 when there is an error.
func (c CLI) check(_ context.Context, args []string) (int, error) {
	var o options
	var specOnly bool
	fs := c.flags("check", &o)
	fs.BoolVar(&specOnly, "spec", false, "look only at the spec itself, not at whether the work matches it")
	if positional, err := parse(fs, args); err != nil {
		return Misused, err
	} else if len(positional) != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	_, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	specErrs, realization := p.SpecErrors(), p.RealizationErrors()
	nothingBuilt := !specOnly && p.NothingBuilt()
	if specOnly || nothingBuilt {
		realization = nil
	}
	total := len(specErrs) + len(realization)
	if o.json {
		doc := struct {
			envelope
			OK                bool         `json:"ok"`
			NothingBuilt      bool         `json:"nothing_built"`
			Definitions       int          `json:"definitions"`
			Packages          int          `json:"packages"`
			Files             int          `json:"files"`
			SpecErrors        int          `json:"spec_errors"`
			RealizationErrors int          `json:"realization_errors"`
			Diagnostics       []diagnostic `json:"diagnostics"`
		}{newEnvelope("check"), total == 0, nothingBuilt, len(p.Spec.OwnDefinitions()), len(p.Spec.Packages), len(p.Registry.Owner), len(specErrs), len(realization), append(diagnostics(severityError, specErrs), diagnostics(severityError, realization)...)}
		if err := c.printJSON(doc); err != nil {
			return Misused, err
		}
	} else {
		for _, d := range append(append(spec.Diagnostics{}, specErrs...), realization...) {
			fmt.Fprintln(c.Stdout, d.String())
		}
		switch {
		case nothingBuilt:
			fmt.Fprintf(c.Stdout, "ok: the spec compiles, with %d Definitions in %d Packages. Nothing is built yet, and /markitect-build builds it.\n", len(p.Spec.OwnDefinitions()), len(p.Spec.Packages))
		case total == 0:
			fmt.Fprintf(c.Stdout, "ok: %d Definitions in %d Packages, %d files accounted\n", len(p.Spec.OwnDefinitions()), len(p.Spec.Packages), len(p.Registry.Owner))
		default:
			fmt.Fprintf(c.Stdout, "%d %s: %d in the spec, for the owner to settle; %d in the realization, so spec and code do not match yet, which /markitect-build settles\n", total, plural(total, "error", "errors"), len(specErrs), len(realization))
		}
	}
	if total > 0 {
		return Failed, nil
	}
	return OK, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
