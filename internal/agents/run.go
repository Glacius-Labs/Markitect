package agents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
)

// RunLabel marks every container and volume of a Run with the Run's ID; the
// Run remote's sweep removes everything that carries it.
const RunLabel = "markitect.run"

// Config is what every agent of a Run shares: the engine, the provider, the
// image, the limits, the networks, the login, the model and the transcripts.
type Config struct {
	Runtime  Runtime
	Provider Provider
	// Image is the Manager image's engine image ID.
	Image  string
	Limits Limits
	// Network reaches the provider; Hosts and DNS go to the engine as they
	// are (the evaluation's network, hosts and name servers).
	Network    string
	Hosts, DNS []string
	// LoginFile is the operator's account login for the provider, read at
	// every start, so a login refreshed on the host reaches later starts.
	LoginFile string
	Model     string
	Effort    string
	// Transcripts, when set, is a directory that receives one transcript
	// per start, RUN/MANAGER-ROLE-ROUND.jsonl.
	Transcripts string
	// Now is the clock; nil is time.Now.
	Now func() time.Time
}

// Agents implements the engine's agent port with one container and one
// provider conversation per start.
type Agents struct{ cfg Config }

var _ agentport.Agents = (*Agents)(nil)

// New checks the configuration.
func New(cfg Config) (*Agents, error) {
	if cfg.Runtime == nil || cfg.Provider == nil || cfg.Image == "" || cfg.LoginFile == "" || cfg.Model == "" || cfg.Effort == "" {
		return nil, errors.New("agents: the configuration needs a runtime, a provider, an image, a login file, a model and an effort")
	}
	return &Agents{cfg: cfg}, nil
}

// Run starts one agent for s, runs its turn and ends it. It returns only
// after the agent's container, login and volumes are gone, or with what may
// remain in End.Cleanup.
func (g *Agents) Run(ctx context.Context, s agentport.Start, host agentport.Host) agentport.End {
	cfg := g.cfg
	end := agentport.End{Agent: cfg.Provider.Name() + ":" + cfg.Model}
	login, err := os.ReadFile(cfg.LoginFile)
	if err != nil {
		return failed(end, LoginFailed, fmt.Errorf("read the %s login: %w", cfg.Provider.Name(), err))
	}
	var transcript io.Writer
	if cfg.Transcripts != "" {
		f, err := createTranscript(cfg.Transcripts, s)
		if err != nil {
			return failed(end, Other, err)
		}
		defer f.Close()
		transcript = f
	}
	spec := Spec{
		Name: containerName(s), Labels: map[string]string{RunLabel: s.Run}, Image: cfg.Image, Limits: cfg.Limits,
		Network: cfg.Network, Hosts: cfg.Hosts, DNS: cfg.DNS, Start: s, Host: host,
		Login: login, Model: cfg.Model, Effort: cfg.Effort, Transcript: transcript, Now: cfg.Now,
	}
	// The deadline bounds the start too: the clone, the base and the login
	// check.
	startCtx, cancel := context.WithDeadline(ctx, s.Deadline)
	defer cancel()
	a, err := Start(startCtx, cfg.Runtime, cfg.Provider, spec)
	if err != nil {
		class := Other
		var se *StartError
		if errors.As(err, &se) {
			class = se.Class
		}
		end = failed(end, class, err)
		switch {
		case ctx.Err() != nil:
			end.Reason = agentport.Cancelled
		case startCtx.Err() != nil:
			end.Reason = agentport.DeadlineReached
		}
		if errors.Is(err, errCleanup) {
			end.Cleanup = err.Error()
		}
		return end
	}
	out, sendErr := a.Send(ctx, "")
	closeErr := a.Close(context.WithoutCancel(ctx))
	if closeErr != nil {
		end.Cleanup = "the agent's container or volumes may remain: " + closeErr.Error()
	}
	if out.Model != "" {
		end.Agent = cfg.Provider.Name() + ":" + out.Model
	}
	end.Usage = agentport.Usage{InputTokens: out.Usage.Input + out.Usage.CacheCreation + out.Usage.CacheRead, OutputTokens: out.Usage.Output}
	switch {
	case sendErr != nil:
		return failed(end, Other, errors.New(string(a.redact([]byte(sendErr.Error())))))
	case out.End == Answered:
		end.Reason = agentport.Finished
	case out.End == Deadline:
		end.Reason, end.Detail = agentport.DeadlineReached, "the start's deadline passed and the agent was stopped"
	case out.End == Stopped:
		end.Reason, end.Detail = agentport.Cancelled, "the Run ended the agent"
	default:
		return failed(end, out.Class, errors.New(out.Detail))
	}
	return end
}

// failed ends a start as failed. The Detail starts with the failure class,
// so the evaluation can void the provider's failures.
func failed(end agentport.End, class Class, err error) agentport.End {
	if class == "" {
		class = Other
	}
	end.Reason, end.Detail = agentport.Failed, string(class)+": "+err.Error()
	return end
}

// errCleanup marks a start whose cleanup after a failure did not finish.
var errCleanup = errors.New("cleanup")

// containerName is unique per Run, Manager, role and round and fits the
// engine's name limits; the hash keeps deep Package paths short.
func containerName(s agentport.Start) string {
	sum := sha256.Sum256([]byte(s.Manager.ID))
	manager := strings.ReplaceAll(s.Manager.ID, "/", ".")
	if s.Manager.ID == "." {
		manager = "root"
	}
	if len(manager) > 20 {
		manager = manager[:20]
	}
	return fmt.Sprintf("markitect-run-%s-%s-%s-%d-%s", s.Run, manager, s.Role, s.Round, hex.EncodeToString(sum[:4]))
}

func createTranscript(dir string, s agentport.Start) (*os.File, error) {
	runDir := filepath.Join(dir, s.Run)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, err
	}
	name := strings.ReplaceAll(s.Manager.ID, "/", ".")
	if s.Manager.ID == "." {
		name = "root"
	}
	return os.OpenFile(filepath.Join(runDir, fmt.Sprintf("%s-%s-%d.jsonl", name, s.Role, s.Round)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
}
