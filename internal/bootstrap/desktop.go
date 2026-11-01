package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
		Chats:    desktopChats{},
		Settings: desktopSettings,
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
		Chats:        desktopChats{},
		Settings:     desktopSettings,
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

// editTemp names the temporary file of a hand edit, next to its target.
const editTemp = ".markitect-edit-*"

// desktopTree is the working tree with the stamp the app's views follow it
// by.
type desktopTree struct{ repo.WorkingTree }

// Write writes a file of the working tree, as a hand edit in the Reader
// does. It refuses a path that runs through a symbolic link or a Windows
// junction, either of which could lead out of the repository, and replaces
// the file in one step, so the views never read half of it.
func (t desktopTree) Write(file string, data []byte) error {
	walked := t.Dir
	for _, part := range strings.Split(file, "/") {
		walked = filepath.Join(walked, part)
		info, err := os.Lstat(walked)
		if errors.Is(err, fs.ErrNotExist) {
			break
		}
		if err != nil {
			return err
		}
		if info.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			return fmt.Errorf("%s runs through a symbolic link; Markitect writes no file there", file)
		}
	}
	full := filepath.Join(t.Dir, filepath.FromSlash(file))
	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// A write cut off before its rename left its temporary file; it goes now.
	if left, err := filepath.Glob(filepath.Join(dir, editTemp)); err == nil {
		for _, f := range left {
			os.Remove(f)
		}
	}
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(full); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(dir, editTemp)
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), full); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

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
