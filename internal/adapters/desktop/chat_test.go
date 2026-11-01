package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeChats starts scripted chat agents. Each agent works on its own copy of
// the files it started from; a turn runs turn on that copy and says what it
// returns.
type fakeChats struct {
	mu      sync.Mutex
	started []ChatStart
	agents  []*fakeAgent
	fail    error
	turn    func(copy map[string]string, message string) string
	// hold, when set, keeps Start from returning until it is closed.
	hold chan struct{}
	// progress, when set, runs at the start of a turn with its say and doing.
	progress func(say, doing func(string))
}

func (f *fakeChats) Start(_ context.Context, s ChatStart) (ChatAgent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, s)
	if f.fail != nil {
		return nil, f.fail
	}
	files := map[string]string{}
	for _, file := range s.Files {
		data, err := s.Tree.Read(file)
		if err != nil {
			return nil, err
		}
		files[file] = string(data)
	}
	a := &fakeAgent{chats: f, start: files, copy: maps.Clone(files)}
	for _, p := range s.Pending {
		if p.Removed {
			delete(a.copy, p.Path)
		} else {
			a.copy[p.Path] = string(p.Data)
		}
	}
	f.agents = append(f.agents, a)
	if f.hold != nil {
		f.mu.Unlock()
		<-f.hold
		f.mu.Lock()
	}
	return a, nil
}

type fakeAgent struct {
	chats       *fakeChats
	mu          sync.Mutex
	start, copy map[string]string
	settled     []bool
	closed      bool
}

func (a *fakeAgent) Send(_ context.Context, message string, say, doing func(string)) (ChatEnd, error) {
	if a.chats.progress != nil {
		a.chats.progress(say, doing)
	}
	a.mu.Lock()
	answer := a.chats.turn(a.copy, message)
	a.mu.Unlock()
	say(answer)
	return ChatEnd{End: "answered"}, nil
}

func (a *fakeAgent) Changes(context.Context) ([]ChatFile, []string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var model []ChatFile
	var outside []string
	add := func(f string, c ChatFile) {
		if strings.HasPrefix(f, ".markitect/") {
			model = append(model, c)
		} else {
			outside = append(outside, f)
		}
	}
	for f, text := range a.copy {
		if old, ok := a.start[f]; !ok || old != text {
			add(f, ChatFile{Path: f, Data: []byte(text)})
		}
	}
	for f := range a.start {
		if _, ok := a.copy[f]; !ok {
			add(f, ChatFile{Path: f, Removed: true})
		}
	}
	return model, outside, nil
}

func (a *fakeAgent) Settle(_ context.Context, keep bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.settled = append(a.settled, keep)
	if keep {
		a.start = maps.Clone(a.copy)
	} else {
		a.copy = maps.Clone(a.start)
	}
	return nil
}

func (a *fakeAgent) Close(context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	return nil
}

const fee = "---\nkind: std/Rule\n---\n# Late fee\n\nA late book costs `1` per day.\n"

// feeChat adds a Rule, rewrites the loan limit and touches a source file,
// which is not proposed.
func feeChat(copy map[string]string, message string) string {
	copy[".markitect/model/fee.md"] = fee
	copy[".markitect/model/policy.md"] = "---\nkind: std/Rule\n---\n# Loan limit\n\nA member holds at most `5` open loans.\n"
	copy["src/loans/loan.go"] = "package loans // touched\n"
	return "I added the late fee and raised the limit, as you asked: " + message
}

func chatServer(t *testing.T, turn func(map[string]string, string) string) (*server, *fakeChats, map[string]string, string) {
	t.Helper()
	files := library(t)
	s, _, _, id := modelServer(t, files)
	chats := &fakeChats{turn: turn}
	s.chat.chats = chats
	t.Cleanup(s.chat.close)
	return s, chats, files, id
}

