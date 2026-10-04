package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/render"
	"go.yaml.in/yaml/v3"
)

const cliNamespace = "sample"

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
	loaded, err := host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := render.Generate(loaded.Graph, loaded.Snapshot.Files)
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
	project := core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "sample-project"}, Spec: core.Spec{
		Areas: []core.Area{{Name: cliNamespace, Path: "docs/general"}}, Targets: []string{"markdown"},
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
	outputs, err := render.Generate(graph, files)
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
	cmd.Env = source.CleanGitEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func invoke(args ...string) (int, string, string) {
	var out, errout bytes.Buffer
	code := Run(args, &out, &errout)
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

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	for dir := filepath.Dir(file); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root from go.mod")
		}
	}
}
