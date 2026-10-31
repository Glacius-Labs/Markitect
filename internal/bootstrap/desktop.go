package bootstrap

import (
	"context"
	"fmt"
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
	dir, err := DesktopFolder()
	if err != nil {
		return err
	}
	return desktop.Start(ctx, desktop.Config{
		Dir:      dir,
		Listen:   o.Listen,
		Open:     o.Open,
		Repos:    desktopRepos{},
		Commands: desktopCommands{},
		Stdout:   d.stdout,
	})
}

// DesktopFolder is the user's config folder for Markitect Desktop: its
// record, its project list and the native window's web view data.
func DesktopFolder() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Markitect"), nil
}

// DesktopService is Markitect Desktop's running service as its native shell
// holds it.
type DesktopService = desktop.Service

// DesktopWindows is what the native shell gives the service: its windows and
// its folder dialog.
type DesktopWindows struct {
	// Open opens a window on url.
	Open func(url string) error
	// ChooseFolder asks for a folder, starting at start; "" means cancelled.
	ChooseFolder func(ctx context.Context, start string) (string, error)
}

// ListenDesktop starts Markitect Desktop for its native shell, on a free
// loopback port. When this user's service already runs, that one opens the
// window and ListenDesktop returns nil. Otherwise it returns the new service,
// which opens its first window once Serve runs.
func ListenDesktop(ctx context.Context, w DesktopWindows) (*DesktopService, error) {
	dir, err := DesktopFolder()
	if err != nil {
		return nil, err
	}
	return desktop.Listen(ctx, desktop.Config{
		Dir:          dir,
		Listen:       "127.0.0.1:0",
		Open:         true,
		Repos:        desktopRepos{},
		Commands:     desktopCommands{},
		Stdout:       io.Discard,
		OpenWindow:   w.Open,
		ChooseFolder: w.ChooseFolder,
	})
}

// desktopRepos opens repositories through the repo adapter, as the command
// line does.
type desktopRepos struct{}

func (desktopRepos) Open(ctx context.Context, dir string) (desktop.Workspace, error) {
	root, err := repo.Toplevel(ctx, dir)
	if err != nil {
		return nil, err
	}
	return desktopTree{repo.WorkingTree{Dir: root, Ctx: ctx}}, nil
}

func (desktopRepos) Commit(ctx context.Context, root, rev string) (desktop.Commit, error) {
	return repo.ResolveCommit(ctx, root, rev)
}

func (desktopRepos) LastLanded(ctx context.Context, root, tip string) (string, bool, error) {
	return repo.LastLanded(ctx, root, tip)
}

// desktopTree is the working tree with the stamp the app's views follow it
// by.
type desktopTree struct{ repo.WorkingTree }

func (t desktopTree) Stamp(file string) string {
	info, err := os.Lstat(filepath.Join(t.Dir, filepath.FromSlash(file)))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d %d", info.Size(), info.ModTime().UnixNano())
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
