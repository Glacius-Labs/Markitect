// Command analysis produces a descriptive, post-run report from an external gauntlet arena.
// It is deliberately outside the frozen protocol, task cards, harness, and evaluators.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

type taskCard struct {
	ID                    string              `yaml:"id"`
	ExpectedAffected      []string            `yaml:"expected_affected"`
	RequiresOwnerDecision bool                `yaml:"requires_owner_decision"`
	AllowedPaths          []string            `yaml:"allowed_paths"`
	AllowedPathsByArm     map[string][]string `yaml:"allowed_paths_by_arm"`
	OriginTaskID          string              `yaml:"origin_task_id"`
	PriorTasksThrough     string              `yaml:"prior_tasks_through"`
}
type taskSet struct {
	Tasks []taskCard `yaml:"tasks"`
}
type projectManifest struct {
	TaskSet  string `yaml:"task_set"`
	Controls struct {
		TaskSet         string `yaml:"task_set"`
		ParallelTaskSet string `yaml:"parallel_task_set"`
	} `yaml:"controls"`
	Shared struct {
		TaskSet string `yaml:"taskSet"`
	} `yaml:"shared"`
}
type freezeInput struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
}
type turnRecord struct {
	Attempt      int    `yaml:"attempt"`
	Status       string `yaml:"status"`
	ResponseFile string `yaml:"response_file"`
	ReceiptsFile string `yaml:"receipts_file"`
}
type nativeRecord struct {
	RunID               string       `yaml:"run_id"`
	Project             string       `yaml:"project"`
	Arm                 string       `yaml:"arm"`
	Trial               int          `yaml:"trial"`
	TaskID              string       `yaml:"task_id"`
	EvaluationTaskID    string       `yaml:"evaluation_task_id"`
	ActorStatus         string       `yaml:"actor_status"`
	ActorStartedUTC     string       `yaml:"actor_started_utc"`
	ActorCompletedUTC   string       `yaml:"actor_completed_utc"`
	DeadlineUTC         string       `yaml:"deadline_utc"`
	TimeoutExceeded     bool         `yaml:"timeout_exceeded"`
	Workspace           string       `yaml:"workspace"`
	BaseRevision        string       `yaml:"base_revision"`
	FinalSnapshot       string       `yaml:"final_snapshot"`
	InitialSnapshot     string       `yaml:"initial_snapshot"`
	ReportedTokenUsage  *int64       `yaml:"reported_token_usage"`
	AttentionSeconds    *int64       `yaml:"attention_seconds"`
	RepairIterations    int          `yaml:"repair_iterations"`
	ParallelIntegration bool         `yaml:"parallel_integration"`
	Parallel            bool         `yaml:"parallel"`
	Turns               []turnRecord `yaml:"turns"`
}
type helperEvent struct {
	TaskID       string     `json:"task_id"`
	Commands     [][]string `json:"commands"`
	ExitCode     int        `json:"exit_code"`
	Snapshot     string     `json:"snapshot"`
	ActorAttempt int        `json:"actor_attempt"`
}
type evaluation struct {
	Task          string           `yaml:"task"`
	Status        string           `yaml:"status"`
	Checks        []map[string]any `yaml:"checks"`
	OwnerDecision any              `yaml:"owner_decision"`
	Manual        []any            `yaml:"manual"`
}
type checkResult struct {
	Name   string `yaml:"name" json:"name"`
	Status string `yaml:"status" json:"status"`
}
type counts struct {
	Tasks                       int `yaml:"tasks" json:"tasks"`
	ImplementationEligibleTasks int `yaml:"implementation_eligible_tasks" json:"implementation_eligible_tasks"`
	Repairs                     int `yaml:"repairs" json:"repairs"`
	Passed                      int `yaml:"passed" json:"passed"`
	Failed                      int `yaml:"failed" json:"failed"`
	Unknown                     int `yaml:"unknown" json:"unknown"`
	Manual                      int `yaml:"manual" json:"manual"`
	Incomplete                  int `yaml:"incomplete" json:"incomplete"`
	Invalidated                 int `yaml:"invalidated" json:"invalidated"`
	OwnerDecisionRequired       int `yaml:"owner_decision_required" json:"owner_decision_required"`
	UnexpectedOwnerDecision     int `yaml:"unexpected_owner_decision" json:"unexpected_owner_decision"`
}
type churnCoverage struct {
	Tasks       int `yaml:"tasks" json:"tasks"`
	Available   int `yaml:"available" json:"available"`
	Unavailable int `yaml:"unavailable" json:"unavailable"`
}
type churn struct {
	Paths        int   `yaml:"paths" json:"paths"`
	BytesAdded   int64 `yaml:"bytes_added" json:"bytes_added"`
	BytesDeleted int64 `yaml:"bytes_deleted" json:"bytes_deleted"`
	LinesAdded   int   `yaml:"lines_added" json:"lines_added"`
	LinesDeleted int   `yaml:"lines_deleted" json:"lines_deleted"`
}
type helperSummary struct {
	Calls                int    `yaml:"calls" json:"calls"`
	Success              int    `yaml:"success" json:"success"`
	Failure              int    `yaml:"failure" json:"failure"`
	Status               string `yaml:"status" json:"status"`
	BeforeRepairStatus   string `yaml:"before_repair_status" json:"before_repair_status"`
	BeforeRepairSnapshot string `yaml:"before_repair_snapshot,omitempty" json:"before_repair_snapshot,omitempty"`
	FinalStatus          string `yaml:"final_status" json:"final_status"`
	FinalSnapshot        string `yaml:"final_snapshot,omitempty" json:"final_snapshot,omitempty"`
}
type taskReport struct {
	RunID                        string            `yaml:"run_id" json:"run_id"`
	RunKind                      string            `yaml:"run_kind" json:"run_kind"`
	Project                      string            `yaml:"project" json:"project"`
	Arm                          string            `yaml:"arm" json:"arm"`
	Trial                        int               `yaml:"trial" json:"trial"`
	TaskID                       string            `yaml:"task_id" json:"task_id"`
	BaseRevision                 string            `yaml:"base_revision" json:"base_revision"`
	FinalRevision                string            `yaml:"final_revision,omitempty" json:"final_revision,omitempty"`
	ActorStatus                  string            `yaml:"actor_status" json:"actor_status"`
	AggregateEvaluationStatus    string            `yaml:"aggregate_evaluation_status" json:"aggregate_evaluation_status"`
	EvaluationStatus             string            `yaml:"evaluation_status" json:"evaluation_status"`
	EvaluationRecordStatus       string            `yaml:"evaluation_record_status" json:"evaluation_record_status"`
	CountedOutcome               string            `yaml:"counted_outcome" json:"counted_outcome"`
	BeforeRepairEvaluationStatus string            `yaml:"before_repair_evaluation_status" json:"before_repair_evaluation_status"`
	BeforeRepairAggregateStatus  string            `yaml:"before_repair_aggregate_status,omitempty" json:"before_repair_aggregate_status,omitempty"`
	FinalEvaluationStatus        string            `yaml:"final_evaluation_status" json:"final_evaluation_status"`
	EvaluationEvidenceConflicts  []string          `yaml:"evaluation_evidence_conflicts,omitempty" json:"evaluation_evidence_conflicts,omitempty"`
	OwnerDecisionRequired        bool              `yaml:"owner_decision_required" json:"owner_decision_required"`
	OwnerAssessment              string            `yaml:"owner_assessment" json:"owner_assessment"`
	UnexpectedOwnerDecision      bool              `yaml:"unexpected_owner_decision" json:"unexpected_owner_decision"`
	UnexpectedOwnerAssessment    string            `yaml:"unexpected_owner_assessment" json:"unexpected_owner_assessment"`
	Checks                       []checkResult     `yaml:"checks" json:"checks"`
	RawChecks                    []map[string]any  `yaml:"raw_checks" json:"raw_checks"`
	BeforeRepairChecks           []checkResult     `yaml:"before_repair_checks,omitempty" json:"before_repair_checks,omitempty"`
	ExpectedAffected             []string          `yaml:"frozen_expected_affected" json:"frozen_expected_affected"`
	ExpectedAffectedStatus       string            `yaml:"expected_affected_status" json:"expected_affected_status"`
	PublicHelper                 helperSummary     `yaml:"public_helper" json:"public_helper"`
	Repairs                      int               `yaml:"repairs" json:"repairs"`
	ReportedTokens               *int64            `yaml:"reported_tokens,omitempty" json:"reported_tokens,omitempty"`
	NativeTokensStatus           string            `yaml:"native_tokens_status" json:"native_tokens_status"`
	PlatformCallsStatus          string            `yaml:"platform_calls_status" json:"platform_calls_status"`
	ReadAuditStatus              string            `yaml:"read_audit_status" json:"read_audit_status"`
	AttentionSeconds             *int64            `yaml:"attention_seconds,omitempty" json:"attention_seconds,omitempty"`
	AttentionStatus              string            `yaml:"attention_status" json:"attention_status"`
	WorkflowTimeStatus           string            `yaml:"workflow_time_status" json:"workflow_time_status"`
	WorkflowStartedUTC           string            `yaml:"workflow_started_utc,omitempty" json:"workflow_started_utc,omitempty"`
	WorkflowCompletedUTC         string            `yaml:"workflow_completed_utc,omitempty" json:"workflow_completed_utc,omitempty"`
	WorkflowDeadlineUTC          string            `yaml:"workflow_deadline_utc,omitempty" json:"workflow_deadline_utc,omitempty"`
	DeadlineCompliance           string            `yaml:"deadline_compliance" json:"deadline_compliance"`
	ComparisonExclusionReason    string            `yaml:"comparison_exclusion_reason,omitempty" json:"comparison_exclusion_reason,omitempty"`
	ElapsedWorkflowSeconds       *float64          `yaml:"elapsed_workflow_seconds,omitempty" json:"elapsed_workflow_seconds,omitempty"`
	WorkflowTimeUnavailableWhy   string            `yaml:"workflow_time_unavailable_why,omitempty" json:"workflow_time_unavailable_why,omitempty"`
	ContextImpactStatus          string            `yaml:"context_impact_status" json:"context_impact_status"`
	ContextStatus                string            `yaml:"context_status" json:"context_status"`
	ImpactStatus                 string            `yaml:"impact_status" json:"impact_status"`
	ContextEntries               []string          `yaml:"context_entries,omitempty" json:"context_entries,omitempty"`
	ImpactDirectSubjects         []string          `yaml:"impact_direct_subjects,omitempty" json:"impact_direct_subjects,omitempty"`
	ImpactAffectedEntries        []string          `yaml:"impact_affected_entries,omitempty" json:"impact_affected_entries,omitempty"`
	Churn                        map[string]churn  `yaml:"churn" json:"churn"`
	ChurnStatus                  string            `yaml:"churn_status" json:"churn_status"`
	ChurnUnavailableReason       string            `yaml:"churn_unavailable_reason,omitempty" json:"churn_unavailable_reason,omitempty"`
	RawReferences                []string          `yaml:"raw_references" json:"raw_references"`
	RawReferenceSHA256           map[string]string `yaml:"raw_reference_sha256" json:"raw_reference_sha256"`
}
type group struct {
	RunKind        string `yaml:"run_kind" json:"run_kind"`
	RunID          string `yaml:"run_id,omitempty" json:"run_id,omitempty"`
	Project        string `yaml:"project" json:"project"`
	Arm            string `yaml:"arm" json:"arm"`
	Trial          int    `yaml:"trial" json:"trial"`
	Outcomes       counts `yaml:"outcomes" json:"outcomes"`
	StreakStatus   string `yaml:"streak_status" json:"streak_status"`
	PassingStreak  int    `yaml:"passing_streak" json:"passing_streak"`
	StreakBoundary string `yaml:"streak_boundary,omitempty" json:"streak_boundary,omitempty"`
}
type report struct {
	Schema                         string                      `yaml:"schema" json:"schema"`
	Protocol                       string                      `yaml:"protocol" json:"protocol"`
	AnalysisContract               string                      `yaml:"analysis_contract" json:"analysis_contract"`
	AnalysisSourceSHA256           string                      `yaml:"analysis_source_sha256" json:"analysis_source_sha256"`
	FreezeDigest                   string                      `yaml:"freeze_digest" json:"freeze_digest"`
	InvalidationRecordPath         string                      `yaml:"invalidation_record_path,omitempty" json:"invalidation_record_path,omitempty"`
	CohortStatus                   string                      `yaml:"cohort_status" json:"cohort_status"`
	ComparisonEligible             string                      `yaml:"comparison_eligible" json:"comparison_eligible"`
	InvalidationReason             string                      `yaml:"invalidation_reason,omitempty" json:"invalidation_reason,omitempty"`
	InvalidationRecordSHA256       string                      `yaml:"invalidation_record_sha256,omitempty" json:"invalidation_record_sha256,omitempty"`
	RunExclusionRecordStatus       string                      `yaml:"run_exclusion_record_status" json:"run_exclusion_record_status"`
	RunExclusionRecordSHA256       string                      `yaml:"run_exclusion_record_sha256,omitempty" json:"run_exclusion_record_sha256,omitempty"`
	OracleExclusionRecordStatus    string                      `yaml:"oracle_exclusion_record_status" json:"oracle_exclusion_record_status"`
	OracleExclusionRecordSHA256    string                      `yaml:"oracle_exclusion_record_sha256,omitempty" json:"oracle_exclusion_record_sha256,omitempty"`
	Limits                         []string                    `yaml:"limits" json:"limits"`
	Tasks                          []taskReport                `yaml:"tasks" json:"tasks"`
	Groups                         []group                     `yaml:"groups" json:"groups"`
	CountsByProjectArmTrial        map[string]counts           `yaml:"counts_by_project_arm_trial" json:"counts_by_project_arm_trial"`
	ChurnByProjectArmTrial         map[string]map[string]churn `yaml:"churn_by_project_arm_trial" json:"churn_by_project_arm_trial"`
	ChurnCoverageByProjectArmTrial map[string]churnCoverage    `yaml:"churn_coverage_by_project_arm_trial" json:"churn_coverage_by_project_arm_trial"`
}
type invalidation struct {
	FreezeDigest       string `yaml:"freeze_digest"`
	Scope              string `yaml:"scope"`
	Reason             string `yaml:"reason"`
	ComparisonEligible bool   `yaml:"comparison_eligible"`
}

