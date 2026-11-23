package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A Windows junction is never followed: the files behind it are no files of
// the project, however they change.
func TestDiffSkipsJunctions(t *testing.T) {
	outside := t.TempDir()
	write(t, outside, "x.go", "package lending\n")
	older, newer := projectOf(t, lendingFiles), projectOf(t, lendingFiles)
	for _, dir := range []string{older, newer} {
		link := filepath.Join(dir, "src", "lending", "junction")
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput(); err != nil {
			t.Skip("no junctions here:", err, string(out))
		}
	}
	write(t, outside, "x.go", "package lending // changed\n")
	write(t, outside, "y.go", "package lending\n")
	if code, out, _ := run("diff", older, newer); code != ExitOK || !strings.HasSuffix(out, "Files of NEW that no module names: 2\n") {
		t.Errorf("%d\n%s", code, out)
	}
}
