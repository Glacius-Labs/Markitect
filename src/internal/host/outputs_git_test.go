package host

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func TestWriteOutputsUsesSelectedRepositoryForProtectedBranchGuard(t *testing.T) {
	root := tempRoot(t)
	gitInRepo(t, root, "init", "-b", "main")
	writeFixture(t, root, branchGuardFixtureFiles(t))
	p, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}

	foreign := t.TempDir()
	gitInRepo(t, foreign, "init", "-b", "feature/foreign")
	t.Setenv("GIT_DIR", filepath.Join(foreign, ".git"))

	if _, err := WriteOutputs(root, p); err == nil || !strings.Contains(err.Error(), "non-protected Git branch") {
		t.Fatalf("WriteOutputs error = %v, want protected-branch refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, "resources/sample/skills/entry.md")); !os.IsNotExist(err) {
		t.Fatalf("protected main branch received generated output (stat error: %v)", err)
	}
}

func TestWriteOutputsRejectsDetachedHeadInAnyProject(t *testing.T) {
	root := tempRoot(t)
	gitInRepo(t, root, "init", "-b", "feature/render")
	writeFixture(t, root, branchGuardFixtureFiles(t))
	gitInRepo(t, root, "add", "-A")
	gitInRepo(t, root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "fixture")
	gitInRepo(t, root, "checkout", "--detach")
	p, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteOutputs(root, p); err == nil || !strings.Contains(err.Error(), "detached HEAD") {
		t.Fatalf("WriteOutputs error = %v, want detached-HEAD refusal", err)
	}
}

func branchGuardFixtureFiles(t *testing.T) map[string][]byte {
	t.Helper()
	project := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "sample"}, Path: "markitect.yaml"}, Spec: authoring.Spec{
		Targets: []string{"codex"}, Areas: []authoring.Area{{Name: "sample", Path: "resources/sample"}},
	}}
	skill := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "entry", Namespace: "sample"}, Path: "resources/sample/skills/entry.yaml"}, Spec: authoring.Spec{Text: "Use the owner source."}}
	projectBytes, err := authoring.Encode(project)
	if err != nil {
		t.Fatal(err)
	}
	skillBytes, err := authoring.Encode(skill)
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{project.Path: projectBytes, skill.Path: skillBytes}
}

func gitInRepo(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-c", "safe.directory=" + filepath.ToSlash(root), "-C", root}, args...)...)
	command.Env = source.CleanGitEnv()
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
