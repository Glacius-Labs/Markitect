package modulecli

import "go.yaml.in/yaml/v3"

import "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"

	"github.com/Glacius-Labs/Markitect/src/internal/modules/githooks"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

const (
	cliHooksConfig = ".markitect/modules/hooks.config"
	cliHookPath    = ".githooks/pre-commit"
	cliArtifactID  = `["project.markitect.example.org/v1alpha1","Artifact","tooling","module-verification"]`
)

func TestMain(m *testing.M) { testkit.Main(m) }

func TestRunRequiresExplicitModuleConfigAndRejectsUnknownArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"--unknown"}, {"--hooks", "hooks.yaml", "unexpected"}} {
		var out, errout bytes.Buffer
		if code := Run(args, &out, &errout); code != 2 || out.Len() != 0 || errout.Len() == 0 {
			t.Fatalf("Run(%v): code=%d stdout=%q stderr=%q", args, code, out.String(), errout.String())
		}
	}
	var out, errout bytes.Buffer
	if code := Run([]string{"--help"}, &out, &errout); code != 0 || !strings.Contains(errout.String(), "-hooks") || !strings.Contains(errout.String(), "-pipelines") {
		t.Fatalf("help: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
}

func TestRunReportsPassFindingsAndInvalidInputExitCodes(t *testing.T) {
	repo := testkit.NewRepo(t)
	hookBytes := []byte("#!/bin/sh\nexit 0\n")
	configBytes := encodeHooksConfig(t, hookBytes)
	for file, data := range map[string][]byte{
		projectvalue.ManifestPath:                           projectManifest(t),
		".markitect/model/manager.yaml":                     definitionBytes(t, core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Manager", Metadata: core.Metadata{Name: "root"}, Purpose: "Own the fixture repository.", Spec: map[string]any{"owns": []any{"."}}}),
		".markitect/model/tooling/module-verification.yaml": definitionBytes(t, core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Artifact", Metadata: core.Metadata{Name: "module-verification", Namespace: "tooling"}, Purpose: "Maps exact module inputs.", Spec: map[string]any{"role": "verification", "paths": []any{cliHooksConfig, cliHookPath}, "required": true, "reason": "The fixture module checks its exact configured files."}}),
		".markitect/model/tooling/go-tests.yaml":            definitionBytes(t, core.Definition{APIVersion: projectmodel.APIVersion, Kind: "Check", Metadata: core.Metadata{Name: "go-tests", Namespace: "tooling"}, Purpose: "Runs the Go tests.", Spec: map[string]any{"command": []any{"go", "test", "./..."}, "uses": []any{}, "limitation": "Does not establish provider execution."}}),
		cliHooksConfig:                                      configBytes,
		cliHookPath:                                         hookBytes,
	} {
		repo.Write(file, string(data))
	}
	revision := repo.Commit("module cli fixture")

	args := []string{"--repo", repo.Dir, "--revision", revision, "--hooks", cliHooksConfig}
	var out, errout bytes.Buffer
	if code := Run(args, &out, &errout); code != 0 || !strings.Contains(out.String(), "status: passed") || !strings.Contains(out.String(), "status: not-configured") || errout.Len() != 0 {
		t.Fatalf("passing configured module: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	repo.Write(cliHookPath, "changed hook\n")
	if code := Run([]string{"--repo", repo.Dir, "--hooks", cliHooksConfig}, &out, &errout); code != 1 || !strings.Contains(out.String(), "status: findings") || !strings.Contains(out.String(), "hooks.digest.mismatch") || errout.Len() != 0 {
		t.Fatalf("stale digest finding: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"--repo", repo.Dir, "--hooks", ".markitect/modules/missing.config"}, &out, &errout); code != 2 || out.Len() != 0 || !strings.Contains(errout.String(), "absent from the selected snapshot") {
		t.Fatalf("invalid module input: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
}

func encodeHooksConfig(t *testing.T, hook []byte) []byte {
	t.Helper()
	sum := sha256.Sum256(hook)
	data, err := yaml.Marshal(githooks.Config{APIVersion: githooks.ConfigVersion, Hooks: []githooks.Hook{{Name: "pre-commit-tests", Stage: "pre-commit", Path: cliHookPath, Digest: hex.EncodeToString(sum[:]), Owner: cliArtifactID}}})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func projectManifest(t *testing.T) []byte {
	t.Helper()
	data, err := yaml.Marshal(projectvalue.Config{
		APIVersion: projectmodel.APIVersion, Name: "module-cli-fixture", CoverageMode: "selected",
		ModelFiles:     []string{".markitect/model/manager.yaml", ".markitect/model/tooling/module-verification.yaml", ".markitect/model/tooling/go-tests.yaml"},
		InventoryRoots: []string{}, Exclusions: []projectvalue.Exclusion{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func definitionBytes(t *testing.T, value core.Definition) []byte {
	t.Helper()
	data, err := yaml.Marshal(map[string]any{
		"apiVersion": value.APIVersion,
		"kind":       value.Kind,
		"metadata":   map[string]any{"name": value.Metadata.Name, "namespace": value.Metadata.Namespace},
		"purpose":    value.Purpose,
		"spec":       value.Spec,
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}
