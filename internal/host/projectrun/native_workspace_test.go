package projectrun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

func nativeWorkspaceFixture(t *testing.T) (string, string, *Project, Agent) {
	t.Helper()
	root := t.TempDir()
	content := []byte("# Project instructions\nFollow the selected project contract.\n")
	path := "AGENTS.md"
	if err := os.WriteFile(filepath.Join(root, path), content, 0o644); err != nil {
		t.Fatal(err)
	}
	gitE2E(t, root, "init", "--quiet")
	gitE2E(t, root, "config", "user.name", "Markitect Fixture")
	gitE2E(t, root, "config", "user.email", "fixture@example.invalid")
	gitE2E(t, root, "add", path)
	gitE2E(t, root, "commit", "--quiet", "-m", "fixture")
	revision := gitE2E(t, root, "rev-parse", "HEAD")
	project := &Project{
		Revision: revision,
		Config:   projectwork.Config{},
		Snapshot: &snapshot.Snapshot{ID: revision, Files: map[string][]byte{path: content}, Modes: map[string]string{path: "100644"}},
	}
	pinnedPath, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	agent := Agent{
		WorkspaceMode:    "scoped",
		InstructionPaths: []string{path},
		RuntimeFiles:     []agentexec.RuntimeFile{{Path: pinnedPath, Mode: "0644", Digest: digestBytes(content)}},
	}
	return root, revision, project, agent
}

func TestBuildNativeWorkspaceSelectsFixedInstructionsAndChecksRuntimePin(t *testing.T) {
	root, revision, project, agent := nativeWorkspaceFixture(t)
	got, err := buildNativeWorkspace(root, revision, project, agent)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.APIVersion != nativeWorkspaceAPIVersion || len(got.Instructions) != 1 {
		t.Fatalf("native workspace context = %#v", got)
	}
	artifact := got.Instructions[0]
	if artifact.Path != "AGENTS.md" || artifact.Mode != "100644" || artifact.Digest != digestBytes(artifact.Content) {
		t.Fatalf("instruction artifact is not bound to fixed bytes: %#v", artifact)
	}

	withoutPin := agent
	withoutPin.RuntimeFiles = nil
	if _, err := buildNativeWorkspace(root, revision, project, withoutPin); err == nil {
		t.Fatal("native instruction without an absolute runtime pin was accepted")
	}
	wrongPin := agent
	wrongPin.RuntimeFiles = append([]agentexec.RuntimeFile(nil), agent.RuntimeFiles...)
	wrongPin.RuntimeFiles[0].Mode = "0755"
	if _, err := buildNativeWorkspace(root, revision, project, wrongPin); err == nil {
		t.Fatal("native instruction with a mismatched runtime mode pin was accepted")
	}
	wrongPin = agent
	wrongPin.RuntimeFiles = append([]agentexec.RuntimeFile(nil), agent.RuntimeFiles...)
	wrongPin.RuntimeFiles[0].Digest = digestBytes([]byte("different"))
	if _, err := buildNativeWorkspace(root, revision, project, wrongPin); err == nil {
		t.Fatal("native instruction with a mismatched runtime digest pin was accepted")
	}
}

func TestBuildNativeWorkspaceRejectsStaleInstructionAndUndeclaredPaths(t *testing.T) {
	root, revision, project, agent := nativeWorkspaceFixture(t)
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("changed after fixed revision\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := buildNativeWorkspace(root, revision, project, agent); err == nil {
		t.Fatal("working instruction that differs from fixed revision was accepted")
	}

	agent.InstructionPaths = []string{"docs/other.md"}
	if _, err := buildNativeWorkspace(root, revision, project, agent); err == nil {
		t.Fatal("instruction path outside exact project ToolPaths was accepted")
	}
	agent.InstructionPaths = []string{"AGENTS.md/"}
	if _, err := buildNativeWorkspace(root, revision, project, agent); err == nil {
		t.Fatal("noncanonical instruction path alias was accepted")
	}
	if !declaredNativeInstructionPath("docs/reference.md", projectwork.Config{DocumentPath: "docs/reference.md"}) {
		t.Fatal("exact declared Markdown reference ToolPath was rejected")
	}
}

func TestBuildNativeWorkspaceRejectsReparseInstruction(t *testing.T) {
	root, revision, project, agent := nativeWorkspaceFixture(t)
	path := filepath.Join(root, "AGENTS.md")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "instructions-target.md")
	if err := os.WriteFile(target, []byte("# linked instructions\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	if _, err := buildNativeWorkspace(root, revision, project, agent); err == nil {
		t.Fatal("reparse instruction path was accepted")
	}
}

func TestBuildNativeWorkspaceProposalModeRemainsCompatible(t *testing.T) {
	got, err := buildNativeWorkspace("", "", nil, Agent{})
	if err != nil || got != nil {
		t.Fatalf("empty workspace mode = %#v, %v; want nil, nil", got, err)
	}
}
