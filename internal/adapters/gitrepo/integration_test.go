package gitrepo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

type integrationRuntime struct {
	roots                map[string]string
	failPatchRead        bool
	failParentApply      bool
	failTempCommit       bool
	backgroundChange     string
	failPatchAfterOutput bool
	mergeSucceededError  bool
	failMergeReadback    bool
	afterMerge           bool
	failUpdateFetch      bool
}

func (f *integrationRuntime) InspectVolume(_ context.Context, name string) (container.VolumeObservation, error) {
	_, exists := f.roots[name]
	return container.VolumeObservation{Name: name, Exists: exists, Labels: map[string]string{workspaceOwnerLabel: "owned"}}, nil
}
func (*integrationRuntime) CreateVolumeExclusive(context.Context, string, map[string]string) (container.VolumeObservation, error) {
	return container.VolumeObservation{}, errors.New("unused")
}
func (*integrationRuntime) VolumeMounts(context.Context, string) ([]string, error) { return nil, nil }
func (*integrationRuntime) RemoveVolume(context.Context, string) error             { return nil }
func (f *integrationRuntime) RunHelper(ctx context.Context, spec container.ContainerSpec, argv []string, stdin io.Reader, stdout io.Writer) error {
	if len(spec.Mounts) < 1 || len(spec.Mounts) > 2 || len(argv) < 2 || argv[0] != "/usr/bin/git" {
		return errors.New("unexpected Git helper")
	}
	var directory, command string
	if index := slices.Index(argv, "-C"); index >= 0 && index+2 < len(argv) {
		directory, command = argv[index+1], argv[index+2]
	}
	if f.failUpdateFetch && command == "fetch" {
		return errors.New("controlled parent fetch failure")
	}
	root := f.roots[spec.Mounts[0].Volume]
	if root == "" {
		return errors.New("unknown assignment volume")
	}
	sourceRoot := ""
	for _, mount := range spec.Mounts[1:] {
		if mount.Target == "/source" {
			sourceRoot = f.roots[mount.Volume]
		}
	}
	if f.failPatchRead && command == "diff" {
		if _, ok := stdout.(*io.PipeWriter); ok {
			return errors.New("child patch read failed")
		}
	}
	if f.failMergeReadback && f.afterMerge && command == "rev-parse" {
		return errors.New("parent head readback failed")
	}
	if strings.Contains(directory, "markitect-integration-") {
		if command == "apply" {
			if f.backgroundChange != "" {
				if err := os.WriteFile(filepath.Join(root, "common.txt"), []byte(f.backgroundChange), 0644); err != nil {
					return err
				}
				f.backgroundChange = ""
			}
			if f.failParentApply {
				return errors.New("parent apply helper failed")
			}
		}
		if command == "commit" && f.failTempCommit {
			return errors.New("temporary commit helper failed")
		}
	}
	args := slices.Clone(argv[1:])
	for i := range args {
		if i > 0 && (args[i-1] == "-m" || args[i-1] == "--message") {
			continue
		}
		if args[i] == "/work" {
			args[i] = root
		} else if strings.HasPrefix(args[i], "/work/") {
			args[i] = filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(args[i], "/work/")))
		} else if args[i] == "/source" {
			args[i] = sourceRoot
		}
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdin, cmd.Stdout = stdin, stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.New(strings.TrimSpace(stderr.String()) + ": " + err.Error())
	}
	if f.failPatchAfterOutput && command == "diff" {
		if _, ok := stdout.(*io.PipeWriter); ok {
			return errors.New("child patch producer failed after output")
		}
	}
	if f.mergeSucceededError && command == "merge" {
		f.afterMerge = true
		return errors.New("merge helper lost its success response")
	}
	return nil
}

