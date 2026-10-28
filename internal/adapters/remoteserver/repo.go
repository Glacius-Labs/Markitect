// Package remoteserver is the Run remote inside its container: the Git smart
// HTTP server with one credential per agent, the pre-receive hook that applies
// the rules of internal/runremote, the required checks under a throwaway user,
// and seeding and exporting the bare repository.
package remoteserver

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// The layout under the remote's root directory.
const (
	repoName = "run.git"
	planName = "plan.json"
	logName  = "log.jsonl"
	tmpName  = "tmp"
	seedName = "seeded"
)

// Layout names the files of one remote.
type Layout struct{ Root string }

func (l Layout) Repo() string { return filepath.Join(l.Root, repoName) }
func (l Layout) Plan() string { return filepath.Join(l.Root, planName) }
func (l Layout) Log() string  { return filepath.Join(l.Root, logName) }
func (l Layout) Tmp() string  { return filepath.Join(l.Root, tmpName) }

// gitConfigEnv keeps system and global Git configuration out of every Git
// process of the remote.
func gitConfigEnv() []string {
	return []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_TERMINAL_PROMPT=0"}
}

// gitRepo answers runremote.Repo with the git CLI. Inside the hook its
// environment carries Git's quarantine, so the pushed objects are visible.
type gitRepo struct {
	git string
	dir string
	env []string
}

func (r gitRepo) run(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.git, append([]string{"--git-dir=" + r.dir}, args...)...)
	cmd.Env = r.env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func (r gitRepo) Type(id string) (string, error) {
	out, err := r.run("cat-file", "-t", id)
	return strings.TrimSpace(string(out)), err
}

func (r gitRepo) Commit(id string) (runremote.Commit, error) {
	out, err := r.run("cat-file", "commit", id)
	if err != nil {
		return runremote.Commit{}, err
	}
	head, message, _ := strings.Cut(string(out), "\n\n")
	c := runremote.Commit{ID: id, Message: message}
	for _, line := range strings.Split(head, "\n") {
		key, value, _ := strings.Cut(line, " ")
		switch key {
		case "tree":
			c.Tree = value
		case "parent":
			c.Parents = append(c.Parents, value)
		}
	}
	if c.Tree == "" {
		return runremote.Commit{}, fmt.Errorf("commit %s has no tree", id)
	}
	return c, nil
}

func (r gitRepo) New(tip string) ([]string, error) {
	out, err := r.run("rev-list", "--reverse", tip, "--not", "--all")
	return strings.Fields(string(out)), err
}

func (r gitRepo) IsAncestor(a, b string) (bool, error) {
	_, err := r.run("merge-base", "--is-ancestor", a, b)
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return err == nil, err
}

func (r gitRepo) Changed(a, b string) ([]string, error) {
	out, err := r.run("diff-tree", "-r", "-z", "--no-renames", "--name-only", a, b)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

func (r gitRepo) Files(commit string) ([]string, error) {
	out, err := r.run("ls-tree", "-r", "-z", "--name-only", "--full-tree", commit)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

// Tips reads the branch's reflog, which Git writes in the same transaction as
// every update and which no agent can touch.
func (r gitRepo) Tips(branch string) ([]string, error) {
	f, err := os.Open(filepath.Join(r.dir, "logs", "refs", "heads", filepath.FromSlash(branch)))
	if err != nil {
		return nil, fmt.Errorf("the record of branch %s: %w", branch, err)
	}
	defer f.Close()
	var tips []string
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64<<10), 1<<20)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) >= 2 && fields[1] != runremote.ZeroID {
			tips = append(tips, fields[1])
		}
	}
	return tips, s.Err()
}
