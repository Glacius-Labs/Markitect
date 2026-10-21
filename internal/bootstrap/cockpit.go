package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/authoring"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// cockpitService serves the cockpit over the selected state directories. A run
// configuration contributes its state directory and, with a Runtime profile,
// the container engine, workspace helper, credentials and harvest limits of
// model sessions. Live views probe the state lock. With a repository, the cockpit also
// observes the project and makes proposals for its .markitect inputs, published through the
// repository's guarded publisher while every observed state directory is
// locked.
type cockpitService struct {
	preparation *apply.PreparationService
	repository  *gitrepo.Repository
	agents      apply.AgentProfilePreparer
	checks      apply.CheckProfilePreparer
	// execute is the execution function markitect apply runs. With a run
	// configuration and a Runtime profile the cockpit starts it in its own
	// process, so a run it started ends with it.
	execute cli.ApplyFunc
	// userConfigDir and setup stand in for os.UserConfigDir and the function
	// behind markitect setup; tests supply them.
	userConfigDir func() (string, error)
	setup         cli.SetupFunc
}

// projectSpec is what builds one project's Server: the state directories
// named directly, the run configuration with its Runtime profile, the
// repository selection and the product build a started run binds.
type projectSpec struct {
	states         []string
	runConfig      string
	runtimeProfile string
	selection      apply.InputSelection
	build          string
	// runsReason replaces the text that says why starting a run is
	// unavailable.
	runsReason string
}

// Serve serves the cockpit until ctx ends: the project the flags name, or,
// in project mode, the owner's per-user project list with the last project
// opened.
func (c cockpitService) Serve(ctx context.Context, request cli.CockpitRequest, ready func(url string)) error {
	host, err := cockpit.NewHost(ctx, c.hostConfig(request.Build))
	if err != nil {
		return err
	}
	if request.Projects {
		host.OpenLast()
	} else {
		spec := projectSpec{states: request.States, runConfig: request.RunConfig, runtimeProfile: request.RuntimeProfile, selection: request.Selection, build: request.Build}
		opened, err := c.open(ctx, spec)
		if err != nil {
			return err
		}
		host.Adopt(directProject(request), opened)
	}
	return host.Serve(ctx, request.Listen, ready)
}

// open builds a project's Server over the selected state directories. A run
// configuration contributes its state directory and, with a Runtime profile,
// the container engine, workspace helper, credentials and harvest limits of
// model sessions. Closing the project ends its model sessions, lets a started
// publication finish, and ends the run it started, waiting until the
// execution wrote the run's report and closed its evidence.
func (c cockpitService) open(ctx context.Context, spec projectSpec) (cockpit.OpenedProject, error) {
	states := append([]string(nil), spec.states...)
	runState := ""
	if spec.runConfig != "" {
		cfg, err := loadRunConfig(ctx, spec.runConfig)
		if err != nil {
			return cockpit.OpenedProject{}, err
		}
		states = append(states, cfg.StateDirectory)
		runState = cfg.StateDirectory
	}
	// The server and the fence name every state directory by its absolute path.
	for i, dir := range states {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return cockpit.OpenedProject{}, err
		}
		states[i] = abs
	}
	fence := &fencedStates{held: map[string]bool{}}
	// While the cockpit itself holds a state lock to publish a proposal, no
	// Apply holds it: the lock probe would otherwise report a run as running.
	liveness := func(dir string) (bool, error) {
		if fence.has(dir) {
			return false, nil
		}
		return stateHeld(dir)
	}
	cfg := cockpit.Config{States: states, Liveness: liveness, RunsReason: spec.runsReason}
	var runs *cockpit.RunController
	var proposals *authoring.Service
	if selection := spec.selection; selection.Repository != "" {
		cfg.Project = func(ctx context.Context) (cockpit.ProjectInfo, error) {
			return c.project(ctx, selection)
		}
		proposals = c.authoring(selection, states, fence)
		if spec.runConfig != "" && spec.runtimeProfile != "" {
			proposals.WithSessions(c.sessionOpener(selection, spec.runConfig, spec.runtimeProfile))
		}
		cfg.Authoring = proposals
		if spec.runConfig != "" && spec.runtimeProfile != "" && c.execute != nil {
			abs, err := filepath.Abs(runState)
			if err != nil {
				return cockpit.OpenedProject{}, err
			}
			runs = c.runController(ctx, selection, spec.runConfig, spec.runtimeProfile, abs, spec.build, fence)
			cfg.Runs = runs
		}
	}
	server, err := cockpit.New(cfg)
	if err != nil {
		return cockpit.OpenedProject{}, err
	}
	return cockpit.OpenedProject{Server: server, Close: func() error {
		if runs != nil {
			// A closing project ends the run it started and waits until the
			// execution wrote the run's report and closed its evidence.
			runs.Wait()
		}
		if proposals != nil {
			// A closing project ends its model sessions and lets a started
			// publication finish.
			err := proposals.CloseSessions(context.Background())
			proposals.Wait()
			return err
		}
		return nil
	}}, nil
}

