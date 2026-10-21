package cockpit

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

//go:embed web
var webFiles embed.FS

// DefaultListen is the loopback address the cockpit listens on by default.
const DefaultListen = "127.0.0.1:7341"

// Config selects the state directories the cockpit observes. The cockpit only
// reads them; Liveness, when set, probes whether an Apply holds one. Project
// observes the repository's identity. Answer, when set, is the owner
// channel's answer operation. Runs, when set, starts and stops the one Apply
// this cockpit runs in its own process.
type Config struct {
	States   []string
	Liveness Liveness
	Project  ProjectObserver
	Answer   Answerer
	// Authoring, when set, makes and decides proposals for the project's
	// .markitect inputs.
	Authoring Authoring
	Runs      *RunController
	// RunsReason, when set, replaces the text that says why starting a run is
	// unavailable.
	RunsReason string
}

// Server serves the cockpit's pages and its read interface over the evidence
// of the configured state directories. It writes an owner's answer through
// the configured owner channel, an accepted proposal through the configured
// authoring service, and starts and stops the one run its own controller
// holds; no request repairs a run, and a run started elsewhere is only
// observed.
type Server struct {
	states     []*stateDirectory
	liveness   Liveness
	project    projectCache
	answer     Answerer
	authoring  Authoring
	runs       *RunController
	runsReason string
	poll       time.Duration
	host       string
	web        fs.FS
}

// New checks the configuration. Every state directory must exist.
func New(cfg Config) (*Server, error) {
	if len(cfg.States) == 0 && cfg.Project == nil {
		return nil, errors.New("the cockpit needs at least one state directory or a repository")
	}
	s := &Server{liveness: cfg.Liveness, project: projectCache{observe: cfg.Project}, answer: cfg.Answer, authoring: cfg.Authoring, runs: cfg.Runs, runsReason: cfg.RunsReason, poll: 400 * time.Millisecond}
	seen := map[string]bool{}
	for _, dir := range cfg.States {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("state directory %s: %w", dir, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("state directory %s is not a directory", dir)
		}
		if !seen[abs] {
			seen[abs] = true
			s.states = append(s.states, &stateDirectory{dir: abs})
		}
	}
	web, err := fs.Sub(webFiles, "web")
	if err != nil {
		return nil, err
	}
	s.web = web
	return s, nil
}

// Serve listens on the loopback address and serves until ctx ends. ready
// receives the address the browser opens.
func (s *Server) Serve(ctx context.Context, listen string, ready func(url string)) error {
	return serveLoopback(ctx, listen, func(host string) http.Handler {
		s.host = host
		return s.Handler()
	}, ready)
}

// serveLoopback listens on a loopback address (the default when listen is
// empty) and serves the handler built for the address it listens on until ctx
// ends. ready receives the address the browser opens.
func serveLoopback(ctx context.Context, listen string, handler func(host string) http.Handler, ready func(url string)) error {
	if listen == "" {
		listen = DefaultListen
	}
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return fmt.Errorf("listen address %q: %w", listen, err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("listen address %q is not a loopback address: the cockpit serves run evidence only to this machine", listen)
	}
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
	address := ln.Addr().String()
	// Requests end with ctx, so open live streams do not hold up the exit.
	srv := &http.Server{Handler: handler(address), ReadHeaderTimeout: 10 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(ln) }()
	if ready != nil {
		ready("http://" + address + "/")
	}
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
		return nil
	case err := <-errs:
		return err
	}
}

