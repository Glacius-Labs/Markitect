package host

import (
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

func TestPlanVerifyCommandsRequiresExplicitNamedArgvChecks(t *testing.T) {
	_, err := planVerifyCommands(nil)
	var verifyErr *VerifyError
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "incomplete-evidence" || !strings.Contains(err.Error(), "graph-only") {
		t.Fatalf("missing checks did not fail as incomplete graph-only evidence: %v", err)
	}

	for name, checks := range map[string][]authoring.Check{
		"blank name":          {{Run: []string{"go", "version"}}},
		"empty argv":          {{Name: "empty", Run: nil}},
		"blank executable":    {{Name: "blank", Run: []string{"  "}}},
		"relative executable": {{Name: "relative", Run: []string{"scripts/check"}}},
		"absolute executable": {{Name: "absolute", Run: []string{filepath.Join(t.TempDir(), "tool")}}},
		"duplicate names":     {{Name: "same", Run: []string{"go", "version"}}, {Name: "same", Run: []string{"go", "env"}}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := planVerifyCommands(checks)
			if !errors.As(err, &verifyErr) || verifyErr.Kind != "incomplete-evidence" {
				t.Fatalf("invalid check was accepted: %v", err)
			}
		})
	}
}

func TestPlanVerifyCommandsPreservesDeclaredArgv(t *testing.T) {
	checks := []authoring.Check{{Name: "literal-argv", Run: []string{"go", "test", "-run", "literal;$(text)"}}}
	commands, err := planVerifyCommands(checks)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].name != "literal-argv" || commands[0].tool != "go" || strings.Join(commands[0].args, "\x00") != "test\x00-run\x00literal;$(text)" {
		t.Fatalf("check argv was reinterpreted or changed: %#v", commands)
	}
}

func TestVerifyUsesEachCheckTimeoutAndRecordsEffectiveBound(t *testing.T) {
	executable := mustTestExecutable(t)
	t.Setenv("PATH", filepath.Dir(executable))
	t.Setenv("MARKITECT_VERIFY_HELPER", "short-sleep")
	// The explicit bound must not expire, even when a race build starts the
	// helper slowly; only the 50ms fallback must.
	seconds := 60
	argv := []string{filepath.Base(executable), "-test.run=^TestVerifyCommandHelper$"}
	checks := []authoring.Check{
		{Name: "explicit", Run: argv, TimeoutSeconds: &seconds},
		{Name: "fallback", Run: argv},
	}
	results, err := verifyRepositoryWithTimeout(verifyProject(nil, checks), 50*time.Millisecond)
	var verifyErr *VerifyError
	if len(results) != 2 || results[0].ExitCode != 0 || results[0].TimeoutMilliseconds != 60000 || results[1].ExitCode != -1 || results[1].TimeoutMilliseconds != 50 || !errors.As(err, &verifyErr) || verifyErr.Kind != "timeout" || verifyErr.Gate != "fallback" || !strings.Contains(err.Error(), "50ms execution limit") {
		t.Fatalf("configured/fallback bounds or incomplete timeout classification changed: results=%#v error=%v", results, err)
	}
	maximum := 5400
	checks[0].TimeoutSeconds = &maximum
	commands, err := planVerifyCommands(checks)
	if err != nil || commands[0].timeout != 90*time.Minute || commands[1].timeout != 0 || verifyDefaultTime != 10*time.Minute {
		t.Fatalf("explicit maximum or unchanged default lost: %#v error=%v", commands, err)
	}
	maximum++
	checks[0].TimeoutSeconds = &maximum
	if _, err := planVerifyCommands(checks); err == nil || !strings.Contains(err.Error(), "between 1 and 5400") {
		t.Fatalf("timeout above the configured maximum was accepted: %v", err)
	}
}

func TestVerifyRunsCheckAgainstMaterializedSnapshot(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go unavailable for the portable snapshot check")
	}
	checkSource := []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"fixed snapshot marker\") }\n")
	files := map[string][]byte{"scripts/verify-check.go": checkSource}
	project := verifyProject(files, []authoring.Check{{Name: "snapshot-check", Run: []string{"go", "run", "scripts/verify-check.go"}}})
	results, err := verifyRepositoryWithTimeout(project, 2*time.Minute)
	if err != nil {
		t.Fatalf("snapshot check failed: %v; results=%#v", err, results)
	}
	if len(results) != 1 || results[0].Name != "snapshot-check" || results[0].Tool != "go" || results[0].ExitCode != 0 || !strings.Contains(results[0].Output, "fixed snapshot marker") {
		t.Fatalf("declared check did not run with the fixed snapshot bytes: %#v", results)
	}
}

func TestVerifyRejectsProvisionalSnapshotAndMissingChecks(t *testing.T) {
	p := verifyProject(nil, nil)
	_, err := verifyRepositoryWithTimeout(p, time.Second)
	var verifyErr *VerifyError
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "incomplete-evidence" || !strings.Contains(err.Error(), "graph-only") {
		t.Fatalf("empty check list was not reported as incomplete: %v", err)
	}
	p.Graph.Project.Spec.Checks = []authoring.Check{{Name: "simple", Run: []string{"go", "version"}}}
	p.Snapshot.Provisional = true
	_, err = VerifyRepository(p)
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "incomplete-evidence" {
		t.Fatalf("provisional snapshot was accepted: %v", err)
	}
}

func TestVerifyDistinguishesMissingToolFromGateFailure(t *testing.T) {
	pathDir := t.TempDir()
	t.Setenv("PATH", pathDir)
	p := verifyProject(nil, []authoring.Check{{Name: "unknown-command", Run: []string{"markitect-no-such-check-tool-20260930"}}})
	results, err := verifyRepositoryWithTimeout(p, time.Second)
	var verifyErr *VerifyError
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "tool-missing" || verifyErr.Gate != "unknown-command" || len(results) != 0 {
		t.Fatalf("missing tool was not distinguished from gate failure: results=%#v error=%v", results, err)
	}
}

