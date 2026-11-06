package bootstrap

import (
	"context"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	runengine "github.com/Glacius-Labs/Markitect/internal/run"
)

// runs builds the engine of a Run over the user's own repository. It starts no
// agent: the owner's session does.
type runs struct{}

func (runs) Open(ctx context.Context, dir string) (*runengine.Engine, error) {
	repo, err := rungit.Open(ctx, dir)
	if err != nil {
		return nil, err
	}
	return &runengine.Engine{Git: repo, Store: runengine.Store{Dir: filepath.Join(repo.Dir, "markitect")}}, nil
}
