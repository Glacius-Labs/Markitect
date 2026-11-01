package agents

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"sort"
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

// Chat is one running chat agent. It takes one call at a time, as a
// conversation does.
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
	switch {
	case err == nil:
		return nil
	case errors.As(err, &exit):
		return &container.ExecError{Code: exit.Code, Diagnostics: string(c.a.redact([]byte(exit.Diagnostics)))}
	default:
		return errors.New(string(c.a.redact([]byte(err.Error()))))
	}
}

// Send runs one turn of the conversation; on hears each message the agent
// writes and what it is doing, as they come. Everything on hears is shown and
// kept, so a secret the agent could read in its home is redacted first. A
// turn whose login copy would expire on the way is refused, so that a fresh
// agent takes over.
func (c *Chat) Send(ctx context.Context, message string, on func(Progress)) (Outcome, error) {
	if err := c.fresh(); err != nil {
		return Outcome{End: Failed, Class: LoginFailed, Detail: err.Error()}, nil
	}
	c.a.spec.Start.Deadline = c.now().Add(c.turn)
	last := ""
	c.a.progress = func(p Progress) {
		p.Text = string(c.a.redact([]byte(p.Text)))
		p.Doing = shorten(string(c.a.redact([]byte(p.Doing))))
		if p.Text != "" {
			last = strings.TrimSpace(p.Text)
		}
		on(p)
	}
	out, err := c.a.Send(ctx, message)
	c.a.progress = nil
	out.Text = string(c.a.redact([]byte(out.Text)))
	// The provider's final answer repeats its last message; a provider that
	// reported none of its messages along the way says it here.
	if answer := strings.TrimSpace(out.Text); answer != "" && (last == "" || !strings.HasSuffix(answer, last)) {
		on(Progress{Text: out.Text})
	}
	return out, err
}

// snapshot hashes every file the copy holds as it is on disk, without the
// project's Git filters or line-end attributes, so that a file nobody
// changed never looks changed; the blobs are written, so that their content
// can be read back. A nested repository is left out. Every file of the copy
// is read each time, which a very large repository will feel.
func (c *Chat) snapshot(ctx context.Context) (map[string]string, error) {
	a := c.a
	var staged, others, deleted bytes.Buffer
	for _, step := range []struct {
		argv []string
		out  *bytes.Buffer
	}{
		{[]string{gitPath, "-C", workDir, "ls-files", "-z", "--stage"}, &staged},
		{[]string{gitPath, "-C", workDir, "ls-files", "-z", "--others", "--exclude-standard"}, &others},
		{[]string{gitPath, "-C", workDir, "ls-files", "-z", "--deleted"}, &deleted},
	} {
		if err := a.git(ctx, step.argv, "", step.out); err != nil {
			return nil, err
		}
	}
	gone := map[string]bool{}
	for _, p := range nulSplit(deleted.Bytes()) {
		gone[p] = true
	}
	present := map[string]bool{}
	for _, entry := range nulSplit(staged.Bytes()) {
		// mode SP object SP stage TAB path
		meta, p, ok := strings.Cut(entry, "\t")
		if ok && !strings.HasPrefix(meta, "160000 ") && !gone[p] {
			present[p] = true
		}
	}
	for _, p := range nulSplit(others.Bytes()) {
		if !strings.HasSuffix(p, "/") {
			present[p] = true
		}
	}
	var paths []string
	for p := range present {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return map[string]string{}, nil
	}
	// --stdin-paths reads a quoted path C-style, so any name passes.
	var list strings.Builder
	for _, p := range paths {
		list.WriteString(importPath(p) + "\n")
	}
	var ids bytes.Buffer
	if err := a.git(ctx, []string{gitPath, "-C", workDir, "hash-object", "-w", "--no-filters", "--stdin-paths"}, list.String(), &ids); err != nil {
		return nil, err
	}
	lines := strings.Fields(ids.String())
	if len(lines) != len(paths) {
		return nil, fmt.Errorf("agents: hashed %d of %d files of the chat's copy", len(lines), len(paths))
	}
	out := make(map[string]string, len(paths))
	for i, p := range paths {
		out[p] = lines[i]
	}
	return out, nil
}

