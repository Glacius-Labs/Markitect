package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	runport "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"go.yaml.in/yaml/v3"
)

// These tests execute the actual controlled Core over real Git/SQLite with
// in-memory workspaces and scripted sessions. No runtime, engine, provider or
// credential resolver is invoked, and no provider acceptance is claimed.
type bootstrapExecutionSessions struct {
	t              *testing.T
	records, steps int
	askOwner       bool
	answered       bool
}

func (f *bootstrapExecutionSessions) meta() runport.RecordMeta {
	f.records++
	return runport.RecordMeta{ID: fmt.Sprintf("bootstrap-fake-%d", f.records), ObservedAt: time.Now().UTC()}
}

func (f *bootstrapExecutionSessions) wire(data string) runport.WirePayload {
	p, err := runport.NewPayload("application/json", "utf-8", []byte(data))
	if err != nil {
		f.t.Fatal(err)
	}
	return runport.WirePayload{Identity: runport.WireIdentity{ContentID: p.Ref().ContentID, Size: p.Ref().Size}, Journal: p}
}

func (f *bootstrapExecutionSessions) send(ctx context.Context, channel runport.SessionIO, step runport.StepID, call runport.ProviderCallID, data string) error {
	m := f.meta()
	if _, err := channel.PrepareSend(ctx, runport.TransportPreparation{RecordMeta: m, Message: runport.MessageID(m.ID), Step: step, Call: call, Wire: f.wire(data)}); err != nil {
		return err
	}
	_, err := channel.ObserveSend(ctx, runport.TransportObservation{RecordMeta: f.meta(), Message: runport.MessageID(m.ID), Outcome: runport.Sent, Reason: "fake observed write"})
	return err
}

func (f *bootstrapExecutionSessions) Open(ctx context.Context, a runport.SessionAdmission, channel runport.SessionIO) (runport.SessionObservation, error) {
	if err := f.send(ctx, channel, "", "", `{"control":"initialize"}`); err != nil {
		return runport.SessionObservation{}, err
	}
	if a.Plan.Resume != "" {
		_, err := channel.Bind(ctx, runport.SessionObservation{RecordMeta: f.meta(), ProviderSession: a.Plan.Resume, Wire: f.wire(`{"resumed":true}`), Reason: "fake resumed session"})
		return runport.SessionObservation{ProviderSession: a.Plan.Resume}, err
	}
	_, err := channel.Ready(ctx, runport.SessionReadiness{RecordMeta: f.meta(), Wire: f.wire(`{"control":"ready"}`), Reason: "fake readiness"})
	return runport.SessionObservation{Ready: true}, err
}

func (f *bootstrapExecutionSessions) call(ctx context.Context, channel runport.SessionIO, step runport.StepID, id, tool, args string) error {
	call := runport.ProviderCallID(id)
	result, err := channel.Call(ctx, runport.ToolCall{RecordMeta: f.meta(), Step: step, Call: call, Tool: tool, Input: f.wire(fmt.Sprintf(`{"id":1,"name":%q,"arguments":%s}`, tool, args))})
	if err != nil {
		return err
	}
	if result.Outcome.Status != runport.ToolOK {
		return fmt.Errorf("fake %s refused: %s", tool, result.Outcome.Reason)
	}
	return f.send(ctx, channel, step, call, string(result.Output.Bytes()))
}

