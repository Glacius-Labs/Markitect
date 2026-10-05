package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const fixtureSource = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestPortfolioKeepsArenaAndRunKindsSeparateAndPreservesExclusion(t *testing.T) {
	dir := t.TempDir()
	one := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "same-run", "01", "passed", "", []string{"module/X"})}, "true")
	two := fixtureReport("freeze-two", []map[string]any{fixtureTask("parallel-integration", "same-run", "08", "passed", "excluded by source record", []string{"ordinary08/expected"})}, "true")
	entries := []reportInput{
		fixtureEntry(t, dir, "arena-one", "report-one", "one.yaml", one, trajectory{ID: "seq-a1", Project: "sample", Arm: "A", Trial: 1, RunID: "same-run", RunKind: "sequential", TaskIDs: []string{"01"}}),
		fixtureEntry(t, dir, "arena-two", "report-two", "two.yaml", two, trajectory{ID: "integration-a1", Project: "sample", Arm: "A", Trial: 1, RunID: "same-run", RunKind: "parallel-integration", TaskIDs: []string{"08"}}),
	}
	got, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: entries}, "manifest-hash", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Reports) != 2 || got.Reports[0].ArenaID == got.Reports[1].ArenaID {
		t.Fatalf("reports were pooled or lost: %+v", got.Reports)
	}
	var integration portfolioTask
	for _, report := range got.Reports {
		for _, tr := range report.Trajectories {
			if tr.TrajectoryID == "integration-a1" {
				integration = tr.TaskRows[0]
			}
		}
	}
	if integration.RawCountedOutcome != "passed" || integration.PortfolioDisposition != "excluded-descriptive" || integration.ComparisonEligible {
		t.Fatalf("excluded raw pass was promoted: %+v", integration)
	}
	if integration.ExpectedSetStatus != "unavailable-integration-card" || integration.EscalationStatus != "unavailable-integration-card" {
		t.Fatalf("integration Task08 was interpreted using ordinary card fields: %+v", integration)
	}
	if !strings.Contains(fmt.Sprint(integration.RawRow["frozen_expected_affected"]), "ordinary08/expected") || integration.RawComparisonExclusion != "excluded by source record" {
		t.Fatalf("raw values/exclusion were not preserved: %+v", integration)
	}
}

func TestPortfolioRejectsUnknownAndHoldoutTaskIDs(t *testing.T) {
	for _, id := range []string{"99", "09"} {
		t.Run(id, func(t *testing.T) {
			dir := t.TempDir()
			report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", id, "unknown", "", nil)}, "true")
			entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
				trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{id}})
			_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
			if err == nil || !strings.Contains(err.Error(), "unknown, holdout") {
				t.Fatalf("expected unknown/holdout refusal, got %v", err)
			}
		})
	}
}

func TestPortfolioRejectsDuplicateDesignationsAndRawRows(t *testing.T) {
	t.Run("duplicate designated task", func(t *testing.T) {
		dir := t.TempDir()
		report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01", "01"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "duplicates designated task") {
			t.Fatalf("expected duplicate selection refusal, got %v", err)
		}
	})
	t.Run("duplicate report task identity", func(t *testing.T) {
		dir := t.TempDir()
		row := fixtureTask("sequential", "run", "01", "passed", "", nil)
		report := fixtureReport("freeze-one", []map[string]any{row, row}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "duplicate task identity") {
			t.Fatalf("expected duplicate report identity refusal, got %v", err)
		}
	})
}

func TestPortfolioRejectsManifestAndSourceIdentityMismatches(t *testing.T) {
	t.Run("report byte digest", func(t *testing.T) {
		dir := t.TempDir()
		report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		entry.ReportSHA256 = strings.Repeat("0", 64)
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
			t.Fatalf("expected digest refusal, got %v", err)
		}
	})
	t.Run("freeze digest", func(t *testing.T) {
		dir := t.TempDir()
		report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		entry.FreezeDigest = strings.Repeat("b", 64)
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "freeze digest mismatch") {
			t.Fatalf("expected freeze refusal, got %v", err)
		}
	})
	t.Run("source version", func(t *testing.T) {
		dir := t.TempDir()
		report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		entry.ReportSourceVersion = "analysis-source-sha256:" + strings.Repeat("b", 64)
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "source version mismatch") {
			t.Fatalf("expected source-version refusal, got %v", err)
		}
	})
}

