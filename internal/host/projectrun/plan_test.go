package projectrun

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

func TestProjectModelEditCannotChangeRuntimeConfiguration(t *testing.T) {
	for _, path := range []string{RuntimePath, ".markitect/runtime.yaml/child", ".markitect/project.yaml"} {
		if err := validateModelEditPaths(Mutation{Files: []FileChange{{Path: path}}}); err == nil {
			t.Fatalf("ModelEdit path %q unexpectedly passed the model-only contract", path)
		}
	}
	if err := validateModelEditPaths(Mutation{Files: []FileChange{{Path: ".markitect/model/managers/root.yaml"}}}); err != nil {
		t.Fatalf("valid model edit path rejected: %v", err)
	}
}

func TestPlanRequiresWorkingSelectedInputsEqualFixedRevision(t *testing.T) {
	fixed := &snapshot.Snapshot{Files: map[string][]byte{"src/main.go": []byte("committed")}, Modes: map[string]string{"src/main.go": snapshot.RegularMode}}
	working := &snapshot.Snapshot{Files: map[string][]byte{"src/main.go": []byte("uncommitted")}, Modes: map[string]string{"src/main.go": snapshot.RegularMode}}
	err := requireCleanSelectedBasis(fixed, working)
	if err == nil || !strings.Contains(err.Error(), "commit accepted selected changes") {
		t.Fatalf("expected dirty selected input to block planning, got %v", err)
	}
	working.Files["src/main.go"] = []byte("committed")
	if err := requireCleanSelectedBasis(fixed, working); err != nil {
		t.Fatalf("matching selected snapshots rejected: %v", err)
	}
}
