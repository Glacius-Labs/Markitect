package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

type notesReadingSessions struct {
	controlled *controlledManagers
	expected   []run.Note
	mu         sync.Mutex
	read       map[string][]run.Note
}

type accountingRepairSessions struct {
	controlled               *controlledManagers
	child                    *accountingRepairSession
	accounting               bool
	omitContinuationJudgment bool
}

type accountingRepairSession struct {
	run.Session
	spec                     run.SessionSpec
	attempts, assignments    int
	accounting               bool
	question                 string
	omitContinuationJudgment bool
	judgments                int
}

func (s *accountingRepairSessions) Open(ctx context.Context, spec run.SessionSpec) (run.Session, error) {
	if spec.Manager.Package != "a/deep" {
		return s.controlled.Open(ctx, spec)
	}
	child := &accountingRepairSession{spec: spec, accounting: s.accounting, omitContinuationJudgment: s.omitContinuationJudgment}
	await := spec.Tools.Await
	spec.Tools.Await = func(ctx context.Context) (run.Event, error) {
		event, err := await(ctx)
		if event.Assignment != nil {
			child.assignments++
		}
		return event, err
	}
	var err error
	child.Session, err = s.controlled.Open(ctx, spec)
	s.child = child
	return child, err
}

func (s *accountingRepairSession) Exchange(ctx context.Context, message run.Message) (run.Response, error) {
	if strings.HasPrefix(message.Task, "Answer pending Manager questions") {
		return s.Session.Exchange(ctx, message)
	}
	s.attempts++
	root := s.spec.Workspace.Location().Directory
	if s.attempts == 2 {
		if strings.Contains(message.Task, "Receive your parent assignment with Await") || !strings.Contains(message.Task, "active assignment") {
			return run.Response{}, fmt.Errorf("candidate repair replayed consumed assignment: %s", message.Task)
		}
		if s.spec.Mandate != run.Implement && strings.Contains(strings.ToLower(message.Task), "repair the candidate") {
			return run.Response{}, fmt.Errorf("review continuation requests implementation: %s", message.Task)
		}
		text := fmt.Sprint(message.Feedback)
		if s.accounting {
			if !strings.Contains(text, "file.unexplained") || !strings.Contains(text, "out/unregistered.txt") || !strings.Contains(text, "neither a registered artifact") {
				return run.Response{}, fmt.Errorf("candidate repair lost actionable accounting diagnostics: %s", text)
			}
			registry := filepath.Join(root, ".markitect/registry/a/deep/registry.yaml")
			data, err := os.ReadFile(registry)
			if err != nil {
				return run.Response{}, err
			}
			data = append(data, []byte("  - path: out/unregistered.txt\n    form: code\n    realizes: [a/deep/item]\n    why: Register the actual child artifact after host rejection.\n")...)
			if err := os.WriteFile(registry, data, 0600); err != nil {
				return run.Response{}, err
			}
		} else {
			if !strings.Contains(text, "Resolve pending question "+s.question) {
				return run.Response{}, fmt.Errorf("continuation lost pending question Findings: %s", text)
			}
			event, err := s.spec.Tools.Await(ctx)
			if err != nil || event.Answer == nil || event.Answer.ID != s.question {
				return run.Response{}, fmt.Errorf("continuation did not resolve the actual pending question: %v", err)
			}
			// Match the retained lending exchange: the previous response judged
			// the assignment, while this response only consumes its pending Answer.
			// The scripted consumer judges again only when the host expressly asks
			// for a fresh judgment, even for question-only continuation feedback.
			fresh := strings.Contains(message.Task, "fresh mandate-appropriate judgment") && strings.Contains(message.Task, "even if they only resolve pending questions or answers")
			if !fresh || s.omitContinuationJudgment {
				return run.Response{}, nil
			}
		}
	}
	response, err := s.Session.Exchange(ctx, message)
	if response.Decision != "" {
		s.judgments++
	}
	if err == nil && s.attempts == 1 {
		if s.accounting {
			err = os.WriteFile(filepath.Join(root, "out/unregistered.txt"), []byte("child artifact\n"), 0600)
		} else {
			// The direct parent is already awaiting this child's handoff and
			// handles incoming questions while waiting. Leave its answer pending.
			s.question, err = s.spec.Tools.Ask(ctx, "a", "Explain this observation before I finish.")
		}
	}
	return response, err
}

