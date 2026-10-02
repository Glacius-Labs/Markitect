package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/render"
)

func TestApplyProjectionNoopDoesNotWrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	project := loadAndWriteFixtureOutputs(t, root)
	plan, err := PlanProjection(project, "test-version", "test-tool")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 0 {
		t.Fatalf("no-op plan has %d operations", len(plan.Operations))
	}
	path := "docs/markitect/sample/rules/policy.rule.md"
	infoBefore, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyProjection(root, project, plan, "test-version", "test-tool"); err != nil {
		t.Fatal(err)
	}
	infoAfter, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	if !infoBefore.ModTime().Equal(infoAfter.ModTime()) {
		t.Fatalf("no-op apply changed output mtime from %s to %s", infoBefore.ModTime(), infoAfter.ModTime())
	}
	if _, err := os.Stat(filepath.Join(root, ".artifacts")); !os.IsNotExist(err) {
		t.Fatalf("no-op apply created a writer artifact (stat error: %v)", err)
	}
}

func TestApplyProjectionWritesOnlyPlannedPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	initial := loadAndWriteFixtureOutputs(t, root)
	outputs, err := render.Generate(initial.Graph, initial.Snapshot.Files)
	if err != nil {
		t.Fatal(err)
	}
	paths := sortedFiles(outputs)
	if len(paths) < 2 {
		t.Fatalf("fixture generated only %d outputs; need at least two", len(paths))
	}
	changedPath := paths[0]
	changedFile := filepath.Join(root, filepath.FromSlash(changedPath))
	oldTime := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.WriteFile(changedFile, []byte("<!-- "+render.Marker+" -->\nstale generated content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(changedFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanProjection(project, "test-version", "test-tool")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 1 || plan.Operations[0].Path != changedPath {
		t.Fatalf("plan operations = %#v, want only %s", plan.Operations, changedPath)
	}
	untouched := make(map[string][]byte, len(paths)-1)
	untouchedTimes := make(map[string]time.Time, len(paths)-1)
	for _, path := range paths[1:] {
		full := filepath.Join(root, filepath.FromSlash(path))
		data, readErr := os.ReadFile(full)
		if readErr != nil {
			t.Fatal(readErr)
		}
		info, statErr := os.Stat(full)
		if statErr != nil {
			t.Fatal(statErr)
		}
		untouched[path] = data
		untouchedTimes[path] = info.ModTime()
	}
	written, err := ApplyProjection(root, project, plan, "test-version", "test-tool")
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 1 || written[0] != changedPath {
		t.Fatalf("ApplyProjection wrote %v, want only %s", written, changedPath)
	}
	for path, expected := range untouched {
		full := filepath.Join(root, filepath.FromSlash(path))
		actual, readErr := os.ReadFile(full)
		if readErr != nil {
			t.Fatal(readErr)
		}
		info, statErr := os.Stat(full)
		if statErr != nil {
			t.Fatal(statErr)
		}
		if !bytes.Equal(actual, expected) || !info.ModTime().Equal(untouchedTimes[path]) {
			t.Errorf("unplanned output %s was changed by apply", path)
		}
	}
}

func loadAndWriteFixtureOutputs(t *testing.T, root string) *Project {
	t.Helper()
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := render.Generate(project.Graph, project.Snapshot.Files)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range outputs {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	project, err = Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	return project
}
