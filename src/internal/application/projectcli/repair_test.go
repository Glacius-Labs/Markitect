package projectcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repair no longer doubles as status: without --execute it is a usage error
// that names the read-only verb and starts or writes nothing.
func TestRepairWithoutExecuteIsAUsageErrorAndWritesNothing(t *testing.T) {
	repo := copyProjectWorld(t)
	code, out, errout := runCLI(t, "repair", "missing-run", "--repo", repo)
	if code != 2 || out != "" || !strings.Contains(errout, "markitect repair: repair starts agents or configured checks and requires --execute; inspect first with `markitect status`") {
		t.Fatalf("repair without --execute exit=%d stderr=%s stdout=%s", code, errout, out)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(".markitect/runs"))); !os.IsNotExist(err) {
		t.Fatalf("rejected repair created runtime state: %v", err)
	}
	code, out, errout = runCLI(t, "status", "missing-run", "--repo", repo)
	if code != 2 || out != "" || !strings.Contains(errout, "markitect status:") {
		t.Fatalf("status for a missing run exit=%d stderr=%s stdout=%s", code, errout, out)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(".markitect/runs"))); !os.IsNotExist(err) {
		t.Fatalf("read-only status created runtime state: %v", err)
	}
}
