package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/agents/claude"
	"github.com/Glacius-Labs/Markitect/internal/agents/codex"
)

// providerAgents builds the Manager agents from the environment
// (internal/agents/README.md#composition):
//
//	MARKITECT_ENGINE         the Docker CLI (default docker on PATH)
//	MARKITECT_MANAGER_IMAGE  the Manager image, a project image FROM packaging/runtime (required)
//	MARKITECT_PROVIDER       claude (default) or codex
//	MARKITECT_LOGIN          the account login file (default the provider's own in the home folder)
//	MARKITECT_MODEL          the model (default claude-sonnet-5-5, or gpt-6-luna for codex)
//	MARKITECT_EFFORT         the effort (default high)
//	MARKITECT_NETWORK        the network that reaches the provider (default markitect-provider,
//	                         a bridge network the host creates on first use)
//	MARKITECT_HOSTS          extra host entries, name:address, comma-separated
//	MARKITECT_DNS            name servers, comma-separated
//	MARKITECT_TRANSCRIPTS    a folder for the agents' transcripts (default none)
func providerAgents(ctx context.Context) (agentport.Agents, error) {
	cfg, _, err := agentConfig(ctx)
	if err != nil {
		return nil, err
	}
	return agents.New(cfg)
}

// agentConfig reads the agents' settings from the environment, as
// providerAgents describes them, and returns the engine they run on. The
// app's chat agents run with the same settings.
func agentConfig(ctx context.Context) (agents.Config, *container.Engine, error) {
	ref := os.Getenv("MARKITECT_MANAGER_IMAGE")
	if ref == "" {
		return agents.Config{}, nil, errors.New("no Manager image: build one from packaging/runtime (with the project's toolchain) and set MARKITECT_MANAGER_IMAGE")
	}
	network := os.Getenv("MARKITECT_NETWORK")
	switch network {
	case "bridge", "host", "none", "default":
		return agents.Config{}, nil, fmt.Errorf("MARKITECT_NETWORK %q is no user-defined network: the agents also join the Run's network, which the Docker CLI on Linux refuses to combine with the engine's own; leave it unset for %s or name a user-defined one", network, providerNetwork)
	}
	cliPath := os.Getenv("MARKITECT_ENGINE")
	if cliPath == "" {
		p, err := exec.LookPath("docker")
		if err != nil {
			return agents.Config{}, nil, errors.New("no container engine: install Docker or set MARKITECT_ENGINE")
		}
		cliPath = p
	}
	engine, err := container.NewEngine(cliPath)
	if err != nil {
		return agents.Config{}, nil, err
	}
	image, err := engine.ImageID(ctx, ref)
	if err != nil {
		return agents.Config{}, nil, fmt.Errorf("the Manager image %s: %w", ref, err)
	}
	if network == "" {
		// The agents also join the Run's network, and the Docker CLI on
		// Linux refuses to combine the engine's default "bridge" with it.
		network = providerNetwork
		if err := engine.EnsureNetwork(ctx, network, map[string]string{"markitect.provider": "1"}); err != nil {
			return agents.Config{}, nil, fmt.Errorf("the provider network: %w; or name one with MARKITECT_NETWORK", err)
		}
	}
	cfg := agents.Config{
		Runtime: agents.EngineRuntime(engine), Image: image,
		Limits:  agents.Limits{CPUs: 1, MemoryBytes: 2 << 30, Pids: 256, TmpBytes: 256 << 20},
		Network: network, Hosts: list("MARKITECT_HOSTS"), DNS: list("MARKITECT_DNS"),
		Effort: env("MARKITECT_EFFORT", "high"), Transcripts: os.Getenv("MARKITECT_TRANSCRIPTS"),
	}
	p, ok := providers[env("MARKITECT_PROVIDER", "claude")]
	if !ok {
		return agents.Config{}, nil, fmt.Errorf("MARKITECT_PROVIDER %q is neither claude nor codex", os.Getenv("MARKITECT_PROVIDER"))
	}
	cfg.Provider, cfg.Model, cfg.LoginFile = p.provider, env("MARKITECT_MODEL", p.model), env("MARKITECT_LOGIN", p.loginFile())
	if _, err := os.Stat(cfg.LoginFile); err != nil {
		return agents.Config{}, nil, fmt.Errorf("the %s login %s: %w; log in with the provider's CLI on this host or set MARKITECT_LOGIN", cfg.Provider.Name(), cfg.LoginFile, err)
	}
	return cfg, engine, nil
}

// providerDefaults are one provider's adapter, default model, account login
// file in the home folder, and the command that logs in.
type providerDefaults struct {
	provider agents.Provider
	model    string
	login    []string // the login file below the home folder
	command  string
}

func (p providerDefaults) loginFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(append([]string{home}, p.login...)...)
}

var providers = map[string]providerDefaults{
	"claude": {claude.Provider{}, "claude-sonnet-5-5", []string{".claude", ".credentials.json"}, "claude"},
	"codex":  {codex.Provider{}, "gpt-6-luna", []string{".codex", "auth.json"}, "codex login"},
}

// providerNetwork is the user-defined bridge network the agents reach their
// provider through unless MARKITECT_NETWORK names another. It stays on the
// engine for later Runs, like an image.
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
