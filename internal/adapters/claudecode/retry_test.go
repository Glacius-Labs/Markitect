package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// hostClock counts the host-time stretches the adapter marks.
type hostClock struct {
	mu     sync.Mutex
	marked int
	open   int
}

func (c *hostClock) mark(context.Context) func() {
	c.mu.Lock()
	c.marked++
	c.open++
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		c.open--
		c.mu.Unlock()
	}
}

// sequence plays the scripted responses in order, the last one repeating.
func sequence(plays ...func(*fakeProcess)) func(*fakeProcess) {
	var mu sync.Mutex
	next := 0
	return func(p *fakeProcess) {
		mu.Lock()
		play := plays[min(next, len(plays)-1)]
		next++
		mu.Unlock()
		play(p)
	}
}

// overloaded fails a response after Claude Code's own retries, with what the
// response did before.
func overloaded(before ...string) func(*fakeProcess) {
	out := append([]string{fixtureInit("SESSION", connectedServers), fixtureRetry("SESSION", "overloaded")}, before...)
	return lines(1, append(out, fixtureFailure("SESSION", "overloaded")...)...)
}

func retrying(t *testing.T, rt *fakeRuntime, spec run.SessionSpec) (*nativeSession, *Agents, *hostClock) {
	t.Helper()
	a := agentsFor(t, rt, credentialsLogin())
	clock := &hostClock{}
	a.hostTime = clock.mark
	a.TranscriptRoot = t.TempDir()
	return openSession(t, a, spec), a, clock
}

