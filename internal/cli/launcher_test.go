package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// pluginDir is the Claude Code plugin. It only fetches the tool.
const pluginDir = "../../plugin"

// The plugin brings the tool through its launcher, bin/markitect, which
// downloads the release binary of the plugin's own version. The marketplace
// that lists the plugin, and pins it to a release tag, is the repository
// Glacius-Labs/Claude-Plugins.

// launcherEnv is set while TestLauncherChild runs as the "binary" a release
// serves: the test binary itself, which the launcher downloads and starts.
const launcherEnv = "MARKITECT_LAUNCHER_TEST"

// TestLauncherChild is what the launcher starts in the tests below.
func TestLauncherChild(t *testing.T) {
	if os.Getenv(launcherEnv) == "" {
		t.Skip("only run by the launcher tests")
	}
	// It returns, so that TestMain removes its temporary cache folder.
	fmt.Println("child: started")
}

// testVersion is the release the tests serve. The plugin of the repository says
// -dev between releases, whose launcher fetches nothing and starts the build on
// the PATH instead, so the tests run a copy of the plugin that names this version.
const testVersion = "9.9.9"

// assetName is the file of a release that holds the binary for a system, as
// the release workflow names it.
func assetName(version, goos, goarch string) string {
	name := "markitect-v" + version + "-" + goos + "-" + goarch
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// checksumsName is the file of a release that lists the sums of its binaries.
func checksumsName(version string) string { return "markitect-v" + version + "-checksums.txt" }

func readJSON(t *testing.T, file string, into any) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
}

// The plugin's version is the version of the source, always: the release tag of
// the catalog pins the plugin to the commit of a release, where the launcher reads
// the version it fetches. Between releases both say -dev, and the launcher then
// fetches nothing and starts the build from source on the PATH.
func TestPluginHasTheVersionOfTheSource(t *testing.T) {
	var manifest struct{ Name, Version string }
	readJSON(t, filepath.Join(pluginDir, ".claude-plugin", "plugin.json"), &manifest)
	if manifest.Name != "markitect" {
		t.Errorf("the plugin is named %q, not markitect", manifest.Name)
	}
	if source := buildinfo.Version(); source != manifest.Version {
		t.Fatalf("the source says %s but the plugin names %s; a release commit and the commit after it set both", source, manifest.Version)
	}
}

// The plugin only fetches the tool: it holds no skills, agents or jobs. Beside
// the launcher and its hook it carries the default, default/, which is plain
// Markdown that a project copies into its spec folder and no component that
// Claude Code loads.
func TestPluginOnlyFetchesTheTool(t *testing.T) {
	for _, name := range []string{"skills", "agents", "jobs.md"} {
		if _, err := os.Stat(filepath.Join(pluginDir, name)); err == nil {
			t.Errorf("the plugin holds %s", name)
		}
	}
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if got := strings.Join(names, " "); got != ".claude-plugin bin default hooks" {
		t.Errorf("the plugin holds %s", got)
	}
	var manifest struct{ Description string }
	readJSON(t, filepath.Join(pluginDir, ".claude-plugin", "plugin.json"), &manifest)
	if !strings.Contains(manifest.Description, "launcher downloads the release binary of the plugin") {
		t.Errorf("the plugin describes itself as %q", manifest.Description)
	}
}

// The shim that runs the launcher from PowerShell and cmd is checked out with
// CRLF line endings, as cmd reads a batch file most reliably.
func TestPluginShimHasWindowsLineEndings(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(pluginDir, "bin", "markitect.cmd"))
	if err != nil {
		t.Fatal(err)
	}
	if lf, crlf := bytes.Count(data, []byte("\n")), bytes.Count(data, []byte("\r\n")); lf == 0 || lf != crlf {
		t.Fatalf("markitect.cmd has %d line ends, %d of them CRLF; .gitattributes should check *.cmd out with CRLF", lf, crlf)
	}
}

