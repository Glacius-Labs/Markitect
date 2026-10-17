// Package container is the Host's client of the container engine. It starts
// session containers that run markitect-agentd, carries the agentd frame
// protocol over the container's standard streams, and manages the volumes
// and images of the runtime. It knows no tasks, models or reviews.
package container

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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
	path string
}

// NewEngine binds the engine executable. The path must be absolute and name
// a regular file; the Host never searches PATH for it.
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
func (e *Engine) Start(ctx context.Context, spec SessionSpec) (*Session, error) {
	args, err := spec.runArgs()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(e.path, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr := &boundedBuffer{max: maxCommandOutput}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start container %s: %w", spec.Name, err)
	}
	var once sync.Once
	var waitErr error
	wait := func() error {
		once.Do(func() {
			if err := cmd.Wait(); err != nil {
				waitErr = fmt.Errorf("container %s: %w: %s", spec.Name, err, strings.TrimSpace(stderr.String()))
			}
		})
		return waitErr
	}
	return newSession(ctx, stdin, stdout, wait)
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

// RemoveContainer stops and removes a container. The Host uses it to end
// containers of its store that it no longer controls.
func (e *Engine) RemoveContainer(ctx context.Context, name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("container name %q is not a plain engine name", name)
	}
	_, err := e.output(ctx, "rm", "--force", name)
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
