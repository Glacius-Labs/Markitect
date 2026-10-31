package bootstrap

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/desktop"
	"github.com/Glacius-Labs/Markitect/internal/adapters/repo"
)

// desktopApp runs Markitect Desktop over the same Git and the same commands
// as the command line.
type desktopApp struct{ stdout io.Writer }

func (d desktopApp) Run(ctx context.Context, o cli.DesktopOptions) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	return desktop.Start(ctx, desktop.Config{
		Dir:      filepath.Join(dir, "Markitect"),
		Listen:   o.Listen,
		Open:     o.Open,
		Repos:    desktopRepos{},
		Commands: desktopCommands{},
		Stdout:   d.stdout,
	})
}

// desktopRepos opens repositories through the repo adapter, as the command
// line does.
type desktopRepos struct{}

func (desktopRepos) Open(ctx context.Context, dir string) (desktop.Workspace, error) {
	ws, err := git{}.Open(ctx, dir)
	if err != nil {
		return nil, err
	}
	return ws, nil
}

func (desktopRepos) Init(ctx context.Context, dir string) error { return repo.Init(ctx, dir) }

func (desktopRepos) Clone(ctx context.Context, url, dir string) error {
	return repo.Clone(ctx, url, dir)
}

// desktopCommands runs the command line's own commands, so the app's init is
// markitect init with its output.
type desktopCommands struct{}

func (desktopCommands) Init(ctx context.Context, dir string, out io.Writer) (bool, error) {
	code := cli.CLI{Git: git{}, Stdout: out, Stderr: out}.Run(ctx, []string{"init", "--repo", dir})
	return code == cli.OK, nil
}
