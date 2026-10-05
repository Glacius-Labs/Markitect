package host

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/consumers/githooks"
	"github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/consumers/pipelines"
	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"go.yaml.in/yaml/v3"
)

const (
	moduleChecksOwner           = "docs/Rule/repository-checks"
	moduleChecksHooksConfig     = "docs/module-config/hooks.config"
	moduleChecksPipelinesConfig = "docs/module-config/pipelines.config"
	moduleChecksHookPath        = ".githooks/pre-commit"
	moduleChecksPipelinePath    = ".github/workflows/tests.yaml"
)

type moduleChecksFixtureOptions struct {
	staleHookDigest bool
	omitHookInput   bool
	missingHook     bool
}

type moduleChecksFixture struct {
	root      string
	revision  string
	hooks     string
	pipelines string
	hookPath  string
}

func TestCheckModulesUsesOneFixedSnapshotAndDoesNotWrite(t *testing.T) {
	fixture := newModuleChecksFixture(t, moduleChecksFixtureOptions{})
	fixed, err := source.Load(fixture.root, fixture.revision)
	if err != nil {
		t.Fatal(err)
	}
	dirty := []byte("working-tree hook differs from selected revision\n")
	if err := os.WriteFile(filepath.Join(fixture.root, filepath.FromSlash(fixture.hookPath)), dirty, 0644); err != nil {
		t.Fatal(err)
	}
	statusBefore := moduleGitOutput(t, fixture.root, "status", "--porcelain")
	options := ModuleChecksOptions{Root: fixture.root, Revision: fixture.revision, HooksConfigPath: fixture.hooks, PipelinesPath: fixture.pipelines}
	report, err := CheckModules(options)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != ModuleChecksPassed || report.SnapshotID != fixture.revision || report.SnapshotDigest != fixed.Digest() || report.ModelDigest == "" {
		t.Fatalf("unexpected fixed-snapshot result: %#v", report)
	}
	if len(report.Modules) != 2 || report.Modules[0].Name != "git-hooks" || report.Modules[1].Name != "pipelines" {
		t.Fatalf("module report order/provenance is unstable: %#v", report.Modules)
	}
	hooks := report.Modules[0]
	if hooks.Status != githooks.StatusPassed || hooks.ConfigPath != fixture.hooks || hooks.ConfigDigest == "" || hooks.GitHooks == nil || hooks.GitHooks.SnapshotID != fixture.revision {
		t.Fatalf("git hooks result lost its status or provenance: %#v", hooks)
	}
	pipelinesResult := report.Modules[1]
	if pipelinesResult.Status != pipelines.StatusPassed || pipelinesResult.ConfigPath != fixture.pipelines || pipelinesResult.ConfigDigest == "" || pipelinesResult.Pipelines == nil || pipelinesResult.Pipelines.CheckReferencesChecked != 1 {
		t.Fatalf("pipeline result lost its status, provenance, or literal check fact: %#v", pipelinesResult)
	}
	actual, err := os.ReadFile(filepath.Join(fixture.root, filepath.FromSlash(fixture.hookPath)))
	if err != nil || !bytes.Equal(actual, dirty) {
		t.Fatalf("module check changed the working tree artifact: data=%q err=%v", actual, err)
	}
	if statusAfter := moduleGitOutput(t, fixture.root, "status", "--porcelain"); statusAfter != statusBefore {
		t.Fatalf("module check changed repository state: before=%q after=%q", statusBefore, statusAfter)
	}
}

func TestCheckModulesReportsStaleArtifactDigestAndOmittedModule(t *testing.T) {
	fixture := newModuleChecksFixture(t, moduleChecksFixtureOptions{staleHookDigest: true})
	report, err := CheckModules(ModuleChecksOptions{Root: fixture.root, Revision: fixture.revision, HooksConfigPath: fixture.hooks})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != ModuleChecksFindings || len(report.Modules) != 2 {
		t.Fatalf("stale digest did not fail the combined report: %#v", report)
	}
	if report.Modules[0].Status != githooks.StatusFindings || report.Modules[0].GitHooks == nil || len(report.Modules[0].GitHooks.Findings) == 0 || report.Modules[0].GitHooks.Findings[0].Code != "hooks.digest.mismatch" {
		t.Fatalf("stale digest finding/provenance was lost: %#v", report.Modules[0])
	}
	if report.Modules[1].Name != "pipelines" || report.Modules[1].Status != pipelines.StatusNotConfigured || report.Modules[1].Pipelines != nil {
		t.Fatalf("omitted pipelines module was not reported as not-configured: %#v", report.Modules[1])
	}
}

