// Package bootstrap constructs the command line with its Git adapter.
package bootstrap

import (
	"context"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/repo"
)

// Run runs the markitect command line and returns its exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return cli.CLI{Git: git{}, Stdout: stdout, Stderr: stderr}.Run(ctx, args)
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

func (git) Base(ctx context.Context, ws cli.Workspace, rev string) (cli.Base, error) {
	return repo.ResolveCommit(ctx, ws.Root(), rev)
}
