package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

const QueueVersion = "markitect.government-queue/v1alpha1"

type QueueOptions struct {
	Repo, BacklogPath, QueueDirectory string
}

type QueueLimits struct {
	ActorStarts        int   `json:"actorStarts"`
	MaxRepairs         int   `json:"maxRepairs"`
	MaxWallTimeSeconds int64 `json:"maxWallTimeSeconds"`
	MaxParallelism     int   `json:"maxParallelism"`
}

type QueueJob struct {
	ID          string   `json:"id"`
	ConfigPath  string   `json:"configPath"`
	OrderPath   string   `json:"orderPath"`
	RuntimePath string   `json:"runtimePath"`
	DependsOn   []string `json:"dependsOn"`
}

type QueueBacklog struct {
	APIVersion     string      `json:"apiVersion"`
	StateDirectory string      `json:"stateDirectory"`
	Limits         QueueLimits `json:"limits"`
	Jobs           []QueueJob  `json:"jobs"`
}

type UsageCounter struct {
	Known   bool  `json:"known"`
	Total   int64 `json:"total"`
	Unknown bool  `json:"unknown"`
}

type QueueUsage struct {
	InputTokens                 UsageCounter `json:"inputTokens"`
	OutputTokens                UsageCounter `json:"outputTokens"`
	CachedTokens                UsageCounter `json:"cachedTokens"`
	ToolCalls                   UsageCounter `json:"toolCalls"`
	ObservedActorWallTimeMillis UsageCounter `json:"observedActorWallTimeMilliseconds"`
}

type QueueJobResult struct {
	ID           string `json:"id"`
	State        string `json:"state"`
	NextStep     string `json:"nextStep"`
	RunID        string `json:"runId,omitempty"`
	ReportPath   string `json:"reportPath,omitempty"`
	ReportDigest string `json:"reportDigest,omitempty"`
	Error        string `json:"error,omitempty"`
}

type QueueReport struct {
	APIVersion       string           `json:"apiVersion"`
	QueueID          string           `json:"queueId"`
	QueueDirectory   string           `json:"queueDirectory"`
	Status           string           `json:"status"`
	NextStep         string           `json:"nextStep"`
	BacklogDigest    string           `json:"backlogDigest"`
	Limits           QueueLimits      `json:"limits"`
	ActorStarts      int              `json:"actorStarts"`
	Repairs          int              `json:"repairs"`
	InFlightActors   int              `json:"inFlightActors"`
	PeakParallelism  int              `json:"peakParallelism"`
	ReservedWallTime int64            `json:"reservedWallTimeSeconds"`
	ElapsedWallTime  int64            `json:"elapsedWallTimeSeconds"`
	Usage            QueueUsage       `json:"usage"`
	Jobs             []QueueJobResult `json:"jobs"`
	JournalSequence  uint64           `json:"journalSequence"`
	JournalDigest    string           `json:"journalDigest"`
	StartedAt        time.Time        `json:"startedAt"`
}

type frozenJob struct {
	QueueJob
	Runtime        Runtime `json:"-"`
	TimeoutSeconds int     `json:"timeoutSeconds"`
	RuntimeDigest  string  `json:"runtimeDigest"`
	RuntimePin     string  `json:"runtimePin"`
	ToolPins       string  `json:"toolPins"`
}

