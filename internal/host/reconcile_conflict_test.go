package host

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectionPlanReportsUnmanagedCollisionBeforeApply(t *testing.T) {
	root := tempRoot(t)
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	project := loadAndWriteFixtureOutputs(t, root)
	path := "docs/markitect/sample/rules/policy.rule.md"
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	ownedByUser := []byte("# Human-owned policy\nKeep this document.\n")
	if err := os.WriteFile(fullPath, ownedByUser, 0644); err != nil {
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
	if plan.Status != "incomplete" || len(plan.Conflicts) != 1 || plan.Conflicts[0] != path {
		t.Fatalf("unmanaged collision was not reported: %+v", plan)
	}
	for _, operation := range plan.Operations {
		if operation.Path == path {
			t.Fatal("plan offered an operation replacing a human-owned file")
		}
	}
	if _, err := ApplyProjection(root, project, plan, "test-version", "test-tool"); err == nil {
		t.Fatal("incomplete collision plan was applied")
	}
	if !bytes.Equal(mustRead(t, fullPath), ownedByUser) {
		t.Fatal("reconciliation changed a human-owned file")
	}
}