func TestCheckModulesRejectsMissingConfigArtifactAndUnownedInputs(t *testing.T) {
	t.Run("missing config", func(t *testing.T) {
		fixture := newModuleChecksFixture(t, moduleChecksFixtureOptions{})
		_, err := CheckModules(ModuleChecksOptions{Root: fixture.root, Revision: fixture.revision, HooksConfigPath: "docs/module-config/missing.yaml"})
		if err == nil || !strings.Contains(err.Error(), "is absent from the selected snapshot") {
			t.Fatalf("missing module config error = %v", err)
		}
	})

	t.Run("missing declared artifact", func(t *testing.T) {
		fixture := newModuleChecksFixture(t, moduleChecksFixtureOptions{missingHook: true})
		_, err := CheckModules(ModuleChecksOptions{Root: fixture.root, Revision: fixture.revision, HooksConfigPath: fixture.hooks})
		if err == nil || !strings.Contains(err.Error(), "strict project check failed") || !strings.Contains(err.Error(), "input.missing") {
			t.Fatalf("missing configured artifact did not fail strict input validation: %v", err)
		}
	})

	t.Run("unowned artifact", func(t *testing.T) {
		fixture := newModuleChecksFixture(t, moduleChecksFixtureOptions{omitHookInput: true})
		_, err := CheckModules(ModuleChecksOptions{Root: fixture.root, Revision: fixture.revision, HooksConfigPath: fixture.hooks})
		if err == nil || !strings.Contains(err.Error(), "not an explicitly declared input of a canonical resource") {
			t.Fatalf("unowned configured artifact error = %v", err)
		}
	})

	t.Run("unowned config", func(t *testing.T) {
		fixture := newModuleChecksFixture(t, moduleChecksFixtureOptions{})
		unownedPath := "unmanaged/hooks.yaml"
		original, err := os.ReadFile(filepath.Join(fixture.root, filepath.FromSlash(fixture.hooks)))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, fixture.root, map[string][]byte{unownedPath: original})
		gitInRepo(t, fixture.root, "add", unownedPath)
		gitInRepo(t, fixture.root, "commit", "-m", "add unowned module config fixture")
		fixed, err := source.Load(fixture.root, "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		_, err = CheckModules(ModuleChecksOptions{Root: fixture.root, Revision: fixed.ID, HooksConfigPath: unownedPath})
		if err == nil || !strings.Contains(err.Error(), "is not an explicitly declared input of a canonical resource") {
			t.Fatalf("unowned module config error = %v", err)
		}
	})
}

func newModuleChecksFixture(t *testing.T, options moduleChecksFixtureOptions) moduleChecksFixture {
	t.Helper()
	root := t.TempDir()
	hookPath := moduleChecksHookPath
	if options.missingHook {
		hookPath = ".githooks/missing-hook"
	}
	hookBytes := []byte("#!/bin/sh\nprintf 'ready\\n'\n")
	workflowBytes := []byte("name: Tests\njobs:\n  build:\n    steps:\n      - run: go test ./...\n")
	hookDigest := moduleTestDigest(hookBytes)
	if options.staleHookDigest {
		hookDigest = strings.Repeat("0", 64)
	}
	hookConfig, err := yaml.Marshal(githooks.Config{APIVersion: githooks.ConfigVersion, Hooks: []githooks.Hook{{Name: "pre-commit-tests", Stage: "pre-commit", Path: hookPath, Digest: hookDigest, Owner: moduleChecksOwner}}})
	if err != nil {
		t.Fatal(err)
	}
	pipelineConfig, err := yaml.Marshal(pipelines.Config{APIVersion: pipelines.ConfigVersion, Pipelines: []pipelines.Pipeline{{Name: "github-tests", Provider: "github-actions", Path: moduleChecksPipelinePath, Digest: moduleTestDigest(workflowBytes), Owner: moduleChecksOwner, ExpectedChecks: []pipelines.ExpectedCheck{{Name: "go-tests", YAMLPath: "/jobs/build/steps/0/run"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	files := []string{moduleChecksHooksConfig, moduleChecksPipelinesConfig, moduleChecksPipelinePath}
	if !options.omitHookInput {
		files = append(files, hookPath)
	}
	project := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "sample"}}, Spec: authoring.Spec{
		Areas: []authoring.Area{
			{Name: "docs", Path: "docs", Imports: []string{"hooks", "workflows"}},
			{Name: "hooks", Path: ".githooks"},
			{Name: "workflows", Path: ".github"},
		},
		Checks: []authoring.Check{{Name: "go-tests", Run: []string{"go", "test", "./..."}}},
	}}
	rule := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "repository-checks", Namespace: "docs"}}, Spec: authoring.Spec{Text: "Keep repository checks explicit.", Files: files}}
	projectBytes, err := authoring.Encode(&project)
	if err != nil {
		t.Fatal(err)
	}
	ruleBytes, err := authoring.Encode(&rule)
	if err != nil {
		t.Fatal(err)
	}
	allFiles := map[string][]byte{
		"markitect.yaml":                 projectBytes,
		"docs/general/rules/checks.yaml": ruleBytes,
		moduleChecksHooksConfig:          hookConfig,
		moduleChecksPipelinesConfig:      pipelineConfig,
		moduleChecksPipelinePath:         workflowBytes,
	}
	if !options.missingHook {
		allFiles[moduleChecksHookPath] = hookBytes
	}
	writeFixture(t, root, allFiles)
	initAppTestRepo(t, root)
	gitInRepo(t, root, "config", "user.name", "Markitect Test")
	gitInRepo(t, root, "config", "user.email", "markitect-test@example.invalid")
	gitInRepo(t, root, "add", "-A")
	gitInRepo(t, root, "commit", "-m", "module checks fixture")
	fixed, err := source.Load(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return moduleChecksFixture{root: root, revision: fixed.ID, hooks: moduleChecksHooksConfig, pipelines: moduleChecksPipelinesConfig, hookPath: moduleChecksHookPath}
}

func moduleTestDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func moduleGitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-c", "safe.directory=" + filepath.ToSlash(root), "-C", root}, args...)
	command := exec.Command("git", commandArgs...)
	command.Env = source.CleanGitEnv()
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, data)
	}
	return strings.TrimSpace(string(data))
}
