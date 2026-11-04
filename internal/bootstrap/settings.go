package bootstrap

import (
	"context"
	"os"
	"os/exec"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/adapters/desktop"
)

// desktopSettings reads what Markitect Desktop's Settings shows when the app
// has no image of its own to build.
func desktopSettings(context.Context) desktop.AppSettings { return appSettings(nil) }

// appSettings reads the agents' settings from the environment, as agentConfig
// reads them but without asking the container engine, and whether each provider
// has its account token. The token's value goes nowhere from here. With no Manager image set, chats run in the app's own image.
func appSettings(image *chatImage) desktop.AppSettings {
	name := env("MARKITECT_PROVIDER", "claude")
	a := desktop.AgentSettings{Provider: name, Effort: env("MARKITECT_EFFORT", "high"), Image: os.Getenv("MARKITECT_MANAGER_IMAGE"),
		Network: env("MARKITECT_NETWORK", providerNetwork), Engine: os.Getenv("MARKITECT_ENGINE"), Problems: []string{}}
	used, ok := providers[name]
	if ok {
		a.Model = env("MARKITECT_MODEL", used.model)
	} else {
		a.Problems = append(a.Problems, "MARKITECT_PROVIDER names "+name+", which is neither claude nor codex.")
	}
	switch a.Network {
	case "bridge", "host", "none", "default":
		a.Problems = append(a.Problems, "MARKITECT_NETWORK names "+a.Network+", the engine's own network; leave it unset or name a user-defined one.")
	}
	if a.Image == "" && image != nil {
		tag, err := image.tag()
		if err != nil {
			a.Problems = append(a.Problems, "The agent image: "+err.Error())
		}
		a.ImageBuilt = tag
	} else if a.Image == "" {
		a.Problems = append(a.Problems, "No Manager image: build one from packaging/runtime with the project's toolchain, and set MARKITECT_MANAGER_IMAGE before you start Markitect.")
	}
	if a.Engine == "" {
		if p, err := exec.LookPath("docker"); err == nil {
			a.Engine = p
		} else {
			a.Problems = append(a.Problems, "No container engine: install Docker, or set MARKITECT_ENGINE to its command line.")
		}
	}
	out := desktop.AppSettings{Agents: a}
	names := make([]string, 0, len(providers))
	for n := range providers {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		p := providers[n]
		l := desktop.LoginState{Provider: n, File: p.variable, Command: p.command, Used: n == name}
		switch {
		case p.variable == "":
			l.State, l.Detail = "refused", "Markitect has no account token route for "+n+" yet."
		case os.Getenv(p.variable) == "":
			l.State = "missing"
		default:
			l.State = "ok"
		}
		if l.Used && l.State != "ok" {
			out.Agents.Problems = append(out.Agents.Problems, "The account of "+n+", the provider the agents use, is not usable: "+l.Hint())
		}
		out.Logins = append(out.Logins, l)
	}
	return out
}
