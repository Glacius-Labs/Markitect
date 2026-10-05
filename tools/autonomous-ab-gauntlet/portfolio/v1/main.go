package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	manifestSchema = "autonomous-ab-portfolio-manifest/v1"
	reportSchema   = "autonomous-ab-analysis/v2"
	outputSchema   = "autonomous-ab-portfolio/v1"
)

var (
	sha256Pattern   = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	identityPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	ordinaryTasks   = map[string]bool{"01": true, "02": true, "03": true, "04": true, "05": true, "06": true, "07": true, "08": true}
)

type manifest struct {
	Schema  string        `yaml:"schema"`
	Reports []reportInput `yaml:"reports"`
}

type reportInput struct {
	ArenaID             string       `yaml:"arena_id"`
	FreezeDigest        string       `yaml:"freeze_digest"`
	ReportID            string       `yaml:"report_id"`
	ReportPath          string       `yaml:"report_path"`
	ReportSHA256        string       `yaml:"report_sha256"`
	ReportSchema        string       `yaml:"report_schema"`
	ReportSourceVersion string       `yaml:"report_source_version"`
	Trajectories        []trajectory `yaml:"trajectories"`
}

type trajectory struct {
	ID      string   `yaml:"trajectory_id"`
	Project string   `yaml:"project"`
	Arm     string   `yaml:"arm"`
	Trial   int      `yaml:"trial"`
	RunID   string   `yaml:"run_id"`
	RunKind string   `yaml:"run_kind"`
	TaskIDs []string `yaml:"task_ids"`
}

type analysisReport struct {
	Schema                          string           `yaml:"schema"`
	AnalysisSourceSHA256            string           `yaml:"analysis_source_sha256"`
	FreezeDigest                    string           `yaml:"freeze_digest"`
	CohortStatus                    string           `yaml:"cohort_status"`
	ComparisonEligible              string           `yaml:"comparison_eligible"`
	InvalidationReason              string           `yaml:"invalidation_reason"`
	InvalidationRecordPath          string           `yaml:"invalidation_record_path"`
	InvalidationRecordSHA256        string           `yaml:"invalidation_record_sha256"`
	RunExclusionRecordStatus        string           `yaml:"run_exclusion_record_status"`
	RunExclusionRecordSHA256        string           `yaml:"run_exclusion_record_sha256"`
	OracleExclusionRecordStatus     string           `yaml:"oracle_exclusion_record_status"`
	OracleExclusionRecordSHA256     string           `yaml:"oracle_exclusion_record_sha256"`
	RuntimeExclusionRecordStatus    string           `yaml:"runtime_exclusion_record_status"`
	RuntimeExclusionRecordSHA256    string           `yaml:"runtime_exclusion_record_sha256"`
	AssessmentExclusionRecordStatus string           `yaml:"assessment_exclusion_record_status"`
	AssessmentExclusionRecordSHA256 string           `yaml:"assessment_exclusion_record_sha256"`
	Tasks                           []map[string]any `yaml:"tasks"`
}

type portfolio struct {
	Schema            string            `yaml:"schema"`
	ManifestSHA256    string            `yaml:"manifest_sha256"`
	Status            string            `yaml:"status"`
	MetricsProduced   bool              `yaml:"metrics_produced"`
	PlannedPopulation plannedPopulation `yaml:"planned_population"`
	ObservedCoverage  observedCoverage  `yaml:"observed_coverage"`
	Reports           []portfolioReport `yaml:"reports"`
	Limitations       []string          `yaml:"limitations"`
}

type plannedPopulation struct {
	Projects                int `yaml:"projects"`
	Arms                    int `yaml:"arms"`
	Trials                  int `yaml:"trials"`
	TasksPerProjectArmTrial int `yaml:"tasks_per_project_arm_trial"`
	DevelopmentTasks        int `yaml:"development_tasks"`
	ReservedHoldoutTasks    int `yaml:"reserved_holdout_tasks"`
	PlannedCells            int `yaml:"planned_cells"`
	DevelopmentCells        int `yaml:"development_cells"`
	ReservedHoldoutCells    int `yaml:"reserved_holdout_cells"`
}

