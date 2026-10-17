package container

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// MaxImageFileObservationBytes bounds a complete image-file observation.
const MaxImageFileObservationBytes int64 = 512 << 20

// ImageFileObservation identifies every byte of one regular file in an image.
// An error always returns the zero value, including when cleanup fails.
type ImageFileObservation struct {
	SHA256 [32]byte
	Size   int64
}

// ObserveImageFile hashes a complete regular file from an immutable image without
// starting its temporary container. maxBytes must be positive and at most
// MaxImageFileObservationBytes. File bytes are streamed, never retained in memory.
// ReadImageFile remains the separate, small metadata reader.
func (e *Engine) ObserveImageFile(ctx context.Context, imageID, file string, maxBytes int64) (observation ImageFileObservation, err error) {
	if !imageIDPattern.MatchString(imageID) {
		return observation, fmt.Errorf("image %q is not an engine image ID", imageID)
	}
	if !absolute(file) || file == "/" || strings.ContainsAny(file, "\\\x00\n\r") {
		return observation, fmt.Errorf("image file %q is not a clean absolute POSIX file path", file)
	}
	if maxBytes <= 0 || maxBytes > MaxImageFileObservationBytes {
		return observation, fmt.Errorf("image file limit must be between 1 and %d", MaxImageFileObservationBytes)
	}
	if err := ctx.Err(); err != nil {
		return observation, err
	}
	cidDir, err := os.MkdirTemp("", "markitect-image-observation-")
	if err != nil {
		return observation, err
	}
	cidPath := filepath.Join(cidDir, "cid")
	defer func() {
		// Only the private engine-written handle supplies deletion authority,
		// including when creation failed after the daemon created its container.
		raw, readErr := os.ReadFile(cidPath)
		if readErr == nil {
			id := strings.TrimSpace(string(raw))
			if !containerIDPattern.MatchString(id) {
				err = errors.Join(err, fmt.Errorf("invalid image observation cidfile"))
			} else {
				cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
				_, removeErr := e.imageObservationOutput(cleanup, "rm", "--force", "--volumes", id)
				cancel()
				if removeErr != nil {
					err = errors.Join(err, fmt.Errorf("image observation cleanup: %w", removeErr))
				}
			}
		} else if !errors.Is(readErr, os.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("read image observation cidfile: %w", readErr))
		}
		err = errors.Join(err, os.RemoveAll(cidDir))
		err = errors.Join(err, ctx.Err())
		if err != nil {
			observation = ImageFileObservation{}
		}
	}()
	out, err := e.imageObservationOutput(ctx, "create", "--cidfile", cidPath, "--entrypoint", "/markitect-never-started", imageID)
	if err != nil {
		return observation, err
	}
	raw, err := os.ReadFile(cidPath)
	if err != nil {
		return observation, fmt.Errorf("read created image container identity: %w", err)
	}
	id := strings.TrimSpace(string(raw))
	if !containerIDPattern.MatchString(id) || id != strings.TrimSpace(string(out)) {
		return observation, fmt.Errorf("engine reported inconsistent image observation container identity")
	}
	cmd := exec.CommandContext(ctx, e.path, "cp", id+":"+file, "-")
	stderr := &boundedBuffer{max: maxCommandOutput}
	cmd.Stderr = stderr
	// Bound inherited stderr handles after the engine process has exited.
	cmd.WaitDelay = 15 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return observation, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		return observation, err
	}
	stopRead := context.AfterFunc(ctx, func() { _ = stdout.Close() })
	defer stopRead()
	observation, err = observeImageArchive(stdout, path.Base(file), maxBytes)
	if err != nil {
		// Stop a producer that could otherwise block writing an invalid archive.
		_ = stdout.Close()
		_ = cmd.Process.Kill()
	}
	if waitErr := cmd.Wait(); waitErr != nil {
		err = errors.Join(err, fmt.Errorf("container engine cp: %w: %s", waitErr, strings.TrimSpace(stderr.String())))
	}
	return observation, err
}

// Unlike metadata output, observation lifecycle commands also bound inherited
// pipe handles after process exit or cancellation. This includes cleanup.
func (e *Engine) imageObservationOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.path, args...)
	stdout, stderr := &boundedBuffer{max: maxCommandOutput}, &boundedBuffer{max: maxCommandOutput}
	cmd.Stdout, cmd.Stderr, cmd.WaitDelay = stdout, stderr, 15*time.Second
	if err := cmd.Run(); err != nil {
		return nil, errors.Join(ctx.Err(), fmt.Errorf("container engine %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String())))
	}
	if stdout.overflow {
		return nil, fmt.Errorf("container engine %s: output exceeds %d bytes", args[0], maxCommandOutput)
	}
	return stdout.Bytes(), ctx.Err()
}

// observeImageArchive permits one file and a complete tar terminator. A small
// fixed allowance bounds headers, PAX metadata and zero padding independently of
// the file bound. No unbounded trailer is drained from a faulty engine.
func observeImageArchive(source io.Reader, name string, maxBytes int64) (ImageFileObservation, error) {
	const archiveAllowance = 1 << 20
	// Next can itself read/allocate sparse metadata; it must never receive the
	// file-size budget until the regular header has been validated.
	limited := &io.LimitedReader{R: source, N: archiveAllowance}
	tr := tar.NewReader(limited)
	h, err := tr.Next()
	if err != nil {
		return ImageFileObservation{}, fmt.Errorf("read image file header: %w", err)
	}
	if h.Typeflag != tar.TypeReg || h.Name != name || h.Linkname != "" || h.Size < 0 || h.Size > maxBytes {
		return ImageFileObservation{}, fmt.Errorf("image archive needs exactly regular file %q of at most %d bytes", name, maxBytes)
	}
	for key := range h.PAXRecords {
		if strings.HasPrefix(key, "GNU.sparse.") {
			return ImageFileObservation{}, fmt.Errorf("sparse image file is not supported")
		}
	}
	limited.N += maxBytes + 1
	hash := sha256.New()
	n, err := io.CopyN(hash, tr, h.Size)
	if err != nil {
		return ImageFileObservation{}, fmt.Errorf("read complete image file: %w", err)
	}
	// Revoke unused payload budget before another Next can parse metadata.
	limited.N = min(limited.N, int64(archiveAllowance))
	beforeEnd := limited.N
	if _, err := tr.Next(); err != io.EOF {
		return ImageFileObservation{}, fmt.Errorf("image archive has an additional entry or invalid ending: %v", err)
	}
	// archive/tar also accepts missing or single-block terminators. Require both
	// zero blocks, with only the current file's block padding before them.
	if beforeEnd-limited.N != (512-h.Size%512)%512+1024 {
		return ImageFileObservation{}, fmt.Errorf("image archive has an incomplete or unexpected terminator")
	}
	var block [512]byte
	for {
		n, readErr := io.ReadFull(limited, block[:])
		if limited.N == 0 {
			return ImageFileObservation{}, fmt.Errorf("image archive exceeds its bounded allowance")
		}
		for _, b := range block[:n] {
			if b != 0 {
				return ImageFileObservation{}, fmt.Errorf("image archive has trailing nonzero data")
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return ImageFileObservation{}, fmt.Errorf("read image archive ending: %w", readErr)
		}
	}
	var observation ImageFileObservation
	copy(observation.SHA256[:], hash.Sum(nil))
	observation.Size = n
	return observation, nil
}
