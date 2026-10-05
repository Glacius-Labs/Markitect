package cli

import (
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/host"
)

func runProjection(o commandOptions, p *host.Project, emit func(any) int, fail func(error) int) int {

	configPath, coveragePath, active, err := host.RegisteredProjection(p)
	if err != nil {
		return fail(err)
	}
	if active && (o.reviewConfig != configPath || o.coverage != coveragePath) {
		return fail(fmt.Errorf("explicit projection configuration differs from registered Project ownership"))
	}
	toolDigest, err := currentToolDigest()
	if err != nil {
		return fail(err)
	}
	if o.action == "plan" {
		plan, err := host.PlanRepresentations(p, o.reviewConfig, o.coverage, version, toolDigest)
		if err != nil {
			return fail(err)
		}
		return emit(plan) // A ready work plan is not a convergence or acceptance claim.
	}
	var report host.ProjectionReport
	switch o.action {
	case "observe":
		report, err = host.ObserveRepresentations(p, o.reviewConfig, o.coverage, version, toolDigest)
	case "verify":
		report, err = host.VerifyRepresentations(p, o.reviewConfig, o.coverage, version, toolDigest)
	case "apply":
		plan, readErr := host.ReadRepresentationPlan(o.plan)
		if readErr != nil {
			return fail(readErr)
		}
		var candidate *host.Materialization
		if o.reviewReport != "" {
			value, readErr := host.ReadMaterialization(o.reviewReport)
			if readErr != nil {
				return fail(readErr)
			}
			candidate = &value
		}
		report, err = host.ApplyRepresentations(o.root, p, o.reviewConfig, o.coverage, version, toolDigest, plan, candidate, o.expect)
	}
	if err != nil {
		if len(report.Written) > 0 {
			if code := emit(report); code != 0 {
				return code
			}
		}
		return fail(err)
	}
	if code := emit(report); code != 0 {
		return code
	}
	switch report.Status {
	case "drift", "blocked", "failed":
		return 1
	case "incomplete":
		return 2
	}
	return 0
}
