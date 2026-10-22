package gitrepo

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Volume/census peers remain synthetic; every Git helper executes real Git in
// a fresh temporary directory. Empty materialization needs no filesystem shim.
type identityRuntime struct {
	*workspaceFake
	git          *integrationRuntime
	t            *testing.T
	failIdentity string
}

func (f *identityRuntime) CreateVolumeExclusive(ctx context.Context, name string, labels map[string]string) (container.VolumeObservation, error) {
	result, err := f.workspaceFake.CreateVolumeExclusive(ctx, name, labels)
	if err == nil {
		f.git.roots[name] = f.t.TempDir()
	}
	return result, err
}

func (f *identityRuntime) RunHelper(ctx context.Context, spec container.ContainerSpec, argv []string, stdin io.Reader, stdout io.Writer) error {
	if argv[0] != "/usr/bin/git" {
		return f.workspaceFake.RunHelper(ctx, spec, argv, stdin, stdout)
	}
	if f.failIdentity != "" && strings.Contains(strings.Join(argv, " "), "config --local --replace-all "+f.failIdentity) {
		return errors.New("controlled local identity write failure")
	}
	return f.git.RunHelper(ctx, spec, argv, stdin, stdout)
}

func identityFixture(t *testing.T) (*Workspaces, *identityRuntime) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git unavailable")
	}
	w, fake := workspaceAdapter(t)
	runtime := &identityRuntime{workspaceFake: fake, git: &integrationRuntime{roots: make(map[string]string)}, t: t}
	w.runtime = runtime
	return w, runtime
}

func ordinaryAssignmentGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	// Model an Agent process with no author/committer environment overrides.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(key) {
		case "GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL":
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ordinary Git %v: %s %v", args, data, err)
	}
	return strings.TrimSpace(string(data))
}

func TestAssignmentCopiesHaveLocalIdentityBeforeOrdinaryGit(t *testing.T) {
	for _, global := range []string{"", "[user]\n name = Private User\n email = private@example.test\n[author]\n name = Private Author\n email = author@example.test\n[committer]\n name = Private Committer\n email = committer@example.test\n"} {
		t.Run(map[bool]string{true: "absent", false: "conflicting"}[global == ""], func(t *testing.T) {
			config := filepath.Join(t.TempDir(), "global.gitconfig")
			if err := os.WriteFile(config, []byte(global), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GIT_CONFIG_GLOBAL", config)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			t.Setenv("GIT_CONFIG_COUNT", "0")
			w, runtime := identityFixture(t)
			ctx := context.Background()
			root, err := w.PrepareAssignment(ctx, run.WorkspaceSpec{ID: "root"})
			if err != nil {
				t.Fatal(err)
			}
			child, err := w.ForkAssignment(ctx, root, "manager")
			if err != nil {
				t.Fatal(err)
			}
			review, err := w.ForkAssignmentCommit(ctx, root, root.StartingCommit(), "review")
			if err != nil {
				t.Fatal(err)
			}
			for _, copy := range []run.AssignmentWorkspace{root, child, review} {
				directory := runtime.git.roots[copy.Volume()]
				for key, want := range map[string]string{
					"user.name": assignmentGitName, "user.email": assignmentGitEmail,
					"author.name": assignmentGitName, "author.email": assignmentGitEmail,
					"committer.name": assignmentGitName, "committer.email": assignmentGitEmail,
				} {
					if got := ordinaryAssignmentGit(t, directory, "config", "--local", "--get", key); got != want {
						t.Fatalf("%s local %s = %q, want %q", copy.ID(), key, got, want)
					}
				}
				want := assignmentGitName + "|" + assignmentGitEmail + "|" + assignmentGitName + "|" + assignmentGitEmail
				if got := ordinaryAssignmentGit(t, directory, "log", "-1", "--format=%an|%ae|%cn|%ce", copy.StartingCommit()); got != want {
					t.Fatalf("%s Host starting commit identity = %q, want %q", copy.ID(), got, want)
				}
				ordinaryAssignmentGit(t, directory, "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "ordinary Agent commit")
				if got := ordinaryAssignmentGit(t, directory, "log", "-1", "--format=%an|%ae|%cn|%ce"); got != want {
					t.Fatalf("%s ordinary commit identity = %q, want %q", copy.ID(), got, want)
				}
			}
			if data, err := os.ReadFile(config); err != nil || string(data) != global {
				t.Fatalf("global user config changed: %q %v", data, err)
			}
		})
	}
}

func TestAssignmentHostIdentityOverridesSpecificGlobalSettings(t *testing.T) {
	config := filepath.Join(t.TempDir(), "global.gitconfig")
	global := "[author]\n name = Private Author\n email = author@example.test\n[committer]\n name = Private Committer\n email = committer@example.test\n"
	if err := os.WriteFile(config, []byte(global), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "0")
	// This fixture initializes without local identity; its real Host commits
	// therefore exercise command overrides, also used in temporary integration.
	_, copies, roots := integrationFixture(t, "host")
	want := assignmentGitName + "|" + assignmentGitEmail + "|" + assignmentGitName + "|" + assignmentGitEmail
	if got := ordinaryAssignmentGit(t, roots["host"], "log", "-1", "--format=%an|%ae|%cn|%ce", copies["host"].StartingCommit()); got != want {
		t.Fatalf("Host identity = %q, want %q", got, want)
	}
	if data, err := os.ReadFile(config); err != nil || string(data) != global {
		t.Fatalf("global user config changed: %q %v", data, err)
	}
}

func TestAssignmentIdentityFailureRetainsIncompleteCopy(t *testing.T) {
	w, runtime := identityFixture(t)
	ctx := context.Background()
	runtime.failIdentity = "user.email"
	partial, err := w.PrepareAssignment(ctx, run.WorkspaceSpec{ID: "partial"})
	if err == nil || partial == nil || partial.Volume() == "" || partial.StartingCommit() != "" || !strings.Contains(err.Error(), "user.email") {
		t.Fatalf("identity failure became a ready assignment: %+v %v", partial, err)
	}
	if !runtime.volumes[partial.Volume()].Exists {
		t.Fatal("partial volume was lost")
	}
	runtime.failIdentity = ""
	source, err := w.PrepareAssignment(ctx, run.WorkspaceSpec{ID: "source"})
	if err != nil {
		t.Fatal(err)
	}
	runtime.failIdentity = "user.name"
	partial, err = w.ForkAssignmentCommit(ctx, source, source.StartingCommit(), "partial-review")
	if err == nil || partial == nil || partial.Volume() == "" || partial.StartingCommit() != "" || !runtime.volumes[partial.Volume()].Exists {
		t.Fatalf("fork identity failure became a ready assignment or lost its volume: %+v %v", partial, err)
	}
}
