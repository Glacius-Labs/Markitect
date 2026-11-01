package agents_test

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/agents/claude"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

// chatTree is a working tree with a binary file and names that need care.
var chatTree = map[string]string{
	".markitect/model/project.md":        "---\nkind: std/Project\n---\n# Library\n",
	".markitect/model/members/member.md": "---\nkind: std/Concept\n---\n# Member\n",
	"src/a.go":                           "package a\n",
	"docs/with space.md":                 "# Spaces\n",
	"docs/größe.md":                      "# Umlaut\n",
	"assets/logo.bin":                    string([]byte{0, 1, 2, 255, '\n', 0}),
}

func chatSpec(t *testing.T, pending ...agents.ChatFile) agents.ChatSpec {
	var files []string
	for f := range chatTree {
		files = append(files, f)
	}
	sort.Strings(files)
	return agents.ChatSpec{
		Name: "markitect-chat-0123456789abcdef", Labels: map[string]string{"markitect.chat": "0123456789abcdef"}, Image: "sha256:" + strings.Repeat("a", 64),
		Limits: agents.Limits{CPUs: 1, MemoryBytes: 2 << 30, Pids: 256, TmpBytes: 256 << 20}, Network: "markitect-provider",
		Login: login(t, time.Now().Add(2*time.Hour)), Model: "claude-sonnet-5-5", Effort: "high",
		Skill: harness.ChatModeling, Earlier: "owner: Members borrow books.\n\nagent: Noted.",
		Files: files,
		Read: func(f string) ([]byte, error) {
			data, ok := chatTree[f]
			if !ok {
				return nil, os.ErrNotExist
			}
			return []byte(data), nil
		},
		Pending: pending,
		Turn:    30 * time.Minute,
	}
}

