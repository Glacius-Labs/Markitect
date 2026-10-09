package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolBuildDigestBindsActualBytes(t *testing.T) {
	name := filepath.Join(t.TempDir(), "host")
	if err := os.WriteFile(name, []byte("first build"), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := toolExecutableDigest(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("other build"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := toolExecutableDigest(name)
	if err != nil || first == second {
		t.Fatalf("changed build identity=%s/%s err=%v", first, second, err)
	}
	running, err := ToolBuildDigest()
	if err != nil || !strings.HasPrefix(running, "sha256:") || len(running) != 71 {
		t.Fatalf("running identity=%s err=%v", running, err)
	}
}
