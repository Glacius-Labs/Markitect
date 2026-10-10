package projectrun

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const checkDescendantHelperEnv = "MARKITECT_PROJECTRUN_CHECK_DESCENDANT"

// Declared checks once ran in a materialized candidate under the repository's
// run store. Below a deep repository (or GOTMPDIR) that directory exceeded the
// Windows working-directory limit and every check failed with "The directory
// name is invalid". Checks now run in a fresh short temporary directory.
func TestVerifyRunsDeclaredChecksOutsideTheRunStore(t *testing.T) {
	root := makeProjectRunFixture(t)
	setupE2EProcess(t, "normal")
	plan := planBothManagers(t, root)
	run, err := Run(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || run.Status != StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	verified, err := Verify(context.Background(), projectworkHost(), ProcessInvoker{}, root, plan.ID)
	if err != nil || verified.Status != StatusVerified {
		t.Fatalf("Verify: report=%+v err=%v", verified, err)
	}
	cleanRoot := strings.ToLower(filepath.Clean(root))
	for _, check := range verified.Checks {
		_, cwd, found := strings.Cut(strings.TrimSpace(check.Stdout), "check cwd: ")
		if !found {
			t.Fatalf("check did not report its working directory: %+v", check)
		}
		if strings.HasPrefix(strings.ToLower(filepath.Clean(cwd)), cleanRoot) {
			t.Fatalf("check ran inside the repository run store: %s", cwd)
		}
	}
}

func TestCheckWorkingDirectoryLengthNamesTheWindowsLimit(t *testing.T) {
	short := strings.Repeat("d", maxWindowsWorkingDirectory)
	long := strings.Repeat("d", maxWindowsWorkingDirectory+1)
	if err := checkWorkingDirectoryLength(short); err != nil {
		t.Fatalf("a directory at the limit was rejected: %v", err)
	}
	err := checkWorkingDirectoryLength(long)
	if runtime.GOOS != "windows" {
		if err != nil {
			t.Fatalf("the Windows limit applied on %s: %v", runtime.GOOS, err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), "at most 258 characters") {
		t.Fatalf("over-long directory error = %v", err)
	}
}

// Candidate paths are checked lexically. Windows also resolves an existing
// entry through its 8.3 short name or another case, so MARKIT~1/project.yaml
// once overwrote .markitect/project.yaml in the copy that checks run in.
func TestMaterializeCandidateRefusesWindowsAliasesOfExistingEntries(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("short names and case aliases are resolved by Windows")
	}
	base := &Snapshot{
		Files: map[string][]byte{".markitect/project.yaml": []byte("project\n"), "Docs/guide.md": []byte("guide\n")},
		Modes: map[string]string{".markitect/project.yaml": "100644", "Docs/guide.md": "100644"},
	}
	materialize := func(t *testing.T, path string) (string, error) {
		t.Helper()
		dest := t.TempDir()
		// .markitect exists first, as when its files sort first, and so owns
		// the short name MARKIT~1 on a volume that generates 8.3 names.
		if err := os.Mkdir(filepath.Join(dest, ".markitect"), 0o700); err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(path, "MARKIT~1/") && !resolvesAsShortName(dest, ".markitect", "MARKIT~1") {
			t.Skip("volume generates no 8.3 short names")
		}
		candidate := candidateData{Files: map[string]File{path: {Path: path, Mode: "100644", Content: []byte("alias\n")}}}
		return dest, materializeCandidate(dest, base, candidate)
	}
	// Every NTFS volume resolves case variants, so only the 8.3 case skips.
	for _, tc := range []struct{ name, path string }{
		{"8.3 short name", "MARKIT~1/project.yaml"},
		{"case variant", "docs/guide.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dest, err := materialize(t, tc.path)
			if err == nil {
				t.Errorf("materializeCandidate accepted alias %s", tc.path)
			}
			for name, want := range base.Files {
				if got, readErr := os.ReadFile(filepath.Join(dest, filepath.FromSlash(name))); readErr != nil || string(got) != string(want) {
					t.Errorf("alias %s changed %s to %q (%v)", tc.path, name, got, readErr)
				}
			}
		})
	}
	// Stored names and new 8.3-shaped names stay writable.
	for _, path := range []string{"Docs/guide.md", "notes~1/new.md"} {
		dest, err := materialize(t, path)
		if err != nil {
			t.Fatalf("materializeCandidate refused %s: %v", path, err)
		}
		if got, readErr := os.ReadFile(filepath.Join(dest, filepath.FromSlash(path))); readErr != nil || string(got) != "alias\n" {
			t.Fatalf("%s = %q (%v)", path, got, readErr)
		}
	}
}

