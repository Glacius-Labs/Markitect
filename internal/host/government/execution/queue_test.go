package execution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

func TestEmptyQueueResumeIsQuietAndDurable(t *testing.T) {
	opts := fixtureOptions(t)
	setRuntimeModelOptions(&opts.Runtime)
	backlogPath := filepath.Join(filepath.Dir(opts.Repo), "empty-queue.json")
	backlog := QueueBacklog{
		APIVersion:     QueueVersion,
		StateDirectory: opts.Runtime.StateDirectory,
		Limits:         QueueLimits{ActorStarts: 1, MaxRepairs: 0, MaxWallTimeSeconds: 10, MaxParallelism: 1},
		Jobs:           []QueueJob{},
	}
	data, err := json.Marshal(backlog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backlogPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	started, err := StartQueue(context.Background(), QueueOptions{Repo: opts.Repo, BacklogPath: backlogPath})
	if err != nil {
		t.Fatal(err)
	}
	if started.Status != "complete" || started.ActorStarts != 0 || len(started.Jobs) != 0 || started.QueueDirectory == "" {
		t.Fatalf("empty finite queue did not finish quietly: %+v", started)
	}
	snapshots, err := filepath.Glob(filepath.Join(started.QueueDirectory, "queue-report-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := ResumeQueue(context.Background(), QueueOptions{Repo: opts.Repo, BacklogPath: backlogPath, QueueDirectory: started.QueueDirectory})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != "complete" || resumed.ActorStarts != 0 || resumed.JournalSequence != started.JournalSequence || resumed.JournalDigest != started.JournalDigest {
		t.Fatalf("unchanged empty resume changed terminal state: before=%+v after=%+v", started, resumed)
	}
	after, err := filepath.Glob(filepath.Join(started.QueueDirectory, "queue-report-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(snapshots) {
		t.Fatalf("quiet resume created report snapshots: before=%d after=%d", len(snapshots), len(after))
	}
}

func TestQueueRejectedActorReservationsDoNotMutateState(t *testing.T) {
	dir := t.TempDir()
	limits := QueueLimits{ActorStarts: 2, MaxRepairs: 0, MaxWallTimeSeconds: 30, MaxParallelism: 1}
	backlog := QueueBacklog{APIVersion: QueueVersion, StateDirectory: dir, Limits: limits, Jobs: []QueueJob{}}
	raw, err := json.Marshal(backlog)
	if err != nil {
		t.Fatal(err)
	}
	if err := government.WriteOperationalRecord(filepath.Join(dir, "events.jsonl"), []byte{}); err != nil {
		t.Fatal(err)
	}
	lease, err := government.AcquireLease(filepath.Join(dir, "queue.lease"))
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	started := time.Now().UTC()
	state := &queueState{BacklogBytes: raw, Jobs: map[string]frozenJob{}, Dependencies: map[string][]string{}, Checkpoints: map[string]Report{}, Report: QueueReport{APIVersion: QueueVersion, QueueID: filepath.Base(dir), QueueDirectory: dir, Status: "queued", BacklogDigest: digestBytes(raw), Limits: limits, StartedAt: started, Jobs: []QueueJobResult{}}}
	if err := appendQueueEvent(dir, state, "created", map[string]any{"backlog": raw, "queueId": state.Report.QueueID, "startedAt": started, "bindings": map[string]frozenJob{}, "repoIdentity": "repo-test"}); err != nil {
		t.Fatal(err)
	}
	q := &queueControl{dir: dir, state: state, lease: lease}
	assertUnchanged := func() {
		t.Helper()
		if state.Report.ActorStarts != 0 || state.Report.InFlightActors != 0 || state.Report.PeakParallelism != 0 || len(state.pendingActors) != 0 || len(state.actorReservations) != 0 {
			t.Fatalf("rejected actor reservation mutated counters or maps: report=%+v pending=%v reservations=%v", state.Report, state.pendingActors, state.actorReservations)
		}
	}
	if err := q.ReserveActor("", 1, "execute", RunnerSpec{}); err == nil {
		t.Fatal("empty run ID was accepted")
	}
	assertUnchanged()
	if err := q.ReserveActor("run", 0, "execute", RunnerSpec{}); err == nil {
		t.Fatal("nonpositive sequence was accepted")
	}
	assertUnchanged()
	if err := q.ReserveActor("run", 1, "execute", RunnerSpec{}); err != nil {
		t.Fatal(err)
	}
	if err := q.ReserveActor("run", 1, "execute", RunnerSpec{}); err == nil {
		t.Fatal("duplicate reservation identity was accepted")
	}
	if state.Report.ActorStarts != 1 || state.Report.InFlightActors != 1 || state.Report.PeakParallelism != 1 || len(state.pendingActors) != 1 || len(state.actorReservations) != 1 {
		t.Fatalf("valid reservation counters are inconsistent: report=%+v pending=%v reservations=%v", state.Report, state.pendingActors, state.actorReservations)
	}
	replayed, err := loadQueueState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Report.ActorStarts != 1 || replayed.Report.InFlightActors != 1 || replayed.Report.PeakParallelism != 1 || len(replayed.pendingActors) != 1 || len(replayed.actorReservations) != 1 {
		t.Fatalf("journal replay disagrees with accepted reservation: report=%+v pending=%v reservations=%v", replayed.Report, replayed.pendingActors, replayed.actorReservations)
	}
}

func TestQueueResumeKeepsReservedBudgetsAndDoesNotReplayPreStartRun(t *testing.T) {
	opts := fixtureOptions(t)
	setRuntimeModelOptions(&opts.Runtime)
	root := filepath.Dir(opts.Repo)
	runtimePath := filepath.Join(root, "runtime.json")
	runtimeBytes, err := json.Marshal(opts.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimePath, runtimeBytes, 0600); err != nil {
		t.Fatal(err)
	}
	backlog := QueueBacklog{APIVersion: QueueVersion, StateDirectory: opts.Runtime.StateDirectory, Limits: QueueLimits{ActorStarts: 1, MaxRepairs: 1, MaxWallTimeSeconds: 3600, MaxParallelism: 1}}
	for _, id := range []string{"actor-run", "pre-start", "independent"} {
		backlog.Jobs = append(backlog.Jobs, QueueJob{ID: id, ConfigPath: opts.ConfigPath, OrderPath: opts.OrderPath, RuntimePath: runtimePath, DependsOn: []string{}})
	}
	backlogPath := filepath.Join(root, "resume-queue.json")
	raw, err := json.Marshal(backlog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backlogPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateQueueBacklog(opts.Repo, backlog, raw); err != nil {
		t.Fatal(err)
	}
	dir, err := government.CreateOperationalDirectory(opts.Runtime.StateDirectory, "government-queue-")
	if err != nil {
		t.Fatal(err)
	}
	lease, err := government.AcquireLease(filepath.Join(dir, "queue.lease"))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := queueRepositoryIdentity(opts.Repo)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := FingerprintRuntime(opts.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	tool, err := toolPins(opts.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := government.WriteOperationalRecord(filepath.Join(dir, "events.jsonl"), []byte{}); err != nil {
		t.Fatal(err)
	}
	state := &queueState{BacklogBytes: raw, Jobs: map[string]frozenJob{}, Dependencies: map[string][]string{}, Checkpoints: map[string]Report{}, pendingActors: map[string]bool{}, actorReservations: map[string]bool{}, repoIdentity: identity}
	state.Report = QueueReport{APIVersion: QueueVersion, QueueID: filepath.Base(dir), QueueDirectory: dir, Status: "queued", NextStep: "run finite backlog", BacklogDigest: digestBytes(raw), Limits: backlog.Limits, StartedAt: time.Now().UTC()}
	bindings := map[string]frozenJob{}
	for _, job := range backlog.Jobs {
		f := frozenJob{QueueJob: job, TimeoutSeconds: opts.Runtime.TimeoutSeconds, RuntimeDigest: digestBytes(runtimeBytes), RuntimePin: pin, ToolPins: tool}
		state.Jobs[job.ID] = f
		state.Dependencies[job.ID] = job.DependsOn
		bindings[job.ID] = f
		state.Report.Jobs = append(state.Report.Jobs, QueueJobResult{ID: job.ID, State: "queued", NextStep: "await dependencies"})
	}
	if err := appendQueueEvent(dir, state, "created", map[string]any{"backlog": raw, "queueId": state.Report.QueueID, "startedAt": state.Report.StartedAt, "bindings": bindings, "repoIdentity": identity}); err != nil {
		t.Fatal(err)
	}
	qc := &queueControl{dir: dir, state: state, lease: lease}
	reserveAttempt := func(id string) {
		t.Helper()
		var job *QueueJobResult
		for i := range state.Report.Jobs {
			if state.Report.Jobs[i].ID == id {
				job = &state.Report.Jobs[i]
			}
		}
		if job == nil {
			t.Fatal("job missing")
		}
		job.State = "running"
		job.NextStep = "run once"
		state.Report.ReservedWallTime += int64(opts.Runtime.TimeoutSeconds)
		if err := qc.event("attempt-reserved", map[string]any{"jobId": id, "job": *job, "wallSeconds": opts.Runtime.TimeoutSeconds, "reserved": state.Report.ReservedWallTime}); err != nil {
			t.Fatal(err)
		}
	}
	reserveAttempt("actor-run")
	qc.state.activeJob = "actor-run"
	runDir := filepath.Join(dir, "actor-run")
	if err := qc.StartRun("interrupted-run", filepath.Join(runDir, "report.json")); err != nil {
		t.Fatal(err)
	}
	if err := qc.ReserveActor("interrupted-run", 1, "execute", opts.Runtime.Executor); err != nil {
		t.Fatal(err)
	}
	if err := qc.ReserveRepair("interrupted-run", "inventory/Area/reservations", 1); err != nil {
		t.Fatal(err)
	}
	reserveAttempt("pre-start") // durable attempt reservation before any run ID exists
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadQueueState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Report.ActorStarts != 1 || loaded.Report.Repairs != 1 || loaded.Report.ReservedWallTime != 3600 {
		t.Fatalf("journal lost cumulative reservations: %+v", loaded.Report)
	}
	lease, err = government.AcquireLease(filepath.Join(dir, "queue.lease"))
	if err != nil {
		t.Fatal(err)
	}
	control := &queueControl{dir: dir, state: loaded, lease: lease}
	if err := control.ReserveActor("later-run", 1, "execute", opts.Runtime.Executor); err == nil {
		t.Fatal("resumed actor budget reset")
	}
	if err := control.ReserveRepair("later-run", "inventory/Area/reservations", 1); err == nil {
		t.Fatal("resumed repair budget reset")
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}

	resumed, err := ResumeQueue(context.Background(), QueueOptions{Repo: opts.Repo, BacklogPath: backlogPath, QueueDirectory: dir})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.ActorStarts != 1 || resumed.Repairs != 1 || resumed.ReservedWallTime != 3600 || resumed.InFlightActors != 1 || resumed.PeakParallelism != 1 {
		t.Fatalf("resume reset durable budget: %+v", resumed)
	}
	if resumed.Jobs[0].State != "incomplete" || resumed.Jobs[1].State != "incomplete" || resumed.Jobs[2].State != "blocked" {
		t.Fatalf("interruption or cumulative wall cap was not preserved: %+v", resumed.Jobs)
	}
	if !resumed.Usage.InputTokens.Unknown || !resumed.Usage.ObservedActorWallTimeMillis.Unknown {
		t.Fatalf("unknown interrupted provider/host usage was represented as known: %+v", resumed.Usage)
	}
	repeated, err := ResumeQueue(context.Background(), QueueOptions{Repo: opts.Repo, BacklogPath: backlogPath, QueueDirectory: dir})
	if err != nil {
		t.Fatal(err)
	}
	if repeated.JournalSequence != resumed.JournalSequence || repeated.ActorStarts != resumed.ActorStarts || repeated.InFlightActors != 1 || repeated.Usage != resumed.Usage {
		t.Fatalf("terminal interrupted queue did not remain stable on repeated resume: first=%+v second=%+v", resumed, repeated)
	}
}

func setRuntimeModelOptions(runtime *Runtime) {
	set := func(spec *RunnerSpec) {
		if len(spec.ModelOptions) == 0 {
			spec.ModelOptions = []byte(`{}`)
		}
	}
	set(&runtime.Executor)
	set(&runtime.Verifier)
	for i := range runtime.Ressorts {
		set(&runtime.Ressorts[i].Runner)
	}
}

func TestQueueBacklogRejectsAmbiguousAndMalformedJSON(t *testing.T) {
	opts := fixtureOptions(t)
	root := filepath.Dir(opts.Repo)
	cases := map[string][]byte{
		"duplicate":    []byte(`{"apiVersion":"markitect.government-queue/v1alpha1","apiVersion":"markitect.government-queue/v1alpha1"}`),
		"case-alias":   []byte(`{"apiVersion":"markitect.government-queue/v1alpha1","APIVERSION":"markitect.government-queue/v1alpha1"}`),
		"null-jobs":    []byte(`{"apiVersion":"markitect.government-queue/v1alpha1","stateDirectory":"C:/unused","limits":{},"jobs":null}`),
		"invalid-utf8": {0xff, '{', '}'},
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "bad-"+name+".json")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := readQueueBacklog(opts.Repo, path); err == nil {
				t.Fatal("malformed backlog was accepted")
			}
		})
	}
}
