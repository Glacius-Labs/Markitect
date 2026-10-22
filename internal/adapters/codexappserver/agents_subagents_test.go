package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

const nativeChildThread = "native-catalog-review"

func TestProviderCommandDisablesNativeDelegation(t *testing.T) {
	for _, mode := range []string{"api-key", "dedicated-runtime-login"} {
		t.Run(mode, func(t *testing.T) {
			argv := providerArgv(mode)
			for _, pair := range [][2]string{{"-c", "agents.enabled=false"}, {"--disable", "multi_agent"}, {"--disable", "multi_agent_v2"}} {
				found := false
				for i := 0; i+1 < len(argv); i++ {
					found = found || argv[i] == pair[0] && argv[i+1] == pair[1]
				}
				if !found {
					t.Fatalf("missing native delegation guard %v: %v", pair, argv)
				}
			}
		})
	}
}

func nativeNotification(method, thread string, fields map[string]any) string {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["threadId"] = thread
	data, _ := json.Marshal(map[string]any{"method": method, "params": fields})
	return string(data)
}

func nativeTurn(method, thread, turn, status string) string {
	return nativeNotification(method, thread, map[string]any{"turn": map[string]any{"id": turn, "rootTurnId": codecTurnID, "status": status, "items": []any{}}})
}

func nativeActivity(method, parent, turn, child string) string {
	return nativeNotification(method, parent, map[string]any{"turnId": turn, "item": map[string]any{"id": "catalog-review", "type": "subAgentActivity", "kind": "started", "agentThreadId": child, "agentPath": "/root/catalog_review"}})
}

// This sequence follows the retained RunID03 parent announcement and child
// start, then adds later child events the old adapter never reached.
func nativeChildSequence(status string) []string {
	return []string{
		nativeTurn("turn/started", string(codecThreadID), string(codecTurnID), "inProgress"),
		nativeActivity("item/started", string(codecThreadID), string(codecTurnID), nativeChildThread),
		nativeActivity("item/completed", string(codecThreadID), string(codecTurnID), nativeChildThread),
		`{"method":"thread/started","params":{"thread":` + strings.ReplaceAll(codecThread, string(codecThreadID), nativeChildThread) + `}}`,
		nativeTurn("turn/started", nativeChildThread, "child-turn", "inProgress"),
		nativeNotification("item/completed", nativeChildThread, map[string]any{"turnId": "child-turn", "item": map[string]any{"type": "agentMessage", "phase": "final_answer", "text": "CHILD MUST NOT BE THE MANAGER FINAL"}}),
		nativeNotification("item/completed", nativeChildThread, map[string]any{"turnId": "child-turn", "item": map[string]any{"type": "commandExecution", "id": "child-command", "command": "child test", "status": "completed", "exitCode": 0, "aggregatedOutput": "child output"}}),
		nativeNotification("thread/tokenUsage/updated", nativeChildThread, map[string]any{"turnId": "child-turn", "tokenUsage": map[string]any{"last": map[string]any{"inputTokens": 99999, "outputTokens": 99999}}}),
		nativeNotification("error", nativeChildThread, map[string]any{"turnId": "child-turn", "error": map[string]any{"message": "child-local failure"}, "willRetry": false}),
		nativeTurn("turn/completed", nativeChildThread, "child-turn", status),
	}
}

func TestRunKeepsAnnouncedNativeChildLifecycleSeparateFromManager(t *testing.T) {
	for _, status := range []string{"completed", "failed"} {
		t.Run(status, func(t *testing.T) {
			rt := newRuntime()
			rt.notifications = nativeChildSequence(status)
			rt.finalText = "Manager reviewed the resulting work"
			rt.usage = true
			rt.commandItems = []map[string]any{{"type": "commandExecution", "id": "manager-check", "command": "go test ./...", "status": "completed", "exitCode": 0, "aggregatedOutput": "manager output"}}
			res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), okHandler(new([]string)))
			if err != nil || !res.Completed || !res.Stopped || res.Conversation != string(codecThreadID) || res.Final != rt.finalText || len(res.Commands) != 1 || res.Commands[0].ID != "manager-check" {
				t.Fatalf("child lifecycle changed manager result: %+v %v", res, err)
			}
			if res.Usage.ProviderTurns != 1 || res.Usage.InputTokens != 60 || res.Usage.OutputTokens != 50 {
				t.Fatalf("child usage contaminated Manager accounting: %+v", res.Usage)
			}
		})
	}
}

func TestNativeChildCompletionCannotCompleteManager(t *testing.T) {
	rt := newRuntime()
	rt.notifications = nativeChildSequence("completed")
	rt.hang = true
	turn := testTurn()
	turn.Timeout = 80 * time.Millisecond
	res, err := agentsFor(t, rt).Run(context.Background(), turn, okHandler(new([]string)))
	if !errors.Is(err, context.DeadlineExceeded) || res.Completed || res.Final != "" || len(res.Commands) != 0 || res.Usage.ProviderTurns != 0 || !res.Stopped {
		t.Fatalf("child completion became parent success: %+v %v", res, err)
	}
}

func TestRunRejectsUnrelatedNotificationsAndWrongManagerTurn(t *testing.T) {
	for name, event := range map[string]string{
		"unannounced child start":      nativeTurn("turn/started", "unrelated", "foreign-turn", "inProgress"),
		"unannounced child completion": nativeTurn("turn/completed", "unrelated", "foreign-turn", "completed"),
		"wrong manager turn":           nativeTurn("turn/started", string(codecThreadID), "wrong-main-turn", "inProgress"),
		"wrong manager item":           nativeNotification("item/completed", string(codecThreadID), map[string]any{"turnId": "wrong-main-turn", "item": map[string]any{"type": "agentMessage", "phase": "final_answer", "text": "unrelated"}}),
		"foreign relation":             nativeActivity("item/started", "unrelated", "foreign-turn", "unrelated-child"),
		"relation missing identity":    nativeActivity("item/completed", string(codecThreadID), string(codecTurnID), ""),
		"relation points to manager":   nativeActivity("item/completed", string(codecThreadID), string(codecTurnID), string(codecThreadID)),
	} {
		t.Run(name, func(t *testing.T) {
			rt := newRuntime()
			rt.notifications = []string{event}
			res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), okHandler(new([]string)))
			if !errors.Is(err, ErrProtocol) || res.Completed || !res.Stopped {
				t.Fatalf("lost main correlation: %+v %v", res, err)
			}
		})
	}
}

func TestNativeDescendantNeedsAnAlreadyAnnouncedParent(t *testing.T) {
	rt := newRuntime()
	rt.notifications = nativeChildSequence("completed")
	rt.notifications = append(rt.notifications, nativeActivity("item/completed", nativeChildThread, "child-turn", "grandchild"), nativeTurn("turn/started", "grandchild", "grandchild-turn", "inProgress"), nativeTurn("turn/completed", "grandchild", "grandchild-turn", "completed"))
	rt.finalText = "Manager result"
	res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), okHandler(new([]string)))
	if err != nil || !res.Completed || res.Final != "Manager result" {
		t.Fatalf("announced descendant: %+v %v", res, err)
	}
}