// chatState waits until the conversation's turn ended and returns it.
func chatState(t *testing.T, s *server, id, chat string) chatView {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		w := call(s, "GET", "/api/projects/"+id+"/chats/"+chat, "")
		var v chatView
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || w.Code != http.StatusOK {
			t.Fatalf("%d %s", w.Code, w.Body)
		}
		if !v.Working {
			return v
		}
		if time.Now().After(deadline) {
			t.Fatal("the turn did not end")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func startChat(t *testing.T, s *server, id, body string) string {
	t.Helper()
	w := call(s, "POST", "/api/projects/"+id+"/chats", body)
	if w.Code != http.StatusOK {
		t.Fatalf("new chat: %d %s", w.Code, w.Body)
	}
	return decode(t, w)["id"].(string)
}

func TestAChatShowsItsMessagesAndWorkWhileTheTurnRuns(t *testing.T) {
	s, chats, _, id := chatServer(t, func(copy map[string]string, message string) string { return "Done." })
	step := make(chan struct{})
	chats.progress = func(say, doing func(string)) {
		doing("Reading model/policy.md")
		step <- struct{}{}
		<-step
		say("I read the policy.")
		step <- struct{}{}
		<-step
		doing("Running markitect check")
		step <- struct{}{}
		<-step
	}
	chat := startChat(t, s, id, `{"text":"Look at the policy."}`)
	current := func() chatView {
		var v chatView
		if err := json.Unmarshal(call(s, "GET", "/api/projects/"+id+"/chats/"+chat, "").Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	<-step
	if v := current(); !v.Working || v.Doing != "Reading model/policy.md" || len(v.Messages) != 1 {
		t.Fatalf("while it reads: %+v", v)
	}
	step <- struct{}{}
	<-step
	// A message arrives during the turn, and what it did before is over.
	if v := current(); !v.Working || v.Doing != "" || len(v.Messages) != 2 || v.Messages[1].Text != "I read the policy." {
		t.Fatalf("after its message: %+v", v)
	}
	step <- struct{}{}
	<-step
	if v := current(); v.Doing != "Running markitect check" {
		t.Fatalf("while it runs: %+v", v)
	}
	step <- struct{}{}
	v := chatState(t, s, id, chat)
	if v.Doing != "" || len(v.Messages) != 3 || v.Messages[2].Text != "Done." {
		t.Fatalf("after the turn: %+v", v)
	}
}

func TestAChatProposesAndTheOwnerAccepts(t *testing.T) {
	s, chats, files, id := chatServer(t, feeChat)
	chat := startChat(t, s, id, `{"text":"Add a late fee and raise the loan limit to five."}`)
	v := chatState(t, s, id, chat)
	if v.Title != "Add a late fee and raise the loan limit to five." || v.Skill != skillModeling {
		t.Errorf("conversation %q %q", v.Title, v.Skill)
	}
	if len(v.Messages) != 2 || v.Messages[0].From != "owner" || v.Messages[1].From != "agent" || !strings.Contains(v.Messages[1].Text, "late fee") {
		t.Fatalf("messages %+v", v.Messages)
	}
	if len(chats.started) != 1 || chats.started[0].Earlier != "" || chats.started[0].Skill != skillModeling {
		t.Errorf("starts %+v", chats.started)
	}
	open := v.open()
	if open == nil || len(open.Files) != 2 || open.Files[0].Path != ".markitect/model/fee.md" || open.Files[0].Change != "added" || open.Files[1].Change != "modified" || strings.Join(open.Outside, ",") != "src/loans/loan.go" {
		t.Fatalf("proposal %+v", open)
	}

	// The views show the proposal on top of the working tree.
	m, res := getModel(t, s, id, "?chat="+chat)
	if res.code != http.StatusOK || m.Base == nil || m.Base.Kind != baseTree || m.definition("fee").Change != "added" || !strings.Contains(m.definition("policy").Text, "`5`") {
		t.Fatalf("proposed model: %d %+v", res.code, m.Base)
	}
	if m.Impact == nil || len(m.Impact.Definitions) != 2 {
		t.Errorf("the proposal's impact %+v", m.Impact)
	}
	if plain, _ := getModel(t, s, id, ""); plain.definition("fee").ID != "" {
		t.Error("the working tree shows the proposal before it is accepted")
	}

	w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/accept", "{}")
	if w.Code != http.StatusOK {
		t.Fatalf("accept: %d %s", w.Code, w.Body)
	}
	if files[".markitect/model/fee.md"] != fee || !strings.Contains(files[".markitect/model/policy.md"], "`5`") || files["src/loans/loan.go"] != "package loans\n" {
		t.Fatal("the working tree does not hold exactly the proposal")
	}
	v = chatState(t, s, id, chat)
	if v.Proposals[0].State != proposalAccepted || v.open() != nil || v.Messages[len(v.Messages)-1].From != "app" {
		t.Errorf("after the accept: %+v", v.Proposals)
	}
	if a := chats.agents[0]; len(a.settled) != 1 || !a.settled[0] {
		t.Errorf("the agent settled %v", a.settled)
	}
	if _, res := getModel(t, s, id, "?chat="+chat); res.code != http.StatusNotFound {
		t.Errorf("an accepted proposal still shows: %d", res.code)
	}
}

func TestAnAcceptWritesNothingOverAChangedWorkingTree(t *testing.T) {
	s, chats, files, id := chatServer(t, feeChat)
	chat := startChat(t, s, id, `{"text":"Add a late fee."}`)
	chatState(t, s, id, chat)
	files[".markitect/model/policy.md"] = "---\nkind: std/Rule\n---\n# Loan limit\n\nChanged by hand meanwhile.\n"

	w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/accept", "{}")
	var v chatView
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	if w.Code != http.StatusConflict || v.open() == nil || strings.Join(v.open().Conflicts, ",") != ".markitect/model/policy.md" {
		t.Fatalf("accept over a change: %d %s", w.Code, w.Body)
	}
	if _, ok := files[".markitect/model/fee.md"]; ok {
		t.Fatal("part of the proposal was written")
	}

	w = call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/reject", "{}")
	if w.Code != http.StatusOK {
		t.Fatalf("reject: %d %s", w.Code, w.Body)
	}
	if v = chatState(t, s, id, chat); v.Proposals[0].State != proposalRejected {
		t.Errorf("after the reject: %+v", v.Proposals)
	}
	// The conflict dropped the agent, so there is none to settle.
	if a := chats.agents[0]; len(a.settled) != 0 || !a.closed {
		t.Errorf("the agent settled %v, closed %v", a.settled, a.closed)
	}
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/accept", "{}"); w.Code != http.StatusConflict {
		t.Errorf("accepting with no proposal: %d", w.Code)
	}
}

func TestTheHubNamesAProposalOnceTheOwnerCanDecideIt(t *testing.T) {
	s, chats, _, id := chatServer(t, feeChat)
	step := make(chan struct{})
	chats.progress = func(say, doing func(string)) {
		step <- struct{}{}
		<-step
	}
	chat := startChat(t, s, id, `{"text":"Add a late fee."}`)
	<-step
	if w := waitingNow(t, s, id); w.Proposal != nil {
		t.Fatalf("during the first turn: %+v", w.Proposal)
	}
	step <- struct{}{}
	chatState(t, s, id, chat)
	if w := waitingNow(t, s, id); w.Proposal == nil || w.Proposal.Chat != chat || w.Proposal.Title != "Add a late fee." {
		t.Fatalf("after the turn: %+v", w.Proposal)
	}
	// A later turn may change the proposal, so it is not offered meanwhile.
	if r := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/messages", `{"text":"And once more."}`); r.Code != http.StatusOK {
		t.Fatalf("message: %d %s", r.Code, r.Body)
	}
	<-step
	if w := waitingNow(t, s, id); w.Proposal != nil {
		t.Fatalf("during a later turn: %+v", w.Proposal)
	}
	step <- struct{}{}
	chatState(t, s, id, chat)
	w := waitingNow(t, s, id)
	if w.Proposal == nil {
		t.Fatal("after the later turn: no proposal")
	}
	before := w.Uncommitted
	if r := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/accept", "{}"); r.Code != http.StatusOK {
		t.Fatalf("accept: %d %s", r.Code, r.Body)
	}
	// Accepted, it waits for a commit.
	if w := waitingNow(t, s, id); w.Proposal != nil || w.Uncommitted <= before {
		t.Fatalf("after the accept: %+v, %d before", w, before)
	}
}

func TestALaterTurnReplacesTheProposal(t *testing.T) {
	turns := 0
	s, _, _, id := chatServer(t, func(copy map[string]string, message string) string {
		turns++
		if turns == 1 {
			return feeChat(copy, message)
		}
		delete(copy, ".markitect/model/fee.md")
		return "The fee is gone again."
	})
	chat := startChat(t, s, id, `{"text":"Add a late fee."}`)
	chatState(t, s, id, chat)
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/messages", `{"text":"Drop the fee."}`); w.Code != http.StatusOK {
		t.Fatalf("second message: %d %s", w.Code, w.Body)
	}
	v := chatState(t, s, id, chat)
	if len(v.Proposals) != 2 || v.Proposals[0].State != proposalSuperseded || v.open() == nil || len(v.open().Files) != 1 || v.open().Files[0].Path != ".markitect/model/policy.md" {
		t.Fatalf("proposals %+v", v.Proposals)
	}
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/messages", `{"text":"x","skill":"extract"}`); w.Code != http.StatusBadRequest {
		t.Errorf("changing the skill: %d", w.Code)
	}
}

