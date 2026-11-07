package bootstrap

import (
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

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/upgrade"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const pluginRoot = "../../plugin"

// launcherEnv is set while TestLauncherChild runs as the "binary" a release
// serves: the test binary itself, which the launcher downloads and starts.
const launcherEnv = "MARKITECT_LAUNCHER_TEST"

// TestLauncherChild is what the launcher starts in the tests below.
func TestLauncherChild(t *testing.T) {
	if os.Getenv(launcherEnv) == "" {
		t.Skip("only run by the launcher tests")
	}
	fmt.Printf("child: plugin %s\n", os.Getenv(cli.PluginEnv))
	os.Exit(0)
}

// testVersion is the release the tests serve. The plugin of the repository says
// -dev between releases, which the launcher refuses, so the tests run a copy of
// the plugin that names this version.
const testVersion = "9.9.9"

// pluginCopy puts the launcher and the manifest of the plugin, with the given
// version, in root.
func pluginCopy(t *testing.T, root, version string) {
	t.Helper()
	for _, f := range []string{"bin/markitect", ".claude-plugin/plugin.json"} {
		data, err := os.ReadFile(filepath.Join(pluginRoot, filepath.FromSlash(f)))
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
	asset := upgrade.AssetName(version, runtime.GOOS, runtime.GOARCH)
	sum := sha256.Sum256(binary)
	listed := hex.EncodeToString(sum[:])
	if tamper {
		listed = strings.Repeat("0", len(listed))
	}
	sums := listed + "  " + asset + "\n"
	downloads = new(atomic.Int32)
	mux := http.NewServeMux()
	prefix := "/download/v" + version + "/"
	mux.HandleFunc(prefix+upgrade.ChecksumsName(version), func(w http.ResponseWriter, r *http.Request) {
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
	if code != 0 || !strings.Contains(out, "child: plugin "+version) {
		t.Fatalf("first call: exit %d\n%s", code, out)
	}
	if n := downloads.Load(); n != 2 {
		t.Fatalf("first call made %d downloads, want the checksums and the binary", n)
	}
	out, code = launch(t, root, mirror, data, nil, childArgs...)
	if code != 0 || !strings.Contains(out, "child: plugin "+version) {
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
		if r := <-results; r.code != 0 || !strings.Contains(r.out, "child: plugin "+version) {
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
	if code != 0 || !strings.Contains(out, "child: plugin "+version) {
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

// hookCommands reads the commands of the hooks of an event, in the order of their
// groups, from the plugin's hooks file.
func hookCommands(t *testing.T, event string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(pluginRoot, "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var h struct {
		Hooks map[string][]struct {
			Hooks []struct{ Command string }
		}
	}
	if err := json.Unmarshal(data, &h); err != nil {
		t.Fatal(err)
	}
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
// variables it exports to hooks, and feeds it stdin.
func runHook(t *testing.T, command, root, data, project, mirror, stdin string) (string, int) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("the hooks are shell commands and there is no sh")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("the launcher downloads with curl")
	}
	cmd := exec.Command(sh, "-c", command)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(),
		"CLAUDE_PLUGIN_ROOT="+filepath.ToSlash(root), "CLAUDE_PLUGIN_DATA="+data,
		"CLAUDE_PROJECT_DIR="+filepath.ToSlash(project), "MARKITECT_RELEASES_URL="+mirror, launcherEnv+"=1")
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
	command := hookCommands(t, "SessionStart")[0]

	plain := testkit.TempDir(t)
	if out, code := runHook(t, command, root, data, plain, mirror, ""); code != 0 || out != "" || downloads.Load() != 0 {
		t.Fatalf("in a project without .markitect: exit %d, downloads %d\n%s", code, downloads.Load(), out)
	}
	if entries, _ := os.ReadDir(data); len(entries) != 0 {
		t.Fatalf("in a project without .markitect the hook wrote %v", entries)
	}

	project := testkit.TempDir(t)
	if err := os.Mkdir(filepath.Join(project, ".markitect"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, code := runHook(t, command, root, data, project, mirror, ""); code != 0 || strings.Contains(out, "child:") || downloads.Load() != 2 {
		t.Fatalf("in a Markitect project: exit %d, downloads %d\n%s", code, downloads.Load(), out)
	}
	if _, err := os.Stat(filepath.Join(data, "v"+testVersion)); err != nil {
		t.Fatalf("the hook fetched no binary: %v", err)
	}
}

// Each guard hook starts the launcher, and with it a download, only for a call it
// guards: the one before shell commands for markitect mark, the one before
// writing tools for a subagent's write under .markitect. Claude Code's own filter
// is not relied on: a hook was seen to start for commands that did not match it.
func TestGuardHooksStartTheLauncherOnlyForTheirCalls(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	unreachable := srv.URL
	srv.Close()
	root, project := testPlugin(t), testkit.TempDir(t)
	commands := hookCommands(t, "PreToolUse")
	shell := func(command string) string {
		return `{"agent_id":"a1","agent_type":"markitect-builder","tool_name":"Bash","tool_input":{"command":"` + command + `"}}`
	}
	edit := func(agent, file string) string {
		id := ""
		if agent != "" {
			id = `"agent_id":"` + agent + `",`
		}
		return `{` + id + `"tool_name":"Edit","tool_input":{"file_path":"` + file + `"}}`
	}
	cases := []struct{ others, guarded []string }{
		{
			others:  []string{shell("ls -la && git status"), shell("markitect status"), edit("a1", "/p/.markitect/model/x.md")},
			guarded: []string{shell("markitect mark fees/fee@0a1b2c ok -e src/fees.go")},
		},
		{
			others:  []string{edit("", "/p/.markitect/model/x.md"), edit("a1", "/p/src/fees.go"), shell("markitect mark x@0a1b2c ok")},
			guarded: []string{edit("a1", "/p/.markitect/model/x.md"), edit("a1", `C:\\p\\.markitect\\review.sum`)},
		},
	}
	if len(commands) != len(cases) {
		t.Fatalf("%d guard hooks, want %d", len(commands), len(cases))
	}
	for i, command := range commands {
		data := testkit.TempDir(t)
		for _, input := range cases[i].others {
			if out, code := runHook(t, command, root, data, project, unreachable, input); code != 0 || out != "" {
				t.Fatalf("hook %d before %s: exit %d\n%s", i, input, code, out)
			}
		}
		if entries, _ := os.ReadDir(data); len(entries) != 0 {
			t.Fatalf("hook %d wrote %v for calls it does not guard", i, entries)
		}
		for _, input := range cases[i].guarded {
			out, code := runHook(t, command, root, data, project, unreachable, input)
			if code != 2 || !strings.Contains(out, "cannot download") {
				t.Fatalf("hook %d before %s: exit %d, want the launcher to have run and failed to download\n%s", i, input, code, out)
			}
		}
	}
}

// A plugin folder of a development version has no release to fetch.
func TestLauncherRefusesADevelopmentVersion(t *testing.T) {
	root := testkit.TempDir(t)
	pluginCopy(t, root, "0.16.0-dev")
	out, code := launch(t, root, "http://unused.invalid", testkit.TempDir(t), nil)
	if code != 2 || !strings.Contains(out, "development version 0.16.0-dev") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}
