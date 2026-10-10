package projectrun

import (
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"reflect"
)

func pendingNativeVerifier(run RunReport, runtime Runtime) bool {
	if runtime.Verifier == nil || runtime.Verifier.Transport != TransportCodexAppServer || (run.Status != StatusVerifying && run.Status != StatusFailed) {
		return false
	}
	for i := len(run.Invocations) - 1; i >= 0; i-- {
		log := run.Invocations[i]
		if log.Role == agentexec.RoleVerifier && log.Phase == "verify" {
			return log.Outcome == "started" || log.Receipt.Lifecycle != nil && !workspaceInvocationTerminal(log.Receipt.Lifecycle)
		}
	}
	return false
}

// savedVerificationChecks retains the original actual process results. Native
// verifier recovery must never rerun checks to manufacture a second attempt.
func savedVerificationChecks(run RunReport, plan PlanRecord, candidateID string) ([]CheckResult, error) {
	var out []CheckResult
	for _, planned := range plan.Checks {
		var selected *CheckResult
		for i := len(run.Checks) - 1; i >= 0; i-- {
			check := run.Checks[i]
			if check.ID == planned.ID && check.CandidateID == candidateID {
				selected = &check
				break
			}
		}
		if selected == nil || selected.Outcome == "started" || (planned.Required && selected.Outcome != "passed") {
			return nil, fmt.Errorf("native verifier recovery lacks a completed original check %s", planned.ID)
		}
		if !reflect.DeepEqual(selected.Command, planned.Command) || selected.ExecutablePath != planned.ExecutablePath || selected.ExecutableDigest != planned.ExecutableDigest {
			return nil, fmt.Errorf("saved check %s does not match the planned command and executable pin", planned.ID)
		}
		out = append(out, *selected)
	}
	return out, nil
}
