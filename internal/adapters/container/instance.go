package container

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

const (
	// reapBound bounds one reap exec: agentd's own wait plus the exec.
	reapBound = agentdReapWait + 25*time.Second
	// agentdReapWait is how long agentd waits for the processes it killed.
	agentdReapWait = 5 * time.Second
	// removeBound bounds the forced removal of an instance.
	removeBound = time.Minute
)

// Instance is one reusable session container. Its init process holds it
// between agentd sessions, each of which joins it through exec as the spec's
// user and sees the spec's mounts; nothing runs in it between sessions once a
// reap observed every other process ended. Removing it ends it.
type Instance struct {
	e    *Engine
	id   string
	spec ContainerSpec
}

// Hold starts a reusable instance of the spec whose init ends it after idle
// without any other process, so an instance its Orchestrator lost ends by
// itself.
func (e *Engine) Hold(ctx context.Context, spec ContainerSpec, idle time.Duration) (*Instance, error) {
	if idle <= 0 {
		return nil, errors.New("a held container needs a positive idle bound")
	}
	args, err := spec.args(true, []string{AgentdPath, "hold", "--idle", idle.String()})
	if err != nil {
		return nil, err
	}
	out, err := e.output(ctx, args...)
	if err != nil {
		// A container the engine created before failing ends here.
		rmCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), removeBound)
		defer cancel()
		_, _ = e.output(rmCtx, "rm", "--force", spec.Name)
		return nil, fmt.Errorf("hold container %s: %w", spec.Name, err)
	}
	id := strings.TrimSpace(string(out))
	if !containerIDPattern.MatchString(id) {
		return nil, fmt.Errorf("engine reported container ID %q", id)
	}
	return &Instance{e: e, id: id, spec: spec}, nil
}

// Name is the instance's container name.
func (i *Instance) Name() string { return i.spec.Name }

// Exec starts one agentd session in the instance and returns its connection
// after agentd's hello. Closing the session ends agentd's exec, not the
// instance; a session that does not end leaves processes a reap ends.
func (i *Instance) Exec(ctx context.Context) (*Session, error) {
	cmd := exec.Command(i.e.path, "exec", "--interactive", "--user", i.spec.User, "--workdir", i.spec.Workdir, i.id, AgentdPath, "serve")
	stderr := &boundedBuffer{max: maxCommandOutput}
	cmd.Stderr = stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("exec in container %s: %w", i.spec.Name, err)
	}
	wait := func() error {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("exec in container %s: %w: %s", i.spec.Name, err, strings.TrimSpace(stderr.String()))
		}
		return nil
	}
	return newSession(ctx, stdin, stdout, wait, func() { _ = cmd.Process.Kill() })
}

// Reap ends every Agent process of the instance and returns those still alive
// after agentd's bound; an empty result is the observed cessation of every
// writer in it. An error means cessation was not observed.
func (i *Instance) Reap(ctx context.Context) ([]frame.Process, error) {
	ctx, cancel := context.WithTimeout(ctx, reapBound)
	defer cancel()
	cmd := exec.CommandContext(ctx, i.e.path, "exec", "--user", i.spec.User, i.id, AgentdPath, "reap", "--timeout", agentdReapWait.String())
	stdout, stderr := &boundedBuffer{max: maxCommandOutput}, &boundedBuffer{max: maxCommandOutput}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	runErr := cmd.Run()
	var report struct {
		Remaining []frame.Process `json:"remaining"`
	}
	if err := json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&report); err != nil {
		return nil, fmt.Errorf("reap in container %s: %w: %s", i.spec.Name, errors.Join(runErr, err), strings.TrimSpace(stderr.String()))
	}
	if runErr != nil && len(report.Remaining) == 0 {
		return nil, fmt.Errorf("reap in container %s: %w: %s", i.spec.Name, runErr, strings.TrimSpace(stderr.String()))
	}
	return report.Remaining, nil
}

// Remove ends the instance by force and confirms that the engine no longer
// knows it.
func (i *Instance) Remove(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), removeBound)
	defer cancel()
	// An instance that ended by itself (its idle bound, an engine restart) is
	// gone already; the inspection below confirms that.
	if _, err := i.e.output(ctx, "rm", "--force", i.id); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no such") {
		return err
	}
	// Only the engine's "no such" answer confirms the removal; any other
	// inspection failure leaves it unobserved.
	switch _, err := i.e.output(ctx, "container", "inspect", "--format", "{{.Id}}", i.id); {
	case err == nil:
		return fmt.Errorf("container %s still exists after its removal", i.spec.Name)
	case !strings.Contains(strings.ToLower(err.Error()), "no such"):
		return fmt.Errorf("container %s: its removal is not confirmed: %w", i.spec.Name, err)
	}
	return nil
}
