package projectonboarding

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func crlfGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}

// A committed onboarding checked out through core.autocrlf (the Git for
// Windows default) previews as unchanged, and an update keeps CRLF endings.
func TestOnboardingIsIdempotentOnACRLFCheckout(t *testing.T) {
	root, project := onboardingRepo(t)
	plan, err := Preview(root, project.Report.ModelDigest, defaultOptions(Claude, Codex))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, plan, plan.Digest); err != nil {
		t.Fatal(err)
	}
	crlfGit(t, root, "add", "--all")
	crlfGit(t, root, "commit", "-m", "onboard")
	crlfGit(t, root, "config", "core.autocrlf", "true")
	for _, file := range plan.Files {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(file.Path))); err != nil {
			t.Fatal(err)
		}
	}
	crlfGit(t, root, "checkout", "--", ".")
	workflow := filepath.Join(root, filepath.FromSlash(workflowPath))
	sample, err := os.ReadFile(workflow)
	if err != nil || !usesCRLF(string(sample)) {
		t.Fatalf("fixture checkout is not CRLF; err=%v", err)
	}
	second, err := Preview(root, project.Report.ModelDigest, defaultOptions(Claude, Codex))
	if err != nil {
		t.Fatalf("preview on CRLF checkout: %v", err)
	}
	for _, file := range second.Files {
		if file.Action != "unchanged" {
			t.Fatalf("%s is %s on an untouched CRLF checkout", file.Path, file.Action)
		}
	}

	// Stale guidance inside the managed block is updated with CRLF endings only.
	edited := strings.Replace(string(sample), "Markitect project workflow", "Outdated workflow", 1)
	if err := os.WriteFile(workflow, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := Preview(root, project.Report.ModelDigest, defaultOptions(Claude, Codex))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, third, third.Digest); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(workflow)
	if err != nil || !usesCRLF(string(updated)) || string(updated) != string(sample) {
		t.Fatalf("update did not restore the CRLF workflow; err=%v", err)
	}
}

func TestMergeFileKeepsOnlyConsistentCRLF(t *testing.T) {
	desired := managedBlock("guidance")
	merged, err := mergeFile(workflowPath, "intro\r\n"+strings.ReplaceAll(desired, "\n", "\r\n")+"\r\n", desired)
	if err != nil || !usesCRLF(merged) {
		t.Fatalf("CRLF merge = %q, err=%v", merged, err)
	}
	mixed := "intro\r\n" + desired + "\n"
	merged, err = mergeFile(workflowPath, mixed, desired)
	if err != nil || merged != mixed {
		t.Fatalf("mixed line endings were rewritten: %q, err=%v", merged, err)
	}
}
