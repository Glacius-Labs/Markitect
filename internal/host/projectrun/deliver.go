package projectrun

import (
	"context"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/host/projectexplore"
)

// DeliverRequest identifies one acknowledged work-item scope. ExecuteAuthorized
// must come from the caller's explicit write/run authorization.
type DeliverRequest struct {
	ExplorationID     string `json:"explorationId"`
	ScopeID           string `json:"scopeId"`
	RunID             string `json:"runId,omitempty"`
	ExecuteAuthorized bool   `json:"executeAuthorized"`
}

// DeliverReport preserves each durable stage so a blocked operation can be
// resumed or repaired explicitly without losing its current run identity.
type DeliverReport struct {
	ExplorationID string          `json:"explorationId"`
	ScopeID       string          `json:"scopeId"`
	RunID         string          `json:"runId,omitempty"`
	Status        string          `json:"status"`
	Plan          *PlanRecord     `json:"plan,omitempty"`
	Run           *RunReport      `json:"run,omitempty"`
	Verification  *VerifyReport   `json:"verification,omitempty"`
	Preflight     *ApplyPreflight `json:"preflight,omitempty"`
	Apply         *ApplyReport    `json:"apply,omitempty"`
}

// Deliver plans an acknowledged scope once, then advances that same durable
// run through integration, verification, guarded Apply, and completion. A
// resumed delivery never creates a replacement proposal.
func Deliver(ctx context.Context, host Host, invoker Invoker, root string, request DeliverRequest) (DeliverReport, error) {
	out := DeliverReport{ExplorationID: request.ExplorationID, ScopeID: request.ScopeID, Status: "blocked"}
	if !request.ExecuteAuthorized {
		return out, fmt.Errorf("deliver requires explicit write authorization")
	}
	if request.ExplorationID == "" || request.ScopeID == "" {
		return out, fmt.Errorf("deliver requires an exploration ID and scope ID")
	}
	if host.Load == nil || host.FromSnapshot == nil || invoker == nil {
		return out, fmt.Errorf("deliver requires a project Host and agent invoker")
	}

	var plan PlanRecord
	var run RunReport
	newPlan := request.RunID == ""
	if newPlan {
		record, scope, _, readiness, err := LoadExplorationReadiness(host, root, request.ExplorationID, request.ScopeID)
		if err != nil {
			return out, err
		}
		if !readiness.Ready {
			return out, fmt.Errorf("scope is not ready: %v", readiness.Blockers)
		}
		plan, err = Plan(host, root, "", PlanRequest{
			ExplorationID: request.ExplorationID, ScopeID: request.ScopeID,
			Operation: scope.Operation, Goal: scope.Goal, Managers: append([]string(nil), scope.ManagerIDs...), ExecuteAuthorized: true,
		})
		if err != nil {
			return out, err
		}
		if plan.ExplorationID != record.ID || plan.ScopeID != scope.ID {
			return out, fmt.Errorf("new plan did not retain its exploration scope binding")
		}
		request.RunID = plan.ID
	} else {
		stored, err := Status(root, request.RunID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return out, err
		}
		plan = stored.Plan
		if plan.ID == "" {
			return out, fmt.Errorf("run %s has no stored plan", request.RunID)
		}
		if plan.ExplorationID != request.ExplorationID || plan.ScopeID != request.ScopeID || plan.ExplorationBinding == nil {
			return out, fmt.Errorf("run %s is not bound to exploration %s scope %s", request.RunID, request.ExplorationID, request.ScopeID)
		}
		// A crash after a durable Apply is recovered from the immutable apply
		// receipt. Current bytes have intentionally moved beyond the old basis.
		if stored.Run.Status == StatusApplied {
			if err := RecoverExplorationCompletion(host, root, request.RunID); err != nil {
				return out, err
			}
			out.RunID, out.Status, out.Plan, out.Run = request.RunID, StatusApplied, &plan, &stored.Run
			return out, nil
		}
		if _, _, binding, readiness, err := LoadExplorationReadiness(host, root, request.ExplorationID, request.ScopeID); err != nil {
			return out, err
		} else {
			currentDigest, digestErr := projectexplore.BindingDigest(binding)
			storedDigest, storedErr := projectexplore.BindingDigest(*plan.ExplorationBinding)
			if digestErr != nil {
				return out, digestErr
			}
			if storedErr != nil {
				return out, storedErr
			}
			if !readiness.Ready || readiness.Digest != plan.ReadinessDigest || currentDigest != storedDigest {
				return out, fmt.Errorf("%w: exploration readiness changed; create a new acknowledged scope or inspect the existing run", ErrStale)
			}
		}
		run = stored.Run
	}

	out.RunID, out.Plan = request.RunID, &plan
	if run.ID == "" || run.Status == "" || run.Status == StatusPlanned {
		var err error
		run, err = Run(ctx, host, invoker, root, request.RunID)
		out.Run = &run
		if err != nil {
			return stopDelivery(out, run, err)
		}
	} else if run.Status == StatusRunning || run.Status == StatusInterrupted {
		var err error
		run, err = Resume(ctx, host, invoker, root, request.RunID)
		out.Run = &run
		if err != nil {
			return stopDelivery(out, run, err)
		}
	}
	out.Run = &run
	switch run.Status {
	case StatusBlocked, StatusFailed, StatusSuperseded:
		return stopDelivery(out, run, fmt.Errorf("run %s is %s; inspect status and use explicit resume or repair after resolving the reported cause", run.ID, run.Status))
	case StatusIntegrated:
		verified, err := Verify(ctx, host, invoker, root, request.RunID)
		out.Verification = &verified
		if err != nil {
			return stopDelivery(out, run, err)
		}
		run.Status = StatusVerified
	case StatusVerified:
		store, err := newRunStore(root)
		if err != nil {
			return stopDelivery(out, run, err)
		}
		dir, err := store.runDir(request.RunID)
		if err != nil {
			return stopDelivery(out, run, err)
		}
		verified, err := latestVerification(dir, run.Candidate.ID)
		if err != nil {
			return stopDelivery(out, run, err)
		}
		out.Verification = &verified
	case StatusApplied:
		if err := RecoverExplorationCompletion(host, root, request.RunID); err != nil {
			return stopDelivery(out, run, err)
		}
		out.Status = StatusApplied
		return out, nil
	default:
		return stopDelivery(out, run, fmt.Errorf("run %s is %s; delivery can continue only from integrated or verified state", run.ID, run.Status))
	}

	preflight, err := PreflightApply(host, root, request.RunID, run.Candidate.ID)
	out.Preflight = &preflight
	if err != nil {
		return stopDelivery(out, run, err)
	}
	apply, err := Apply(host, invoker, root, ApplyRequest{
		RunID: preflight.RunID, PlanID: preflight.PlanID, CandidateID: preflight.CandidateID,
		ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch,
		ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree,
	})
	out.Apply = &apply
	if err != nil {
		return stopDelivery(out, run, err)
	}
	if apply.Status != StatusApplied {
		return stopDelivery(out, run, fmt.Errorf("Apply ended in status %s", apply.Status))
	}
	if err := RecoverExplorationCompletion(host, root, request.RunID); err != nil {
		return stopDelivery(out, run, err)
	}
	out.Status = StatusApplied
	return out, nil
}

func stopDelivery(out DeliverReport, run RunReport, err error) (DeliverReport, error) {
	out.Run = &run
	if run.Status != "" {
		out.Status = run.Status
	}
	return out, err
}
