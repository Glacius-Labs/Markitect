package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const (
	// holdIdle ends a kept instance its Orchestrator lost: the instance ends
	// after this long without any process, which a waiting Manager can reach;
	// its next turn then starts a new one.
	holdIdle = 45 * time.Minute
	// maxViews bounds the instances an environment keeps: its read-only view
	// and its current write view. A further capability view replaces the
	// least recently used one.
	maxViews = 2
)

// Runtime starts the reusable instances turns run in. Production peers must
// preserve container.Engine.Hold's semantics.
type Runtime interface {
	Hold(context.Context, container.ContainerSpec) (Instance, error)
}

// Instance is one reusable container: each turn joins it with a fresh agentd
// session, after which Reap must observe that no process of the turn is left.
type Instance interface {
	Name() string
	Exec(context.Context) (RuntimeSession, error)
	Reap(context.Context) ([]frame.Process, error)
	Remove(context.Context) error
}

// engineRuntime holds instances on the container engine.
type engineRuntime struct{ engine *container.Engine }

func (r engineRuntime) Hold(ctx context.Context, spec container.ContainerSpec) (Instance, error) {
	inst, err := r.engine.Hold(ctx, spec, holdIdle)
	if err != nil {
		return nil, err
	}
	return engineInstance{inst}, nil
}

type engineInstance struct{ *container.Instance }

