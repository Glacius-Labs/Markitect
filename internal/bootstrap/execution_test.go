package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	service := newExecution(apply.NewPreparationService(apply.NewModelService(nil, nil)), &gitrepo.Repository{}, nil, nil, &bytes.Buffer{})
	report, err := service.Execute(context.Background(), cli.ApplyRequest{RunConfig: writeRunConfig(t, document)})
	if !errors.Is(err, errRunConfig) || report.Status != model.Failed || report.EvidenceDirectory != "" {
		t.Fatalf("%v %+v", err, report)
	}
	if _, err := os.Stat(document["stateDirectory"].(string)); !os.IsNotExist(err) {
		t.Fatal("an invalid configuration created the state directory")
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
	// Preparation would panic on this empty composition; the lock comes first.
	service := newExecution(apply.NewPreparationService(apply.NewModelService(nil, nil)), &gitrepo.Repository{}, nil, nil, &bytes.Buffer{})
	report, err := service.Execute(context.Background(), cli.ApplyRequest{RunConfig: writeRunConfig(t, document)})
	if !errors.Is(err, errStateLocked) || report.Status != model.NotCompleted || report.Preparation != nil {
		t.Fatalf("%v %+v", err, report)
	}
	if entries, _ := os.ReadDir(filepath.Join(state, "runs")); len(entries) != 0 {
		t.Fatalf("a refused Apply left evidence: %v", entries)
	}
}

func TestAnswerWritesForTheRunningApplyOfTheConfiguredStateDirectory(t *testing.T) {
	document := runConfigFixture(t)
	state := document["stateDirectory"].(string)
	config := writeRunConfig(t, document)
	if report, err := answer(context.Background(), cli.AnswerRequest{Question: "q1", Text: "Euro.", RunConfig: config}); err == nil || report.Status != model.NotCompleted {
		t.Fatalf("answer without a running Apply: %+v %v", report, err)
	}
	owner, err := newFileOwner(state, "run-1", config, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	held, err := lockState(state)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	if err := owner.Ask(context.Background(), runport.Question{ID: "q1", Text: "Which currency?"}); err != nil {
		t.Fatal(err)
	}
	report, err := answer(context.Background(), cli.AnswerRequest{Question: "q1", Text: "Euro.", RunConfig: config})
	if err != nil || report.Status != model.Passed || report.Path != filepath.Join(state, "answers", "q1.txt") || report.Bytes != len("Euro.") {
		t.Fatalf("%+v %v", report, err)
	}
	if text, answered, err := owner.Answer(context.Background(), "q1"); err != nil || !answered || text != "Euro." {
		t.Fatalf("the running Apply did not see the answer: %q %v %v", text, answered, err)
	}
	document["formatVersion"] = 1
	if report, err := answer(context.Background(), cli.AnswerRequest{Question: "q1", Text: "x", RunConfig: writeRunConfig(t, document)}); !errors.Is(err, errRunConfig) || report.Status != model.Failed {
		t.Fatalf("invalid configuration: %+v %v", report, err)
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
}
