package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// Run the Linux Review consumer's actual Git commands on local repositories.
// This test double starts no Engine and claims no container isolation or physical Linux modes.
type reviewHostGit struct{ container.ExecSession }

func (reviewHostGit) Exec(ctx context.Context, argv []string, input io.Reader, output io.Writer) error {
	if len(argv) == 0 || argv[0] != "/usr/bin/git" {
		return fmt.Errorf("expected the Linux Review Git command")
	}
	command := exec.CommandContext(ctx, "git", argv[1:]...)
	command.Env = testkit.GitEnv()
	command.Stdin, command.Stdout = input, output
	command.WaitDelay = time.Second
	return command.Run()
}

func TestLinuxChildReviewPreservesCompleteContextWithoutBinaryPatch(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), runDeadline(t))
	defer cancel()
	parent := testkit.NewRepo(t)
	parent.Write("text.txt", "before\n")
	base := run.Commit{ID: parent.Commit("review base")}
	base.Tree = parent.Git("rev-parse", "HEAD^{tree}")
	child := testkit.NewRepo(t)
	child.Git("fetch", "--quiet", parent.Dir, base.ID)
	child.Git("checkout", "--quiet", "--detach", base.ID)
	child.Write("text.txt", "after\n")
	child.Write("review.bin", "\x00\x01\x02binary review\x00")
	// Commit stages again, so Linux also needs the executable worktree mode.
	if err := os.Chmod(filepath.Join(child.Dir, "review.bin"), 0755); err != nil {
		t.Fatal(err)
	}
	child.Git("add", "review.bin")
	child.Git("update-index", "--chmod=+x", "review.bin")
	fixed := run.Commit{ID: child.Commit("binary and text child")}
	fixed.Tree = child.Git("rev-parse", "HEAD^{tree}")
	service := &linuxService{exec: reviewHostGit{}}
	p := &linuxCopy{service: service, directory: parent.Dir}
	c := &linuxCopy{service: service, directory: child.Dir}
	diff, err := p.Review(ctx, c, base, fixed)
	if err != nil {
		t.Fatal(err)
	}
	blob := child.Git("rev-parse", fixed.ID+":review.bin")
	for _, required := range []string{"diff --git a/review.bin b/review.bin", "new file mode 100755", strings.Repeat("0", len(blob)) + ".." + blob, "Binary files /dev/null and b/review.bin differ"} {
		if !strings.Contains(diff, required) {
			t.Fatal("Linux review lost binary path, mode or complete object identity", required)
		}
	}
	if strings.Contains(diff, "GIT binary patch") {
		t.Fatal("Linux review transported opaque binary patch data")
	}
	text := parent.Git("diff", "--no-ext-diff", "--no-textconv", "--binary", "--full-index", base.ID, fixed.ID, "--", "text.txt")
	if !strings.Contains(diff, text) {
		t.Fatal("Linux review changed or omitted the complete text diff")
	}
	if parent.Git("rev-parse", "HEAD") != base.ID || parent.Git("cat-file", "-s", fixed.ID+":review.bin") != "17" {
		t.Fatal("Review must import the complete fixed child without changing the parent")
	}
}

// hostLinuxTools runs the Linux service's Git and Node commands with the host's
// own binaries. It starts no Engine and claims no container isolation.
type hostLinuxTools struct{ container.ExecSession }

func (hostLinuxTools) Exec(ctx context.Context, argv []string, input io.Reader, output io.Writer) error {
	name := map[string]string{"/usr/bin/git": "git", "/usr/local/bin/node": "node"}[argv[0]]
	if name == "" {
		return fmt.Errorf("unexpected Linux service command %q", argv[0])
	}
	command := exec.CommandContext(ctx, name, argv[1:]...)
	command.Env = testkit.GitEnv()
	command.Stdin, command.Stdout = input, output
	command.WaitDelay = time.Second
	return command.Run()
}

