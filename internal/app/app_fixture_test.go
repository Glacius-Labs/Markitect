package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

const (
	projectPath = "markitect.yaml"
	rulePath    = "docs/general/rules/policy.yaml"
	skillPath   = "docs/general/skills/entry.yaml"
	projectNS   = "sample"
)

func projectResource(namespace string) core.Resource {
	return core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: "sample-project"},
		Spec: core.Spec{Targets: []string{"codex"}, Areas: []core.Area{
			{Name: projectNS, Path: "docs/general"},
		}},
	}
}

func fixtureFiles(t *testing.T, revision, ruleText, skillNamespace string) *source.Snapshot {
	t.Helper()
	config := projectResource(projectNS)
	rule := core.Resource{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "policy", Namespace: projectNS}, Spec: core.Spec{Text: ruleText}}
	skill := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "entry", Namespace: skillNamespace}, Spec: core.Spec{Text: "Use the policy.", Rules: []core.Ref{{Name: "policy"}}}}
	files := map[string][]byte{
		projectPath: encodeResource(t, config),
		rulePath:    encodeResource(t, rule),
		skillPath:   encodeResource(t, skill),
	}
	modes := map[string]string{}
	for name := range files {
		modes[name] = "100644"
	}
	return &source.Snapshot{Revision: revision, Provisional: revision == "", Files: files, Modes: modes}
}

func encodeResource(t *testing.T, value core.Resource) []byte {
	t.Helper()
	b, err := format.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func loadFixture(t *testing.T, root string) *Project {
	t.Helper()
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	p, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		t.Fatalf("fixture has project diagnostics: %#v", p.Diagnostics)
	}
	return p
}

func writeFixture(t *testing.T, root string, files map[string][]byte) {
	t.Helper()
	for name, data := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func tempRoot(t *testing.T) string {
	t.Helper()
	if err := os.MkdirAll(".cache", 0755); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(".cache", "app-test-")
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(abs); err != nil {
			t.Errorf("remove test root: %v", err)
		}
	})
	return abs
}

func initAppTestRepo(t *testing.T, root string) {
	t.Helper()
	cmd := exec.Command("git", "-c", "safe.directory="+filepath.ToSlash(root), "-C", root, "init", "-b", "feature/app-test")
	cmd.Env = source.CleanGitEnv()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init app test repo: %v\n%s", err, output)
	}
}
