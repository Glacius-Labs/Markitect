package projectrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	projectconfig "github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
)

// prepareRunExecution validates the durable plan bindings and admits a new or
// resumed invocation while the caller holds the run store lock.
func prepareRunExecution(ctx context.Context, host Host, invoker Invoker, root, id string, resume, repair bool, store *runStore) (*runExecution, RunReport, error) {
	plan, err := store.readPlan(id)
	if err != nil {
		return nil, RunReport{}, err
	}
	if !plan.ExecuteAuthorized {
		return nil, RunReport{}, ErrNotRunnable
	}
	if err := validateExplorationReadiness(host, root, plan); err != nil {
		report, supersedeErr := supersedeExisting(store, id, err)
		return nil, report, supersedeErr
	}
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		return nil, RunReport{}, err
	}
	if resume && !repair {
		ctx = withNativeResume(ctx)
	}
	project, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		report, supersedeErr := supersedeExisting(store, id, fmt.Errorf("reload fixed plan base: %w", err))
		return nil, report, supersedeErr
	}
	if project.Snapshot == nil || project.Snapshot.Digest() != plan.BaseSnapshot || project.Digest != plan.BaseProjectDigest || project.Report.ModelDigest != plan.BaseModelDigest {
		report, supersedeErr := supersedeExisting(store, id, fmt.Errorf("%w: fixed source or model binding differs from the plan", ErrStale))
		return nil, report, supersedeErr
	}
	if err := validateChangeImpact(host, root, project, plan); err != nil {
		report, supersedeErr := supersedeExisting(store, id, fmt.Errorf("change impact binding: %w", err))
		return nil, report, supersedeErr
	}
	working, err := host.Load(root, "")
	if err != nil {
		report, supersedeErr := supersedeExisting(store, id, err)
		return nil, report, supersedeErr
	}
	if working.Snapshot == nil || working.Snapshot.Digest() != plan.WorkingSnapshot || working.Digest != plan.WorkingProjectDigest {
		report, supersedeErr := supersedeExisting(store, id, fmt.Errorf("%w: working source or model binding differs from the plan", ErrStale))
		return nil, report, supersedeErr
	}
	if err := repositoryMatches(root, plan); err != nil {
		report, supersedeErr := supersedeExisting(store, id, fmt.Errorf("Git repository or target binding: %w", err))
		return nil, report, supersedeErr
	}
	if err := validateRuntimeBinding(invoker, runtime, plan); err != nil {
		report, supersedeErr := supersedeExisting(store, id, fmt.Errorf("method policy binding: %w", err))
		return nil, report, supersedeErr
	}
	dir, err := store.runDir(id)
	if err != nil {
		return nil, RunReport{}, err
	}
	report, stateErr := store.readLatestState(id)
	if stateErr != nil && !errors.Is(stateErr, ErrNotFound) {
		return nil, RunReport{}, stateErr
	}
	e := &runExecution{ctx: ctx, host: host, invoker: invoker, root: root, id: id, store: store, dir: dir, plan: plan, runtime: runtime, project: project, report: report}
	if errors.Is(stateErr, ErrNotFound) {
		if resume {
			return nil, RunReport{}, ErrNotFound
		}
		if len(plan.Blockers) > 0 {
			return nil, RunReport{}, fmt.Errorf("plan has blocking obligations: %s", strings.Join(plan.Blockers, "; "))
		}
		e.report = RunReport{APIVersion: APIVersion, Operation: plan.Operation, ID: id, PlanID: id, Status: StatusRunning, ExternalExecution: externalExecutionEnabled(ctx),
			Mode: projectconfig.ModeControlledLocal, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot, ModelDigest: plan.ModelDigest,
			RuntimeDigest: plan.RuntimeDigest, Tasks: cloneTasks(plan.Managers), Candidate: CandidateRef{ID: plan.InitialCandidateID,
				Files: map[string]string{}, Integrated: false}, Revision: 1, CostAccounting: CostAccountingComplete}
		initial, err := store.readCandidate(dir, plan.InitialCandidateID)
		if err != nil {
			return nil, RunReport{}, err
		}
		e.report.Candidate.Snapshot = initial.Digest
		if err := store.appendState(e.report); err != nil {
			return nil, RunReport{}, err
		}
	} else {
		if report.ExternalExecution != externalExecutionEnabled(ctx) {
			return nil, report, fmt.Errorf("run execution mode is fixed; use the original internal or external interface")
		}
		if !resume {
			return nil, report, fmt.Errorf("run %s already has durable state; use Resume", id)
		}
		if repair {
			e.report, err = beginRepair(store, dir, host, root, plan, runtime, project, report)
			if err != nil {
				return nil, e.report, err
			}
		} else {
			keepGoing, terminal, resumeErr := e.resumeExisting()
			if !keepGoing {
				return nil, terminal, resumeErr
			}
		}
	}
	e.recoveredResults = make(map[string]managerInvocationResult, len(e.pendingRecoveries))
	return e, RunReport{}, nil
}
