package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"

	projectconfig "github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"

	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// fullVerifyFrame owns phase state for one verification. The caller retains
// the deadline and the single report persistence boundary.
type fullVerifyFrame struct {
	ctx              context.Context
	host             Host
	invoker          Invoker
	root             string
	project          *projectvalue.Project
	runtime          projectconfig.Runtime
	binding          FullVerifyBinding
	runtimePin       string
	out              FullVerifyReport
	briefingContexts map[string]BriefingContext
	checkErr         error
	knownCost        int
	unknownCost      int
}

// prepareFullVerifyFrame runs all preflight gates before any check or Manager
// invocation. A false result means the report is complete enough to persist.
func prepareFullVerifyFrame(frame *fullVerifyFrame) bool {
	project, runtime, binding := frame.project, frame.runtime, frame.binding
	out := &frame.out
	if hasErrorFinding(project.Report.Findings) {
		out.Status, out.Error = "failed", "compiled project contains error findings"
		return false
	}
	if project.Config.CoverageMode != "full" {
		out.Status, out.Error = "incomplete", "full verification requires project CoverageMode: full"
		return false
	}
	if project.Coverage == nil {
		out.Status, out.Error = "incomplete", "full verification requires a repository coverage census bound to the project snapshot"
		return false
	}
	if !project.Coverage.Accounted || !project.Coverage.Conforming {
		out.Status, out.Error = "failed", "repository coverage census is not conforming"
		return false
	}
	if err := requireArtifacts(project.Report); err != nil {
		out.Status, out.Error = "failed", err.Error()
		return false
	}
	if err := validateCandidateDocument(frame.host, project); err != nil {
		out.Status, out.Error = "failed", err.Error()
		return false
	}
	managers := append([]projectmodel.Manager(nil), project.Report.Managers...)
	sortFullManagers(managers, project.Report.Managers)
	if len(managers) == 0 {
		out.Status, out.Error = "failed", "project has no active Managers"
		return false
	}
	if err := validateStrictnessManagers(runtime.Strictness, managers); err != nil {
		out.Status, out.Error = "failed", err.Error()
		return false
	}
	if binding.PriorStarts < 0 || binding.PriorCostMicros < 0 || binding.PriorStarts > runtime.Limits.MaxStarts || binding.PriorCostMicros > runtime.Limits.MaxCostMicros {
		out.Status, out.Error = "incomplete", "cumulative runtime budget already exceeded"
		return false
	}
	if len(binding.PreverifiedChecks) > binding.PriorStarts {
		out.Status, out.Error = "incomplete", "preverified checks are not covered by the cumulative start count"
		return false
	}
	for _, manager := range managers {
		profile, profileErr := projectconfig.ResolveStrictness(runtime, manager.ID)
		row := FullManagerAssessment{ManagerID: manager.ID, ParentID: manager.Parent, Strictness: profile, Assessments: []FullAssessment{}, Findings: []string{}}
		if profileErr != nil {
			row.Status, row.Error = "incomplete", profileErr.Error()
		}
		out.Managers = append(out.Managers, row)
	}
	frame.briefingContexts = make(map[string]BriefingContext, len(managers))
	briefingBindingInvalid := false
	for _, manager := range managers {
		briefing, briefingErr := managerBriefingIn(frame.host, frame.root, binding.acceptedBriefings, project.Report.ModelDigest, manager.ID, project.Revision)
		if briefingErr != nil {
			row := findFullManagerAssessment(out.Managers, manager.ID)
			row.Status, row.Error = "incomplete", briefingErr.Error()
			out.Error = "accepted model briefing is missing or stale for one or more Managers"
			briefingBindingInvalid = true
			continue
		}
		frame.briefingContexts[manager.ID] = briefing
		out.BriefingDigests[manager.ID] = briefing.Digest
		if len(binding.ExpectedBriefings) != 0 {
			expected, ok := binding.ExpectedBriefings[manager.ID]
			if !ok || expected != briefing.Digest {
				row := findFullManagerAssessment(out.Managers, manager.ID)
				row.Status, row.Error = "incomplete", "briefing digest differs from the caller binding"
				out.Error = ErrStale.Error()
				briefingBindingInvalid = true
			}
		}
	}
	if (len(binding.ExpectedBriefings) != 0 && len(binding.ExpectedBriefings) != len(managers)) || briefingBindingInvalid {
		out.Status, out.Error = "incomplete", "caller briefing binding does not cover every active Manager"
		return false
	}
	if len(binding.PreverifiedChecks) > 0 {
		if len(binding.PreverifiedChecks) > len(project.Report.Checks) {
			out.Status, out.Error = "incomplete", "preverified check binding contains unrelated entries"
			return false
		}
		out.Checks = append(out.Checks, binding.PreverifiedChecks...)
	}
	frame.knownCost, frame.unknownCost = binding.PriorKnownCostInvocations, binding.PriorUnknownCostInvocations
	return true
}

