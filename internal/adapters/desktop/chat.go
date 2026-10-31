package desktop

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The chat is the main way to change the model: a conversation with an agent
// that works on a copy of the project in its own container and proposes a
// model change, which the owner accepts into the working tree or rejects
// (concept 2.0 §8). The agents are the composition's; this file keeps the
// conversations, runs their turns and decides what reaches the working tree.

// Chats starts the agents of the app's chats.
type Chats interface {
	Start(ctx context.Context, s ChatStart) (ChatAgent, error)
}

// ChatStart is what a chat agent starts from.
type ChatStart struct {
	// Name is unique among the user's chats, for the agent's container.
	Name string
	// Tree is the working tree; Files are its files as Git sees them, which
	// the agent copies, uncommitted changes included. That copy is the
	// agent's start.
	Tree  project.Repository
	Files []string
	// Pending is the conversation's open proposal, which the agent applies
	// on top of its start, so that an agent taking over a conversation keeps
	// proposing it.
	Pending []ChatFile
	// Skill is the skill the conversation works with: modeling or extract.
	Skill string
	// Earlier is the conversation so far, for an agent that takes over a
	// conversation; empty for a new one.
	Earlier string
}

// ChatAgent is one running chat agent.
type ChatAgent interface {
	// Send runs one turn; each message the agent writes reaches say as it
	// arrives.
	Send(ctx context.Context, message string, say func(text string)) (ChatEnd, error)
	// Changes lists what the conversation changed since its start: the
	// files below Markitect's folder with their new content, and the other
	// paths it changed, which are not proposed.
	Changes(ctx context.Context) (model []ChatFile, outside []string, err error)
	// Settle makes the copy's current state its start after an accept
	// (keep), or puts the start back after a reject.
	Settle(ctx context.Context, keep bool) error
	Close(ctx context.Context) error
}

// ChatFile is one changed file of a chat's copy.
type ChatFile struct {
	Path    string
	Data    []byte
	Removed bool
}

// ChatEnd says how a turn ended: answered, failed, deadline or stopped, with
// what happened.
type ChatEnd struct {
	End    string
	Detail string
}

// The skills a conversation works with.
const (
	skillModeling = "modeling"
	skillExtract  = "extract"
)

