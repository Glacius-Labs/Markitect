//go:build !runtime_unit

package projectrun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

func TestProjectRunReviewerFindingsTriggerTargetedImplementerRepair(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "review-defect-fix")
	updateE2ERuntime(t, root, func(config *projectconfig.Runtime) {
		config.Review = &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{}, MaxRounds: 2, MaxManagerRounds: 2}
		for id, agent := range config.Agents {
			config.Review.Agents[id] = agent
		}
		config.Limits.MaxStarts = 32
	})
	host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and reconcile the result.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != StatusIntegrated || !run.Candidate.Integrated {
		t.Fatalf("review loop did not integrate: %+v", run)
	}
	var ordersReviews []ReviewRecord
	for _, review := range run.Reviews {
		if review.ManagerID == e2eManagerID("orders", "orders") && review.Phase == "work" {
			ordersReviews = append(ordersReviews, review)
		}
	}
	if len(ordersReviews) != 2 || ordersReviews[0].Outcome != "fail" || ordersReviews[1].Outcome != "pass" || ordersReviews[0].CandidateDigest == ordersReviews[1].CandidateDigest {
		t.Fatalf("expected a defect-bound failed review followed by a fresh passing candidate review: %+v", ordersReviews)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i, review := range ordersReviews {
		candidate, err := store.readCandidate(dir, review.CandidateID)
		if err != nil || candidate.Digest != review.CandidateDigest {
			t.Fatalf("review %d is not bound to its retained candidate: candidate=%+v err=%v", i, candidate, err)
		}
		if got := string(candidate.Files["src/orders/implementation.txt"].Content); i == 0 && !strings.Contains(got, "DEFECT") || i == 1 && got != "orders implementation v2\n" {
			t.Fatalf("review %d candidate bytes were %q", i, got)
		}
	}
	if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("orders", "orders"), "work"); calls != 2 {
		t.Fatalf("implementer ran %d times after review findings, want 2", calls)
	}
	if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("inventory", "inventory"), "work"); calls != 1 {
		t.Fatalf("unaffected sibling ran %d times, want 1", calls)
	}
	latest, err := store.readLatestState(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	staleCandidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	staleCandidate.ID, err = newID()
	if err != nil {
		t.Fatal(err)
	}
	staleCandidate.Parents = []string{run.Candidate.ID}
	ordersFile := staleCandidate.Files["src/orders/implementation.txt"]
	ordersFile.Content = []byte("changed after review\n")
	staleCandidate.Files[ordersFile.Path] = ordersFile
	if err := store.writeCandidate(dir, staleCandidate); err != nil {
		t.Fatal(err)
	}
	staleCandidate, err = store.readCandidate(dir, staleCandidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	latest.Candidate = candidateRef(staleCandidate, true)
	if err := persistState(store, &latest); err != nil {
		t.Fatal(err)
	}
	// The changed child file is in both the child's work review and the
	// parent's integration review scope, so either one is stale.
	if _, staleErr := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID); staleErr == nil || !strings.Contains(staleErr.Error(), "has no fresh passed") {
		t.Fatalf("Verify accepted bytes outside the passed review scope: %v", staleErr)
	}
	latest.Candidate = run.Candidate
	if err := persistState(store, &latest); err != nil {
		t.Fatal(err)
	}
	verified, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != "verified" {
		t.Fatalf("fresh scoped reviews did not permit independent verification: report=%+v err=%v", verified, err)
	}
	paths, err := ApplyPaths(host, root, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	worktreeDigest, err := CaptureTarget(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	request := ApplyRequest{RunID: plan.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID, ExpectedVerificationDigest: verified.Digest,
		TargetBranch: gitE2E(t, root, "branch", "--show-current"), ExpectedHead: identityHead(t, root), ExpectedWorktree: worktreeDigest}
	if _, err := Apply(host, projectexecutor.ProcessInvoker{}, root, request); err != nil {
		t.Fatalf("freshly reviewed candidate could not pass guarded Apply: %v", err)
	}
}

func TestProjectRunReviewerRoundLimitBlocksRepeatedDefect(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "review-always-fail")
	enableE2EReviews(t, root, 100000)
	updateE2ERuntime(t, root, func(config *projectconfig.Runtime) { config.Review.MaxRounds = 2 })
	host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement and review the owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err == nil || run.Status != StatusBlocked {
		t.Fatalf("persistent review defect did not block at the configured round limit: status=%s err=%v", run.Status, err)
	}
	orders := e2eManagerID("orders", "orders")
	if reviews := reviewCount(run.Reviews, orders, "work"); reviews != 2 {
		t.Fatalf("work review rounds=%d, want bounded limit 2", reviews)
	}
	if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), orders, "work"); calls != 2 {
		t.Fatalf("bounded reviewer loop launched %d implementer calls, want 2", calls)
	}
}

