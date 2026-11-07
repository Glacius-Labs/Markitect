// Package bootstrap constructs the command line with its adapters: the project
// folder on disk, the work over it with its state in the user's cache folder, and
// the releases update installs. Nothing here uses Git (DEC-097).
package bootstrap

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/folder"
	"github.com/Glacius-Labs/Markitect/internal/adapters/upgrade"
	runs "github.com/Glacius-Labs/Markitect/internal/run"
)

// Run runs the markitect command line and returns its exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return cli.CLI{Folders: folders{}, Work: work(), Stdin: os.Stdin, Stdout: stdout, Stderr: stderr, PathWarning: ownPathWarning, Releases: upgrade.New()}.Run(ctx, args)
}

// work is the engine of the work over a project folder, with its state in the
// user's cache folder; without one, the work says so when it needs it.
func work() *runs.Engine {
	e := &runs.Engine{Folder: folder.Disk{}}
	if dir, err := os.UserCacheDir(); err == nil {
		e.Cache = filepath.Join(dir, "markitect")
	}
	return e
}

// folders opens the project folders of the command line on disk.
type folders struct{}

func (folders) Project(dir string) (cli.Workspace, error) { return folder.Disk{}.Project(dir) }

func (folders) At(dir string) (cli.Workspace, error) { return folder.Disk{}.At(dir) }

// ownPathWarning checks the running binary against the PATH. A binary in the
// temporary folder is one `go run` or `go test` built, which no PATH should lead
// to, so it is left alone. So is a binary the plugin's launcher started: the
// launcher is what the PATH leads to, by design.
func ownPathWarning() string {
	exe, err := os.Executable()
	if err != nil || under(exe, os.TempDir()) || os.Getenv(cli.PluginEnv) != "" {
		return ""
	}
	return pathWarning(exec.LookPath, func() (string, error) { return exe, nil })
}

// under says whether path lies below dir, after resolving links and short names.
func under(path, dir string) bool {
	resolve := func(p string) string {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			p = r
		}
		return strings.ToLower(filepath.ToSlash(filepath.Clean(p))) + "/"
	}
	return strings.HasPrefix(resolve(path), resolve(dir))
}

// pathWarning says what is wrong when a shell does not find this binary under the
// name markitect: the apply and verify skills and the agents call it by that name.
// It is empty when the PATH leads to this very binary.
func pathWarning(look func(string) (string, error), executable func() (string, error)) string {
	exe, err := executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	found, err := look("markitect")
	if err != nil {
		return fmt.Sprintf("warning: markitect is not on your PATH, but the apply and verify skills and the agents call it by that name; add %s to the PATH", dir)
	}
	onPath, errPath := os.Stat(found)
	own, errOwn := os.Stat(exe)
	if errPath == nil && errOwn == nil && !os.SameFile(onPath, own) {
		return fmt.Sprintf("warning: the markitect on your PATH is %s, not this one (%s); the skills and agents call the one on the PATH, so remove it or put %s before it", found, exe, dir)
	}
	return ""
}
