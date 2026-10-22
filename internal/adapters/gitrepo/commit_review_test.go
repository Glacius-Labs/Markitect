package gitrepo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Only volume allocation and empty materialization are controlled. Clone,
// checkout, commits and every Git readback run the actual local Git executable.
type commitReviewRuntime struct {
	*workspaceFake
	git       *integrationRuntime
	t         *testing.T
	failClone bool
	paths     map[string]string
}

func (f *commitReviewRuntime) CreateVolumeExclusive(ctx context.Context, name string, labels map[string]string) (container.VolumeObservation, error) {
	v, err := f.workspaceFake.CreateVolumeExclusive(ctx, name, labels)
	if err == nil {
		f.git.roots[name] = f.t.TempDir()
		f.paths[name] = "/assignments/" + name
	}
	return v, err
}

func (f *commitReviewRuntime) RunHelper(ctx context.Context, spec container.ContainerSpec, argv []string, stdin io.Reader, stdout io.Writer) error {
	if argv[0] == "/usr/bin/git" {
		if f.failClone && strings.Contains(strings.Join(argv, " "), " clone ") {
			return errors.New("clone failed")
		}
		return f.git.RunHelper(ctx, spec, argv, stdin, stdout)
	}
	return f.workspaceFake.RunHelper(ctx, spec, argv, stdin, stdout)
}

type commitReviewAgent struct {
	t       *testing.T
	runtime *commitReviewRuntime
	spec    run.AssignmentSpec
	handler run.ToolHandler
	commit  string
}

func (a *commitReviewAgent) OpenAssignment(_ context.Context, spec run.AssignmentSpec, h run.ToolHandler) (run.AssignmentSession, error) {
	a.spec, a.handler = spec, h
	return a, nil
}
func (a *commitReviewAgent) Message(ctx context.Context, _ string, _ time.Duration) (run.TurnResult, error) {
	bytes, err := os.ReadFile(filepath.Join(a.runtime.git.roots[a.spec.Workspace], "common.txt"))
	if err != nil || string(bytes) != "shared baseline\n" {
		a.t.Fatalf("review did not receive exact older result: %q %v", bytes, err)
	}
	args, _ := json.Marshal(map[string]string{"commit": a.commit, "reason": "Inspected fixed result with actual local Git."})
	if answer := a.handler(ctx, run.AcceptResultTool, args); answer.Refused {
		a.t.Fatal(answer.Text)
	}
	return run.TurnResult{Completed: true, Final: "Reviewed the fixed result."}, nil
}
func (a *commitReviewAgent) Close(context.Context) (run.TurnResult, error) {
	return run.TurnResult{Stopped: true}, nil
}

func TestIndependentReviewForksExactCommitWithoutMutatingSource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	w, copies, roots := integrationFixture(t, "source")
	git := w.runtime.(*integrationRuntime)
	_, volumeFake := workspaceAdapter(t)
	volume := copies["source"].Volume()
	volumeFake.volumes[volume] = container.VolumeObservation{Name: volume, Exists: true, Labels: map[string]string{workspaceOwnerLabel: "owned"}}
	runtime := &commitReviewRuntime{workspaceFake: volumeFake, git: git, t: t, paths: w.gitPaths}
	w.runtime = runtime
	commissioned := copies["source"].StartingCommit()
	integrationWrite(t, roots["source"], "common.txt", "later committed correction\n")
	latest, err := copies["source"].Commit(ctx, "later source result")
	if err != nil {
		t.Fatal(err)
	}
	integrationWrite(t, roots["source"], "common.txt", "uncommitted further work\n")
	beforeStatus, err := copies["source"].git(ctx, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	agent := &commitReviewAgent{t: t, runtime: runtime, commit: commissioned}
	result, err := run.ReviewAssignment(ctx, w, agent, run.AssignmentReview{ID: "review", Purpose: run.Verify, Source: copies["source"], Commit: commissioned, Profile: "review", Context: "Fixed model", Timeout: time.Second, Transcript: filepath.Join(t.TempDir(), "review.jsonl"), CreateHome: func(ctx context.Context, name string) error {
		_, err := runtime.CreateVolumeExclusive(ctx, name, map[string]string{"markitect.review": "test"})
		return err
	}})
	if err != nil || result.AcceptedCommit != commissioned || result.Workspace.StartingCommit() != commissioned {
		t.Fatalf("real Git review: %+v %v", result, err)
	}
	afterHead, _ := copies["source"].git(ctx, "rev-parse", "HEAD")
	afterStatus, _ := copies["source"].git(ctx, "status", "--porcelain")
	content, _ := os.ReadFile(filepath.Join(roots["source"], "common.txt"))
	if afterHead != latest || afterStatus != beforeStatus || string(content) != "uncommitted further work\n" {
		t.Fatal("review changed the source HEAD, index or working files")
	}
	runtime.failClone = true
	partial, err := w.ForkAssignmentCommit(ctx, copies["source"], commissioned, "partial-review")
	if err == nil || partial == nil || partial.Volume() == "" || !runtime.volumes[partial.Volume()].Exists {
		t.Fatal("failed exact clone did not retain its partial copy")
	}
}
