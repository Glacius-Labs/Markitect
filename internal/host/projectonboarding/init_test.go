package projectonboarding

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

func TestInitOnUnbornFeatureBranchCanPreviewAndApplyOnboardingIdempotently(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-b", "codex/unborn-onboarding"}, {"config", "user.name", "Onboarding Test"}, {"config", "user.email", "onboarding@example.invalid"}} {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# New project\n"), 0644); err != nil {
		t.Fatal(err)
	}
	agentsPrefix := []byte("# Existing contributor guidance\r\nKeep this text and its line endings.\r\n\r\n")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), agentsPrefix, 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := projectwork.Init(root, "unborn-onboarding-fixture", true); err != nil {
		t.Fatalf("initialize Markitect project without a commit: %v", err)
	}
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatalf("load initialized project without a commit: %v", err)
	}
	if got := projectwork.DocumentPath(project.Config); got != "docs/markitect/project.md" {
		t.Fatalf("document path = %q, want initialized default", got)
	}
	headCheck := exec.Command("git", "rev-parse", "--verify", "HEAD")
	headCheck.Dir = root
	if _, err := headCheck.Output(); err == nil {
		t.Fatal("fixture unexpectedly has a committed HEAD")
	}

	plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex))
	if err != nil {
		t.Fatalf("preview onboarding on unborn feature branch: %v", err)
	}
	if plan.Branch != "codex/unborn-onboarding" || plan.Head != "unborn:refs/heads/codex/unborn-onboarding" || plan.Digest == "" {
		t.Fatalf("preview did not bind unborn branch state: branch=%q head=%q digest=%q", plan.Branch, plan.Head, plan.Digest)
	}
	first, err := Apply(root, plan, plan.Digest)
	if err != nil {
		t.Fatalf("apply onboarding on unborn feature branch: %v", err)
	}
	if len(first.Written) == 0 {
		t.Fatal("first apply wrote no onboarding files")
	}
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(agents, agentsPrefix) {
		t.Fatalf("existing AGENTS.md bytes outside managed block were changed:\n%q", agents)
	}

	secondPlan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Codex))
	if err != nil {
		t.Fatalf("preview idempotent onboarding: %v", err)
	}
	for _, file := range secondPlan.Files {
		if file.Action != "unchanged" {
			t.Errorf("second preview action for %s = %q, want unchanged", file.Path, file.Action)
		}
	}
	second, err := Apply(root, secondPlan, secondPlan.Digest)
	if err != nil {
		t.Fatalf("apply idempotent onboarding: %v", err)
	}
	if len(second.Written) != 0 {
		t.Fatalf("idempotent apply rewrote files: %v", second.Written)
	}
	agentsAfter, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(agents, agentsAfter) {
		t.Fatal("second onboarding apply changed AGENTS.md")
	}
	headCheck = exec.Command("git", "rev-parse", "--verify", "HEAD")
	headCheck.Dir = root
	if _, err := headCheck.Output(); err == nil {
		t.Fatal("onboarding unexpectedly created a Git commit")
	}
}
