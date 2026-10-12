package modulechecks

import "go.yaml.in/yaml/v3"

import "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/githooks"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/pipelines"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

const (
	testHooksConfig     = ".markitect/modules/githooks.config"
	testPipelinesConfig = ".markitect/modules/pipelines.config"
	testHookPath        = ".githooks/pre-commit"
	testWorkflowPath    = ".github/workflows/ci.yaml"
)

func TestMain(m *testing.M) { testkit.Main(m) }

type moduleFixture struct {
	repo     *testkit.Repo
	revision string
	owner    string
	hook     []byte
	workflow []byte
}

func TestCheckUsesOneFixedSnapshotAndDoesNotWrite(t *testing.T) {
	fixture := newModuleFixture(t, true, true)
	fixed, err := source.Load(fixture.repo.Dir, fixture.revision)
	if err != nil {
		t.Fatal(err)
	}
	dirty := []byte("working-tree hook differs from selected revision\n")
	fixture.repo.Write(testHookPath, string(dirty))
	statusBefore := fixture.repo.Git("status", "--porcelain")

	report, err := Check(Options{Root: fixture.repo.Dir, Revision: fixture.revision, HooksConfigPath: testHooksConfig, PipelinesPath: testPipelinesConfig})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusPassed || report.SnapshotID != fixture.revision || report.SnapshotDigest != fixed.Digest() || report.ModelDigest == "" {
		t.Fatalf("unexpected fixed-snapshot result: %#v", report)
	}
	if len(report.Modules) != 2 || report.Modules[0].Name != "git-hooks" || report.Modules[1].Name != "pipelines" {
		t.Fatalf("module report order/provenance is unstable: %#v", report.Modules)
	}
	if got := report.Modules[0]; got.Status != githooks.StatusPassed || got.ConfigPath != testHooksConfig || got.ConfigDigest == "" || got.GitHooks == nil || got.GitHooks.SnapshotID != fixture.revision {
		t.Fatalf("hook result lost status or provenance: %#v", got)
	}
	if got := report.Modules[1]; got.Status != pipelines.StatusPassed || got.ConfigPath != testPipelinesConfig || got.ConfigDigest == "" || got.Pipelines == nil || got.Pipelines.CheckReferencesChecked != 1 {
		t.Fatalf("pipeline result lost status, provenance or literal Check fact: %#v", got)
	}
	actual, err := os.ReadFile(filepath.Join(fixture.repo.Dir, filepath.FromSlash(testHookPath)))
	if err != nil || !bytes.Equal(actual, dirty) {
		t.Fatalf("check changed the working-tree artifact: bytes=%q err=%v", actual, err)
	}
	if statusAfter := fixture.repo.Git("status", "--porcelain"); statusAfter != statusBefore {
		t.Fatalf("check changed repository state: before=%q after=%q", statusBefore, statusAfter)
	}
}

func TestCheckReportsStaleArtifactAndOmittedModule(t *testing.T) {
	fixture := newModuleFixture(t, true, true)
	configBytes := encodeHooks(t, fixture.hook, fixture.owner, strings.Repeat("0", 64))
	fixture.repo.Write(testHooksConfig, string(configBytes))
	fixture.revision = fixture.repo.Commit("record stale hook pin")

	report, err := Check(Options{Root: fixture.repo.Dir, Revision: fixture.revision, HooksConfigPath: testHooksConfig})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusFindings || len(report.Modules) != 2 {
		t.Fatalf("stale digest did not fail the combined report: %#v", report)
	}
	if got := report.Modules[0]; got.Status != githooks.StatusFindings || got.GitHooks == nil || len(got.GitHooks.Findings) == 0 || got.GitHooks.Findings[0].Code != "hooks.digest.mismatch" {
		t.Fatalf("stale digest finding/provenance was lost: %#v", got)
	}
	if got := report.Modules[1]; got.Name != "pipelines" || got.Status != pipelines.StatusNotConfigured || got.Pipelines != nil {
		t.Fatalf("omitted pipelines module was not reported: %#v", got)
	}
}

