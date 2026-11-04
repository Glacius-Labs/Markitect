// Package run is the process of a Run, in the tool (draft concept 4.0): apply
// and verify start a background session and end; the session calls the tool,
// and submit is the one door: it says what is next each time and decides whether
// the work counts. Nothing here drives a session. Sessions start through
// Sessions, the required checks run through Checks, Git work goes through Git,
// and what a Run knows between commands lives in a Store.
//
// The process is described in internal/run/README.md.
package run

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

const (
	// repairRounds bounds how often a review's wrong answers go back.
	repairRounds = 2
	// maxSubmits bounds the calls of submit in one Run, which keeps a stuck Run
	// finite: a provisional number.
	maxSubmits = 30
	// maxBlocks bounds how often the Stop hook sends a session back since its
	// last submit.
	maxBlocks = 5
	// defaultDeadline bounds a Run.
	defaultDeadline = 6 * time.Hour
	// maxRefusals bounds how many errors one refusal lists.
	maxRefusals = 30
)

// RunBranchPrefix is where work and a Result that do not land stay.
const RunBranchPrefix = "markitect/runs/"

// worktreeBranch is the branch the provider makes for a session's worktree.
func worktreeBranch(run string) string { return "worktree-" + run }

// runName is what a Run may be called: it names a branch and a folder.
var runName = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,62}$`)

// ErrUsage marks a request the tool cannot run, such as an unknown base.
var ErrUsage = errors.New("cannot run")

// Engine runs the commands of a Run in one repository.
type Engine struct {
	Git      Git
	Sessions Sessions
	Checks   Checks
	Store    Store
	// Now defaults to the clock.
	Now func() time.Time
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Request is one apply or verify.
type Request struct {
	// Dir is a folder of the owner's working tree.
	Dir string
	// Base is the commit impact is computed from; empty means the work of the
	// last landing.
	Base     string
	Full     bool
	Findings []analysis.Finding
	// Deadline bounds the Run; zero is the default.
	Deadline time.Duration
}

// Started says what apply or verify did. Text is what the owner reads.
type Started struct {
	// State is empty for a Run that started; otherwise StateCheckFailed or
	// StateNothing, or "open" for a Run that was already running.
	State string
	Run   string
	Text  string
}

// Apply starts the session that makes the project match the spec, and returns
// as soon as it is up. The spec stays as the owner left it: an error in it stops
// here, a change to it is committed here, and the Run works on that commit.
func (e *Engine) Apply(ctx context.Context, req Request) (Started, error) {
	root, err := e.Git.Root(ctx, req.Dir)
	if err != nil {
		return Started{}, err
	}
	branch, err := e.Git.Branch(ctx, root)
	if err != nil {
		return Started{}, err
	}
	if branch == "" {
		return Started{}, fmt.Errorf("%w: HEAD is detached, and a Run lands on a branch; check one out first", ErrUsage)
	}
	e.cleanUp(ctx)
	runs, err := e.Store.All()
	if err != nil {
		return Started{}, err
	}
	for _, st := range runs {
		if st.Open && st.Command == "apply" && st.Branch == branch {
			return e.continueOpen(ctx, st)
		}
	}
	dirty, err := e.Git.Uncommitted(ctx, root)
	if err != nil {
		return Started{}, err
	}
	if len(dirty) > 0 {
		return Started{State: StateCheckFailed, Text: "Changes outside the spec are not committed, and a run would silently leave them out. Commit or remove them first:\n\n- " + strings.Join(dirty, "\n- ") + "\n"}, nil
	}
	cur, err := e.loadWorking(ctx, root)
	if err != nil {
		return Started{}, err
	}
	if errs := cur.SpecErrors(); len(errs) > 0 {
		return Started{State: StateCheckFailed, Text: "The spec has errors, so no agent starts. Fix them in the spec:\n\n" + listErrors(errs)}, nil
	}
	start, err := e.Git.CommitSpec(ctx, root, "The spec, as markitect apply found it")
	if err != nil {
		return Started{}, err
	}
	if !e.hasHarness(ctx, start, root) {
		return Started{State: StateCheckFailed, Text: noHarness}, nil
	}
	landedOne := e.retryLandings(ctx, runs, branch, start)
	if landedOne != "" {
		start = landedOne
	}
	st, err := e.newRun(ctx, "apply", req, root, branch, start)
	if err != nil {
		return Started{}, err
	}
	if len(st.Items) == 0 {
		return Started{State: StateNothing, Text: fmt.Sprintf("Nothing to do: nothing changed between %s and %s that reaches the spec.\n", short(st.Base), short(start))}, nil
	}
	p, err := project.Load(e.Git.State(ctx, start))
	if err != nil {
		return Started{}, err
	}
	st.Spec = SessionSpec{Run: st.Run, Dir: root, Model: p.SessionModel, Effort: p.SessionEffort, Prompt: implementerTask(st.Run), Store: e.Store.Dir}
	return e.launch(ctx, st, fmt.Sprintf("Run %s started on %s: %d items, base %s. It works in the background in a worktree of its own, and you can keep modeling. Its result lands on %s by itself; `markitect status` says how it stands.\n", st.Run, branch, len(st.Items), short(st.Base), branch))
}

// Verify starts a reviewer's session on the last commit. It changes nothing the
// owner has, and never holds the branch.
func (e *Engine) Verify(ctx context.Context, req Request) (Started, error) {
	root, err := e.Git.Root(ctx, req.Dir)
	if err != nil {
		return Started{}, err
	}
	branch, err := e.Git.Branch(ctx, root)
	if err != nil {
		return Started{}, err
	}
	e.cleanUp(ctx)
	runs, err := e.Store.All()
	if err != nil {
		return Started{}, err
	}
	for _, st := range runs {
		if st.Open && st.Command == "verify" && st.Branch == branch {
			return e.continueOpen(ctx, st)
		}
	}
	start, err := e.Git.Head(ctx, root)
	if err != nil {
		return Started{}, err
	}
	cur, err := project.Load(e.Git.State(ctx, start))
	if err != nil {
		return Started{}, err
	}
	if errs := cur.SpecErrors(); len(errs) > 0 {
		return Started{State: StateCheckFailed, Text: "The spec has errors at the last commit, so no agent starts:\n\n" + listErrors(errs)}, nil
	}
	if !e.hasHarness(ctx, start, root) {
		return Started{State: StateCheckFailed, Text: noHarness}, nil
	}
	st, err := e.newRun(ctx, "verify", req, root, branch, start)
	if err != nil {
		return Started{}, err
	}
	if len(st.Items) == 0 {
		return Started{State: StateNothing, Text: fmt.Sprintf("Nothing to do: nothing changed between %s and %s that reaches the spec.\n", short(st.Base), short(start))}, nil
	}
	commit, err := e.Git.Commit(ctx, start)
	if err != nil {
		return Started{}, err
	}
	st.Review = &Review{Commit: start, Tree: commit.Tree, Items: st.Items, Answered: map[string]Answer{}}
	st.Spec = SessionSpec{Run: st.Run, Dir: root, Agent: ReviewerAgent, Model: cur.SessionModel, Effort: cur.SessionEffort, Prompt: verifyTask(st.Run), Store: e.Store.Dir}
	dirtyNote := ""
	if dirty, err := e.Git.Uncommitted(ctx, root); err == nil && len(dirty) > 0 {
		dirtyNote = " Uncommitted work is not looked at."
	}
	return e.launch(ctx, st, fmt.Sprintf("Run %s started: a reviewer checks %d items at %s in the background. It changes nothing you have; `markitect status` says how it stands.%s\n", st.Run, len(st.Items), short(start), dirtyNote))
}

// reviewerFile is the harness's reviewer agent, without which no Run can finish.
const reviewerFile = ".claude/agents/" + ReviewerAgent + ".md"

const noHarness = "The harness is not in this project's last commit, so a run could not start its reviewer. Run markitect init, which commits what it writes, and try again.\n"

// hasHarness looks in the commit and on disk: the provider reads agents that are
// not tracked through from the main checkout.
func (e *Engine) hasHarness(ctx context.Context, commit, root string) bool {
	if _, err := e.Git.State(ctx, commit).Read(reviewerFile); err == nil {
		return true
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(reviewerFile)))
	return err == nil
}

// newRun works out the base and the list. A Run with an empty list is not saved.
func (e *Engine) newRun(ctx context.Context, command string, req Request, root, branch, start string) (*RunState, error) {
	base := req.Base
	full := req.Full
	if base == "" {
		landed, ok, err := e.lastBase(ctx, start)
		switch {
		case err != nil:
			return nil, err
		case ok:
			base = landed
		case len(req.Findings) > 0:
			base = start
		default:
			// With no Result yet there is nothing to compare with, so the first
			// Run covers everything.
			base, full = start, true
		}
	}
	bc, err := e.Git.Commit(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("%w: base %q is no commit: %v", ErrUsage, base, err)
	}
	base = bc.ID
	before, err := project.Load(e.Git.State(ctx, base))
	if err != nil {
		return nil, err
	}
	cur, err := project.Load(e.Git.State(ctx, start))
	if err != nil {
		return nil, err
	}
	imp, err := e.impact(ctx, base, start, before, cur, req.Findings, full)
	if err != nil {
		return nil, err
	}
	items := checklist(imp, before, cur, req.Findings)
	now := e.now()
	name, err := e.runName(now)
	if err != nil {
		return nil, err
	}
	limit := req.Deadline
	if limit <= 0 {
		limit = defaultDeadline
	}
	return &RunState{Run: name, Command: command, Branch: branch, Repo: root, Base: base, Start: start, Full: full, Findings: req.Findings,
		Items: items, Answers: map[string]Answer{}, Impact: impactKeys(imp, items), Notes: imp.Notes, Passed: map[string]bool{},
		Began: now, Deadline: now.Add(limit), Limit: limit, Open: true}, nil
}

func (e *Engine) runName(now time.Time) (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "r" + now.UTC().Format("20060102-150405") + "-" + hex.EncodeToString(b), nil
}

// launch starts the Run's session and waits until it is up.
func (e *Engine) launch(ctx context.Context, st *RunState, text string) (Started, error) {
	if err := e.Store.Save(st); err != nil {
		return Started{}, err
	}
	id, err := e.Sessions.Start(ctx, st.Spec)
	if err != nil {
		st.Open, st.State, st.Why, st.Closed = false, StateIncomplete, fmt.Sprintf("the session did not start: %v", err), e.now()
		if serr := e.Store.Save(st); serr != nil {
			return Started{}, serr
		}
		return Started{State: StateIncomplete, Run: st.Run, Text: "The session did not start: " + err.Error() + "\n"}, nil
	}
	// A quick session may already have written to the state.
	if err := e.Store.Update(st.Run, func(s *RunState) error {
		s.Session = id
		return nil
	}); err != nil {
		return Started{}, err
	}
	return Started{Run: st.Run, Text: text}, nil
}

// continueOpen handles an apply while a Run is open: one that works is left
// alone, and one whose session has ended is resumed where it stopped.
func (e *Engine) continueOpen(ctx context.Context, st *RunState) (Started, error) {
	state := SessionUnknown
	if st.Session != "" {
		var err error
		if state, err = e.Sessions.State(ctx, st.Session); err != nil {
			return Started{}, err
		}
	}
	if state == SessionWorking {
		return Started{State: "open", Run: st.Run, Text: fmt.Sprintf("Run %s is still working on %s, so no new one starts. `markitect status` says how it stands, and `markitect stop WHY` ends it. You can keep modeling; the next apply takes what you have.\n", st.Run, st.Branch)}, nil
	}
	// It resumes in its own worktree, or not at all: the provider would otherwise
	// carry on in the owner's folder without isolation.
	wt, err := e.Git.WorktreeOf(ctx, worktreeBranch(st.Run))
	if err != nil {
		return Started{}, err
	}
	if wt == "" {
		return Started{}, fmt.Errorf("run %s could not be resumed, since its worktree is gone; `markitect stop WHY` ends it", st.Run)
	}
	if st.Session != "" {
		_ = e.Sessions.Stop(ctx, st.Session)
	}
	spec := st.Spec
	spec.Dir = wt
	id, err := e.Sessions.Resume(ctx, st.Session, spec)
	if err != nil {
		return Started{}, fmt.Errorf("run %s could not be resumed; `markitect stop WHY` ends it: %w", st.Run, err)
	}
	if err := e.Store.Update(st.Run, func(s *RunState) error {
		s.Session, s.Failure, s.Blocks = id, "", 0
		// A usage limit or a night's sleep must not end it at its first tool call.
		s.Deadline = e.now().Add(s.Limit)
		return nil
	}); err != nil {
		return Started{}, err
	}
	return Started{State: "open", Run: st.Run, Text: fmt.Sprintf("Run %s had stopped working and is resumed where it was, with its repair count. The last apply's changes stay for the next one.\n", st.Run)}, nil
}

// Current is the Run whose worktree holds a folder. What a session calls works
// only there: a session that ended up in the owner's folder must not be told to go
// on working.
func (e *Engine) Current(ctx context.Context, dir string) (*RunState, error) {
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
		return nil, err
	}
	st, found, err := e.Store.Load(filepath.Base(root))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%w: this is not the worktree of a run", ErrUsage)
	}
	return st, nil
}

// currentOrOpen is the Run a folder belongs to, else the open Run on its branch:
// an apply first, since only one can be open.
func (e *Engine) currentOrOpen(ctx context.Context, dir string) (*RunState, error) {
	if st, err := e.Current(ctx, dir); err == nil {
		return st, nil
	}
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
		return nil, err
	}
	branch, err := e.Git.Branch(ctx, root)
	if err != nil {
		return nil, err
	}
	runs, err := e.Store.All()
	if err != nil {
		return nil, err
	}
	var open *RunState
	for _, st := range runs {
		if st.Open && st.Branch == branch && (open == nil || st.Command == "apply") {
			open = st
		}
	}
	if open == nil {
		return nil, fmt.Errorf("%w: no run is open here", ErrUsage)
	}
	return open, nil
}

// Checklist prints the implementer's task and to-do list, rebuilt from the
// current files and registry. The Run's first call also tries the required checks
// on the start. With review it prints what the review must answer.
func (e *Engine) Checklist(ctx context.Context, dir string, review bool) (string, error) {
	st, err := e.Current(ctx, dir)
	if err != nil {
		return "", err
	}
	if !st.Open {
		return fmt.Sprintf("Run %s is closed: %s. %s. End now.\n", st.Run, st.State, capitalize(st.Why)), nil
	}
	if review {
		if st.Review == nil {
			return "", fmt.Errorf("%w: nothing is under review yet", ErrUsage)
		}
		return reviewPrompt(st, st.Review), nil
	}
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(implementerTask(st.Run))
	if !st.Tried {
		text, closed, err := e.tryChecks(ctx, st)
		if err != nil {
			return "", err
		}
		b.WriteString(text)
		if closed {
			return b.String(), nil
		}
	}
	cur, err := e.loadWorking(ctx, root)
	if err != nil {
		return "", err
	}
	before, err := project.Load(e.Git.State(ctx, st.Base))
	if err != nil {
		return "", err
	}
	changed, err := e.changedSinceBase(ctx, st, root)
	if err != nil {
		return "", err
	}
	imp := analysis.ComputeImpact(analysis.ImpactInput{Base: before, Current: cur, Changed: changed, Findings: st.Findings, Full: st.Full})
	items := mergeItems(st.Items, checklist(imp, before, cur, st.Findings))
	fmt.Fprintf(&b, "\nThe base is %s: `markitect impact --base %s` shows what changed. This list says what the change reaches and where the registry says to look. It is your to-do list, and you answer nothing on it:\n\n", st.Base, st.Base)
	b.WriteString(renderList(items))
	b.WriteString("\n")
	b.WriteString(checksList(cur.Checks))
	return b.String(), nil
}

// changedSinceBase lists what the worktree changed since the base, outside
// Markitect's folder.
func (e *Engine) changedSinceBase(ctx context.Context, st *RunState, root string) ([]analysis.Change, error) {
	cand, err := e.Git.Candidate(ctx, root, "the work so far")
	if err != nil {
		return nil, err
	}
	all, err := e.Git.Changed(ctx, st.Base, cand)
	if err != nil {
		return nil, err
	}
	return outsideOwnFolder(all), nil
}

func outsideOwnFolder(all []analysis.Change) []analysis.Change {
	var changed []analysis.Change
	for _, f := range all {
		if !strings.HasPrefix(f.Path, project.Dir+"/") {
			changed = append(changed, f)
		}
	}
	return changed
}

// tryChecks runs the required checks once on the start, before the work is
// spent. A check that cannot start ends the Run; one that fails is told to the
// implementer and to the Result, as failing before the work.
func (e *Engine) tryChecks(ctx context.Context, st *RunState) (string, bool, error) {
	p, err := project.Load(e.Git.State(ctx, st.Start))
	if err != nil {
		return "", false, err
	}
	results, err := e.runChecks(ctx, st.Start, p.Checks)
	if err != nil {
		return "", false, err
	}
	var failing []string
	for _, res := range results {
		if res.NotRun {
			why := fmt.Sprintf("the required check %s could not start, before any work (exit %d): %s", res.Name, res.Exit, tail(res.Output))
			if err := e.close(ctx, st, StateIncomplete, why, nil); err != nil {
				return "", false, err
			}
			return "\nThis run has ended: " + why + "\nEnd now. Nothing lands.\n", true, nil
		}
		if !res.Passed {
			failing = append(failing, res.Name)
		}
	}
	if err := e.Store.Update(st.Run, func(s *RunState) error {
		s.Tried, s.Failing, s.Checks, s.ChecksOn = true, failing, results, st.Start
		return nil
	}); err != nil {
		return "", false, err
	}
	st.Tried, st.Failing = true, failing
	if len(failing) > 0 {
		return fmt.Sprintf("\nThese required checks already fail before any work: %s. If the cause is a missing service or tool, say so with: markitect stop --environment \"why\". Otherwise they must pass by the end.\n", strings.Join(failing, ", ")), false, nil
	}
	return "", false, nil
}

// runChecks runs the required checks in a clean export of a commit.
func (e *Engine) runChecks(ctx context.Context, commit string, checks []project.Check) ([]CheckResult, error) {
	if len(checks) == 0 {
		return nil, nil
	}
	dir, err := os.MkdirTemp("", "markitect-check-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := e.Git.Export(ctx, commit, dir); err != nil {
		return nil, err
	}
	return e.Checks.Run(ctx, dir, checks), nil
}

// Answer stores a reviewer's answers against the state under review. What it
// cannot accept is refused at once, with what is still missing.
func (e *Engine) Answer(ctx context.Context, dir string, raw []byte) (string, bool, error) {
	st, err := e.Current(ctx, dir)
	if err != nil {
		return "", false, err
	}
	if !st.Open {
		return fmt.Sprintf("Run %s is closed: %s. End now.\n", st.Run, st.State), false, nil
	}
	if st.Review == nil {
		return "Nothing is under review. End now.\n", false, nil
	}
	state := e.Git.State(ctx, st.Review.Commit)
	answers, problems := parseAnswers(raw, st.Review.Items)
	checked, links := checkAnswers(state, st.Review.Items, answers)
	problems = append(problems, checked...)
	if len(problems) > 0 {
		var b strings.Builder
		b.WriteString("These answers are not accepted, and nothing from this call is stored. Correct them and call markitect answer again:\n\n")
		for _, p := range problems {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		return b.String(), false, nil
	}
	var missing []string
	if err := e.Store.Update(st.Run, func(s *RunState) error {
		if s.Review == nil {
			return nil
		}
		for k, a := range answers {
			s.Review.Answered[k] = a
			s.Answers[k] = a
		}
		s.MissingFor = dedupe(append(s.MissingFor, links...))
		missing = s.Review.Missing()
		return nil
	}); err != nil {
		return "", false, err
	}
	if len(missing) > 0 {
		return fmt.Sprintf("Stored. %d items still have no answer:\n\n- %s\n", len(missing), strings.Join(missing, "\n- ")), false, nil
	}
	if st.Command == "verify" {
		return "Stored: every item has an answer. Now run: markitect submit\n", true, nil
	}
	return "Stored: every item has an answer. You are done; report in one line and end.\n", true, nil
}

// Stop ends a Run without landing. Someone else's stop ends the session first, so
// that nothing changes the work afterwards, and a submit that is running is left to
// finish, since it may be landing. With environment, the session says the machine
// lacks something, which is no question about the spec.
func (e *Engine) Stop(ctx context.Context, dir, why string, environment bool) (string, error) {
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
		return "", err
	}
	st, err := e.currentOrOpen(ctx, dir)
	if err != nil {
		return "", err
	}
	if !st.Open {
		return fmt.Sprintf("Run %s is already closed: %s.\n", st.Run, st.State), nil
	}
	release, err := e.Store.TrySubmitLock(st.Run)
	if errors.Is(err, ErrBusy) {
		return fmt.Sprintf("A markitect submit of run %s is running and may be landing, so the run is not stopped. Try again when it has finished.\n", st.Run), nil
	}
	if err != nil {
		return "", err
	}
	defer release()
	if st, _, err = e.Store.Load(st.Run); err != nil {
		return "", err
	}
	if !st.Open {
		return fmt.Sprintf("Run %s is already closed: %s.\n", st.Run, st.State), nil
	}
	fromSession := filepath.Base(root) == st.Run
	if !fromSession && st.Session != "" {
		_ = e.Sessions.Stop(ctx, st.Session)
	}
	state, decisions := StateSpecProblem, []OwnerDecision{{Matter: why, Recommendation: "Settle it in the spec and run apply again."}}
	switch {
	case !fromSession:
		state, decisions = StateStopped, nil
	case environment:
		state, decisions = StateIncomplete, nil
	}
	if err := e.close(ctx, st, state, "stopped: "+why, decisions); err != nil {
		return "", err
	}
	return fmt.Sprintf("Run %s has ended without landing. Its work and Result are on %s. End now.\n", st.Run, st.Kept), nil
}

// Status says how the Runs of the owner's branch stand.
func (e *Engine) Status(ctx context.Context, dir string) (string, error) {
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
		return "", err
	}
	branch, err := e.Git.Branch(ctx, root)
	if err != nil {
		return "", err
	}
	runs, err := e.Store.All()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	shown := 0
	for i := len(runs) - 1; i >= 0 && shown < 5; i-- {
		st := runs[i]
		if st.Branch != branch && branch != "" {
			continue
		}
		shown++
		b.WriteString(e.statusLine(ctx, st))
		if !st.Open && !st.Told {
			_ = e.Store.Update(st.Run, func(s *RunState) error {
				s.Told = true
				return nil
			})
		}
	}
	if shown == 0 {
		b.WriteString("No run on this branch yet.\n")
	}
	if branch != "" {
		if tip, err := e.Git.Head(ctx, root); err == nil {
			if landing, work, ok, err := e.lastLanding(ctx, tip); err == nil && ok {
				fmt.Fprintf(&b, "\nThe last work landed here is %s", short(work))
				if n, err := e.Git.CountCommits(ctx, landing, tip); err == nil {
					fmt.Fprintf(&b, ", with %d %s since its Result", n, plural(n, "commit", "commits"))
				}
				b.WriteString(".\n")
			}
		}
	}
	return b.String(), nil
}

func (e *Engine) statusLine(ctx context.Context, st *RunState) string {
	if !st.Open {
		where := st.Kept
		if st.Landed != "" {
			where = short(st.Landed)
		}
		return fmt.Sprintf("%s  %s  %s  %s: %s\n", st.Run, st.Command, st.State, where, st.Why)
	}
	state := SessionUnknown
	if st.Session != "" {
		if s, err := e.Sessions.State(ctx, st.Session); err == nil {
			state = s
		}
	}
	line := fmt.Sprintf("%s  %s  open, session %s, %d items, %d repair rounds", st.Run, st.Command, state, len(st.Items), st.Repairs)
	if state != SessionWorking {
		line += "; it has stopped working, and `markitect " + st.Command + "` resumes it or `markitect stop WHY` ends it"
		if st.Failure != "" {
			line += " (" + st.Failure + ")"
		}
	}
	return line + "\n"
}

// Notice is the one line every command prints about a Run the owner should hear
// of: one that closed and they have not seen, or one that has stopped working.
func (e *Engine) Notice(ctx context.Context, dir string) string {
	root, err := e.Git.Root(ctx, dir)
	if err != nil || filepathIsRun(e, root) {
		return ""
	}
	branch, err := e.Git.Branch(ctx, root)
	if err != nil {
		return ""
	}
	runs, err := e.Store.All()
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, st := range runs {
		if st.Branch != branch {
			continue
		}
		switch {
		case !st.Open && !st.Told:
			if st.Landed != "" {
				fmt.Fprintf(&b, "markitect: run %s landed on %s as %s.\n", st.Run, st.Branch, short(st.Landed))
			} else {
				fmt.Fprintf(&b, "markitect: run %s ended as %s and did not land: %s. Its work is on %s.\n", st.Run, st.State, st.Why, st.Kept)
			}
			_ = e.Store.Update(st.Run, func(s *RunState) error {
				s.Told = true
				return nil
			})
		case st.Open && st.Session != "":
			if s, err := e.Sessions.State(ctx, st.Session); err == nil && s != SessionWorking {
				why := ""
				if st.Failure != "" {
					why = " (" + st.Failure + ")"
				}
				fmt.Fprintf(&b, "markitect: run %s has stopped working%s; `markitect %s` resumes it, `markitect stop WHY` ends it.\n", st.Run, why, st.Command)
			}
		}
	}
	return b.String()
}

func filepathIsRun(e *Engine, root string) bool {
	_, found, err := e.Store.Load(filepath.Base(root))
	return err == nil && found
}

// cleanUp removes the worktrees of closed Runs. The provider holds a lock on a
// worktree while its session runs, so this waits for the next apply.
func (e *Engine) cleanUp(ctx context.Context) {
	runs, err := e.Store.All()
	if err != nil {
		return
	}
	for _, st := range runs {
		if st.Open || st.Cleaned {
			continue
		}
		if st.Session != "" {
			_ = e.Sessions.Stop(ctx, st.Session)
			_ = e.Sessions.Remove(ctx, st.Session)
		}
		if err := e.Git.RemoveWorktree(ctx, worktreeBranch(st.Run)); err != nil {
			continue
		}
		if err := e.Git.DeleteBranch(ctx, worktreeBranch(st.Run)); err != nil {
			continue
		}
		_ = e.Store.Update(st.Run, func(s *RunState) error {
			s.Cleaned = true
			return nil
		})
	}
}

// loadWorking loads the project in a working tree.
func (e *Engine) loadWorking(ctx context.Context, root string) (*project.Project, error) {
	cand, err := e.Git.Candidate(ctx, root, "the state of the working tree")
	if err != nil {
		return nil, err
	}
	return project.Load(e.Git.State(ctx, cand))
}

// impact is what the change from the base to a commit reaches, with that
// commit's registry.
func (e *Engine) impact(ctx context.Context, base, commit string, before, cur *project.Project, findings []analysis.Finding, full bool) (analysis.Impact, error) {
	all, err := e.Git.Changed(ctx, base, commit)
	if err != nil {
		return analysis.Impact{}, err
	}
	return analysis.ComputeImpact(analysis.ImpactInput{Base: before, Current: cur, Changed: outsideOwnFolder(all), Findings: findings, Full: full}), nil
}

// LastBase is the commit the next Run's impact starts from, and false when
// nothing has landed yet.
func (e *Engine) LastBase(ctx context.Context, dir string) (string, bool, error) {
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
		return "", false, err
	}
	head, err := e.Git.Head(ctx, root)
	if err != nil {
		return "", false, err
	}
	return e.lastBase(ctx, head)
}

func listErrors(errs model.Diagnostics) string {
	var b strings.Builder
	for i, d := range errs {
		if i == maxRefusals {
			fmt.Fprintf(&b, "and %d more\n", len(errs)-maxRefusals)
			break
		}
		fmt.Fprintf(&b, "- %s\n", d.String())
	}
	return b.String()
}

func tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 2000 {
		return "…" + s[len(s)-2000:]
	}
	return s
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
