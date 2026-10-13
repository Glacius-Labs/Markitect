package projectcli

import (
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

// These finite output copies preserve the ordinary delivery wire contract.
// Internal stage authority stays in Runtime and the complete status readback.
type runReportOutput struct {
	ExternalExecution       bool                              `json:"externalExecution,omitempty"`
	Operation               string                            `json:"operation,omitempty"`
	APIVersion              string                            `json:"apiVersion"`
	ID                      string                            `json:"id"`
	PlanID                  string                            `json:"planId"`
	Status                  string                            `json:"status"`
	Mode                    string                            `json:"mode"`
	StartedAt               time.Time                         `json:"startedAt"`
	UpdatedAt               time.Time                         `json:"updatedAt"`
	BaseRevision            string                            `json:"baseRevision"`
	BaseSnapshot            string                            `json:"baseSnapshot"`
	ModelDigest             string                            `json:"modelDigest"`
	RuntimeDigest           string                            `json:"runtimeDigest"`
	Candidate               projectrun.CandidateRef           `json:"candidate"`
	Tasks                   []projectrun.ManagerTask          `json:"tasks"`
	Invocations             []invocationOutput                `json:"invocations"`
	RoleStartReservations   []projectrun.RoleStartReservation `json:"roleStartReservations,omitempty"`
	CostMicros              int64                             `json:"costMicros"`
	CostAccounting          string                            `json:"costAccounting"`
	Checks                  []projectrun.CheckResult          `json:"checks"`
	Findings                []string                          `json:"findings,omitempty"`
	Escalations             []projectrun.Escalation           `json:"escalations,omitempty"`
	RepairRounds            []projectrun.RepairRound          `json:"repairRounds,omitempty"`
	Reviews                 []projectrun.ReviewRecord         `json:"reviews,omitempty"`
	ManagerReworkRounds     []projectrun.ManagerReworkRound   `json:"managerReworkRounds,omitempty"`
	ActiveRepairCandidateID string                            `json:"activeRepairCandidateId,omitempty"`
	Revision                uint64                            `json:"revision"`
	Digest                  string                            `json:"digest"`
}

type invocationOutput struct {
	TaskID       string            `json:"taskId"`
	Role         string            `json:"role"`
	Phase        string            `json:"phase"`
	InputDigest  string            `json:"inputDigest"`
	Receipt      agentexec.Receipt `json:"receipt"`
	ReportID     string            `json:"reportId"`
	Outcome      string            `json:"outcome"`
	CostMicros   int64             `json:"costMicros"`
	CostKnown    bool              `json:"costKnown"`
	CostOverflow bool              `json:"costOverflow,omitempty"`
}

type deliverOutput struct {
	ExplorationID string                     `json:"explorationId"`
	ScopeID       string                     `json:"scopeId"`
	RunID         string                     `json:"runId,omitempty"`
	Status        string                     `json:"status"`
	Plan          *projectrun.PlanRecord     `json:"plan,omitempty"`
	Run           *runReportOutput           `json:"run,omitempty"`
	Verification  *projectrun.VerifyReport   `json:"verification,omitempty"`
	Preflight     *projectrun.ApplyPreflight `json:"preflight,omitempty"`
	Apply         *projectrun.ApplyReport    `json:"apply,omitempty"`
}

func outputRunReport(report projectrun.RunReport) runReportOutput {
	out := runReportOutput{
		ExternalExecution:       report.ExternalExecution,
		Operation:               report.Operation,
		APIVersion:              report.APIVersion,
		ID:                      report.ID,
		PlanID:                  report.PlanID,
		Status:                  report.Status,
		Mode:                    report.Mode,
		StartedAt:               report.StartedAt,
		UpdatedAt:               report.UpdatedAt,
		BaseRevision:            report.BaseRevision,
		BaseSnapshot:            report.BaseSnapshot,
		ModelDigest:             report.ModelDigest,
		RuntimeDigest:           report.RuntimeDigest,
		Candidate:               report.Candidate,
		Tasks:                   report.Tasks,
		RoleStartReservations:   report.RoleStartReservations,
		CostMicros:              report.CostMicros,
		CostAccounting:          report.CostAccounting,
		Checks:                  report.Checks,
		Findings:                report.Findings,
		Escalations:             report.Escalations,
		RepairRounds:            report.RepairRounds,
		Reviews:                 report.Reviews,
		ManagerReworkRounds:     report.ManagerReworkRounds,
		ActiveRepairCandidateID: report.ActiveRepairCandidateID,
		Revision:                report.Revision,
		Digest:                  report.Digest,
	}
	if report.Invocations != nil {
		out.Invocations = make([]invocationOutput, len(report.Invocations))
		for i, log := range report.Invocations {
			out.Invocations[i] = invocationOutput{
				TaskID:       log.TaskID,
				Role:         log.Role,
				Phase:        log.Phase,
				InputDigest:  log.InputDigest,
				Receipt:      log.Receipt,
				ReportID:     log.ReportID,
				Outcome:      log.Outcome,
				CostMicros:   log.CostMicros,
				CostKnown:    log.CostKnown,
				CostOverflow: log.CostOverflow,
			}
		}
	}
	return out
}

func outputDeliverReport(report projectrun.DeliverReport) deliverOutput {
	out := deliverOutput{
		ExplorationID: report.ExplorationID,
		ScopeID:       report.ScopeID,
		RunID:         report.RunID,
		Status:        report.Status,
		Plan:          report.Plan,
		Verification:  report.Verification,
		Preflight:     report.Preflight,
		Apply:         report.Apply,
	}
	if report.Run != nil {
		run := outputRunReport(*report.Run)
		out.Run = &run
	}
	return out
}