func sortFullManagers(managers []projectmodel.Manager, all []projectmodel.Manager) {
	managerMap := projectManagerMap(all)
	sort.Slice(managers, func(i, j int) bool {
		di, dj := managerDepth(managerMap, managers[i].ID), managerDepth(managerMap, managers[j].ID)
		if di != dj {
			return di > dj
		}
		return managers[i].ID < managers[j].ID
	})
}

// executeFullVerifyFrame runs checks and Manager audits against the preflight
// rows. The slice is not appended after row pointers are taken.
func executeFullVerifyFrame(frame *fullVerifyFrame) {
	project, runtime, binding := frame.project, frame.runtime, frame.binding
	out := &frame.out
	if out.Starts < runtime.Limits.MaxStarts && out.CostMicros <= runtime.Limits.MaxCostMicros {
		checks, checkErr := fullRunChecks(frame.ctx, frame.root, project, runtime, out, binding.CheckCandidateID)
		frame.checkErr = checkErr
		out.Checks = append(out.Checks, checks...)
	}
	if frame.checkErr != nil {
		out.Error = frame.checkErr.Error()
	}
	for i := range out.Managers {
		row := &out.Managers[i]
		if err := frame.ctx.Err(); err != nil {
			row.Status, row.Error = "incomplete", "cumulative duration budget exhausted"
			markRemainingFullRows(out.Managers, i+1, "cumulative duration budget exhausted")
			out.Error = "full verification did not finish within the cumulative duration limit"
			break
		}
		if out.Starts >= runtime.Limits.MaxStarts {
			row.Status, row.Error = "incomplete", "cumulative start budget exhausted"
			markRemainingFullRows(out.Managers, i+1, "cumulative start budget exhausted")
			out.Error = "full verification did not finish within the cumulative start limit"
			break
		}
		if out.CostMicros >= runtime.Limits.MaxCostMicros {
			row.Status, row.Error = "incomplete", "cumulative cost budget exhausted"
			markRemainingFullRows(out.Managers, i+1, "cumulative cost budget exhausted")
			out.Error = "full verification did not finish within the cumulative cost limit"
			break
		}
		if row.Error != "" {
			out.Error = "strictness could not be resolved for every Manager"
			continue
		}
		if _, ok := frame.briefingContexts[row.ManagerID]; !ok {
			continue
		}
		if err := freshFullBriefingBindings(frame.host, frame.root, binding.acceptedBriefings, project.Report.ModelDigest, out.BriefingDigests, project.Revision); err != nil {
			row.Status, row.Error = "incomplete", err.Error()
			out.Error = ErrStale.Error()
			markRemainingFullRows(out.Managers, i+1, "not started after stale briefing binding")
			break
		}
		children := fullChildAssessments(out.Managers, project.Report, row.ManagerID)
		integrationReviews := fullAuditIntegrationReviewsForManager(project.Report, row.ManagerID, binding.CheckCandidateID, binding.checkCandidateDigest, binding.freshReviews)
		assessment, callErr := fullAuditManager(frame.ctx, frame.host, frame.invoker, frame.root, project, runtime, row.ManagerID, row.Strictness, frame.briefingContexts[row.ManagerID], children, relevantManagerChecks(project.Report, row.ManagerID, out.Checks), integrationReviews, binding.CheckSource, "full-verify-"+binding.ExpectedSnapshot)
		*row = assessment
		if frame.recordManagerAudit(i, row, callErr) {
			break
		}
	}
}

