package cockpit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"
)

// RunFacts are what the owner confirms before a run starts: the committed
// execution budget, the run configuration's deadline and the Execute Agent
// profile with the model its Runtime profile names. Bootstrap reads them with
// the product's own decoders when the confirmation is asked for.
type RunFacts struct {
	Head     string    `json:"head,omitempty"`
	Budget   RunBudget `json:"budget"`
	Deadline string    `json:"deadline,omitempty"`
	Profile  string    `json:"profile"`
	Model    string    `json:"model,omitempty"`
}

// RunBudget is the execution budget of the committed .markitect/execution.yaml.
type RunBudget struct {
	ExecutionTime string `json:"executionTime"`
	TurnTimeout   string `json:"turnTimeout"`
	AgentTurns    int    `json:"agentTurns"`
	Passes        int    `json:"passes"`
}

// RunResult is what the execution function reported when it returned. Outcome
// is the run outcome (success, failed or incomplete) or refused when the
// execution refused before it created evidence. Evidence is the run's
// evidence directory, empty for a refusal.
type RunResult struct {
	Outcome  string
	Evidence string
	Error    string
}

// RunControlConfig composes the controller. Execute is the same Apply that
// markitect apply runs, with the cockpit's own inputs; it returns when the
// run ended and its report is written. Held reports why a start is refused
// because something holds the state directory, or an empty reason.
type RunControlConfig struct {
	StateDir string
	Facts    func(context.Context) (RunFacts, error)
	Execute  func(context.Context) RunResult
	Held     func() (string, error)
}

// RunRefused is the outcome of an execution that refused before it created
// evidence, such as for uncommitted changes.
const RunRefused = "refused"

// Run states of a run the cockpit started.
const (
	RunStarting = "starting"
	RunRunning  = "running"
	RunStopping = "stopping"
	RunEnded    = "ended"
)

// RunStatus is the run this cockpit started, from its start until the next
// start replaces it. Run is the identity from the state directory's
// current-run once the run named itself there.
type RunStatus struct {
	State   string    `json:"state"`
	Run     string    `json:"run,omitempty"`
	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended,omitzero"`
	Outcome string    `json:"outcome,omitempty"`
	Error   string    `json:"error,omitempty"`
	Report  string    `json:"report,omitempty"`
}

var (
	// ErrRunActive refuses a start while a run this cockpit started is active.
	ErrRunActive = errors.New("a run started in this cockpit is still active")
	// ErrRunHeld refuses a start while something holds the state directory.
	ErrRunHeld = errors.New("the state directory is held")
	// ErrNoActiveRun refuses a stop without an active run of this cockpit.
	ErrNoActiveRun = errors.New("this cockpit has no active run to stop")
	// ErrRunEnding refuses a start while the cockpit itself ends.
	ErrRunEnding = errors.New("the cockpit is ending")
)

// RunController starts and stops the one Apply this cockpit runs in its own
// process. The run's context derives from the cockpit's; cancelling it makes
// the run end incomplete with its report and evidence, as an interrupted
// markitect apply. The handle stays until Execute returned, and its result
// stays for display until the next start.
type RunController struct {
	cfg  RunControlConfig
	base context.Context
	wg   sync.WaitGroup

	mu     sync.Mutex
	cancel context.CancelFunc
	cur    *runHandle
}

type runHandle struct {
	status  RunStatus
	before  string // current-run before the start; the run's own differs
	stopped bool
}

// NewRunController composes a controller whose runs end with ctx.
func NewRunController(ctx context.Context, cfg RunControlConfig) *RunController {
	return &RunController{cfg: cfg, base: ctx}
}

func (c *RunController) current() string {
	return (&stateDirectory{dir: c.cfg.StateDir}).currentRun()
}

// Start begins the run unless the cockpit ends, a run it started is active, or
// something holds the state directory. Execute's own lock remains the real
// guard; the probe only gives the owner the reason at once. Whatever Execute
// itself refuses, such as uncommitted changes, shows as the ended run's error.
func (c *RunController) Start() (RunStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.base.Err() != nil {
		return RunStatus{}, ErrRunEnding
	}
	if c.cur != nil && c.cur.status.State != RunEnded {
		return c.statusLocked(), ErrRunActive
	}
	reason, err := c.cfg.Held()
	if err != nil {
		return RunStatus{}, fmt.Errorf("the state directory could not be probed: %w", err)
	}
	if reason != "" {
		return RunStatus{}, fmt.Errorf("%w: %s", ErrRunHeld, reason)
	}
	ctx, cancel := context.WithCancel(c.base)
	h := &runHandle{status: RunStatus{State: RunStarting, Started: time.Now().UTC()}, before: c.current()}
	c.cur, c.cancel = h, cancel
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer cancel()
		c.finish(h, c.execute(ctx))
	}()
	return c.statusLocked(), nil
}

