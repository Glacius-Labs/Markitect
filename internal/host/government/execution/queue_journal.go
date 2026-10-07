package execution

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

func digestBytes(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func persistRecoveredReport(report Report) (string, string, error) {
	dir := filepath.Dir(report.ReportPath)
	path := report.ReportPath
	if _, err := os.Stat(path); err == nil {
		path = filepath.Join(dir, "recovered-report.json")
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", "", err
	}
	data = append(data, '\n')
	if prior, e := os.ReadFile(path); e == nil {
		if !bytes.Equal(prior, data) {
			return "", "", errors.New("recovered run report already exists with different bytes")
		}
		return path, digestBytes(prior), nil
	}
	if err := government.WriteOperationalRecord(path, data); err != nil {
		return "", "", err
	}
	return path, digestBytes(data), nil
}

func appendQueueEvent(dir string, s *queueState, typ string, payload any) error {
	return appendQueueEventLocked(dir, s, typ, payload)
}

func appendQueueEventLocked(dir string, s *queueState, typ string, payload any) error {
	p, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	e := queueJournalEvent{Sequence: s.Report.JournalSequence + 1, Previous: s.Report.JournalDigest, Type: typ, Payload: p}
	unsigned, err := json.Marshal(struct {
		Sequence uint64          `json:"sequence"`
		Previous string          `json:"previous"`
		Type     string          `json:"type"`
		Payload  json.RawMessage `json:"payload"`
	}{e.Sequence, e.Previous, e.Type, e.Payload})
	if err != nil {
		return err
	}
	e.Digest = digestBytes(unsigned)
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	written, writeErr := f.Write(append(line, '\n'))
	if writeErr == nil && written != len(line)+1 {
		writeErr = errors.New("short queue journal write")
	}
	syncErr := f.Sync()
	closeErr := f.Close()
	if err = errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	s.Report.JournalSequence = e.Sequence
	s.Report.JournalDigest = e.Digest
	return nil
}

func loadQueueState(dir string) (*queueState, error) {
	data, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		return nil, err
	}
	if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
		return nil, errors.New("queue journal ends with an incomplete record")
	}
	s := &queueState{Jobs: map[string]frozenJob{}, Dependencies: map[string][]string{}, Checkpoints: map[string]Report{}, pendingActors: map[string]bool{}, actorReservations: map[string]bool{}}
	var seq uint64
	prev := ""
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var e queueJournalEvent
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("decode queue journal: %w", err)
		}
		unsigned, err := json.Marshal(struct {
			Sequence uint64          `json:"sequence"`
			Previous string          `json:"previous"`
			Type     string          `json:"type"`
			Payload  json.RawMessage `json:"payload"`
		}{e.Sequence, e.Previous, e.Type, e.Payload})
		if err != nil {
			return nil, err
		}
		if e.Sequence != seq+1 || e.Previous != prev || e.Digest != digestBytes(unsigned) {
			return nil, errors.New("queue journal hash chain is invalid")
		}
		seq = e.Sequence
		prev = e.Digest
		s.Report.JournalSequence = e.Sequence - 1
		s.Report.JournalDigest = e.Previous
		switch e.Type {
		case "created":
			if s.Report.APIVersion != "" || e.Sequence != 1 {
				return nil, errors.New("queue journal has duplicate or out-of-order creation record")
			}
			var v struct {
				Backlog      []byte               `json:"backlog"`
				QueueID      string               `json:"queueId"`
				StartedAt    time.Time            `json:"startedAt"`
				Bindings     map[string]frozenJob `json:"bindings"`
				RepoIdentity string               `json:"repoIdentity"`
			}
			if err := json.Unmarshal(e.Payload, &v); err != nil {
				return nil, err
			}
			if v.QueueID == "" || v.QueueID != filepath.Base(dir) || v.Backlog == nil || v.RepoIdentity == "" {
				return nil, errors.New("queue creation record has invalid identity bindings")
			}
			s.BacklogBytes = v.Backlog
			b, _, err := decodeQueueBacklog(v.Backlog)
			if err != nil {
				return nil, err
			}
			s.Report = QueueReport{APIVersion: QueueVersion, QueueID: v.QueueID, QueueDirectory: dir, Status: "queued", NextStep: "run finite backlog", BacklogDigest: digestBytes(v.Backlog), Limits: b.Limits, StartedAt: v.StartedAt, Jobs: []QueueJobResult{}}
			s.repoIdentity = v.RepoIdentity
			for _, j := range b.Jobs {
				s.Report.Jobs = append(s.Report.Jobs, QueueJobResult{ID: j.ID, State: "queued", NextStep: "await dependencies"})
				s.Dependencies[j.ID] = append([]string(nil), j.DependsOn...)
				if frozen, ok := v.Bindings[j.ID]; ok {
					s.Jobs[j.ID] = frozen
				}
			}
			if len(s.Jobs) != len(b.Jobs) {
				return nil, errors.New("queue creation record omits a frozen job binding")
			}
		case "job-blocked", "job-finished", "interrupted-incomplete", "attempt-reserved":
			var j QueueJobResult
			if e.Type == "attempt-reserved" {
				var v struct {
					Job   QueueJobResult `json:"job"`
					JobID string         `json:"jobId"`
					Wall  int64          `json:"wallSeconds"`
				}
				if err := json.Unmarshal(e.Payload, &v); err != nil {
					return nil, err
				}
				frozen, ok := s.Jobs[v.JobID]
				if !ok || v.JobID != v.Job.ID || v.Job.State != "running" || v.Wall < 1 || v.Wall != int64(frozen.TimeoutSeconds) || s.Report.ReservedWallTime+v.Wall > s.Report.Limits.MaxWallTimeSeconds {
					return nil, errors.New("queue wall-time reservation is invalid or exceeds its frozen budget")
				}
				s.Report.ReservedWallTime += v.Wall
				j = v.Job
			} else if err := json.Unmarshal(e.Payload, &j); err != nil {
				return nil, err
			}
			if j.ID != "" {
				setQueueJob(&s.Report, j)
			}
		case "run-started":
			var v struct {
				JobID      string `json:"jobId"`
				RunID      string `json:"runId"`
				ReportPath string `json:"reportPath"`
			}
			if err := json.Unmarshal(e.Payload, &v); err != nil {
				return nil, err
			}
			for i := range s.Report.Jobs {
				if s.Report.Jobs[i].ID == v.JobID {
					s.Report.Jobs[i].RunID = v.RunID
					s.Report.Jobs[i].ReportPath = v.ReportPath
				}
			}
		case "checkpoint":
			var r Report
			if err := json.Unmarshal(e.Payload, &r); err != nil {
				return nil, err
			}
			s.Checkpoints[r.RunID] = r
		case "recovered-promotion":
			var v struct {
				Report Report         `json:"report"`
				Job    QueueJobResult `json:"job"`
			}
			if err := json.Unmarshal(e.Payload, &v); err != nil {
				return nil, err
			}
			s.Checkpoints[v.Report.RunID] = v.Report
			setQueueJob(&s.Report, v.Job)
		case "queue-finished":
			var report QueueReport
			if err := json.Unmarshal(e.Payload, &report); err != nil {
				return nil, err
			}
			markUnknownPendingUsage(s)
			jobsA, err := json.Marshal(report.Jobs)
			if err != nil {
				return nil, err
			}
			jobsB, err := json.Marshal(s.Report.Jobs)
			if err != nil {
				return nil, err
			}
			if report.QueueID != s.Report.QueueID || report.BacklogDigest != s.Report.BacklogDigest || report.ActorStarts != s.Report.ActorStarts || report.Repairs != s.Report.Repairs || report.ReservedWallTime != s.Report.ReservedWallTime || report.InFlightActors != s.Report.InFlightActors || report.PeakParallelism != s.Report.PeakParallelism || report.Limits != s.Report.Limits || !bytes.Equal(jobsA, jobsB) || report.Usage != s.Report.Usage || report.JournalSequence != e.Sequence-1 || report.JournalDigest != e.Previous || !report.StartedAt.Equal(s.Report.StartedAt) {
				return nil, errors.New("terminal queue report counters or bindings differ from replayed journal")
			}
			for _, prior := range s.Report.Jobs {
				found := false
				for _, current := range report.Jobs {
					if current.ID == prior.ID {
						found = true
						break
					}
				}
				if !found {
					return nil, errors.New("terminal queue report job identities differ from replayed journal")
				}
			}
			s.Report = report
			s.TerminalStop = true
		case "actor-reserved":
			s.Report.ActorStarts++
			if s.Report.ActorStarts > s.Report.Limits.ActorStarts {
				return nil, errors.New("queue journal exceeds actor-start limit")
			}
			var v struct {
				RunID    string `json:"runId"`
				Sequence int    `json:"sequence"`
			}
			if err := json.Unmarshal(e.Payload, &v); err != nil || v.RunID == "" || v.Sequence < 1 {
				return nil, errors.New("queue actor reservation record is invalid")
			}
			if s.pendingActors == nil {
				s.pendingActors = map[string]bool{}
			}
			if s.actorReservations == nil {
				s.actorReservations = map[string]bool{}
			}
			key := fmt.Sprintf("%s/%d", v.RunID, v.Sequence)
			if s.actorReservations[key] {
				return nil, errors.New("queue actor reservation identity is duplicated")
			}
			s.actorReservations[key] = true
			s.pendingActors[key] = true
			s.Report.InFlightActors++
			if s.Report.InFlightActors > s.Report.Limits.MaxParallelism {
				return nil, errors.New("queue journal exceeds frozen parallelism limit")
			}
			if s.Report.InFlightActors > s.Report.PeakParallelism {
				s.Report.PeakParallelism = s.Report.InFlightActors
			}
			var counts struct {
				InFlight int `json:"inFlight"`
				Peak     int `json:"peak"`
			}
			if err := json.Unmarshal(e.Payload, &counts); err != nil || counts.InFlight != s.Report.InFlightActors || counts.Peak != s.Report.PeakParallelism {
				return nil, errors.New("queue actor reservation counters do not match replay")
			}
		case "repair-reserved":
			s.Report.Repairs++
			if s.Report.Repairs > s.Report.Limits.MaxRepairs {
				return nil, errors.New("queue journal exceeds repair limit")
			}
		case "actor-completed":
			var v struct {
				RunID  string      `json:"runId"`
				Record ActorRecord `json:"record"`
				Usage  QueueUsage  `json:"usage"`
			}
			if err := json.Unmarshal(e.Payload, &v); err != nil || v.RunID == "" || v.Record.Sequence < 1 {
				return nil, errors.New("queue actor completion record is invalid")
			}
			s.Report.Usage = v.Usage
			key := fmt.Sprintf("%s/%d", v.RunID, v.Record.Sequence)
			if !s.pendingActors[key] || s.Report.InFlightActors < 1 {
				return nil, errors.New("queue actor completion lacks an outstanding reservation")
			}
			delete(s.pendingActors, key)
			s.Report.InFlightActors--
			var counts struct {
				InFlight int `json:"inFlight"`
				Peak     int `json:"peak"`
			}
			if err := json.Unmarshal(e.Payload, &counts); err != nil || counts.InFlight != s.Report.InFlightActors || counts.Peak != s.Report.PeakParallelism {
				return nil, errors.New("queue actor completion counters do not match replay")
			}
		case "stale-bindings":
			var v struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(e.Payload, &v); err != nil || v.Error == "" {
				return nil, errors.New("queue stale-stop record is invalid")
			}
			markQueueStale(s, v.Error)
		default:
			return nil, fmt.Errorf("unknown queue journal event %q", e.Type)
		}
	}
	if seq == 0 || s.Report.APIVersion == "" {
		return nil, errors.New("queue journal has no creation record")
	}
	s.Report.JournalSequence = seq
	s.Report.JournalDigest = prev
	s.Report.QueueDirectory = dir
	if s.Report.QueueID != filepath.Base(dir) {
		return nil, errors.New("queue directory identity does not match creation record")
	}
	markUnknownPendingUsage(s)
	return s, nil
}

