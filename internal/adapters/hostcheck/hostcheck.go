// Package hostcheck runs a project's required checks on the user's machine, in a
// clean folder, each with sh -c. They run code the agent wrote with the user's
// rights, as the agent's own shell already does (draft concept 4.0).
package hostcheck

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// maxOutput bounds how much of a check's output a result keeps.
const maxOutput = 8 << 10

// Runner runs required checks.
type Runner struct{}

var _ run.Checks = Runner{}

// Run runs the checks in a clean folder, stopping at the first that does not pass.
func (Runner) Run(ctx context.Context, dir string, checks []project.Check) []run.CheckResult {
	sh, err := shell()
	results := make([]run.CheckResult, 0, len(checks))
	for _, c := range checks {
		if err != nil {
			return append(results, run.CheckResult{Name: c.Name, NotRun: true, Output: err.Error()})
		}
		res := one(ctx, sh, dir, c)
		results = append(results, res)
		if !res.Passed {
			break
		}
	}
	return results
}

func one(ctx context.Context, sh, dir string, c project.Check) run.CheckResult {
	res := run.CheckResult{Name: c.Name}
	checkCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	cmd := exec.CommandContext(checkCtx, sh, "-c", c.Command)
	cmd.Dir = filepath.Join(dir, filepath.FromSlash(c.Directory))
	if info, err := os.Stat(cmd.Dir); err != nil || !info.IsDir() {
		// A folder the work has not made yet is the work's, as in a new project.
		res.Exit = 1
		res.Output = fmt.Sprintf("the folder %s does not exist\n", c.Directory)
		return res
	}
	cmd.Env = environment()
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	cmd.WaitDelay = 5 * time.Second
	cmd.Cancel = func() error { return kill(cmd) }
	start := time.Now()
	err := cmd.Run()
	res.Seconds = time.Since(start).Round(10 * time.Millisecond).Seconds()
	res.Output = lastPart(out.String())
	res.TimedOut = errors.Is(checkCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil
	var exit *exec.ExitError
	switch {
	// A check that exits but leaves a child holding its output, as a build server
	// does, has passed.
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
		res.Exit, res.Passed = 0, true
	case res.TimedOut:
		res.Output += "\n[the check timed out]\n"
	case errors.As(err, &exit):
		res.Exit = exit.ExitCode()
		// 126 and 127 are the shell's: a program that cannot run or is not there.
		// That is the environment's failure only when the check's own first word is
		// missing; a program in the repository, or one a script calls, is the work's.
		res.NotRun = (res.Exit == 126 || res.Exit == 127) && !inRepository(c.Command) && !available(sh, cmd.Dir, c.Command)
	default:
		res.NotRun = true
		res.Output += "\n[the check did not start: " + err.Error() + "]\n"
	}
	return res
}

// kill ends a check with what it started. On Windows, killing the shell alone
// leaves its children running and holding the output open.
func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run(); err == nil {
			return nil
		}
	}
	return cmd.Process.Kill()
}

// environment is the user's without what a session left in it, so that code
// under test sees none of a session's or Git's own variables.
func environment() []string {
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(name)
		if strings.HasPrefix(upper, "CLAUDE") || strings.HasPrefix(upper, "ANTHROPIC") || strings.HasPrefix(upper, "GIT_") {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// shell finds sh: on the path, else the one that ships with Git for Windows.
func shell() (string, error) {
	if sh, err := exec.LookPath("sh"); err == nil {
		return sh, nil
	}
	if out, err := exec.Command("git", "--exec-path").Output(); err == nil {
		// <root>/mingw64/libexec/git-core or <root>/libexec/git-core
		dir := filepath.FromSlash(strings.TrimSpace(string(out)))
		for i := 0; i < 4; i++ {
			dir = filepath.Dir(dir)
			sh := filepath.Join(dir, "bin", "sh.exe")
			if _, err := os.Stat(sh); err == nil {
				return sh, nil
			}
		}
	}
	return "", fmt.Errorf("no sh found to run the required checks; install Git for Windows or put sh on the path")
}

// available says whether the shell knows the first word of a command.
func available(sh, dir, command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	probe := exec.Command(sh, "-c", `command -v "$0"`, fields[0])
	probe.Dir = dir
	probe.Env = environment()
	return probe.Run() == nil
}

// inRepository says whether a check's command starts with a program given by a
// relative path.
func inRepository(command string) bool {
	fields := strings.Fields(command)
	return len(fields) > 0 && strings.Contains(fields[0], "/") && !strings.HasPrefix(fields[0], "/")
}

func lastPart(s string) string {
	if len(s) > maxOutput {
		return "…" + s[len(s)-maxOutput:]
	}
	return s
}
