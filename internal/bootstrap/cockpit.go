package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/authoring"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// cockpitService serves the cockpit over the selected state directories. A run
// configuration contributes its state directory; nothing else of it is used.
// Live views probe the state lock the way the answer command does, and an
// owner's answer goes through that command's owner channel operation. With a
// repository, the cockpit also observes the project and makes proposals for
// its .markitect inputs, published through the repository's guarded
// publisher while every observed state directory is locked.
type cockpitService struct {
	preparation *apply.PreparationService
	repository  *gitrepo.Repository
}

func (c cockpitService) Serve(ctx context.Context, request cli.CockpitRequest, ready func(url string)) error {
	states := append([]string(nil), request.States...)
	if request.RunConfig != "" {
		cfg, err := loadRunConfig(ctx, request.RunConfig)
		if err != nil {
			return err
		}
		states = append(states, cfg.StateDirectory)
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
	cfg := cockpit.Config{States: states, Liveness: liveness, Answer: writeAnswer}
	var proposals *authoring.Service
	if request.Repository != "" {
		repository, err := filepath.Abs(request.Repository)
		if err != nil {
			return err
		}
		cfg.Project = func(ctx context.Context) (cockpit.ProjectInfo, error) {
			return c.project(ctx, repository)
		}
		proposals = c.authoring(repository, states, fence)
		cfg.Authoring = proposals
	}
	server, err := cockpit.New(cfg)
	if err != nil {
		return err
	}
	err = server.Serve(ctx, request.Listen, ready)
	if proposals != nil {
		proposals.Wait()
	}
	return err
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
func (c cockpitService) project(ctx context.Context, repository string) (cockpit.ProjectInfo, error) {
	info := cockpit.ProjectInfo{Repository: repository}
	selection, err := repositorySelection(repository)
	if err != nil {
		return info, err
	}
	result, err := c.preparation.Preview(ctx, apply.PreviewRequest{Selection: selection}, model.Limits{})
	info.Status = string(result.Status())
	attempts := result.Attempts()
	if len(attempts) == 0 {
		return info, errors.Join(err, errors.New("structural preparation made no attempt"))
	}
	final := attempts[len(attempts)-1]
	info.Head = final.Model().Inputs().Head
	if selections, complete := final.Contexts().Selections(); complete {
		info.Packages = len(selections)
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
func (c cockpitService) authoring(repository string, states []string, fence *fencedStates) *authoring.Service {
	capture := func(ctx context.Context) (apply.CapturedPreparation, error) {
		selection, err := repositorySelection(repository)
		if err != nil {
			return apply.CapturedPreparation{}, err
		}
		captured, err := c.preparation.Capture(ctx, apply.CaptureRequest{Selection: selection}, model.Limits{})
		if errors.Is(err, gitrepo.ErrDirty) {
			err = fmt.Errorf("%w (%v)", authoring.ErrNotClean, err)
		}
		return captured, err
	}
	lock := func(ctx context.Context) (apply.ControlCheck, func(), error) {
		var held stateLocks
		for _, dir := range states {
			l, err := lockStateWithin(ctx, dir, startLockWait)
			if err != nil {
				held.release()
				if errors.Is(err, errStateLocked) {
					return nil, nil, fmt.Errorf("an Apply holds the state directory %s; accept after it ends", dir)
				}
				return nil, nil, err
			}
			held = append(held, l)
		}
		fence.set(states, true)
		return held, func() {
			fence.set(states, false)
			held.release()
		}, nil
	}
	return authoring.New(repository, capture, c.repository, lock)
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
