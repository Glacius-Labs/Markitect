// Package agentfake is a scripted implementation of agentport.Agents for
// tests: every start of a Manager's maker or checker runs the next scripted
// Step, which works on the Run's remote with real Git and calls the host's
// tools as an agent would. Only tests import it.
package agentfake

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
)

// Step answers one start.
type Step func(ctx context.Context, s agentport.Start, host agentport.Host) agentport.End

// Agents runs scripted Steps. A start with no Step left fails, so a test
// sees every start it did not expect.
type Agents struct {
	mu     sync.Mutex
	steps  map[key][]Step
	starts []agentport.Start
	// Dir holds the agents' working copies; New sets it to a fresh
	// temporary directory.
	Dir string
}

type key struct {
	manager string
	role    agentport.Role
}

var _ agentport.Agents = (*Agents)(nil)

// New returns Agents without Steps whose working copies live below dir.
func New(dir string) *Agents {
	return &Agents{steps: map[key][]Step{}, Dir: dir}
}

// On appends Steps for the starts of one Manager's maker or checker, in the
// order of the starts.
func (a *Agents) On(manager string, role agentport.Role, steps ...Step) *Agents {
	a.mu.Lock()
	defer a.mu.Unlock()
	k := key{manager, role}
	a.steps[k] = append(a.steps[k], steps...)
	return a
}

// Starts returns every start so far, in order.
func (a *Agents) Starts() []agentport.Start {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]agentport.Start(nil), a.starts...)
}

// Left returns the Steps no start has used, by "manager/role".
func (a *Agents) Left() map[string]int {
	a.mu.Lock()
	defer a.mu.Unlock()
	left := map[string]int{}
	for k, steps := range a.steps {
		if len(steps) > 0 {
			left[k.manager+" "+string(k.role)] = len(steps)
		}
	}
	return left
}

// Run runs the next Step for the start's Manager and role.
func (a *Agents) Run(ctx context.Context, s agentport.Start, host agentport.Host) agentport.End {
	a.mu.Lock()
	a.starts = append(a.starts, s)
	k := key{s.Manager.ID, s.Role}
	var step Step
	if len(a.steps[k]) > 0 {
		step, a.steps[k] = a.steps[k][0], a.steps[k][1:]
	}
	a.mu.Unlock()
	if step == nil {
		return agentport.End{Reason: agentport.Failed, Detail: fmt.Sprintf("agentfake: no step for %s %s round %d", s.Manager.ID, s.Role, s.Round), Agent: "agentfake"}
	}
	ctx, cancel := context.WithDeadline(ctx, s.Deadline)
	defer cancel()
	end := step(ctx, s, host)
	if end.Agent == "" {
		end.Agent = "agentfake"
	}
	return end
}

// Done ends a start as finished.
func Done() agentport.End { return agentport.End{Reason: agentport.Finished} }

// Fail ends a start as failed with the error's text.
func Fail(err error) agentport.End {
	return agentport.End{Reason: agentport.Failed, Detail: err.Error()}
}

// Work is an agent's working copy of its Manager's branch.
type Work struct {
	Dir   string
	start agentport.Start
	ctx   context.Context
}

// Clone clones the Run's remote into a fresh directory below a.Dir and
// checks out the start's branch.
func (a *Agents) Clone(ctx context.Context, s agentport.Start) (*Work, error) {
	if err := os.MkdirAll(a.Dir, 0o755); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(a.Dir, "work-")
	if err != nil {
		return nil, err
	}
	w := &Work{Dir: dir, start: s, ctx: ctx}
	if _, err := w.Git("clone", "--quiet", "--branch", s.Git.Branch, s.Git.URL, "."); err != nil {
		return nil, err
	}
	for _, kv := range [][2]string{{"user.name", s.Git.User}, {"user.email", s.Git.User + "@markitect.invalid"}, {"core.autocrlf", "false"}} {
		if _, err := w.Git("config", kv[0], kv[1]); err != nil {
			return nil, err
		}
	}
	return w, nil
}

