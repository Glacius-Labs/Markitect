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
//	MARKITECT_MODEL          the model (default claude-sonnet-5-5, or gpt-6-sol for codex)
//	MARKITECT_EFFORT         the effort (default high)
//	MARKITECT_NETWORK        the network that reaches the provider (default bridge)
//	MARKITECT_HOSTS          extra host entries, name:address, comma-separated
//	MARKITECT_DNS            name servers, comma-separated
//	MARKITECT_TRANSCRIPTS    a folder for the agents' transcripts (default none)
func providerAgents(ctx context.Context) (agentport.Agents, error) {
	ref := os.Getenv("MARKITECT_MANAGER_IMAGE")
	if ref == "" {
		return nil, errors.New("no Manager image: build one from packaging/runtime (with the project's toolchain) and set MARKITECT_MANAGER_IMAGE")
	}
	cliPath := os.Getenv("MARKITECT_ENGINE")
	if cliPath == "" {
		p, err := exec.LookPath("docker")
		if err != nil {
			return nil, errors.New("no container engine: install Docker or set MARKITECT_ENGINE")
		}
		cliPath = p
	}
	engine, err := container.NewEngine(cliPath)
	if err != nil {
		return nil, err
	}
	image, err := engine.ImageID(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("the Manager image %s: %w", ref, err)
	}
	home, _ := os.UserHomeDir()
	cfg := agents.Config{
		Runtime: agents.EngineRuntime(engine), Image: image,
		Limits:  agents.Limits{CPUs: 1, MemoryBytes: 2 << 30, Pids: 256, TmpBytes: 256 << 20},
		Network: env("MARKITECT_NETWORK", "bridge"), Hosts: list("MARKITECT_HOSTS"), DNS: list("MARKITECT_DNS"),
		Effort: env("MARKITECT_EFFORT", "high"), Transcripts: os.Getenv("MARKITECT_TRANSCRIPTS"),
	}
	switch provider := env("MARKITECT_PROVIDER", "claude"); provider {
	case "claude":
		cfg.Provider, cfg.Model = claude.Provider{}, env("MARKITECT_MODEL", "claude-sonnet-5-5")
		cfg.LoginFile = env("MARKITECT_LOGIN", filepath.Join(home, ".claude", ".credentials.json"))
	case "codex":
		cfg.Provider, cfg.Model = codex.Provider{}, env("MARKITECT_MODEL", "gpt-6-sol")
		cfg.LoginFile = env("MARKITECT_LOGIN", filepath.Join(home, ".codex", "auth.json"))
	default:
		return nil, fmt.Errorf("MARKITECT_PROVIDER %q is neither claude nor codex", provider)
	}
	if _, err := os.Stat(cfg.LoginFile); err != nil {
		return nil, fmt.Errorf("the %s login %s: %w; log in with the provider's CLI on this host or set MARKITECT_LOGIN", cfg.Provider.Name(), cfg.LoginFile, err)
	}
	return agents.New(cfg)
}

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