type runExclusion struct {
	RunID  string `yaml:"run_id"`
	Reason string `yaml:"reason"`
}

type runExclusions struct {
	FreezeDigest string         `yaml:"freeze_digest"`
	Runs         []runExclusion `yaml:"runs"`
}

func invalidationApplies(inv invalidation, freezeDigest string) bool {
	return inv.FreezeDigest == freezeDigest && inv.Scope == "all-native-records" && !inv.ComparisonEligible
}

func main() {
	arena := flag.String("arena", "", "external gauntlet arena containing frozen inputs and run records")
	out := flag.String("out", "", "new output directory; must not already exist or be inside the arena")
	flag.Parse()
	if *arena == "" || *out == "" {
		fatal(errors.New("--arena and --out are required"))
	}
	if err := analyze(*arena, *out); err != nil {
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, "analysis:", err); os.Exit(1) }

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func canonicalOutputPath(arena, output string) (string, error) {
	parent, err := filepath.EvalSymlinks(filepath.Dir(output))
	if err != nil {
		return "", fmt.Errorf("resolve output parent: %w", err)
	}
	out := filepath.Join(parent, filepath.Base(output))
	if pathWithin(arena, out) {
		return "", errors.New("output directory must be outside the arena after resolving parent links")
	}
	return out, nil
}

