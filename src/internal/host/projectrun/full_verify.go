package projectrun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// FullVerifyRequest selects an immutable source revision and whether to write
// the resulting report and receipts below .markitect/runs.
type FullVerifyRequest struct {
	Revision string `json:"revision"`
	Write    bool   `json:"write,omitempty"`
}

// FullVerifyBinding binds verification of an already composed immutable
// project, including a Run candidate. ExpectedSnapshot is required and lets
// the caller prevent accidental verification of a different candidate.
type FullVerifyBinding struct {
	ExpectedSnapshot     string `json:"expectedSnapshot"`
	CheckCandidateID     string `json:"checkCandidateId,omitempty"`
	checkCandidateDigest string
	freshReviews         []fullReviewEvidence
	ExpectedBriefings    map[string]string `json:"expectedBriefings,omitempty"`
	Write                bool              `json:"write,omitempty"`
	CheckSource          bool              `json:"checkSource,omitempty"`
	StartedAt            time.Time         `json:"startedAt,omitempty"`
	// PriorStarts includes starts already consumed by preverified checks.
	PriorStarts                 int           `json:"priorStarts,omitempty"`
	PriorCostMicros             int64         `json:"priorCostMicros,omitempty"`
	PriorKnownCostInvocations   int           `json:"priorKnownCostInvocations,omitempty"`
	PriorUnknownCostInvocations int           `json:"priorUnknownCostInvocations,omitempty"`
	PreverifiedChecks           []CheckResult `json:"preverifiedChecks,omitempty"`
	// acceptedBriefings is the accepted history a preview computed in memory;
	// nil reads the persisted store.
	acceptedBriefings *projectbriefing.Store
}

// fullReviewEvidence preserves both the reviewed candidate and the final
// candidate whose current scope was used by the existing freshness gate.
// These records are supplemental audit context, not Full Verify verdicts.
type fullReviewEvidence struct {
	TaskID                string                    `json:"taskId"`
	ManagerID             string                    `json:"managerId"`
	Phase                 string                    `json:"phase"`
	ReviewCandidateID     string                    `json:"reviewCandidateId"`
	ReviewCandidateDigest string                    `json:"reviewCandidateDigest"`
	FinalCandidateID      string                    `json:"finalCandidateId"`
	FinalCandidateDigest  string                    `json:"finalCandidateDigest"`
	ScopeDigest           string                    `json:"scopeDigest"`
	InputDigest           string                    `json:"inputDigest"`
	ReceiptRunID          string                    `json:"receiptRunId"`
	Outcome               string                    `json:"outcome"`
	Findings              []ReviewFinding           `json:"findings"`
	ReviewedFiles         []fullReviewFileReference `json:"reviewedFiles"`
}

type fullReviewFileReference struct {
	Path          string `json:"path"`
	Mode          string `json:"mode"`
	ContentDigest string `json:"contentDigest"`
}

type FullVerifyReport struct {
	APIVersion      string                  `json:"apiVersion"`
	Status          string                  `json:"status"` // passed, failed, or incomplete
	Revision        string                  `json:"revision"`
	CandidateID     string                  `json:"candidateId,omitempty"`
	SnapshotDigest  string                  `json:"snapshotDigest"`
	ProjectDigest   string                  `json:"projectDigest"`
	CoverageDigest  string                  `json:"coverageDigest"`
	ModelDigest     string                  `json:"modelDigest"`
	ReportDigest    string                  `json:"reportDigest"`
	RuntimeDigest   string                  `json:"runtimeDigest"`
	BriefingDigests map[string]string       `json:"briefingDigests"`
	StartedAt       time.Time               `json:"startedAt"`
	FinishedAt      time.Time               `json:"finishedAt"`
	Managers        []FullManagerAssessment `json:"managers"`
	Checks          []CheckResult           `json:"checks"`
	Starts          int                     `json:"starts"`
	CostMicros      int64                   `json:"costMicros"`
	CostAccounting  string                  `json:"costAccounting"`
	Error           string                  `json:"error,omitempty"`
	Digest          string                  `json:"digest"`
	PersistedPath   string                  `json:"persistedPath,omitempty"`
	AcceptanceNote  string                  `json:"acceptanceNote"`
}

type FullManagerAssessment struct {
	ManagerID       string               `json:"managerId"`
	ParentID        string               `json:"parentId,omitempty"`
	Strictness      StrictnessProfile    `json:"strictness"`
	ScopeDigest     string               `json:"scopeDigest"`
	Subjects        []string             `json:"subjects"`
	Status          string               `json:"status"` // passed, failed, or incomplete
	Summary         string               `json:"summary,omitempty"`
	Assessments     []FullAssessment     `json:"assessments"`
	Counterexamples []FullCounterexample `json:"counterexamples"`
	Findings        []string             `json:"findings"`
	InputDigest     string               `json:"inputDigest,omitempty"`
	Receipt         *agentexec.Receipt   `json:"receipt,omitempty"`
	CostMicros      int64                `json:"costMicros,omitempty"`
	CostKnown       bool                 `json:"costKnown"`
	CostOverflow    bool                 `json:"costOverflow,omitempty"`
	Error           string               `json:"error,omitempty"`
}

type FullAssessment struct {
	Subject string `json:"subject"`
	Outcome string `json:"outcome"`
	Detail  string `json:"detail"`
}

type FullCounterexample struct {
	Expected     string   `json:"expected"`
	Observed     string   `json:"observed"`
	EvidenceRefs []string `json:"evidenceRefs"`
}

type fullAuditResponse struct {
	Status          string               `json:"status"`
	Summary         string               `json:"summary"`
	Assessments     []FullAssessment     `json:"assessments"`
	Findings        []string             `json:"findings"`
	Counterexamples []FullCounterexample `json:"counterexamples"`
}

type fullIntegrationObligation struct {
	ChildManager string                   `json:"childManager"`
	Contracts    []projectmodel.Statement `json:"contracts"`
	Artifacts    []projectmodel.Artifact  `json:"artifacts"`
	Checks       []projectmodel.Check     `json:"checks"`
}

type fullChildAssessment struct {
	ManagerID   string             `json:"managerId"`
	Status      string             `json:"status"`
	Summary     string             `json:"summary"`
	ScopeDigest string             `json:"scopeDigest"`
	Subjects    []fullChildSubject `json:"subjects"`
}

type fullChildSubject struct {
	Subject string `json:"subject"`
	Outcome string `json:"outcome"`
}

