package gitrepo

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type boundedOutput struct {
	bytes.Buffer
	limit int64
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if int64(len(p)) > b.limit-int64(b.Len()) {
		return 0, ErrLimit
	}
	return b.Buffer.Write(p)
}

type gitEntry struct {
	path, mode, kind, oid string
	size                  int64
}
type gitRepository struct {
	ctx                             context.Context
	path, gitDir, commonDir, format string
	rootInfo, gitInfo, commonInfo   os.FileInfo
	tree                            map[string]gitEntry
	blobs                           *blobStream
}

func gitEnvironment() []string {
	var env []string
	for _, value := range os.Environ() {
		name, _, ok := strings.Cut(value, "=")
		if ok && !strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			env = append(env, value)
		}
	}
	return append(env, "GIT_NO_LAZY_FETCH=1", "GIT_ALLOW_PROTOCOL=", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
}

func (g *gitRepository) command(args ...string) *exec.Cmd {
	prefix := []string{"--no-replace-objects", "--literal-pathspecs", "-c", "safe.directory=" + filepath.ToSlash(g.path), "-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "core.hooksPath=", "-C", g.path}
	cmd := exec.CommandContext(g.ctx, "git", append(prefix, args...)...)
	cmd.Env = gitEnvironment()
	cmd.WaitDelay = time.Second
	return cmd
}

func (g *gitRepository) run(args ...string) ([]byte, error) {
	cmd := g.command(args...)
	out, stderr := &boundedOutput{limit: 64 << 20}, &boundedOutput{limit: 16 << 10}
	cmd.Stdout, cmd.Stderr = out, stderr
	if err := cmd.Run(); err != nil {
		if g.ctx.Err() != nil {
			return nil, g.ctx.Err()
		}
		return nil, fmt.Errorf("Git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out.Bytes(), nil
}

func identifyGit(ctx context.Context, root string) (*gitRepository, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := realDirectory(abs)
	if err != nil {
		return nil, err
	}
	g := &gitRepository{ctx: ctx, path: abs, rootInfo: info}
	out, err := g.run("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	top := strings.TrimSuffix(strings.TrimSuffix(string(out), "\n"), "\r")
	topInfo, err := os.Lstat(top)
	if err != nil || !os.SameFile(info, topInfo) {
		return nil, fmt.Errorf("%w: repository must be the worktree root", ErrUnsafe)
	}
	g.path = top
	for _, target := range []struct {
		arg  string
		path *string
		info *os.FileInfo
	}{
		{"--absolute-git-dir", &g.gitDir, &g.gitInfo}, {"--git-common-dir", &g.commonDir, &g.commonInfo},
	} {
		out, err := g.run("rev-parse", "--path-format=absolute", target.arg)
		if err != nil {
			return nil, err
		}
		*target.path = strings.TrimSuffix(strings.TrimSuffix(string(out), "\n"), "\r")
		*target.info, err = realDirectory(*target.path)
		if err != nil {
			return nil, err
		}
	}
	out, err = g.run("rev-parse", "--show-object-format")
	if err != nil {
		return nil, err
	}
	g.format = strings.TrimSpace(string(out))
	if g.format != "sha1" && g.format != "sha256" {
		return nil, fmt.Errorf("%w: unsupported Git object format", ErrUnsafe)
	}
	return g, nil
}

func (g *gitRepository) commit(selector string, fixed bool) (string, error) {
	if fixed {
		length := 40
		if g.format == "sha256" {
			length = 64
		}
		if len(selector) != length || selector != strings.ToLower(selector) {
			return "", fmt.Errorf("%w: fixed revision must be a full resolved commit", ErrUnsafe)
		}
		if _, err := hex.DecodeString(selector); err != nil {
			return "", fmt.Errorf("%w: nonhex fixed revision", ErrUnsafe)
		}
	}
	out, err := g.run("rev-parse", "--verify", "--end-of-options", selector+"^{commit}")
	if err != nil {
		return "", err
	}
	commit := strings.TrimSpace(string(out))
	length := 40
	if g.format == "sha256" {
		length = 64
	}
	if len(commit) != length || commit != strings.ToLower(commit) {
		return "", fmt.Errorf("%w: invalid resolved Git commit", ErrUnsafe)
	}
	if _, err := hex.DecodeString(commit); err != nil {
		return "", fmt.Errorf("%w: invalid resolved Git commit", ErrUnsafe)
	}
	if fixed && commit != selector {
		return "", fmt.Errorf("%w: fixed revision resolved differently", ErrChanged)
	}
	return commit, nil
}

func (g *gitRepository) inventory(commit string, max int) error {
	out, err := g.run("ls-tree", "-r", "-t", "-z", "--full-tree", "--long", commit)
	if err != nil {
		return err
	}
	g.tree = make(map[string]gitEntry)
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := bytes.Cut(record, []byte{'\t'})
		fields := strings.Fields(string(header))
		if !ok || len(fields) != 4 {
			return fmt.Errorf("%w: malformed Git tree record", ErrUnsafe)
		}
		p := string(name)
		if technical(p) || p == ".markitect/state" && fields[1] == "tree" {
			continue
		}
		if _, err := snapshot.NewPath(p); err != nil {
			return err
		}
		if _, exists := g.tree[p]; exists {
			return fmt.Errorf("%w: duplicate Git tree path", ErrUnsafe)
		}
		e := gitEntry{path: p, mode: fields[0], kind: fields[1], oid: fields[2]}
		if e.kind == "blob" {
			e.size, err = strconv.ParseInt(fields[3], 10, 64)
			if err != nil || e.size < 0 {
				return fmt.Errorf("%w: unavailable Git blob", ErrUnsafe)
			}
		} else if fields[3] != "-" {
			return fmt.Errorf("%w: malformed Git tree size", ErrUnsafe)
		}
		g.tree[p] = e
		if len(g.tree) > max {
			return fmt.Errorf("%w: Git entries", ErrLimit)
		}
	}
	return nil
}

func (g *gitRepository) confirm() error {
	after, err := identifyGit(g.ctx, g.path)
	if err != nil {
		return err
	}
	if g.path != after.path || g.gitDir != after.gitDir || g.commonDir != after.commonDir || g.format != after.format || !os.SameFile(g.rootInfo, after.rootInfo) || !os.SameFile(g.gitInfo, after.gitInfo) || !os.SameFile(g.commonInfo, after.commonInfo) {
		return ErrChanged
	}
	return nil
}

func (g *gitRepository) clean() error {
	// Status may execute attribute clean filters. Read index metadata and raw
	// files instead; no filter, fsmonitor, hook or user program is invoked.
	out, err := g.run("ls-files", "--stage", "-z")
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := bytes.Cut(record, []byte{'\t'})
		fields := strings.Fields(string(header))
		p := string(name)
		if !ok || len(fields) != 3 {
			return fmt.Errorf("%w: malformed index", ErrUnsafe)
		}
		if technical(p) {
			continue
		}
		expected, exists := g.tree[p]
		if !exists || fields[2] != "0" || fields[0] != expected.mode || fields[1] != expected.oid || seen[p] {
			return fmt.Errorf("%w: index at %q", ErrDirty, p)
		}
		seen[p] = true
	}
	for p, entry := range g.tree {
		if entry.kind != "tree" && !seen[p] {
			return fmt.Errorf("%w: missing index file %q", ErrDirty, p)
		}
	}
	out, err = g.run("ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return err
	}
	for _, name := range bytes.Split(out, []byte{0}) {
		if len(name) > 0 && !technical(string(name)) {
			return fmt.Errorf("%w: untracked %q", ErrDirty, name)
		}
	}
	return nil
}

func technical(p string) bool {
	if strings.HasPrefix(p, ".markitect/state/") {
		return true
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".git" {
			return true
		}
	}
	return false
}