// Git runs git in the working copy with the agent's credential and returns
// its standard output.
func (w *Work) Git(args ...string) (string, error) {
	full := args
	if strings.HasPrefix(w.start.Git.URL, "http") {
		auth := base64.StdEncoding.EncodeToString([]byte(w.start.Git.User + ":" + w.start.Git.Token))
		full = append([]string{"-c", "http.extraHeader=Authorization: Basic " + auth}, args...)
	}
	cmd := exec.CommandContext(w.ctx, "git", full...)
	cmd.Dir = w.Dir
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// Write writes a file of the working copy.
func (w *Work) Write(file, content string) error {
	p := filepath.Join(w.Dir, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}

// Remove removes a file of the working copy.
func (w *Work) Remove(file string) error {
	return os.Remove(filepath.Join(w.Dir, filepath.FromSlash(file)))
}

// Commit commits every change of the working copy, also none.
func (w *Work) Commit(message string) error {
	if _, err := w.Git("add", "-A"); err != nil {
		return err
	}
	_, err := w.Git("commit", "--quiet", "--allow-empty", "-m", message)
	return err
}

// Merge fetches and merges another Manager's branch, such as a child's.
func (w *Work) Merge(branch string) error {
	if _, err := w.Git("fetch", "--quiet", "origin", branch); err != nil {
		return err
	}
	_, err := w.Git("merge", "--quiet", "--no-edit", "FETCH_HEAD")
	return err
}

// Push pushes the working copy's branch.
func (w *Work) Push() error {
	_, err := w.Git("push", "--quiet", "origin", "HEAD:refs/heads/"+w.start.Git.Branch)
	return err
}

// Message returns the message of the branch tip.
func (w *Work) Message() (string, error) {
	return w.Git("log", "-1", "--format=%B")
}

// Verdict is one verdict line of a submission: the item's key, the verdict
// word and the evidence.
type Verdict struct{ Item, Word, Evidence string }

// Conforms gives every item of the start's checklist the verdict
// conforms, with the same evidence.
func Conforms(s agentport.Start, evidence string) []Verdict {
	var vs []Verdict
	for _, item := range s.Checklist {
		vs = append(vs, Verdict{item.Key, "conforms", evidence})
	}
	return vs
}

// SubmissionMessage writes a submission's commit message with its verdict
// trailers.
func SubmissionMessage(manager, subject string, verdicts []Verdict) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\nMarkitect-Manager: %s\n", subject, manager)
	for _, v := range verdicts {
		fmt.Fprintf(&b, "Markitect-Verdict: %s | %s | %s\n", v.Item, v.Word, v.Evidence)
	}
	return b.String()
}

// ReviewMessage writes a checker's review: approved when no item is
// disputed, rejected otherwise. disputed maps an item key to the defect.
func ReviewMessage(manager string, items []string, disputed map[string]string) string {
	review := "approved"
	if len(disputed) > 0 {
		review = "rejected"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Review %s\n\nMarkitect-Manager: %s\nMarkitect-Review: %s\n", manager, manager, review)
	for _, item := range items {
		if why, ok := disputed[item]; ok {
			fmt.Fprintf(&b, "Markitect-Check: %s | disputed | %s\n", item, why)
		} else {
			fmt.Fprintf(&b, "Markitect-Check: %s | confirmed | checked\n", item)
		}
	}
	return b.String()
}

// Keys returns the keys of the start's checklist.
func Keys(s agentport.Start) []string {
	var keys []string
	for _, item := range s.Checklist {
		keys = append(keys, item.Key)
	}
	return keys
}

// Submit returns a maker Step that merges the given branches, writes files
// (an empty content removes the file), and pushes one submission whose
// verdicts verdicts returns for the start.
func (a *Agents) Submit(merge []string, files map[string]string, verdicts func(agentport.Start) []Verdict) Step {
	return func(ctx context.Context, s agentport.Start, _ agentport.Host) agentport.End {
		w, err := a.Clone(ctx, s)
		if err != nil {
			return Fail(err)
		}
		for _, b := range merge {
			if err := w.Merge(b); err != nil {
				return Fail(err)
			}
		}
		for f, content := range files {
			if content == "" {
				err = w.Remove(f)
			} else {
				err = w.Write(f, content)
			}
			if err != nil {
				return Fail(err)
			}
		}
		if err := w.Commit(SubmissionMessage(s.Manager.ID, "Submit "+s.Manager.ID, verdicts(s))); err != nil {
			return Fail(err)
		}
		if err := w.Push(); err != nil {
			return Fail(err)
		}
		return Done()
	}
}

// Review returns a checker Step that answers the submission at the branch
// tip: it approves when disputed is empty and rejects otherwise.
func (a *Agents) Review(disputed map[string]string) Step {
	return func(ctx context.Context, s agentport.Start, _ agentport.Host) agentport.End {
		w, err := a.Clone(ctx, s)
		if err != nil {
			return Fail(err)
		}
		if err := w.Commit(ReviewMessage(s.Manager.ID, Keys(s), disputed)); err != nil {
			return Fail(err)
		}
		if err := w.Push(); err != nil {
			return Fail(err)
		}
		return Done()
	}
}

// Verdicts returns a verify checker Step that pushes its verdicts as one
// commit without file changes.
func (a *Agents) Verdicts(verdicts func(agentport.Start) []Verdict) Step {
	return func(ctx context.Context, s agentport.Start, _ agentport.Host) agentport.End {
		w, err := a.Clone(ctx, s)
		if err != nil {
			return Fail(err)
		}
		if err := w.Commit(SubmissionMessage(s.Manager.ID, "Verify "+s.Manager.ID, verdicts(s))); err != nil {
			return Fail(err)
		}
		if err := w.Push(); err != nil {
			return Fail(err)
		}
		return Done()
	}
}

// Call returns a Step that only calls the host, for example to ask, escalate
// or answer, and then finishes. An error of the call fails the start.
func Call(call func(ctx context.Context, s agentport.Start, host agentport.Host) error) Step {
	return func(ctx context.Context, s agentport.Start, host agentport.Host) agentport.End {
		if err := call(ctx, s, host); err != nil {
			return Fail(err)
		}
		return Done()
	}
}

// Hang returns a Step that does nothing until its deadline or the context
// ends, as a stuck agent would.
func Hang() Step {
	return func(ctx context.Context, s agentport.Start, _ agentport.Host) agentport.End {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			return agentport.End{Reason: agentport.DeadlineReached, Detail: "stopped at the deadline"}
		}
		return agentport.End{Reason: agentport.Cancelled, Detail: ctx.Err().Error()}
	}
}