func TestCheckSelectedPassingModuleKeepsAggregatePassedWhenOtherIsOmitted(t *testing.T) {
	fixture := newModuleFixture(t, true, true)
	report, err := Check(Options{Root: fixture.repo.Dir, Revision: fixture.revision, HooksConfigPath: testHooksConfig})
	if err != nil {
		t.Fatal(err)
	}
	if report.Modules[0].Status != githooks.StatusPassed || report.Modules[1].Status != pipelines.StatusNotConfigured || report.Status != StatusPassed {
		t.Fatalf("omitted module must be reported without changing the selected module aggregate: %#v", report)
	}
}

func TestCheckRequiresExactDeclaredArtifactOwnership(t *testing.T) {
	fixture := newModuleFixture(t, false, true)
	_, err := Check(Options{Root: fixture.repo.Dir, Revision: fixture.revision, HooksConfigPath: testHooksConfig})
	if err == nil || !strings.Contains(err.Error(), "not an explicitly declared input of a canonical Artifact") {
		t.Fatalf("missing exact Artifact path association error = %v", err)
	}
}

func TestCheckRejectsAbsentAndUnownedConfigPaths(t *testing.T) {
	t.Run("absent config", func(t *testing.T) {
		fixture := newModuleFixture(t, true, true)
		_, err := Check(Options{Root: fixture.repo.Dir, Revision: fixture.revision, HooksConfigPath: ".markitect/modules/missing.config"})
		if err == nil || !strings.Contains(err.Error(), "absent from the selected snapshot") {
			t.Fatalf("missing config error = %v", err)
		}
	})

	t.Run("unowned config", func(t *testing.T) {
		fixture := newModuleFixture(t, true, true)
		path := ".markitect/modules/unowned.config"
		fixture.repo.Write(path, string(encodeHooks(t, fixture.hook, fixture.owner, digestBytes(fixture.hook))))
		fixture.revision = fixture.repo.Commit("add unowned module config")
		_, err := Check(Options{Root: fixture.repo.Dir, Revision: fixture.revision, HooksConfigPath: path})
		if err == nil || !strings.Contains(err.Error(), "not an explicitly declared input of a canonical Artifact") {
			t.Fatalf("unowned config error = %v", err)
		}
	})
}

func TestCheckFailsClosedOnIncompleteCurrentProjectModel(t *testing.T) {
	fixture := newModuleFixture(t, false, true)
	// Retain the canonical files but remove the root Manager's whole-repository
	// ownership. The current projectmodel analysis must reject its unknown paths
	// before the independent module check can run.
	rootManager := core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Manager", Metadata: core.Metadata{Name: "root"}, Purpose: "Own the fixture repository.", Spec: map[string]any{"owns": []any{}}}
	fixture.repo.Write(".markitect/model/manager.yaml", string(encodeDefinition(t, rootManager)))
	fixture.revision = fixture.repo.Commit("make current model incomplete")
	_, err := Check(Options{Root: fixture.repo.Dir, Revision: fixture.revision, HooksConfigPath: testHooksConfig})
	if err == nil || !strings.Contains(err.Error(), "strict project model analysis did not pass") {
		t.Fatalf("incomplete current project model error = %v", err)
	}
}