type observedCoverage struct {
	ReportEntries                  int `yaml:"report_entries"`
	DesignatedTrajectories         int `yaml:"designated_trajectories"`
	ReportBackedTrajectories       int `yaml:"report_backed_trajectories"`
	UnbackedDesignatedTrajectories int `yaml:"unbacked_designated_trajectories"`
	DesignatedTrajectoryTaskSlots  int `yaml:"designated_trajectory_task_slots"`
	ObservedTaskRows               int `yaml:"observed_task_rows"`
	UnavailableDesignatedSlots     int `yaml:"unavailable_designated_slots"`
}

type portfolioReport struct {
	ArenaID             string                `yaml:"arena_id"`
	FreezeDigest        string                `yaml:"freeze_digest"`
	ReportID            string                `yaml:"report_id"`
	ReportPath          string                `yaml:"report_path"`
	ReportSHA256        string                `yaml:"report_sha256"`
	ReportSchema        string                `yaml:"report_schema"`
	ReportSourceVersion string                `yaml:"report_source_version"`
	SourceExclusions    map[string]string     `yaml:"source_exclusions"`
	Trajectories        []portfolioTrajectory `yaml:"trajectories"`
}

type portfolioTrajectory struct {
	TrajectoryID         string          `yaml:"trajectory_id"`
	SourceIdentityStatus string          `yaml:"source_identity_status"`
	Project              string          `yaml:"project"`
	Arm                  string          `yaml:"arm"`
	Trial                int             `yaml:"trial"`
	RunID                string          `yaml:"run_id"`
	RunKind              string          `yaml:"run_kind"`
	TaskRows             []portfolioTask `yaml:"task_rows"`
}