func (i engineInstance) Exec(ctx context.Context) (RuntimeSession, error) {
	s, err := i.Instance.Exec(ctx)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// environment is what the adapter keeps of one Turn.Environment: its
// instances by capability view, least recently used first, the generation of
// its newest instance and why its next instance replaces an earlier one.
type environment struct {
	name       string
	views      []*view
	generation int
	replaced   string
	busy       bool
}

// view is one kept instance with the capabilities it was created with.
type view struct {
	inst       Instance
	key        string
	generation int
}

// capabilities identifies what an instance lets a turn do: everything of its
// spec but its name.
func capabilities(spec container.ContainerSpec) string {
	spec.Name = ""
	b, _ := json.Marshal(spec)
	return string(b)
}

// lease is the instance a turn runs in and what to do with it afterwards.
type lease struct {
	inst Instance
	obs  run.InstanceObservation
	env  *environment
	view *view
}

// acquire returns the instance of the turn: a fresh one for a turn without an
// environment, otherwise the environment's instance with exactly the turn's
// capabilities or a new one, started outside the adapter's lock.
func (a *Agents) acquire(ctx context.Context, turn run.Turn, spec container.ContainerSpec) (*lease, error) {
	if turn.Environment == "" {
		return a.start(ctx, spec, nil, "fresh", 1)
	}
	key := capabilities(spec)
	a.mu.Lock()
	env := a.envs[turn.Environment]
	if env == nil {
		env = &environment{name: turn.Environment}
		a.envs[turn.Environment] = env
	}
	if env.busy {
		a.mu.Unlock()
		return nil, fmt.Errorf("%w: environment %s is in use by another turn", ErrSession, turn.Environment)
	}
	env.busy = true
	if i := slices.IndexFunc(env.views, func(v *view) bool { return v.key == key }); i >= 0 {
		v := env.views[i]
		env.views = append(slices.Delete(env.views, i, i+1), v)
		a.mu.Unlock()
		return &lease{inst: v.inst, obs: run.InstanceObservation{Name: v.inst.Name(), Generation: v.generation, Reused: true}, env: env, view: v}, nil
	}
	why := "first"
	switch {
	case env.replaced != "":
		why, env.replaced = env.replaced, ""
	case env.generation > 0:
		why = "capabilities"
	}
	env.generation++
	generation := env.generation
	var evicted *view
	if len(env.views) >= maxViews {
		evicted, env.views = env.views[0], env.views[1:]
	}
	a.mu.Unlock()
	if evicted != nil {
		// Nothing runs in a kept instance between turns; its end is cleanup.
		_ = evicted.inst.Remove(ctx)
	}
	spec.Name = "markitect-" + turn.Environment + "-g" + strconv.Itoa(generation)
	l, err := a.start(ctx, spec, env, why, generation)
	if err != nil {
		a.mu.Lock()
		env.busy = false
		a.mu.Unlock()
		return nil, err
	}
	l.view = &view{inst: l.inst, key: key, generation: generation}
	a.mu.Lock()
	env.views = append(env.views, l.view)
	a.mu.Unlock()
	return l, nil
}

func (a *Agents) start(ctx context.Context, spec container.ContainerSpec, env *environment, why string, generation int) (*lease, error) {
	began := time.Now()
	inst, err := a.runtime.Hold(ctx, spec)
	if err == nil && inst == nil {
		err = ErrSession
	}
	if err != nil {
		return nil, err
	}
	return &lease{inst: inst, env: env, obs: run.InstanceObservation{Name: inst.Name(), Generation: generation, Started: why, StartMillis: time.Since(began).Milliseconds()}}, nil
}

// settle ends the turn's use of its instance. A quiet instance stays with its
// environment, a fresh one ends; an instance whose writers did not stop is
// removed and its environment's next instance replaces it. It reports whether
// every process of the turn was observed to end, and what happened.
func (a *Agents) settle(ctx context.Context, l *lease) (bool, string) {
	remaining, err := l.inst.Reap(ctx)
	quiet := err == nil && len(remaining) == 0
	note := "every process of the turn ended"
	if !quiet {
		note = fmt.Sprintf("processes of the turn did not end in place (%d left, %v); the instance was removed", len(remaining), err)
	}
	keep := quiet && l.env != nil
	if !keep {
		if rmErr := l.inst.Remove(ctx); rmErr != nil {
			if !quiet {
				note = fmt.Sprintf("processes of the turn did not end in place (%d left, %v) and the instance's removal was not observed: %v", len(remaining), err, rmErr)
				a.dropLocked(l, "")
				return false, note
			}
			note += "; the fresh instance's removal was not observed: " + rmErr.Error()
		}
	}
	if l.env != nil {
		if keep {
			a.release(l)
		} else {
			a.dropLocked(l, "the writers of its predecessor did not stop in place")
		}
	}
	return true, note
}

// release frees an environment for its next turn.
func (a *Agents) release(l *lease) {
	a.mu.Lock()
	defer a.mu.Unlock()
	l.env.busy = false
}

// dropLocked forgets a removed instance of an environment; its next instance
// says why it replaces it.
func (a *Agents) dropLocked(l *lease, why string) {
	if l.env == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	l.env.views = slices.DeleteFunc(l.env.views, func(v *view) bool { return v == l.view })
	if why != "" {
		l.env.replaced = why
	}
	l.env.busy = false
}

// Release implements run.Environments: it removes every kept instance of the
// named environments, or of all when names is nil.
func (a *Agents) Release(ctx context.Context, names []string) []run.EnvironmentRelease {
	a.mu.Lock()
	var envs []*environment
	for name, env := range a.envs {
		if names == nil || slices.Contains(names, name) {
			envs = append(envs, env)
			delete(a.envs, name)
		}
	}
	a.mu.Unlock()
	slices.SortFunc(envs, func(x, y *environment) int {
		switch {
		case x.name < y.name:
			return -1
		case x.name > y.name:
			return 1
		}
		return 0
	})
	var out []run.EnvironmentRelease
	for _, env := range envs {
		for _, v := range env.views {
			r := run.EnvironmentRelease{Environment: env.name, Instance: v.inst.Name(), Removed: true}
			if err := v.inst.Remove(context.WithoutCancel(ctx)); err != nil {
				r.Removed, r.Error = false, err.Error()
			}
			out = append(out, r)
		}
	}
	return out
}

var _ run.Environments = (*Agents)(nil)
