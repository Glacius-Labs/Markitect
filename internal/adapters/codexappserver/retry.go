package codexappserver

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

// transientErrors is the closed list of codexErrorInfo kinds after which a
// response that has not acted yet is sent again. An entry needs observed
// evidence: serverOverloaded ("Selected model is at capacity", willRetry
// false) ended three Phase C Applies. Every other kind stays final.
var transientErrors = map[string]bool{"serverOverloaded": true}

// retryPolicy bounds the repeats of one response after a transient error.
type retryPolicy struct {
	waits []time.Duration // the pauses before each extra attempt
	grace time.Duration   // how long the failed turn may take to end
}

// A response has at most two extra attempts, after 30 s and 120 s.
var defaultRetry = retryPolicy{waits: []time.Duration{30 * time.Second, 120 * time.Second}, grace: 30 * time.Second}

// providerError is a Codex error notification that Codex will not retry
// itself. ended says why a transient one ends the response.
type providerError struct {
	kind, message, ended string
}

func (e *providerError) Error() string {
	text := fmt.Sprintf("%v: provider error: %s", ErrProtocol, e.message)
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

// errorKind names a codexErrorInfo value: a plain string such as
// "serverOverloaded" or the single key of an object variant such as
// {"httpConnectionFailed":{"httpStatusCode":502}}. Other shapes have no kind.
func errorKind(raw json.RawMessage) string {
	var kind string
	if json.Unmarshal(raw, &kind) == nil {
		return kind
	}
	var variant map[string]json.RawMessage
	if json.Unmarshal(raw, &variant) == nil && len(variant) == 1 {
		for kind = range variant {
			return kind
		}
	}
	return ""
}

// runTurn sends the response and repeats it after a transient provider error
// while it has not acted, within the extra attempts and the Run deadline.
func (t *nativeSession) runTurn(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.thread == "" {
		if err := t.startThread(ctx); err != nil {
			return err
		}
	}
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

// unserved ends the response for a tool call of the failed turn: its result
// reaches no model, so the call stays unserved.
func unserved(failure *providerError, call frame.ToolCall) error {
	return failure.end("the tool call %s reached the host after it and was not served, so the response's state is unclear", call.Name)
}

// recovery decides on a transient provider error once its failed turn ended.
// Only a response that did nothing is sent again: it served no Run tool call
// and no native item ran or runs. An acted response is never repeated;
// continuing it from the actual thread, workspace and assignment state is not
// supported yet. When it is unclear whether something ran, the response ends.
func (t *nativeSession) recovery(ctx context.Context, failure *providerError, retries int) (time.Duration, error) {
	waits := t.agents.retry.waits
	switch {
	case len(t.running) > 0:
		keys := slices.Sorted(maps.Keys(t.running))
		return 0, failure.end("native %s %s started and did not complete, so whether it ran is unclear", t.running[keys[0]], keys[0])
	case t.served > 0:
		return 0, failure.end("the response already acted (%d Run tool calls served) and is not repeated", t.served)
	case t.acted:
		return 0, failure.end("the response already acted (native executions completed) and is not repeated")
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
		Kind    string `json:"codexErrorInfo"`
		Waited  string `json:"waited"`
	}{attempt, failure.kind, wait.String()})
	t.tr.record("retry", data)
}

// ProviderRetries counts the responses this profile sent again after a
// transient provider error, across all of its sessions.
func (a *Agents) ProviderRetries() int { return int(a.retries.Load()) }

// attempt sends the response once. It returns the transient provider error
// that failed it once the failed turn has ended.
func (t *nativeSession) attempt(ctx context.Context) (*providerError, error) {
	t.final, t.turnID, t.running, t.runningBytes = "", "", nil, 0
	t.nextRequest++
	id := IntegerRequestID(t.nextRequest)
	raw, err := t.enc.EncodeTurnStart(id, TurnStartParams{Thread: t.thread, Text: t.prompt, Effort: t.profile.Session.Effort})
	if err != nil {
		return nil, err
	}
	if err = t.send(ctx, raw); err != nil {
		return nil, err
	}
	tools := t.peer.ToolCalls()
	responseSeen := false
	var failure *providerError
	var ended <-chan time.Time // the failed turn's grace, once it failed
	for {
		var m Message
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ended:
			return nil, failure.end("the failed turn did not end within %s", t.agents.retry.grace)
		case call, ok := <-tools:
			if !ok {
				tools = nil
				continue
			}
			if failure != nil {
				return nil, unserved(failure, call)
			}
			if err = t.answer(ctx, call); err != nil {
				return nil, err
			}
			t.served++
			continue
		case e, ok := <-t.events:
			if m, err = t.accept(e, ok); err != nil {
				return nil, err
			}
		}
		status, err := t.observe(m, id, &responseSeen)
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
		if status == "" {
			continue
		}
		select {
		case call, ok := <-tools:
			if ok && failure != nil {
				return nil, unserved(failure, call)
			}
			if ok {
				return nil, fmt.Errorf("%w: unanswered tool at native completion", ErrProtocolIncomplete)
			}
		default:
		}
		if failure != nil {
			if status != TurnFailed {
				return nil, failure.end("the provider turn ended %s", status)
			}
			return failure, nil
		}
		if status != TurnCompleted {
			return nil, fmt.Errorf("%w: provider turn %s", ErrSession, status)
		}
		return nil, nil
	}
}