// FullVerify loads one fixed revision and audits every active Manager against
// the same compiled project snapshot. The selected runtime configuration must
// be the exact runtime file in that snapshot.
func FullVerify(ctx context.Context, host Host, invoker Invoker, root string, request FullVerifyRequest) (FullVerifyReport, error) {
	var empty FullVerifyReport
	if host.Load == nil || invoker == nil || strings.TrimSpace(request.Revision) == "" {
		return empty, fmt.Errorf("full verification requires Host.Load, an invoker, and an explicit revision")
	}
	project, err := host.Load(root, request.Revision)
	if err != nil {
		return empty, err
	}
	if project == nil {
		return empty, fmt.Errorf("host returned an empty project for the selected revision")
	}
	if project.Provisional || !fullVerifyCommitID(project.Revision) {
		return empty, fmt.Errorf("full verification requires a committed immutable revision")
	}
	// Only a writing verification persists accepted history; a preview reads
	// it in memory and audits against that view.
	var briefingView *projectbriefing.Store
	if project.Config.WorkflowMode == "guided" {
		if briefingView, err = captureAcceptedHistory(root, project.Revision, request.Write); err != nil {
			return empty, err
		}
	}
	runtime, err := LoadRuntime(root)
	if err != nil {
		return empty, err
	}
	if !sameRuntimeSnapshot(project) {
		return empty, ErrStale
	}
	return FullVerifyProject(ctx, host, invoker, root, project, runtime, FullVerifyBinding{
		ExpectedSnapshot: project.Snapshot.Digest(), CheckCandidateID: project.Snapshot.Digest(), Write: request.Write, CheckSource: true,
		acceptedBriefings: briefingView,
	})
}