func newModuleFixture(t *testing.T, includeHookOwnership, includeBothConfigs bool) moduleFixture {
	t.Helper()
	repo := testkit.NewRepo(t)
	owner := (core.DefinitionIdentity{APIVersion: projectmodel.APIVersion, Kind: "Artifact", Namespace: "tooling", Name: "module-verification"}).Key()
	hook := []byte("#!/bin/sh\nprintf 'ready\\n'\n")
	workflow := []byte("name: CI\njobs:\n  quality:\n    steps:\n      - run: go test ./...\n")
	paths := []any{testHooksConfig, testPipelinesConfig, testWorkflowPath}
	if includeHookOwnership {
		paths = append(paths, testHookPath)
	}
	artifact := core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Artifact", Metadata: core.Metadata{Name: "module-verification", Namespace: "tooling"}, Purpose: "Maps exact module check inputs.", Spec: map[string]any{
		"role": "verification", "paths": paths, "checks": []any{map[string]any{"namespace": "tooling", "name": "go-tests"}}, "required": true,
		"reason": "The configured modules verify exact hook and workflow bytes.",
	}}
	definitions := map[string]core.Definition{
		".markitect/model/manager.yaml":                     {APIVersion: projectmodel.APIVersion, Kind: "Manager", Metadata: core.Metadata{Name: "root"}, Purpose: "Own the fixture repository.", Spec: map[string]any{"owns": []any{"."}}},
		".markitect/model/tooling/module-verification.yaml": artifact,
		".markitect/model/tooling/go-tests.yaml":            {APIVersion: projectmodel.APIVersion, Kind: "Check", Metadata: core.Metadata{Name: "go-tests", Namespace: "tooling"}, Purpose: "Runs Go tests.", Spec: map[string]any{"command": []any{"go", "test", "./..."}, "uses": []any{}, "limitation": "The literal command is not provider execution proof."}},
	}
	for path, definition := range definitions {
		repo.Write(path, string(encodeDefinition(t, definition)))
	}
	repo.Write(projectvalue.ManifestPath, string(encodeProjectConfig(t)))
	repo.Write(testHooksConfig, string(encodeHooks(t, hook, owner, digestBytes(hook))))
	if includeBothConfigs {
		repo.Write(testPipelinesConfig, string(encodePipelines(t, workflow, owner)))
	}
	repo.Write(testHookPath, string(hook))
	repo.Write(testWorkflowPath, string(workflow))
	revision := repo.Commit("module check fixture")
	return moduleFixture{repo: repo, revision: revision, owner: owner, hook: hook, workflow: workflow}
}

func encodeProjectConfig(t *testing.T) []byte {
	t.Helper()
	data, err := yaml.Marshal(projectvalue.Config{APIVersion: projectmodel.APIVersion, Name: "module-check-fixture", CoverageMode: "selected",
		ModelFiles:     []string{".markitect/model/manager.yaml", ".markitect/model/tooling/module-verification.yaml", ".markitect/model/tooling/go-tests.yaml"},
		InventoryRoots: []string{}, Exclusions: []projectvalue.Exclusion{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func encodeDefinition(t *testing.T, definition core.Definition) []byte {
	t.Helper()
	data, err := yaml.Marshal(map[string]any{
		"apiVersion": definition.APIVersion,
		"kind":       definition.Kind,
		"metadata":   map[string]any{"name": definition.Metadata.Name, "namespace": definition.Metadata.Namespace},
		"purpose":    definition.Purpose,
		"spec":       definition.Spec,
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func encodeHooks(t *testing.T, hook []byte, owner, digest string) []byte {
	t.Helper()
	data, err := yaml.Marshal(githooks.Config{APIVersion: githooks.ConfigVersion, Hooks: []githooks.Hook{{Name: "pre-commit", Stage: "pre-commit", Path: testHookPath, Digest: digest, Owner: owner}}})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func encodePipelines(t *testing.T, workflow []byte, owner string) []byte {
	t.Helper()
	data, err := yaml.Marshal(pipelines.Config{APIVersion: pipelines.ConfigVersion, Pipelines: []pipelines.Pipeline{{Name: "ci", Provider: "github-actions", Path: testWorkflowPath, Digest: digestBytes(workflow), Owner: owner,
		ExpectedChecks: []pipelines.ExpectedCheck{{Name: "go-tests", YAMLPath: "/jobs/quality/steps/0/run"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
