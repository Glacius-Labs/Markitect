package authoring

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// Session states. A preparing session captures the source and materializes its
// workspace; a working one runs one turn; a waiting one waits for the owner's
// next prompt; a halted one takes no prompt, because the end of its last
// container was not observed or its record could not be written; a closing
// one releases its workspace; a closed one has released it.
const (
	Preparing = "preparing"
	Working   = "working"
	Waiting   = "waiting"
	Halted    = "halted"
	Closing   = "closing"
	Closed    = "closed"
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

// SessionTurn is one owner prompt and the Agent turn that answered it.
// Answers and Final are the Agent's own text; nothing in them is a fact the
// cockpit established. Stopped says that the end of the turn's container was
// observed.
type SessionTurn struct {
	ID        string    `json:"id"`
	Prompt    string    `json:"prompt"`
	Started   time.Time `json:"started"`
	Ended     time.Time `json:"ended,omitzero"`
	Answers   []string  `json:"answers"`
	Final     string    `json:"final,omitempty"`
	Usage     run.Usage `json:"usage"`
	Stopped   bool      `json:"stopped"`
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

// Refusals of the session operations.
var (
	ErrNoSessions    = errors.New("model sessions need the cockpit to be started with --run-config and --runtime-profile")
	ErrSessionOpen   = errors.New("a model session is open; close it before starting another")
	ErrBusy          = errors.New("the session is still working on the last prompt")
	ErrLimit         = errors.New("the session used all its Agent turns; close it and start a new one")
	ErrHalted        = errors.New("the session is halted and takes no prompt; close it")
	ErrNoSession     = errors.New("no such session")
	ErrSessionClosed = errors.New("the session is closed")
	ErrPrompt        = fmt.Errorf("a prompt is non-blank UTF-8 text of at most %d bytes", maxPromptBytes)
)

const (
	maxPromptBytes = 16 << 10
	maxToolText    = 4000
)

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
	{Name: "answer", Description: "Answer the owner without a proposal: what you found, why no change is needed, or what you need to know.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","minLength":1,"maxLength":4000}},"required":["text"],"additionalProperties":false}`)},
}

// session is a Session with what runs it. s.mu guards it.
type session struct {
	Session
	start        SessionStart
	volume, home string
	conversation string
	containers   bool // a turn observed a started container, so the home volume exists
	cancel       context.CancelFunc
	done         chan struct{} // closed when the current turn ended
	prepared     chan struct{} // closed when StartSession finished
	closed       chan struct{} // closed when the session is closed
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
// open at a time. The caller should not cancel ctx when the requester goes
// away: an interrupted preparation leaves nothing behind, but it ends the
// session.
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
	id := fmt.Sprintf("model-%s-%s", s.now().UTC().Format("20060102t150405"), randomSuffix())
	// The reservation keeps a second start out while this one prepares.
	ss := &session{Session: Session{ID: id, State: Preparing, Turns: []SessionTurn{}}, prepared: make(chan struct{}), closed: make(chan struct{})}
	s.sessions[id] = ss
	s.sessionOrder = append(s.sessionOrder, id)
	s.mu.Unlock()
	defer close(ss.prepared)

	var record *sessionRecord
	materialized := false
	// fail ends the session: it releases a materialized workspace with a
	// context of its own, records the failure and closes the session.
	fail := func(ports SessionPorts, err error) (Session, error) {
		if materialized {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
			if observed, closeErr := ports.Workspaces.Close(cleanup, run.WorkspaceID(id+"-work")); closeErr != nil || !observed.Closed {
				err = errors.Join(err, closeErr, fmt.Errorf("the workspace %s-work was not observed closed: %s", id, observed.Reason))
			}
			cancel()
		}
		record.write("session.fail", map[string]any{"error": err.Error()})
		err = errors.Join(err, record.close())
		s.mu.Lock()
		ss.State, ss.Error = Closed, err.Error()
		view := ss.view()
		s.mu.Unlock()
		close(ss.closed)
		return view, err
	}
	start, err := s.openSession(ctx)
	if err != nil {
		return fail(start.Ports, err)
	}
	if start.Basis.Turns < 1 || start.Basis.TurnTimeout <= 0 || start.Basis.Profile == "" {
		return fail(start.Ports, errors.New("the execution budget gives the session no Agent turn or no turn timeout"))
	}
	inputs := start.Basis.Attempt.Model().Inputs()
	if record, err = newSessionRecord(filepath.Join(start.Ports.Evidence, "sessions", id)); err != nil {
		return fail(start.Ports, err)
	}
	if err := start.Ports.RequireVolumeSubpaths(ctx); err != nil {
		return fail(start.Ports, err)
	}
	directories := make([]string, len(sessionMounts))
	for i, m := range sessionMounts {
		directories[i] = m.Path
	}
	// A failed or uncertain materialization may still have created the
	// volume; the failure closes it.
	materialized = true
	observed, err := start.Ports.Workspaces.Materialize(ctx, run.WorkspaceSpec{ID: run.WorkspaceID(id + "-work"), Tree: inputs.Raw.Census, Directories: directories})
	if err != nil {
		return fail(start.Ports, err)
	}
	if observed.Uncertain || observed.Volume == "" {
		return fail(start.Ports, fmt.Errorf("the workspace was not observed as materialized: %s", observed.Reason))
	}
	s.mu.Lock()
	if ss.State != Preparing {
		// The owner closed the session while it was prepared.
		s.mu.Unlock()
		return fail(start.Ports, errors.New("the session was closed while it was prepared"))
	}
	ss.start, ss.volume, ss.home, ss.record = start, observed.Volume, id+"-home", record
	ss.Created, ss.Head, ss.Profile = s.now().UTC(), inputs.Head, start.Basis.Profile
	ss.TurnTimeout, ss.MaxTurns, ss.Evidence = shortDuration(start.Basis.TurnTimeout), start.Basis.Turns, record.dir
	record.write("session.start", map[string]any{"head": ss.Head, "profile": ss.Profile, "turnTimeout": ss.TurnTimeout, "maxTurns": ss.MaxTurns, "workspace": ss.volume, "home": ss.home})
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
	case ss.State == Closing || ss.State == Closed:
		return ss.view(), fmt.Errorf("%w: %s", ErrSessionClosed, id)
	case ss.State == Halted:
		return ss.view(), ErrHalted
	case ss.State != Waiting:
		return ss.view(), ErrBusy
	case len(ss.Turns) >= ss.MaxTurns:
		return ss.view(), ErrLimit
	}
	return s.beginTurnLocked(ss, prompt), nil
}

// CloseSession stops a working turn, closes the session's workspace, removes
// its Agent home and closes its record. The proposals it made stay. A session
// still preparing is closed when its preparation ends.
func (s *Service) CloseSession(ctx context.Context, id string) (Session, error) {
	s.mu.Lock()
	ss, ok := s.sessions[id]
	if !ok {
		s.mu.Unlock()
		return Session{}, fmt.Errorf("%w: %s", ErrNoSession, id)
	}
	switch ss.State {
	case Closed:
		view := ss.view()
		s.mu.Unlock()
		return view, nil
	case Closing:
		// Another close is releasing the session; wait for it.
		closed := ss.closed
		s.mu.Unlock()
		<-closed
		return s.closedView(ss)
	case Preparing:
		ss.State = Closing
		prepared := ss.prepared
		s.mu.Unlock()
		<-prepared
		return s.closedView(ss)
	}
	ss.State = Closing
	if ss.cancel != nil {
		ss.cancel()
	}
	done := ss.done
	s.mu.Unlock()
	if done != nil {
		<-done
	}

	// Released without the lock: closing a volume takes seconds.
	var errs []error
	ports := ss.start.Ports
	if observed, err := ports.Workspaces.Close(ctx, run.WorkspaceID(id+"-work")); err != nil || !observed.Closed {
		errs = append(errs, errors.Join(err, fmt.Errorf("the workspace %s-work was not observed closed: %s", id, observed.Reason)))
	}
	s.mu.Lock()
	containers := ss.containers
	s.mu.Unlock()
	var homeErr error
	if ports.RemoveVolume != nil {
		homeErr = ports.RemoveVolume(ctx, ss.home)
	}
	// A home that no observed container used may never have been created.
	if homeErr != nil && containers {
		errs = append(errs, homeErr)
	}

	s.mu.Lock()
	err := errors.Join(errs...)
	if err != nil {
		ss.Error = strings.TrimSpace(ss.Error + " " + err.Error())
	}
	ss.record.write("session.close", map[string]any{"error": errText(err), "home": errText(homeErr)})
	err = errors.Join(err, ss.record.close())
	ss.State = Closed
	view := ss.view()
	s.mu.Unlock()
	close(ss.closed)
	return view, err
}

func (s *Service) closedView(ss *session) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ss.Error != "" {
		return ss.view(), errors.New(ss.Error)
	}
	return ss.view(), nil
}

// CloseSessions closes every session that is not closed, as a stopping cockpit
// does; it waits for a session that is still preparing.
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
	turn := SessionTurn{ID: fmt.Sprintf("%s-t%d", ss.ID, n), Prompt: prompt, Started: s.now().UTC(), Answers: []string{}, Tools: []ToolUse{}, Proposals: []string{}}
	ss.Turns = append(ss.Turns, turn)
	ss.State = Working
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	ss.cancel, ss.done = cancel, done
	request := run.Turn{
		ID: turn.ID, Manager: ".", Kind: run.ExecuteTurn, Profile: ss.Profile,
		Workspace: ss.volume, ReadOnly: true, Mounts: sessionMounts, Home: ss.home, Resume: ss.conversation,
		Prompt: sessionPrompt(prompt, ss.start.Basis.Guide, n == 1), Tools: sessionTools, Timeout: ss.start.Basis.TurnTimeout,
		Transcript: filepath.Join(ss.record.dir, "transcripts", turn.ID+".jsonl"),
	}
	ss.record.write("prompt", map[string]any{"turn": turn.ID, "text": prompt})
	ss.record.write("turn.start", map[string]any{"turn": turn.ID, "profile": request.Profile, "workspace": request.Workspace, "readOnly": request.ReadOnly, "mounts": request.Mounts, "home": request.Home, "resume": request.Resume, "timeout": request.Timeout.String()})
	go s.runTurn(ctx, cancel, done, ss, n-1, request)
	return ss.view()
}

// runTurn runs one Agent turn and records what the Agents port observed. A
// turn whose container end was not observed, or a record that could not be
// written, halts the session.
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
	if !result.Started.IsZero() {
		ss.containers = true
	}
	ss.Usage = ss.Usage.Add(result.Usage)
	ss.record.write("turn.end", map[string]any{"turn": request.ID, "usage": result.Usage, "conversation": result.Conversation, "stopped": result.Stopped, "reason": t.Reason})
	if ss.State != Working {
		return // the session is being closed
	}
	switch {
	case !result.Stopped:
		ss.State, ss.Error = Halted, "the end of the Agent's container was not observed; close the session before anything else uses its workspace"
	case ss.record.failed() != nil:
		ss.State, ss.Error = Halted, "the session record could not be written: "+ss.record.failed().Error()
	default:
		ss.State = Waiting
	}
}

// sessionTool answers the session's tools: check and submit from the
// workspace as it is now, answer by keeping the Agent's text.
func (s *Service) sessionTool(ctx context.Context, ss *session, index int, name string, arguments json.RawMessage) run.ToolAnswer {
	var text string
	switch name {
	case "check":
	case "submit", "answer":
		var args struct {
			Summary string `json:"summary"`
			Text    string `json:"text"`
		}
		dec := json.NewDecoder(strings.NewReader(string(arguments)))
		dec.DisallowUnknownFields()
		err := dec.Decode(&args)
		text = args.Summary
		if name == "answer" {
			text = args.Text
		}
		if err != nil || strings.TrimSpace(text) == "" || len(text) > maxToolText || name == "submit" && args.Text != "" || name == "answer" && args.Summary != "" {
			field := map[string]string{"submit": "summary", "answer": "text"}[name]
			return run.ToolAnswer{Text: fmt.Sprintf("%s needs a %s of at most %d characters and nothing else", name, field, maxToolText), Refused: true}
		}
		if name == "answer" {
			s.mu.Lock()
			ss.Turns[index].Answers = append(ss.Turns[index].Answers, text)
			s.mu.Unlock()
			return run.ToolAnswer{Text: "The owner sees your answer."}
		}
	default:
		return run.ToolAnswer{Text: "unknown tool " + name, Refused: true}
	}
	harvest, err := s.peek(ctx, ss)
	if err != nil {
		return run.ToolAnswer{Text: "the workspace could not be read completely: " + err.Error(), Refused: true}
	}
	inputs := ss.start.Basis.Attempt.Model().Inputs()
	changes := snapshot.Compare(inputs.Raw.Census, harvest.Tree)
	if name == "check" {
		return run.ToolAnswer{Text: checkText(changes, apply.CheckProposal(ss.start.Basis.Attempt, harvest.Tree))}
	}
	st := stand{attempt: ss.start.Basis.Attempt, tree: inputs.Raw.Census, head: inputs.Head}
	p, err := s.propose(st, Proposal{Kind: "model", Title: "Model change from session " + ss.ID, Session: ss.ID, Summary: text}, harvest.Tree, sessionPath)
	if err != nil {
		return run.ToolAnswer{Text: err.Error(), Refused: true}
	}
	paths := make([]string, len(changes))
	for i, c := range changes {
		paths[i] = c.Path().String()
	}
	s.mu.Lock()
	ss.Turns[index].Proposals = append(ss.Turns[index].Proposals, p.ID)
	ss.record.write("proposal", map[string]any{"turn": ss.Turns[index].ID, "proposal": p.ID, "check": p.Check.Status, "paths": paths, "summary": text})
	s.mu.Unlock()
	return run.ToolAnswer{Text: fmt.Sprintf("Proposal %s was made; its structural check is %s with %d finding(s). The owner reviews the difference and decides; nothing is written before that.", p.ID, p.Check.Status, len(p.Check.Findings))}
}

// peek observes the session's workspace while its turn may be running; an
// uncertain observation is tried again, as the run engine does.
func (s *Service) peek(ctx context.Context, ss *session) (run.WorkspaceHarvest, error) {
	ports := ss.start.Ports
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		h, err := ports.Workspaces.Peek(ctx, run.HarvestRequest{ID: run.WorkspaceID(ss.ID + "-work"), MaxEntries: ports.MaxEntries, MaxBytes: ports.MaxBytes})
		if err == nil && h.Complete() {
			return h, nil
		}
		last = err
		if err == nil {
			last = fmt.Errorf("the observation is uncertain or holds unsupported entries: %s %v", h.Reason, h.Unsupported)
		}
		select {
		case <-ctx.Done():
			return run.WorkspaceHarvest{}, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 200 * time.Millisecond):
		}
	}
	return run.WorkspaceHarvest{}, last
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

// sessionPrompt states the task, the model's language and how to answer; a
// later prompt continues the same conversation.
func sessionPrompt(prompt, guide string, first bool) string {
	if !first {
		return "The owner writes:\n\n" + prompt + "\n\nThe same rules apply: change only .markitect/model/ and .markitect/registry/, use check, submit a proposal when a change is ready, and use answer otherwise."
	}
	return `You work on the canonical model of the project in /work. The owner asks:

` + prompt + `

Change only files under .markitect/model/ and .markitect/registry/; everything else in /work is read-only. The owner is the author of intent: change the model as the owner asks and do not decide open questions on the owner's behalf. If the model or the request is contradictory or unclear, say so instead of resolving it.
` + guide + `
Answer through the tools. Use check to see what the project's structural check reports for your workspace. When a change is ready, call submit with a summary for the owner that names what you changed and anything you found contradictory or unclear. If no change is needed or you need to know more, do not submit; call answer with what you found or need to know.`
}

func (ss *session) view() Session {
	v := ss.Session
	v.Turns = make([]SessionTurn, len(ss.Turns))
	for i, t := range ss.Turns {
		t.Answers = append([]string{}, t.Answers...)
		t.Tools = append([]ToolUse{}, t.Tools...)
		t.Proposals = append([]string{}, t.Proposals...)
		v.Turns[i] = t
	}
	return v
}

func randomSuffix() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// sessionRecord appends the session's evidence as JSON lines. A nil record
// writes nothing.
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

// failed returns the first write error.
func (r *sessionRecord) failed() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
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
