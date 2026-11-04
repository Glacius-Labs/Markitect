package hostcheck_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/hostcheck"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

func check(command string) project.Check {
	return project.Check{Name: "c", Command: command, Directory: ".", Timeout: time.Minute}
}

func TestPassFailAndWhatCannotStart(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	run := func(c project.Check) (passed, notRun bool, output string) {
		t.Helper()
		res := hostcheck.Runner{}.Run(ctx, dir, []project.Check{c})
		if len(res) != 1 {
			t.Fatalf("%+v", res)
		}
		return res[0].Passed, res[0].NotRun, res[0].Output
	}
	if passed, _, out := run(check("echo hello")); !passed || !strings.Contains(out, "hello") {
		t.Fatalf("%v %q", passed, out)
	}
	if passed, notRun, _ := run(check("exit 3")); passed || notRun {
		t.Fatalf("a failing check: %v %v", passed, notRun)
	}
	if _, notRun, _ := run(check("no-such-program-at-all")); !notRun {
		t.Fatal("a missing program is the environment's failure")
	}
	if passed, notRun, _ := run(check("./bin/no-such-script")); passed || notRun {
		t.Fatalf("a missing program of the repository is the work's failure: %v %v", passed, notRun)
	}
}

// A folder the work has not made yet is a failure of the work, as in a new
// project, not of the environment.
func TestMissingFolderIsTheWorks(t *testing.T) {
	c := check("echo hello")
	c.Directory = "not/there"
	res := hostcheck.Runner{}.Run(context.Background(), t.TempDir(), []project.Check{c})
	if len(res) != 1 || res[0].Passed || res[0].NotRun {
		t.Fatalf("%+v", res)
	}
}

func TestStopsAtTheFirstFailure(t *testing.T) {
	res := hostcheck.Runner{}.Run(context.Background(), t.TempDir(), []project.Check{check("exit 1"), check("echo second")})
	if len(res) != 1 {
		t.Fatalf("%+v", res)
	}
}

func TestTimeoutIsAFailure(t *testing.T) {
	c := check("sleep 30")
	c.Timeout = 200 * time.Millisecond
	start := time.Now()
	res := hostcheck.Runner{}.Run(context.Background(), t.TempDir(), []project.Check{c})
	if len(res) != 1 || res[0].Passed || res[0].NotRun || !res[0].TimedOut {
		t.Fatalf("%+v", res)
	}
	if took := time.Since(start); took > 20*time.Second {
		t.Fatalf("took %s", took)
	}
}

// Code under test sees nothing a session left in the environment.
func TestEnvironmentIsClean(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SESSION_ID", "secret-session")
	t.Setenv("ANTHROPIC_API_KEY", "secret-key")
	t.Setenv("GIT_DIR", "/somewhere")
	dir := t.TempDir()
	res := hostcheck.Runner{}.Run(context.Background(), dir, []project.Check{check("env > env.txt")})
	if len(res) != 1 || !res[0].Passed {
		t.Fatalf("%+v", res)
	}
	data, err := os.ReadFile(filepath.Join(dir, "env.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"secret-session", "secret-key", "GIT_DIR"} {
		if strings.Contains(string(data), leak) {
			t.Fatalf("%s reached the check", leak)
		}
	}
}