func markUnknownPendingUsage(s *queueState) {
	if len(s.pendingActors) > 0 {
		s.Report.Usage.InputTokens.Unknown = true
		s.Report.Usage.OutputTokens.Unknown = true
		s.Report.Usage.CachedTokens.Unknown = true
		s.Report.Usage.ToolCalls.Unknown = true
		s.Report.Usage.ObservedActorWallTimeMillis.Unknown = true
	}
}

func decodeQueueBacklog(data []byte) (QueueBacklog, []byte, error) {
	if err := rejectRuntimeDuplicateKeys(data); err != nil {
		return QueueBacklog{}, nil, err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var b QueueBacklog
	if err := d.Decode(&b); err != nil {
		return b, nil, err
	}
	return b, data, nil
}

func setQueueJob(r *QueueReport, j QueueJobResult) {
	for i := range r.Jobs {
		if r.Jobs[i].ID == j.ID {
			r.Jobs[i] = j
			return
		}
	}
}

func saveQueueReport(dir string, s *queueState) error {
	data, err := json.MarshalIndent(s.Report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := filepath.Join(dir, fmt.Sprintf("queue-report-%08d.json", s.Report.JournalSequence))
	if prior, e := os.ReadFile(path); e == nil {
		if bytes.Equal(prior, data) {
			return nil
		}
		return errors.New("immutable queue report snapshot already exists with different contents")
	}
	return government.WriteOperationalRecord(path, data)
}
