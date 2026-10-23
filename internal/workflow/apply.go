// Package workflow selects fully closed Runs. It owns no Git, storage or provider.
package workflow

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
			e = ValidateClosed(result, r)
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

// ValidateClosed rejects incomplete, misbound or uncertain port results before
// another Run can open. Review may retain identical objects under a new key.
func ValidateClosed(value run.Result, request run.Request) error {
	if !request.Prepared.Valid() {
		return fmt.Errorf("closed result requires checked Project preparation")
	}
	if value.ID != request.ID || value.Project != request.Project || value.Binding != request.Prepared.Binding() || value.Mandate != request.Mandate || value.Input != request.Input || value.Technical != run.Completed || len(value.Errors) != 0 || value.Candidate == nil {
		return fmt.Errorf("Run did not return a complete bound durable result")
	}
	c := *value.Candidate
	if c.Project != request.Project || c.Binding != value.Binding || c.Key == "" || c.Format != request.Input.Format || c.Commit == "" || c.Tree == "" {
		return fmt.Errorf("Run candidate binding mismatch")
	}
	if request.Mandate != run.Implement && (c.Commit != request.Input.Commit || c.Tree != request.Input.Tree) {
		return fmt.Errorf("review changed its fixed input candidate")
	}
	valid := value.Decision == run.ModelBlocker || value.Decision == run.Reject
	if request.Mandate == run.Implement {
		valid = valid || value.Decision == run.Submitted
	} else {
		valid = valid || value.Decision == run.Approve
	}
	if !valid {
		return fmt.Errorf("mandate returned an invalid decision")
	}
	if value.Decision == run.Reject || value.Decision == run.ModelBlocker {
		concrete := false
		for _, f := range value.Findings {
			concrete = concrete || strings.TrimSpace(f.Text) != ""
		}
		if !concrete {
			return fmt.Errorf("rejection or model blocker requires mandatory Findings")
		}
	}
	closed := map[string]bool{}
	for _, c := range value.Cleanup {
		if !c.Closed || c.Error != "" || closed[c.Resource] {
			return fmt.Errorf("Run cleanup is uncertain or duplicated")
		}
		closed[c.Resource] = true
	}
	for _, m := range request.Prepared.Topology().Managers() {
		if !closed["session:"+m.Package] || !closed["copy:"+m.Package] {
			return fmt.Errorf("Manager resources not confirmed closed")
		}
	}
	if !closed["manager-workers"] || !closed["git-service"] || len(value.Handoffs) != len(request.Prepared.Topology().Managers()) {
		return fmt.Errorf("complete hierarchy or Git closure missing")
	}
	return nil
}
