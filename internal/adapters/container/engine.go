// Package container is the Orchestrator's client of the container engine. It starts
// session containers that run markitect-agentd, carries the agentd frame
// protocol over the container's standard streams, and manages the volumes
// and images of the runtime. It knows no tasks, models or reviews.
package container

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// maxCommandOutput bounds the captured output of one engine command.
	maxCommandOutput = 4 << 20
	// maxImageFile bounds a metadata file read out of an image.
	maxImageFile = 1 << 20
)

// Engine runs the container engine's command line, such as Docker or a
// compatible engine, from one absolute executable path.
type Engine struct {
	path         string
	observations imageObservationMemo
}

// NewEngine binds the engine executable. The path must be absolute and name
// a regular file; the Orchestrator never searches PATH for it.
func NewEngine(path string) (*Engine, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("container engine %q is not an absolute path", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("container engine: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("container engine %q is not a regular file", path)
	}
	return &Engine{path: path}, nil
}

// Start runs one session container and returns its agentd connection after
// agentd's hello. Closing the session ends the container.
func (e *Engine) Start(ctx context.Context, spec ContainerSpec) (*Session, error) {
	l, err := e.prepare(spec, []string{AgentdPath, "serve"})
	if err != nil {
		return nil, err
	}
	stdin, err := l.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := l.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := l.start(); err != nil {
		return nil, err
	}
	wait := func() error {
		err := l.wait()
		l.cleanup()
		return err
	}
	return newSession(ctx, stdin, stdout, wait, l.kill)
}

// RunHelper runs one command in a fresh container under the same hardening as
// a session, for example to fill or read a workspace volume. argv[0] is an
// absolute path inside the image. stdin feeds the command and its standard
// output goes to stdout. A nonzero exit is an error with the command's
// diagnostics. Cancelling ctx removes the container.
func (e *Engine) RunHelper(ctx context.Context, spec ContainerSpec, argv []string, stdin io.Reader, stdout io.Writer) error {
	if len(argv) == 0 || !absolute(argv[0]) {
		return fmt.Errorf("helper command %q does not start with an absolute path", argv)
	}
	l, err := e.prepare(spec, argv)
	if err != nil {
		return err
	}
	l.cmd.Stdin, l.cmd.Stdout = stdin, stdout
	if err := l.start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- l.wait() }()
	defer l.cleanup() // only after kill has read the cidfile
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		l.kill()
		<-done
		return ctx.Err()
	}
}

// launch is one engine run process and the container it creates.
type launch struct {
	e       *Engine
	name    string
	cmd     *exec.Cmd
	stderr  *boundedBuffer
	cidDir  string
	cidPath string
	once    sync.Once
	waitErr error
}

// prepare validates the spec and builds the run command. The engine writes
// the created container's ID to a cidfile; forced removal uses that ID,
// never the name, so a name conflict cannot remove another container.
func (e *Engine) prepare(spec ContainerSpec, entry []string) (*launch, error) {
	args, err := spec.runArgs(entry)
	if err != nil {
		return nil, err
	}
	cidDir, err := os.MkdirTemp("", "markitect-cid")
	if err != nil {
		return nil, err
	}
	cidPath := filepath.Join(cidDir, "cid")
	args = append([]string{args[0], "--cidfile", cidPath}, args[1:]...)
	l := &launch{e: e, name: spec.Name, cmd: exec.Command(e.path, args...), stderr: &boundedBuffer{max: maxCommandOutput}, cidDir: cidDir, cidPath: cidPath}
	l.cmd.Stderr = l.stderr
	return l, nil
}

func (l *launch) start() error {
	if err := l.cmd.Start(); err != nil {
		_ = os.RemoveAll(l.cidDir)
		return fmt.Errorf("start container %s: %w", l.name, err)
	}
	return nil
}

// wait reports the end of the engine run process once.
func (l *launch) wait() error {
	l.once.Do(func() {
		if err := l.cmd.Wait(); err != nil {
			l.waitErr = fmt.Errorf("container %s: %w: %s", l.name, err, strings.TrimSpace(l.stderr.String()))
		}
	})
	return l.waitErr
}

// cleanup removes the cidfile directory once nothing needs the container ID.
func (l *launch) cleanup() { _ = os.RemoveAll(l.cidDir) }

// kill ends the engine run process and removes the container it created.
func (l *launch) kill() {
	_ = l.cmd.Process.Kill()
	raw, err := os.ReadFile(l.cidPath)
	id := bytes.TrimSpace(raw)
	if err != nil || !containerIDPattern.Match(id) {
		return // the engine created no container
	}
	rmCtx, cancel := context.WithTimeout(context.Background(), closeTimeout)
	defer cancel()
	_, _ = l.e.output(rmCtx, "rm", "--force", string(id))
}

