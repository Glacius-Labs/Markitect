package agents

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

// A chat agent is one conversation of Markitect Desktop's chat (concept 2.0
// §8): the container, login and provider of a Manager, but no Run. Its /work
// is a copy of the owner's working tree, sent in through Git as the start the
// conversation's changes are measured from; what the conversation changes
// there is read back as a proposal. It reuses the Manager agent's turn,
// login check and cleanup, and offers no Manager tools.

// ChatFile is one file of a chat's copy.
type ChatFile struct {
	Path    string
	Data    []byte
	Removed bool
}

// ChatSpec is everything one chat agent needs.
type ChatSpec struct {
	// Name is the container's unique name; its volumes carry it as prefix.
	Name   string
	Labels map[string]string
	Image  string
	Limits Limits
	// Network reaches the provider; a chat joins no other network.
	Network    string
	Hosts, DNS []string
	// Login is the operator's login file for the provider.
	Login  []byte
	Model  string
	Effort string
	// Skill and Earlier render the chat's documents (harness.Chat).
	Skill, Earlier string
	// Files are the working tree's files, which Read gives; they become the
	// copy's start. A file that cannot be read any more is left out.
	Files []string
	Read  func(file string) ([]byte, error)
	// Pending goes on top of the start: the open proposal of a conversation
	// this agent takes over.
	Pending []ChatFile
	// Turn bounds each turn.
	Turn       time.Duration
	Transcript io.Writer
	Now        func() time.Time
}

// Chat is one running chat agent.
type Chat struct {
	a       *Agent
	turn    time.Duration
	now     func() time.Time
	expires time.Time
}

// chatBranch is the branch of a chat's copy.
const chatBranch = "refs/heads/chat"

// StartChat starts a chat agent's container, copies the files into /work as
// its start with the pending proposal on top, installs the login and the
// chat's documents, and checks that the CLI uses the account login. It
// starts no turn. A failed start removes what it created.
func StartChat(ctx context.Context, rt Runtime, p Provider, spec ChatSpec) (*Chat, error) {
	now := time.Now
	if spec.Now != nil {
		now = spec.Now
	}
	if !namePattern.MatchString(spec.Name) || len(spec.Labels) == 0 || spec.Image == "" || spec.Network == "" || spec.Model == "" || spec.Effort == "" || spec.Turn <= 0 || spec.Read == nil {
		return nil, errors.New("agents: a chat needs a plain name, labels, an image, the provider's network, a model, an effort, a turn limit and its files")
	}
	doc, err := harness.Chat(harness.ChatInput{Skill: spec.Skill, Earlier: spec.Earlier})
	if err != nil {
		return nil, err
	}
	login, err := p.Login(spec.Login, now())
	if err != nil {
		return nil, &StartError{Class: LoginFailed, Err: err}
	}
	c := &Chat{turn: spec.Turn, now: now, expires: login.Expires}
	if err := c.fresh(); err != nil {
		return nil, &StartError{Class: LoginFailed, Err: err}
	}
	c.a = &Agent{
		spec: Spec{Name: spec.Name, Labels: spec.Labels, Image: spec.Image, Limits: spec.Limits, Network: spec.Network, Hosts: spec.Hosts, DNS: spec.DNS,
			Login: spec.Login, Model: spec.Model, Effort: spec.Effort, Transcript: spec.Transcript, Now: spec.Now,
			Start: agentport.Start{Deadline: now().Add(spec.Turn)}},
		provider: p, rt: rt, doc: doc, secrets: login.Secrets,
	}
	if err := c.start(ctx, login, spec); err != nil {
		if cerr := c.a.Close(context.WithoutCancel(ctx)); cerr != nil {
			err = errors.Join(err, fmt.Errorf("%w: %v", errCleanup, cerr))
		}
		return nil, err
	}
	return c, nil
}

// fresh refuses a login copy that would expire during the next turn: the
// copy cannot refresh itself, so a fresh agent takes a fresh copy.
func (c *Chat) fresh() error {
	if !c.expires.IsZero() && !c.expires.After(c.now().Add(c.turn+loginMargin)) {
		return fmt.Errorf("the login's access token expires at %s, within the next turn; refresh the login on the host first", c.expires.UTC().Format(time.RFC3339))
	}
	return nil
}

