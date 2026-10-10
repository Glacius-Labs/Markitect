package projectrun

import (
	"bytes"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func workspaceDeltaFixture() (projectwork.Config, projectmodel.Report, ManagerTask, Limits) {
	manager := ManagerTask{ManagerID: "manager", Owns: []string{"src/"}}
	config := projectwork.Config{CoverageMode: "selected", InventoryRoots: []string{"src/"}}
	report := projectmodel.Report{Managers: []projectmodel.Manager{{ID: manager.ManagerID, Owns: []string{"src/"}}}}
	limits := Limits{MaxCandidateFileBytes: 16, MaxCandidateBytes: 32}
	return config, report, manager, limits
}

func TestApplyWorkspaceDeltaPreservesBinaryBytesAndModes(t *testing.T) {
	config, report, manager, limits := workspaceDeltaFixture()
	content := []byte{0x00, 0xff, 0x80, 'x'}
	out, err := applyWorkspaceDelta(candidateData{Files: map[string]File{}}, projectworkspace.Delta{
		Changes: []projectworkspace.Change{{Kind: projectworkspace.ChangeAdd, Path: "src/blob.bin", Mode: "100755", Content: content}},
	}, nil, config, report, manager, "work", nil, limits, nil)
	if err != nil {
		t.Fatal(err)
	}
	file := out.Files["src/blob.bin"]
	if file.Mode != "100755" || !bytes.Equal(file.Content, content) {
		t.Fatalf("observed file = mode %q bytes %v", file.Mode, file.Content)
	}
	content[0] = 0x7f
	if out.Files["src/blob.bin"].Content[0] != 0x00 {
		t.Fatal("candidate aliases caller-owned delta bytes")
	}
}

func TestApplyWorkspaceDeltaRepresentsDeleteAndRename(t *testing.T) {
	config, report, manager, limits := workspaceDeltaFixture()
	report.Files = []projectmodel.FileEntry{{Path: "src/old.bin", Owner: manager.ManagerID, Exists: true}, {Path: "src/remove.bin", Owner: manager.ManagerID, Exists: true}}
	base := candidateData{Files: map[string]File{"src/prior.txt": {Path: "src/prior.txt", Mode: "100644", Content: []byte("prior")}}}
	content := []byte("renamed bytes")
	out, err := applyWorkspaceDelta(base, projectworkspace.Delta{Changes: []projectworkspace.Change{
		{Kind: projectworkspace.ChangeDelete, Path: "src/remove.bin"},
		{Kind: projectworkspace.ChangeRename, OldPath: "src/old.bin", Path: "src/new.bin", Mode: "100644", Content: content},
	}}, []agentexec.CandidateFile{{Path: "src/new.bin", Mode: "0644", Content: string(content)}}, config, report, manager, "work", nil, limits, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Files["src/remove.bin"].Delete || !out.Files["src/old.bin"].Delete {
		t.Fatalf("delete/rename source were not represented as deletes: %+v", out.Files)
	}
	if got := out.Files["src/new.bin"]; got.Delete || got.Mode != "100644" || string(got.Content) != string(content) {
		t.Fatalf("rename destination = %+v", got)
	}
	if string(base.Files["src/prior.txt"].Content) != "prior" {
		t.Fatal("base candidate was changed")
	}
}

func TestApplyWorkspaceDeltaRejectsOutOfScopeWithoutMutatingBase(t *testing.T) {
	config, report, manager, limits := workspaceDeltaFixture()
	base := candidateData{Files: map[string]File{"src/existing.txt": {Path: "src/existing.txt", Mode: "100644", Content: []byte("unchanged")}}}
	_, err := applyWorkspaceDelta(base, projectworkspace.Delta{Changes: []projectworkspace.Change{
		{Kind: projectworkspace.ChangeAdd, Path: "outside/file.txt", Mode: "100644", Content: []byte("bad")},
	}}, nil, config, report, manager, "work", nil, limits, nil)
	if err == nil {
		t.Fatal("out-of-scope workspace delta was accepted")
	}
	if len(base.Files) != 1 || string(base.Files["src/existing.txt"].Content) != "unchanged" {
		t.Fatalf("error mutated base candidate: %+v", base.Files)
	}
}

func TestApplyWorkspaceDeltaRejectsUnobservedAndContradictoryModelWrites(t *testing.T) {
	config, report, manager, limits := workspaceDeltaFixture()
	observed := projectworkspace.Delta{Changes: []projectworkspace.Change{
		{Kind: projectworkspace.ChangeAdd, Path: "src/observed.txt", Mode: "100644", Content: []byte("observed")},
	}}
	cases := []struct {
		name      string
		proposals []agentexec.CandidateFile
	}{
		{
			name:      "unobserved write",
			proposals: []agentexec.CandidateFile{{Path: "src/unobserved.txt", Mode: "0644", Content: "invented"}},
		},
		{
			name:      "contradictory bytes",
			proposals: []agentexec.CandidateFile{{Path: "src/observed.txt", Mode: "0644", Content: "different"}},
		},
		{
			name:      "contradictory mode",
			proposals: []agentexec.CandidateFile{{Path: "src/observed.txt", Mode: "0755", Content: "observed"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := candidateData{Files: map[string]File{"src/prior.txt": {Path: "src/prior.txt", Mode: "100644", Content: []byte("prior")}}}
			_, err := applyWorkspaceDelta(base, observed, tc.proposals, config, report, manager, "work", nil, limits, nil)
			if err == nil {
				t.Fatal("invalid model candidate proposal was accepted")
			}
			if len(base.Files) != 1 || string(base.Files["src/prior.txt"].Content) != "prior" {
				t.Fatalf("error mutated base candidate: %+v", base.Files)
			}
		})
	}
}

func TestApplyWorkspaceDeltaEnforcesByteLimits(t *testing.T) {
	config, report, manager, limits := workspaceDeltaFixture()
	limits.MaxCandidateFileBytes = 3
	_, err := applyWorkspaceDelta(candidateData{Files: map[string]File{}}, projectworkspace.Delta{Changes: []projectworkspace.Change{
		{Kind: projectworkspace.ChangeAdd, Path: "src/large.bin", Mode: "100644", Content: []byte("four")},
	}}, nil, config, report, manager, "work", nil, limits, nil)
	if err == nil {
		t.Fatal("oversized workspace delta was accepted")
	}
}