// startTree lists the start's files with their blobs.
func (c *Chat) startTree(ctx context.Context) (map[string]string, error) {
	var tree bytes.Buffer
	if err := c.a.git(ctx, []string{gitPath, "-C", workDir, "ls-tree", "-r", "-z", "--full-tree", harness.ChatStart}, "", &tree); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, entry := range nulSplit(tree.Bytes()) {
		// mode SP type SP object TAB path
		meta, p, ok := strings.Cut(entry, "\t")
		if f := strings.Fields(meta); ok && len(f) == 3 && f[1] == "blob" {
			out[p] = f[2]
		}
	}
	return out, nil
}

func nulSplit(b []byte) []string {
	var out []string
	for _, part := range bytes.Split(b, []byte{0}) {
		if len(part) > 0 {
			out = append(out, string(part))
		}
	}
	return out
}

// maxDoing bounds the activity a chat shows.
const maxDoing = 110

func shorten(s string) string {
	if r := []rune(s); len(r) > maxDoing {
		return string(r[:maxDoing-1]) + "…"
	}
	return s
}

// Changes lists what the conversation changed since its start, comparing
// the files as they are on disk: the files below Markitect's folder with
// their content, and the other paths.
func (c *Chat) Changes(ctx context.Context) (model []ChatFile, outside []string, err error) {
	start, err := c.startTree(ctx)
	if err != nil {
		return nil, nil, err
	}
	now, err := c.snapshot(ctx)
	if err != nil {
		return nil, nil, err
	}
	var changed []string
	for p, id := range now {
		if start[p] != id {
			changed = append(changed, p)
		}
	}
	for p := range start {
		if _, ok := now[p]; !ok {
			changed = append(changed, p)
		}
	}
	sort.Strings(changed)
	for _, p := range changed {
		if !strings.HasPrefix(p, ".markitect/") {
			outside = append(outside, p)
			continue
		}
		id, ok := now[p]
		if !ok {
			model = append(model, ChatFile{Path: p, Removed: true})
			continue
		}
		var data bytes.Buffer
		if err := c.a.git(ctx, []string{gitPath, "-C", workDir, "cat-file", "blob", id}, "", &data); err != nil {
			return nil, nil, err
		}
		model = append(model, ChatFile{Path: p, Data: data.Bytes()})
	}
	return model, outside, nil
}

// Settle moves the start after an accept (keep): the start takes the
// copy's files below Markitect's folder, which the owner accepted, while
// changes elsewhere stay changes. After a reject it puts the copy back at
// the start.
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
	start, err := c.startTree(ctx)
	if err != nil {
		return err
	}
	now, err := c.snapshot(ctx)
	if err != nil {
		return err
	}
	var info strings.Builder
	for p, id := range now {
		if strings.HasPrefix(p, ".markitect/") && start[p] != id {
			fmt.Fprintf(&info, "100644 %s\t%s\n", id, importPath(p))
		}
	}
	for p := range start {
		if _, ok := now[p]; !ok && strings.HasPrefix(p, ".markitect/") {
			fmt.Fprintf(&info, "0 %s\t%s\n", strings.Repeat("0", 40), importPath(p))
		}
	}
	var tree, commit bytes.Buffer
	for _, step := range []struct {
		argv  []string
		stdin string
		out   io.Writer
	}{
		{[]string{gitPath, "-C", workDir, "read-tree", harness.ChatStart}, "", io.Discard},
		{[]string{gitPath, "-C", workDir, "update-index", "--index-info"}, info.String(), io.Discard},
		{[]string{gitPath, "-C", workDir, "write-tree"}, "", &tree},
	} {
		if err := a.git(ctx, step.argv, step.stdin, step.out); err != nil {
			return err
		}
	}
	if err := a.git(ctx, []string{gitPath, "-C", workDir, "commit-tree", strings.TrimSpace(tree.String()), "-p", harness.ChatStart, "-m", "The accepted proposal"}, "", &commit); err != nil {
		return err
	}
	return a.git(ctx, []string{gitPath, "-C", workDir, "update-ref", harness.ChatStart, strings.TrimSpace(commit.String())}, "", io.Discard)
}

// Close stops the provider, removes the login, ends the container and
// removes the volumes.
func (c *Chat) Close(ctx context.Context) error { return c.a.Close(ctx) }
