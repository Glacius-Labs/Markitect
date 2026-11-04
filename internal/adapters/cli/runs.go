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
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Runs opens the engine of a Run for a repository, or for a Run's store.
type Runs interface {
	Open(ctx context.Context, dir string) (*run.Engine, error)
	OpenStore(store string) *run.Engine
}

func (c CLI) engine(ctx context.Context, dir string) (*run.Engine, error) {
	if c.Runs == nil {
		return nil, errors.New("this build starts no agents")
	}
	return c.Runs.Open(ctx, dir)
}

// start runs apply or verify. Exit 0 means a Run started, one is already open or
// there is nothing to do; 1 that the spec stops it. With --skill it always exits 0
// and prints everything on standard output, for the skill that calls it: a
// provider aborts a skill on any other exit, and then its agent sees nothing.
func (c CLI) start(ctx context.Context, name string, args []string) (int, error) {
	var o options
	var findings string
	var skill bool
	var deadline time.Duration
	var req run.Request
	fs := flag.NewFlagSet("markitect "+name, flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder inside the project's Git repository")
	fs.StringVar(&req.Base, "base", "", "the commit impact is computed from (default: the work of the last landing)")
	fs.BoolVar(&req.Full, "full", false, "every Definition is reached")
	fs.BoolVar(&skill, "skill", false, "print everything on standard output and exit 0, for the skill that calls this")
	fs.DurationVar(&deadline, "deadline", 0, "how long the Run may take (default: 6h)")
	if name == "apply" {
		fs.StringVar(&findings, "findings", "", "a Findings page whose Findings become checklist items")
	}
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	e, err := c.engine(ctx, o.repo)
	if err != nil {
		return c.startFailed(skill, err)
	}
	if findings != "" {
		data, err := os.ReadFile(findings)
		if err != nil {
			return c.startFailed(skill, err)
		}
		list, errs := analysis.ParseFindings(findings, data)
		if len(errs) > 0 {
			var b strings.Builder
			for _, d := range errs {
				fmt.Fprintln(&b, d.String())
			}
			return c.startFailed(skill, errors.New(b.String()))
		}
		req.Findings = list
	}
	req.Dir, req.Deadline = o.repo, deadline
	var started run.Started
	if name == "apply" {
		started, err = e.Apply(ctx, req)
	} else {
		started, err = e.Verify(ctx, req)
	}
	if err != nil {
		return c.startFailed(skill, err)
	}
	fmt.Fprint(c.Stdout, started.Text)
	if skill {
		return OK, nil
	}
	switch started.State {
	case "", "open", run.StateNothing:
		return OK, nil
	}
	return Failed, nil
}

func (c CLI) startFailed(skill bool, err error) (int, error) {
	if skill {
		fmt.Fprintf(c.Stdout, "markitect could not start the run: %v\n", err)
		return OK, nil
	}
	return Misused, err
}

func (c CLI) checklist(ctx context.Context, args []string) (int, error) {
	var o options
	var review bool
	fs := flag.NewFlagSet("markitect checklist", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder inside the Run's worktree")
	fs.BoolVar(&review, "review", false, "print what the review must answer")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	e, err := c.engine(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	text, err := e.Checklist(ctx, o.repo, review)
	if err != nil {
		return Misused, err
	}
	fmt.Fprint(c.Stdout, text)
	return OK, nil
}

// answer stores a reviewer's answers from a file or standard input. Exit 0 means
// every item has an answer, 1 that something is missing or was refused.
func (c CLI) answer(ctx context.Context, args []string) (int, error) {
	var o options
	fs := flag.NewFlagSet("markitect answer", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder inside the Run's worktree")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	var raw []byte
	var err error
	switch fs.NArg() {
	case 0:
		raw, err = io.ReadAll(c.Stdin)
	case 1:
		raw, err = os.ReadFile(fs.Arg(0))
	default:
		return Misused, errors.New("takes at most one file; without one it reads standard input")
	}
	if err != nil {
		return Misused, err
	}
	e, err := c.engine(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	text, complete, err := e.Answer(ctx, o.repo, raw)
	if err != nil {
		return Misused, err
	}
	fmt.Fprint(c.Stdout, text)
	if !complete {
		return Failed, nil
	}
	return OK, nil
}

// submit is the one door. Exit 0 means the Run is closed and the session should
// end; 1 that the output is the next step.
func (c CLI) submit(ctx context.Context, args []string) (int, error) {
	var o options
	fs := flag.NewFlagSet("markitect submit", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder inside the Run's worktree")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	e, err := c.engine(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	v, err := e.Submit(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	fmt.Fprint(c.Stdout, v.Text)
	if !v.Done {
		return Failed, nil
	}
	return OK, nil
}

func (c CLI) stop(ctx context.Context, args []string) (int, error) {
	var o options
	var environment bool
	fs := flag.NewFlagSet("markitect stop", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the repository")
	fs.BoolVar(&environment, "environment", false, "from a session: the machine lacks something, which is no question about the spec")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	why := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if why == "" {
		return Misused, errors.New("says why: markitect stop \"why\"")
	}
	e, err := c.engine(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	text, err := e.Stop(ctx, o.repo, why, environment)
	if err != nil {
		return Misused, err
	}
	fmt.Fprint(c.Stdout, text)
	return OK, nil
}

func (c CLI) status(ctx context.Context, args []string) (int, error) {
	var o options
	fs := flag.NewFlagSet("markitect status", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the repository")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	e, err := c.engine(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	text, err := e.Status(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	fmt.Fprint(c.Stdout, text)
	return OK, nil
}

// hook is what the provider's hooks call. It reads their input on standard input
// and always exits 0: a problem of its own must not stop an agent.
func (c CLI) hook(ctx context.Context, args []string) (int, error) {
	if len(args) == 0 {
		return Misused, errors.New("names an event")
	}
	event := args[0]
	var store, runName string
	fs := flag.NewFlagSet("markitect hook", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&store, "store", "", "the run state's folder")
	fs.StringVar(&runName, "run", "", "the Run")
	if err := fs.Parse(args[1:]); err != nil {
		return Misused, err
	}
	if c.Runs == nil || store == "" || runName == "" {
		return OK, nil
	}
	input, err := io.ReadAll(c.Stdin)
	if err != nil || !json.Valid(input) {
		return OK, nil
	}
	if out := c.Runs.OpenStore(store).Hook(ctx, event, runName, input); len(out) > 0 {
		c.Stdout.Write(out)
	}
	return OK, nil
}

// notice prints the one line about a Run the owner should hear of. It is best
// effort and never changes a command's result.
func (c CLI) notice(ctx context.Context, args []string) {
	if c.Runs == nil {
		return
	}
	dir := "."
	for i, a := range args {
		if a == "--repo" && i+1 < len(args) {
			dir = args[i+1]
		}
		if v, ok := strings.CutPrefix(a, "--repo="); ok {
			dir = v
		}
	}
	e, err := c.Runs.Open(ctx, dir)
	if err != nil {
		return
	}
	fmt.Fprint(c.Stderr, e.Notice(ctx, dir))
}
