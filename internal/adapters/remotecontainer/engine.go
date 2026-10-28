// Package remotecontainer starts and removes a Run's remote container from the
// host, natively on Windows, Linux and macOS with only the Docker CLI. The
// host mounts nothing: the remote's program and plan go in as a tar stream,
// the starting commit as a bundle on the seed command's standard input, and
// the result comes out as a bundle on the export command's standard output.
package remotecontainer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

// Engine runs the Docker CLI at an absolute path; the host never searches PATH.
type Engine struct{ path string }

// NewEngine binds the Docker CLI at path.
func NewEngine(path string) (*Engine, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("the container engine %q must be an absolute path", path)
	}
	return &Engine{path: path}, nil
}

const maxEngineError = 16 << 10

// run runs one docker command; stdout may be nil when its output is not needed.
func (e *Engine) run(ctx context.Context, stdin io.Reader, stdout io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, e.path, args...)
	cmd.Stdin = stdin
	var out bytes.Buffer
	if stdout == nil {
		stdout = &out
	}
	stderr := &limited{limit: maxEngineError}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("docker %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (e *Engine) output(ctx context.Context, args ...string) (string, error) {
	var out bytes.Buffer
	err := e.run(ctx, nil, &out, args...)
	return strings.TrimSpace(out.String()), err
}

// Arch returns the engine's architecture as GOARCH names it, the one the
// remote's program must be built for.
func (e *Engine) Arch(ctx context.Context) (string, error) {
	arch, err := e.output(ctx, "version", "--format", "{{.Server.Arch}}")
	if err != nil {
		return "", err
	}
	if system, err := e.output(ctx, "version", "--format", "{{.Server.Os}}"); err != nil || system != "linux" {
		return "", errors.Join(err, fmt.Errorf("the container engine runs %q containers; a Run needs Linux", system))
	}
	return arch, nil
}

type limited struct {
	limit int
	buf   bytes.Buffer
}

func (l *limited) Write(p []byte) (int, error) {
	if room := l.limit - l.buf.Len(); room > 0 {
		l.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func (l *limited) String() string { return l.buf.String() }
