package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/agents/claude"
	"github.com/Glacius-Labs/Markitect/internal/agents/codex"
)

// TokenVariable is Markitect's own variable for the user's Claude account token,
// made with `claude setup-token` (DEC-093). It is Markitect's own so that it
// does not outrank the login of the user's other Claude sessions, as the CLI's
// variable set for the whole machine would.
const TokenVariable = "MARKITECT_CLAUDE_TOKEN"

// agentConfig is what a chat starts with, read from the environment (internal/agents/README.md#composition):
//
//	MARKITECT_ENGINE         the Docker CLI (default docker on PATH)
//	MARKITECT_MANAGER_IMAGE  the project's image, FROM packaging/runtime (required)
//	MARKITECT_PROVIDER       claude (default); codex has no token route yet
//	MARKITECT_CLAUDE_TOKEN   the user's token from `claude setup-token`
//	MARKITECT_MODEL          the model (default claude-sonnet-5-5)
//	MARKITECT_EFFORT         the effort (default high)
//	MARKITECT_NETWORK        the network that reaches the provider (default
//	                         markitect-provider, a bridge network the host creates
//	                         on first use)
//	MARKITECT_HOSTS          extra host entries, name:address, comma-separated
//	MARKITECT_DNS            name servers, comma-separated
type agentConfig struct {
	Runtime    agents.Runtime
	Provider   agents.Provider
	CLI        string
	Image      string
	Network    string
	Hosts, DNS []string
	Limits     agents.Limits
	Model      string
	Effort     string
	Token      string
}

// readAgentConfig reads the settings, as agentConfig describes them, and returns
// the engine they run on. build makes the image when none is set; without build,
// none set is an error.
func readAgentConfig(ctx context.Context, build func(context.Context, *container.Engine) (string, error)) (agentConfig, *container.Engine, error) {
	ref := os.Getenv("MARKITECT_MANAGER_IMAGE")
	if ref == "" && build == nil {
		return agentConfig{}, nil, errors.New("no image: build one from packaging/runtime (with the project's toolchain) and set MARKITECT_MANAGER_IMAGE")
	}
	network := os.Getenv("MARKITECT_NETWORK")
	switch network {
	case "bridge", "host", "none", "default":
		return agentConfig{}, nil, fmt.Errorf("MARKITECT_NETWORK %q is no user-defined network; leave it unset for %s or name a user-defined one", network, providerNetwork)
	}
	cliPath := os.Getenv("MARKITECT_ENGINE")
	if cliPath == "" {
		p, err := exec.LookPath("docker")
		if err != nil {
			return agentConfig{}, nil, errors.New("no container engine: install Docker or set MARKITECT_ENGINE")
		}
		cliPath = p
	}
	engine, err := container.NewEngine(cliPath)
	if err != nil {
		return agentConfig{}, nil, err
	}
	cfg := agentConfig{
		Runtime: agents.EngineRuntime(engine), CLI: cliPath,
		Limits: agents.Limits{CPUs: 1, MemoryBytes: 2 << 30, Pids: 256, TmpBytes: 256 << 20},
		Hosts:  list("MARKITECT_HOSTS"), DNS: list("MARKITECT_DNS"),
		Effort: env("MARKITECT_EFFORT", "high"),
	}
	p, ok := providers[env("MARKITECT_PROVIDER", "claude")]
	if !ok {
		return agentConfig{}, nil, fmt.Errorf("MARKITECT_PROVIDER %q is neither claude nor codex", os.Getenv("MARKITECT_PROVIDER"))
	}
	if p.provider.TokenVariable() == "" {
		return agentConfig{}, nil, fmt.Errorf("%s has no account token route yet; use claude", p.provider.Name())
	}
	cfg.Provider, cfg.Model, cfg.Token = p.provider, env("MARKITECT_MODEL", p.model), os.Getenv(p.variable)
	if cfg.Token == "" {
		return agentConfig{}, nil, fmt.Errorf("no account token: run `%s`, keep the token in %s, and never set it for the whole machine under the CLI's own name", p.command, p.variable)
	}
	// Last, since a build takes minutes and the settings above can still fail.
	var image string
	if ref != "" {
		if image, err = engine.ImageID(ctx, ref); err != nil {
			return agentConfig{}, nil, fmt.Errorf("the image %s: %w", ref, err)
		}
	} else if image, err = build(ctx, engine); err != nil {
		return agentConfig{}, nil, err
	}
	if network == "" {
		network = providerNetwork
		if err := engine.EnsureNetwork(ctx, network, map[string]string{"markitect.provider": "1"}); err != nil {
			return agentConfig{}, nil, fmt.Errorf("the provider network: %w; or name one with MARKITECT_NETWORK", err)
		}
	}
	cfg.Image, cfg.Network = image, network
	return cfg, engine, nil
}

// providerDefaults are one provider's adapter, default model, the variable that
// holds the user's token, and the command that makes one.
type providerDefaults struct {
	provider agents.Provider
	model    string
	variable string
	command  string
}

var providers = map[string]providerDefaults{
	"claude": {claude.Provider{}, "claude-sonnet-5-5", TokenVariable, "claude setup-token"},
	"codex":  {codex.Provider{}, "gpt-6-luna", "", ""},
}

// providerNetwork is the user-defined bridge network the sessions reach their
// provider through unless MARKITECT_NETWORK names another. It stays on the
// engine for later chats, like an image.
const providerNetwork = "markitect-provider"

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func list(name string) []string {
	var out []string
	for _, v := range strings.Split(os.Getenv(name), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
