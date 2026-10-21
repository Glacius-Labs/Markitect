package cockpit

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeRun is an execution function the test steers: it blocks until its
// context ends or release is closed, then returns the prepared result.
type fakeRun struct {
	started   chan struct{}
	release   chan struct{}
	finished  atomic.Bool
	calls     atomic.Int32
	cancelled atomic.Bool
	result    RunResult
	// delay keeps the execution busy after its context ended, as writing the
	// report does.
	delay time.Duration
}

func newFakeRun() *fakeRun {
	return &fakeRun{started: make(chan struct{}, 4), release: make(chan struct{}), result: RunResult{Outcome: "incomplete"}}
}

func (f *fakeRun) execute(ctx context.Context) RunResult {
	f.calls.Add(1)
	f.started <- struct{}{}
	select {
	case <-ctx.Done():
		f.cancelled.Store(true)
		time.Sleep(f.delay)
	case <-f.release:
	}
	f.finished.Store(true)
	return f.result
}

type runFixture struct {
	state string
	fake  *fakeRun
	runs  *RunController
	h     http.Handler
	held  atomic.Value // string: the reason a start is refused
	facts atomic.Int32
}

func newRunFixture(t *testing.T, base context.Context) *runFixture {
	t.Helper()
	f := &runFixture{state: t.TempDir(), fake: newFakeRun()}
	f.held.Store("")
	f.runs = NewRunController(base, RunControlConfig{
		StateDir: f.state,
		Facts: func(context.Context) (RunFacts, error) {
			f.facts.Add(1)
			return RunFacts{Budget: RunBudget{ExecutionTime: "25m", TurnTimeout: "5m", AgentTurns: 40, Passes: 3}, Profile: "execute", Model: "model-x"}, nil
		},
		Execute: f.fake.execute,
		Held:    func() (string, error) { return f.held.Load().(string), nil },
	})
	srv, err := New(Config{States: []string{f.state}, Runs: f.runs})
	if err != nil {
		t.Fatal(err)
	}
	srv.host = local
	f.h = srv.Handler()
	return f
}

func (f *runFixture) status(t *testing.T) runControlView {
	t.Helper()
	var v runControlView
	if rec := get(t, f.h, local, "/api/run-control", &v); rec.Code != http.StatusOK {
		t.Fatalf("run-control = %d", rec.Code)
	}
	return v
}