func TestManagerDirectedReworkRerunsOnlyRequestedLeafAndReintegratesAncestors(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "manager-directed-rework")
	updateE2ERuntime(t, root, func(config *projectconfig.Runtime) {
		config.Review = &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{}, MaxRounds: 3, MaxManagerRounds: 2}
		for id, agent := range config.Agents {
			config.Review.Agents[id] = agent
		}
		config.Limits.MaxStarts = 32
	})
	host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and reconcile the result.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != StatusIntegrated || len(run.ManagerReworkRounds) != 1 || run.ManagerReworkRounds[0].Status != "completed" {
		t.Fatalf("manager-directed rework did not close: status=%s rounds=%+v", run.Status, run.ManagerReworkRounds)
	}
	if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("orders", "orders"), "work"); calls != 2 {
		t.Fatalf("requested leaf work ran %d times, want initial plus targeted rework", calls)
	}
	if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("inventory", "inventory"), "work"); calls != 1 {
		t.Fatalf("unaffected sibling work ran %d times, want 1", calls)
	}
	if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("", "project-owner"), "integrate"); calls != 2 {
		t.Fatalf("affected root integrations ran %d times, want initial plus reintegration", calls)
	}
	if len(run.Reviews) < 4 {
		t.Fatalf("expected work and fresh reintegration reviews, got %d", len(run.Reviews))
	}
	if got := reviewCount(run.Reviews, e2eManagerID("", "project-owner"), "integrate"); got != 1 {
		t.Fatalf("initial parent candidate with pending child rework consumed %d integration reviews, want only the fresh post-rework review", got)
	}
	ordersTask := findTask(run.Tasks, e2eManagerID("orders", "orders"))
	if ordersTask == nil || ordersTask.State != "worked" {
		t.Fatalf("successfully reworked leaf was not durably marked worked: %+v", ordersTask)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	finalCandidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(finalCandidate.Files["src/project-integration.txt"].Content); got != "parent integration edit survives child rework\n" {
		t.Fatalf("targeted child rework discarded parent-owned integration edit: %q", got)
	}
	// Model a durable interruption just after the completed rework round and
	// before the caller persists final integration closure. Resume must trust
	// the worked task and its reviews without replaying its subprocesses.
	latest, err := store.readLatestState(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	latest.Status = StatusInterrupted
	if err := persistState(store, &latest); err != nil {
		t.Fatal(err)
	}
	resumed, resumeErr := Resume(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	resumedOrders := findTask(resumed.Tasks, ordersTask.ManagerID)
	if resumeErr != nil || resumed.Status != StatusIntegrated || resumedOrders == nil || resumedOrders.State != "worked" || len(resumed.Invocations) != len(run.Invocations) {
		t.Fatalf("known-complete targeted rework did not resume without losing state: status=%s task=%+v err=%v", resumed.Status, resumedOrders, resumeErr)
	}
}

func TestPureRoutingManagerSkipsWorkReviewButReviewsDeliveredChildOutputs(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	enableE2EReviews(t, root, 100000)
	host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("pure routing fixture failed: status=%s err=%v", run.Status, err)
	}
	rootID := e2eManagerID("", "project-owner")
	rootTask := findTask(run.Tasks, rootID)
	if rootTask == nil || len(rootTask.WrittenPaths)+len(rootTask.IntegratedPaths) != 0 || rootTask.ReviewStatus != "pass" {
		t.Fatalf("file-less router whose children delivered was not integration-reviewed: %+v", rootTask)
	}
	for _, review := range run.Reviews {
		if review.ManagerID == rootID && review.Phase != "integrate" {
			t.Fatalf("pure router's delegation-only %s phase was reviewed: %+v", review.Phase, review)
		}
	}
	records, err := readE2ERecords(runtimeFixtureEnv(t, e2eLogEnv))
	if err != nil {
		t.Fatal(err)
	}
	var reviewed []string
	for _, record := range records {
		if record["phase"] == "review" && record["managerId"] == rootID {
			artifacts, _ := record["artifacts"].([]any)
			for _, artifact := range artifacts {
				fields, _ := artifact.(map[string]any)
				path, _ := fields["path"].(string)
				reviewed = append(reviewed, path)
			}
		}
	}
	for _, path := range []string{"src/orders/implementation.txt", "src/inventory/implementation.txt"} {
		if !containsString(reviewed, path) {
			t.Fatalf("router integration review input omitted child-delivered %s: %v", path, reviewed)
		}
	}
	for _, managerID := range []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")} {
		if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), managerID, "review"); calls == 0 {
			t.Fatalf("implementation Manager %s skipped mandatory review", managerID)
		}
	}
}

