package agentd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

func TestClaudeHomeInstallsInstructionsAndLoginThenClearsThem(t *testing.T) {
	h := newHarness(t)
	home := frame.ClaudeHome{Credentials: []byte(`{"claudeAiOauth":{"accessToken":"synthetic"}}`), SystemPrompt: "BASE ✓\n", AppendSystemPrompt: "DEVELOPER\n"}
	h.sendControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 7, Initialize: &frame.Initialize{Claude: &home}})
	if c := h.control(frame.TypeInitialized); c.Request != 7 {
		t.Fatal("setup reply lost request identity")
	}
	files := map[string]string{
		filepath.Join(h.home, ".claude", ".credentials.json"):          string(home.Credentials),
		filepath.Join(h.home, ".markitect", "system-prompt.md"):        home.SystemPrompt,
		filepath.Join(h.home, ".markitect", "append-system-prompt.md"): home.AppendSystemPrompt,
	}
	for path, want := range files {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != want {
			t.Fatalf("%s not installed exactly: %v", path, err)
		}
		if runtime.GOOS != "windows" {
			for p, mode := range map[string]os.FileMode{filepath.Dir(path): 0700, path: 0600} {
				if info, err := os.Stat(p); err != nil || info.Mode().Perm() != mode {
					t.Fatalf("%s permissions not %o", p, mode)
				}
			}
		}
	}
	if _, err := os.Lstat(filepath.Join(h.home, ".codex")); !os.IsNotExist(err) {
		t.Fatal("a Claude setup created the Codex home")
	}
	// Two provider processes run on one setup; cleanup follows the last exit.
	for range 2 {
		h.start("echo")
		h.sendControl(frame.ControlMessage{Type: frame.TypeStop, Stop: &frame.Stop{GraceMillis: 100}})
		h.control(frame.TypeExited)
	}
	h.sendControl(frame.ControlMessage{Type: frame.TypeClearPrivateHome, Request: 8, ClearPrivateHome: &frame.ClearPrivateHome{}})
	h.control(frame.TypePrivateHomeCleared)
	for path := range files {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("%s remains after cleanup", path)
		}
	}
}

func TestClaudeHomeWithoutCredentialsInstallsOnlyInstructions(t *testing.T) {
	h := newHarness(t)
	h.sendControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 1, Initialize: &frame.Initialize{Claude: &frame.ClaudeHome{SystemPrompt: "BASE", AppendSystemPrompt: "DEVELOPER"}}})
	h.control(frame.TypeInitialized)
	if _, err := os.Lstat(filepath.Join(h.home, ".claude")); !os.IsNotExist(err) {
		t.Fatal("a token login created a credentials directory")
	}
	if data, err := os.ReadFile(filepath.Join(h.home, ".markitect", "system-prompt.md")); err != nil || string(data) != "BASE" {
		t.Fatal("instructions not installed", err)
	}
}

func TestClaudeHomeRefusesAnExistingFileWithoutChangingIt(t *testing.T) {
	h := newHarness(t)
	dir := filepath.Join(h.home, ".markitect")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "system-prompt.md")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	h.sendControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 1, Initialize: &frame.Initialize{Claude: &frame.ClaudeHome{Credentials: []byte("secret-login"), SystemPrompt: "BASE", AppendSystemPrompt: "DEVELOPER"}}})
	if c := h.control(frame.TypeFailure); strings.Contains(c.Failure.Message, "secret") {
		t.Fatal("setup failure leaked private data")
	}
	if data, _ := os.ReadFile(path); string(data) != "existing" {
		t.Fatal("refused setup changed an existing file")
	}
	h.sendControl(frame.ControlMessage{Type: frame.TypeStart, Request: 2, Start: &frame.Start{Argv: []string{os.Args[0]}, Dir: h.home}})
	h.control(frame.TypeFailure)
}

func TestToolMetaReachesTheProviderCatalogue(t *testing.T) {
	h := newHarness(t)
	tool := submitWork
	tool.Meta = json.RawMessage(`{"anthropic/maxResultSizeChars":500000}`)
	h.start("client", tool)
	var call frame.ToolCall
	for {
		f := h.next()
		if f.Channel == frame.Tool {
			var err error
			if call, err = frame.DecodeToolCall(f.Payload); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	payload, _ := frame.EncodeToolReply(frame.ToolReply{ID: call.ID, Result: json.RawMessage(`{"content":[{"type":"text","text":"accepted"}]}`)})
	h.send(frame.Frame{Channel: frame.Tool, Payload: payload})
	out := h.output(frame.ProviderOut, "accepted")
	if !strings.Contains(out, `"_meta":{"anthropic/maxResultSizeChars":500000}`) {
		t.Fatalf("catalogue lacks the tool meta: %q", out)
	}
	h.control(frame.TypeExited)
}

func TestToolMetaMustBeAnObject(t *testing.T) {
	h := newHarness(t)
	tool := submitWork
	tool.Meta = json.RawMessage(`[1]`)
	h.sendControl(frame.ControlMessage{Type: frame.TypeStart, Start: &frame.Start{Argv: []string{os.Args[0]}, Dir: t.TempDir(), Tools: []frame.ToolSpec{tool}}})
	h.control(frame.TypeFailure)
	select {
	case err := <-h.served:
		if err == nil {
			t.Fatal("a malformed tool catalogue ended the session without an error")
		}
		h.served <- err
	case <-time.After(10 * time.Second):
		t.Fatal("session continued with a malformed tool catalogue")
	}
}
