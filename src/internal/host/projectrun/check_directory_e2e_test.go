//go:build !runtime_unit

package projectrun

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

// TestFinalProjectCandidateRetainsModelledFileBaseBytes keeps the private
// candidate-composition predicate at its Runtime owner. The Application E2E
// in projectapp separately proves that actual Verify checks observe those
// bytes through the composed frontend and process invoker.
func TestFinalProjectCandidateRetainsModelledFileBaseBytes(t *testing.T) {
	t.Parallel()
	legacyPath := "src/orders/legacy.txt"
	baseSnapshot := &snapshot.Snapshot{
		ID:    "base",
		Files: map[string][]byte{legacyPath: []byte("legacy orders bytes\n"), projectvalue.ManifestPath: []byte("base model\n")},
		Modes: map[string]string{legacyPath: snapshot.RegularMode, projectvalue.ManifestPath: snapshot.RegularMode},
	}
	base := &projectvalue.Project{Snapshot: baseSnapshot}
	fromSnapshotCalls, classifyCalls := 0, 0
	host := Host{
		FromSnapshot: func(root string, candidate *snapshot.Snapshot) (*projectvalue.Project, error) {
			fromSnapshotCalls++
			return &projectvalue.Project{Root: root, Snapshot: candidate}, nil
		},
		ClassifyCandidate: func(gotBase, candidate *projectvalue.Project, _ ...string) (*projectvalue.Project, error) {
			classifyCalls++
			if gotBase != base {
				t.Fatal("candidate classification did not receive the original project")
			}
			return candidate, nil
		},
	}
	final := candidateData{Files: map[string]File{projectvalue.ManifestPath: {Path: projectvalue.ManifestPath, Mode: snapshot.RegularMode, Content: []byte("candidate model\n")}}}
	compiled, err := finalProjectForCandidate(host, "fixture", base, final)
	if err != nil {
		t.Fatal(err)
	}
	if fromSnapshotCalls != 1 || classifyCalls != 1 {
		t.Fatalf("candidate composition calls: FromSnapshot=%d ClassifyCandidate=%d", fromSnapshotCalls, classifyCalls)
	}
	if got := string(compiled.Snapshot.Files[legacyPath]); got != "legacy orders bytes\n" {
		t.Fatalf("candidate composition did not bind the modelled file's base bytes: %q", got)
	}
}
