package agents

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
)

// Sessions implements the agent port: one provider session at a time per call,
// in the Command's one container, as the role's own Linux user, with the user's
// token in that session's environment and nowhere else.
type Sessions struct {
	Provider Provider
	Model    string
	Effort   string
	// Token is the user's account token (DEC-093).
	Token string
	// Budget is the most one session may spend, in the provider's own estimate;
	// zero sets no limit, and the Command's deadlines still hold.
	Budget float64
	// Transcripts, when set, is a folder for the sessions' transcripts: one
	// file per session, RUN/ROLE-ROUND.jsonl, every secret redacted.
	Transcripts string
	// ReviewerWithoutShell starts reviewers without a tool that runs commands, so
	// that test 1 of concept 3.0 §13 can compare the two. The implementer keeps
	// its shell.
	ReviewerWithoutShell bool

	mu sync.Mutex
	// checked holds the roles whose login was checked in a Run.
	checked map[string]bool
	// totals holds what each conversation reported so far: a continued session
	// reports the conversation's whole total.
	totals map[string]Usage
}

var _ agentport.Agents = (*Sessions)(nil)

// NewSessions checks the configuration.
func NewSessions(p Provider, model, effort, token, transcripts string, budget float64) (*Sessions, error) {
	if p == nil || model == "" || effort == "" {
		return nil, errors.New("agents: sessions need a provider, a model and an effort")
	}
	if p.TokenVariable() == "" {
		return nil, fmt.Errorf("agents: %s has no account token route yet, so a Command cannot start its sessions", p.Name())
	}
	if token == "" {
		return nil, errors.New("agents: sessions need the user's account token")
	}
	return &Sessions{Provider: p, Model: model, Effort: effort, Token: token, Budget: budget, Transcripts: transcripts, checked: map[string]bool{}, totals: map[string]Usage{}}, nil
}