// FullVerifyProject audits an immutable host-composed project. This is the
// reusable seam for final Run candidates; it performs no candidate/model
// mutation and never reopens candidate source from the working tree.
func FullVerifyProject(ctx context.Context, host Host, invoker Invoker, root string, project *projectwork.Project, runtime Runtime, binding FullVerifyBinding) (FullVerifyReport, error) {
	started := time.Now().UTC()
	if !binding.StartedAt.IsZero() {
		started = binding.StartedAt.UTC()
	}
	out := FullVerifyReport{APIVersion: APIVersion, Status: "incomplete", StartedAt: started,
		Managers: []FullManagerAssessment{}, Checks: []CheckResult{}, BriefingDigests: map[string]string{}, Starts: binding.PriorStarts, CostMicros: binding.PriorCostMicros,
		CostAccounting: costAccountingFromCounts(binding.PriorKnownCostInvocations, binding.PriorUnknownCostInvocations),
		AcceptanceNote: "A passed result means every configured Manager audit and declared project check in this bounded contract passed. It does not establish semantic completeness or human acceptance."}
	if project == nil || project.Snapshot == nil || invoker == nil || strings.TrimSpace(root) == "" {
		return out, fmt.Errorf("full verification requires an immutable Project, invoker, and root")
	}
	if project.Provisional || !fullVerifyCommitID(project.Revision) {
		return out, fmt.Errorf("full verification requires a committed immutable revision")
	}
	if binding.ExpectedSnapshot == "" || binding.ExpectedSnapshot != project.Snapshot.Digest() {
		return out, ErrStale
	}
	if err := ValidateRuntime(runtime); err != nil {
		return out, err
	}
	if runtime.Review == nil {
		return out, fmt.Errorf("full verification requires a configured reviewer for every Manager")
	}
	if !sameRuntimeSnapshot(project) {
		return out, ErrStale
	}
	runtimePin, err := runtimeDigestWithInvoker(invoker, runtime)
	if err != nil {
		return out, err
	}
	if binding.CheckCandidateID == "" {
		binding.CheckCandidateID = project.Snapshot.Digest()
	}
	if binding.checkCandidateDigest == "" {
		binding.checkCandidateDigest = project.Snapshot.Digest()
	}
	for _, review := range binding.freshReviews {
		if review.FinalCandidateID != binding.CheckCandidateID || review.FinalCandidateDigest != binding.checkCandidateDigest || review.Outcome != "pass" {
			return out, ErrStale
		}
	}
	out.Revision, out.CandidateID, out.SnapshotDigest, out.ProjectDigest = project.Revision, binding.CheckCandidateID, project.Snapshot.Digest(), project.Digest
	if project.Coverage != nil {
		out.CoverageDigest = project.Coverage.Digest
	}
	out.ModelDigest, out.ReportDigest, out.RuntimeDigest = project.Report.ModelDigest, project.Report.Digest, runtimePin
	deadline := started.Add(time.Duration(runtime.Limits.MaxDuration))
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if hasErrorFinding(project.Report.Findings) {
		out.Status, out.Error = "failed", "compiled project contains error findings"
		return finishFullVerify(root, binding.Write, out, nil)
	}
	if project.Config.CoverageMode != "full" {
		out.Status, out.Error = "incomplete", "full verification requires project CoverageMode: full"
		return finishFullVerify(root, binding.Write, out, nil)
	}
	if project.Coverage == nil {
		out.Status, out.Error = "incomplete", "full verification requires a repository coverage census bound to the project snapshot"
		return finishFullVerify(root, binding.Write, out, nil)
	}
	if !project.Coverage.Accounted || !project.Coverage.Conforming {
		out.Status, out.Error = "failed", "repository coverage census is not conforming"
		return finishFullVerify(root, binding.Write, out, nil)
	}
	if err := requireArtifacts(project.Report); err != nil {
		out.Status, out.Error = "failed", err.Error()
		return finishFullVerify(root, binding.Write, out, nil)
	}
	if err := validateCandidateDocument(project); err != nil {
		out.Status, out.Error = "failed", err.Error()
		return finishFullVerify(root, binding.Write, out, nil)
	}
	managers := append([]projectmodel.Manager(nil), project.Report.Managers...)
	sort.Slice(managers, func(i, j int) bool {
		di, dj := managerDepth(projectManagerMap(project.Report.Managers), managers[i].ID), managerDepth(projectManagerMap(project.Report.Managers), managers[j].ID)
		if di != dj {
			return di > dj
		}
		return managers[i].ID < managers[j].ID
	})
	if len(managers) == 0 {
		out.Status, out.Error = "failed", "project has no active Managers"
		return finishFullVerify(root, binding.Write, out, nil)
	}
	if err := validateStrictnessManagers(runtime.Strictness, managers); err != nil {
		out.Status, out.Error = "failed", err.Error()
		return finishFullVerify(root, binding.Write, out, nil)
	}
	if binding.PriorStarts < 0 || binding.PriorCostMicros < 0 || binding.PriorStarts > runtime.Limits.MaxStarts || binding.PriorCostMicros > runtime.Limits.MaxCostMicros {
		out.Status, out.Error = "incomplete", "cumulative runtime budget already exceeded"
		return finishFullVerify(root, binding.Write, out, nil)
	}
	if len(binding.PreverifiedChecks) > binding.PriorStarts {
		out.Status, out.Error = "incomplete", "preverified checks are not covered by the cumulative start count"
		return finishFullVerify(root, binding.Write, out, nil)
	}
	for _, manager := range managers {
		profile, profileErr := ResolveStrictness(runtime, manager.ID)
		row := FullManagerAssessment{ManagerID: manager.ID, ParentID: manager.Parent, Strictness: profile, Assessments: []FullAssessment{}, Findings: []string{}}
		if profileErr != nil {
			row.Status, row.Error = "incomplete", profileErr.Error()
		}
		out.Managers = append(out.Managers, row)
	}
	briefingContexts := make(map[string]BriefingContext, len(managers))
	briefingBindingInvalid := false
	for _, manager := range managers {
		briefing, briefingErr := managerBriefingIn(root, binding.acceptedBriefings, project.Report.ModelDigest, manager.ID, project.Revision)
		if briefingErr != nil {
			row := findFullManagerAssessment(out.Managers, manager.ID)
			row.Status, row.Error = "incomplete", briefingErr.Error()
			out.Error = "accepted model briefing is missing or stale for one or more Managers"
			briefingBindingInvalid = true
			continue
		}
		briefingContexts[manager.ID] = briefing
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
		return finishFullVerify(root, binding.Write, out, nil)
	}
	if len(binding.PreverifiedChecks) > 0 {
		if len(binding.PreverifiedChecks) > len(project.Report.Checks) {
			out.Status, out.Error = "incomplete", "preverified check binding contains unrelated entries"
			return finishFullVerify(root, binding.Write, out, nil)
		}
		out.Checks = append(out.Checks, binding.PreverifiedChecks...)
	}
	checkErr := error(nil)
	if out.Starts < runtime.Limits.MaxStarts && out.CostMicros <= runtime.Limits.MaxCostMicros {
		var checks []CheckResult
		checks, checkErr = fullRunChecks(ctx, root, project, runtime, &out, binding.CheckCandidateID)
		out.Checks = append(out.Checks, checks...)
	}
	if checkErr != nil {
		out.Error = checkErr.Error()
	}
	knownCostInvocations, unknownCostInvocations := binding.PriorKnownCostInvocations, binding.PriorUnknownCostInvocations
	for i := range out.Managers {
		row := &out.Managers[i]
		if err := ctx.Err(); err != nil {
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
		if _, ok := briefingContexts[row.ManagerID]; !ok {
			continue
		}
		if err := freshFullBriefingBindings(root, binding.acceptedBriefings, project.Report.ModelDigest, out.BriefingDigests, project.Revision); err != nil {
			row.Status, row.Error = "incomplete", err.Error()
			out.Error = ErrStale.Error()
			markRemainingFullRows(out.Managers, i+1, "not started after stale briefing binding")
			break
		}
		children := fullChildAssessments(out.Managers, project.Report, row.ManagerID)
		integrationReviews := fullAuditIntegrationReviewsForManager(project.Report, row.ManagerID, binding.CheckCandidateID, binding.checkCandidateDigest, binding.freshReviews)
		assessment, callErr := fullAuditManager(ctx, host, invoker, root, project, runtime, row.ManagerID, row.Strictness, briefingContexts[row.ManagerID], children, relevantManagerChecks(project.Report, row.ManagerID, out.Checks), integrationReviews, binding.CheckSource, "full-verify-"+binding.ExpectedSnapshot)
		*row = assessment
		attempted := row.Receipt != nil && row.Receipt.RunID != ""
		if attempted {
			out.Starts++
		} // A failed or interrupted external attempt still consumes a start.
		if attempted {
			cost, usageKnown, overflow := estimateAgentCost(row.Receipt.Usage, runtime.Review.Agents[row.ManagerID])
			row.CostMicros, row.CostKnown, row.CostOverflow = cost, usageKnown, overflow
			known, unknown := costCounts([]InvocationLog{{CostKnown: usageKnown, Receipt: *row.Receipt}})
			knownCostInvocations += known
			unknownCostInvocations += unknown
			freshPin, pinErr := runtimeDigestWithInvoker(invoker, runtime)
			briefingErr := freshFullBriefingBindings(root, binding.acceptedBriefings, project.Report.ModelDigest, out.BriefingDigests, project.Revision)
			sourceErr := fullAuditFreshBinding(host, root, project, invoker, runtime, binding.CheckSource)
			if !sameRuntimeSnapshot(project) || pinErr != nil || freshPin != runtimePin || briefingErr != nil || sourceErr != nil {
				row.Status, row.Error = "incomplete", "snapshot, runtime, or briefing binding changed during audit"
				out.Error = ErrStale.Error()
				markRemainingFullRows(out.Managers, i+1, "not started after stale verification binding")
				break
			}
			if row.CostOverflow {
				_ = addFullKnownCost(&out.CostMicros, row.CostMicros)
				out.CostMicros = math.MaxInt64
				row.Status, row.Error = "incomplete", "known cost estimate exceeds the supported int64 range"
				out.Error = "full verification exceeded the cumulative cost limit"
				markRemainingFullRows(out.Managers, i+1, "not started after cost estimate overflow")
				break
			}
		}
		if callErr != nil {
			if row.Status != "failed" {
				row.Status = "incomplete"
			}
			row.Error = callErr.Error()
			unknownUsage := false
			if attempted {
				if row.CostKnown {
					cost := row.CostMicros
					if addFullKnownCost(&out.CostMicros, cost) {
						row.Status, row.Error = "incomplete", "known cumulative cost exceeds int64; total is saturated at MaxInt64"
						out.Error = "full verification cannot represent exact cumulative cost after int64 overflow"
						markRemainingFullRows(out.Managers, i+1, "not started because exact cumulative cost overflowed")
						break
					}
					if out.CostMicros > runtime.Limits.MaxCostMicros {
						out.Error = "full verification exceeded the cumulative cost limit"
						markRemainingFullRows(out.Managers, i+1, "not started after cumulative cost limit was exceeded")
						break
					}
				} else {
					if runtime.Review.Agents[row.ManagerID].requiresReportedUsage() {
						unknownUsage = true
						row.Error += "; invocation usage missing, cumulative cost cannot be asserted"
					}
				}
			}
			if unknownUsage {
				out.Error = "full verification cannot assert cumulative cost after a Manager invocation"
				markRemainingFullRows(out.Managers, i+1, "not started because preceding usage was unavailable")
				break
			}
			if out.CostMicros > runtime.Limits.MaxCostMicros {
				out.Error = "full verification exceeded the cumulative cost limit"
				markRemainingFullRows(out.Managers, i+1, "not started after cumulative cost limit was exceeded")
				break
			}
			out.Error = "one or more Manager assessments are missing, failed, or incomplete"
			continue
		}
		if !row.CostKnown {
			if runtime.Review.Agents[row.ManagerID].requiresReportedUsage() {
				row.Status, row.Error = "incomplete", "reviewer usage is missing; bounded cumulative cost cannot be asserted"
				out.Error = "one or more Manager assessments are missing, failed, or incomplete"
				markRemainingFullRows(out.Managers, i+1, "not started because preceding usage was unavailable")
				break
			}
		} else {
			cost := row.CostMicros
			if addFullKnownCost(&out.CostMicros, cost) {
				row.Status, row.Error = "incomplete", "known cumulative cost exceeds int64; total is saturated at MaxInt64"
				out.Error = "full verification cannot represent exact cumulative cost after int64 overflow"
				markRemainingFullRows(out.Managers, i+1, "not started because exact cumulative cost overflowed")
				break
			}
			if out.CostMicros > runtime.Limits.MaxCostMicros {
				row.Status, row.Error = "incomplete", "cumulative cost limit exceeded"
				out.Error = "full verification exceeded the cumulative cost limit"
				markRemainingFullRows(out.Managers, i+1, "not started after cumulative cost limit was exceeded")
				break
			}
		}
		if row.Status != "passed" {
			out.Error = "one or more Manager assessments did not pass"
			continue
		}
	}
	out.CostAccounting = costAccountingFromCounts(knownCostInvocations, unknownCostInvocations)
	bindingStale := false
	if !sameRuntimeSnapshot(project) || freshFullBriefingBindings(root, binding.acceptedBriefings, project.Report.ModelDigest, out.BriefingDigests, project.Revision) != nil {
		bindingStale, out.Error = true, ErrStale.Error()
	} else if freshPin, pinErr := runtimeDigestWithInvoker(invoker, runtime); pinErr != nil || freshPin != runtimePin {
		bindingStale, out.Error = true, ErrStale.Error()
	} else if binding.CheckSource && host.Load != nil {
		fresh, sourceErr := host.Load(root, project.Revision)
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
	if checkErr != nil && (strings.Contains(checkErr.Error(), "maxStarts") || strings.Contains(checkErr.Error(), "duration") || errors.Is(ctx.Err(), context.DeadlineExceeded)) && out.Status != "failed" {
		out.Status = "incomplete"
	}
	if out.Status == "passed" && out.Error != "" {
		out.Status = "incomplete"
	}
	return finishFullVerify(root, binding.Write, out, nil)
}

// addFullKnownCost always records a known nonnegative cost. If the exact sum
// cannot fit in the report's int64 field, it saturates and reports overflow.
func addFullKnownCost(total *int64, cost int64) bool {
	if cost < 0 || *total < 0 {
		return true
	}
	if cost > math.MaxInt64-*total {
		*total = math.MaxInt64
		return true
	}
	*total += cost
	return false
}

func fullVerifyCommitID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded)*2 == len(value)
}