func (f *bootstrapExecutionSessions) Step(ctx context.Context, a runport.StepAdmission, channel runport.SessionIO) (runport.StepObservation, error) {
	f.steps++
	var packet runport.StepPacket
	if err := json.Unmarshal(a.Input.Bytes(), &packet); err != nil {
		return runport.StepObservation{}, err
	}
	step := a.Plan.ID
	if err := f.send(ctx, channel, step, "", `{"user":"fake turn"}`); err != nil {
		return runport.StepObservation{}, err
	}
	if a.Session.Resume == "" {
		if _, err := channel.Bind(ctx, runport.SessionObservation{RecordMeta: f.meta(), ProviderSession: runport.ProviderSessionID("fake-" + string(a.Session.ID)), Wire: f.wire(`{"thread":"new"}`), Reason: "fake binding"}); err != nil {
			return runport.StepObservation{}, err
		}
	}
	if _, err := channel.ObserveStep(ctx, runport.StepObservation{RecordMeta: f.meta(), Step: step, ProviderStep: "turn", Wire: f.wire(`{"turn":"started"}`), Reason: "fake start"}); err != nil {
		return runport.StepObservation{}, err
	}
	asked := f.askOwner && packet.Phase == runport.PhaseWork
	if asked {
		f.askOwner = false
		if err := f.call(ctx, channel, step, "ask-"+string(step), runport.ToolSend, `{"kind":"question","to":"owner","body":"May the fake no-change realization continue?"}`); err != nil {
			return runport.StepObservation{}, err
		}
	}
	if packet.Phase == runport.PhaseWork && bytes.Contains(packet.Context.Bytes, []byte("Continue the fake no-change realization.")) {
		f.answered = true
	}
	var tool, args string
	switch packet.Phase {
	case runport.PhaseWork:
		tool, args = runport.ToolSubmit, `{"purpose":"local","scope":"complete","summary":"No change.","assessments":[],"dependencies":[]}`
	case runport.PhaseIntegrate:
		tool, args = runport.ToolSubmit, `{"purpose":"integration","scope":"complete","summary":"No change.","assessments":[],"dependencies":[]}`
	case runport.PhaseReviewerExpectations:
		tool, args = runport.ToolRecordExpectations, `{"cases":[{"name":"area","expectation":"Unchanged.","discriminates":"Any change."}]}`
	case runport.PhaseReviewerCandidate:
		tool, args = runport.ToolSubmitReview, `{"verdict":"pass","summary":"Fake review passed.","evidence":[],"cases":[{"name":"area","assessment":"evidenced","grounds":"Unchanged."}],"findings":[],"resolutions":[]}`
	default:
		return runport.StepObservation{}, fmt.Errorf("unexpected %s step", packet.Phase)
	}
	if asked {
		tool, args = runport.ToolWait, fmt.Sprintf(`{"on":[{"kind":"owner","ref":%q}],"reason":"Await the explicit owner answer."}`, "question-"+string(step)+"-1")
	}
	if err := f.call(ctx, channel, step, "call-"+string(step), tool, args); err != nil {
		return runport.StepObservation{}, err
	}
	observed := runport.StepObservation{RecordMeta: f.meta(), Step: step, ProviderStep: "turn", Wire: f.wire(`{"turn":"done"}`), Terminal: true, Reason: "fake terminal turn"}
	_, err := channel.ObserveStep(ctx, observed)
	return observed, err
}

func (f *bootstrapExecutionSessions) Inspect(context.Context, runport.InspectionRequest) (runport.StepObservation, error) {
	return runport.StepObservation{}, errors.New("fake session was not inspected")
}
func (f *bootstrapExecutionSessions) Stop(ctx context.Context, _ runport.StopRequest, channel runport.SessionIO) (runport.StopObservation, error) {
	observed := runport.StopObservation{RecordMeta: f.meta(), Terminated: true, Reason: "fake observed termination"}
	_, err := channel.ObserveStop(ctx, observed)
	return observed, err
}

type bootstrapExecutionWorkspaces map[runport.WorkspaceID]snapshot.Tree

func (w bootstrapExecutionWorkspaces) Materialize(_ context.Context, request runport.WorkspaceSpec) (runport.WorkspaceObservation, error) {
	w[request.ID] = request.Tree
	return runport.WorkspaceObservation{ID: request.ID, Volume: "fake-" + string(request.ID), Tree: request.Tree.Digest()}, nil
}
func (w bootstrapExecutionWorkspaces) Harvest(_ context.Context, request runport.HarvestRequest) (runport.WorkspaceHarvest, error) {
	return runport.WorkspaceHarvest{ID: request.ID, Tree: w[request.ID]}, nil
}
func (w bootstrapExecutionWorkspaces) Peek(_ context.Context, request runport.HarvestRequest) (runport.WorkspaceHarvest, error) {
	return runport.WorkspaceHarvest{ID: request.ID, Tree: w[request.ID]}, nil
}
func (w bootstrapExecutionWorkspaces) Close(_ context.Context, id runport.WorkspaceID) (runport.WorkspaceObservation, error) {
	delete(w, id)
	return runport.WorkspaceObservation{ID: id, Closed: true}, nil
}

type bootstrapExecutionChecks struct{}

func (bootstrapExecutionChecks) Run(_ context.Context, request runport.CheckRequest) (runport.CheckReceipt, error) {
	return runport.CheckReceipt{ID: request.ID, ImageID: request.ImageID, Tree: request.Tree.Digest(), Started: true, Exited: true}, nil
}

