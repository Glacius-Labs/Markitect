package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	runport "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func writeRunConfig(t *testing.T, document map[string]any) string {
	t.Helper()
	path := filepath.Join(testkit.TempDir(t), "run.json")
	if err := os.WriteFile(path, configBytes(t, document), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExecuteRefusesAnInvalidRunConfigurationBeforeAnyEffect(t *testing.T) {
	document := runConfigFixture(t)
	document["allocation"] = map[string]any{"runs": 1}
	service := newExecution(apply.NewPreparationService(apply.NewModelService(nil, nil)), &gitrepo.Repository{}, nil, nil)
	report, err := service.Execute(context.Background(), cli.ApplyRequest{RunConfig: writeRunConfig(t, document)})
	if !errors.Is(err, errRunConfig) || report.Status != model.Failed || report.EvidenceDirectory != "" {
		t.Fatalf("%v %+v", err, report)
	}
	if _, err := os.Stat(document["stateDirectory"].(string)); !os.IsNotExist(err) {
		t.Fatal("an invalid configuration created the state directory")
	}
}

func TestAssignmentApplyRefusesLegacyWorkDeadlineBeforePreparation(t *testing.T) {
	document := runConfigFixture(t)
	document["workDeadline"] = "2029-01-01T00:00:00Z"
	document["deadline"] = "2030-01-01T00:00:00Z"
	service := newExecution(apply.NewPreparationService(apply.NewModelService(nil, nil)), &gitrepo.Repository{}, nil, nil)
	report, err := service.Execute(context.Background(), cli.ApplyRequest{Selection: apply.InputSelection{Repository: testkit.TempDir(t)}, RunConfig: writeRunConfig(t, document), Build: document["productBuild"].(string)})
	if err == nil || !strings.Contains(err.Error(), "workDeadline") || report.Preparation != nil || report.EvidenceDirectory != "" || report.Status != model.NotCompleted {
		t.Fatalf("legacy work deadline reached assignment preparation: %v %+v", err, report)
	}
}

type controlledGitLifetime struct {
	started, closed, worked int
	startErr, closeErr      error
	ids                     []runport.WorkspaceID
}

func (g *controlledGitLifetime) StartGit(_ context.Context, ids []runport.WorkspaceID) error {
	g.started++
	g.ids = append([]runport.WorkspaceID(nil), ids...)
	return g.startErr
}
func (g *controlledGitLifetime) CloseGit(context.Context) error {
	g.closed++
	return g.closeErr
}

func TestAssignmentGitLifetimeClosesOnEveryExitAndRetainsErrors(t *testing.T) {
	startErr, workErr, closeErr := errors.New("start"), errors.New("work"), errors.New("close")
	for _, tc := range []struct {
		name               string
		start, work, close error
		wantWork           int
	}{
		{name: "success", wantWork: 1},
		{name: "failed start", start: startErr},
		{name: "failed realization or publication", work: workErr, wantWork: 1},
		{name: "failed cleanup", close: closeErr, wantWork: 1},
		{name: "both failures", work: workErr, close: closeErr, wantWork: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			git := &controlledGitLifetime{startErr: tc.start, closeErr: tc.close}
			ids := []runport.WorkspaceID{"run-work-0", "run-verify-1"}
			err := withPersistentGit(context.Background(), git, ids, func() error { git.worked++; return tc.work })
			if git.started != 1 || git.closed != 1 || git.worked != tc.wantWork || !reflect.DeepEqual(git.ids, ids) ||
				(tc.start != nil && !errors.Is(err, tc.start)) || (tc.work != nil && !errors.Is(err, tc.work)) ||
				(tc.close != nil && !errors.Is(err, tc.close)) || (tc.start == nil && tc.work == nil && tc.close == nil && err != nil) {
				t.Fatalf("git lifecycle: %+v, error: %v", git, err)
			}
		})
	}
}

func TestExecuteRefusesAHeldStateDirectoryBeforePreparation(t *testing.T) {
	document := runConfigFixture(t)
	state := document["stateDirectory"].(string)
	held, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	previous := startLockWait
	startLockWait = 20 * time.Millisecond
	t.Cleanup(func() { startLockWait = previous })
	// Preparation would panic on this empty composition; the lock comes first.
	service := newExecution(apply.NewPreparationService(apply.NewModelService(nil, nil)), &gitrepo.Repository{}, nil, nil)
	report, err := service.Execute(context.Background(), cli.ApplyRequest{Selection: apply.InputSelection{Repository: testkit.TempDir(t)}, RunConfig: writeRunConfig(t, document)})
	if !errors.Is(err, errStateLocked) || report.Status != model.NotCompleted || report.Preparation != nil || report.FormatVersion != 1 || report.Operation != "apply" {
		t.Fatalf("%v %+v", err, report)
	}
	if entries, _ := os.ReadDir(filepath.Join(state, "runs")); len(entries) != 0 {
		t.Fatalf("a refused Apply left evidence: %v", entries)
	}
}