// Run runs one session. The deadline and ctx end it, and the shell ends every
// process of that user before it returns.
func (g *Sessions) Run(ctx context.Context, sh agentport.Shell, s agentport.Session) agentport.End {
	end := agentport.End{Agent: g.Provider.Name() + ":" + g.Model}
	ctx, cancel := context.WithDeadline(ctx, s.Deadline)
	defer cancel()
	user, home := s.User, path.Join("/home", s.User)
	secret := g.tokenEnv()
	if err := g.checkLogin(ctx, sh, s, user, home); err != nil {
		end.Reason, end.Class, end.Detail = agentport.Failed, agentport.Login, g.redact(err.Error())
		return g.ended(ctx, end)
	}
	launch := Launch{Home: home, Conversation: s.Conversation, Prompt: s.Prompt, Model: g.Model, Effort: g.Effort, Budget: g.Budget,
		NoShell: g.ReviewerWithoutShell && s.Role == agentport.Reviewer}
	argv, env, stdin, err := g.Provider.Launch(launch)
	if err != nil {
		end.Reason, end.Class, end.Detail = agentport.Failed, agentport.Other, g.redact(err.Error())
		return end
	}
	var transcript io.Writer = io.Discard
	if g.Transcripts != "" {
		f, err := g.transcript(s)
		if err != nil {
			end.Reason, end.Class, end.Detail = agentport.Failed, agentport.Other, err.Error()
			return end
		}
		defer f.Close()
		transcript = f
	}
	reader := g.Provider.SessionReader(launch)
	out, in := io.Pipe()
	stderr := &tail{limit: 8 << 10}
	done := make(chan error, 1)
	go func() {
		err := sh.Exec(ctx, agentport.Exec{User: user, Dir: s.Dir, Argv: argv, Env: env, Secrets: []string{secret}, Stdin: bytes.NewReader(stdin), Stdout: in, Stderr: stderr})
		in.CloseWithError(io.EOF)
		done <- err
	}()
	lineErr := g.read(out, reader, transcript)
	if lineErr != nil {
		// A line that breaks the launch contract ends the session.
		cancel()
		go io.Copy(io.Discard, out)
	}
	execErr := <-done
	o := reader.Outcome()
	end.Conversation, end.Seen, end.Models = o.Conversation, g.redactSeen(reader.Seen()), reader.Models()
	if o.Model != "" {
		end.Agent = g.Provider.Name() + ":" + o.Model
	}
	end.Usage = g.usage(o)
	switch {
	case lineErr != nil:
		end.Reason, end.Class, end.Detail = agentport.Failed, agentport.Other, g.redact(lineErr.Error())
	case errors.Is(context.Cause(ctx), context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded:
		end.Reason, end.Detail = agentport.DeadlineReached, "the session's deadline passed"
	case ctx.Err() != nil:
		end.Reason, end.Detail = agentport.Cancelled, "the Command ended the session"
	case o.End == Answered:
		end.Reason, end.Text = agentport.Finished, g.redact(o.Text)
	case o.End == Failed:
		end.Reason, end.Class, end.Detail = agentport.Failed, class(o.Class), g.redact(o.Detail)
	case execErr != nil:
		end.Reason, end.Class, end.Detail = agentport.Failed, agentport.Other, g.redact(fmt.Sprintf("the provider ended without a result: %v: %s", execErr, stderr.String()))
	default:
		end.Reason, end.Class, end.Detail = agentport.Failed, agentport.Other, g.redact("the provider ended without a result: "+stderr.String())
	}
	return end
}

// ended is a session that did not start, sorted by what ended it.
func (g *Sessions) ended(ctx context.Context, end agentport.End) agentport.End {
	switch ctx.Err() {
	case context.DeadlineExceeded:
		end.Reason, end.Class, end.Detail = agentport.DeadlineReached, "", "the deadline passed before the session started"
	case context.Canceled:
		end.Reason, end.Class, end.Detail = agentport.Cancelled, "", "the Command ended before the session started"
	}
	return end
}

// checkLogin runs the CLI's own login report once per role and Run, before any
// inference: the account login, never an API key.
func (g *Sessions) checkLogin(ctx context.Context, sh agentport.Shell, s agentport.Session, user, home string) error {
	key := s.Run + "/" + user
	g.mu.Lock()
	done := g.checked[key]
	g.mu.Unlock()
	if done {
		return nil
	}
	argv, env := g.Provider.AuthStatus(home)
	var out, stderr bytes.Buffer
	err := sh.Exec(ctx, agentport.Exec{User: user, Dir: home, Argv: argv, Env: env, Secrets: []string{g.tokenEnv()}, Stdout: &out, Stderr: &stderr})
	if err != nil {
		return fmt.Errorf("%s reported no usable login: %v: %s", g.Provider.Name(), err, stderr.String())
	}
	if err := g.Provider.CheckAuth(out.Bytes()); err != nil {
		return err
	}
	g.mu.Lock()
	g.checked[key] = true
	g.mu.Unlock()
	return nil
}

func (g *Sessions) tokenEnv() string { return g.Provider.TokenVariable() + "=" + g.Token }

// read feeds the session's output to its reader, line by line.
func (g *Sessions) read(out io.Reader, r Reader, transcript io.Writer) error {
	scanner := bufio.NewScanner(out)
	scanner.Buffer(make([]byte, 64<<10), maxLine)
	for scanner.Scan() {
		line := scanner.Bytes()
		fmt.Fprintln(transcript, g.redact(string(line)))
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if err := r.Line(line); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// usage is what this session spent: a continued session reports the
// conversation's whole total.
func (g *Sessions) usage(o Outcome) agentport.Usage {
	u := o.Usage
	g.mu.Lock()
	defer g.mu.Unlock()
	before := g.totals[o.Conversation]
	if o.Conversation != "" {
		g.totals[o.Conversation] = u
	}
	return agentport.Usage{
		InputTokens:  max(u.Input+u.CacheCreation+u.CacheRead-before.Input-before.CacheCreation-before.CacheRead, 0),
		OutputTokens: max(u.Output-before.Output, 0),
		Partial:      u.Partial,
	}
}

func (g *Sessions) redact(s string) string {
	if len(g.Token) < 8 {
		return s
	}
	return string(bytes.ReplaceAll([]byte(s), []byte(g.Token), []byte("[redacted]")))
}

func (g *Sessions) redactSeen(s agentport.Seen) agentport.Seen {
	for i, c := range s.Commands {
		s.Commands[i] = g.redact(c)
	}
	return s
}

func (g *Sessions) transcript(s agentport.Session) (*os.File, error) {
	dir := filepath.Join(g.Transcripts, s.Run)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return os.Create(filepath.Join(dir, fmt.Sprintf("%s-%d.jsonl", s.Role, s.Round)))
}

func class(c Class) agentport.FailureClass {
	switch c {
	case Capacity:
		return agentport.Capacity
	case RateLimit:
		return agentport.RateLimit
	case ServerError:
		return agentport.ServerError
	case LoginFailed:
		return agentport.Login
	}
	return agentport.Other
}

// tail keeps the last limit bytes written to it.
type tail struct {
	limit int
	mu    sync.Mutex
	buf   []byte
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.limit {
		t.buf = append([]byte(nil), t.buf[len(t.buf)-t.limit:]...)
	}
	return len(p), nil
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}
