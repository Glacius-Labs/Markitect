package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/render"
)

func TestVerifyUnsupportedGenericProfileReturnsFixedIncompleteReport(t *testing.T) {
	repo := newCLIRepo(t, false)
	code, output, stderr := invoke("verify", "--repo", repo.root, "--revision", repo.base)
	if code != 2 {
		t.Fatalf("verify exit=%d, want incomplete exit 2; stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[report](t, output)
	if result.Status != "incomplete" || result.Revision != repo.base || result.Provisional || result.Digest == "" || result.ToolDigest == "" {
		t.Fatalf("verify omitted fixed snapshot identity or incomplete status: %#v", result)
	}
	if len(result.Gates) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "verify.incomplete-evidence" || !strings.Contains(result.Diagnostics[0].Message, "profile generic") {
		t.Fatalf("unexpected unsupported-profile diagnostic or gates: %#v", result)
	}
	if strings.Contains(result.Coverage, "all fixed profile repository gates") || !strings.Contains(stderr, "incomplete-evidence") {
		t.Fatalf("incomplete verification claimed complete gates or omitted stderr summary: coverage=%q stderr=%q", result.Coverage, stderr)
	}
}

func TestVerifyGateFailurePreservesPassedAndFailingGateOutput(t *testing.T) {
	if _, err := exec.LookPath("python"); err != nil {
		if _, err = exec.LookPath("python3"); err != nil {
			t.Skip("Python unavailable for existing Konfyra compatibility gates")
		}
	}
	repo := newKonfyraVerifyRepo(t)
	code, output, stderr := invoke("verify", "--repo", repo.root, "--revision", repo.base)
	if code != 1 {
		t.Fatalf("verify exit=%d, want gate-failure exit 1; stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[report](t, output)
	if result.Status != "failed" || result.Revision != repo.base || result.Provisional || result.Digest == "" || result.ToolDigest == "" {
		t.Fatalf("verify omitted fixed snapshot identity or failed status: %#v", result)
	}
	if len(result.Gates) != 2 {
		t.Fatalf("verify discarded completed or current gate results: %#v", result.Gates)
	}
	if result.Gates[0].Name != "scripts/render-governance-adapters.py --check" || result.Gates[0].ExitCode != 0 || !strings.Contains(result.Gates[0].Output, "renderer compatibility fixture passed") {
		t.Errorf("first gate result was not preserved: %#v", result.Gates[0])
	}
	if result.Gates[1].Name != "python unittest discover scripts/tests" || result.Gates[1].ExitCode == 0 || !strings.Contains(result.Gates[1].Output, "fixture consumer check marker") {
		t.Errorf("failing gate output/exit code was not preserved: %#v", result.Gates[1])
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "verify.gate-failure" || result.Diagnostics[0].Path != result.Gates[1].Name {
		t.Fatalf("verify diagnostic did not identify the failed gate: %#v", result.Diagnostics)
	}
	if strings.Contains(result.Coverage, "all fixed profile repository gates passed") || !strings.Contains(result.Coverage, "later gates were not run") || !strings.Contains(stderr, "gate-failure") {
		t.Fatalf("failed verification made a complete-coverage claim or omitted stderr summary: coverage=%q stderr=%q", result.Coverage, stderr)
	}
}

func newKonfyraVerifyRepo(t *testing.T) cliRepo {
	t.Helper()
	root := t.TempDir()
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-b", "verify-fixture")
	git(t, root, "config", "user.name", "Markitect Verify Test")
	git(t, root, "config", "user.email", "markitect-verify-test@example.invalid")

	project := core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: "cockpit"},
		Spec: core.Spec{
			Profile: "konfyra",
			Targets: []string{"codex"},
			Areas:   []core.Area{{Name: cliNamespace, Path: "docs/general"}},
		},
	}
	rule := core.Resource{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "policy", Namespace: cliNamespace}, Spec: core.Spec{Text: "Preserve canonical ownership."}}
	skill := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "entry", Namespace: cliNamespace}, Spec: core.Spec{Description: "Apply this policy to review work.", Text: "Use the policy.", Rules: []core.Ref{{Name: "policy"}}}}
	resources := []*core.Resource{&project, &rule, &skill}
	paths := []string{"markitect.yaml", "docs/general/rules/policy.yaml", "docs/general/skills/entry.yaml"}
	for i, resource := range resources {
		resource.Path = paths[i]
		data, err := format.Encode(resource)
		if err != nil {
			t.Fatal(err)
		}
		writeRepoFile(t, root, paths[i], data)
	}
	writeRepoFile(t, root, "scripts/render-governance-adapters.py", []byte("print('renderer compatibility fixture passed')\n"))
	writeRepoFile(t, root, "scripts/tests/test_gate_failure.py", []byte("import unittest\n\nclass FixtureCompatibilityGate(unittest.TestCase):\n    def test_failure_output_is_captured(self):\n        self.fail('fixture consumer check marker')\n"))

	projectState, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(projectState.Diagnostics) > 0 {
		t.Fatalf("Konfyra verify fixture has graph diagnostics: %#v", projectState.Diagnostics)
	}
	outputs, err := render.Generate(projectState.Graph)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range outputs {
		writeRepoFile(t, root, name, data)
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "Konfyra verify fixture")
	return cliRepo{root: root, base: git(t, root, "rev-parse", "HEAD")}
}
