package cli

import (
	"context"
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

// Runs opens the engine of a Run for a repository, or the engine of the work
// without a Run for a project folder, which needs no Git.
type Runs interface {
	Open(ctx context.Context, dir string) (*run.Engine, error)
	OpenFolder(ctx context.Context, dir string) (*run.Engine, error)
}

func (c CLI) engine(ctx context.Context, dir string) (*run.Engine, error) {
	if c.Runs == nil {
		return nil, errors.New("this build starts no agents")
	}
	return c.Runs.Open(ctx, dir)
}

// folderEngine is the engine of checklist, answer and gate, which work on the
// project folder and need no Git.
func (c CLI) folderEngine(ctx context.Context, dir string) (*run.Engine, error) {
	if c.Runs == nil {
		return nil, errors.New("this build reads no project folder")
	}
	return c.Runs.OpenFolder(ctx, dir)
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
		list, err := readFindings(findings)
		if err != nil {
			return c.startFailed(skill, err)
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

// readFindings reads a Findings page.
func readFindings(file string) ([]analysis.Finding, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	list, errs := analysis.ParseFindings(file, data)
	if len(errs) > 0 {
		var b strings.Builder
		for _, d := range errs {
			fmt.Fprintln(&b, d.String())
		}
		return nil, errors.New(b.String())
	}
	return list, nil
}

// checklist prints the task and the list: a Run's in a folder with an open one,
// otherwise the change from the newest certificate to the folder. Exit 1 means
// there is no list, because the spec has errors.
func (c CLI) checklist(ctx context.Context, args []string) (int, error) {
	var o options
	var findings string
	req := run.ChecklistRequest{}
	fs := flag.NewFlagSet("markitect checklist", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the project")
	fs.BoolVar(&req.Review, "review", false, "print what the review must answer")
	fs.StringVar(&req.Base, "base", "", "without a run: the certificate the change starts from (default: the newest one)")
	fs.BoolVar(&req.Full, "full", false, "without a run: every Definition is reached")
	fs.StringVar(&findings, "findings", "", "without a run: a Findings page whose Findings become checklist items")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	if findings != "" {
		list, err := readFindings(findings)
		if err != nil {
			return Misused, err
		}
		req.Findings = list
	}
	e, err := c.folderEngine(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	req.Dir = o.repo
	text, ok, err := e.Checklist(ctx, req)
	if err != nil {
		return Misused, err
	}
	fmt.Fprint(c.Stdout, text)
	if !ok {
		return Failed, nil
	}
	return OK, nil
}

// gate checks the work in the project folder without a run. Exit 0 means it
// passed and wrote its certificate, the folder matches its certificate, or the
// report of a review is written; 1 that the output is the next step.
func (c CLI) gate(ctx context.Context, args []string) (int, error) {
	var o options
	req := run.GateRequest{}
	fs := flag.NewFlagSet("markitect gate", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the project")
	fs.BoolVar(&req.Verify, "verify", false, "only check that the folder matches its newest certificate")
	fs.BoolVar(&req.Findings, "findings", false, "close a review of finished work: write its Findings into the folder, and no certificate")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	e, err := c.folderEngine(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	req.Dir = o.repo
	v, err := e.Gate(ctx, req)
	if err != nil {
		return Misused, err
	}
	fmt.Fprint(c.Stdout, v.Text)
	if !v.Done {
		return Failed, nil
	}
	return OK, nil
}

// answer stores a reviewer's answers from a file or standard input. Exit 0 means
// every item has an answer, 1 that something is missing or was refused.
func (c CLI) answer(ctx context.Context, args []string) (int, error) {
	var o options
	fs := flag.NewFlagSet("markitect answer", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the project")
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
	e, err := c.folderEngine(ctx, o.repo)
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
	fs.StringVar(&o.repo, "repo", ".", "a folder of the project")
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

// hook is what the project's hook calls before a shell command. It reads the
// provider's input on standard input and always exits 0: a problem of its own
// must not stop an agent.
func (c CLI) hook(_ context.Context, args []string) (int, error) {
	if len(args) != 1 || args[0] != "pretool" {
		return Misused, errors.New("names its event: markitect hook pretool")
	}
	input, err := io.ReadAll(c.Stdin)
	if err != nil {
		return OK, nil
	}
	if out := run.AnswerGuard(input); len(out) > 0 {
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
	dir := repoArg(args)
	e, err := c.Runs.Open(ctx, dir)
	if err != nil {
		return
	}
	fmt.Fprint(c.Stderr, e.Notice(ctx, dir))
}

// repoArg is the folder a command line names with --repo, or the current folder.
func repoArg(args []string) string {
	dir := "."
	for i, a := range args {
		if a == "--repo" && i+1 < len(args) {
			dir = args[i+1]
		}
		if v, ok := strings.CutPrefix(a, "--repo="); ok {
			dir = v
		}
	}
	return dir
}
