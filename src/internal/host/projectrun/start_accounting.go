package projectrun

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

// StartAccounting separates durable Host-root attempts from transport-observed
// nested starts. ObservedTotal is a lower-bound sum whenever Accounting is not
// complete; zero ObservedNested never implies that nested starts were absent.
type StartAccounting struct {
	Roots          int    `json:"roots"`
	ObservedNested int    `json:"observedNested"`
	ObservedTotal  int    `json:"observedTotal"`
	Accounting     string `json:"accounting"`
}

// reportStartAccounting counts Manager work/integration attempts from durable
// task counters, including failed attempts with no receipt. Other Host roles
// are counted from their invocation records. Nested starts come only from
// observed lifecycle receipts; checks are runtime processes, not role starts.
func reportStartAccounting(report RunReport) (StartAccounting, []agentexec.RoleStartRequest) {
	rootCount := 0
	taskRootAttempts := 0
	taskRootInvocationLogs := 0
	missingLifecycle := false
	partialLifecycle := false
	unavailableLifecycle := false
	seenNested := map[string]int{}
	var nested []agentexec.RoleStartRequest

	for _, task := range report.Tasks {
		attempts := task.WorkAttempts + task.IntegrationAttempts
		rootCount += attempts
		taskRootAttempts += attempts
	}
	for _, round := range report.RepairRounds {
		for _, task := range round.PriorTasks {
			attempts := task.WorkAttempts + task.IntegrationAttempts
			rootCount += attempts
			taskRootAttempts += attempts
		}
	}

	for _, invocation := range report.Invocations {
		taskRoot := invocation.Role == agentexec.RoleExecutor && (invocation.Phase == "work" || invocation.Phase == "integrate")
		if taskRoot {
			taskRootInvocationLogs++
		} else {
			rootCount++
		}
		lifecycle := invocation.Receipt.Lifecycle
		if lifecycle == nil {
			missingLifecycle = true
			continue
		}
		switch strings.ToLower(lifecycle.Accounting) {
		case "complete":
		case "partial":
			partialLifecycle = true
		case "unavailable":
			unavailableLifecycle = true
		default:
			// Unknown/empty status cannot establish that the nested list is whole.
			partialLifecycle = true
		}
		for index, request := range lifecycle.StartRequests {
			if index == 0 { // the transport-observed root is already counted above
				continue
			}
			key := nestedStartKey(lifecycle.Provider, lifecycle.SessionID, request)
			if key == "" {
				partialLifecycle = true
				nested = append(nested, request)
				continue
			}
			if prior, ok := seenNested[key]; ok {
				// Receipts may repeat a cumulative lifecycle snapshot. Prefer the
				// latest appearance so requested/failed/unknown states are retained.
				nested[prior] = request
				continue
			}
			seenNested[key] = len(nested)
			nested = append(nested, request)
		}
	}
	if taskRootInvocationLogs < taskRootAttempts {
		// Durable counters record the attempted starts even if execution failed
		// before a transport receipt or invocation record could be persisted.
		unavailableLifecycle = true
	}

	accounting := "complete"
	if unavailableLifecycle || missingLifecycle {
		accounting = "unavailable"
	} else if partialLifecycle {
		accounting = "partial"
	}
	summary := StartAccounting{Roots: rootCount, ObservedNested: len(nested),
		ObservedTotal: rootCount + len(nested), Accounting: accounting}
	return summary, nested
}

func nestedStartKey(provider, sessionID string, request agentexec.RoleStartRequest) string {
	if request.RequestID == "" {
		return ""
	}
	session := sessionID
	if session == "" {
		session = request.ParentSessionID
	}
	if session == "" {
		return ""
	}
	return fmt.Sprintf("%s\x00%s\x00%s", provider, session, request.RequestID)
}
