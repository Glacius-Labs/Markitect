package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func gitOutput(ctx context.Context, repo string, input []byte, extraEnv []string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	hooks, err := os.MkdirTemp("", "government-no-hooks-")
	if err != nil {
		return "", err
	}
	defer os.Remove(hooks)
	argv := []string{"--no-replace-objects", "-c", "safe.directory=" + filepath.ToSlash(repo), "-c", "core.hooksPath=" + filepath.ToSlash(hooks), "-c", "core.fsmonitor=false", "-C", repo}
	cmd := exec.CommandContext(ctx, "git", append(argv, args...)...)
	cmd.Env = append(source.CleanGitEnv(), extraEnv...)
	cmd.WaitDelay = 2 * time.Second
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Git %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func realDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("directory must be absolute")
	}
	return government.ResolveOperationalDirectory(path)
}

func readWorkspace(root string, expectedModes map[string]string) (*snapshot.Snapshot, error) {
	s := &snapshot.Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("candidate contains a link")
		}
		if entry.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("candidate contains an unsupported file")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !government.SafePath(rel) {
			return fmt.Errorf("unsafe candidate path %q", rel)
		}
		if len(s.Files) >= source.DefaultMaxFiles || info.Size() > source.DefaultMaxFileBytes || total+info.Size() > source.DefaultMaxTotalBytes {
			return errors.New("candidate exceeds capture limits")
		}
		data, err := inventory.ReadInput(root, rel, source.DefaultMaxFileBytes)
		if err != nil {
			return err
		}
		total += int64(len(data))
		s.Files[rel] = data
		mode := snapshot.RegularMode
		if runtime.GOOS == "windows" && expectedModes[rel] != "" {
			mode = expectedModes[rel]
		} else if info.Mode().Perm()&0111 != 0 {
			mode = snapshot.ExecutableMode
		}
		s.Modes[rel] = mode
		return nil
	})
	return s, err
}

func confirmWorkspace(root string, expected *snapshot.Snapshot) error {
	actual, err := readWorkspace(root, expected.Modes)
	if err != nil {
		return err
	}
	if actual.Digest() != expected.Digest() {
		return errors.New("candidate bytes, paths or modes changed after material binding")
	}
	return nil
}

// commitCandidate writes objects via a private index. It neither updates HEAD
// nor assumes that the user's checkout is at the dedicated active revision.
func commitCandidate(ctx context.Context, repo, base, runID, state string, changed []string, candidate *snapshot.Snapshot) (commit, tree string, err error) {
	index := filepath.Join(state, "candidate.index")
	env := []string{"GIT_INDEX_FILE=" + index}
	if _, err = gitOutput(ctx, repo, nil, env, "read-tree", base); err != nil {
		return
	}
	for _, path := range changed {
		var blob string
		blob, err = gitOutput(ctx, repo, candidate.Files[path], env, "hash-object", "-w", "--stdin")
		if err != nil {
			return
		}
		if _, err = gitOutput(ctx, repo, nil, env, "update-index", "--add", "--cacheinfo", candidate.Modes[path]+","+blob+","+path); err != nil {
			return
		}
	}
	if tree, err = gitOutput(ctx, repo, nil, env, "write-tree"); err != nil {
		return
	}
	env = append(env, "GIT_AUTHOR_NAME=Markitect Government", "GIT_AUTHOR_EMAIL=government@example.invalid", "GIT_COMMITTER_NAME=Markitect Government", "GIT_COMMITTER_EMAIL=government@example.invalid")
	commit, err = gitOutput(ctx, repo, []byte("Government G2 "+runID+"\n"), env, "-c", "commit.gpgsign=false", "commit-tree", tree, "-p", base)
	return
}

func persistJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return government.WriteOperationalRecord(path, append(data, '\n'))
}

func sortedPaths(files map[string][]byte) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