func integrationFixture(t *testing.T, ids ...run.WorkspaceID) (*Workspaces, map[run.WorkspaceID]*assignmentWorkspace, map[run.WorkspaceID]string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git unavailable")
	}
	runtime := &integrationRuntime{roots: make(map[string]string)}
	w := &Workspaces{runtime: runtime, helper: container.ContainerSpec{Name: "test", Labels: map[string]string{"markitect.store": "test"}}, owned: make(map[run.WorkspaceID]string)}
	attachTestGit(w)
	copies, roots := make(map[run.WorkspaceID]*assignmentWorkspace), make(map[run.WorkspaceID]string)
	for _, id := range ids {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "common.txt"), []byte("shared baseline\n"), 0644); err != nil {
			t.Fatal(err)
		}
		volume := w.volume(id)
		runtime.roots[volume], roots[id], w.owned[id] = root, root, "owned"
		registerTestGitVolume(w, volume)
		a := &assignmentWorkspace{owner: w, id: id, volume: volume}
		if _, err := a.git(context.Background(), "init", "-b", "assignment"); err != nil {
			t.Fatal(err)
		}
		var err error
		a.start, err = a.commit(context.Background(), "independent starting stand "+string(id))
		if err != nil {
			t.Fatal(err)
		}
		copies[id] = a
	}
	return w, copies, roots
}

func integrationWrite(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func integrationHandoff(t *testing.T, a *assignmentWorkspace) run.AssignmentHandoff {
	t.Helper()
	result, err := a.Commit(context.Background(), "child result")
	if err != nil {
		t.Fatal(err)
	}
	return run.AssignmentHandoff{StartingCommit: a.start, ResultCommit: result, Summary: "Changed and tested own area"}
}

func TestChildGitCloneHasFixedFoundationAndSeparateLiveFiles(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "parent")
	integrationWrite(t, roots["parent"], "foundation.txt", "public contract\n")
	foundation, err := copies["parent"].Commit(ctx, "Parent foundation")
	if err != nil {
		t.Fatal(err)
	}
	childRoot := t.TempDir()
	childID := run.WorkspaceID("child")
	childVolume := w.volume(childID)
	w.runtime.(*integrationRuntime).roots[childVolume] = childRoot
	w.owned[childID] = "owned"
	registerTestGitVolume(w, childVolume)
	child := &assignmentWorkspace{owner: w, id: childID, volume: childVolume}
	if err := w.cloneAssignment(ctx, copies["parent"], child); err != nil {
		t.Fatal(err)
	}
	child.start, err = child.git(ctx, "rev-parse", "HEAD")
	if err != nil || child.start != foundation {
		t.Fatalf("child did not start from exact parent Git stand: %q %v", child.start, err)
	}
	integrationWrite(t, roots["parent"], "foundation.txt", "later parent edit\n")
	got, err := os.ReadFile(filepath.Join(childRoot, "foundation.txt"))
	if err != nil || string(got) != "public contract\n" {
		t.Fatalf("parent's later edit leaked into child copy: %q %v", got, err)
	}
}

func TestIndependentAssignmentGitResultsIntegrateIntoParent(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "library", "lending", "storage")
	integrationWrite(t, roots["library"], "library.txt", "public interface\n")
	integrationWrite(t, roots["lending"], "lending.txt", "loan limit = 3\n")
	integrationWrite(t, roots["storage"], "storage.txt", "loan records\n")
	lending, storage := integrationHandoff(t, copies["lending"]), integrationHandoff(t, copies["storage"])
	if lending.StartingCommit == copies["library"].StartingCommit() {
		t.Fatal("test did not establish independent Git histories")
	}
	if diff, err := w.Changes(ctx, copies["lending"], lending); err != nil || !strings.Contains(diff, "loan limit = 3") {
		t.Fatalf("parent could not inspect child Git changes: %q %v", diff, err)
	}
	first, err := w.Integrate(ctx, copies["library"], copies["lending"], lending, "")
	if err != nil || first == lending.ResultCommit || first == "" {
		t.Fatalf("first parent integration %q: %v", first, err)
	}
	second, err := w.Integrate(ctx, copies["library"], copies["storage"], storage, "")
	if err != nil || second == first {
		t.Fatalf("second parent integration %q after %q: %v", second, first, err)
	}
	for name, want := range map[string]string{"library.txt": "public interface\n", "lending.txt": "loan limit = 3\n", "storage.txt": "loan records\n"} {
		got, err := os.ReadFile(filepath.Join(roots["library"], name))
		if err != nil || string(got) != want {
			t.Fatalf("parent %s = %q: %v", name, got, err)
		}
	}
	integrationWrite(t, roots["lending"], "lending.txt", "loan limit = 5\n")
	corrected := integrationHandoff(t, copies["lending"])
	third, err := w.Integrate(ctx, copies["library"], copies["lending"], corrected, lending.ResultCommit)
	if err != nil || third == second {
		t.Fatalf("corrected child result was not integrated once: %q %v", third, err)
	}
	got, err := os.ReadFile(filepath.Join(roots["library"], "lending.txt"))
	if err != nil || string(got) != "loan limit = 5\n" {
		t.Fatalf("correction in parent: %q %v", got, err)
	}
	if present, err := copies["lending"].ContainsCommit(ctx, lending.ResultCommit); err != nil || !present {
		t.Fatal("earlier child result was not retained")
	}
}