func TestExecuteRefusesAStateDirectoryThatOverlapsTheRepository(t *testing.T) {
	root := testkit.TempDir(t)
	repository := filepath.Join(root, "repo")
	for name, state := range map[string]string{"inside": filepath.Join(repository, ".state"), "equal": repository, "containing": root} {
		document := runConfigFixture(t)
		document["stateDirectory"] = state
		service := newExecution(apply.NewPreparationService(apply.NewModelService(nil, nil)), &gitrepo.Repository{}, nil, nil)
		report, err := service.Execute(context.Background(), cli.ApplyRequest{Selection: apply.InputSelection{Repository: repository}, RunConfig: writeRunConfig(t, document)})
		if err == nil || !strings.Contains(err.Error(), "overlap") || report.Status != model.Failed {
			t.Errorf("%s: %v %+v", name, err, report)
		}
		if _, err := os.Stat(filepath.Join(state, lockFileName)); !os.IsNotExist(err) {
			t.Errorf("%s: an overlapping state directory was written to", name)
		}
	}
	for _, state := range []string{filepath.Join(root, "state"), filepath.Join(root, "repo-state")} {
		if err := checkStateOutsideRepository(state, repository); err != nil {
			t.Fatalf("%s was refused: %v", state, err)
		}
	}
}

func TestExtractRefusesBeforeAnyEffectLikeApply(t *testing.T) {
	// An invalid run configuration is a definite refusal and creates nothing.
	document := runConfigFixture(t)
	document["allocation"] = map[string]any{"runs": 1}
	service := newExecution(apply.NewPreparationService(apply.NewModelService(nil, nil)), &gitrepo.Repository{}, nil, nil)
	report, err := service.Extract(context.Background(), cli.ExtractRequest{RunConfig: writeRunConfig(t, document)})
	if !errors.Is(err, errRunConfig) || report.Status != model.Failed || report.Operation != "extract" || report.EvidenceDirectory != "" || report.Deviations == nil {
		t.Fatalf("%v %+v", err, report)
	}
	if _, err := os.Stat(document["stateDirectory"].(string)); !os.IsNotExist(err) {
		t.Fatal("an invalid configuration created the state directory")
	}
	// A state directory that overlaps the repository is refused as well.
	root := testkit.TempDir(t)
	repository := filepath.Join(root, "repo")
	document = runConfigFixture(t)
	document["stateDirectory"] = filepath.Join(repository, ".state")
	report, err = service.Extract(context.Background(), cli.ExtractRequest{Selection: apply.InputSelection{Repository: repository}, RunConfig: writeRunConfig(t, document)})
	if err == nil || !strings.Contains(err.Error(), "overlap") || report.Status != model.Failed {
		t.Fatalf("%v %+v", err, report)
	}
	// A held state directory comes before preparation and leaves no evidence.
	document = runConfigFixture(t)
	state := document["stateDirectory"].(string)
	held, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	previous := startLockWait
	startLockWait = 20 * time.Millisecond
	t.Cleanup(func() { startLockWait = previous })
	report, err = service.Extract(context.Background(), cli.ExtractRequest{Selection: apply.InputSelection{Repository: testkit.TempDir(t)}, RunConfig: writeRunConfig(t, document)})
	if !errors.Is(err, errStateLocked) || report.Status != model.NotCompleted || report.Preparation != nil || report.FormatVersion != 1 {
		t.Fatalf("%v %+v", err, report)
	}
	if entries, _ := os.ReadDir(filepath.Join(state, "runs")); len(entries) != 0 {
		t.Fatalf("a refused Extract left evidence: %v", entries)
	}
}

// The cockpit tells the run that holds the state directory by current-run;
// nothing else is written to the state directory before the run's evidence.
func TestOpenRecordsTheRunIdentityAndNoQuestionOrAnswerFiles(t *testing.T) {
	document := runConfigFixture(t)
	state := document["stateDirectory"].(string)
	service := newExecution(apply.NewPreparationService(apply.NewModelService(nil, nil)), &gitrepo.Repository{}, nil, nil)
	session, failed, err := service.open(context.Background(), testkit.TempDir(t), writeRunConfig(t, document))
	if err != nil || failed {
		t.Fatal(failed, err)
	}
	defer session.lock.Release()
	data, err := os.ReadFile(filepath.Join(state, "current-run"))
	if err != nil || strings.TrimSpace(string(data)) != session.runID || !stateName.MatchString(session.runID) {
		t.Fatalf("current-run = %q, %v", data, err)
	}
	for _, name := range []string{"questions", "answers"} {
		if _, err := os.Stat(filepath.Join(state, name)); !os.IsNotExist(err) {
			t.Fatalf("a closed Run created %s: %v", name, err)
		}
	}
}

