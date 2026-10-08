package projectcli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairWithoutWriteReadsStatusOnly(t *testing.T) {
	repo := copyProjectWorld(t)
	var out, errout bytes.Buffer
	code := Run([]string{"repair", "--repo", repo, "--run", "missing-run"}, &out, &errout)
	if code == 0 || !strings.Contains(errout.String(), "missing-run") {
		t.Fatalf("repair status for missing run exit=%d stderr=%s stdout=%s", code, errout.String(), out.String())
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(".markitect/runs"))); !os.IsNotExist(err) {
		t.Fatalf("read-only repair created runtime state: %v", err)
	}
}
