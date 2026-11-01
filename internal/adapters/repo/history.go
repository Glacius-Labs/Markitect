package repo

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// ModelCommit is one commit that changed Markitect's folder.
type ModelCommit struct {
	ID      string
	When    time.Time
	Subject string
	// Landed: the commit adds a Run's Result page.
	Landed bool
}

// ModelHistory returns up to limit commits reachable from HEAD that changed
// Markitect's folder, newest first. A repository without a commit has none.
func ModelHistory(ctx context.Context, root string, limit int) ([]ModelCommit, error) {
	if _, err := git(ctx, root, "rev-parse", "--verify", "--quiet", "HEAD^{commit}"); err != nil {
		return nil, nil
	}
	out, err := git(ctx, root, "log", "-n", strconv.Itoa(limit), "--format=%H%x00%ct%x00%s", "HEAD", "--", project.Dir+"/")
	if err != nil {
		return nil, err
	}
	landed, err := git(ctx, root, "log", "--format=%H", "--diff-filter=A", "HEAD", "--", project.ResultsDir+"/")
	if err != nil {
		return nil, err
	}
	lands := map[string]bool{}
	for _, id := range strings.Fields(string(landed)) {
		lands[id] = true
	}
	var commits []ModelCommit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\x00", 3)
		if len(parts) != 3 {
			continue
		}
		secs, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		commits = append(commits, ModelCommit{ID: parts[0], When: time.Unix(secs, 0).UTC(), Subject: parts[2], Landed: lands[parts[0]]})
	}
	return commits, nil
}

// Snapshot reads a commit as a repository state in two Git processes: its
// file list, and the content of every file below Markitect's folder, which
// is all a project's load reads.
func Snapshot(ctx context.Context, root, rev string) (project.Repository, error) {
	out, err := git(ctx, root, "ls-tree", "-r", "-z", "--name-only", "--full-tree", rev)
	if err != nil {
		return nil, err
	}
	s := snapshot{data: map[string][]byte{}}
	var wanted []string
	for _, f := range split(out) {
		s.files = append(s.files, f)
		if strings.HasPrefix(f, project.Dir+"/") && !strings.ContainsAny(f, "\n\r") {
			wanted = append(wanted, f)
		}
	}
	if len(wanted) == 0 {
		return s, nil
	}
	var in strings.Builder
	for _, f := range wanted {
		in.WriteString(rev + ":" + f + "\n")
	}
	cmd := exec.CommandContext(ctx, "git", "-c", "core.quotepath=off", "cat-file", "--batch")
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(in.String())
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git cat-file: %s", strings.TrimSpace(stderr.String()))
	}
	r := bufio.NewReader(&stdout)
	for _, f := range wanted {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("git cat-file: %w", err)
		}
		if strings.HasSuffix(strings.TrimSpace(header), " missing") {
			continue
		}
		fields := strings.Fields(header)
		if len(fields) != 3 {
			return nil, fmt.Errorf("git cat-file: %q", header)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("git cat-file: %q", header)
		}
		// Every object's content follows its header, also one that is no file.
		data := make([]byte, size+1) // the content and its closing LF
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("git cat-file: %w", err)
		}
		if fields[1] == "blob" {
			s.data[f] = data[:size]
		}
	}
	return s, nil
}

// snapshot is a commit's file list with the content of the files below
// Markitect's folder.
type snapshot struct {
	files []string
	data  map[string][]byte
}

func (s snapshot) Files() ([]string, error) { return append([]string(nil), s.files...), nil }

func (s snapshot) Read(file string) ([]byte, error) {
	if data, ok := s.data[file]; ok {
		return bytes.Clone(data), nil
	}
	return nil, fs.ErrNotExist
}

// GitIgnored is false: a commit holds only what Git tracks.
func (snapshot) GitIgnored(string) bool { return false }