// conversation is one chat as the app keeps it.
type conversation struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Skill    string    `json:"skill"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
	Messages []message `json:"messages"`
	// Proposals are the conversation's proposals, oldest first; only the
	// last may be open.
	Proposals []proposal `json:"proposals"`
	// Start holds the digest of each file below Markitect's folder as the
	// working tree had it when the agent started or last settled; a file
	// that did not exist has none.
	Start map[string]string `json:"start,omitempty"`
}

type message struct {
	From string    `json:"from"` // owner, agent or app
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

// The states of a proposal.
const (
	proposalOpen       = "open"
	proposalAccepted   = "accepted"
	proposalRejected   = "rejected"
	proposalSuperseded = "superseded" // a later turn changed it
	proposalWithdrawn  = "withdrawn"  // a later turn undid it
)

type proposal struct {
	N       int            `json:"n"`
	At      time.Time      `json:"at"`
	State   string         `json:"state"`
	Files   []proposedFile `json:"files"`
	Outside []string       `json:"outside,omitempty"`
	// Conflicts are the files that changed in the working tree since the
	// conversation's start, which stopped an accept.
	Conflicts []string `json:"conflicts,omitempty"`
}

type proposedFile struct {
	Path   string `json:"path"`
	Change string `json:"change"`
	Text   string `json:"text,omitempty"`
}

// open returns the conversation's open proposal, or nil.
func (c *conversation) open() *proposal {
	if n := len(c.Proposals); n > 0 && c.Proposals[n-1].State == proposalOpen {
		return &c.Proposals[n-1]
	}
	return nil
}

// proposable reports whether a chat may propose a file: a clean path below
// Markitect's folder whose names mean the same on every platform. Whatever a
// chat agent reports, nothing else reaches the working tree.
func proposable(p string) bool {
	rest, ok := strings.CutPrefix(p, project.Dir+"/")
	if !ok || path.Clean(p) != p {
		return false
	}
	for _, part := range strings.Split(rest, "/") {
		if !plainName(part) {
			return false
		}
	}
	return true
}

// chatService keeps the conversations of every project and the agents that
// run while the app does. A conversation is read from disk once and then kept
// in memory, where every change happens under mu and is written to disk at
// once; the windows read the memory, never the file a turn is writing.
type chatService struct {
	dir    string // the config folder's chats folder
	chats  Chats  // nil when the app has no chat agents
	repos  Repos
	edits  *sync.Mutex     // the server's lock of hand edits, which an accept also writes against
	ctx    context.Context // ends when the service closes
	cancel context.CancelFunc
	turns  sync.WaitGroup

	mu      sync.Mutex
	closed  bool
	convs   map[string]*conversation // by key
	live    map[string]*liveChat     // by key
	unsaved map[string]string        // by key: why the last change could not be saved
}

type liveChat struct {
	agent   ChatAgent
	working bool // a turn, an accept or a reject runs
}

// key names a conversation of a project; another project's routes do not
// reach it.
func key(p Project, id string) string { return p.ID + "/" + id }

func newChatService(dir string, chats Chats, repos Repos, edits *sync.Mutex) *chatService {
	ctx, cancel := context.WithCancel(context.Background())
	return &chatService{dir: dir, chats: chats, repos: repos, edits: edits, ctx: ctx, cancel: cancel,
		convs: map[string]*conversation{}, live: map[string]*liveChat{}, unsaved: map[string]string{}}
}

// errClosed refuses work once the app quits.
var errClosed = errors.New("Markitect is quitting")

// close ends every turn and agent: turns stop, then the agents close, each
// within ten seconds.
func (c *chatService) close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	c.cancel()
	waited := make(chan struct{})
	go func() {
		c.turns.Wait()
		close(waited)
	}()
	select {
	case <-waited:
	case <-time.After(10 * time.Second):
	}
	c.mu.Lock()
	var agents []ChatAgent
	for _, l := range c.live {
		if l.agent != nil {
			agents = append(agents, l.agent)
			l.agent = nil
		}
	}
	c.mu.Unlock()
	stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for _, a := range agents {
		wg.Go(func() { _ = a.Close(stop) })
	}
	wg.Wait()
}

func (c *chatService) file(p Project, id string) string {
	return filepath.Join(c.dir, p.ID, id+".json")
}

// get returns a conversation, read from disk the first time. The caller
// holds mu.
func (c *chatService) get(p Project, id string) (*conversation, error) {
	if conv := c.convs[key(p, id)]; conv != nil {
		return conv, nil
	}
	data, err := os.ReadFile(c.file(p, id))
	if err != nil {
		return nil, err
	}
	var conv conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("%s: %w", c.file(p, id), err)
	}
	c.convs[key(p, id)] = &conv
	return &conv, nil
}

// save writes a conversation to disk. A conversation that cannot be saved
// stays in memory while the app runs, and the window says why. The caller
// holds mu.
func (c *chatService) save(p Project, conv *conversation) {
	conv.Updated = time.Now().UTC()
	err := func() error {
		data, err := json.MarshalIndent(conv, "", "  ")
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(c.dir, p.ID), 0o700); err != nil {
			return err
		}
		return writeFile(c.file(p, conv.ID), append(data, '\n'))
	}()
	if err != nil {
		c.unsaved[key(p, conv.ID)] = err.Error()
	} else {
		delete(c.unsaved, key(p, conv.ID))
	}
}

// change applies fn to a conversation and saves it.
func (c *chatService) change(p Project, id string, fn func(*conversation) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	conv, err := c.get(p, id)
	if err != nil {
		return err
	}
	if err := fn(conv); err != nil {
		return err
	}
	c.save(p, conv)
	return nil
}

// say adds a message to a conversation.
func (c *chatService) say(p Project, id, from, text string) {
	_ = c.change(p, id, func(conv *conversation) error {
		conv.Messages = append(conv.Messages, message{From: from, Text: text, At: time.Now().UTC()})
		return nil
	})
}

// chatSummary is a conversation in the panel's list.
type chatSummary struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Skill    string    `json:"skill"`
	Updated  time.Time `json:"updated"`
	Working  bool      `json:"working"`
	Proposal bool      `json:"proposal"` // a proposal waits
}

// chatView is one conversation as the window shows it.
type chatView struct {
	conversation
	Working bool `json:"working"`
	// Unsaved says why the conversation's last change is only in memory.
	Unsaved string `json:"unsaved,omitempty"`
}

// view returns a conversation as the window shows it, as JSON.
func (c *chatService) view(p Project, id string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	conv, err := c.get(p, id)
	if err != nil {
		return nil, err
	}
	l := c.live[key(p, id)]
	return json.Marshal(chatView{conversation: *conv, Working: l != nil && l.working, Unsaved: c.unsaved[key(p, id)]})
}

// openProposal returns a copy of the conversation's open proposal, or nil.
func (c *chatService) openProposal(p Project, id string) (*proposal, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	conv, err := c.get(p, id)
	if err != nil {
		return nil, err
	}
	open := conv.open()
	if open == nil {
		return nil, nil
	}
	copied := *open
	return &copied, nil
}

// list returns the project's conversations, newest first. A conversation
// whose file cannot be read is left out, not the list.
func (c *chatService) list(p Project) ([]chatSummary, error) {
	entries, err := os.ReadDir(filepath.Join(c.dir, p.ID))
	if errors.Is(err, fs.ErrNotExist) {
		entries = nil
	} else if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	seen := map[string]bool{}
	for _, e := range entries {
		if id, ok := strings.CutSuffix(e.Name(), ".json"); ok && chatID(id) {
			seen[id] = true
		}
	}
	// A conversation only in memory, whose save failed, is listed too.
	for k := range c.convs {
		if pid, id, _ := strings.Cut(k, "/"); pid == p.ID {
			seen[id] = true
		}
	}
	out := []chatSummary{}
	for id := range seen {
		conv, err := c.get(p, id)
		if err != nil {
			continue
		}
		l := c.live[key(p, id)]
		out = append(out, chatSummary{ID: conv.ID, Title: conv.Title, Skill: conv.Skill, Updated: conv.Updated, Working: l != nil && l.working, Proposal: conv.open() != nil})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Updated.After(out[j].Updated) })
	return out, nil
}

func chatID(id string) bool {
	if len(id) != 16 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func newChatID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// title is a conversation's name: the first line of its first message,
// shortened.
func title(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	if r := []rune(line); len(r) > 60 {
		line = strings.TrimSpace(string(r[:57])) + "…"
	}
	return line
}

// errBusy refuses a step while the conversation's turn, accept or reject
// runs.
var errBusy = errors.New("the chat is still at work; wait for it to finish")

// claim marks the conversation as at work, or refuses when it is.
func (c *chatService) claim(p Project, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errClosed
	}
	l := c.live[key(p, id)]
	if l == nil {
		l = &liveChat{}
		c.live[key(p, id)] = l
	}
	if l.working {
		return errBusy
	}
	l.working = true
	return nil
}

func (c *chatService) done(p Project, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if l := c.live[key(p, id)]; l != nil {
		l.working = false
	}
}

// create keeps a new conversation and starts its first turn.
func (c *chatService) create(p Project, conv *conversation, text string) error {
	c.mu.Lock()
	c.convs[key(p, conv.ID)] = conv
	c.mu.Unlock()
	return c.begin(p, conv.ID, text)
}

// begin records the owner's message and runs the turn in the background.
func (c *chatService) begin(p Project, id, text string) error {
	if err := c.claim(p, id); err != nil {
		return err
	}
	err := c.change(p, id, func(conv *conversation) error {
		conv.Messages = append(conv.Messages, message{From: "owner", Text: text, At: time.Now().UTC()})
		return nil
	})
	if err != nil {
		c.done(p, id)
		return err
	}
	c.turns.Add(1)
	go c.turn(p, id, text)
	return nil
}

// turn runs one turn: it starts the agent if the conversation has none,
// sends the message, keeps each answer as it comes and reads the proposal.
// A turn that does not answer leaves its agent: the next turn starts a fresh
// one, which takes over the conversation and its open proposal.
func (c *chatService) turn(p Project, id, text string) {
	defer c.turns.Done()
	defer c.done(p, id)
	ctx := c.ctx
	agent, err := c.agent(ctx, p, id)
	if err != nil {
		c.say(p, id, "app", "The chat could not start: "+err.Error())
		return
	}
	end, err := agent.Send(ctx, text, func(s string) { c.say(p, id, "agent", s) })
	switch {
	case ctx.Err() != nil:
		return
	case err != nil:
		c.say(p, id, "app", "The turn broke off: "+err.Error())
		c.drop(p, id)
		return
	case end.End != "answered":
		c.say(p, id, "app", strings.TrimSpace(fmt.Sprintf("The turn ended without an answer (%s). %s", end.End, end.Detail)))
		c.drop(p, id)
		return
	}
	model, outside, err := agent.Changes(ctx)
	if err != nil {
		if ctx.Err() == nil {
			c.say(p, id, "app", "The chat's changes could not be read: "+err.Error())
			c.drop(p, id)
		}
		return
	}
	_ = c.change(p, id, func(conv *conversation) error {
		propose(conv, model, outside)
		return nil
	})
}

// drop ends the conversation's agent; its next turn starts a fresh one.
func (c *chatService) drop(p Project, id string) {
	c.mu.Lock()
	l := c.live[key(p, id)]
	var agent ChatAgent
	if l != nil {
		agent, l.agent = l.agent, nil
	}
	c.mu.Unlock()
	if agent != nil {
		stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = agent.Close(stop)
	}
}

// agent returns the conversation's running agent, starting one on the
// working tree as it is now, with the open proposal on top, when there is
// none.
func (c *chatService) agent(ctx context.Context, p Project, id string) (ChatAgent, error) {
	c.mu.Lock()
	l := c.live[key(p, id)]
	conv, err := c.get(p, id)
	var skill string
	var earlier strings.Builder
	var pending []ChatFile
	if err == nil {
		skill = conv.Skill
		// Earlier turns reach a fresh agent as text; the owner's message of
		// this turn is not among them.
		for _, m := range conv.Messages[:max(len(conv.Messages)-1, 0)] {
			fmt.Fprintf(&earlier, "%s: %s\n\n", m.From, m.Text)
		}
		if open := conv.open(); open != nil {
			for _, f := range open.Files {
				pending = append(pending, ChatFile{Path: f.Path, Data: []byte(f.Text), Removed: f.Change == analysis.Removed})
			}
		}
	}
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if l != nil && l.agent != nil {
		return l.agent, nil
	}
	if c.chats == nil {
		return nil, errors.New("this Markitect has no chat agents")
	}
	ws, err := c.repos.Open(ctx, p.Path)
	if err != nil {
		return nil, err
	}
	files, err := ws.Files()
	if err != nil {
		return nil, err
	}
	start := map[string]string{}
	for _, f := range files {
		if strings.HasPrefix(f, project.Dir+"/") {
			data, err := ws.Read(f)
			if err != nil {
				return nil, err
			}
			start[f] = digest(data)
		}
	}
	agent, err := c.chats.Start(ctx, ChatStart{Name: "markitect-chat-" + id, Tree: listed{ws, files}, Files: files, Pending: pending, Skill: skill, Earlier: earlier.String()})
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	if c.closed {
		// The app quit while the agent started; it is not kept.
		c.mu.Unlock()
		stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = agent.Close(stop)
		return nil, errClosed
	}
	if l = c.live[key(p, id)]; l == nil {
		l = &liveChat{working: true}
		c.live[key(p, id)] = l
	}
	l.agent = agent
	c.mu.Unlock()
	err = c.change(p, id, func(conv *conversation) error {
		conv.Start = start
		return nil
	})
	return agent, err
}

// propose turns the conversation's changes into its open proposal. The
// changes are the copy's against its start, so a later turn's proposal
// replaces the one before. Only files a chat may propose are proposed; the
// rest is named.
func propose(conv *conversation, model []ChatFile, outside []string) {
	var files []ChatFile
	for _, f := range model {
		if proposable(f.Path) {
			files = append(files, f)
		} else {
			outside = append(outside, f.Path)
		}
	}
	sort.Strings(outside)
	if open := conv.open(); open != nil {
		open.State = proposalSuperseded
		if len(files) == 0 {
			open.State = proposalWithdrawn
		}
		open.dropText()
	}
	if len(files) == 0 {
		if len(outside) > 0 {
			conv.Messages = append(conv.Messages, message{From: "app", Text: "The chat changed files outside the model, which are not proposed: " + strings.Join(outside, ", "), At: time.Now().UTC()})
		}
		return
	}
	p := proposal{N: len(conv.Proposals) + 1, At: time.Now().UTC(), State: proposalOpen, Outside: outside}
	for _, f := range files {
		pf := proposedFile{Path: f.Path, Change: analysis.Modified, Text: string(f.Data)}
		switch {
		case f.Removed:
			pf.Change, pf.Text = analysis.Removed, ""
		case conv.Start[f.Path] == "":
			pf.Change = analysis.Added
		}
		p.Files = append(p.Files, pf)
	}
	sort.Slice(p.Files, func(i, j int) bool { return p.Files[i].Path < p.Files[j].Path })
	conv.Proposals = append(conv.Proposals, p)
}

// dropText forgets a closed proposal's texts; its files and changes stay.
func (p *proposal) dropText() {
	for i := range p.Files {
		p.Files[i].Text = ""
	}
}

// errConflict is an accept that found the working tree changed under the
// proposal.
var errConflict = errors.New("files the proposal changes were changed in the working tree since the chat started; nothing was written")

// accept writes the open proposal into the working tree, unless a file it
// touches changed there since the conversation's start; then it writes
// nothing, names the files and drops the agent, so that the next turn starts
// from the working tree as it is, with the proposal on top. A file that
// already holds what the proposal writes, as after an accept cut off midway,
// counts as written. The caller holds the conversation's claim.
func (c *chatService) accept(ctx context.Context, p Project, id string) error {
	c.mu.Lock()
	conv, err := c.get(p, id)
	var files []proposedFile
	start := map[string]string{}
	if err == nil {
		if open := conv.open(); open == nil {
			err = errors.New("the chat has no proposal waiting")
		} else {
			files = slices.Clone(open.Files)
			maps.Copy(start, conv.Start)
		}
	}
	c.mu.Unlock()
	if err != nil {
		return err
	}
	for _, f := range files {
		if !proposable(f.Path) {
			return fmt.Errorf("%s is no file a chat may propose", f.Path)
		}
	}
	ws, err := c.repos.Open(ctx, p.Path)
	if err != nil {
		return err
	}
	// A hand edit does not run between the check and the write.
	c.edits.Lock()
	defer c.edits.Unlock()
	var conflicts []string
	for _, f := range files {
		now := ""
		if data, err := ws.Read(f.Path); err == nil {
			now = digest(data)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		want := ""
		if f.Change != analysis.Removed {
			want = digest([]byte(f.Text))
		}
		if now != start[f.Path] && now != want {
			conflicts = append(conflicts, f.Path)
		}
	}
	if len(conflicts) > 0 {
		if err := c.change(p, id, func(conv *conversation) error {
			conv.open().Conflicts = conflicts
			conv.Messages = append(conv.Messages, message{From: "app", Text: "Nothing was written: " + strings.Join(conflicts, ", ") + " changed in the working tree since the chat started. The chat's next turn starts from the working tree as it is now, with its proposal on top.", At: time.Now().UTC()})
			return nil
		}); err != nil {
			return err
		}
		c.drop(p, id)
		return errConflict
	}
	for _, f := range files {
		if f.Change == analysis.Removed {
			err = ws.Remove(f.Path)
		} else {
			err = ws.Write(f.Path, []byte(f.Text))
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%s: %w; the proposal is written only in part, and accepting it again writes the rest", f.Path, err)
		}
	}
	err = c.change(p, id, func(conv *conversation) error {
		if conv.Start == nil {
			conv.Start = map[string]string{}
		}
		for _, f := range files {
			if f.Change == analysis.Removed {
				delete(conv.Start, f.Path)
			} else {
				conv.Start[f.Path] = digest([]byte(f.Text))
			}
		}
		open := conv.open()
		open.State, open.Conflicts = proposalAccepted, nil
		open.dropText()
		conv.Messages = append(conv.Messages, message{From: "app", Text: fmt.Sprintf("Accepted: %d %s written into the working tree. Commit them as usual.", len(files), plural(len(files), "file", "files")), At: time.Now().UTC()})
		return nil
	})
	c.settle(ctx, p, id, true)
	return err
}

// reject drops the open proposal. The caller holds the conversation's
// claim.
func (c *chatService) reject(ctx context.Context, p Project, id string) error {
	err := c.change(p, id, func(conv *conversation) error {
		open := conv.open()
		if open == nil {
			return errors.New("the chat has no proposal waiting")
		}
		open.State = proposalRejected
		open.dropText()
		conv.Messages = append(conv.Messages, message{From: "app", Text: "Rejected: the chat's copy is back where it started.", At: time.Now().UTC()})
		return nil
	})
	if err != nil {
		return err
	}
	c.settle(ctx, p, id, false)
	return nil
}

// settle moves the running agent's start; an agent that cannot is dropped,
// and the next turn starts a fresh one from the working tree.
func (c *chatService) settle(ctx context.Context, p Project, id string, keep bool) {
	c.mu.Lock()
	l := c.live[key(p, id)]
	var agent ChatAgent
	if l != nil {
		agent = l.agent
	}
	c.mu.Unlock()
	if agent != nil && agent.Settle(ctx, keep) != nil {
		c.drop(p, id)
	}
}

func (c *chatService) remove(p Project, id string) error {
	c.mu.Lock()
	l := c.live[key(p, id)]
	if l != nil && l.working {
		c.mu.Unlock()
		return errBusy
	}
	delete(c.live, key(p, id))
	delete(c.convs, key(p, id))
	delete(c.unsaved, key(p, id))
	err := os.Remove(c.file(p, id))
	c.mu.Unlock()
	if l != nil && l.agent != nil {
		stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = l.agent.Close(stop)
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// overlay is the working tree with a chat's open proposal on top, for the
// Model views.
type overlay struct {
	Workspace
	files []string
	text  map[string][]byte // proposed text; nil for a removed file
}

func newOverlay(ws Workspace, files []string, p *proposal) overlay {
	o := overlay{Workspace: ws, text: map[string][]byte{}}
	have := map[string]bool{}
	for _, f := range files {
		have[f] = true
	}
	for _, f := range p.Files {
		if f.Change == analysis.Removed {
			o.text[f.Path] = nil
			delete(have, f.Path)
		} else {
			o.text[f.Path] = []byte(f.Text)
			have[f.Path] = true
		}
	}
	for f := range have {
		o.files = append(o.files, f)
	}
	sort.Strings(o.files)
	return o
}

func (o overlay) Files() ([]string, error) { return append([]string(nil), o.files...), nil }

func (o overlay) Read(file string) ([]byte, error) {
	if data, ok := o.text[file]; ok {
		if data == nil {
			return nil, fs.ErrNotExist
		}
		return bytes.Clone(data), nil
	}
	return o.Workspace.Read(file)
}

// The routes of the chat.

func (s *server) chatRoutes() {
	s.mux.HandleFunc("GET /api/projects/{id}/chats", s.listChats)
	s.mux.HandleFunc("POST /api/projects/{id}/chats", s.newChat)
	s.mux.HandleFunc("GET /api/projects/{id}/chats/{chat}", s.getChat)
	s.mux.HandleFunc("DELETE /api/projects/{id}/chats/{chat}", s.deleteChat)
	s.mux.HandleFunc("POST /api/projects/{id}/chats/{chat}/messages", s.sendChat)
	s.mux.HandleFunc("POST /api/projects/{id}/chats/{chat}/accept", s.acceptChat)
	s.mux.HandleFunc("POST /api/projects/{id}/chats/{chat}/reject", s.rejectChat)
}

func (s *server) listChats(w http.ResponseWriter, r *http.Request) {
	p, ok := s.find(w, r)
	if !ok {
		return
	}
	list, err := s.chat.list(p)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	reply(w, map[string]any{"chats": list, "available": s.chat.chats != nil})
}

// findChat names the project and the conversation the route names.
func (s *server) findChat(w http.ResponseWriter, r *http.Request) (Project, string, bool) {
	p, ok := s.find(w, r)
	if !ok {
		return p, "", false
	}
	id := r.PathValue("chat")
	if !chatID(id) {
		fail(w, http.StatusNotFound, errors.New("no such chat"))
		return p, "", false
	}
	return p, id, true
}

// sendView answers with the conversation as it is now.
func (s *server) sendView(w http.ResponseWriter, p Project, id string, code int) {
	body, err := s.chat.view(p, id)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		fail(w, http.StatusNotFound, errors.New("no such chat"))
	case err != nil:
		fail(w, http.StatusInternalServerError, err)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write(body)
	}
}

func (s *server) getChat(w http.ResponseWriter, r *http.Request) {
	if p, id, ok := s.findChat(w, r); ok {
		s.sendView(w, p, id, http.StatusOK)
	}
}

type chatRequest struct {
	Skill string `json:"skill"` // new chats: modeling (the default) or extract
	Text  string `json:"text"`
}

func readChatRequest(w http.ResponseWriter, r *http.Request) (chatRequest, bool) {
	var req chatRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, err)
		return req, false
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		fail(w, http.StatusBadRequest, errors.New("write a message"))
		return req, false
	}
	return req, true
}

func (s *server) newChat(w http.ResponseWriter, r *http.Request) {
	p, ok := s.find(w, r)
	if !ok {
		return
	}
	req, ok := readChatRequest(w, r)
	if !ok {
		return
	}
	if req.Skill == "" {
		req.Skill = skillModeling
	}
	if req.Skill != skillModeling && req.Skill != skillExtract {
		fail(w, http.StatusBadRequest, fmt.Errorf("unknown skill %q: modeling or extract", req.Skill))
		return
	}
	if s.chat.chats == nil {
		fail(w, http.StatusServiceUnavailable, errors.New("this Markitect has no chat agents"))
		return
	}
	id, err := newChatID()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	conv := &conversation{ID: id, Title: title(req.Text), Skill: req.Skill, Created: time.Now().UTC(), Messages: []message{}, Proposals: []proposal{}}
	if err := s.chat.create(p, conv, req.Text); err != nil {
		chatFail(w, err)
		return
	}
	s.sendView(w, p, id, http.StatusOK)
}

func (s *server) sendChat(w http.ResponseWriter, r *http.Request) {
	p, id, ok := s.findChat(w, r)
	if !ok {
		return
	}
	req, ok := readChatRequest(w, r)
	if !ok {
		return
	}
	if req.Skill != "" {
		fail(w, http.StatusBadRequest, errors.New("a conversation keeps its skill"))
		return
	}
	if err := s.chat.begin(p, id, req.Text); err != nil {
		chatFail(w, err)
		return
	}
	s.sendView(w, p, id, http.StatusOK)
}

func chatFail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errClosed):
		fail(w, http.StatusServiceUnavailable, err)
	case errors.Is(err, errBusy):
		fail(w, http.StatusConflict, err)
	case errors.Is(err, fs.ErrNotExist):
		fail(w, http.StatusNotFound, errors.New("no such chat"))
	default:
		fail(w, http.StatusInternalServerError, err)
	}
}

// decide accepts or rejects the open proposal; neither runs during a turn,
// which may still change it. An accept that found the working tree changed
// answers 409 with the conversation, whose proposal names the files.
func (s *server) decide(w http.ResponseWriter, r *http.Request, accept bool) {
	p, id, ok := s.findChat(w, r)
	if !ok {
		return
	}
	if err := s.chat.claim(p, id); err != nil {
		chatFail(w, err)
		return
	}
	defer s.chat.done(p, id)
	var err error
	if accept {
		err = s.chat.accept(r.Context(), p, id)
	} else {
		err = s.chat.reject(r.Context(), p, id)
	}
	switch {
	case errors.Is(err, errConflict):
		s.sendView(w, p, id, http.StatusConflict)
	case errors.Is(err, fs.ErrNotExist):
		fail(w, http.StatusNotFound, errors.New("no such chat"))
	case err != nil:
		fail(w, http.StatusConflict, err)
	default:
		s.sendView(w, p, id, http.StatusOK)
	}
}

func (s *server) acceptChat(w http.ResponseWriter, r *http.Request) { s.decide(w, r, true) }
func (s *server) rejectChat(w http.ResponseWriter, r *http.Request) { s.decide(w, r, false) }

func (s *server) deleteChat(w http.ResponseWriter, r *http.Request) {
	p, id, ok := s.findChat(w, r)
	if !ok {
		return
	}
	if err := s.chat.remove(p, id); err != nil {
		chatFail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
