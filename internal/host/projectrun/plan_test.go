package projectrun

import "testing"

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
