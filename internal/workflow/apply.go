// Package workflow selects fully closed Runs. It owns no Git, storage or provider.
package workflow

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type Executor interface {
	Execute(context.Context, run.Request) (run.Result, error)
}

type Result struct {
	Runs  []run.Result `json:"runs"`
	Final run.Result   `json:"final"`
}

// Apply uses the same original allowance for every mandate and correction.
// Assessment approval enables Implement even when implementation changes nothing.
func Apply(ctx context.Context, executor Executor, base run.Request) (out Result, err error) {
	if executor == nil {
		return out, fmt.Errorf("workflow executor unavailable")
	}
	// The bound input stays the command's capture across correction loops, so
	// every Verify judges all changes made since then.
	if base.Origin == (run.Origin{}) {
		base.Origin.Input = base.Input
	}
	sequence := 0
	execute := func(m run.Mandate) (run.Result, error) {
		if err := ctx.Err(); err != nil {
			return run.Result{}, err
		}
		sequence++
		r := base
		r.ID = base.ID + "-" + strconv.Itoa(sequence) + "-" + string(m)
		r.Mandate = m
		result, e := executor.Execute(ctx, r)
		out.Runs = append(out.Runs, result)
		out.Final = result
		if e == nil {
			e = run.ValidateClosed(result, r)
		}
		return result, e
	}
	assessment, err := execute(run.Assessment)
	if err != nil || assessment.Decision != run.Approve {
		return out, err
	}
	// Only Findings that name a current Definition reach Implement; Assessment
	// and Verify keep the others as Notes.
	base.Findings = run.Initial(base.Prepared, assessment.Findings)
	for {
		result, e := execute(run.Implement)
		if e != nil || result.Decision != run.Submitted {
			return out, e
		}
		// Verify sees what Implement left out of the candidate as build outputs.
		base.Input, base.BuildOutputs = *result.Candidate, result.BuildOutputs()
		verified, e := execute(run.Verify)
		if e != nil || verified.Decision != run.Reject {
			return out, e
		}
		// Implement and Verify each need every Manager once. A pair the
		// remaining allowance cannot start ends the command incomplete; the
		// check promises no fit for a pair it starts.
		if err := base.Allowance.Covers(2 * len(base.Prepared.Topology().Managers())); err != nil {
			return out, fmt.Errorf("budget does not cover a correction pair, in which Implement and Verify each need every Manager once: %w", err)
		}
		base.Findings = run.Initial(base.Prepared, verified.Findings)
		base.Input, base.BuildOutputs = *verified.Candidate, nil
	}
}
