package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/modules/projections"
	"go.yaml.in/yaml/v3"
)

const (
	projectionConfigPath   = "projections.config"
	projectionCoveragePath = "markitect-artifacts.yaml"
	projectionTargetPath   = "docs/representations/summary.md"
)

func TestProjectionCLIRequiresExplicitPathsAndWriteAuthorization(t *testing.T) {
	repo := newProjectionCLIRepo(t)

	code, _, stderr := invoke("projection", "--action", "observe", "--repo", repo.root, "--coverage", projectionCoveragePath)
	if code != 2 || !strings.Contains(stderr, "explicit --config and --coverage") {
		t.Fatalf("projection without explicit config exit=%d stderr=%q", code, stderr)
	}

	configArgs := []string{"--repo", repo.root, "--config", projectionConfigPath, "--coverage", projectionCoveragePath}
	code, _, stderr = invoke(append([]string{"projection", "--action", "apply"}, append(configArgs, "--plan", filepath.Join(t.TempDir(), "plan.yaml"))...)...)
	if code != 2 || !strings.Contains(stderr, "projection apply requires --write and --plan") {
		t.Fatalf("projection apply without --write exit=%d stderr=%q", code, stderr)
	}

	code, _, stderr = invoke("projection", "--action", "observe", "--write", "--repo", repo.root, "--config", projectionConfigPath, "--coverage", projectionCoveragePath)
	if code != 2 || !strings.Contains(stderr, "--write supports") {
		t.Fatalf("projection observe with --write exit=%d stderr=%q", code, stderr)
	}
}

func TestProjectionVerifyRequiresFullImmutableRevision(t *testing.T) {
	repo := newProjectionCLIRepo(t)
	code, _, stderr := invoke("projection", "--action", "verify", "--repo", repo.root,
		"--revision", "abc", "--config", projectionConfigPath, "--coverage", projectionCoveragePath)
	if code != 2 || !strings.Contains(stderr, "projection verify requires a full immutable --revision") {
		t.Fatalf("short projection revision exit=%d stderr=%q", code, stderr)
	}
}

