package source

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
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

func loadCommit(root, commit string, s *snapshot.Snapshot, limits Limits) error {
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
		blobSize, parseErr := strconv.ParseInt(fields[3], 10, 64)
		if parseErr != nil || blobSize < 0 {
			return fmt.Errorf("invalid Git blob size for %q", string(name))
		}
		p := string(name)
		if excludedFilePath(p) {
			continue
		}
		if err := validateRepoPath(p); err != nil {
			return err
		}
		if kind == "commit" || mode == "160000" {
			return fmt.Errorf("Git submodule is not a source file: %q", p)
		}
		if mode == "120000" {
			return fmt.Errorf("Git symlink is not a source file: %q", p)
		}
		if kind != "blob" || (mode != snapshot.RegularMode && mode != snapshot.ExecutableMode) {
			return fmt.Errorf("unsupported Git tree entry %q (mode %s, type %s)", p, mode, kind)
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
	if err := validatePortablePaths(paths); err != nil {
		return err
	}
	return loadBlobs(root, files, s)
}

func loadBlobs(root string, files []treeFile, s *snapshot.Snapshot) error {
	return loadBlobsWithCommand(root, files, s, gitCommand)
}

func loadBlobsWithCommand(root string, files []treeFile, s *snapshot.Snapshot, command func(root string, args ...string) *exec.Cmd) error {
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
	return GitOutput(root, args...)
}

// GitOutput runs Git against root while discarding inherited Git environment
// variables. Those variables can silently redirect repository discovery,
// indexes, objects or configuration to another checkout. The exact repository
// is selected explicitly and marked safe for Git's ownership check.
func GitOutput(root string, args ...string) ([]byte, error) {
	cmd := gitCommand(root, args...)
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

// CleanGitEnv returns the process environment without Git-specific variables.
// It is also used for child gate processes so an ambient GIT_DIR cannot make
// their repository checks inspect a different checkout.
func CleanGitEnv() []string {
	current := os.Environ()
	clean := make([]string, 0, len(current))
	for _, entry := range current {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || strings.HasPrefix(strings.ToUpper(key), "GIT_") {
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}

func gitCommand(root string, args ...string) *exec.Cmd {
	return gitCommandWithEnv(root, CleanGitEnv(), args...)
}

func gitCommandWithEnv(root string, env []string, args ...string) *exec.Cmd {
	abs, err := filepath.Abs(root)
	if err != nil {
		// Callers validate and resolve root before invoking Git. Preserve a
		// deterministic failing command if Abs nevertheless fails.
		abs = root
	}
	gitArgs := append([]string{"--no-replace-objects", "-c", "safe.directory=" + filepath.ToSlash(abs), "-C", abs}, args...)
	cmd := exec.Command("git", gitArgs...)
	cmd.Env = env
	return cmd
}