func TestExecutionServiceUsesControlledExecuteAndAnswerOnRealGitJournal(t *testing.T) {
	f := gitControlFixtureForRepository(t, executionRepositoryFixture(t))
	// Make this source fixture's explicit project ceilings large enough for
	// every Manager's real Core steps. No runtime resource is allocated.
	executionPath := filepath.Join(f.repo.Dir, ".markitect", "execution.yaml")
	raw, err := os.ReadFile(executionPath)
	if err != nil {
		t.Fatal(err)
	}
	var execution map[string]any
	if err := yaml.Unmarshal(raw, &execution); err != nil {
		t.Fatal(err)
	}
	limits := execution["limits"].(map[string]any)
	limits["maxSessionStarts"], limits["maxSteps"] = 10000, 10000
	raw, err = yaml.Marshal(execution)
	if err != nil {
		t.Fatal(err)
	}
	f.repo.Write(".markitect/execution.yaml", string(raw))
	f.repo.Commit("explicit fake execution ceilings")
	document := runConfigFixture(t)
	document["repositoryIdentity"], document["productBuild"], document["ownerRequest"] = f.facts.RepositoryIdentity, "frozen-bootstrap-execution-test", "Apply this source fixture through fake runtime ports only."
	document["stateDirectory"] = f.options.StateDirectory
	document["allocation"].(map[string]any)["executionTime"] = int64(7 * time.Hour)
	document["allocation"].(map[string]any)["runs"] = f.facts.Allocation.Runs
	configPath := filepath.Join(testkit.TempDir(t), "run.json")
	if err := os.WriteFile(configPath, configBytes(t, document), 0600); err != nil {
		t.Fatal(err)
	}
	sessions := &bootstrapExecutionSessions{t: t, askOwner: true}
	service := newExecution(apply.NewPreparationService(apply.NewModelService(f.source, f.source)), f.source, fakeControlMetadata{}, nil)
	service.ports = func(cfg runConfig, controlled *apply.ControlledRun, _ apply.CapturedSettingsResult) (apply.ExecutionPorts, error) {
		profile, err := runport.NewPayload("application/json", "utf-8", []byte(`{"fake":true}`))
		if err != nil {
			return apply.ExecutionPorts{}, err
		}
		return apply.ExecutionPorts{Sessions: sessions, Workspaces: bootstrapExecutionWorkspaces{}, Checks: bootstrapExecutionChecks{}, Publisher: f.source, Clock: time.Now, Limits: cfg.EngineLimits,
			Profile: func(request runport.ProfileRequest) (runport.SessionProfile, error) {
				if _, err := controlled.RoleProfile(request.Manager, request.Role); err != nil {
					return runport.SessionProfile{}, err
				}
				return runport.SessionProfile{ImageID: cfg.WorkspaceHelper.Image, CredentialRef: "fake-not-resolved", Profile: profile, MaxWireBytes: 1 << 20}, nil
			}, CheckRequest: func(apply.CheckSelection) (runport.CheckRequest, error) {
				return runport.CheckRequest{}, errors.New("fixture declares no checks")
			}}, nil
	}
	request := cli.ExecutionRequest{Selection: f.prepared.Request().Selection, RunConfig: configPath}
	before, err := f.source.Capture(context.Background(), apply.CaptureRequest{Selection: request.Selection})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Execute(context.Background(), request)
	if err != nil || first.Status != model.NotCompleted || first.Execution == nil || first.Execution.Run.State != runport.Waiting {
		t.Fatalf("actual controlled run did not wait for owner: status=%s execution=%+v err=%v", first.Status, first.Execution, err)
	}
	questions, ok := first.OpenQuestions.([]runport.Issue)
	if !ok || len(questions) != 1 || sessions.steps == 0 {
		t.Fatal("actual owner question or admitted fake steps missing")
	}
	request.Question, request.Answer = questions[0].ID, "Continue the fake no-change realization."
	second, err := service.Execute(context.Background(), request)
	if err != nil || second.Status != model.Passed || second.Execution == nil || second.Execution.Run.Outcome != runport.NoChange || second.Execution.Publication == nil || second.Execution.Publication.Outcome != apply.PublicationNoChange || !sessions.answered {
		t.Fatalf("actual Answer/Execute did not finish no-change publication: status=%s execution=%+v err=%v", second.Status, second.Execution, err)
	}
	after, err := f.source.Capture(context.Background(), apply.CaptureRequest{Selection: request.Selection})
	if err != nil || before.Head != after.Head || before.Raw.Census.Digest() != after.Raw.Census.Digest() {
		t.Fatalf("fake no-change run altered owner tree or commit: %v", err)
	}
}

