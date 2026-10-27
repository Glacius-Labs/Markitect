package container

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// maxArchiveNames bounds the entries one directory listing returns.
const maxArchiveNames = 100000

// ImageEnvironment returns the environment entries the image configures for
// every container, read from the image's configuration without starting it.
func (e *Engine) ImageEnvironment(ctx context.Context, imageID string) ([]string, error) {
	if !imageIDPattern.MatchString(imageID) {
		return nil, fmt.Errorf("image %q is not an engine image ID", imageID)
	}
	out, err := e.output(ctx, "image", "inspect", "--format", "{{json .Config.Env}}", imageID)
	if err != nil {
		return nil, err
	}
	return parseImageEnvironment(out)
}

func parseImageEnvironment(out []byte) ([]string, error) {
	var env []string
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("engine reported an unreadable image environment: %w", err)
	}
	for _, entry := range env {
		if !strings.Contains(entry, "=") {
			return nil, fmt.Errorf("engine reported an image environment entry without '='")
		}
	}
	return env, nil
}

// ImageDirectoryNames lists the entry paths of one directory tree of the
// image, as the engine's archive names them (for /etc: "etc", "etc/hosts",
// ...). It reads a created, never started container and streams the archive
// through a header-only reader, so file contents are skipped, bounded by
// maxBytes. An absent directory is an error, never an empty listing.
func (e *Engine) ImageDirectoryNames(ctx context.Context, imageID, dir string, maxBytes int64) ([]string, error) {
	if !imageIDPattern.MatchString(imageID) {
		return nil, fmt.Errorf("image %q is not an engine image ID", imageID)
	}
	if !absolute(dir) || maxBytes <= 0 {
		return nil, fmt.Errorf("image directory %q is not an absolute path with a positive bound", dir)
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
	listing, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(listing, e.path, "cp", id+":"+dir, "-")
	stderr := &boundedBuffer{max: maxCommandOutput}
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	names, listErr := archiveNames(stdout, maxBytes)
	if listErr != nil {
		cancel() // stop the copy instead of draining an oversized archive
	}
	waitErr := cmd.Wait()
	if listErr != nil {
		return nil, listErr
	}
	if waitErr != nil {
		return nil, fmt.Errorf("container engine cp: %w: %s", waitErr, strings.TrimSpace(stderr.String()))
	}
	return names, nil
}

var errArchiveLimit = errors.New("image directory archive exceeds its bound")

// archiveNames reads only the headers of a tar stream of at most maxBytes.
func archiveNames(r io.Reader, maxBytes int64) ([]string, error) {
	limited := &io.LimitedReader{R: r, N: maxBytes + 1}
	tr := tar.NewReader(limited)
	var names []string
	for {
		h, err := tr.Next()
		if limited.N <= 0 {
			return nil, errArchiveLimit
		}
		if errors.Is(err, io.EOF) {
			return names, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read image directory archive: %w", err)
		}
		if len(names) == maxArchiveNames {
			return nil, errArchiveLimit
		}
		names = append(names, strings.TrimSuffix(h.Name, "/"))
	}
}