func (c *Chat) start(ctx context.Context, login Login, spec ChatSpec) error {
	a := c.a
	for _, v := range []string{spec.Name + "-home", spec.Name + "-work"} {
		if err := a.rt.CreateVolume(ctx, v, spec.Labels); err != nil {
			return err
		}
		a.volumes = append(a.volumes, v)
	}
	session, err := a.rt.Start(ctx, container.ContainerSpec{
		Name: spec.Name, Image: spec.Image, Labels: maps.Clone(spec.Labels), User: user, Workdir: workDir,
		Mounts: []container.Mount{{Volume: spec.Name + "-home", Target: homeDir}, {Volume: spec.Name + "-work", Target: workDir}},
		CPUs:   spec.Limits.CPUs, MemoryBytes: spec.Limits.MemoryBytes, Pids: spec.Limits.Pids, TmpBytes: spec.Limits.TmpBytes,
		Network: spec.Network, Hosts: spec.Hosts, DNS: spec.DNS,
	})
	if err != nil {
		return err
	}
	a.session = session
	if err := c.copyIn(ctx, spec); err != nil {
		return fmt.Errorf("copy the project into the chat: %w", err)
	}
	home, err := a.provider.Home(login, a.doc)
	if err != nil {
		return err
	}
	if err := session.Initialize(ctx, home); err != nil {
		return err
	}
	if err := a.checkLogin(ctx); err != nil {
		return &StartError{Class: LoginFailed, Err: err}
	}
	return nil
}

// copyIn makes /work a repository whose start ref holds the working tree's
// files and whose checkout holds them with the pending proposal on top. The
// files travel through git fast-import, so binary files arrive as they are.
func (c *Chat) copyIn(ctx context.Context, spec ChatSpec) error {
	a := c.a
	for _, argv := range [][]string{
		{gitPath, "config", "--global", "user.name", "Markitect chat"},
		{gitPath, "config", "--global", "user.email", "chat@markitect.invalid"},
		{gitPath, "config", "--global", "core.autocrlf", "false"},
		{gitPath, "init", "--quiet", workDir},
	} {
		if err := a.git(ctx, argv, "", io.Discard); err != nil {
			return err
		}
	}
	r, w := io.Pipe()
	go func() { w.CloseWithError(c.importStream(w, spec)) }()
	err := c.gitStream(ctx, []string{gitPath, "-C", workDir, "fast-import", "--quiet", "--done"}, r)
	// A fast-import that failed stops reading; the stream's writer ends.
	r.Close()
	if err != nil {
		return err
	}
	for _, argv := range [][]string{
		{gitPath, "-C", workDir, "symbolic-ref", "HEAD", chatBranch},
		{gitPath, "-C", workDir, "reset", "--quiet", "--hard"},
	} {
		if err := a.git(ctx, argv, "", io.Discard); err != nil {
			return err
		}
	}
	return nil
}

