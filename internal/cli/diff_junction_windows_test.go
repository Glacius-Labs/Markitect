package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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

// shortName is the 8.3 name of a path, or the path itself where the file system
// has none.
func shortName(t *testing.T, p string) string {
	t.Helper()
	from, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]uint16, syscall.MAX_PATH)
	n, err := syscall.GetShortPathName(from, &buf[0], uint32(len(buf)))
	if err != nil || n == 0 || int(n) > len(buf) {
		return p
	}
	return syscall.UTF16ToString(buf[:n])
}

// A project named by its short name is the same folder: a copy inside it is left
// out, whichever of the two is named by the short name.
func TestDiffLeavesOutAProjectNamedByAShortName(t *testing.T) {
	outer := projectOf(t, lendingFiles)
	inner := projectIn(t, filepath.Join(outer, "backup", "old"), lendingFiles)
	shortOuter, shortInner := shortName(t, outer), shortName(t, inner)
	if shortOuter == outer && shortInner == inner {
		t.Skip("this file system has no short names")
	}
	const want = "## Files no module names (0)\n\n(none)\n\nFiles of NEW that no module names: 2\n"
	for name, args := range map[string][]string{
		"OLD inside NEW, NEW short": {inner, shortOuter},
		"OLD inside NEW, OLD short": {shortInner, outer},
		"NEW inside OLD, OLD short": {shortOuter, inner},
		"NEW inside OLD, NEW short": {outer, shortInner},
		"both short":                {shortInner, shortOuter},
	} {
		code, out, errOut := run(append([]string{"diff"}, args...)...)
		if code != ExitOK || errOut != "" || !strings.HasSuffix(out, want) {
			t.Errorf("%s: %d %q\n%s", name, code, errOut, out)
		}
	}
}
