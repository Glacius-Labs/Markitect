package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type controlledCheckPreparer struct{}

const controlledCheckImage = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func (controlledCheckPreparer) PrepareCheckProfile(_ context.Context, p runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
	path, _ := snapshot.NewPath("observed-image-executable")
	entry, _ := snapshot.NewEntry(path, snapshot.Regular, []byte("controlled static image observation"))
	return runtimeconfig.ProfileObservation{InputBinding: p.Binding(), Status: model.Passed, Contract: runtimeconfig.ProfileContract{Name: "markitect.container-check", Version: 1}, CheckImage: &runtimeconfig.CheckImageObservation{ImageID: controlledCheckImage, Executable: runtimeconfig.AgentImageMetadataFile{Path: "/usr/bin/check", ContentID: entry.ContentID(), Size: int64(len(entry.Bytes()))}}}, nil
}

// Only the volume transport and command process are controlled peers. Project
// acquisition, file copies, directory readback, cleanup and persistence are real.
type controlledCheckWorkspaces struct {
	run.Workspaces
	mu     sync.Mutex
	copies []run.DisposableCopy
}

func (w *controlledCheckWorkspaces) Open(ctx context.Context, r run.Request) (run.WorkspaceService, error) {
	s, err := w.Workspaces.Open(ctx, r)
	return controlledCheckService{WorkspaceService: s, owner: w}, err
}

type controlledCheckService struct {
	run.WorkspaceService
	owner *controlledCheckWorkspaces
}

func (s controlledCheckService) Copy(ctx context.Context, name string, base run.Commit) (run.Copy, error) {
	c, err := s.WorkspaceService.Copy(ctx, name, base)
	if c != nil {
		return controlledCheckCopy{Copy: c, owner: s.owner}, err
	}
	return c, err
}
func (s controlledCheckService) Retain(ctx context.Context, c run.Copy, fixed run.Commit) (project.CandidateRef, error) {
	return s.WorkspaceService.Retain(ctx, actualCopy(c), fixed)
}

type controlledCheckCopy struct {
	run.Copy
	owner *controlledCheckWorkspaces
}

// Git operations between Run copies use the actual local copies.
func actualCopy(c run.Copy) run.Copy {
	if wrapped, ok := c.(controlledCheckCopy); ok {
		return wrapped.Copy
	}
	return c
}
func (c controlledCheckCopy) Update(ctx context.Context, source run.Copy, base run.Commit) error {
	return c.Copy.Update(ctx, actualCopy(source), base)
}
func (c controlledCheckCopy) Integrate(ctx context.Context, source run.Copy, fixed run.Commit) (run.Commit, error) {
	return c.Copy.Integrate(ctx, actualCopy(source), fixed)
}
func (c controlledCheckCopy) Review(ctx context.Context, source run.Copy, base, result run.Commit) (string, error) {
	return c.Copy.Review(ctx, actualCopy(source), base, result)
}

func (c controlledCheckCopy) Materialize(ctx context.Context, tree snapshot.Tree, dirs []snapshot.Path) (run.DisposableCopy, error) {
	actual, err := c.Copy.Materialize(ctx, tree, dirs)
	if actual == nil {
		return nil, err
	}
	c.owner.mu.Lock()
	c.owner.copies = append(c.owner.copies, actual)
	id := len(c.owner.copies)
	c.owner.mu.Unlock()
	return controlledVolumeCopy{DisposableCopy: actual, id: id}, err
}

type controlledVolumeCopy struct {
	run.DisposableCopy
	id int
}

func (c controlledVolumeCopy) Location() run.CopyLocation {
	return run.CopyLocation{Volume: "controlled-check-volume", Subpath: fmt.Sprintf("copy-%d", c.id)}
}

type controlledCheckCommand struct {
	owner *controlledCheckWorkspaces
	kind  string
	calls int
	t     *testing.T
}