func TestIndependentAssignmentGitConflictReturnsToOwningChild(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "library", "lending")
	integrationWrite(t, roots["library"], "common.txt", "parent decision\n")
	integrationWrite(t, roots["lending"], "common.txt", "child decision\n")
	child := integrationHandoff(t, copies["lending"])
	_, err := w.Integrate(ctx, copies["library"], copies["lending"], child, "")
	if !errors.Is(err, ErrIntegrationConflict) {
		t.Fatalf("real Git conflict was not returned to child owner: %v", err)
	}
	parent, readErr := os.ReadFile(filepath.Join(roots["library"], "common.txt"))
	if readErr != nil || string(parent) != "parent decision\n" {
		t.Fatalf("conflict overlaid parent stand: %q %v", parent, readErr)
	}
	if status, statusErr := copies["library"].git(ctx, "status", "--porcelain"); statusErr != nil || status != "" {
		t.Fatalf("conflict left parent Git dirty: %q %v", status, statusErr)
	}
	if present, err := copies["lending"].ContainsCommit(ctx, child.ResultCommit); err != nil || !present {
		t.Fatal("conflicted child result was not retained for correction")
	}
}

func TestIndependentAssignmentPatchReadFailureIsNotAConflict(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "library", "lending")
	integrationWrite(t, roots["lending"], "lending.txt", "new loan behavior\n")
	child := integrationHandoff(t, copies["lending"])
	w.runtime.(*integrationRuntime).failPatchRead = true
	_, err := w.Integrate(ctx, copies["library"], copies["lending"], child, "")
	if err == nil || errors.Is(err, ErrIntegrationConflict) || !strings.Contains(err.Error(), "patch read failed") {
		t.Fatalf("technical patch read became a child conflict: %v", err)
	}
	if _, err := os.Stat(filepath.Join(roots["library"], "lending.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed patch changed parent file: %v", err)
	}
}

func TestTemporaryIntegrationFailureLeavesParentRetryable(t *testing.T) {
	for _, phase := range []string{"apply", "commit"} {
		t.Run(phase, func(t *testing.T) {
			ctx := context.Background()
			w, copies, roots := integrationFixture(t, "library", "lending")
			integrationWrite(t, roots["lending"], "lending.txt", "child behavior\n")
			child := integrationHandoff(t, copies["lending"])
			runtime := w.runtime.(*integrationRuntime)
			runtime.failParentApply = phase == "apply"
			runtime.failTempCommit = phase == "commit"
			if _, err := w.Integrate(ctx, copies["library"], copies["lending"], child, ""); err == nil || errors.Is(err, ErrIntegrationConflict) {
				t.Fatalf("temporary %s failure was accepted or misclassified: %v", phase, err)
			}
			if _, err := os.Stat(filepath.Join(roots["library"], "lending.txt")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed %s changed live parent: %v", phase, err)
			}
			runtime.failParentApply, runtime.failTempCommit = false, false
			if result, err := w.Integrate(ctx, copies["library"], copies["lending"], child, ""); err != nil || result == "" {
				t.Fatalf("clean retry after temporary %s failure: %q %v", phase, result, err)
			}
			got, err := os.ReadFile(filepath.Join(roots["library"], "lending.txt"))
			if err != nil || string(got) != "child behavior\n" {
				t.Fatalf("retry after %s: %q %v", phase, got, err)
			}
		})
	}
}