func TestRejectedChildCandidateRepairsActiveAssignmentAndReachesParent(t *testing.T) {
	for _, scenario := range []struct {
		mandate                  run.Mandate
		accounting               bool
		omitContinuationJudgment bool
	}{{run.Implement, true, false}, {run.Implement, false, false}, {run.Assessment, false, false}, {run.Verify, false, false}, {run.Implement, false, true}, {run.Assessment, false, true}, {run.Verify, false, true}} {
		t.Run(fmt.Sprintf("%s/accounting=%v/omit=%v", scenario.mandate, scenario.accounting, scenario.omitContinuationJudgment), func(t *testing.T) {
			_, prepared, git, input := runFixture(t)
			records, err := runstore.New(git.StateRoot)
			if err != nil {
				t.Fatal(err)
			}
			sessions := &accountingRepairSessions{controlled: &controlledManagers{crossed: make(chan struct{}), noChange: true, acceptedRepair: true}, accounting: scenario.accounting, omitContinuationJudgment: scenario.omitContinuationJudgment}
			request := run.Request{ID: "accounting-repair", Project: "fixture", Prepared: prepared, Input: input, Mandate: scenario.mandate, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 100), MessageTimeout: 30 * time.Second, CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
			result, err := (&run.Executor{Sessions: sessions, Workspaces: git, Records: records}).Execute(context.Background(), request)
			if scenario.omitContinuationJudgment {
				if err == nil || !strings.Contains(err.Error(), "Manager a/deep supplied no semantic result") || result.Technical == run.Completed || result.Candidate != nil || sessions.child.judgments != 1 {
					t.Fatal("an old judgment was reused for an empty active response", result.Technical, err, sessions.child.judgments)
				}
				for _, closure := range result.Cleanup {
					if !closure.Closed {
						t.Fatal("empty active response left a live resource", closure)
					}
				}
				return
			}
			if err != nil || result.Technical != run.Completed || result.Candidate == nil {
				t.Fatalf("child repair did not complete: %s %v", result.Technical, err)
			}
			if sessions.child.attempts != 2 || sessions.child.assignments != 1 || sessions.child.judgments != 2 || len(result.Handoffs) != 5 || len(sessions.controlled.sessions) != 5 {
				t.Fatal("repair replaced the Session/assignment or lost the full hierarchy", sessions.child.attempts, sessions.child.assignments, len(result.Handoffs))
			}
			// Ordinary post-handoff question replies return an empty Response in
			// controlledManager. They must still complete without another judgment.
			if sessions.controlled.receiptCalls < 4 {
				t.Fatal("fixed handoff receipt-only wakeups were not exercised", sessions.controlled.receiptCalls)
			}
			for _, closure := range result.Cleanup {
				if !closure.Closed {
					t.Fatal("repair left a live resource", closure)
				}
			}
			found := false
			for _, handoff := range result.Handoffs {
				if handoff.Manager == "a" {
					for _, child := range handoff.Children {
						found = found || child.Manager == "a/deep"
					}
				}
			}
			if !found && scenario.mandate == run.Implement {
				t.Fatal("parent did not review and integrate the repaired child")
			}
			destination := t.TempDir()
			if err := git.Reopen(context.Background(), *result.Candidate, destination); err != nil {
				t.Fatal(err)
			}
			if scenario.accounting {
				if data, err := os.ReadFile(filepath.Join(destination, "out/unregistered.txt")); err != nil || string(data) != "child artifact\n" {
					t.Fatal("repaired artifact did not survive parent integration and closure", err)
				}
			}
			if scenario.mandate != run.Implement && (result.Decision != run.Approve || result.Candidate.Commit != input.Commit || result.Candidate.Tree != input.Tree) {
				t.Fatal("review continuation changed its judgment mandate or fixed candidate", result.Decision, result.Candidate)
			}
		})
	}
}

type notesReadingSession struct {
	run.Session
	owner   *notesReadingSessions
	spec    run.SessionSpec
	once    sync.Once
	readErr error
}

func (s *notesReadingSessions) Open(ctx context.Context, spec run.SessionSpec) (run.Session, error) {
	session, err := s.controlled.Open(ctx, spec)
	if session == nil {
		return nil, err
	}
	return &notesReadingSession{Session: session, owner: s, spec: spec}, err
}
func (s *notesReadingSession) Exchange(ctx context.Context, message run.Message) (run.Response, error) {
	s.once.Do(func() {
		notes, err := s.spec.Tools.ReadNotes(ctx)
		if err != nil {
			s.readErr = err
			return
		}
		var earlier []run.Note
		for _, note := range notes {
			if note.Run == s.owner.expected[0].Run {
				earlier = append(earlier, note)
			}
		}
		if !reflect.DeepEqual(earlier, s.owner.expected) {
			s.readErr = fmt.Errorf("Manager %s did not read the exact earlier Run Notes", s.spec.Manager.Package)
			return
		}
		s.owner.mu.Lock()
		s.owner.read[s.spec.Manager.Package] = earlier
		s.owner.mu.Unlock()
	})
	if s.readErr != nil {
		return run.Response{}, s.readErr
	}
	return s.Session.Exchange(ctx, message)
}

