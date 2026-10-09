package projectworkspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSameGitReportedPathRejectsDifferentAndMissingPaths(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	if err := os.Mkdir(left, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(right, 0700); err != nil {
		t.Fatal(err)
	}
	if sameGitReportedPath(left, right) {
		t.Fatal("distinct existing directories were treated as the same path")
	}
	if sameGitReportedPath(filepath.Join(root, "missing-left"), filepath.Join(root, "missing-right")) {
		t.Fatal("distinct nonexistent paths were treated as the same path")
	}
}

func TestSamePathRecognizesWindowsDirectoryAliasByIdentity(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows physical path aliases are the behavior under test")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	alias := filepath.Join(root, "alias")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	// A directory junction is available without symlink privilege on supported
	// Windows filesystems and gives the same directory a distinct absolute path.
	command := "mklink /J " + alias + " " + target
	if output, err := exec.Command("cmd.exe", "/d", "/c", command).CombinedOutput(); err != nil {
		t.Skipf("cannot create a test directory junction: %v (%s)", err, output)
	}
	left, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	right, err := os.Stat(alias)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(left, right) {
		t.Skip("this filesystem does not expose junction targets as one file identity")
	}
	if filepath.Clean(target) == filepath.Clean(alias) {
		t.Fatal("junction did not produce a distinct path spelling")
	}
	if !sameGitReportedPath(target, alias) {
		t.Fatal("same physical Windows directory was rejected through its junction alias")
	}
	if sameGitReportedPath(target, filepath.Join(root, "missing")) {
		t.Fatal("an unrelated missing path was accepted as a physical alias")
	}
}

func TestSameGitReportedPathRejectsFileIdentityAlias(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows filesystem identity comparison is the behavior under test")
	}
	root := t.TempDir()
	original := filepath.Join(root, "original")
	alias := filepath.Join(root, "alias")
	if err := os.WriteFile(original, []byte("identity"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(original, alias); err != nil {
		t.Skipf("cannot create a test hard link: %v", err)
	}
	left, err := os.Stat(original)
	if err != nil {
		t.Fatal(err)
	}
	right, err := os.Stat(alias)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(left, right) {
		t.Skip("this filesystem does not expose hard links as one file identity")
	}
	if sameGitReportedPath(original, alias) {
		t.Fatal("Git-reported directory identity check accepted a file hard-link alias")
	}
}
