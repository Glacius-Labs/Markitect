package execution

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
)

func TestRecursiveMergeRejectsCollisionNonOwnedDeletionAndModeDrift(t *testing.T) {
	base := &snapshot.Snapshot{Files: map[string][]byte{"a.txt": []byte("a"), "b.txt": []byte("b")}, Modes: map[string]string{"a.txt": snapshot.RegularMode, "b.txt": snapshot.RegularMode}}
	tests := []struct {
		name    string
		prepare func(*snapshot.Snapshot, *snapshot.Snapshot)
		paths   []string
		want    string
	}{
		{"collision", func(assembled, branch *snapshot.Snapshot) {
			assembled.Files["a.txt"] = []byte("first")
			branch.Files["a.txt"] = []byte("second")
		}, []string{"a.txt"}, "collision"},
		{"non-owned", func(_, branch *snapshot.Snapshot) { branch.Files["b.txt"] = []byte("outside") }, []string{"a.txt"}, "non-owned"},
		{"deletion", func(_, branch *snapshot.Snapshot) { delete(branch.Files, "a.txt"); delete(branch.Modes, "a.txt") }, []string{"a.txt"}, "deletion"},
		{"mode collision", func(assembled, branch *snapshot.Snapshot) {
			assembled.Modes["a.txt"] = snapshot.ExecutableMode
			branch.Files["a.txt"] = []byte("new")
		}, []string{"a.txt"}, "collision"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assembled, branch := cloneSnapshot(base), cloneSnapshot(base)
			tc.prepare(assembled, branch)
			err := mergeBranch(assembled, base, branch, tc.paths)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("merge = %v, want %s", err, tc.want)
			}
		})
	}
	first, err := applyProposal(base, []string{"a.txt"}, []agentexec.CandidateFile{{Path: "a.txt", Mode: "0644", Content: "first"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := applyProposal(base, []string{"b.txt"}, []agentexec.CandidateFile{{Path: "b.txt", Mode: "0755", Content: "second"}})
	if err != nil {
		t.Fatal(err)
	}
	assembled := cloneSnapshot(base)
	if err := mergeBranch(assembled, base, first, []string{"a.txt"}); err != nil {
		t.Fatal(err)
	}
	if err := mergeBranch(assembled, base, second, []string{"b.txt"}); err != nil {
		t.Fatal(err)
	}
	if string(assembled.Files["a.txt"]) != "first" || string(assembled.Files["b.txt"]) != "second" || assembled.Modes["b.txt"] != snapshot.ExecutableMode {
		t.Fatal("actual child bytes/modes were not composed")
	}
	if string(base.Files["a.txt"]) != "a" || base.Modes["b.txt"] != snapshot.RegularMode {
		t.Fatal("branch integration mutated its immutable input")
	}
}
