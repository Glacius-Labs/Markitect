package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// fakeReleases stands in for GitHub: it names a newest release and records what
// update asked it to install and to run.
type fakeReleases struct {
	latest     string
	latestErr  error
	installErr error
	runCode    int

	looked    bool
	installed []string // "version exe"
	ran       []string // the exe and its arguments
}

func (f *fakeReleases) Latest(context.Context) (string, error) {
	f.looked = true
	return f.latest, f.latestErr
}

func (f *fakeReleases) Install(_ context.Context, version, exe string) error {
	if f.installErr != nil {
		return f.installErr
	}
	f.installed = append(f.installed, version+" "+exe)
	return nil
}

func (f *fakeReleases) Run(_ context.Context, exe string, args []string, stdout, _ io.Writer) (int, error) {
	f.ran = append([]string{exe}, args...)
	io.WriteString(stdout, "the new markitect finishes the update\n")
	return f.runCode, nil
}

// updateWith runs markitect update in a folder that is no project, as a binary of
// the given version that finds the given releases.
func updateWith(t *testing.T, version string, rel *fakeReleases, args ...string) (int, string, string) {
	t.Helper()
	emptyHome(t)
	var out, errOut bytes.Buffer
	c := cli.CLI{
		Git: git{}, Runs: runs{}, Stdout: &out, Stderr: &errOut,
		Releases:   rel,
		Executable: func() (string, error) { return filepath.Join("bin", "markitect"), nil },
		Version:    func() string { return version },
	}
	code := c.Run(context.Background(), append([]string{"update", "--repo", testkit.TempDir(t)}, args...))
	return code, out.String(), errOut.String()
}

func TestUpdateInstallsTheNewestRelease(t *testing.T) {
	rel := &fakeReleases{latest: "0.15.0"}
	code, out, errOut := updateWith(t, "0.14.1", rel)
	if code != 0 {
		t.Fatalf("update = %d\n%s%s", code, out, errOut)
	}
	if len(rel.installed) != 1 || !strings.HasPrefix(rel.installed[0], "0.15.0 ") || !strings.HasSuffix(rel.installed[0], "markitect") {
		t.Fatalf("installed %v, want the release 0.15.0 in the place of the executable", rel.installed)
	}
	// The new binary, not this old one, knows which files it ships.
	if len(rel.ran) < 3 || filepath.Base(rel.ran[0]) != "markitect" || strings.Join(rel.ran[1:3], " ") != "update --files" {
		t.Fatalf("ran %v, want the new binary to run update --files", rel.ran)
	}
	wantAll(t, out, "binary: updated 0.14.1 to 0.15.0", "the new markitect finishes the update")
	if strings.Contains(out, "personal skill:") {
		t.Fatalf("the old binary refreshed files itself:\n%s", out)
	}
}

func TestUpdatePassesOnTheExitCodeOfTheNewBinary(t *testing.T) {
	code, _, _ := updateWith(t, "0.14.1", &fakeReleases{latest: "0.15.0", runCode: 1})
	if code != 1 {
		t.Fatalf("update = %d, want the 1 of the new binary", code)
	}
}

func TestUpdateCheckSaysThereIsANewerRelease(t *testing.T) {
	rel := &fakeReleases{latest: "0.15.0"}
	code, out, _ := updateWith(t, "0.14.1", rel, "--check")
	if code != 1 {
		t.Fatalf("update --check = %d, want 1\n%s", code, out)
	}
	wantAll(t, out, "binary: 0.14.1, release 0.15.0 is newer", "personal skill:")
	if len(rel.installed) != 0 || len(rel.ran) != 0 {
		t.Fatalf("--check installed %v and ran %v", rel.installed, rel.ran)
	}
}

func TestUpdateWhenTheBinaryIsCurrent(t *testing.T) {
	for _, version := range []string{"0.15.0", "0.16.0-dev"} {
		rel := &fakeReleases{latest: "0.15.0"}
		code, out, _ := updateWith(t, version, rel, "--check")
		if code != 0 || len(rel.installed) != 0 {
			t.Fatalf("%s: update --check = %d, installed %v\n%s", version, code, rel.installed, out)
		}
		wantAll(t, out, "binary: up to date ("+version+")", "project: none here")
	}
}

func TestUpdateDoesNotReplaceADevelopmentBuild(t *testing.T) {
	rel := &fakeReleases{latest: "0.15.0"}
	code, out, _ := updateWith(t, "0.0.0-dev", rel)
	if code != 0 || len(rel.installed) != 0 {
		t.Fatalf("update = %d, installed %v\n%s", code, rel.installed, out)
	}
	wantAll(t, out, "development build (0.0.0-dev)", "release 0.15.0 exists", "--force")

	if code, out, _ := updateWith(t, "0.0.0-dev", rel, "--check"); code != 0 {
		t.Fatalf("a development build is not out of date, yet update --check = %d\n%s", code, out)
	}

	rel = &fakeReleases{latest: "0.15.0"}
	if code, out, _ := updateWith(t, "0.0.0-dev", rel, "--force"); code != 0 || len(rel.installed) != 1 {
		t.Fatalf("update --force = %d, installed %v\n%s", code, rel.installed, out)
	}
}