func TestAChatAnswersOneMessageAtATime(t *testing.T) {
	release := make(chan struct{})
	s, _, _, id := chatServer(t, func(copy map[string]string, message string) string {
		<-release
		return "done"
	})
	chat := startChat(t, s, id, `{"text":"Think about it."}`)
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/messages", `{"text":"And now?"}`); w.Code != http.StatusConflict {
		t.Errorf("a second message during a turn: %d", w.Code)
	}
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/accept", "{}"); w.Code != http.StatusConflict {
		t.Errorf("an accept during a turn: %d", w.Code)
	}
	if w := call(s, "DELETE", "/api/projects/"+id+"/chats/"+chat, ""); w.Code != http.StatusConflict {
		t.Errorf("a delete during a turn: %d", w.Code)
	}
	close(release)
	if v := chatState(t, s, id, chat); len(v.Messages) != 2 {
		t.Errorf("messages %+v", v.Messages)
	}
	if w := call(s, "DELETE", "/api/projects/"+id+"/chats/"+chat, ""); w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	if w := call(s, "GET", "/api/projects/"+id+"/chats/"+chat, ""); w.Code != http.StatusNotFound {
		t.Errorf("a deleted chat: %d", w.Code)
	}
}

func TestAReopenedChatStartsAFreshAgentWithTheConversation(t *testing.T) {
	s, chats, _, id := chatServer(t, func(copy map[string]string, message string) string { return "noted: " + message })
	chat := startChat(t, s, id, `{"text":"Members may borrow three books."}`)
	chatState(t, s, id, chat)
	// The app restarts: the conversation stays, its agent does not.
	s.chat.close()
	s.chat = newChatService(s.chat.dir, chats, s.repos, &s.edits)
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/messages", `{"text":"And renew them once."}`); w.Code != http.StatusOK {
		t.Fatalf("message after the restart: %d %s", w.Code, w.Body)
	}
	chatState(t, s, id, chat)
	if len(chats.started) != 2 {
		t.Fatalf("starts %d", len(chats.started))
	}
	earlier := chats.started[1].Earlier
	if !strings.Contains(earlier, "owner: Members may borrow three books.") || !strings.Contains(earlier, "agent: noted:") || strings.Contains(earlier, "renew") {
		t.Errorf("earlier %q", earlier)
	}
	if !chats.agents[0].closed {
		t.Error("the first agent outlived the app")
	}
	w := call(s, "GET", "/api/projects/"+id+"/chats", "")
	list := decode(t, w)
	if list["available"] != true || len(list["chats"].([]any)) != 1 {
		t.Errorf("list %v", list)
	}
}

