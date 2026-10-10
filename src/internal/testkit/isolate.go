// Package testkit provides hermetic Git and file-system fixtures for tests.
//
// Only test files may import it; the architecture import check enforces
// this. It depends on the standard library alone.
package testkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Main isolates the test process with Isolate, runs the tests and exits.
// Use it as a package's TestMain:
//
//	func TestMain(m *testing.M) { testkit.Main(m) }
func Main(m *testing.M) {
	restore, err := Isolate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "testkit:", err)
		os.Exit(2)
	}
	code := m.Run()
	restore()
	os.Exit(code)
}

// GlobalConfig is the global Git configuration of an isolated test process.
// Fixture repositories may override it in their own configuration.
const GlobalConfig = `[user]
	name = Markitect Test
	email = markitect-test@example.invalid
[init]
	defaultBranch = main
[core]
	autocrlf = false
	fsmonitor = false
[commit]
	gpgsign = false
[tag]
	gpgsign = false
[gc]
	auto = 0
[maintenance]
	auto = false
`

// goLocations keep the Go build and module caches and the go env file when
// HOME changes, so that tests which run the go command stay warm and offline.
var goLocations = []string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOENV"}

// Isolate shields the process, and every git or go process it starts, from
// the user's and the machine's Git configuration. It removes inherited GIT_*
// variables and points HOME and XDG_CONFIG_HOME at a fresh directory whose
// global configuration is GlobalConfig. Production code that drops GIT_*
// variables before running git still reads that global configuration through
// HOME. Git's system configuration stays visible to such code; fixtures set
// what they rely on, such as core.autocrlf, in their repository config.
//
// Isolate keeps the Go cache locations of the original environment. The
// returned function restores the environment and removes the directory.
func Isolate() (func(), error) {
	goEnv, err := resolveGoLocations()
	if err != nil {
		return nil, err
	}
	root, err := os.MkdirTemp("", "mkhome-")
	if err != nil {
		return nil, fmt.Errorf("create isolated home: %w", err)
	}
	home := filepath.Join(root, "home")
	global := filepath.Join(home, ".gitconfig")
	if err := os.MkdirAll(filepath.Join(home, ".config", "git"), 0o700); err != nil {
		return nil, errors.Join(fmt.Errorf("create isolated home: %w", err), os.RemoveAll(root))
	}
	if err := os.WriteFile(global, []byte(GlobalConfig), 0o600); err != nil {
		return nil, errors.Join(fmt.Errorf("write isolated git config: %w", err), os.RemoveAll(root))
	}
	env := environment{}
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			env.set(name, nil)
		}
	}
	for name, value := range goEnv {
		env.set(name, &value)
	}
	for name, value := range map[string]string{
		"HOME":                home,
		"XDG_CONFIG_HOME":     filepath.Join(home, ".config"),
		"GIT_CONFIG_GLOBAL":   global,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		env.set(name, &value)
	}
	return func() {
		env.restore()
		removeAll(root)
	}, nil
}

// environment records the original value of each variable it changes.
type environment map[string]*string

func (e environment) set(name string, value *string) {
	if _, recorded := e[name]; !recorded {
		if original, ok := os.LookupEnv(name); ok {
			e[name] = &original
		} else {
			e[name] = nil
		}
	}
	if value == nil {
		os.Unsetenv(name)
	} else {
		os.Setenv(name, *value)
	}
}

func (e environment) restore() {
	for name, original := range e {
		e.set(name, original)
	}
}

// resolveGoLocations asks the go command for the cache locations of the
// current environment, which may come from defaults or a go env file. It
// returns no locations when the go command is not available.
func resolveGoLocations() (map[string]string, error) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		return nil, nil
	}
	out, err := exec.Command(goTool, append([]string{"env", "-json"}, goLocations...)...).Output()
	if err != nil {
		return nil, fmt.Errorf("resolve go cache locations: %w", err)
	}
	values := map[string]string{}
	if err := json.Unmarshal(out, &values); err != nil {
		return nil, fmt.Errorf("resolve go cache locations: %w", err)
	}
	for name, value := range values {
		if value == "" {
			delete(values, name)
		}
	}
	return values, nil
}
