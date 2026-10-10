package projectrun

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestFullProposalCannotWriteIgnoredPathWithinOwnedArtifactScope(t *testing.T) {
	config := Config{CoverageMode: "full", InventoryRoots: []string{"old-root"}}
	input := &Snapshot{Files: map[string][]byte{projectcoverage.IgnorePath: []byte("apiVersion: " + projectcoverage.IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries:\n  - path: src/cache/\n    reason: Local disposable cache\n")}}
	report := projectmodel.Report{
		Managers:  []projectmodel.Manager{{ID: "owner", Owns: []string{"src/"}}},
		Artifacts: []projectmodel.Artifact{{ID: "source", Owner: "owner", Paths: []string{"src/", "src/cache/", "src/cache/data.txt"}}},
	}
	task := ManagerTask{ManagerID: "owner", Artifacts: []string{"source"}}
	allowed := allowedWritePaths(config, report, task, "work", nil, input)
	if !containsString(allowed, "src/") || containsString(allowed, "src/cache/") || containsString(allowed, "src/cache/data.txt") {
		t.Fatalf("scope should retain nonignored source and remove exact ignored admissions: %v", allowed)
	}
	limits := Limits{MaxCandidateFileBytes: 1024, MaxCandidateBytes: 2048}
	base := candidateData{Files: map[string]File{}}
	_, err := applyProposal(base, []agentexec.CandidateFile{{Path: "src/cache/data.txt", Mode: "0644", Content: "forbidden"}}, config, report, task, "work", nil, limits, input)
	if err == nil || !strings.Contains(err.Error(), "explicitly ignored") {
		t.Fatalf("owned broad artifact bypassed ignore policy: %v", err)
	}
	accepted, err := applyProposal(base, []agentexec.CandidateFile{{Path: "src/realization.txt", Mode: "0644", Content: "allowed outside old roots"}}, config, report, task, "work", nil, limits, input)
	if err != nil || accepted.Files["src/realization.txt"].Path == "" {
		t.Fatalf("nonignored realization should be writable: %v", err)
	}
}

// BUG-01: a top-level Manager delegated exactly its root's selector owns, and
// may write, the files under it.
func TestTopLevelDelegateWritesFilesUnderItsRootSelector(t *testing.T) {
	api := projectmodel.APIVersion
	parent := map[string]any{"apiVersion": api, "kind": "Manager", "namespace": "", "name": "root"}
	model, diagnostics := core.Compile([]core.Schema{projectmodel.Schema()}, []core.Definition{
		{APIVersion: api, Kind: "Manager", Metadata: core.Metadata{Name: "root"}, Purpose: "Root manager.", Spec: map[string]any{"owns": []any{"src/", "docs/"}}},
		{APIVersion: api, Kind: "Manager", Metadata: core.Metadata{Namespace: "backend", Name: "backend"}, Purpose: "Backend manager.", Spec: map[string]any{"parent": parent, "owns": []any{"src/"}}},
	}, "delegated-root-selector")
	if len(diagnostics) != 0 {
		t.Fatalf("compile: %+v", diagnostics)
	}
	report := projectmodel.Analyze(model, []projectmodel.File{{Path: "src/main.go", Digest: "sha256:main", Mode: "100644"}})
	backend := core.DefinitionIdentity{APIVersion: api, Kind: "Manager", Namespace: "backend", Name: "backend"}.Key()
	allowed := allowedWritePaths(Config{InventoryRoots: []string{"src", "docs"}}, report, ManagerTask{ManagerID: backend}, "work", nil)
	if !containsString(allowed, "src/main.go") {
		t.Fatalf("backend cannot write its own src/main.go: allowed=%v files=%+v", allowed, report.Files)
	}
}
