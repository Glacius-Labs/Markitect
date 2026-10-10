// Package testkit provides hermetic Git and file-system fixtures for tests.
//
// Only test files may import it; the architecture import check enforces
// this. It depends on the standard library alone.
package testkit

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

// HomeVariable holds the isolated home directory. Child processes that
// inherit it, or that inherit the isolated HOME, are already isolated.
const HomeVariable = "MARKITECT_TESTKIT_HOME"

// Isolate shields the process, and every git or go process it starts, from
// the user's and the machine's Git configuration. It removes inherited GIT_*
// variables and points HOME and XDG_CONFIG_HOME at a dedicated directory whose
// global configuration is GlobalConfig. Production code that drops GIT_*
// variables before running git still reads that global configuration through
// HOME. Git's system configuration stays visible to such code; fixtures set
// what they rely on, such as core.autocrlf, in their repository config.
//
// Isolate keeps the Go cache locations of the original environment. The
// returned function restores the environment.
//
// A re-executed test binary, such as a fake agent, does nothing here: it
// either inherits its parent's isolation or runs with a reduced environment
// that has no home or temporary directory, where git reads no global
// configuration.
func Isolate() (func(), error) {
	if inheritsIsolation() || reducedEnvironment() {
		return func() {}, nil
	}
	goEnv := goLocations()
	home, err := isolatedHome()
	if err != nil {
		return nil, err
	}
	global := filepath.Join(home, ".gitconfig")
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
		HomeVariable:          home,
		"HOME":                home,
		"XDG_CONFIG_HOME":     filepath.Join(home, ".config"),
		"GIT_CONFIG_GLOBAL":   global,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		env.set(name, &value)
	}
	return env.restore, nil
}

// isolatedHome returns the isolated home directory of this user and this
// GlobalConfig, creating it when needed. All test processes share it, and
// its content never changes, so nothing removes it and nothing leaks when a
// process exits early.
func isolatedHome() (string, error) {
	sum := sha256.Sum256([]byte(GlobalConfig))
	name := fmt.Sprintf("markitect-testkit-%x", sum[:4])
	if uid := os.Getuid(); uid >= 0 {
		name = fmt.Sprintf("markitect-testkit-%d-%x", uid, sum[:4])
	}
	home := filepath.Join(os.TempDir(), name, "home")
	global := filepath.Join(home, ".gitconfig")
	if configured(global) {
		return home, nil
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "git"), 0o700); err != nil {
		return "", fmt.Errorf("create isolated home: %w", err)
	}
	// Concurrent test processes must never read a partial file: write a
	// unique file, then rename it into place.
	file, err := os.CreateTemp(home, ".gitconfig-*")
	if err != nil {
		return "", fmt.Errorf("write isolated git config: %w", err)
	}
	defer os.Remove(file.Name())
	_, writeErr := file.WriteString(GlobalConfig)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return "", fmt.Errorf("write isolated git config: %w", err)
	}
	// On Windows the rename fails while another process reads or replaces
	// the file; that process then provides the same content.
	deadline := time.Now().Add(cleanupDeadline)
	for {
		err := os.Rename(file.Name(), global)
		if err == nil || configured(global) {
			return home, nil
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("write isolated git config: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// configured reports whether the file holds GlobalConfig.
func configured(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && string(data) == GlobalConfig
}

// inheritsIsolation reports whether an ancestor process already isolated
// this one, through HomeVariable or an inherited HOME.
func inheritsIsolation() bool {
	for _, home := range []string{os.Getenv(HomeVariable), os.Getenv("HOME")} {
		if home == "" {
			continue
		}
		if data, err := os.ReadFile(filepath.Join(home, ".gitconfig")); err == nil && string(data) == GlobalConfig {
			return true
		}
	}
	return false
}

// reducedEnvironment reports whether the process received an explicit,
// reduced environment from its parent: no temporary directory on Windows,
// no HOME elsewhere.
func reducedEnvironment() bool {
	if runtime.GOOS == "windows" {
		return os.Getenv("TEMP") == "" && os.Getenv("TMP") == ""
	}
	return os.Getenv("HOME") == ""
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

// goLocations returns the Go cache and configuration locations of the
// current environment the way the go command derives them: from the
// environment, then the go env file, then the defaults. Isolate pins them so
// that go commands in tests keep their warm caches when HOME changes.
func goLocations() map[string]string {
	values := map[string]string{}
	goenv := os.Getenv("GOENV")
	if goenv == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			goenv = filepath.Join(dir, "go", "env")
		}
	}
	file := map[string]string{}
	if goenv != "" {
		values["GOENV"] = goenv
		if data, err := os.ReadFile(goenv); err == nil && goenv != "off" {
			for _, line := range strings.Split(string(data), "\n") {
				if name, value, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
					file[name] = value
				}
			}
		}
	}
	lookup := func(name string) string {
		if value := os.Getenv(name); value != "" {
			return value
		}
		return file[name]
	}
	if cache := lookup("GOCACHE"); cache != "" {
		values["GOCACHE"] = cache
	} else if dir, err := os.UserCacheDir(); err == nil {
		values["GOCACHE"] = filepath.Join(dir, "go-build")
	}
	gopath := lookup("GOPATH")
	if gopath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			gopath = filepath.Join(home, "go")
		}
	}
	if gopath != "" {
		values["GOPATH"] = gopath
	}
	if modcache := lookup("GOMODCACHE"); modcache != "" {
		values["GOMODCACHE"] = modcache
	} else if gopath != "" {
		values["GOMODCACHE"] = filepath.Join(filepath.SplitList(gopath)[0], "pkg", "mod")
	}
	return values
}
