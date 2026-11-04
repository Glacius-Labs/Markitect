package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	"github.com/Glacius-Labs/Markitect/internal/adapters/sandbox"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	runengine "github.com/Glacius-Labs/Markitect/internal/run"
)

// sandboxLimits bound a Command's container. It holds the implementer, its
// subagents, the required checks and the reviewers, so it gets more than a chat.
var sandboxLimits = sandbox.Limits{CPUs: "4", Memory: "8g", Pids: 2048, Tmp: "1g"}

// runs builds one Command engine per command: the host's Git over the target
// repository, the Command's one container and the provider's sessions.
type runs struct{}

func (runs) Run(ctx context.Context, o cli.RunOptions) (*runengine.Result, error) {
	cfg, _, err := readAgentConfig(ctx)
	if err != nil {
		return nil, err
	}
	budget := 0.0
	if v := os.Getenv("MARKITECT_SESSION_BUDGET"); v != "" {
		if budget, err = strconv.ParseFloat(v, 64); err != nil || budget < 0 {
			return nil, fmt.Errorf("MARKITECT_SESSION_BUDGET %q is no amount", v)
		}
	}
	sessions, err := agents.NewSessions(cfg.Provider, cfg.Model, cfg.Effort, cfg.Token, cfg.Transcripts, budget)
	if err != nil {
		return nil, err
	}
	engine, err := sandbox.NewEngine(cfg.CLI)
	if err != nil {
		return nil, err
	}
	if o.Request.Mode == agentport.Apply {
		// A Run starts from a commit, so an uncommitted spec would be silently
		// left out (concept 3.0 §14).
		dirty, err := rungit.UncommittedSpec(ctx, o.Repo)
		if err != nil {
			return nil, err
		}
		if len(dirty) > 0 {
			return nil, fmt.Errorf("%w: the spec has uncommitted changes, which a Run would leave out; commit them first: %v", runengine.ErrUsage, dirty)
		}
	}
	target, branch, err := rungit.Target(ctx, o.Repo, o.Remote, o.Branch)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "markitect-run-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	host, err := rungit.Open(ctx, filepath.Join(dir, "host.git"), target)
	if err != nil {
		return nil, err
	}
	o.Request.Target = branch
	e := &runengine.Engine{Git: host, Agents: sessions, Secrets: [][]byte{[]byte(cfg.Token)},
		Sandboxes: &sandbox.Sandboxes{Engine: engine, Image: cfg.Image, Network: cfg.Network, Hosts: cfg.Hosts, DNS: cfg.DNS, Limits: sandboxLimits}}
	return e.Run(ctx, o.Request)
}