// Both workspace backends leave exactly the untracked ELF files, the build
// outputs, out of a snapshot. Text, other binary content, short files and
// tracked ELF content are captured as before; directories and vanished paths
// are no build outputs.
func TestSnapshotLeavesUntrackedELFOutputsInTheCopy(t *testing.T) {
	ctx := context.Background()
	elf := "\x7fELF\x02\x01\x01\x00build output"
	for _, backend := range []string{"local", "linux"} {
		t.Run(backend, func(t *testing.T) {
			r := testkit.NewRepo(t)
			r.Write("tracked.elf", elf)
			r.Write("source.go", "package main\n")
			r.Commit("base")
			r.Write("tracked.elf", elf+" changed")
			r.Write("source.go", "package main\n\nfunc main() {}\n")
			r.Write("library", elf)
			r.Write("nested/library.test", elf+"\x00")
			r.Write("short", "\x7fEL")
			r.Write("data.bin", "\x00\x01binary data")
			r.Write("empty", "")
			executables, command := localExecutables(r.Dir), gitCommand(localCommand)
			if backend == "linux" {
				if _, err := exec.LookPath("node"); err != nil {
					t.Skip("the Linux probe runs Node, which this host lacks")
				}
				c := &linuxCopy{service: &linuxService{exec: hostLinuxTools{}}, directory: r.Dir}
				executables, command = c.executables, c.service.command
			}
			found, err := executables(ctx, []string{"library", "nested/library.test", "short", "data.bin", "empty", "nested", "vanished"})
			if err != nil || !reflect.DeepEqual(found, []string{"library", "nested/library.test"}) {
				t.Fatal("probe selected the wrong files", found, err)
			}
			fixed, err := snapshotGit(ctx, r.Dir, command, nil, executables)
			if err != nil {
				t.Fatal(err)
			}
			if files := strings.Fields(r.Git("ls-tree", "-r", "--name-only", fixed.ID)); !reflect.DeepEqual(files, []string{"data.bin", "empty", "short", "source.go", "tracked.elf"}) {
				t.Fatal("snapshot captured a build output or lost a file", files)
			}
			if r.Git("cat-file", "-s", fixed.ID+":tracked.elf") != fmt.Sprint(len(elf+" changed")) {
				t.Fatal("tracked ELF content change was not captured")
			}
			if status := r.Git("status", "--porcelain", "--untracked-files=all"); status != "?? library\n?? nested/library.test" {
				t.Fatalf("build outputs did not stay untracked in the copy: %q", status)
			}
			if again, err := snapshotGit(ctx, r.Dir, command, nil, executables); err != nil || again != fixed {
				t.Fatal("an unchanged copy with build outputs gave a new state", again, err)
			}
		})
	}
}

type notesReadingSessions struct {
	controlled *controlledManagers
	expected   []run.Note
	mu         sync.Mutex
	read       map[string][]run.Note
}

type accountingRepairSessions struct {
	controlled *controlledManagers
	child      *accountingRepairSession
}

// accountingRepairSession first submits an unregistered artifact, then repairs
// its registry from the parent's correction Findings.
type accountingRepairSession struct {
	*controlledManager
	attempts int
}

func (s *accountingRepairSessions) Open(ctx context.Context, spec run.SessionSpec) (run.Session, error) {
	session, err := s.controlled.Open(ctx, spec)
	if spec.Manager.Package != "a/deep" {
		return session, err
	}
	s.child = &accountingRepairSession{controlledManager: session.(*controlledManager)}
	return s.child, err
}

func (s *accountingRepairSession) Exchange(ctx context.Context, message run.Message) error {
	for _, e := range message.Events {
		if e.Assignment == nil {
			continue
		}
		s.attempts++
		root := s.spec.Workspace.Location().Directory
		if s.attempts == 1 {
			if err := os.WriteFile(filepath.Join(root, "out/unregistered.txt"), []byte("child artifact\n"), 0600); err != nil {
				return err
			}
			continue
		}
		text := fmt.Sprint(e.Assignment.Findings)
		if !strings.Contains(text, "file.unexplained") || !strings.Contains(text, "out/unregistered.txt") || !strings.Contains(text, "neither a registered artifact") {
			return fmt.Errorf("correction lost the actionable accounting diagnostic: %s", text)
		}
		registry := filepath.Join(root, ".markitect/registry/a/deep/registry.yaml")
		data, err := os.ReadFile(registry)
		if err != nil {
			return err
		}
		data = append(data, []byte("  - path: out/unregistered.txt\n    form: code\n    realizes: [a/deep/item]\n    why: Register the actual child artifact after parent rejection.\n")...)
		if err := os.WriteFile(registry, data, 0600); err != nil {
			return err
		}
	}
	return s.controlledManager.Exchange(ctx, message)
}