func TestRunIdentitiesAreSafeAndDistinct(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 30, 45, 0, time.FixedZone("CEST", 2*3600))
	first, err := newRunID(now)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := newRunID(now)
	if !regexp.MustCompile(`^20261019T103045Z-[0-9a-f]{8}$`).MatchString(first) || !stateName.MatchString(first) || first == second {
		t.Fatalf("%s %s", first, second)
	}
}

func TestApplyReportRetainedAsEvidenceMatchesThePrintedShape(t *testing.T) {
	e, err := newFileEvidence(testkit.TempDir(t), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	started := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	report := cli.ApplyReport{FormatVersion: 1, Operation: "apply", Status: model.NotCompleted, EvidenceDirectory: e.Dir(), Error: "x"}
	report.Run = cli.NewRunReport("run-1", runport.Outcome{Status: runport.Incomplete, Questions: []runport.Question{{ID: "q1", Text: "Which currency?", Asked: started}}})
	if err := e.WriteReport(cli.ReportValue(report)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(e.Dir(), "report.json"))
	if err != nil || !strings.Contains(string(data), `"asked": "2026-10-19T12:00:00Z"`) || !strings.Contains(string(data), `"operation": "apply"`) {
		t.Fatalf("%s %v", data, err)
	}
	var decoded map[string]any
	if json.Unmarshal(data, &decoded) != nil {
		t.Fatal("report is not JSON")
	}
	if decoded["outcome"] != "incomplete" {
		t.Fatalf("retained report lacks the command outcome: %s", data)
	}
}

// Persist before returning to the CLI, as Execute does. Comparing only two
// calls to ReportValue would miss enrichment performed later by the command.
func TestApplyFileAndStdoutKeepTheSameOutcomeAndEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		runStatus  runport.Status
		reason     string
		receipt    *apply.PublicationReceipt
		err        error
		code       int
	}{
		{"applied", "success", runport.Verified, "", &apply.PublicationReceipt{Outcome: apply.PublicationApplied}, nil, 0},
		{"no_change", "success", runport.Verified, "", &apply.PublicationReceipt{Outcome: apply.PublicationNoChange}, nil, 0},
		{"assessment_failed", "failed", runport.Failed, "the model assessment confirmed a material model problem: missing public policy", nil, errors.New("model assessment rejected"), 1},
		{"technical_incomplete", "incomplete", runport.Incomplete, "capture is uncertain", nil, errors.New("workspace remains mounted"), 3},
		{"publication_unknown", "incomplete", runport.Verified, "", &apply.PublicationReceipt{Outcome: apply.PublicationUnknown, Reason: "read-back uncertain"}, nil, 3},
		{"publication_failed", "failed", runport.Verified, "", &apply.PublicationReceipt{Outcome: apply.PublicationFailed, Reason: "publication refused"}, nil, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, err := newFileEvidence(testkit.TempDir(t), "run-1")
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			outcome := runport.Outcome{Status: tc.runStatus, Reason: tc.reason}
			report := cli.ApplyReport{FormatVersion: 1, Operation: "apply", Status: cli.ApplyStatus(outcome, tc.receipt, tc.err), Run: cli.NewRunReport("run-1", outcome), Publication: tc.receipt, EvidenceDirectory: e.Dir()}
			if tc.err != nil {
				report.Error = tc.err.Error()
			}
			command := cli.New(nil, nil, nil, nil, nil, func(context.Context, cli.ApplyRequest) (cli.ApplyReport, error) {
				if err := e.WriteReport(cli.ReportValue(report)); err != nil {
					t.Fatal(err)
				}
				return report, tc.err
			})
			var stdout, stderr bytes.Buffer
			if code := command.Run(context.Background(), []string{"apply", "--runtime-profile", filepath.Join(e.Dir(), "runtime.yaml"), "--run-config", filepath.Join(e.Dir(), "run.json")}, &stdout, &stderr); code != tc.code {
				t.Fatalf("exit %d, want %d: %s", code, tc.code, stderr.String())
			}
			data, err := os.ReadFile(filepath.Join(e.Dir(), "report.json"))
			if err != nil {
				t.Fatal(err)
			}
			var retained, printed map[string]any
			if err := json.Unmarshal(data, &retained); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(stdout.Bytes(), &printed); err != nil {
				t.Fatal(err)
			}
			if retained["outcome"] != tc.want || !reflect.DeepEqual(retained, printed) {
				t.Fatalf("want outcome %q and matching reports\nretained: %s\nprinted: %s", tc.want, data, stdout.String())
			}
			gotRun := retained["run"].(map[string]any)
			if gotRun["status"] != string(tc.runStatus) || tc.reason != "" && gotRun["reason"] != tc.reason {
				t.Fatalf("run conclusion changed: %v", gotRun)
			}
			if tc.receipt != nil && retained["publication"].(map[string]any)["Outcome"] != string(tc.receipt.Outcome) {
				t.Fatalf("publication outcome changed: %v", retained["publication"])
			}
		})
	}
}

