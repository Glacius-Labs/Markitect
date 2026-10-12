package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"

import (
	"context"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	checkDescendantHelperEnv = "MARKITECT_PROJECTRUN_CHECK_DESCENDANT"
	modelledFileCheckEnv     = "MARKITECT_PROJECTRUN_CHECK_MODELLED_FILE"
	legacyOrdersBytes        = "legacy orders bytes\n"
)

func assertChecksObservedLegacyFile(t *testing.T, checks []CheckResult) {
	t.Helper()
	if len(checks) == 0 {
		t.Fatal("no declared check ran")
	}
	for _, check := range checks {
		if check.Outcome != "passed" || !strings.Contains(check.Stdout, "observed "+fixtureLegacyOrdersFile) {
			t.Fatalf("check %s did not observe the modelled file: %+v", check.ID, check)
		}
	}
}

// TestModelledFileCheckProcess is a declared check that passes only when the
// check tree holds the legacy orders file with its committed bytes.
func TestModelledFileCheckProcess(t *testing.T) {
	if os.Getenv(modelledFileCheckEnv) != "1" {
		return
	}
	content, err := os.ReadFile(fixtureLegacyOrdersFile)
	if err != nil || string(content) != legacyOrdersBytes {
		processExit(1, fmt.Sprintf("modelled file check failed: content=%q error=%v", content, err))
	}
	fmt.Println("observed " + fixtureLegacyOrdersFile)
	processExit(0, "")
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
	base := &snapshot.Snapshot{
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
	agent := projectconfig.Agent{Timeout: projectconfig.Duration(time.Second), Environment: []string{checkDescendantHelperEnv, "PATH", "SystemRoot"}}
	started := time.Now()
	result := runCheck(context.Background(), t.TempDir(), check, agent, projectconfig.Duration(time.Minute), nil)
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
