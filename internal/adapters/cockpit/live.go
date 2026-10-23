package cockpit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Liveness reports whether an Apply holds a state directory's lock. Bootstrap
// supplies the probe; it takes the lock for an instant
// when nobody holds it, which the brief retries of a starting Apply and of
// setup absorb.
type Liveness func(stateDirectory string) (bool, error)

// Health is the current observation of a run: the state of its evidence when
// last read and whether its process still holds the state directory. It is
// separate from the record: a run can be recorded up to some boundary and be
// running, ended or unknown now.
type Health struct {
	Source Source `json:"source"`
	// Run is "running", "finishing" (outcome recorded, report not yet
	// written), "ended" (report written), "ended-without-report" or "unknown".
	Run string `json:"run"`
	Why string `json:"why"`
}

// health observes the run's process: the report ends it; otherwise the state
// lock and current-run say whether this run still holds the directory. An
// Apply takes the lock, then names itself in current-run, then creates its
// evidence, and writes its report before it releases the lock.
func (s *Server) health(st *stateDirectory, l *runLog, records []Record, src Source, report *Report) Health {
	h := Health{Source: src}
	switch {
	case report != nil:
		h.Run, h.Why = "ended", "report.json is written"
		return h
	case s.liveness == nil:
		h.Run, h.Why = "unknown", "no state lock probe is configured"
		return h
	}
	held, err := st.held(s.liveness)
	switch {
	case err != nil:
		h.Run, h.Why = "unknown", "the state lock could not be probed: "+err.Error()
	case held && st.currentRun() == l.id:
		h.Run, h.Why = "running", "this run holds the state directory"
		for _, r := range records {
			if r.Kind == "outcome" {
				h.Run, h.Why = "finishing", "the outcome is recorded; publication and report are not written yet"
			}
		}
	case held:
		h.Run, h.Why = "ended-without-report", "another run holds the state directory; this one ended without a report"
	default:
		h.Run, h.Why = "ended-without-report", "no process holds the state directory; the run ended without writing its report"
	}
	return h
}

// held probes the state lock at most once per probeInterval and state
// directory, however many views observe it.
func (st *stateDirectory) held(probe Liveness) (bool, error) {
	st.probe.Lock()
	defer st.probe.Unlock()
	if time.Since(st.probedAt) < probeInterval {
		return st.heldNow, st.probeErr
	}
	st.heldNow, st.probeErr = probe(st.dir)
	st.probedAt = time.Now()
	return st.heldNow, st.probeErr
}

const probeInterval = 2 * time.Second

// handleStream streams a run's evidence as server-sent events: "entries" with
// the records after the client's last one (its id is that record's seq and
// mark), "health" with the current observation, and "replaced" when the events
// file no longer holds the client's history, after which the stream ends. A
// reconnecting browser sends Last-Event-ID and receives exactly the records
// after it: none twice, none skipped; a different record at that position,
// also one replaced while the client was away, ends the stream as replaced.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	st, l, ok := s.findRun(w, r)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}
	position := r.Header.Get("Last-Event-ID")
	if position == "" {
		position = r.URL.Query().Get("after") + "." + r.URL.Query().Get("mark")
	}
	seqText, mark, _ := strings.Cut(position, ".")
	after, err := strconv.Atoi(seqText)
	if err != nil || after < 0 {
		after, mark = 0, ""
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	send := func(event string, id any, data any) bool {
		payload, err := json.Marshal(data)
		if err != nil {
			return false
		}
		if id != 0 {
			_, err = fmt.Fprintf(w, "id: %v\n", id)
		}
		if err == nil {
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
		}
		flusher.Flush()
		return err == nil
	}
	_, _ = fmt.Fprintf(w, "retry: %d\n\n", reconnectDelay.Milliseconds())
	records, src, _ := l.snapshot()
	replaced := src.Replaced
	if after > 0 && after <= len(records) && records[after-1].Mark != mark {
		send("replaced", 0, src)
		return
	}
	ticker := time.NewTicker(s.poll)
	defer ticker.Stop()
	var lastHealth time.Time
	for {
		_, _ = l.refresh()
		records, src, report := l.snapshot()
		if src.Replaced != replaced || after > len(records) {
			send("replaced", 0, src)
			return
		}
		if len(records) > after {
			if !send("entries", fmt.Sprintf("%d.%s", len(records), records[len(records)-1].Mark), struct {
				From    int     `json:"from"`
				Source  Source  `json:"source"`
				Entries []Entry `json:"entries"`
			}{after + 1, src, Entries(records[after:])}) {
				return
			}
			after = len(records)
		}
		if time.Since(lastHealth) >= healthInterval {
			if !send("health", 0, s.health(st, l, records, src, report)) {
				return
			}
			lastHealth = time.Now()
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

const (
	healthInterval = 2 * time.Second
	reconnectDelay = 1500 * time.Millisecond
)