// fencedStates are the state directories the cockpit holds to publish.
type fencedStates struct {
	mu   sync.Mutex
	held map[string]bool
}

func (f *fencedStates) has(dir string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.held[dir]
}

func (f *fencedStates) set(dirs []string, held bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, dir := range dirs {
		if held {
			f.held[dir] = true
		} else {
			delete(f.held, dir)
		}
	}
}

// repositorySelection selects a repository's own model at the default manifest
// and no dependency.
func repositorySelection(repository string) (apply.InputSelection, error) {
	manifest, err := snapshot.NewPath(".markitect/model.yaml")
	if err != nil {
		return apply.InputSelection{}, err
	}
	return apply.InputSelection{Repository: repository, Local: apply.ModelSelection{Boundary: ".", ManifestPath: manifest}}, nil
}

// project observes the repository's identity by structural preparation of its
// working tree, as markitect check does: model, canonical digest, head and
// Packages. It writes nothing.
func (c cockpitService) project(ctx context.Context, selection apply.InputSelection) (cockpit.ProjectInfo, error) {
	info := cockpit.ProjectInfo{Repository: selection.Repository}
	result, err := c.preparation.Preview(ctx, apply.PreviewRequest{Selection: selection}, model.Limits{})
	info.Status = string(result.Status())
	attempts := result.Attempts()
	if len(attempts) == 0 {
		return info, errors.Join(err, errors.New("structural preparation made no attempt"))
	}
	final := attempts[len(attempts)-1]
	info.Head = final.Model().Inputs().Head
	checked := final.Model().Checked()
	if len(checked.Models) > 0 {
		info.Model = cockpitModel(checked, final.Registry().Records(), final.Registry().Status())
		for _, scope := range checked.Models {
			if scope.Local {
				info.Definitions = len(scope.Definitions)
			}
		}
	}
	// The overview counts the model's own Packages, as the model browser does.
	packages := -1
	for _, scope := range checked.Models {
		if scope.Local {
			packages = len(scope.Packages)
		}
	}
	if selections, complete := final.Contexts().Selections(); complete {
		info.Packages = len(selections)
		if packages >= 0 {
			info.Packages = packages
		}
		if len(selections) > 0 {
			info.ModelID, info.ModelDigest = selections[0].ModelID(), selections[0].ModelDigest()
		}
	}
	return info, err
}

// authoring composes the proposal service: the committed capture Apply uses,
// the repository's guarded publisher, and a fence that holds the lock of every
// observed state directory, so no Apply observed here runs while an accepted
// proposal is written. An Apply started meanwhile refuses as if another Apply
// held its directory; the publication takes seconds.
func (c cockpitService) authoring(selection apply.InputSelection, states []string, fence *fencedStates) *authoring.Service {
	capture := func(ctx context.Context) (apply.CapturedPreparation, error) {
		captured, err := c.preparation.Capture(ctx, apply.CaptureRequest{Selection: selection}, model.Limits{})
		if errors.Is(err, gitrepo.ErrDirty) {
			err = fmt.Errorf("%w (%v)", authoring.ErrNotClean, err)
		}
		return captured, err
	}
	// The fence is marked before the locks are taken and cleared after they
	// are released, so the cockpit's own probe never sees its lock as a run.
	lock := func(ctx context.Context) (apply.ControlCheck, func(), error) {
		var held stateLocks
		fence.set(states, true)
		for _, dir := range states {
			l, err := lockStateWithin(ctx, dir, startLockWait)
			if err != nil {
				held.release()
				fence.set(states, false)
				if errors.Is(err, errStateLocked) {
					return nil, nil, fmt.Errorf("an Apply holds the state directory %s; accept after it ends", dir)
				}
				return nil, nil, err
			}
			held = append(held, l)
		}
		return held, func() {
			held.release()
			fence.set(states, false)
		}, nil
	}
	return authoring.New(selection.Repository, capture, c.repository, lock)
}

