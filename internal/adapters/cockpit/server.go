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
)

//go:embed web
var webFiles embed.FS

// DefaultListen is the loopback address the cockpit listens on by default.
const DefaultListen = "127.0.0.1:7341"

// Config selects the state directories the cockpit observes and its loopback
// listen address. The cockpit only reads them.
type Config struct {
	States []string
	Listen string
}

// Server serves the cockpit's pages and its read interface over the evidence
// of the configured state directories. No request starts, answers, repairs or
// publishes anything.
type Server struct {
	states []*stateDirectory
	host   string
	web    fs.FS
}

// New checks the configuration. Every state directory must exist.
func New(cfg Config) (*Server, error) {
	if len(cfg.States) == 0 {
		return nil, errors.New("the cockpit needs at least one state directory")
	}
	s := &Server{}
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
	s.host = ln.Addr().String()
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(ln) }()
	if ready != nil {
		ready("http://" + s.host + "/")
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
	mux.HandleFunc("GET /api/runs", s.handleRuns)
	mux.HandleFunc("GET /api/runs/{run}/view", s.handleView)
	mux.HandleFunc("GET /api/runs/{run}/entries", s.handleEntries)
	mux.HandleFunc("GET /api/runs/{run}/detail", s.handleDetail)
	mux.HandleFunc("GET /api/runs/{run}/transcript", s.handleTranscript)
	files := http.FileServerFS(s.web)
	mux.Handle("GET /", files)
	return s.guard(mux)
}

// guard admits only requests addressed to this loopback server, so another
// site cannot read run evidence through a rebound host name, and sets a
// restrictive content policy.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.admitHost(r.Host) {
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

func (s *Server) admitHost(host string) bool {
	if s.host == "" {
		return true // Handler used without Serve, as in tests
	}
	_, port, err := net.SplitHostPort(s.host)
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
	ID          string    `json:"id"`
	State       string    `json:"state"`
	Current     bool      `json:"current"`
	Started     time.Time `json:"started,omitzero"`
	Last        time.Time `json:"last,omitzero"`
	Records     int       `json:"records"`
	Managers    int       `json:"managers"`
	Passes      int       `json:"passes"`
	Outcome     string    `json:"outcome,omitempty"`
	RunOutcome  string    `json:"runOutcome,omitempty"`
	Publication string    `json:"publication,omitempty"`
	Error       string    `json:"error,omitempty"`
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
			out = append(out, summarize(st, entry))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	writeJSON(w, out)
}

func summarize(st *stateDirectory, entry RunEntry) RunSummary {
	sum := RunSummary{ID: entry.ID, State: entry.State, Current: entry.Current}
	l := st.log(entry.ID)
	if _, err := l.refresh(); err != nil {
		sum.Error = err.Error()
	}
	records, _, report := l.snapshot()
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
			}
			_ = json.Unmarshal(rec.Data, &d)
			sum.Outcome = d.Status
		}
		if !rec.At.IsZero() {
			sum.Last = rec.At
		}
	}
	if report != nil {
		sum.RunOutcome = report.Outcome
		if report.Publication != nil {
			sum.Publication = report.Publication.Outcome
		}
	}
	return sum
}

// runLog finds a run by identity in the configured state directories and
// reads what was appended to its evidence.
func (s *Server) runLog(w http.ResponseWriter, r *http.Request) (*runLog, bool) {
	id := r.PathValue("run")
	if !safeName.MatchString(id) {
		http.Error(w, "not a run identity", http.StatusBadRequest)
		return nil, false
	}
	var found *stateDirectory
	for _, st := range s.states {
		if _, err := os.Stat(filepath.Join(st.dir, "runs", id, "events.jsonl")); err == nil {
			if found != nil {
				http.Error(w, "run "+id+" exists in more than one state directory", http.StatusConflict)
				return nil, false
			}
			found = st
		}
	}
	if found == nil {
		http.Error(w, "no run "+id+" in the observed state directories", http.StatusNotFound)
		return nil, false
	}
	l := found.log(id)
	_, _ = l.refresh() // a failed read stays visible as the source's error
	return l, true
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
	l, ok := s.runLog(w, r)
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
	l, ok := s.runLog(w, r)
	if !ok {
		return
	}
	records, src, _ := l.snapshot()
	after, _ := strconv.Atoi(r.URL.Query().Get("after"))
	if after < 0 || after > len(records) {
		after = 0
	}
	writeJSON(w, struct {
		Source  Source  `json:"source"`
		Entries []Entry `json:"entries"`
	}{src, Entries(records[after:])})
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	l, ok := s.runLog(w, r)
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
	l, ok := s.runLog(w, r)
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