// The hooks file is plain JSON of a fixed shape, with two hooks at the start of
// a session and none before a tool. The first puts the launcher on the PATH of
// the Bash tool, and Claude Code waits for it, so that the first command finds
// it. The second fetches the binary in the background.
func TestPluginHooksServeTheLauncher(t *testing.T) {
	var h struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []struct {
				Command string
				Async   bool
			}
		}
	}
	readJSON(t, filepath.Join(pluginDir, "hooks", "hooks.json"), &h)
	if len(h.Hooks) != 1 {
		t.Fatalf("the plugin has hooks for %d events, want SessionStart alone", len(h.Hooks))
	}
	const launcher, path = "${CLAUDE_PLUGIN_ROOT}/bin/markitect", "${CLAUDE_PLUGIN_ROOT}/hooks/path.sh"
	want := []struct {
		matcher, runs string
		async         bool
	}{{"", path, false}, {"startup|resume", launcher, true}}
	got := h.Hooks["SessionStart"]
	if len(got) != len(want) {
		t.Fatalf("SessionStart: %+v, want %d groups", got, len(want))
	}
	for i, w := range want {
		if got[i].Matcher != w.matcher || len(got[i].Hooks) != 1 {
			t.Fatalf("group %d: %+v, want one hook for %q", i, got[i], w.matcher)
		}
		hook := got[i].Hooks[0]
		if !strings.Contains(hook.Command, w.runs) || hook.Async != w.async {
			t.Errorf("group %d runs %q async=%v, want %s async=%v", i, hook.Command, hook.Async, w.runs, w.async)
		}
	}
}

