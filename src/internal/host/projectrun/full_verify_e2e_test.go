//go:build !runtime_unit

package projectrun

import (
	"context"
	"encoding/json"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

func TestFullVerifyAuditsAllManagersAndParentsReceiveIntegrationScope(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	loaded, loadErr := projectwork.Load(root, gitE2E(t, root, "rev-parse", "HEAD"))
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if loaded.Coverage != nil && !loaded.Coverage.Conforming {
		t.Fatalf("fixture coverage not conforming: %+v", loaded.Coverage.Findings)
	}
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	before := gitE2E(t, root, "status", "--porcelain")
	host := Host{Load: projectwork.Load}
	got, err := FullVerify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err != nil {
		t.Fatalf("full verify failed: %v report=%+v", err, got)
	}
	if got.Status != "passed" || len(got.Managers) != 3 || len(got.Checks) != 2 {
		t.Fatalf("unexpected full verification result: %+v", got)
	}
	if got.Starts != 5 {
		t.Fatalf("starts = %d, want three Manager audits and two checks", got.Starts)
	}
	if after := gitE2E(t, root, "status", "--porcelain"); after != before {
		t.Fatalf("read-only verification changed working tree: before %q after %q", before, after)
	}
	data, err := os.ReadFile(runtimeFixtureEnv(t, fullVerifyLogEnv))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), `"context":`) != 3 {
		t.Fatalf("logged Manager calls = %d, want 3", strings.Count(string(data), `"context":`))
	}
	if !strings.Contains(string(data), "integration:manager:") || !strings.Contains(string(data), "integration:artifact:") {
		t.Fatalf("root Manager did not receive direct-child integration obligations: %s", data)
	}
	if !strings.Contains(string(data), `"childAssessments":[{"managerId"`) || !strings.Contains(string(data), `"checkResults":[{"id"`) {
		t.Fatalf("root Manager did not receive direct-child assessment and check evidence: %s", data)
	}
	written, err := FullVerify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, FullVerifyRequest{Revision: head, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := os.ReadFile(filepath.FromSlash(written.PersistedPath))
	if err != nil {
		t.Fatalf("persisted report missing: %v", err)
	}
	var stored FullVerifyReport
	if err := json.Unmarshal(persisted, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Status != "passed" || len(stored.Managers) != 3 || stored.Managers[0].Receipt == nil || len(stored.Checks) != 2 {
		t.Fatalf("persisted report omitted receipt or check evidence: %+v", stored)
	}
}

func TestFullVerifyMissingManagerAssessmentCannotPassAndStaleSnapshotRejected(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	before := gitE2E(t, root, "status", "--porcelain")
	host := Host{Load: projectwork.Load}
	setRuntimeFixtureEnv(t, fullVerifyBadEnv, "orders")
	report, err := FullVerify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status == "passed" {
		t.Fatalf("missing Manager evidence passed: report=%+v err=%v", report, err)
	}
	if after := gitE2E(t, root, "status", "--porcelain"); after != before {
		t.Fatalf("failed read-only audit changed project tree: before=%q after=%q", before, after)
	}
	setRuntimeFixtureEnv(t, fullVerifyBadEnv, "")
	project, err := projectwork.Load(root, head)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FullVerifyProject(context.Background(), host, projectexecutor.ProcessInvoker{}, root, project, runtime, FullVerifyBinding{ExpectedSnapshot: "stale"}); err != ErrStale {
		t.Fatalf("stale snapshot error = %v, want ErrStale", err)
	}
}

func TestFullVerifyProjectAuditsComposedCandidateBytesWithoutReopeningBaseRevision(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := Host{Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot}
	base, err := host.Load(root, head)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	baseDigest := base.Snapshot.Digest()
	base.Snapshot.Files["src/orders/implementation.txt"] = []byte("uncommitted composed candidate bytes\n")
	candidate, err := host.FromSnapshot(root, base.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Snapshot.Digest() == baseDigest {
		t.Fatal("candidate composition did not change snapshot binding")
	}
	report, err := FullVerifyProject(context.Background(), host, projectexecutor.ProcessInvoker{}, root, candidate, runtime, FullVerifyBinding{ExpectedSnapshot: candidate.Snapshot.Digest()})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || report.SnapshotDigest != candidate.Snapshot.Digest() {
		t.Fatalf("candidate verification did not bind to composed bytes: %+v", report)
	}
	assertFileContents(t, root, "src/orders/implementation.txt", "orders implementation v1\n")
}

func TestFullVerifyStopsWhenUsageCannotBoundCumulativeCost(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := Host{Load: projectwork.Load}
	setRuntimeFixtureEnv(t, fullVerifyNoUsageEnv, "inventory")
	report, err := FullVerify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status != "incomplete" {
		t.Fatalf("missing usage did not leave full verification incomplete: report=%+v err=%v", report, err)
	}
	if report.Starts != 3 {
		t.Fatalf("starts=%d; want two checks and exactly one Manager before stopping", report.Starts)
	}
	if report.Managers[1].Status != "incomplete" || report.Managers[2].Status != "incomplete" {
		t.Fatalf("later Managers ran without a bounded cost basis: %+v", report.Managers)
	}
}

func TestFullVerifyRecordsKnownCostBeyondLimitForSuccessfulAndFailedCalls(t *testing.T) {
	t.Parallel()
	for _, failReview := range []bool{false, true} {
		name := "successful review"
		if failReview {
			name = "failed review"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := makeFullVerifyFixture(t)
			setupFullVerifyProcesses(t)
			configureFullVerifyRuntime(t, root)
			updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
				runtime.Limits.MaxCostMicros = 1
				for id, reviewer := range runtime.Review.Agents {
					reviewer.Pricing.InputMicrosPerMillion = 1_000_000
					reviewer.Pricing.OutputMicrosPerMillion = 1_000_000
					runtime.Review.Agents[id] = reviewer
				}
			})
			if failReview {
				setRuntimeFixtureEnv(t, fullVerifyBadEnv, "inventory")
			}
			head := gitE2E(t, root, "rev-parse", "HEAD")
			report, err := FullVerify(context.Background(), Host{Load: projectwork.Load}, projectexecutor.ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
			if err == nil || report.Status == "passed" {
				t.Fatalf("over-budget Manager call passed: report=%+v err=%v", report, err)
			}
			if report.CostMicros != 15 || report.Starts != 3 || report.Managers[0].CostMicros != 15 {
				t.Fatalf("known over-budget cost was not retained: total=%d starts=%d first=%+v", report.CostMicros, report.Starts, report.Managers[0])
			}
		})
	}
}

func TestFullVerifyCumulativeStartLimitLeavesUnstartedManagersIncomplete(t *testing.T) {
	t.Parallel()
	root := makeFullVerifyFixture(t)
	setupFullVerifyProcesses(t)
	configureFullVerifyRuntime(t, root)
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) { runtime.Limits.MaxStarts = 4 })
	head := gitE2E(t, root, "rev-parse", "HEAD")
	host := Host{Load: projectwork.Load}
	report, err := FullVerify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, FullVerifyRequest{Revision: head})
	if err == nil || report.Status != "incomplete" {
		t.Fatalf("start-budget exhaustion passed: report=%+v err=%v", report, err)
	}
	if report.Starts != 4 || report.Managers[0].Status != "passed" || report.Managers[1].Status != "passed" || report.Managers[2].Status != "incomplete" {
		t.Fatalf("cumulative start budget was not shared across checks and Managers: %+v", report)
	}
}