func TestLiveParentEditDuringIntegrationIsPreserved(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "library", "lending")
	integrationWrite(t, roots["lending"], "common.txt", "child decision\n")
	child := integrationHandoff(t, copies["lending"])
	w.runtime.(*integrationRuntime).backgroundChange = "new live parent work\n"
	_, err := w.Integrate(ctx, copies["library"], copies["lending"], child, "")
	if err == nil || errors.Is(err, ErrIntegrationConflict) {
		t.Fatalf("live parent drift became a child conflict or success: %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(roots["library"], "common.txt"))
	if readErr != nil || string(got) != "new live parent work\n" {
		t.Fatalf("live parent edit was discarded: %q %v", got, readErr)
	}
	if status, statusErr := copies["library"].git(ctx, "status", "--porcelain"); statusErr != nil || !strings.Contains(status, "common.txt") {
		t.Fatalf("live parent edit not retained as work: %q %v", status, statusErr)
	}
}

func TestCompletedMergeWithLostHelperResponseRetainsParentCommit(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "library", "lending")
	integrationWrite(t, roots["lending"], "lending.txt", "child result\n")
	child := integrationHandoff(t, copies["lending"])
	w.runtime.(*integrationRuntime).mergeSucceededError = true
	result, err := w.Integrate(ctx, copies["library"], copies["lending"], child, "")
	if result == "" || err == nil || errors.Is(err, ErrIntegrationConflict) {
		t.Fatalf("ambiguous merge lost its applied parent commit: %q %v", result, err)
	}
	head, headErr := copies["library"].git(ctx, "rev-parse", "HEAD")
	if headErr != nil || head != result {
		t.Fatalf("read-back parent commit %q differs from applied %q: %v", head, result, headErr)
	}
	got, readErr := os.ReadFile(filepath.Join(roots["library"], "lending.txt"))
	if readErr != nil || string(got) != "child result\n" {
		t.Fatalf("applied child content absent: %q %v", got, readErr)
	}
}

func TestUnreadableAmbiguousMergeBlocksRetry(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "library", "lending")
	integrationWrite(t, roots["lending"], "lending.txt", "child result\n")
	child := integrationHandoff(t, copies["lending"])
	runtime := w.runtime.(*integrationRuntime)
	runtime.mergeSucceededError, runtime.failMergeReadback = true, true
	if _, err := w.Integrate(ctx, copies["library"], copies["lending"], child, ""); !errors.Is(err, ErrIntegrationUncertain) {
		t.Fatalf("unknown merge outcome was not guarded: %v", err)
	}
	runtime.mergeSucceededError, runtime.failMergeReadback = false, false
	if _, err := w.Integrate(ctx, copies["library"], copies["lending"], child, ""); !errors.Is(err, ErrIntegrationUncertain) {
		t.Fatalf("ordinary retry bypassed unknown parent result: %v", err)
	}
}

func TestPostOutputChildPatchFailureStaysTechnicalDespiteConflict(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "library", "lending")
	integrationWrite(t, roots["library"], "common.txt", "parent decision\n")
	integrationWrite(t, roots["lending"], "common.txt", "child decision\n")
	child := integrationHandoff(t, copies["lending"])
	w.runtime.(*integrationRuntime).failPatchAfterOutput = true
	_, err := w.Integrate(ctx, copies["library"], copies["lending"], child, "")
	if err == nil || errors.Is(err, ErrIntegrationConflict) || !strings.Contains(err.Error(), "producer failed") {
		t.Fatalf("technical child read was blamed as a Git conflict: %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(roots["library"], "common.txt"))
	if readErr != nil || string(got) != "parent decision\n" {
		t.Fatalf("technical source failure changed live parent: %q %v", got, readErr)
	}
}

func TestParentReviewDiffIsBoundedBeforeHostAllocation(t *testing.T) {
	var sink reviewOutput
	sink.limit = 3
	if n, err := sink.Write([]byte("123456789")); err != nil || n != 9 || !sink.exceeded || sink.Len() != 3 {
		t.Fatalf("bounded writer retained too much or failed to drain: %d %v %+v", n, err, sink)
	}
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "lending")
	integrationWrite(t, roots["lending"], "large.txt", strings.Repeat("x", 8<<20)+"overflow\n")
	child := integrationHandoff(t, copies["lending"])
	if diff, err := w.Changes(ctx, copies["lending"], child); err == nil || diff != "" || !strings.Contains(err.Error(), "delivery limit") {
		t.Fatalf("over-limit real Git diff was allocated/delivered: %d bytes, %v", len(diff), err)
	}
}

