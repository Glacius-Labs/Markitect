// Package cli is the markitect command line: install, update, init, check, lock,
// refs, graph, impact and lint; apply and verify, which open a Run and say which
// agent the owner's session starts; the commands of a Run; version and licenses.
// None of them starts an agent.
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
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// Workspace is the working tree of the project's repository.
type Workspace interface {
	project.Repository
	Root() string
	Write(file string, data []byte) error
	Exists(file string) bool
	List(dir string) ([]string, error)
	Remove(file string) error
	// CommitOwn commits the files init wrote with the user's identity, and nothing
	// else, and says whether there was anything to commit.
	CommitOwn(files []string, message string) (bool, error)
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
	// Create makes a folder that lies in no repository a Git repository.
	Create(ctx context.Context, dir string) error
	Base(ctx context.Context, ws Workspace, rev string) (Base, error)
}

// Exit codes.
const (
	OK       = 0
	Failed   = 1 // the command ran and found errors
	Misused  = 2 // the command could not run
	usageTop = skills.About + `
No command starts an agent: apply and verify say which one your session starts.

Usage:
  markitect install                               tell Claude Code what Markitect is, in every folder
  markitect update [--repo DIR] [--check]         after an upgrade: refresh the personal skill, and the project's skills, agents and lock
  markitect init   [--repo DIR] [--name TITLE]   set up a project and install the harness
  markitect check  [--repo DIR] [--json] [--spec] compile the project; --spec looks only at the spec
  markitect lock   [--repo DIR]                   write the lock from the manifest
  markitect refs   [--repo DIR] [--json] TARGET   who links to a Definition or Package
  markitect graph  [--repo DIR] [--json]          the Package dependency graph
  markitect impact [--repo DIR] [--json] [--base COMMIT] [--full] [--findings FILE]
                                                  what a change reaches, with distance and reason
  markitect lint   [--repo DIR] [--json]          measure the spec like code: warnings only
  markitect apply  [--repo DIR] [--base COMMIT] [--full] [--findings FILE] [--deadline D]
                                                  open a run: a builder agent makes the project match the spec, and the work lands
  markitect verify [--repo DIR] [--base COMMIT] [--full] [--deadline D]
                                                  open a review of the last commit; changes nothing you have
  markitect status [--repo DIR]                   how the runs of this branch stand
  markitect stop   [--repo DIR] WHY               end a run without landing
  markitect version [--repo DIR]                  which markitect this is, and whether the personal skill and the project match it
  markitect licenses

Called from inside a run:
  markitect checklist [--review]                  the task and the list
  markitect answer [FILE]                         store a reviewer's answers
  markitect submit                                the one door: says what is next, and lands the work
`
)

// CLI runs one command.
type CLI struct {
	Git    Git
	Runs   Runs
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// PathWarning says what is wrong when the PATH does not lead to this
	// markitect, or "" when nothing is. Nil means no check.
	PathWarning func() string
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
	case "apply":
		code, err = c.start(ctx, "apply", rest)
	case "verify":
		code, err = c.start(ctx, "verify", rest)
	case "checklist":
		code, err = c.checklist(ctx, rest)
	case "answer":
		code, err = c.answer(ctx, rest)
	case "submit":
		code, err = c.submit(ctx, rest)
	case "stop":
		code, err = c.stop(ctx, rest)
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
			fmt.Fprintf(c.Stderr, "markitect %s: %v\n", cmd, hintInit(err))
		}
		return Misused
	}
	switch cmd {
	case "init", "check", "lock", "refs", "graph", "impact", "lint", "apply", "verify", "update":
		c.notice(ctx, rest)
	}
	switch cmd {
	case "check", "lock", "refs", "graph", "impact", "lint", "status":
		c.staleNotice(ctx, rest)
	}
	return code
}

// hintInit tells the user who ran a command outside any Git repository how to
// start a project. Both Git adapters word the error alike; a test pins it.
func hintInit(err error) error {
	if strings.Contains(err.Error(), "is not inside a Git repository") {
		return fmt.Errorf("%w; markitect init makes this folder a Markitect project", err)
	}
	return err
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
	switch name {
	case "init", "lock", "update", "version":
	default:
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
	_, p, err := c.load(ctx, o)
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
			fmt.Fprintf(c.Stdout, "In the realization, so spec and code do not match yet; markitect apply settles these:\n")
			for _, d := range realization {
				fmt.Fprintln(c.Stdout, d.String())
			}
		}
		switch {
		case nothingBuilt:
			fmt.Fprintf(c.Stdout, "ok: the spec compiles, with %d Definitions in %d Packages. Nothing is built yet, and markitect apply builds it.\n", len(p.Model.ModelDefinitions()), len(p.Model.Packages))
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
	fs.StringVar(&base, "base", "", "the commit to compare the working tree with (default: the work of the last landing)")
	fs.BoolVar(&full, "full", false, "every Definition is reached and every Manager works")
	fs.StringVar(&findings, "findings", "", "a Findings page whose Findings become checklist items")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	ws, p, err := c.load(ctx, o)
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
	if base == "" {
		e, err := c.engine(ctx, o.repo)
		if err != nil {
			return Misused, err
		}
		landed, ok, err := e.LastBase(ctx, o.repo)
		if err != nil {
			return Misused, err
		}
		if !ok {
			return Misused, fmt.Errorf("nothing has landed yet, so there is no default base; name one with --base")
		}
		base = landed
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
