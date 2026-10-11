package projectcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

var testCommitEnv = []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid"}

// runCLI runs one product CLI invocation exactly as the binary does.
func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errout bytes.Buffer
	code := Main(args, &out, &errout, "test")
	return code, out.String(), errout.String()
}

// runCLIWith runs the CLI path with replaced operations, such as a counting
// agent invoker, from parsing through the exit-code mapping.
func runCLIWith(t *testing.T, ops projectapp.Operations, args ...string) (int, string, string) {
	t.Helper()
	v, ok := lookupVerb(args[0])
	if !ok {
		t.Fatalf("unknown verb %q", args[0])
	}
	inv, help, err := parseInvocation(v, args[1:])
	if err != nil || help {
		t.Fatalf("parse %v: help=%v err=%v", args, help, err)
	}
	inv.env.ops = ops
	var out, errout bytes.Buffer
	result, err := v.invoke(context.Background(), inv.env, inv.raw)
	code := finish(v, result, err, &out, &errout)
	return code, out.String(), errout.String()
}

// mustCLI runs an invocation that must exit 0 and returns its stdout.
func mustCLI(t *testing.T, args ...string) []byte {
	t.Helper()
	code, out, errout := runCLI(t, args...)
	if code != 0 {
		t.Fatalf("%v: exit=%d stderr=%s stdout=%s", args, code, errout, out)
	}
	return []byte(out)
}

func decodeOutput[T any](t *testing.T, data []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode output: %v\n%s", err, data)
	}
	return value
}

// writeDraft writes one --input record below the repository and returns its
// repository-relative path. A value that is not []byte is encoded as JSON.
func writeDraft(t *testing.T, repo, path string, value any) string {
	t.Helper()
	data, ok := value.([]byte)
	if !ok {
		var err error
		if data, err = json.Marshal(value); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(repo, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProjectAgentFixtureMatchesClosedWorkResponseContract(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test caller path unavailable")
	}
	fixtures := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "examples", "project-world", "tests", "agent-fixtures")
	invocationBytes, err := os.ReadFile(filepath.Join(fixtures, "invocation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var invocation struct {
		Request struct {
			Context struct {
				ResponseSchema json.RawMessage `json:"responseSchema"`
			} `json:"context"`
		} `json:"request"`
	}
	if err := json.Unmarshal(invocationBytes, &invocation); err != nil {
		t.Fatal(err)
	}
	wantSchema, err := projectrun.TaskResponseSchema("work")
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(invocation.Request.Context.ResponseSchema, &gotValue); err != nil {
		t.Fatalf("decode fixture response schema: %v", err)
	}
	if err := json.Unmarshal(wantSchema, &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatal("invocation fixture responseSchema differs from projectrun's work contract")
	}
	responseBytes, err := os.ReadFile(filepath.Join(fixtures, "response.json"))
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		ReportJSON json.RawMessage `json:"reportJson"`
		Candidate  json.RawMessage `json:"candidateJson"`
	}
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.ReportJSON) == 0 || len(response.Candidate) != 0 {
		t.Fatal("fixture must carry the typed executor report, not infer-only candidateJson")
	}
}