func TestUpdateLeavesABinaryThePluginManagesToThePlugin(t *testing.T) {
	t.Setenv(cli.PluginEnv, "0.14.1")
	rel := &fakeReleases{latest: "0.15.0"}
	code, out, _ := updateWith(t, "0.14.1", rel)
	if code != 0 || rel.looked || len(rel.installed) != 0 {
		t.Fatalf("update = %d, looked %v, installed %v\n%s", code, rel.looked, rel.installed, out)
	}
	wantAll(t, out, "binary: 0.14.1 comes with the Claude Code plugin (0.14.1)", "personal skill: comes with the plugin")
}

// Under the plugin there is no personal skill file, and no command should send the
// user to markitect install for one.
func TestThePluginBringsThePersonalSkill(t *testing.T) {
	emptyHome(t)
	t.Setenv(cli.PluginEnv, "0.15.0")
	r := testkit.NewRepo(t)
	out := mustRun(t, r.Dir, "init", "--name", "Demo")
	if strings.Contains(out, "tip:") {
		t.Fatalf("init sent a plugin user to markitect install:\n%s", out)
	}
	wantAll(t, mustRun(t, r.Dir, "version"), "personal skill: comes with the plugin")
}

func TestUpdateGoesOnWhenNoReleaseCanBeFound(t *testing.T) {
	for name, err := range map[string]error{
		"offline":    errors.New("dial tcp: no route to host"),
		"no release": errors.New("there is no release yet"),
	} {
		rel := &fakeReleases{latestErr: err}
		code, out, _ := updateWith(t, "0.14.1", rel)
		if code != 0 {
			t.Fatalf("%s: update = %d\n%s", name, code, out)
		}
		wantAll(t, out, "binary: could not look for a newer release: "+err.Error(), "personal skill:", "project: none here")
	}
}

func TestUpdateStopsWhenTheInstallFails(t *testing.T) {
	rel := &fakeReleases{latest: "0.15.0", installErr: errors.New("the download does not match its checksum")}
	code, out, errOut := updateWith(t, "0.14.1", rel)
	if code != 2 || len(rel.ran) != 0 {
		t.Fatalf("update = %d, ran %v\n%s%s", code, rel.ran, out, errOut)
	}
	wantAll(t, errOut, "could not install release 0.15.0", "does not match its checksum")
	if strings.Contains(out, "personal skill:") {
		t.Fatalf("update went on after a failed install:\n%s", out)
	}
}

// On Windows an update leaves the old executable beside the new one, since it ran
// while the update worked. The next update removes it.
func TestUpdateRemovesWhatAnEarlierUpdateMovedAside(t *testing.T) {
	emptyHome(t)
	dir := testkit.TempDir(t)
	exe := filepath.Join(dir, "markitect")
	for _, f := range []string{exe, exe + ".old"} {
		if err := os.WriteFile(f, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	c := cli.CLI{
		Git: git{}, Runs: runs{}, Stdout: &out, Stderr: io.Discard,
		Releases:   &fakeReleases{latest: "0.15.0"},
		Executable: func() (string, error) { return exe, nil },
		Version:    func() string { return "0.15.0" },
	}
	if code := c.Run(context.Background(), []string{"update", "--repo", testkit.TempDir(t)}); code != 0 {
		t.Fatalf("update = %d\n%s", code, out.String())
	}
	if _, err := os.Stat(exe + ".old"); !os.IsNotExist(err) {
		t.Fatalf("the old executable is still there: %v", err)
	}
	if _, err := os.Stat(exe); err != nil {
		t.Fatalf("update removed the executable: %v", err)
	}
}

func TestUpdateFilesLeavesTheBinaryAlone(t *testing.T) {
	rel := &fakeReleases{latest: "0.15.0"}
	code, out, _ := updateWith(t, "0.14.1", rel, "--files")
	if code != 0 || rel.looked {
		t.Fatalf("update --files = %d, looked %v\n%s", code, rel.looked, out)
	}
	if strings.Contains(out, "binary:") {
		t.Fatalf("update --files spoke of the binary:\n%s", out)
	}
}

func TestVersionNamesAPluginBinary(t *testing.T) {
	emptyHome(t)
	t.Setenv(cli.PluginEnv, "0.15.0")
	var out bytes.Buffer
	cli.CLI{Git: git{}, Runs: runs{}, Stdout: &out, Stderr: io.Discard}.Run(context.Background(), []string{"version", "--repo", testkit.TempDir(t)})
	wantAll(t, out.String(), "binary: comes with the Claude Code plugin (0.15.0)", "personal skill: comes with the plugin")
}

func TestNoPathWarningForABinaryThePluginStarted(t *testing.T) {
	t.Setenv(cli.PluginEnv, "0.15.0")
	if got := ownPathWarning(); got != "" {
		t.Fatalf("the plugin's launcher is what the PATH leads to, yet: %s", got)
	}
}