func TestFreshRunManagersReadEarlierClosedRunNotesThroughActualTool(t *testing.T) {
	_, prepared, git, input := runFixture(t)
	ctx := context.Background()
	firstStore, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	firstSessions := &controlledManagers{crossed: make(chan struct{})}
	request := run.Request{ID: "notes-first", Project: "fixture", Prepared: prepared, Input: input, Mandate: run.Assessment, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
	first, err := (&run.Executor{Sessions: firstSessions, Workspaces: git, Records: firstStore}).Execute(ctx, request)
	if err != nil || first.Technical != run.Completed || first.Decision != run.Approve || first.Candidate == nil {
		t.Fatal(first.Technical, first.Decision, err)
	}
	for _, session := range firstSessions.sessions {
		if !session.closed {
			t.Fatal("first Run Session still live")
		}
		if _, err := os.Stat(session.spec.Workspace.Location().Directory); !os.IsNotExist(err) {
			t.Fatal("first Run copy survived closure", err)
		}
	}
	// New instances share only durable state, never a Session or workspace.
	secondStore, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := secondStore.Notes(ctx, "fixture")
	if err != nil || len(expected) != 5 {
		t.Fatal("first Run Notes unavailable after closure", len(expected), err)
	}
	writers := map[string]bool{}
	for _, note := range expected {
		if note.ID == "" || note.Project != "fixture" || note.Run != first.ID || note.Context != "controlled test" || note.Text != note.Manager+" participated" || writers[note.Manager] {
			t.Fatal("first Run Note provenance changed", note)
		}
		writers[note.Manager] = true
	}
	for _, manager := range prepared.Topology().Managers() {
		if !writers[manager.Package] {
			t.Fatal("Manager did not append its attributed Note", manager.Package)
		}
	}
	secondSessions := &notesReadingSessions{controlled: &controlledManagers{crossed: make(chan struct{})}, expected: expected, read: map[string][]run.Note{}}
	request.ID, request.Mandate, request.Input = "notes-second", run.Verify, *first.Candidate
	request.Allowance = run.NewAllowance(time.Now().Add(time.Minute), 100)
	secondGit := &LocalGit{StateRoot: git.StateRoot, TempRoot: git.TempRoot}
	second, err := (&run.Executor{Sessions: secondSessions, Workspaces: secondGit, Records: secondStore}).Execute(ctx, request)
	if err != nil || second.Technical != run.Completed || second.Decision != run.Approve {
		t.Fatal(second.Technical, second.Decision, err)
	}
	secondSessions.mu.Lock()
	defer secondSessions.mu.Unlock()
	if len(secondSessions.read) != 5 {
		t.Fatal("fresh complete hierarchy did not read earlier Notes", len(secondSessions.read))
	}
	for _, manager := range prepared.Topology().Managers() {
		if !reflect.DeepEqual(secondSessions.read[manager.Package], expected) {
			t.Fatal("actual ReadNotes lost earlier provenance", manager.Package)
		}
	}
	fresh, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	all, err := fresh.Notes(ctx, "fixture")
	if err != nil || len(all) != 10 {
		t.Fatal("later Run overwrote earlier Notes", len(all), err)
	}
	var retained []run.Note
	for _, note := range all {
		if note.Run == first.ID {
			retained = append(retained, note)
		}
	}
	if !reflect.DeepEqual(retained, expected) {
		t.Fatal("earlier durable Notes changed")
	}
}

func runFixture(t *testing.T, configure ...func(*testkit.Repo)) (*testkit.Repo, project.Prepared, *LocalGit, project.CandidateRef) {
	t.Helper()
	r := testkit.NewRepo(t)
	r.Git("switch", "-c", "codex/run-fixture")
	core := "../../../docs/design/canonical-model/minimal-language/exchange-example/dependencies/core"
	if err := fs.WalkDir(os.DirFS(core), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(filepath.Join(core, filepath.FromSlash(p)))
		if err == nil {
			r.Write(".markitect/dependencies/core/"+p, string(data))
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	r.Write(".markitect/model.yaml", "formatVersion: 1\nid: run-fixture\nroot: model\ndependencies: {markitect-core: '1.0.0'}\n")
	r.Write(".markitect/model-lock.yaml", "formatVersion: 1\nmodels:\n  markitect-core: {version: '1.0.0', digest: 'sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1'}\n")
	r.Write(".markitect/model/a/thing.yaml", "imports: [{package: markitect/core, as: Core}]\nid: Thing\nkind: Core.Kind\nbody: A tiny realization obligation.\nspec: {}\n")
	r.Write(".markitect/model/a/deep/item.yaml", "imports: [{package: a}]\nid: item\nkind: a/Thing\nbody: A child artifact.\nspec: {}\n")
	r.Write(".markitect/execution.yaml", "apiVersion: markitect.execution/v2\nchecks: []\n")
	r.Write(".markitect/registry/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts: []\n")
	for _, a := range []struct{ owner, path, definition string }{{"a", "out/a.txt", "a/Thing"}, {"a/deep", "out/deep.txt", "a/deep/item"}} {
		r.Write(a.path, "initial\n")
		r.Write(".markitect/registry/"+a.owner+"/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts:\n  - path: "+a.path+"\n    form: code\n    realizes: ["+a.definition+"]\n    why: Tiny controlled realization.\n")
	}
	r.Write(".gitignore", "*.ignored\n")
	r.Write(".markitect/ignore.yaml", "apiVersion: markitect.ignore/v1\nfiles:\n  - path: .gitignore\n    reason: Git fixture exclusions do not supply accounting.\n")
	r.Write(".markitect/registry/a/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts:\n  - path: out/a.txt\n    form: code\n    realizes: [a/Thing]\n    why: Controlled parent artifact.\n  - path: out/accounted.ignored\n    form: code\n    realizes: [a/Thing]\n    why: Accounted Gitignored untracked input must survive capture.\n")
	// Git permits explicit empty trees; the index does not. The fixture commits
	// the empty Package as an actual tree, just as the production snapshot does.
	for _, configure := range configure {
		configure(r)
	}
	r.Commit("committed fixture")
	ctx := context.Background()
	tree := r.Git("rev-parse", "HEAD^{tree}")
	tree, err := withDirectories(ctx, r.Dir, tree, []string{".markitect/model/b/empty"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := localCommand(ctx, r.Dir, []byte("Retain complete Package inventory\n"), "commit-tree", tree, "-p", r.Git("rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	r.Git("reset", "--mixed", id)
	if err = os.MkdirAll(filepath.Join(r.Dir, ".markitect/model/b/empty"), 0700); err != nil {
		t.Fatal(err)
	}
	r.Write("out/accounted.ignored", "original ignored input\n")
	selection := project.InputSelection{Repository: r.Dir, Local: project.ModelSelection{Boundary: ".", ManifestPath: fixturePath(t, ".markitect/model.yaml")}, Dependencies: []project.ModelSelection{{Boundary: ".markitect/dependencies/core", ManifestPath: fixturePath(t, "model.yaml")}}}
	repo := adapter(t)
	prepared, err := project.NewPreparationService(project.NewModelService(repo, repo)).Capture(ctx, project.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	attempt, ok := prepared.Prepared()
	if !ok {
		t.Fatalf("fixture preparation=%s attempts=%+v", prepared.Status(), prepared.Attempts())
	}
	p, err := project.Prepare(attempt)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(p.Topology().Managers()); got != 5 {
		t.Fatalf("expected Root, two children, grandchild and empty Package: %d", got)
	}
	g := &LocalGit{StateRoot: t.TempDir(), TempRoot: t.TempDir()}
	ref, err := g.Capture(ctx, "fixture", p)
	if err != nil {
		t.Fatal(err)
	}
	return r, p, g, ref
}

type controlledManagers struct {
	mu                  sync.Mutex
	sessions            []*controlledManager
	crossed             chan struct{}
	asked               int
	corrections         int
	failOpen, failClose bool
	lateQuestions       int
	receiptCalls        int
	acceptedRepair      bool
	forceDecision       run.Decision
	lateReject          bool
	badManifest         bool
	didLateReject       bool
	observation         *run.SessionObservation
	exchangeBlock       <-chan struct{}
	closeBlock          <-chan struct{}
	noChange            bool
}
type controlledManager struct {
	owner  *controlledManagers
	spec   run.SessionSpec
	calls  int
	closed bool
}

func (c *controlledManagers) Open(_ context.Context, s run.SessionSpec) (run.Session, error) {
	m := &controlledManager{owner: c, spec: s}
	c.mu.Lock()
	c.sessions = append(c.sessions, m)
	c.mu.Unlock()
	if c.failOpen && s.Manager.Package == "a" {
		return m, fmt.Errorf("injected partial session startup")
	}
	if c.observation != nil {
		return observedControlledSession{Session: m, value: *c.observation}, nil
	}
	return m, nil
}

type observedControlledSession struct {
	run.Session
	value run.SessionObservation
}

func (s observedControlledSession) Observation() run.SessionObservation { return s.value }
func (m *controlledManager) Close(context.Context) error {
	if m.owner.closeBlock != nil && m.spec.Manager.Package == "a" {
		<-m.owner.closeBlock
	}
	m.closed = true
	if m.owner.failClose && m.spec.Manager.Package == "a" {
		return fmt.Errorf("injected uncertain closure")
	}
	return nil
}
func (m *controlledManager) Exchange(ctx context.Context, message run.Message) (run.Response, error) {
	m.calls++
	tools := m.spec.Tools
	if m.owner.exchangeBlock != nil && m.spec.Manager.Package == "." {
		<-m.owner.exchangeBlock
	}
	if strings.HasPrefix(message.Task, "Answer pending Manager questions") {
		event, err := tools.Await(ctx)
		if err != nil {
			return run.Response{}, err
		}
		if event.Answer != nil {
			return run.Response{}, nil
		}
		if event.Question == nil {
			return run.Response{}, fmt.Errorf("receipt message lost event")
		}
		if event.Question.Text == "Reassess the already accepted handoff." {
			if err = tools.Reply(ctx, event.Question.ID, "late required correction"); err != nil {
				return run.Response{}, err
			}
			return run.Response{Decision: run.Reject, Findings: []run.Finding{{Text: "Late required correction discovered after accepted integration."}}}, nil
		}
		if err = tools.Reply(ctx, event.Question.ID, "controlled answer from "+m.spec.Manager.Package); err != nil {
			return run.Response{}, err
		}
		m.owner.mu.Lock()
		m.owner.receiptCalls++
		m.owner.mu.Unlock()
		return run.Response{}, nil
	}
	if m.spec.Manager.Package != "." && m.calls == 1 {
		for {
			e, err := tools.Await(ctx)
			if err != nil {
				return run.Response{}, err
			}
			if e.Question != nil {
				if err = tools.Reply(ctx, e.Question.ID, "controlled answer from "+m.spec.Manager.Package); err != nil {
					return run.Response{}, err
				}
			}
			if e.Assignment != nil {
				message.Feedback = e.Assignment.Feedback
				break
			}
		}
	}
	if m.calls == 1 {
		for _, child := range m.spec.Manager.Children {
			if err := tools.Delegate(ctx, child, "Realize or review your own modeled responsibility.", nil); err != nil {
				return run.Response{}, err
			}
		}
	}
	var pending []*run.Handoff
	if m.calls == 1 && (m.spec.Manager.Package == "a" || m.spec.Manager.Package == "b") {
		other := "a"
		if m.spec.Manager.Package == "a" {
			other = "b"
		}
		id, err := tools.Ask(ctx, other, "What is your controlled observation?")
		if err != nil {
			return run.Response{}, err
		}
		m.owner.mu.Lock()
		m.owner.asked++
		if m.owner.asked == 2 {
			close(m.owner.crossed)
		}
		m.owner.mu.Unlock()
		select {
		case <-m.owner.crossed:
		case <-ctx.Done():
			return run.Response{}, ctx.Err()
		}
		for {
			e, err := tools.Await(ctx)
			if err != nil {
				return run.Response{}, err
			}
			if e.Question != nil {
				if err = tools.Reply(ctx, e.Question.ID, "controlled answer from "+m.spec.Manager.Package); err != nil {
					return run.Response{}, err
				}
			}
			if e.Handoff != nil {
				pending = append(pending, e.Handoff)
			}
			if e.Answer != nil && e.Answer.ID == id {
				if e.Answer.Text != "controlled answer from "+other {
					return run.Response{}, fmt.Errorf("wrong correlated answer")
				}
				break
			}
		}
	}
	remaining := 0
	if m.calls == 1 {
		remaining = len(m.spec.Manager.Children)
	}
	for remaining > 0 {
		var h *run.Handoff
		if len(pending) > 0 {
			h = pending[0]
			pending = pending[1:]
		} else {
			e, err := tools.Await(ctx)
			if err != nil {
				return run.Response{}, err
			}
			if e.Question != nil {
				if err = tools.Reply(ctx, e.Question.ID, "controlled answer from "+m.spec.Manager.Package); err != nil {
					return run.Response{}, err
				}
			}
			h = e.Handoff
			if h == nil {
				continue
			}
		}
		// The child already supplied a fixed handoff. Its same Session must
		// remain reachable through an ordinary follow-up message.
		id, err := tools.Ask(ctx, h.Manager, "Explain your completed fixed handoff.")
		if err != nil {
			return run.Response{}, err
		}
		for {
			event, err := tools.Await(ctx)
			if err != nil {
				return run.Response{}, err
			}
			if event.Handoff != nil {
				pending = append(pending, event.Handoff)
			}
			if event.Question != nil {
				if err = tools.Reply(ctx, event.Question.ID, "controlled answer from "+m.spec.Manager.Package); err != nil {
					return run.Response{}, err
				}
			}
			if event.Answer != nil && event.Answer.ID == id {
				if event.Answer.Text != "controlled answer from "+h.Manager {
					return run.Response{}, fmt.Errorf("late answer mismatch")
				}
				break
			}
		}
		m.owner.mu.Lock()
		m.owner.lateQuestions++
		m.owner.mu.Unlock()
		if m.spec.Mandate == run.Implement && h.Manager == "a/deep" && strings.Contains(h.Diff, "+wrong") {
			finding := []run.Finding{{Text: "The complete fixed diff shows wrong; correct the child artifact."}}
			if err := tools.Review(ctx, h.Manager, h.Result.ID, run.Reject, finding); err != nil {
				return run.Response{}, err
			}
			if err := tools.Delegate(ctx, h.Manager, "Correct the reviewed child artifact.", finding); err != nil {
				return run.Response{}, err
			}
			m.owner.mu.Lock()
			m.owner.corrections++
			m.owner.mu.Unlock()
			continue
		}
		if err := tools.Review(ctx, h.Manager, h.Result.ID, run.Approve, nil); err != nil {
			return run.Response{}, err
		}
		if m.owner.lateReject && h.Manager == "a/deep" && !m.owner.didLateReject {
			m.owner.didLateReject = true
			id, err := tools.Ask(ctx, h.Manager, "Reassess the already accepted handoff.")
			if err != nil {
				return run.Response{}, err
			}
			for {
				event, err := tools.Await(ctx)
				if err != nil {
					return run.Response{}, err
				}
				if event.Question != nil {
					if err = tools.Reply(ctx, event.Question.ID, "controlled answer from "+m.spec.Manager.Package); err != nil {
						return run.Response{}, err
					}
				}
				if event.Handoff != nil {
					pending = append(pending, event.Handoff)
				}
				if event.Answer != nil && event.Answer.ID == id {
					if event.Answer.Text != "late required correction" {
						return run.Response{}, fmt.Errorf("late finding not actually delivered")
					}
					break
				}
			}
			findings := []run.Finding{{Text: "Correct the actual late required finding."}}
			if err = tools.Delegate(ctx, h.Manager, "Correct late rejection.", findings); err != nil {
				return run.Response{}, err
			}
			m.owner.corrections++
			continue
		}
		if m.spec.Mandate == run.Implement && h.Manager == "a/deep" && !m.owner.acceptedRepair && !m.owner.lateReject {
			m.owner.acceptedRepair = true
			if err := os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, "out/deep.txt"), []byte("parent revision\n"), 0600); err != nil {
				return run.Response{}, err
			}
			findings := []run.Finding{{Text: "Correct accepted child work on the integrated parent baseline."}}
			if err := tools.Delegate(ctx, h.Manager, "Repair after parent integration.", findings); err != nil {
				return run.Response{}, err
			}
			m.owner.mu.Lock()
			m.owner.corrections++
			m.owner.mu.Unlock()
			continue
		}
		remaining--
	}
	if _, err := tools.AppendNote(ctx, "controlled test", m.spec.Manager.Package+" participated"); err != nil {
		return run.Response{}, err
	}
	if m.spec.Mandate == run.Implement && m.owner.badManifest && m.spec.Manager.Package == "." {
		entry, found := m.spec.Prepared.Attempt().Model().Inputs().Raw.Census.Lookup(fixturePathForControlled(".markitect/model.yaml"))
		if !found {
			return run.Response{}, fmt.Errorf("captured manifest absent")
		}
		data := entry.Bytes()
		if m.calls == 1 {
			data = []byte("formatVersion: invalid\nid: run-fixture\n")
		}
		if err := os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, ".markitect/model.yaml"), data, 0600); err != nil {
			return run.Response{}, err
		}
	}
	if m.spec.Mandate == run.Implement {
		if m.owner.noChange {
			return run.Response{Decision: run.Submitted}, nil
		}
		path, value := "", ""
		if m.spec.Manager.Package == "a" {
			path, value = "out/a.txt", "parent result\n"
		}
		if m.spec.Manager.Package == "a/deep" {
			path, value = "out/deep.txt", "wrong\n"
			if len(message.Feedback) > 0 {
				value = "corrected\n"
				if message.Task == "Correct late rejection." {
					value = "late corrected\n"
				}
				if message.Task == "Repair after parent integration." {
					value = "repaired after integration\n"
				}
			}
		}
		if path != "" {
			if err := os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, filepath.FromSlash(path)), []byte(value), 0600); err != nil {
				return run.Response{}, err
			}
		}
		return run.Response{Decision: run.Submitted}, nil
	}
	// Review scratch is deliberately invalid product content; it must never
	// replace the fixed input candidate returned by Assessment or Verify.
	if err := os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, "review-scratch.txt"), []byte("scratch"), 0600); err != nil {
		return run.Response{}, err
	}
	if m.owner.forceDecision != "" {
		return run.Response{Decision: m.owner.forceDecision, Findings: []run.Finding{{Text: "controlled semantic rejection"}}}, nil
	}
	return run.Response{Decision: run.Approve}, nil
}

