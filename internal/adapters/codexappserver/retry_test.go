package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const overloaded = `"serverOverloaded"`

// hostClock counts the waits a session spent as host time.
type hostClock struct{ paused, resumed int }

// retrying opens a controlled session that waits only briefly before a repeat
// and records its transcript.
func retrying(t *testing.T, rt *fakeRuntime, spec run.SessionSpec, waits ...time.Duration) (*nativeSession, *Agents, *hostClock) {
	t.Helper()
	a := agentsFor(t, rt)
	a.TranscriptRoot = t.TempDir()
	if waits == nil {
		waits = []time.Duration{time.Millisecond, time.Millisecond}
	}
	a.retry = retryPolicy{waits: waits, grace: time.Minute}
	clock := &hostClock{}
	a.hostTime = func(context.Context) func() {
		clock.paused++
		return func() { clock.resumed++ }
	}
	s, err := a.Open(context.Background(), spec)
	if err != nil {
		if s != nil {
			_ = s.Close(context.Background())
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close(context.Background()) })
	return s.(*nativeSession), a, clock
}

// turnTexts returns the message text of every turn/start the provider received.
func turnTexts(rt *fakeRuntime) []string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	var texts []string
	for _, write := range rt.writes {
		var m struct {
			Method string
			Params struct{ Input []struct{ Text string } }
		}
		if json.Unmarshal([]byte(write), &m) == nil && m.Method == "turn/start" && len(m.Params.Input) == 1 {
			texts = append(texts, m.Params.Input[0].Text)
		}
	}
	return texts
}

// transcriptRetries returns the retry records of the session's transcript.
func transcriptRetries(t *testing.T, root string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(root, "*.jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatal("expected one transcript", files, err)
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var retries []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var record struct{ Direction, Data string }
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record.Direction == "retry" {
			retries = append(retries, record.Data)
		}
	}
	return retries
}

// A transient provider error before the response did anything repeats it: the
// same message in the same thread, after a wait that is host time. Each
// repeat is counted for the Apply and recorded in the transcript.
func TestTransientErrorBeforeAnyActionRepeatsTheResponse(t *testing.T) {
	for _, failures := range [][]string{{overloaded}, {overloaded, overloaded}} {
		t.Run(strings.Repeat("failure", len(failures)), func(t *testing.T) {
			rt := newRuntime()
			rt.failures = failures
			spec := testSessionSpec(t)
			got := judgments(&spec)
			s, a, clock := retrying(t, rt, spec)
			if err := s.Exchange(context.Background(), task("work")); err != nil || rt.closed {
				t.Fatal(err, rt.closed)
			}
			texts := turnTexts(rt)
			if len(texts) != len(failures)+1 {
				t.Fatal("expected one repeat per transient failure", len(texts))
			}
			for _, text := range texts {
				if text != texts[0] {
					t.Fatal("a repeat changed the message", text, texts[0])
				}
			}
			if !reflect.DeepEqual(*got, []run.Decision{run.Submitted}) {
				t.Fatal("only the completed attempt judges", *got)
			}
			if a.ProviderRetries() != len(failures) || clock.paused != len(failures) || clock.resumed != len(failures) {
				t.Fatal("repeats were not counted or did not wait as host time", a.ProviderRetries(), clock)
			}
			records := transcriptRetries(t, a.TranscriptRoot)
			if len(records) != len(failures) || !strings.Contains(records[0], `"attempt":2`) || !strings.Contains(records[0], `"codexErrorInfo":"serverOverloaded"`) {
				t.Fatal("transcript does not count the repeats", records)
			}
			if err := s.Exchange(context.Background(), task("next")); err != nil || len(turnTexts(rt)) != len(failures)+2 {
				t.Fatal("the session did not continue after a repeated response", err)
			}
		})
	}
}

// A response that served a Run tool call is never sent again, so the served
// call is not repeated. Continuing an acted response is not supported yet:
// the Exchange ends with the cause.
func TestTransientErrorAfterServedToolCallIsNotRepeated(t *testing.T) {
	rt := newRuntime()
	rt.failures, rt.failAfterTool = []string{overloaded}, true
	spec := testSessionSpec(t)
	checks := 0
	spec.Tools.Check = func(context.Context) (run.Check, error) {
		checks++
		return run.Check{Valid: true}, nil
	}
	s, a, clock := retrying(t, rt, spec)
	err := s.Exchange(context.Background(), task("work"))
	if !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "provider error: Selected model is at capacity.") || !strings.Contains(err.Error(), "transient (serverOverloaded), but the response already acted (1 Run tool calls served)") {
		t.Fatal(err)
	}
	if checks != 1 || len(turnTexts(rt)) != 1 || a.ProviderRetries() != 0 || clock.paused != 0 || !rt.closed {
		t.Fatal("an acted response was repeated or the session stayed open", checks, len(turnTexts(rt)), a.ProviderRetries(), clock, rt.closed)
	}
}

