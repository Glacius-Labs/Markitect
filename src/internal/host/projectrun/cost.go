package projectrun

const (
	CostAccountingComplete = "complete"
	CostAccountingPartial  = "partial"
	CostAccountingUnknown  = "unknown"
)

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