func readWork(t *testing.T, rt *fakeRuntime, f string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(rt.work, filepath.FromSlash(f)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAChatCopiesTheWorkingTreeAndProposesItsChanges(t *testing.T) {
	rt := newRuntime(t, t.TempDir())
	pending := []agents.ChatFile{
		{Path: ".markitect/model/members/member.md", Data: []byte("---\nkind: std/Concept\n---\n# Member\n\nSomeone who borrows.\n")},
		{Path: ".markitect/model/fee.md", Data: []byte("---\nkind: std/Rule\n---\n# Late fee\n")},
		{Path: ".markitect/model/project.md", Removed: true},
	}
	chat, err := agents.StartChat(context.Background(), rt, claude.Provider{}, chatSpec(t, pending...))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = chat.Close(context.Background()) })

	// The copy holds every file as it is, with the pending proposal on top;
	// the start holds the working tree.
	if got := readWork(t, rt, "assets/logo.bin"); got != chatTree["assets/logo.bin"] {
		t.Errorf("the binary file arrived as %q", got)
	}
	for _, f := range []string{"docs/with space.md", "docs/größe.md", "src/a.go"} {
		if got := readWork(t, rt, f); got != chatTree[f] {
			t.Errorf("%s arrived as %q", f, got)
		}
	}
	if got := readWork(t, rt, ".markitect/model/members/member.md"); !strings.Contains(got, "Someone who borrows.") {
		t.Errorf("the pending change is not in the copy: %q", got)
	}
	if _, err := os.Stat(filepath.Join(rt.work, ".markitect", "model", "project.md")); !os.IsNotExist(err) {
		t.Errorf("the pending removal is not in the copy: %v", err)
	}
	if got := git(t, rt.work, "show", harness.ChatStart+":.markitect/model/members/member.md"); got != strings.TrimSpace(chatTree[".markitect/model/members/member.md"]) {
		t.Errorf("the start holds %q", got)
	}

	// The chat's documents: its role and skill, no Manager tools, and the
	// conversation so far before the owner's first message.
	if home := rt.session.initialized.Claude; home == nil || !strings.Contains(home.SystemPrompt, "# Markitect chat") || !strings.Contains(home.SystemPrompt, "markitect-modeling/SKILL.md") {
		t.Fatalf("the chat's home %+v", rt.session.initialized)
	}
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		os.WriteFile(filepath.Join(s.rt.work, ".markitect", "model", "loan.md"), []byte("---\nkind: std/Concept\n---\n# Loan\n"), 0o644)
		os.WriteFile(filepath.Join(s.rt.work, "src", "a.go"), []byte("package a // touched\n"), 0o644)
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":"I added the Loan."`, 1))
		return 0
	}
	var heard []agents.Progress
	out, err := chat.Send(context.Background(), "Add the loan.", func(p agents.Progress) { heard = append(heard, p) })
	if err != nil || out.End != agents.Answered || out.Text != "I added the Loan." {
		t.Fatalf("turn %+v %v", out, err)
	}
	if len(heard) != 1 || heard[0].Text != "I added the Loan." {
		t.Errorf("the final answer was heard as %+v", heard)
	}
	if in := rt.session.stdin.String(); !strings.Contains(in, "The conversation so far") || !strings.Contains(in, "Members borrow books.") || !strings.Contains(in, "Add the loan.") {
		t.Errorf("first message %s", in)
	}
	if argv := strings.Join(rt.session.argv[1], " "); strings.Contains(argv, "mcp__markitect__") {
		t.Errorf("a chat offers a Manager tool: %s", argv)
	}

	model, outside, err := chat.Changes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range model {
		got[f.Path] = string(f.Data)
		if f.Removed {
			got[f.Path] = "removed"
		}
	}
	if len(got) != 4 || got[".markitect/model/project.md"] != "removed" || !strings.Contains(got[".markitect/model/loan.md"], "# Loan") || !strings.Contains(got[".markitect/model/fee.md"], "Late fee") || !strings.Contains(got[".markitect/model/members/member.md"], "borrows") {
		t.Errorf("model changes %v", got)
	}
	if strings.Join(outside, ",") != "src/a.go" {
		t.Errorf("outside %v", outside)
	}

	// An accept makes the copy's model its start, while the change outside
	// stays a change; a reject puts the copy back.
	if err := chat.Settle(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if model, outside, err := chat.Changes(context.Background()); err != nil || len(model) != 0 || strings.Join(outside, ",") != "src/a.go" {
		t.Fatalf("after the accept: %v %v %v", model, outside, err)
	}
	os.WriteFile(filepath.Join(rt.work, ".markitect", "model", "loan.md"), []byte("changed again\n"), 0o644)
	os.WriteFile(filepath.Join(rt.work, ".markitect", "model", "stray.md"), []byte("stray\n"), 0o644)
	if err := chat.Settle(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if model, _, err := chat.Changes(context.Background()); err != nil || len(model) != 0 {
		t.Fatalf("after the reject: %v %v", model, err)
	}
	if got := readWork(t, rt, ".markitect/model/loan.md"); !strings.Contains(got, "# Loan") {
		t.Errorf("the reject left %q", got)
	}
}

// The copy is compared as it is on disk: the project's line-end attributes
// make no unchanged file look changed, and a nested repository is left out.
func TestAChatComparesTheCopyAsItIsOnDisk(t *testing.T) {
	saved := maps.Clone(chatTree)
	t.Cleanup(func() { chatTree = saved })
	chatTree[".gitattributes"] = "* text eol=lf\n"
	chatTree[".markitect/model/windows.md"] = "---\r\nkind: std/Concept\r\n---\r\n# Windows\r\n"
	rt := newRuntime(t, t.TempDir())
	chat, err := agents.StartChat(context.Background(), rt, claude.Provider{}, chatSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = chat.Close(context.Background()) })
	if model, outside, err := chat.Changes(context.Background()); err != nil || len(model)+len(outside) != 0 {
		t.Fatalf("an untouched copy changed: %v %v %v", model, outside, err)
	}
	nested := filepath.Join(rt.work, ".markitect", "model", "nested")
	git(t, filepath.Dir(nested), "init", "--quiet", nested)
	os.WriteFile(filepath.Join(nested, "x.md"), []byte("x\n"), 0o644)
	if model, _, err := chat.Changes(context.Background()); err != nil {
		t.Fatalf("a nested repository: %v %v", model, err)
	}

	// A turn is heard as it runs: its messages and what it does, the final
	// answer once though it repeats the last message, and never a secret of
	// the login copy.
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"text","text":"Your token is sk-ant-oat01-ACCESS-SECRET."}]},"session_id":"s-1"}`)
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"echo sk-ant-oat01-ACCESS-SECRET"}}]},"session_id":"s-1"}`)
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"text","text":"Done."}]},"session_id":"s-1"}`)
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":"Done."`, 1))
		return 0
	}
	var heard []agents.Progress
	out, err := chat.Send(context.Background(), "Tell me a secret.", func(p agents.Progress) { heard = append(heard, p) })
	if err != nil || out.Text != "Done." || len(heard) != 3 || heard[0].Text != "Your token is [redacted]." || heard[1].Doing != "Running echo [redacted]" || heard[2].Text != "Done." {
		t.Fatalf("the turn %+v, heard %+v, %v", out, heard, err)
	}

	// An activity is shortened only after its secret is gone, and a long one
	// stays one short line. The limit falls inside the secret, which would be
	// no longer recognized if it were cut first.
	secret := "sk-ant-oat01-ACCESS-SECRET"
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		long := strings.Repeat("x", 90) + " " + secret + " " + strings.Repeat("y", 100)
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"` + long + `"}}]},"session_id":"s-1"}`)
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":""`, 1))
		return 0
	}
	heard = nil
	if _, err := chat.Send(context.Background(), "A long one.", func(p agents.Progress) { heard = append(heard, p) }); err != nil || len(heard) != 1 {
		t.Fatalf("heard %+v, %v", heard, err)
	}
	if d := heard[0].Doing; strings.Contains(d, "sk-ant") || !strings.Contains(d, "[redacted]") || len([]rune(d)) > 110 || !strings.HasPrefix(d, "Running xxx") {
		t.Errorf("the activity %q", d)
	}

	// A provider that reports none of its messages along the way still has
	// its final answer heard, redacted.
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":"Your token is sk-ant-oat01-ACCESS-SECRET."`, 1))
		return 0
	}
	heard = nil
	if out, err := chat.Send(context.Background(), "Again.", func(p agents.Progress) { heard = append(heard, p) }); err != nil || strings.Contains(out.Text, "SECRET") || len(heard) != 1 || strings.Contains(heard[0].Text, "SECRET") {
		t.Fatalf("the answer %q, heard %+v, %v", out.Text, heard, err)
	}
}

