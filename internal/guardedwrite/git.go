package guardedwrite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitIdentity identifies the repository location and object format used to
// acquire a fixed Git tree. Remote configuration is deliberately omitted:
// it is informational, mutable, and may contain credentials.
type GitIdentity struct {
	Root         string `yaml:"root"`
	GitDir       string `yaml:"gitDir"`
	CommonDir    string `yaml:"commonDir"`
	ObjectFormat string `yaml:"objectFormat"`
	Digest       string `yaml:"digest"`
}

// IdentifyGitContext requires root to be the real top-level directory of a
// Git repository and returns its canonical worktree and Git-directory
// identity. It uses one caller context for the combined metadata query and
// every fallback query.
func IdentifyGitContext(ctx context.Context, root string) (GitIdentity, error) {
	return identifyGit(root, func(root string, args ...string) ([]byte, error) {
		return gitOutputContext(ctx, root, identityGitEnvironment(), args...)
	})
}

type gitOutputFunc func(root string, args ...string) ([]byte, error)

func identifyGit(root string, run gitOutputFunc) (GitIdentity, error) {
	var zero GitIdentity
	if root == "" {
		return zero, errors.New("source root is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return zero, fmt.Errorf("resolve source root: %w", err)
	}
	if err := RejectSymlinkAncestors(abs); err != nil {
		return zero, fmt.Errorf("source root: %w", err)
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return zero, fmt.Errorf("stat source root: %w", err)
	}
	if IsReparsePoint(info) || !info.IsDir() {
		return zero, fmt.Errorf("source root must be a real directory: %s", abs)
	}
	canonicalRoot, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return zero, fmt.Errorf("canonicalize source root: %w", err)
	}
	canonicalRoot, err = filepath.Abs(canonicalRoot)
	if err != nil {
		return zero, fmt.Errorf("resolve canonical source root: %w", err)
	}
	canonicalRoot = filepath.Clean(canonicalRoot)

	top, gitDirOut, commonDirOut, objectFormatOut, err := identifyGitMetadata(canonicalRoot, run)
	if err != nil {
		return zero, err
	}
	canonicalTop, err := canonicalExistingPath(strings.TrimSpace(string(top)))
	if err != nil {
		return zero, fmt.Errorf("canonicalize Git worktree root: %w", err)
	}
	topInfo, err := os.Stat(canonicalTop)
	if err != nil || !topInfo.IsDir() || !os.SameFile(info, topInfo) {
		return zero, fmt.Errorf("source root must be the Git worktree top-level directory: %s", canonicalRoot)
	}
	// Use Git's spelling of the top-level path. On Windows this also expands
	// short 8.3 path aliases returned by callers.
	canonicalRoot = canonicalTop
	gitDir, err := canonicalExistingPath(strings.TrimSpace(string(gitDirOut)))
	if err != nil {
		return zero, fmt.Errorf("canonicalize Git directory: %w", err)
	}
	commonDir, err := canonicalExistingPath(strings.TrimSpace(string(commonDirOut)))
	if err != nil {
		return zero, fmt.Errorf("canonicalize common Git directory: %w", err)
	}
	gitInfo, err := os.Stat(gitDir)
	if err != nil || !gitInfo.IsDir() {
		return zero, fmt.Errorf("Git directory is not a readable directory: %s", gitDir)
	}
	commonInfo, err := os.Stat(commonDir)
	if err != nil || !commonInfo.IsDir() {
		return zero, fmt.Errorf("common Git directory is not a readable directory: %s", commonDir)
	}
	objectFormat := strings.TrimSpace(string(objectFormatOut))
	if objectFormat != "sha1" && objectFormat != "sha256" {
		return zero, fmt.Errorf("unsupported Git object format %q", objectFormat)
	}
	identity := GitIdentity{Root: canonicalRoot, GitDir: gitDir, CommonDir: commonDir, ObjectFormat: objectFormat}
	identity.Digest = gitIdentityDigest(identity)
	return identity, nil
}

