// Package run is the process of a Run, in the tool (draft concept 4.0): apply
// and verify open a Run and say which agent to start; the owner's own session
// starts it, and the agents work in the owner's folder and call the tool. submit
// is the one door: it says what is next each time and decides whether the work
// counts. Nothing here starts or drives an agent. Git work goes through Git, and
// what a Run knows between commands lives in a Store.
//
// Beside the Run, gate checks work done without one and without Git (DEC-096,
// DEC-097): checklist, answer and gate read the project folder, start from its
// newest certificate, and leave every Git step to whoever drives the agents.
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
	"runtime"
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
	// defaultDeadline bounds a Run.
	defaultDeadline = 6 * time.Hour
	// maxRefusals bounds how many errors one refusal lists.
	maxRefusals = 30
)

// RunBranchPrefix is where work and a Result that do not land stay.
const RunBranchPrefix = "markitect/runs/"

// runName is what a Run may be called: it names a branch and a folder.
var runName = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,62}$`)

// ErrUsage marks a request the tool cannot run, such as an unknown base.
var ErrUsage = errors.New("cannot run")

// Engine runs the commands of a Run in one repository, and the work without a
// Run in one project folder.
type Engine struct {
	// Git and Store serve the Run. Without them, no Run is open anywhere.
	Git   Git
	Store Store
	// Folder reads the project folder for the work without a Run, and Cache is
	// where that work's state lives between commands, outside the project.
	Folder Folder
	Cache  string
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
	runs, err := e.Store.All()
	if err != nil {
		return Started{}, err
	}
	if st := openIn(runs, root); st != nil {
		return stillOpen(st), nil
	}
	// Kept work lands first, while it may still be in the folder: a file that
	// already holds it stays as it is.
	e.retryLandings(ctx, runs, branch, root)
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
	st.Model = p.Manifest.Model
	if err := e.Store.Save(st); err != nil {
		return Started{}, err
	}
	return Started{Run: st.Run, Text: fmt.Sprintf("Run %s is open on %s: %d items, base %s. Its work lands on %s when the reviewer finds nothing wrong.\n\n", st.Run, branch, len(st.Items), short(st.Base), branch) + builderStep(st)}, nil
}

// Verify opens a review of the last commit. It changes nothing the owner has. The
// reviewer reads the owner's folder, so that must hold the last commit.
func (e *Engine) Verify(ctx context.Context, req Request) (Started, error) {
	root, err := e.Git.Root(ctx, req.Dir)
	if err != nil {
		return Started{}, err
	}
	branch, err := e.Git.Branch(ctx, root)
	if err != nil {
		return Started{}, err
	}
	runs, err := e.Store.All()
	if err != nil {
		return Started{}, err
	}
	if st := openIn(runs, root); st != nil {
		return stillOpen(st), nil
	}
	if dirty, err := e.Git.Uncommitted(ctx, root); err != nil {
		return Started{}, err
	} else if len(dirty) > 0 {
		return Started{State: StateCheckFailed, Text: "The reviewer reads this folder, and it holds changes that are not committed. Commit or remove them first:\n\n- " + strings.Join(dirty, "\n- ") + "\n"}, nil
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
	st.Model = cur.Manifest.Model
	if err := e.Store.Save(st); err != nil {
		return Started{}, err
	}
	return Started{Run: st.Run, Text: fmt.Sprintf("Run %s is open: a review of %d items at %s. It changes nothing you have.\n\n", st.Run, len(st.Items), short(start)) + verifyStep(st)}, nil
}

// stillOpen answers an apply or verify while a Run of the folder is open: only one
// can be, since its agents work in the folder.
func stillOpen(st *RunState) Started {
	text := fmt.Sprintf("Run %s (%s) is still open in this folder, so no new one starts. To go on with it:\n\n", st.Run, st.Command)
	if st.Command == "verify" {
		text += verifyStep(st)
	} else {
		text += builderStep(st)
	}
	return Started{State: "open", Run: st.Run, Text: text + "\nTo end it instead: markitect stop \"why\"\n"}
}

// reviewerFile and builderFile are the harness's agents, without which no Run can
// finish.
const (
	reviewerFile = ".claude/agents/" + ReviewerAgent + ".md"
	builderFile  = ".claude/agents/" + BuilderAgent + ".md"
)

const noHarness = "The harness is not complete in this project: a run needs the builder and reviewer agents, and the hook on answer in " + SettingsFile + ". Run markitect init, which commits what it writes, and try again; if init says it left the settings as they are, add the hook it names by hand.\n"

// hasHarness looks in the commit and on disk for both agents of a Run and for the
// hook that lets only the reviewer answer: without it, nothing would keep the
// builder from answering for itself.
func (e *Engine) hasHarness(ctx context.Context, commit, root string) bool {
	for _, file := range []string{reviewerFile, builderFile, SettingsFile} {
		data, err := e.Git.State(ctx, commit).Read(file)
		if err != nil {
			if data, err = os.ReadFile(filepath.Join(root, filepath.FromSlash(file))); err != nil {
				return false
			}
		}
		if file == SettingsFile && !HasGuard(data) {
			return false
		}
	}
	return true
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
		Items: items, Answers: map[string]Answer{}, Impact: impactKeys(imp, items), Notes: imp.Notes,
		Began: now, Deadline: now.Add(limit), Limit: limit, Open: true}, nil
}

func (e *Engine) runName(now time.Time) (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "r" + now.UTC().Format("20060102-150405") + "-" + hex.EncodeToString(b), nil
}

// Current is the Run of the folder that holds dir: the open one, or else the last
// one, so that an agent of a closed Run hears that it is over.
func (e *Engine) Current(ctx context.Context, dir string) (*RunState, error) {
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
		return nil, err
	}
	runs, err := e.Store.All()
	if err != nil {
		return nil, err
	}
	if st := openIn(runs, root); st != nil {
		return st, nil
	}
	var last *RunState
	for _, st := range runs {
		if samePath(st.Repo, root) && (last == nil || st.Began.After(last.Began)) {
			last = st
		}
	}
	if last == nil {
		return nil, fmt.Errorf("%w: no run is open here; markitect apply opens one", ErrUsage)
	}
	return last, nil
}

// openIn is the open Run of a folder.
func openIn(runs []*RunState, root string) *RunState {
	for _, st := range runs {
		if st.Open && samePath(st.Repo, root) {
			return st
		}
	}
	return nil
}

// samePath compares two folders, without regard to case on Windows.
func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// runChecklist prints the builder's task and to-do list of a Run, rebuilt from the
// current files and registry. With review it prints what the review must answer.
func (e *Engine) runChecklist(ctx context.Context, dir string, review bool) (string, error) {
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

// Answer stores a reviewer's answers: in a folder with an open Run against the
// Run's state under review, otherwise against the folder's state (answerState).
// What it cannot accept is refused at once, with what is still missing.
func (e *Engine) Answer(ctx context.Context, dir string, raw []byte) (string, bool, error) {
	if open, err := e.openRun(ctx, dir); err != nil {
		return "", false, err
	} else if open == nil {
		return e.answerState(dir, raw)
	}
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
	st, err := e.Current(ctx, dir)
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
	state, decisions := StateStopped, []OwnerDecision{{Matter: why, Recommendation: "If the spec is the cause, settle it there and run apply again."}}
	if environment {
		state, decisions = StateIncomplete, nil
	}
	if err := e.close(ctx, st, state, "stopped: "+why, decisions); err != nil {
		return "", err
	}
	return fmt.Sprintf("Run %s has ended without landing. Its work and Result are on %s.%s End now.\n", st.Run, st.Kept, inFolder(st)), nil
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
	if openIn(runs, root) == nil && e.Folder != nil {
		b.WriteString(e.stateStatus(dir))
	}
	return b.String(), nil
}

// openRun is the open Run of the folder that holds dir, or nil. Without Git there
// is none.
func (e *Engine) openRun(ctx context.Context, dir string) (*RunState, error) {
	if e.Git == nil {
		return nil, nil
	}
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
		return nil, err
	}
	runs, err := e.Store.All()
	if err != nil {
		return nil, err
	}
	return openIn(runs, root), nil
}

func (e *Engine) statusLine(ctx context.Context, st *RunState) string {
	if !st.Open {
		where := st.Kept
		if st.Landed != "" {
			where = short(st.Landed)
		}
		return fmt.Sprintf("%s  %s  %s  %s: %s\n", st.Run, st.Command, st.State, where, st.Why)
	}
	return fmt.Sprintf("%s  %s  open, %d items, %d repair rounds; `markitect %s` says how to go on, and `markitect stop WHY` ends it\n", st.Run, st.Command, len(st.Items), st.Repairs, st.Command)
}

// Notice is the one line every command prints about a Run the owner should hear
// of: one that closed and they have not seen.
func (e *Engine) Notice(ctx context.Context, dir string) string {
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
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
		if !st.Open && !st.Told {
			if st.Landed != "" {
				fmt.Fprintf(&b, "markitect: run %s landed on %s as %s.\n", st.Run, st.Branch, short(st.Landed))
			} else {
				fmt.Fprintf(&b, "markitect: run %s ended as %s and did not land: %s. Its work is on %s.\n", st.Run, st.State, st.Why, st.Kept)
			}
			_ = e.Store.Update(st.Run, func(s *RunState) error {
				s.Told = true
				return nil
			})
		}
	}
	return b.String()
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

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