func TestPortfolioRejectsInvalidTrialIdentity(t *testing.T) {
	t.Run("fractional report trial", func(t *testing.T) {
		dir := t.TempDir()
		row := fixtureTask("sequential", "run", "01", "passed", "", nil)
		row["trial"] = 1.5
		report := fixtureReport("freeze-one", []map[string]any{row}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "incomplete trajectory identity") {
			t.Fatalf("expected fractional trial refusal, got %v", err)
		}
	})
	t.Run("manifest trial outside frozen range", func(t *testing.T) {
		dir := t.TempDir()
		report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 4, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "incomplete trajectory selector") {
			t.Fatalf("expected out-of-range trial refusal, got %v", err)
		}
	})
	t.Run("report trial outside frozen range", func(t *testing.T) {
		dir := t.TempDir()
		row := fixtureTask("sequential", "run", "01", "passed", "", nil)
		row["trial"] = 4
		report := fixtureReport("freeze-one", []map[string]any{row}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "incomplete trajectory identity") {
			t.Fatalf("expected out-of-range report trial refusal, got %v", err)
		}
	})
}

func TestPortfolioRejectsDelimiterBearingIdentityComponents(t *testing.T) {
	dir := t.TempDir()
	row := fixtureTask("sequential", "run", "01", "passed", "", nil)
	row["project"] = "sample|A|1"
	report := fixtureReport("freeze-one", []map[string]any{row}, "true")
	entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
		trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
	_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
	if err == nil || !strings.Contains(err.Error(), "incomplete trajectory identity") {
		t.Fatalf("expected delimiter collision refusal, got %v", err)
	}
}

func TestPortfolioRejectsPaddedIdentityFields(t *testing.T) {
	for _, field := range []string{"project", "run_id", "task_id"} {
		t.Run("source "+field, func(t *testing.T) {
			dir := t.TempDir()
			row := fixtureTask("sequential", "run", "01", "passed", "", nil)
			row[field] = map[string]string{"project": " sample", "run_id": "run ", "task_id": " 01"}[field]
			report := fixtureReport("freeze-one", []map[string]any{row}, "true")
			entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
				trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
			_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
			if err == nil {
				t.Fatalf("expected padded source %s refusal", field)
			}
		})
	}
	for _, field := range []string{"project", "run_id", "task_id"} {
		t.Run("manifest "+field, func(t *testing.T) {
			dir := t.TempDir()
			report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
			selected := trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}}
			switch field {
			case "project":
				selected.Project = "sample "
			case "run_id":
				selected.RunID = " run"
			case "task_id":
				selected.TaskIDs = []string{"01 "}
			}
			entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report, selected)
			_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
			if err == nil {
				t.Fatalf("expected padded manifest %s refusal", field)
			}
		})
	}
}

func TestPortfolioRejectsRunKindTaskMismatchAndExtraReportDocument(t *testing.T) {
	t.Run("absent fork task cannot be designated sequentially", func(t *testing.T) {
		dir := t.TempDir()
		report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"P01"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "run-kind-incompatible") {
			t.Fatalf("expected run-kind/task refusal, got %v", err)
		}
	})
	for _, alias := range []string{"01x", "07junk"} {
		t.Run("manifest task alias "+alias, func(t *testing.T) {
			dir := t.TempDir()
			report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
			entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
				trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{alias}})
			_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
			if err == nil || !strings.Contains(err.Error(), "run-kind-incompatible") {
				t.Fatalf("expected exact manifest task identity refusal, got %v", err)
			}
		})
		t.Run("source task alias "+alias, func(t *testing.T) {
			dir := t.TempDir()
			report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", alias, "passed", "", nil)}, "true")
			entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
				trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
			_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
			if err == nil || !strings.Contains(err.Error(), "run-kind-incompatible") {
				t.Fatalf("expected exact report task identity refusal, got %v", err)
			}
		})
	}
	t.Run("report with second YAML document", func(t *testing.T) {
		dir := t.TempDir()
		report := append(fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true"), []byte("---\nextra: document\n")...)
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "exactly one YAML document") {
			t.Fatalf("expected extra document refusal, got %v", err)
		}
	})
}