func TestAChatThatCannotStartSaysWhy(t *testing.T) {
	s, chats, _, id := chatServer(t, feeChat)
	chats.fail = errors.New("Docker is not running")
	chat := startChat(t, s, id, `{"text":"Hello"}`)
	v := chatState(t, s, id, chat)
	if last := v.Messages[len(v.Messages)-1]; last.From != "app" || !strings.Contains(last.Text, "Docker is not running") {
		t.Errorf("messages %+v", v.Messages)
	}

	s.chat.chats = nil
	if w := call(s, "POST", "/api/projects/"+id+"/chats", `{"text":"Hello"}`); w.Code != http.StatusServiceUnavailable {
		t.Errorf("a chat without agents: %d", w.Code)
	}
	if list := decode(t, call(s, "GET", "/api/projects/"+id+"/chats", "")); list["available"] != false {
		t.Errorf("list %v", list)
	}
	for _, body := range []string{`{"text":"  "}`, `{"text":"x","skill":"poetry"}`, `{"text":"x","model":"y"}`} {
		s.chat.chats = chats
		if w := call(s, "POST", "/api/projects/"+id+"/chats", body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", body, w.Code)
		}
	}
	if w := call(s, "GET", "/api/projects/"+id+"/chats/../x", ""); w.Code == http.StatusOK {
		t.Error("a chat outside the folder")
	}
}