func (c *controlledCheckCommand) RunCommand(_ context.Context, spec container.ContainerSpec, args, env []string, max int64) (container.CommandResult, error) {
	c.calls++
	c.owner.mu.Lock()
	copies := append([]run.DisposableCopy(nil), c.owner.copies...)
	c.owner.mu.Unlock()
	if len(copies) != 2*c.calls {
		c.t.Error("check did not receive independent input/home")
		return container.CommandResult{}, errors.New("missing check copies")
	}
	input, home := copies[len(copies)-2].Location().Directory, copies[len(copies)-1].Location().Directory
	if _, err := os.Stat(filepath.Join(input, ".markitect/model/b/empty")); err != nil {
		c.t.Fatal("empty canonical Package lost before declared check", err)
	}
	if data, err := os.ReadFile(filepath.Join(input, "out/accounted.ignored")); err != nil || string(data) != "original ignored input\n" {
		c.t.Fatal("fixed input bytes lost", err)
	}
	if entries, err := os.ReadDir(home); err != nil || len(entries) != 0 {
		c.t.Fatal("check home is not empty", entries, err)
	}
	if len(spec.Mounts) != 2 || spec.Mounts[0].Volume != spec.Mounts[1].Volume || spec.Mounts[0].Subpath == spec.Mounts[1].Subpath || !spec.Mounts[0].NoCopy || !spec.Mounts[1].NoCopy {
		c.t.Fatal("same Run volume subpaths were confused", spec.Mounts)
	}
	if !reflect.DeepEqual(args, []string{"/usr/bin/check", "literal $(argument)"}) || len(env) != 0 || max != 4096 {
		c.t.Fatal("declared literal command changed", args, env, max)
	}
	r := container.CommandResult{ImageID: controlledCheckImage, Started: true, Exited: true, Output: []byte("controlled command observation")}
	if c.kind == "nonzero" {
		r.ExitCode = 23
	}
	if c.kind == "lost observation" {
		return r, errors.New("injected command observation failure")
	}
	return r, nil
}