func fullRunChecks(ctx context.Context, root string, project *projectwork.Project, runtime Runtime, report *FullVerifyReport, candidateID string) ([]CheckResult, error) {
	selected := map[string]bool{}
	for _, manager := range project.Report.Managers {
		selected[manager.ID] = true
	}
	checks, findings := planChecks(project.Report, selected)
	if len(findings) != 0 {
		return nil, fmt.Errorf("declared check contract is invalid: %s", strings.Join(findings, "; "))
	}
	checks, pinFindings, err := bindCheckExecutables(checks, runtime)
	if err != nil {
		return nil, err
	}
	if len(pinFindings) != 0 {
		return nil, fmt.Errorf("declared check executable could not be pinned: %s", strings.Join(pinFindings, "; "))
	}
	if len(checks) == 0 {
		return nil, fmt.Errorf("project declares no independent checks; full verification cannot pass")
	}
	if candidateID == "" {
		candidateID = project.Snapshot.Digest()
	}
	covered := map[string]CheckResult{}
	declared := map[string]bool{}
	for _, check := range checks {
		declared[check.ID] = true
	}
	for _, check := range report.Checks {
		if !declared[check.ID] {
			return nil, fmt.Errorf("preverified check %s is not declared by this project", check.ID)
		}
		if _, exists := covered[check.ID]; exists {
			return nil, fmt.Errorf("preverified check %s is duplicated", check.ID)
		}
		covered[check.ID] = check
	}
	for _, check := range checks {
		if prior, ok := covered[check.ID]; ok {
			if prior.Outcome != "passed" || !equalStrings(prior.Command, check.Command) || prior.ExecutablePath != check.ExecutablePath || prior.ExecutableDigest != check.ExecutableDigest || prior.CandidateID != candidateID {
				return nil, fmt.Errorf("preverified check %s is stale or failed", check.ID)
			}
			if err := checkExecutableUnchanged(check); err != nil {
				return nil, err
			}
		}
	}
	needed := 0
	for _, check := range checks {
		if _, ok := covered[check.ID]; !ok {
			needed++
		}
	}
	if report.Starts+needed > runtime.Limits.MaxStarts {
		return nil, fmt.Errorf("declared checks would exceed cumulative maxStarts")
	}
	if report.CostMicros > runtime.Limits.MaxCostMicros {
		return nil, fmt.Errorf("cumulative maxCostMicros exceeded before declared checks")
	}
	tmp, err := os.MkdirTemp("", "markitect-full-verify-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	if err := materializeCandidate(tmp, project.Snapshot, candidateData{Files: map[string]File{}}); err != nil {
		return nil, err
	}
	var results []CheckResult
	var failures []error
	for _, check := range checks {
		if _, ok := covered[check.ID]; ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			return results, err
		}
		config, ok := runtime.Agents[check.Owner]
		if !ok {
			return results, fmt.Errorf("check %s owner %s has no runtime environment policy", check.ID, check.Owner)
		}
		result := runCheck(ctx, tmp, check, config, runtime.Limits.MaxDuration, nil)
		result.CandidateID = candidateID
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || result.Error == "check timed out" {
			result.Outcome = "incomplete"
		}
		results = append(results, result)
		report.Starts++
		if result.Outcome != "passed" {
			failures = append(failures, fmt.Errorf("required declared check %s did not pass: %s", check.ID, result.Error))
		}
	}
	return results, errors.Join(failures...)
}

