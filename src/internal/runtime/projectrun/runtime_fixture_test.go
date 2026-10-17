package projectrun

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const runtimeFixtureArg = "-markitect-runtime-fixture="

// Each test owns a descriptor outside its repository snapshots. Only the
// re-executed child loads its environment; parallel parents never use Setenv.
type runtimeProcessFixture struct {
	path string
	env  map[string]string
}

var runtimeProcessFixtures sync.Map

func runtimeFixture(t *testing.T) *runtimeProcessFixture {
	t.Helper()
	if fixture, ok := runtimeProcessFixtures.Load(t); ok {
		return fixture.(*runtimeProcessFixture)
	}
	dir := testkit.TempDir(t)
	temp := filepath.Join(dir, "tmp")
	if err := os.Mkdir(temp, 0700); err != nil {
		t.Fatal(err)
	}
	fixture := &runtimeProcessFixture{path: filepath.Join(dir, "process.json"), env: map[string]string{
		e2eExecutorEnv: "", e2eCheckEnv: "", e2eLogEnv: filepath.Join(dir, "requests.jsonl"),
		e2eBehaviorEnv: "", e2eRootEnv: "", e2eOmitUsageEnv: "", e2eIntegrationCheckEnv: "",
		"TEMP": temp, "TMP": temp, "TMPDIR": temp,
	}}
	// Checks receive a deliberately reduced production environment. Restore
	// the test process's hermetic Git configuration in this fixture child only.
	for _, key := range []string{testkit.HomeVariable, "HOME", "XDG_CONFIG_HOME", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM", "GIT_TERMINAL_PROMPT", "GOCACHE", "GOMODCACHE", "GOPATH", "GOENV"} {
		if value, ok := os.LookupEnv(key); ok {
			fixture.env[key] = value
		}
	}
	runtimeProcessFixtures.Store(t, fixture)
	t.Cleanup(func() { runtimeProcessFixtures.Delete(t) })
	writeRuntimeFixture(t, fixture)
	return fixture
}

func writeRuntimeFixture(t *testing.T, fixture *runtimeProcessFixture) {
	t.Helper()
	data, err := json.Marshal(fixture.env)
	if err != nil {
		t.Fatal(err)
	}
	// Setup changes happen between invocations. Rename keeps a child from ever
	// reading partial configuration, including when a check starts concurrently.
	path := fixture.path + ".new"
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, fixture.path); err != nil {
		t.Fatal(err)
	}
}

func setRuntimeFixtureEnv(t *testing.T, key, value string) {
	t.Helper()
	fixture := runtimeFixture(t)
	fixture.env[key] = value
	writeRuntimeFixture(t, fixture)
}

func runtimeFixtureEnv(t *testing.T, key string) string {
	t.Helper()
	return runtimeFixture(t).env[key]
}

func runtimeFixtureArgs(t *testing.T, args []string) []string {
	t.Helper()
	result := append([]string(nil), args...)
	for _, arg := range result {
		if strings.HasPrefix(arg, runtimeFixtureArg) {
			return result
		}
	}
	return append(result, runtimeFixtureArg+runtimeFixture(t).path)
}

func runtimeFixtureCommand(t *testing.T, executable string, args ...string) string {
	t.Helper()
	data, err := json.Marshal(append([]string{executable}, runtimeFixtureArgs(t, args)...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data) // JSON is a quoted YAML flow sequence, including Windows paths.
}

// Called before TestMain dispatch or testing flag parsing. Only descriptor
// children load scenario settings; no production environment policy changes.
func prepareRuntimeFixtureProcess() error {
	args := os.Args[:1]
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, runtimeFixtureArg) {
			args = append(args, arg)
			continue
		}
		data, err := os.ReadFile(strings.TrimPrefix(arg, runtimeFixtureArg))
		if err != nil {
			return fmt.Errorf("read runtime fixture: %w", err)
		}
		var env map[string]string
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("decode runtime fixture: %w", err)
		}
		for key, value := range env {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	os.Args = args
	return nil
}

// Like testkit's Git setup, this is immutable package-process setup before
// m.Run. The parent's check resolver must find and pin the bare test binary
// on its allowlisted PATH. Every scenario uses that same binary; no parallel
// test changes PATH, and the setting dies with this test process.
func initializeRuntimeFixturePATH() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return os.Setenv("PATH", filepath.Dir(executable)+string(os.PathListSeparator)+os.Getenv("PATH"))
}
