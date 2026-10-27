package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// transientErrors is the closed list of Claude Code error categories after
// which a response that has not acted yet is sent again. Claude Code retries
// a failed API request itself first (system/api_retry); this list applies only
// once it gave up. It mirrors the Codex adapter's one observed transient kind,
// a model at capacity: Claude Code calls it overloaded (HTTP 529). An entry
// needs observed evidence, which the native qualification K1b collects; every
// other category, or none, stays final.
var transientErrors = map[string]bool{"overloaded": true}

// retryPolicy bounds the repeats of one response after a transient error.
type retryPolicy struct {
	waits []time.Duration // the pauses before each extra attempt
	grace time.Duration   // how long the failed process may take to end
}

// A response has at most two extra attempts, after 30 s and 120 s.
var defaultRetry = retryPolicy{waits: []time.Duration{30 * time.Second, 120 * time.Second}, grace: 30 * time.Second}

// providerError is a result message that reports an error. kind is the error
// category of the attempt; ended says why a transient one ends the response.
type providerError struct {
	kind, message, ended string
}

func (e *providerError) Error() string {
	text := fmt.Sprintf("%v: provider error", ErrProtocol)
	if e.kind != "" {
		text += " (" + e.kind + ")"
	}
	text += ": " + e.message
	if e.ended != "" {
		text += "; " + e.ended
	}
	return text
}
func (e *providerError) Unwrap() error   { return ErrProtocol }
func (e *providerError) transient() bool { return transientErrors[e.kind] }

// end records why the transient error ends the response instead of a repeat.
func (e *providerError) end(format string, args ...any) *providerError {
	e.ended = fmt.Sprintf("transient (%s), but ", e.kind) + fmt.Sprintf(format, args...)
	return e
}

// runTurn sends the response and repeats it after a transient provider error
// while it has not acted, within the extra attempts and the Run deadline. A
// repeat resumes the same conversation when it already exists.
func (t *nativeSession) runTurn(ctx context.Context) error {
	for retries := 0; ; retries++ {
		failure, err := t.attempt(ctx)
		if failure == nil || err != nil {
			return err
		}
		wait, err := t.recovery(ctx, failure, retries)
		if err != nil {
			return err
		}
		if err = t.pause(ctx, wait); err != nil {
			return errors.Join(err, failure.end("the wait before attempt %d was interrupted", retries+2))
		}
		select {
		case call, ok := <-t.peer.ToolCalls():
			if ok {
				return unserved(failure, call)
			}
		default:
		}
		t.retried(failure, retries+2, wait)
	}
}

// staleCall refuses a Run tool call that is queued while no running response
// can receive its result: it stays unserved, and the Exchange ends instead of
// serving it in a later response.
func (t *nativeSession) staleCall(failure *providerError, when string) error {
	select {
	case call, ok := <-t.peer.ToolCalls():
		if !ok {
			return nil
		}
		if failure != nil {
			return unserved(failure, call)
		}
		return fmt.Errorf("%w: the Run tool call %s arrived %s and was not served", ErrProtocolIncomplete, call.Name, when)
	default:
		return nil
	}
}

// unserved ends the response for a tool call of the failed attempt: its result
// reaches no model, so the call stays unserved.
func unserved(failure *providerError, call frame.ToolCall) error {
	return failure.end("the tool call %s reached the host after it and was not served, so the response's state is unclear", call.Name)
}

// recovery decides on a transient provider error once its process ended. Only
// a response that did nothing is sent again: it served no Run tool call and no
// tool use ran or runs. An acted response is never repeated; continuing it
// from the actual conversation, workspace and assignment state is not
// supported yet. When it is unclear whether something ran, the response ends.
func (t *nativeSession) recovery(ctx context.Context, failure *providerError, retries int) (time.Duration, error) {
	waits := t.agents.retry.waits
	switch {
	case len(t.running) > 0:
		ids := slices.Sorted(maps.Keys(t.running))
		return 0, failure.end("tool use %s %s started and did not complete, so whether it ran is unclear", t.running[ids[0]], ids[0])
	case t.served > 0:
		return 0, failure.end("the response already acted (%d Run tool calls served) and is not repeated", t.served)
	case t.acted:
		return 0, failure.end("the response already acted (tool uses completed) and is not repeated")
	case retries >= len(waits):
		return 0, failure.end("it still failed after %d attempts", retries+1)
	}
	wait := waits[retries]
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Add(wait).Before(deadline) {
		return 0, failure.end("a repeat after %s would pass the Run deadline", wait)
	}
	return wait, nil
}

