// Package bootstrap constructs the command line with its adapters: Git for the
// spec's commands, and for a Run the process over the user's own repository.
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
	"github.com/Glacius-Labs/Markitect/internal/adapters/repo"
)

// Run runs the markitect command line and returns its exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return cli.CLI{Git: git{}, Runs: runs{}, Stdin: os.Stdin, Stdout: stdout, Stderr: stderr, PathWarning: ownPathWarning}.Run(ctx, args)
}

// ownPathWarning checks the running binary against the PATH. A binary in the
// temporary folder is one `go run` or `go test` built, which no PATH should lead
// to, so it is left alone.
func ownPathWarning() string {
	exe, err := os.Executable()
	if err != nil || under(exe, os.TempDir()) {
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

// git opens repository states through the repo adapter.
type git struct{}

func (git) Open(ctx context.Context, dir string) (cli.Workspace, error) {
	root, err := repo.Toplevel(ctx, dir)
	if err != nil {
		return nil, err
	}
	return repo.WorkingTree{Dir: root, Ctx: ctx}, nil
}

func (git) Create(ctx context.Context, dir string) error { return repo.Init(ctx, dir) }

func (git) Base(ctx context.Context, ws cli.Workspace, rev string) (cli.Base, error) {
	return repo.ResolveCommit(ctx, ws.Root(), rev)
}
