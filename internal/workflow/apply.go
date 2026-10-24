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
	base.Findings = append([]run.Finding(nil), assessment.Findings...)
	for {
		result, e := execute(run.Implement)
		if e != nil || result.Decision != run.Submitted {
			return out, e
		}
		base.Input = *result.Candidate
		verified, e := execute(run.Verify)
		if e != nil || verified.Decision != run.Reject {
			return out, e
		}
		base.Findings = append([]run.Finding(nil), verified.Findings...)
		base.Input = *verified.Candidate
	}
}
