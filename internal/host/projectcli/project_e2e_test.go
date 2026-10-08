package projectcli

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestProjectWorldFixtureAndReviewedEdit(t *testing.T) {
	repo := copyProjectWorld(t)
	initial, err := projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if initial.Report.Status != "succeeded" {
		t.Fatalf("fixture model status = %q, findings=%+v, unknown=%v", initial.Report.Status, initial.Report.Findings, initial.Report.Unknown)
	}
	if len(initial.Report.Managers) != 6 || len(initial.Report.Artifacts) != 2 || len(initial.Report.Checks) != 1 {
		t.Fatalf("fixture report counts = managers:%d artifacts:%d checks:%d", len(initial.Report.Managers), len(initial.Report.Artifacts), len(initial.Report.Checks))
	}

	var checkOut, checkErr bytes.Buffer
	if code := Run([]string{"project", "check", "--repo", repo}, &checkOut, &checkErr); code != 0 {
		t.Fatalf("project check exit=%d stderr=%s", code, checkErr.String())
	}
	var report struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(checkOut.Bytes(), &report); err != nil || report.Status != "succeeded" {
		t.Fatalf("check output report=%+v err=%v", report, err)
	}
	var documentOut, documentErr bytes.Buffer
	if code := Run([]string{"project", "document", "--repo", repo, "--write"}, &documentOut, &documentErr); code != 0 {
		t.Fatalf("project document --write exit=%d stderr=%s", code, documentErr.String())
	}
	view, err := os.ReadFile(filepath.Join(repo, ".markitect", "views", "project.md"))
	if err != nil || !bytes.Contains(view, []byte("cancel-before-shipped")) {
		t.Fatalf("generated project view omitted selected model: err=%v", err)
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
	var contextOut, contextErr bytes.Buffer
	if code := Run([]string{"project", "context", "--repo", repo, "--manager", manager}, &contextOut, &contextErr); code != 0 {
		t.Fatalf("project context exit=%d stderr=%s", code, contextErr.String())
	}
	if !bytes.Contains(contextOut.Bytes(), []byte("cancel-before-shipped")) {
		t.Fatalf("orders context omitted its cancellation rule: %s", contextOut.String())
	}

	const rulePath = ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml"
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rulePath)))
	if err != nil {
		t.Fatal(err)
	}
	mutation := projectwork.Mutation{
		APIVersion: projectwork.APIVersion,
		BaseDigest: initial.Digest,
		Actor:      projectwork.HumanActor,
		Goal:       "Clarify the cancellation transition",
		Files: []projectwork.FileChange{{
			Path:    rulePath,
			Content: strings.Replace(string(content), "Cancellation is valid only while the order is confirmed; a shipped order cannot be cancelled.", "Cancellation is valid only before shipment; shipped orders remain unchanged.", 1),
		}},
	}
	plan, err := projectwork.PlanEdit(initial, mutation)
	if err != nil {
		t.Fatal(err)
	}
	proposalBytes, err := projectwork.EncodeMutation(mutation)
	if err != nil {
		t.Fatal(err)
	}
	proposalPath := filepath.Join(repo, ".markitect", "drafts", "cancel-rule.json")
	if err := os.MkdirAll(filepath.Dir(proposalPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proposalPath, proposalBytes, 0644); err != nil {
		t.Fatal(err)
	}
	input := filepath.ToSlash(strings.TrimPrefix(proposalPath, repo+string(filepath.Separator)))
	var previewOut, previewErr bytes.Buffer
	if code := Run([]string{"project", "edit", "--repo", repo, "--input", input}, &previewOut, &previewErr); code != 0 {
		t.Fatalf("edit preview exit=%d stderr=%s", code, previewErr.String())
	}
	var preview projectwork.EditPlan
	if err := json.Unmarshal(previewOut.Bytes(), &preview); err != nil || preview.Digest != plan.Digest {
		t.Fatalf("preview digest=%q want %q err=%v", preview.Digest, plan.Digest, err)
	}

	changed := mutation
	changed.Goal = "Replace the previously reviewed proposal"
	changedBytes, err := projectwork.EncodeMutation(changed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proposalPath, changedBytes, 0644); err != nil {
		t.Fatal(err)
	}
	var staleOut, staleErr bytes.Buffer
	if code := Run([]string{"project", "edit", "--repo", repo, "--input", input, "--expect", plan.Digest, "--write"}, &staleOut, &staleErr); code == 0 || !strings.Contains(staleErr.String(), "exact edit plan digest") {
		t.Fatalf("changed proposal unexpectedly passed reviewed digest: exit=%d stderr=%s", code, staleErr.String())
	}
	current, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rulePath)))
	if err != nil || !bytes.Equal(current, content) {
		t.Fatalf("stale proposal changed model bytes: err=%v", err)
	}

	if err := os.WriteFile(proposalPath, proposalBytes, 0644); err != nil {
		t.Fatal(err)
	}
	var applyOut, applyErr bytes.Buffer
	if code := Run([]string{"project", "edit", "--repo", repo, "--input", input, "--expect", plan.Digest, "--write"}, &applyOut, &applyErr); code != 0 {
		t.Fatalf("reviewed edit exit=%d stderr=%s", code, applyErr.String())
	}
	updated, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rulePath)))
	if err != nil || bytes.Equal(updated, content) || !bytes.Contains(updated, []byte("before shipment")) {
		t.Fatalf("reviewed edit was not applied: err=%v content=%s", err, updated)
	}
}

