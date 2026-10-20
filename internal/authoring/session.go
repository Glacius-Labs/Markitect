package authoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// SessionPorts run the turns of a model session. Bootstrap composes them from
// the run configuration and the Runtime profile: the Agents run each turn in a
// Linux container, the workspace volume holds the session's copy of the
// committed source, and Evidence is the state directory under whose sessions/
// directory the session's record and transcripts are kept.
type SessionPorts struct {
	Agents                run.Agents
	Workspaces            run.Workspaces
	RemoveVolume          func(context.Context, string) error
	RequireVolumeSubpaths func(context.Context) error
	MaxEntries            int
	MaxBytes              int64
	Evidence              string
}

// SessionStart is a captured start of a model session.
type SessionStart struct {
	Basis apply.SessionBasis
	Ports SessionPorts
}

// SessionOpener captures the committed source with its settings and Runtime
// profile and composes the ports of a new session.
type SessionOpener func(context.Context) (SessionStart, error)

// Session states. A working session runs one turn; a waiting one waits for
// the owner's next prompt; a closed one has released its workspace.
const (
	Working = "working"
	Waiting = "waiting"
	Closed  = "closed"
)

// Session is what the owner sees of a model session: its prompts and the
// Agent's answers, the tools it used, its proposals, its usage and limits.
type Session struct {
	ID          string        `json:"id"`
	Created     time.Time     `json:"created"`
	Head        string        `json:"head"`
	Profile     string        `json:"profile"`
	TurnTimeout string        `json:"turnTimeout"`
	MaxTurns    int           `json:"maxTurns"`
	State       string        `json:"state"`
	Turns       []SessionTurn `json:"turns"`
	Usage       run.Usage     `json:"usage"`
	Evidence    string        `json:"evidence"`
	Error       string        `json:"error,omitempty"`
}

// SessionTurn is one owner prompt and the Agent turn that answered it. Final
// is the Agent's own text; nothing in it is a fact the cockpit established.
type SessionTurn struct {
	ID        string    `json:"id"`
	Prompt    string    `json:"prompt"`
	Started   time.Time `json:"started"`
	Ended     time.Time `json:"ended,omitzero"`
	Final     string    `json:"final,omitempty"`
	Usage     run.Usage `json:"usage"`
	Stopped   bool      `json:"stopped,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	Tools     []ToolUse `json:"tools"`
	Proposals []string  `json:"proposals"`
}

// ToolUse is one tool call of a turn and what the cockpit answered.
type ToolUse struct {
	Name    string    `json:"name"`
	At      time.Time `json:"at"`
	Answer  string    `json:"answer"`
	Refused bool      `json:"refused,omitempty"`
}

// ErrNoSessions reports a cockpit without a run configuration and Runtime
// profile; ErrSessionOpen one that already has an open session; ErrBusy a
// session that still runs a turn; ErrLimit one that used its Agent turns.
var (
	ErrNoSessions    = errors.New("model sessions need the cockpit to be started with --run-config and --runtime-profile")
	ErrSessionOpen   = errors.New("a model session is open; close it before starting another")
	ErrBusy          = errors.New("the session is still working on the last prompt")
	ErrLimit         = errors.New("the session used all its Agent turns; close it and start a new one")
	ErrNoSession     = errors.New("no such session")
	ErrSessionClosed = errors.New("the session is closed")
	ErrPrompt        = fmt.Errorf("a prompt is non-blank UTF-8 text of at most %d bytes", maxPromptBytes)
)

const maxPromptBytes = 16 << 10

// The writable part of a session's workspace: the model and the registry.
var sessionMounts = []run.PathMount{{Path: ".markitect/model", Writable: true}, {Path: ".markitect/registry", Writable: true}}

func sessionPath(path string) bool {
	return strings.HasPrefix(path, ".markitect/model/") || strings.HasPrefix(path, ".markitect/registry/")
}

var sessionTools = []run.ToolSpec{
	{Name: "check", Description: "Run the project's structural check (Model check, registry, File check, execution settings and context selection) on your workspace as it is now. It lists the changed files and every finding and changes nothing.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)},
	{Name: "submit", Description: "Submit the current .markitect/model and .markitect/registry of your workspace to the owner as a proposal, with a summary of what you changed and of anything in the model or the request you found contradictory or unclear. The owner reviews the difference and the check and decides; you may submit again after further changes.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"summary":{"type":"string","minLength":1,"maxLength":4000}},"required":["summary"],"additionalProperties":false}`)},
}