func (frame *fullVerifyFrame) recordManagerAudit(i int, row *FullManagerAssessment, callErr error) bool {
	project, runtime, binding := frame.project, frame.runtime, frame.binding
	out := &frame.out
	attempted := row.Receipt != nil && row.Receipt.RunID != ""
	if attempted {
		out.Starts++
		cost, usageKnown, overflow := estimateAgentCost(row.Receipt.Usage, runtime.Review.Agents[row.ManagerID])
		row.CostMicros, row.CostKnown, row.CostOverflow = cost, usageKnown, overflow
		known, unknown := costCounts([]InvocationLog{{CostKnown: usageKnown, Receipt: *row.Receipt}})
		frame.knownCost += known
		frame.unknownCost += unknown
		freshPin, pinErr := runtimeDigestWithInvoker(frame.invoker, runtime)
		briefingErr := freshFullBriefingBindings(frame.host, frame.root, binding.acceptedBriefings, project.Report.ModelDigest, out.BriefingDigests, project.Revision)
		sourceErr := fullAuditFreshBinding(frame.host, frame.root, project, frame.invoker, runtime, binding.CheckSource)
		if !sameRuntimeSnapshot(project) || pinErr != nil || freshPin != frame.runtimePin || briefingErr != nil || sourceErr != nil {
			row.Status, row.Error = "incomplete", "snapshot, runtime, or briefing binding changed during audit"
			out.Error = ErrStale.Error()
			markRemainingFullRows(out.Managers, i+1, "not started after stale verification binding")
			return true
		}
		if row.CostOverflow {
			_ = addFullKnownCost(&out.CostMicros, row.CostMicros)
			out.CostMicros = math.MaxInt64
			row.Status, row.Error = "incomplete", "known cost estimate exceeds the supported int64 range"
			out.Error = "full verification exceeded the cumulative cost limit"
			markRemainingFullRows(out.Managers, i+1, "not started after cost estimate overflow")
			return true
		}
	}
	if callErr != nil {
		return frame.handleAuditError(i, row, attempted, callErr)
	}
	return frame.accountSuccessfulAudit(i, row)
}

func (frame *fullVerifyFrame) handleAuditError(i int, row *FullManagerAssessment, attempted bool, callErr error) bool {
	runtime, out := frame.runtime, &frame.out
	if row.Status != "failed" {
		row.Status = "incomplete"
	}
	row.Error = callErr.Error()
	unknownUsage := false
	if attempted {
		if row.CostKnown {
			if addFullKnownCost(&out.CostMicros, row.CostMicros) {
				row.Status, row.Error = "incomplete", "known cumulative cost exceeds int64; total is saturated at MaxInt64"
				out.Error = "full verification cannot represent exact cumulative cost after int64 overflow"
				markRemainingFullRows(out.Managers, i+1, "not started because exact cumulative cost overflowed")
				return true
			}
			if out.CostMicros > runtime.Limits.MaxCostMicros {
				out.Error = "full verification exceeded the cumulative cost limit"
				markRemainingFullRows(out.Managers, i+1, "not started after cumulative cost limit was exceeded")
				return true
			}
		} else if requiresReportedUsage(runtime.Review.Agents[row.ManagerID]) {
			unknownUsage = true
			row.Error += "; invocation usage missing, cumulative cost cannot be asserted"
		}
	}
	if unknownUsage {
		out.Error = "full verification cannot assert cumulative cost after a Manager invocation"
		markRemainingFullRows(out.Managers, i+1, "not started because preceding usage was unavailable")
		return true
	}
	if out.CostMicros > runtime.Limits.MaxCostMicros {
		out.Error = "full verification exceeded the cumulative cost limit"
		markRemainingFullRows(out.Managers, i+1, "not started after cumulative cost limit was exceeded")
		return true
	}
	out.Error = "one or more Manager assessments are missing, failed, or incomplete"
	return false
}

