package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"

const (
	CostAccountingComplete = "complete"
	CostAccountingPartial  = "partial"
	CostAccountingUnknown  = "unknown"

	CostModeMetered   = "metered"
	CostModeUnmetered = "unmetered"
)

// Unmetered reports whether the agent declares that its invocations have no
// Markitect price. Start, duration and timeout limits still apply to it.
func (a Agent) Unmetered() bool { return a.CostMode == CostModeUnmetered }

// requiresReportedUsage reports whether missing provider usage must stop the
// invocation. A metered process executor must report usage so the known-cost
// budget stays bounded; the native App Server and unmetered agents record
// missing usage as unknown cost instead.
func (a Agent) requiresReportedUsage() bool {
	return a.Transport != TransportCodexAppServer && !a.Unmetered()
}

// estimateAgentCost prices reported usage with the agent's declared rates. An
// unmetered agent has no rates, so its cost stays unknown even when the
// executor reports usage; it never becomes an invented zero.
func estimateAgentCost(usage *agentexec.Usage, agent Agent) (int64, bool, bool) {
	if agent.Unmetered() {
		return 0, false, false
	}
	return estimateCostDetailed(usage, agent.Pricing)
}

// costAccounting reports telemetry completeness independently from the
// known-cost estimate used by MaxCostMicros. Unknown usage is not zero cost.
func costAccounting(logs []InvocationLog) string {
	known, unknown := costCounts(logs)
	return costAccountingFromCounts(known, unknown)
}

func costCounts(logs []InvocationLog) (known, unknown int) {
	for _, log := range logs {
		if log.CostKnown {
			known++
		} else {
			unknown++
		}
		if life := log.Receipt.Lifecycle; life != nil && life.Provider == TransportCodexAppServer {
			// Native child/helper role starts have no separately aggregated Usage
			// in the parent receipt, and partial lifecycle accounting cannot prove
			// that all model work is represented by this estimate.
			if life.Accounting != "complete" {
				unknown++
			}
			for _, request := range life.StartRequests {
				if request.Role == "helper" {
					unknown++
				}
			}
		}
	}
	return known, unknown
}

func runCostCounts(report RunReport) (known, unknown int) {
	known, unknown = costCounts(report.Invocations)
	seen := map[string]bool{}
	for _, invocation := range report.Invocations {
		if life := invocation.Receipt.Lifecycle; life != nil {
			for _, request := range life.StartRequests {
				seen[request.RequestID] = true
			}
		}
	}
	for _, reservation := range report.RoleStartReservations {
		if reservation.Kind == "helper" && !seen[reservation.Request.RequestID] {
			unknown++
		}
	}
	return known, unknown
}

func costAccountingFromCounts(known, unknown int) string {
	switch {
	case unknown == 0:
		return CostAccountingComplete
	case known == 0:
		return CostAccountingUnknown
	default:
		return CostAccountingPartial
	}
}