// resolvesAsShortName reports whether Windows resolves alias in dir to the
// existing entry long, which needs a volume that generates 8.3 short names.
func resolvesAsShortName(dir, long, alias string) bool {
	longInfo, longErr := os.Stat(filepath.Join(dir, long))
	aliasInfo, aliasErr := os.Stat(filepath.Join(dir, alias))
	return longErr == nil && aliasErr == nil && os.SameFile(longInfo, aliasInfo)
}

// A check once killed only its direct child on timeout and set no WaitDelay,
// so a descendant holding the inherited output pipes kept runCheck blocked
// past the check timeout (and forever if it never exited). The timeout now
// stops the whole process tree.
func TestCheckTimeoutStopsDescendantsHoldingOutput(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	resolved, raw, err := readPinnedExecutable(executable)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(checkDescendantHelperEnv, "spawn")
	check := CheckPlan{ID: "descendant-timeout", Owner: "orders", Required: true,
		Command:        []string{filepath.Base(executable), "-test.run=^TestCheckDescendantHelperProcess$"},
		ExecutablePath: resolved, ExecutableDigest: rawContentDigest(raw)}
	agent := Agent{Timeout: Duration(time.Second), Environment: []string{checkDescendantHelperEnv, "PATH", "SystemRoot"}}
	started := time.Now()
	result := runCheck(context.Background(), t.TempDir(), check, agent, Duration(time.Minute), nil)
	elapsed := time.Since(started)
	if result.Outcome != "failed" || result.Error != "check timed out" {
		t.Fatalf("timed-out check = outcome %q error %q, want failed / check timed out", result.Outcome, result.Error)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("runCheck with a 1s check timeout returned after %s; a descendant holding its output kept it blocked", elapsed.Round(10*time.Millisecond))
	}
	_, pidText, found := strings.Cut(strings.TrimSpace(result.Stdout), "spawned descendant ")
	pid, err := strconv.Atoi(pidText)
	if !found || err != nil {
		t.Fatalf("check did not report its descendant: stdout=%q stderr=%q", result.Stdout, result.Stderr)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		alive, err := processAlive(pid)
		if err != nil {
			t.Fatal(err)
		}
		if !alive {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("check descendant %d outlived the check timeout", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestCheckDescendantHelperProcess is the re-executed check for
// TestCheckTimeoutStopsDescendantsHoldingOutput. It is a no-op unless the
// helper environment selects a role.
func TestCheckDescendantHelperProcess(t *testing.T) {
	switch os.Getenv(checkDescendantHelperEnv) {
	case "spawn":
		executable, err := os.Executable()
		if err != nil {
			os.Exit(3)
		}
		descendant := exec.Command(executable, "-test.run=^TestCheckDescendantHelperProcess$")
		descendant.Env = append(os.Environ(), checkDescendantHelperEnv+"=descendant")
		descendant.Stdout = os.Stdout
		descendant.Stderr = os.Stderr
		if err := descendant.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "start descendant:", err)
			os.Exit(3)
		}
		fmt.Println("spawned descendant", descendant.Process.Pid)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "descendant":
		time.Sleep(8 * time.Second)
		os.Exit(0)
	}
}
