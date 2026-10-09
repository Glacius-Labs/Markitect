package projectrun

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

func TestReportClosureBlocksNonterminalHostHelperReservations(t *testing.T) {
	for _, state := range []string{"requested", "started", "unknown", ""} {
		t.Run("state-"+state, func(t *testing.T) {
			report := completeClosureReport()
			report.RoleStartReservations = []RoleStartReservation{{Kind: "helper", Request: agentexec.RoleStartRequest{
				RequestID: "helper-request-7", Role: "helper", State: state,
			}}}
			err := validateReportClosure(report)
			if err == nil || !strings.Contains(err.Error(), "helper-request-7") || !strings.Contains(err.Error(), "nonterminal") {
				t.Fatalf("closure did not identify nonterminal helper request %q: %v", state, err)
			}
		})
	}
}

func TestReportClosureAllowsTerminalHostHelperReservations(t *testing.T) {
	for _, state := range []string{"completed", "failed", "interrupted"} {
		t.Run(state, func(t *testing.T) {
			report := completeClosureReport()
			report.RoleStartReservations = []RoleStartReservation{{Kind: "helper", Request: agentexec.RoleStartRequest{
				RequestID: "helper-request-terminal", Role: "helper", State: state,
			}}}
			if err := validateReportClosure(report); err != nil {
				t.Fatalf("known terminal helper request %q blocked closure: %v", state, err)
			}
		})
	}
}

func TestReportClosureDoesNotTreatPartialNativeAccountingAsOpenHelper(t *testing.T) {
	report := completeClosureReport()
	report.RoleStartReservations = []RoleStartReservation{{Kind: "root", Request: agentexec.RoleStartRequest{
		RequestID: "root-request", Role: "executor", State: "unknown",
	}}}
	report.Invocations = []InvocationLog{{Role: agentexec.RoleExecutor, Phase: "work", Receipt: agentexec.Receipt{
		Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "parent-session", Accounting: "partial",
			StartRequests: []agentexec.RoleStartRequest{{RequestID: "root-request", Role: "executor", State: "unknown"},
				{RequestID: "untracked-child", ParentSessionID: "parent-session", Role: "worker", State: "unknown"}}},
	}}}
	if err := validateReportClosure(report); err != nil {
		t.Fatalf("generic partial lifecycle telemetry or a non-helper root reservation blocked closure: %v", err)
	}
}

func completeClosureReport() RunReport {
	return RunReport{Tasks: []ManagerTask{{ManagerID: "root", ReportStatus: "complete"}}}
}