// pluginCopy puts the launcher, its shim for Windows, the script of the hook that
// puts them on the PATH and the manifest of the plugin, with the given version,
// in root.
func pluginCopy(t *testing.T, root, version string) {
	t.Helper()
	for _, f := range []string{"bin/markitect", "bin/markitect.cmd", "hooks/path.sh", "hooks/hooks.json", ".claude-plugin/plugin.json"} {
		data, err := os.ReadFile(filepath.Join(pluginDir, filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(f, "plugin.json") {
			data = regexp.MustCompile(`"version"\s*:\s*"[^"]*"`).ReplaceAll(data, []byte(`"version": "`+version+`"`))
		}
		dst := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// testPlugin is a copy of the plugin that names testVersion.
func testPlugin(t *testing.T) string {
	t.Helper()
	root := testkit.TempDir(t)
	pluginCopy(t, root, testVersion)
	return root
}

// releasePage serves the release of the plugin's version, with the test binary as
// the binary for this platform. It counts the downloads.
func releasePage(t *testing.T, version string, tamper bool) (url string, downloads *atomic.Int32) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	asset := assetName(version, runtime.GOOS, runtime.GOARCH)
	sum := sha256.Sum256(binary)
	listed := hex.EncodeToString(sum[:])
	if tamper {
		listed = strings.Repeat("0", len(listed))
	}
	sums := listed + "  " + asset + "\n"
	downloads = new(atomic.Int32)
	mux := http.NewServeMux()
	prefix := "/download/v" + version + "/"
	mux.HandleFunc(prefix+checksumsName(version), func(w http.ResponseWriter, r *http.Request) {
		downloads.Add(1)
		fmt.Fprint(w, sums)
	})
	mux.HandleFunc(prefix+asset, func(w http.ResponseWriter, r *http.Request) {
		downloads.Add(1)
		w.Write(binary)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL, downloads
}

// launch runs the plugin's launcher as Claude Code's shell would, with a data
// folder of its own, and returns its output and exit code.
func launch(t *testing.T, root, mirror, data string, extraEnv []string, args ...string) (string, int) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("the launcher is a POSIX shell script and there is no sh")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("the launcher downloads with curl")
	}
	cmd := exec.Command(sh, append([]string{filepath.Join(root, "bin", "markitect")}, args...)...)
	cmd.Env = append(os.Environ(), "CLAUDE_PLUGIN_DATA="+data, "MARKITECT_RELEASES_URL="+mirror, launcherEnv+"=1")
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

// The first call downloads the binary of the plugin's release, checks it and starts
// it; later calls start it without downloading.
func TestLauncherFetchesTheReleaseOfThePlugin(t *testing.T) {
	version, root := testVersion, testPlugin(t)
	mirror, downloads := releasePage(t, version, false)
	data := testkit.TempDir(t)
	childArgs := []string{"-test.run=TestLauncherChild"}

	out, code := launch(t, root, mirror, data, nil, childArgs...)
	if code != 0 || !strings.Contains(out, "child: started") {
		t.Fatalf("first call: exit %d\n%s", code, out)
	}
	if n := downloads.Load(); n != 2 {
		t.Fatalf("first call made %d downloads, want the checksums and the binary", n)
	}
	out, code = launch(t, root, mirror, data, nil, childArgs...)
	if code != 0 || !strings.Contains(out, "child: started") {
		t.Fatalf("second call: exit %d\n%s", code, out)
	}
	if n := downloads.Load(); n != 2 {
		t.Fatalf("second call downloaded again (%d)", n)
	}
}

// Claude Code runs shell commands side by side, and the session-start hook runs
// beside them: calls that arrive while the binary is being fetched must all work.
func TestLauncherCalledAtTheSameTime(t *testing.T) {
	for _, tool := range []string{"sh", "curl"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("the launcher needs %s", tool)
		}
	}
	version, root := testVersion, testPlugin(t)
	mirror, _ := releasePage(t, version, false)
	data := testkit.TempDir(t)
	const calls = 5
	type result struct {
		out  string
		code int
	}
	results := make(chan result, calls)
	for i := 0; i < calls; i++ {
		go func() {
			out, code := launch(t, root, mirror, data, nil, "-test.run=TestLauncherChild")
			results <- result{out, code}
		}()
	}
	for i := 0; i < calls; i++ {
		if r := <-results; r.code != 0 || !strings.Contains(r.out, "child: started") {
			t.Errorf("a call failed: exit %d\n%s", r.code, r.out)
		}
	}
	if leftovers, _ := filepath.Glob(filepath.Join(data, ".fetch.*")); len(leftovers) != 0 {
		t.Errorf("calls left %v", leftovers)
	}
}

// The session-start hook only fetches.
func TestLauncherCanFetchWithoutStarting(t *testing.T) {
	version, root := testVersion, testPlugin(t)
	mirror, downloads := releasePage(t, version, false)
	data := testkit.TempDir(t)
	out, code := launch(t, root, mirror, data, []string{"MARKITECT_ENSURE_ONLY=1"}, "-test.run=TestLauncherChild")
	if code != 0 || strings.Contains(out, "child:") {
		t.Fatalf("exit %d, and the binary ran:\n%s", code, out)
	}
	if downloads.Load() != 2 {
		t.Fatalf("downloads = %d, want 2", downloads.Load())
	}
	// What the hook fetched is there for the next call, which downloads nothing.
	out, code = launch(t, root, mirror, data, nil, "-test.run=TestLauncherChild")
	if code != 0 || !strings.Contains(out, "child:") || downloads.Load() != 2 {
		t.Fatalf("exit %d, downloads %d:\n%s", code, downloads.Load(), out)
	}
}

func TestLauncherRefusesWhatDoesNotMatchTheChecksums(t *testing.T) {
	version, root := testVersion, testPlugin(t)
	mirror, _ := releasePage(t, version, true)
	data := testkit.TempDir(t)
	out, code := launch(t, root, mirror, data, nil, "-test.run=TestLauncherChild")
	if code != 2 || !strings.Contains(out, "does not match its checksum") || strings.Contains(out, "child:") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	entries, _ := filepath.Glob(filepath.Join(data, "*", "*"))
	if len(entries) != 0 {
		t.Fatalf("a refused download left %v", entries)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(data, ".fetch.*")); len(leftovers) != 0 {
		t.Fatalf("a refused download left %v", leftovers)
	}
}

func TestLauncherSaysWhenThereIsNoNetwork(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	out, code := launch(t, testPlugin(t), url, testkit.TempDir(t), nil)
	if code != 2 || !strings.Contains(out, "cannot download") || !strings.Contains(out, "sandbox") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}

// Without CLAUDE_PLUGIN_DATA, which only hooks get, the launcher finds the folder
// Claude Code would give the plugin: plugins/data/<plugin>-<marketplace>, with the
// marketplace taken from where the plugin is installed, whatever it is called.
func TestLauncherFindsTheDataFolderOfAnInstalledPlugin(t *testing.T) {
	version := testVersion
	mirror, downloads := releasePage(t, version, false)
	plugins := testkit.TempDir(t)
	root := filepath.Join(plugins, "cache", "some.market", "markitect", version)
	pluginCopy(t, root, version)
	out, code := launch(t, root, mirror, "", nil, "-test.run=TestLauncherChild")
	if code != 0 || !strings.Contains(out, "child: started") {
		t.Fatalf("exit %d\n%s", code, out)
	}
	// Every character that is no letter, digit, underscore or hyphen becomes a hyphen.
	want := filepath.Join(plugins, "data", "markitect-some-market", "v"+version)
	if entries, _ := os.ReadDir(want); len(entries) != 1 {
		t.Fatalf("the binary is not in %s: %v", want, entries)
	}
	if downloads.Load() != 2 {
		t.Fatalf("downloads = %d, want 2", downloads.Load())
	}
}

// pathWithoutMarkitect is the PATH of the test without the folders that hold a
// markitect, so that a build installed on the machine does not answer.
func pathWithoutMarkitect() string {
	var kept []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		found := false
		for _, name := range []string{"markitect", "markitect.exe"} {
			if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() {
				found = true
			}
		}
		if !found {
			kept = append(kept, dir)
		}
	}
	return strings.Join(kept, string(os.PathListSeparator))
}

// devPlugin is a copy of the plugin with a development version, and its bin.
func devPlugin(t *testing.T) (root, bin string) {
	t.Helper()
	root = filepath.Join(testkit.TempDir(t), "dev plugin")
	pluginCopy(t, root, "0.16.0-dev")
	return root, filepath.Join(root, "bin")
}

// A plugin folder of a development version has no release to fetch: it starts
// the next markitect on the PATH, a build from source, with every argument,
// passing over its own folder although the plugin's bin comes first.
func TestDevelopmentLauncherStartsTheBuildOnThePath(t *testing.T) {
	root, bin := devPlugin(t)
	source := filepath.Join(testkit.TempDir(t), "go bin")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "markitect"), []byte("#!/bin/sh\necho \"from source: $*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := "PATH=" + strings.Join([]string{bin, source, pathWithoutMarkitect()}, string(os.PathListSeparator))
	data := testkit.TempDir(t)
	out, code := launch(t, root, "http://unused.invalid", data, []string{path}, "check", "--json", "two words")
	if code != 0 || !strings.Contains(out, "from source: check --json two words") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if entries, _ := os.ReadDir(data); len(entries) != 0 {
		t.Fatalf("a development plugin fetched %v", entries)
	}
	// PowerShell and cmd reach the same fall-through through markitect.cmd. The
	// bash of Git for Windows puts HOME's bin folder on its PATH, so HOME is a
	// folder of the test's own.
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("git"); err == nil {
			cmd := exec.Command("cmd", "/d", "/c", "markitect check")
			cmd.Env = append(os.Environ(), path, "CLAUDE_PLUGIN_DATA="+data, "HOME="+testkit.TempDir(t))
			if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "from source: check") {
				t.Fatalf("markitect.cmd: %v\n%s", err, out)
			}
		}
	}
	// The session-start hook has nothing to fetch and starts nothing.
	if out, code := launch(t, root, "http://unused.invalid", data, []string{path, "MARKITECT_ENSURE_ONLY=1"}); code != 0 || out != "" {
		t.Fatalf("ensure only: exit %d:\n%s", code, out)
	}
}