func TestCompleteHierarchyUsesRealGitAndClosedDurableStateForEveryMandate(t *testing.T) {
	r, p, g, ref := runFixture(t)
	ctx := context.Background()
	// The initial input also reopens independently with the exact empty Package.
	reopened := t.TempDir()
	if err := g.Reopen(ctx, ref, reopened); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(reopened, ".markitect/model/b/empty")); err != nil {
		t.Fatal("initial input lost empty Package:", err)
	}
	for _, mandate := range []run.Mandate{run.Assessment, run.Implement, run.Verify} {
		t.Run(string(mandate), func(t *testing.T) {
			store, err := runstore.New(g.StateRoot)
			if err != nil {
				t.Fatal(err)
			}
			managers := &controlledManagers{crossed: make(chan struct{})}
			executor := run.Executor{Sessions: managers, Workspaces: g, Records: store}
			request := run.Request{MaxManagers: 5, ID: string(mandate), Project: "fixture", Prepared: p, Input: ref, Mandate: mandate, Allowance: run.NewAllowance(time.Now().Add(45*time.Second), 80), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
			result, err := executor.Execute(ctx, request)
			if err != nil || result.Technical != run.Completed {
				t.Fatalf("%+v error=%v", result, err)
			}
			if len(result.Handoffs) != 5 || len(result.Cleanup) != 12 || managers.asked != 2 || managers.lateQuestions < 4 || managers.receiptCalls < 4 {
				t.Fatalf("incomplete hierarchy/closure/questions: %+v", result)
			}
			for _, s := range managers.sessions {
				if !s.closed {
					t.Fatal("live returned session")
				}
				if _, err := os.Stat(s.spec.Workspace.Location().Directory); !os.IsNotExist(err) {
					t.Fatal("copy survived Execute")
				}
			}
			fresh, _ := runstore.New(g.StateRoot)
			saved, err := fresh.Read(ctx, "fixture", string(mandate))
			if err != nil || saved.Technical != run.Completed {
				t.Fatal(saved, err)
			}
			notes, err := fresh.Notes(ctx, "fixture")
			if err != nil || len(notes) < 5 {
				t.Fatal("Notes unavailable from new instance", notes, err)
			}
			candidate := *result.Candidate
			destination := t.TempDir()
			if err = g.Reopen(ctx, candidate, destination); err != nil {
				t.Fatal(err)
			}
			if testkit.Git(t, destination, "rev-parse", "HEAD") != candidate.Commit || testkit.Git(t, destination, "rev-parse", "HEAD^{tree}") != candidate.Tree {
				t.Fatal("durable object identity changed")
			}
			if data, err := os.ReadFile(filepath.Join(destination, "out/accounted.ignored")); err != nil || string(data) != "original ignored input\n" {
				t.Fatal("accounted ignored input lost", err)
			}
			if mandate == run.Implement {
				data, err := os.ReadFile(filepath.Join(destination, "out/deep.txt"))
				if err != nil || string(data) != "repaired after integration\n" || managers.corrections != 2 {
					t.Fatal("reviewed correction missing", string(data), err)
				}
				for _, h := range result.Handoffs {
					for _, i := range h.Children {
						if _, err := localCommand(ctx, destination, nil, "merge-base", "--is-ancestor", i.Child.ID, candidate.Commit); err != nil {
							t.Fatal("accepted child not integrated exactly", err)
						}
					}
				}
			} else if candidate.Commit != ref.Commit || candidate.Tree != ref.Tree {
				t.Fatal("review scratch became product")
			}
		})
	}
	// Retained ancestry remains self-contained after the original source closes.
	if err := os.RemoveAll(r.Dir); err != nil {
		t.Fatal(err)
	}
	if err := g.Reopen(ctx, ref, t.TempDir()); err != nil {
		t.Fatal("candidate depended on original source", err)
	}
}

