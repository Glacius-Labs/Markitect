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

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func assignmentWireMethods(t *testing.T, rt *fakeRuntime) []string {
	t.Helper()
	rt.mu.Lock()
	defer rt.mu.Unlock()
	var methods []string
	for _, raw := range rt.writes {
		var message struct{ Method string }
		if err := json.Unmarshal([]byte(raw), &message); err != nil {
			t.Fatal(err)
		}
		methods = append(methods, message.Method)
	}
	return methods
}

func openDormantAssignment(t *testing.T, rt *fakeRuntime, transcript string) run.AssignmentSession {
	t.Helper()
	session, err := agentsFor(t, rt).OpenAssignment(context.Background(), run.AssignmentSpec{
		ID: "lazy-assignment", Profile: "codex", Workspace: "git-work", Home: "agent-home", Transcript: transcript,
	}, okHandler(new([]string)))
	if err != nil {
		t.Fatal(err)
	}
	if got := assignmentWireMethods(t, rt); !reflect.DeepEqual(got, []string{"initialize", "initialized"}) || rt.starts != 1 {
		t.Fatalf("open started a conversation: methods=%v starts=%d", got, rt.starts)
	}
	return session
}

func TestAssignmentStartsThreadOnlyOnFirstMessage(t *testing.T) {
	rt := newRuntime()
	rt.finalText = "Manager reviewed work"
	rt.notifications = nativeChildSequence("completed")
	session := openDormantAssignment(t, rt, "")
	defer session.Close(context.Background())
	for _, text := range []string{"implement", "correct after Verify"} {
		result, err := session.Message(context.Background(), text, time.Second)
		if err != nil || !result.Completed || result.Stopped || result.Conversation != string(codecThreadID) {
			t.Fatalf("message %q: %+v %v", text, result, err)
		}
		if result.Final != rt.finalText || len(result.Commands) != 0 || result.Usage != (run.Usage{}) {
			t.Fatalf("child result attributed to Manager: %+v", result)
		}
	}
	want := []string{"initialize", "initialized", "thread/start", "turn/start", "turn/start"}
	if got := assignmentWireMethods(t, rt); !reflect.DeepEqual(got, want) || rt.starts != 1 || rt.stops != 0 || rt.closed {
		t.Fatalf("assignment did not retain provider/thread: methods=%v starts=%d stops=%d closed=%t", got, rt.starts, rt.stops, rt.closed)
	}
}

func TestAssignmentDormantCloseHasNoConversation(t *testing.T) {
	rt := newRuntime()
	session := openDormantAssignment(t, rt, "")
	for i := 0; i < 2; i++ {
		result, err := session.Close(context.Background())
		if err != nil || !result.Stopped || result.Completed || result.Conversation != "" || result.Usage != (run.Usage{}) || result.Final != "" || len(result.Commands) != 0 {
			t.Fatalf("dormant close: %+v %v", result, err)
		}
	}
	if got := assignmentWireMethods(t, rt); !reflect.DeepEqual(got, []string{"initialize", "initialized"}) || rt.stops != 1 || !rt.closed {
		t.Fatalf("dormant close: methods=%v stops=%d closed=%t", got, rt.stops, rt.closed)
	}
}

func TestAssignmentCancelledFirstMessageDoesNotStartThread(t *testing.T) {
	rt := newRuntime()
	session := openDormantAssignment(t, rt, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := session.Message(ctx, "never assigned", time.Second)
	if !errors.Is(err, context.Canceled) || !result.Stopped || result.Completed || result.Conversation != "" {
		t.Fatalf("cancelled first message: %+v %v", result, err)
	}
	retained, closeErr := session.Close(context.Background())
	// Message stamps its return time after retaining the terminal observation.
	result.Ended = retained.Ended
	if !errors.Is(closeErr, context.Canceled) || !reflect.DeepEqual(result, retained) || rt.stops != 1 || !rt.closed {
		t.Fatalf("lost cancelled closure: %+v %v", retained, closeErr)
	}
	if got := assignmentWireMethods(t, rt); !reflect.DeepEqual(got, []string{"initialize", "initialized"}) {
		t.Fatalf("cancelled message started conversation: %v", got)
	}
}

func TestAssignmentThreadStartFailureRetainsPartialTranscript(t *testing.T) {
	rt := newRuntime()
	rt.threadError = true
	path := filepath.Join(t.TempDir(), "failed-thread.jsonl")
	session := openDormantAssignment(t, rt, path)
	result, err := session.Message(context.Background(), "first assignment", time.Second)
	if err == nil || !result.Stopped || result.Completed || result.Conversation != "" || result.Final != "" || result.Usage != (run.Usage{}) {
		t.Fatalf("failed start: %+v %v", result, err)
	}
	retained, closeErr := session.Close(context.Background())
	result.Ended = retained.Ended
	if closeErr == nil || !reflect.DeepEqual(result, retained) || rt.stops != 1 || !rt.closed {
		t.Fatalf("lost failed start: %+v %v", retained, closeErr)
	}
	if got := assignmentWireMethods(t, rt); !reflect.DeepEqual(got, []string{"initialize", "initialized", "thread/start"}) {
		t.Fatalf("turn sent after failed start: %v", got)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil || !strings.Contains(string(data), "thread/start") || !strings.Contains(string(data), "thread unavailable") {
		t.Fatalf("missing partial thread evidence: %s %v", data, readErr)
	}
}