func TestApplyReportIncludesFinalizationFailures(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		status                         model.Status
		closeErr, releaseErr, writeErr bool
	}{
		{"clean", model.Passed, false, false, false},
		{"close", model.Passed, true, false, false},
		{"release", model.Passed, false, true, false},
		{"close_and_release", model.Passed, true, true, false},
		{"write", model.Passed, false, false, true},
		{"failed", model.Failed, false, false, false},
		{"failed_with_close", model.Failed, true, false, false},
		{"failed_with_write", model.Failed, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, err := newFileEvidence(testkit.TempDir(t), "run-1")
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			path := filepath.Join(e.Dir(), "report.json")
			if tc.closeErr {
				if err := e.events.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if tc.writeErr {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			runStatus := runport.Verified
			receipt := &apply.PublicationReceipt{Outcome: apply.PublicationNoChange}
			var runErr error
			if tc.status == model.Failed {
				runStatus, receipt, runErr = runport.Failed, nil, errors.New("model assessment rejected")
			}
			runReport := cli.NewRunReport("run-1", runport.Outcome{Status: runStatus, Reason: "original run reason"})
			report := cli.ApplyReport{Status: tc.status, Run: runReport, Publication: receipt}
			releaseErr := errors.New("release failed")
			releases := 0
			err = finishApplyReport(&report, runErr, e, func() error {
				releases++
				if e.events != nil {
					t.Fatal("lock released before closing evidence")
				}
				if _, err := os.Stat(path); !tc.writeErr && !os.IsNotExist(err) {
					t.Fatal("report written before lock release")
				}
				if tc.releaseErr {
					return releaseErr
				}
				return nil
			})
			wantStatus := tc.status
			if (tc.closeErr || tc.releaseErr || tc.writeErr) && wantStatus != model.Failed {
				wantStatus = model.NotCompleted
			}
			if releases != 1 || report.Status != wantStatus || (err != nil) != (runErr != nil || tc.closeErr || tc.releaseErr || tc.writeErr) || (report.Error != "") != (err != nil) {
				t.Fatalf("releases %d, report %+v, error %v", releases, report, err)
			}
			if tc.closeErr && !errors.Is(err, os.ErrClosed) || tc.releaseErr && !errors.Is(err, releaseErr) {
				t.Fatalf("lost finalization error: %v", err)
			}
			if report.Run.Status != runStatus || report.Run.Reason != "original run reason" || report.Publication != receipt || receipt != nil && receipt.Outcome != apply.PublicationNoChange {
				t.Fatalf("finalization changed execution evidence: %+v", report)
			}
			if !tc.writeErr {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var retained map[string]any
				if err := json.Unmarshal(data, &retained); err != nil {
					t.Fatal(err)
				}
				if retained["status"] != string(wantStatus) || retained["outcome"] != cli.RunOutcome(wantStatus) || report.Error != "" && retained["error"] != report.Error {
					t.Fatalf("retained report missed finalization: %s", data)
				}
			}
		})
	}
}

func TestApplyFinalizationReleasesWithoutEvidence(t *testing.T) {
	report := cli.ApplyReport{Status: model.Failed}
	sourceErr, releaseErr := errors.New("preparation rejected"), errors.New("release failed")
	releases := 0
	err := finishApplyReport(&report, sourceErr, nil, func() error {
		releases++
		return releaseErr
	})
	if releases != 1 || !errors.Is(err, sourceErr) || !errors.Is(err, releaseErr) || report.Status != model.Failed || report.Error != err.Error() {
		t.Fatalf("releases %d, report %+v, error %v", releases, report, err)
	}
}

func TestAgentProfilesNeedAnExactCredentialMappingOfTheirProvider(t *testing.T) {
	cfg := runConfig{Credentials: map[string]credentialSource{
		"codex-ref":  {Provider: "codex-app-server", Mode: "api-key", Path: "/abs/codex"},
		"claude-ref": {Provider: "claude-code", Mode: "dedicated-runtime-token", Path: "/abs/claude"},
	}}
	if credentialMapped(cfg, "codex-ref", "codex-app-server") != nil || credentialMapped(cfg, "claude-ref", "claude-code") != nil {
		t.Fatal("exact mappings refused")
	}
	for _, tc := range [][2]string{{"missing", "codex-app-server"}, {"codex-ref", "claude-code"}, {"claude-ref", "codex-app-server"}, {"", "claude-code"}} {
		if credentialMapped(cfg, tc[0], tc[1]) == nil {
			t.Errorf("%v accepted", tc)
		}
	}
}