// CreateVolume creates a labeled named volume.
func (e *Engine) CreateVolume(ctx context.Context, name string, labels map[string]string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("volume %q is not a plain engine name", name)
	}
	args, err := labelArgs(labels)
	if err != nil {
		return err
	}
	_, err = e.output(ctx, append(append([]string{"volume", "create"}, args...), name)...)
	return err
}

// RemoveVolume deletes a named volume.
func (e *Engine) RemoveVolume(ctx context.Context, name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("volume %q is not a plain engine name", name)
	}
	_, err := e.output(ctx, "volume", "rm", name)
	return err
}

// Containers lists the names of all containers, running or not, that carry
// the label key with the value.
func (e *Engine) Containers(ctx context.Context, key, value string) ([]string, error) {
	if !labelPattern.MatchString(key) || strings.ContainsAny(value, "\n\r\x00") {
		return nil, fmt.Errorf("label %q is not a plain engine label", key)
	}
	out, err := e.output(ctx, "ps", "--all", "--no-trunc", "--filter", "label="+key+"="+value, "--format", "{{.Names}}")
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}

// Volumes lists the names of all volumes that carry the label key with the
// value.
func (e *Engine) Volumes(ctx context.Context, key, value string) ([]string, error) {
	if !labelPattern.MatchString(key) || strings.ContainsAny(value, "\n\r\x00") {
		return nil, fmt.Errorf("label %q is not a plain engine label", key)
	}
	out, err := e.output(ctx, "volume", "ls", "--quiet", "--filter", "label="+key+"="+value)
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}

// ExecError is a command that ran in the container and exited nonzero. Its
// diagnostics are the command's bounded standard error.
type ExecError struct {
	Code        int
	Diagnostics string
}

func (e *ExecError) Error() string {
	return fmt.Sprintf("exit status %d: %s", e.Code, e.Diagnostics)
}

// Exec runs one literal command in a running container as the container's
// user, in directory dir, with stdin as its standard input. Its standard
// output goes to stdout. A nonzero exit is an *ExecError; any other error
// means the command may not have run. argv[0] is an absolute path inside the
// image; no shell interprets the arguments.
func (e *Engine) Exec(ctx context.Context, name, dir string, argv []string, stdin io.Reader, stdout io.Writer) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("container name %q is not a plain engine name", name)
	}
	if !absolute(dir) || len(argv) == 0 || !absolute(argv[0]) {
		return fmt.Errorf("exec in %s needs an absolute directory and command, not %q in %q", name, argv, dir)
	}
	args := append([]string{"exec", "--interactive", "--workdir", dir, name}, argv...)
	cmd := exec.CommandContext(ctx, e.path, args...)
	stderr := &boundedBuffer{max: maxCommandOutput}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	// A cancelled exec ends the engine client; its streams are not waited
	// for beyond this.
	cmd.WaitDelay = 5 * time.Second
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		return ctx.Err()
	case errors.As(err, &exit) && exit.ExitCode() > 0 && exit.ExitCode() < 125:
		// 125 to 127 are the engine's own failures to run the command.
		return &ExecError{Code: exit.ExitCode(), Diagnostics: strings.TrimSpace(stderr.String())}
	default:
		return fmt.Errorf("container engine exec in %s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
}

// RemoveContainer stops and removes a container. The Orchestrator uses it to end
// containers of its store that it no longer controls.
func (e *Engine) RemoveContainer(ctx context.Context, name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("container name %q is not a plain engine name", name)
	}
	_, err := e.output(ctx, "rm", "--force", name)
	return err
}

// Pause freezes every process in a container through the engine's cgroup
// freezer. Frozen processes cannot write, start children or answer callbacks
// until Unpause, which makes Pause the write barrier for a workspace-basis
// transition. A frozen process keeps its open files and its in-memory view;
// the next message must make it re-read what changed.
func (e *Engine) Pause(ctx context.Context, name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("container name %q is not a plain engine name", name)
	}
	_, err := e.output(ctx, "pause", name)
	return err
}

// Unpause resumes a container frozen by Pause.
func (e *Engine) Unpause(ctx context.Context, name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("container name %q is not a plain engine name", name)
	}
	_, err := e.output(ctx, "unpause", name)
	return err
}

// ImageID resolves an image reference to its engine image ID.
func (e *Engine) ImageID(ctx context.Context, ref string) (string, error) {
	if ref == "" || strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, " \n\r\x00") {
		return "", fmt.Errorf("image reference %q is not plain", ref)
	}
	out, err := e.output(ctx, "image", "inspect", "--format", "{{.Id}}", ref)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(out))
	if !imageIDPattern.MatchString(id) {
		return "", fmt.Errorf("engine reported image ID %q", id)
	}
	return id, nil
}