func TestTargetedWorkRejectsNoOpWithCandidateWrites(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "manager-rework-invalid-noop")
	enableE2EReviews(t, root, 100000)
	host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err == nil || run.Status != StatusBlocked || !strings.Contains(err.Error(), "claimed no-op while proposing files") {
		t.Fatalf("targeted work accepted no-op alongside candidate writes: status=%s err=%v", run.Status, err)
	}
	orders := e2eManagerID("orders", "orders")
	if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), orders, "work"); calls != 2 {
		t.Fatalf("targeted work calls=%d, want one normal and one rejected rework response", calls)
	}
}

func TestIntegrationReviewFindingsRepairAndRereviewParentCandidate(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	projectPath := filepath.Join(root, filepath.FromSlash(projectvalue.ManifestPath))
	projectYAML, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	projectYAML = []byte(strings.Replace(string(projectYAML), "  - .markitect/model/manager.yaml\n", "  - .markitect/model/manager.yaml\n  - .markitect/model/root-statement.yaml\n", 1))
	writeE2E(t, root, projectvalue.ManifestPath, string(projectYAML))
	writeE2E(t, root, ".markitect/model/root-statement.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Statement\nmetadata:\n  name: integration-quality\n  namespace: \"\"\npurpose: Parent integration must produce a clean summary.\nspec:\n  category: concept\n  description: Parent-owned integration files contain the corrected summary.\n")
	gitE2E(t, root, "add", projectvalue.ManifestPath, ".markitect/model/root-statement.yaml")
	gitE2E(t, root, "commit", "--amend", "--no-edit")
	setupE2EProcess(t, "integration-review-fix")
	enableE2EReviews(t, root, 100000)
	host := processRuntimeHost(root, processRuntimeProjectWithRootStatement(t, root, identityHead(t, root)))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement and reconcile the two owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("integration reviewer findings did not lead to corrected closure: status=%s err=%v", run.Status, err)
	}
	var rootReviews []ReviewRecord
	for _, review := range run.Reviews {
		if review.ManagerID == e2eManagerID("", "project-owner") && review.Phase == "integrate" {
			rootReviews = append(rootReviews, review)
		}
	}
	if len(rootReviews) != 2 || rootReviews[0].Outcome != "fail" || rootReviews[1].Outcome != "pass" {
		t.Fatalf("expected failed and fresh passed root integration reviews, got %+v", rootReviews)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(candidate.Files["src/project-integration.txt"].Content); got != "corrected parent integration\n" {
		t.Fatalf("final candidate retained reviewer finding: %q", got)
	}
	if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("", "project-owner"), "integrate"); calls != 2 {
		t.Fatalf("parent integration ran %d times, want initial plus bounded correction", calls)
	}
}