func TestDeclaredCheckUsesFixedCopiesDirectoriesAndDurableObservations(t *testing.T) {
	r, old, g, _ := runFixture(t)
	r.Write(".markitect/execution.yaml", "apiVersion: markitect.execution/v2\nchecks:\n  - id: declared\n    profile: controlled\n    directory: .\n    args: ['literal $(argument)']\n    timeout: 2s\n")
	r.Commit("Declare one actual fixed-input check")
	ctx := context.Background()
	tree, err := withDirectories(ctx, r.Dir, r.Git("rev-parse", "HEAD^{tree}"), []string{".markitect/model/b/empty"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := localCommand(ctx, r.Dir, []byte("Preserve empty Package\n"), "commit-tree", tree, "-p", r.Git("rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	r.Git("reset", "--mixed", id)
	repository := adapter(t)
	captured, err := project.NewPreparationService(project.NewModelService(repository, repository)).Capture(ctx, project.CaptureRequest{Selection: old.Selection()}, model.Limits{})
	a, ok := captured.Prepared()
	if err != nil || !ok {
		t.Fatal(captured.Status(), err)
	}
	p, err := project.Prepare(a)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := g.Capture(ctx, "checked", p)
	if err != nil {
		t.Fatal(err)
	}
	engine, _ := os.Executable()
	document := fmt.Sprintf("apiVersion: markitect.runtime/v1\nagentProfiles: {}\ncheckProfiles:\n  controlled:\n    container:\n      profileVersion: 1\n      engineExecutable: %q\n      imageID: %s\n      executable: /usr/bin/check\n      argumentPrefix: []\n      env: []\n      limits: {milliCPUs: 1000, memoryBytes: 268435456, processes: 64}\n      maxOutputBytes: 4096\n", engine, controlledCheckImage)
	filePath := fixturePath(t, "runtime.yaml")
	file, _ := snapshot.NewEntry(filePath, snapshot.Regular, []byte(document))
	metadata, err := runtimeconfig.Prepare(ctx, project.RawInputs{RuntimeProfiles: []project.SourceFile{{Location: project.SourceLocation{Boundary: "controlled-profile", Path: filePath}, File: file}}}, nil, controlledCheckPreparer{})
	if err != nil || metadata.Status != model.Passed {
		t.Fatal(metadata, err)
	}
	// "checked before ..." also has Root run the check tool on its state before
	// judging the same state: the declared check runs and is recorded once.
	for _, kind := range []string{"pass", "nonzero", "lost observation", "invalid observation", "assessment readiness", "checked before verify", "checked before implement"} {
		t.Run(kind, func(t *testing.T) {
			workspaces := &controlledCheckWorkspaces{Workspaces: g}
			command := &controlledCheckCommand{owner: workspaces, kind: kind, t: t}
			mandate := run.Verify
			if kind == "assessment readiness" {
				// The fixture has no realized Go project. A product check would
				// fail before Implement, which must not reject model readiness.
				mandate = run.Assessment
				command.kind = "nonzero"
			}
			if kind == "checked before implement" {
				mandate = run.Implement
			}
			checker, err := checks.NewCandidateRunner(metadata.Profiles, command, nil)
			if err != nil {
				t.Fatal(err)
			}
			observation := run.SessionObservation{ProfileIdentity: "sha256:" + strings.Repeat("b", 64), ImageID: "sha256:" + strings.Repeat("c", 64), HarnessDigest: "sha256:" + strings.Repeat("d", 64)}
			if kind == "invalid observation" {
				observation.HarnessDigest = ""
			}
			sessions := &controlledManagers{crossed: make(chan struct{}), observation: &observation, checkFirst: strings.HasPrefix(kind, "checked before")}
			if kind == "nonzero" {
				sessions.forceDecision = run.Reject
			}
			store, _ := runstore.New(g.StateRoot)
			executor := run.Executor{Workspaces: workspaces, Sessions: sessions, Records: store, Checks: checker}
			runID := strings.ReplaceAll(kind, " ", "-")
			result, executeErr := executor.Execute(ctx, run.Request{MaxManagers: 5, ID: runID, Project: "checked", Prepared: p, Input: ref, Mandate: mandate, Allowance: run.NewAllowance(runDeadline(t), 80), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second})
			if kind == "lost observation" || kind == "invalid observation" {
				if executeErr == nil || result.Technical == run.Completed {
					t.Fatal(result, executeErr)
				}
			} else if executeErr != nil || result.Technical != run.Completed {
				t.Fatal(result, executeErr)
			}
			if kind == "assessment readiness" {
				if result.Decision != run.Approve || command.calls != 0 || len(result.Checks) != 0 || len(result.Handoffs) != 5 {
					t.Fatal("unrealized product blocked complete model-readiness Assessment", result, command.calls)
				}
			} else if kind == "invalid observation" {
				if command.calls != 0 || len(result.Sessions) != 0 {
					t.Fatal("invalid native identity reached command execution")
				}
			} else {
				if command.calls != 1 || len(result.Checks) != 1 || len(result.Sessions) != 5 || result.Sessions[0].Observation != observation {
					t.Fatal("actual consumer observation lost", result)
				}
				if len(result.Checks[0].Receipt.Directories) == 0 || !result.Checks[0].Receipt.CopiesClosed {
					t.Fatal("directory/cleanup receipt lost", result.Checks)
				}
				if kind == "nonzero" && (result.Decision != run.Reject || result.Checks[0].Receipt.ExitCode != 23) {
					t.Fatal("known check failure became success", result)
				}
				for _, h := range result.Handoffs {
					if h.Manager == "." && strings.HasPrefix(kind, "checked before") && (h.Check == nil || !h.Check.Valid || !reflect.DeepEqual(h.Check.Checks, []string{"declared"}) || h.Check.Commit.ID != result.Checks[0].Commit) {
						t.Fatal("Root's judgment did not rest on its one recorded check", h.Check, result.Checks)
					}
				}
			}
			for _, copy := range workspaces.copies {
				if _, err = os.Stat(copy.Location().Directory); !os.IsNotExist(err) {
					t.Fatal("disposable copy survived", err)
				}
			}
			fresh, _ := runstore.New(g.StateRoot)
			saved, err := fresh.Read(ctx, "checked", runID)
			if err != nil || !reflect.DeepEqual(saved, result) {
				t.Fatal("provided session/check observations did not survive new-store readback", saved, err)
			}
		})
	}
}