// RunningImage observes the immutable image of an actual running container.
// name must be the fresh container name retained by its owning native caller.
func (e *Engine) RunningImage(ctx context.Context, name string) (string, error) {
	if !namePattern.MatchString(name) {
		return "", fmt.Errorf("invalid owned container name")
	}
	out, err := e.output(ctx, "inspect", "--type", "container", name)
	if err != nil {
		return "", err
	}
	var rows []struct {
		ID, Name, Image string
		State           struct{ Running *bool }
	}
	if json.Unmarshal(out, &rows) != nil || len(rows) != 1 || !containerIDPattern.MatchString(rows[0].ID) || rows[0].Name != "/"+name || !imageIDPattern.MatchString(rows[0].Image) || rows[0].State.Running == nil || !*rows[0].State.Running {
		return "", fmt.Errorf("running container image was not observed")
	}
	return rows[0].Image, nil
}

// minSubpathServerMajor is the first Docker server version that mounts a
// directory of a named volume (volume-subpath).
const minSubpathServerMajor = 26

// ServerVersion asks the engine for its server version.
func (e *Engine) ServerVersion(ctx context.Context) (string, error) {
	out, err := e.output(ctx, "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return "", fmt.Errorf("the container engine's server version could not be read: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RequireVolumeSubpaths refuses an engine whose server cannot mount a directory
// of a volume (Mount.Subpath): Docker 26 or newer. It asks the engine for its
// server version and refuses an unreadable answer.
func (e *Engine) RequireVolumeSubpaths(ctx context.Context) error {
	version, err := e.ServerVersion(ctx)
	if err != nil {
		return err
	}
	return RequireSubpathVersion(version)
}

// RequireSubpathVersion refuses a server version that cannot mount a directory
// of a volume. It is RequireVolumeSubpaths for a caller that has read the
// version already.
func RequireSubpathVersion(version string) error {
	digits, _, _ := strings.Cut(version, ".")
	major, err := strconv.Atoi(digits)
	if err != nil || major < 0 {
		return fmt.Errorf("the container engine reports the server version %q, which is not understood; volume subpath mounts need Docker %d or newer", version, minSubpathServerMajor)
	}
	if major < minSubpathServerMajor {
		return fmt.Errorf("the container engine is Docker server %s; volume subpath mounts, which enforced ownership needs, require Docker %d or newer", version, minSubpathServerMajor)
	}
	return nil
}

// ReadImageFile copies one regular file out of an image without running it:
// it creates a container that never starts, copies the file and removes the
// container again.
func (e *Engine) ReadImageFile(ctx context.Context, imageID, file string) ([]byte, error) {
	if !imageIDPattern.MatchString(imageID) {
		return nil, fmt.Errorf("image %q is not an engine image ID", imageID)
	}
	if !absolute(file) {
		return nil, fmt.Errorf("image file %q is not an absolute path", file)
	}
	out, err := e.output(ctx, "create", "--entrypoint", "/markitect-never-started", imageID)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(out))
	if !namePattern.MatchString(id) {
		return nil, fmt.Errorf("engine reported container ID %q", id)
	}
	defer func() { _, _ = e.output(context.WithoutCancel(ctx), "rm", "--force", id) }()
	archive, err := e.output(ctx, "cp", id+":"+file, "-")
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(bytes.NewReader(archive))
	h, err := tr.Next()
	if err != nil {
		return nil, fmt.Errorf("read %s from image: %w", file, err)
	}
	if h.Typeflag != tar.TypeReg || h.Size > maxImageFile {
		return nil, fmt.Errorf("image file %s is not a regular file of at most %d bytes", file, maxImageFile)
	}
	return io.ReadAll(io.LimitReader(tr, maxImageFile))
}

// output runs one bounded engine command and returns its standard output.
func (e *Engine) output(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.path, args...)
	stdout := &boundedBuffer{max: maxCommandOutput}
	stderr := &boundedBuffer{max: maxCommandOutput}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("container engine %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	if stdout.overflow {
		return nil, fmt.Errorf("container engine %s: output exceeds %d bytes", args[0], maxCommandOutput)
	}
	return stdout.Bytes(), nil
}

// boundedBuffer keeps at most max bytes and records any overflow.
type boundedBuffer struct {
	mu       sync.Mutex
	buf      bytes.Buffer
	max      int
	overflow bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	room := b.max - b.buf.Len()
	if len(p) > room {
		b.overflow = true
		if room > 0 {
			b.buf.Write(p[:room])
		}
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *boundedBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

func (b *boundedBuffer) String() string { return string(b.Bytes()) }
