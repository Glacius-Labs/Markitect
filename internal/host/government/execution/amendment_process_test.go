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

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

var amendmentFixtureBuild struct {
	once sync.Once
	path string
	err  error
}

func g4FixtureRunner(t *testing.T) string {
	t.Helper()
	// Reuse TestMain's package-owned temporary directory for the external actor
	// binary so the focused native tests do not leave build artifacts behind.
	_ = fixtureRunner(t)
	amendmentFixtureBuild.once.Do(func() {
		amendmentFixtureBuild.path = filepath.Join(fixtureBuild.dir, "runner-g4.exe")
		command := exec.Command("go", "build", "-o", amendmentFixtureBuild.path, "./runner")
		command.Dir = "../../../../examples/government-g4"
		if output, err := command.CombinedOutput(); err != nil {
			amendmentFixtureBuild.err = fmt.Errorf("build actual G4 process fixture: %w: %s", err, output)
		}
	})
	if amendmentFixtureBuild.err != nil {
		t.Fatal(amendmentFixtureBuild.err)
	}
	return amendmentFixtureBuild.path
}

func amendmentFixtureOptions(t *testing.T, action, subjectNamespace, subjectName string) Options {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo, state, temporary := filepath.Join(root, "repo"), filepath.Join(root, "state"), filepath.Join(root, "temporary")
	for _, directory := range []string{repo, state, temporary} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	fixture, err := filepath.Abs("../../../../examples/government-g4")
	if err != nil {
		t.Fatal(err)
	}
	if err := filepath.WalkDir(fixture, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(fixture, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(repo, relative)
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
	configBytes, err := os.ReadFile(filepath.Join(repo, "government.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var source government.Source
	if err := government.Decode(configBytes, &source); err != nil {
		t.Fatal(err)
	}
	model := government.Compile(source)
	if len(model.Findings) > 0 {
		t.Fatalf("public G4 fixture prior model is invalid: %+v", model.Findings)
	}
	order := government.Order{
		APIVersion: government.OrderVersion, Kind: "Order", Purpose: "Native G4 process regression order.",
		ActiveConstitution: model.Digest, Action: action,
		Subjects: []core.DefinitionIdentity{{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: subjectNamespace, Name: subjectName}},
	}
	writeOrder := func(name string, value government.Order) {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, name), encoded, 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeOrder("order.yaml", order)
	independent := order
	independent.Action = "implement"
	independent.Purpose = "Independent receipt order after a blocked invoice amendment."
	independent.Subjects = []core.DefinitionIdentity{{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "receipt", Name: "receipt-format"}}
	writeOrder("independent-order.yaml", independent)
	git := func(args ...string) string { return fixtureGit(t, repo, args...) }
	git("init", "-b", "main")
	git("config", "user.name", "Government G4 process test")
	git("config", "user.email", "government-g4-test@example.invalid")
	git("config", "core.autocrlf", "false")
	git("add", "--all")
	git("-c", "commit.gpgsign=false", "commit", "-m", "prior active G4 fixture")
	base := git("rev-parse", "HEAD")
	activeRef := "refs/markitect/government/active/g4-process-test"
	git("update-ref", activeRef, base)
	runner := g4FixtureRunner(t)
	runnerSpec := func(slot, mode string, args ...string) RunnerSpec {
		return RunnerSpec{SlotID: slot, Command: runner, Args: append([]string{mode}, args...), Model: "deterministic-g4-model-amendment-mechanics", ModelOptions: json.RawMessage(`{}`), ProviderVersion: "fixture-v1", TimeoutSeconds: 120, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20}
	}
	runtimeConfig := Runtime{
		APIVersion: RuntimeVersion, ActiveRef: activeRef, ExpectedBase: base, TimeoutSeconds: 1800,
		StateDirectory: state, TemporaryDirectory: temporary,
		Executor: runnerSpec("root-executor", "propose"), Verifier: runnerSpec("root-review", "review"),
		Ressorts: []RessortRunner{
			{Ressort: core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Ressort", Namespace: "invoice", Name: "arithmetic"}, Runner: runnerSpec("ressort-arithmetic", "assent")},
			{Ressort: core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Ressort", Namespace: "invoice", Name: "unaffected"}, Runner: runnerSpec("ressort-unaffected", "assent-unaffected")},
		},
		Checks:    []authoring.Check{{Name: "candidate-model-realization", Run: []string{"go", "test", "./...", "-count=1", "-timeout=20m"}}},
		Amendment: &AmendmentRuntime{MaxRepairs: 1},
	}
	return Options{Repo: repo, ConfigPath: "government.yaml", OrderPath: "order.yaml", Runtime: runtimeConfig}
}

func setAmendmentActor(spec *RunnerSpec, mode string, args ...string) {
	spec.Args = append([]string{mode}, args...)
}

func domainToDomainReferences(model government.Model) [][2]string {
	domainIDs := map[string]bool{}
	for _, definition := range model.Canonical.Definitions {
		if definition.APIVersion != government.APIVersion {
			domainIDs[definition.Identity().Key()] = true
		}
	}
	edges := [][2]string{}
	var visit func(string, any)
	visit = func(from string, value any) {
		switch typed := value.(type) {
		case map[string]any:
			apiVersion, _ := typed["apiVersion"].(string)
			kind, _ := typed["kind"].(string)
			namespace, _ := typed["namespace"].(string)
			name, _ := typed["name"].(string)
			if apiVersion != "" && kind != "" && namespace != "" && name != "" {
				key := core.DefinitionIdentity{APIVersion: apiVersion, Kind: kind, Namespace: namespace, Name: name}.Key()
				if domainIDs[key] {
					edges = append(edges, [2]string{from, key})
				}
			}
			for _, child := range typed {
				visit(from, child)
			}
		case []any:
			for _, child := range typed {
				visit(from, child)
			}
		}
	}
	for _, definition := range model.Canonical.Definitions {
		if !domainIDs[definition.Identity().Key()] {
			continue // Responsibility and Realization are Government structural edges.
		}
		visit(definition.Identity().Key(), definition.Spec)
	}
	return edges
}

func reportPathSet(report Report) map[string]bool {
	paths := map[string]bool{}
	for _, work := range report.Plan.Work {
		for _, path := range work.Paths {
			paths[path] = true
		}
	}
	return paths
}

func mandateHasAction(model government.Model, mandate core.DefinitionIdentity, action string) bool {
	for _, definition := range model.Canonical.Definitions {
		if definition.Identity().Key() != mandate.Key() {
			continue
		}
		switch actions := definition.Spec["actions"].(type) {
		case []string:
			return containsString(actions, action)
		case []any:
			for _, value := range actions {
				if value == action {
					return true
				}
			}
		}
	}
	return false
}

func configureSplitWriterAmendment(t *testing.T, opts Options) Options {
	t.Helper()
	configPath := filepath.Join(opts.Repo, opts.ConfigPath)
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var source government.Source
	if err := government.Decode(configBytes, &source); err != nil {
		t.Fatal(err)
	}
	root := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Area", Namespace: "invoice", Name: "root"}
	rootMandate := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Mandate", Namespace: "invoice", Name: "root-prior"}
	code := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Area", Namespace: "invoice", Name: "code"}
	delivery := core.DefinitionIdentity{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "invoice", Name: "delivery-rule"}
	definitions := make([]core.Definition, 0, len(source.Definitions)+2)
	for _, definition := range source.Definitions {
		key := definition.Kind + "/" + definition.Metadata.Name
		switch key {
		case "Artifact/delivery-rate", "Artifact/invoice-source", "Artifact/invoice-check":
			definition.Spec["writer"] = code
		case "Realization/published-delivery-value":
			// This test order amends only delivery-rule. Keep the separate
			// published-delivery subject outside this frozen descendant task.
			continue
		}
		definitions = append(definitions, definition)
	}
	source.Definitions = append(definitions,
		core.Definition{APIVersion: government.APIVersion, Kind: "Area", Metadata: core.Metadata{Namespace: "invoice", Name: "code"}, Purpose: "Implements the invoice realization under the existing root authority.", Spec: map[string]any{"parent": root, "capabilities": []string{}}},
		core.Definition{APIVersion: government.APIVersion, Kind: "Mandate", Metadata: core.Metadata{Namespace: "invoice", Name: "code-implementation"}, Purpose: "Prior delegated authority for invoice realization code only.", Spec: map[string]any{"area": code, "parent": rootMandate, "scope": []core.DefinitionIdentity{delivery}, "actions": []string{"implement", "review"}}},
	)
	encoded, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	var normalizedSource government.Source
	if err := government.Decode(encoded, &normalizedSource); err != nil {
		t.Fatal(err)
	}
	model := government.Compile(normalizedSource)
	if len(model.Findings) > 0 {
		t.Fatalf("split-writer prior model is invalid: %+v", model.Findings)
	}
	orderPath := filepath.Join(opts.Repo, opts.OrderPath)
	orderBytes, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	var order government.Order
	if err := government.Decode(orderBytes, &order); err != nil {
		t.Fatal(err)
	}
	order.ActiveConstitution = model.Digest
	order.Subjects = []core.DefinitionIdentity{delivery}
	encodedOrder, err := json.Marshal(order)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orderPath, encodedOrder, 0644); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, opts.Repo, "add", opts.ConfigPath, opts.OrderPath)
	fixtureGit(t, opts.Repo, "-c", "commit.gpgsign=false", "commit", "-m", "freeze split model and realization writers")
	base := fixtureGit(t, opts.Repo, "rev-parse", "HEAD")
	fixtureGit(t, opts.Repo, "update-ref", opts.Runtime.ActiveRef, base)
	opts.Runtime.ExpectedBase = base
	return opts
}

func assertUnchangedActive(t *testing.T, opts Options, base string) {
	t.Helper()
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != base {
		t.Fatalf("blocked amendment moved Active from prior base %s to %s", base, active)
	}
	if head := fixtureGit(t, opts.Repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("amendment moved user checkout HEAD from prior base %s to %s", base, head)
	}
}

func TestAmendmentRunPromotesAuthorizedModelAndRealizationWithFreshCabinet(t *testing.T) {
	opts := amendmentFixtureOptions(t, "amend-model", "invoice", "delivery-rule")
	base := opts.Runtime.ExpectedBase
	originalSource, err := os.ReadFile(filepath.Join(opts.Repo, "government.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("authorized native amendment failed at %s: %v", report.Stage, err)
	}
	if report.Status != "accepted-scoped" || report.Stage != "complete" || report.Promotion == nil || report.Promotion.Status != "promoted" {
		t.Fatalf("authorized amendment did not reach protected promotion: %s/%s %+v", report.Status, report.Stage, report.Promotion)
	}
	if report.PriorModelDigest == "" || report.ProposedModelDigest == "" || report.PriorModelDigest == report.ProposedModelDigest || len(report.AmendmentRounds) != 1 {
		t.Fatalf("report did not bind prior and proposed law: %+v", report)
	}
	round := report.AmendmentRounds[0]
	if round.Status != "accepted-scoped" || round.Candidate == nil || round.Assessment == nil || round.Assessment.Status != "eligible-for-fresh-review" || round.Evidence == nil || round.Decision == nil {
		t.Fatalf("fresh candidate/model assessment/evidence/decision is incomplete: %+v", round)
	}
	if round.Candidate.Input.PriorConstitutionDigest != report.PriorModelDigest || round.Candidate.Input.ModelDigest != report.ProposedModelDigest || round.Evidence.MaterialCandidateID != round.Candidate.ID || round.Evidence.Round != 1 {
		t.Fatalf("candidate and evidence do not bind the old law and new model: candidate=%+v evidence=%+v", round.Candidate, round.Evidence)
	}
	if len(round.Checks) != 1 || round.Checks[0].ExitCode != 0 || len(round.ReviewActorSequences) == 0 || len(round.Votes) != 2 || len(report.Cabinet) != 2 {
		t.Fatalf("final candidate lacks fresh technical, independent, or full-cabinet evidence: %+v", round)
	}
	if len(report.Votes) != 2 || report.Votes[1].Outcome != government.VoteAssentUnaffected || round.Decision.EvidenceID != round.Evidence.ID || len(round.Decision.VoteIDs) != 2 {
		t.Fatalf("frozen cabinet did not explicitly assent to this final evidence: %+v", report.Votes)
	}
	for _, vote := range round.Votes {
		if vote.MaterialCandidateID != round.Candidate.ID || vote.EvidenceID != round.Evidence.ID || vote.Round != 1 {
			t.Fatalf("vote is not bound to the exact amended candidate and evidence: %+v", vote)
		}
	}
	if !bytes.Contains(originalSource, []byte("delivery is included without a separate charge")) {
		t.Fatal("fixture did not start from its frozen prior law")
	}
	for _, changed := range []string{"government.yaml", "invoice/invoice.go", "invoice/invoice_test.go"} {
		if !containsString(round.ChangedPaths, changed) {
			t.Fatalf("model amendment omitted coupled realization path %q: %v", changed, round.ChangedPaths)
		}
	}
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != round.CandidateCommit || round.CandidateCommit != report.CandidateCommit {
		t.Fatalf("Active did not promote the exact final amendment commit: active=%s round=%s report=%s", active, round.CandidateCommit, report.CandidateCommit)
	}
	if head := fixtureGit(t, opts.Repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("user checkout HEAD moved from prior G4 base: %s", head)
	}
	if got := fixtureGit(t, opts.Repo, "show", round.CandidateCommit+":government.yaml"); strings.Contains(got, "delivery is included without a separate charge") || !strings.Contains(got, "plus the published delivery charge") {
		t.Fatalf("promoted candidate does not contain the authorized new rule: %s", got)
	}
	assertRetainedReport(t, report)
}

func TestAmendmentObjectionRepairsCandidateAndRecollectsFreshEvidence(t *testing.T) {
	opts := amendmentFixtureOptions(t, "amend-model", "invoice", "delivery-rule")
	setAmendmentActor(&opts.Runtime.Executor, "veto-repair")
	setAmendmentActor(&opts.Runtime.Ressorts[0].Runner, "object-once")
	base := opts.Runtime.ExpectedBase

	report, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("bounded objection repair did not complete: stage=%s err=%v", report.Stage, err)
	}
	if report.Status != "accepted-scoped" || report.Promotion == nil || report.Promotion.Status != "promoted" || len(report.AmendmentRounds) != 2 {
		t.Fatalf("objection did not produce one bounded repair and final promotion: status=%s promotion=%+v rounds=%+v", report.Status, report.Promotion, report.AmendmentRounds)
	}
	first, final := report.AmendmentRounds[0], report.AmendmentRounds[1]
	if first.Evidence == nil || first.Candidate == nil || len(first.Votes) != 2 || first.Votes[0].Outcome != government.VoteObjection || !strings.Contains(first.Votes[0].Reason, "delivery/rate.txt") {
		t.Fatalf("round one did not retain the exact actual Ressort objection: %+v", first)
	}
	if final.Round != 2 || final.Candidate == nil || final.Evidence == nil || final.Candidate.ID == first.Candidate.ID || final.Evidence.ID == first.Evidence.ID {
		t.Fatalf("repair reused candidate or evidence identity instead of binding a fresh round: first=%+v final=%+v", first, final)
	}
	if len(final.Checks) != 1 || final.Checks[0].ExitCode != 0 || len(final.ReviewActorSequences) == 0 || len(final.Votes) != 2 || final.Decision == nil || len(final.Decision.VoteIDs) != 2 {
		t.Fatalf("repaired candidate lacks complete fresh technical/review/cabinet evidence: %+v", final)
	}
	for _, vote := range final.Votes {
		if vote.MaterialCandidateID != final.Candidate.ID || vote.EvidenceID != final.Evidence.ID || vote.Round != 2 || vote.Outcome == government.VoteObjection {
			t.Fatalf("final cabinet vote is stale or not assent: %+v", vote)
		}
	}
	if first.Votes[0].EvidenceID == final.Votes[0].EvidenceID || first.Votes[0].ID == final.Votes[0].ID {
		t.Fatalf("prior objection vote was reused for the repaired candidate: old=%+v new=%+v", first.Votes[0], final.Votes[0])
	}
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != final.CandidateCommit || final.CandidateCommit != report.CandidateCommit {
		t.Fatalf("Active did not promote the final repaired candidate: active=%s final=%s report=%s", active, final.CandidateCommit, report.CandidateCommit)
	}
	if head := fixtureGit(t, opts.Repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("repair changed user checkout HEAD from prior base: %s", head)
	}
	assertRetainedReport(t, report)
}

func TestAmendmentPersistentCabinetVetoEscalatesWithoutChangingAuthority(t *testing.T) {
	opts := amendmentFixtureOptions(t, "amend-model", "invoice", "delivery-rule")
	setAmendmentActor(&opts.Runtime.Executor, "persistent-veto")
	setAmendmentActor(&opts.Runtime.Ressorts[0].Runner, "persistent-objection")
	base := opts.Runtime.ExpectedBase

	report, err := Run(context.Background(), opts)
	if err == nil {
		t.Fatalf("persistent actual Ressort veto unexpectedly completed: %+v", report)
	}
	if len(report.AmendmentRounds) != 2 || len(report.Escalations) != 1 || report.Escalations[0].PromotionAttempted || report.Promotion != nil {
		t.Fatalf("bounded persistent veto did not end in a concrete escalation without promotion: err=%v report=%+v", err, report)
	}
	if len(report.Cabinet) != 2 || report.AmendmentRounds[0].Candidate == nil || report.AmendmentRounds[1].Candidate == nil || report.AmendmentRounds[0].Candidate.ID == report.AmendmentRounds[1].Candidate.ID {
		t.Fatalf("veto repair failed to preserve the frozen cabinet and create a fresh candidate: %+v", report)
	}
	for _, round := range report.AmendmentRounds {
		if round.Evidence == nil || len(round.Votes) != 2 || len(round.Checks) != 1 || round.Checks[0].ExitCode != 0 || len(round.ReviewActorSequences) == 0 {
			t.Fatalf("round %d omitted fresh evidence despite the veto: %+v", round.Round, round)
		}
		if round.Votes[0].Outcome != government.VoteObjection || round.Votes[0].MaterialCandidateID != round.Candidate.ID || round.Votes[0].EvidenceID != round.Evidence.ID || round.Votes[0].Round != uint64(round.Round) {
			t.Fatalf("round %d did not retain its bound cabinet veto: %+v", round.Round, round.Votes)
		}
	}
	if report.Escalations[0].Round != 2 || report.Escalations[0].MaterialCandidateID != report.AmendmentRounds[1].Candidate.ID || report.Escalations[0].EvidenceID != report.AmendmentRounds[1].Evidence.ID || len(report.Escalations[0].VoteIDs) != 2 {
		t.Fatalf("escalation is not bound to the final vetoed candidate and votes: %+v", report.Escalations[0])
	}
	assertUnchangedActive(t, opts, base)
	assertRetainedReport(t, report)
}

func TestAmendmentCannotSelfAuthorizeProtectedGoalAndIndependentOrderCanProceed(t *testing.T) {
	// The order stays within the existing invoice mandate; the malicious actor
	// independently tries to rewrite protected-total outside that mandate.
	opts := amendmentFixtureOptions(t, "amend-model", "invoice", "delivery-rule")
	setAmendmentActor(&opts.Runtime.Executor, "self-authorize")
	base := opts.Runtime.ExpectedBase

	blocked, err := Run(context.Background(), opts)
	if err == nil || len(blocked.Escalations) == 0 || blocked.Promotion != nil || len(blocked.AmendmentRounds) != 1 {
		t.Fatalf("protected root-goal self-authorization did not block and escalate: err=%v report=%+v", err, blocked)
	}
	foundProtected := false
	for _, finding := range blocked.AmendmentRounds[0].Assessment.Findings {
		if finding.Code == "amendment.protected-subject" {
			foundProtected = true
		}
	}
	if !foundProtected {
		t.Fatalf("blocked self-authorization lacks a protected-subject assessment finding: %+v", blocked.AmendmentRounds[0].Assessment)
	}
	assertUnchangedActive(t, opts, base)
	priorBytes, err := os.ReadFile(filepath.Join(opts.Repo, opts.ConfigPath))
	if err != nil {
		t.Fatal(err)
	}
	var priorSource government.Source
	if err := government.Decode(priorBytes, &priorSource); err != nil {
		t.Fatal(err)
	}
	priorModel := government.Compile(priorSource)
	if len(priorModel.Findings) != 0 || priorModel.Digest != blocked.PriorModelDigest {
		t.Fatalf("protected amendment did not preserve a valid, exact prior source model: findings=%+v digest=%s report=%s", priorModel.Findings, priorModel.Digest, blocked.PriorModelDigest)
	}

	// A separate explicit order for a disjoint, already-authorized subject can
	// proceed against the same unchanged prior model after the invoice conflict.
	opts.OrderPath = "independent-order.yaml"
	opts.Runtime.Amendment = nil
	setAmendmentActor(&opts.Runtime.Executor, "independent")
	setAmendmentActor(&opts.Runtime.Verifier, "review-independent")
	setAmendmentActor(&opts.Runtime.Ressorts[0].Runner, "assent-unaffected")
	setAmendmentActor(&opts.Runtime.Ressorts[1].Runner, "assent")

	accepted, err := Run(context.Background(), opts)
	if err != nil || accepted.Status != "accepted-scoped" || accepted.Promotion == nil || accepted.Promotion.Status != "promoted" {
		t.Fatalf("independent order did not proceed under the unchanged prior law: status=%s err=%v report=%+v", accepted.Status, err, accepted)
	}
	if accepted.BaseRevision != blocked.BaseRevision || accepted.PriorModelDigest != blocked.PriorModelDigest || !containsString(accepted.ChangedPaths, "independent/receipt.txt") || containsString(accepted.ChangedPaths, "government.yaml") {
		t.Fatalf("independent order did not remain disjoint on the same prior base: blocked=%+v accepted=%+v", blocked, accepted)
	}
	blockedSubjects := map[string]bool{}
	for _, subject := range blocked.Plan.Affected {
		blockedSubjects[subject.Key()] = true
	}
	independentSubjects := map[string]bool{}
	for _, subject := range accepted.Plan.Affected {
		independentSubjects[subject.Key()] = true
	}
	for subject := range blockedSubjects {
		if independentSubjects[subject] {
			t.Fatalf("protected invoice conflict and independent order share an affected domain subject: %s", subject)
		}
	}
	blockedPaths, independentPaths := reportPathSet(blocked), reportPathSet(accepted)
	for path := range blockedPaths {
		if independentPaths[path] {
			t.Fatalf("protected invoice conflict and independent order share a writer path: %s", path)
		}
	}
	crossDomainEdges := [][2]string{}
	for _, edge := range domainToDomainReferences(priorModel) {
		if blockedSubjects[edge[0]] && independentSubjects[edge[1]] || independentSubjects[edge[0]] && blockedSubjects[edge[1]] {
			crossDomainEdges = append(crossDomainEdges, edge)
		}
	}
	if len(crossDomainEdges) != 0 {
		t.Fatalf("pre-amendment source declares domain-to-domain dependencies between conflicting and independent subjects: %v", crossDomainEdges)
	}
	t.Logf("prior model %s has %d typed domain-to-domain references; cross-order affected-subject references=0; blocked and independent Plan.Affected/Work.Paths are disjoint", priorModel.Digest, len(domainToDomainReferences(priorModel)))
	if fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef) != accepted.CandidateCommit {
		t.Fatalf("Active does not point to the independently accepted candidate")
	}
	if head := fixtureGit(t, opts.Repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("independent order changed user checkout HEAD from prior base: %s", head)
	}
	assertRetainedReport(t, blocked)
	assertRetainedReport(t, accepted)
}

func TestAmendmentRejectsActualStaleVoteFromPriorCandidate(t *testing.T) {
	firstOptions := amendmentFixtureOptions(t, "amend-model", "invoice", "delivery-rule")
	first, err := Run(context.Background(), firstOptions)
	if err != nil || first.Status != "accepted-scoped" || len(first.AmendmentRounds) != 1 || len(first.AmendmentRounds[0].Votes) != 2 {
		t.Fatalf("could not obtain prior accepted native votes for stale replay: status=%s err=%v", first.Status, err)
	}
	oldVote := first.AmendmentRounds[0].Votes[0]
	if oldVote.MaterialCandidateID != first.AmendmentRounds[0].Candidate.ID || oldVote.EvidenceID != first.AmendmentRounds[0].Evidence.ID || oldVote.Round != 1 {
		t.Fatalf("first process did not yield a bound actual vote to replay: %+v", oldVote)
	}

	secondOptions := amendmentFixtureOptions(t, "amend-model", "invoice", "delivery-rule")
	setAmendmentActor(&secondOptions.Runtime.Ressorts[0].Runner, "stale-assent", oldVote.MaterialCandidateID, oldVote.EvidenceID, fmt.Sprint(oldVote.Round))
	base := secondOptions.Runtime.ExpectedBase
	second, err := Run(context.Background(), secondOptions)
	if err == nil || !strings.Contains(err.Error(), "stale or foreign candidate/evidence/round") {
		t.Fatalf("actual prior vote body was not rejected against the new candidate/evidence: err=%v report=%+v", err, second)
	}
	if len(second.AmendmentRounds) != 1 || second.AmendmentRounds[0].Candidate == nil || second.AmendmentRounds[0].Evidence == nil || second.AmendmentRounds[0].Candidate.ID == oldVote.MaterialCandidateID || second.AmendmentRounds[0].Evidence.ID == oldVote.EvidenceID {
		t.Fatalf("stale replay did not target a distinct candidate and evidence: old=%+v report=%+v", oldVote, second)
	}
	if second.Promotion != nil || len(second.Votes) != 0 {
		t.Fatalf("stale vote was recorded as accepted or caused promotion: %+v", second)
	}
	assertUnchangedActive(t, secondOptions, base)
	assertRetainedReport(t, first)
	assertRetainedReport(t, second)
}

func TestAmendmentRecursivelySplitsCanonicalModelAndCodeWriters(t *testing.T) {
	opts := amendmentFixtureOptions(t, "amend-model", "invoice", "delivery-rule")
	opts = configureSplitWriterAmendment(t, opts)
	base := opts.Runtime.ExpectedBase
	code := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Area", Namespace: "invoice", Name: "code"}
	runner := opts.Runtime.Executor.Command
	runnerSpec := func(slot, mode string) RunnerSpec {
		return RunnerSpec{SlotID: slot, Command: runner, Args: []string{mode}, Model: "deterministic-g4-model-amendment-mechanics", ModelOptions: json.RawMessage(`{}`), ProviderVersion: "fixture-v1", TimeoutSeconds: 120, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20}
	}
	setAmendmentActor(&opts.Runtime.Executor, "propose-model")
	opts.Runtime.Executor.SlotID = "root-model-amender"
	setAmendmentActor(&opts.Runtime.Verifier, "review")
	opts.Runtime.Verifier.SlotID = "root-integrator-reviewer"
	opts.Runtime.Recursion = &RecursiveRuntime{
		Limits: government.DelegationLimits{MaxDepth: 2, MaxFanout: 1, MaxCalls: 32}, Parallelism: 1, MaxRepairs: 1,
		Areas: []AreaRunner{{
			Area: code, Executor: runnerSpec("invoice-code-writer", "propose-realization"), Verifier: runnerSpec("invoice-code-reviewer", "child-review"),
			Checks: []authoring.Check{{Name: "invoice-code-compiles", Run: []string{"go", "test", "./invoice", "-run", "^$", "-count=1"}, TimeoutSeconds: intPointer(120)}},
		}},
	}
	priorModel := government.Compile(func() government.Source {
		data, err := os.ReadFile(filepath.Join(opts.Repo, opts.ConfigPath))
		if err != nil {
			t.Fatal(err)
		}
		var source government.Source
		if err := government.Decode(data, &source); err != nil {
			t.Fatal(err)
		}
		return source
	}())
	report, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("recursive split-writer amendment failed at %s: %v; rootArea=%s actors=%s", report.Stage, err, jsonText(report.RootArea), jsonText(report.Actors))
	}
	if report.Status != "accepted-scoped" || report.Promotion == nil || report.Promotion.Status != "promoted" || len(report.AmendmentRounds) != 1 {
		t.Fatalf("split writers did not reach one final promoted model candidate: status=%s promotion=%+v rounds=%+v", report.Status, report.Promotion, report.AmendmentRounds)
	}
	if report.Delegation == nil || report.Delegation.Status != "planned" && report.Delegation.Status != "planned-scoped" || len(report.Delegation.Root.Children) != 1 {
		t.Fatalf("old active authority did not freeze one recursive code child: %+v", report.Delegation)
	}
	rootNode, childNode := report.Delegation.Root, report.Delegation.Root.Children[0]
	if !containsString(rootNode.Actions, "amend-model") || containsString(childNode.Actions, "amend-model") || !containsString(childNode.Actions, "implement") {
		t.Fatalf("frozen local actions do not separate root model amendment from child implementation: root=%v child=%v", rootNode.Actions, childNode.Actions)
	}
	if len(rootNode.Work.Paths) != 1 || rootNode.Work.Paths[0] != "government.yaml" || !containsString(childNode.Work.Paths, "invoice/invoice.go") || !containsString(childNode.Work.Paths, "invoice/invoice_test.go") {
		t.Fatalf("canonical source and code do not have disjoint exact local writers: root=%v child=%v", rootNode.Work.Paths, childNode.Work.Paths)
	}
	codeMandate := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Mandate", Namespace: "invoice", Name: "code-implementation"}
	rootMandate := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Mandate", Namespace: "invoice", Name: "root-prior"}
	if !mandateHasAction(priorModel, rootMandate, "amend-model") || !mandateHasAction(priorModel, rootMandate, "implement") || !mandateHasAction(priorModel, codeMandate, "implement") || mandateHasAction(priorModel, codeMandate, "amend-model") {
		t.Fatalf("prior authority does not separate canonical-model amendment from code implementation: rootMandate=%+v codeMandate=%+v", rootMandate, codeMandate)
	}
	if len(report.RootArea.Attempts) != 1 || len(report.RootArea.Attempts[0].Children) != 1 {
		t.Fatalf("root report omitted recursive child lineage: %+v", report.RootArea)
	}
	childReport := report.RootArea.Attempts[0].Children[0]
	if childReport.Area.Key() != code.Key() || childReport.Status != "passed-scoped" || len(childReport.Attempts) != 1 || childReport.Attempts[0].Status != "passed-scoped" || childReport.Attempts[0].Review == nil || len(childReport.Attempts[0].Checks) != 1 || childReport.Attempts[0].Checks[0].ExitCode != 0 {
		t.Fatalf("child implementation lacks fresh check and read-only review evidence: %+v", childReport)
	}
	childReviewScopes := []string{}
	for _, observation := range childReport.Attempts[0].Review.VerifierObservations {
		childReviewScopes = append(childReviewScopes, observation.Subject)
	}
	if len(childReviewScopes) != 2 || !containsString(childReviewScopes, code.Key()) || !containsString(childReviewScopes, "[\"markitect.government-example/v1alpha1\",\"Requirement\",\"invoice\",\"delivery-rule\"]") {
		t.Fatalf("child review receipt is not bound to the exact child Area and requirement scopes: %+v", childReviewScopes)
	}
	round := report.AmendmentRounds[0]
	if round.Assessment == nil || round.Assessment.Status != "eligible-for-fresh-review" || round.Candidate == nil || round.Candidate.Input.PriorConstitutionDigest != priorModel.Digest || round.Candidate.Input.ModelDigest != round.ProposedModelDigest || round.Evidence == nil || len(round.Checks) != 1 || round.Checks[0].ExitCode != 0 || len(round.ReviewActorSequences) == 0 || len(round.Votes) != 2 {
		t.Fatalf("root did not bind the merged model and recursive realization into fresh final evidence: %+v", round)
	}
	for _, changed := range []string{"government.yaml", "invoice/invoice.go", "invoice/invoice_test.go"} {
		if !containsString(round.ChangedPaths, changed) {
			t.Fatalf("final amendment omitted split writer path %q: %v", changed, round.ChangedPaths)
		}
	}
	actorSlots := map[string]bool{}
	for _, actor := range report.Actors {
		actorSlots[actor.SlotID+"/"+actor.Phase] = true
	}
	for _, required := range []string{"root-model-amender/execute", "invoice-code-writer/execute", "invoice-code-reviewer/review", "root-integrator-reviewer/review"} {
		if !actorSlots[required] {
			t.Fatalf("native split writer did not invoke required actual actor %s; actor slots=%v", required, actorSlots)
		}
	}
	if active := fixtureGit(t, opts.Repo, "rev-parse", opts.Runtime.ActiveRef); active != report.CandidateCommit || report.CandidateCommit != round.CandidateCommit {
		t.Fatalf("only the merged root candidate should promote: active=%s report=%s round=%s", active, report.CandidateCommit, round.CandidateCommit)
	}
	if head := fixtureGit(t, opts.Repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("recursive amendment changed user checkout HEAD from frozen base: %s", head)
	}
	if promotedCode := fixtureGit(t, opts.Repo, "show", report.CandidateCommit+":invoice/invoice.go"); !strings.Contains(promotedCode, "units*unitPrice + deliveryCharge") {
		t.Fatalf("promoted root commit omitted the child-authored code: %s", promotedCode)
	}
	if priorModel.Digest != report.PriorModelDigest || report.ProposedModelDigest == report.PriorModelDigest {
		t.Fatalf("split writer run did not bind the prior and amended model digests: prior=%s report prior=%s proposed=%s", priorModel.Digest, report.PriorModelDigest, report.ProposedModelDigest)
	}
	assertRetainedReport(t, report)
}
