//go:build !runtime_unit

package projectrun

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectexecutor"
)

func TestResumeReconcilesDurableRunningStateAfterCrash(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name          string
		inFlight      bool
		retryReady    bool
		wantInvokes   int
		wantStatus    string
		wantWorkCount int
	}{
		{name: "queued work continues", wantInvokes: 4, wantStatus: StatusIntegrated, wantWorkCount: 1},
		{name: "known retry ready continues without resetting ledger", retryReady: true, wantInvokes: 4, wantStatus: StatusIntegrated, wantWorkCount: 2},
		{name: "in flight is blocked without replay", inFlight: true, wantInvokes: 0, wantStatus: StatusBlocked, wantWorkCount: 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			root := makeProjectRunFixture(t)
			setupE2EProcess(t, "")
			revision := identityHead(t, root)
			host := resumeCrashHost(root, resumeCrashProject(t, root, revision))
			plan, err := Plan(host, root, revision, PlanRequest{
				Goal:              "Implement both owned artifacts and integrate them.",
				Managers:          []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")},
				ExecuteAuthorized: true,
			})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			report := persistRunningCrashState(t, root, plan, scenario.inFlight, scenario.retryReady)
			var originalRequest []byte
			var originalSource *snapshot.Snapshot
			if scenario.inFlight {
				originalSource, err = source.Load(root, "")
				if err != nil {
					t.Fatal(err)
				}
				if len(report.Invocations) != 1 || len(report.RoleStartReservations) != 1 || report.Invocations[0].Attempt == nil || report.Invocations[0].Attempt.Status != "started" || report.RoleStartReservations[0].Request.State != "requested" {
					t.Fatal("fixture did not seed one exact charged pre-dispatch attempt")
				}
				store, err := newRunStore(root)
				if err != nil {
					t.Fatal(err)
				}
				originalRequest, err = readStageEvidence(store, plan.ID, report.Invocations[0].Attempt.RequestRef)
				if err != nil {
					t.Fatal(err)
				}
				if report.Invocations[0].Attempt.RequestDigest != rawContentDigest(originalRequest) || report.Invocations[0].Receipt.RunID != "" || report.Invocations[0].Attempt.ResponseRef != nil || report.Invocations[0].Attempt.ReportRef != nil || !report.Invocations[0].Attempt.FinishedAt.IsZero() {
					t.Fatal("fixture fabricated output or lost original request binding")
				}
			}
			invoker := &resumeCountingInvoker{inner: projectexecutor.ProcessInvoker{}}
			resumed, resumeErr := Resume(context.Background(), host, invoker, root, plan.ID)
			if scenario.inFlight {
				if resumeErr == nil || !strings.Contains(resumeErr.Error(), "uncertain in-flight") {
					t.Fatalf("in-flight process was not reported uncertain: %v", resumeErr)
				}
				if resumed.Status != scenario.wantStatus || invoker.calls != scenario.wantInvokes {
					t.Fatalf("in-flight process was replayed or not blocked: status=%s invokes=%d", resumed.Status, invoker.calls)
				}
				uncertainTask := findTask(resumed.Tasks, e2eManagerID("", "project-owner"))
				if len(resumed.Invocations) != 1 || len(resumed.RoleStartReservations) != 1 || uncertainTask == nil || uncertainTask.State != "uncertain" {
					t.Fatalf("uncertain task state was not retained: %+v", uncertainTask)
				}
				store, err := newRunStore(root)
				if err != nil {
					t.Fatal(err)
				}
				persisted, err := store.readLatestState(plan.ID)
				if err != nil {
					t.Fatal(err)
				}
				persistedTask := findTask(persisted.Tasks, uncertainTask.ManagerID)
				if persisted.Status != StatusBlocked || persistedTask == nil || persistedTask.State != "uncertain" {
					t.Fatalf("uncertainty was not durably blocked: status=%s task=%+v", persisted.Status, persistedTask)
				}
				original := report.Invocations[0]
				current := persisted.Invocations[0]
				if current.Attempt == nil || current.Attempt.Status != "uncertain" || current.Receipt.RunID != "" || current.Attempt.ResponseRef != nil || current.Attempt.ReportRef != nil || !current.Attempt.FinishedAt.IsZero() || persisted.RoleStartReservations[0].Request.State != "unknown" {
					t.Fatal("unknown original attempt gained invented terminal output or lost its charged ticket")
				}
				attempt := *current.Attempt
				attempt.Status = original.Attempt.Status
				current.Attempt = &attempt
				a, _ := digest(current)
				b, _ := digest(original)
				if a != b {
					t.Fatal("resume changed original immutable invocation/request/receipt fields")
				}
				reservation := persisted.RoleStartReservations[0]
				if reservation.RecordedAt.Before(report.RoleStartReservations[0].RecordedAt) {
					t.Fatal("reservation lifecycle timestamp regressed")
				}
				reservation.RecordedAt = report.RoleStartReservations[0].RecordedAt
				reservation.Request.State = report.RoleStartReservations[0].Request.State
				a, _ = digest(reservation)
				b, _ = digest(report.RoleStartReservations[0])
				if a != b {
					t.Fatal("resume replaced the original charged reservation")
				}
				request, err := readStageEvidence(store, plan.ID, persisted.Invocations[0].Attempt.RequestRef)
				if err != nil || !bytes.Equal(request, originalRequest) {
					t.Fatal("protected original request bytes changed")
				}
				again, againErr := Resume(context.Background(), host, invoker, root, plan.ID)
				if againErr == nil || !strings.Contains(againErr.Error(), "uncertain in-flight") || again.Status != StatusBlocked || invoker.calls != 0 {
					t.Fatal("repeat resume replayed or accepted unknown process")
				}
				a, _ = digest(again.Invocations)
				b, _ = digest(persisted.Invocations)
				x, _ := digest(again.RoleStartReservations)
				y, _ := digest(persisted.RoleStartReservations)
				starts, _ := reportStartAccounting(again)
				repeatedTask := findTask(again.Tasks, uncertainTask.ManagerID)
				if a != b || x != y || starts.ObservedTotal != 1 || len(again.Invocations) != 1 || len(again.RoleStartReservations) != 1 || repeatedTask == nil || repeatedTask.WorkAttempts != uncertainTask.WorkAttempts || repeatedTask.Attempts != uncertainTask.Attempts {
					t.Fatal("repeat resume changed original authority or refilled its one charged start")
				}
				snapshotAfter, err := source.Load(root, "")
				if err != nil {
					t.Fatal(err)
				}
				changes := snapshot.Compare(originalSource, snapshotAfter)
				if len(changes.Modified) != 0 || len(changes.Removed) != 0 {
					t.Fatalf("blocked resume modified or removed original source/history: %+v", changes)
				}
				for _, changed := range changes.Added {
					// Only the existing run's durable status history may grow.
					// Request evidence, source/model/runtime and other runs stay exact.
					if !strings.HasPrefix(changed, RunsPath+"/"+plan.ID+"/states/") {
						t.Fatalf("blocked resume changed source outside original status history: %s", changed)
					}
				}
				request, err = readStageEvidence(store, plan.ID, again.Invocations[0].Attempt.RequestRef)
				if err != nil || !bytes.Equal(request, originalRequest) {
					t.Fatal("repeat resume changed original request bytes")
				}
				return
			}
			if resumeErr != nil {
				t.Fatalf("Resume running crash state: status=%s err=%v", resumed.Status, resumeErr)
			}
			if resumed.Status != scenario.wantStatus || invoker.calls != scenario.wantInvokes {
				t.Fatalf("queued work did not continue: status=%s invokes=%d", resumed.Status, invoker.calls)
			}
			rootTask := findTask(resumed.Tasks, e2eManagerID("", "project-owner"))
			if rootTask == nil || rootTask.WorkAttempts != scenario.wantWorkCount || rootTask.Attempts != scenario.wantWorkCount+1 {
				t.Fatalf("attempt ledger was reset or duplicated: %+v", rootTask)
			}
			if scenario.retryReady {
				if len(resumed.Invocations) != len(report.Invocations)+scenario.wantInvokes || resumed.Invocations[0].CostMicros != 17 {
					t.Fatalf("durable prior invocation/cost was not retained: %+v", resumed.Invocations)
				}
			}
		})
	}
}