func TestExhaustedIntegrationReviewBlocksBeforeInvokingManager(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	projectPath := filepath.Join(root, filepath.FromSlash(projectvalue.ManifestPath))
	projectYAML, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	projectYAML = []byte(strings.Replace(string(projectYAML), "  - .markitect/model/manager.yaml\n", "  - .markitect/model/manager.yaml\n  - .markitect/model/root-statement.yaml\n", 1))
	writeE2E(t, root, projectvalue.ManifestPath, string(projectYAML))
	writeE2E(t, root, ".markitect/model/root-statement.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Statement\nmetadata:\n  name: integration-quality\n  namespace: \"\"\npurpose: Parent integration must produce a clean summary.\nspec:\n  category: concept\n  description: Parent-owned integration files contain the corrected summary.\n")
	gitE2E(t, root, "add", projectvalue.ManifestPath, ".markitect/model/root-statement.yaml")
	gitE2E(t, root, "commit", "--amend", "--no-edit")
	setupE2EProcess(t, "integration-review-fix")
	enableE2EReviews(t, root, 100000)
	updateE2ERuntime(t, root, func(config *projectconfig.Runtime) { config.Review.MaxRounds = 1 })
	host := processRuntimeHost(root, processRuntimeProjectWithRootStatement(t, root, identityHead(t, root)))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement and reconcile the two owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err == nil || run.Status != StatusBlocked || !strings.Contains(err.Error(), "exhausted the cumulative integration review round limit") {
		t.Fatalf("exhausted integration review did not block on its round limit: status=%s err=%v", run.Status, err)
	}
	rootID := e2eManagerID("", "project-owner")
	if got := reviewCount(run.Reviews, rootID, "integrate"); got != 1 {
		t.Fatalf("root integration reviews=%d, want exactly the single allowed round", got)
	}
	if calls := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), rootID, "integrate"); calls != 1 {
		t.Fatalf("root integration Manager ran %d times with MaxRounds=1; a candidate after the last failed review can never be reviewed", calls)
	}
}

func TestIntegrationReviewRoutesChildReworkBeforeFreshParentReview(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	projectPath := filepath.Join(root, filepath.FromSlash(projectvalue.ManifestPath))
	projectYAML, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	projectYAML = []byte(strings.Replace(string(projectYAML), "  - .markitect/model/manager.yaml\n", "  - .markitect/model/manager.yaml\n  - .markitect/model/root-statement.yaml\n", 1))
	writeE2E(t, root, projectvalue.ManifestPath, string(projectYAML))
	writeE2E(t, root, ".markitect/model/root-statement.yaml", "apiVersion: "+projectmodel.APIVersion+"\nkind: Statement\nmetadata:\n  name: integration-quality\n  namespace: \"\"\npurpose: Parent integration must preserve corrected child artifacts.\nspec:\n  category: concept\n  description: Integration candidate must contain corrected child artifacts.\n")
	gitE2E(t, root, "add", projectvalue.ManifestPath, ".markitect/model/root-statement.yaml")
	gitE2E(t, root, "commit", "--amend", "--no-edit")
	setupE2EProcess(t, "integration-review-child-rework")
	updateE2ERuntime(t, root, func(config *projectconfig.Runtime) {
		config.Review = &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{}, MaxRounds: 3, MaxManagerRounds: 2}
		for id, agent := range config.Agents {
			config.Review.Agents[id] = agent
		}
		config.Limits.MaxStarts = 32
	})
	host := processRuntimeHost(root, processRuntimeProjectWithRootStatement(t, root, identityHead(t, root)))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and reconcile the result.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("review-directed child correction did not close: status=%s err=%v", run.Status, err)
	}
	rootID, ordersID, inventoryID := e2eManagerID("", "project-owner"), e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")
	if got := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), rootID, "integrate"); got != 3 {
		t.Fatalf("root integrated %d times, want initial, request-bearing defer, and post-child reintegration", got)
	}
	if got := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), rootID, "review"); got != 2 {
		t.Fatalf("root reviewed %d candidates, want failed original and fresh post-rework candidate", got)
	}
	if got := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), ordersID, "work"); got != 2 {
		t.Fatalf("requested orders child ran %d times, want initial plus targeted repair", got)
	}
	if got := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), inventoryID, "work"); got != 1 {
		t.Fatalf("unrequested inventory child ran %d times, want one initial run", got)
	}
	var rootReviews []ReviewRecord
	for _, review := range run.Reviews {
		if review.ManagerID == rootID && review.Phase == "integrate" {
			rootReviews = append(rootReviews, review)
		}
	}
	if len(rootReviews) != 2 || rootReviews[0].Outcome != "fail" || rootReviews[1].Outcome != "pass" || rootReviews[0].CandidateID == rootReviews[1].CandidateID {
		t.Fatalf("expected retained failure followed by fresh post-rework pass, got %+v", rootReviews)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	finalCandidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(finalCandidate.Files["src/project-integration.txt"].Content); got != "parent integration edit survives child rework\n" {
		t.Fatalf("child correction discarded parent-owned integration edit: %q", got)
	}
	if got := string(finalCandidate.Files["src/orders/implementation.txt"].Content); strings.Contains(got, "DEFECT") {
		t.Fatalf("final candidate retained the reviewed child defect: %q", got)
	}
	records, err := readE2ERecords(runtimeFixtureEnv(t, e2eLogEnv))
	if err != nil {
		t.Fatal(err)
	}
	firstFailedRootReview, secondOrdersWork, thirdRootIntegration, finalRootReview := -1, -1, -1, -1
	rootReviewsSeen, ordersWorkSeen, rootIntegrationsSeen := 0, 0, 0
	for i, record := range records {
		manager, _ := record["managerId"].(string)
		phase, _ := record["phase"].(string)
		switch {
		case manager == rootID && phase == "review":
			rootReviewsSeen++
			if rootReviewsSeen == 1 {
				firstFailedRootReview = i
			} else if rootReviewsSeen == 2 {
				finalRootReview = i
			}
		case manager == ordersID && phase == "work":
			ordersWorkSeen++
			if ordersWorkSeen == 2 {
				secondOrdersWork = i
			}
		case manager == rootID && phase == "integrate":
			rootIntegrationsSeen++
			if rootIntegrationsSeen == 3 {
				thirdRootIntegration = i
			}
		}
	}
	if !(firstFailedRootReview >= 0 && firstFailedRootReview < secondOrdersWork && secondOrdersWork < thirdRootIntegration && thirdRootIntegration < finalRootReview) {
		t.Fatalf("child correction/fresh reintegration did not precede final parent review: failed=%d child=%d reintegrate=%d final-review=%d", firstFailedRootReview, secondOrdersWork, thirdRootIntegration, finalRootReview)
	}
}