func fullAuditManager(ctx context.Context, host Host, invoker Invoker, root string, project *projectwork.Project, runtime Runtime, managerID string, strictness StrictnessProfile, briefing BriefingContext, childAssessments []fullChildAssessment, checkResults []CheckResult, integrationReviews []fullReviewEvidence, checkSource bool, ownerRunID string) (FullManagerAssessment, error) {
	row := FullManagerAssessment{ManagerID: managerID, Status: "incomplete", Assessments: []FullAssessment{}, Findings: []string{}}
	manager, ok := runtime.Review.Agents[managerID]
	if !ok {
		return row, fmt.Errorf("no configured reviewer for Manager %s", managerID)
	}
	config, err := manager.AgentConfig()
	if err != nil {
		return row, err
	}
	if manager.Transport != TransportCodexAppServer {
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return row, context.DeadlineExceeded
			}
			if remaining < config.Timeout {
				config.Timeout = remaining
			}
		}
	}
	modelContext, err := projectmodel.Context(project.Report, managerID)
	if err != nil {
		return row, err
	}
	supportingStatements, supportingChecks := fullAuditArtifactSupport(project.Report, managerID)
	files, err := fullManagerFiles(project, managerID)
	if err != nil {
		return row, err
	}
	children := fullIntegrationObligations(project.Report, managerID)
	subjects := fullAuditSubjects(modelContext, children)
	for _, item := range briefing.Briefings {
		subjects = append(subjects, "briefing:"+item.ID)
	}
	for _, evidence := range strictness.Evidence {
		subjects = append(subjects, "evidence:"+evidence)
	}
	subjects = uniqueSorted(subjects)
	if len(subjects) == 0 {
		subjects = []string{"manager:" + managerID}
	}
	row.Subjects = subjects
	row.ParentID = modelContext.Manager.Parent
	fileRefs := make([]reviewFileRef, 0, len(files))
	for _, file := range files {
		fileRefs = append(fileRefs, reviewFileRef{Path: file.Path, Mode: file.Mode, Digest: file.Digest, Grounding: []string{"file-bytes", "file-mode"}})
	}
	contextPayload := struct {
		Kind                    string                      `json:"kind"`
		SnapshotDigest          string                      `json:"snapshotDigest"`
		ProjectDigest           string                      `json:"projectDigest"`
		ModelDigest             string                      `json:"modelDigest"`
		Manager                 projectmodel.ManagerContext `json:"manager"`
		SupportingStatements    []projectmodel.Statement    `json:"supportingStatements"`
		SupportingChecks        []projectmodel.Check        `json:"supportingChecks"`
		Briefing                BriefingContext             `json:"briefing"`
		IntegrationObligations  []fullIntegrationObligation `json:"integrationObligations"`
		ChildAssessments        []fullChildAssessment       `json:"childAssessments"`
		CheckResults            []CheckResult               `json:"checkResults"`
		FreshIntegrationReviews []fullReviewEvidence        `json:"freshIntegrationReviews"`
		Files                   []reviewFileRef             `json:"files"`
		Subjects                []string                    `json:"requiredSubjects"`
		Strictness              StrictnessProfile           `json:"strictness"`
		ResponseSchema          json.RawMessage             `json:"responseSchema"`
	}{"projectrun-full-verify/v1", project.Snapshot.Digest(), project.Digest, project.Report.ModelDigest, modelContext, supportingStatements, supportingChecks, briefing, children, childAssessments, checkResults, integrationReviews, fileRefs, subjects, strictness, fullVerifyResponseSchema(subjects, strictness.Counterexamples)}
	contextJSON, err := json.Marshal(contextPayload)
	if err != nil {
		return row, err
	}
	scope := []string{managerID}
	for _, subject := range subjects {
		scope = append(scope, subject)
	}
	request := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: project.Revision, ModelDigest: project.Report.ModelDigest,
		ModulePin: project.Report.Digest, ProjectionID: project.Report.Digest, ScopeIDs: uniqueSorted(scope), PolicyIDs: []string{}, Context: contextJSON, Artifacts: files}
	if err := canonicalizeReviewContext(&request); err != nil {
		return row, err
	}
	contextJSON = request.Context
	row.ScopeDigest, err = digest(struct {
		Context        json.RawMessage      `json:"context"`
		Artifacts      []agentexec.Artifact `json:"artifacts"`
		BriefingDigest string               `json:"briefingDigest"`
	}{contextJSON, files, briefing.Digest})
	if err != nil {
		return row, err
	}
	row.InputDigest, err = digest(request)
	if err != nil {
		return row, err
	}
	var result agentexec.RunResult
	var invokeErr error
	if manager.Transport == TransportCodexAppServer {
		taskID, err := readOnlyRepositoryTaskID(request)
		if err != nil {
			return row, fmt.Errorf("allocate unique full-verification workspace task: %w", err)
		}
		taskID = "full-verify-assessment-" + strings.TrimPrefix(taskID, "readonly-")
		result, invokeErr = invokeProjectAgent(ctx, host, invoker, root, project, manager, ownerRunID, taskID, []string{}, nil, runtime.Limits, config, request)
	} else {
		result, invokeErr = invokeAgent(ctx, invoker, config, request, agentexec.RunOptions{PrivateLogDirectory: filepath.Join(root, ".markitect", "runs", "private")})
	}
	row.Receipt = &result.Receipt
	if invokeErr != nil {
		return row, invokeErr
	}
	if result.Receipt.InputDigest != row.InputDigest || result.Response.InputDigest != row.InputDigest || result.Response.Role != agentexec.RoleExecutor {
		return row, fmt.Errorf("Manager %s response is not bound to its exact audit request (expected %s, receipt %s, response %s)", managerID, row.InputDigest, result.Receipt.InputDigest, result.Response.InputDigest)
	}
	if result.Receipt.Outcome != agentexec.OutcomeProposed || result.Response.Outcome != agentexec.OutcomeProposed {
		return row, fmt.Errorf("Manager %s did not return a proposed typed audit report", managerID)
	}
	if len(result.Response.CandidateFiles) != 0 || len(result.Response.CandidateJSON) != 0 || len(result.Response.VerifierObservations) != 0 {
		return row, fmt.Errorf("Manager %s returned output outside the read-only audit contract", managerID)
	}
	if len(result.Response.Uncertainty) != 0 {
		return row, fmt.Errorf("Manager %s reported unresolved uncertainty outside the typed audit report", managerID)
	}
	parsed, err := decodeFullAuditResponse(result.Response.ReportJSON)
	if err != nil {
		return row, err
	}
	row.Summary, row.Assessments, row.Findings = parsed.Summary, parsed.Assessments, parsed.Findings
	row.Counterexamples = parsed.Counterexamples
	if err := validateFullAssessments(parsed, subjects, strictness.Counterexamples, subjects, reviewScopePaths(files)); err != nil {
		for _, item := range parsed.Assessments {
			if item.Outcome == "fail" {
				row.Status = "failed"
				break
			}
		}
		return row, err
	}
	if parsed.Status != "pass" {
		row.Status = "incomplete"
		if parsed.Status == "fail" {
			row.Status = "failed"
		}
		return row, fmt.Errorf("Manager %s audit status is %s", managerID, parsed.Status)
	}
	row.Status = "passed"
	if err := fullAuditFreshBinding(host, root, project, invoker, runtime, checkSource); err != nil {
		return row, err
	}
	return row, nil
}

