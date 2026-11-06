package bootstrap

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/adapters/repo"
	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	runengine "github.com/Glacius-Labs/Markitect/internal/run"
)

// runs builds the engine of a Run over the user's own repository, and the engine
// of the work without a Run over a project folder. It starts no agent: the
// owner's session does.
type runs struct{}

func (runs) Open(ctx context.Context, dir string) (*runengine.Engine, error) {
	git, err := rungit.Open(ctx, dir)
	if err != nil {
		return nil, err
	}
	e := withFolder(&runengine.Engine{Git: git, Store: runengine.Store{Dir: filepath.Join(git.Dir, "markitect")}})
	return e, nil
}

// OpenFolder needs no Git. Inside a Git repository it still sees whether a Run is
// open in the folder, so that checklist and answer stay the Run's while one is;
// a repository that Git cannot open there is an error, not a folder without one.
func (runs) OpenFolder(ctx context.Context, dir string) (*runengine.Engine, error) {
	e := withFolder(&runengine.Engine{})
	git, err := rungit.Open(ctx, dir)
	switch {
	case err == nil:
		e.Git, e.Store = git, runengine.Store{Dir: filepath.Join(git.Dir, "markitect")}
	case errors.Is(err, exec.ErrNotFound):
		// No Git on this machine, so no Run either.
	case insideRepository(dir):
		return nil, err
	}
	return e, nil
}

// insideRepository says whether dir or a folder above it holds a .git.
func insideRepository(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	for at := abs; ; {
		if _, err := os.Lstat(filepath.Join(at, ".git")); err == nil {
			return true
		}
		up := filepath.Dir(at)
		if up == at {
			return false
		}
		at = up
	}
}

// withFolder adds the project folder and the cache of its work, the user's cache
// folder; without one, the work without a Run says so when it needs it.
func withFolder(e *runengine.Engine) *runengine.Engine {
	e.Folder = repo.Folder{}
	if dir, err := os.UserCacheDir(); err == nil {
		e.Cache = filepath.Join(dir, "markitect")
	}
	return e
}