func TestNestedReworkDefersEveryAncestorIntegrationAndReview(t *testing.T) {
	t.Parallel()
	root := makeMultilevelObligationFixture(t)
	setupE2EProcess(t, "nested-child-rework")
	host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), true))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement nested Commerce behavior and integrate all owned artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory"), e2eManagerID("orders.commerce", "commerce")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("nested child rework did not close: status=%s err=%v", run.Status, err)
	}
	rootID, ordersID, commerceID := e2eManagerID("", "project-owner"), e2eManagerID("orders", "orders"), e2eManagerID("orders.commerce", "commerce")
	if len(run.ManagerReworkRounds) != 2 || run.ManagerReworkRounds[0].Status != "completed" || run.ManagerReworkRounds[1].Status != "completed" {
		t.Fatalf("nested requests did not use two completed bounded rework rounds: %+v", run.ManagerReworkRounds)
	}
	if got := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), commerceID, "work"); got != 3 {
		t.Fatalf("nested child work ran %d times, want initial plus two bounded targeted repairs", got)
	}
	if got := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), ordersID, "integrate"); got != 3 {
		t.Fatalf("requesting parent integrated %d times, want initial request, second request, and fresh integration", got)
	}
	if got := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), rootID, "integrate"); got != 1 {
		t.Fatalf("root integrated %d times, want only the post-rework candidate", got)
	}
	if got := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), rootID, "review"); got != 1 {
		t.Fatalf("root reviewed %d times, want only after nested rework", got)
	}
	if got := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("inventory", "inventory"), "work"); got != 1 {
		t.Fatalf("unrequested sibling work ran %d times, want one initial run", got)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	finalCandidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(finalCandidate.Files["src/project-integration.txt"].Content); got != "root integration after nested rework\n" {
		t.Fatalf("nested child rework discarded root-owned integration edit: %q", got)
	}
	records, err := readE2ERecords(runtimeFixtureEnv(t, e2eLogEnv))
	if err != nil {
		t.Fatal(err)
	}
	finalCommerceWork, finalOrdersIntegration, rootIntegration, rootReview := -1, -1, -1, -1
	commerceWorkSeen, ordersIntegrationSeen := 0, 0
	for i, record := range records {
		manager, _ := record["managerId"].(string)
		phase, _ := record["phase"].(string)
		switch {
		case manager == commerceID && phase == "work":
			commerceWorkSeen++
			if commerceWorkSeen == 3 {
				finalCommerceWork = i
			}
		case manager == ordersID && phase == "integrate":
			ordersIntegrationSeen++
			if ordersIntegrationSeen == 3 {
				finalOrdersIntegration = i
			}
		case manager == rootID && phase == "integrate":
			rootIntegration = i
		case manager == rootID && phase == "review":
			rootReview = i
		}
	}
	if !(finalCommerceWork >= 0 && finalCommerceWork < finalOrdersIntegration && finalOrdersIntegration < rootIntegration && rootIntegration < rootReview) {
		t.Fatalf("ancestor ran before both nested child rework rounds settled: child=%d parent=%d root-integration=%d root-review=%d", finalCommerceWork, finalOrdersIntegration, rootIntegration, rootReview)
	}
}