func TestDurableCandidateRetentionAndReopenUnderLongPaths(t *testing.T) {
	r, prepared, original, input := runFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	source := t.TempDir()
	if err := original.Reopen(ctx, input, source); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for len(root) < 170 {
		root = filepath.Join(root, "long-state-path")
	}
	g := &LocalGit{StateRoot: filepath.Join(root, "state"), TempRoot: filepath.Join(root, "temporary")}
	if err := os.MkdirAll(g.TempRoot, 0700); err != nil {
		t.Fatal(err)
	}
	// Native Run retention supplies a bundle to this same host-side store. Use
	// actual Git to make that bundle without a provider or container, then run
	// the real independent bare fetch/validation before durable publication.
	fixed := run.Commit{ID: input.Commit, Tree: input.Tree}
	retained, err := g.storeBundle(ctx, "fixture", prepared, input.Format, fixed, func(ctx context.Context, key, bundle string) error {
		name := "refs/markitect/retain/" + key
		if _, err := localCommand(ctx, source, nil, "update-ref", name, fixed.ID); err != nil {
			return err
		}
		_, err := localCommand(ctx, source, nil, "bundle", "create", bundle, name)
		return err
	})
	if err != nil {
		t.Fatal("retention under long state/temp paths failed", err)
	}
	if retained.Commit != fixed.ID || retained.Tree != fixed.Tree || retained.Binding != input.Binding {
		t.Fatal("retention changed the fixed candidate", retained)
	}
	for _, directory := range []string{source, r.Dir, original.StateRoot, g.TempRoot} {
		if err := os.RemoveAll(directory); err != nil {
			t.Fatal(err)
		}
	}
	destination := filepath.Join(root, "fresh-reopen")
	if err := os.MkdirAll(destination, 0700); err != nil {
		t.Fatal(err)
	}
	fresh := &LocalGit{StateRoot: g.StateRoot}
	if err := fresh.Reopen(ctx, retained, destination); err != nil {
		t.Fatal("long-path candidate depended on old workspaces or temporary state", err)
	}
	actual, err := object(ctx, destination, "HEAD")
	if err != nil || actual != fixed {
		t.Fatal("reopened commit/tree differs", actual, err)
	}
	if data, err := os.ReadFile(filepath.Join(destination, "out/accounted.ignored")); err != nil || string(data) != "original ignored input\n" {
		t.Fatal("reopened candidate lost accounted bytes", err)
	}
	if info, err := os.Stat(filepath.Join(destination, ".markitect/model/b/empty")); err != nil || !info.IsDir() {
		t.Fatal("reopened candidate lost empty Package", err)
	}
}

