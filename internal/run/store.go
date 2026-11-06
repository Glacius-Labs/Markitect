package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
)

// RunState is everything a Run knows between one command and the next. It lives
// under the repository's Git directory, which every worktree shares, and never
// reaches a working tree.
type RunState struct {
	Run string `json:"run"`
	// Command is apply or verify.
	Command string `json:"command"`
	// Branch is the owner's branch the Run started from and lands on; Repo the
	// owner's folder.
	Branch string `json:"branch"`
	Repo   string `json:"repo"`
	// Base is the commit impact is computed from, Start the commit the Run
	// started from.
	Base  string `json:"base"`
	Start string `json:"start"`
	Full  bool   `json:"full"`
	// Findings are the Findings an apply --findings started from.
	Findings []analysis.Finding `json:"findings,omitempty"`
	// Items is the list as it stands, Answers each item's latest answer.
	Items   []Item            `json:"items"`
	Answers map[string]Answer `json:"answers"`
	// Review is the review the work is under.
	Review *Review `json:"review,omitempty"`
	// Reviewed is the last commit a review finished on.
	Reviewed string `json:"reviewed,omitempty"`
	// Repairs counts the repair rounds, SentBack what each went back for.
	Repairs  int      `json:"repairs"`
	SentBack []string `json:"sentBack,omitempty"`
	// Submits counts the calls of submit.
	Submits    int      `json:"submits"`
	Impact     []string `json:"impact,omitempty"`
	Notes      []string `json:"notes,omitempty"`
	MissingFor []string `json:"missingFor,omitempty"`
	// Model is the model the project's manifest names for a Run's agents, empty for
	// the one their definitions name.
	Model string `json:"model,omitempty"`
	// Began and Deadline bound the Run.
	Began    time.Time `json:"began"`
	Deadline time.Time `json:"deadline"`
	// Limit is how long the Run may work.
	Limit time.Duration `json:"limit"`

	// Open is true until submit or stop closes the Run.
	Open bool `json:"open"`
	// State and Why say how it closed, Landed the commit that holds its result,
	// Kept the branch that holds its work when it did not land.
	State  string    `json:"state,omitempty"`
	Why    string    `json:"why,omitempty"`
	Landed string    `json:"landed,omitempty"`
	Kept   string    `json:"kept,omitempty"`
	Closed time.Time `json:"closed,omitzero"`
	// RetryLanding says the next apply tries once more to land Kept.
	RetryLanding bool `json:"retryLanding,omitempty"`
	// Told says the owner has seen how it closed.
	Told bool `json:"told,omitempty"`
}

// Review is one review of one state of the work.
type Review struct {
	// Commit is the commit under review, Tree its tree.
	Commit string `json:"commit"`
	Tree   string `json:"tree"`
	// From is the commit the earlier review ended on, "" on the first.
	From string `json:"from,omitempty"`
	// Items are what this review must answer; Answered the answers it has.
	Items    []Item            `json:"items"`
	Answered map[string]Answer `json:"answered"`
	// Earlier holds what an earlier review sent back.
	Earlier []string `json:"earlier,omitempty"`
	// Counted says this review's wrong answers have used a repair round.
	Counted bool `json:"counted,omitempty"`
}

// Complete says every item of the review has an answer.
func (r *Review) Complete() bool {
	return r != nil && len(r.Missing()) == 0
}

// Missing lists the items without an answer.
func (r *Review) Missing() []string {
	var out []string
	for _, it := range r.Items {
		if _, ok := r.Answered[it.Key]; !ok {
			out = append(out, it.Key)
		}
	}
	return out
}

// Store keeps the Runs of one repository.
type Store struct {
	// Dir is the folder under the Git directory.
	Dir string
}

// lockWait bounds how long a change of a Run's state waits for another.
const lockWait = 10 * time.Second

func (s Store) file(run string) string { return filepath.Join(s.Dir, "runs", run+".json") }

