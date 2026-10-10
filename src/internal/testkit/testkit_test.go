package testkit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) { Main(m) }

func TestNewRepoCommitsAreDeterministic(t *testing.T) {
	var commits []string
	for range 2 {
		repo := NewRepo(t)
		repo.Write("docs/readme.md", "line one\r\nline two\n")
		first := repo.Commit("first")
		repo.Write("docs/readme.md", "changed\n")
		commits = append(commits, first, repo.Commit("second"))
		if got := repo.Git("cat-file", "-p", first+":docs/readme.md"); got != "line one\r\nline two" {
			t.Fatalf("committed bytes changed: %q", got)
		}
		if got := repo.Git("branch", "--show-current"); got != "main" {
			t.Fatalf("branch = %q, want main", got)
		}
	}
	if commits[0] != commits[2] || commits[1] != commits[3] {
		t.Fatalf("commit identifiers differ between identical fixtures: %v", commits)
	}
}

func TestTempDirIsShortAndCanonical(t *testing.T) {
	dir := TempDir(t)
	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dir != canonical {
		t.Fatalf("TempDir = %q, canonical spelling %q", dir, canonical)
	}
	if strings.Contains(dir, t.Name()) {
		t.Fatalf("TempDir %q contains the test name", dir)
	}
}

func TestRemoveAllRetriesWhileAFileIsOpen(t *testing.T) {
	dir, err := os.MkdirTemp("", "mk")
	if err != nil {
		t.Fatal(err)
	}
	held, err := os.Create(filepath.Join(dir, "held"))
	if err != nil {
		t.Fatal(err)
	}
	// On Windows the open handle blocks removal until it is closed.
	go func() {
		time.Sleep(200 * time.Millisecond)
		held.Close()
	}()
	if err := removeAll(dir); err != nil {
		t.Fatalf("removeAll: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("directory still exists: %v", err)
	}
}

// TestIsolateShieldsGitFromAHostileConfiguration runs git the way production
// code does, with inherited GIT_* variables removed, after a hostile global
// configuration has been installed through HOME, XDG_CONFIG_HOME and GIT_*
// variables.
func TestIsolateShieldsGitFromAHostileConfiguration(t *testing.T) {
	hostileHome := t.TempDir()
	hostile := filepath.Join(hostileHome, ".gitconfig")
	data, err := os.ReadFile(filepath.Join("testdata", "hostile.gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hostile, data, 0o600); err != nil {
		t.Fatal(err)
	}
	hooks := filepath.Join(hostileHome, "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, hook := range []string{"pre-commit", "commit-msg"} {
		if err := os.WriteFile(filepath.Join(hooks, hook), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	excludes := filepath.Join(hostileHome, "ignore-everything")
	if err := os.WriteFile(excludes, []byte("*\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, setting := range [][2]string{{"core.hooksPath", hooks}, {"core.excludesFile", excludes}} {
		if out, err := exec.Command("git", "config", "--file", hostile, setting[0], setting[1]).CombinedOutput(); err != nil {
			t.Fatalf("write hostile %s: %v: %s", setting[0], err, out)
		}
	}
	t.Setenv("HOME", hostileHome)
	t.Setenv("XDG_CONFIG_HOME", hostileHome)
	t.Setenv("GIT_CONFIG_GLOBAL", hostile)
	t.Setenv("GIT_CONFIG_PARAMETERS", "'core.autocrlf'='true'")
	t.Setenv("GIT_DIR", filepath.Join(hostileHome, "missing"))
	t.Setenv("GIT_AUTHOR_DATE", "garbage")

	restore, err := Isolate()
	if err != nil {
		t.Fatal(err)
	}
	dir := TempDir(t)
	productionGit := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(strings.ToUpper(entry), "GIT_") {
				cmd.Env = append(cmd.Env, entry)
			}
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	productionGit("init", "--quiet")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("a\r\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	productionGit("add", "file.txt")
	productionGit("commit", "--quiet", "--message", "plain commit")
	if got := productionGit("branch", "--show-current"); got != "main" {
		t.Fatalf("default branch = %q, want main", got)
	}
	if got := productionGit("cat-file", "-p", "HEAD:file.txt"); got != "a\r\nb" {
		t.Fatalf("line endings were converted: %q", got)
	}
	if got := productionGit("log", "-1", "--format=%an <%ae>"); got != AuthorName+" <"+AuthorEmail+">" {
		t.Fatalf("author = %q", got)
	}
	if _, set := os.LookupEnv("GIT_DIR"); set {
		t.Fatal("GIT_DIR survived isolation")
	}

	restore()
	if got := os.Getenv("HOME"); got != hostileHome {
		t.Fatalf("HOME after restore = %q, want %q", got, hostileHome)
	}
	if got := os.Getenv("GIT_AUTHOR_DATE"); got != "garbage" {
		t.Fatalf("GIT_AUTHOR_DATE after restore = %q", got)
	}
}

// TestIsolatedProcessKeepsGoCaches checks that go commands started by an
// isolated test still use the original caches, not the temporary HOME.
func TestIsolatedProcessKeepsGoCaches(t *testing.T) {
	out, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("go command unavailable: %v", err)
	}
	home := os.Getenv("HOME")
	for _, dir := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if dir = strings.TrimSpace(dir); dir == "" || strings.HasPrefix(dir, home) {
			t.Fatalf("go cache %q is empty or lies in the isolated home %q", dir, home)
		}
	}
}
