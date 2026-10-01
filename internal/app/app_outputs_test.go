package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/render"
)

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
