package testkit

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Fixture identity and the time of a repository's first commit. Later commits
// are one minute apart, so commit identifiers are the same in every run.
const (
	AuthorName  = "Markitect Test"
	AuthorEmail = "markitect-test@example.invalid"
)

var firstCommitTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Repo is a Git repository fixture in a short temporary directory.
type Repo struct {
	t       testing.TB
	Dir     string
	commits int
}

// NewRepo initializes an empty repository on branch main. Its own
// configuration fixes the identity, keeps line endings unchanged and turns
// off signing and background maintenance.
func NewRepo(t testing.TB) *Repo {
	t.Helper()
	r := &Repo{t: t, Dir: TempDir(t)}
	r.Git("init", "--quiet", "--initial-branch=main")
	for _, setting := range [][2]string{
		{"user.name", AuthorName},
		{"user.email", AuthorEmail},
		{"core.autocrlf", "false"},
		{"commit.gpgsign", "false"},
		{"tag.gpgsign", "false"},
		{"gc.auto", "0"},
		{"maintenance.auto", "false"},
	} {
		r.Git("config", setting[0], setting[1])
	}
	return r
}

// Git runs git in the repository and returns its trimmed standard output. It
// fails the test when git fails. Git runs without the global and system
// configuration of the machine and without inherited GIT_* variables.
func (r *Repo) Git(args ...string) string {
	r.t.Helper()
	out, err := r.run(nil, args...)
	if err != nil {
		r.t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return out
}

// Write writes a file relative to the repository root, creating parent
// directories.
func (r *Repo) Write(path, content string) {
	r.t.Helper()
	full := filepath.Join(r.Dir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		r.t.Fatalf("create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		r.t.Fatalf("write %s: %v", path, err)
	}
}

// Commit stages every change and commits it with the fixture identity and the
// next fixed date. It returns the new commit identifier.
func (r *Repo) Commit(message string) string {
	r.t.Helper()
	r.Git("add", "--all")
	when := firstCommitTime.Add(time.Duration(r.commits) * time.Minute).Format(time.RFC3339)
	r.commits++
	if _, err := r.run([]string{"GIT_AUTHOR_DATE=" + when, "GIT_COMMITTER_DATE=" + when}, "commit", "--quiet", "--allow-empty", "--no-verify", "--message", message); err != nil {
		r.t.Fatalf("git commit: %v", err)
	}
	return r.Git("rev-parse", "HEAD")
}

func (r *Repo) run(extraEnv []string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", r.Dir}, args...)...)
	cmd.Env = append(GitEnv(), extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GitEnv returns the current environment without GIT_* variables and with
// the global and system Git configuration turned off. Use it for git
// processes a test starts itself.
func GitEnv() []string {
	env := make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			env = append(env, entry)
		}
	}
	return append(env, "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
}
