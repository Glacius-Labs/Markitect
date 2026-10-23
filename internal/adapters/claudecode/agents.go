package claudecode

import (
	"context"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
)

// Agents advertises the actual Run session boundary, while explicitly refusing
// unqualified Claude execution. The retired Turn engine is never invoked.
type Agents struct{}

var _ run.Sessions = (*Agents)(nil)

func NewAgents(engine *container.Engine, prepared runtimeconfig.PreparedProfile, credentials CredentialResolver) (*Agents, error) {
	if engine == nil || credentials == nil {
		return nil, ErrSession
	}
	if _, err := ProfileFromPrepared(prepared); err != nil {
		return nil, err
	}
	return &Agents{}, nil
}
func (*Agents) Open(context.Context, run.SessionSpec) (run.Session, error) {
	return nil, fmt.Errorf("%w: Claude native instruction loading, normal-login lifetime and message drain have not been qualified for the replacement Session consumer", run.ErrUnavailable)
}
