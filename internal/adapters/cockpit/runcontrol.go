package cockpit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
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
	// Profiles lists the distinct Execute Agent profiles of the Managers,
	// Root's first.
	Profiles []RunProfile `json:"profiles"`
}

// RunProfile is an Execute Agent profile with the model its Runtime profile
// states, empty when it states none, and the Managers (Package paths, "." for
// Root) that use it.
type RunProfile struct {
	Name     string   `json:"name"`
	Model    string   `json:"model,omitempty"`
	Managers []string `json:"managers"`
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
//
// Head reads the repository's committed HEAD; a start must name the head its
// confirmation was read from. AttributeAfter is how long a started run is not
// named after current-run: Execute refuses a state directory that a foreign
// run holds within its lock wait, so a current-run seen after that while
// Execute still runs belongs to this run.
type RunControlConfig struct {
	StateDir       string
	Facts          func(context.Context) (RunFacts, error)
	Head           func(context.Context) (string, error)
	Execute        func(context.Context) RunResult
	Held           func() (string, error)
	AttributeAfter time.Duration
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
	// ErrRunSource refuses a start whose confirmation was read from another
	// committed head than the repository's now.
	ErrRunSource = errors.New("the committed source changed since the confirmation; review it again")
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
	closed bool // Wait ran: no run starts any more
	cancel context.CancelFunc
	cur    *runHandle
	facts  *factsCall
}

// factsCall is one read of the confirmation facts that concurrent requests
// share.
type factsCall struct {
	done  chan struct{}
	facts RunFacts
	err   error
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
func (c *RunController) Start(ctx context.Context, head string) (RunStatus, error) {
	if head == "" {
		return RunStatus{}, fmt.Errorf("%w: the start names no head", ErrRunSource)
	}
	// The head is read before the lock because it runs git. A commit between
	// this read and Execute's own capture remains possible; Execute then runs
	// the newer committed source.
	if c.cfg.Head != nil {
		now, err := c.cfg.Head(ctx)
		if err != nil {
			return RunStatus{}, fmt.Errorf("the committed head could not be read: %w", err)
		}
		if now != head {
			return RunStatus{}, ErrRunSource
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.base.Err() != nil {
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
	runCtx, cancel := context.WithCancel(c.base)
	h := &runHandle{status: RunStatus{State: RunStarting, Started: time.Now().UTC()}, before: c.current()}
	c.cur, c.cancel = h, cancel
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer cancel()
		c.finish(h, c.execute(runCtx))
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
	if result.Evidence != "" {
		h.status.Run = filepath.Base(result.Evidence)
		h.status.Report = filepath.Join(result.Evidence, "report.json")
	} else if result.Outcome == RunRefused {
		// No evidence exists to open. A stop before the run created any is a
		// stop, not a refusal of the execution.
		h.status.Run = ""
		if h.stopped {
			result.Outcome = "incomplete"
			result.Error = "stopped before the run created evidence: nothing ran and no run evidence or report exists"
		}
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
	if h.status.Run == "" && time.Since(h.status.Started) >= c.cfg.AttributeAfter {
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
	c.closed = true
	if c.cur != nil && c.cur.status.State != RunEnded {
		c.cur.stopped = true
		c.cancel()
	}
	c.mu.Unlock()
	c.wg.Wait()
}

// ReadFacts reads the confirmation facts. Concurrent reads share one capture:
// the first starts it, the others wait for its result.
func (c *RunController) ReadFacts(ctx context.Context) (RunFacts, error) {
	c.mu.Lock()
	call := c.facts
	if call == nil {
		if c.closed || c.base.Err() != nil {
			c.mu.Unlock()
			return RunFacts{}, ErrRunEnding
		}
		call = &factsCall{done: make(chan struct{})}
		c.facts = call
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			// The capture belongs to no request, so a waiter that leaves does
			// not cancel it for the others.
			readCtx, cancel := context.WithTimeout(c.base, factsTimeout)
			defer cancel()
			call.facts, call.err = c.cfg.Facts(readCtx)
			c.mu.Lock()
			c.facts = nil
			c.mu.Unlock()
			close(call.done)
		}()
	}
	c.mu.Unlock()
	select {
	case <-call.done:
		return call.facts, call.err
	case <-ctx.Done():
		return RunFacts{}, ctx.Err()
	}
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
		facts, err := s.runs.ReadFacts(r.Context())
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
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "a start is a JSON request naming the head of its confirmation", http.StatusForbidden)
		return
	}
	var body struct {
		Head string `json:"head"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil || dec.More() {
		http.Error(w, "the start is not a JSON object with head", http.StatusBadRequest)
		return
	}
	status, err := s.runs.Start(r.Context(), body.Head)
	if err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, ErrRunActive) || errors.Is(err, ErrRunHeld) || errors.Is(err, ErrRunEnding) || errors.Is(err, ErrRunSource) {
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