func TestReviewerContractFailuresAndUncertainInvocationNeverReplay(t *testing.T) {
	t.Parallel()
	for _, behavior := range []string{"review-proposes-write", "review-ungrounded", "review-incomplete"} {
		t.Run(behavior, func(t *testing.T) {
			t.Parallel()
			root := makeProjectRunFixture(t)
			setupE2EProcess(t, behavior)
			enableE2EReviews(t, root, 100000)
			host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false))
			plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both fixture artifacts.",
				Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
			if err != nil {
				t.Fatal(err)
			}
			run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
			if err == nil || run.Status != StatusFailed {
				t.Fatalf("invalid or uncertain review was accepted: status=%s err=%v", run.Status, err)
			}
			if len(run.Invocations) == 0 || run.Invocations[len(run.Invocations)-1].Role != "reviewer" {
				t.Fatalf("failed reviewer invocation was not retained: %+v", run.Invocations)
			}
			before := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("orders", "orders"), "review") + countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("", "project-owner"), "review")
			_, resumeErr := Resume(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
			if resumeErr == nil {
				t.Fatal("uncertain review invocation was automatically replayed")
			}
			after := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("orders", "orders"), "review") + countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), e2eManagerID("", "project-owner"), "review")
			if after != before {
				t.Fatalf("Resume replayed reviewer process: before=%d after=%d", before, after)
			}
		})
	}
}

func TestReviewerCostOverrunBlocksBeforeAcceptingReview(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "review-over-budget")
	enableE2EReviews(t, root, 1)
	host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both fixture artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err == nil || run.Status != StatusBlocked || len(run.Invocations) == 0 || run.Invocations[len(run.Invocations)-1].CostMicros <= 1 {
		t.Fatalf("review cost overrun was not durably blocked: status=%s last=%+v err=%v", run.Status, lastInvocation(run.Invocations), err)
	}
}

func TestReviewerWorkingTreeDriftRetainsUncertainCallWithoutReplay(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "review-working-drift")
	setRuntimeFixtureEnv(t, e2eRootEnv, root)
	enableE2EReviews(t, root, 100000)
	host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false))
	orders := e2eManagerID("orders", "orders")
	baseLoad := host.Load
	logPath := runtimeFixtureEnv(t, e2eLogEnv)
	host.Load = func(loadRoot, revision string) (*projectvalue.Project, error) {
		project, err := baseLoad(loadRoot, revision)
		if err != nil {
			return nil, err
		}
		if countE2EProcessCalls(logPath, orders, "review") > 0 {
			project.Snapshot.Files["src/orders/implementation.txt"] = []byte("changed during reviewer invocation\n")
		}
		return project, nil
	}
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement both fixture artifacts.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	run, runErr := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if runErr == nil || run.Status != StatusFailed {
		t.Fatalf("reviewer changed the working snapshot but its review was consumed: status=%s err=%v", run.Status, runErr)
	}
	task := findTask(run.Tasks, orders)
	if task == nil || task.ReviewStatus != "uncertain" {
		t.Fatalf("post-review binding failure was not retained as uncertain: %+v", task)
	}
	if len(run.Invocations) == 0 || run.Invocations[len(run.Invocations)-1].Role != "reviewer" {
		t.Fatalf("reviewer call receipt was not retained after binding drift: %+v", run.Invocations)
	}
	before := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), orders, "review")
	if _, err := Resume(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID); err == nil {
		t.Fatal("uncertain reviewer invocation was resumed automatically")
	}
	if after := countE2EProcessCalls(runtimeFixtureEnv(t, e2eLogEnv), orders, "review"); after != before {
		t.Fatalf("Resume replayed reviewer invocation after binding drift: before=%d after=%d", before, after)
	}
}