func TestCLIUsesExclusiveOutput(t *testing.T) {
	dir := t.TempDir()
	report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
	entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
		trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
	manifestBytes, err := yaml.Marshal(manifest{Schema: manifestSchema, Reports: []reportInput{entry}})
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "portfolio.yaml")
	if err := run([]string{"--manifest", manifestPath, "--out", out}, &strings.Builder{}, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var decoded portfolio
	if err := yaml.Unmarshal(first, &decoded); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}
	if decoded.Schema != outputSchema || decoded.PlannedPopulation.PlannedCells != 216 || decoded.PlannedPopulation.DevelopmentCells != 144 || decoded.PlannedPopulation.ReservedHoldoutCells != 72 {
		t.Fatalf("YAML output lost fixed denominator metadata: %+v", decoded)
	}
	if err := run([]string{"--manifest", manifestPath, "--out", out}, &strings.Builder{}, &strings.Builder{}); err == nil {
		t.Fatal("existing output was overwritten")
	}
	second, _ := os.ReadFile(out)
	if string(first) != string(second) {
		t.Fatal("exclusive refusal modified existing output")
	}
}

func fixtureReport(freeze string, tasks []map[string]any, comparisonEligible string) []byte {
	switch freeze {
	case "freeze-one":
		freeze = strings.Repeat("a", 64)
	case "freeze-two":
		freeze = strings.Repeat("b", 64)
	}
	value := map[string]any{
		"schema": reportSchema, "analysis_source_sha256": fixtureSource, "freeze_digest": freeze,
		"cohort_status": "available", "comparison_eligible": comparisonEligible,
		"run_exclusion_record_status": "unavailable", "oracle_exclusion_record_status": "unavailable",
		"runtime_exclusion_record_status": "unavailable", "assessment_exclusion_record_status": "unavailable",
		"tasks": tasks,
	}
	encoded, err := yaml.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func fixtureTask(kind, runID, taskID, outcome, exclusion string, expected []string) map[string]any {
	return map[string]any{
		"project": "sample", "arm": "A", "trial": 1, "run_id": runID, "run_kind": kind, "task_id": taskID,
		"counted_outcome": outcome, "comparison_exclusion_reason": exclusion,
		"frozen_expected_affected": expected, "expected_affected_status": "available",
		"owner_decision_required": false, "unexpected_owner_decision": false,
	}
}

func fixtureEntry(t *testing.T, dir, arenaID, reportID, filename string, reportBytes []byte, selected trajectory) reportInput {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, reportBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(reportBytes)
	var source analysisReport
	decoder := yaml.NewDecoder(strings.NewReader(string(reportBytes)))
	if err := decoder.Decode(&source); err != nil {
		t.Fatal(err)
	}
	return reportInput{
		ArenaID: arenaID, FreezeDigest: source.FreezeDigest, ReportID: reportID,
		ReportPath: path, ReportSHA256: hex.EncodeToString(sum[:]), ReportSchema: reportSchema,
		ReportSourceVersion: "analysis-source-sha256:" + fixtureSource, Trajectories: []trajectory{selected},
	}
}

func TestPortfolioRejectsAmbiguousSequentialAndTrajectoryIdentity(t *testing.T) {
	t.Run("two sequential runs", func(t *testing.T) {
		dir := t.TempDir()
		rows := []map[string]any{
			fixtureTask("sequential", "run-one", "01", "passed", "", nil),
			fixtureTask("sequential", "run-two", "02", "passed", "", nil),
		}
		report := fixtureReport("freeze-one", rows, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t1", Project: "sample", Arm: "A", Trial: 1, RunID: "run-one", RunKind: "sequential", TaskIDs: []string{"01"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "ambiguous sequential") {
			t.Fatalf("expected sequential ambiguity refusal, got %v", err)
		}
	})
	t.Run("duplicate observation across same arena freeze", func(t *testing.T) {
		dir := t.TempDir()
		report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "failed", "", nil)}, "true")
		first := fixtureEntry(t, dir, "arena", "report-one", "one.yaml", report,
			trajectory{ID: "first", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		second := fixtureEntry(t, dir, "arena", "report-two", "two.yaml", report,
			trajectory{ID: "second", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{first, second}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "duplicate designated observation") {
			t.Fatalf("expected duplicate observation refusal, got %v", err)
		}
	})
	t.Run("sequential run identity across reports", func(t *testing.T) {
		dir := t.TempDir()
		one := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run-one", "01", "failed", "", nil)}, "true")
		two := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run-two", "02", "passed", "", nil)}, "true")
		first := fixtureEntry(t, dir, "arena", "report-one", "one.yaml", one,
			trajectory{ID: "first", Project: "sample", Arm: "A", Trial: 1, RunID: "run-one", RunKind: "sequential", TaskIDs: []string{"01"}})
		second := fixtureEntry(t, dir, "arena", "report-two", "two.yaml", two,
			trajectory{ID: "second", Project: "sample", Arm: "A", Trial: 1, RunID: "run-two", RunKind: "sequential", TaskIDs: []string{"02"}})
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{first, second}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "ambiguous sequential run identity across reports") {
			t.Fatalf("expected cross-report sequential identity refusal, got %v", err)
		}
	})
	t.Run("missing known task remains unavailable", func(t *testing.T) {
		dir := t.TempDir()
		report := fixtureReport("freeze-one", []map[string]any{fixtureTask("sequential", "run", "01", "passed", "", nil)}, "true")
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
			trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01", "02"}})
		got, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err != nil {
			t.Fatal(err)
		}
		if got.ObservedCoverage.DesignatedTrajectoryTaskSlots != 2 || got.ObservedCoverage.ObservedTaskRows != 1 || got.ObservedCoverage.UnavailableDesignatedSlots != 1 {
			t.Fatalf("denominator and observed coverage were conflated: %+v", got.ObservedCoverage)
		}
		missing := got.Reports[0].Trajectories[0].TaskRows[1]
		if missing.TaskID != "02" || missing.RawCountedOutcome != "unavailable" || missing.ComparisonEligible {
			t.Fatalf("missing known task was lost or promoted: %+v", missing)
		}
	})
	t.Run("duplicate trajectory selectors", func(t *testing.T) {
		dir := t.TempDir()
		report := fixtureReport("freeze-one", []map[string]any{fixtureTask("parallel-fork", "fork-run", "P01", "passed", "", nil)}, "true")
		first := trajectory{ID: "fork-1", Project: "sample", Arm: "A", Trial: 1, RunID: "fork-run", RunKind: "parallel-fork", TaskIDs: []string{"P01"}}
		second := trajectory{ID: "fork-2", Project: "sample", Arm: "A", Trial: 1, RunID: "fork-run", RunKind: "parallel-fork", TaskIDs: []string{"P01"}}
		entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report, first)
		entry.Trajectories = append(entry.Trajectories, second)
		_, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
		if err == nil || !strings.Contains(err.Error(), "duplicate designated trajectory") {
			t.Fatalf("expected trajectory ambiguity refusal, got %v", err)
		}
	})
}

