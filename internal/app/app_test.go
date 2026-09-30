package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/render"
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

func TestLoadParseAndCompileContext(t *testing.T) {
	root := tempRoot(t)
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	p, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Snapshot == nil || !p.Snapshot.Provisional || len(p.Resources) != 3 || len(p.Inventory) != 3 {
		t.Fatalf("unexpected loaded project: snapshot=%#v resources=%d inventory=%d", p.Snapshot, len(p.Resources), len(p.Inventory))
	}

	parsed, err := Parse(fixtureFiles(t, "fixture-commit", "Keep the owner source.", projectNS))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Snapshot.Revision != "fixture-commit" || len(parsed.Diagnostics) != 0 {
		t.Fatalf("unexpected parsed project: %#v", parsed)
	}

	ctx, err := CompileContext(parsed, projectNS+"/Skill/entry", "test-tool")
	if err != nil {
		t.Fatal(err)
	}
	wantKeys := []string{"/Project/sample-project", projectNS + "/Rule/policy", projectNS + "/Skill/entry"}
	gotKeys := make([]string, 0, len(ctx.Inputs))
	for _, input := range ctx.Inputs {
		gotKeys = append(gotKeys, input.Key)
		if input.Hash != Hash(parsed.Snapshot.Files[input.Path]) {
			t.Fatalf("context hash for %s does not match captured bytes", input.Key)
		}
	}
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Fatalf("context inputs = %v, want %v", gotKeys, wantKeys)
	}
}

func TestChangesIncludesOldAndNewDependencyClosures(t *testing.T) {
	before, err := Parse(fixtureFiles(t, "base", "Old policy.", projectNS))
	if err != nil {
		t.Fatal(err)
	}
	after, err := Parse(fixtureFiles(t, "candidate", "Revised policy.", projectNS))
	if err != nil {
		t.Fatal(err)
	}
	impact := Changes(before, after)
	want := []string{projectNS + "/Rule/policy", projectNS + "/Skill/entry"}
	if !reflect.DeepEqual(impact.Affected, want) {
		t.Fatalf("affected = %v, want dependency closure %v", impact.Affected, want)
	}
	if !reflect.DeepEqual(impact.Changed, []string{rulePath}) {
		t.Fatalf("changed = %v, want [%s]", impact.Changed, rulePath)
	}
}