// stateLocks fences a publication to the holder of every lock.
type stateLocks []*stateLock

func (s stateLocks) Check(ctx context.Context) error {
	for _, l := range s {
		if err := l.Check(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s stateLocks) release() {
	for _, l := range s {
		_ = l.Release()
	}
}

// sessionOpener captures the committed source with its settings and the
// Runtime profile, as an Apply does, and composes a model session's ports from
// the run configuration: the container engine, the Agents of the prepared
// profiles with their mapped credentials, and the workspace helper. Session
// evidence goes to the run configuration's state directory.
func (c cockpitService) sessionOpener(base apply.InputSelection, runConfig, runtimeProfile string) authoring.SessionOpener {
	return func(ctx context.Context) (authoring.SessionStart, error) {
		cfg, a, err := c.captureWithRuntime(ctx, base, runConfig, runtimeProfile)
		if err != nil {
			return authoring.SessionStart{}, err
		}
		basis, err := apply.SessionBasisOf(a)
		if err != nil {
			return authoring.SessionStart{}, err
		}
		engine, err := container.NewEngine(cfg.EngineExecutable)
		if err != nil {
			return authoring.SessionStart{}, err
		}
		agents, err := newAgents(engine, a.Settings().ProfileObservations(), cfg)
		if err != nil {
			return authoring.SessionStart{}, err
		}
		helper := cfg.WorkspaceHelper
		helper.Workdir = "/work"
		workspaces, err := gitrepo.NewWorkspaces(engine, helper)
		if err != nil {
			return authoring.SessionStart{}, err
		}
		return authoring.SessionStart{Basis: basis, Ports: authoring.SessionPorts{
			Agents: agents, Workspaces: workspaces, RemoveVolume: engine.RemoveVolume, RequireVolumeSubpaths: engine.RequireVolumeSubpaths,
			MaxEntries: cfg.Harvest.MaxEntries, MaxBytes: cfg.Harvest.MaxBytes, Evidence: cfg.StateDirectory,
		}}, nil
	}
}

// withRuntimeProfile selects the Runtime profile as markitect apply does.
func withRuntimeProfile(base apply.InputSelection, runtimeProfile string) (apply.InputSelection, error) {
	profile, err := snapshot.NewPath(filepath.Base(runtimeProfile))
	if err != nil {
		return apply.InputSelection{}, err
	}
	base.RuntimeProfiles = []apply.SourceLocation{{Boundary: filepath.Dir(runtimeProfile), Path: profile}}
	return base, nil
}

// captureWithRuntime captures the committed source with its settings and the
// Runtime profile, as an Apply does, and requires that it passes. It is the
// start of a model session and the source of a run's confirmation facts.
func (c cockpitService) captureWithRuntime(ctx context.Context, base apply.InputSelection, configPath, runtimeProfile string) (runConfig, apply.CapturedSettingsAttempt, error) {
	cfg, err := loadRunConfig(ctx, configPath)
	if err != nil {
		return cfg, apply.CapturedSettingsAttempt{}, err
	}
	if err := checkStateOutsideRepository(cfg.StateDirectory, base.Repository); err != nil {
		return cfg, apply.CapturedSettingsAttempt{}, err
	}
	selection, err := withRuntimeProfile(base, runtimeProfile)
	if err != nil {
		return cfg, apply.CapturedSettingsAttempt{}, err
	}
	prepared, err := c.preparation.CaptureSettings(ctx, apply.CaptureRequest{Selection: selection}, model.Limits{}, c.agents, c.checks)
	if errors.Is(err, gitrepo.ErrDirty) {
		return cfg, apply.CapturedSettingsAttempt{}, fmt.Errorf("%w (%v)", authoring.ErrNotClean, err)
	}
	if err != nil {
		return cfg, apply.CapturedSettingsAttempt{}, err
	}
	a, ok := prepared.Prepared()
	if prepared.Status() != model.Passed || !ok {
		return cfg, apply.CapturedSettingsAttempt{}, fmt.Errorf("the committed source with the Runtime profile does not pass preparation (%s); run markitect check and markitect setup", prepared.Status())
	}
	return cfg, a, nil
}