// session is a Session with what runs it.
type session struct {
	Session
	start        SessionStart
	volume, home string
	conversation string
	cancel       context.CancelFunc
	done         chan struct{}
	record       *sessionRecord
}

// SessionsAvailable says whether the cockpit can open model sessions.
func (s *Service) SessionsAvailable() bool { return s.openSession != nil }

// WithSessions enables model sessions through the given opener.
func (s *Service) WithSessions(open SessionOpener) *Service {
	s.openSession = open
	return s
}

// Sessions lists the sessions of this cockpit, newest first.
func (s *Service) Sessions() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Session, 0, len(s.sessions))
	for i := len(s.sessionOrder) - 1; i >= 0; i-- {
		out = append(out, s.sessions[s.sessionOrder[i]].view())
	}
	return out
}

// Session returns one session.
func (s *Service) Session(id string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.sessions[id]
	if !ok {
		return Session{}, false
	}
	return ss.view(), true
}

// StartSession captures the committed source, materializes the session's
// workspace and starts its first turn with the owner's prompt. The turn runs
// in the background; the session reports its state. At most one session is
// open at a time.
func (s *Service) StartSession(ctx context.Context, prompt string) (Session, error) {
	if s.openSession == nil {
		return Session{}, ErrNoSessions
	}
	if err := validPrompt(prompt); err != nil {
		return Session{}, err
	}
	s.mu.Lock()
	for _, ss := range s.sessions {
		if ss.State != Closed {
			s.mu.Unlock()
			return Session{}, ErrSessionOpen
		}
	}
	s.sessionNext++
	id := fmt.Sprintf("model-%s-%d", s.now().UTC().Format("20060102t150405"), s.sessionNext)
	// The reservation keeps a second start out while this one prepares.
	ss := &session{Session: Session{ID: id, State: Working, Turns: []SessionTurn{}}, done: make(chan struct{})}
	close(ss.done)
	s.sessions[id] = ss
	s.sessionOrder = append(s.sessionOrder, id)
	s.mu.Unlock()

	fail := func(err error) (Session, error) {
		s.mu.Lock()
		ss.State, ss.Error = Closed, err.Error()
		view := ss.view()
		s.mu.Unlock()
		return view, err
	}
	start, err := s.openSession(ctx)
	if err != nil {
		return fail(err)
	}
	if start.Basis.Turns < 1 || start.Basis.TurnTimeout <= 0 || start.Basis.Profile == "" {
		return fail(errors.New("the execution budget gives the session no Agent turn or no turn timeout"))
	}
	inputs := start.Basis.Attempt.Model().Inputs()
	record, err := newSessionRecord(filepath.Join(start.Ports.Evidence, "sessions", id))
	if err != nil {
		return fail(err)
	}
	if err := start.Ports.RequireVolumeSubpaths(ctx); err != nil {
		return fail(errors.Join(err, record.close()))
	}
	directories := make([]string, len(sessionMounts))
	for i, m := range sessionMounts {
		directories[i] = m.Path
	}
	observed, err := start.Ports.Workspaces.Materialize(ctx, run.WorkspaceSpec{ID: run.WorkspaceID(id + "-work"), Tree: inputs.Raw.Census, Directories: directories})
	if err != nil {
		return fail(errors.Join(err, record.close()))
	}
	s.mu.Lock()
	if ss.State == Closed {
		// The owner closed the session while it was prepared.
		s.mu.Unlock()
		_, closeErr := start.Ports.Workspaces.Close(ctx, run.WorkspaceID(id+"-work"))
		return fail(errors.Join(errors.New("the session was closed while it was prepared"), closeErr, record.close()))
	}
	ss.start, ss.volume, ss.home, ss.record = start, observed.Volume, id+"-home", record
	ss.Created, ss.Head, ss.Profile = s.now().UTC(), inputs.Head, start.Basis.Profile
	ss.TurnTimeout, ss.MaxTurns, ss.Evidence = shortDuration(start.Basis.TurnTimeout), start.Basis.Turns, record.dir
	record.write("session.start", map[string]any{"head": ss.Head, "profile": ss.Profile, "turnTimeout": ss.TurnTimeout, "maxTurns": ss.MaxTurns, "workspace": ss.volume})
	view := s.beginTurnLocked(ss, prompt)
	s.mu.Unlock()
	return view, nil
}

