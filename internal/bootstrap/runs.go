package bootstrap

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/adapters/claude"
	"github.com/Glacius-Labs/Markitect/internal/adapters/hostcheck"
	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	runengine "github.com/Glacius-Labs/Markitect/internal/run"
)

// runs builds the engine of a Run: the user's own repository, the required checks
// on the user's machine and the provider's background sessions.
type runs struct{}

func (runs) Open(ctx context.Context, dir string) (*runengine.Engine, error) {
	repo, err := rungit.Open(ctx, dir)
	if err != nil {
		return nil, err
	}
	return newEngine(repo, filepath.Join(repo.Dir, "markitect")), nil
}

// OpenStore builds the engine for a hook, which knows the Run's store. Its Git
// directory is the store's parent.
func (runs) OpenStore(store string) *runengine.Engine {
	store = filepath.FromSlash(store)
	return newEngine(&rungit.Repo{Dir: filepath.Dir(store)}, store)
}

func newEngine(repo *rungit.Repo, store string) *runengine.Engine {
	exe, err := os.Executable()
	if err != nil {
		exe = "markitect"
	}
	return &runengine.Engine{Git: repo, Checks: hostcheck.Runner{}, Sessions: claude.CLI{Executable: exe}, Store: runengine.Store{Dir: store}}
}