// Handler is the cockpit's HTTP interface.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/project", s.handleProject)
	mux.HandleFunc("POST /api/project/observe", s.handleObserve)
	mux.HandleFunc("GET /api/model", s.handleModel)
	mux.HandleFunc("GET /api/questions", s.handleQuestions)
	mux.HandleFunc("POST /api/runs/{run}/questions/{question}/answer", s.handleAnswer)
	mux.HandleFunc("GET /api/settings", s.handleSettings)
	mux.HandleFunc("POST /api/proposals/settings", s.handleProposeSettings)
	mux.HandleFunc("GET /api/proposals", s.handleProposals)
	mux.HandleFunc("GET /api/proposals/{id}", s.handleProposal)
	mux.HandleFunc("POST /api/proposals/{id}/{decision}", s.handleDecide)
	mux.HandleFunc("GET /api/sessions", s.handleSessions)
	mux.HandleFunc("POST /api/sessions", s.handleStartSession)
	mux.HandleFunc("GET /api/sessions/{id}", s.handleSession)
	mux.HandleFunc("POST /api/sessions/{id}/prompt", s.handlePromptSession)
	mux.HandleFunc("POST /api/sessions/{id}/close", s.handleCloseSession)
	mux.HandleFunc("GET /api/run-control", s.handleRunControl)
	mux.HandleFunc("POST /api/run-control/start", s.handleStartRun)
	mux.HandleFunc("POST /api/run-control/stop", s.handleStopRun)
	mux.HandleFunc("GET /api/runs", s.handleRuns)
	mux.HandleFunc("GET /api/runs/{run}/view", s.handleView)
	mux.HandleFunc("GET /api/runs/{run}/entries", s.handleEntries)
	mux.HandleFunc("GET /api/runs/{run}/detail", s.handleDetail)
	mux.HandleFunc("GET /api/runs/{run}/transcript", s.handleTranscript)
	mux.HandleFunc("GET /api/runs/{run}/stream", s.handleStream)
	files := http.FileServerFS(s.web)
	mux.Handle("GET /", files)
	return s.guard(mux)
}

// guard admits only requests addressed to this loopback server, so another
// site cannot read run evidence through a rebound host name, and sets a
// restrictive content policy.
func (s *Server) guard(next http.Handler) http.Handler { return guardHandler(next, s.admitHost) }

func guardHandler(next http.Handler, admit func(host string) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !admit(r.Host) {
			http.Error(w, "unknown host", http.StatusMisdirectedRequest)
			return
		}
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) admitHost(host string) bool { return admitLoopback(s.host, host) }

// admitLoopback admits the host names that reach the loopback address served;
// without one (a handler used without Serve, as in tests) every host.
func admitLoopback(served, host string) bool {
	if served == "" {
		return true
	}
	_, port, err := net.SplitHostPort(served)
	if err != nil {
		return false
	}
	for _, h := range []string{"127.0.0.1", "localhost", "[::1]"} {
		if host == h+":"+port {
			return true
		}
	}
	return false
}

