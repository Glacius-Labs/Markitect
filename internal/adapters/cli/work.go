package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// work is the engine of checklist, answer, gate and status, which work on the
// project folder (DEC-096, DEC-097).
func (c CLI) work() (*run.Engine, error) {
	if c.Work == nil {
		return nil, errors.New("this build does no work on a project folder")
	}
	return c.Work, nil
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

// checklist prints the task and the list of the change from the newest
// certificate to the folder. Exit 1 means there is no list, because the spec has
// errors. With --start it prints the step of the session that drives the work
// instead, and always exits 0 with everything on standard output, for the skill
// that calls it: a provider aborts a skill on any other exit, and then its agent
// sees nothing.
func (c CLI) checklist(ctx context.Context, args []string) (int, error) {
	var o options
	var findings string
	req := run.ChecklistRequest{}
	fs := flag.NewFlagSet("markitect checklist", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the project")
	fs.BoolVar(&req.Review, "review", false, "print what the review must answer")
	fs.StringVar(&req.Base, "base", "", "the certificate the change starts from (default: the newest one)")
	fs.BoolVar(&req.Full, "full", false, "every Definition is reached")
	fs.StringVar(&findings, "findings", "", "a Findings page whose Findings become checklist items")
	fs.BoolVar(&req.Start, "start", false, "begin the work and say which agent to start, for the skill that drives it; exits 0")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	failed := func(err error) (int, error) {
		if req.Start {
			fmt.Fprintf(c.Stdout, "markitect could not begin the work: %v\n", err)
			return OK, nil
		}
		return Misused, err
	}
	if findings != "" {
		list, err := readFindings(findings)
		if err != nil {
			return failed(err)
		}
		req.Findings = list
	}
	e, err := c.work()
	if err != nil {
		return failed(err)
	}
	req.Dir = o.repo
	text, ok, err := e.Checklist(ctx, req)
	if err != nil {
		return failed(err)
	}
	fmt.Fprint(c.Stdout, text)
	if !ok && !req.Start {
		return Failed, nil
	}
	return OK, nil
}

// gate checks the work in the project folder. Exit 0 means it passed and wrote
// its certificate, the folder matches its certificate, or the report of a review
// is written; 1 that the output is the next step.
func (c CLI) gate(ctx context.Context, args []string) (int, error) {
	var o options
	req := run.GateRequest{}
	fs := flag.NewFlagSet("markitect gate", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the project")
	fs.BoolVar(&req.Verify, "verify", false, "only check that the folder matches its newest certificate")
	fs.BoolVar(&req.Findings, "findings", false, "close a review of finished work: write its Findings into the folder, and no certificate")
	fs.StringVar(&req.SpecProblem, "spec-problem", "", "from the builder: the spec has a problem only the owner can settle, which a reviewer confirms first")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments; quote the text of --spec-problem")
	}
	// An empty report is refused rather than taken for none.
	empty := false
	fs.Visit(func(f *flag.Flag) {
		empty = empty || (f.Name == "spec-problem" && strings.TrimSpace(req.SpecProblem) == "")
	})
	if empty {
		return Misused, errors.New("--spec-problem says what the problem is, and where in the spec")
	}
	e, err := c.work()
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
	e, err := c.work()
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

// status says how the work in the folder stands, and changes nothing.
func (c CLI) status(ctx context.Context, args []string) (int, error) {
	var o options
	fs := flag.NewFlagSet("markitect status", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	fs.StringVar(&o.repo, "repo", ".", "a folder of the project")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	e, err := c.work()
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
	if out := run.Guard(input); len(out) > 0 {
		c.Stdout.Write(out)
	}
	return OK, nil
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
