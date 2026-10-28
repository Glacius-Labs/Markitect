package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Runs starts the Runs of apply and verify.
type Runs interface {
	Run(ctx context.Context, o RunOptions) (*run.Result, error)
}

// RunOptions are one apply or verify as the command line states it.
type RunOptions struct {
	// Repo is a folder inside the project's repository.
	Repo string
	// Remote and Branch name the target; empty values take the current
	// branch and its upstream.
	Remote, Branch string
	Request        run.Request
}

// runCommand runs apply or verify. Exit 0 means landed (apply), every item
// conforms (verify) or nothing to do; 1 that the Run ran and did not land or
// found something; 2 that it could not run.
func (c CLI) runCommand(ctx context.Context, mode agentport.Mode, args []string) (int, error) {
	var o options
	var opt RunOptions
	var findings, findingsOut string
	var deadline time.Duration
	lim := run.DefaultLimits(time.Now())
	fs := c.flags(string(mode), &o)
	fs.StringVar(&opt.Request.Base, "base", "", "the commit impact is computed from (default: the last commit Markitect landed)")
	fs.BoolVar(&opt.Request.Full, "full", false, "every Definition is reached and every Manager works")
	fs.StringVar(&opt.Remote, "remote", "", "the remote to start from and land on (default: the current branch's upstream, else origin, else this repository)")
	fs.StringVar(&opt.Branch, "branch", "", "the target branch (default: the upstream's branch, else the current one)")
	fs.StringVar(&opt.Request.Run, "run", "", "the Run's name (default: from the time)")
	fs.DurationVar(&deadline, "deadline", 4*time.Hour, "how long the whole Run may take")
	fs.IntVar(&lim.Starts, "starts", lim.Starts, "how many agent starts the Run may make")
	fs.IntVar(&lim.Rounds, "rounds", lim.Rounds, "how many maker or checker starts one Manager may make")
	fs.DurationVar(&lim.Start, "start-timeout", lim.Start, "how long one agent start may take")
	if mode == agentport.Apply {
		fs.StringVar(&findings, "findings", "", "a Findings page whose Findings become checklist items")
	} else {
		fs.StringVar(&findingsOut, "findings-out", "", "write the Findings as a Findings page, the input of apply --findings")
	}
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	if c.Runs == nil {
		return Misused, errors.New("this build starts no agents")
	}
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
		opt.Request.Findings = list
	}
	lim.Deadline = time.Now().Add(deadline)
	opt.Repo, opt.Request.Mode, opt.Request.Limits = o.repo, mode, lim
	r, err := c.Runs.Run(ctx, opt)
	if err != nil {
		return Misused, err
	}
	if findingsOut != "" {
		if err := os.WriteFile(findingsOut, run.FindingsPage(r), 0o644); err != nil {
			return Misused, err
		}
	}
	if o.json {
		if err := c.printJSON(r); err != nil {
			return Misused, err
		}
	} else {
		c.Stdout.Write(run.Page(r))
	}
	switch r.State {
	case run.StateLanded, run.StateVerified, run.StateNothing:
		return OK, nil
	}
	return Failed, nil
}
