package cockpit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Bounds of one read. Evidence beyond them is reported as a limitation, never
// silently truncated into a smaller history.
const (
	maxEventBytes      = 64 << 20
	maxReportBytes     = 16 << 20
	maxTranscriptBytes = 32 << 20
	maxQuestionBytes   = 1 << 20
)

// safeName admits the run, turn and question identities the producer uses as
// path elements; nothing else may steer a path.
var safeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

// Record is one complete line of a run's events.jsonl. Seq is its 1-based line
// number, which is the observation order. A line that does not decode keeps its
// place with Invalid set, so later sequence numbers never shift.
type Record struct {
	Seq     int
	At      time.Time
	Kind    string
	Pass    string
	Manager string
	Turn    string
	Data    json.RawMessage
	Invalid string
}

// Source identifies exactly what a view was projected from: the consumed bytes
// of complete event lines and their digest, a trailing partial line, and the
// report's presence. Replaced reports that the file no longer starts with the
// bytes read before; the history was then read again from its start.
type Source struct {
	EventBytes  int64     `json:"eventBytes"`
	EventDigest string    `json:"eventDigest"`
	Records     int       `json:"records"`
	TailBytes   int       `json:"tailBytes"`
	Report      bool      `json:"report"`
	Replaced    int       `json:"replaced"`
	ReadAt      time.Time `json:"readAt"`
	Error       string    `json:"error,omitempty"`
}

// runLog is the incrementally read evidence of one run directory. Records are
// append-only: an element once read is never changed, so a slice of them can
// be handed out under the lock.
type runLog struct {
	state, id, dir string

	mu       sync.Mutex
	records  []Record
	consumed int64
	digest   []byte // sha256 of the consumed bytes
	tail     int
	size     int64
	modified time.Time
	replaced int
	report   *Report
	reportAt time.Time
	readAt   time.Time
	err      error
}

func newRunLog(state, id string) *runLog {
	return &runLog{state: state, id: id, dir: filepath.Join(state, "runs", id), digest: sum(nil)}
}

// snapshot returns the records read so far and the source identity they form.
func (l *runLog) snapshot() ([]Record, Source, *Report) {
	l.mu.Lock()
	defer l.mu.Unlock()
	src := Source{EventBytes: l.consumed, EventDigest: hex.EncodeToString(l.digest), Records: len(l.records), TailBytes: l.tail, Report: l.report != nil, Replaced: l.replaced, ReadAt: l.readAt}
	if l.err != nil {
		src.Error = l.err.Error()
	}
	return l.records[:len(l.records):len(l.records)], src, l.report
}

// refresh reads what was appended since the last read. It reports whether the
// records or the report changed. A file that no longer begins with the bytes
// already consumed is a replaced source: the history is read again and the
// replacement counted, never spliced onto the old records.
func (l *runLog) refresh() (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.readAt = time.Now().UTC()
	events, err := l.refreshEventsLocked()
	report, rerr := l.refreshReportLocked()
	l.err = errors.Join(err, rerr)
	return events || report, l.err
}

func (l *runLog) refreshEventsLocked() (bool, error) {
	path := filepath.Join(l.dir, "events.jsonl")
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.Size() == l.size && info.ModTime().Equal(l.modified) {
		return false, nil
	}
	if info.Size() > maxEventBytes {
		return false, fmt.Errorf("events.jsonl has %d bytes, more than the cockpit reads (%d)", info.Size(), maxEventBytes)
	}
	data, err := readBounded(path, maxEventBytes)
	if err != nil {
		return false, err
	}
	if int64(len(data)) < l.consumed || !bytes.Equal(sum(data[:l.consumed]), l.digest) {
		// The file was replaced or truncated: read it again from the start.
		l.records, l.consumed, l.digest, l.replaced = nil, 0, sum(nil), l.replaced+1
	}
	l.size, l.modified = info.Size(), info.ModTime()
	rest := data[l.consumed:]
	before := len(l.records)
	for {
		i := bytes.IndexByte(rest, '\n')
		if i < 0 {
			break
		}
		l.records = append(l.records, decodeRecord(len(l.records)+1, rest[:i]))
		l.consumed += int64(i + 1)
		rest = rest[i+1:]
	}
	l.digest = sum(data[:l.consumed])
	tail := len(rest)
	changed := len(l.records) != before || tail != l.tail
	l.tail = tail
	return changed, nil
}

func (l *runLog) refreshReportLocked() (bool, error) {
	path := filepath.Join(l.dir, "report.json")
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		changed := l.report != nil
		l.report, l.reportAt = nil, time.Time{}
		return changed, nil
	}
	if err != nil {
		return false, err
	}
	if l.report != nil && info.ModTime().Equal(l.reportAt) {
		return false, nil
	}
	data, err := readBounded(path, maxReportBytes)
	if err != nil {
		return false, err
	}
	report, err := decodeReport(data)
	if err != nil {
		return false, fmt.Errorf("report.json: %w", err)
	}
	l.report, l.reportAt = report, info.ModTime()
	return true, nil
}

func sum(data []byte) []byte {
	s := sha256.Sum256(data)
	return s[:]
}

func readBounded(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", filepath.Base(path), limit)
	}
	return data, nil
}

func decodeRecord(seq int, line []byte) Record {
	var raw struct {
		At      time.Time       `json:"at"`
		Kind    string          `json:"kind"`
		Pass    string          `json:"pass"`
		Manager string          `json:"manager"`
		Turn    string          `json:"turn"`
		Data    json.RawMessage `json:"data"`
	}
	r := Record{Seq: seq}
	if err := json.Unmarshal(bytes.TrimSuffix(line, []byte("\r")), &raw); err != nil {
		r.Invalid = err.Error()
		return r
	}
	if raw.Kind == "" {
		r.Invalid = "the record has no kind"
		return r
	}
	r.At, r.Kind, r.Pass, r.Manager, r.Turn, r.Data = raw.At, raw.Kind, raw.Pass, raw.Manager, raw.Turn, raw.Data
	return r
}

// stateDirectory is one state directory the cockpit observes read-only.
type stateDirectory struct {
	dir  string
	mu   sync.Mutex
	runs map[string]*runLog
}

// RunEntry is one run found in a state directory.
type RunEntry struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	Current bool   `json:"current"`
}

// list finds the runs of the state directory: directories under runs/ with
// an events.jsonl. current-run names the run that last opened the owner
// channel; it does not by itself say that run is still running.
func (s *stateDirectory) list() ([]RunEntry, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, "runs"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	current := s.currentRun()
	var out []RunEntry
	for _, e := range entries {
		if !e.IsDir() || !safeName.MatchString(e.Name()) {
			continue
		}
		if _, err := os.Stat(filepath.Join(s.dir, "runs", e.Name(), "events.jsonl")); err != nil {
			continue
		}
		out = append(out, RunEntry{ID: e.Name(), State: s.dir, Current: e.Name() == current})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func (s *stateDirectory) currentRun() string {
	data, err := readBounded(filepath.Join(s.dir, "current-run"), 4096)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (s *stateDirectory) log(id string) *runLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runs == nil {
		s.runs = map[string]*runLog{}
	}
	if l := s.runs[id]; l != nil {
		return l
	}
	l := newRunLog(s.dir, id)
	s.runs[id] = l
	return l
}