func TestProjectWorldFixtureAndReviewedEdit(t *testing.T) {
	repo := copyProjectWorld(t)
	initial, err := projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if initial.Report.Status != "succeeded" {
		t.Fatalf("fixture model status = %q, findings=%+v, unknown=%v", initial.Report.Status, initial.Report.Findings, initial.Report.Unknown)
	}
	if len(initial.Report.Managers) != 6 || len(initial.Report.Artifacts) != 5 || len(initial.Report.Checks) != 2 {
		t.Fatalf("fixture report counts = managers:%d artifacts:%d checks:%d", len(initial.Report.Managers), len(initial.Report.Artifacts), len(initial.Report.Checks))
	}

	report := decodeOutput[checkReport](t, mustCLI(t, "check", "--repo", repo))
	if report.Status != "succeeded" || report.ProjectDigest != initial.Digest {
		t.Fatalf("check output report=%+v", report)
	}

	// docs previews, then writes exactly the reviewed preview.
	viewPath := filepath.Join(repo, "docs", "markitect", "project.md")
	assertNoView := func(stage string) {
		t.Helper()
		if _, err := os.Stat(viewPath); !os.IsNotExist(err) {
			t.Fatalf("%s wrote the document: %v", stage, err)
		}
	}
	assertNoView("fixture")
	preview := decodeOutput[projectapp.DocumentResult](t, mustCLI(t, "docs", "--repo", repo))
	if preview.Written || preview.Path != "docs/markitect/project.md" || preview.Digest == "" || !strings.Contains(preview.Content, "cancel-before-shipped") {
		t.Fatalf("docs preview = path %q digest %q written %v", preview.Path, preview.Digest, preview.Written)
	}
	assertNoView("docs preview")
	if code, out, errout := runCLI(t, "docs", "--repo", repo, "--expect", "sha256:stale", "--write"); code != 2 || out != "" || !strings.Contains(errout, "markitect docs:") {
		t.Fatalf("docs write with a stale digest exit=%d stdout=%s stderr=%s", code, out, errout)
	}
	assertNoView("stale docs write")
	written := decodeOutput[projectapp.DocumentResult](t, mustCLI(t, "docs", "--repo", repo, "--expect", preview.Digest, "--write"))
	view, err := os.ReadFile(viewPath)
	if err != nil || !written.Written || written.Digest != preview.Digest || !bytes.Contains(view, []byte("cancel-before-shipped")) {
		t.Fatalf("docs write = written %v digest %q err=%v", written.Written, written.Digest, err)
	}
	editBasis, err := projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}

	manager := ""
	for _, candidate := range initial.Report.Managers {
		if candidate.Namespace == "commerce.sales.orders" && candidate.Name == "orders" {
			manager = candidate.ID
		}
	}
	if manager == "" {
		t.Fatal("orders manager identity was not present in the fixture report")
	}
	if contextOut := mustCLI(t, "context", manager, "--repo", repo); !bytes.Contains(contextOut, []byte("cancel-before-shipped")) {
		t.Fatalf("orders context omitted its cancellation rule: %s", contextOut)
	}

	const rulePath = ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml"
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rulePath)))
	if err != nil {
		t.Fatal(err)
	}
	mutation := projectwork.Mutation{
		APIVersion: projectwork.APIVersion,
		BaseDigest: editBasis.Digest,
		Actor:      projectwork.HumanActor,
		Goal:       "Clarify the cancellation transition",
		Files: []projectwork.FileChange{{
			Path:    rulePath,
			Content: strings.Replace(string(content), "Cancellation is valid only while the order is confirmed; a shipped order cannot be cancelled.", "Cancellation is valid only before shipment; shipped orders remain unchanged.", 1),
		}},
	}
	plan, err := projectwork.PlanEdit(editBasis, mutation)
	if err != nil {
		t.Fatal(err)
	}
	proposalBytes, err := projectwork.EncodeMutation(mutation)
	if err != nil {
		t.Fatal(err)
	}
	input := writeDraft(t, repo, ".markitect/drafts/cancel-rule.json", proposalBytes)
	editPreview := decodeOutput[projectwork.EditPlan](t, mustCLI(t, "edit", "--repo", repo, "--input", input))
	if editPreview.Digest != plan.Digest {
		t.Fatalf("preview digest=%q want %q", editPreview.Digest, plan.Digest)
	}

	changed := mutation
	changed.Goal = "Replace the previously reviewed proposal"
	changedBytes, err := projectwork.EncodeMutation(changed)
	if err != nil {
		t.Fatal(err)
	}
	writeDraft(t, repo, input, changedBytes)
	if code, _, errout := runCLI(t, "edit", "--repo", repo, "--input", input, "--expect", plan.Digest, "--write"); code != 2 || !strings.Contains(errout, "exact edit plan digest") {
		t.Fatalf("changed proposal unexpectedly passed reviewed digest: exit=%d stderr=%s", code, errout)
	}
	current, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rulePath)))
	if err != nil || !bytes.Equal(current, content) {
		t.Fatalf("stale proposal changed model bytes: err=%v", err)
	}

	writeDraft(t, repo, input, proposalBytes)
	mustCLI(t, "edit", "--repo", repo, "--input", input, "--expect", plan.Digest, "--write")
	updated, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rulePath)))
	if err != nil || bytes.Equal(updated, content) || !bytes.Contains(updated, []byte("before shipment")) {
		t.Fatalf("reviewed edit was not applied: err=%v content=%s", err, updated)
	}
}