func fixturePathForControlled(s string) snapshot.Path { p, _ := snapshot.NewPath(s); return p }

// This uses real local Git and the unchanged Go command on a disposable exact
// candidate. The receipt's image identity is controlled; no container is started.
type aggregateGoChecks struct{}

func (aggregateGoChecks) Check(ctx context.Context, checks []project.ExecutionCheck, copy run.Copy, fixed run.Commit) ([]run.CheckObservation, error) {
	if len(checks) == 0 {
		return nil, nil
	}
	acquired, err := copy.Inspect(ctx, fixed)
	if err != nil {
		return nil, err
	}
	raw := acquired.Model().Inputs().Raw
	dirs, err := project.CheckDirectories(raw)
	if err != nil {
		return nil, err
	}
	var observed []run.CheckObservation
	for _, check := range checks {
		input, err := copy.Materialize(ctx, raw.Census, dirs)
		if err != nil {
			return nil, err
		}
		command := exec.CommandContext(ctx, "go", check.Args.Value...)
		command.Dir = filepath.Join(input.Location().Directory, check.Directory.Value.String())
		command.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
		output, runErr := command.CombinedOutput()
		closeErr := input.Close(ctx)
		exit := 0
		if runErr != nil {
			var failed *exec.ExitError
			if !errors.As(runErr, &failed) {
				return nil, errors.Join(runErr, closeErr)
			}
			exit = failed.ExitCode()
		}
		if closeErr != nil {
			return nil, closeErr
		}
		receipt := run.CheckReceipt{ID: run.CheckID(check.ID.Value), ImageID: "sha256:" + strings.Repeat("0", 64), Tree: raw.Census.Digest(), Started: true, Exited: true, ExitCode: exit, Output: output, CopiesClosed: true}
		for _, directory := range dirs {
			receipt.Directories = append(receipt.Directories, directory.String())
		}
		observed = append(observed, run.CheckObservation{ID: check.ID.Value, Commit: fixed.ID, Receipt: receipt})
	}
	return observed, nil
}

