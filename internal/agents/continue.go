package agents

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

// shell writes the renewed login copy in the container; the image's Debian
// base has it.
const shell = "/bin/sh"

// Continue runs a later turn of a living agent for a new start of the same
// Manager and role (concept 2.0 §6.2, long-lived Managers): it renews the
// login copy from the operator's login, fetches the Run's remote into the
// clone instead of cloning again, and sends the conversation only what
// brought the turn about. A maker's branch moves to its tip on the remote
// when it can, so a review or the host's integration on it is the maker's
// too; a checker's working tree becomes the submission it reviews. Nothing
// the agent did not commit reaches a checker or the Result.
func (a *Agent) Continue(ctx context.Context, s agentport.Start, host agentport.Host, login []byte, transcript io.Writer) (Outcome, error) {
	prev := a.spec.Start
	switch {
	case s.Run != prev.Run || s.Mode != prev.Mode || s.Role != prev.Role || s.Manager.ID != prev.Manager.ID:
		return Outcome{}, fmt.Errorf("agents: the agent of %s %s cannot take a turn of %s %s", prev.Manager.ID, prev.Role, s.Manager.ID, s.Role)
	case s.Git.URL != prev.Git.URL || s.Git.Branch != prev.Git.Branch || s.Git.User != prev.Git.User || s.Git.Token != prev.Git.Token:
		return Outcome{}, errors.New("agents: a later turn reaches the remote as the agent's first did")
	case host == nil || s.Deadline.IsZero() || !commitPattern.MatchString(s.Git.Base):
		return Outcome{}, errors.New("agents: a turn needs a host, a deadline and a plain base")
	}
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return Outcome{}, errors.New("agents: the agent is closed")
	}
	a.spec.Start, a.spec.Host, a.spec.Transcript = s, host, transcript
	a.mu.Unlock()
	turnCtx, cancel := context.WithDeadline(ctx, s.Deadline)
	defer cancel()
	if err := a.renewLogin(turnCtx, login); err != nil {
		return Outcome{}, err
	}
	note, err := a.update(turnCtx)
	if err != nil {
		return Outcome{}, err
	}
	// The conversation holds the system documents, the first turn's task and
	// the checklist, which do not change within a Run; agentd verifies the
	// installed system documents before every provider start.
	message, err := a.turn(turnCtx)
	if err != nil {
		return Outcome{}, err
	}
	a.tools = harness.Tools(s)
	if note != "" {
		message += "\n\n" + note
	}
	return a.Send(ctx, message)
}

// renewLogin replaces the login copy in the private home with one made from
// the operator's login as it is now, so a login refreshed on the host reaches
// a container that lives for the Run. It must outlive the turn's deadline as
// at the agent's start.
func (a *Agent) renewLogin(ctx context.Context, source []byte) error {
	now := time.Now
	if a.spec.Now != nil {
		now = a.spec.Now
	}
	login, err := a.provider.Login(source, now())
	if err != nil {
		return &StartError{Class: LoginFailed, Err: err}
	}
	deadline := a.spec.Start.Deadline
	if !login.Expires.IsZero() && !login.Expires.After(deadline.Add(loginMargin)) {
		return &StartError{Class: LoginFailed, Err: fmt.Errorf("the %s login's access token expires at %s, before the deadline %s; refresh the login on the host first", a.provider.Name(), login.Expires.UTC().Format(time.RFC3339), deadline.UTC().Format(time.RFC3339))}
	}
	a.mu.Lock()
	a.secrets = append(a.secrets, login.Secrets...)
	a.mu.Unlock()
	// A new file renamed over the old one, so a failed write leaves the old
	// copy; the private home's own file name stays, which agentd removes at
	// the end.
	file := a.provider.LoginFile()
	script := `umask 077 && cat > "$1.new" && mv -f "$1.new" "$1"`
	if err := a.rt.Exec(ctx, a.spec.Name, homeDir, []string{shell, "-c", script, "renew", file}, bytes.NewReader(login.Copy), io.Discard); err != nil {
		return &StartError{Class: LoginFailed, Err: fmt.Errorf("renew the login copy: %s", a.redact([]byte(err.Error())))}
	}
	return nil
}

// update fetches the Run's remote into the clone. A maker's branch moves
// forward to its tip on the remote when the clone allows it; otherwise the
// returned note tells the maker. A checker's working tree is reset to the
// submission at its branch's tip, without the files a build left untracked
// but keeping ignored ones, such as caches.
func (a *Agent) update(ctx context.Context) (string, error) {
	branch := a.branch()
	remote := "refs/remotes/origin/" + branch
	if err := a.git(ctx, []string{gitPath, "fetch", "--quiet", "origin"}, "", io.Discard); err != nil {
		return "", fmt.Errorf("fetch the Run's remote: %w", err)
	}
	if a.spec.Start.Role == agentport.Checker {
		for _, argv := range [][]string{
			{gitPath, "checkout", "--quiet", "--force", "-B", branch, remote},
			{gitPath, "clean", "--quiet", "-ffd"},
		} {
			if err := a.git(ctx, argv, "", io.Discard); err != nil {
				return "", fmt.Errorf("check out the submission: %w", err)
			}
		}
		return "", nil
	}
	tip, err := a.rev(ctx, remote)
	if err != nil {
		return "", err
	}
	var head bytes.Buffer
	if err := a.git(ctx, []string{gitPath, "symbolic-ref", "--quiet", "HEAD"}, "", &head); err != nil || strings.TrimSpace(head.String()) != "refs/heads/"+branch {
		return fmt.Sprintf("Your clone is not on your branch `%s`, whose tip on the remote is `%s`: bring your work onto it before you submit.", branch, tip), nil
	}
	if err := a.git(ctx, []string{gitPath, "merge", "--quiet", "--ff-only", remote}, "", io.Discard); err != nil {
		return fmt.Sprintf("Your branch `%s` moved on the remote to `%s`, and your clone could not follow it by a fast-forward (`git merge --ff-only origin/%s`): merge it before you submit.", branch, tip, branch), nil
	}
	return "", nil
}