// pause waits before a repeated response. Waiting is host time: it pauses
// Run's clock of the response, and the Run deadline still bounds it.
func (t *nativeSession) pause(ctx context.Context, wait time.Duration) error {
	resume := t.agents.hostTime(ctx)
	defer resume()
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retried counts one repeated response for the Apply and records it in the
// transcript.
func (t *nativeSession) retried(failure *providerError, attempt int, wait time.Duration) {
	t.agents.retries.Add(1)
	data, _ := json.Marshal(struct {
		Attempt int    `json:"attempt"`
		Kind    string `json:"error"`
		Waited  string `json:"waited"`
	}{attempt, failure.kind, wait.String()})
	t.tr.record("retry", data)
}

// ProviderRetries counts the responses this profile sent again after a
// transient provider error, across all of its sessions.
func (a *Agents) ProviderRetries() int { return int(a.retries.Load()) }

// attempt runs the response once in a fresh provider process: one user
// message on standard input, which then closes, and the stream until the
// process exits. It returns the transient provider error that failed it once
// the process has ended, and only after the census found no native process
// left behind.
func (t *nativeSession) attempt(ctx context.Context) (*providerError, error) {
	t.final, t.running, t.runningBytes, t.lastError, t.result = "", nil, 0, "", nil
	codec, err := NewCodec(int(t.profile.MaxWireBytes))
	if err != nil {
		return nil, err
	}
	line, err := codec.EncodeUser(t.prompt)
	if err != nil {
		return nil, err
	}
	// A call queued before this process exists belongs to an earlier one.
	if err = t.staleCall(nil, "from an earlier response"); err != nil {
		return nil, err
	}
	env := append(providerEnvironment(), toolTimeout(ctx, time.Now()))
	if err = t.startProcess(ctx, providerArgv(t.session, t.persisted, t.profile.Session), "/work", env, toolSpecs()); err != nil {
		return nil, err
	}
	t.tr.record("send", line)
	if err = t.io(ctx, func() error { return t.peer.WriteInput(line) }); err != nil {
		return nil, err
	}
	if err = t.io(ctx, t.peer.CloseInput); err != nil {
		return nil, err
	}
	events := readEvents(t.peer.Stdout(), codec, t.done)
	tools := t.peer.ToolCalls()
	var failure *providerError
	var ended <-chan time.Time // the failed process's grace, once it failed
	// A Run tool call is served only after this launch reported the checked
	// configuration. The output and the call travel apart, so a call that
	// overtakes the system/init line waits for it.
	var held *frame.ToolCall
	t.launched = false
	for output := true; output; {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ended:
			return nil, failure.end("the failed response did not end within %s", t.agents.retry.grace)
		case call, ok := <-tools:
			if !ok {
				tools = nil
				continue
			}
			if failure != nil {
				return nil, unserved(failure, call)
			}
			if !t.launched {
				held, tools = &call, nil
				continue
			}
			if err = t.answer(ctx, call); err != nil {
				return nil, err
			}
			t.served++
		case e, ok := <-events:
			if !ok {
				output = false
				continue
			}
			m, err := t.accept(e)
			if err != nil {
				return nil, err
			}
			err = t.observe(m)
			var provider *providerError
			if errors.As(err, &provider) && provider.transient() {
				if failure == nil {
					failure, ended = provider, time.After(t.agents.retry.grace)
				}
				continue
			}
			if err != nil {
				return nil, err
			}
			if held != nil && t.launched {
				call := *held
				held, tools = nil, t.peer.ToolCalls()
				if failure != nil {
					return nil, unserved(failure, call)
				}
				if err = t.answer(ctx, call); err != nil {
					return nil, err
				}
				t.served++
			}
		}
	}
	if held != nil {
		return nil, fmt.Errorf("%w: the Run tool call %s arrived before the launch reported its configuration and was not served", ErrProtocolIncomplete, held.Name)
	}
	exit, err := t.collect(ctx)
	if err != nil {
		return nil, err
	}
	if err = t.staleCall(failure, "at native completion"); err != nil {
		return nil, err
	}
	if err = t.quiescent(ctx); err != nil {
		return nil, err
	}
	// The census waited; a call that arrived meanwhile is never served later.
	if err = t.staleCall(failure, "after native completion"); err != nil {
		return nil, err
	}
	if failure != nil {
		return failure, nil
	}
	if t.result == nil {
		return nil, fmt.Errorf("%w: provider output ended before the response finished (exit code %d %s)", ErrProtocolIncomplete, exit.Code, exit.Signal)
	}
	if exit.Code != 0 || exit.Signal != "" {
		return nil, fmt.Errorf("%w: provider exited with code %d %s after a successful result", ErrSession, exit.Code, exit.Signal)
	}
	return nil, nil
}
