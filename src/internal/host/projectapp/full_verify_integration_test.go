//go:build !runtime_unit

package projectapp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
)

func TestApplicationFullVerifyAuditsEveryManagerAndPersistsReceiptAndChecks(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
	logPath := filepath.Join(t.TempDir(), "full-verify.jsonl")
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyLogEnv, logPath)
	head := deliveryGit(t, root, "rev-parse", "HEAD")
	before := deliveryGit(t, root, "status", "--porcelain")
	operations := deliveryOperations()
	report, err := operations.FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
	if err != nil {
		t.Fatalf("Application FullVerify: %v report=%+v", err, report)
	}
	if report.Status != "passed" || len(report.Managers) != 3 || len(report.Checks) != 2 {
		t.Fatalf("unexpected full verification result: %+v", report)
	}
	if report.Starts != 5 {
		t.Fatalf("starts = %d, want three Manager audits and two checks", report.Starts)
	}
	if after := deliveryGit(t, root, "status", "--porcelain"); after != before {
		t.Fatalf("read-only verification changed working tree: before %q after %q", before, after)
	}
	data, err := os.ReadFile(logPath)
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
	written, err := operations.FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head, Write: true}})
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(written.PersistedPath)))
	if err != nil {
		t.Fatalf("persisted report missing: %v", err)
	}
	var stored projectrun.FullVerifyReport
	if err := json.Unmarshal(persisted, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Status != "passed" || len(stored.Managers) != 3 || stored.Managers[0].Receipt == nil || len(stored.Checks) != 2 {
		t.Fatalf("persisted report omitted receipt or check evidence: %+v", stored)
	}
}

func TestApplicationFullVerifyRejectsMissingManagerEvidenceWithoutTreeChanges(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyBadEnv, "orders")
	head := deliveryGit(t, root, "rev-parse", "HEAD")
	before := deliveryGit(t, root, "status", "--porcelain")
	report, err := deliveryOperations().FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
	if err == nil || report.Status == "passed" {
		t.Fatalf("missing Manager evidence passed: report=%+v err=%v", report, err)
	}
	if after := deliveryGit(t, root, "status", "--porcelain"); after != before {
		t.Fatalf("failed read-only audit changed project tree: before=%q after=%q", before, after)
	}
}

func TestApplicationComposedCandidateFullVerifyUsesCandidateBytesWithoutChangingSource(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
	head := deliveryGit(t, root, "rev-parse", "HEAD")
	operations := deliveryOperations()
	base, err := operations.Host.Load(root, head)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	baseDigest := base.Snapshot.Digest()
	base.Snapshot.Files["src/orders/implementation.txt"] = []byte("uncommitted composed candidate bytes\n")
	candidate, err := operations.Host.FromSnapshot(root, base.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Snapshot.Digest() == baseDigest {
		t.Fatal("candidate composition did not change snapshot binding")
	}
	report, err := projectrun.FullVerifyProject(context.Background(), operations.Host, operations.Invoker, root, candidate, runtime,
		projectrun.FullVerifyBinding{ExpectedSnapshot: candidate.Snapshot.Digest()})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || report.SnapshotDigest != candidate.Snapshot.Digest() {
		t.Fatalf("candidate verification did not bind to composed bytes: %+v", report)
	}
	assertDeliveryFileContents(t, root, "src/orders/implementation.txt", "orders implementation v1\n")
}

func TestApplicationFullVerifyStopsWhenUsageCannotBoundCumulativeCost(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyNoUsageEnv, "inventory")
	head := deliveryGit(t, root, "rev-parse", "HEAD")
	report, err := deliveryOperations().FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
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

func TestApplicationFullVerifyRetainsKnownCostForSuccessfulAndFailedCalls(t *testing.T) {
	t.Parallel()
	for _, failReview := range []bool{false, true} {
		name := "successful review"
		if failReview {
			name = "failed review"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := deliveryFixture(t)
			setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
			updateDeliveryRuntime(t, root, func(runtime *projectconfig.Runtime) {
				runtime.Limits.MaxCostMicros = 1
				for id, reviewer := range runtime.Review.Agents {
					reviewer.Pricing.InputMicrosPerMillion = 1_000_000
					reviewer.Pricing.OutputMicrosPerMillion = 1_000_000
					runtime.Review.Agents[id] = reviewer
				}
			})
			if failReview {
				setDeliveryProcessFixtureEnv(t, deliveryFullVerifyFailEnv, "inventory")
			}
			head := deliveryGit(t, root, "rev-parse", "HEAD")
			report, err := deliveryOperations().FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
			if err == nil || report.Status == "passed" {
				t.Fatalf("over-budget Manager call passed: report=%+v err=%v", report, err)
			}
			if report.CostMicros != 15 || report.Starts != 3 || report.Managers[0].CostMicros != 15 {
				t.Fatalf("known over-budget cost was not retained: total=%d starts=%d first=%+v", report.CostMicros, report.Starts, report.Managers[0])
			}
		})
	}
}

func TestApplicationFullVerifyCumulativeStartLimitLeavesUnstartedManagersIncomplete(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
	updateDeliveryRuntime(t, root, func(runtime *projectconfig.Runtime) { runtime.Limits.MaxStarts = 4 })
	head := deliveryGit(t, root, "rev-parse", "HEAD")
	report, err := deliveryOperations().FullVerify(context.Background(), FullVerifyOperation{Root: root, Request: projectrun.FullVerifyRequest{Revision: head}})
	if err == nil || report.Status != "incomplete" {
		t.Fatalf("start-budget exhaustion passed: report=%+v err=%v", report, err)
	}
	if report.Starts != 4 || report.Managers[0].Status != "passed" || report.Managers[1].Status != "passed" || report.Managers[2].Status != "incomplete" {
		t.Fatalf("cumulative start budget was not shared across checks and Managers: %+v", report)
	}
}

func assertDeliveryFileContents(t *testing.T, root, path, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
}
