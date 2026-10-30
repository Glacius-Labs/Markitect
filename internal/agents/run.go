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
	"sort"
	"strings"
	"sync"
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
	// per turn, RUN/MANAGER-ROLE-ROUND.jsonl.
	Transcripts string
	// ContextLimit is the conversation size, in input tokens of a turn's
	// last model request, from which an agent ends after its turn and its
	// next turn starts it fresh from the Run's remote; zero is
	// DefaultContextLimit.
	ContextLimit int64
	// Now is the clock; nil is time.Now.
	Now func() time.Time
}

// DefaultContextLimit leaves room below a 200,000-token context window, the
// smallest of the default models', for the next turn's task and work.
const DefaultContextLimit = 150_000

// Agents implements the engine's agent port with one container and one
// provider conversation per agent, which live for the Run while its turns
// finish.
type Agents struct {
	cfg Config

	mu   sync.Mutex
	live map[agentKey]*Agent
}

// agentKey names one agent: a Run's Manager and role.
type agentKey struct {
	run, manager string
	role         agentport.Role
}

var _ agentport.Agents = (*Agents)(nil)

// New checks the configuration.
func New(cfg Config) (*Agents, error) {
	if cfg.Runtime == nil || cfg.Provider == nil || cfg.Image == "" || cfg.LoginFile == "" || cfg.Model == "" || cfg.Effort == "" || cfg.ContextLimit < 0 {
		return nil, errors.New("agents: the configuration needs a runtime, a provider, an image, a login file, a model and an effort")
	}
	if cfg.ContextLimit == 0 {
		cfg.ContextLimit = DefaultContextLimit
	}
	return &Agents{cfg: cfg, live: map[agentKey]*Agent{}}, nil
}

// Run runs one turn of the agent s names. The agent's first turn starts it
// in its own container with a fresh clone; a later one continues its
// conversation there. After a turn that did not finish, or whose
// conversation reached the context limit, the agent ends, and its next turn
// starts it again from the Run's remote. Run returns once no process of the
// turn runs, with what an ended agent may have left in End.Cleanup.
func (g *Agents) Run(ctx context.Context, s agentport.Start, host agentport.Host) agentport.End {
	cfg := g.cfg
	end := agentport.End{Agent: cfg.Provider.Name() + ":" + cfg.Model}
	key := agentKey{s.Run, s.Manager.ID, s.Role}
	g.mu.Lock()
	a := g.live[key]
	delete(g.live, key)
	g.mu.Unlock()
	login, err := os.ReadFile(cfg.LoginFile)
	if err != nil {
		end = failed(end, LoginFailed, fmt.Errorf("read the %s login: %w", cfg.Provider.Name(), err))
		return retire(ctx, a, end)
	}
	var transcript io.Writer
	if cfg.Transcripts != "" {
		f, err := createTranscript(cfg.Transcripts, s)
		if err != nil {
			return retire(ctx, a, failed(end, Other, err))
		}
		defer f.Close()
		transcript = f
	}
	var out Outcome
	var turnErr error
	if a != nil {
		end.Continued = true
		out, turnErr = a.Continue(ctx, s, host, login, transcript)
	} else {
		spec := Spec{
			Name: containerName(s), Labels: map[string]string{RunLabel: s.Run}, Image: cfg.Image, Limits: cfg.Limits,
			Network: cfg.Network, Hosts: cfg.Hosts, DNS: cfg.DNS, Start: s, Host: host,
			Login: login, Model: cfg.Model, Effort: cfg.Effort, Transcript: transcript, Now: cfg.Now,
		}
		if a, err = start(ctx, cfg, spec); err != nil {
			return startFailed(ctx, s, end, err)
		}
		out, turnErr = a.Send(ctx, "")
	}
	if out.Model != "" {
		end.Agent = cfg.Provider.Name() + ":" + out.Model
	}
	// A turn that broke off reports no total; what it showed is a lower bound.
	end.Usage = agentport.Usage{InputTokens: out.Usage.Input + out.Usage.CacheCreation + out.Usage.CacheRead, OutputTokens: out.Usage.Output, Partial: out.Usage.Partial || turnErr != nil}
	switch {
	case turnErr != nil:
		class := Other
		var se *StartError
		if errors.As(turnErr, &se) {
			class = se.Class
		}
		end = failed(end, class, errors.New(string(a.redact([]byte(turnErr.Error())))))
		switch {
		case ctx.Err() != nil:
			end.Reason, end.Class = agentport.Cancelled, ""
		case !time.Now().Before(s.Deadline):
			// The deadline, not the step it cut off, ended the turn.
			end.Reason, end.Class, end.Detail = agentport.DeadlineReached, "", "the turn's deadline passed: "+end.Detail
		}
	case out.End == Answered:
		end.Reason = agentport.Finished
	case out.End == Deadline:
		end.Reason, end.Detail = agentport.DeadlineReached, "the turn's deadline passed and the agent was stopped"
	case out.End == Stopped:
		end.Reason, end.Detail = agentport.Cancelled, "the Run ended the agent"
	default:
		end = failed(end, out.Class, errors.New(out.Detail))
	}
	if end.Reason == agentport.Finished && out.Context < cfg.ContextLimit {
		g.mu.Lock()
		g.live[key] = a
		g.mu.Unlock()
		return end
	}
	return retire(ctx, a, end)
}

