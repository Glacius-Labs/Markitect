// Package cli is the markitect command line: install, update, init, check, lock,
// refs, graph, impact and lint; checklist, answer, gate and status, the work that
// whoever drives the agents calls; the hook; version and licenses. None of them
// starts an agent or uses Git (DEC-096, DEC-097).
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/skills"
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

// Exit codes.
const (
	OK       = 0
	Failed   = 1 // the command ran and found errors
	Misused  = 2 // the command could not run
	usageTop = skills.About + `
No command starts an agent or uses Git: the /markitect-apply and /markitect-verify
skills have your Claude Code session start the agents, and the session commits.

Usage:
  markitect install                               tell Claude Code what Markitect is, in every folder
  markitect update [--repo DIR] [--check] [--files] [--force]
                                                  install the newest release, then refresh the personal skill and the project's skills, agents and lock
  markitect init   [--repo DIR] [--name TITLE]   make a folder a Markitect project and install the harness
  markitect check  [--repo DIR] [--json] [--spec] compile the project; --spec looks only at the spec
  markitect lock   [--repo DIR]                   write the lock from the manifest
  markitect refs   [--repo DIR] [--json] TARGET   who links to a Definition or Package
  markitect graph  [--repo DIR] [--json]          the Package dependency graph
  markitect impact [--repo DIR] [--json] [--base CERTIFICATE] [--full] [--findings FILE]
                                                  what a change since the newest certificate reaches, with distance and reason
  markitect lint   [--repo DIR] [--json]          measure the spec like code: warnings only
  markitect status [--repo DIR]                   how the work in the folder stands, and what differs from the newest certificate
  markitect version [--repo DIR]                  which markitect this is, and whether the personal skill and the project match it
  markitect licenses

The work, for whoever drives the agents:
  markitect checklist [--base CERTIFICATE] [--full] [--findings FILE] [--review] [--start]
                                                  the list of the change from the newest certificate to the folder;
                                                  --start begins it and says which agent to start
  markitect answer [FILE]                         store a reviewer's answers for the files as they are now
  markitect gate   [--verify] [--findings] [--spec-problem WHY]
                                                  check the work and say what is next; when it passes, write its certificate;
                                                  --spec-problem: the builder reports a problem of the spec, which a reviewer confirms
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
	// Work is the engine of checklist, answer, gate and status. Nil means this
	// build does no work.
	Work   *run.Engine
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
	case "check":
		code, err = c.check(ctx, rest)
	case "lock":
		code, err = c.lock(ctx, rest)
	case "refs":
		code, err = c.refs(ctx, rest)
	case "graph":
		code, err = c.graph(ctx, rest)
	case "impact":
		code, err = c.impact(ctx, rest)
	case "lint":
		code, err = c.lint(ctx, rest)
	case "checklist":
		code, err = c.checklist(ctx, rest)
	case "answer":
		code, err = c.answer(ctx, rest)
	case "gate":
		code, err = c.gate(ctx, rest)
	case "status":
		code, err = c.status(ctx, rest)
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
	case "check", "lock", "refs", "graph", "impact", "lint", "status":
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
	case "init", "lock", "update", "version":
	default:
		fs.BoolVar(&o.json, "json", false, "print JSON instead of Markdown")
	}
	return fs
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

func (c CLI) printJSON(v any) error {
	enc := json.NewEncoder(c.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (c CLI) check(ctx context.Context, args []string) (int, error) {
	var o options
	var specOnly bool
	fs := c.flags("check", &o)
	fs.BoolVar(&specOnly, "spec", false, "look only at the spec itself, not at whether the work matches it")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	_, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	spec, realization := p.SpecErrors(), p.RealizationErrors()
	nothingBuilt := !specOnly && p.NothingBuilt()
	if specOnly || nothingBuilt {
		realization = nil
	}
	total := len(spec) + len(realization)
	if o.json {
		list := append(append(model.Diagnostics{}, spec...), realization...)
		if err := c.printJSON(map[string]any{"ok": total == 0, "errors": list, "specErrors": len(spec), "realizationErrors": len(realization), "nothingBuilt": nothingBuilt}); err != nil {
			return Misused, err
		}
	} else {
		if len(spec) > 0 {
			fmt.Fprintf(c.Stdout, "In the spec, for the owner to settle:\n")
			for _, d := range spec {
				fmt.Fprintln(c.Stdout, d.String())
			}
		}
		if len(realization) > 0 {
			fmt.Fprintf(c.Stdout, "In the realization, so spec and code do not match yet; /markitect-apply settles these:\n")
			for _, d := range realization {
				fmt.Fprintln(c.Stdout, d.String())
			}
		}
		switch {
		case nothingBuilt:
			fmt.Fprintf(c.Stdout, "ok: the spec compiles, with %d Definitions in %d Packages. Nothing is built yet, and /markitect-apply builds it.\n", len(p.Model.ModelDefinitions()), len(p.Model.Packages))
		case total == 0:
			fmt.Fprintf(c.Stdout, "ok: %d Definitions in %d Packages, %d files accounted\n", len(p.Model.ModelDefinitions()), len(p.Model.Packages), len(p.Registry.Owner))
		default:
			fmt.Fprintf(c.Stdout, "%d errors\n", total)
		}
	}
	if total > 0 {
		return Failed, nil
	}
	return OK, nil
}

func (c CLI) refs(ctx context.Context, args []string) (int, error) {
	var o options
	fs := c.flags("refs", &o)
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 1 {
		return Misused, fmt.Errorf("names one Definition or Package, such as members/member")
	}
	_, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	c.warnErrors(p)
	target := fs.Arg(0)
	refs, err := analysis.Refs(p, target)
	if err != nil {
		return Misused, err
	}
	if o.json {
		if refs == nil {
			refs = []analysis.Ref{}
		}
		return OK, c.printJSON(map[string]any{"target": target, "refs": refs})
	}
	fmt.Fprint(c.Stdout, renderRefs(strings.Trim(target, "[]"), refs))
	return OK, nil
}

func (c CLI) graph(ctx context.Context, args []string) (int, error) {
	var o options
	fs := c.flags("graph", &o)
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	_, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	c.warnErrors(p)
	g := analysis.BuildGraph(p)
	if o.json {
		return OK, c.printJSON(g)
	}
	fmt.Fprint(c.Stdout, renderGraph(g))
	return OK, nil
}

func (c CLI) lint(ctx context.Context, args []string) (int, error) {
	var o options
	fs := c.flags("lint", &o)
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	_, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	c.warnErrors(p)
	warnings := analysis.Lint(p)
	g := analysis.BuildGraph(p)
	if o.json {
		if warnings == nil {
			warnings = model.Diagnostics{}
		}
		return OK, c.printJSON(map[string]any{"warnings": warnings, "packages": g.Packages})
	}
	fmt.Fprint(c.Stdout, renderLint(warnings, g))
	return OK, nil
}

func (c CLI) impact(_ context.Context, args []string) (int, error) {
	var o options
	var base, findings string
	var full bool
	fs := c.flags("impact", &o)
	fs.StringVar(&base, "base", "", "the certificate gate wrote, its .json or .md, to compare the folder with (default: the newest one)")
	fs.BoolVar(&full, "full", false, "every Definition is reached and every Manager works")
	fs.StringVar(&findings, "findings", "", "a Findings page whose Findings become checklist items")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	ws, p, err := c.load(o)
	if err != nil {
		return Misused, err
	}
	if errs := p.SpecErrors(); len(errs) > 0 {
		for _, d := range errs {
			fmt.Fprintln(c.Stderr, d.String())
		}
		fmt.Fprintf(c.Stderr, "markitect impact: the spec has %d errors; impact needs a spec that compiles\n", len(errs))
		return Failed, nil
	}
	name, cert, err := run.BaseCertificate(ws.Root(), base)
	if err != nil {
		return Misused, err
	}
	if cert == nil {
		return Misused, fmt.Errorf("no certificate yet, so there is no base to compare with: gate writes one when work passes")
	}
	baseProject, err := cert.Project()
	if err != nil {
		return Misused, err
	}
	hashes, err := run.HashFiles(ws)
	if err != nil {
		return Misused, err
	}
	in := analysis.ImpactInput{Base: baseProject, Current: p, Changed: cert.Changed(hashes), Full: full}
	if findings != "" {
		data, err := os.ReadFile(findings)
		if err != nil {
			return Misused, err
		}
		list, errs := analysis.ParseFindings(findings, data)
		if len(errs) > 0 {
			for _, d := range errs {
				fmt.Fprintln(c.Stderr, d.String())
			}
			return Failed, nil
		}
		in.Findings = list
	}
	im := analysis.ComputeImpact(in)
	if o.json {
		return OK, c.printJSON(struct {
			Base string `json:"base"`
			analysis.Impact
		}{name, im})
	}
	fmt.Fprint(c.Stdout, renderImpact(name, im))
	return OK, nil
}