func (f *runFixture) name(t *testing.T, run string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.state, "current-run"), []byte(run+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRunControlStartsRunsAndStopCancelsItsContext(t *testing.T) {
	f := newRunFixture(t, context.Background())
	f.name(t, "20261020T100000Z-old") // an earlier run is not this one
	if v := f.status(t); !v.Available || v.Run != nil {
		t.Fatalf("before a start = %+v", v)
	}
	rec := post(f.h, local, "", "start-run", "/api/run-control/start", ``)
	if rec.Code != http.StatusOK {
		t.Fatalf("start = %d %s", rec.Code, rec.Body)
	}
	<-f.fake.started
	if v := f.status(t); v.Run == nil || v.Run.State != RunStarting || v.Run.Run != "" {
		t.Fatalf("before the run names itself = %+v", v.Run)
	}
	f.name(t, "20261021T100000Z-new")
	if v := f.status(t); v.Run.State != RunRunning || v.Run.Run != "20261021T100000Z-new" {
		t.Fatalf("running = %+v", v.Run)
	}

	f.fake.result = RunResult{Outcome: "incomplete", Evidence: filepath.Join(f.state, "runs", "20261021T100000Z-new"), Error: "the run was cancelled"}
	f.fake.delay = 50 * time.Millisecond
	if rec := post(f.h, local, "", "stop-run", "/api/run-control/stop", ``); rec.Code != http.StatusOK {
		t.Fatalf("stop = %d %s", rec.Code, rec.Body)
	}
	if v := f.status(t); v.Run.State != RunStopping {
		t.Fatalf("while the execution writes its report = %+v", v.Run)
	}
	waitFor(t, "the run to end", func() bool { return f.status(t).Run.State == RunEnded })
	end := f.status(t).Run
	if !f.fake.cancelled.Load() || end.Outcome != "incomplete" || end.Run != "20261021T100000Z-new" || end.Ended.IsZero() ||
		end.Report != filepath.Join(f.state, "runs", "20261021T100000Z-new", "report.json") || end.Error != "the run was cancelled" {
		t.Fatalf("ended = %+v cancelled=%v", end, f.fake.cancelled.Load())
	}
	// Stopping what ended is refused; the result stays for display.
	if rec := post(f.h, local, "", "stop-run", "/api/run-control/stop", ``); rec.Code != http.StatusConflict {
		t.Fatalf("stop after the end = %d", rec.Code)
	}
	// A new start replaces the result.
	f.fake.result = RunResult{Outcome: "success"}
	if rec := post(f.h, local, "", "start-run", "/api/run-control/start", ``); rec.Code != http.StatusOK {
		t.Fatalf("second start = %d %s", rec.Code, rec.Body)
	}
	<-f.fake.started
	close(f.fake.release)
	waitFor(t, "the second run to end", func() bool { return f.status(t).Run.State == RunEnded })
	// The second run never named itself: current-run still holds the first.
	if v := f.status(t); v.Run.Outcome != "success" || v.Run.Run != "" {
		t.Fatalf("second run = %+v", v.Run)
	}
}

func TestRunControlRefusesWhileTheStateIsHeldOrARunIsActive(t *testing.T) {
	f := newRunFixture(t, context.Background())
	f.held.Store("an Apply holds the state directory")
	rec := post(f.h, local, "", "start-run", "/api/run-control/start", ``)
	if rec.Code != http.StatusConflict || f.fake.calls.Load() != 0 || !strings.Contains(rec.Body.String(), "an Apply holds the state directory") {
		t.Fatalf("held start = %d %s, calls %d", rec.Code, rec.Body, f.fake.calls.Load())
	}
	if f.status(t).Run != nil {
		t.Fatal("a refused start left a run behind")
	}

	f.held.Store("")
	if rec := post(f.h, local, "", "start-run", "/api/run-control/start", ``); rec.Code != http.StatusOK {
		t.Fatalf("start = %d %s", rec.Code, rec.Body)
	}
	<-f.fake.started
	if rec := post(f.h, local, "", "start-run", "/api/run-control/start", ``); rec.Code != http.StatusConflict || f.fake.calls.Load() != 1 {
		t.Fatalf("start while active = %d, calls %d", rec.Code, f.fake.calls.Load())
	}
	close(f.fake.release)
	waitFor(t, "the run to end", func() bool { return f.status(t).Run.State == RunEnded })
}

func TestRunControlShowsARefusalOfTheExecution(t *testing.T) {
	f := newRunFixture(t, context.Background())
	f.name(t, "20261021T100000Z-refused")
	f.fake.result = RunResult{Outcome: RunRefused, Error: "the working tree has uncommitted changes"}
	close(f.fake.release)
	if rec := post(f.h, local, "", "start-run", "/api/run-control/start", ``); rec.Code != http.StatusOK {
		t.Fatalf("start = %d", rec.Code)
	}
	waitFor(t, "the refusal", func() bool { return f.status(t).Run.State == RunEnded })
	if run := f.status(t).Run; run.Outcome != RunRefused || run.Error != "the working tree has uncommitted changes" || run.Run != "" || run.Report != "" {
		t.Fatalf("refused run = %+v", run)
	}
}

func TestRunControlEndsItsRunAndWaitsWhenTheCockpitStops(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	f := newRunFixture(t, ctx)
	f.fake.delay = 100 * time.Millisecond
	f.fake.result = RunResult{Outcome: "incomplete", Error: "the run was cancelled"}
	if _, err := f.runs.Start(); err != nil {
		t.Fatal(err)
	}
	<-f.fake.started
	stop()
	f.runs.Wait()
	if !f.fake.cancelled.Load() || !f.fake.finished.Load() {
		t.Fatalf("Wait returned before the execution returned: cancelled=%v finished=%v", f.fake.cancelled.Load(), f.fake.finished.Load())
	}
	if st := f.runs.Status(); st == nil || st.State != RunEnded || st.Outcome != "incomplete" {
		t.Fatalf("status after Wait = %+v", st)
	}
	if _, err := f.runs.Start(); err == nil || f.fake.calls.Load() != 1 {
		t.Fatalf("a stopping cockpit started a run: %v", err)
	}
	// Waiting without a run returns at once.
	newRunFixture(t, context.Background()).runs.Wait()
}

func TestRunControlKeepsAPanickingExecutionFromHangingTheHandle(t *testing.T) {
	f := newRunFixture(t, context.Background())
	f.runs.cfg.Execute = func(context.Context) RunResult { panic("boom") }
	if _, err := f.runs.Start(); err != nil {
		t.Fatal(err)
	}
	f.runs.Wait()
	if st := f.runs.Status(); st.State != RunEnded || !strings.Contains(st.Error, "boom") {
		t.Fatalf("status = %+v", st)
	}
}

func TestRunControlFactsAreReadOnlyWhenAskedFor(t *testing.T) {
	f := newRunFixture(t, context.Background())
	f.status(t)
	f.status(t)
	if f.facts.Load() != 0 {
		t.Fatalf("polling read the facts %d times", f.facts.Load())
	}
	var v runControlView
	get(t, f.h, local, "/api/run-control?facts=1", &v)
	if f.facts.Load() != 1 || v.Facts == nil || v.Facts.Budget.AgentTurns != 40 || v.Facts.Model != "model-x" || v.Facts.Profile != "execute" {
		t.Fatalf("facts = %+v (%d reads)", v.Facts, f.facts.Load())
	}
}

func TestRunControlWritesNeedTheCockpitsOwnRequest(t *testing.T) {
	f := newRunFixture(t, context.Background())
	for name, rec := range map[string]int{
		"no header":      post(f.h, local, "", "", "/api/run-control/start", ``).Code,
		"wrong purpose":  post(f.h, local, "", "start", "/api/run-control/start", ``).Code,
		"foreign origin": post(f.h, local, "http://evil.example", "start-run", "/api/run-control/start", ``).Code,
		"stop no header": post(f.h, local, "", "", "/api/run-control/stop", ``).Code,
		"stop as start":  post(f.h, local, "", "start-run", "/api/run-control/stop", ``).Code,
	} {
		if rec != http.StatusForbidden {
			t.Errorf("%s = %d, want 403", name, rec)
		}
	}
	if rec := post(f.h, "evil.example", "", "start-run", "/api/run-control/start", ``); rec.Code != http.StatusMisdirectedRequest {
		t.Errorf("foreign host = %d", rec.Code)
	}
	if f.fake.calls.Load() != 0 {
		t.Fatal("a refused write started a run")
	}
	// The write routes take only POST; a GET reaches the page files and finds
	// nothing there.
	for _, target := range []string{"/api/run-control/start", "/api/run-control/stop"} {
		if rec := get(t, f.h, local, target, nil); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d", target, rec.Code)
		}
	}
	if rec := post(f.h, local, "", "start-run", "/api/run-control", ``); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/run-control = %d", rec.Code)
	}
	// Stopping without a run of this cockpit is refused, not ignored.
	if rec := post(f.h, local, "", "stop-run", "/api/run-control/stop", ``); rec.Code != http.StatusConflict {
		t.Errorf("stop without a run = %d", rec.Code)
	}
}

func TestRunControlIsUnavailableWithoutAController(t *testing.T) {
	srv, err := New(Config{States: []string{t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	srv.host = local
	h := srv.Handler()
	var v runControlView
	get(t, h, local, "/api/run-control", &v)
	if v.Available || v.Run != nil || !strings.Contains(v.Reason, "--run-config") || !strings.Contains(v.Reason, "--runtime-profile") {
		t.Fatalf("view = %+v", v)
	}
	for _, purpose := range []string{"start-run", "stop-run"} {
		target := "/api/run-control/" + purpose[:len(purpose)-4]
		if rec := post(h, local, "", purpose, target, ``); rec.Code != http.StatusNotImplemented {
			t.Errorf("%s without a controller = %d", target, rec.Code)
		}
	}
	// The view encodes as a JSON object even when unavailable.
	var raw map[string]any
	rec := get(t, h, local, "/api/run-control", nil)
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil || raw["run"] != nil {
		t.Fatalf("raw = %v %v", raw, err)
	}
}