// identifyGitMetadata obtains the same four identity fields with one Git
// process in the usual case. rev-parse emits newline-delimited values rather
// than a NUL-delimited record; if a repository path itself contains a newline
// (or the output is otherwise ambiguous), fall back to the original one-field
// queries so those paths retain their established handling.
func identifyGitMetadata(root string, run gitOutputFunc) (top, gitDir, commonDir, objectFormat []byte, err error) {
	combined, combinedErr := run(root, "rev-parse", "--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir", "--show-object-format")
	if combinedErr == nil {
		if fields, ok := parseGitIdentityMetadata(combined); ok {
			return fields[0], fields[1], fields[2], fields[3], nil
		}
	}

	top, err = run(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("identify Git worktree root: %w", err)
	}
	gitDir, err = run(root, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("identify Git directory: %w", err)
	}
	commonDir, err = run(root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("identify common Git directory: %w", err)
	}
	objectFormat, err = run(root, "rev-parse", "--show-object-format")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("identify Git object format: %w", err)
	}
	return top, gitDir, commonDir, objectFormat, nil
}

func parseGitIdentityMetadata(output []byte) ([4][]byte, bool) {
	var fields [4][]byte
	parts := bytes.Split(output, []byte{'\n'})
	if len(parts) != len(fields)+1 || len(parts[len(parts)-1]) != 0 {
		return fields, false
	}
	for i := range fields {
		if len(parts[i]) == 0 || bytes.ContainsAny(parts[i], "\r") {
			return fields, false
		}
		fields[i] = parts[i]
	}
	return fields, true
}

func canonicalExistingPath(value string) (string, error) {
	if value == "" {
		return "", errors.New("path is empty")
	}
	resolved, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func gitIdentityDigest(identity GitIdentity) string {
	h := sha256.New()
	for _, value := range []string{identity.Root, identity.GitDir, identity.CommonDir, identity.ObjectFormat} {
		_, _ = fmt.Fprintf(h, "%d:", len(value))
		_, _ = h.Write([]byte(value))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// identityGitEnvironment keeps identity queries from fetching promised
// objects, taking optional locks or prompting.
func identityGitEnvironment() []string {
	env := CleanGitEnv()
	return append(env, "GIT_NO_LAZY_FETCH=1", "GIT_ALLOW_PROTOCOL=", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
}

// GitOutput runs Git against root while discarding inherited Git environment
// variables. Those variables can silently redirect repository discovery,
// indexes, objects or configuration to another checkout. The exact repository
// is selected explicitly and marked safe for Git's ownership check.
func GitOutput(root string, args ...string) ([]byte, error) {
	cmd := GitCommand(root, args...)
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

// GitOutputContext binds the Git process and pipe drain to the caller's
// original execution window. Cancellation is reported only after Wait returns.
func GitOutputContext(ctx context.Context, root string, args ...string) ([]byte, error) {
	return gitOutputContext(ctx, root, CleanGitEnv(), args...)
}

func gitOutputContext(ctx context.Context, root string, env []string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cmd := gitCommandWithEnvContext(ctx, root, env, args...)
	cmd.WaitDelay = time.Second
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exit.Stderr)))
	}
	return out, err
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

// GitCommand returns an unstarted Git command against root with the
// environment of CleanGitEnv, for callers that stream its input or output.
func GitCommand(root string, args ...string) *exec.Cmd {
	return gitCommandWithEnvContext(context.Background(), root, CleanGitEnv(), args...)
}

func gitCommandWithEnvContext(ctx context.Context, root string, env []string, args ...string) *exec.Cmd {
	abs, err := filepath.Abs(root)
	if err != nil {
		// Callers validate and resolve root before invoking Git. Preserve a
		// deterministic failing command if Abs nevertheless fails.
		abs = root
	}
	gitArgs := append([]string{"--no-replace-objects", "-c", "safe.directory=" + filepath.ToSlash(abs), "-C", abs}, args...)
	cmd := exec.CommandContext(ctx, "git", gitArgs...)
	cmd.Env = env
	return cmd
}