func (frame *fullVerifyFrame) accountSuccessfulAudit(i int, row *FullManagerAssessment) bool {
	runtime, out := frame.runtime, &frame.out
	if !row.CostKnown {
		if requiresReportedUsage(runtime.Review.Agents[row.ManagerID]) {
			row.Status, row.Error = "incomplete", "reviewer usage is missing; bounded cumulative cost cannot be asserted"
			out.Error = "one or more Manager assessments are missing, failed, or incomplete"
			markRemainingFullRows(out.Managers, i+1, "not started because preceding usage was unavailable")
			return true
		}
	} else {
		if addFullKnownCost(&out.CostMicros, row.CostMicros) {
			row.Status, row.Error = "incomplete", "known cumulative cost exceeds int64; total is saturated at MaxInt64"
			out.Error = "full verification cannot represent exact cumulative cost after int64 overflow"
			markRemainingFullRows(out.Managers, i+1, "not started because exact cumulative cost overflowed")
			return true
		}
		if out.CostMicros > runtime.Limits.MaxCostMicros {
			row.Status, row.Error = "incomplete", "cumulative cost limit exceeded"
			out.Error = "full verification exceeded the cumulative cost limit"
			markRemainingFullRows(out.Managers, i+1, "not started after cumulative cost limit was exceeded")
			return true
		}
	}
	if row.Status != "passed" {
		out.Error = "one or more Manager assessments did not pass"
	}
	return false
}

func finalizeFullVerifyFrame(frame *fullVerifyFrame) {
	project, runtime, binding := frame.project, frame.runtime, frame.binding
	out := &frame.out
	out.CostAccounting = costAccountingFromCounts(frame.knownCost, frame.unknownCost)
	bindingStale := false
	if !sameRuntimeSnapshot(project) || freshFullBriefingBindings(frame.host, frame.root, binding.acceptedBriefings, project.Report.ModelDigest, out.BriefingDigests, project.Revision) != nil {
		bindingStale, out.Error = true, ErrStale.Error()
	} else if freshPin, pinErr := runtimeDigestWithInvoker(frame.invoker, runtime); pinErr != nil || freshPin != frame.runtimePin {
		bindingStale, out.Error = true, ErrStale.Error()
	} else if binding.CheckSource && frame.host.Load != nil {
		fresh, sourceErr := frame.host.Load(frame.root, project.Revision)
		if sourceErr != nil || fresh == nil || fresh.Snapshot == nil || fresh.Snapshot.Digest() != project.Snapshot.Digest() {
			bindingStale, out.Error = true, ErrStale.Error()
		}
	}
	out.Status = "passed"
	for _, manager := range out.Managers {
		if manager.Status == "failed" {
			out.Status = "failed"
			break
		}
		if manager.Status != "passed" && out.Status == "passed" {
			out.Status = "incomplete"
		}
	}
	for _, check := range out.Checks {
		if check.Outcome != "passed" {
			if check.Outcome == "failed" {
				out.Status = "failed"
			} else if out.Status == "passed" {
				out.Status = "incomplete"
			}
		}
	}
	if bindingStale && out.Status != "failed" {
		out.Status = "incomplete"
	}
	if frame.checkErr != nil && (strings.Contains(frame.checkErr.Error(), "maxStarts") || strings.Contains(frame.checkErr.Error(), "duration") || errors.Is(frame.ctx.Err(), context.DeadlineExceeded)) && out.Status != "failed" {
		out.Status = "incomplete"
	}
	if out.Status == "passed" && out.Error != "" {
		out.Status = "incomplete"
	}
}