type scriptedCopies struct {
	copies map[run.WorkspaceID]*assignmentWorkspace
}

func (s scriptedCopies) PrepareAssignment(_ context.Context, spec run.WorkspaceSpec) (run.AssignmentWorkspace, error) {
	copy := s.copies[spec.ID]
	if copy == nil {
		return nil, errors.New("unknown scripted Agent copy")
	}
	return copy, nil
}

type scriptedAgents struct {
	roots    map[run.WorkspaceID]string
	sessions map[string]*scriptedSession
}

func (s *scriptedAgents) OpenAssignment(_ context.Context, spec run.AssignmentSpec, handler run.ToolHandler) (run.AssignmentSession, error) {
	session := &scriptedSession{id: spec.ID, root: s.roots[run.WorkspaceID(spec.ID)], handler: handler}
	if session.root == "" {
		return nil, errors.New("scripted Agent has no independent copy")
	}
	s.sessions[spec.ID] = session
	return session, nil
}

type scriptedSession struct {
	id       string
	root     string
	handler  run.ToolHandler
	messages []string
}

func (s *scriptedSession) Message(ctx context.Context, message string, _ time.Duration) (run.TurnResult, error) {
	if err := ctx.Err(); err != nil {
		return run.TurnResult{}, err
	}
	s.messages = append(s.messages, message)
	final := message
	switch message {
	case "library foundation":
		if err := os.WriteFile(filepath.Join(s.root, "library.txt"), []byte("public contract\n"), 0644); err != nil {
			return run.TurnResult{}, err
		}
		if err := os.WriteFile(filepath.Join(s.root, "common.txt"), []byte("parent decision\n"), 0644); err != nil {
			return run.TurnResult{}, err
		}
	case "lending first":
		if err := os.WriteFile(filepath.Join(s.root, "lending.txt"), []byte("incorrect limit\n"), 0644); err != nil {
			return run.TurnResult{}, err
		}
	case "correct lending":
		if err := os.WriteFile(filepath.Join(s.root, "lending.txt"), []byte("loan limit = 3\n"), 0644); err != nil {
			return run.TurnResult{}, err
		}
	case "storage question":
		answer := s.handler(ctx, run.AskParentTool, json.RawMessage(`{"question":"Which storage contract?"}`))
		if answer.Refused {
			return run.TurnResult{}, errors.New(answer.Text)
		}
		final = answer.Text
	case "storage conflicting result":
		if err := os.WriteFile(filepath.Join(s.root, "common.txt"), []byte("child decision\n"), 0644); err != nil {
			return run.TurnResult{}, err
		}
	case "correct storage conflict":
		if err := os.WriteFile(filepath.Join(s.root, "common.txt"), []byte("shared baseline\n"), 0644); err != nil {
			return run.TurnResult{}, err
		}
		if err := os.WriteFile(filepath.Join(s.root, "storage.txt"), []byte("loan records\n"), 0644); err != nil {
			return run.TurnResult{}, err
		}
	default:
		if strings.Contains(message, "asks: Which storage contract?") {
			final = "Use the public storage contract"
		}
	}
	return run.TurnResult{Final: final, Completed: true}, nil
}

func (*scriptedSession) Close(context.Context) (run.TurnResult, error) {
	return run.TurnResult{Stopped: true}, nil
}

