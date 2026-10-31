package bootstrap

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/desktop"
)

// desktopSettings reads what Markitect Desktop's Settings shows: the
// agents' settings from the environment, as agentConfig reads them but
// without asking the container engine, and each provider's account login.
func desktopSettings(context.Context) desktop.AppSettings {
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
	if a.Image == "" {
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
	now := time.Now()
	for _, n := range names {
		p := providers[n]
		l := desktop.LoginState{Provider: n, File: p.loginFile(), Command: p.command, Used: n == name}
		if l.Used {
			l.File = env("MARKITECT_LOGIN", l.File)
		}
		data, err := os.ReadFile(l.File)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			l.State = "missing"
		case err != nil:
			l.State, l.Detail = "refused", err.Error()
		default:
			// The provider admits the login as an agent would; nothing of
			// the login leaves this function but its expiry.
			login, err := p.provider.Login(data, now)
			if err != nil {
				l.State, l.Detail = "refused", err.Error()
				break
			}
			l.State = "ok"
			if !login.Expires.IsZero() {
				expires := login.Expires
				l.Expires = &expires
			}
		}
		if l.Used && l.State != "ok" {
			out.Agents.Problems = append(out.Agents.Problems, "The login of "+n+", the provider the agents use, is not usable: log in again with "+p.command+".")
		}
		out.Logins = append(out.Logins, l)
	}
	return out
}