// start starts an agent within the turn's deadline, which bounds the clone,
// the base and the login check too.
func start(ctx context.Context, cfg Config, spec Spec) (*Agent, error) {
	startCtx, cancel := context.WithDeadline(ctx, spec.Start.Deadline)
	defer cancel()
	a, err := Start(startCtx, cfg.Runtime, cfg.Provider, spec)
	if err != nil && startCtx.Err() != nil && ctx.Err() == nil {
		return nil, &deadlineError{err}
	}
	return a, err
}

// deadlineError is a start the turn's deadline cut off.
type deadlineError struct{ error }

func (e *deadlineError) Unwrap() error { return e.error }

func startFailed(ctx context.Context, s agentport.Start, end agentport.End, err error) agentport.End {
	class := Other
	var se *StartError
	if errors.As(err, &se) {
		class = se.Class
	}
	end = failed(end, class, err)
	var de *deadlineError
	switch {
	case ctx.Err() != nil:
		end.Reason, end.Class = agentport.Cancelled, ""
	case errors.As(err, &de):
		// The deadline, not the step it cut off, ended the start.
		end.Reason, end.Class, end.Detail = agentport.DeadlineReached, "", "the turn's deadline passed before the agent's turn began: "+end.Detail
	}
	if errors.Is(err, errCleanup) {
		end.Cleanup = err.Error()
	}
	return end
}

// retire ends an agent after its turn, or before it when the turn could not
// begin, and says in the End what may remain.
func retire(ctx context.Context, a *Agent, end agentport.End) agentport.End {
	if a == nil {
		return end
	}
	if err := a.Close(context.WithoutCancel(ctx)); err != nil {
		end.Cleanup = "the agent's container or volumes may remain: " + err.Error()
	}
	return end
}

// closeAtOnce bounds how many agents Close ends at the same time: each takes
// a few engine calls, and a large Run has two agents per working Manager.
const closeAtOnce = 8

// Close ends every living agent of the Run, several at once.
func (g *Agents) Close(ctx context.Context, run string) []string {
	g.mu.Lock()
	ended := map[string]*Agent{}
	for k, a := range g.live {
		if k.run == run {
			ended[fmt.Sprintf("%s %s", k.manager, k.role)] = a
			delete(g.live, k)
		}
	}
	g.mu.Unlock()
	var (
		mu   sync.Mutex
		left []string
		wg   sync.WaitGroup
		slot = make(chan struct{}, closeAtOnce)
	)
	for name, a := range ended {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slot <- struct{}{}
			defer func() { <-slot }()
			if err := a.Close(ctx); err != nil {
				mu.Lock()
				left = append(left, fmt.Sprintf("%s: the agent's container or volumes may remain: %v", name, err))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	sort.Strings(left)
	return left
}

// failed ends a start as failed with its failure class, so the evaluation
// can void the provider's failures. The agent runtime's classes carry the
// port's values.
func failed(end agentport.End, class Class, err error) agentport.End {
	if class == "" {
		class = Other
	}
	end.Reason, end.Class, end.Detail = agentport.Failed, agentport.FailureClass(class), err.Error()
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
