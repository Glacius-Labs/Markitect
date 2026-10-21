package bootstrap

import (
	"context"
	"errors"
	"testing"

	runport "github.com/Glacius-Labs/Markitect/internal/run"
)

type recordingAgents struct {
	turns  []runport.Turn
	result runport.TurnResult
	err    error
}

func (a *recordingAgents) Run(_ context.Context, turn runport.Turn, _ runport.ToolHandler) (runport.TurnResult, error) {
	a.turns = append(a.turns, turn)
	return a.result, a.err
}

func TestAgentsRouterDispatchesByProfileAndKeepsTheAdapterResult(t *testing.T) {
	claude := &recordingAgents{result: runport.TurnResult{Conversation: "claude-conversation", Stopped: true}}
	codex := &recordingAgents{result: runport.TurnResult{Conversation: "codex-conversation", Stopped: true}, err: errors.New("partial turn")}
	router := newAgentsRouter(map[string]runport.Agents{"claude-profile": claude, "codex-profile": codex})
	first, err := router.Run(context.Background(), runport.Turn{ID: "t1", Profile: "claude-profile"}, nil)
	if err != nil || first.Conversation != "claude-conversation" {
		t.Fatalf("%+v %v", first, err)
	}
	second, err := router.Run(context.Background(), runport.Turn{ID: "t2", Profile: "codex-profile"}, nil)
	if err == nil || err.Error() != "partial turn" || second.Conversation != "codex-conversation" {
		t.Fatalf("adapter result or error changed: %+v %v", second, err)
	}
	if len(claude.turns) != 1 || claude.turns[0].ID != "t1" || len(codex.turns) != 1 || codex.turns[0].ID != "t2" {
		t.Fatalf("turns reached the wrong adapter: %v %v", claude.turns, codex.turns)
	}
}

func TestAgentsRouterRefusesUnroutedProfilesWithoutFallback(t *testing.T) {
	adapter := &recordingAgents{}
	routes := map[string]runport.Agents{"known": adapter, "nil-route": nil}
	router := newAgentsRouter(routes)
	delete(routes, "known") // the router owns its routes
	for _, profile := range []string{"", "unknown", "nil-route", "Known"} {
		result, err := router.Run(context.Background(), runport.Turn{ID: "t", Profile: profile}, nil)
		if !errors.Is(err, runport.ErrInvalid) || !result.Stopped {
			t.Fatalf("profile %q: %+v %v", profile, result, err)
		}
	}
	if len(adapter.turns) != 0 {
		t.Fatal("refused turn reached an adapter")
	}
	if _, err := router.Run(context.Background(), runport.Turn{ID: "t", Profile: "known"}, nil); err != nil {
		t.Fatal(err)
	}
}