type partialLibrarySessions struct {
	controlled      *controlledManagers
	libraryAttempts int
	fail            bool
}
type partialLibrarySession struct {
	run.Session
	owner *partialLibrarySessions
	spec  run.SessionSpec
}

func (s *partialLibrarySessions) Open(ctx context.Context, spec run.SessionSpec) (run.Session, error) {
	base, err := s.controlled.Open(ctx, spec)
	if err != nil {
		return nil, err
	}
	return &partialLibrarySession{Session: base, owner: s, spec: spec}, nil
}
func (s *partialLibrarySession) Exchange(ctx context.Context, message run.Message) (run.Response, error) {
	if strings.HasPrefix(message.Task, "Answer pending Manager questions") {
		return s.Session.Exchange(ctx, message)
	}
	feedback := fmt.Sprint(message.Feedback)
	if strings.Contains(feedback, "check project-tests exited") {
		if s.spec.Manager.Package != "." {
			return run.Response{}, fmt.Errorf("project-wide check blocked partial library before handoff: %s", feedback)
		}
		return run.Response{Decision: run.Reject, Findings: []run.Finding{{Text: "The exact aggregate Go suite failed; this candidate is not acceptable."}}}, nil
	}
	response, err := s.Session.Exchange(ctx, message)
	if err != nil {
		return response, err
	}
	root := s.spec.Workspace.Location().Directory
	write := func(name, content string) error {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(content), 0600)
	}
	register := func(owner, name string) error {
		shard := ".markitect/registry/registry.yaml"
		if owner != "." {
			shard = ".markitect/registry/" + owner + "/registry.yaml"
		}
		path := filepath.Join(root, filepath.FromSlash(shard))
		content, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			content = []byte("apiVersion: markitect.registry/v1\nartifacts:\n")
		} else if err != nil {
			return err
		}
		content = []byte(strings.Replace(string(content), "artifacts: []", "artifacts:", 1))
		form := "code"
		if strings.HasSuffix(name, "_test.go") {
			form = "test"
		}
		content = append(content, []byte("  - path: "+name+"\n    form: "+form+"\n    realizes: [a/Thing]\n    why: Controlled aggregate Go regression.\n")...)
		return write(shard, string(content))
	}
	switch s.spec.Manager.Package {
	case "a":
		s.owner.libraryAttempts++
		// Root scaffolding and peer code were authored after initial delegation.
		// They must not appear through live propagation into this private copy.
		for _, path := range []string{"realization/go.mod", "realization/peer/peer.go"} {
			if _, err := os.Stat(filepath.Join(root, path)); !errors.Is(err, os.ErrNotExist) {
				return run.Response{}, fmt.Errorf("partial library unexpectedly has %s: %v", path, err)
			}
		}
		if s.owner.libraryAttempts == 1 {
			if err := write("realization/library/library_test.go", "package library\nimport (\"testing\"; \"fixture/peer\")\nfunc TestIntegratedPeer(t *testing.T){if peer.Value()!=1{t.Fatal(\"peer integration failed\")}}\n"); err != nil {
				return response, err
			}
			// First submission is deliberately unregistered: structural validation
			// must still refuse it before a valid partial handoff can reach Root.
		} else {
			if !strings.Contains(feedback, "file.unexplained") || !strings.Contains(feedback, "realization/library/library_test.go") {
				return run.Response{}, fmt.Errorf("missing child structural refusal: %s", feedback)
			}
			if err := register("a", "realization/library/library_test.go"); err != nil {
				return response, err
			}
		}
	case "b":
		value := "1"
		if s.owner.fail {
			value = "2"
		}
		if err := write("realization/peer/peer.go", "package peer\nfunc Value()int{return "+value+"}\n"); err != nil {
			return response, err
		}
		if err := register("b", "realization/peer/peer.go"); err != nil {
			return response, err
		}
	case ".":
		if err := write("realization/go.mod", "module fixture\ngo 1.23\n"); err != nil {
			return response, err
		}
		if err := register(".", "realization/go.mod"); err != nil {
			return response, err
		}
	}
	return response, nil
}

