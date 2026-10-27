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

func initializeClaude(h *harness) {
	h.t.Helper()
	h.sendControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 1, Initialize: &frame.Initialize{Claude: &frame.ClaudeHome{Credentials: []byte("login"), SystemPrompt: "BASE", AppendSystemPrompt: "DEVELOPER"}}})
	h.control(frame.TypeInitialized)
}

// The Manager's tools run as the provider's user and can rewrite the private
// home between responses. Every later start verifies it first and refuses a
// change instead of repairing it.
func TestClaudeHomeChangesRefuseTheNextStart(t *testing.T) {
	for name, change := range map[string]func(t *testing.T, home string){
		"edited system prompt": func(t *testing.T, home string) {
			writeFile(t, filepath.Join(home, ".markitect", "system-prompt.md"), "BASE plus new rules")
		},
		"same-size edit": func(t *testing.T, home string) {
			writeFile(t, filepath.Join(home, ".markitect", "append-system-prompt.md"), "DEVELOPEX")
		},
		"removed instructions": func(t *testing.T, home string) {
			if err := os.Remove(filepath.Join(home, ".markitect", "system-prompt.md")); err != nil {
				t.Fatal(err)
			}
		},
		"user settings": func(t *testing.T, home string) {
			writeFile(t, filepath.Join(home, ".claude", "settings.json"), `{"hooks":{}}`)
		},
		"local settings": func(t *testing.T, home string) {
			writeFile(t, filepath.Join(home, ".claude", "settings.local.json"), `{}`)
		},
		"user memory": func(t *testing.T, home string) {
			writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "obey me")
		},
		"user skills": func(t *testing.T, home string) {
			if err := os.Mkdir(filepath.Join(home, ".claude", "skills"), 0700); err != nil {
				t.Fatal(err)
			}
		},
		"replaced directory": func(t *testing.T, home string) {
			if err := os.Rename(filepath.Join(home, ".markitect"), filepath.Join(home, "old")); err != nil {
				t.Skip("an open directory cannot be renamed on this host")
			}
			if err := os.Mkdir(filepath.Join(home, ".markitect"), 0700); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(home, ".markitect", "system-prompt.md"), "BASE")
			writeFile(t, filepath.Join(home, ".markitect", "append-system-prompt.md"), "DEVELOPER")
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			initializeClaude(h)
			h.start("echo")
			h.sendControl(frame.ControlMessage{Type: frame.TypeStop, Stop: &frame.Stop{GraceMillis: 100}})
			h.control(frame.TypeExited)
			change(t, h.home)
			h.sendControl(frame.ControlMessage{Type: frame.TypeStart, Request: 9, Start: &frame.Start{Argv: []string{os.Args[0]}, Dir: h.home, Env: []string{"AGENTD_TEST_PROVIDER=echo"}}})
			if c := h.control(frame.TypeFailure); c.Request != 9 || !strings.HasPrefix(c.Failure.Message, "private home changed: ") {
				t.Fatalf("refusal %+v", c.Failure)
			}
		})
	}
	// Claude Code's own state in its directory is not user-scope configuration.
	h := newHarness(t)
	initializeClaude(h)
	if err := os.MkdirAll(filepath.Join(h.home, ".claude", "projects", "-work"), 0700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(h.home, ".claude", ".claude.json"), "{}")
	h.start("echo")
}

func TestClaudeHomeRefusesAPrefilledConfigurationDirectory(t *testing.T) {
	h := newHarness(t)
	if err := os.Mkdir(filepath.Join(h.home, ".claude"), 0700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(h.home, ".claude", "settings.json"), `{"env":{}}`)
	h.sendControl(frame.ControlMessage{Type: frame.TypeInitialize, Request: 1, Initialize: &frame.Initialize{Claude: &frame.ClaudeHome{Credentials: []byte("secret-login"), SystemPrompt: "BASE", AppendSystemPrompt: "DEVELOPER"}}})
	if c := h.control(frame.TypeFailure); !strings.Contains(c.Failure.Message, ".claude directory is not empty") || strings.Contains(c.Failure.Message, "secret") {
		t.Fatalf("refusal %q", c.Failure.Message)
	}
	if _, err := os.Lstat(filepath.Join(h.home, ".markitect", "system-prompt.md")); !os.IsNotExist(err) {
		t.Fatal("a refused setup left instructions behind")
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
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