// RunSummary is one run in the run list: what its record says so far.
type RunSummary struct {
	ID          string     `json:"id"`
	State       string     `json:"state"`
	Current     bool       `json:"current"`
	Started     time.Time  `json:"started,omitzero"`
	Last        time.Time  `json:"last,omitzero"`
	Records     int        `json:"records"`
	Managers    int        `json:"managers"`
	Passes      int        `json:"passes"`
	Outcome     string     `json:"outcome,omitempty"`
	RunOutcome  string     `json:"runOutcome,omitempty"`
	Publication string     `json:"publication,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	Head        string     `json:"head,omitempty"`
	Usage       *run.Usage `json:"usage,omitempty"`
	Health      *Health    `json:"health,omitempty"`
	Account     []Sentence `json:"account,omitempty"`
	Error       string     `json:"error,omitempty"`
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	out := []RunSummary{}
	for _, st := range s.states {
		runs, err := st.list()
		if err != nil {
			out = append(out, RunSummary{State: st.dir, Error: err.Error()})
			continue
		}
		for _, entry := range runs {
			out = append(out, s.summarize(st, entry))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	writeJSON(w, out)
}

// summarize states what a run's record says so far. A run without a report
// also gets its current health and, while it runs, the account of its latest
// boundary.
func (s *Server) summarize(st *stateDirectory, entry RunEntry) RunSummary {
	sum := RunSummary{ID: entry.ID, State: entry.State, Current: entry.Current}
	l := st.log(entry.ID)
	if _, err := l.refresh(); err != nil {
		sum.Error = err.Error()
	}
	records, src, report := l.snapshot()
	if report == nil {
		h := s.health(st, l, records, src, report)
		sum.Health = &h
		if h.Run == "running" || h.Run == "finishing" {
			sum.Account = Project(l.id, records, len(records), src, report).Account
		}
	}
	sum.Records = len(records)
	for _, rec := range records {
		switch rec.Kind {
		case "run.start":
			sum.Started = rec.At
			var d struct {
				Managers []string `json:"managers"`
			}
			_ = json.Unmarshal(rec.Data, &d)
			sum.Managers = len(d.Managers)
		case "pass.start":
			sum.Passes++
		case "outcome":
			var d struct {
				Status string `json:"status"`
				Reason string `json:"reason"`
			}
			_ = json.Unmarshal(rec.Data, &d)
			sum.Outcome, sum.Reason = d.Status, d.Reason
		}
		if !rec.At.IsZero() {
			sum.Last = rec.At
		}
	}
	if report != nil {
		sum.RunOutcome, sum.Head = report.Outcome, report.Head
		if report.Run != nil {
			usage := report.Run.Usage
			sum.Usage = &usage
		}
		if report.Publication != nil {
			sum.Publication = report.Publication.Outcome
		}
	}
	return sum
}

// findRun finds a run by identity in the configured state directories and
// reads what was appended to its evidence.
func (s *Server) findRun(w http.ResponseWriter, r *http.Request) (*stateDirectory, *runLog, bool) {
	id := r.PathValue("run")
	if !safeName.MatchString(id) {
		http.Error(w, "not a run identity", http.StatusBadRequest)
		return nil, nil, false
	}
	var found *stateDirectory
	for _, st := range s.states {
		if _, err := os.Stat(filepath.Join(st.dir, "runs", id, "events.jsonl")); err == nil {
			if found != nil {
				http.Error(w, "run "+id+" exists in more than one state directory", http.StatusConflict)
				return nil, nil, false
			}
			found = st
		}
	}
	if found == nil {
		http.Error(w, "no run "+id+" in the observed state directories", http.StatusNotFound)
		return nil, nil, false
	}
	l := found.log(id)
	_, _ = l.refresh() // a failed read stays visible as the source's error
	return found, l, true
}

// boundary reads the at parameter: a record count, or every record without it.
func boundary(r *http.Request, total int) (int, error) {
	at := r.URL.Query().Get("at")
	if at == "" {
		return total, nil
	}
	n, err := strconv.Atoi(at)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("at must be a record count")
	}
	if n > total {
		n = total
	}
	return n, nil
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	_, l, ok := s.findRun(w, r)
	if !ok {
		return
	}
	records, src, report := l.snapshot()
	n, err := boundary(r, len(records))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, Project(l.id, records, n, src, report))
}

// handleEntries lists the record's entries for navigation: the scrubber, the
// timeline and the event lists. The page shows only those up to its boundary;
// views, details and transcripts are bound to the boundary on the server.
func (s *Server) handleEntries(w http.ResponseWriter, r *http.Request) {
	_, l, ok := s.findRun(w, r)
	if !ok {
		return
	}
	records, src, _ := l.snapshot()
	after, _ := strconv.Atoi(r.URL.Query().Get("after"))
	if after < 0 || after > len(records) {
		http.Error(w, "after lies beyond the record; read it again from its start", http.StatusConflict)
		return
	}
	writeJSON(w, struct {
		Source  Source  `json:"source"`
		Entries []Entry `json:"entries"`
	}{src, Entries(records[after:])})
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	_, l, ok := s.findRun(w, r)
	if !ok {
		return
	}
	records, _, _ := l.snapshot()
	n, err := boundary(r, len(records))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, Select(records, n, r.URL.Query().Get("ref")))
}

func (s *Server) handleTranscript(w http.ResponseWriter, r *http.Request) {
	_, l, ok := s.findRun(w, r)
	if !ok {
		return
	}
	records, _, _ := l.snapshot()
	n, err := boundary(r, len(records))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	turn := r.URL.Query().Get("turn")
	started := false
	for _, rec := range records[:n] {
		if rec.Kind == "turn.start" && rec.Turn == turn {
			started = true
			break
		}
	}
	if !safeName.MatchString(turn) || !started {
		http.Error(w, "no turn "+turn+" started up to this boundary", http.StatusNotFound)
		return
	}
	// The cut is the latest recorded time up to the boundary; an unreadable
	// record has none.
	var until time.Time
	if n < len(records) {
		for i := n - 1; i >= 0 && until.IsZero(); i-- {
			until = records[i].At
		}
		if until.IsZero() {
			until = time.Unix(0, 0)
		}
	}
	data, err := readBounded(filepath.Join(l.dir, "transcripts", turn+".jsonl"), maxTranscriptBytes)
	if errors.Is(err, os.ErrNotExist) {
		http.Error(w, "no transcript was recorded for turn "+turn, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, parseTranscript(turn, data, until))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