// PromptSession continues a waiting session with the owner's next prompt in
// the same provider conversation and workspace.
func (s *Service) PromptSession(id, prompt string) (Session, error) {
	if err := validPrompt(prompt); err != nil {
		return Session{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.sessions[id]
	switch {
	case !ok:
		return Session{}, fmt.Errorf("%w: %s", ErrNoSession, id)
	case ss.State == Closed:
		return ss.view(), fmt.Errorf("%w: %s", ErrSessionClosed, id)
	case ss.State == Working:
		return ss.view(), ErrBusy
	case len(ss.Turns) >= ss.MaxTurns:
		return ss.view(), ErrLimit
	}
	return s.beginTurnLocked(ss, prompt), nil
}

// CloseSession stops a running turn, removes the session's workspace and
// Agent home and closes its record. The proposals it made stay.
func (s *Service) CloseSession(ctx context.Context, id string) (Session, error) {
	s.mu.Lock()
	ss, ok := s.sessions[id]
	if !ok {
		s.mu.Unlock()
		return Session{}, fmt.Errorf("%w: %s", ErrNoSession, id)
	}
	if ss.State == Closed {
		view := ss.view()
		s.mu.Unlock()
		return view, nil
	}
	if ss.cancel != nil {
		ss.cancel()
	}
	done := ss.done
	s.mu.Unlock()
	<-done

	s.mu.Lock()
	defer s.mu.Unlock()
	if ss.State == Closed {
		return ss.view(), nil
	}
	var errs []error
	if ss.volume != "" {
		if observed, err := ss.start.Ports.Workspaces.Close(ctx, run.WorkspaceID(id+"-work")); err != nil || !observed.Closed {
			errs = append(errs, errors.Join(err, fmt.Errorf("the workspace %s was not observed closed: %s", id, observed.Reason)))
		}
	}
	if ss.home != "" && ss.start.Ports.RemoveVolume != nil {
		if err := ss.start.Ports.RemoveVolume(ctx, ss.home); err != nil {
			errs = append(errs, err)
		}
	}
	ss.State = Closed
	err := errors.Join(errs...)
	if err != nil {
		ss.Error = err.Error()
	}
	if ss.record != nil {
		ss.record.write("session.close", map[string]any{"error": ss.Error})
		err = errors.Join(err, ss.record.close())
	}
	return ss.view(), err
}

// CloseSessions closes every open session, as a stopping cockpit does.
func (s *Service) CloseSessions(ctx context.Context) error {
	s.mu.Lock()
	var open []string
	for id, ss := range s.sessions {
		if ss.State != Closed {
			open = append(open, id)
		}
	}
	s.mu.Unlock()
	var errs []error
	for _, id := range open {
		_, err := s.CloseSession(ctx, id)
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func validPrompt(prompt string) error {
	if strings.TrimSpace(prompt) == "" || len(prompt) > maxPromptBytes || !utf8.ValidString(prompt) || strings.ContainsRune(prompt, 0) {
		return ErrPrompt
	}
	return nil
}

// beginTurnLocked records the owner's prompt and runs the turn in the
// background. The caller holds s.mu.
func (s *Service) beginTurnLocked(ss *session, prompt string) Session {
	n := len(ss.Turns) + 1
	turn := SessionTurn{ID: fmt.Sprintf("%s-t%d", ss.ID, n), Prompt: prompt, Started: s.now().UTC(), Tools: []ToolUse{}, Proposals: []string{}}
	ss.Turns = append(ss.Turns, turn)
	ss.State = Working
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	ss.cancel, ss.done = cancel, done
	ss.record.write("prompt", map[string]any{"turn": turn.ID, "text": prompt})
	request := run.Turn{
		ID: turn.ID, Manager: ".", Kind: run.ExecuteTurn, Profile: ss.Profile,
		Workspace: ss.volume, ReadOnly: true, Mounts: sessionMounts, Home: ss.home, Resume: ss.conversation,
		Prompt: sessionPrompt(prompt, n == 1), Tools: sessionTools, Timeout: ss.start.Basis.TurnTimeout,
		Transcript: filepath.Join(ss.record.dir, "transcripts", turn.ID+".jsonl"),
	}
	go s.runTurn(ctx, cancel, done, ss, n-1, request)
	return ss.view()
}

// runTurn runs one Agent turn and records what the Agents port observed.
func (s *Service) runTurn(ctx context.Context, cancel context.CancelFunc, done chan struct{}, ss *session, index int, request run.Turn) {
	defer close(done)
	defer cancel()
	result, err := ss.start.Ports.Agents.Run(ctx, request, func(ctx context.Context, name string, arguments json.RawMessage) run.ToolAnswer {
		answer := s.sessionTool(ctx, ss, index, name, arguments)
		s.mu.Lock()
		ss.Turns[index].Tools = append(ss.Turns[index].Tools, ToolUse{Name: name, At: s.now().UTC(), Answer: answer.Text, Refused: answer.Refused})
		ss.record.write("tool", map[string]any{"turn": request.ID, "name": name, "refused": answer.Refused, "answer": answer.Text})
		s.mu.Unlock()
		return answer
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	t := &ss.Turns[index]
	t.Ended, t.Final, t.Usage, t.Stopped, t.Reason = s.now().UTC(), result.Final, result.Usage, result.Stopped, result.Reason
	if err != nil {
		t.Reason = strings.TrimSpace(t.Reason + " " + err.Error())
	}
	if result.Conversation != "" {
		ss.conversation = result.Conversation
	}
	ss.Usage = ss.Usage.Add(result.Usage)
	ss.State = Waiting
	ss.record.write("turn.end", map[string]any{"turn": request.ID, "usage": result.Usage, "conversation": result.Conversation, "stopped": result.Stopped, "reason": t.Reason})
}

// sessionTool answers the session's check and submit tools from the
// workspace as it is now.
func (s *Service) sessionTool(ctx context.Context, ss *session, index int, name string, arguments json.RawMessage) run.ToolAnswer {
	var summary string
	switch name {
	case "check":
	case "submit":
		var args struct {
			Summary string `json:"summary"`
		}
		dec := json.NewDecoder(strings.NewReader(string(arguments)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&args); err != nil || strings.TrimSpace(args.Summary) == "" || len(args.Summary) > 4000 {
			return run.ToolAnswer{Text: "submit needs a summary of at most 4000 characters", Refused: true}
		}
		summary = args.Summary
	default:
		return run.ToolAnswer{Text: "unknown tool " + name, Refused: true}
	}
	ports := ss.start.Ports
	harvest, err := ports.Workspaces.Peek(ctx, run.HarvestRequest{ID: run.WorkspaceID(ss.ID + "-work"), MaxEntries: ports.MaxEntries, MaxBytes: ports.MaxBytes})
	if err != nil {
		return run.ToolAnswer{Text: "the workspace could not be read: " + err.Error(), Refused: true}
	}
	if !harvest.Complete() {
		return run.ToolAnswer{Text: "the workspace could not be read completely: " + harvest.Reason, Refused: true}
	}
	inputs := ss.start.Basis.Attempt.Model().Inputs()
	if name == "check" {
		return run.ToolAnswer{Text: checkText(snapshot.Compare(inputs.Raw.Census, harvest.Tree), apply.CheckProposal(ss.start.Basis.Attempt, harvest.Tree))}
	}
	st := stand{attempt: ss.start.Basis.Attempt, tree: inputs.Raw.Census, head: inputs.Head}
	p, err := s.propose(st, Proposal{Kind: "model", Title: "Model change from session " + ss.ID, Session: ss.ID, Summary: summary}, harvest.Tree, sessionPath)
	if err != nil {
		return run.ToolAnswer{Text: err.Error(), Refused: true}
	}
	s.mu.Lock()
	ss.Turns[index].Proposals = append(ss.Turns[index].Proposals, p.ID)
	ss.record.write("proposal", map[string]any{"turn": ss.Turns[index].ID, "proposal": p.ID, "check": p.Check.Status})
	s.mu.Unlock()
	return run.ToolAnswer{Text: fmt.Sprintf("Proposal %s was made; its structural check is %s with %d finding(s). The owner reviews the difference and decides; nothing is written before that.", p.ID, p.Check.Status, len(p.Check.Findings))}
}

// checkText is the check tool's answer: the changed files and the findings.
func checkText(changes []snapshot.Change, c apply.ProposalCheck) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Structural check: %s.\n", c.Status)
	if len(changes) == 0 {
		b.WriteString("Your workspace does not differ from the committed source.\n")
	}
	for _, ch := range changes {
		fmt.Fprintf(&b, "changed: %s\n", ch.Path())
	}
	for i, f := range c.Findings {
		if i == 50 {
			fmt.Fprintf(&b, "... %d more findings\n", len(c.Findings)-50)
			break
		}
		fmt.Fprintf(&b, "finding (%s) %s %s: %s\n", f.Stage, f.Code, f.Path, f.Message)
	}
	return b.String()
}

// sessionPrompt states the task and how to answer; a later prompt continues
// the same conversation.
func sessionPrompt(prompt string, first bool) string {
	if !first {
		return "The owner writes:\n\n" + prompt + "\n\nThe same rules apply: change only .markitect/model/ and .markitect/registry/, use check, and submit a proposal when a change is ready."
	}
	return `You work on the canonical model of the project in /work. The owner asks:

` + prompt + `

Change only files under .markitect/model/ and .markitect/registry/; everything else in /work is read-only. The model's own Definitions and the project's documentation describe its language and conventions. The owner is the author of intent: change the model as the owner asks and do not decide open questions on the owner's behalf.

Use the check tool to see what the project's structural check reports for your workspace. When a change is ready, call submit with a summary for the owner that names what you changed and anything you found contradictory or unclear. If no change is needed or the request is unclear, do not submit; answer the owner with what you found or need to know.`
}

func (ss *session) view() Session {
	v := ss.Session
	v.Turns = make([]SessionTurn, len(ss.Turns))
	for i, t := range ss.Turns {
		t.Tools = append([]ToolUse{}, t.Tools...)
		t.Proposals = append([]string{}, t.Proposals...)
		v.Turns[i] = t
	}
	return v
}

// sessionRecord appends the session's evidence as JSON lines.
type sessionRecord struct {
	dir string
	mu  sync.Mutex
	f   *os.File
	err error
}

func newSessionRecord(dir string) (*sessionRecord, error) {
	if err := os.MkdirAll(filepath.Join(dir, "transcripts"), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &sessionRecord{dir: dir, f: f}, nil
}

func (r *sessionRecord) write(kind string, data map[string]any) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil || r.err != nil {
		return
	}
	line, err := json.Marshal(map[string]any{"time": time.Now().UTC(), "kind": kind, "data": data})
	if err == nil {
		_, err = r.f.Write(append(line, '\n'))
	}
	r.err = err
}

func (r *sessionRecord) close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return r.err
	}
	err := errors.Join(r.err, r.f.Close())
	r.f = nil
	return err
}

// shortDuration writes a duration as the execution settings do, such as 10m
// or 1h30m.
func shortDuration(d time.Duration) string {
	text := d.String()
	if strings.HasSuffix(text, "m0s") {
		text = strings.TrimSuffix(text, "0s")
	}
	if strings.HasSuffix(text, "h0m") {
		text = strings.TrimSuffix(text, "0m")
	}
	return text
}
