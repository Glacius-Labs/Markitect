package checks

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
)

// CandidateRunner binds the declared suite to complete static profile evidence
// and the exact fixed Git candidate. Every command receives its own copy/home.
type CandidateRunner struct {
	profiles map[string]ContainerProfile
	runtime  CommandRuntime
	labels   map[string]string
}

func NewCandidateRunner(prepared []runtimeconfig.PreparedProfile, runtime CommandRuntime, labels map[string]string) (*CandidateRunner, error) {
	if runtime == nil {
		return nil, fmt.Errorf("missing concrete check runner")
	}
	c := &CandidateRunner{profiles: map[string]ContainerProfile{}, runtime: runtime, labels: maps.Clone(labels)}
	for _, p := range prepared {
		if p.Input().Namespace() != "checkProfiles" {
			continue
		}
		name := p.Input().Name()
		if _, exists := c.profiles[name]; exists {
			return nil, fmt.Errorf("duplicate prepared check profile")
		}
		profile, err := ContainerProfileFromPrepared(p)
		if err != nil {
			return nil, err
		}
		c.profiles[name] = profile
	}
	return c, nil
}

func (c *CandidateRunner) Check(ctx context.Context, declared []project.ExecutionCheck, copy run.Copy, fixed run.Commit) ([]run.CheckObservation, error) {
	if len(declared) == 0 {
		return nil, nil
	}
	a, err := copy.Inspect(ctx, fixed)
	if err != nil {
		return nil, err
	}
	if a.Error() != nil {
		return nil, a.Error()
	}
	tree := a.Model().Inputs().Raw.Census
	var observations []run.CheckObservation
	for _, d := range declared {
		profile, exists := c.profiles[d.Profile.Value]
		if !exists {
			return observations, fmt.Errorf("missing prepared check profile %s", d.Profile.Value)
		}
		req, err := profile.Request(d, tree)
		if err != nil {
			return observations, err
		}
		runner, err := NewRunnerWithRuntime(c.runtime, &copyMaterializer{source: copy, copies: map[CopyID]run.DisposableCopy{}}, c.labels)
		if err != nil {
			return observations, err
		}
		receipt, commandErr := runner.Run(ctx, req)
		o := run.CheckObservation{ID: d.ID.Value, Commit: fixed.ID, Receipt: receipt}
		if commandErr != nil {
			o.Error = commandErr.Error()
		}
		observations = append(observations, o)
		if commandErr != nil {
			if receipt.Uncertain || receipt.TimedOut {
				return observations, errors.Join(run.ErrObservationIncomplete, commandErr)
			}
			return observations, commandErr
		}
		if receipt.ID != req.ID || receipt.Tree != req.Tree.Digest() || receipt.ImageID != req.ImageID || !receipt.Started || !receipt.Exited || receipt.Uncertain || receipt.TimedOut || !receipt.CopiesClosed {
			return observations, fmt.Errorf("%w: check physical observations do not establish the fixed request", run.ErrObservationIncomplete)
		}
	}
	return observations, nil
}