// One actual canonical Core Package keeps this outer-composition test bounded.
// The ordinary root and pass-through ancestor remain in the real topology.
func executionRepositoryFixture(t *testing.T) *testkit.Repo {
	t.Helper()
	repo := testkit.NewRepo(t)
	err := filepath.WalkDir("../../.markitect/model/markitect/core", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		repo.Write(".markitect/model/markitect/core/"+entry.Name(), string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"model.yaml", "model-lock.yaml", "execution.yaml"} {
		data, err := os.ReadFile("../../.markitect/" + name)
		if err != nil {
			t.Fatal(err)
		}
		repo.Write(".markitect/"+name, string(data))
	}
	repo.Write("docs/target.md", "Unchanged fake realization.\n")
	repo.Write(".markitect/registry/markitect/core/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts:\n  - path: docs/target.md\n    form: doc\n    realizes: [markitect/core/Kind]\n    why: Exact artifact used only by the fake outer composition fixture.\n")
	repo.Commit("small committed canonical execution fixture")
	return repo
}

func TestExecutionRefusesPreparedLegacyProcessCheck(t *testing.T) {
	f := gitControlFixtureForRepository(t, executionRepositoryFixture(t))
	executionPath := filepath.Join(f.repo.Dir, ".markitect", "execution.yaml")
	raw, err := os.ReadFile(executionPath)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "checks: []", "checks:\n  - id: legacy\n    profile: legacy\n    directory: .\n    args: []\n    requiredBy: markitect/core/Kind\n    timeout: 1s", 1)
	f.repo.Write(".markitect/execution.yaml", text)
	f.repo.Commit("explicit historical metadata-only check")
	data := []byte("Fake credential-free file; deliberately not an executable image.\n")
	executable := filepath.Join(testkit.TempDir(t), "not-executed")
	if err := os.WriteFile(executable, data, 0600); err != nil {
		t.Fatal(err)
	}
	selection := f.prepared.Request().Selection
	host := selection.HostProfiles[0]
	document := fmt.Sprintf("apiVersion: markitect.host/v1\nagentProfiles:\n  markitect-agent:\n    codexAppServer: {profileVersion: 1}\ncheckProfiles:\n  legacy:\n    process:\n      profileVersion: 1\n      executable: %q\n      sha256: '%x'\n", executable, sha256.Sum256(data))
	if err := os.WriteFile(filepath.Join(host.Boundary, host.Path.String()), []byte(document), 0600); err != nil {
		t.Fatal(err)
	}
	prepared, err := apply.NewPreparationService(apply.NewModelService(f.source, f.source)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{}, fakeControlMetadata{}, checks.MetadataPreparer{})
	attempt, ok := prepared.Prepared()
	if err != nil || !ok || prepared.Status() != model.Passed {
		t.Fatalf("historical metadata preparation failed: %s %v", prepared.Status(), err)
	}
	selected, ok := attempt.Settings().Checks()
	if !ok || len(selected) != 1 || selected[0].Profile().Input().Section() != "process" {
		t.Fatal("actual process metadata selection missing")
	}
	if _, err := executionCheck(selected[0], runConfig{}); err == nil || err.Error() != "execution requires a prepared container check using the explicit run engine" {
		t.Fatalf("metadata-only process profile became executable: %v", err)
	}
}

func TestExecutionReportRetainsDefaultsAndContextDiagnostics(t *testing.T) {
	f := gitControlFixtureForRepository(t, executionRepositoryFixture(t))
	raw, err := os.ReadFile(filepath.Join(f.repo.Dir, ".markitect", "execution.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var execution map[string]any
	if err := yaml.Unmarshal(raw, &execution); err != nil {
		t.Fatal(err)
	}
	// The omitted project value inherits product rejectionLimit=3, which
	// exceeds this explicit project ceiling. The precise refusal belongs to
	// the Defaults producer; Settings points the caller to those diagnostics.
	delete(execution["defaults"].(map[string]any), "rejectionLimit")
	execution["limits"].(map[string]any)["maxRejectionLimit"] = 2
	raw, err = yaml.Marshal(execution)
	if err != nil {
		t.Fatal(err)
	}
	f.repo.Write(".markitect/execution.yaml", string(raw))
	f.repo.Commit("explicit invalid inherited rejection limit")
	prepared, err := apply.NewPreparationService(apply.NewModelService(f.source, f.source)).CaptureSettings(context.Background(), apply.CaptureRequest{Selection: f.prepared.Request().Selection}, model.Limits{}, fakeControlMetadata{}, nil)
	if err != nil || prepared.Status() != model.Failed {
		t.Fatalf("actual defaults refusal missing: %s %v", prepared.Status(), err)
	}
	report := capturedExecutionReport(prepared, "explicit-source-test-assertion").(map[string]any)
	entries := report["attempts"].([]any)
	if len(entries) != len(prepared.Attempts()) {
		t.Fatal("actual capture attempts were lost")
	}
	for i, attempt := range prepared.Attempts() {
		entry := entries[i].(map[string]any)
		if len(attempt.Defaults().Diagnostics()) == 0 || entry["defaultsStatus"] != attempt.Defaults().Status() || !reflect.DeepEqual(entry["defaultsDiagnostics"], attempt.Defaults().Diagnostics()) || entry["contextStatus"] != attempt.Structural().Contexts().Status() || !reflect.DeepEqual(entry["contextDiagnostics"], attempt.Structural().Contexts().Diagnostics()) {
			t.Fatal("defaults or context status/diagnostics omitted from actual report")
		}
	}
}