func TestProjectRuntimeSetupUsesReviewedEditWithoutHandEditing(t *testing.T) {
	repo := copyProjectWorld(t)
	project, err := projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	mutation := projectwork.Mutation{
		APIVersion: projectwork.APIVersion,
		BaseDigest: project.Digest,
		Actor:      projectwork.HumanActor,
		Goal:       "Select the project-owned bounded agent runtime",
		Files: []projectwork.FileChange{{
			Path:    projectwork.RuntimePath,
			Content: "mode: controlled-local\n",
		}},
	}
	encoded, err := projectwork.EncodeMutation(mutation)
	if err != nil {
		t.Fatal(err)
	}
	path := writeDraft(t, repo, ".markitect/drafts/runtime-edit.json", encoded)
	plan := decodeOutput[projectwork.EditPlan](t, mustCLI(t, "edit", "--repo", repo, "--input", path))
	if plan.CandidateDigest == project.Digest || plan.Report.Digest != project.Report.Digest || len(plan.Mutation.Files) != 1 || plan.Mutation.Files[0].Path != projectwork.RuntimePath {
		t.Fatalf("runtime edit preview changed unexpected project data: %+v", plan)
	}
	mustCLI(t, "edit", "--repo", repo, "--input", path, "--expect", plan.Digest, "--write")
	updated, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(projectwork.RuntimePath)))
	if err != nil || string(updated) != mutation.Files[0].Content {
		t.Fatalf("runtime edit bytes=%q err=%v", updated, err)
	}
}

func TestProjectInitCreatesControlPlaneAndReadableDocumentationOnUnbornFeatureBranch(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "--initial-branch=feature-init")
	assertOnlyGit := func(stage string) {
		t.Helper()
		if entries, err := os.ReadDir(repo); err != nil || len(entries) != 1 || entries[0].Name() != ".git" {
			t.Fatalf("%s wrote files: entries=%v err=%v", stage, entries, err)
		}
	}
	preview := decodeOutput[projectwork.InitPlan](t, mustCLI(t, "init", "--repo", repo, "--name", "new-shop"))
	if preview.Digest == "" || len(preview.Written) != 0 {
		t.Fatalf("init preview = %+v", preview)
	}
	assertOnlyGit("init preview")
	if again := decodeOutput[projectwork.InitPlan](t, mustCLI(t, "init", "--repo", repo, "--name", "new-shop")); again.Digest != preview.Digest {
		t.Fatalf("init preview digest is not stable: %s != %s", again.Digest, preview.Digest)
	}
	if code, out, errout := runCLI(t, "init", "--repo", repo, "--name", "other-shop", "--expect", preview.Digest, "--write"); code != 2 || out != "" || !strings.Contains(errout, "markitect init:") {
		t.Fatalf("init write bound to another preview exit=%d stdout=%s stderr=%s", code, out, errout)
	}
	assertOnlyGit("stale init write")
	mustCLI(t, "init", "--repo", repo, "--name", "new-shop", "--expect", preview.Digest, "--write")
	for _, path := range []string{
		".markitect/project.yaml",
		".markitect/runtime.yaml",
		".markitect/model/manager.yaml",
		"docs/markitect/project.md",
		".markitect/.gitignore",
	} {
		info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("init did not create %s: %v", path, err)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("init target %s is not a regular file: %v", path, info.Mode())
		}
	}
	entries, err := os.ReadDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != ".git" && entry.Name() != ".markitect" && entry.Name() != "docs" {
			t.Fatalf("init wrote outside its control plane and documentation destination: %s", entry.Name())
		}
	}
}

