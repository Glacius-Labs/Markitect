package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

const cockpitHost = "127.0.0.1:7341"

// runControlFixture is a cockpit over a repository with a prepared Runtime,
// whose execution function is a fake that waits for its context. It starts no
// provider and no container.
type runControlFixture struct {
	setup    setupFixture
	handler  http.Handler
	runs     *cockpit.RunController
	fence    *fencedStates
	state    string
	selected apply.InputSelection
	profile  string
	config   string

	mu       sync.Mutex
	requests []cli.ApplyRequest
	started  chan struct{}
	cancel   chan struct{} // closed when the execution's context ended
}

func newRunControlFixture(t *testing.T, ctx context.Context, prepare ...func(*testing.T, setupFixture)) *runControlFixture {
	t.Helper()
	f := &runControlFixture{setup: newSetupFixture(t), fence: &fencedStates{held: map[string]bool{}}, started: make(chan struct{}, 2), cancel: make(chan struct{})}
	for _, fn := range prepare {
		fn(t, f.setup)
	}
	if code, out, errout := f.setup.run(t); code != 0 {
		t.Fatalf("setup exit %d\n%s\n%s", code, out, errout)
	}
	f.profile, f.config = filepath.Join(f.setup.runtimeDir, "runtime.yaml"), filepath.Join(f.setup.runtimeDir, "run.json")
	cfg, err := loadRunConfig(context.Background(), f.config)
	if err != nil {
		t.Fatal(err)
	}
	f.state = cfg.StateDirectory
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	fake := f.setup.fake
	service := cockpitService{
		preparation: apply.NewPreparationService(apply.NewModelService(repository, repository)),
		repository:  repository,
		agents:      agentMetadataRouter{claude: claudecode.NewMetadataPreparer(func(claudecode.EngineConfig) (claudecode.ImageReader, error) { return fake, nil })},
		checks:      checks.NewMetadataPreparer(func(checks.EngineConfig) (checks.ImageReader, error) { return fake, nil }),
		execute: func(ctx context.Context, request cli.ApplyRequest) (cli.ApplyReport, error) {
			f.mu.Lock()
			f.requests = append(f.requests, request)
			f.mu.Unlock()
			f.started <- struct{}{}
			<-ctx.Done()
			close(f.cancel)
			return cli.ApplyReport{Status: model.NotCompleted, EvidenceDirectory: filepath.Join(f.state, "runs", "20261021T100000Z-aaaa")}, ctx.Err()
		},
	}
	f.selected, err = repositorySelection(f.setup.repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	f.runs = service.runController(ctx, f.selected, f.config, f.profile, f.state, "test-build", f.fence)
	server, err := cockpit.New(cockpit.Config{States: []string{f.state}, Runs: f.runs})
	if err != nil {
		t.Fatal(err)
	}
	f.handler = server.Handler()
	return f
}

func (f *runControlFixture) call(method, target, purpose string, body ...string) *httptest.ResponseRecorder {
	var reader io.Reader
	if len(body) > 0 {
		reader = strings.NewReader(body[0])
	}
	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Host = cockpitHost
	if purpose != "" {
		req.Header.Set("X-Markitect-Cockpit", purpose)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

// start posts the start with the committed head of the repository, as a page
// that read the confirmation at this head does.
func (f *runControlFixture) start() *httptest.ResponseRecorder {
	return f.call(http.MethodPost, "/api/run-control/start", "start-run", `{"head":"`+f.setup.repo.Git("rev-parse", "HEAD")+`"}`)
}

func (f *runControlFixture) view(t *testing.T, target string) (view struct {
	Available  bool
	Facts      *cockpit.RunFacts
	FactsError string
	Run        *cockpit.RunStatus
}) {
	t.Helper()
	rec := f.call(http.MethodGet, target, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("%s = %d %s", target, rec.Code, rec.Body)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	return view
}

func TestCockpitConfirmsTheBudgetTheDeadlineAndTheAgentProfileBeforeAStart(t *testing.T) {
	f := newRunControlFixture(t, context.Background())
	view := f.view(t, "/api/run-control?facts=1")
	facts := view.Facts
	if !view.Available || facts == nil || view.FactsError != "" {
		t.Fatalf("view = %+v", view)
	}
	// The head of the confirmation is the one the start is checked against.
	if head := f.setup.repo.Git("rev-parse", "HEAD"); facts.Head != head {
		t.Fatalf("facts head %q, repository HEAD %q", facts.Head, head)
	}
	if len(facts.Profiles) != 1 || facts.Profiles[0].Name != "markitect-agent" || facts.Profiles[0].Model != "claude-haiku-5-5" || len(facts.Profiles[0].Managers) == 0 || facts.Profiles[0].Managers[0] != "." || facts.Budget.AgentTurns <= 0 || facts.Budget.Passes <= 0 ||
		facts.Budget.ExecutionTime == "" || facts.Budget.TurnTimeout == "" || facts.Head == "" || facts.Deadline != "" {
		t.Fatalf("facts = %+v", facts)
	}
	// A working tree with uncommitted changes shows why no run would start.
	f.setup.repo.Write("README.md", "changed but not committed\n")
	view = f.view(t, "/api/run-control?facts=1")
	if view.Facts != nil || !strings.Contains(view.FactsError, "uncommitted") && !strings.Contains(view.FactsError, "not clean") {
		t.Fatalf("a dirty tree = %+v", view)
	}
}

func TestCockpitConfirmationListsEveryDistinctExecuteProfile(t *testing.T) {
	f := newRunControlFixture(t, context.Background(), func(t *testing.T, setup setupFixture) {
		execution, err := os.ReadFile(filepath.Join(setup.repo.Dir, ".markitect", "execution.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ReplaceAll(string(execution), "\r\n", "\n")
		if !strings.Contains(text, "packages: {}") {
			t.Fatal("fixture execution declarations changed")
		}
		setup.repo.Write(".markitect/execution.yaml", strings.Replace(text, "packages: {}", "packages:\n  product/cli:\n    profiles: {execute: second-agent}", 1))
		setup.repo.Commit("select a second Execute profile for one Package")
	})
	facts := f.view(t, "/api/run-control?facts=1").Facts
	if facts == nil || len(facts.Profiles) != 2 || facts.Profiles[0].Name != "markitect-agent" || facts.Profiles[1].Name != "second-agent" ||
		!slices.Equal(facts.Profiles[1].Managers, []string{"product/cli"}) || facts.Profiles[1].Model != "claude-haiku-5-5" {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestCockpitRefusesAStartAfterAnotherCommit(t *testing.T) {
	f := newRunControlFixture(t, context.Background())
	confirmed := f.setup.repo.Git("rev-parse", "HEAD")
	f.setup.repo.Write("NOTES.md", "a later commit\n")
	f.setup.repo.Commit("commit after the confirmation")
	rec := f.call(http.MethodPost, "/api/run-control/start", "start-run", `{"head":"`+confirmed+`"}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "committed source changed since the confirmation") {
		t.Fatalf("start after a commit = %d %s", rec.Code, rec.Body)
	}
	f.mu.Lock()
	calls := len(f.requests)
	f.mu.Unlock()
	if calls != 0 {
		t.Fatal("a stale confirmation reached the execution")
	}
}

func TestCockpitStartsTheSameApplyAndStopCancelsIt(t *testing.T) {
	f := newRunControlFixture(t, context.Background())
	if rec := f.start(); rec.Code != http.StatusOK {
		t.Fatalf("start = %d %s", rec.Code, rec.Body)
	}
	<-f.started
	f.mu.Lock()
	request := f.requests[0]
	f.mu.Unlock()
	if request.RunConfig != f.config || request.Selection.Repository != f.setup.repo.Dir || request.Selection.Local != f.selected.Local ||
		len(request.Selection.RuntimeProfiles) != 1 || request.Selection.RuntimeProfiles[0].Boundary != f.setup.runtimeDir || request.Selection.RuntimeProfiles[0].Path != mustPath(t, "runtime.yaml") || request.Build != "test-build" {
		t.Fatalf("the request differs from markitect apply's: %+v", request)
	}
	if rec := f.start(); rec.Code != http.StatusConflict {
		t.Fatalf("a second start = %d", rec.Code)
	}
	if rec := f.call(http.MethodPost, "/api/run-control/stop", "stop-run"); rec.Code != http.StatusOK {
		t.Fatalf("stop = %d %s", rec.Code, rec.Body)
	}
	<-f.cancel
	deadline := time.Now().Add(5 * time.Second)
	for f.view(t, "/api/run-control").Run.State != cockpit.RunEnded {
		if time.Now().After(deadline) {
			t.Fatal("the run did not end")
		}
		time.Sleep(5 * time.Millisecond)
	}
	run := f.view(t, "/api/run-control").Run
	if run.Outcome != "incomplete" || run.Run != "20261021T100000Z-aaaa" || run.Report != filepath.Join(f.state, "runs", "20261021T100000Z-aaaa", "report.json") || run.Error == "" {
		t.Fatalf("ended = %+v", run)
	}
}

func TestCockpitRefusesToStartWhileTheStateIsHeld(t *testing.T) {
	f := newRunControlFixture(t, context.Background())
	// A proposal publication holds the state directory through the fence.
	f.fence.set([]string{f.state}, true)
	rec := f.start()
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "publishing") {
		t.Fatalf("start during a publication = %d %s", rec.Code, rec.Body)
	}
	f.fence.set([]string{f.state}, false)
	// Another process, here a lock holder, runs an Apply.
	held, err := lockState(f.state)
	if err != nil {
		t.Fatal(err)
	}
	rec = f.start()
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "an Apply holds the state directory") {
		t.Fatalf("start while an Apply holds the state = %d %s", rec.Code, rec.Body)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	calls := len(f.requests)
	f.mu.Unlock()
	if calls != 0 {
		t.Fatalf("a refused start reached the execution %d times", calls)
	}
}

func TestCockpitServiceEndsItsRunWhenItStops(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	f := newRunControlFixture(t, ctx)
	if _, err := f.runs.Start(context.Background(), f.setup.repo.Git("rev-parse", "HEAD")); err != nil {
		t.Fatal(err)
	}
	<-f.started
	stop()
	f.runs.Wait()
	select {
	case <-f.cancel:
	default:
		t.Fatal("the execution's context did not end with the cockpit's")
	}
	if st := f.runs.Status(); st == nil || st.State != cockpit.RunEnded {
		t.Fatalf("status = %+v", st)
	}
}

func TestRunResultSeparatesRefusalsFromRuns(t *testing.T) {
	refused := runResult(cli.ApplyReport{Status: model.NotCompleted}, errors.New("the working tree has uncommitted changes"))
	if refused.Outcome != cockpit.RunRefused || refused.Evidence != "" || !strings.Contains(refused.Error, "uncommitted") {
		t.Fatalf("refused = %+v", refused)
	}
	cancelled := runResult(cli.ApplyReport{Status: model.NotCompleted, EvidenceDirectory: "/state/runs/r1"}, context.Canceled)
	if cancelled.Outcome != "incomplete" || cancelled.Evidence != "/state/runs/r1" || cancelled.Error == "" {
		t.Fatalf("cancelled = %+v", cancelled)
	}
	done := runResult(cli.ApplyReport{Status: model.Passed, EvidenceDirectory: "/state/runs/r2"}, nil)
	if done.Outcome != "success" || done.Error != "" {
		t.Fatalf("done = %+v", done)
	}
	if got := runResult(cli.ApplyReport{}, nil); got.Outcome != cockpit.RunRefused || got.Error == "" {
		t.Fatalf("empty = %+v", got)
	}
}

func TestDurationTextKeepsTheSettingsNotation(t *testing.T) {
	for in, want := range map[time.Duration]string{25 * time.Minute: "25m", 90 * time.Minute: "1h30m", time.Hour: "1h", 90 * time.Second: "1m30s", 30 * time.Second: "30s"} {
		if got := durationText(in); got != want {
			t.Errorf("%s = %q, want %q", in, got, want)
		}
	}
}