// V05 and scenario 26.4: an invalid Implement child candidate reaches its
// parent with the actual diagnostic; the parent cannot accept or integrate
// it, rejects it with Findings, and the child repairs it locally.
func TestInvalidChildCandidateReachesParentAndIsCorrected(t *testing.T) {
	_, prepared, git, input := runFixture(t)
	records, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	sessions := &accountingRepairSessions{controlled: &controlledManagers{crossed: make(chan struct{}), noChange: true, acceptedRepair: true}}
	request := run.Request{ID: "accounting-repair", Project: "fixture", Prepared: prepared, Input: input, Mandate: run.Implement, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
	result, err := (&run.Executor{Sessions: sessions, Workspaces: git, Records: records}).Execute(context.Background(), request)
	if err != nil || result.Technical != run.Completed || result.Decision != run.Submitted || result.Candidate == nil {
		t.Fatalf("child repair did not complete: %s %s %v", result.Technical, result.Decision, err)
	}
	if sessions.child.attempts != 2 || sessions.child.tasks != 2 || sessions.child.judgments != 2 || len(result.Handoffs) != 5 || len(sessions.controlled.sessions) != 5 {
		t.Fatal("repair replaced the Session or lost the full hierarchy", sessions.child.attempts, sessions.child.tasks, sessions.child.judgments, len(result.Handoffs))
	}
	if len(sessions.controlled.refusals) != 1 || !strings.Contains(sessions.controlled.refusals[0], "fails its check and cannot be accepted or integrated") || !strings.Contains(sessions.controlled.refusals[0], "out/unregistered.txt") {
		t.Fatal("invalid child candidate did not reach its parent with the refused approval", sessions.controlled.refusals)
	}
	integrated := false
	for _, h := range result.Handoffs {
		if h.Manager == "a/deep" && (h.Check == nil || !h.Check.Valid || h.Review == nil || h.Review.Decision != run.Approve) {
			t.Fatal("repaired child handoff lacks its passing check or acceptance", h.Check, h.Review)
		}
		if h.Manager == "a" {
			for _, child := range h.Children {
				integrated = integrated || child.Manager == "a/deep"
			}
		}
	}
	if !integrated {
		t.Fatal("parent did not review and integrate the repaired child")
	}
	for _, closure := range result.Cleanup {
		if !closure.Closed {
			t.Fatal("repair left a live resource", closure)
		}
	}
	destination := t.TempDir()
	if err := git.Reopen(context.Background(), *result.Candidate, destination); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(destination, "out/unregistered.txt")); err != nil || string(data) != "child artifact\n" {
		t.Fatal("repaired artifact did not survive parent integration and closure", err)
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
func (s *notesReadingSession) Exchange(ctx context.Context, message run.Message) error {
	s.once.Do(func() {
		notes, err := readAllNotes(ctx, s.spec.Tools.ReadNotes)
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
		return s.readErr
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
	request := run.Request{ID: "notes-first", Project: "fixture", Prepared: prepared, Input: input, Mandate: run.Assessment, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
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
	expected, err := allNotes(ctx, secondStore, "fixture")
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
	request.Allowance = run.NewAllowance(runDeadline(t), 100)
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
	all, err := allNotes(ctx, fresh, "fixture")
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
	core := "../../../testdata/core"
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
	r.Write(".markitect/model.yaml", "formatVersion: 1\nid: run-fixture\nroot: model\ndependencies: {markitect-core: '1.1.0'}\n")
	r.Write(".markitect/model-lock.yaml", "formatVersion: 1\nmodels:\n  markitect-core: {version: '1.1.0', digest: 'sha256:9552a7cdd560d8822854c286cd74d7dbb5d4277ee1fd16b25ef275213868e043'}\n")
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
	exchangeEntered     chan struct{}
	exchangeDrained     chan struct{}
	closeBlock          <-chan struct{}
	noChange            bool
	checkFirst          bool // Root runs the check tool on its state before judging it
	refusals            []string
	messages            []run.Message
	// realization replaces the default Implement writes; it may judge itself.
	realization func(context.Context, *controlledManager) error
}

// controlledManager scripts a well-behaved Manager under the Run tool
// contract: it handles delivered events, works on its active task, reviews
// every direct child and records its judgment through the judge tool.
type controlledManager struct {
	owner     *controlledManagers
	spec      run.SessionSpec
	calls     int
	closed    bool
	task      *run.Assignment
	tasks     int
	started   bool
	judged    bool
	judgments int
	pending   []run.Handoff
	answers   map[string]string
}

func (c *controlledManagers) Open(_ context.Context, s run.SessionSpec) (run.Session, error) {
	m := &controlledManager{owner: c, spec: s, answers: map[string]string{}}
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

func (c *controlledManagers) refused(err error) {
	c.mu.Lock()
	c.refusals = append(c.refusals, err.Error())
	c.mu.Unlock()
}

func (m *controlledManager) Exchange(ctx context.Context, message run.Message) error {
	m.calls++
	m.owner.mu.Lock()
	m.owner.messages = append(m.owner.messages, message)
	m.owner.mu.Unlock()
	if m.owner.exchangeBlock != nil && m.spec.Manager.Package == "." {
		if m.owner.exchangeEntered != nil {
			close(m.owner.exchangeEntered)
		}
		if m.owner.exchangeDrained != nil {
			defer close(m.owner.exchangeDrained)
		}
		<-m.owner.exchangeBlock
	}
	var later []run.Event
	for _, e := range message.Events {
		if e.Assignment != nil {
			m.task = e.Assignment
			m.tasks++
			continue
		}
		later = append(later, e)
	}
	question := ""
	if m.task != nil && !m.started {
		m.started = true
		var err error
		if question, err = m.start(ctx); err != nil {
			return err
		}
	}
	for _, e := range later {
		if err := m.handle(ctx, e); err != nil {
			return err
		}
	}
	if question != "" {
		if err := m.answer(ctx, question); err != nil {
			return err
		}
		other := "a"
		if m.spec.Manager.Package == "a" {
			other = "."
		}
		if m.answers[question] != "controlled answer from "+other {
			return fmt.Errorf("wrong correlated answer %q", m.answers[question])
		}
	}
	if m.task == nil {
		m.owner.mu.Lock()
		m.owner.receiptCalls++
		m.owner.mu.Unlock()
		return nil
	}
	return m.work(ctx)
}

// start delegates every direct child and, for Root and a, crosses one question
// along the hierarchy: both ask before either answers.
func (m *controlledManager) start(ctx context.Context) (string, error) {
	tools := m.spec.Tools
	for _, child := range m.spec.Manager.Children {
		if err := tools.Delegate(ctx, child, "Realize or review your own modeled responsibility.", nil); err != nil {
			return "", err
		}
	}
	if m.owner.crossed == nil || m.spec.Manager.Package != "." && m.spec.Manager.Package != "a" {
		return "", nil
	}
	other := "a"
	if m.spec.Manager.Package == "a" {
		other = "."
	}
	id, err := tools.Ask(ctx, other, "What is your controlled observation?")
	if err != nil {
		return "", err
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
		return "", ctx.Err()
	}
	return id, nil
}

// handle answers incoming questions and keeps answers and child handoffs. A
// reviewed child asked to reassess reports a late Reject through judge.
func (m *controlledManager) handle(ctx context.Context, e run.Event) error {
	tools := m.spec.Tools
	switch {
	case e.Question != nil:
		if e.Question.Text == "Reassess the already accepted handoff." {
			// Record the finding before answering. The parent awaits the changed
			// handoff once it has the answer; answering first lets that await run
			// before the finding exists, and it is then refused.
			if err := tools.Judge(ctx, run.Reject, []run.Finding{{Text: "Late required correction discovered after accepted integration."}}); err != nil {
				return err
			}
			return tools.Reply(ctx, e.Question.ID, "late required correction")
		}
		return tools.Reply(ctx, e.Question.ID, "controlled answer from "+m.spec.Manager.Package)
	case e.Answer != nil:
		m.answers[e.Answer.ID] = e.Answer.Text
	case e.Handoff != nil:
		m.pending = append(m.pending, *e.Handoff)
	}
	return nil
}

// answer awaits the answer to the asker's own question, handling other events.
func (m *controlledManager) answer(ctx context.Context, id string) error {
	for {
		if _, ok := m.answers[id]; ok {
			return nil
		}
		e, err := m.spec.Tools.Await(ctx)
		if err != nil {
			return err
		}
		if err = m.handle(ctx, e); err != nil {
			return err
		}
	}
}

func (m *controlledManager) handoff(ctx context.Context) (run.Handoff, error) {
	for len(m.pending) == 0 {
		e, err := m.spec.Tools.Await(ctx)
		if err != nil {
			return run.Handoff{}, err
		}
		if err = m.handle(ctx, e); err != nil {
			return run.Handoff{}, err
		}
	}
	h := m.pending[0]
	m.pending = m.pending[1:]
	return h, nil
}

func (m *controlledManager) work(ctx context.Context) error {
	tools := m.spec.Tools
	open := map[string]bool{}
	if m.tasks == 1 {
		for _, child := range m.spec.Manager.Children {
			open[child] = true
		}
	}
	for len(open) != 0 {
		h, err := m.handoff(ctx)
		if err != nil {
			return err
		}
		// A fixed child stays reachable for questions in its own Session.
		id, err := tools.Ask(ctx, h.Manager, "Explain your completed fixed handoff.")
		if err != nil {
			return err
		}
		if err = m.answer(ctx, id); err != nil {
			return err
		}
		if m.answers[id] != "controlled answer from "+h.Manager {
			return fmt.Errorf("late answer mismatch %q", m.answers[id])
		}
		m.owner.mu.Lock()
		m.owner.lateQuestions++
		m.owner.mu.Unlock()
		correct := func(text, task string) error {
			findings := []run.Finding{{Text: text}}
			if err := tools.Review(ctx, h.Manager, h.Result.ID, run.Reject, findings); err != nil {
				return err
			}
			if err := tools.Delegate(ctx, h.Manager, task, findings); err != nil {
				return err
			}
			m.owner.mu.Lock()
			m.owner.corrections++
			m.owner.mu.Unlock()
			return nil
		}
		switch {
		case h.Decision == run.Submitted && h.Check != nil && !h.Check.Valid:
			refusal := tools.Review(ctx, h.Manager, h.Result.ID, run.Approve, nil)
			if refusal == nil {
				return fmt.Errorf("invalid child candidate %s was accepted", h.Result.ID)
			}
			m.owner.refused(refusal)
			if err = correct(h.Check.Diagnostic, "Repair the rejected child candidate."); err != nil {
				return err
			}
			continue
		case m.spec.Mandate == run.Implement && h.Manager == "a/deep" && strings.Contains(h.Diff, "+wrong"):
			if err = correct("The complete fixed diff shows wrong; correct the child artifact.", "Correct the reviewed child artifact."); err != nil {
				return err
			}
			continue
		case h.Decision == run.Reject && m.owner.lateReject && h.Manager == "a/deep":
			if err = correct("Correct the actual late required finding.", "Correct late rejection."); err != nil {
				return err
			}
			continue
		}
		if err = tools.Review(ctx, h.Manager, h.Result.ID, run.Approve, nil); err != nil {
			return err
		}
		delete(open, h.Manager)
		if m.owner.lateReject && h.Manager == "a/deep" && !m.owner.didLateReject {
			m.owner.didLateReject = true
			open[h.Manager] = true
			id, err := tools.Ask(ctx, h.Manager, "Reassess the already accepted handoff.")
			if err != nil {
				return err
			}
			if err = m.answer(ctx, id); err != nil {
				return err
			}
			if m.answers[id] != "late required correction" {
				return fmt.Errorf("late finding not actually delivered")
			}
			continue
		}
		if m.spec.Mandate == run.Implement && h.Manager == "a/deep" && !m.owner.acceptedRepair && !m.owner.lateReject {
			m.owner.acceptedRepair = true
			open[h.Manager] = true
			if err := os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, "out/deep.txt"), []byte("parent revision\n"), 0600); err != nil {
				return err
			}
			findings := []run.Finding{{Text: "Correct accepted child work on the integrated parent baseline."}}
			if err := tools.Delegate(ctx, h.Manager, "Repair after parent integration.", findings); err != nil {
				return err
			}
			m.owner.mu.Lock()
			m.owner.corrections++
			m.owner.mu.Unlock()
		}
	}
	if _, err := tools.AppendNote(ctx, "controlled test", m.spec.Manager.Package+" participated"); err != nil {
		return err
	}
	decision, findings := run.Submitted, []run.Finding(nil)
	if m.spec.Mandate == run.Implement {
		realize := m.realize
		if m.owner.realization != nil {
			realize = func(ctx context.Context) error { return m.owner.realization(ctx, m) }
		}
		if err := realize(ctx); err != nil {
			return err
		}
		if m.judged {
			m.task, m.judged = nil, false
			return nil
		}
	} else {
		// Review scratch is deliberately invalid product content; it must never
		// replace the fixed input candidate returned by Assessment or Verify.
		if err := os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, "review-scratch.txt"), []byte("scratch"), 0600); err != nil {
			return err
		}
		decision = run.Approve
		if m.owner.forceDecision != "" {
			decision, findings = m.owner.forceDecision, []run.Finding{{Text: "controlled semantic rejection"}}
		}
	}
	if m.owner.checkFirst && m.spec.Manager.Package == "." {
		check, err := tools.Check(ctx)
		if err != nil {
			return err
		}
		if !check.Valid {
			return fmt.Errorf("Root's state fails its own check: %s", check.Diagnostic)
		}
	}
	if err := m.judge(ctx, decision, findings); err != nil {
		return err
	}
	m.task, m.judged = nil, false
	return nil
}

func (m *controlledManager) judge(ctx context.Context, decision run.Decision, findings []run.Finding) error {
	err := m.spec.Tools.Judge(ctx, decision, findings)
	if err == nil {
		m.judgments++
		m.judged = true
	}
	return err
}

func (m *controlledManager) realize(ctx context.Context) error {
	root := m.spec.Workspace.Location().Directory
	if m.owner.badManifest && m.spec.Manager.Package == "." {
		entry, found := m.spec.Prepared.Attempt().Model().Inputs().Raw.Census.Lookup(fixturePathForControlled(".markitect/model.yaml"))
		if !found {
			return fmt.Errorf("captured manifest absent")
		}
		manifest := filepath.Join(root, ".markitect/model.yaml")
		if err := os.WriteFile(manifest, []byte("formatVersion: invalid\nid: run-fixture\n"), 0600); err != nil {
			return err
		}
		// Root has no parent: its own completion check refuses the judgment.
		err := m.judge(ctx, run.Submitted, nil)
		if err == nil {
			return fmt.Errorf("invalid Root manifest was accepted")
		}
		m.owner.refused(err)
		if err := os.WriteFile(manifest, entry.Bytes(), 0600); err != nil {
			return err
		}
	}
	if m.owner.noChange {
		return nil
	}
	path, value := "", ""
	if m.spec.Manager.Package == "a" {
		path, value = "out/a.txt", "parent result\n"
	}
	if m.spec.Manager.Package == "a/deep" {
		path, value = "out/deep.txt", "wrong\n"
		if len(m.task.Findings) > 0 {
			value = "corrected\n"
			if m.task.Task == "Correct late rejection." {
				value = "late corrected\n"
			}
			if m.task.Task == "Repair after parent integration." {
				value = "repaired after integration\n"
			}
		}
	}
	if path == "" {
		return nil
	}
	return os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte(value), 0600)
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
			request := run.Request{MaxManagers: 5, ID: string(mandate), Project: "fixture", Prepared: p, Input: ref, Mandate: mandate, Allowance: run.NewAllowance(runDeadline(t), 80), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
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
			notes, err := allNotes(ctx, fresh, "fixture")
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

// The basis survives realization, registry maintenance and another working
// directory, and changes with the model or the execution configuration.
func TestPreparedBasisLeavesOutRealizationAndLocation(t *testing.T) {
	_, base, _, _ := runFixture(t)
	_, realized, _, _ := runFixture(t, func(r *testkit.Repo) {
		r.Write("out/b.txt", "realized\n")
		r.Write(".markitect/registry/a/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts:\n  - path: out/a.txt\n    form: code\n    realizes: [a/Thing]\n    why: Controlled parent artifact.\n  - path: out/accounted.ignored\n    form: code\n    realizes: [a/Thing]\n    why: Accounted Gitignored untracked input must survive capture.\n  - path: out/b.txt\n    form: code\n    realizes: [a/Thing]\n    why: A later realization.\n")
	})
	if realized.Binding() == base.Binding() || realized.Basis() != base.Basis() || base.Basis() == "" {
		t.Fatal("realization changed the basis or left the binding", base.Basis(), realized.Basis())
	}
	for name, configure := range map[string]func(*testkit.Repo){
		"model": func(r *testkit.Repo) {
			r.Write(".markitect/model/a/thing.yaml", "imports: [{package: markitect/core, as: Core}]\nid: Thing\nkind: Core.Kind\nbody: A changed realization obligation.\nspec: {}\n")
		},
		"execution": func(r *testkit.Repo) {
			r.Write(".markitect/execution.yaml", "# Changed execution configuration.\napiVersion: markitect.execution/v2\nchecks: []\n")
		},
	} {
		if _, changed, _, _ := runFixture(t, configure); changed.Basis() == base.Basis() {
			t.Fatal("a changed", name, "kept the basis")
		}
	}
}

// Changes diffs two stored candidates from durable state alone, also when
// their histories share no commit.
func TestChangesDiffsStoredCandidatesWithoutCommonHistory(t *testing.T) {
	r, p, git, input := runFixture(t)
	ctx := context.Background()
	r.Git("checkout", "--quiet", "--orphan", "unrelated")
	r.Write("out/a.txt", "changed\n")
	r.Commit("unrelated history")
	head, err := object(ctx, r.Dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	other, err := git.retain(ctx, input.Project, p, r.Dir, head)
	if err != nil {
		t.Fatal(err)
	}
	if commits := strings.Fields(r.Git("rev-list", "--parents", "-n", "1", "HEAD")); len(commits) != 1 {
		t.Fatal("the unrelated candidate has a parent", commits)
	}
	diff, err := git.Changes(ctx, input, other)
	if err != nil || !strings.Contains(diff, "out/a.txt") || !strings.Contains(diff, "-initial") || !strings.Contains(diff, "+changed") {
		t.Fatalf("incomplete stored diff: %v\n%s", err, diff)
	}
	if same, err := git.Changes(ctx, other, other); err != nil || same != "" {
		t.Fatal("an unchanged candidate has changes", err, same)
	}
	foreign := other
	foreign.Project = "other"
	if _, err = git.Changes(ctx, input, foreign); err == nil {
		t.Fatal("diffed candidates of different projects")
	}
}

func TestDurableCandidateRetentionAndReopenUnderLongPaths(t *testing.T) {
	r, prepared, original, input := runFixture(t)
	ctx, cancel := context.WithDeadline(context.Background(), runDeadline(t))
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
	rootRefusal     string
}

func newPartialLibrarySessions(fail bool) *partialLibrarySessions {
	s := &partialLibrarySessions{controlled: &controlledManagers{crossed: make(chan struct{}), noChange: true, acceptedRepair: true}, fail: fail}
	s.controlled.realization = s.realize
	return s
}

func (s *partialLibrarySessions) Open(ctx context.Context, spec run.SessionSpec) (run.Session, error) {
	return s.controlled.Open(ctx, spec)
}

// realize writes each Manager's part after its delegation and child reviews.
// Root's own completion check runs the aggregate suite on its judged state.
func (s *partialLibrarySessions) realize(ctx context.Context, m *controlledManager) error {
	root := m.spec.Workspace.Location().Directory
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
	peer := "package peer\nfunc Value()int{return 1}\n"
	if s.fail {
		peer = "package peer\nfunc Value()int{return 2}\n"
	}
	switch m.spec.Manager.Package {
	case "a":
		s.libraryAttempts++
		// Root scaffolding and peer code were authored after initial delegation.
		// They must not appear through live propagation into this private copy.
		// A correction merges Root's baseline at delegation, which holds b's code
		// once Root has integrated b first; Root writes its scaffolding last.
		unexpected := []string{"realization/go.mod"}
		if s.libraryAttempts == 1 {
			unexpected = append(unexpected, "realization/peer/peer.go")
		}
		for _, path := range unexpected {
			if _, err := os.Stat(filepath.Join(root, path)); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("partial library unexpectedly has %s: %v", path, err)
			}
		}
		if data, err := os.ReadFile(filepath.Join(root, "realization/peer/peer.go")); err == nil && string(data) != peer {
			return fmt.Errorf("correction baseline does not hold b's integrated code: %q", data)
		}
		if s.libraryAttempts == 1 {
			// First submission is deliberately unregistered: its parent receives it
			// with the failing structural check and cannot accept it.
			return write("realization/library/library_test.go", "package library\nimport (\"testing\"; \"fixture/peer\")\nfunc TestIntegratedPeer(t *testing.T){if peer.Value()!=1{t.Fatal(\"peer integration failed\")}}\n")
		}
		feedback := fmt.Sprint(m.task.Findings)
		if !strings.Contains(feedback, "file.unexplained") || !strings.Contains(feedback, "realization/library/library_test.go") {
			return fmt.Errorf("missing child structural refusal: %s", feedback)
		}
		return register("a", "realization/library/library_test.go")
	case "b":
		if err := write("realization/peer/peer.go", peer); err != nil {
			return err
		}
		return register("b", "realization/peer/peer.go")
	case ".":
		if err := write("realization/go.mod", "module fixture\ngo 1.23\n"); err != nil {
			return err
		}
		if err := register(".", "realization/go.mod"); err != nil {
			return err
		}
		err := m.judge(ctx, run.Submitted, nil)
		if err == nil || !strings.Contains(err.Error(), "check project-tests exited") {
			return err
		}
		s.rootRefusal = err.Error()
		return m.judge(ctx, run.Reject, []run.Finding{{Text: "The exact aggregate Go suite failed; this candidate is not acceptable."}})
	}
	return nil
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
			sessions := newPartialLibrarySessions(fail)
			request := run.Request{ID: "partial-library", Project: "fixture", Prepared: prepared, Input: input, Mandate: run.Implement, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
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
			if sessions.libraryAttempts != 2 || len(sessions.controlled.refusals) != 1 || !strings.Contains(sessions.controlled.refusals[0], "realization/library/library_test.go") {
				t.Fatal("invalid child did not reach its parent and get repaired", sessions.libraryAttempts, sessions.controlled.refusals)
			}
			var root run.Handoff
			for _, h := range result.Handoffs {
				if h.Manager == "." {
					root = h
				}
				if h.Manager == "a" && (h.Review == nil || h.Review.Decision != run.Approve) {
					t.Fatal("partial library did not reach parent review", h)
				}
				// The library imports sibling code integrated only at Root. That
				// outstanding integration is not invalidity of the child state.
				if h.Manager == "a" && (h.Check == nil || !h.Check.Valid || len(h.Check.Checks) != 0) {
					t.Fatal("child lacking only a sibling's integration was judged invalid or reported an unrun check", h.Check)
				}
			}
			if fail != (sessions.rootRefusal != "") {
				t.Fatal("Root completion diagnostic did not match the aggregate suite", sessions.rootRefusal)
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

// allNotes reads every page of project Notes in order.
func allNotes(ctx context.Context, records interface {
	Notes(context.Context, string, string) (run.NotePage, error)
}, project string) ([]run.Note, error) {
	return readAllNotes(ctx, func(ctx context.Context, after string) (run.NotePage, error) {
		return records.Notes(ctx, project, after)
	})
}

// readAllNotes follows the page cursor to the last page.
func readAllNotes(ctx context.Context, read func(context.Context, string) (run.NotePage, error)) ([]run.Note, error) {
	var all []run.Note
	for after := ""; ; {
		page, err := read(ctx, after)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Notes...)
		if page.Next == "" {
			return all, nil
		}
		after = page.Next
	}
}