// arenaPath resolves existing links component by component and refuses any path that escapes the canonical arena.
// For not-yet-existing output leaves, only the already-resolved existing parent is trusted.
func arenaPath(arena, rel string) (string, error) {
	root, err := filepath.EvalSymlinks(arena)
	if err != nil {
		return "", err
	}
	if rel == "" || filepath.IsAbs(rel) {
		return "", fmt.Errorf("arena path must be non-empty and relative: %q", rel)
	}
	normalized := strings.ReplaceAll(filepath.ToSlash(rel), `\`, "/")
	if strings.HasPrefix(normalized, "/") || strings.Contains(normalized, ":") {
		return "", fmt.Errorf("arena path must be relative: %q", rel)
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return "", fmt.Errorf("arena path escapes root: %q", rel)
		}
	}
	clean := filepath.Clean(filepath.FromSlash(normalized))
	if clean == "." {
		return root, nil
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("arena path must be relative: %q", rel)
	}
	parts := strings.Split(clean, string(os.PathSeparator))
	cur := root
	for i, part := range parts {
		candidate := filepath.Join(cur, part)
		_, e := os.Lstat(candidate)
		if e == nil {
			resolved, er := filepath.EvalSymlinks(candidate)
			if er != nil {
				return "", fmt.Errorf("resolve arena path %q: %w", rel, er)
			}
			if !pathWithin(root, resolved) {
				return "", fmt.Errorf("arena path resolves outside root: %q", rel)
			}
			cur = resolved
			continue
		}
		if !errors.Is(e, os.ErrNotExist) {
			return "", e
		}
		cur = candidate
		for _, remaining := range parts[i+1:] {
			cur = filepath.Join(cur, remaining)
		}
		if !pathWithin(root, cur) {
			return "", fmt.Errorf("arena path escapes root: %q", rel)
		}
		return cur, nil
	}
	if !pathWithin(root, cur) {
		return "", fmt.Errorf("arena path resolves outside root: %q", rel)
	}
	return cur, nil
}

func safeArenaFile(arena, path string) (string, error) {
	canonical, err := filepath.EvalSymlinks(arena)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(path) {
		root := canonical
		if pathWithin(arena, path) {
			root = arena
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		path = rel
	}
	return arenaPath(canonical, path)
}

func readArenaYAML(arena, path string, target any) error {
	resolved, err := safeArenaFile(arena, path)
	if err != nil {
		return err
	}
	return readYAML(resolved, target)
}

func validComponent(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, `/\:`)
}

func runKind(n nativeRecord) string {
	if n.ParallelIntegration {
		return "parallel-integration"
	}
	if n.Parallel {
		return "parallel-fork"
	}
	return "sequential"
}

func freezeDigest(inputs []freezeInput) string {
	ordered := append([]freezeInput(nil), inputs...)
	sort.Slice(ordered, func(i, j int) bool { return filepath.ToSlash(ordered[i].Path) < filepath.ToSlash(ordered[j].Path) })
	h := sha256.New()
	for _, input := range ordered {
		_, _ = h.Write([]byte(filepath.ToSlash(input.Path) + "\x00" + input.SHA256 + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func analyze(arena, out string) error {
	arena, err := filepath.Abs(arena)
	if err != nil {
		return err
	}
	arena, err = filepath.EvalSymlinks(arena)
	if err != nil {
		return fmt.Errorf("resolve arena root: %w", err)
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	out, err = canonicalOutputPath(arena, out)
	if err != nil {
		return err
	}
	if pathWithin(arena, out) {
		return errors.New("output directory must be outside the arena")
	}
	if _, err = os.Stat(out); err == nil {
		return fmt.Errorf("output already exists: %s", out)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	protocolPath := filepath.Join(arena, "protocol.yaml")
	var protocol struct {
		ID string `yaml:"id"`
	}
	if err = readArenaYAML(arena, protocolPath, &protocol); err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}
	if err = verifyFrozenInputs(arena); err != nil {
		return err
	}
	_, sourcePath, _, _ := runtime.Caller(0)
	sourceSum, err := sourceDigest(sourcePath)
	if err != nil {
		return fmt.Errorf("hash analysis source: %w", err)
	}
	var contract struct {
		ID string `yaml:"id"`
	}
	if err = readArenaYAML(arena, filepath.Join(arena, "analysis-contract.yaml"), &contract); err != nil {
		return fmt.Errorf("read frozen analysis contract: %w", err)
	}
	r := report{Schema: "autonomous-ab-analysis/v2", Protocol: protocol.ID, AnalysisContract: contract.ID, AnalysisSourceSHA256: sourceSum, CohortStatus: "invalidation-not-recorded", ComparisonEligible: "unavailable",
		Limits:                  []string{"Post-freeze descriptive collector; it cannot change assessment criteria.", "No composite winner, byte-token-time-attention equivalence, productivity, or statistical claim.", "Public helper success is reported separately from hidden evaluation correctness.", "Missing, manual, and unknown evaluations never count as passed.", "Tokens, backend calls, read audit, and human attention remain unavailable unless actual retained evidence supplies them.", "Churn is byte and line change volume, not time, attention, tokens, or savings."},
		CountsByProjectArmTrial: map[string]counts{}, ChurnByProjectArmTrial: map[string]map[string]churn{}, ChurnCoverageByProjectArmTrial: map[string]churnCoverage{}}
	var frozen struct {
		Digest string `yaml:"digest"`
	}
	if err = readArenaYAML(arena, filepath.Join(arena, "freeze.yaml"), &frozen); err != nil {
		return err
	}
	invalidationPath := filepath.Join(arena, "decisions", "invalidation.yaml")
	var inv invalidation
	if err = readArenaYAML(arena, invalidationPath, &inv); err == nil {
		if sum, e := hashFile(invalidationPath); e == nil {
			r.InvalidationRecordSHA256 = "sha256:" + sum
		}
		if invalidationApplies(inv, frozen.Digest) {
			r.CohortStatus = "invalidated"
			r.ComparisonEligible = "false"
			r.InvalidationReason = inv.Reason
			r.InvalidationRecordPath = relative(arena, invalidationPath)
		} else {
			r.CohortStatus = "invalidation-record-unmatched"
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read invalidation record: %w", err)
	}
	r.FreezeDigest = frozen.Digest
	excludedRuns, exclusionStatus, exclusionHash, err := readRunExclusions(arena, frozen.Digest)
	if err != nil {
		return err
	}
	r.RunExclusionRecordStatus, r.RunExclusionRecordSHA256 = exclusionStatus, exclusionHash
	exclusionRefs, exclusionHashes := map[string]string{}, map[string]string{}
	for id := range excludedRuns {
		exclusionRefs[id], exclusionHashes[id] = "decisions/run-exclusions.yaml", exclusionHash
	}
	oracleExcluded, oracleStatus, oracleHash, err := readNamedRunExclusions(arena, frozen.Digest, "oracle-run-exclusions.yaml")
	if err != nil {
		return err
	}
	r.OracleExclusionRecordStatus, r.OracleExclusionRecordSHA256 = oracleStatus, oracleHash
	for id, reason := range oracleExcluded {
		if _, duplicate := excludedRuns[id]; duplicate {
			return fmt.Errorf("run %s is excluded by more than one control record", id)
		}
		excludedRuns[id] = reason
		exclusionRefs[id], exclusionHashes[id] = "decisions/oracle-run-exclusions.yaml", oracleHash
	}
	cardCache := map[string]map[string]taskCard{}
	runsRoot, err := arenaPath(arena, "runs")
	if err != nil {
		return fmt.Errorf("resolve runs directory: %w", err)
	}
	files, err := filepath.Glob(filepath.Join(runsRoot, "*", "*.native.yaml"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	seenRuns := map[string]bool{}
	for _, p := range files {
		var n nativeRecord
		if err := readArenaYAML(arena, p, &n); err != nil {
			return fmt.Errorf("%s: %w", relative(arena, p), err)
		}
		if !validComponent(n.Project) || !validComponent(n.RunID) || !validComponent(n.TaskID) || (n.EvaluationTaskID != "" && !validComponent(n.EvaluationTaskID)) {
			return fmt.Errorf("invalid native record identity: %s", p)
		}
		if filepath.Base(filepath.Dir(p)) != n.RunID || filepath.Base(p) != n.TaskID+".native.yaml" {
			return fmt.Errorf("native record identity does not match its path: %s", p)
		}
		seenRuns[n.RunID] = true
		key := n.Project + "/" + strings.ToLower(n.Arm) + "/" + runKind(n)
		if _, ok := cardCache[key]; !ok {
			cards, e := loadCardsForRun(arena, n.Project, strings.ToLower(n.Arm), n.Parallel)
			if e != nil {
				return e
			}
			cardCache[key] = cards
		}
		card, cardOK := cardCache[key][n.TaskID]
		reportTask, err := analyzeTask(arena, n, card, cardOK)
		if err != nil {
			return fmt.Errorf("%s/%s/%s: %w", n.Project, n.Arm, n.TaskID, err)
		}
		groupKey := fmt.Sprintf("%s/%s/trial-%d/%s", n.Project, strings.ToUpper(n.Arm), n.Trial, runKind(n))
		c := r.CountsByProjectArmTrial[groupKey]
		c.Tasks++
		c.Repairs += reportTask.Repairs
		if !reportTask.OwnerDecisionRequired {
			c.ImplementationEligibleTasks++
		}
		outcome := reportTask.CountedOutcome
		if r.CohortStatus == "invalidated" {
			outcome = "invalidated"
			reportTask.CountedOutcome = outcome
		}
		if reason, excluded := excludedRuns[n.RunID]; excluded {
			outcome, reportTask.CountedOutcome = "invalidated", "invalidated"
			reportTask.ComparisonExclusionReason = reason
			ref := exclusionRefs[n.RunID]
			reportTask.RawReferences = append(reportTask.RawReferences, ref)
			reportTask.RawReferenceSHA256[ref] = "sha256:" + exclusionHashes[n.RunID]
		}
		r.Tasks = append(r.Tasks, reportTask)
		countOutcome(&c, outcome, reportTask.OwnerDecisionRequired)
		r.CountsByProjectArmTrial[groupKey] = c
		coverage := r.ChurnCoverageByProjectArmTrial[groupKey]
		coverage.Tasks++
		if reportTask.ChurnStatus == "available" {
			coverage.Available++
		} else {
			coverage.Unavailable++
		}
		r.ChurnCoverageByProjectArmTrial[groupKey] = coverage
		if r.ChurnByProjectArmTrial[groupKey] == nil {
			r.ChurnByProjectArmTrial[groupKey] = map[string]churn{}
		}
		for cat, v := range reportTask.Churn {
			old := r.ChurnByProjectArmTrial[groupKey][cat]
			old.Paths += v.Paths
			old.BytesAdded += v.BytesAdded
			old.BytesDeleted += v.BytesDeleted
			old.LinesAdded += v.LinesAdded
			old.LinesDeleted += v.LinesDeleted
			r.ChurnByProjectArmTrial[groupKey][cat] = old
		}
	}
	missingExcludedRuns := []string{}
	for id := range excludedRuns {
		if !seenRuns[id] {
			missingExcludedRuns = append(missingExcludedRuns, id)
		}
	}
	if len(missingExcludedRuns) != 0 {
		sort.Strings(missingExcludedRuns)
		return fmt.Errorf("run exclusions reference absent native run IDs: %s", strings.Join(missingExcludedRuns, ", "))
	}
	sort.Slice(r.Tasks, func(i, j int) bool {
		a, b := r.Tasks[i], r.Tasks[j]
		if a.Project != b.Project {
			return a.Project < b.Project
		}
		if a.Arm != b.Arm {
			return a.Arm < b.Arm
		}
		if a.Trial != b.Trial {
			return a.Trial < b.Trial
		}
		return taskOrder(a.TaskID) < taskOrder(b.TaskID)
	})
	r.Groups = makeGroups(r.Tasks)
	if len(r.Tasks) == 0 {
		return errors.New("no native task records found")
	}
	if err = os.Mkdir(out, 0700); err != nil {
		return fmt.Errorf("create exclusive output directory: %w", err)
	}
	b, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(out, "report.yaml"), b, 0600); err != nil {
		return err
	}
	return nil
}
func verifyFrozenInputs(arena string) error {
	var f struct {
		Inputs []freezeInput `yaml:"inputs"`
	}
	if err := readArenaYAML(arena, filepath.Join(arena, "freeze.yaml"), &f); err != nil {
		return fmt.Errorf("read freeze manifest: %w", err)
	}
	if len(f.Inputs) == 0 {
		return errors.New("freeze manifest has no inputs")
	}
	seen := map[string]bool{}
	for _, in := range f.Inputs {
		key := filepath.ToSlash(in.Path)
		if seen[key] {
			return fmt.Errorf("duplicate frozen input path: %s", key)
		}
		seen[key] = true
		p, e := arenaPath(arena, in.Path)
		if e != nil {
			return fmt.Errorf("frozen input path %s: %w", in.Path, e)
		}
		got, e := hashFile(p)
		if e != nil {
			return e
		}
		if !strings.EqualFold(got, in.SHA256) {
			return fmt.Errorf("frozen input digest mismatch: %s", in.Path)
		}
	}
	gotDigest := freezeDigest(f.Inputs)
	var manifest struct {
		Digest string `yaml:"digest"`
	}
	if err := readArenaYAML(arena, filepath.Join(arena, "freeze.yaml"), &manifest); err != nil {
		return err
	}
	if !strings.EqualFold(gotDigest, manifest.Digest) {
		return fmt.Errorf("freeze aggregate digest mismatch: calculated %s", gotDigest)
	}
	return nil
}
func loadCards(arena, project, arm string) (map[string]taskCard, error) {
	return loadCardsForRun(arena, project, arm, false)
}
func loadCardsForRun(arena, project, arm string, parallel bool) (map[string]taskCard, error) {
	if !validComponent(project) {
		return nil, errors.New("invalid project identity")
	}
	root := filepath.Join(arena, "projects", project)
	var m projectManifest
	if err := readArenaYAML(arena, filepath.Join(root, "project.yaml"), &m); err != nil {
		return nil, fmt.Errorf("project manifest %s: %w", project, err)
	}
	setPath := filepath.Join(root, "task-set.yaml")
	for _, candidate := range []string{m.Shared.TaskSet, m.Controls.TaskSet, m.TaskSet} {
		if candidate != "" {
			setPath = filepath.Join(root, filepath.FromSlash(candidate))
		}
	}
	if parallel {
		name := m.Controls.ParallelTaskSet
		if name == "" {
			name = "parallel-task-set.yaml"
		}
		setPath = filepath.Join(root, filepath.FromSlash(name))
	}
	var ts taskSet
	if err := readArenaYAML(arena, setPath, &ts); err != nil {
		return nil, fmt.Errorf("task set %s: %w", setPath, err)
	}
	out := map[string]taskCard{}
	for _, c := range ts.Tasks {
		out[c.ID] = c
	}
	return out, nil
}
func analyzeTask(arena string, n nativeRecord, c taskCard, cardOK bool) (taskReport, error) {
	var canonicalErr error
	arena, canonicalErr = filepath.EvalSymlinks(arena)
	if canonicalErr != nil {
		return taskReport{}, canonicalErr
	}
	if !validComponent(n.RunID) || !validComponent(n.TaskID) || !validComponent(n.Project) || (n.EvaluationTaskID != "" && !validComponent(n.EvaluationTaskID)) {
		return taskReport{}, errors.New("invalid task identity")
	}
	t := taskReport{Project: n.Project, Arm: strings.ToUpper(n.Arm), Trial: n.Trial, TaskID: n.TaskID, BaseRevision: n.BaseRevision, ActorStatus: n.ActorStatus, EvaluationStatus: "unknown", CountedOutcome: "unknown", BeforeRepairEvaluationStatus: "unknown", FinalEvaluationStatus: "unknown", NativeTokensStatus: "unavailable", PlatformCallsStatus: "unavailable", ReadAuditStatus: "unavailable", AttentionStatus: "unavailable", OwnerDecisionRequired: cardOK && c.RequiresOwnerDecision, OwnerAssessment: "unavailable", ExpectedAffectedStatus: "available", ContextImpactStatus: "unavailable", ContextStatus: "unavailable", ImpactStatus: "unavailable", Churn: map[string]churn{}, ChurnStatus: "unavailable", RawReferenceSHA256: map[string]string{}, PublicHelper: helperSummary{Status: "unavailable"}}
	t.RunID, t.RunKind = n.RunID, runKind(n)
	t.DeadlineCompliance = deadlineCompliance(n)
	t.WorkflowStartedUTC, t.WorkflowCompletedUTC, t.WorkflowDeadlineUTC = n.ActorStartedUTC, n.ActorCompletedUTC, n.DeadlineUTC
	t.ElapsedWorkflowSeconds, t.WorkflowTimeUnavailableWhy = workflowElapsed(n.ActorStartedUTC, n.ActorCompletedUTC)
	t.WorkflowTimeStatus = "unavailable"
	if t.ElapsedWorkflowSeconds != nil {
		t.WorkflowTimeStatus = "available"
	}
	t.EvaluationRecordStatus = "unavailable"
	t.UnexpectedOwnerAssessment = "unavailable"
	if !cardOK {
		t.ExpectedAffectedStatus = "unavailable"
	} else {
		t.ExpectedAffected = append([]string(nil), c.ExpectedAffected...)
	}
	if n.ReportedTokenUsage != nil {
		v := *n.ReportedTokenUsage
		t.ReportedTokens = &v
		t.NativeTokensStatus = "available"
	}
	if n.AttentionSeconds != nil {
		v := *n.AttentionSeconds
		t.AttentionSeconds = &v
		t.AttentionStatus = "available"
	}
	t.Repairs = n.RepairIterations
	runDir := filepath.Join(arena, "runs", n.RunID)
	t.RawReferences = append(t.RawReferences, relative(arena, filepath.Join(runDir, n.TaskID+".native.yaml")))
	evalTask := n.EvaluationTaskID
	if evalTask == "" {
		evalTask = n.TaskID
	}
	evalPath := filepath.Join(runDir, n.TaskID+".evaluation.yaml")
	var ev evaluation
	if err := readArenaYAML(arena, evalPath, &ev); err == nil {
		t.AggregateEvaluationStatus = ev.Status
		t.EvaluationStatus = normalizeStatus(ev.Status)
		t.EvaluationRecordStatus = "matched"
		if ev.Task != evalTask {
			t.EvaluationRecordStatus = "task-identity-mismatch"
			t.EvaluationStatus = "unknown"
			t.EvaluationEvidenceConflicts = append(t.EvaluationEvidenceConflicts, "evaluator task identity does not match the native evaluation task")
		}
		t.FinalEvaluationStatus = t.EvaluationStatus
		t.RawChecks = ev.Checks
		for index, check := range ev.Checks {
			name := stringValue(check["name"])
			status := checkStatus(check)
			if name == "" {
				name = fmt.Sprintf("unnamed-check-%d", index+1)
			}
			t.Checks = append(t.Checks, checkResult{name, status})
		}
		if t.OwnerDecisionRequired {
			if status := assessmentFile(arena, n); status != "" {
				t.OwnerAssessment = status
			}
		}
		t.RawReferences = append(t.RawReferences, relative(arena, evalPath))
	} else if !errors.Is(err, os.ErrNotExist) {
		return t, err
	}
	beforeEval := filepath.Join(runDir, n.TaskID+".before-repair.evaluation.yaml")
	var before evaluation
	if err := readArenaYAML(arena, beforeEval, &before); err == nil {
		t.BeforeRepairAggregateStatus = before.Status
		t.BeforeRepairEvaluationStatus = normalizeStatus(before.Status)
		if before.Task != evalTask {
			t.BeforeRepairEvaluationStatus = "unknown"
			t.EvaluationEvidenceConflicts = append(t.EvaluationEvidenceConflicts, "before-repair evaluator task identity mismatch")
		}
		for index, check := range before.Checks {
			name := stringValue(check["name"])
			if name == "" {
				name = fmt.Sprintf("unnamed-check-%d", index+1)
			}
			t.BeforeRepairChecks = append(t.BeforeRepairChecks, checkResult{name, checkStatus(check)})
		}
		if t.BeforeRepairEvaluationStatus == "passed" {
			if before.Manual != nil {
				t.BeforeRepairEvaluationStatus = "manual"
			}
			for _, check := range t.BeforeRepairChecks {
				if check.Status != "passed" {
					t.BeforeRepairEvaluationStatus = "incomplete"
					t.EvaluationEvidenceConflicts = append(t.EvaluationEvidenceConflicts, "before-repair evaluation status passed while check "+check.Name+" is "+check.Status)
				}
			}
		}
		t.RawReferences = append(t.RawReferences, relative(arena, beforeEval))
	} else if !errors.Is(err, os.ErrNotExist) {
		return t, err
	}
	if t.OwnerDecisionRequired {
		if t.OwnerAssessment == "unavailable" {
			t.OwnerAssessment = "unavailable"
		}
	}
	if ev.Manual != nil && t.EvaluationStatus == "passed" {
		t.EvaluationStatus = "manual"
	}
	if len(t.Checks) > 0 {
		for _, x := range t.Checks {
			if x.Status == "manual" && t.EvaluationStatus == "passed" {
				t.EvaluationStatus = "manual"
			}
			if x.Status == "unknown" && t.EvaluationStatus == "passed" {
				t.EvaluationStatus = "unknown"
			}
		}
	}
	if cardOK && !c.RequiresOwnerDecision && t.EvaluationStatus == "owner_decision_required" {
		t.UnexpectedOwnerDecision = true
	}
	t.FinalEvaluationStatus = t.EvaluationStatus
	t.CountedOutcome = t.EvaluationStatus
	if ev.Status != "" {
		declared := normalizeStatus(ev.Status)
		for _, check := range t.Checks {
			if declared == "passed" && check.Status != "passed" {
				t.EvaluationEvidenceConflicts = append(t.EvaluationEvidenceConflicts, "evaluation status passed while check "+check.Name+" is "+check.Status)
			}
		}
	}
	if t.EvaluationStatus == "passed" && finalEvidenceConflict(t.EvaluationEvidenceConflicts) {
		t.CountedOutcome = "incomplete"
	}
	basePath := ""
	if n.FinalSnapshot != "" {
		var err error
		basePath, err = arenaPath(arena, filepath.Join(n.FinalSnapshot, "workspace"))
		if err != nil {
			return t, err
		}
	}
	if n.FinalSnapshot == "" {
		basePath = ""
	}
	if basePath != "" {
		if _, e := os.Stat(basePath); e == nil {
			if rev, e := gitRevision(basePath); e == nil {
				t.FinalRevision = rev
			}
			metrics, e := measureChurn(basePath, n.BaseRevision, n.Project, t.Arm)
			if e != nil {
				t.ChurnUnavailableReason = e.Error()
			} else {
				t.Churn = metrics
				t.ChurnStatus = "available"
			}
			t.RawReferences = append(t.RawReferences, relative(arena, filepath.Join(arena, n.FinalSnapshot, "tree.yaml")))
		}
	}
	if err := readContextImpact(arena, n, &t); err != nil {
		return t, err
	}
	if cardOK {
		if scopes, ok := c.AllowedPathsByArm[strings.ToLower(n.Arm)]; ok && len(scopes) > 0 {
			_ = scopes
		} else if len(c.AllowedPaths) > 0 {
			_ = c.AllowedPaths
		}
	}
	helpRoot, err := arenaPath(arena, filepath.Join("raw", n.RunID, n.TaskID))
	if err != nil {
		return t, err
	}
	helpPaths, _ := filepath.Glob(filepath.Join(helpRoot, "helper*.jsonl"))
	if n.ParallelIntegration {
		integrationRoot, err := arenaPath(arena, filepath.Join("raw", n.RunID))
		if err != nil {
			return t, err
		}
		additional, _ := filepath.Glob(filepath.Join(integrationRoot, n.TaskID+".attempt-*.helper.jsonl"))
		helpPaths = append(helpPaths, additional...)
	}
	sort.Strings(helpPaths)
	var events []helperEvent
	seenHelpers := map[string]bool{}
	for _, helpPath := range helpPaths {
		helpPath, err = safeArenaFile(arena, helpPath)
		if err != nil {
			return t, err
		}
		if seenHelpers[helpPath] {
			continue
		}
		seenHelpers[helpPath] = true
		part, e := readHelperEvents(helpPath)
		if e != nil {
			return t, e
		}
		for _, event := range part {
			if event.TaskID != n.TaskID {
				return t, fmt.Errorf("helper task identity mismatch in %s", helpPath)
			}
			if event.Snapshot != "" {
				if _, e := safeArenaFile(arena, event.Snapshot); e != nil {
					return t, e
				}
			}
			events = append(events, event)
		}
		t.RawReferences = append(t.RawReferences, relative(arena, helpPath))
	}
	if len(helpPaths) > 0 {
		t.PublicHelper.Calls = len(events)
		var first, last *helperEvent
		for i := range events {
			event := events[i]
			if event.ExitCode == 0 {
				t.PublicHelper.Success++
			} else {
				t.PublicHelper.Failure++
			}
			if first == nil || event.ActorAttempt < first.ActorAttempt {
				copy := event
				first = &copy
			}
			if last == nil || event.ActorAttempt >= last.ActorAttempt {
				copy := event
				last = &copy
			}
		}
		if first != nil {
			t.PublicHelper.BeforeRepairStatus = helperStatus(first.ExitCode)
			t.PublicHelper.BeforeRepairSnapshot = first.Snapshot
			if ref := snapshotManifestReference(arena, first.Snapshot); ref != "" {
				t.RawReferences = append(t.RawReferences, ref)
			}
		}
		if last != nil {
			t.PublicHelper.FinalStatus = helperStatus(last.ExitCode)
			t.PublicHelper.FinalSnapshot = last.Snapshot
			if ref := snapshotManifestReference(arena, last.Snapshot); ref != "" {
				t.RawReferences = append(t.RawReferences, ref)
			}
			t.PublicHelper.Status = t.PublicHelper.FinalStatus
		}
	}
	if t.EvaluationStatus == "passed" && (t.ActorStatus != "completed" || t.PublicHelper.Status != "success") {
		t.CountedOutcome = "incomplete"
	}
	if t.CountedOutcome == "passed" && t.DeadlineCompliance == "overrun" {
		t.CountedOutcome = "incomplete"
	}
	for _, turn := range n.Turns {
		for _, ref := range []string{turn.ResponseFile, turn.ReceiptsFile} {
			if ref != "" {
				p := filepath.Join(runDir, filepath.Base(ref))
				p, err = safeArenaFile(arena, p)
				if err != nil {
					return t, err
				}
				if _, e := os.Stat(p); e == nil {
					t.RawReferences = append(t.RawReferences, relative(arena, p))
					if strings.HasSuffix(strings.ToLower(ref), ".jsonl") {
						t.ReadAuditStatus = "partial"
					}
				}
			}
		}
	}
	sort.Strings(t.RawReferences)
	for _, ref := range t.RawReferences {
		path, err := arenaPath(arena, ref)
		if err != nil {
			return t, err
		}
		if sum, e := hashFile(path); e == nil {
			t.RawReferenceSHA256[ref] = "sha256:" + sum
		} else {
			t.RawReferenceSHA256[ref] = "unavailable"
		}
	}
	return t, nil
}

func gitRevision(repo string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repo
	b, e := cmd.Output()
	if e != nil {
		return "", e
	}
	return strings.TrimSpace(string(b)), nil
}

func readContextImpact(arena string, n nativeRecord, t *taskReport) error {
	root := filepath.Join(arena, "raw", n.RunID, n.TaskID)
	contextPath, err := safeArenaFile(arena, filepath.Join(root, "context.yaml"))
	if err != nil {
		return err
	}
	impactPath, err := safeArenaFile(arena, filepath.Join(root, "impact.yaml"))
	if err != nil {
		return err
	}
	var ctx map[string]any
	if readYAML(contextPath, &ctx) == nil {
		if entries, ok := normalizedSet(ctx["entries"]); ok {
			t.ContextEntries = entries
			t.ContextStatus = "available"
			t.RawReferences = append(t.RawReferences, relative(arena, contextPath))
		}
	}
	var impact map[string]any
	if readYAML(impactPath, &impact) == nil {
		direct, directOK := normalizedSet(firstField(impact, "directSubjects", "direct_subjects"))
		affected, affectedOK := normalizedSet(firstField(impact, "affectedEntries", "affected_entries"))
		if directOK || affectedOK {
			t.ImpactDirectSubjects = direct
			t.ImpactAffectedEntries = affected
			t.ImpactStatus = "available"
			t.RawReferences = append(t.RawReferences, relative(arena, impactPath))
		}
	}
	if t.ContextStatus == "available" && t.ImpactStatus == "available" {
		t.ContextImpactStatus = "available"
	} else if t.ContextStatus == "available" || t.ImpactStatus == "available" {
		t.ContextImpactStatus = "partial"
	}
	return nil
}
func firstField(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}
func normalizedSet(v any) ([]string, bool) {
	items, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch x := item.(type) {
		case string:
			out = append(out, x)
		case map[string]any:
			value := firstField(x, "key", "id", "name")
			if value == nil {
				return nil, false
			}
			out = append(out, fmt.Sprint(value))
		default:
			return nil, false
		}
	}
	sort.Strings(out)
	return out, true
}
func helperStatus(exitCode int) string {
	if exitCode == 0 {
		return "success"
	}
	return "failed"
}
func snapshotManifestReference(arena, snapshot string) string {
	if snapshot == "" {
		return ""
	}
	p := snapshot
	if !filepath.IsAbs(p) {
		p = filepath.Join(arena, filepath.FromSlash(p))
	}
	abs, err := safeArenaFile(arena, p)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(arena, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return ""
	}
	manifest, err := safeArenaFile(arena, filepath.Join(abs, "tree.yaml"))
	if err != nil {
		return ""
	}
	if _, err := os.Stat(manifest); err != nil {
		return ""
	}
	return filepath.ToSlash(filepath.Join(rel, "tree.yaml"))
}
func assessmentFile(arena string, n nativeRecord) string {
	p := filepath.Join(arena, "decisions", n.RunID, n.TaskID+".yaml")
	var v map[string]any
	if readArenaYAML(arena, p, &v) != nil {
		return ""
	}
	s := strings.ToLower(stringValue(v["assessment_status"]))
	switch s {
	case "correct", "incorrect", "unnecessary", "pending", "unknown":
		return s
	default:
		return ""
	}
}
func normalizeStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "passed", "pass":
		return "passed"
	case "failed", "fail":
		return "failed"
	case "manual", "manual-review-required", "manual_review_required":
		return "manual"
	case "owner_decision_required", "owner-decision-required":
		return "owner_decision_required"
	default:
		return "unknown"
	}
}
func checkStatus(m map[string]any) string {
	if v, ok := m["status"]; ok {
		return normalizeStatus(fmt.Sprint(v))
	}
	if v, ok := m["passed"].(bool); ok {
		if v {
			return "passed"
		}
		return "failed"
	}
	return "unknown"
}
func stringValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func countOutcome(c *counts, s string, decision bool) {
	if s == "invalidated" {
		c.Invalidated++
		return
	}
	if decision {
		c.OwnerDecisionRequired++
		return
	}
	switch s {
	case "owner_decision_required":
		c.UnexpectedOwnerDecision++
		c.Incomplete++
	case "passed":
		c.Passed++
	case "failed":
		c.Failed++
	case "manual":
		c.Manual++
	case "incomplete":
		c.Incomplete++
	default:
		c.Unknown++
	}
}

func finalEvidenceConflict(conflicts []string) bool {
	for _, conflict := range conflicts {
		if !strings.HasPrefix(conflict, "before-repair ") {
			return true
		}
	}
	return false
}

// The harness timestamps span preparation through finish, including queues and repairs.
// This is elapsed workflow time, never agent compute time or human attention.
func workflowElapsed(start, end string) (*float64, string) {
	if start == "" || end == "" {
		return nil, "recorded workflow start or completion is unavailable"
	}
	from, err := time.Parse(time.RFC3339Nano, start)
	if err != nil {
		return nil, "recorded workflow start is not an RFC3339 timestamp"
	}
	to, err := time.Parse(time.RFC3339Nano, end)
	if err != nil {
		return nil, "recorded workflow completion is not an RFC3339 timestamp"
	}
	if to.Before(from) {
		return nil, "recorded workflow completion precedes its start"
	}
	seconds := to.Sub(from).Seconds()
	return &seconds, ""
}

func deadlineCompliance(n nativeRecord) string {
	if n.TimeoutExceeded {
		return "overrun"
	}
	completed, err := time.Parse(time.RFC3339Nano, n.ActorCompletedUTC)
	if err != nil {
		return "unavailable"
	}
	deadline, err := time.Parse(time.RFC3339Nano, n.DeadlineUTC)
	if err != nil {
		return "unavailable"
	}
	if completed.After(deadline) {
		return "overrun"
	}
	return "within-deadline"
}

func readRunExclusions(arena, freeze string) (map[string]string, string, string, error) {
	return readNamedRunExclusions(arena, freeze, "run-exclusions.yaml")
}

func readNamedRunExclusions(arena, freeze, name string) (map[string]string, string, string, error) {
	if name != "run-exclusions.yaml" && name != "oracle-run-exclusions.yaml" {
		return nil, "invalid", "", errors.New("unsupported exclusion control record")
	}
	path := filepath.Join(arena, "decisions", name)
	var record runExclusions
	if err := readArenaYAML(arena, path, &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, "unavailable", "", nil
		}
		return nil, "invalid", "", err
	}
	resolved, err := safeArenaFile(arena, path)
	if err != nil {
		return nil, "invalid", "", err
	}
	hash, err := hashFile(resolved)
	if err != nil {
		return nil, "invalid", "", err
	}
	if record.FreezeDigest != freeze {
		return map[string]string{}, "unmatched-freeze", hash, nil
	}
	excluded := map[string]string{}
	for _, run := range record.Runs {
		if !validComponent(run.RunID) || strings.TrimSpace(run.Reason) == "" {
			return nil, "invalid", hash, errors.New("run exclusion needs an exact run identity and reason")
		}
		if _, duplicate := excluded[run.RunID]; duplicate {
			return nil, "invalid", hash, fmt.Errorf("duplicate excluded run %s", run.RunID)
		}
		excluded[run.RunID] = run.Reason
	}
	return excluded, "matched", hash, nil
}

func makeGroups(tasks []taskReport) []group {
	by := map[string][]taskReport{}
	for _, t := range tasks {
		k := fmt.Sprintf("%s/%s/%d/%s", t.Project, t.Arm, t.Trial, t.RunKind)
		if t.RunKind != "" && t.RunKind != "sequential" {
			k += "/" + t.RunID
		}
		by[k] = append(by[k], t)
	}
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []group{}
	for _, k := range keys {
		list := by[k]
		g := group{Project: list[0].Project, Arm: list[0].Arm, Trial: list[0].Trial, RunKind: list[0].RunKind, StreakStatus: "available"}
		if g.RunKind != "" && g.RunKind != "sequential" {
			g.RunID = list[0].RunID
			g.StreakStatus = "not-longitudinal"
		}
		sort.Slice(list, func(i, j int) bool { return taskOrder(list[i].TaskID) < taskOrder(list[j].TaskID) })
		unknownSeen := false
		for _, t := range list {
			g.Outcomes.Tasks++
			g.Outcomes.Repairs += t.Repairs
			if t.OwnerDecisionRequired && t.CountedOutcome != "invalidated" {
				g.Outcomes.OwnerDecisionRequired++
				continue
			}
			if !t.OwnerDecisionRequired {
				g.Outcomes.ImplementationEligibleTasks++
			}
			switch t.CountedOutcome {
			case "passed":
				if !unknownSeen {
					g.PassingStreak++
				}
				g.Outcomes.Passed++
			case "failed":
				g.Outcomes.Failed++
				if !unknownSeen {
					g.StreakBoundary = "first failure at task " + t.TaskID
				}
				unknownSeen = true
			case "manual":
				g.Outcomes.Manual++
				if !unknownSeen {
					g.StreakBoundary = "manual outcome at task " + t.TaskID
				}
				unknownSeen = true
				g.StreakStatus = "unavailable"
			case "unknown":
				g.Outcomes.Unknown++
				if !unknownSeen {
					g.StreakBoundary = "unknown outcome at task " + t.TaskID
				}
				unknownSeen = true
				g.StreakStatus = "unavailable"
			case "incomplete":
				g.Outcomes.Incomplete++
				if !unknownSeen {
					g.StreakBoundary = "incomplete evidence at task " + t.TaskID
				}
				unknownSeen = true
				g.StreakStatus = "unavailable"
			case "invalidated":
				g.Outcomes.Invalidated++
				if !unknownSeen {
					g.StreakBoundary = "invalidated cohort at task " + t.TaskID
				}
				unknownSeen = true
				g.StreakStatus = "unavailable"
			case "owner_decision_required":
				g.Outcomes.UnexpectedOwnerDecision++
				g.Outcomes.Incomplete++
				if !unknownSeen {
					g.StreakBoundary = "unexpected escalation at task " + t.TaskID
				}
				unknownSeen = true
				g.StreakStatus = "unavailable"
			default:
				g.Outcomes.Unknown++
				if !unknownSeen {
					g.StreakBoundary = "non-implementation outcome at task " + t.TaskID
				}
				unknownSeen = true
				g.StreakStatus = "unavailable"
			}
		}
		if unknownSeen && g.StreakStatus == "available" {
			g.StreakStatus = "bounded-by-failure"
		}
		if g.RunKind != "" && g.RunKind != "sequential" {
			g.StreakStatus = "not-longitudinal"
			g.PassingStreak = 0
			g.StreakBoundary = ""
		}
		out = append(out, g)
	}
	return out
}
func readHelperEvents(path string) ([]helperEvent, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	var out []helperEvent
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 8*1024*1024)
	for s.Scan() {
		var v helperEvent
		if err := json.Unmarshal(s.Bytes(), &v); err != nil {
			return nil, fmt.Errorf("malformed helper receipt %s: %w", path, err)
		}
		out = append(out, v)
	}
	return out, s.Err()
}
func measureChurn(repo, base, project, arm string) (map[string]churn, error) {
	if base == "" {
		return nil, errors.New("base revision unavailable; cannot calculate churn")
	}
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", base)
	cmd.Dir = repo
	b, e := cmd.Output()
	if e != nil {
		return nil, fmt.Errorf("list task-start commit: %w", e)
	}
	before := map[string][]byte{}
	for _, p := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if p == "" {
			continue
		}
		if excludedCapturedPath(project, p) {
			continue
		}
		show := exec.Command("git", "show", base+":"+p)
		show.Dir = repo
		data, er := show.Output()
		if er != nil {
			return nil, fmt.Errorf("read task-start bytes %s: %w", p, er)
		}
		before[filepath.ToSlash(p)] = data
	}
	after := map[string][]byte{}
	err := filepath.WalkDir(repo, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symlink entry prevents exact captured-byte comparison: %s", p)
		}
		rel, er := filepath.Rel(repo, p)
		if er != nil {
			return er
		}
		if rel == "." {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if d.IsDir() && excludedCapturedPath(project, slash) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if excludedCapturedPath(project, slash) {
			return nil
		}
		data, er := os.ReadFile(p)
		if er != nil {
			return er
		}
		after[slash] = data
		return nil
	})
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{}
	for p := range before {
		paths[p] = true
	}
	for p := range after {
		paths[p] = true
	}
	out := map[string]churn{}
	for p := range paths {
		x, y := before[p], after[p]
		if bytes.Equal(x, y) {
			continue
		}
		category := categoryFor(project, arm, p)
		v := out[category]
		v.Paths++
		ba, bd, la, ld := lineDelta(x, y)
		v.BytesAdded += ba
		v.BytesDeleted += bd
		v.LinesAdded += la
		v.LinesDeleted += ld
		out[category] = v
	}
	return out, nil
}
func excludedCapturedPath(project, path string) bool {
	p := strings.ToLower(filepath.ToSlash(path))
	if p == ".git" || strings.HasPrefix(p, ".git/") {
		return true
	}
	if project == "vertical-slices" {
		for _, root := range []string{"src/commerce/bin", "src/commerce/obj", "checks/bin", "checks/obj"} {
			if p == root || strings.HasPrefix(p, root+"/") {
				return true
			}
		}
	}
	return false
}

// These path-owner rules are explicit experiment-seed mappings. They classify paths only; they do not infer file meaning.
func categoryFor(project, arm, p string) string {
	base := filepath.Base(p)
	low := strings.ToLower(p)
	if inAny(p, ".markitect/exceptions/", "resources/exceptions/", "constitution/exceptions/") {
		return "exceptions"
	}
	if strings.HasSuffix(low, "_test.go") || strings.HasSuffix(low, "tests.cs") || strings.Contains(low, "/tests/") || strings.Contains(low, "/test/") || strings.HasPrefix(low, "testdata/") || strings.HasSuffix(low, "check_operations.py") {
		return "tests"
	}
	if project == "vertical-slices" && arm == "B" && inAny(p, ".agents/", "docs/markitect/") {
		return "generated-projections"
	}
	if project == "engineering-ops" && arm == "B" && inAny(p, ".agents/", ".claude/", "docs/markitect/") {
		return "generated-projections"
	}
	if project == "modular-service" && arm == "B" && inAny(p, ".agents/", "docs/markitect/") {
		return "generated-projections"
	}
	if project == "vertical-slices" && arm == "B" && (base == "markitect-artifacts.yaml" || p == "markitect.yaml" || inAny(p, ".github/workflows/") || p == "Directory.Build.props" || p == "NuGet.Config" || p == ".gitignore" || p == "global.json") {
		return "check-configuration"
	}
	if project == "engineering-ops" && arm == "B" && (base == "markitect-artifacts.yaml" || p == "markitect.yaml" || strings.HasPrefix(p, ".markitect/modules/")) {
		return "check-configuration"
	}
	if project == "modular-service" && arm == "B" && (p == "markitect.yaml" || p == ".markitect/coverage.yaml" || inAny(p, ".github/workflows/")) {
		return "check-configuration"
	}
	if project == "engineering-ops" && arm == "B" && (p == "go.mod" || inAny(p, ".github/workflows/")) {
		return "check-configuration"
	}
	if isGovernance(project, arm, p) {
		return "authored-governance"
	}
	if strings.HasPrefix(p, "scripts/") && strings.HasSuffix(p, ".py") {
		return "check-configuration"
	}
	if isImplementation(project, p) {
		return "implementation"
	}
	return "other"
}
func inAny(p string, prefixes ...string) bool {
	for _, x := range prefixes {
		if strings.HasPrefix(p, x) {
			return true
		}
	}
	return false
}
func isGovernance(project, arm, p string) bool {
	if project == "modular-service" && arm == "B" {
		return strings.HasPrefix(p, "resources/") || strings.HasPrefix(p, ".markitect/areas/")
	}
	if project == "vertical-slices" && arm == "B" {
		return strings.HasPrefix(p, "resources/") || strings.HasPrefix(p, ".markitect/areas/") || strings.HasPrefix(p, ".markitect/packages/")
	}
	if project == "engineering-ops" && arm == "B" {
		return strings.HasPrefix(p, ".markitect/areas/") || strings.HasPrefix(p, ".markitect/packages/")
	}
	if strings.HasSuffix(strings.ToLower(p), ".md") || p == "AGENTS.md" {
		return true
	}
	return false
}
func isImplementation(project, p string) bool {
	switch project {
	case "vertical-slices":
		return strings.HasPrefix(p, "src/Commerce/")
	case "modular-service":
		return inAny(p, "cmd/", "internal/")
	case "engineering-ops":
		return inAny(p, "cmd/", "tools/process-sentinel/", "internal/config/")
	}
	return false
}
func lineDelta(a, b []byte) (int64, int64, int, int) {
	x, y := splitLines(a), splitLines(b)
	n, m := len(x), len(y)
	if n == 0 {
		var size int64
		for _, line := range y {
			size += int64(len(line))
		}
		return size, 0, m, 0
	}
	if m == 0 {
		var size int64
		for _, line := range x {
			size += int64(len(line))
		}
		return 0, size, 0, n
	}
	v := map[int]int{1: 0}
	trace := make([]map[int]int, 0, n+m+1)
	for d := 0; d <= n+m; d++ {
		snap := map[int]int{}
		for k, z := range v {
			snap[k] = z
		}
		trace = append(trace, snap)
		for k := -d; k <= d; k += 2 {
			var xx int
			if k == -d || (k != d && v[k-1] < v[k+1]) {
				xx = v[k+1]
			} else {
				xx = v[k-1] + 1
			}
			yy := xx - k
			for xx < n && yy < m && bytes.Equal(x[xx], y[yy]) {
				xx++
				yy++
			}
			v[k] = xx
			if xx >= n && yy >= m {
				return editDelta(trace, x, y, d)
			}
		}
	}
	return 0, 0, 0, 0
}
func editDelta(trace []map[int]int, a, b [][]byte, dmax int) (int64, int64, int, int) {
	x, y := len(a), len(b)
	ins, del := 0, 0
	var bytesAdded, bytesDeleted int64
	for d := dmax; d > 0; d-- {
		v := trace[d]
		k := x - y
		var prevK int
		var added bool
		if k == -d || (k != d && v[k-1] < v[k+1]) {
			prevK = k + 1
			added = true
		} else {
			prevK = k - 1
		}
		prevX := v[prevK]
		prevY := prevX - prevK
		if added {
			ins++
			bytesAdded += int64(len(b[prevY]))
		} else {
			del++
			bytesDeleted += int64(len(a[prevX]))
		}
		for x > prevX && y > prevY {
			x--
			y--
		}
		x, y = prevX, prevY
	}
	return bytesAdded, bytesDeleted, ins, del
}
func splitLines(b []byte) [][]byte {
	if len(b) == 0 {
		return nil
	}
	parts := bytes.SplitAfter(b, []byte("\n"))
	return parts
}
func readYAML(p string, v any) error {
	b, e := os.ReadFile(p)
	if e != nil {
		return e
	}
	return yaml.Unmarshal(b, v)
}
func hashFile(p string) (string, error) {
	b, e := os.ReadFile(p)
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func sourceDigest(entry string) (string, error) {
	files, err := filepath.Glob(filepath.Join(filepath.Dir(entry), "*.go"))
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, p := range files {
		data, e := os.ReadFile(p)
		if e != nil {
			return "", e
		}
		_, _ = h.Write([]byte(filepath.Base(p) + "\x00"))
		_, _ = h.Write(data)
		_, _ = h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func relative(root, p string) string {
	r, e := filepath.Rel(root, p)
	if e != nil {
		return p
	}
	return filepath.ToSlash(r)
}
func taskOrder(s string) int {
	var n int
	if _, e := fmt.Sscanf(s, "%d", &n); e != nil {
		return 999
	}
	return n
}
