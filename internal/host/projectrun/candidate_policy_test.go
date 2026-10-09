package projectrun

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
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
