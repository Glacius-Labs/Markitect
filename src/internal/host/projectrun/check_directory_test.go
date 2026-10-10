package projectrun

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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