// importStream writes the fast-import stream of the copy: the start commit
// with every file, and the chat branch with the pending proposal on top.
func (c *Chat) importStream(w io.Writer, spec ChatSpec) error {
	when := c.now().Unix()
	header := func(ref, message string) string {
		return fmt.Sprintf("commit %s\ncommitter Markitect chat <chat@markitect.invalid> %d +0000\ndata %d\n%s\n", ref, when, len(message), message)
	}
	if _, err := io.WriteString(w, header(harness.ChatStart, "The working tree the chat starts from")); err != nil {
		return err
	}
	for _, f := range spec.Files {
		data, err := spec.Read(f)
		if err != nil {
			continue
		}
		if _, err := fmt.Fprintf(w, "M 100644 inline %s\ndata %d\n", importPath(f), len(data)); err != nil {
			return err
		}
		if _, err := w.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	if len(spec.Pending) == 0 {
		_, err := fmt.Fprintf(w, "\nreset %s\nfrom %s\n\ndone\n", chatBranch, harness.ChatStart)
		return err
	}
	if _, err := io.WriteString(w, "\n"+header(chatBranch, "The open proposal the chat takes over")+"from "+harness.ChatStart+"\n"); err != nil {
		return err
	}
	for _, f := range spec.Pending {
		if f.Removed {
			if _, err := fmt.Fprintf(w, "D %s\n", importPath(f.Path)); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "M 100644 inline %s\ndata %d\n", importPath(f.Path), len(f.Data)); err != nil {
			return err
		}
		if _, err := w.Write(append(f.Data, '\n')); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\ndone\n")
	return err
}

// importPath quotes a path for git fast-import, C-style.
func importPath(p string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(p); i++ {
		switch ch := p[i]; {
		case ch == '"' || ch == '\\':
			b.WriteByte('\\')
			b.WriteByte(ch)
		case ch == '\n':
			b.WriteString(`\n`)
		case ch < 0x20 || ch == 0x7f:
			fmt.Fprintf(&b, `\%03o`, ch)
		default:
			b.WriteByte(ch)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// gitStream runs one Git command in the container with a stream as its
// standard input, hardened like the Manager agent's own Git.
func (c *Chat) gitStream(ctx context.Context, argv []string, stdin io.Reader) error {
	argv = append([]string{argv[0], "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "--no-replace-objects"}, argv[1:]...)
	err := c.a.rt.Exec(ctx, c.a.spec.Name, workDir, argv, stdin, io.Discard)
	var exit *container.ExecError
	if errors.As(err, &exit) {
		return &container.ExecError{Code: exit.Code, Diagnostics: string(c.a.redact([]byte(exit.Diagnostics)))}
	}
	return err
}

// Send runs one turn of the conversation. A turn whose login copy would
// expire on the way is refused, so that a fresh agent takes over.
func (c *Chat) Send(ctx context.Context, message string) (Outcome, error) {
	if err := c.fresh(); err != nil {
		return Outcome{End: Failed, Class: LoginFailed, Detail: err.Error()}, nil
	}
	c.a.spec.Start.Deadline = c.now().Add(c.turn)
	return c.a.Send(ctx, message)
}

// Changes lists what the conversation changed since its start: the files
// below Markitect's folder with their content, and the other paths.
func (c *Chat) Changes(ctx context.Context) (model []ChatFile, outside []string, err error) {
	a := c.a
	if err := a.git(ctx, []string{gitPath, "-C", workDir, "add", "--all"}, "", io.Discard); err != nil {
		return nil, nil, err
	}
	var out bytes.Buffer
	if err := a.git(ctx, []string{gitPath, "-C", workDir, "diff", "--cached", "--no-renames", "--name-status", "-z", harness.ChatStart, "--"}, "", &out); err != nil {
		return nil, nil, err
	}
	fields := bytes.Split(bytes.TrimRight(out.Bytes(), "\x00"), []byte{0})
	for i := 0; i+1 < len(fields); i += 2 {
		status, path := string(fields[i]), string(fields[i+1])
		if !strings.HasPrefix(path, ".markitect/") {
			outside = append(outside, path)
			continue
		}
		if status == "D" {
			model = append(model, ChatFile{Path: path, Removed: true})
			continue
		}
		var data bytes.Buffer
		if err := a.git(ctx, []string{gitPath, "-C", workDir, "cat-file", "blob", ":" + path}, "", &data); err != nil {
			return nil, nil, err
		}
		model = append(model, ChatFile{Path: path, Data: data.Bytes()})
	}
	return model, outside, nil
}

// Settle makes the copy's current state its start after an accept (keep),
// or puts the start back after a reject.
func (c *Chat) Settle(ctx context.Context, keep bool) error {
	a := c.a
	if !keep {
		for _, argv := range [][]string{
			{gitPath, "-C", workDir, "reset", "--quiet", "--hard", harness.ChatStart},
			{gitPath, "-C", workDir, "clean", "-fdq"},
		} {
			if err := a.git(ctx, argv, "", io.Discard); err != nil {
				return err
			}
		}
		return nil
	}
	if err := a.git(ctx, []string{gitPath, "-C", workDir, "add", "--all"}, "", io.Discard); err != nil {
		return err
	}
	var tree, commit bytes.Buffer
	if err := a.git(ctx, []string{gitPath, "-C", workDir, "write-tree"}, "", &tree); err != nil {
		return err
	}
	if err := a.git(ctx, []string{gitPath, "-C", workDir, "commit-tree", strings.TrimSpace(tree.String()), "-p", harness.ChatStart, "-m", "The accepted proposal"}, "", &commit); err != nil {
		return err
	}
	return a.git(ctx, []string{gitPath, "-C", workDir, "update-ref", harness.ChatStart, strings.TrimSpace(commit.String())}, "", io.Discard)
}

// Close stops the provider, removes the login, ends the container and
// removes the volumes.
func (c *Chat) Close(ctx context.Context) error { return c.a.Close(ctx) }