// Without a build on the PATH, a development plugin says what it needs, also
// when a second development plugin on the PATH would pass the call back.
func TestDevelopmentLauncherWithoutABuildSaysSo(t *testing.T) {
	root, bin := devPlugin(t)
	_, other := devPlugin(t)
	path := "PATH=" + strings.Join([]string{bin, other, pathWithoutMarkitect()}, string(os.PathListSeparator))
	out, code := launch(t, root, "http://unused.invalid", testkit.TempDir(t), []string{path}, "version")
	if code != 2 || !strings.Contains(out, "a development plugin needs markitect built from source on the PATH, after the plugin's bin") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}

// hookCommands reads the commands of the hooks of an event, in the order of their
// groups, from the plugin's hooks file.
func hookCommands(t *testing.T, event string) []string {
	t.Helper()
	var h struct {
		Hooks map[string][]struct {
			Hooks []struct{ Command string }
		}
	}
	readJSON(t, filepath.Join(pluginDir, "hooks", "hooks.json"), &h)
	var commands []string
	for _, group := range h.Hooks[event] {
		for _, hook := range group.Hooks {
			commands = append(commands, hook.Command)
		}
	}
	if len(commands) == 0 {
		t.Fatalf("%s: no hook", event)
	}
	return commands
}

// runHook runs the command of a hook the way Claude Code does, in a shell with the
// variables it exports to hooks.
func runHook(t *testing.T, command, root, data, project, mirror string, extraEnv ...string) (string, int) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("the hooks are shell commands and there is no sh")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("the launcher downloads with curl")
	}
	cmd := exec.Command(sh, "-c", command)
	cmd.Env = append(os.Environ(),
		"CLAUDE_PLUGIN_ROOT="+filepath.ToSlash(root), "CLAUDE_PLUGIN_DATA="+data,
		"CLAUDE_PROJECT_DIR="+filepath.ToSlash(project), "MARKITECT_RELEASES_URL="+mirror, launcherEnv+"=1")
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