func TestProjectionApplyRequiresCompleteReviewedCandidate(t *testing.T) {
	repo := newProjectionCLIRepo(t)
	configArgs := []string{"--repo", repo.root, "--config", projectionConfigPath, "--coverage", projectionCoveragePath}

	code, planText, stderr := invoke(append([]string{"projection", "--action", "plan"}, configArgs...)...)
	if code != 0 {
		t.Fatalf("projection plan exit=%d stderr=%s output=%s", code, stderr, planText)
	}
	planPath := filepath.Join(t.TempDir(), "plan.yaml")
	if err := os.WriteFile(planPath, []byte(planText), 0600); err != nil {
		t.Fatal(err)
	}
	plan := decodeYAML[projections.Plan](t, planText)
	if plan.PlanDigest == "" || plan.Status != projections.StatusDrift || len(plan.Contracts) != 1 || plan.Contracts[0].Status != projections.StatusIncomplete {
		t.Fatalf("unexpected AI projection plan: %#v", plan)
	}

	candidatePath := filepath.Join(t.TempDir(), "candidate.yaml")
	candidateBytes, err := yaml.Marshal(host.Materialization{
		APIVersion: host.MaterializationVersion,
		PlanDigest: plan.PlanDigest,
		Files:      map[string]string{projectionTargetPath: "# Reviewed summary\n"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidatePath, candidateBytes, 0600); err != nil {
		t.Fatal(err)
	}
	reviewed, err := host.ReadMaterialization(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	expect := host.MaterializationRecordDigest(reviewed)
	if expect == "" {
		t.Fatal("candidate record digest is empty")
	}

	applyArgs := append([]string{"projection", "--action", "apply", "--write"}, configArgs...)
	applyArgs = append(applyArgs, "--plan", planPath, "--report", candidatePath, "--expect", expect)
	code, output, stderr := invoke(applyArgs...)
	if code != 0 {
		t.Fatalf("reviewed projection apply exit=%d stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[host.ProjectionReport](t, output)
	if result.Status != "materialized-unverified" || len(result.Written) != 1 || result.Written[0] != projectionTargetPath {
		t.Fatalf("apply did not report the exact reviewed write as unverified: %#v", result)
	}
	actual, err := os.ReadFile(filepath.Join(repo.root, filepath.FromSlash(projectionTargetPath)))
	if err != nil || string(actual) != "# Reviewed summary\n" {
		t.Fatalf("reviewed target bytes differ: data=%q err=%v", actual, err)
	}
}

func TestRegisteredProjectionIsReportedByCheckAndVerifiedByVerify(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go unavailable for the declared orchestration-only check")
	}
	repo := newProjectionCLIRepo(t)

	code, output, stderr := invoke("check", "--repo", repo.root, "--revision", repo.base)
	if code != 0 {
		t.Fatalf("structural check exit=%d stderr=%s output=%s", code, stderr, output)
	}
	check := decodeYAML[report](t, output)
	if check.Status != "passed" || check.Projections == nil || check.Projections.Status == "" {
		t.Fatalf("check lost structural pass or registered projection status: %#v", check)
	}
	if !strings.Contains(check.Coverage, "registered projection state is separately reported") {
		t.Fatalf("check did not explain the separate projection status: %q", check.Coverage)
	}

	code, output, stderr = invoke("verify", "--repo", repo.root, "--revision", repo.base)
	if code != 1 {
		t.Fatalf("verify exit=%d, want projection failure from the missing AI target (1); stderr=%s output=%s", code, stderr, output)
	}
	verified := decodeYAML[report](t, output)
	if verified.Status != "failed" || verified.Projections == nil || verified.Projections.Status != projections.StatusDrift {
		t.Fatalf("verify silently skipped the registered missing AI target: %#v", verified)
	}
	if len(verified.Gates) != 1 || verified.Gates[0].Name != "go-version" || verified.Gates[0].ExitCode != 0 {
		t.Fatalf("verify did not run the declared orchestration check: %#v", verified.Gates)
	}
	if !strings.Contains(verified.Coverage, "check does not establish AI semantics") {
		t.Fatalf("verify made an unsupported semantic claim: %q", verified.Coverage)
	}
}

func newProjectionCLIRepo(t *testing.T) cliRepo {
	t.Helper()
	root, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-b", "projection-fixture")
	git(t, root, "config", "user.name", "Markitect Projection Test")
	git(t, root, "config", "user.email", "markitect-projection-test@example.invalid")

	project := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "projection-fixture"}}, Spec: authoring.Spec{
		Areas:   []authoring.Area{{Name: "engineering", Path: "docs/general"}},
		Targets: []string{"markdown"},
		Checks:  []authoring.Check{{Name: "go-version", Run: []string{"go", "version"}}},
		Adapters: []authoring.AdapterConfig{{Name: "local-representations", Type: "local-projection", Version: "v1alpha1", Config: map[string]any{
			"contracts": projectionConfigPath,
			"coverage":  projectionCoveragePath,
		}}},
	}}
	rule := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "policy", Namespace: "engineering"}}, Spec: authoring.Spec{Text: "Keep declared intent."}}
	resources := []*authoring.Resource{&project, &rule}
	names := []string{"markitect.yaml", "docs/general/rules/policy.yaml"}
	files := map[string][]byte{}
	for i, resource := range resources {
		resource.Path = names[i]
		data, err := authoring.Encode(resource)
		if err != nil {
			t.Fatal(err)
		}
		files[names[i]] = data
	}
	files[projectionConfigPath] = []byte("apiVersion: markitect.example.org/projections/v1alpha1\nversion: \"1\"\ncontracts:\n  - id: rule-summary\n    sources: [engineering/Rule/policy]\n    representation: concise summary\n    materializer: {name: external-agent, version: \"1\", mode: ai}\n    targets: [{path: docs/representations/summary.md}]\n    verificationChecks: [go-version]\n")
	files[projectionCoveragePath] = []byte("apiVersion: markitect.example.org/artifact-coverage/v1alpha1\nkind: ArtifactCoverage\nspec:\n  roots: [markitect.yaml, docs/general, projections.config, markitect-artifacts.yaml, docs/markitect, docs/representations]\n  tooling:\n    - path: markitect-artifacts.yaml\n      owner: projection-test-coverage\n")
	for name, data := range files {
		writeRepoFile(t, root, name, data)
	}
	loaded, err := host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := host.GenerateOutputs(loaded)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range outputs {
		writeRepoFile(t, root, name, data)
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "projection CLI fixture")
	return cliRepo{root: root, base: git(t, root, "rev-parse", "HEAD")}
}