func TestTransientErrorBeforeAnyActionRepeatsInTheSameConversation(t *testing.T) {
	rt := newRuntime()
	rt.play = sequence(overloaded(), judged("submitted"))
	spec := testSessionSpec(t)
	decisions := judgments(&spec)
	s, a, clock := retrying(t, rt, spec)
	if err := s.Exchange(context.Background(), task("work")); err != nil {
		t.Fatal(err)
	}
	responses := rt.responses()
	if len(responses) != 2 || len(*decisions) != 1 || a.ProviderRetries() != 1 {
		t.Fatalf("responses=%d judgments=%v retries=%d", len(responses), *decisions, a.ProviderRetries())
	}
	if flagValue(responses[0].start.Argv, "--session-id") != s.session || flagValue(responses[1].start.Argv, "--resume") != s.session {
		t.Fatal("the repeat did not resume the same conversation")
	}
	if string(responses[0].input) != string(responses[1].input) {
		t.Fatal("the repeat changed the message")
	}
	if clock.marked != 1 || clock.open != 0 {
		t.Fatalf("the wait was not marked as host time: %+v", clock)
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(a.TranscriptRoot)
	data, err := os.ReadFile(filepath.Join(a.TranscriptRoot, entries[0].Name()))
	if err != nil || !strings.Contains(string(data), `"direction":"retry"`) || !strings.Contains(string(data), `\"error\":\"overloaded\"`) {
		t.Fatal("transcript lacks the retry record", err)
	}
}

// A launch that never reported system/init is refused, even when it ends in
// a successful result or a transient failure: its configuration is unchecked.
func TestLaunchWithoutInitIsRefused(t *testing.T) {
	for name, tc := range map[string]struct {
		play func(*fakeProcess)
		want string
	}{
		"successful result":  {lines(0, fixtureAssistant("SESSION", textBlock("Done.")), fixtureResult("SESSION", "Done.")), "before its launch reported system/init"},
		"bare result":        {lines(0, fixtureResult("SESSION", "Done.")), "before its launch reported system/init"},
		"transient failure":  {lines(1, fixtureFailure("SESSION", "overloaded")...), "before its launch reported system/init"},
		"tool use":           {lines(0, fixtureAssistant("SESSION", toolUse("toolu_1", "Bash")), fixtureToolResult("SESSION", "toolu_1"), fixtureResult("SESSION", "Done.")), "before its launch reported system/init"},
		"nothing but status": {lines(0, `{"type":"system","subtype":"status","session_id":"SESSION"}`), "without reporting its launch"},
	} {
		t.Run(name, func(t *testing.T) {
			rt := newRuntime()
			rt.play = sequence(tc.play, judged("submitted"))
			s, a, _ := retrying(t, rt, testSessionSpec(t))
			err := s.Exchange(context.Background(), task("work"))
			if err == nil || !strings.Contains(err.Error(), tc.want) || len(rt.responses()) != 1 || a.ProviderRetries() != 0 || s.Acted() {
				t.Fatalf("got %v after %d responses, want %q", err, len(rt.responses()), tc.want)
			}
		})
	}
}

func TestActedOrUnclearResponsesAreNotRepeated(t *testing.T) {
	for name, tc := range map[string]struct {
		play func(*fakeProcess)
		want string
	}{
		"served Run tool call": {func(p *fakeProcess) {
			p.emit(fixtureInit(p.session, connectedServers))
			if p.runTool("toolu_check", "check", "{}") {
				p.emit(fixtureFailure(p.session, "overloaded")...)
				p.end(1)
			}
		}, "Run tool calls served"},
		"completed tool use": {overloaded(fixtureAssistant("SESSION", toolUse("toolu_1", "Bash")), fixtureToolResult("SESSION", "toolu_1")), "tool uses completed"},
		"open tool use":      {overloaded(fixtureAssistant("SESSION", toolUse("toolu_2", "Edit"))), "toolu_2 started and did not complete"},
		"final category":     {lines(1, append([]string{fixtureInit("SESSION", connectedServers)}, fixtureFailure("SESSION", "server_error")...)...), "(server_error)"},
		"no category":        {lines(1, fixtureInit("SESSION", connectedServers), `{"type":"result","subtype":"error_during_execution","is_error":true,"session_id":"SESSION","errors":["boom"]}`), "boom"},
		// Claude Code's own retry recovered; the later failure has no category.
		"recovered retry": {lines(1, fixtureInit("SESSION", connectedServers), fixtureRetry("SESSION", "overloaded"), fixtureAssistant("SESSION", textBlock("Recovered.")), `{"type":"result","subtype":"error_during_execution","is_error":true,"session_id":"SESSION","errors":["late failure"]}`), "provider error: late failure"},
	} {
		t.Run(name, func(t *testing.T) {
			rt := newRuntime()
			rt.play = sequence(tc.play, judged("submitted"))
			s, a, _ := retrying(t, rt, testSessionSpec(t))
			err := s.Exchange(context.Background(), task("work"))
			if err == nil || !strings.Contains(err.Error(), tc.want) || len(rt.responses()) != 1 || a.ProviderRetries() != 0 {
				t.Fatalf("got %v after %d responses, want %q", err, len(rt.responses()), tc.want)
			}
		})
	}
}

func TestToolCallAfterTheTransientErrorStaysUnserved(t *testing.T) {
	rt := newRuntime()
	rt.play = sequence(func(p *fakeProcess) {
		p.emit(fixtureInit(p.session, connectedServers))
		p.emit(fixtureFailure(p.session, "overloaded")...)
		// The host reads the failure before the late call arrives.
		time.Sleep(200 * time.Millisecond)
		p.call("check", "{}")
	}, judged("submitted"))
	s, a, _ := retrying(t, rt, testSessionSpec(t))
	a.retry.grace = 10 * time.Second
	err := s.Exchange(context.Background(), task("work"))
	if err == nil || !strings.Contains(err.Error(), "reached the host after it and was not served") || len(rt.replies) != 0 {
		t.Fatalf("late tool call: %v %+v", err, rt.replies)
	}
}

func TestFailedProcessThatDoesNotExitEndsTheResponse(t *testing.T) {
	rt := newRuntime()
	rt.play = func(p *fakeProcess) {
		p.emit(fixtureInit(p.session, connectedServers))
		p.emit(fixtureFailure(p.session, "overloaded")...)
		<-p.killed
	}
	s, _, _ := retrying(t, rt, testSessionSpec(t))
	err := s.Exchange(context.Background(), task("work"))
	if err == nil || !strings.Contains(err.Error(), "did not end within") {
		t.Fatalf("hanging failed process: %v", err)
	}
}

func TestTransientErrorEndsAfterTwoExtraAttempts(t *testing.T) {
	rt := newRuntime()
	rt.play = overloaded()
	s, a, _ := retrying(t, rt, testSessionSpec(t))
	err := s.Exchange(context.Background(), task("work"))
	if err == nil || !strings.Contains(err.Error(), "still failed after 3 attempts") || len(rt.responses()) != 3 || a.ProviderRetries() != 2 {
		t.Fatalf("got %v after %d responses", err, len(rt.responses()))
	}
}

func TestRepeatStaysWithinTheRunDeadline(t *testing.T) {
	rt := newRuntime()
	rt.play = sequence(overloaded(), judged("submitted"))
	s, a, _ := retrying(t, rt, testSessionSpec(t))
	a.retry.waits = []time.Duration{time.Hour}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := s.Exchange(ctx, task("work"))
	if err == nil || !strings.Contains(err.Error(), "would pass the Run deadline") || len(rt.responses()) != 1 {
		t.Fatalf("deadline: %v", err)
	}
}

func TestCloseInterruptsTheWaitBeforeARepeat(t *testing.T) {
	rt := newRuntime()
	rt.play = sequence(overloaded(), judged("submitted"))
	s, a, _ := retrying(t, rt, testSessionSpec(t))
	a.retry.waits = []time.Duration{time.Hour}
	waiting := make(chan struct{})
	a.hostTime = func(context.Context) func() { close(waiting); return func() {} }
	done := make(chan error, 1)
	go func() { done <- s.Exchange(context.Background(), task("work")) }()
	select {
	case <-waiting:
	case <-time.After(10 * time.Second):
		t.Fatal("the wait did not begin")
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "interrupted") {
			t.Fatalf("interrupted wait: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close did not interrupt the wait")
	}
	if len(rt.responses()) != 1 {
		t.Fatal("a repeat started after Close")
	}
}