func TestProjectRunProcessRejectsStaleNonceOutOfScopeAndFailedIntegration(t *testing.T) {
	t.Parallel()
	for _, behavior := range []string{"stale-nonce", "out-of-scope", "failed-integration"} {
		t.Run(behavior, func(t *testing.T) {
			t.Parallel()
			root := makeProjectRunFixture(t)
			setupE2EProcess(t, behavior)
			plan, err := Plan(processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false)), root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and integrate them.",
				Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			run, runErr := Run(context.Background(), processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false)), projectexecutor.ProcessInvoker{}, root, plan.ID)
			if runErr == nil || run.Status == StatusIntegrated || run.Status == StatusVerified || run.Status == StatusApplied {
				t.Fatalf("behavior %s unexpectedly closed successfully: status=%s err=%v", behavior, run.Status, runErr)
			}
		})
	}
}

func TestProjectRunRepairsKnownValidationFailuresInWorkAndIntegration(t *testing.T) {
	t.Parallel()
	root := makeProjectRunFixture(t)
	updateE2ERuntime(t, root, func(config *projectconfig.Runtime) {
		config.Limits.MaxRetries = 1
		config.Limits.MaxDuration += projectconfig.Duration(time.Second)
	})
	setupE2EProcess(t, "repair-foreign-first")
	logPath := runtimeFixtureEnv(t, e2eLogEnv)
	plan, err := Plan(processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false)), root, identityHead(t, root), PlanRequest{Goal: "Implement both owned artifacts and integrate them.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := Run(context.Background(), processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false)), projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil {
		t.Fatalf("Run did not repair known validation failures: status=%s err=%v", run.Status, err)
	}
	if run.Status != StatusIntegrated || len(run.Invocations) != 6 {
		t.Fatalf("repair did not finish a fully integrated run: status=%s invocations=%d", run.Status, len(run.Invocations))
	}
	orders := findTask(run.Tasks, e2eManagerID("orders", "orders"))
	rootTask := findTask(run.Tasks, e2eManagerID("", "project-owner"))
	if orders == nil || orders.WorkAttempts != 2 || orders.Attempts != 2 || rootTask == nil || rootTask.IntegrationAttempts != 2 || rootTask.Attempts != 3 {
		t.Fatalf("attempt counts were not retained: orders=%+v root=%+v", orders, rootTask)
	}
	if err := assertRepairTrace(t, logPath, orders.ManagerID, "work"); err != nil {
		t.Fatal(err)
	}
	if err := assertRepairTrace(t, logPath, rootTask.ManagerID, "integrate"); err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := store.readLatestState(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != StatusIntegrated || findTask(persisted.Tasks, orders.ManagerID).WorkAttempts != 2 || findTask(persisted.Tasks, rootTask.ManagerID).IntegrationAttempts != 2 {
		t.Fatalf("retry ledger was not durable in latest state: %+v", persisted.Tasks)
	}
}