func TestProjectPlanRejectsUncommittedSelectedInputs(t *testing.T) {
	repo := copyProjectWorld(t)
	sourceFile := filepath.Join(repo, "src", "shop", "orders", "order.py")
	contents, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceFile, append(contents, []byte("\n# uncommitted selected change\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	code, _, errout := runCLI(t, "plan", "--repo", repo, "--goal", "Implement a bounded order change",
		"--manager", `["project.markitect.example.org/v1alpha1","Manager","commerce.sales.orders","orders"]`)
	if code != 2 || !strings.Contains(errout, "selected project inputs differ from fixed base revision; commit accepted selected changes before planning") {
		t.Fatalf("dirty plan exit=%d stderr=%s", code, errout)
	}
	if _, err := os.Stat(filepath.Join(repo, ".markitect", "runs")); !os.IsNotExist(err) {
		t.Fatalf("rejected read-only plan created persisted run state: %v", err)
	}
}

func TestCheckReportsIncompleteCoverageWithNonzeroExit(t *testing.T) {
	repo := copyProjectWorld(t)
	if err := os.Remove(filepath.Join(repo, "docs", "cancellation.md")); err != nil {
		t.Fatal(err)
	}
	code, out, errout := runCLI(t, "check", "--repo", repo)
	if code != 1 || !strings.Contains(errout, "markitect check:") {
		t.Fatalf("incomplete project check exit=%d, want 1; stderr=%s", code, errout)
	}
	if result := decodeOutput[checkReport](t, []byte(out)); result.Status != "incomplete" {
		t.Fatalf("incomplete check result=%+v", result)
	}
}

func TestAdoptStagesKeepDeferredScopeOutOfModel(t *testing.T) {
	repo := copyProjectWorld(t)
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	request := projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion, ID: "cancel-scope-review",
		Purpose: "Capture cancellation documentation for initial model adoption", Review: "owner-review-2026-10-08",
		Commit: commit, ScopeRoots: []string{"docs"},
		Selected:   []projectadoption.SelectedPath{{ID: "cancellation-doc", Path: "docs/cancellation.md", Reason: "Owner-selected current behavior documentation", Basis: "documentation"}},
		Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{},
	}
	session := adoptStart(t, repo, commit, request)
	discovery, err := projectadoption.Discover(repo, request)
	if err != nil {
		t.Fatal(err)
	}
	target, err := projectwork.Load(repo, commit)
	if err != nil {
		t.Fatal(err)
	}
	schemaDigest, _, err := projectadoption.CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil {
		t.Fatal(err)
	}
	managerID := loaded.TargetContext.RootManagerID
	line := strings.Split(strings.ReplaceAll(discovery.Evidence[0].Content, "\r\n", "\n"), "\n")[0]
	evidence := []projectadoption.EvidenceRef{{EvidenceID: "cancellation-doc", StartLine: 1, EndLine: 1, Excerpt: line}}
	report := makeStagedReport(discovery, target, session.Session.TargetContextDigest, schemaDigest,
		[]projectadoption.Claim{
			{ID: "documented-cancellation", ScopeID: "orders", Kind: "documented-intent", Method: "documentation", Statement: "The selected documentation allows cancellation before shipment.", Evidence: evidence, Uncertainty: []string{}},
			{ID: "inventory-deferred", ScopeID: "inventory", Kind: "hypothesis", Method: "synthesis", Statement: "Inventory may be considered as a separate adoption scope.", Evidence: evidence, Uncertainty: []string{"No inventory-specific evidence was selected."}},
		},
		[]projectadoption.ScopeProposal{
			{ID: "orders", Name: "Orders", ClaimIDs: []string{"documented-cancellation"}},
			{ID: "inventory", Name: "Inventory", ClaimIDs: []string{"inventory-deferred"}},
		},
		[]projectadoption.ProposedFile{
			{ScopeID: "orders", Path: ".markitect/model/commerce/sales/orders/adopted-intent.yaml", Content: "apiVersion: project.markitect.example.org/v1alpha1\nkind: Statement\nmetadata:\n  name: adopted-intent\n  namespace: commerce.sales.orders\npurpose: Records owner-adopted cancellation intent.\nspec:\n  category: rule\n  description: A confirmed order may be cancelled before shipment.\n  public: true\n  uses: []\n  requires: []\n"},
			{ScopeID: "inventory", Path: ".markitect/model/commerce/sales/inventory/adopted-intent.yaml", Content: "apiVersion: project.markitect.example.org/v1alpha1\nkind: Statement\nmetadata:\n  name: adopted-intent\n  namespace: commerce.sales.inventory\npurpose: Proposed but deferred inventory scope.\nspec:\n  category: concept\n  description: Inventory responsibilities remain unadopted.\n  public: true\n  uses: []\n  requires: []\n"},
		})
	iterate := projectapp.BrownfieldIterateInput{
		Request:  projectadoption.ReverseIterationRequest{ID: "root-pass", ManagerID: managerID, EvidenceIDs: []string{"cancellation-doc"}, DelegationEvidenceIDs: []string{}, Purpose: "Model cancellation intent", Review: "manager-review-adoption"},
		Proposal: projectadoption.ManagerProposal{ManagerID: managerID, EvidenceIDs: []string{"cancellation-doc"}, Hierarchy: []projectadoption.ProposedManager{}, PublicContracts: []projectadoption.ManagerPublicContract{}, Report: report},
		Integration: &projectadoption.ManagerIntegration{ManagerID: managerID, ChildProposalDigests: []string{}, ChildContracts: []projectadoption.IntegratedChildContracts{},
			Report: report, Conflicts: []projectadoption.SessionConflict{}},
	}
	result := runBrownfieldMutation(t, repo, discovery.ID, "iterate", iterate, session.SessionDigest)
	if len(result.Session.Iterations) != 1 || result.Session.Iterations[0].IntegrationDigest == "" {
		t.Fatalf("iterate did not begin, propose and integrate in one step: %+v", result.Session.Iterations)
	}
	choices := projectapp.BrownfieldResolveInput{IterationID: "root-pass", Choices: projectapp.ResolutionChoices{
		Actor: "user", AuthorityClaim: "Project owner directed adoption of Orders", DecisionReference: "review-2026-10-08",
		Questions: []projectadoption.QuestionResolution{},
		Scopes: []projectadoption.ScopeResolution{
			{ScopeID: "orders", Status: "adopt", Reason: "Owner confirmed documented cancellation intent"},
			{ScopeID: "inventory", Status: "defer", Reason: "No inventory-specific evidence selected"},
		},
	}}
	result = runBrownfieldMutation(t, repo, discovery.ID, "resolve", choices, result.SessionDigest)
	plan := runBrownfieldPlan(t, repo, discovery.ID, "root-pass")
	if plan.Plan == nil || plan.Plan.Status != "partial" || len(plan.Plan.AdoptedScopes) != 1 || plan.Plan.AdoptedScopes[0] != "orders" || len(plan.Plan.DeferredScopes) != 1 || plan.Plan.DeferredScopes[0] != "inventory" {
		t.Fatalf("adoption plan did not preserve partial scope decision: %+v", plan.Plan)
	}
	apply := writeDraft(t, repo, ".markitect/drafts/apply-partial.json", map[string]any{"apply": projectapp.BrownfieldApplyAdoptionInput{IterationID: "root-pass", ExpectedPlanDigest: plan.Plan.PlanDigest}})
	mustCLI(t, "adopt", "apply", "--repo", repo, "--session", discovery.ID, "--input", apply, "--expect", result.SessionDigest, "--write")
	if _, err := os.Stat(filepath.Join(repo, ".markitect", "model", "commerce", "sales", "orders", "adopted-intent.yaml")); err != nil {
		t.Fatalf("adopted Orders model missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".markitect", "model", "commerce", "sales", "inventory", "adopted-intent.yaml")); !os.IsNotExist(err) {
		t.Fatalf("deferred Inventory model was written: %v", err)
	}
}

// adoptStart previews and writes `adopt start` from a discovery request, so
// the Host runs discovery itself, and returns the written result.
func adoptStart(t *testing.T, repo, commit string, request projectadoption.DiscoveryRequest) projectapp.BrownfieldResult {
	t.Helper()
	input := writeDraft(t, repo, ".markitect/drafts/start-"+request.ID+".json", map[string]any{"start": projectapp.BrownfieldStartInput{Request: request, ScopeStatuses: []projectadoption.ScopeStatus{}}})
	args := []string{"adopt", "start", "--repo", repo, "--revision", commit, "--input", input}
	preview := decodeOutput[projectapp.BrownfieldResult](t, mustCLI(t, args...))
	if preview.Status != "preview" || preview.Session == nil || preview.SessionDigest == "" {
		t.Fatalf("adopt start preview = %+v", preview)
	}
	written := decodeOutput[projectapp.BrownfieldResult](t, mustCLI(t, append(args, "--expect", preview.SessionDigest, "--write")...))
	if written.Status != "recorded" || written.SessionDigest != preview.SessionDigest || written.Session == nil {
		t.Fatalf("adopt start write = %+v", written)
	}
	return written
}

const failingManagerEnv = "MARKITECT_FAILING_MANAGER_HELPER"

// TestFailingManagerHelperProcess is the process executor that stands in for a
// failed Manager; it does nothing in an ordinary test run.
func TestFailingManagerHelperProcess(t *testing.T) {
	if os.Getenv(failingManagerEnv) != "1" {
		return
	}
	fmt.Fprintln(os.Stderr, "helper: simulated Manager failure")
	os.Exit(3)
}

// failingManagerRepo is a committed staged project whose only Manager runs the
// failing helper process.
func failingManagerRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	initPlan, err := projectwork.Init(root, "failing-manager", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range initPlan.Files {
		content := file.Content
		if file.Path == projectwork.ManifestPath {
			content = strings.Replace(content, "coverageMode: full", "coverageMode: selected", 1)
			// Staged (non-guided) mode: plan without an exploration scope.
			content = strings.Replace(content, "workflowMode: guided", "workflowMode: empty", 1)
		}
		path := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	managerID := core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Manager", Name: "project-owner"}.Key()
	runtimeConfig := projectrun.Runtime{
		APIVersion: projectrun.APIVersion, Mode: projectrun.ModeControlledLocal,
		Agents: map[string]projectrun.Agent{managerID: {
			Command: executable, Args: []string{"-test.run=^TestFailingManagerHelperProcess$"},
			Model: "failing-model", ProviderVersion: "failing/1", Environment: []string{failingManagerEnv},
			Timeout: projectrun.Duration(30 * time.Second), MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
			Pricing: projectrun.Pricing{InputMicrosPerMillion: 1},
		}},
		Limits: projectrun.Limits{MaxDepth: 2, MaxStarts: 4, MaxParallel: 1, MaxDuration: projectrun.Duration(time.Minute), MaxCostMicros: 1000,
			MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 1 << 20},
	}
	encoded, err := yaml.Marshal(runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(projectwork.RuntimePath)), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "init", "--initial-branch=feature-failing-manager")
	runGit(t, root, "add", "--all")
	runGitWithEnv(t, root, testCommitEnv, "commit", "-m", "fixture")
	return root
}

// A run whose Manager fails has completed with a durable outcome: the CLI
// prints the partial RunReport and exits 1, not with the usage code 2.
func TestRunPrintsPartialReportWhenAManagerFails(t *testing.T) {
	root := failingManagerRepo(t)
	t.Setenv(failingManagerEnv, "1") // bound into the runtime fingerprint at plan time
	preview := decodeOutput[planReport](t, mustCLI(t, "plan", "--repo", root, "--goal", "Exercise a failing Manager"))
	plan := decodeOutput[planReport](t, mustCLI(t, "plan", "--repo", root, "--goal", "Exercise a failing Manager", "--expect", preview.PreviewDigest, "--write"))
	if plan.ID == "" || !plan.ExecuteAuthorized {
		t.Fatalf("plan output: %+v", plan.PlanRecord)
	}
	code, out, errout := runCLI(t, "run", plan.ID, "--repo", root, "--execute")
	status, err := projectOperations().Status(projectapp.RunOperation{Root: root, RunID: plan.ID})
	if err != nil || (status.Run.Status != projectrun.StatusFailed && status.Run.Status != projectrun.StatusBlocked) {
		t.Fatalf("durable run is not failed/blocked; fixture problem: %+v %v (run exit=%d stderr=%s)", status.Run.Status, err, code, errout)
	}
	if report := decodeOutput[projectrun.RunReport](t, []byte(out)); report.ID != plan.ID || report.Status != status.Run.Status {
		t.Fatalf("partial report id=%q status=%q, want %q %q", report.ID, report.Status, plan.ID, status.Run.Status)
	}
	if code != 1 || !strings.Contains(errout, "markitect run:") {
		t.Fatalf("failed Manager run exit=%d stderr=%s, want 1 with the error on stderr", code, errout)
	}

	summary := decodeOutput[overview](t, mustCLI(t, "status", "--repo", root))
	if len(summary.Runs) != 1 || summary.Runs[0].ID != plan.ID || summary.Runs[0].Status != status.Run.Status || summary.Runs[0].Next != "repair" {
		t.Fatalf("status overview runs = %+v", summary.Runs)
	}
	if summary.Project.Name != "failing-manager" || summary.Project.Status != "succeeded" || !summary.Runtime.Configured {
		t.Fatalf("status overview project=%+v runtime=%+v", summary.Project, summary.Runtime)
	}
	if one := decodeOutput[projectrun.StatusReport](t, mustCLI(t, "status", plan.ID, "--repo", root)); one.Plan.ID != plan.ID || one.Run.Status != status.Run.Status {
		t.Fatalf("status RUN = plan %q run %q", one.Plan.ID, one.Run.Status)
	}
}

// plan --write persists exactly the reviewed preview: the previewDigest is
// stable across previews, and a different digest fails as stale.
func TestPlanWriteIsBoundToThePreviewDigest(t *testing.T) {
	root := failingManagerRepo(t)
	args := []string{"plan", "--repo", root, "--goal", "Bind the plan to its preview"}
	first := decodeOutput[planReport](t, mustCLI(t, args...))
	second := decodeOutput[planReport](t, mustCLI(t, args...))
	if first.PreviewDigest == "" || first.PreviewDigest != second.PreviewDigest {
		t.Fatalf("previewDigest is not stable across previews: %q != %q", first.PreviewDigest, second.PreviewDigest)
	}
	if first.ExecuteAuthorized {
		t.Fatalf("preview is marked as persisted by an authorized caller: %+v", first.PlanRecord)
	}
	assertRuns := func(want int) []projectrun.RunSummary {
		t.Helper()
		runs, err := projectrun.ListRuns(root)
		if err != nil || len(runs) != want {
			t.Fatalf("persisted runs = %+v err=%v, want %d", runs, err, want)
		}
		return runs
	}
	assertRuns(0)
	code, out, errout := runCLI(t, append(args, "--expect", "sha256:"+strings.Repeat("0", 64), "--write")...)
	if code != 2 || out != "" || !strings.Contains(errout, "markitect plan:") || !strings.Contains(errout, "stale") {
		t.Fatalf("plan write with a wrong digest exit=%d stdout=%s stderr=%s", code, out, errout)
	}
	assertRuns(0)
	written := decodeOutput[planReport](t, mustCLI(t, append(args, "--expect", first.PreviewDigest, "--write")...))
	if written.ID == "" || !written.ExecuteAuthorized || written.PreviewDigest != first.PreviewDigest {
		t.Fatalf("plan write = id %q authorized %v previewDigest %q", written.ID, written.ExecuteAuthorized, written.PreviewDigest)
	}
	if runs := assertRuns(1); runs[0].ID != written.ID || runs[0].Status != projectrun.StatusPlanned {
		t.Fatalf("persisted run = %+v, want plan %s", runs[0], written.ID)
	}
}

func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output))
}

