package source

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const evidenceGitTimeout = 30 * time.Second

// WriteSelectedEvidenceCommit writes only Git objects for an immutable evidence
// commit. It starts from parentFullCommit and replaces the exact regular-file
// paths in files. Modes must contain exactly those same paths and be 100644 or
// 100755. No ref, HEAD, index, or worktree path is updated. When write is false,
// all input and parent-tree validation runs, but no Git objects are written and
// the returned commit ID is empty.
//
// The parent tree is loaded through an isolated temporary index. Unselected
// blobs are never opened; write-tree uses --missing-ok so unavailable unrelated
// blobs in a partial clone do not prevent the selected evidence commit. Such a
// commit can therefore reference parent-tree blobs that are not present locally.
func WriteSelectedEvidenceCommit(root, parentFullCommit string, files map[string][]byte, modes map[string]string, write bool) (string, error) {
	return WriteSelectedEvidenceCommitContext(context.Background(), root, parentFullCommit, files, modes, write)
}

// WriteSelectedEvidenceCommitContext is the cancelable form of
// WriteSelectedEvidenceCommit. Each Git process is also bounded by
// evidenceGitTimeout.
func WriteSelectedEvidenceCommitContext(ctx context.Context, root, parentFullCommit string, files map[string][]byte, modes map[string]string, write bool) (string, error) {
	if ctx == nil {
		return "", errors.New("evidence commit context is nil")
	}
	run := func(repo string, args ...string) ([]byte, error) {
		return evidenceGitOutputContext(ctx, repo, args...)
	}
	paths, err := validateEvidenceInput(files, modes)
	if err != nil {
		return "", err
	}
	identity, initialStats, err := identifyGit(root, run)
	if err != nil {
		return "", err
	}
	if err := validateFullCommit(identity.ObjectFormat, parentFullCommit); err != nil {
		return "", err
	}
	resolved, err := run(identity.Root, "rev-parse", "--verify", "--end-of-options", parentFullCommit+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("verify existing parent commit: %w", err)
	}
	if strings.TrimSpace(string(resolved)) != parentFullCommit {
		return "", errors.New("Git resolved the parent to a different commit ID")
	}
	head, err := run(identity.Root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", fmt.Errorf("read expected worktree HEAD: %w", err)
	}
	if strings.TrimSpace(string(head)) != parentFullCommit {
		return "", fmt.Errorf("worktree HEAD does not match expected parent commit %s", parentFullCommit)
	}

	tree, err := readEvidenceTreeWith(identity.Root, parentFullCommit, run)
	if err != nil {
		return "", err
	}
	if err := validateEvidenceTree(paths, modes, tree); err != nil {
		return "", err
	}
	if !write {
		return "", nil
	}

	if err := confirmEvidenceHeadWith(identity.Root, parentFullCommit, run); err != nil {
		return "", err
	}
	tempDir, err := os.MkdirTemp("", "markitect-evidence-index-")
	if err != nil {
		return "", fmt.Errorf("create external temporary index directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	indexPath := filepath.Join(tempDir, "index")
	env := evidenceGitEnvironment(indexPath)
	var created []string
	fail := func(err error) (string, error) {
		if len(created) != 0 {
			return "", fmt.Errorf("%w (Git objects already written: %s)", err, strings.Join(created, ", "))
		}
		return "", err
	}
	if _, err := evidenceGitOutputEnvContext(ctx, identity.Root, env, "read-tree", parentFullCommit); err != nil {
		return fail(fmt.Errorf("read parent tree into isolated index: %w", err))
	}
	for _, path := range paths {
		oidOut, err := evidenceGitInputEnvContext(ctx, identity.Root, env, files[path], "hash-object", "-w", "--stdin")
		if err != nil {
			return fail(fmt.Errorf("write selected evidence blob %q: %w", path, err))
		}
		oid := strings.TrimSpace(string(oidOut))
		if !validObjectID(identity.ObjectFormat, oid) {
			return fail(fmt.Errorf("Git returned invalid blob ID for %q", path))
		}
		created = append(created, oid)
		if _, err := evidenceGitOutputEnvContext(ctx, identity.Root, env, "update-index", "--add", "--cacheinfo", modes[path], oid, path); err != nil {
			return fail(fmt.Errorf("update isolated index for %q: %w", path, err))
		}
	}
	treeOut, err := evidenceGitOutputEnvContext(ctx, identity.Root, env, "write-tree", "--missing-ok")
	if err != nil {
		return fail(fmt.Errorf("write evidence tree: %w", err))
	}
	treeID := strings.TrimSpace(string(treeOut))
	if !validObjectID(identity.ObjectFormat, treeID) {
		return fail(errors.New("Git returned invalid evidence tree ID"))
	}
	created = append(created, treeID)

	commitOut, err := evidenceGitInputEnvContext(ctx, identity.Root, env, []byte("immutable selected evidence\n"), "commit-tree", treeID, "-p", parentFullCommit)
	if err != nil {
		return fail(fmt.Errorf("write immutable evidence commit: %w", err))
	}
	commitID := strings.TrimSpace(string(commitOut))
	if !validObjectID(identity.ObjectFormat, commitID) {
		return fail(errors.New("Git returned invalid evidence commit ID"))
	}
	created = append(created, commitID)

	if err := confirmEvidenceHeadWith(identity.Root, parentFullCommit, run); err != nil {
		return fail(err)
	}
	if err := confirmGitIdentity(identity, initialStats, run); err != nil {
		return fail(fmt.Errorf("recheck Git repository identity: %w", err))
	}
	return commitID, nil
}

func validateEvidenceInput(files map[string][]byte, modes map[string]string) ([]string, error) {
	if len(files) == 0 {
		return nil, errors.New("selected evidence must contain at least one file")
	}
	if len(files) > DefaultMaxFiles || len(modes) != len(files) {
		return nil, errors.New("selected evidence paths and modes must be nonempty, bounded, and have matching keys")
	}
	paths := make([]string, 0, len(files))
	var total int64
	for path, data := range files {
		if err := validateRepoPath(path); err != nil {
			return nil, err
		}
		for _, component := range strings.Split(path, "/") {
			if component == ".git" {
				return nil, fmt.Errorf("selected evidence path %q enters Git metadata", path)
			}
		}
		if len(data) > DefaultMaxFileBytes || total > DefaultMaxTotalBytes-int64(len(data)) {
			return nil, fmt.Errorf("selected evidence exceeds size limits at %q", path)
		}
		if !utf8.Valid(data) {
			return nil, fmt.Errorf("selected evidence file %q is not valid UTF-8", path)
		}
		mode, ok := modes[path]
		if !ok || (mode != "100644" && mode != "100755") {
			return nil, fmt.Errorf("selected evidence file %q has unsupported or missing mode %q", path, mode)
		}
		total += int64(len(data))
		paths = append(paths, path)
	}
	for path := range modes {
		if _, ok := files[path]; !ok {
			return nil, fmt.Errorf("mode supplied without selected evidence file %q", path)
		}
	}
	if err := validatePortablePaths(paths); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func validateFullCommit(objectFormat, commit string) error {
	expected := 40
	if objectFormat == "sha256" {
		expected = 64
	}
	if len(commit) != expected || commit != strings.ToLower(commit) || !validObjectID(objectFormat, commit) {
		return errors.New("parent must be a full lowercase commit ID matching the repository object format")
	}
	return nil
}

func validObjectID(objectFormat, oid string) bool {
	n := 40
	if objectFormat == "sha256" {
		n = 64
	}
	if len(oid) != n || oid != strings.ToLower(oid) {
		return false
	}
	_, err := hex.DecodeString(oid)
	return err == nil
}

func readEvidenceTreeWith(root, commit string, run gitOutputFunc) (map[string]selectedTreeEntry, error) {
	out, err := run(root, "ls-tree", "-r", "-z", "--full-tree", "--long", commit)
	if err != nil {
		return nil, fmt.Errorf("read parent tree metadata: %w", err)
	}
	tree := make(map[string]selectedTreeEntry)
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := bytes.Cut(record, []byte{'\t'})
		if !ok {
			return nil, errors.New("malformed parent Git tree record")
		}
		fields := strings.Fields(string(header))
		if len(fields) != 4 {
			return nil, errors.New("malformed parent Git tree metadata")
		}
		path := string(name)
		if err := validateRepoPath(path); err != nil {
			return nil, fmt.Errorf("unsafe parent tree path: %w", err)
		}
		entry := selectedTreeEntry{treeFile: treeFile{path: path, mode: fields[0], oid: fields[2]}, kind: fields[1]}
		if _, exists := tree[path]; exists {
			return nil, fmt.Errorf("duplicate parent tree path %q", path)
		}
		tree[path] = entry
	}
	return tree, nil
}

func validateEvidenceTree(paths []string, modes map[string]string, tree map[string]selectedTreeEntry) error {
	all := make([]string, 0, len(tree)+len(paths))
	for path := range tree {
		all = append(all, path)
	}
	all = append(all, paths...)
	if err := validatePortablePaths(all); err != nil {
		return fmt.Errorf("parent tree and selected paths have a portable path collision: %w", err)
	}
	for _, path := range paths {
		if prior, ok := tree[path]; ok {
			if prior.kind != "blob" || (prior.mode != "100644" && prior.mode != "100755") {
				return fmt.Errorf("selected path %q may replace only an existing regular file", path)
			}
		}
		if mode := modes[path]; mode != "100644" && mode != "100755" {
			return fmt.Errorf("unsupported selected file mode %q for %q", mode, path)
		}
	}
	return nil
}

func evidenceGitEnvironment(indexPath string) []string {
	env := selectiveGitEnvironment()
	env = append(env,
		"GIT_INDEX_FILE="+indexPath,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=NUL",
		"GIT_AUTHOR_NAME=Markitect Evidence",
		"GIT_AUTHOR_EMAIL=evidence@invalid",
		"GIT_AUTHOR_DATE=2000-01-01T00:00:00+00:00",
		"GIT_COMMITTER_NAME=Markitect Evidence",
		"GIT_COMMITTER_EMAIL=evidence@invalid",
		"GIT_COMMITTER_DATE=2000-01-01T00:00:00+00:00",
	)
	return env
}

func evidenceGitOutputContext(ctx context.Context, root string, args ...string) ([]byte, error) {
	return evidenceGitOutputEnvWithContext(ctx, root, selectiveGitEnvironment(), nil, args...)
}

func evidenceGitOutputEnvContext(ctx context.Context, root string, env []string, args ...string) ([]byte, error) {
	return evidenceGitInputEnvContext(ctx, root, env, nil, args...)
}

func evidenceGitInputEnvContext(ctx context.Context, root string, env []string, input []byte, args ...string) ([]byte, error) {
	bounded, cancel := context.WithTimeout(ctx, evidenceGitTimeout)
	defer cancel()
	return evidenceGitOutputEnvWithContext(bounded, root, env, input, args...)
}

func evidenceGitOutputEnvWithContext(ctx context.Context, root string, env []string, input []byte, args ...string) ([]byte, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Git root: %w", err)
	}
	cmdArgs := append([]string{"--no-replace-objects", "-c", "commit.gpgSign=false", "-c", "safe.directory=" + filepath.ToSlash(abs), "-C", abs}, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	cmd.Env = env
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("Git command timed out or was canceled: %w", ctx.Err())
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return out, nil
}

func confirmEvidenceHeadWith(root, expected string, run gitOutputFunc) error {
	out, err := run(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return fmt.Errorf("recheck expected worktree HEAD: %w", err)
	}
	if strings.TrimSpace(string(out)) != expected {
		return fmt.Errorf("worktree HEAD changed during evidence commit creation; expected %s", expected)
	}
	return nil
}