type queueJournalEvent struct {
	Sequence uint64          `json:"sequence"`
	Previous string          `json:"previous"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
	Digest   string          `json:"digest"`
}

type queueState struct {
	Report            QueueReport
	BacklogBytes      []byte
	Jobs              map[string]frozenJob
	Dependencies      map[string][]string
	Checkpoints       map[string]Report
	TerminalStop      bool
	pendingActors     map[string]bool
	actorReservations map[string]bool
	activeJob         string
	repoIdentity      string
}

// StartQueue fixes the caller-provided finite backlog and its runtime bindings
// in a new durable directory. A later invocation must name that directory.
func StartQueue(ctx context.Context, opts QueueOptions) (QueueReport, error) {
	if ctx == nil {
		return QueueReport{}, errors.New("queue context is required")
	}
	backlog, raw, err := readQueueBacklog(opts.Repo, opts.BacklogPath)
	if err != nil {
		return QueueReport{}, err
	}
	parent, err := realDirectory(backlog.StateDirectory)
	if err != nil {
		return QueueReport{}, fmt.Errorf("queue stateDirectory: %w", err)
	}
	if _, err := queueRepo(opts.Repo, parent); err != nil {
		return QueueReport{}, err
	}
	// CreateOperationalDirectory returns a unique path; do not derive queue
	// identity from a mutable pointer or scan the parent's contents.
	queueDir, err := government.CreateOperationalDirectory(parent, "government-queue-")
	if err != nil {
		return QueueReport{}, err
	}
	return createQueue(ctx, opts, queueDir, backlog, raw)
}

// ResumeQueue reconstructs the queue only from its hash-chained journal and
// rejects a changed backlog or any changed runtime binding.

func ResumeQueue(ctx context.Context, opts QueueOptions) (QueueReport, error) {
	if ctx == nil {
		return QueueReport{}, errors.New("queue context is required")
	}
	if opts.QueueDirectory == "" || !filepath.IsAbs(opts.QueueDirectory) {
		return QueueReport{}, errors.New("resume requires an absolute queue directory")
	}
	dir, err := realDirectory(opts.QueueDirectory)
	if err != nil {
		return QueueReport{}, err
	}
	lease, err := government.AcquireLease(filepath.Join(dir, "queue.lease"))
	if err != nil {
		return QueueReport{}, err
	}
	defer lease.Close()
	state, err := loadQueueState(dir)
	if err != nil {
		return QueueReport{}, err
	}
	raw, inputErr := readExternalQueueFile(opts.Repo, opts.BacklogPath)
	if inputErr != nil {
		return persistStaleQueue(dir, state, "backlog is unavailable: "+inputErr.Error())
	}
	if state.Report.BacklogDigest != digestBytes(raw) || !bytes.Equal(state.BacklogBytes, raw) {
		return persistStaleQueue(dir, state, "backlog bytes changed; old queue assent is stale")
	}
	backlog, _, decodeErr := decodeQueueBacklog(raw)
	if decodeErr != nil {
		return persistStaleQueue(dir, state, "backlog is no longer valid: "+decodeErr.Error())
	}
	identity, err := queueRepositoryIdentity(opts.Repo)
	if err != nil {
		return persistStaleQueue(dir, state, "repository identity cannot be revalidated: "+err.Error())
	}
	if state.repoIdentity != identity {
		return persistStaleQueue(dir, state, "repository identity changed; old queue assent is stale")
	}
	if err := validateQueueBacklog(opts.Repo, backlog, raw); err != nil {
		return persistStaleQueue(dir, state, "backlog bindings are invalid: "+err.Error())
	}
	if err := rehydrateFrozenJobs(opts.Repo, backlog, state); err != nil {
		return persistStaleQueue(dir, state, "runtime binding cannot be reloaded: "+err.Error())
	}
	if err := revalidateFrozenJobs(opts.Repo, backlog, state); err != nil {
		return persistStaleQueue(dir, state, "runtime binding changed: "+err.Error())
	}
	qc := &queueControl{dir: dir, state: state, lease: lease}
	if err := recoverInterrupted(ctx, opts.Repo, qc); err != nil {
		return state.Report, err
	}
	if state.TerminalStop || queueTerminal(state.Report) {
		return state.Report, nil
	}
	return scheduleQueue(ctx, opts.Repo, qc)
}

func createQueue(ctx context.Context, opts QueueOptions, dir string, backlog QueueBacklog, raw []byte) (QueueReport, error) {
	if err := validateQueueBacklog(opts.Repo, backlog, raw); err != nil {
		return QueueReport{}, err
	}
	lease, err := government.AcquireLease(filepath.Join(dir, "queue.lease"))
	if err != nil {
		return QueueReport{}, err
	}
	defer lease.Close()
	state := &queueState{BacklogBytes: append([]byte(nil), raw...), Jobs: map[string]frozenJob{}, Dependencies: map[string][]string{}, Checkpoints: map[string]Report{}, pendingActors: map[string]bool{}, actorReservations: map[string]bool{}}
	identity, err := queueRepositoryIdentity(opts.Repo)
	if err != nil {
		return QueueReport{}, err
	}
	state.repoIdentity = identity
	state.Report = QueueReport{APIVersion: QueueVersion, QueueID: filepath.Base(dir), QueueDirectory: dir, Status: "queued", NextStep: "run finite backlog", BacklogDigest: digestBytes(raw), Limits: backlog.Limits, StartedAt: time.Now().UTC(), Jobs: []QueueJobResult{}}
	for _, job := range backlog.Jobs {
		runtime, err := ReadRuntime(opts.Repo, job.RuntimePath)
		if err != nil {
			return state.Report, fmt.Errorf("job %q runtime: %w", job.ID, err)
		}
		pin, err := FingerprintRuntime(runtime)
		if err != nil {
			return state.Report, fmt.Errorf("job %q runtime fingerprint: %w", job.ID, err)
		}
		if backlog.Limits.MaxParallelism < runtimeParallelism(runtime) {
			return state.Report, fmt.Errorf("queue maxParallelism is below job %q runtime parallelism", job.ID)
		}
		tool, err := toolPins(runtime)
		if err != nil {
			return state.Report, fmt.Errorf("job %q tool pins: %w", job.ID, err)
		}
		runtimeBytes, err := readExternalQueueFile(opts.Repo, job.RuntimePath)
		if err != nil {
			return state.Report, err
		}
		state.Jobs[job.ID] = frozenJob{QueueJob: job, Runtime: runtime, TimeoutSeconds: runtime.TimeoutSeconds, RuntimeDigest: digestBytes(runtimeBytes), RuntimePin: pin, ToolPins: tool}
		state.Dependencies[job.ID] = append([]string(nil), job.DependsOn...)
		state.Report.Jobs = append(state.Report.Jobs, QueueJobResult{ID: job.ID, State: "queued", NextStep: "await dependencies"})
	}
	bindings := map[string]frozenJob{}
	for id, job := range state.Jobs {
		bindings[id] = frozenJob{QueueJob: job.QueueJob, TimeoutSeconds: job.TimeoutSeconds, RuntimeDigest: job.RuntimeDigest, RuntimePin: job.RuntimePin, ToolPins: job.ToolPins}
	}
	if err := government.WriteOperationalRecord(filepath.Join(dir, "events.jsonl"), []byte{}); err != nil {
		return state.Report, err
	}
	if err := appendQueueEvent(dir, state, "created", map[string]any{"backlog": raw, "queueId": state.Report.QueueID, "startedAt": state.Report.StartedAt, "bindings": bindings, "repoIdentity": identity}); err != nil {
		return state.Report, err
	}
	if err := saveQueueReport(dir, state); err != nil {
		return state.Report, err
	}
	return scheduleQueue(ctx, opts.Repo, &queueControl{dir: dir, state: state, lease: lease})
}

func scheduleQueue(ctx context.Context, repo string, q *queueControl) (QueueReport, error) {
	byID := map[string]*QueueJobResult{}
	for i := range q.state.Report.Jobs {
		byID[q.state.Report.Jobs[i].ID] = &q.state.Report.Jobs[i]
	}
	for _, f := range frozenInOrder(q.state) {
		if err := q.lease.Verify(); err != nil {
			return q.state.Report, err
		}
		result := byID[f.ID]
		if result.State == "accepted-scoped" || result.State == "blocked" || result.State == "incomplete" {
			continue
		}
		depBlocked, depPending := false, false
		for _, dep := range f.DependsOn {
			state := byID[dep].State
			if state == "blocked" || state == "incomplete" {
				depBlocked = true
			}
			if state != "accepted-scoped" {
				depPending = true
			}
		}
		if depBlocked {
			result.State = "blocked"
			result.NextStep = "dependency blocked"
			result.Error = "an explicit prerequisite did not complete"
			if err := q.event("job-blocked", result); err != nil {
				return q.state.Report, err
			}
			continue
		}
		if depPending {
			result.State = "blocked"
			result.NextStep = "dependency pending"
			result.Error = "an explicit prerequisite is unresolved"
			if err := q.event("job-blocked", result); err != nil {
				return q.state.Report, err
			}
			continue
		}
		requiredParallelism := runtimeParallelism(f.Runtime)
		if q.state.Report.InFlightActors+requiredParallelism > q.state.Report.Limits.MaxParallelism {
			result.State = "blocked"
			result.NextStep = "inspect unknown in-flight actors and issue a fresh queue if capacity is unavailable"
			result.Error = fmt.Sprintf("unknown in-flight actor reservations occupy %d queue slots; job runtime requires %d of maximum %d", q.state.Report.InFlightActors, requiredParallelism, q.state.Report.Limits.MaxParallelism)
			if err := q.event("job-blocked", result); err != nil {
				return q.state.Report, err
			}
			continue
		}
		elapsed := int64(time.Since(q.state.Report.StartedAt).Seconds())
		if q.state.Report.ReservedWallTime+int64(f.Runtime.TimeoutSeconds) > q.state.Report.Limits.MaxWallTimeSeconds || elapsed+int64(f.Runtime.TimeoutSeconds) > q.state.Report.Limits.MaxWallTimeSeconds {
			result.State = "blocked"
			result.NextStep = "none"
			result.Error = "cumulative reserved wall-time limit exhausted"
			if err := q.event("job-blocked", result); err != nil {
				return q.state.Report, err
			}
			continue
		}
		active, err := gitOutput(ctx, repo, nil, nil, "rev-parse", "--verify", f.Runtime.ActiveRef+"^{commit}")
		if err != nil || active != f.Runtime.ExpectedBase {
			result.State = "blocked"
			result.NextStep = "issue a fresh order and runtime binding"
			result.Error = "active ref differs from immutable runtime base"
			if err := q.event("job-blocked", result); err != nil {
				return q.state.Report, err
			}
			continue
		}
		q.state.Report.ReservedWallTime += int64(f.Runtime.TimeoutSeconds)
		q.state.Report.ElapsedWallTime = int64(time.Since(q.state.Report.StartedAt).Seconds())
		result.State = "running"
		result.NextStep = "run once"
		result.Error = ""
		q.state.activeJob = f.ID
		if err := q.event("attempt-reserved", map[string]any{"jobId": f.ID, "job": *result, "wallSeconds": f.Runtime.TimeoutSeconds, "reserved": q.state.Report.ReservedWallTime}); err != nil {
			return q.state.Report, err
		}
		if err := saveQueueReport(q.dir, q.state); err != nil {
			return q.state.Report, err
		}
		runReport, runErr := Run(ctx, Options{Repo: repo, ConfigPath: f.ConfigPath, OrderPath: f.OrderPath, Runtime: f.Runtime, Control: q})
		result.RunID = runReport.RunID
		result.ReportPath = runReport.ReportPath
		if runReport.Status == "accepted-scoped" && runErr == nil {
			result.State = "accepted-scoped"
			result.NextStep = "complete"
			result.Error = ""
		} else if runReport.Status == "blocked" {
			result.State = "blocked"
			result.NextStep = "none"
			result.Error = errorText(runErr)
		} else {
			result.State = "incomplete"
			result.NextStep = "inspect durable report and escalate"
			result.Error = errorText(runErr)
		}
		if runReport.ReportPath != "" {
			if data, e := os.ReadFile(runReport.ReportPath); e == nil {
				result.ReportDigest = digestBytes(data)
			}
		}
		if err := q.event("job-finished", result); err != nil {
			return q.state.Report, err
		}
		if q.fatal != nil {
			return q.state.Report, q.fatal
		}
		// A blocked or incomplete job does not stop independent explicit branches.
	}
	allDone, anyIncomplete, anyBlocked := true, false, false
	for _, j := range q.state.Report.Jobs {
		if j.State == "incomplete" {
			anyIncomplete = true
		}
		if j.State == "blocked" {
			anyBlocked = true
		}
		if j.State != "accepted-scoped" && j.State != "blocked" && j.State != "incomplete" {
			allDone = false
		}
	}
	switch {
	case anyIncomplete:
		q.state.Report.Status = "incomplete"
		q.state.Report.NextStep = "inspect incomplete jobs and escalation"
	case allDone && anyBlocked:
		q.state.Report.Status = "blocked"
		q.state.Report.NextStep = "none"
	case allDone:
		q.state.Report.Status = "complete"
		q.state.Report.NextStep = "none"
	default:
		q.state.Report.Status = "incomplete"
		q.state.Report.NextStep = "resume finite queue"
	}
	q.state.Report.ElapsedWallTime = int64(time.Since(q.state.Report.StartedAt).Seconds())
	if err := q.event("queue-finished", q.state.Report); err != nil {
		return q.state.Report, err
	}
	q.state.TerminalStop = true
	if err := saveQueueReport(q.dir, q.state); err != nil {
		return q.state.Report, err
	}
	return q.state.Report, nil
}

func recoverInterrupted(ctx context.Context, repo string, q *queueControl) error {
	for i := range q.state.Report.Jobs {
		job := &q.state.Report.Jobs[i]
		if job.State != "running" {
			continue
		}
		report, ok := q.state.Checkpoints[job.RunID]
		promotionCheckpoint := report.Stage == "promote" || report.Stage == "amendment-promote" || (report.Stage == "complete" && report.Promotion != nil && report.Promotion.Status == "promoted")
		if ok && promotionCheckpoint && report.Candidate != nil && report.Evidence != nil && report.Decision != nil && report.Candidate.ID != "" && report.Evidence.ID != "" && report.Decision.ID != "" && report.CandidateCommit != "" && report.CandidateTree != "" && report.RunID == job.RunID {
			f := q.state.Jobs[job.ID]
			request := government.PromotionRequest{Repo: repo, ActiveRef: f.Runtime.ActiveRef, ExpectedOld: report.BaseRevision, NewCommit: report.CandidateCommit, ExpectedTreeID: report.CandidateTree, MaterialCandidateID: report.Candidate.ID, EvidenceID: report.Evidence.ID, DecisionID: report.Decision.ID, StateDirectory: filepath.Dir(report.ReportPath), IdempotencyKey: report.RunID}
			if request.ExpectedOld == f.Runtime.ExpectedBase && report.ActiveRef == request.ActiveRef && report.RuntimePin == f.RuntimePin && report.ToolPins == f.ToolPins {
				promotion, err := government.RecoverPromotion(ctx, request)
				report.Promotion = &promotion
				if err == nil && promotion.Status == "promoted" {
					report.Stage = "complete"
					report.Status = "accepted-scoped"
					reportPath, digest, err := persistRecoveredReport(report)
					if err != nil {
						return err
					}
					job.State = "accepted-scoped"
					job.NextStep = "complete"
					job.ReportPath = reportPath
					job.ReportDigest = digest
					if err := q.event("recovered-promotion", map[string]any{"report": report, "job": *job}); err != nil {
						return err
					}
					continue
				}
				job.Error = errors.Join(errors.New("promotion recovery remains incomplete"), err).Error()
			}
		}
		job.State = "incomplete"
		job.NextStep = "inspect durable report and escalate"
		if job.Error == "" {
			job.Error = "runner was interrupted; actor effects were not replayed"
		}
		if ok && report.ReportPath != "" {
			report.Status = "incomplete"
			report.Error = job.Error
			if path, digest, publishErr := persistRecoveredReport(report); publishErr == nil {
				job.ReportPath = path
				job.ReportDigest = digest
			} else {
				job.Error = errors.Join(errors.New(job.Error), publishErr).Error()
			}
		}
		if job.ReportPath != "" {
			if _, err := os.Stat(job.ReportPath); err != nil {
				job.ReportPath = ""
				job.Error += "; no immutable run report was published before interruption"
			}
		}
		if err := q.event("interrupted-incomplete", *job); err != nil {
			return err
		}
	}
	return saveQueueReport(q.dir, q.state)
}

func markQueueStale(s *queueState, reason string) {
	s.TerminalStop = true
	for i := range s.Report.Jobs {
		if s.Report.Jobs[i].State != "accepted-scoped" {
			s.Report.Jobs[i].State = "blocked"
			s.Report.Jobs[i].NextStep = "issue a fresh queue"
		}
		s.Report.Jobs[i].Error = reason
	}
	s.Report.Status = "blocked"
	s.Report.NextStep = "none"
}

func persistStaleQueue(dir string, s *queueState, reason string) (QueueReport, error) {
	if s.TerminalStop && s.Report.Status == "blocked" {
		return s.Report, errors.New(reason)
	}
	markQueueStale(s, reason)
	if err := appendQueueEvent(dir, s, "stale-bindings", map[string]string{"error": reason}); err != nil {
		return s.Report, errors.Join(errors.New(reason), err)
	}
	if err := saveQueueReport(dir, s); err != nil {
		return s.Report, errors.Join(errors.New(reason), err)
	}
	return s.Report, errors.New(reason)
}

func frozenInOrder(s *queueState) []frozenJob {
	out := make([]frozenJob, 0, len(s.Jobs))
	for _, j := range s.Report.Jobs {
		if f, ok := s.Jobs[j.ID]; ok {
			out = append(out, f)
		}
	}
	return out
}

func queueTerminal(r QueueReport) bool { return r.Status == "complete" || r.Status == "blocked" }

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