// A completed native execution is an action too. A native item that started
// and did not complete may have run, also a Run tool call on its way to the
// host: the response ends honestly.
func TestTransientErrorAfterNativeActionEndsTheResponse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*fakeRuntime)
		cause string
	}{
		{"completed", func(rt *fakeRuntime) { rt.failed = true }, "the response already acted (native executions completed)"},
		{"running", func(rt *fakeRuntime) { rt.failRunning = "commandExecution" }, "native commandExecution " + string(codecThreadID) + "/item-1 started and did not complete, so whether it ran is unclear"},
		{"tool call on its way", func(rt *fakeRuntime) { rt.failRunning = "mcpToolCall" }, "native mcpToolCall " + string(codecThreadID) + "/item-1 started and did not complete, so whether it ran is unclear"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := newRuntime()
			rt.failures = []string{overloaded}
			tc.setup(rt)
			s, a, clock := retrying(t, rt, testSessionSpec(t))
			err := s.Exchange(context.Background(), task("work"))
			if !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "transient (serverOverloaded), but "+tc.cause) {
				t.Fatal(err)
			}
			if len(turnTexts(rt)) != 1 || a.ProviderRetries() != 0 || clock.paused != 0 || !rt.closed {
				t.Fatal("a response that may have acted was repeated", len(turnTexts(rt)), a.ProviderRetries(), clock, rt.closed)
			}
		})
	}
}

// A tool call that reaches the host after the provider error stays unserved:
// its result reaches no model, and the response's state is unclear.
func TestToolCallAfterTransientErrorIsNotServed(t *testing.T) {
	rt := newRuntime()
	rt.hang, rt.tools = true, make(chan frame.ToolCall)
	spec := testSessionSpec(t)
	checks := 0
	spec.Tools.Check = func(context.Context) (run.Check, error) {
		checks++
		return run.Check{}, nil
	}
	a := agentsFor(t, rt)
	events := make(chan event)
	s := &nativeSession{agents: a, spec: spec, profile: a.profile, peer: rt, enc: testCodec(t, maxProtocolMessage), events: events, thread: codecThreadID}
	defer rt.Close()
	// Unbuffered channels order the error before the call deterministically.
	response := codecMessage(t, `{"id":1,"result":{"turn":{"id":"`+string(codecTurnID)+`","status":"inProgress","items":[],"error":null}}}`)
	failure := codecMessage(t, `{"method":"error","params":{"threadId":"`+string(codecThreadID)+`","error":{"message":"Selected model is at capacity.","codexErrorInfo":"serverOverloaded"},"willRetry":false}}`)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for _, m := range []Message{response, failure} {
			select {
			case events <- event{msg: m}:
			case <-done:
				return
			}
		}
		select {
		case rt.tools <- frame.ToolCall{ID: 4, Name: "check", Arguments: json.RawMessage(`{}`), Meta: json.RawMessage(`{"threadId":"` + string(codecThreadID) + `"}`)}:
		case <-done:
		}
	}()
	repeat, err := s.attempt(context.Background())
	if repeat != nil || !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "the tool call check reached the host after it and was not served") {
		t.Fatal(repeat, err)
	}
	if checks != 0 || len(rt.replies) != 0 {
		t.Fatal("a call of the failed response was served", checks, rt.replies)
	}
}

// A tool call of the failed turn that reaches the host during the wait is not
// served in the repeat as if the repeat had made it.
func TestToolCallDuringTheWaitEndsTheResponse(t *testing.T) {
	rt := newRuntime()
	rt.failures = []string{overloaded}
	spec := testSessionSpec(t)
	checks := 0
	spec.Tools.Check = func(context.Context) (run.Check, error) {
		checks++
		return run.Check{}, nil
	}
	s, a, _ := retrying(t, rt, spec)
	a.hostTime = func(context.Context) func() {
		rt.tools <- frame.ToolCall{ID: 5, Name: "check", Arguments: json.RawMessage(`{}`), Meta: json.RawMessage(`{"threadId":"` + string(codecThreadID) + `"}`)}
		return func() {}
	}
	err := s.Exchange(context.Background(), task("work"))
	if !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "the tool call check reached the host after it and was not served") {
		t.Fatal(err)
	}
	if checks != 0 || len(turnTexts(rt)) != 1 || a.ProviderRetries() != 0 {
		t.Fatal("a late call was served or the response repeated", checks, len(turnTexts(rt)), a.ProviderRetries())
	}
}