func TestAChatProposesOnlyFilesBelowMarkitectsFolder(t *testing.T) {
	s, _, files, id := chatServer(t, func(copy map[string]string, message string) string {
		copy[".markitect/model/fee.md"] = fee
		copy[".markitect/../src/escape.go"] = "package escape\n"
		copy["../outside.md"] = "outside\n"
		copy[".markitect/model/CON.md"] = "device\n"
		copy[".markitect/model/./policy.md"] = "unclean\n"
		return "done"
	})
	chat := startChat(t, s, id, `{"text":"Write a few files."}`)
	v := chatState(t, s, id, chat)
	open := v.open()
	if open == nil || len(open.Files) != 1 || open.Files[0].Path != ".markitect/model/fee.md" || len(open.Outside) != 4 {
		t.Fatalf("proposal %+v", open)
	}
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/accept", "{}"); w.Code != http.StatusOK {
		t.Fatalf("accept: %d %s", w.Code, w.Body)
	}
	for f := range files {
		if strings.Contains(f, "..") || strings.Contains(f, "CON") || strings.Contains(f, "/./") || strings.Contains(f, "escape") {
			t.Errorf("accept wrote %s", f)
		}
	}
}

func TestAConflictLetsTheNextTurnStartFreshWithTheProposal(t *testing.T) {
	turns := 0
	s, chats, files, id := chatServer(t, func(copy map[string]string, message string) string {
		turns++
		if turns == 1 {
			return feeChat(copy, message)
		}
		return "I took your edit into account."
	})
	chat := startChat(t, s, id, `{"text":"Add a late fee."}`)
	chatState(t, s, id, chat)
	files[".markitect/model/policy.md"] = "---\nkind: std/Rule\n---\n# Loan limit\n\nChanged by hand.\n"
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/accept", "{}"); w.Code != http.StatusConflict {
		t.Fatalf("accept: %d", w.Code)
	}
	if !chats.agents[0].closed {
		t.Error("the agent outlived the conflict")
	}
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/messages", `{"text":"I changed the limit by hand."}`); w.Code != http.StatusOK {
		t.Fatalf("message: %d %s", w.Code, w.Body)
	}
	v := chatState(t, s, id, chat)
	// The fresh agent starts from the working tree as it is: it takes over
	// the proposal's new Rule, but sees the owner's version of the edited
	// policy instead of the proposal's.
	if len(chats.started) != 2 || len(chats.started[1].Pending) != 1 || chats.started[1].Pending[0].Path != ".markitect/model/fee.md" {
		t.Fatalf("the fresh agent got %+v", chats.started)
	}
	if open := v.open(); open == nil || len(open.Files) != 1 || open.N != 2 {
		t.Fatalf("proposal after the fresh start %+v", v.Proposals)
	}
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/accept", "{}"); w.Code != http.StatusOK {
		t.Fatalf("accept after the fresh start: %d %s", w.Code, w.Body)
	}
	if files[".markitect/model/fee.md"] != fee || !strings.Contains(files[".markitect/model/policy.md"], "Changed by hand.") {
		t.Error("the accept did not keep the hand edit and write the Rule")
	}
}

