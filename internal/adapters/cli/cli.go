// Package cli is the markitect command line: init, check, lock, refs, graph,
// impact and lint, which start no agent; apply and verify, which start a Run;
// desktop, which opens the app over these commands; version and licenses.
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

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Workspace is the working tree of the project's repository.
type Workspace interface {
	project.Repository
	Root() string
	Write(file string, data []byte) error
	Exists(file string) bool
	List(dir string) ([]string, error)
	Remove(file string) error
}

// Base is the state of the repository at a base commit.
type Base interface {
	project.Repository
	ID() string
	// Changed lists the files that differ between the base and the working
	// tree, outside Markitect's folder.
	Changed() ([]analysis.Change, error)
}

// Git opens repository states.
type Git interface {
	Open(ctx context.Context, dir string) (Workspace, error)
	Base(ctx context.Context, ws Workspace, rev string) (Base, error)
}

// Desktop runs Markitect Desktop, the app over these commands.
type Desktop interface {
	Run(ctx context.Context, o DesktopOptions) error
}

// DesktopOptions are the flags of markitect desktop.
type DesktopOptions struct {
	Listen string // the loopback address to serve on
	Open   bool   // open a window on it
}

// Exit codes.
const (
	OK       = 0
	Failed   = 1 // the command ran and found errors
	Misused  = 2 // the command could not run
	usageTop = `markitect is the canonical model tool of Markitect. Only apply and verify start agents.

Usage:
  markitect init   [--repo DIR] [--name TITLE]   set up a project and install the skills
  markitect check  [--repo DIR] [--json]          compile the model: errors only
  markitect lock   [--repo DIR]                   write the lock from the manifest
  markitect refs   [--repo DIR] [--json] TARGET   who links to a Definition or Package
  markitect graph  [--repo DIR] [--json]          the Package dependency graph
  markitect impact [--repo DIR] [--json] --base COMMIT [--full] [--findings FILE]
                                                  what a change reaches, and each Manager's checklist
  markitect lint   [--repo DIR] [--json]          measure the model like code: warnings only
  markitect apply  [--repo DIR] [--json] [--base COMMIT] [--full] [--findings FILE] [--rounds N] [RUN OPTIONS]
                                                  realize the change with a Run and land it
  markitect verify [--repo DIR] [--json] [--base COMMIT] [--full] [--findings-out FILE] [RUN OPTIONS]
                                                  check the change with checkers only; changes nothing
        RUN OPTIONS: --remote NAME --branch NAME --run NAME --deadline D --starts N --start-timeout D
  markitect desktop [--listen ADDR] [--no-open]  open Markitect, the app over these commands
  markitect version
  markitect licenses
`
)

// CLI runs one command.
type CLI struct {
	Git     Git
	Runs    Runs
	Desktop Desktop
	Stdout  io.Writer
	Stderr  io.Writer
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
	case "apply":
		code, err = c.runCommand(ctx, agentport.Apply, rest)
	case "verify":
		code, err = c.runCommand(ctx, agentport.Verify, rest)
	case "desktop":
		code, err = c.desktop(ctx, rest)
	case "version":
		fmt.Fprintln(c.Stdout, "markitect", buildinfo.Build())
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
	fs.StringVar(&o.repo, "repo", ".", "a folder inside the project's Git repository")
	if name != "init" && name != "lock" {
		fs.BoolVar(&o.json, "json", false, "print JSON instead of Markdown")
	}
	return fs
}

// load opens the repository and loads the project in its working tree.
func (c CLI) load(ctx context.Context, o options) (Workspace, *project.Project, error) {
	ws, err := c.Git.Open(ctx, o.repo)
	if err != nil {
		return nil, nil, err
	}
	p, err := project.Load(ws)
	if err != nil {
		return nil, nil, err
	}
	return ws, p, nil
}

// warnErrors tells the user that an analysis ran over a model with errors.
func (c CLI) warnErrors(p *project.Project) {
	if n := len(p.Errors); n > 0 {
		fmt.Fprintf(c.Stderr, "markitect: the project has %d compile errors; run markitect check. The result below may be incomplete.\n", n)
	}
}

func (c CLI) printJSON(v any) error {
	enc := json.NewEncoder(c.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (c CLI) desktop(ctx context.Context, args []string) (int, error) {
	fs := flag.NewFlagSet("markitect desktop", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	var o DesktopOptions
	var noOpen bool
	fs.StringVar(&o.Listen, "listen", "127.0.0.1:0", "the loopback address to serve on; port 0 picks a free one")
	fs.BoolVar(&noOpen, "no-open", false, "print the address instead of opening a window")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	if c.Desktop == nil {
		return Misused, errors.New("this build has no desktop")
	}
	o.Open = !noOpen
	return OK, c.Desktop.Run(ctx, o)
}

func (c CLI) check(ctx context.Context, args []string) (int, error) {
	var o options
	fs := c.flags("check", &o)
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	_, p, err := c.load(ctx, o)
	if err != nil {
		return Misused, err
	}
	if o.json {
		errs := p.Errors
		if errs == nil {
			errs = model.Diagnostics{}
		}
		if err := c.printJSON(map[string]any{"ok": len(p.Errors) == 0, "errors": errs}); err != nil {
			return Misused, err
		}
	} else {
		for _, d := range p.Errors {
			fmt.Fprintln(c.Stdout, d.String())
		}
		if len(p.Errors) == 0 {
			fmt.Fprintf(c.Stdout, "ok: %d Definitions in %d Packages, %d files accounted\n", len(p.Model.ModelDefinitions()), len(p.Model.Packages), len(p.Registry.Owner))
		} else {
			fmt.Fprintf(c.Stdout, "%d errors\n", len(p.Errors))
		}
	}
	if len(p.Errors) > 0 {
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
	_, p, err := c.load(ctx, o)
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
	_, p, err := c.load(ctx, o)
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
	_, p, err := c.load(ctx, o)
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

func (c CLI) impact(ctx context.Context, args []string) (int, error) {
	var o options
	var base, findings string
	var full bool
	fs := c.flags("impact", &o)
	fs.StringVar(&base, "base", "", "the commit to compare the working tree with (required)")
	fs.BoolVar(&full, "full", false, "every Definition is reached and every Manager works")
	fs.StringVar(&findings, "findings", "", "a Findings page whose Findings become checklist items")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 || base == "" {
		return Misused, fmt.Errorf("needs --base COMMIT and takes no arguments")
	}
	ws, p, err := c.load(ctx, o)
	if err != nil {
		return Misused, err
	}
	if len(p.Errors) > 0 {
		for _, d := range p.Errors {
			fmt.Fprintln(c.Stderr, d.String())
		}
		fmt.Fprintf(c.Stderr, "markitect impact: the project has %d compile errors; impact needs a model that compiles\n", len(p.Errors))
		return Failed, nil
	}
	b, err := c.Git.Base(ctx, ws, base)
	if err != nil {
		return Misused, err
	}
	baseProject, err := project.Load(b)
	if err != nil {
		return Misused, err
	}
	changed, err := b.Changed()
	if err != nil {
		return Misused, err
	}
	in := analysis.ImpactInput{Base: baseProject, Current: p, Changed: changed, Full: full}
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
		}{b.ID(), im})
	}
	fmt.Fprint(c.Stdout, renderImpact(b.ID(), im))
	return OK, nil
}
