package bootstrap

import (
	"context"
	"os"
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
// open in the folder, so that checklist and answer stay the Run's while one is.
func (runs) OpenFolder(ctx context.Context, dir string) (*runengine.Engine, error) {
	e := withFolder(&runengine.Engine{})
	if git, err := rungit.Open(ctx, dir); err == nil {
		e.Git, e.Store = git, runengine.Store{Dir: filepath.Join(git.Dir, "markitect")}
	}
	return e, nil
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
