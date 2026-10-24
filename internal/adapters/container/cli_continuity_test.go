package container_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// This capability probe answers a runtime question of the step-basis design
// with a real Codex session. Besides the engine and the runtime image it needs
// the provider's credential (see cli_smoke_test.go).

// TestCodexConversationContinuesInANewEnvironment ends a Codex thread's
// container, advances the workspace basis while no session runs, and resumes
// the persistent thread from its retained provider home in a new container.
func TestCodexConversationContinuesInANewEnvironment(t *testing.T) {
	r := newRuntimeSmoke(t)
	word := fmt.Sprintf("harbor-%d", time.Now().UnixNano()%100000)
	mounts := r.volumes("codex-resume")
	env, credentialed := r.codexCredential("codex-resume", mounts)
	if !credentialed {
		t.Skip("set OPENAI_API_KEY or MARKITECT_TEST_CODEX_AUTH_VOLUME")
	}

	first := r.start("codex-resume-a", mounts)
	a := startCodex(t, r, first, env, make(chan frame.ToolCall, 4))
	thread := a.call("thread/start", map[string]any{"cwd": "/work", "approvalPolicy": "never", "sandbox": "danger-full-access", "ephemeral": false, "model": codexModel})
	if thread["error"] != nil {
		t.Fatalf("thread/start: %v; diagnostics %q", thread["error"], first.Diagnostics())
	}
	threadID := thread["result"].(map[string]any)["thread"].(map[string]any)["id"]
	codexTurn(a, threadID, "Remember this code word for later: "+word+". Do not write any file. Reply with OK.")
	if err := first.Close(); err != nil {
		t.Fatalf("close first environment: %v", err)
	}

	r.writeFile("codex-resume-basis", mounts, "/work/basis.txt", "basis-2\n")

	second := r.start("codex-resume-b", mounts)
	b := startCodex(t, r, second, env, make(chan frame.ToolCall, 4))
	resumed := b.call("thread/resume", map[string]any{"threadId": threadID, "cwd": "/work", "approvalPolicy": "never", "sandbox": "danger-full-access", "model": codexModel})
	if resumed["error"] != nil {
		t.Fatalf("thread/resume: %v; diagnostics %q", resumed["error"], second.Diagnostics())
	}
	if model := resumed["result"].(map[string]any)["model"]; model != codexModel {
		t.Fatalf("resumed thread runs %v, want %s", model, codexModel)
	}
	answer := codexTurn(b, threadID, "Read the file basis.txt in the working directory now, with a shell command if needed. "+
		"Then answer in one line: what was the code word I gave you earlier, and what does basis.txt contain?")
	if !strings.Contains(answer, word) || !strings.Contains(answer, "basis-2") {
		t.Fatalf("resumed answer %q lacks the code word %s or the new basis", answer, word)
	}
}
