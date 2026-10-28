package agents_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/agents/claude"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// TestNativeClaudeMakerAndChecker is the bounded native qualification of the
// Claude Code adapter: a real Claude maker realizes one changed Definition
// against a real Run remote and its checker reviews the submission. It runs
// only under a written dispatch (docs/validation/) with the shared engine
// lock, and records the ends and the remote's branch in the evidence folder.
//
//	MARKITECT_TEST_ENGINE         absolute path of the Docker CLI
//	MARKITECT_TEST_MANAGER_IMAGE  the Manager image (packaging/runtime)
//	MARKITECT_TEST_CLAUDE_LOGIN   the host's Claude Code .credentials.json
//	MARKITECT_TEST_EVIDENCE       an empty folder for transcripts and ends
func TestNativeClaudeMakerAndChecker(t *testing.T) {
	login, evidence := os.Getenv("MARKITECT_TEST_CLAUDE_LOGIN"), os.Getenv("MARKITECT_TEST_EVIDENCE")
	if login == "" || evidence == "" {
		t.Skip("set MARKITECT_TEST_CLAUDE_LOGIN and MARKITECT_TEST_EVIDENCE under a dispatch")
	}
	r := newRealRun(t, map[string]string{
		project.ModelDir + "/members/member.md": def("std/Concept", "Member",
			"A person who borrows. A `Member` has a `Name`, the member's full name as text, and an `ID`, a positive whole number that no other member has."),
	})
	cfg := agents.Config{Provider: claude.Provider{}, Limits: agents.Limits{CPUs: 1, MemoryBytes: 2 << 30, Pids: 256, TmpBytes: 256 << 20},
		Network: "bridge", LoginFile: login, Model: "claude-sonnet-5-5", Effort: "high", Transcripts: filepath.Join(evidence, "transcripts")}
	record := func(name string, v any) {
		raw, _ := json.MarshalIndent(v, "", "  ")
		if err := os.WriteFile(filepath.Join(evidence, name), raw, 0o600); err != nil {
			t.Error(err)
		}
	}
	record("run.json", map[string]any{"run": r.run, "base": r.base, "image": r.imageID})
	maker := r.start(t, cfg, agentport.Maker, 15*time.Minute)
	record("maker-end.json", maker)
	t.Logf("maker: %+v", maker)
	branch := func() string {
		var out []byte
		out, _ = gitOutput(r, "log", "--format=%H %s%n%b", "managers/members")
		return string(out)
	}
	if maker.Reason != agentport.Finished || maker.Cleanup != "" {
		t.Fatalf("maker end %+v\nbranch:\n%s", maker, branch())
	}
	checker := r.start(t, cfg, agentport.Checker, 15*time.Minute)
	record("checker-end.json", checker)
	t.Logf("checker: %+v", checker)
	log := branch()
	if err := os.WriteFile(filepath.Join(evidence, "branch.txt"), []byte(log), 0o600); err != nil {
		t.Error(err)
	}
	if checker.Reason != agentport.Finished || checker.Cleanup != "" {
		t.Fatalf("checker end %+v\nbranch:\n%s", checker, log)
	}
}

// gitOutput runs Git on a clone of the remote's exported bundle.
func gitOutput(r *realRun, args ...string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "t4b-export")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	bundle, err := os.Create(filepath.Join(dir, "run.bundle"))
	if err != nil {
		return nil, err
	}
	err = r.remote.Export(r.ctx, bundle)
	bundle.Close()
	if err != nil {
		return nil, err
	}
	clone := filepath.Join(dir, "clone")
	if out, err := exec.Command("git", "clone", "-q", "--mirror", filepath.Join(dir, "run.bundle"), clone).CombinedOutput(); err != nil {
		return out, err
	}
	return exec.Command("git", append([]string{"-C", clone}, args...)...).CombinedOutput()
}
