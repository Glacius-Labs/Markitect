package main

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/app"
)

func runReconcile(o commandOptions, p *app.Project, emit func(any) int, fail func(error) int) int {
	toolDigest, err := currentToolDigest()
	if err != nil {
		return fail(err)
	}
	if o.adapter != "markitect-render" {
		var result app.AdapterResult
		switch o.action {
		case "observe":
			result, err = app.ObserveCommandAdapter(p, o.adapter)
		case "plan":
			plan, planErr := app.PlanCommandAdapter(p, o.adapter, version, toolDigest)
			if planErr != nil {
				return fail(planErr)
			}
			return emit(plan)
		case "apply":
			plan, planErr := app.ReadCommandAdapterPlan(o.plan)
			if planErr != nil {
				return fail(planErr)
			}
			result, err = app.ApplyCommandAdapter(p, o.adapter, version, toolDigest, plan)
		case "verify":
			plan, planErr := app.ReadCommandAdapterPlan(o.plan)
			if planErr != nil {
				return fail(planErr)
			}
			result, err = app.VerifyCommandAdapter(p, o.adapter, version, toolDigest, plan)
		default:
			return fail(fmt.Errorf("unsupported reconciliation action %q", o.action))
		}
		if err != nil {
			return fail(err)
		}
		if code := emit(result); code != 0 {
			return code
		}
		return adapterResultExitCode(result)
	}

	switch o.action {
	case "observe":
		observation, observeErr := app.ObserveProjection(p, version, toolDigest)
		if observeErr != nil {
			return fail(observeErr)
		}
		status := "reconciled"
		if len(observation.Drift) > 0 || len(observation.Stale) > 0 {
			status = "drift"
		}
		if code := emit(map[string]any{"status": status, "observation": observation}); code != 0 {
			return code
		}
		if status == "drift" {
			return 1
		}
		return 0
	case "plan":
		plan, planErr := app.PlanProjection(p, version, toolDigest)
		if planErr != nil {
			return fail(planErr)
		}
		if code := emit(plan); code != 0 {
			return code
		}
		if plan.Status == "incomplete" {
			return 2
		}
		return 0
	case "apply":
		plan, planErr := app.ReadPlan(o.plan)
		if planErr != nil {
			return fail(planErr)
		}
		written, applyErr := app.ApplyProjection(o.root, p, plan, version, toolDigest)
		if applyErr != nil {
			return fail(applyErr)
		}
		return emit(map[string]any{"status": "applied", "written": written, "plan": o.plan})
	case "verify":
		plan, planErr := app.ReadPlan(o.plan)
		if planErr != nil {
			return fail(planErr)
		}
		if verifyErr := app.VerifyProjectionPlan(p, plan, version, toolDigest); verifyErr != nil {
			return fail(verifyErr)
		}
		return emit(map[string]any{"status": "verified", "adapter": o.adapter, "sourceDigest": plan.SourceDigest, "operations": len(plan.Operations)})
	default:
		return fail(fmt.Errorf("unsupported reconciliation action %q", o.action))
	}
}

func adapterResultExitCode(result app.AdapterResult) int {
	if result.Status == "incomplete" {
		return 2
	}
	if result.Status == "failed" {
		return 1
	}
	for _, finding := range result.Findings {
		if finding.Severity == "error" {
			return 1
		}
	}
	return 0
}