func TestAnAcceptCutOffMidwayCompletesWhenRepeated(t *testing.T) {
	s, _, files, id := chatServer(t, feeChat)
	chat := startChat(t, s, id, `{"text":"Add a late fee."}`)
	v := chatState(t, s, id, chat)
	// The first file of the proposal reached the working tree already.
	files[".markitect/model/fee.md"] = v.open().Files[0].Text
	if w := call(s, "POST", "/api/projects/"+id+"/chats/"+chat+"/accept", "{}"); w.Code != http.StatusOK {
		t.Fatalf("accept: %d %s", w.Code, w.Body)
	}
	if !strings.Contains(files[".markitect/model/policy.md"], "`5`") {
		t.Error("the rest of the proposal was not written")
	}
	if v = chatState(t, s, id, chat); v.Proposals[0].Files[0].Text != "" {
		t.Error("an accepted proposal keeps its texts")
	}
}

func TestQuittingWhileAnAgentStartsLeavesNoAgent(t *testing.T) {
	s, chats, _, id := chatServer(t, feeChat)
	chats.hold = make(chan struct{})
	startChat(t, s, id, `{"text":"Hello"}`)
	deadline := time.Now().Add(5 * time.Second)
	for {
		chats.mu.Lock()
		n := len(chats.agents)
		chats.mu.Unlock()
		if n == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the agent did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	closed := make(chan struct{})
	go func() {
		s.chat.close()
		close(closed)
	}()
	time.Sleep(20 * time.Millisecond)
	close(chats.hold)
	<-closed
	if a := chats.agents[0]; !a.closed {
		t.Error("an agent that started while the app quit was left running")
	}
	if w := call(s, "POST", "/api/projects/"+id+"/chats", `{"text":"Again"}`); w.Code != http.StatusServiceUnavailable {
		t.Errorf("a chat after quitting: %d", w.Code)
	}
}

func TestAnUnreadableConversationLeavesTheOthersListed(t *testing.T) {
	s, _, _, id := chatServer(t, feeChat)
	chat := startChat(t, s, id, `{"text":"Add a late fee."}`)
	chatState(t, s, id, chat)
	if err := os.WriteFile(filepath.Join(s.chat.dir, id, "0123456789abcdef.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	list := decode(t, call(s, "GET", "/api/projects/"+id+"/chats", ""))
	if chats := list["chats"].([]any); len(chats) != 1 {
		t.Errorf("list %v", list)
	}
	// Another project's route does not reach the conversation.
	other := filepath.Join(t.TempDir(), "other")
	s.repos.(*fakeRepos).repos[other] = memRepo{root: other, files: library(t)}
	w := call(s, "POST", "/api/projects", `{"action":"open","path":`+jsonString(other)+`}`)
	otherID := decode(t, w)["project"].(map[string]any)["id"].(string)
	if w := call(s, "GET", "/api/projects/"+otherID+"/chats/"+chat, ""); w.Code != http.StatusNotFound {
		t.Errorf("another project's chat: %d", w.Code)
	}
}
