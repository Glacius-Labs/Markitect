package testkit

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// fatalRecorder records a fatal failure instead of ending the test.
type fatalRecorder struct {
	testing.TB
	message string
}

func (r *fatalRecorder) Fatalf(format string, args ...any) { r.message = fmt.Sprintf(format, args...) }

func TestGitRunsInDirWithoutInheritedVariables(t *testing.T) {
	repo := NewRepo(t)
	repo.Write("file.txt", "content\n")
	head := repo.Commit("first")
	t.Setenv("GIT_DIR", filepath.Join(repo.Dir, "missing"))
	if got := Git(t, repo.Dir, "rev-parse", "HEAD"); got != head || len(got) != 40 {
		t.Fatalf("Git rev-parse HEAD = %q, want trimmed %q", got, head)
	}
	failed := &fatalRecorder{TB: t}
	Git(failed, repo.Dir, "rev-parse", "--verify", "missing-ref")
	if !strings.Contains(failed.message, "git rev-parse --verify missing-ref") || !strings.Contains(failed.message, "fatal:") {
		t.Fatalf("failure = %q, want the command and git's standard error", failed.message)
	}
}

func TestTempDirIsShortAndCanonical(t *testing.T) {
	dir := TempDir(t)
	tempRoot, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// One short entry ("mk" and at most ten digits) directly below the
	// canonical spelling of the temporary directory.
	if parent, base := filepath.Dir(dir), filepath.Base(dir); parent != tempRoot || !strings.HasPrefix(base, "mk") || len(base) > 12 {
		t.Fatalf("TempDir = %q, want a short entry below %q", dir, tempRoot)
	}
}

// TestReExecutedChildren runs this test binary again the way fake agents
// do: with a reduced environment, and with an inherited one.
func TestReExecutedChildren(t *testing.T) {
	if os.Getenv("MARKITECT_TESTKIT_CHILD") == "1" {
		return
	}
	run := func(env []string) string {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=^TestReExecutedChildHome$", "-test.v")
		cmd.Env = append(env, "MARKITECT_TESTKIT_CHILD=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("child failed: %v\n%s", err, out)
		}
		_, home, _ := strings.Cut(string(out), "child home=")
		home, _, _ = strings.Cut(home, "\n")
		return strings.TrimSpace(home)
	}
	// Only what an allowlist typically keeps: no HOME, TEMP or TMP.
	var reduced []string
	for _, name := range []string{"PATH", "SystemRoot"} {
		if value, ok := os.LookupEnv(name); ok {
			reduced = append(reduced, name+"="+value)
		}
	}
	if home := run(reduced); home != "" {
		t.Fatalf("child with a reduced environment created home %q", home)
	}
	if home := run(os.Environ()); home != os.Getenv("HOME") {
		t.Fatalf("child home = %q, want the inherited %q", home, os.Getenv("HOME"))
	}
}

// TestIsolatedHomeIsSharedAndStable creates the home concurrently, as
// parallel test binaries do, in a private temporary directory.
func TestIsolatedHomeIsSharedAndStable(t *testing.T) {
	temp := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, temp)
	}
	homes := make(chan string, 8)
	errs := make(chan error, 8)
	for range 8 {
		go func() {
			home, err := isolatedHome()
			homes <- home
			errs <- err
		}()
	}
	first := ""
	for range 8 {
		home, err := <-homes, <-errs
		if err != nil {
			t.Fatal(err)
		}
		if first == "" {
			first = home
		} else if home != first {
			t.Fatalf("isolated homes differ: %q and %q", first, home)
		}
	}
	if data, err := os.ReadFile(filepath.Join(first, ".gitconfig")); err != nil || string(data) != GlobalConfig {
		t.Fatalf("isolated git config = %q, %v", data, err)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(first, ".gitconfig-*")); len(leftovers) != 0 {
		t.Fatalf("temporary config files left behind: %v", leftovers)
	}
}

func TestReExecutedChildHome(t *testing.T) {
	if os.Getenv("MARKITECT_TESTKIT_CHILD") == "1" {
		fmt.Printf("child home=%s\n", os.Getenv("HOME"))
	}
}

// TestGoLocationsMatchTheGoCommand compares the locations Isolate pins with
// those the go command derives in a fresh environment.
func TestGoLocationsMatchTheGoCommand(t *testing.T) {
	// The go command may leave a background process, such as its telemetry,
	// writing below the fresh home; TempDir retries the cleanup.
	home := TempDir(t)
	for _, name := range []string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOENV", "XDG_CACHE_HOME", "XDG_CONFIG_HOME"} {
		t.Setenv(name, "")
	}
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
		t.Setenv("LOCALAPPDATA", filepath.Join(home, "local"))
		t.Setenv("APPDATA", filepath.Join(home, "roaming"))
	}
	envFile := filepath.Join(home, "go.env")
	if err := os.WriteFile(envFile, []byte("GOPATH="+filepath.Join(home, "gopath")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", envFile)
	cmd := exec.Command("go", "env", "-json", "GOCACHE", "GOMODCACHE", "GOPATH", "GOENV")
	cmd.Dir = TempDir(t)
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("go command unavailable: %v", err)
	}
	want := map[string]string{}
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatal(err)
	}
	if got := goLocations(); !maps.Equal(got, want) {
		t.Fatalf("goLocations = %v, go env = %v", got, want)
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
	t.Setenv(HomeVariable, "")

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