func TestPersistedClosureRejectsHiddenTypedObligationsWithoutReplay(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "")
	revision := identityHead(t, root)
	host := resumeCrashHost(root, resumeCrashProject(t, root, revision))
	plan, err := Plan(host, root, revision, PlanRequest{
		Goal:              "Implement both owned artifacts and integrate them.",
		Managers:          []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")},
		ExecuteAuthorized: true,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	invoker := &resumeCountingInvoker{inner: projectexecutor.ProcessInvoker{}}
	run, err := Run(context.Background(), host, invoker, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	if _, err := Verify(context.Background(), host, invoker, root, plan.ID); err != nil {
		t.Fatalf("Verify baseline: %v", err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, run.Candidate.ID)
	if err != nil {
		t.Fatalf("PreflightApply baseline: %v", err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}

	mutateLatest := func(status string) RunReport {
		t.Helper()
		latest, err := store.readLatestState(plan.ID)
		if err != nil {
			t.Fatal(err)
		}
		rootTask := findTask(latest.Tasks, e2eManagerID("", "project-owner"))
		if rootTask == nil {
			t.Fatal("integrated report omitted root task")
		}
		rootTask.Obligations = []Obligation{{ID: "hidden-obligation", Kind: "question", Text: "hidden unresolved question", OriginManager: rootTask.ManagerID, Chain: []string{rootTask.ManagerID}}}
		latest.Status = status
		if err := persistState(store, &latest); err != nil {
			t.Fatal(err)
		}
		return latest
	}

	mutateLatest(StatusIntegrated)
	if _, err := Verify(context.Background(), host, invoker, root, plan.ID); err == nil || !strings.Contains(err.Error(), "unresolved report status or obligations") {
		t.Fatalf("Verify accepted an integrated report with a hidden typed obligation: %v", err)
	}
	mutateLatest(StatusVerified)
	if _, err := PreflightApply(host, root, plan.ID, run.Candidate.ID); err == nil || !strings.Contains(err.Error(), "unresolved report status or obligations") {
		t.Fatalf("PreflightApply accepted a verified report with a hidden typed obligation: %v", err)
	}
	if _, err := Apply(host, invoker, root, applyRequestFromPreflight(preflight)); err == nil || !strings.Contains(err.Error(), "unresolved report status or obligations") {
		t.Fatalf("Apply accepted a verified report with a hidden typed obligation: %v", err)
	}

	mutateLatest(StatusInterrupted)
	callsBeforeResume := invoker.calls
	resumed, err := Resume(context.Background(), host, invoker, root, plan.ID)
	if err == nil || !strings.Contains(err.Error(), "unresolved report status or obligations") || resumed.Status != StatusBlocked {
		t.Fatalf("Resume closed a report with a hidden typed obligation: status=%s err=%v", resumed.Status, err)
	}
	if invoker.calls != callsBeforeResume {
		t.Fatalf("closure guard replayed an invocation: before=%d after=%d", callsBeforeResume, invoker.calls)
	}
}

func TestApplyRechecksReportClosureAtGuardedWriteBoundary(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "")
	revision := identityHead(t, root)
	host := resumeCrashHost(root, resumeCrashProject(t, root, revision))
	plan, err := Plan(host, root, revision, PlanRequest{
		Goal:              "Implement both owned artifacts and integrate them.",
		Managers:          []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")},
		ExecuteAuthorized: true,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	invoker := &resumeCountingInvoker{inner: projectexecutor.ProcessInvoker{}}
	run, err := Run(context.Background(), host, invoker, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	if _, err := Verify(context.Background(), host, invoker, root, plan.ID); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, run.Candidate.ID)
	if err != nil {
		t.Fatalf("PreflightApply: %v", err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	workingLoads := 0
	guardedHost := host
	guardedHost.Load = func(loadRoot, revision string) (*projectvalue.Project, error) {
		if revision == "" {
			workingLoads++
			if workingLoads == 2 {
				latest, loadErr := store.readLatestState(plan.ID)
				if loadErr != nil {
					return nil, loadErr
				}
				rootTask := findTask(latest.Tasks, e2eManagerID("", "project-owner"))
				if rootTask == nil {
					return nil, fmt.Errorf("integrated report omitted root task")
				}
				rootTask.Obligations = []Obligation{{ID: "late-hidden-obligation", Kind: "question", Text: "late unresolved question", OriginManager: rootTask.ManagerID, Chain: []string{rootTask.ManagerID}}}
				if err := persistState(store, &latest); err != nil {
					return nil, err
				}
			}
		}
		return host.Load(loadRoot, revision)
	}
	result, err := Apply(guardedHost, invoker, root, applyRequestFromPreflight(preflight))
	if err == nil || !strings.Contains(err.Error(), "unresolved report status or obligations") {
		t.Fatalf("Apply did not reject a late hidden obligation at its guarded recheck: result=%+v err=%v", result, err)
	}
	if workingLoads != 2 || len(result.Written) != 0 || len(result.Journal) != 0 {
		t.Fatalf("guarded closure mutation reached a write: loads=%d result=%+v", workingLoads, result)
	}
	assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v1\n")
}