// A failed turn that does not end is not awaited without bound.
func TestFailedTurnThatDoesNotEndEndsTheResponse(t *testing.T) {
	rt := newRuntime()
	rt.failures, rt.unended = []string{overloaded}, true
	s, a, clock := retrying(t, rt, testSessionSpec(t))
	a.retry.grace = 10 * time.Millisecond
	err := s.Exchange(context.Background(), task("work"))
	if !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "transient (serverOverloaded), but the failed turn did not end within 10ms") {
		t.Fatal(err)
	}
	if len(turnTexts(rt)) != 1 || a.ProviderRetries() != 0 || clock.paused != 0 || !rt.closed {
		t.Fatal("an unended turn was repeated", len(turnTexts(rt)), a.ProviderRetries(), clock, rt.closed)
	}
}

// A response gets at most two extra attempts.
func TestTransientErrorEndsAfterTwoExtraAttempts(t *testing.T) {
	rt := newRuntime()
	rt.failures = []string{overloaded, overloaded, overloaded}
	s, a, clock := retrying(t, rt, testSessionSpec(t))
	err := s.Exchange(context.Background(), task("work"))
	if !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "transient (serverOverloaded), but it still failed after 3 attempts") {
		t.Fatal(err)
	}
	if len(turnTexts(rt)) != 3 || a.ProviderRetries() != 2 || clock.paused != 2 || !rt.closed || len(transcriptRetries(t, a.TranscriptRoot)) != 2 {
		t.Fatal("attempts were not bounded at three", len(turnTexts(rt)), a.ProviderRetries(), clock, rt.closed)
	}
}

// Every error kind outside the transient list ends the response at once, as
// before; so does an error without a kind.
func TestFinalProviderErrorsEndTheResponseWithoutRepeat(t *testing.T) {
	for _, info := range []string{`"unauthorized"`, `"usageLimitExceeded"`, `{"httpConnectionFailed":{"httpStatusCode":503}}`, `null`} {
		t.Run(info, func(t *testing.T) {
			rt := newRuntime()
			rt.failures = []string{info}
			s, a, clock := retrying(t, rt, testSessionSpec(t))
			err := s.Exchange(context.Background(), task("work"))
			if !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "provider error: Selected model is at capacity.") || strings.Contains(err.Error(), "transient") {
				t.Fatal(err)
			}
			if len(turnTexts(rt)) != 1 || a.ProviderRetries() != 0 || clock.paused != 0 || !rt.closed {
				t.Fatal("a final provider error was repeated", len(turnTexts(rt)), a.ProviderRetries(), clock)
			}
		})
	}
}

// A repeat whose wait would end after the Run deadline does not start; the
// response ends at once with the cause.
func TestTransientRepeatStaysWithinTheRunDeadline(t *testing.T) {
	rt := newRuntime()
	rt.failures = []string{overloaded}
	s, a, clock := retrying(t, rt, testSessionSpec(t), time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := s.Exchange(ctx, task("work"))
	if !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "a repeat after 1h0m0s would pass the Run deadline") {
		t.Fatal(err)
	}
	if len(turnTexts(rt)) != 1 || a.ProviderRetries() != 0 || clock.paused != 0 {
		t.Fatal("a repeat started beyond the Run deadline", len(turnTexts(rt)), a.ProviderRetries(), clock)
	}
}

// Close interrupts the wait before a repeat; nothing is sent again.
func TestCloseInterruptsTheWaitBeforeARepeat(t *testing.T) {
	rt := newRuntime()
	rt.failures = []string{overloaded}
	s, a, _ := retrying(t, rt, testSessionSpec(t), time.Hour)
	waiting := make(chan struct{})
	a.hostTime = func(context.Context) func() {
		close(waiting)
		return func() {}
	}
	closed := make(chan error, 1)
	go func() {
		<-waiting
		closed <- s.Close(context.Background())
	}()
	err := s.Exchange(context.Background(), task("work"))
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "the wait before attempt 2 was interrupted") {
		t.Fatal(err)
	}
	if err := <-closed; err != nil || !rt.closed || len(turnTexts(rt)) != 1 || a.ProviderRetries() != 0 {
		t.Fatal("Close did not end the wait", err, rt.closed, len(turnTexts(rt)), a.ProviderRetries())
	}
}

func TestErrorKindReadsBothWireShapes(t *testing.T) {
	for raw, want := range map[string]string{`"serverOverloaded"`: "serverOverloaded", `{"responseStreamDisconnected":{"httpStatusCode":null}}`: "responseStreamDisconnected", `null`: "", ``: "", `{}`: "", `{"a":1,"b":2}`: "", `[]`: "", `7`: ""} {
		if got := errorKind(json.RawMessage(raw)); got != want {
			t.Fatal(raw, got, want)
		}
	}
}
