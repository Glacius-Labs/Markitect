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
		return nil, ctx.Err()
	}
	n := strconv.Itoa(limit)
	out, err := git(ctx, root, "log", "-n", n, "--format=%H%x00%ct%x00%s", "HEAD", "--", project.Dir+"/")
	if err != nil {
		return nil, err
	}
	// A landing changes Markitect's folder, so the last limit landings hold
	// every landing among the commits above.
	landed, err := git(ctx, root, "log", "-n", n, "--format=%H", "--diff-filter=A", "HEAD", "--", project.ResultsDir+"/")
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

// loaded reports whether a project's load reads a file. It leaves out
// Markitect's other files, such as the installed tool's source archive and
// the Results, which may be large.
func loaded(f string) bool {
	switch f {
	case project.ManifestFile, project.LockFile, project.ChecksFile, project.IgnoreFile, project.RootPage:
		return true
	}
	for _, dir := range []string{project.ModelDir, project.RegistryDir, project.LibraryDir} {
		if strings.HasPrefix(f, dir+"/") {
			return true
		}
	}
	return false
}

// Snapshot reads a commit as a repository state in two Git processes: its
// file list, and the content of the files a project's load reads.
func Snapshot(ctx context.Context, root, rev string) (project.Repository, error) {
	out, err := git(ctx, root, "ls-tree", "-r", "-z", "--name-only", "--full-tree", rev)
	if err != nil {
		return nil, err
	}
	s := snapshot{data: map[string][]byte{}}
	var wanted []string
	for _, f := range split(out) {
		s.files = append(s.files, f)
		if loaded(f) && !strings.ContainsAny(f, "\n\r") {
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
		// A missing object ("NAME missing") and a submodule ("ID submodule")
		// have no content; the load then reports the file it cannot read.
		fields := strings.Fields(header)
		if len(fields) < 3 || fields[len(fields)-1] == "missing" {
			continue
		}
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
