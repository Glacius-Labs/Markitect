package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func TestPlanVerifyCommandsUsesFixedSnapshotBootstrapPair(t *testing.T) {
	historical, err := planVerifyCommands("konfyra", map[string][]byte{
		"scripts/render-governance-adapters.py": nil,
		"scripts/tests/test_example.py":         nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(historical) != 2 || historical[0].tool != "python" || historical[1].tool != "python" {
		t.Fatalf("historical Python snapshot unexpectedly requires Go evidence: %#v", historical)
	}

	withGo, err := planVerifyCommands("konfyra", map[string][]byte{
		"scripts/run-markitect.go":            nil,
		"scripts/markitect-bootstrap_test.go": nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(withGo) != 3 || withGo[2].tool != "go" || strings.Join(withGo[2].args, " ") != "test -count=1 -v scripts/run-markitect.go scripts/markitect-bootstrap_test.go" {
		t.Fatalf("Go bootstrap gate was not added accurately: %#v", withGo)
	}
	for _, profile := range []string{"konfyra", "cockpit"} {
		_, err := planVerifyCommands(profile, map[string][]byte{"scripts/run-markitect.go": nil})
		var verifyErr *VerifyError
		if !errors.As(err, &verifyErr) || verifyErr.Kind != "incomplete-evidence" || verifyErr.Profile != profile {
			t.Fatalf("incomplete bootstrap pair was not rejected for %s: %v", profile, err)
		}
	}
}

func TestPlanVerifyCommandsUsesCockpitGoPairOrLegacyPython(t *testing.T) {
	legacy, err := planVerifyCommands("cockpit", map[string][]byte{
		"scripts/check_docs.py":      nil,
		"scripts/render_adapters.py": nil,
	})
	if err != nil || len(legacy) != 2 || legacy[0].tool != "python" || legacy[1].tool != "python" {
		t.Fatalf("historical Cockpit snapshot did not retain Python gates: commands=%#v error=%v", legacy, err)
	}

	native, err := planVerifyCommands("cockpit", map[string][]byte{
		"scripts/check_docs.py":         nil,
		"scripts/render_adapters.py":    nil,
		"scripts/check-cockpit.go":      nil,
		"scripts/check-cockpit_test.go": nil,
	})
	if err != nil || len(native) != 2 {
		t.Fatalf("native Cockpit pair did not replace Python gates: commands=%#v error=%v", native, err)
	}
	if native[0].tool != "go" || strings.Join(native[0].args, " ") != "run scripts/check-cockpit.go" ||
		native[1].tool != "go" || strings.Join(native[1].args, " ") != "test -count=1 -v scripts/check-cockpit.go scripts/check-cockpit_test.go" {
		t.Fatalf("native Cockpit commands were not planned exactly: %#v", native)
	}
	nativeWithBootstrap, err := planVerifyCommands("cockpit", map[string][]byte{
		"scripts/check-cockpit.go":            nil,
		"scripts/check-cockpit_test.go":       nil,
		"scripts/run-markitect.go":            nil,
		"scripts/markitect-bootstrap_test.go": nil,
	})
	if err != nil || len(nativeWithBootstrap) != 3 || nativeWithBootstrap[2].tool != "go" ||
		strings.Join(nativeWithBootstrap[2].args, " ") != "test -count=1 -v scripts/run-markitect.go scripts/markitect-bootstrap_test.go" {
		t.Fatalf("Cockpit bootstrap pair was not checked as well: commands=%#v error=%v", nativeWithBootstrap, err)
	}

	_, err = planVerifyCommands("cockpit", map[string][]byte{"scripts/check-cockpit.go": nil})
	var verifyErr *VerifyError
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "incomplete-evidence" || verifyErr.Profile != "cockpit" {
		t.Fatalf("half Cockpit Go pair fell back instead of failing closed: %v", err)
	}
}

func TestVerifyRunsBootstrapTestFromMaterializedSnapshot(t *testing.T) {
	_, err := exec.LookPath("python")
	if err != nil {
		_, err = exec.LookPath("python3")
	}
	if err != nil {
		t.Skip("Python unavailable for existing Konfyra gates")
	}
	goEnvFile := filepath.Join(t.TempDir(), "go.env")
	if err := os.WriteFile(goEnvFile, []byte("GOFLAGS=-run=^DefinitelyNoSuchTest$\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOFLAGS", "-run=^DefinitelyNoSuchTest$")
	t.Setenv("GOENV", goEnvFile)
	// A nonexistent absolute workspace would make an unsanitized go test fail.
	t.Setenv("GOWORK", filepath.Join(t.TempDir(), "ambient.work"))
	files := map[string][]byte{
		"scripts/render-governance-adapters.py": []byte("print('adapter check from snapshot')\n"),
		"scripts/tests/test_snapshot.py":        []byte("import unittest\nclass SnapshotTest(unittest.TestCase):\n def test_snapshot(self): self.assertTrue(True)\n"),
		"scripts/run-markitect.go":              []byte("package main\nfunc main() {}\n"),
		"scripts/markitect-bootstrap_test.go":   []byte("package main\nimport \"testing\"\nfunc TestSnapshotBootstrap(t *testing.T) { t.Log(\"bootstrap from fixed snapshot\") }\n"),
	}
	p := konfyraVerifyProject(files)
	results, err := verifyRepositoryWithTimeout(p, 2*time.Minute)
	if err != nil {
		t.Fatalf("verification failed: %v; results=%#v", err, results)
	}
	if len(results) != 3 || results[2].Profile != "konfyra" || results[2].Tool != "go" || results[2].ExitCode != 0 || !strings.Contains(results[2].Output, "bootstrap from fixed snapshot") {
		t.Fatalf("Go snapshot gate missing or inaccurate: %#v", results)
	}
}

func TestVerifyRunsCockpitGoGatesFromMaterializedSnapshot(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go unavailable for native Cockpit gates")
	}
	files := map[string][]byte{
		"scripts/check-cockpit.go":      []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"Cockpit checker from fixed snapshot\") }\n"),
		"scripts/check-cockpit_test.go": []byte("package main\nimport \"testing\"\nfunc TestCockpitSnapshot(t *testing.T) { t.Log(\"Cockpit paired test from fixed snapshot\") }\n"),
	}
	results, err := verifyRepositoryWithTimeout(cockpitVerifyProject(files), 2*time.Minute)
	if err != nil {
		t.Fatalf("native Cockpit verification failed: %v; results=%#v", err, results)
	}
	if len(results) != 2 || results[0].Tool != "go" || !strings.Contains(results[0].Output, "Cockpit checker from fixed snapshot") ||
		results[1].Tool != "go" || !strings.Contains(results[1].Output, "Cockpit paired test from fixed snapshot") {
		t.Fatalf("native Cockpit commands did not run from the snapshot: %#v", results)
	}
}

func TestVerifyCommandSanitizesGoEnvironmentForEveryGate(t *testing.T) {
	goEnvFile := filepath.Join(t.TempDir(), "go.env")
	if err := os.WriteFile(goEnvFile, []byte("GOFLAGS=-run=^DefinitelyNoSuchTest$\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOFLAGS", "-run=^DefinitelyNoSuchTest$")
	t.Setenv("GOENV", goEnvFile)
	t.Setenv("GOWORK", filepath.Join(t.TempDir(), "ambient.work"))
	result, err := runVerifyCommand(verifyCommand{
		profile: "konfyra", name: "nested Go environment", tool: "python",
		args: []string{"-test.run=^TestVerifyCommandHelper$"},
		env:  []string{"MARKITECT_VERIFY_HELPER=go-env"},
	}, mustTestExecutable(t), t.TempDir(), time.Second)
	if err != nil || !strings.Contains(result.Output, "GOFLAGS= GOENV=off GOWORK=off") {
		t.Fatalf("gate did not receive isolated Go environment: result=%#v error=%v", result, err)
	}
}

func TestVerifyCommandStartFailureIsIncompleteEvidence(t *testing.T) {
	result, err := runVerifyCommand(verifyCommand{profile: "test", name: "missing command", tool: "test"}, filepath.Join(t.TempDir(), "missing-tool"), t.TempDir(), time.Second)
	var verifyErr *VerifyError
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "incomplete-evidence" || result.ExitCode != -1 {
		t.Fatalf("process start failure was reported as a gate result: result=%#v error=%v", result, err)
	}
}

func TestVerifyRejectsProvisionalSnapshotAndIncompleteGoPair(t *testing.T) {
	p := konfyraVerifyProject(map[string][]byte{"scripts/run-markitect.go": []byte("package main\n")})
	_, err := verifyRepositoryWithTimeout(p, time.Second)
	var verifyErr *VerifyError
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "incomplete-evidence" {
		t.Fatalf("incomplete Go pair was not reported: %v", err)
	}
	p.Snapshot.Provisional = true
	_, err = VerifyRepository(p)
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "incomplete-evidence" {
		t.Fatalf("provisional snapshot was accepted: %v", err)
	}
}

func TestVerifyDistinguishesMissingToolFromGateFailure(t *testing.T) {
	pathDir := t.TempDir()
	t.Setenv("PATH", pathDir)
	p := konfyraVerifyProject(map[string][]byte{
		"scripts/render-governance-adapters.py": []byte("pass\n"),
		"scripts/tests/test_snapshot.py":        []byte("pass\n"),
	})
	_, err := verifyRepositoryWithTimeout(p, time.Second)
	var verifyErr *VerifyError
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "tool-missing" || verifyErr.Gate != "scripts/render-governance-adapters.py --check" {
		t.Fatalf("missing tool was not distinguished from gate failure: %v", err)
	}
}

func TestVerifyCommandBoundsOutputTimeoutAndExitFailure(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	base := verifyCommand{profile: "test", tool: "test", args: []string{"-test.run=^TestVerifyCommandHelper$"}}

	large := base
	large.name = "output overflow"
	large.env = []string{"MARKITECT_VERIFY_HELPER=large"}
	result, err := runVerifyCommand(large, executable, dir, time.Second)
	var verifyErr *VerifyError
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "output-limit" || len(result.Output) != verifyOutputLimit {
		t.Fatalf("output was not capped and cancelled: output=%d error=%v", len(result.Output), err)
	}

	slow := base
	slow.name = "timeout"
	slow.env = []string{"MARKITECT_VERIFY_HELPER=sleep"}
	result, err = runVerifyCommand(slow, executable, dir, 100*time.Millisecond)
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "timeout" || result.ExitCode != -1 {
		t.Fatalf("timeout was not enforced: result=%#v error=%v", result, err)
	}

	failure := base
	failure.name = "nonzero exit"
	failure.env = []string{"MARKITECT_VERIFY_HELPER=exit"}
	result, err = runVerifyCommand(failure, executable, dir, time.Second)
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "gate-failure" || result.ExitCode != 7 {
		t.Fatalf("nonzero exit did not propagate as a gate failure: result=%#v error=%v", result, err)
	}

	killed := base
	killed.name = "self-terminated process"
	killed.env = []string{"MARKITECT_VERIFY_HELPER=self-kill"}
	result, err = runVerifyCommand(killed, executable, dir, time.Second)
	if !errors.As(err, &verifyErr) || verifyErr.Kind != "gate-failure" || result.ExitCode == 0 {
		t.Fatalf("self-terminated child was not retained as a gate failure: result=%#v error=%v", result, err)
	}
}

func TestVerifyCommandRemovesInheritedGitRepositoryEnvironment(t *testing.T) {
	t.Setenv("GIT_DIR", t.TempDir())
	t.Setenv("GIT_WORK_TREE", t.TempDir())
	t.Setenv("GIT_INDEX_FILE", "foreign-index")
	result, err := runVerifyCommand(verifyCommand{
		profile: "test", name: "git environment isolation", tool: "test",
		args: []string{"-test.run=^TestVerifyCommandHelper$"},
		env:  []string{"MARKITECT_VERIFY_HELPER=git-env"},
	}, mustTestExecutable(t), t.TempDir(), time.Second)
	if err != nil || !strings.Contains(result.Output, "git environment clean") {
		t.Fatalf("gate inherited a foreign Git repository environment: %#v, %v", result, err)
	}
}

func TestVerifyCommandHelper(t *testing.T) {
	switch os.Getenv("MARKITECT_VERIFY_HELPER") {
	case "large":
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", verifyOutputLimit+1)))
		return
	case "sleep":
		time.Sleep(30 * time.Second)
		return
	case "exit":
		_, _ = fmt.Fprint(os.Stdout, "failing gate output")
		os.Exit(7)
	case "git-env":
		for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE"} {
			if os.Getenv(key) != "" {
				os.Exit(9)
			}
		}
		_, _ = fmt.Fprintln(os.Stdout, "git environment clean")
	case "go-env":
		_, _ = fmt.Fprintf(os.Stdout, "GOFLAGS=%s GOENV=%s GOWORK=%s\n", os.Getenv("GOFLAGS"), os.Getenv("GOENV"), os.Getenv("GOWORK"))
		if os.Getenv("GOFLAGS") != "" || os.Getenv("GOENV") != "off" || os.Getenv("GOWORK") != "off" {
			os.Exit(10)
		}
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

func konfyraVerifyProject(files map[string][]byte) *Project {
	modes := make(map[string]string, len(files))
	for path := range files {
		modes[path] = "100644"
	}
	snapshot := &source.Snapshot{Revision: "fixed-test-revision", Files: files, Modes: modes}
	project := &core.Resource{Kind: "Project", Metadata: core.Metadata{Name: "test"}, Path: "markitect.yaml", Spec: core.Spec{Profile: "konfyra"}}
	return &Project{Snapshot: snapshot, Graph: &core.Graph{Project: project}}
}

func cockpitVerifyProject(files map[string][]byte) *Project {
	modes := make(map[string]string, len(files))
	for path := range files {
		modes[path] = "100644"
	}
	snapshot := &source.Snapshot{Revision: "fixed-test-revision", Files: files, Modes: modes}
	project := &core.Resource{Kind: "Project", Metadata: core.Metadata{Name: "test"}, Path: "markitect.yaml", Spec: core.Spec{Profile: "cockpit"}}
	return &Project{Snapshot: snapshot, Graph: &core.Graph{Project: project}}
}

func mustTestExecutable(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return path
}