func freshFullBriefingBindings(root string, view *projectbriefing.Store, modelDigest string, expected map[string]string, revision ...string) error {
	for managerID, digest := range expected {
		briefing, err := managerBriefingIn(root, view, modelDigest, managerID, revision...)
		if err != nil {
			return err
		}
		if briefing.Digest != digest {
			return ErrStale
		}
	}
	return nil
}

func findFullManagerAssessment(rows []FullManagerAssessment, id string) *FullManagerAssessment {
	for i := range rows {
		if rows[i].ManagerID == id {
			return &rows[i]
		}
	}
	return nil
}

func fullAuditFreshBinding(host Host, root string, project *projectwork.Project, invoker Invoker, runtime Runtime, checkSource bool) error {
	if !sameRuntimeSnapshot(project) {
		return ErrStale
	}
	if checkSource && host.Load != nil && project.Revision != "" {
		fresh, err := host.Load(root, project.Revision)
		if err != nil {
			return err
		}
		if fresh.Snapshot == nil || fresh.Snapshot.Digest() != project.Snapshot.Digest() {
			return ErrStale
		}
	}
	_, err := runtimeDigestWithInvoker(invoker, runtime)
	return err
}

func sameRuntimeSnapshot(project *projectwork.Project) bool {
	if project == nil || project.Snapshot == nil {
		return false
	}
	data, ok := project.Snapshot.Files[RuntimePath]
	if !ok {
		return false
	}
	current, err := os.ReadFile(filepath.Join(project.Root, filepath.FromSlash(RuntimePath)))
	if err != nil {
		return false
	}
	a, b := sha256.Sum256(data), sha256.Sum256(current)
	return a == b
}

func fullManagerFiles(project *projectwork.Project, managerID string) ([]agentexec.Artifact, error) {
	selected := map[string]bool{}
	for _, file := range project.Report.Files {
		if file.Owner == managerID && file.Exists {
			selected[file.Path] = true
		}
	}
	// Parent integration review receives direct child implementation bytes in
	// addition to its own files, constrained to artifacts named by the model.
	for _, obligation := range fullIntegrationObligations(project.Report, managerID) {
		for _, artifact := range obligation.Artifacts {
			for _, path := range artifact.Paths {
				if _, exists := project.Snapshot.Files[path]; exists {
					selected[path] = true
				}
			}
		}
	}
	paths := make([]string, 0, len(selected))
	for path := range selected {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	files := make([]agentexec.Artifact, 0, len(paths))
	for _, path := range paths {
		if !projectPathAllowed(project.Config, path) {
			return nil, fmt.Errorf("Manager audit scope includes excluded path %s", path)
		}
		content, ok := project.Snapshot.Files[path]
		if !ok {
			return nil, fmt.Errorf("Manager audit file %s disappeared from snapshot", path)
		}
		mode := protocolMode(project.Snapshot.Modes[path])
		if mode == "" {
			return nil, fmt.Errorf("Manager audit file %s has no bound mode", path)
		}
		files = append(files, agentexec.Artifact{Path: path, Mode: mode, Digest: rawContentDigest(content), Content: append([]byte(nil), content...)})
	}
	return files, nil
}

func fullIntegrationObligations(report projectmodel.Report, parentID string) []fullIntegrationObligation {
	var out []fullIntegrationObligation
	for _, child := range report.Managers {
		if child.Parent != parentID {
			continue
		}
		entry := fullIntegrationObligation{ChildManager: child.ID, Contracts: []projectmodel.Statement{}, Artifacts: []projectmodel.Artifact{}}
		for _, statement := range report.Statements {
			if statement.Owner == child.ID && statement.Public {
				entry.Contracts = append(entry.Contracts, statement)
			}
		}
		for _, artifact := range report.Artifacts {
			if artifact.Owner == child.ID && artifact.Required {
				entry.Artifacts = append(entry.Artifacts, artifact)
			}
		}
		entry.Checks = []projectmodel.Check{}
		for _, check := range report.Checks {
			if check.Owner == child.ID {
				entry.Checks = append(entry.Checks, check)
			}
		}
		sort.Slice(entry.Contracts, func(i, j int) bool { return entry.Contracts[i].ID < entry.Contracts[j].ID })
		sort.Slice(entry.Artifacts, func(i, j int) bool { return entry.Artifacts[i].ID < entry.Artifacts[j].ID })
		sort.Slice(entry.Checks, func(i, j int) bool { return entry.Checks[i].ID < entry.Checks[j].ID })
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChildManager < out[j].ChildManager })
	return out
}

func fullAuditSubjects(mc projectmodel.ManagerContext, children []fullIntegrationObligation) []string {
	var out []string
	for _, statement := range mc.Statements {
		out = append(out, "statement:"+statement.ID)
	}
	for _, artifact := range mc.Artifacts {
		out = append(out, "artifact:"+artifact.ID)
	}
	for _, check := range mc.Checks {
		out = append(out, "check:"+check.ID)
	}
	for _, child := range children {
		out = append(out, "integration:manager:"+child.ChildManager)
		for _, statement := range child.Contracts {
			out = append(out, "integration:statement:"+statement.ID)
		}
		for _, artifact := range child.Artifacts {
			out = append(out, "integration:artifact:"+artifact.ID)
		}
		for _, check := range child.Checks {
			out = append(out, "integration:check:"+check.ID)
		}
	}
	return uniqueSorted(out)
}

func fullChildAssessments(rows []FullManagerAssessment, report projectmodel.Report, parentID string) []fullChildAssessment {
	var out []fullChildAssessment
	for _, manager := range report.Managers {
		if manager.Parent != parentID {
			continue
		}
		row := findFullManagerAssessment(rows, manager.ID)
		if row == nil {
			continue
		}
		child := fullChildAssessment{ManagerID: row.ManagerID, Status: row.Status, Summary: row.Summary, ScopeDigest: row.ScopeDigest, Subjects: []fullChildSubject{}}
		for _, assessment := range row.Assessments {
			child.Subjects = append(child.Subjects, fullChildSubject{Subject: assessment.Subject, Outcome: assessment.Outcome})
		}
		sort.Slice(child.Subjects, func(i, j int) bool { return child.Subjects[i].Subject < child.Subjects[j].Subject })
		out = append(out, child)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ManagerID < out[j].ManagerID })
	return out
}