func TestProjectInitCreatesOnlyMarkitectFilesOnUnbornFeatureBranch(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "--initial-branch=feature-init")
	var out, errout bytes.Buffer
	if code := Run([]string{"project", "init", "--repo", repo, "--name", "new-shop"}, &out, &errout); code != 0 {
		t.Fatalf("init preview exit=%d stderr=%s", code, errout.String())
	}
	if entries, err := os.ReadDir(repo); err != nil || len(entries) != 1 || entries[0].Name() != ".git" {
		t.Fatalf("init preview wrote files: entries=%v err=%v", entries, err)
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"project", "init", "--repo", repo, "--name", "new-shop", "--write"}, &out, &errout); code != 0 {
		t.Fatalf("init write exit=%d stderr=%s", code, errout.String())
	}
	for _, path := range []string{
		".markitect/project.yaml",
		".markitect/runtime.yaml",
		".markitect/model/manager.yaml",
		".markitect/views/project.md",
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
		if entry.Name() != ".git" && entry.Name() != ".markitect" {
			t.Fatalf("init wrote outside .markitect: %s", entry.Name())
		}
	}
}

func TestCheckReportsIncompleteCoverageWithNonzeroExit(t *testing.T) {
	repo := copyProjectWorld(t)
	if err := os.Remove(filepath.Join(repo, "docs", "cancellation.md")); err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	code := Run([]string{"project", "check", "--repo", repo}, &out, &errout)
	if code != 1 {
		t.Fatalf("incomplete project check exit=%d, want 1; stderr=%s", code, errout.String())
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.Status != "incomplete" {
		t.Fatalf("incomplete check result=%+v err=%v", result, err)
	}
}

func TestProjectAdoptionCommandsKeepDeferredScopeOutOfModel(t *testing.T) {
	repo := copyProjectWorld(t)
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	bindingsSchema, bindingsBuild, err := projectadoption.CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	request := projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion,
		ID:         "cancel-scope-review",
		Purpose:    "Capture cancellation documentation for initial model adoption",
		Review:     "owner-review-2026-10-08",
		Commit:     commit,
		ScopeRoots: []string{"docs"},
		Selected: []projectadoption.SelectedPath{{
			ID: "cancellation-doc", Path: "docs/cancellation.md", Reason: "Owner-selected current behavior documentation", Basis: "documentation",
		}},
		Exclusions: []projectadoption.PathReason{},
		Unselected: []projectadoption.PathReason{},
	}
	requestBytes, err := projectadoption.EncodeDiscoveryRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/discovery-request.json", requestBytes); err != nil {
		t.Fatal(err)
	}
	var discoveryOut, discoveryErr bytes.Buffer
	if code := Run([]string{"project", "discover", "--repo", repo, "--request", ".markitect/drafts/discovery-request.json", "--output", ".markitect/drafts/discovery.json"}, &discoveryOut, &discoveryErr); code != 0 {
		t.Fatalf("discover exit=%d stderr=%s", code, discoveryErr.String())
	}
	discoveryBytes, err := readRecord(repo, ".markitect/drafts/discovery.json")
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := projectadoption.DecodeDiscovery(discoveryBytes)
	if err != nil {
		t.Fatal(err)
	}
	target, err := projectwork.Load(repo, commit)
	if err != nil {
		t.Fatal(err)
	}
	proposal := projectadoption.ModelProposal{
		Goal: "Adopt confirmed order cancellation while deferring inventory scope",
		Files: []projectadoption.ProposedFile{
			{ScopeID: "orders", Path: ".markitect/model/commerce/sales/orders/adopted-intent.yaml", Content: "apiVersion: project.markitect.example.org/v1alpha1\nkind: Statement\nmetadata:\n  name: adopted-intent\n  namespace: commerce.sales.orders\npurpose: Records owner-adopted cancellation intent.\nspec:\n  category: rule\n  description: A confirmed order may be cancelled before shipment.\n  public: true\n  uses: []\n  requires: []\n"},
			{ScopeID: "inventory", Path: ".markitect/model/commerce/sales/inventory/adopted-intent.yaml", Content: "apiVersion: project.markitect.example.org/v1alpha1\nkind: Statement\nmetadata:\n  name: adopted-intent\n  namespace: commerce.sales.inventory\npurpose: Proposed but deferred inventory scope.\nspec:\n  category: concept\n  description: Inventory responsibilities remain unadopted.\n  public: true\n  uses: []\n  requires: []\n"},
		},
	}
	report := projectadoption.Distillation{
		APIVersion:      projectadoption.DistillationVersion,
		DiscoveryDigest: discovery.Digest,
		Method:          "human-review",
		SchemaDigest:    bindingsSchema,
		Claims: []projectadoption.Claim{
			{ID: "documented-cancellation", ScopeID: "orders", Kind: "documented-intent", Method: "documentation", Statement: "The selected documentation allows cancellation before shipment.", Evidence: []projectadoption.EvidenceRef{{EvidenceID: "cancellation-doc", StartLine: 3, EndLine: 3, Excerpt: "A confirmed order can be cancelled before shipment."}}, Uncertainty: []string{}},
			{ID: "inventory-deferred", ScopeID: "inventory", Kind: "hypothesis", Method: "synthesis", Statement: "Inventory may be considered as a separate adoption scope.", Evidence: []projectadoption.EvidenceRef{{EvidenceID: "cancellation-doc", StartLine: 3, EndLine: 3, Excerpt: "A confirmed order can be cancelled before shipment."}}, Uncertainty: []string{"No inventory-specific evidence was selected."}},
		},
		Terms:          []projectadoption.Term{},
		Contradictions: []projectadoption.Contradiction{},
		Questions:      []projectadoption.Question{},
		Scopes: []projectadoption.ScopeProposal{
			{ID: "orders", Name: "Orders", ClaimIDs: []string{"documented-cancellation"}},
			{ID: "inventory", Name: "Inventory", ClaimIDs: []string{"inventory-deferred"}},
		},
		Proposal: proposal,
	}
	projectadoption.SealDistillation(&report)
	reportBytes, err := projectadoption.EncodeDistillation(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/distillation.json", reportBytes); err != nil {
		t.Fatal(err)
	}
	var distillOut, distillErr bytes.Buffer
	if code := Run([]string{"project", "distill", "--repo", repo, "--discovery", ".markitect/drafts/discovery.json", "--report", ".markitect/drafts/distillation.json", "--output", ".markitect/drafts/validated-distillation.json"}, &distillOut, &distillErr); code != 0 {
		t.Fatalf("distill exit=%d stderr=%s", code, distillErr.String())
	}
	resolution := projectadoption.Resolution{
		APIVersion: projectadoption.ResolutionVersion, DiscoveryDigest: discovery.Digest,
		DistillationDigest: report.Digest, ProposalDigest: projectadoption.ProposalDigest(proposal),
		TargetBasis: target.Digest, SchemaDigest: bindingsSchema, BuildDigest: bindingsBuild,
		Actor: "user", AuthorityClaim: "Project owner directed adoption of Orders",
		DecisionReference: "review-2026-10-08", Authenticated: boolPointer(false),
		Questions: []projectadoption.QuestionResolution{},
		Scopes: []projectadoption.ScopeResolution{
			{ScopeID: "orders", Status: "adopt", Reason: "Owner confirmed documented cancellation intent"},
			{ScopeID: "inventory", Status: "defer", Reason: "No inventory-specific evidence selected"},
		},
	}
	projectadoption.SealResolution(&resolution)
	resolutionBytes, err := projectadoption.EncodeResolution(resolution)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/resolution.json", resolutionBytes); err != nil {
		t.Fatal(err)
	}
	args := []string{"project", "adopt", "--repo", repo, "--source-repo", repo, "--revision", commit, "--discovery", ".markitect/drafts/discovery.json", "--report", ".markitect/drafts/distillation.json", "--resolution", ".markitect/drafts/resolution.json"}
	var previewOut, previewErr bytes.Buffer
	previewArgs := append(append([]string(nil), args...), "--output", ".markitect/drafts/adoption-plan.json")
	if code := Run(previewArgs, &previewOut, &previewErr); code != 0 {
		t.Fatalf("adoption preview exit=%d stderr=%s", code, previewErr.String())
	}
	planBytes, err := readRecord(repo, ".markitect/drafts/adoption-plan.json")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := projectadoption.DecodeAdoptionPlan(planBytes)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "partial" || len(plan.AdoptedScopes) != 1 || plan.AdoptedScopes[0] != "orders" || len(plan.DeferredScopes) != 1 || plan.DeferredScopes[0] != "inventory" {
		t.Fatalf("adoption plan did not preserve partial scope decision: %+v", plan)
	}
	applyArgs := append(append([]string(nil), args...), "--plan", ".markitect/drafts/adoption-plan.json", "--expect", plan.PlanDigest, "--write")
	var applyOut, applyErr bytes.Buffer
	if code := Run(applyArgs, &applyOut, &applyErr); code != 0 {
		t.Fatalf("adoption apply exit=%d stderr=%s", code, applyErr.String())
	}
	if _, err := os.Stat(filepath.Join(repo, ".markitect", "model", "commerce", "sales", "orders", "adopted-intent.yaml")); err != nil {
		t.Fatalf("adopted Orders model missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".markitect", "model", "commerce", "sales", "inventory", "adopted-intent.yaml")); !os.IsNotExist(err) {
		t.Fatalf("deferred Inventory model was written: %v", err)
	}
}

func boolPointer(value bool) *bool { return &value }

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
	repository := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
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
	runGitWithEnv(t, destination, []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "commit", "-m", "fixture")
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
