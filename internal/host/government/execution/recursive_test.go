package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
	"go.yaml.in/yaml/v3"
)

var recursiveFixtureBuild struct {
	once sync.Once
	path string
	err  error
}

func g3FixtureRunner(t *testing.T) string {
	t.Helper()
	// Reuse the package's TestMain-owned temporary root so repeated focused G3
	// runs do not leave compiled actor binaries behind.
	_ = fixtureRunner(t)
	recursiveFixtureBuild.once.Do(func() {
		recursiveFixtureBuild.path = filepath.Join(fixtureBuild.dir, "runner-g3.exe")
		cmd := exec.Command("go", "build", "-o", recursiveFixtureBuild.path, "./runner")
		cmd.Dir = "../../../../examples/government-g3"
		if out, err := cmd.CombinedOutput(); err != nil {
			recursiveFixtureBuild.err = fmt.Errorf("build actual G3 process fixture: %w: %s", err, out)
		}
	})
	if recursiveFixtureBuild.err != nil {
		t.Fatal(recursiveFixtureBuild.err)
	}
	return recursiveFixtureBuild.path
}

func recursiveFixtureOptions(t *testing.T) Options {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo, state, temporary := filepath.Join(root, "repo"), filepath.Join(root, "state"), filepath.Join(root, "temporary")
	for _, dir := range []string{repo, state, temporary} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	fixture, err := filepath.Abs("../../../../examples/government-g3")
	if err != nil {
		t.Fatal(err)
	}
	if err := filepath.WalkDir(fixture, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(fixture, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(repo, rel)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string { return fixtureGit(t, repo, args...) }
	git("init", "-b", "feature")
	git("config", "user.name", "G3 mechanics test")
	git("config", "user.email", "g3@example.invalid")
	git("config", "core.autocrlf", "false")
	git("add", ".")
	git("-c", "commit.gpgsign=false", "commit", "-m", "prior active G3 fixture")
	base := git("rev-parse", "HEAD")
	activeRef := "refs/markitect/government/active/example"
	git("update-ref", activeRef, base)
	runner := g3FixtureRunner(t)
	parallelToken := "g3-" + strings.ReplaceAll(filepath.Base(filepath.Dir(root)), "_", "-")
	runnerFor := func(slot, mode string) RunnerSpec {
		args := []string{mode}
		if mode == "quantity" || mode == "price" {
			args = append(args, parallelToken)
		}
		return RunnerSpec{SlotID: slot, Command: runner, Args: args, Model: "deterministic-g3-mechanics-fixture", ProviderVersion: "fixture-g3-v1", TimeoutSeconds: 30, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20}
	}
	rootCheck := authoring.Check{Name: "invoice-composition", Run: []string{"go", "test", "./...", "-count=1"}, TimeoutSeconds: intPointer(120)}
	quantityCheck := authoring.Check{Name: "quantity-local", Run: []string{"go", "test", "./quantity", "-count=1"}, TimeoutSeconds: intPointer(120)}
	priceCheck := authoring.Check{Name: "price-local", Run: []string{"go", "test", "./price", "-count=1"}, TimeoutSeconds: intPointer(120)}
	quantity := g3AreaIdentity("quantity")
	price := g3AreaIdentity("price")
	runtimeConfig := Runtime{
		APIVersion: RuntimeVersion, ActiveRef: activeRef, ExpectedBase: base, TimeoutSeconds: 1800,
		StateDirectory: state, TemporaryDirectory: temporary,
		Executor: runnerFor("root-writer", "root"), Verifier: runnerFor("root-reviewer", "root"),
		Checks: []authoring.Check{rootCheck},
		Ressorts: []RessortRunner{
			{Ressort: core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Ressort", Namespace: "invoice", Name: "arithmetic"}, Runner: runnerFor("ressort-arithmetic", "assent")},
			{Ressort: core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Ressort", Namespace: "invoice", Name: "unaffected"}, Runner: runnerFor("ressort-unaffected", "assent-unaffected")},
		},
		Recursion: &RecursiveRuntime{
			Limits: government.DelegationLimits{MaxDepth: 2, MaxFanout: 2, MaxCalls: 32}, Parallelism: 2, MaxRepairs: 1,
			Areas: []AreaRunner{
				{Area: quantity, Executor: runnerFor("quantity-writer", "quantity"), Verifier: runnerFor("quantity-reviewer", "quantity"), Checks: []authoring.Check{quantityCheck}},
				{Area: price, Executor: runnerFor("price-writer", "price"), Verifier: runnerFor("price-reviewer", "price"), Checks: []authoring.Check{priceCheck}},
			},
		},
	}
	return Options{Repo: repo, ConfigPath: "government.yaml", OrderPath: "order.yaml", Runtime: runtimeConfig}
}

func configureG3ThreeLevelFixture(t *testing.T, repo, activeRef string) string {
	t.Helper()
	configPath := filepath.Join(repo, "government.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var source government.Source
	if err := government.Decode(data, &source); err != nil {
		t.Fatal(err)
	}
	bridge := g3AreaIdentity("bridge")
	root := g3AreaIdentity("root")
	rootMandate := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Mandate", Namespace: "invoice", Name: "root-prior"}
	bridgeMandate := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Mandate", Namespace: "invoice", Name: "bridge-delegation"}
	quantitySubject := core.DefinitionIdentity{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "invoice", Name: "quantity"}
	totalSubject := core.DefinitionIdentity{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "invoice", Name: "composed-total"}
	for i := range source.Definitions {
		definition := &source.Definitions[i]
		switch definition.Kind + "/" + definition.Metadata.Name {
		case "Area/quantity":
			definition.Spec["parent"] = bridge
		case "Area/price":
			definition.Spec["parent"] = bridge
		case "Mandate/quantity-delegation":
			definition.Spec["parent"] = bridgeMandate
		case "Mandate/price-delegation":
			definition.Spec["parent"] = bridgeMandate
		case "Responsibility/integration-owner":
			definition.Spec["area"] = bridge
		case "Artifact/invoice-total":
			definition.Spec["writer"] = bridge
		}
	}
	source.Definitions = append(source.Definitions,
		core.Definition{APIVersion: government.APIVersion, Kind: "Area", Metadata: core.Metadata{Namespace: "invoice", Name: "bridge"}, Purpose: "Recursive parent that integrates two independently delegated invoice values.", Spec: map[string]any{"parent": root, "capabilities": []string{}}},
		core.Definition{APIVersion: government.APIVersion, Kind: "Mandate", Metadata: core.Metadata{Namespace: "invoice", Name: "bridge-delegation"}, Purpose: "Prior authority for the bridge Area to implement the total and delegate both value children.", Spec: map[string]any{"area": bridge, "parent": rootMandate, "scope": []core.DefinitionIdentity{quantitySubject, {APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "invoice", Name: "unit-price"}, totalSubject}, "actions": []string{"implement", "review"}}},
	)
	updatedConfig, err := yaml.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, updatedConfig, 0644); err != nil {
		t.Fatal(err)
	}
	var normalizedSource government.Source
	if err := government.Decode(updatedConfig, &normalizedSource); err != nil {
		t.Fatal(err)
	}
	model := government.Compile(normalizedSource)
	if len(model.Findings) > 0 {
		t.Fatalf("three-level prior Government fixture is invalid: %+v", model.Findings)
	}
	orderPath := filepath.Join(repo, "order.yaml")
	orderBytes, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	var order government.Order
	if err := government.Decode(orderBytes, &order); err != nil {
		t.Fatal(err)
	}
	order.ActiveConstitution = model.Digest
	updatedOrder, err := json.Marshal(order)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orderPath, updatedOrder, 0644); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, repo, "add", "government.yaml", "order.yaml")
	fixtureGit(t, repo, "-c", "commit.gpgsign=false", "commit", "-m", "add recursive structural bridge")
	base := fixtureGit(t, repo, "rev-parse", "HEAD")
	fixtureGit(t, repo, "update-ref", activeRef, base)
	return base
}

func TestRecursiveRunIntegratesChildrenRejectsCompositionRepairsAndPromotes(t *testing.T) {
	opts := recursiveFixtureOptions(t)
	before, err := os.ReadFile(filepath.Join(opts.Repo, "integration", "invoice.txt"))
	if err != nil {
		t.Fatal(err)
	}
	base := opts.Runtime.ExpectedBase
	report, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("recursive G3 run failed at %s: %v; actor logs: %s", report.Stage, err, recursiveActorLogs(report))
	}
	if report.Status != "accepted-scoped" || report.Stage != "complete" || report.Promotion == nil || report.Promotion.Status != "promoted" {
		t.Fatalf("final composed candidate was not promoted: %s/%s %+v", report.Status, report.Stage, report.Promotion)
	}
	if report.RootArea == nil || len(report.RootArea.Attempts) != 2 {
		t.Fatalf("expected rejected root candidate and one bounded repair: %+v", report.RootArea)
	}
	first, repaired := report.RootArea.Attempts[0], report.RootArea.Attempts[1]
	if first.Status != "blocked" || repaired.Status != "passed-scoped" || first.CandidateID == repaired.CandidateID {
		t.Fatalf("root did not reject then freshly review repaired material: first=%+v repaired=%+v", first, repaired)
	}
	if first.Review == nil || first.Review.Outcome != agentexec.OutcomeFailed || repaired.Review == nil || repaired.Review.Outcome != agentexec.OutcomePassed {
		t.Fatalf("parent reviewer did not reject the first assembled candidate and pass the repair: first=%+v repaired=%+v", first.Review, repaired.Review)
	}
	if len(first.Children) != 2 || len(repaired.Children) != 2 {
		t.Fatalf("parent reports must retain both child lineages for both attempts")
	}
	childExecutions := map[string]ActorRecord{}
	for _, actor := range report.Actors {
		if actor.Phase == "execute" && (actor.SlotID == "quantity-writer" || actor.SlotID == "price-writer") {
			childExecutions[actor.SlotID] = actor
		}
	}
	if len(childExecutions) != 2 {
		t.Fatalf("expected actual execution receipts from two independent children, got %v", childExecutions)
	}
	quantityRun, priceRun := childExecutions["quantity-writer"], childExecutions["price-writer"]
	if quantityRun.Result.Receipt.RunID == "" || priceRun.Result.Receipt.RunID == "" || quantityRun.Result.Receipt.RunID == priceRun.Result.Receipt.RunID {
		t.Fatal("child invocations lack distinct actual process receipts")
	}
	assertG3ParallelEvents(t, filepath.Dir(report.ReportPath), opts.Runtime.Recursion.Areas[0].Executor.Args[1], quantityRun, priceRun)
	for _, tc := range []struct {
		run     ActorRecord
		area    string
		subject string
		path    string
	}{
		{quantityRun, "quantity", "quantity", "quantity/quantity.txt"},
		{priceRun, "price", "unit-price", "price/price.txt"},
	} {
		wantScope := core.DefinitionIdentity{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "invoice", Name: tc.subject}.Key()
		if len(tc.run.Scopes) != 1 || tc.run.Scopes[0] != wantScope || len(tc.run.Result.Response.CandidateFiles) != 1 || tc.run.Result.Response.CandidateFiles[0].Path != tc.path {
			t.Fatalf("%s executor received or wrote beyond its frozen local responsibility: %+v", tc.area, tc.run)
		}
	}
	if quantityRun.StartedAt.After(priceRun.FinishedAt) || priceRun.StartedAt.After(quantityRun.FinishedAt) {
		t.Fatalf("independent child process intervals did not overlap: quantity=%s..%s price=%s..%s", quantityRun.StartedAt, quantityRun.FinishedAt, priceRun.StartedAt, priceRun.FinishedAt)
	}
	for _, attempt := range []AreaAttempt{first, repaired} {
		if len(attempt.Children) != 2 || attempt.Children[0].Status != "passed-scoped" || attempt.Children[1].Status != "passed-scoped" {
			t.Fatalf("green child reports were not preserved: %+v", attempt.Children)
		}
		workspaces := map[string]bool{}
		for _, child := range attempt.Children {
			childAttempt := child.Attempts[0]
			if childAttempt.Workspace == "" || workspaces[filepath.Clean(childAttempt.Workspace)] {
				t.Fatalf("child Areas did not receive separate workspaces: %+v", attempt.Children)
			}
			workspaces[filepath.Clean(childAttempt.Workspace)] = true
		}
	}
	for _, tc := range []struct {
		slot, area, subject, sibling string
	}{
		{"quantity-reviewer", "quantity", "quantity", "unit-price"},
		{"price-reviewer", "price", "unit-price", "quantity"},
	} {
		var review *ActorRecord
		for i := range report.Actors {
			if report.Actors[i].SlotID == tc.slot {
				review = &report.Actors[i]
				break
			}
		}
		areaKey := g3AreaIdentity(tc.area).Key()
		subjectKey := core.DefinitionIdentity{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "invoice", Name: tc.subject}.Key()
		siblingKey := core.DefinitionIdentity{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "invoice", Name: tc.sibling}.Key()
		if review == nil || len(review.Scopes) != 2 || !containsString(review.Scopes, areaKey) || !containsString(review.Scopes, subjectKey) || containsString(review.Scopes, siblingKey) {
			t.Fatalf("%s reviewer received another child's mandate scope: %+v", tc.area, review)
		}
	}
	if len(first.Checks) == 0 || first.Checks[0].ExitCode == 0 || len(repaired.Checks) == 0 || repaired.Checks[0].ExitCode != 0 {
		t.Fatalf("fresh parent composition checks did not reject then pass: first=%+v repaired=%+v", first.Checks, repaired.Checks)
	}
	var firstRootReview *ActorRecord
	for i := range report.Actors {
		if report.Actors[i].Result.Receipt.RunID == first.ReviewerRunID {
			firstRootReview = &report.Actors[i]
			break
		}
	}
	if first.Review.InputDigest == "" || firstRootReview == nil || first.Review.InputDigest != firstRootReview.Result.Receipt.InputDigest {
		t.Fatal("first root review response was not tied to its actual invocation")
	}
	if len(report.Actors) == 0 || len(report.Votes) != 2 || report.Votes[1].Outcome != government.VoteAssentUnaffected {
		t.Fatalf("final root candidate lacks root actors or both explicit Ressort votes: votes=%+v", report.Votes)
	}
	if report.Candidate == nil || report.Evidence == nil || report.Decision == nil || report.Evidence.MaterialCandidateID != report.Candidate.ID || report.Decision.EvidenceID != report.Evidence.ID {
		t.Fatal("final root evidence and decision are not bound to the repaired candidate")
	}
	for _, vote := range report.Votes {
		if vote.MaterialCandidateID != report.Candidate.ID || vote.EvidenceID != report.Evidence.ID {
			t.Fatalf("a vote was carried over from an earlier intermediate candidate: %+v", vote)
		}
	}
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != report.CandidateCommit {
		t.Fatalf("Active did not advance directly to the final root candidate: %s", active)
	}
	if head := fixtureGit(t, opts.Repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("user checkout HEAD moved from the frozen base: %s", head)
	}
	final := fixtureGit(t, opts.Repo, "show", report.CandidateCommit+":integration/invoice.txt")
	if strings.TrimSpace(final) != "total=12" {
		t.Fatalf("promoted root candidate does not integrate actual child values: %q", final)
	}
	after, err := os.ReadFile(filepath.Join(opts.Repo, "integration", "invoice.txt"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("native recursive run changed the user's checkout files")
	}
	assertRetainedReport(t, report)
}

func recursiveActorLogs(report Report) string {
	var output strings.Builder
	for _, actor := range report.Actors {
		path := filepath.Join(filepath.Dir(report.ReportPath), actor.Result.Receipt.RunID+".jsonl")
		data, err := os.ReadFile(path)
		fmt.Fprintf(&output, "\n%s/%s run=%s error=%q log=%s readError=%v", actor.Phase, actor.SlotID, actor.Result.Receipt.RunID, actor.Error, data, err)
	}
	return output.String()
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertG3ParallelEvents(t *testing.T, stateDirectory, token string, quantity, price ActorRecord) {
	t.Helper()
	type processEvent struct {
		Event       string `json:"event"`
		Token       string `json:"token"`
		Generation  string `json:"generation"`
		Attempt     int    `json:"attempt"`
		Side        string `json:"side"`
		PID         int    `json:"pid"`
		RunID       string `json:"runId"`
		InputDigest string `json:"inputDigest"`
		AtUTC       string `json:"atUtc"`
	}
	read := func(actor ActorRecord, side string) []processEvent {
		path := filepath.Join(stateDirectory, actor.Result.Receipt.RunID+".jsonl")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("actual %s process event log missing: %v", side, err)
		}
		if government.BytesDigest(data) != actor.Result.Receipt.PrivateLogDigest {
			t.Fatalf("actual %s process events are not bound by the private-log receipt digest", side)
		}
		events := []processEvent{}
		for _, line := range bytes.Split(bytes.TrimSpace(data), []byte{'\n'}) {
			var event processEvent
			if err := json.Unmarshal(line, &event); err != nil {
				t.Fatal(err)
			}
			if event.Token != token || event.Side != side || event.RunID != actor.Result.Receipt.RunID || event.InputDigest != actor.Result.Receipt.InputDigest || event.PID == 0 {
				t.Fatalf("%s process event is not bound to its actor receipt: %+v", side, event)
			}
			events = append(events, event)
		}
		return events
	}
	qEvents := read(quantity, "quantity")
	pEvents := read(price, "price")
	byName := func(events []processEvent, name string) processEvent {
		for _, event := range events {
			if event.Event == name {
				return event
			}
		}
		t.Fatalf("process log omitted %q event: %+v", name, events)
		return processEvent{}
	}
	qStartEvent, qOverlapEvent, qFinishEvent := byName(qEvents, "process-start"), byName(qEvents, "sibling-process-overlap"), byName(qEvents, "process-finish")
	pStartEvent, pOverlapEvent, pFinishEvent := byName(pEvents, "process-start"), byName(pEvents, "sibling-process-overlap"), byName(pEvents, "process-finish")
	if qStartEvent.PID == pStartEvent.PID || qStartEvent.Generation != pStartEvent.Generation || qStartEvent.Attempt != pStartEvent.Attempt {
		t.Fatalf("child start events do not prove one parallel generation: quantity=%+v price=%+v", qStartEvent, pStartEvent)
	}
	qs, err := time.Parse(time.RFC3339Nano, qStartEvent.AtUTC)
	if err != nil {
		t.Fatal(err)
	}
	ps, err := time.Parse(time.RFC3339Nano, pStartEvent.AtUTC)
	if err != nil {
		t.Fatal(err)
	}
	qf, err := time.Parse(time.RFC3339Nano, qFinishEvent.AtUTC)
	if err != nil {
		t.Fatal(err)
	}
	pf, err := time.Parse(time.RFC3339Nano, pFinishEvent.AtUTC)
	if err != nil {
		t.Fatal(err)
	}
	if qs.After(pf) || ps.After(qf) || qOverlapEvent.AtUTC == "" || pOverlapEvent.AtUTC == "" || qOverlapEvent.Generation != qStartEvent.Generation || pOverlapEvent.Generation != qStartEvent.Generation {
		t.Fatalf("actual child process lifetimes did not overlap: quantity=%s..%s price=%s..%s", qs, qf, ps, pf)
	}
}

func g3AreaIdentity(name string) core.DefinitionIdentity {
	return core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Area", Namespace: "invoice", Name: name}
}

func TestRecursiveRunWithoutRepairBlocksBadRootDespiteGreenChildren(t *testing.T) {
	opts := recursiveFixtureOptions(t)
	opts.Runtime.Recursion.MaxRepairs = 0
	report, err := Run(context.Background(), opts)
	if err == nil || report.Status != "blocked" || report.RootArea == nil || report.Promotion != nil {
		t.Fatalf("bad composition was accepted without repair: %s/%s %+v: %v", report.Status, report.Stage, report.Promotion, err)
	}
	if len(report.RootArea.Attempts) != 1 || report.RootArea.Attempts[0].Review == nil || report.RootArea.Attempts[0].Review.Outcome != agentexec.OutcomeFailed {
		t.Fatalf("root rejection was not retained despite successful child reports: %+v", report.RootArea)
	}
	for _, child := range report.RootArea.Attempts[0].Children {
		if child.Status != "passed-scoped" {
			t.Fatalf("expected green child, got %+v", child)
		}
	}
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != opts.Runtime.ExpectedBase {
		t.Fatal("bad root composition advanced Active")
	}
	assertRetainedReport(t, report)
}

func TestRecursiveRunRejectsChildWriterScopeViolation(t *testing.T) {
	opts := recursiveFixtureOptions(t)
	for i := range opts.Runtime.Recursion.Areas {
		if opts.Runtime.Recursion.Areas[i].Area.Key() == g3AreaIdentity("quantity").Key() {
			opts.Runtime.Recursion.Areas[i].Executor.Args[0] = "outside-scope"
		}
	}
	base := opts.Runtime.ExpectedBase
	report, err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "outside frozen Writer scope") || report.Promotion != nil || report.RootArea == nil {
		t.Fatalf("child proposal outside its exact writer scope was not rejected: %s/%s %+v: %v", report.Status, report.Stage, report.RootArea, err)
	}
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != base {
		t.Fatal("out-of-scope child material advanced Active")
	}
	if head := fixtureGit(t, opts.Repo, "rev-parse", "HEAD"); head != base {
		t.Fatal("out-of-scope child material moved user checkout HEAD")
	}
	assertRetainedReport(t, report)
}

func TestRecursiveRunReviewsStructuralThreeLevelAreaAndRootCandidate(t *testing.T) {
	opts := recursiveFixtureOptions(t)
	opts.Runtime.ExpectedBase = configureG3ThreeLevelFixture(t, opts.Repo, opts.Runtime.ActiveRef)
	bridge := g3AreaIdentity("bridge")
	quantity := g3AreaIdentity("quantity")
	price := g3AreaIdentity("price")
	base := opts.Runtime.ExpectedBase
	bridgeCheck := authoring.Check{Name: "bridge-composition", Run: []string{"go", "test", "./...", "-count=1"}, TimeoutSeconds: intPointer(120)}
	opts.Runtime.Checks = []authoring.Check{{Name: "invoice-composition", Run: []string{"go", "test", "./...", "-count=1"}, TimeoutSeconds: intPointer(120)}}
	opts.Runtime.Recursion.Limits.MaxDepth = 3
	bridgeExecutor := opts.Runtime.Recursion.Areas[0].Executor
	bridgeExecutor.SlotID = "bridge-writer"
	bridgeVerifier := opts.Runtime.Recursion.Areas[0].Verifier
	bridgeVerifier.SlotID = "bridge-reviewer"
	opts.Runtime.Recursion.Areas = append(opts.Runtime.Recursion.Areas, AreaRunner{Area: bridge, Executor: bridgeExecutor, Verifier: bridgeVerifier, Checks: []authoring.Check{bridgeCheck}})
	for i := range opts.Runtime.Recursion.Areas {
		switch opts.Runtime.Recursion.Areas[i].Area.Key() {
		case quantity.Key():
			opts.Runtime.Recursion.Areas[i].Executor.Args[0] = "quantity"
		case price.Key():
			opts.Runtime.Recursion.Areas[i].Executor.Args[0] = "price"
		case bridge.Key():
			opts.Runtime.Recursion.Areas[i].Executor.Args = []string{"bridge"}
			opts.Runtime.Recursion.Areas[i].Verifier.Args = []string{"bridge"}
		}
	}
	report, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("three-level structural-parent run failed at %s: %v; actor logs: %s", report.Stage, err, recursiveActorLogs(report))
	}
	if report.Status != "accepted-scoped" || report.Candidate == nil || report.Promotion == nil || report.Promotion.Status != "promoted" {
		t.Fatalf("three-level reviewed root candidate was not promoted: %+v", report)
	}
	root := report.RootArea
	if root == nil || len(root.Attempts) != 1 || root.Attempts[0].Review == nil || root.Attempts[0].Review.Outcome != agentexec.OutcomePassed {
		t.Fatalf("structural root did not independently review the fully integrated candidate: %+v", root)
	}
	var bridgeReport *AreaReport
	for i := range root.Attempts[0].Children {
		if root.Attempts[0].Children[i].Area.Key() == bridge.Key() {
			bridgeReport = &root.Attempts[0].Children[i]
		}
	}
	if bridgeReport == nil || bridgeReport.Status != "passed-scoped" || len(bridgeReport.Attempts) != 2 || bridgeReport.Attempts[0].Review == nil || bridgeReport.Attempts[0].Review.Outcome != agentexec.OutcomeFailed || bridgeReport.Attempts[1].Review == nil || bridgeReport.Attempts[1].Review.Outcome != agentexec.OutcomePassed || len(bridgeReport.Attempts[1].Children) != 2 {
		t.Fatalf("three-level parent did not reject then repair its actual merged children: %+v", bridgeReport)
	}
	childAreas := map[string]bool{}
	for _, child := range bridgeReport.Attempts[1].Children {
		childAreas[child.Area.Key()] = child.Status == "passed-scoped"
	}
	if !childAreas[quantity.Key()] || !childAreas[price.Key()] {
		t.Fatalf("recursive middle Area did not retain both green child lineages: %+v", bridgeReport.Attempts[1].Children)
	}
	if len(root.Attempts[0].Checks) == 0 || root.Attempts[0].Checks[0].ExitCode != 0 {
		t.Fatalf("structural root did not run a fresh check on the complete candidate: %+v", root.Attempts[0].Checks)
	}
	var finalRootReview *ActorRecord
	for i := range report.Actors {
		if report.Actors[i].Phase == "review" && report.Actors[i].SlotID == "root-reviewer" && report.Actors[i].Result.Receipt.InputDigest != "" {
			finalRootReview = &report.Actors[i]
		}
	}
	if finalRootReview == nil || !containsString(finalRootReview.Scopes, g3AreaIdentity("root").Key()) {
		t.Fatal("final root review was not independently invoked for structural root")
	}
	if finalRootReview.Result.Response.InputDigest != finalRootReview.Result.Receipt.InputDigest {
		t.Fatal("final root review response is not bound to the final native invocation")
	}
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != report.CandidateCommit {
		t.Fatal("final candidate was not the only promoted root revision")
	}
	if head := fixtureGit(t, opts.Repo, "rev-parse", "HEAD"); head != base {
		t.Fatal("three-level recursive run moved user checkout HEAD")
	}
	assertRetainedReport(t, report)
}

