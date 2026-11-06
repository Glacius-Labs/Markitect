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

// pluginVersion reads the version of the plugin in the repository.
func pluginVersion(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(pluginRoot, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct{ Version string }
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m.Version
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
	version := pluginVersion(t)
	mirror, downloads := releasePage(t, version, false)
	data := testkit.TempDir(t)
	childArgs := []string{"-test.run=TestLauncherChild"}

	out, code := launch(t, pluginRoot, mirror, data, nil, childArgs...)
	if code != 0 || !strings.Contains(out, "child: plugin "+version) {
		t.Fatalf("first call: exit %d\n%s", code, out)
	}
	if n := downloads.Load(); n != 2 {
		t.Fatalf("first call made %d downloads, want the checksums and the binary", n)
	}
	out, code = launch(t, pluginRoot, mirror, data, nil, childArgs...)
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
	version := pluginVersion(t)
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
			out, code := launch(t, pluginRoot, mirror, data, nil, "-test.run=TestLauncherChild")
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
	version := pluginVersion(t)
	mirror, downloads := releasePage(t, version, false)
	data := testkit.TempDir(t)
	out, code := launch(t, pluginRoot, mirror, data, []string{"MARKITECT_ENSURE_ONLY=1"}, "-test.run=TestLauncherChild")
	if code != 0 || strings.Contains(out, "child:") {
		t.Fatalf("exit %d, and the binary ran:\n%s", code, out)
	}
	if downloads.Load() != 2 {
		t.Fatalf("downloads = %d, want 2", downloads.Load())
	}
	// What the hook fetched is there for the next call, which downloads nothing.
	out, code = launch(t, pluginRoot, mirror, data, nil, "-test.run=TestLauncherChild")
	if code != 0 || !strings.Contains(out, "child:") || downloads.Load() != 2 {
		t.Fatalf("exit %d, downloads %d:\n%s", code, downloads.Load(), out)
	}
}

func TestLauncherRefusesWhatDoesNotMatchTheChecksums(t *testing.T) {
	version := pluginVersion(t)
	mirror, _ := releasePage(t, version, true)
	data := testkit.TempDir(t)
	out, code := launch(t, pluginRoot, mirror, data, nil, "-test.run=TestLauncherChild")
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
	out, code := launch(t, pluginRoot, url, testkit.TempDir(t), nil)
	if code != 2 || !strings.Contains(out, "cannot download") || !strings.Contains(out, "sandbox") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}

// A plugin folder of a development version has no release to fetch.
func TestLauncherRefusesADevelopmentVersion(t *testing.T) {
	root := testkit.TempDir(t)
	for _, f := range []string{"bin/markitect", ".claude-plugin/plugin.json"} {
		data, err := os.ReadFile(filepath.Join(pluginRoot, filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(f, "plugin.json") {
			data = regexp.MustCompile(`"version"\s*:\s*"[^"]*"`).ReplaceAll(data, []byte(`"version": "0.16.0-dev"`))
		}
		dst := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	out, code := launch(t, root, "http://unused.invalid", testkit.TempDir(t), nil)
	if code != 2 || !strings.Contains(out, "development version 0.16.0-dev") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}