// Load reads a Run, and false when there is none of that name.
func (s Store) Load(run string) (*RunState, bool, error) {
	var data []byte
	var err error
	for i := 0; i < 20; i++ {
		if data, err = os.ReadFile(s.file(run)); err == nil || errors.Is(err, os.ErrNotExist) {
			break
		}
		// A rename beside a read is refused for a moment on Windows.
		time.Sleep(50 * time.Millisecond)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var st RunState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, false, fmt.Errorf("run %s: %w", run, err)
	}
	return &st, true, nil
}

// Save writes a Run in one step.
func (s Store) Save(st *RunState) error {
	if err := os.MkdirAll(filepath.Join(s.Dir, "runs"), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Join(s.Dir, "runs"), st.Run+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	// A crash must not leave an empty state behind the rename.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	for i := 0; ; i++ {
		err = os.Rename(name, s.file(st.Run))
		if err == nil || i == 20 {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// All reads every Run, oldest first.
func (s Store) All() ([]*RunState, error) {
	entries, err := os.ReadDir(filepath.Join(s.Dir, "runs"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []*RunState
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || strings.HasSuffix(name, ".settings") {
			continue
		}
		// One unreadable Run must not block every command.
		if st, found, err := s.Load(name); err == nil && found {
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Began.Before(out[j].Began) })
	return out, nil
}

// Update changes a Run under a short lock, for what changes it beside a submit.
func (s Store) Update(run string, change func(*RunState) error) error {
	release, err := s.lock(run+".state", lockWait)
	if err != nil {
		return err
	}
	defer release()
	st, found, err := s.Load(run)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no run %s", run)
	}
	if err := change(st); err != nil {
		return err
	}
	return s.Save(st)
}

// Settle writes a Run a long command has held a copy of. That command never writes
// answers, so what others wrote meanwhile stands.
func (s Store) Settle(st *RunState) error {
	release, err := s.lock(st.Run+".state", lockWait)
	if err != nil {
		return err
	}
	defer release()
	fresh, found, err := s.Load(st.Run)
	if err != nil {
		return err
	}
	if found {
		st.Answers, st.MissingFor = fresh.Answers, fresh.MissingFor
		if st.Review != nil && fresh.Review != nil && st.Review.Commit == fresh.Review.Commit {
			st.Review.Answered = fresh.Review.Answered
		}
	}
	return s.Save(st)
}

// ErrBusy says another process holds a Run's lock.
var ErrBusy = errors.New("busy")

// TrySubmitLock takes the lock of a submit, or ErrBusy while another runs.
func (s Store) TrySubmitLock(run string) (func(), error) {
	return s.lock(run+".submit", 0)
}

// SubmitRunning says a submit holds the Run's lock.
func (s Store) SubmitRunning(run string) bool {
	file := filepath.Join(s.Dir, "runs", run+".submit.lock")
	if _, err := os.Stat(file); err != nil {
		return false
	}
	return !staleLock(file)
}

// lockLifetime is the longest a lock can be real: a shell call is bounded by two
// hours, so an older one names a process ID that has been used again.
const lockLifetime = 2*time.Hour + 10*time.Minute

// lockSettling is how long a lock file may stay empty between being made and
// being written.
const lockSettling = 5 * time.Second

// lock takes a lock file that holds this process's ID. A lock whose process has
// gone is taken over, so that a command that was killed can be run again.
func (s Store) lock(name string, wait time.Duration) (func(), error) {
	if err := os.MkdirAll(filepath.Join(s.Dir, "runs"), 0o755); err != nil {
		return nil, err
	}
	file := filepath.Join(s.Dir, "runs", name+".lock")
	until := time.Now().Add(wait)
	for {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(file) }, nil
		}
		if staleLock(file) {
			os.Remove(file)
			continue
		}
		if !time.Now().Before(until) {
			return nil, ErrBusy
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// staleLock says a lock can no longer be real: it is too old, empty for longer
// than it takes to write, or its process has gone.
func staleLock(file string) bool {
	info, err := os.Stat(file)
	if err != nil {
		return false
	}
	age := time.Since(info.ModTime())
	if age > lockLifetime {
		return true
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return age > lockSettling
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return true
	}
	defer p.Release()
	if runtime.GOOS == "windows" {
		// FindProcess opens the process, which fails once it has gone.
		return false
	}
	return p.Signal(syscall.Signal(0)) != nil
}