func TestPortfolioDoesNotAssignEligibilityEvenForRawPass(t *testing.T) {
	dir := t.TempDir()
	fixture := fixtureTask("sequential", "run", "01", "passed", "", nil)
	fixture["counted_outcome"] = " passed "
	report := fixtureReport("freeze-one", []map[string]any{fixture}, "true")
	entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", report,
		trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
	got, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
	if err != nil {
		t.Fatal(err)
	}
	row := got.Reports[0].Trajectories[0].TaskRows[0]
	if row.RawCountedOutcome != " passed " || row.ComparisonEligible || got.MetricsProduced {
		t.Fatalf("portfolio promoted a raw pass: %+v", row)
	}
}

func TestReportLevelExclusionCannotPromoteRawPass(t *testing.T) {
	dir := t.TempDir()
	reportBytes, err := yaml.Marshal(map[string]any{
		"schema": reportSchema, "analysis_source_sha256": fixtureSource, "freeze_digest": strings.Repeat("a", 64),
		"cohort_status": "invalidated", "comparison_eligible": "false", "tasks": []map[string]any{
			fixtureTask("sequential", "run", "01", "passed", "", nil),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	entry := fixtureEntry(t, dir, "arena", "report", "report.yaml", reportBytes,
		trajectory{ID: "t", Project: "sample", Arm: "A", Trial: 1, RunID: "run", RunKind: "sequential", TaskIDs: []string{"01"}})
	got, err := buildPortfolio(manifest{Schema: manifestSchema, Reports: []reportInput{entry}}, "m", dir)
	if err != nil {
		t.Fatal(err)
	}
	row := got.Reports[0].Trajectories[0].TaskRows[0]
	if row.RawCountedOutcome != "passed" || row.PortfolioDisposition != "excluded-descriptive" || row.ComparisonEligible {
		t.Fatalf("report-level exclusion did not remain descriptive: %+v", row)
	}
}