func TestChangesInvalidatesAllForNonResourceMarkdown(t *testing.T) {
	beforeSnapshot := fixtureFiles(t, "base", "Stable policy.", projectNS)
	afterSnapshot := fixtureFiles(t, "candidate", "Stable policy.", projectNS)
	beforeSnapshot.Files["docs/general/notes.md"] = []byte("old note\n")
	afterSnapshot.Files["docs/general/notes.md"] = []byte("new note\n")
	before, err := Parse(beforeSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Parse(afterSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	impact := Changes(before, after)
	want := []string{"/Project/sample-project", projectNS + "/Rule/policy", projectNS + "/Skill/entry"}
	if !reflect.DeepEqual(impact.Affected, want) {
		t.Fatalf("non-resource Markdown change affected %v, want conservative invalidation %v", impact.Affected, want)
	}
}

func TestChangesInvalidatesInventoryOnSamePathNamespaceEdit(t *testing.T) {
	before, err := Parse(fixtureFiles(t, "base", "Stable policy.", projectNS))
	if err != nil {
		t.Fatal(err)
	}
	after, err := Parse(fixtureFiles(t, "candidate", "Stable policy.", "other-area"))
	if err != nil {
		t.Fatal(err)
	}
	impact := Changes(before, after)
	want := []string{"/Project/sample-project", "other-area/Skill/entry", projectNS + "/Rule/policy", projectNS + "/Skill/entry"}
	if !reflect.DeepEqual(impact.Affected, want) {
		t.Fatalf("namespace inventory edit affected %v, want old and new resource inventories %v", impact.Affected, want)
	}
}

func TestCheckOutputsReportsMissingAndDrift(t *testing.T) {
	snapshot := fixtureFiles(t, "fixture", "Keep the owner source.", projectNS)
	p, err := Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := render.Generate(p.Graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(CheckOutputs(p)) == 0 {
		t.Fatal("missing generated outputs were not reported")
	}
	for name, data := range generated {
		snapshot.Files[name] = append([]byte(nil), data...)
		snapshot.Modes[name] = "100644"
	}
	p, err = Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if findings := CheckOutputs(p); len(findings) != 0 {
		t.Fatalf("matching generated outputs have findings: %#v", findings)
	}
	for name := range generated {
		snapshot.Files[name] = append(snapshot.Files[name], []byte("drift\n")...)
		break
	}
	p, err = Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	findings := CheckOutputs(p)
	if len(findings) != 1 || findings[0].Code != "output-drift" {
		t.Fatalf("drift findings = %#v, want one output-drift diagnostic", findings)
	}
}

func TestWriteOutputsWritesGeneratedPlan(t *testing.T) {
	root := tempRoot(t)
	initAppTestRepo(t, root)
	p := loadFixture(t, root)
	written, err := WriteOutputs(root, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) == 0 {
		t.Fatal("WriteOutputs wrote no generated files")
	}
	for _, name := range written {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if !Generated(data) {
			t.Errorf("%s lacks the generated ownership marker", name)
		}
	}
}

func TestWriteOutputsRefusesConcurrentSourceEdit(t *testing.T) {
	root := tempRoot(t)
	initAppTestRepo(t, root)
	p := loadFixture(t, root)
	changed := filepath.Join(root, filepath.FromSlash(rulePath))
	if err := os.WriteFile(changed, []byte("changed after capture\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteOutputs(root, p); err == nil || !strings.Contains(err.Error(), "source inventory changed since capture") {
		t.Fatalf("WriteOutputs error = %v, want concurrent source refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/general/skills/entry.md")); !os.IsNotExist(err) {
		t.Fatalf("output was written despite concurrent source edit: stat error = %v", err)
	}
}

func TestWriteOutputsRefusesSourceAddedAfterCapture(t *testing.T) {
	root := tempRoot(t)
	initAppTestRepo(t, root)
	p := loadFixture(t, root)
	newSource := core.Resource{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Name: "added", Namespace: projectNS}, Spec: core.Spec{Text: "Added after capture."}}
	writeFixture(t, root, map[string][]byte{"docs/general/added.yaml": encodeResource(t, newSource)})
	if _, err := WriteOutputs(root, p); err == nil || !strings.Contains(err.Error(), "inventory changed since capture") {
		t.Fatalf("WriteOutputs error = %v, want changed-inventory refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/general/skills/entry.md")); !os.IsNotExist(err) {
		t.Fatalf("output was written despite a new source file: stat error = %v", err)
	}
}

func TestWriteOutputsRefusesUnmanagedFile(t *testing.T) {
	root := tempRoot(t)
	initAppTestRepo(t, root)
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	unmanaged := filepath.Join(root, "docs/general/skills/entry.md")
	if err := os.MkdirAll(filepath.Dir(unmanaged), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unmanaged, []byte("Human authored content.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteOutputs(root, p); err == nil || !strings.Contains(err.Error(), "unmanaged file") {
		t.Fatalf("WriteOutputs error = %v, want unmanaged-file refusal", err)
	}
	data, err := os.ReadFile(unmanaged)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte("Human authored content.\n")) {
		t.Fatal("unmanaged content was changed")
	}
}

func TestWriteOutputsRejectsUnsafeDestinationAndSnapshot(t *testing.T) {
	root := tempRoot(t)
	initAppTestRepo(t, root)
	for _, name := range []string{"../outside", "docs\\outside.md", ".git/config", "C:/outside"} {
		if _, err := safeDestination(root, name); err == nil {
			t.Errorf("safeDestination(%q) accepted an unsafe path", name)
		}
	}
	p := loadFixture(t, root)
	for _, resource := range p.Resources {
		if resource.Kind == "Skill" {
			resource.Path = "../outside.yaml"
		}
	}
	if _, err := WriteOutputs(root, p); err == nil || !strings.Contains(err.Error(), "path must be repository-relative and normalized") {
		t.Fatalf("WriteOutputs error = %v, want unsafe-resource-path refusal", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "outside.md")); !os.IsNotExist(err) {
		t.Fatalf("unsafe output escaped the workspace: stat error = %v", err)
	}
}