func fullAuditIntegrationReviewsForManager(report projectmodel.Report, managerID, finalCandidateID, finalCandidateDigest string, reviews []fullReviewEvidence) []fullReviewEvidence {
	var integrationManagerID string
	for _, manager := range report.Managers {
		if manager.ID == managerID {
			integrationManagerID = manager.Parent
			if integrationManagerID == "" {
				integrationManagerID = manager.ID
			}
			break
		}
	}
	if integrationManagerID == "" {
		return []fullReviewEvidence{}
	}
	for _, review := range reviews {
		if review.ManagerID == integrationManagerID && review.Phase == "integrate" && review.Outcome == "pass" &&
			review.FinalCandidateID == finalCandidateID && review.FinalCandidateDigest == finalCandidateDigest {
			return []fullReviewEvidence{review}
		}
	}
	return []fullReviewEvidence{}
}

func relevantManagerChecks(report projectmodel.Report, managerID string, results []CheckResult) []CheckResult {
	visibleManagers := map[string]bool{managerID: true}
	for _, manager := range report.Managers {
		if manager.Parent == managerID {
			visibleManagers[manager.ID] = true
		}
	}
	checkIDs := map[string]bool{}
	definedChecks := map[string]bool{}
	for _, check := range report.Checks {
		definedChecks[check.ID] = true
		if visibleManagers[check.Owner] {
			checkIDs[check.ID] = true
		}
	}
	// An owned artifact can explicitly depend on a check owned by another
	// Manager. Include its recorded result as supporting evidence without
	// making that check an additional audit subject for this Manager.
	for _, artifact := range report.Artifacts {
		if artifact.Owner == managerID {
			for _, checkID := range artifact.Checks {
				if definedChecks[checkID] {
					checkIDs[checkID] = true
				}
			}
		}
	}
	var out []CheckResult
	for _, result := range results {
		if checkIDs[result.ID] {
			result.Stdout = boundedFullCheckOutput(result.Stdout)
			result.Stderr = boundedFullCheckOutput(result.Stderr)
			out = append(out, result)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// fullAuditArtifactSupport exposes only the same-snapshot definitions that an
// owned artifact explicitly references. These are audit context, not Manager
// obligations: callers keep them separate from ManagerContext and required
// audit subjects. Private Statement definitions and unrelated definitions are
// intentionally excluded.
func fullAuditArtifactSupport(report projectmodel.Report, managerID string) ([]projectmodel.Statement, []projectmodel.Check) {
	statementsByID := make(map[string]projectmodel.Statement, len(report.Statements))
	checksByID := make(map[string]projectmodel.Check, len(report.Checks))
	for _, statement := range report.Statements {
		statementsByID[statement.ID] = statement
	}
	for _, check := range report.Checks {
		checksByID[check.ID] = check
	}

	statementIDs := map[string]bool{}
	checkIDs := map[string]bool{}
	for _, artifact := range report.Artifacts {
		if artifact.Owner != managerID {
			continue
		}
		for _, statementID := range artifact.Realizes {
			if statement, ok := statementsByID[statementID]; ok && statement.Public && statement.Owner != managerID {
				statementIDs[statementID] = true
			}
		}
		for _, checkID := range artifact.Checks {
			if _, ok := checksByID[checkID]; ok {
				checkIDs[checkID] = true
			}
		}
	}

	statements := make([]projectmodel.Statement, 0, len(statementIDs))
	for statementID := range statementIDs {
		statement := statementsByID[statementID]
		statement.Uses = visibleFullStatementRelations(statement.Uses, statementsByID)
		statement.Requires = visibleFullStatementRelations(statement.Requires, statementsByID)
		statements = append(statements, statement)
	}
	sort.Slice(statements, func(i, j int) bool { return statements[i].ID < statements[j].ID })

	checks := make([]projectmodel.Check, 0, len(checkIDs))
	for checkID := range checkIDs {
		checks = append(checks, checksByID[checkID])
	}
	sort.Slice(checks, func(i, j int) bool { return checks[i].ID < checks[j].ID })
	return statements, checks
}

func visibleFullStatementRelations(ids []string, statements map[string]projectmodel.Statement) []string {
	visible := make([]string, 0, len(ids))
	for _, id := range ids {
		if statement, ok := statements[id]; ok && statement.Public {
			visible = append(visible, id)
		}
	}
	return uniqueSorted(visible)
}

func boundedFullCheckOutput(value string) string {
	const max = 16 << 10
	if len(value) <= max {
		return value
	}
	return value[:max] + "\n[truncated for Manager context]"
}

func fullVerifyResponseSchema(subjects []string, counterexamples int) json.RawMessage {
	encodedSubjects, err := json.Marshal(subjects)
	if err != nil {
		return nil
	}
	return json.RawMessage(fmt.Sprintf(`{"type":"object","additionalProperties":false,"required":["status","summary","assessments","findings","counterexamples"],"properties":{"status":{"type":"string","enum":["pass","fail","incomplete"],"description":"Semantic result for this bounded audit. Pass only when every required subject is supported by evidence; use fail for contradictory evidence and incomplete when relevant evidence is missing."},"summary":{"type":"string","minLength":1,"maxLength":4096},"assessments":{"type":"array","minItems":%d,"maxItems":%d,"items":{"type":"object","additionalProperties":false,"required":["subject","outcome","detail"],"properties":{"subject":{"type":"string","minLength":1,"maxLength":1024,"enum":%s,"description":"Copy one value from request.context.requiredSubjects exactly; assess every required subject once and add no others."},"outcome":{"type":"string","enum":["pass","fail","incomplete"],"description":"Evidence result for this required subject; use incomplete when relevant evidence is unavailable."},"detail":{"type":"string","minLength":1,"maxLength":2048,"description":"Concise evidence and reasoning for this subject, grounded in the supplied fixed-snapshot context."}}}},"findings":{"type":"array","items":{"type":"string","minLength":1,"maxLength":2048}},"counterexamples":{"type":"array","minItems":%d,"items":{"type":"object","additionalProperties":false,"required":["expected","observed","evidenceRefs"],"properties":{"expected":{"type":"string","minLength":1,"maxLength":2048},"observed":{"type":"string","minLength":1,"maxLength":2048},"evidenceRefs":{"type":"array","minItems":1,"items":{"type":"string","minLength":1,"maxLength":1024}}}}}}}`, len(subjects), len(subjects), string(encodedSubjects), counterexamples))
}

func decodeFullAuditResponse(raw json.RawMessage) (fullAuditResponse, error) {
	var out fullAuditResponse
	if len(raw) == 0 {
		return out, fmt.Errorf("Manager omitted the typed full audit report")
	}
	if err := validateExactObjectKeys(raw, map[string]bool{"status": true, "summary": true, "assessments": true, "findings": true, "counterexamples": true}); err != nil {
		return out, err
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return out, err
	}
	for _, key := range []string{"assessments", "findings", "counterexamples"} {
		if len(wrapper[key]) == 0 || strings.TrimSpace(string(wrapper[key]))[0] != '[' {
			return out, fmt.Errorf("Manager audit %s must be an array", key)
		}
	}
	if err := validateArrayObjectKeys(wrapper["assessments"], map[string]bool{"subject": true, "outcome": true, "detail": true}); err != nil {
		return out, err
	}
	if err := validateArrayObjectKeys(wrapper["counterexamples"], map[string]bool{"expected": true, "observed": true, "evidenceRefs": true}); err != nil {
		return out, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return out, err
	}
	if (out.Status != "pass" && out.Status != "fail" && out.Status != "incomplete") || strings.TrimSpace(out.Summary) == "" || len(out.Summary) > 4096 {
		return out, fmt.Errorf("Manager audit report has invalid status or summary")
	}
	return out, nil
}

func validateFullAssessments(report fullAuditResponse, required []string, minimumCounterexamples int, allowedSubjects, allowedPaths []string) error {
	wanted := map[string]bool{}
	for _, subject := range required {
		wanted[subject] = true
	}
	seen := map[string]bool{}
	for _, a := range report.Assessments {
		if !wanted[a.Subject] || seen[a.Subject] {
			return fmt.Errorf("Manager audit contains unrelated or duplicate subject %q", a.Subject)
		}
		if a.Outcome != "pass" && a.Outcome != "fail" && a.Outcome != "incomplete" {
			return fmt.Errorf("Manager audit subject %q has invalid outcome", a.Subject)
		}
		if strings.TrimSpace(a.Detail) == "" || len(a.Detail) > 2048 {
			return fmt.Errorf("Manager audit subject %q omitted bounded detail", a.Subject)
		}
		seen[a.Subject] = true
	}
	for _, subject := range required {
		if !seen[subject] {
			return fmt.Errorf("Manager audit omitted required subject %q", subject)
		}
	}
	for _, a := range report.Assessments {
		if a.Outcome != "pass" {
			return fmt.Errorf("Manager audit subject %q is %s", a.Subject, a.Outcome)
		}
	}
	if len(report.Findings) > 0 {
		return fmt.Errorf("Manager audit reported findings")
	}
	for _, finding := range report.Findings {
		if strings.TrimSpace(finding) == "" || len(finding) > 2048 {
			return fmt.Errorf("Manager audit finding is empty or oversized")
		}
	}
	if len(report.Counterexamples) < minimumCounterexamples {
		return fmt.Errorf("Manager audit supplied %d counterexamples; strictness requires %d", len(report.Counterexamples), minimumCounterexamples)
	}
	allowedRefs := map[string]bool{}
	for _, subject := range allowedSubjects {
		allowedRefs[subject] = true
	}
	for _, path := range allowedPaths {
		allowedRefs["file:"+path] = true
	}
	seenCounterexamples := map[string]bool{}
	for _, counterexample := range report.Counterexamples {
		if strings.TrimSpace(counterexample.Expected) == "" || strings.TrimSpace(counterexample.Observed) == "" || len(counterexample.Expected) > 2048 || len(counterexample.Observed) > 2048 || len(counterexample.EvidenceRefs) == 0 {
			return fmt.Errorf("Manager audit counterexample lacks bounded expected/observed evidence")
		}
		refs := uniqueSorted(counterexample.EvidenceRefs)
		if len(refs) != len(counterexample.EvidenceRefs) {
			return fmt.Errorf("Manager audit counterexample repeats evidence references")
		}
		for _, ref := range refs {
			if strings.TrimSpace(ref) == "" || len(ref) > 1024 {
				return fmt.Errorf("Manager audit counterexample evidence reference is empty or oversized")
			}
			if !allowedRefs[ref] {
				return fmt.Errorf("Manager audit counterexample references out-of-scope evidence %q", ref)
			}
		}
		key, _ := digest(counterexample)
		if seenCounterexamples[key] {
			return fmt.Errorf("Manager audit duplicated a counterexample")
		}
		seenCounterexamples[key] = true
	}
	return nil
}

func markRemainingFullRows(rows []FullManagerAssessment, start int, reason string) {
	for i := start; i < len(rows); i++ {
		rows[i].Status, rows[i].Error = "incomplete", reason
	}
}

func finishFullVerify(root string, write bool, report FullVerifyReport, cause error) (FullVerifyReport, error) {
	report.FinishedAt = time.Now().UTC()
	digestValue, err := FullVerifyReportDigest(report)
	if err != nil {
		return report, errors.Join(cause, err)
	}
	report.Digest = digestValue
	if write {
		path := filepath.Join(root, ".markitect", "runs", "full-verifications", report.Digest+".json")
		store, err := newRunStore(root)
		if err != nil {
			return report, errors.Join(cause, err)
		}
		if err := ensureDirectory(filepath.Dir(store.base)); err != nil {
			return report, errors.Join(cause, err)
		}
		if err := ensureDirectory(store.base); err != nil {
			return report, errors.Join(cause, err)
		}
		if err := ensureDirectory(filepath.Dir(path)); err != nil {
			return report, errors.Join(cause, err)
		}
		report.PersistedPath = filepath.ToSlash(path)
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return report, errors.Join(cause, err)
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return report, errors.Join(cause, err)
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			_ = file.Close()
			return report, errors.Join(cause, err)
		}
		if err := file.Close(); err != nil {
			return report, errors.Join(cause, err)
		}
	}
	if report.Status != "passed" {
		if cause != nil {
			return report, cause
		}
		if report.Error != "" {
			return report, fmt.Errorf("full verification %s: %s", report.Status, report.Error)
		}
		return report, fmt.Errorf("full verification %s", report.Status)
	}
	return report, nil
}

// FullVerifyReportDigest returns the stable content binding used by persisted
// full verification reports. Digest and local PersistedPath are excluded.
func FullVerifyReportDigest(report FullVerifyReport) (string, error) {
	report.Digest = ""
	report.PersistedPath = ""
	return digest(report)
}

func projectManagerMap(managers []projectmodel.Manager) map[string]projectmodel.Manager {
	out := make(map[string]projectmodel.Manager, len(managers))
	for _, manager := range managers {
		out[manager.ID] = manager
	}
	return out
}