// The session-start hook fetches the binary for a project that uses Markitect, and
// does nothing in any other, where the plugin must stay out of the way.
func TestStartHookFetchesOnlyForAMarkitectProject(t *testing.T) {
	mirror, downloads := releasePage(t, testVersion, false)
	root, data := testPlugin(t), testkit.TempDir(t)
	command := hookCommands(t, "SessionStart")[1]

	plain := testkit.TempDir(t)
	if out, code := runHook(t, command, root, data, plain, mirror); code != 0 || out != "" || downloads.Load() != 0 {
		t.Fatalf("in a project without .markitect: exit %d, downloads %d\n%s", code, downloads.Load(), out)
	}
	if entries, _ := os.ReadDir(data); len(entries) != 0 {
		t.Fatalf("in a project without .markitect the hook wrote %v", entries)
	}

	project := testkit.TempDir(t)
	if err := os.Mkdir(filepath.Join(project, ".markitect"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, code := runHook(t, command, root, data, project, mirror); code != 0 || strings.Contains(out, "child:") || downloads.Load() != 2 {
		t.Fatalf("in a Markitect project: exit %d, downloads %d\n%s", code, downloads.Load(), out)
	}
	if _, err := os.Stat(filepath.Join(data, "v"+testVersion)); err != nil {
		t.Fatalf("the hook fetched no binary: %v", err)
	}
}

// The session-start hook Claude Code waits for puts the launcher on the PATH of its
// Bash tool, which on Windows does not get the plugin's bin folder: it appends one line
// to the file CLAUDE_ENV_FILE names, which Claude Code runs before each command.
// The line puts the folder on the PATH once, however often the hook ran or the
// file was run, and survives a folder whose name has a space and a quote. Without
// CLAUDE_ENV_FILE the hook does nothing.
func TestPathHookPutsTheLauncherOnThePath(t *testing.T) {
	mirror, _ := releasePage(t, testVersion, false)
	root := filepath.Join(testkit.TempDir(t), "it's a plugin")
	pluginCopy(t, root, testVersion)
	data, project := testkit.TempDir(t), testkit.TempDir(t)
	command := hookCommands(t, "SessionStart")[0]

	if out, code := runHook(t, command, root, data, project, mirror, "CLAUDE_ENV_FILE="); code != 0 || out != "" {
		t.Fatalf("without CLAUDE_ENV_FILE: exit %d\n%s", code, out)
	}
	envFile := filepath.Join(testkit.TempDir(t), "sessionstart-hook.sh")
	for i := 0; i < 2; i++ {
		if out, code := runHook(t, command, root, data, project, mirror, "CLAUDE_ENV_FILE="+filepath.ToSlash(envFile)); code != 0 || out != "" {
			t.Fatalf("run %d: exit %d\n%s", i+1, code, out)
		}
	}
	written, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(written), "\n"); lines != 1 {
		t.Fatalf("two runs of the hook wrote %d lines, want 1:\n%s", lines, written)
	}

	// What Claude Code does before a command: run the file, here twice, then the
	// command, which names markitect alone.
	check := `. "$1"; . "$1"
bin=$(cd "$2/bin" && pwd)
found=$(command -v markitect) || { echo "markitect is not on the PATH: $PATH"; exit 1; }
[ "$found" = "$bin/markitect" ] || { echo "markitect is $found, want $bin/markitect"; exit 1; }
n=$(printf '%s\n' "$PATH" | tr ':' '\n' | grep -cxF "$bin")
[ "$n" = 1 ] || { echo "the PATH names $bin $n times: $PATH"; exit 1; }
markitect -test.run=TestLauncherChild`
	sh, _ := exec.LookPath("sh")
	cmd := exec.Command(sh, "-c", check, "check", filepath.ToSlash(envFile), filepath.ToSlash(root))
	cmd.Env = append(os.Environ(), "CLAUDE_PLUGIN_DATA="+data, "MARKITECT_RELEASES_URL="+mirror, launcherEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "child: started") {
		t.Fatalf("markitect by name after the hook's line: %v\n%s", err, out)
	}
}

// On Windows the PowerShell tool has the plugin's bin folder on its PATH but cannot
// run the launcher, a shell script; markitect.cmd beside it runs the launcher with
// the bash of Git for Windows, by name as PowerShell and cmd find it. A
// CLAUDE_CODE_GIT_BASH_PATH that names no file is passed over for the bash beside
// git, and without either the shim says what is missing.
func TestShimRunsTheLauncherOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("markitect.cmd is for Windows")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("the shim finds bash beside git, and there is no git")
	}
	mirror, _ := releasePage(t, testVersion, false)
	root := filepath.Join(testkit.TempDir(t), "a plugin")
	pluginCopy(t, root, testVersion)
	data := testkit.TempDir(t)
	env := func(path, gitBash string) []string {
		var kept []string
		for _, kv := range os.Environ() {
			name, _, _ := strings.Cut(kv, "=")
			if !strings.EqualFold(name, "PATH") && !strings.EqualFold(name, "CLAUDE_CODE_GIT_BASH_PATH") {
				kept = append(kept, kv)
			}
		}
		return append(kept, "PATH="+path, "CLAUDE_CODE_GIT_BASH_PATH="+gitBash, "CLAUDE_PLUGIN_DATA="+data, "MARKITECT_RELEASES_URL="+mirror, launcherEnv+"=1")
	}
	bin := filepath.Join(root, "bin")

	for _, gitBash := range []string{"", filepath.Join(testkit.TempDir(t), "no", "bash.exe")} {
		cmd := exec.Command("cmd", "/d", "/c", "markitect -test.run=TestLauncherChild")
		cmd.Env = env(bin+string(os.PathListSeparator)+os.Getenv("PATH"), gitBash)
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), "child: started") {
			t.Fatalf("markitect by name in cmd, CLAUDE_CODE_GIT_BASH_PATH=%q: %v\n%s", gitBash, err, out)
		}
	}

	system := filepath.Join(os.Getenv("SystemRoot"), "System32")
	cmd := exec.Command(filepath.Join(system, "cmd.exe"), "/d", "/c", "markitect -test.run=TestLauncherChild")
	cmd.Env = env(bin+string(os.PathListSeparator)+system, "")
	out, err := cmd.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 2 || !strings.Contains(string(out), "needs the bash of Git for Windows") || strings.Contains(string(out), "child:") {
		t.Fatalf("without git on the PATH: %v\n%s", err, out)
	}
}