func TestRecursiveRunRejectsMissingAreaAssignmentAndInsufficientCallBudget(t *testing.T) {
	for _, name := range []string{"missing-area-assignment", "preflight-call-budget", "repair-call-budget"} {
		t.Run(name, func(t *testing.T) {
			opts := recursiveFixtureOptions(t)
			switch name {
			case "missing-area-assignment":
				opts.Runtime.Recursion.Areas = opts.Runtime.Recursion.Areas[:1]
			case "preflight-call-budget":
				opts.Runtime.Recursion.Limits.MaxCalls = 1
			case "repair-call-budget":
				// Six calls cover the initial tree; two final votes and the root
				// evidence review bring the preflight minimum to nine. This bound
				// allows preflight but cannot fund one complete repair tree.
				opts.Runtime.Recursion.Limits.MaxCalls = 9
			}
			report, err := Run(context.Background(), opts)
			if err == nil || report.Promotion != nil {
				t.Fatalf("invalid frozen recursive assignment/budget was admitted: %+v: %v", report, err)
			}
			if name != "repair-call-budget" && len(report.Actors) != 0 {
				t.Fatalf("invalid frozen assignment/preflight budget invoked actors: %+v", report.Actors)
			}
			if name == "repair-call-budget" && (len(report.Actors) != 9 || report.RootArea == nil || len(report.RootArea.Attempts) != 2 || report.Promotion != nil) {
				t.Fatalf("global invocation limit did not stop the bounded repair before final-root acceptance: actors=%d root=%+v", len(report.Actors), report.RootArea)
			}
			if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != opts.Runtime.ExpectedBase {
				t.Fatal("invalid recursive runtime changed Active")
			}
			assertRetainedReport(t, report)
		})
	}
}