func TestControlledRootLibraryTwoChildrenThroughGitHandoffs(t *testing.T) {
	ctx := context.Background()
	w, copies, roots := integrationFixture(t, "root", "library", "lending", "storage")
	agents := &scriptedAgents{roots: roots, sessions: make(map[string]*scriptedSession)}
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}},
		{Path: "library", Parent: ".", Children: []string{"library/lending", "library/storage"}},
		{Path: "library/lending", Parent: "library"},
		{Path: "library/storage", Parent: "library"},
	}
	flow, err := run.NewAssignmentHandoffs(team, scriptedCopies{copies}, agents, w)
	if err != nil {
		t.Fatal(err)
	}
	request := func(id string) run.AssignmentRequest {
		return run.AssignmentRequest{ID: run.WorkspaceID(id), Profile: "controlled", Home: "home-" + id}
	}
	if err := flow.StartRoot(ctx, request("root")); err != nil {
		t.Fatal(err)
	}
	if _, err := flow.Message(ctx, ".", "root work", time.Second); err != nil {
		t.Fatal(err)
	}
	if err := flow.Delegate(ctx, ".", "library", request("library")); err != nil {
		t.Fatal(err)
	}
	if _, err := flow.Message(ctx, "library", "library foundation", time.Second); err != nil {
		t.Fatal(err)
	}
	for _, child := range []struct{ path, id string }{{"library/lending", "lending"}, {"library/storage", "storage"}} {
		if err := flow.Delegate(ctx, "library", child.path, request(child.id)); err != nil {
			t.Fatal(err)
		}
	}
	question, err := flow.Message(ctx, "library/storage", "storage question", time.Second)
	if err != nil || question.Final != "Use the public storage contract" {
		t.Fatalf("parent question/answer: %+v %v", question, err)
	}
	for _, work := range []struct{ path, message string }{{"library/lending", "lending first"}, {"library/storage", "storage conflicting result"}} {
		if _, err := flow.Message(ctx, work.path, work.message, time.Second); err != nil {
			t.Fatal(err)
		}
		if _, err := flow.Submit(ctx, work.path); err != nil {
			t.Fatal(err)
		}
	}
	_, firstLending, err := flow.Changes(ctx, "library", "library/lending")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := flow.Return(ctx, "library", "library/lending", "correct lending", time.Second); err != nil {
		t.Fatal(err)
	}
	correctedLending, err := flow.Submit(ctx, "library/lending")
	if err != nil || correctedLending.ResultCommit == firstLending.ResultCommit {
		t.Fatalf("child correction did not retain named results: %+v %v", correctedLending, err)
	}
	if _, _, err := flow.Changes(ctx, "library", "library/lending"); err != nil {
		t.Fatal(err)
	}
	firstParent, err := flow.Accept(ctx, "library", "library/lending", correctedLending.ResultCommit)
	if err != nil || firstParent == correctedLending.ResultCommit {
		t.Fatalf("parent did not integrate corrected child into its own stand: %q %v", firstParent, err)
	}
	_, firstStorage, err := flow.Changes(ctx, "library", "library/storage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := flow.Accept(ctx, "library", "library/storage", firstStorage.ResultCommit); !errors.Is(err, ErrIntegrationConflict) {
		t.Fatalf("real conflict did not return to owning child: %v", err)
	}
	if _, err := flow.Return(ctx, "library", "library/storage", "correct storage conflict", time.Second); err != nil {
		t.Fatal(err)
	}
	correctedStorage, err := flow.Submit(ctx, "library/storage")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := flow.Changes(ctx, "library", "library/storage"); err != nil {
		t.Fatal(err)
	}
	if _, err := flow.Accept(ctx, "library", "library/storage", correctedStorage.ResultCommit); err != nil {
		t.Fatal(err)
	}
	library, err := flow.Submit(ctx, "library")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := flow.Changes(ctx, ".", "library"); err != nil {
		t.Fatal(err)
	}
	rootIntegrated, err := flow.Accept(ctx, ".", "library", library.ResultCommit)
	if err != nil || rootIntegrated == library.ResultCommit {
		t.Fatalf("Root did not integrate Library's result: %q %v", rootIntegrated, err)
	}
	if _, err := flow.Submit(ctx, "."); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"library.txt": "public contract\n", "lending.txt": "loan limit = 3\n", "storage.txt": "loan records\n", "common.txt": "parent decision\n"} {
		got, err := os.ReadFile(filepath.Join(roots["root"], name))
		if err != nil || string(got) != want {
			t.Fatalf("Root's integrated %s = %q: %v", name, got, err)
		}
	}
	if len(agents.sessions["library"].messages) < 2 || len(agents.sessions["lending"].messages) != 2 || len(agents.sessions["storage"].messages) != 3 {
		t.Fatalf("missing parent/child messages: library=%v lending=%v storage=%v", agents.sessions["library"].messages, agents.sessions["lending"].messages, agents.sessions["storage"].messages)
	}
}