func TestAChatRefusesALoginThatExpiresWithinATurn(t *testing.T) {
	rt := newRuntime(t, t.TempDir())
	spec := chatSpec(t)
	spec.Login = login(t, time.Now().Add(20*time.Minute))
	var start *agents.StartError
	if _, err := agents.StartChat(context.Background(), rt, claude.Provider{}, spec); !errors.As(err, &start) || start.Class != agents.LoginFailed {
		t.Fatalf("start with a short-lived login: %v", err)
	}

	// A chat whose login copy would expire later refuses that turn, so a
	// fresh agent with a fresh copy takes over.
	spec = chatSpec(t)
	now := time.Now()
	spec.Now = func() time.Time { return now }
	chat, err := agents.StartChat(context.Background(), rt, claude.Provider{}, spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = chat.Close(context.Background()) })
	now = now.Add(90 * time.Minute)
	if out, err := chat.Send(context.Background(), "Hello", func(agents.Progress) {}); err != nil || out.End != agents.Failed || out.Class != agents.LoginFailed {
		t.Fatalf("a turn past the login: %+v %v", out, err)
	}
}

func TestAChatNeedsItsSettings(t *testing.T) {
	rt := newRuntime(t, t.TempDir())
	spec := chatSpec(t)
	spec.Network = ""
	if _, err := agents.StartChat(context.Background(), rt, claude.Provider{}, spec); err == nil {
		t.Error("a chat without the provider's network started")
	}
	spec = chatSpec(t)
	spec.Skill = "poetry"
	if _, err := agents.StartChat(context.Background(), rt, claude.Provider{}, spec); err == nil {
		t.Error("a chat with an unknown skill started")
	}
}

func TestImportPathQuotesWhatFastImportNeeds(t *testing.T) {
	for in, want := range map[string]string{
		"a/b.md":      `"a/b.md"`,
		`a "q".md`:    `"a \"q\".md"`,
		`back\slash`:  `"back\\slash"`,
		"line\nbreak": `"line\nbreak"`,
		"tab\there":   `"tab\011here"`,
		"größe.md":    `"größe.md"`,
	} {
		if got := agents.ImportPath(in); got != want {
			t.Errorf("%q: %s, want %s", in, got, want)
		}
	}
}
