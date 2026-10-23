package gitrepo

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func runFixture(t *testing.T) (*testkit.Repo, project.Prepared, *LocalGit, project.CandidateRef) {
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

func fixturePathForControlled(s string) snapshot.Path { p, _ := snapshot.NewPath(s); return p }
