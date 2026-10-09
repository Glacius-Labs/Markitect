package projectworkspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Git's similarity detection retains rename provenance for ordinary edits and
// mode changes. Completely rewritten moves remain delete/add, as Git cannot
// establish a reliable rename from unrelated bytes alone.
func detectRenames(ctx context.Context, storage string, before, after inventory, deleted, added []string) (map[string]string, error) {
	result := map[string]string{}
	if len(deleted) == 0 || len(added) == 0 {
		return result, nil
	}
	// Scratch has a separate finite ceiling because deleted content has zero
	// delta bytes but still costs disk space during similarity detection.
	var scratchBytes int64
	for _, p := range deleted {
		scratchBytes += int64(len(before[p].Content))
	}
	for _, p := range added {
		scratchBytes += int64(len(after[p].Content))
	}
	if scratchBytes > 256<<20 {
		return nil, fmt.Errorf("%w: rename scratch inventory exceeds 256 MiB", ErrInvalidDelta)
	}
	dir, err := os.MkdirTemp(storage, "rename-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	oldRoot, newRoot := filepath.Join(dir, "before"), filepath.Join(dir, "after")
	for _, root := range []string{oldRoot, newRoot} {
		if err := os.MkdirAll(root, 0700); err != nil {
			return nil, err
		}
	}
	for i, paths := range [][]string{deleted, added} {
		root, inv := oldRoot, before
		if i == 1 {
			root, inv = newRoot, after
		}
		for _, p := range paths {
			dest := filepath.Join(root, filepath.FromSlash(p))
			if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
				return nil, err
			}
			mode := fs.FileMode(0644)
			if inv[p].Mode == "100755" {
				mode = 0755
			}
			if err := os.WriteFile(dest, inv[p].Content, mode); err != nil {
				return nil, err
			}
		}
	}
	cmd := gitCommand(ctx, storage, "diff", "--no-index", "--no-ext-diff", "--no-textconv", "--find-renames=50%", "--name-status", "-z", "--", oldRoot, newRoot)
	output, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return nil, fmt.Errorf("rename detection: %w", err)
		}
	}
	fields := bytes.Split(output, []byte{0})
	for i := 0; i < len(fields) && len(fields[i]) > 0; {
		status := string(fields[i])
		i++
		if i >= len(fields) {
			return nil, ErrInvalidDelta
		}
		first := string(fields[i])
		i++
		if strings.HasPrefix(status, "R") {
			if i >= len(fields) {
				return nil, ErrInvalidDelta
			}
			second := string(fields[i])
			i++
			old, err := filepath.Rel(oldRoot, filepath.FromSlash(first))
			if err != nil {
				return nil, err
			}
			next, err := filepath.Rel(newRoot, filepath.FromSlash(second))
			if err != nil {
				return nil, err
			}
			old, next = filepath.ToSlash(old), filepath.ToSlash(next)
			if _, ok := before[old]; !ok {
				return nil, ErrInvalidDelta
			}
			if _, ok := after[next]; !ok {
				return nil, ErrInvalidDelta
			}
			result[old] = next
		}
	}
	return result, nil
}
