package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/render"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

const cliNamespace = "cockpit-general"

type cliRepo struct {
	root string
	base string
}

func newCLIRepo(t *testing.T, badReference bool) cliRepo {
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
	git(t, root, "init", "-b", "work")
	git(t, root, "config", "user.name", "Markitect Test")
	git(t, root, "config", "user.email", "markitect-test@example.invalid")
	files := cliFixture(t, badReference)
	for name, data := range files {
		writeRepoFile(t, root, name, data)
	}
	loaded, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := render.Generate(loaded.Graph)
	if err != nil {
		t.Fatal(err)
	}
	for name, expected := range outputs {
		actual, ok := files[name]
		if !ok || !bytes.Equal(actual, expected) {
			t.Fatalf("fixture output %s differs from app parse\nactual:\n%s\nexpected:\n%s", name, actual, expected)
		}
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "initial fixture")
	return cliRepo{root: root, base: git(t, root, "rev-parse", "HEAD")}
}

func cliFixture(t *testing.T, badReference bool) map[string][]byte {
	t.Helper()
	project := core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "cockpit"}, Spec: core.Spec{
		Profile: "generic",
		Areas:   []core.Area{{Name: cliNamespace, Path: "docs/general"}},
	}}
	rule := core.Resource{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "policy", Namespace: cliNamespace}, Spec: core.Spec{Text: "Keep the canonical source."}}
	refName := "policy"
	if badReference {
		refName = "missing-rule"
	}
	skill := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "entry", Namespace: cliNamespace}, Spec: core.Spec{Text: "Use the policy.", Rules: []core.Ref{{Name: refName}}}}
	resources := []*core.Resource{&project, &rule, &skill}
	names := []string{"markitect.yaml", "docs/general/rules/policy.yaml", "docs/general/skills/entry.yaml"}
	files := make(map[string][]byte, 5)
	for i, resource := range resources {
		resource.Path = names[i]
		data, err := format.Encode(resource)
		if err != nil {
			t.Fatal(err)
		}
		files[names[i]] = data
	}
	graph := core.Build(resources)
	if len(graph.Diagnostics) != 0 && !badReference {
		t.Fatalf("CLI fixture graph has diagnostics: %#v", graph.Diagnostics)
	}
	outputs, err := render.Generate(graph)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range outputs {
		files[name] = data
	}
	files["docs/general/rules/legacy.md"] = []byte("# Legacy policy candidate\n")
	return files
}

func writeRepoFile(t *testing.T, root, name string, data []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"-c", "safe.directory=" + filepath.ToSlash(root), "-C", root}, args...)
	cmd := exec.Command("git", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func invoke(args ...string) (int, string, string) {
	var out, errout bytes.Buffer
	code := run(args, &out, &errout)
	return code, out.String(), errout.String()
}

func decodeYAML[T any](t *testing.T, data string) T {
	t.Helper()
	var value T
	if err := yaml.Unmarshal([]byte(data), &value); err != nil {
		t.Fatalf("decode CLI YAML: %v\n%s", err, data)
	}
	return value
}

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

func TestCheckUsesFixedGitSnapshotInsteadOfDirtyTree(t *testing.T) {
	repo := newCLIRepo(t, false)
	generatedPath := filepath.Join(repo.root, "docs/general/skills/entry.md")
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

func TestContextIncludesSnapshotDigestAndRequiredRule(t *testing.T) {
	repo := newCLIRepo(t, false)
	code, output, stderr := invoke("context", "--repo", repo.root, "--revision", repo.base, "--kind", "Skill", "--name", "entry", "--namespace", cliNamespace)
	if code != 0 {
		t.Fatalf("context exit=%d stderr=%s output=%s", code, stderr, output)
	}
	ctx := decodeYAML[app.Context](t, output)
	snapshot, err := source.Load(repo.root, repo.base)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.SnapshotDigest == "" || ctx.SnapshotDigest != snapshot.Digest() || ctx.ToolDigest == "" || ctx.Provisional || ctx.Revision != repo.base {
		t.Fatalf("context has incorrect snapshot identity: %#v, actual digest %s", ctx, snapshot.Digest())
	}
	foundRule := false
	for _, input := range ctx.Inputs {
		if input.Resource != nil && input.Resource.Kind == "Rule" && input.Resource.Metadata.Name == "policy" {
			foundRule = true
			if input.Hash != app.Hash(snapshot.Files[input.Path]) {
				t.Errorf("rule input hash %s does not match snapshot bytes", input.Key)
			}
		}
	}
	if !foundRule {
		t.Fatalf("context omitted the required Rule: %#v", ctx.Inputs)
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

	rule := core.Resource{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "policy", Namespace: cliNamespace}, Spec: core.Spec{Text: "Changed policy."}}
	data, err := format.Encode(rule)
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
	impact := decodeYAML[app.Impact](t, output)
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
	if seen["docs/general/rules/policy.md"] || seen["docs/general/skills/entry.md"] {
		t.Errorf("inventory duplicated generated views as legacy inputs: %v", seen)
	}

	code, output, _ = invoke("version")
	wantVersion := fmt.Sprintf("Markitect %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	if code != 0 || output != wantVersion {
		t.Fatalf("version output = (%d, %q), want (%d, %q)", code, output, 0, wantVersion)
	}
}