func TestProjectRunPropagatesRootResolutionThroughForwardedObligationChain(t *testing.T) {
	t.Parallel()
	root := makeMultilevelObligationFixture(t)
	setupE2EProcess(t, "provenance-escalation")
	host := processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), true))
	plan, err := Plan(host, root, identityHead(t, root), PlanRequest{Goal: "Implement and verify nested Commerce cancellation behavior.",
		Managers: []string{e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory"), e2eManagerID("orders.commerce", "commerce")}, ExecuteAuthorized: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := Run(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("multilevel escalated run did not integrate after root resolution: status=%s err=%v tasks=%+v escalations=%+v", run.Status, err, run.Tasks, run.Escalations)
	}
	for _, managerID := range []string{e2eManagerID("orders.commerce", "commerce"), e2eManagerID("orders", "orders"), e2eManagerID("", "project-owner")} {
		task := findTask(run.Tasks, managerID)
		if task == nil || task.ReportStatus != "complete" || len(task.Questions)+len(task.Risks)+len(task.Obligations) != 0 {
			t.Fatalf("root resolution did not clear only the forwarded chain at %s: %+v", managerID, task)
		}
	}
	if len(run.Escalations) != 2 {
		t.Fatalf("expected the retained Commerce→Orders→root escalation chain: %+v", run.Escalations)
	}
	for _, escalation := range run.Escalations {
		if escalation.Status != "resolved" || len(escalation.ObligationIDs) != 2 {
			t.Fatalf("resolution history lacks typed obligation IDs or terminal status: %+v", escalation)
		}
	}
	for _, managerID := range []string{e2eManagerID("orders.commerce", "commerce"), e2eManagerID("orders", "orders"), e2eManagerID("inventory", "inventory")} {
		passed := false
		for _, review := range run.Reviews {
			if review.ManagerID == managerID && review.Outcome == "pass" {
				passed = true
				break
			}
		}
		if !passed {
			t.Fatalf("candidate review evidence for %s did not survive ancestor-only obligation resolution: %+v", managerID, run.Reviews)
		}
	}
	verified, err := Verify(context.Background(), host, projectexecutor.ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != StatusVerified {
		t.Fatalf("Verify rejected the reviewed candidate after root-only obligation resolution: status=%s err=%v", verified.Status, err)
	}
	preflight, err := PreflightApply(host, root, plan.ID, run.Candidate.ID)
	if err != nil {
		t.Fatalf("PreflightApply: %v", err)
	}
	if _, err := Apply(host, projectexecutor.ProcessInvoker{}, root, applyRequestFromPreflight(preflight)); err != nil {
		t.Fatalf("Apply rejected the freshly verified provenance-resolved candidate: %v", err)
	}
}

func TestProjectRunRetryCeilingAndCostAccountingAreDurable(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name        string
		retries     int
		maxCost     int64
		priceOrders int64
		wantStatus  string
	}{
		{name: "retry ceiling", retries: 0, maxCost: 100000, priceOrders: 1, wantStatus: StatusFailed},
		{name: "cost ceiling before retry", retries: 1, maxCost: 54, priceOrders: 1_000_000, wantStatus: StatusBlocked},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			root := makeProjectRunFixture(t)
			updateE2ERuntime(t, root, func(config *projectconfig.Runtime) {
				config.Limits.MaxRetries = scenario.retries
				config.Limits.MaxCostMicros = scenario.maxCost
				config.Limits.MaxDuration += projectconfig.Duration(time.Second)
				orders := e2eManagerID("orders", "orders")
				agent := config.Agents[orders]
				agent.Pricing = projectconfig.Pricing{InputMicrosPerMillion: scenario.priceOrders, OutputMicrosPerMillion: scenario.priceOrders}
				config.Agents[orders] = agent
			})
			setupE2EProcess(t, "repair-foreign-first")
			logPath := runtimeFixtureEnv(t, e2eLogEnv)
			plan, err := Plan(processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false)), root, identityHead(t, root), PlanRequest{Goal: "Implement the owned orders artifact.",
				Managers: []string{e2eManagerID("orders", "orders")}, ExecuteAuthorized: true})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			run, runErr := Run(context.Background(), processRuntimeHost(root, processRuntimeProject(t, root, identityHead(t, root), false)), projectexecutor.ProcessInvoker{}, root, plan.ID)
			if runErr == nil || run.Status != scenario.wantStatus {
				t.Fatalf("invalid proposal did not stop at the expected bounded state: status=%s err=%v", run.Status, runErr)
			}
			orders := findTask(run.Tasks, e2eManagerID("orders", "orders"))
			if orders == nil || orders.WorkAttempts != 1 || orders.Attempts != 1 || len(run.Invocations) != 2 {
				t.Fatalf("first failed attempt accounting missing: orders=%+v invocations=%d", orders, len(run.Invocations))
			}
			if got := countE2EProcessCalls(logPath, orders.ManagerID, "work"); got != 1 {
				t.Fatalf("runtime started %d orders attempts, want exactly one", got)
			}
			store, err := newRunStore(root)
			if err != nil {
				t.Fatal(err)
			}
			persisted, err := store.readLatestState(run.ID)
			if err != nil {
				t.Fatal(err)
			}
			persistedOrders := findTask(persisted.Tasks, orders.ManagerID)
			if persisted.Status != scenario.wantStatus || persistedOrders.WorkAttempts != 1 || persistedOrders.Attempts != 1 || strings.TrimSpace(persistedOrders.RepairDiagnostic) == "" {
				t.Fatalf("failed-attempt and retry diagnostic were not durably recorded: status=%s task=%+v", persisted.Status, persistedOrders)
			}
			if scenario.name == "cost ceiling before retry" && totalCost(persisted.Invocations) != scenario.maxCost {
				t.Fatalf("reported spend %d, want bound %d", totalCost(persisted.Invocations), scenario.maxCost)
			}
		})
	}
}
