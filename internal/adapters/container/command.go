package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// CommandResult reports actual container state, separately from the engine CLI
// process. ExitCode is meaningful only when Exited is true. Output combines the
// attached stdout/stderr streams; Truncated marks a retained prefix.
type CommandResult struct {
	Started, Exited      bool
	ExitCode             int
	Output               []byte
	Truncated, Uncertain bool
	Reason, ImageID      string
}

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// RunCommand runs a literal executable/argument vector in one hardened container.
// Environment entries are literal KEY=value additions to the pinned image's
// environment. It preserves RunHelper behavior by using a separate lifecycle.
// A known nonzero container exit is a result, not a transport error. Cleanup and
// inspection use an independent bounded context after caller cancellation.
func (e *Engine) RunCommand(ctx context.Context, spec ContainerSpec, argv, env []string, maxOutputBytes int64) (result CommandResult, err error) {
	if maxOutputBytes <= 0 || maxOutputBytes > maxCommandOutput {
		return result, fmt.Errorf("command output limit must be between 1 and %d", maxCommandOutput)
	}
	for _, arg := range argv {
		if strings.ContainsRune(arg, 0) {
			return result, fmt.Errorf("command argument contains NUL")
		}
	}
	args, err := spec.runArgs(argv)
	if err != nil {
		return result, err
	}
	// Keep the same hardening, but retain the container until state readback.
	create := []string{"create"}
	imageIndex := len(args) - len(argv)
	for i := 1; i < imageIndex; i++ {
		if args[i] == "--interactive" || args[i] == "--rm" {
			continue
		}
		create = append(create, args[i])
	}
	create = append(create, args[imageIndex:]...)
	seen := map[string]bool{}
	var envArgs []string
	for _, value := range env {
		key, _, ok := strings.Cut(value, "=")
		if !ok || !envKeyPattern.MatchString(key) || strings.ContainsRune(value, 0) || seen[key] {
			return result, fmt.Errorf("invalid or duplicate command environment key %q", key)
		}
		seen[key] = true
		envArgs = append(envArgs, "--env", value)
	}
	// Options precede the image and entrypoint; argv remains literal.
	create = append(create[:1], append(envArgs, create[1:]...)...)
	out, err := e.output(ctx, create...)
	if err != nil {
		result.Uncertain, result.Reason = true, "container creation failed; ownership was not observed"
		return result, err
	}
	id := strings.TrimSpace(string(out))
	if !containerIDPattern.MatchString(id) {
		result.Uncertain, result.Reason = true, "engine did not return a full container identity"
		return result, fmt.Errorf("invalid created container ID %q", id)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if _, removeErr := e.output(cleanup, "rm", "--force", id); removeErr != nil {
			result.Uncertain = true
			result.Reason = "container cleanup failed"
			err = errors.Join(err, removeErr)
		}
	}()
	buffer := &boundedBuffer{max: int(maxOutputBytes)}
	cmd := exec.CommandContext(ctx, e.path, "start", "--attach", id)
	cmd.Stdout, cmd.Stderr = buffer, buffer
	startErr := cmd.Run()
	result.Output, result.Truncated = buffer.Bytes(), buffer.overflow
	observe, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	out, inspectErr := e.output(observe, "inspect", "--type", "container", id)
	if inspectErr != nil {
		result.Uncertain, result.Reason = true, "container state observation failed"
		return result, errors.Join(ctx.Err(), startErr, inspectErr)
	}
	var rows []struct {
		ID, Image string
		State     struct {
			Status                string
			Running               bool
			StartedAt, FinishedAt string
			ExitCode              int
			Error                 string
		}
	}
	if decodeErr := json.Unmarshal(out, &rows); decodeErr != nil {
		result.Uncertain, result.Reason = true, "invalid container state observation"
		return result, errors.Join(startErr, decodeErr)
	}
	if len(rows) != 1 || rows[0].ID != id || rows[0].Image != spec.Image {
		result.Uncertain, result.Reason = true, "container identity or image mismatch"
		return result, fmt.Errorf("container identity or image mismatch")
	}
	state := rows[0].State
	result.ImageID = rows[0].Image
	started, timeErr := time.Parse(time.RFC3339Nano, state.StartedAt)
	result.Started = timeErr == nil && !started.IsZero()
	finished, finishErr := time.Parse(time.RFC3339Nano, state.FinishedAt)
	result.Exited = result.Started && !state.Running && state.Status == "exited" && finishErr == nil && !finished.IsZero()
	if result.Exited {
		result.ExitCode = state.ExitCode
	}
	if ctx.Err() != nil {
		result.Uncertain = !result.Exited
		result.Reason = "command context ended"
		return result, ctx.Err()
	}
	if !result.Started || !result.Exited {
		result.Uncertain, result.Reason = true, "target container terminal state was not observed"
		return result, errors.Join(startErr, fmt.Errorf("target container terminal state was not observed: %s", state.Error))
	}
	// The inspected target exit is authoritative even when the CLI exits nonzero.
	return result, nil
}
