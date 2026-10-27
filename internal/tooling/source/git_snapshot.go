package source

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/guardedwrite"
)

type treeFile struct {
	path string
	mode string
	oid  string
	size int64
}

func resolveCommit(root, revision string) (string, error) {
	// --end-of-options prevents a revision beginning with '-' from becoming a
	// Git option. The commit peel rejects trees, tags that do not resolve, and
	// arbitrary rev expressions that are not commits.
	out, err := git(root, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve Git revision %q: %w", revision, err)
	}
	commit := strings.TrimSpace(string(out))
	if len(commit) != 40 && len(commit) != 64 {
		return "", fmt.Errorf("Git returned a non-full commit id for %q", revision)
	}
	if _, err := hex.DecodeString(commit); err != nil {
		return "", fmt.Errorf("Git returned an invalid commit id: %w", err)
	}
	return strings.ToLower(commit), nil
}

func loadCommit(root, commit string, s *Input, limits Limits) error {
	// NUL-delimited output preserves all Git path bytes except NUL, which is
	// forbidden by Git itself. --long supplies blob object IDs for cat-file.
	out, err := git(root, "ls-tree", "-r", "-z", "--full-tree", "--long", commit)
	if err != nil {
		return fmt.Errorf("list Git tree %s: %w", commit, err)
	}
	files := make([]treeFile, 0)
	var total int64
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := bytes.Cut(record, []byte{'\t'})
		if !ok {
			return errors.New("malformed git ls-tree record")
		}
		fields := strings.Fields(string(header))
		if len(fields) != 4 {
			return errors.New("malformed git ls-tree header")
		}
		mode, kind, oid := fields[0], fields[1], fields[2]
		p := string(name)
		gitlink := kind == "commit" || mode == "160000"
		// A gitlink is a directory in the working tree, so it is skipped
		// wherever the working-tree reader would skip that directory.
		if guardedwrite.ExcludedFile(p) || (gitlink && guardedwrite.ExcludedDirectory(p)) {
			continue
		}
		if err := guardedwrite.ValidateRepoPath(p); err != nil {
			return err
		}
		if gitlink {
			return fmt.Errorf("Git submodule is not a source file: %q", p)
		}
		if mode == "120000" {
			return fmt.Errorf("Git symlink is not a source file: %q", p)
		}
		if kind != "blob" || (mode != RegularMode && mode != ExecutableMode) {
			return fmt.Errorf("unsupported Git tree entry %q (mode %s, type %s)", p, mode, kind)
		}
		// Only blobs carry a size; gitlinks report "-".
		blobSize, parseErr := strconv.ParseInt(fields[3], 10, 64)
		if parseErr != nil || blobSize < 0 {
			return fmt.Errorf("invalid Git blob size for %q", p)
		}
		if blobSize > limits.MaxFileBytes {
			return fmt.Errorf("source file %q exceeds per-file limit", p)
		}
		if total > limits.MaxTotalBytes-blobSize {
			return fmt.Errorf("source snapshot exceeds byte limit at %q", p)
		}
		if len(files) >= limits.MaxFiles {
			return errors.New("source snapshot exceeds file-count limit")
		}
		total += blobSize
		files = append(files, treeFile{path: p, mode: mode, oid: oid, size: blobSize})
	}
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.path
	}
	if err := guardedwrite.ValidatePortablePaths(paths); err != nil {
		return err
	}
	return loadBlobs(root, files, s)
}

func loadBlobs(root string, files []treeFile, s *Input) error {
	return loadBlobsWithCommand(root, files, s, guardedwrite.GitCommand)
}

func loadBlobsWithCommand(root string, files []treeFile, s *Input, command func(root string, args ...string) *exec.Cmd) error {
	if len(files) == 0 {
		return nil
	}
	var input bytes.Buffer
	for _, file := range files {
		input.WriteString(file.oid)
		input.WriteByte('\n')
	}
	cmd := command(root, "cat-file", "--batch")
	cmd.Stdin = &input
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open Git blob stream: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Git blob stream: %w", err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	r := bufio.NewReaderSize(stdout, 512)
	for _, file := range files {
		header, err := readBatchHeader(r)
		if err != nil {
			return fmt.Errorf("read Git blob header for %q: %w", file.path, err)
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[0] != file.oid || fields[1] != "blob" {
			return fmt.Errorf("unexpected Git blob header for %q: %q", file.path, header)
		}
		size, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || size != file.size {
			return fmt.Errorf("Git blob size changed while loading %q", file.path)
		}
		data := make([]byte, int(file.size))
		if _, err := io.ReadFull(r, data); err != nil {
			return fmt.Errorf("read Git blob for %q: %w", file.path, err)
		}
		separator, err := r.ReadByte()
		if err != nil || separator != '\n' {
			return fmt.Errorf("malformed Git blob separator for %q", file.path)
		}
		s.Files[file.path] = data
		s.Modes[file.path] = file.mode
	}
	if _, err := r.ReadByte(); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing data in Git blob stream")
		}
		return fmt.Errorf("finish Git blob stream: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		waited = true
		return fmt.Errorf("Git blob stream failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	waited = true
	return nil
}

func readBatchHeader(r *bufio.Reader) (string, error) {
	const maxHeader = 256
	var header strings.Builder
	for header.Len() <= maxHeader {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '\n' {
			return header.String(), nil
		}
		header.WriteByte(b)
	}
	return "", errors.New("Git blob header exceeds limit")
}

func git(root string, args ...string) ([]byte, error) {
	return guardedwrite.GitOutput(root, args...)
}

// GitOutputInput runs Git against root with input on stdin while preserving
// the same explicit-repository and sanitized-environment guarantees as
// GitOutput. It is intended for operations such as hashing an already-captured
// working-tree file through Git's path-specific clean filters.
func GitOutputInput(root string, input []byte, args ...string) ([]byte, error) {
	cmd := guardedwrite.GitCommand(root, args...)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
	}
	return nil, err
}
