package container

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
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
	out, err := e.output(ctx, "create", "--entrypoint", "/markitect-never-started", imageID)
	if err != nil {
		return observation, err
	}
	id := strings.TrimSpace(string(out))
	if !containerIDPattern.MatchString(id) {
		return observation, fmt.Errorf("engine reported invalid full container ID %q", id)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if _, removeErr := e.output(cleanup, "rm", "--force", id); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("image observation cleanup: %w", removeErr))
		}
		err = errors.Join(err, ctx.Err())
		if err != nil {
			observation = ImageFileObservation{}
		}
	}()
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

// observeImageArchive permits one file and a complete tar terminator. A small
// fixed allowance bounds headers, PAX metadata and zero padding independently of
// the file bound. No unbounded trailer is drained from a faulty engine.
func observeImageArchive(source io.Reader, name string, maxBytes int64) (ImageFileObservation, error) {
	const archiveAllowance = 1 << 20
	limited := &io.LimitedReader{R: source, N: maxBytes + archiveAllowance + 1}
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
	hash := sha256.New()
	n, err := io.CopyN(hash, tr, h.Size)
	if err != nil {
		return ImageFileObservation{}, fmt.Errorf("read complete image file: %w", err)
	}
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