func copyProjectWorld(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate project fixture test")
	}
	repository := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	sourceRoot := filepath.Join(repository, "examples", "project-world")
	destination := t.TempDir()
	err := filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, destination, "init", "--initial-branch=feature-project")
	runGit(t, destination, "add", ".")
	runGitWithEnv(t, destination, testCommitEnv, "commit", "-m", "fixture")
	return destination
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	runGitWithEnv(t, root, nil, args...)
}

func runGitWithEnv(t *testing.T, root string, environment []string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = append(os.Environ(), environment...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

// Every --revision and --since accepts what Git resolves to a commit, and the
// Host records the full commit ID it bound.
func TestRevisionsAcceptAnythingGitResolvesToACommit(t *testing.T) {
	repo := copyProjectWorld(t)
	head := gitOutput(t, repo, "rev-parse", "HEAD")
	short := gitOutput(t, repo, "rev-parse", "--short", "HEAD")
	for _, revision := range []string{"HEAD", short, head} {
		code, out, errout := runCLI(t, "check", "--repo", repo, "--revision", revision)
		var report struct {
			Revision string `json:"revision"`
		}
		if code != 0 || json.Unmarshal([]byte(out), &report) != nil || report.Revision != head {
			t.Fatalf("check --revision %s: exit=%d revision=%q stderr=%s", revision, code, report.Revision, errout)
		}
	}
	if code, out, errout := runCLI(t, "docs", "--repo", repo, "--revision", "HEAD"); code != 0 || !strings.Contains(out, `"revision": "`+head+`"`) {
		t.Fatalf("docs --revision HEAD: exit=%d stderr=%s out=%.200s", code, errout, out)
	}
	if code, _, errout := runCLI(t, "impact", "--repo", repo, "--since", "HEAD", "--revision", short); code != 0 {
		t.Fatalf("impact with HEAD and a short ID: exit=%d stderr=%s", code, errout)
	}
	code, out, errout := runCLI(t, "check", "--repo", repo, "--revision", "no-such-revision")
	if code != 2 || out != "" || !strings.Contains(errout, `--revision "no-such-revision" does not name a commit in this repository`) {
		t.Fatalf("unknown revision: exit=%d stdout=%q stderr=%s", code, out, errout)
	}
	server, err := newMCPServer(env{root: repo, ops: projectOperations()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.Call(context.Background(), "model", []byte(`{"revision":"HEAD"}`))
	if err != nil || result.IsError {
		t.Fatalf("MCP model with revision HEAD: %+v %v", result, err)
	}
	result, err = server.Call(context.Background(), "check", []byte(`{"revision":"no-such-revision"}`))
	if err != nil || !result.IsError || !strings.Contains(result.Content[0].Text, "invalid_arguments") {
		t.Fatalf("MCP unknown revision: %+v %v", result, err)
	}
}

// context accepts a Manager's short name when it is unique in the project.
func TestContextAcceptsAUniqueShortManagerName(t *testing.T) {
	repo := copyProjectWorld(t)
	code, out, errout := runCLI(t, "context", "orders", "--repo", repo)
	if code != 0 || !strings.Contains(out, `"commerce.sales.orders"`) {
		t.Fatalf("context orders: exit=%d stderr=%s out=%.200s", code, errout, out)
	}
	if code, _, errout := runCLI(t, "context", "no-such-manager", "--repo", repo); code != 2 || !strings.Contains(errout, `no Manager is named "no-such-manager"`) {
		t.Fatalf("unknown name: exit=%d stderr=%s", code, errout)
	}
	duplicate := filepath.Join(repo, ".markitect", "model", "engineering", "orders", "manager.yaml")
	if err := os.MkdirAll(filepath.Dir(duplicate), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(duplicate, []byte("apiVersion: project.markitect.example.org/v1alpha1\nkind: Manager\nmetadata:\n  name: orders\n  namespace: engineering.orders\npurpose: A second Manager with the same short name.\nspec:\n  parent:\n    namespace: engineering\n    name: engineering\n  owns: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(repo, ".markitect", "project.yaml")
	selected, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	selected = []byte(strings.Replace(string(selected), "modelFiles:\n", "modelFiles:\n  - .markitect/model/engineering/orders/manager.yaml\n", 1))
	if err := os.WriteFile(manifest, selected, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errout := runCLI(t, "context", "orders", "--repo", repo); code != 2 || !strings.Contains(errout, `Manager name "orders" is ambiguous`) {
		t.Fatalf("ambiguous name: exit=%d stderr=%s", code, errout)
	}
}