// The failing path traverses the actual Apply/Workflow gate and must never
// reach publication. Assessment still judges the fixed unrealized input.
type partialLibraryExecutor struct {
	sessions *partialLibrarySessions
	git      *LocalGit
	records  run.Records
}

func (e partialLibraryExecutor) Execute(ctx context.Context, request run.Request) (run.Result, error) {
	var sessions run.Sessions = e.sessions
	if request.Mandate == run.Assessment {
		sessions = &controlledManagers{crossed: make(chan struct{}), noChange: true, acceptedRepair: true}
	}
	return (&run.Executor{Sessions: sessions, Workspaces: e.git, Records: e.records, Checks: aggregateGoChecks{}}).Execute(ctx, request)
}

func TestPartialLibraryHandoffDefersProjectSuiteToExactAggregateRoot(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("failingRoot=%v", fail), func(t *testing.T) {
			_, prepared, git, input := runFixture(t, func(r *testkit.Repo) {
				r.Write(".markitect/execution.yaml", "apiVersion: markitect.execution/v2\nownership:\n  - {path: './', owner: .}\n  - {path: out/a.txt, owner: a}\n  - {path: out/accounted.ignored, owner: a}\n  - {path: out/deep.txt, owner: a/deep}\n  - {path: realization/library/, owner: a}\n  - {path: realization/peer/, owner: b}\nchecks:\n  - id: project-tests\n    profile: go\n    directory: realization\n    args: [test, ./...]\n    requiredBy: a/Thing\n    timeout: 2m\n")
			})
			records, err := runstore.New(git.StateRoot)
			if err != nil {
				t.Fatal(err)
			}
			sessions := &partialLibrarySessions{controlled: &controlledManagers{crossed: make(chan struct{}), noChange: true, acceptedRepair: true}, fail: fail}
			request := run.Request{ID: "partial-library", Project: "fixture", Prepared: prepared, Input: input, Mandate: run.Implement, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(2*time.Minute), 100), MessageTimeout: time.Minute, CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
			executor := partialLibraryExecutor{sessions: sessions, git: git, records: records}
			var result run.Result
			if fail {
				publisher := &unexpectedPublisher{}
				report, applyErr := (&apply.Command{Executor: executor, Candidates: git, Records: records, Publisher: publisher}).Execute(context.Background(), request)
				result, err = report.Workflow.Final, applyErr
				if report.Status != model.Failed || report.Publication != nil || publisher.calls != 0 || len(report.Workflow.Runs) != 2 {
					t.Fatal("failing aggregate crossed Apply publication gate", report.Status, publisher.calls, len(report.Workflow.Runs), applyErr)
				}
			} else {
				result, err = executor.Execute(context.Background(), request)
			}
			if err != nil || result.Technical != run.Completed {
				t.Fatalf("partial handoff did not reach checked aggregate: %s %v", result.Technical, err)
			}
			if sessions.libraryAttempts != 2 {
				t.Fatal("invalid child was not repaired before delivery", sessions.libraryAttempts)
			}
			var root run.Handoff
			for _, h := range result.Handoffs {
				if h.Manager == "." {
					root = h
				}
				if h.Manager == "a" && (h.Review == nil || h.Review.Decision != run.Approve) {
					t.Fatal("partial library did not reach parent review", h)
				}
			}
			if len(result.Checks) != 1 || result.Checks[0].Commit != root.Result.ID || result.Checks[0].ID != "project-tests" {
				t.Fatal("full suite lost exact aggregate binding", result.Checks, root.Result)
			}
			check := result.Checks[0].Receipt
			if !check.CopiesClosed || check.TimedOut || check.Uncertain || (check.ExitCode != 0) != fail {
				t.Fatal("wrong actual Go observation", check)
			}
			if fail {
				if result.Decision != run.Reject {
					t.Fatal("failing Root suite yielded acceptance", result.Decision)
				}
			} else {
				if result.Decision != run.Submitted || result.Candidate == nil || result.Candidate.Commit != root.Result.ID {
					t.Fatal("passing aggregate not retained", result.Decision, result.Candidate)
				}
				destination := t.TempDir()
				if err := git.Reopen(context.Background(), *result.Candidate, destination); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"realization/go.mod", "realization/peer/peer.go", "realization/library/library_test.go"} {
					if _, err := os.Stat(filepath.Join(destination, name)); err != nil {
						t.Fatal("aggregate missing after closure", name, err)
					}
				}
			}
			for _, closure := range result.Cleanup {
				if !closure.Closed {
					t.Fatal("unclosed resource", closure)
				}
			}
		})
	}
}