type portfolioTask struct {
	TaskID                 string         `yaml:"task_id"`
	RawCountedOutcome      string         `yaml:"raw_counted_outcome"`
	RawComparisonExclusion string         `yaml:"raw_comparison_exclusion_reason,omitempty"`
	PortfolioDisposition   string         `yaml:"portfolio_disposition"`
	ComparisonEligible     bool           `yaml:"comparison_eligible"`
	ExpectedSetStatus      string         `yaml:"expected_set_status"`
	EscalationStatus       string         `yaml:"escalation_status"`
	RawRow                 map[string]any `yaml:"raw_row,omitempty"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("portfolio-v1", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifestPath := flags.String("manifest", "", "explicit byte-pinned portfolio manifest")
	outputPath := flags.String("out", "", "new YAML output path (must not exist)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *manifestPath == "" || *outputPath == "" {
		return errors.New("usage: portfolio-v1 --manifest <manifest.yaml> --out <new-report.yaml>")
	}
	manifestAbs, err := filepath.Abs(*manifestPath)
	if err != nil {
		return err
	}
	manifestBytes, err := readRegular(manifestAbs)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	manifestHash := digest(manifestBytes)
	var input manifest
	decoder := yaml.NewDecoder(strings.NewReader(string(manifestBytes)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("manifest must contain exactly one YAML document")
	}
	if input.Schema != manifestSchema || len(input.Reports) == 0 {
		return fmt.Errorf("manifest schema must be %q and include reports", manifestSchema)
	}
	result, err := buildPortfolio(input, manifestHash, filepath.Dir(manifestAbs))
	if err != nil {
		return err
	}
	encoded, err := yaml.Marshal(result)
	if err != nil {
		return err
	}
	outAbs, err := filepath.Abs(*outputPath)
	if err != nil {
		return err
	}
	if outAbs == manifestAbs {
		return errors.New("output path aliases manifest")
	}
	for _, report := range input.Reports {
		reportAbs, e := resolveReportPath(filepath.Dir(manifestAbs), report.ReportPath)
		if e != nil {
			return e
		}
		if outAbs == reportAbs {
			return errors.New("output path aliases a pinned report")
		}
	}
	if err := writeExclusive(outAbs, encoded); err != nil {
		return fmt.Errorf("write portfolio output: %w", err)
	}
	_, _ = fmt.Fprintf(stdout, "portfolio: %s\nmanifest_sha256: %s\n", outAbs, manifestHash)
	return nil
}

func buildPortfolio(input manifest, manifestHash, manifestDir string) (portfolio, error) {
	result := portfolio{
		Schema:          outputSchema,
		ManifestSHA256:  manifestHash,
		Status:          "validated-descriptive-only",
		MetricsProduced: false,
		PlannedPopulation: plannedPopulation{Projects: 3, Arms: 2, Trials: 3, TasksPerProjectArmTrial: 12,
			DevelopmentTasks: 8, ReservedHoldoutTasks: 4, PlannedCells: 216, DevelopmentCells: 144, ReservedHoldoutCells: 72},
		ObservedCoverage: observedCoverage{ReportEntries: len(input.Reports)},
		Limitations: []string{
			"No actor collection, evaluator invocation, rescoring, pass aggregation, or winner comparison is performed.",
			"Each trajectory remains bound to one report and arena; repeated run IDs across arenas are never pooled.",
			"Integration Task08 expected-set and escalation semantics are unavailable; raw report values are preserved without interpretation.",
			"Non-exhaustive reports and absent rows cannot establish a task denominator beyond the explicit manifest selection.",
		},
	}
	seenReport, seenTrajectory, seenTrajectoryKey, seenObservation, seenSequential := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]string{}
	for _, entry := range input.Reports {
		if err := validateReportEntry(entry, seenReport); err != nil {
			return portfolio{}, err
		}
		reportPath, err := resolveReportPath(manifestDir, entry.ReportPath)
		if err != nil {
			return portfolio{}, err
		}
		reportBytes, err := readRegular(reportPath)
		if err != nil {
			return portfolio{}, fmt.Errorf("read report %q: %w", entry.ReportID, err)
		}
		actualHash := digest(reportBytes)
		if !strings.EqualFold(actualHash, entry.ReportSHA256) {
			return portfolio{}, fmt.Errorf("report %q digest mismatch: declared %s, actual %s", entry.ReportID, entry.ReportSHA256, actualHash)
		}
		var source analysisReport
		reportDecoder := yaml.NewDecoder(strings.NewReader(string(reportBytes)))
		if err := reportDecoder.Decode(&source); err != nil {
			return portfolio{}, fmt.Errorf("decode report %q: %w", entry.ReportID, err)
		}
		var trailing any
		if err := reportDecoder.Decode(&trailing); err != io.EOF {
			return portfolio{}, fmt.Errorf("report %q must contain exactly one YAML document", entry.ReportID)
		}
		if source.Schema != entry.ReportSchema || source.Schema != reportSchema {
			return portfolio{}, fmt.Errorf("report %q schema mismatch or unsupported schema %q", entry.ReportID, source.Schema)
		}
		if !strings.EqualFold(source.FreezeDigest, entry.FreezeDigest) {
			return portfolio{}, fmt.Errorf("report %q freeze digest mismatch", entry.ReportID)
		}
		if !sha256Pattern.MatchString(source.AnalysisSourceSHA256) || entry.ReportSourceVersion != "analysis-source-sha256:"+strings.ToLower(source.AnalysisSourceSHA256) {
			return portfolio{}, fmt.Errorf("report %q source version mismatch", entry.ReportID)
		}
		rows, kinds, err := indexRows(source.Tasks)
		if err != nil {
			return portfolio{}, fmt.Errorf("report %q: %w", entry.ReportID, err)
		}
		if err := validateTrajectories(entry, rows, kinds, seenTrajectory, seenTrajectoryKey, seenObservation, seenSequential); err != nil {
			return portfolio{}, err
		}
		outReport := portfolioReport{
			ArenaID: entry.ArenaID, FreezeDigest: strings.ToLower(entry.FreezeDigest),
			ReportID: entry.ReportID, ReportPath: reportPath, ReportSHA256: actualHash,
			ReportSchema: source.Schema, ReportSourceVersion: entry.ReportSourceVersion,
			SourceExclusions: map[string]string{
				"cohort_status": source.CohortStatus, "comparison_eligible": source.ComparisonEligible,
				"invalidation_reason": source.InvalidationReason, "invalidation_record_path": source.InvalidationRecordPath,
				"invalidation_record_sha256": source.InvalidationRecordSHA256,
				"run_exclusion_status":       source.RunExclusionRecordStatus, "run_exclusion_sha256": source.RunExclusionRecordSHA256,
				"oracle_exclusion_status": source.OracleExclusionRecordStatus, "oracle_exclusion_sha256": source.OracleExclusionRecordSHA256,
				"runtime_exclusion_status": source.RuntimeExclusionRecordStatus, "runtime_exclusion_sha256": source.RuntimeExclusionRecordSHA256,
				"assessment_exclusion_status": source.AssessmentExclusionRecordStatus, "assessment_exclusion_sha256": source.AssessmentExclusionRecordSHA256,
			},
		}
		for _, selected := range entry.Trajectories {
			outTrajectory := portfolioTrajectory{
				TrajectoryID: selected.ID, Project: selected.Project, Arm: strings.ToUpper(selected.Arm),
				Trial: selected.Trial, RunID: selected.RunID, RunKind: selected.RunKind,
			}
			trajectoryIndex := trajectoryKey(selected.Project, selected.Arm, selected.Trial, selected.RunID)
			if _, ok := kinds[trajectoryIndex]; ok {
				outTrajectory.SourceIdentityStatus = "matched"
				result.ObservedCoverage.ReportBackedTrajectories++
			} else {
				outTrajectory.SourceIdentityStatus = "unavailable-no-source-row"
				result.ObservedCoverage.UnbackedDesignatedTrajectories++
			}
			result.ObservedCoverage.DesignatedTrajectories++
			for _, taskID := range selected.TaskIDs {
				key := rowKey(selected.Project, selected.Arm, selected.Trial, selected.RunID, taskID)
				raw := rows[key]
				result.ObservedCoverage.DesignatedTrajectoryTaskSlots++
				if raw == nil {
					result.ObservedCoverage.UnavailableDesignatedSlots++
					expectedStatus, escalationStatus := "unavailable-no-source-row", "unavailable-no-source-row"
					if selected.RunKind == "parallel-integration" && taskID == "08" {
						expectedStatus, escalationStatus = "unavailable-integration-card", "unavailable-integration-card"
					}
					outTrajectory.TaskRows = append(outTrajectory.TaskRows, portfolioTask{
						TaskID: taskID, RawCountedOutcome: "unavailable",
						PortfolioDisposition: "unavailable-designated-task", ComparisonEligible: false,
						ExpectedSetStatus: expectedStatus, EscalationStatus: escalationStatus,
					})
					continue
				}
				result.ObservedCoverage.ObservedTaskRows++
				counted := stringField(raw, "counted_outcome")
				exclusion := stringField(raw, "comparison_exclusion_reason")
				excluded := source.CohortStatus == "invalidated" || strings.EqualFold(source.ComparisonEligible, "false") || counted == "invalidated" || exclusion != ""
				disposition := "retained-descriptive"
				if excluded {
					disposition = "excluded-descriptive"
				}
				expectedStatus := stringField(raw, "expected_affected_status")
				escalationStatus := "source-report-only"
				if selected.RunKind == "parallel-integration" && taskID == "08" {
					expectedStatus = "unavailable-integration-card"
					escalationStatus = "unavailable-integration-card"
				}
				outTrajectory.TaskRows = append(outTrajectory.TaskRows, portfolioTask{
					TaskID: taskID, RawCountedOutcome: counted, RawComparisonExclusion: exclusion,
					PortfolioDisposition: disposition, ComparisonEligible: false,
					ExpectedSetStatus: expectedStatus, EscalationStatus: escalationStatus, RawRow: raw,
				})
			}
			outReport.Trajectories = append(outReport.Trajectories, outTrajectory)
		}
		result.Reports = append(result.Reports, outReport)
	}
	sort.Slice(result.Reports, func(i, j int) bool { return result.Reports[i].ReportID < result.Reports[j].ReportID })
	return result, nil
}

func validateReportEntry(entry reportInput, seenReport map[string]bool) error {
	for _, required := range []struct{ name, value string }{{"arena_id", entry.ArenaID}, {"report_id", entry.ReportID}, {"report_path", entry.ReportPath}} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("report entry has empty %s", required.name)
		}
	}
	if seenReport[entry.ReportID] {
		return fmt.Errorf("duplicate report_id %q", entry.ReportID)
	}
	if !identityPattern.MatchString(entry.ArenaID) || !identityPattern.MatchString(entry.ReportID) {
		return fmt.Errorf("report %q has unsafe arena_id or report_id identity component", entry.ReportID)
	}
	seenReport[entry.ReportID] = true
	if !sha256Pattern.MatchString(entry.FreezeDigest) || !sha256Pattern.MatchString(entry.ReportSHA256) {
		return fmt.Errorf("report %q requires SHA-256 freeze and report digests", entry.ReportID)
	}
	if entry.ReportSchema != reportSchema {
		return fmt.Errorf("report %q declares unsupported schema %q", entry.ReportID, entry.ReportSchema)
	}
	if len(entry.Trajectories) == 0 {
		return fmt.Errorf("report %q has no designated trajectories", entry.ReportID)
	}
	return nil
}

func indexRows(tasks []map[string]any) (map[string]map[string]any, map[string]string, error) {
	rows, kinds := map[string]map[string]any{}, map[string]string{}
	sequential := map[string]string{}
	for _, row := range tasks {
		project, projectOK := identityField(row, "project")
		arm, armOK := identityField(row, "arm")
		runID, runIDOK := identityField(row, "run_id")
		taskID, taskIDOK := identityField(row, "task_id")
		runKind, runKindOK := identityField(row, "run_kind")
		trial, validTrial := trialField(row, "trial")
		arm = strings.ToUpper(arm)
		if !projectOK || !armOK || (arm != "A" && arm != "B") || !runIDOK || !taskIDOK || !runKindOK || !validTrial {
			return nil, nil, errors.New("task row has incomplete trajectory identity")
		}
		if !compatibleTask(runKind, taskID) {
			return nil, nil, fmt.Errorf("report contains unknown, holdout, or run-kind-incompatible task identity %q for %q", taskID, runKind)
		}
		key := rowKey(project, arm, trial, runID, taskID)
		if _, exists := rows[key]; exists {
			return nil, nil, fmt.Errorf("duplicate task identity %s/%s/%d/%s/%s", project, arm, trial, runID, taskID)
		}
		base := trajectoryKey(project, arm, trial, runID)
		if previous, exists := kinds[base]; exists && previous != runKind {
			return nil, nil, fmt.Errorf("ambiguous run kind for %s/%s/%d/%s", project, arm, trial, runID)
		}
		kinds[base] = runKind
		if runKind == "sequential" {
			sequence := fmt.Sprintf("%s/%s/%d", project, arm, trial)
			if previous, exists := sequential[sequence]; exists && previous != runID {
				return nil, nil, fmt.Errorf("ambiguous sequential run identity for %s: %s and %s", sequence, previous, runID)
			}
			sequential[sequence] = runID
		}
		rows[key] = row
	}
	return rows, kinds, nil
}

func validateTrajectories(entry reportInput, rows map[string]map[string]any, kinds map[string]string, seenIDs, seenKeys, seenObservations map[string]bool, seenSequential map[string]string) error {
	for _, selected := range entry.Trajectories {
		if !identityPattern.MatchString(selected.ID) || !identityPattern.MatchString(selected.Project) || !identityPattern.MatchString(selected.RunID) || selected.Trial < 1 || selected.Trial > 3 || (strings.ToUpper(selected.Arm) != "A" && strings.ToUpper(selected.Arm) != "B") {
			return fmt.Errorf("report %q has incomplete trajectory selector", entry.ReportID)
		}
		if selected.RunKind != "sequential" && selected.RunKind != "parallel-fork" && selected.RunKind != "parallel-integration" {
			return fmt.Errorf("trajectory %q has unknown run_kind %q", selected.ID, selected.RunKind)
		}
		if seenIDs[selected.ID] {
			return fmt.Errorf("duplicate trajectory_id %q", selected.ID)
		}
		seenIDs[selected.ID] = true
		key := entry.ReportID + "/" + trajectoryKey(selected.Project, selected.Arm, selected.Trial, selected.RunID)
		if seenKeys[key] {
			return fmt.Errorf("duplicate designated trajectory identity %s", key)
		}
		seenKeys[key] = true
		actualKey := trajectoryKey(selected.Project, selected.Arm, selected.Trial, selected.RunID)
		if actualKind, ok := kinds[actualKey]; ok && actualKind != selected.RunKind {
			return fmt.Errorf("trajectory %q does not uniquely match its declared run kind", selected.ID)
		}
		if selected.RunKind == "sequential" {
			sequentialKey := strings.Join([]string{entry.ArenaID, strings.ToLower(entry.FreezeDigest), selected.Project, strings.ToUpper(selected.Arm), fmt.Sprint(selected.Trial)}, "|")
			if previous, ok := seenSequential[sequentialKey]; ok && previous != selected.RunID {
				return fmt.Errorf("ambiguous sequential run identity across reports for %s: %s and %s", sequentialKey, previous, selected.RunID)
			}
			seenSequential[sequentialKey] = selected.RunID
		}
		if len(selected.TaskIDs) == 0 {
			return fmt.Errorf("trajectory %q has no designated task IDs", selected.ID)
		}
		seenTasks := map[string]bool{}
		for _, taskID := range selected.TaskIDs {
			if !compatibleTask(selected.RunKind, taskID) {
				return fmt.Errorf("trajectory %q has unknown, holdout, or run-kind-incompatible task identity %q", selected.ID, taskID)
			}
			if seenTasks[taskID] {
				return fmt.Errorf("trajectory %q duplicates designated task %q", selected.ID, taskID)
			}
			seenTasks[taskID] = true
			rowKey := rowKey(selected.Project, selected.Arm, selected.Trial, selected.RunID, taskID)
			if raw, ok := rows[rowKey]; ok {
				if stringField(raw, "run_kind") != selected.RunKind {
					return fmt.Errorf("trajectory %q task %q has conflicting run kind", selected.ID, taskID)
				}
			}
			observationKey := strings.Join([]string{entry.ArenaID, strings.ToLower(entry.FreezeDigest), selected.RunID, selected.Project, strings.ToUpper(selected.Arm), fmt.Sprint(selected.Trial), taskID, selected.RunKind}, "|")
			if seenObservations[observationKey] {
				return fmt.Errorf("duplicate designated observation identity %s", observationKey)
			}
			seenObservations[observationKey] = true
		}
	}
	return nil
}

func resolveReportPath(manifestDir, value string) (string, error) {
	path := value
	if !filepath.IsAbs(path) {
		path = filepath.Join(manifestDir, filepath.FromSlash(path))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular non-symlink file: %s", path)
	}
	return os.ReadFile(path)
}

func writeExclusive(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func stringField(row map[string]any, name string) string {
	value, _ := row[name].(string)
	return value
}

func identityField(row map[string]any, name string) (string, bool) {
	value, ok := row[name].(string)
	return value, ok && value == strings.TrimSpace(value) && identityPattern.MatchString(value)
}

func trialField(row map[string]any, name string) (int, bool) {
	switch value := row[name].(type) {
	case int:
		return value, value >= 1 && value <= 3
	case int64:
		return int(value), value >= 1 && value <= 3
	case uint64:
		return int(value), value >= 1 && value <= 3
	case float64:
		if value != float64(int(value)) {
			return 0, false
		}
		trial := int(value)
		return trial, trial >= 1 && trial <= 3
	default:
		return 0, false
	}
}

func compatibleTask(runKind, taskID string) bool {
	switch runKind {
	case "sequential":
		return ordinaryTasks[taskID]
	case "parallel-fork":
		return taskID == "P01" || taskID == "P02"
	case "parallel-integration":
		return taskID == "08"
	default:
		return false
	}
}

func rowKey(project, arm string, trial int, runID, taskID string) string {
	return fmt.Sprintf("%s/%s/%d/%s/%s", project, strings.ToUpper(arm), trial, runID, taskID)
}

func trajectoryKey(project, arm string, trial int, runID string) string {
	return fmt.Sprintf("%s/%s/%d/%s", project, strings.ToUpper(arm), trial, runID)
}
