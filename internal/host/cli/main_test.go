package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
)

func TestRunInvalidArgumentsAndUnknownFlagsReturnTwo(t *testing.T) {
	repo := newCLIRepo(t, false)
	tests := []struct {
		name string
		args []string
	}{
		{"missing command", nil},
		{"unknown command", []string{"no-such-command"}},
		{"unknown flag", []string{"check", "--repo", repo.root, "--not-a-flag"}},
		{"unexpected argument", []string{"check", "--repo", repo.root, "unexpected"}},
		{"missing context fields", []string{"context", "--repo", repo.root, "--kind", "Skill", "--name", "entry"}},
		{"invalid write combination", []string{"render", "--repo", repo.root, "--write", "--revision", repo.base}},
		{"version rejects arguments", []string{"version", "--not-a-flag"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, _ := invoke(tt.args...)
			if code != 2 {
				t.Fatalf("run(%v) = %d, want exit 2", tt.args, code)
			}
		})
	}
}

func TestAnalyzePolicyFailuresFlagIsLimitedToResourceContextAndImpact(t *testing.T) {
	for _, command := range []string{"check", "verify", "model", "reconcile", "render", "format", "review", "explain"} {
		t.Run(command, func(t *testing.T) {
			code, _, stderr := invoke(command, "--analyze-policy-failures")
			if code != 2 || !strings.Contains(stderr, "does not apply") {
				t.Fatalf("%s accepted diagnostic analysis option: exit=%d stderr=%q", command, code, stderr)
			}
		})
	}

	for _, value := range []string{"true", "false"} {
		code, _, stderr := invoke("context", "--run", strings.Repeat("a", 40), "--revision", strings.Repeat("a", 40), "--analyze-policy-failures="+value)
		if code != 2 || !strings.Contains(stderr, "does not apply to context --run") {
			t.Fatalf("context --run accepted explicit resource policy analysis value %s: exit=%d stderr=%q", value, code, stderr)
		}
	}
}

func TestAnalyzePolicyFailuresHelpIsExplicitlyScoped(t *testing.T) {
	for _, command := range []string{"context", "impact"} {
		code, output, stderr := invoke(command, "--help")
		if code != 0 || stderr != "" || !strings.Contains(output, "--analyze-policy-failures") {
			t.Fatalf("%s help omitted diagnostic analysis option: exit=%d stderr=%q output=%q", command, code, stderr, output)
		}
	}
	for _, command := range []string{"check", "verify", "reconcile"} {
		code, output, stderr := invoke(command, "--help")
		if code != 0 || stderr != "" || strings.Contains(output, "--analyze-policy-failures") {
			t.Fatalf("%s help exposed diagnostic analysis option: exit=%d stderr=%q output=%q", command, code, stderr, output)
		}
	}
}

func TestCheckUsesFixedGitSnapshotInsteadOfDirtyTree(t *testing.T) {
	repo := newCLIRepo(t, false)
	generatedPath := filepath.Join(repo.root, "docs/markitect/sample/skills/entry.skill.md")
	if err := os.WriteFile(generatedPath, []byte("dirty local output\n"), 0644); err != nil {
		t.Fatal(err)
	}
	args := []string{"check", "--repo", repo.root, "--revision", repo.base}
	code, output, stderr := invoke(args...)
	if code != 0 {
		t.Fatalf("fixed-snapshot check exit=%d stderr=%s output=%s", code, stderr, output)
	}
	fixed := decodeYAML[report](t, output)
	if fixed.Status != "passed" || fixed.Provisional || fixed.Revision != repo.base {
		t.Fatalf("unexpected fixed-snapshot report: %#v", fixed)
	}
	code, output, _ = invoke("check", "--repo", repo.root)
	if code != 1 {
		t.Fatalf("working-tree check exit=%d, want drift diagnostic exit 1; output=%s", code, output)
	}
	dirty := decodeYAML[report](t, output)
	if dirty.Status != "failed" || !dirty.Provisional || len(dirty.Diagnostics) == 0 || dirty.Diagnostics[0].Code != "output-drift" {
		t.Fatalf("unexpected dirty-tree report: %#v", dirty)
	}
}

func TestCheckDiagnosticsReturnOne(t *testing.T) {
	repo := newCLIRepo(t, true)
	code, output, stderr := invoke("check", "--repo", repo.root, "--revision", repo.base)
	if code != 1 {
		t.Fatalf("invalid graph check exit=%d stderr=%s output=%s, want 1", code, stderr, output)
	}
	result := decodeYAML[report](t, output)
	if result.Status != "failed" || len(result.Diagnostics) == 0 {
		t.Fatalf("failed check did not return diagnostics: %#v", result)
	}
}

func TestImpactRenderInventoryAndVersionCommands(t *testing.T) {
	repo := newCLIRepo(t, false)
	code, output, stderr := invoke("render", "--repo", repo.root, "--revision", repo.base)
	if code != 0 {
		t.Fatalf("render exit=%d stderr=%s output=%s", code, stderr, output)
	}
	if result := decodeYAML[report](t, output); result.Status != "passed" || result.Revision != repo.base {
		t.Fatalf("unexpected render report: %#v", result)
	}

	rule := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "policy", Namespace: cliNamespace}}, Spec: authoring.Spec{Text: "Changed policy."}}
	data, err := authoring.Encode(rule)
	if err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo.root, "docs/general/rules/policy.yaml", data)
	git(t, repo.root, "add", "docs/general/rules/policy.yaml")
	git(t, repo.root, "commit", "-m", "revise rule")
	candidate := git(t, repo.root, "rev-parse", "HEAD")
	code, output, stderr = invoke("impact", "--repo", repo.root, "--base", repo.base, "--revision", candidate)
	if code != 0 {
		t.Fatalf("impact exit=%d stderr=%s output=%s", code, stderr, output)
	}
	impact := decodeYAML[host.Impact](t, output)
	if impact.Base != repo.base || impact.Candidate != candidate || len(impact.Affected) != 2 {
		t.Fatalf("unexpected impact result: %#v", impact)
	}

	code, output, stderr = invoke("inventory", "--repo", repo.root, "--revision", repo.base)
	if code != 0 {
		t.Fatalf("inventory exit=%d stderr=%s output=%s", code, stderr, output)
	}
	result := decodeYAML[report](t, output)
	if result.Status != "inventory" || result.Revision != repo.base || len(result.Inventory) != 4 {
		t.Fatalf("unexpected legacy inventory: %#v", result)
	}
	seen := map[string]bool{}
	for _, entry := range result.Inventory {
		seen[entry.Path] = true
	}
	for _, expected := range []string{"markitect.yaml", "docs/general/rules/policy.yaml", "docs/general/skills/entry.yaml", "docs/general/rules/legacy.md"} {
		if !seen[expected] {
			t.Errorf("inventory omitted expected canonical or legacy input %s: %v", expected, seen)
		}
	}
	if seen["docs/markitect/sample/rules/policy.rule.md"] || seen["docs/markitect/sample/skills/entry.skill.md"] {
		t.Errorf("inventory duplicated generated views as legacy inputs: %v", seen)
	}

	code, output, _ = invoke("version")
	wantVersion := fmt.Sprintf("Markitect %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	if code != 0 || output != wantVersion {
		t.Fatalf("version output = (%d, %q), want (%d, %q)", code, output, 0, wantVersion)
	}
}