func TestFindVerifyToolRejectsRelativePATHResolution(t *testing.T) {
	toolDir := t.TempDir()
	t.Chdir(toolDir)
	toolName := "markitect-relative-path-probe.exe"
	if err := os.WriteFile(filepath.Join(toolDir, toolName), []byte("not executed"), 0755); err != nil {
		t.Fatal(err)
	}
	// Keep the probe on one volume even when the checkout and TEMP differ.
	// Markitect must reject relative resolution even if Go's ErrDot guard is off.
	t.Setenv("PATH", ".")
	t.Setenv("GODEBUG", "execerrdot=0")
	resolved, err := findVerifyTool(toolName)
	if err == nil || filepath.IsAbs(resolved) {
		t.Fatalf("relative PATH entry was accepted: resolved=%q error=%v", resolved, err)
	}
}

func TestVerifyCommandBoundsOutputTimeoutAndExitFailure(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	base := verifyCommand{name: "helper", tool: "test", args: []string{"-test.run=^TestVerifyCommandHelper$"}}

	t.Setenv("MARKITECT_VERIFY_HELPER", "large")
	result, err := runVerifyCommand(base, executable, dir, time.Second)
	var verifyErr *VerifyError
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "output-limit" || len(result.Output) != verifyOutputLimit {
		t.Fatalf("output was not capped and cancelled: output=%d error=%v", len(result.Output), err)
	}

	t.Setenv("MARKITECT_VERIFY_HELPER", "sleep")
	result, err = runVerifyCommand(base, executable, dir, 100*time.Millisecond)
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "timeout" || result.ExitCode != -1 {
		t.Fatalf("timeout was not enforced: result=%#v error=%v", result, err)
	}

	t.Setenv("MARKITECT_VERIFY_HELPER", "exit")
	result, err = runVerifyCommand(base, executable, dir, time.Second)
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "gate-failure" || result.ExitCode != 7 || !strings.Contains(result.Output, "failing check output") {
		t.Fatalf("nonzero exit did not propagate as a check failure: result=%#v error=%v", result, err)
	}

	t.Setenv("MARKITECT_VERIFY_HELPER", "self-kill")
	result, err = runVerifyCommand(base, executable, dir, time.Second)
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "gate-failure" || result.ExitCode == 0 {
		t.Fatalf("self-terminated child was not retained as a check failure: result=%#v error=%v", result, err)
	}
}

func TestVerifyCommandSanitizesGitAndGoEnvironment(t *testing.T) {
	goEnvFile := filepath.Join(t.TempDir(), "go.env")
	if err := os.WriteFile(goEnvFile, []byte("GOFLAGS=-run=^DefinitelyNoSuchTest$\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOFLAGS", "-run=^DefinitelyNoSuchTest$")
	t.Setenv("GOENV", goEnvFile)
	t.Setenv("GOWORK", filepath.Join(t.TempDir(), "ambient.work"))
	t.Setenv("GIT_DIR", t.TempDir())
	t.Setenv("GIT_WORK_TREE", t.TempDir())
	t.Setenv("GIT_INDEX_FILE", "foreign-index")
	t.Setenv("MARKITECT_VERIFY_HELPER", "env")
	// Generous, because this test does not test time.
	result, err := runVerifyCommand(verifyCommand{name: "environment isolation", tool: "test", args: []string{"-test.run=^TestVerifyCommandHelper$"}}, mustTestExecutable(t), t.TempDir(), time.Minute)
	if err != nil || !strings.Contains(result.Output, "GOFLAGS= GOENV=off GOWORK=off") || !strings.Contains(result.Output, "git environment clean") {
		t.Fatalf("check inherited forbidden Git/Go environment: result=%#v error=%v", result, err)
	}
}

func TestVerifyCommandHelper(t *testing.T) {
	switch os.Getenv("MARKITECT_VERIFY_HELPER") {
	case "short-sleep":
		time.Sleep(150 * time.Millisecond)
	case "large":
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", verifyOutputLimit+1)))
	case "sleep":
		time.Sleep(30 * time.Second)
	case "exit":
		_, _ = fmt.Fprint(os.Stdout, "failing check output")
		os.Exit(7)
	case "env":
		for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE"} {
			if os.Getenv(key) != "" {
				os.Exit(9)
			}
		}
		_, _ = fmt.Fprintf(os.Stdout, "GOFLAGS=%s GOENV=%s GOWORK=%s\n", os.Getenv("GOFLAGS"), os.Getenv("GOENV"), os.Getenv("GOWORK"))
		if os.Getenv("GOFLAGS") != "" || os.Getenv("GOENV") != "off" || os.Getenv("GOWORK") != "off" {
			os.Exit(10)
		}
		_, _ = fmt.Fprintln(os.Stdout, "git environment clean")
	case "self-kill":
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			os.Exit(11)
		}
		if err := process.Kill(); err != nil {
			os.Exit(12)
		}
	}
}

func verifyProject(files map[string][]byte, checks []authoring.Check) *Project {
	modes := make(map[string]string, len(files))
	for name := range files {
		modes[name] = "100644"
	}
	snapshot := &snapshot.Snapshot{ID: "fixed-test-revision", Files: files, Modes: modes}
	project := &authoring.Resource{Core: authoring.Core{Kind: "Project", Metadata: core.Metadata{Name: "verification-fixture"}, Path: "markitect.yaml"}, Spec: authoring.Spec{Checks: checks}}
	return &Project{Snapshot: snapshot, Graph: &authoring.Graph{Project: project}}
}

func mustTestExecutable(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return path
}