// execute runs the function and turns a panic into a failed result, so the
// handle always ends.
func (c *RunController) execute(ctx context.Context) (result RunResult) {
	defer func() {
		if r := recover(); r != nil {
			result = RunResult{Outcome: "incomplete", Error: fmt.Sprint("the run function failed: ", r)}
		}
	}()
	return c.cfg.Execute(ctx)
}

func (c *RunController) finish(h *runHandle, result RunResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if h.status.Run == "" {
		if run := c.current(); run != "" && run != h.before {
			h.status.Run = run
		}
	}
	if result.Evidence != "" {
		h.status.Run = filepath.Base(result.Evidence)
		h.status.Report = filepath.Join(result.Evidence, "report.json")
	} else if result.Outcome == RunRefused {
		h.status.Run = "" // a refused run left no evidence to open
	}
	h.status.State, h.status.Ended = RunEnded, time.Now().UTC()
	h.status.Outcome, h.status.Error = result.Outcome, result.Error
}

// Stop cancels the run this cockpit started. The run ends incomplete with its
// evidence and report; nothing is restored. Stopping twice is harmless.
func (c *RunController) Stop() (RunStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cur == nil || c.cur.status.State == RunEnded {
		return RunStatus{}, ErrNoActiveRun
	}
	c.cur.stopped = true
	c.cancel()
	return c.statusLocked(), nil
}

// Status is the run this cockpit started, or nil when it started none.
func (c *RunController) Status() *RunStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cur == nil {
		return nil
	}
	s := c.statusLocked()
	return &s
}

// statusLocked refreshes the run identity and the state from the state
// directory: a run names itself in current-run once it holds the lock.
func (c *RunController) statusLocked() RunStatus {
	h := c.cur
	if h.status.State == RunEnded {
		return h.status
	}
	if h.status.Run == "" {
		if run := c.current(); run != "" && run != h.before {
			h.status.Run = run
		}
	}
	switch {
	case h.stopped:
		h.status.State = RunStopping
	case h.status.Run != "":
		h.status.State = RunRunning
	default:
		h.status.State = RunStarting
	}
	return h.status
}

// Wait ends the run this cockpit started, if any, and waits until Execute
// returned, so that its report and evidence are written before the cockpit
// exits.
func (c *RunController) Wait() {
	c.mu.Lock()
	if c.cur != nil && c.cur.status.State != RunEnded {
		c.cur.stopped = true
		c.cancel()
	}
	c.mu.Unlock()
	c.wg.Wait()
}

// runControlView is the answer of GET /api/run-control.
type runControlView struct {
	Available  bool       `json:"available"`
	Reason     string     `json:"reason,omitempty"`
	Run        *RunStatus `json:"run"`
	Facts      *RunFacts  `json:"facts,omitempty"`
	FactsError string     `json:"factsError,omitempty"`
}

const runControlUnavailable = "Starting a run needs the cockpit to be started with --run-config PATH, --runtime-profile PATH and --repo PATH (absolute run configuration and Runtime profile paths)."

// factsTimeout bounds reading the confirmation facts, which captures the
// committed source with the Runtime profile.
const factsTimeout = 2 * time.Minute

// handleRunControl states whether starting is available, the run this cockpit
// started and, when asked for with facts=1, the facts the owner confirms.
// Reading the facts captures the committed source, so the page asks for them
// only when it shows the confirmation.
func (s *Server) handleRunControl(w http.ResponseWriter, r *http.Request) {
	if s.runs == nil {
		writeJSON(w, runControlView{Reason: runControlUnavailable})
		return
	}
	view := runControlView{Available: true, Run: s.runs.Status()}
	if r.URL.Query().Get("facts") == "1" {
		ctx, cancel := context.WithTimeout(r.Context(), factsTimeout)
		defer cancel()
		facts, err := s.runs.cfg.Facts(ctx)
		if err != nil {
			view.FactsError = err.Error()
		} else {
			view.Facts = &facts
		}
	}
	writeJSON(w, view)
}

func (s *Server) runsOrRefuse(w http.ResponseWriter) bool {
	if s.runs == nil {
		http.Error(w, runControlUnavailable, http.StatusNotImplemented)
		return false
	}
	return true
}

// handleStartRun starts the run after the owner's confirmation in the page.
// The cockpit never starts a run by itself.
func (s *Server) handleStartRun(w http.ResponseWriter, r *http.Request) {
	if !s.runsOrRefuse(w) || !s.ownRequest(w, r, "start-run") {
		return
	}
	status, err := s.runs.Start()
	if err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, ErrRunActive) || errors.Is(err, ErrRunHeld) || errors.Is(err, ErrRunEnding) {
			code = http.StatusConflict
		}
		http.Error(w, err.Error(), code)
		return
	}
	writeJSON(w, status)
}

// handleStopRun stops only the run this cockpit started.
func (s *Server) handleStopRun(w http.ResponseWriter, r *http.Request) {
	if !s.runsOrRefuse(w) || !s.ownRequest(w, r, "stop-run") {
		return
	}
	status, err := s.runs.Stop()
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, status)
}
