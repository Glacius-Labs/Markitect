package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

// Recover reopens only a trusted Host-journaled thread and inspects the exact
// dispatched turn. It never submits input or invokes turn/start. Missing, running
// or unidentifiable turns remain uncertain, including a lost turn/start reply.
func (a *Adapter) Recover(parent context.Context, cfg agentexec.Config, h RecoveryHandle) (result agentexec.RunResult, err error) {
	if parent == nil {
		return result, errors.New("context required")
	}
	fp, err := a.Fingerprint(cfg)
	if err != nil {
		return result, err
	}
	if h.Protocol != protocolIdentity || h.Fingerprint != fp || h.ThreadID == "" || h.SessionID == "" || !filepath.IsAbs(h.Workspace.CWD) || h.Workspace.BaseSHA != h.Invocation.Request.SourceRevision || !h.TurnDispatched {
		return result, errors.New("recovery handle does not match the owned invocation/configuration")
	}
	// Validate the saved invocation binding without allocating a new identity.
	b, _ := json.Marshal(h.Invocation)
	var cloned RecoveryHandle
	wire, _ := json.Marshal(h)
	if json.Unmarshal(wire, &cloned) != nil {
		return result, ErrProtocol
	}
	h = cloned
	if h.Invocation.APIVersion != agentexec.APIVersion || h.Invocation.RunID == "" || h.Invocation.Nonce == "" || len(b) == 0 {
		return result, ErrProtocol
	}
	ctx, cancel := context.WithTimeout(parent, a.config.Timeout)
	defer cancel()
	start := time.Now()
	s := &session{a: a, ctx: ctx, h: h, output: map[string]string{}, children: map[string]int{}, spawns: map[string]int{}, outputLimit: cfg.MaxStdoutBytes, toolCalls: map[string]bool{}}
	s.sessions = map[string]string{}
	s.receivers = map[string]int{}
	s.life = &agentexec.Lifecycle{Provider: "codex-app-server", SessionID: h.SessionID, TurnID: h.TurnID, State: "unknown", Accounting: "partial", Requested: agentexec.SessionSettings{Model: a.config.Model, ReasoningEffort: a.config.ReasoningEffort, CWD: h.Workspace.CWD, PermissionProfile: a.config.PermissionProfile, WindowsSandboxBackend: string(a.config.WindowsSandboxBackend)}}
	// This is the original trusted attempt, not a newly requested role start.
	// Keep it unknown until the exact saved turn supplies a terminal observation.
	s.life.StartRequests = []agentexec.RoleStartRequest{{RequestID: h.Invocation.RunID, SessionID: h.SessionID, Role: h.Invocation.Request.Role, Model: a.config.Model, ReasoningEffort: a.config.ReasoningEffort, State: "unknown"}}
	result.Receipt = agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: h.Invocation.RunID, InputDigest: h.Invocation.InputDigest, ConfigDigest: fp, ProviderVersion: a.config.ProviderVersion, Outcome: agentexec.OutcomeIncomplete, Lifecycle: s.life}
	defer func() {
		result.Receipt.WallTimeMilliseconds = time.Since(start).Milliseconds()
		if !s.terminal {
			err = errors.Join(err, ErrUncertain)
		}
	}()
	env := selectedEnvironment(cfg.EnvironmentAllowlist)
	if err = bindReceipt(&result.Receipt, cfg, env); err != nil {
		return result, err
	}
	if err = verifyVersion(ctx, a.config, env); err != nil {
		return result, err
	}
	p, err := startProcess(a.config, h.Workspace.CWD, env, cfg.MaxStderrBytes)
	if err != nil {
		return result, err
	}
	defer func() {
		if closeErr := p.Close(); closeErr != nil {
			err = errors.Join(err, closeErr, ErrUncertain)
			s.life.State = "unknown"
		}
		result.Receipt.StderrDigest = p.stderr.digest()
	}()
	s.c, err = NewClient(p, a.config.MaxEventBytes, s.observe)
	if err != nil {
		return result, err
	}
	s.c.experimental = len(a.options.DynamicTools) > 0 || a.config.PermissionProfile != ""
	defer func() {
		_ = s.c.Close()
		s.life.EventLogDigest = s.c.EventDigest()
		result.Receipt.StdoutDigest = digest([]byte(s.final))
	}()
	if err = s.c.initialize(ctx); err != nil {
		return result, err
	}
	var resumed threadResponse
	if err = s.c.call(ctx, "thread/resume", map[string]any{"threadId": h.ThreadID}, &resumed); err != nil {
		return result, err
	}
	if resumed.Thread.ID != h.ThreadID || resumed.Thread.SessionID != h.SessionID {
		return result, ErrProtocol
	}
	if err = s.bind(resumed); err != nil {
		return result, err
	}
	var snapshot struct {
		Thread thread `json:"thread"`
	}
	if err = s.c.call(ctx, "thread/read", map[string]any{"threadId": h.ThreadID, "includeTurns": true}, &snapshot); err != nil {
		return result, err
	}
	if snapshot.Thread.ID != h.ThreadID {
		return result, ErrProtocol
	}
	if h.TurnID == "" {
		return result, ErrUncertain
	} // cannot guess which turn belongs to a lost dispatch
	for _, t := range snapshot.Thread.Turns {
		if t.ID == h.TurnID {
			for _, v := range t.Items {
				if err = s.takeItem(h.ThreadID, v, true); err != nil {
					return result, err
				}
			}
			if t.Status == "inProgress" {
				return result, ErrUncertain
			}
			if err = s.finish(t); err != nil {
				return result, err
			}
			if s.life.State != "completed" {
				return result, errors.New("recovered turn did not complete successfully")
			}
			result.Response, err = agentexec.DecodeResponse([]byte(s.final), h.Invocation, "")
			if err != nil {
				return result, err
			}
			result.Receipt.Outcome = result.Response.Outcome
			if checkErr := s.verifyInstructions(); checkErr != nil {
				return result, checkErr
			}
			after, checkErr := a.Fingerprint(cfg)
			if checkErr != nil || after != fp {
				return result, agentexec.ErrInputChanged
			}
			return result, nil
		}
	}
	return result, ErrUncertain
}
