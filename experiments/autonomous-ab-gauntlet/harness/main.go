// Command gauntlet runs the frozen, paired autonomous engineering exercise.
// It deliberately stores prompts and observations outside actor workspaces.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

type taskSet struct {
	Project        string              `yaml:"project" json:"project"`
	Evaluator      EvaluatorConfig     `yaml:"evaluator" json:"evaluator"`
	Validator      []string            `yaml:"validator" json:"validator"`
	ValidatorByArm map[string][]string `yaml:"validator_by_arm" json:"validator_by_arm"`
	BaselineChecks [][]string          `yaml:"baseline_checks" json:"baseline_checks"`
	TimeoutSeconds int                 `yaml:"timeout_seconds" json:"timeout_seconds"`
	ForkFrom       struct {
		MainTaskID        string `yaml:"main_task_id"`
		MainTrial         int    `yaml:"main_trial"`
		PriorTasksThrough string `yaml:"prior_tasks_through"`
	} `yaml:"fork_from" json:"fork_from"`
	Tasks []Task `yaml:"tasks" json:"tasks"`
}
type Task struct {
	ID                     string              `yaml:"id" json:"id"`
	Prompt                 string              `yaml:"prompt" json:"prompt"`
	Category               string              `yaml:"category" json:"category"`
	ExpectedClassification string              `yaml:"expected_classification" json:"expected_classification"`
	RequiresOwnerDecision  bool                `yaml:"requires_owner_decision" json:"requires_owner_decision"`
	AllowedPaths           []string            `yaml:"allowed_paths" json:"allowed_paths"`
	AllowedPathsByArm      map[string][]string `yaml:"allowed_paths_by_arm" json:"allowed_paths_by_arm"`
	ExpectedAffected       []string            `yaml:"expected_affected" json:"expected_affected"`
	Validator              []string            `yaml:"validator" json:"validator"`
	TimeoutSeconds         int                 `yaml:"timeout_seconds" json:"timeout_seconds"`
	OriginTaskID           string              `yaml:"origin_task_id" json:"origin_task_id"`
	PriorTasksThrough      string              `yaml:"prior_tasks_through" json:"prior_tasks_through"`
	PublicValidators       [][]string          `yaml:"-" json:"-"`
	ValidatorManifest      string              `yaml:"-" json:"-"`
}
type Protocol struct {
	Agent     AgentSettings     `yaml:"agent"`
	Execution ExecutionSettings `yaml:"execution"`
	Projects  []ProjectRef      `yaml:"projects"`
}
type ProjectRef struct {
	ID         string                `yaml:"id"`
	Path       string                `yaml:"path"`
	Arms       map[string]ArmConfig  `yaml:"arms"`
	A          string                `yaml:"a"`
	B          string                `yaml:"b"`
	Evaluator  string                `yaml:"evaluator"`
	Validators map[string][][]string `yaml:"validators"`
}
type ArmConfig struct {
	Path           string     `yaml:"path"`
	BaseRevision   string     `yaml:"base_revision"`
	Validator      [][]string `yaml:"validator"`
	FullAcceptance [][]string `yaml:"full_acceptance"`
}

func (a *ArmConfig) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		a.Path = n.Value
		return nil
	}
	type plain ArmConfig
	return n.Decode((*plain)(a))
}

type ProjectManifest struct {
	ID       string               `yaml:"id"`
	Arms     map[string]ArmConfig `yaml:"arms"`
	TaskSet  string               `yaml:"task_set"`
	Controls struct {
		TaskSet string `yaml:"task_set"`
	} `yaml:"controls"`
	Shared struct {
		TaskSet string `yaml:"taskSet"`
	} `yaml:"shared"`
	Evaluator EvaluatorConfig `yaml:"evaluator"`
}

type EvaluatorConfig struct {
	Command []string `yaml:"command"`
}

func (e *EvaluatorConfig) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		e.Command = []string{n.Value}
		return nil
	case yaml.SequenceNode:
		return n.Decode(&e.Command)
	case yaml.MappingNode:
		type plain EvaluatorConfig
		return n.Decode((*plain)(e))
	default:
		return fmt.Errorf("unsupported evaluator configuration")
	}
}

type ValidatorManifest struct {
	BaseRevision string     `yaml:"base_revision"`
	Commands     [][]string `yaml:"commands"`
}

func (p *ProjectRef) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		p.ID = n.Value
		return nil
	}
	type plain ProjectRef
	return n.Decode((*plain)(p))
}

type AgentSettings struct {
	Runtime             string `yaml:"runtime"`
	RuntimeVersion      string `yaml:"runtime_version"`
	Model               string `yaml:"model"`
	ReasoningEffort     string `yaml:"reasoning_effort"`
	Sandbox             string `yaml:"sandbox"`
	ApprovalPolicy      string `yaml:"approval_policy"`
	WebSearch           string `yaml:"web_search"`
	IgnoreUserConfig    bool   `yaml:"ignore_user_config"`
	FreshProcessPerTask bool   `yaml:"fresh_process_per_task"`
	Ephemeral           bool   `yaml:"ephemeral"`
}
type ExecutionSettings struct {
	TrialsPerArm                int    `yaml:"trials_per_arm"`
	TasksPerProject             int    `yaml:"tasks_per_project"`
	MaximumConcurrentActors     int    `yaml:"maximum_concurrent_actors"`
	InitialValidationAttempts   int    `yaml:"initial_validation_attempts"`
	MaximumSelfRepairIterations int    `yaml:"maximum_self_repair_iterations"`
	TaskTimeoutSeconds          int    `yaml:"task_timeout_seconds"`
	ArmOrder                    string `yaml:"arm_order"`
	Continuation                string `yaml:"continuation"`
	BudgetViolation             string `yaml:"budget_violation"`
}
type FileDigest struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
}
type FreezeRecord struct {
	Version    int          `yaml:"version"`
	CreatedUTC string       `yaml:"created_utc"`
	Inputs     []FileDigest `yaml:"inputs"`
	Digest     string       `yaml:"digest"`
}

func verifyFreeze(arena string, r FreezeRecord) error {
	want := make(map[string]bool, len(r.Inputs))
	for _, input := range r.Inputs {
		want[filepath.ToSlash(input.Path)] = true
		actual, err := hashFile(filepath.Join(arena, filepath.FromSlash(input.Path)))
		if err != nil {
			return fmt.Errorf("frozen input %s: %w", input.Path, err)
		}
		if actual != input.SHA256 {
			return fmt.Errorf("frozen input changed: %s", input.Path)
		}
	}
	seen := 0
	err := filepath.WalkDir(arena, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(arena, p)
		if d.IsDir() {
			if rel != "." && frozenMutableDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "freeze.yaml" {
			return nil
		}
		if !want[rel] {
			return fmt.Errorf("new file after freeze: %s", rel)
		}
		seen++
		return nil
	})
	if err != nil {
		return err
	}
	if seen != len(want) {
		return fmt.Errorf("frozen input set changed: expected %d, found %d", len(want), seen)
	}
	return nil
}
func frozenMutableDir(rel string) bool {
	first := strings.Split(filepath.ToSlash(rel), "/")[0]
	return first == "runs" || first == "raw" || first == "snapshots" || first == "results" || first == "decisions"
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "prepare":
		err = prepare(os.Args[2:])
	case "freeze":
		err = freeze(os.Args[2:])
	case "smoke":
		err = run(os.Args[2:], true)
	case "check":
		err = publicCheck(os.Args[2:])
	case "prepare-task":
		err = prepareTask(os.Args[2:])
	case "prepare-integration-repair":
		err = prepareIntegrationRepair(os.Args[2:])
	case "finish-task":
		err = finishTask(os.Args[2:])
	case "integrate-parallel":
		err = integrateParallel(os.Args[2:])
	case "record-holdout-decision":
		err = recordHoldoutDecision(os.Args[2:])
	case "run":
		err = run(os.Args[2:], false)
	case "collect":
		err = collect(os.Args[2:])
	case "evaluate":
		err = evaluate(os.Args[2:])
	case "summarize":
		err = summarize(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gauntlet:", err)
		os.Exit(1)
	}
}
func usage() {
	fmt.Fprintln(os.Stderr, "usage: gauntlet {prepare|freeze|smoke|check|prepare-task|prepare-integration-repair|finish-task|integrate-parallel|record-holdout-decision|run|collect|evaluate|summarize} [flags]")
}
func mustFlags(name string, args []string, setup func(*flag.FlagSet)) (*flag.FlagSet, error) {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	setup(f)
	return f, f.Parse(args)
}
func readYAML(path string, out any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return yaml.Unmarshal(b, out)
}
func writeYAMLNew(path string, v any) error {
	b, e := yaml.Marshal(v)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return e
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = f.Write(b)
	return e
}
func writeYAMLReplace(path string, v any) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".pending-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
func hashFile(path string) (string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func prepare(args []string) error {
	var arena string
	f, e := mustFlags("prepare", args, func(f *flag.FlagSet) { f.StringVar(&arena, "arena", "", "external arena root") })
	if e != nil {
		return e
	}
	_ = f
	if arena == "" {
		return errors.New("--arena is required")
	}
	for _, d := range []string{"runs", "raw", "snapshots", "results", "decisions"} {
		if e = os.MkdirAll(filepath.Join(arena, d), 0755); e != nil {
			return e
		}
	}
	fmt.Println("prepared", arena)
	return nil
}
func freeze(args []string) error {
	var arena, out string
	f, e := mustFlags("freeze", args, func(f *flag.FlagSet) {
		f.StringVar(&arena, "arena", "", "arena root")
		f.StringVar(&out, "out", "", "freeze record path")
	})
	if e != nil {
		return e
	}
	_ = f
	if arena == "" {
		return errors.New("--arena required")
	}
	if out == "" {
		out = filepath.Join(arena, "freeze.yaml")
	}
	var files []string
	e = filepath.WalkDir(arena, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(arena, p)
			if p != arena && frozenMutableDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(arena, p)
		if rel == "freeze.yaml" || frozenMutableDir(rel) {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if e != nil {
		return e
	}
	sort.Strings(files)
	r := FreezeRecord{Version: 1, CreatedUTC: time.Now().UTC().Format(time.RFC3339)}
	h := sha256.New()
	for _, p := range files {
		rel, _ := filepath.Rel(arena, p)
		sum, er := hashFile(p)
		if er != nil {
			return er
		}
		r.Inputs = append(r.Inputs, FileDigest{filepath.ToSlash(rel), sum})
		io.WriteString(h, filepath.ToSlash(rel)+"\x00"+sum+"\n")
	}
	r.Digest = hex.EncodeToString(h.Sum(nil))
	if e = writeYAMLNew(out, r); e != nil {
		return fmt.Errorf("freeze is write-once: %w", e)
	}
	fmt.Println(r.Digest)
	return nil
}

type taskResult struct {
	TaskID                string   `yaml:"task_id" json:"task_id"`
	Status                string   `yaml:"status" json:"status"`
	StartedUTC            string   `yaml:"started_utc" json:"started_utc"`
	EndedUTC              string   `yaml:"ended_utc" json:"ended_utc"`
	ExitCode              int      `yaml:"exit_code" json:"exit_code"`
	ValidatorExit         *int     `yaml:"validator_exit,omitempty" json:"validator_exit,omitempty"`
	Error                 string   `yaml:"error,omitempty" json:"error,omitempty"`
	Repairs               int      `yaml:"repairs" json:"repairs"`
	Output                string   `yaml:"output,omitempty" json:"output,omitempty"`
	StdoutFiles           []string `yaml:"stdout_files" json:"stdout_files"`
	ReportedTokens        *int64   `yaml:"reported_tokens,omitempty" json:"reported_tokens,omitempty"`
	ValidationCalls       int      `yaml:"validation_calls" json:"validation_calls"`
	BeforeRepairAvailable bool     `yaml:"before_repair_available" json:"before_repair_available"`
	ProtocolValid         bool     `yaml:"protocol_valid" json:"protocol_valid"`
}
type RunRecord struct {
	Version         int          `yaml:"version" json:"version"`
	RunID           string       `yaml:"run_id" json:"run_id"`
	Project         string       `yaml:"project" json:"project"`
	Variant         string       `yaml:"variant" json:"variant"`
	Trial           int          `yaml:"trial" json:"trial"`
	Model           string       `yaml:"model" json:"model"`
	ReasoningEffort string       `yaml:"reasoning_effort" json:"reasoning_effort"`
	FreezeDigest    string       `yaml:"freeze_digest" json:"freeze_digest"`
	StartedUTC      string       `yaml:"started_utc" json:"started_utc"`
	EndedUTC        string       `yaml:"ended_utc" json:"ended_utc"`
	Tasks           []taskResult `yaml:"tasks" json:"tasks"`
}

type NativeEnvelope struct {
	Version                     int                 `yaml:"version" json:"version"`
	RunID                       string              `yaml:"run_id" json:"run_id"`
	Project                     string              `yaml:"project" json:"project"`
	Arm                         string              `yaml:"arm" json:"arm"`
	Trial                       int                 `yaml:"trial" json:"trial"`
	TaskID                      string              `yaml:"task_id" json:"task_id"`
	EvaluationTaskID            string              `yaml:"evaluation_task_id,omitempty" json:"evaluation_task_id,omitempty"`
	Workspace                   string              `yaml:"workspace" json:"workspace"`
	Prompt                      string              `yaml:"prompt" json:"prompt"`
	ActorPrompt                 string              `yaml:"actor_prompt" json:"actor_prompt"`
	AllowedPaths                []string            `yaml:"allowed_paths" json:"allowed_paths"`
	PublicValidators            [][]string          `yaml:"public_validators" json:"public_validators"`
	ValidatorManifest           string              `yaml:"validator_manifest" json:"validator_manifest"`
	PublicCheckInstruction      string              `yaml:"public_check_instruction" json:"public_check_instruction"`
	HelperCommand               string              `yaml:"helper_command" json:"helper_command"`
	HelperEnvironment           map[string]string   `yaml:"helper_environment" json:"helper_environment"`
	HelperRecord                string              `yaml:"helper_record" json:"helper_record"`
	InputDigest                 string              `yaml:"input_digest" json:"input_digest"`
	Model                       string              `yaml:"protocol_model" json:"protocol_model"`
	ReasoningEffort             string              `yaml:"protocol_reasoning_effort" json:"protocol_reasoning_effort"`
	DeadlineUTC                 string              `yaml:"deadline_utc" json:"deadline_utc"`
	IntegrationTaskIDs          []string            `yaml:"integration_task_ids,omitempty" json:"integration_task_ids,omitempty"`
	IntegrationPlan             []IntegrationCommit `yaml:"integration_plan,omitempty" json:"integration_plan,omitempty"`
	IntegrationRemainingCommits []IntegrationCommit `yaml:"integration_remaining_commits,omitempty" json:"integration_remaining_commits,omitempty"`
}
type IntegrationCommit struct {
	TaskID string `yaml:"task_id" json:"task_id"`
	Commit string `yaml:"commit" json:"commit"`
}
type NativeTaskRecord struct {
	Version                   int                 `yaml:"version" json:"version"`
	RunID                     string              `yaml:"run_id" json:"run_id"`
	Project                   string              `yaml:"project" json:"project"`
	Arm                       string              `yaml:"arm" json:"arm"`
	Trial                     int                 `yaml:"trial" json:"trial"`
	TaskID                    string              `yaml:"task_id" json:"task_id"`
	EvaluationTaskID          string              `yaml:"evaluation_task_id,omitempty" json:"evaluation_task_id,omitempty"`
	PriorTasksThrough         string              `yaml:"prior_tasks_through,omitempty" json:"prior_tasks_through,omitempty"`
	Parallel                  bool                `yaml:"parallel,omitempty" json:"parallel,omitempty"`
	InputDigest               string              `yaml:"input_digest" json:"input_digest"`
	ActorStartedUTC           string              `yaml:"actor_started_utc" json:"actor_started_utc"`
	ActorCompletedUTC         string              `yaml:"actor_completed_utc,omitempty" json:"actor_completed_utc,omitempty"`
	ActorStatus               string              `yaml:"actor_status" json:"actor_status"`
	DeadlineUTC               string              `yaml:"deadline_utc" json:"deadline_utc"`
	TimeoutExceeded           bool                `yaml:"timeout_exceeded,omitempty" json:"timeout_exceeded,omitempty"`
	Workspace                 string              `yaml:"workspace" json:"workspace"`
	ResponseFile              string              `yaml:"response_file,omitempty" json:"response_file,omitempty"`
	ReceiptsFile              string              `yaml:"receipts_file,omitempty" json:"receipts_file,omitempty"`
	FinalSnapshot             string              `yaml:"final_snapshot,omitempty" json:"final_snapshot,omitempty"`
	ReportedTokenUsage        *int64              `yaml:"reported_token_usage,omitempty" json:"reported_token_usage,omitempty"`
	AttentionSeconds          *int64              `yaml:"attention_seconds,omitempty" json:"attention_seconds,omitempty"`
	RepairIterations          int                 `yaml:"repair_iterations" json:"repair_iterations"`
	ActiveAttempt             int                 `yaml:"active_attempt" json:"active_attempt"`
	LastTurnDigest            string              `yaml:"last_turn_digest" json:"last_turn_digest"`
	Turns                     []NativeTurnRecord  `yaml:"turns" json:"turns"`
	SeedRevision              string              `yaml:"seed_revision" json:"seed_revision"`
	BaseRevision              string              `yaml:"base_revision" json:"base_revision"`
	InitialSnapshot           string              `yaml:"initial_snapshot,omitempty" json:"initial_snapshot,omitempty"`
	ParallelIntegration       bool                `yaml:"parallel_integration,omitempty" json:"parallel_integration,omitempty"`
	MaximumRepairIterations   int                 `yaml:"maximum_repair_iterations,omitempty" json:"maximum_repair_iterations,omitempty"`
	IntegrationPlanPath       string              `yaml:"integration_plan_path,omitempty" json:"integration_plan_path,omitempty"`
	AppliedIntegrationCommits []IntegrationCommit `yaml:"applied_integration_commits,omitempty" json:"applied_integration_commits,omitempty"`
}
type NativeTurnRecord struct {
	Attempt           int    `yaml:"attempt" json:"attempt"`
	InputDigest       string `yaml:"input_digest" json:"input_digest"`
	ActorStartedUTC   string `yaml:"actor_started_utc" json:"actor_started_utc"`
	ActorCompletedUTC string `yaml:"actor_completed_utc,omitempty" json:"actor_completed_utc,omitempty"`
	Status            string `yaml:"status" json:"status"`
	ResponseFile      string `yaml:"response_file,omitempty" json:"response_file,omitempty"`
	ReceiptsFile      string `yaml:"receipts_file,omitempty" json:"receipts_file,omitempty"`
	CompletedDigest   string `yaml:"completed_digest,omitempty" json:"completed_digest,omitempty"`
}
type HoldoutDecision struct {
	Decision    string `yaml:"decision"`
	Rationale   string `yaml:"rationale"`
	RecordedUTC string `yaml:"recorded_utc"`
	Candidate   string `yaml:"candidate"`
}
type SeedRecord struct {
	SeedRevision string `yaml:"seed_revision"`
}

func prepareTask(args []string) error {
	var arena, project, arm, protocolPath, taskID, feedbackPath, taskSetName, forkFromTask string
	var trial, repairIteration int
	var parallel bool
	f, e := mustFlags("prepare-task", args, func(f *flag.FlagSet) {
		f.StringVar(&arena, "arena", "", "arena root")
		f.StringVar(&project, "project", "", "project name")
		f.StringVar(&arm, "arm", "", "A or B")
		f.IntVar(&trial, "trial", 1, "trial number")
		f.StringVar(&protocolPath, "protocol", "", "protocol YAML")
		f.StringVar(&taskID, "task-id", "", "current task ID")
		f.IntVar(&repairIteration, "repair-iteration", 0, "fresh repair actor turn, 1 or 2")
		f.StringVar(&feedbackPath, "feedback-file", "", "saved public validator output")
		f.StringVar(&taskSetName, "task-set", "", "alternate frozen task set path relative to project")
		f.BoolVar(&parallel, "parallel", false, "prepare an isolated parallel fork")
		f.StringVar(&forkFromTask, "fork-from-task", "", "completed main-chain task used as the fork base")
	})
	if e != nil {
		return e
	}
	_ = f
	if arena == "" || project == "" || arm == "" || protocolPath == "" || taskID == "" {
		return errors.New("prepare-task requires --arena --project --arm --protocol --task-id")
	}
	arm = strings.ToLower(strings.TrimPrefix(arm, "arm-"))
	if arm != "a" && arm != "b" {
		return errors.New("--arm must be A or B")
	}
	if trial < 1 {
		return errors.New("--trial must be positive")
	}
	var protocol Protocol
	if e = readYAML(protocolPath, &protocol); e != nil {
		return e
	}
	if protocol.Execution.TaskTimeoutSeconds <= 0 {
		return errors.New("frozen task_timeout_seconds must be positive")
	}
	if e = verifyArenaFreeze(arena); e != nil {
		return e
	}
	projectRoot := filepath.Join(arena, "projects", project)
	projectRoot = resolveProjectRoot(arena, protocol, project)
	tasksPath := filepath.Join(projectRoot, "task-set.yaml")
	if _, e = os.Stat(tasksPath); e != nil {
		tasksPath = filepath.Join(projectRoot, "tasks.yaml")
	}
	var projectManifest ProjectManifest
	projectManifestPath := filepath.Join(projectRoot, "project.yaml")
	if _, statErr := os.Stat(projectManifestPath); statErr != nil {
		return fmt.Errorf("project manifest is required: %w", statErr)
	}
	if e = readYAML(projectManifestPath, &projectManifest); e != nil {
		return fmt.Errorf("read project manifest: %w", e)
	}
	if projectManifest.ID != "" && projectManifest.ID != project {
		return fmt.Errorf("project manifest id %q does not match %q", projectManifest.ID, project)
	}
	if projectManifest.Arms[arm].Path == "" {
		return fmt.Errorf("project manifest must declare the %s seed path", arm)
	}
	{
		configured := projectManifest.TaskSet
		if configured == "" {
			configured = projectManifest.Controls.TaskSet
		}
		if configured == "" {
			configured = projectManifest.Shared.TaskSet
		}
		if configured != "" {
			tasksPath = filepath.Join(projectRoot, filepath.FromSlash(configured))
		}
	}
	if parallel && taskSetName == "" {
		taskSetName = "parallel-task-set.yaml"
	}
	if taskSetName != "" {
		tasksPath = filepath.Join(projectRoot, filepath.FromSlash(taskSetName))
	}
	var set taskSet
	if e = readYAML(tasksPath, &set); e != nil {
		return e
	}
	index := -1
	for i, t := range set.Tasks {
		if t.ID == taskID {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("task %s not found", taskID)
	}
	if parallel {
		if forkFromTask == "" || set.ForkFrom.MainTaskID == "" || set.ForkFrom.MainTaskID != forkFromTask {
			return errors.New("parallel task set requires --fork-from-task matching its frozen fork_from.main_task_id")
		}
		if set.ForkFrom.MainTrial > 0 && set.ForkFrom.MainTrial != trial {
			return errors.New("parallel task trial does not match frozen fork_from.main_trial")
		}
		if set.Tasks[index].OriginTaskID == "" || set.Tasks[index].PriorTasksThrough == "" || set.Tasks[index].PriorTasksThrough != set.ForkFrom.PriorTasksThrough {
			return errors.New("parallel card must declare origin_task_id and matching prior_tasks_through")
		}
	} else if forkFromTask != "" || taskSetName != "" {
		return errors.New("--task-set and --fork-from-task require --parallel")
	}
	if isHoldoutID(taskID) {
		var gate HoldoutDecision
		if e = readYAML(filepath.Join(arena, "decisions", "holdout.yaml"), &gate); e != nil {
			return errors.New("holdout tasks remain sealed until record-holdout-decision is saved")
		}
		if gate.Decision != "fix" && gate.Decision != "no-fix" {
			return errors.New("invalid holdout decision record")
		}
	}
	runID := fmt.Sprintf("%s-%s-t%02d", project, arm, trial)
	if parallel {
		runID += "-parallel-" + taskID
	}
	runDir := filepath.Join(arena, "runs", runID)
	if e = os.MkdirAll(runDir, 0755); e != nil {
		return e
	}
	workspace := filepath.Join(arena, "snapshots", runID)
	seedRevision := ""
	baseRevision := ""
	seedRecordPath := filepath.Join(runDir, "seed.yaml")
	if _, e = os.Stat(workspace); os.IsNotExist(e) {
		if parallel {
			mainRunID := fmt.Sprintf("%s-%s-t%02d", project, arm, trial)
			var mainState NativeTaskRecord
			mainStatePath := filepath.Join(arena, "runs", mainRunID, forkFromTask+".native.yaml")
			if e = readYAML(mainStatePath, &mainState); e != nil {
				return fmt.Errorf("parallel fork requires completed task %s: %w", forkFromTask, e)
			}
			if mainState.ActorCompletedUTC == "" || mainState.FinalSnapshot == "" || mainState.ActorStatus == "unusable" {
				return errors.New("parallel fork source task has no usable completed snapshot")
			}
			source := filepath.Join(arena, mainState.FinalSnapshot, "workspace")
			if e = copyTreeMode(source, workspace, true); e != nil {
				return e
			}
			seedRevision = mainState.SeedRevision
		} else {
			seedPath := resolveConfiguredSeed(arena, projectRoot, protocol, project, arm)
			if configured := projectManifest.Arms[arm].Path; configured != "" {
				if filepath.IsAbs(configured) {
					seedPath = configured
				} else {
					seedPath = filepath.Join(projectRoot, filepath.FromSlash(configured))
				}
			}
			if e = copyTree(seedPath, workspace); e != nil {
				return e
			}
			if seedRevision, e = initializeSeedRepository(workspace); e != nil {
				return e
			}
		}
		if e = writeYAMLNew(seedRecordPath, SeedRecord{SeedRevision: seedRevision}); e != nil {
			return e
		}
	} else {
		var seedRecord SeedRecord
		if e = readYAML(seedRecordPath, &seedRecord); e != nil {
			return fmt.Errorf("existing workspace lacks immutable seed mapping: %w", e)
		}
		seedRevision = seedRecord.SeedRevision
	}
	baseRevision, e = gitHead(workspace)
	if e != nil {
		return e
	}
	for i := 0; i < index && !parallel; i++ {
		prev := filepath.Join(runDir, set.Tasks[i].ID+".native.yaml")
		var state NativeTaskRecord
		if e = readYAML(prev, &state); e != nil {
			return fmt.Errorf("cannot start %s before task %s is finished: %w", taskID, set.Tasks[i].ID, e)
		}
		if state.ActorCompletedUTC == "" {
			return fmt.Errorf("task %s is still in progress", set.Tasks[i].ID)
		}
		if state.ActorStatus == "unusable" {
			return fmt.Errorf("run chain ended at unusable workspace after task %s", set.Tasks[i].ID)
		}
	}
	t := set.Tasks[index]
	t.AllowedPaths = allowedFor(t, arm)
	tree, e := treeDigest(workspace)
	if e != nil {
		return e
	}
	promptHash := sha256.Sum256([]byte(t.Prompt))
	protocolHash, e := hashFile(protocolPath)
	if e != nil {
		return e
	}
	var frozen FreezeRecord
	if e = readYAML(filepath.Join(arena, "freeze.yaml"), &frozen); e != nil {
		return e
	}
	combined := sha256.Sum256([]byte(frozen.Digest + "\x00" + protocolHash + "\x00" + tree + "\x00" + hex.EncodeToString(promptHash[:]) + "\x00" + taskID))
	digest := hex.EncodeToString(combined[:])
	ref := projectRef(protocol, project)
	if projectManifest.ID != "" {
		ref.Arms = projectManifest.Arms
	}
	validators := publicValidatorsFor(t, set, ref, arm)
	if len(t.AllowedPaths) == 0 {
		return errors.New("task card must declare the exact allowed_paths boundary")
	}
	if len(validators) == 0 {
		return errors.New("task card or project config must declare public validator argv")
	}
	helper := filepath.Join(arena, "bin", "gauntlet.exe")
	if runtime.GOOS != "windows" {
		helper = filepath.Join(arena, "bin", "gauntlet")
	}
	if _, err := os.Stat(helper); os.IsNotExist(err) {
		helper = "gauntlet"
	}
	validatorManifest := filepath.Join(arena, "raw", runID, t.ID, "validators.yaml")
	if e = os.MkdirAll(filepath.Dir(validatorManifest), 0755); e != nil {
		return e
	}
	manifest, e := yaml.Marshal(ValidatorManifest{BaseRevision: baseRevision, Commands: validators})
	if e != nil {
		return e
	}
	if _, e = os.Stat(validatorManifest); os.IsNotExist(e) {
		if e = writeFileNew(validatorManifest, manifest); e != nil {
			return e
		}
	}
	checkCmd := helper + " check --task-id " + t.ID + " --manifest " + validatorManifest
	// Do not expose the harness build cache or repository paths to actors. An
	// explicit experiment cache override must use the GAUNTLET_GOCACHE name.
	goCache := os.Getenv("GAUNTLET_GOCACHE")
	if goCache == "" {
		goCache = filepath.Join(arena, "raw", "shared-go-cache")
	}
	helperEnv := map[string]string{"PATH": filepath.Join(arena, "bin") + string(os.PathListSeparator) + os.Getenv("PATH"), "GOCACHE": goCache, "PYTHONDONTWRITEBYTECODE": "1", "GAUNTLET_RUN_ID": runID, "GAUNTLET_TASK_ID": t.ID, "GAUNTLET_ACTOR_ATTEMPT": fmt.Sprint(repairIteration + 1), "GAUNTLET_HELPER_RECORD": filepath.Join(arena, "raw", runID, t.ID, fmt.Sprintf("helper-attempt-%d.jsonl", repairIteration)), "GAUNTLET_SNAPSHOT_ROOT": filepath.Join(arena, "raw", runID, t.ID, "snapshots"), "GAUNTLET_PUBLIC_VALIDATORS": validatorManifest, "GAUNTLET_PUBLIC_VALIDATOR_OUTPUT": filepath.Join(arena, "raw", runID, t.ID, fmt.Sprintf("attempt-%d.public-validator.txt", repairIteration))}
	instruction := "This task is assigned to one actor session. Do not spawn, invoke, or delegate work to any other agents or subagents. Read-only engineering guidance and analysis available in this arm may be used while planning; these are not public acceptance tests. If the current task explicitly requires generated-projection regeneration or reconciliation, you may use the corresponding mutating apply/render operation only for that requested work and only when every resulting write stays within the exact authorized mutation boundary below. Do not perform implicit adoption, unapproved exceptions, or external/unscoped apply/render operations. Do not run builds, application tests, or configured acceptance validators outside the public-check helper. Use normal tool sandbox escalation for Git, helper, or build commands only when the authorized arena or cache permissions require it; do not change global Git trust/configuration or bypass tool rules. Commit your task changes, then set the supplied GAUNTLET variables in the actor shell and invoke the exact helper command below once. It snapshots the workspace before validation and preserves actual check output. If any public check fails, stop; a fresh repair actor may receive only the exact recorded public output. Never inspect hidden evaluator data, future tasks, or other arms/trials."
	helperRecord := filepath.Join(arena, "raw", runID, t.ID, "helper.jsonl")
	boundary := "\n\nAuthorized mutation boundary (exact): " + strings.Join(t.AllowedPaths, ", ") + ". Do not change files outside it. Do not weaken, delete, skip, or rewrite existing tests or check assertions; add focused tests when needed. Work only on this task."
	helperBlock := "\n\nGAUNTLET shell environment (set each variable exactly):\n"
	keys := make([]string, 0, len(helperEnv))
	for key := range helperEnv {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := helperEnv[key]
		helperBlock += "$env:" + key + "='" + value + "'\n"
	}
	helperBlock += "\nPublic check command:\n& '" + helper + "' check --task-id " + t.ID + " --manifest '" + validatorManifest + "'"
	evaluationTaskID := t.ID
	env := NativeEnvelope{Version: 1, RunID: runID, Project: project, Arm: arm, Trial: trial, TaskID: t.ID, EvaluationTaskID: evaluationTaskID, Workspace: workspace, Prompt: t.Prompt, ActorPrompt: t.Prompt + boundary + helperBlock + "\n\n" + instruction, AllowedPaths: t.AllowedPaths, PublicValidators: validators, ValidatorManifest: validatorManifest, PublicCheckInstruction: instruction, HelperCommand: checkCmd, HelperEnvironment: helperEnv, HelperRecord: helperRecord, InputDigest: digest, Model: protocol.Agent.Model, ReasoningEffort: protocol.Agent.ReasoningEffort}
	statePath := filepath.Join(runDir, t.ID+".native.yaml")
	state := NativeTaskRecord{Version: 1, RunID: runID, Project: project, Arm: arm, Trial: trial, TaskID: t.ID, EvaluationTaskID: evaluationTaskID, PriorTasksThrough: t.PriorTasksThrough, Parallel: parallel, InputDigest: digest, ActorStartedUTC: time.Now().UTC().Format(time.RFC3339), ActorStatus: "actor_started", Workspace: workspace, SeedRevision: seedRevision, BaseRevision: baseRevision, MaximumRepairIterations: protocol.Execution.MaximumSelfRepairIterations}
	newState := false
	if repairIteration < 0 || repairIteration > protocol.Execution.MaximumSelfRepairIterations {
		return errors.New("repair iteration exceeds frozen protocol limit")
	}
	if repairIteration > 0 {
		if feedbackPath == "" {
			return errors.New("repair actor requires --feedback-file")
		}
		feedback, er := os.ReadFile(feedbackPath)
		if er != nil {
			return er
		}
		if len(feedback) == 0 {
			return errors.New("repair feedback is empty")
		}
		fh := sha256.Sum256(feedback)
		turnHash := sha256.Sum256([]byte(digest + "\x00" + hex.EncodeToString(fh[:]) + "\x00" + fmt.Sprint(repairIteration)))
		digest = hex.EncodeToString(turnHash[:])
		instruction += "\n\nExact public repair feedback from the prior actor turn (unmodified):\n" + string(feedback) + "\n\nApply only the current task's requested correction, then invoke the public-check helper once. Do not make additional changes based on assumptions."
	}
	if _, e = os.Stat(statePath); e == nil {
		if e = readYAML(statePath, &state); e != nil {
			return e
		}
		if state.ActorCompletedUTC != "" {
			return errors.New("task already completed; run IDs are write-once")
		}
		if repairIteration == 0 {
			return errors.New("task already prepared; use --repair-iteration with public feedback")
		}
		if state.RepairIterations != repairIteration || state.ActorStatus != "repair-needed" {
			return errors.New("repair actor is out of order or prior turn did not request repair")
		}
		if deadline, er := time.Parse(time.RFC3339, state.DeadlineUTC); er == nil && time.Now().After(deadline) {
			return errors.New("task wall-clock deadline expired before repair actor start")
		}
		state.ActorStatus = "actor_started"
	} else {
		if repairIteration != 0 {
			return errors.New("repair cannot start before the initial actor turn")
		}
		seconds := protocol.Execution.TaskTimeoutSeconds
		if t.TimeoutSeconds > 0 && t.TimeoutSeconds < seconds {
			seconds = t.TimeoutSeconds
		}
		state.DeadlineUTC = time.Now().Add(time.Duration(seconds) * time.Second).UTC().Format(time.RFC3339)
		newState = true
	}
	state.ActiveAttempt = repairIteration
	state.LastTurnDigest = digest
	state.Turns = append(state.Turns, NativeTurnRecord{Attempt: repairIteration, InputDigest: digest, ActorStartedUTC: time.Now().UTC().Format(time.RFC3339), Status: "actor_started"})
	env.InputDigest = digest
	env.DeadlineUTC = state.DeadlineUTC
	env.PublicCheckInstruction = instruction
	env.ActorPrompt = env.Prompt + boundary + helperBlock + "\n\n" + instruction
	envelopePath := filepath.Join(runDir, fmt.Sprintf("%s.attempt-%d.envelope.yaml", t.ID, repairIteration))
	if e = writeEnvelopeOnce(envelopePath, env); e != nil {
		return e
	}
	if newState {
		e = writeYAMLNew(statePath, state)
	} else {
		e = writeYAMLReplace(statePath, state)
	}
	if e != nil {
		return e
	}
	b, e := json.MarshalIndent(env, "", "  ")
	if e != nil {
		return e
	}
	fmt.Println(string(b))
	return nil
}

func writeEnvelopeOnce(path string, env NativeEnvelope) error {
	if err := writeYAMLNew(path, env); err == nil {
		return nil
	} else if !os.IsExist(err) {
		return err
	}
	var existing NativeEnvelope
	if err := readYAML(path, &existing); err != nil {
		return fmt.Errorf("existing actor envelope is unreadable: %w", err)
	}
	if existing.InputDigest != env.InputDigest || existing.ActorPrompt != env.ActorPrompt || existing.Workspace != env.Workspace {
		return errors.New("an envelope already exists for this attempt with different inputs")
	}
	return nil
}

func finishTask(args []string) error {
	var arena, runID, taskID, digest, responsePath, receiptsPath, status string
	var attempt int
	f, e := mustFlags("finish-task", args, func(f *flag.FlagSet) {
		f.StringVar(&arena, "arena", "", "arena root")
		f.StringVar(&runID, "run-id", "", "run ID")
		f.StringVar(&taskID, "task-id", "", "task ID")
		f.StringVar(&digest, "input-digest", "", "digest from prepare-task")
		f.StringVar(&responsePath, "response-file", "", "exact actor final response file")
		f.StringVar(&receiptsPath, "receipts-file", "", "native tool receipt JSONL")
		f.StringVar(&status, "status", "completed", "completed, failed, protocol-invalid, or unusable")
		f.IntVar(&attempt, "attempt", -1, "zero-based actor turn index from prepare-task")
	})
	if e != nil {
		return e
	}
	_ = f
	if arena == "" || runID == "" || taskID == "" || digest == "" || responsePath == "" {
		return errors.New("finish-task requires --arena --run-id --task-id --input-digest --response-file")
	}
	switch status {
	case "completed", "failed", "protocol-invalid", "unusable", "repair-needed":
	default:
		return errors.New("unsupported actor status")
	}
	runDir := filepath.Join(arena, "runs", runID)
	statePath := filepath.Join(runDir, taskID+".native.yaml")
	var state NativeTaskRecord
	if e = readYAML(statePath, &state); e != nil {
		return e
	}
	if attempt < 0 {
		attempt = state.ActiveAttempt
	}
	if state.ActiveAttempt != attempt || state.LastTurnDigest != digest {
		return errors.New("input digest does not match prepared task")
	}
	if deadline, er := time.Parse(time.RFC3339, state.DeadlineUTC); er == nil && time.Now().After(deadline) {
		state.TimeoutExceeded = true
		if status == "completed" || status == "repair-needed" {
			status = "failed"
		}
	}
	if state.ActorCompletedUTC != "" {
		return errors.New("task completion is write-once")
	}
	response, e := os.ReadFile(responsePath)
	if e != nil {
		return e
	}
	responseOut := filepath.Join(runDir, fmt.Sprintf("%s.attempt-%d.response.txt", taskID, attempt))
	if e = writeFileNew(responseOut, response); e != nil {
		return e
	}
	state.ResponseFile = filepath.Base(responseOut)
	state.Turns[len(state.Turns)-1].ResponseFile = filepath.Base(responseOut)
	if receiptsPath != "" {
		receipt, e := os.ReadFile(receiptsPath)
		if e != nil {
			return e
		}
		receiptOut := filepath.Join(runDir, fmt.Sprintf("%s.attempt-%d.receipts.jsonl", taskID, attempt))
		if e = writeFileNew(receiptOut, receipt); e != nil {
			return e
		}
		state.ReceiptsFile = filepath.Base(receiptOut)
		state.Turns[len(state.Turns)-1].ReceiptsFile = filepath.Base(receiptOut)
	}
	if len(state.Turns) == 0 || state.Turns[len(state.Turns)-1].Attempt != attempt {
		return errors.New("prepared actor attempt record is missing")
	}
	if status == "repair-needed" && attempt >= state.MaximumRepairIterations {
		status = "failed"
	}
	state.ActorStatus = status
	completed := time.Now().UTC().Format(time.RFC3339)
	state.Turns[len(state.Turns)-1].Status = status
	state.Turns[len(state.Turns)-1].ActorCompletedUTC = completed
	state.Turns[len(state.Turns)-1].CompletedDigest = digest
	if status == "repair-needed" {
		state.RepairIterations = attempt + 1
		state.ActorCompletedUTC = ""
		if state.ParallelIntegration {
			publicOutput := filepath.Join(arena, "raw", runID, fmt.Sprintf("08.attempt-%d.public-validator.txt", attempt))
			feedback := filepath.Join(arena, "raw", runID, fmt.Sprintf("08.attempt-%d.feedback.txt", attempt))
			contents, readErr := os.ReadFile(publicOutput)
			if readErr != nil {
				return fmt.Errorf("integration repair requires recorded public helper output: %w", readErr)
			}
			if e = writeFileNew(feedback, contents); e != nil {
				return e
			}
		}
	} else {
		state.ActorCompletedUTC = completed
		state.FinalSnapshot = filepath.Join("results", runID, "snapshots", taskID)
		snapshot := filepath.Join(arena, state.FinalSnapshot)
		if e = capturePatch(state.Workspace, snapshot); e != nil {
			return e
		}
	}
	return writeYAMLReplace(statePath, state)
}

func integrateParallel(args []string) error {
	var arena, project, arm, mainTaskID, taskIDs, protocolPath string
	var trial int
	f, err := mustFlags("integrate-parallel", args, func(f *flag.FlagSet) {
		f.StringVar(&arena, "arena", "", "external arena")
		f.StringVar(&project, "project", "", "project id")
		f.StringVar(&arm, "arm", "", "A or B")
		f.IntVar(&trial, "trial", 1, "trial number")
		f.StringVar(&mainTaskID, "main-task-id", "06", "completed sequential fork base task")
		f.StringVar(&taskIDs, "parallel-task-ids", "P01,P02", "parallel cards to integrate")
		f.StringVar(&protocolPath, "protocol", "", "frozen protocol YAML")
	})
	if err != nil {
		return err
	}
	_ = f
	if arena == "" || project == "" || arm == "" || protocolPath == "" {
		return errors.New("integrate-parallel requires --arena --project --arm --protocol")
	}
	arm = strings.ToLower(strings.TrimPrefix(arm, "arm-"))
	if arm != "a" && arm != "b" {
		return errors.New("--arm must be A or B")
	}
	if err = verifyArenaFreeze(arena); err != nil {
		return err
	}
	var protocol Protocol
	if err = readYAML(protocolPath, &protocol); err != nil {
		return err
	}
	if protocol.Execution.TaskTimeoutSeconds <= 0 || protocol.Execution.MaximumSelfRepairIterations < 0 || protocol.Execution.MaximumSelfRepairIterations > 2 {
		return errors.New("integration requires a positive frozen task timeout and at most two repair iterations")
	}
	mainRunID := fmt.Sprintf("%s-%s-t%02d", project, arm, trial)
	var mainState NativeTaskRecord
	if err = readYAML(filepath.Join(arena, "runs", mainRunID, mainTaskID+".native.yaml"), &mainState); err != nil {
		return err
	}
	if mainState.ActorCompletedUTC == "" || mainState.FinalSnapshot == "" || mainState.ActorStatus == "unusable" {
		return errors.New("main fork base must be a usable completed task snapshot")
	}
	baseWorkspace := filepath.Join(arena, mainState.FinalSnapshot, "workspace")
	baseRevision, err := gitHead(baseWorkspace)
	if err != nil {
		return err
	}
	ids := strings.Split(taskIDs, ",")
	sort.Strings(ids)
	parallelStates := make([]NativeTaskRecord, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		runID := mainRunID + "-parallel-" + id
		var s NativeTaskRecord
		if err = readYAML(filepath.Join(arena, "runs", runID, id+".native.yaml"), &s); err != nil {
			return fmt.Errorf("parallel task %s is missing: %w", id, err)
		}
		if !s.Parallel || s.ActorCompletedUTC == "" || s.FinalSnapshot == "" || s.ActorStatus == "unusable" {
			return fmt.Errorf("parallel task %s is not a usable completed fork", id)
		}
		if s.BaseRevision != baseRevision {
			return fmt.Errorf("parallel task %s fork base %s differs from main task commit %s", id, s.BaseRevision, baseRevision)
		}
		parallelStates = append(parallelStates, s)
	}
	if len(parallelStates) == 0 {
		return errors.New("no parallel task IDs supplied")
	}
	if len(parallelStates) != 2 || parallelStates[0].TaskID != "P01" || parallelStates[1].TaskID != "P02" {
		return errors.New("combined integration requires both P01 and P02 in lexical order")
	}
	integrationID := fmt.Sprintf("%s-%s-t%02d-integration", project, arm, trial)
	integrationStarted := time.Now().UTC()
	workspace := filepath.Join(arena, "snapshots", integrationID)
	if _, err = os.Stat(workspace); err == nil {
		return errors.New("integration workspace already exists; integrations are write-once")
	}
	if err = copyTreeMode(baseWorkspace, workspace, true); err != nil {
		return err
	}
	branch := "codex/gauntlet-integration"
	if err = gitRun(workspace, "switch", "-c", branch); err != nil {
		return err
	}
	var integrationError string
	var integrationPlan []IntegrationCommit
	var appliedIntegrationCommits []IntegrationCommit
	for _, s := range parallelStates {
		fork := filepath.Join(arena, s.FinalSnapshot, "workspace")
		head, er := gitHead(fork)
		if er != nil {
			integrationError = er.Error()
			break
		}
		fetchRef := "refs/heads/codex/gauntlet-import-" + s.TaskID
		if er = gitRun(workspace, "fetch", "--no-tags", fork, "HEAD:"+fetchRef); er != nil {
			integrationError = fmt.Sprintf("import commits from %s: %v", s.TaskID, er)
			break
		}
		commitsCmd := exec.Command("git", "rev-list", "--reverse", baseRevision+".."+head)
		commitsCmd.Dir = fork
		commitBytes, er := commitsCmd.Output()
		if er != nil {
			integrationError = er.Error()
			break
		}
		commits := strings.Fields(string(commitBytes))
		if len(commits) == 0 {
			integrationError = "parallel branch has no actor commit after fork base: " + s.TaskID
			break
		}
		for _, commit := range commits {
			integrationPlan = append(integrationPlan, IntegrationCommit{TaskID: s.TaskID, Commit: commit})
		}
	}
	planPath := filepath.Join(arena, "raw", integrationID, "integration-plan.yaml")
	if err = writeYAMLNew(planPath, integrationPlan); err != nil {
		return err
	}
	if integrationError == "" {
		for _, entry := range integrationPlan {
			if er := gitRun(workspace, "cherry-pick", "-x", entry.Commit); er != nil {
				integrationError = fmt.Sprintf("cherry-pick %s from %s: %v", entry.Commit, entry.TaskID, er)
				break
			}
			appliedIntegrationCommits = append(appliedIntegrationCommits, entry)
		}
	}
	projectRoot := filepath.Join(arena, "projects", project)
	var projectManifest ProjectManifest
	projectManifestPath := filepath.Join(projectRoot, "project.yaml")
	if _, statErr := os.Stat(projectManifestPath); statErr != nil {
		return fmt.Errorf("project manifest is required: %w", statErr)
	}
	if err = readYAML(projectManifestPath, &projectManifest); err != nil {
		return fmt.Errorf("read project manifest: %w", err)
	}
	var set taskSet
	setPath := filepath.Join(projectRoot, "task-set.yaml")
	if projectManifest.Shared.TaskSet != "" {
		setPath = filepath.Join(projectRoot, filepath.FromSlash(projectManifest.Shared.TaskSet))
	}
	if projectManifest.Controls.TaskSet != "" {
		setPath = filepath.Join(projectRoot, filepath.FromSlash(projectManifest.Controls.TaskSet))
	}
	if projectManifest.TaskSet != "" {
		setPath = filepath.Join(projectRoot, filepath.FromSlash(projectManifest.TaskSet))
	}
	if err = readYAML(setPath, &set); err != nil {
		return err
	}
	var finalTask Task
	for _, t := range set.Tasks {
		if t.ID == "08" {
			finalTask = t
			break
		}
	}
	if finalTask.ID == "" {
		return errors.New("main task set lacks final combined task 08")
	}
	ref := projectRef(Protocol{}, project)
	ref.Arms = projectManifest.Arms
	validators := publicValidatorsFor(finalTask, set, ref, arm)
	if len(validators) == 0 {
		return errors.New("combined integration has no public validator configuration")
	}
	runDir := filepath.Join(arena, "runs", integrationID)
	if err = os.MkdirAll(runDir, 0755); err != nil {
		return err
	}
	manifestPath := filepath.Join(arena, "raw", integrationID, "08.validators.yaml")
	if err = writeYAMLNew(manifestPath, ValidatorManifest{BaseRevision: baseRevision, Commands: validators}); err != nil {
		return err
	}
	checkRecord := filepath.Join(arena, "raw", integrationID, "08.attempt-0.helper.jsonl")
	checkSnapshot := filepath.Join(arena, "raw", integrationID, "snapshots", "attempt-0")
	checkOutput := filepath.Join(arena, "raw", integrationID, "08.attempt-0.public-validator.txt")
	helper := filepath.Join(arena, "bin", "gauntlet.exe")
	if runtime.GOOS != "windows" {
		helper = filepath.Join(arena, "bin", "gauntlet")
	}
	if _, statErr := os.Stat(helper); statErr != nil {
		helper, _ = os.Executable()
	}
	cmd := exec.Command(helper, "check", "--task-id", "08", "--manifest", manifestPath)
	cmd.Dir = workspace
	goCache := os.Getenv("GAUNTLET_GOCACHE")
	if goCache == "" {
		goCache = filepath.Join(arena, "raw", "shared-go-cache")
	}
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(arena, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"), "GOCACHE="+goCache, "PYTHONDONTWRITEBYTECODE=1", "GAUNTLET_RUN_ID="+integrationID, "GAUNTLET_TASK_ID=08", "GAUNTLET_ACTOR_ATTEMPT=1", "GAUNTLET_HELPER_RECORD="+checkRecord, "GAUNTLET_SNAPSHOT_ROOT="+checkSnapshot, "GAUNTLET_PUBLIC_VALIDATORS="+manifestPath, "GAUNTLET_PUBLIC_VALIDATOR_OUTPUT="+checkOutput)
	stdoutPath := filepath.Join(arena, "raw", integrationID, "attempt-0.check.stdout.txt")
	stderrPath := filepath.Join(arena, "raw", integrationID, "attempt-0.check.stderr.txt")
	if err = os.MkdirAll(filepath.Dir(stdoutPath), 0755); err != nil {
		return err
	}
	stdout, err := os.Create(stdoutPath)
	if err != nil {
		return err
	}
	stderr, err := os.Create(stderrPath)
	if err != nil {
		stdout.Close()
		return err
	}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	checkErr := cmd.Run()
	_ = stdout.Close()
	_ = stderr.Close()
	if checkErr != nil && integrationError == "" {
		integrationError = checkErr.Error()
	}
	if _, validationCode, _, _ := helperResults(checkRecord); validationCode != 0 && integrationError == "" {
		integrationError = "combined task-08 public validator failed"
	}
	initialSnapshot := filepath.Join("results", integrationID, "snapshots", "08-attempt-0")
	if err = capturePatch(workspace, filepath.Join(arena, initialSnapshot)); err != nil {
		return err
	}
	status := "completed"
	if integrationError != "" {
		if protocol.Execution.MaximumSelfRepairIterations > 0 {
			status = "repair-needed"
		} else {
			status = "failed"
		}
	}
	completed := time.Now().UTC().Format(time.RFC3339)
	seconds := protocol.Execution.TaskTimeoutSeconds
	if finalTask.TimeoutSeconds > 0 && finalTask.TimeoutSeconds < seconds {
		seconds = finalTask.TimeoutSeconds
	}
	deadline := integrationStarted.Add(time.Duration(seconds) * time.Second).UTC().Format(time.RFC3339)
	finalSnapshot := initialSnapshot
	if status == "completed" {
		finalSnapshot = filepath.Join("results", integrationID, "snapshots", "08")
		if err = capturePatch(workspace, filepath.Join(arena, finalSnapshot)); err != nil {
			return err
		}
	}
	var frozen FreezeRecord
	if err = readYAML(filepath.Join(arena, "freeze.yaml"), &frozen); err != nil {
		return err
	}
	workspaceDigest, err := treeDigest(workspace)
	if err != nil {
		return err
	}
	planBytes, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}
	planDigest := sha256.Sum256(planBytes)
	inputDigest := sha256.Sum256([]byte(frozen.Digest + "\x00" + project + "\x00" + arm + "\x00" + fmt.Sprint(trial) + "\x00" + baseRevision + "\x00" + workspaceDigest + "\x00" + hex.EncodeToString(planDigest[:])))
	turnStatus := status
	if status == "repair-needed" {
		turnStatus = "repair-needed"
	}
	state := NativeTaskRecord{Version: 1, RunID: integrationID, Project: project, Arm: arm, Trial: trial, TaskID: "08", EvaluationTaskID: "08", PriorTasksThrough: mainTaskID, ParallelIntegration: true, MaximumRepairIterations: protocol.Execution.MaximumSelfRepairIterations, ActorStatus: status, ActorStartedUTC: integrationStarted.Format(time.RFC3339), ActorCompletedUTC: completed, DeadlineUTC: deadline, Workspace: workspace, FinalSnapshot: finalSnapshot, InitialSnapshot: initialSnapshot, SeedRevision: mainState.SeedRevision, BaseRevision: baseRevision, IntegrationPlanPath: filepath.ToSlash(filepath.Join("raw", integrationID, "integration-plan.yaml")), AppliedIntegrationCommits: appliedIntegrationCommits, InputDigest: hex.EncodeToString(inputDigest[:]), LastTurnDigest: hex.EncodeToString(inputDigest[:]), Turns: []NativeTurnRecord{{Attempt: 0, InputDigest: hex.EncodeToString(inputDigest[:]), ActorStartedUTC: integrationStarted.Format(time.RFC3339), ActorCompletedUTC: completed, Status: turnStatus, CompletedDigest: hex.EncodeToString(inputDigest[:])}}}
	if status == "repair-needed" {
		state.RepairIterations = 1
		state.ActorCompletedUTC = ""
	}
	if integrationError != "" {
		checkBytes, _ := os.ReadFile(checkOutput)
		feedback := []byte("Mechanical integration output:\n" + integrationError + "\n\nRecorded public validator output follows exactly:\n")
		feedback = append(feedback, checkBytes...)
		if err = writeFileNew(filepath.Join(arena, "raw", integrationID, "08.attempt-0.feedback.txt"), feedback); err != nil {
			return err
		}
	}
	statePath := filepath.Join(runDir, "08.native.yaml")
	if err = writeYAMLNew(statePath, state); err != nil {
		return err
	}
	if integrationError != "" {
		return fmt.Errorf("integration preserved with failed status: %s", integrationError)
	}
	fmt.Println("integrated parallel cards into", integrationID)
	return nil
}

func prepareIntegrationRepair(args []string) error {
	var arena, project, arm, protocolPath, feedbackPath string
	var trial, attempt int
	f, err := mustFlags("prepare-integration-repair", args, func(f *flag.FlagSet) {
		f.StringVar(&arena, "arena", "", "external arena")
		f.StringVar(&project, "project", "", "project id")
		f.StringVar(&arm, "arm", "", "A or B")
		f.IntVar(&trial, "trial", 1, "trial number")
		f.StringVar(&protocolPath, "protocol", "", "frozen protocol YAML")
		f.IntVar(&attempt, "attempt", 0, "one-based fresh integration repair actor attempt (1 or 2)")
		f.StringVar(&feedbackPath, "feedback-file", "", "exact saved public integration feedback from the previous attempt")
	})
	if err != nil {
		return err
	}
	_ = f
	if arena == "" || project == "" || arm == "" || protocolPath == "" || trial < 1 || feedbackPath == "" {
		return errors.New("prepare-integration-repair requires --arena --project --arm --trial --protocol --attempt --feedback-file")
	}
	arm = strings.ToLower(strings.TrimPrefix(arm, "arm-"))
	if arm != "a" && arm != "b" {
		return errors.New("--arm must be A or B")
	}
	if attempt < 1 || attempt > 2 {
		return errors.New("integration repair attempt must be 1 or 2")
	}
	var protocol Protocol
	if err = readYAML(protocolPath, &protocol); err != nil {
		return err
	}
	if protocol.Execution.TaskTimeoutSeconds <= 0 || attempt > protocol.Execution.MaximumSelfRepairIterations {
		return errors.New("integration repair attempt exceeds the frozen positive-timeout repair budget")
	}
	if err = verifyArenaFreeze(arena); err != nil {
		return err
	}
	runID := fmt.Sprintf("%s-%s-t%02d-integration", project, arm, trial)
	runDir := filepath.Join(arena, "runs", runID)
	statePath := filepath.Join(runDir, "08.native.yaml")
	var state NativeTaskRecord
	if err = readYAML(statePath, &state); err != nil {
		return err
	}
	if !state.ParallelIntegration || state.ActorCompletedUTC != "" || state.ActorStatus != "repair-needed" {
		return errors.New("integration repair requires an unfinished integration recorded as repair-needed")
	}
	if state.RepairIterations != attempt || state.MaximumRepairIterations < attempt || len(state.Turns) == 0 || state.Turns[len(state.Turns)-1].Attempt != attempt-1 {
		return errors.New("integration repair attempt is out of order or exceeds its preserved repair budget")
	}
	deadline, err := time.Parse(time.RFC3339, state.DeadlineUTC)
	if err != nil || time.Now().After(deadline) {
		return errors.New("integration task's frozen wall-clock deadline expired before repair actor start")
	}
	expectedFeedback := filepath.Join(arena, "raw", runID, fmt.Sprintf("08.attempt-%d.feedback.txt", attempt-1))
	actualFeedback, err := os.ReadFile(feedbackPath)
	if err != nil {
		return err
	}
	savedFeedback, err := os.ReadFile(expectedFeedback)
	if err != nil {
		return fmt.Errorf("saved prior integration feedback is unavailable: %w", err)
	}
	if len(savedFeedback) == 0 || !bytes.Equal(actualFeedback, savedFeedback) {
		return errors.New("integration repair feedback must byte-match the previous attempt's saved public output")
	}
	projectRoot := resolveProjectRoot(arena, protocol, project)
	projectManifestPath := filepath.Join(projectRoot, "project.yaml")
	var projectManifest ProjectManifest
	if err = readYAML(projectManifestPath, &projectManifest); err != nil {
		return fmt.Errorf("read project manifest: %w", err)
	}
	mainTaskSetPath := filepath.Join(projectRoot, "task-set.yaml")
	if projectManifest.Controls.TaskSet != "" {
		mainTaskSetPath = filepath.Join(projectRoot, filepath.FromSlash(projectManifest.Controls.TaskSet))
	}
	if projectManifest.Shared.TaskSet != "" {
		mainTaskSetPath = filepath.Join(projectRoot, filepath.FromSlash(projectManifest.Shared.TaskSet))
	}
	if projectManifest.TaskSet != "" {
		mainTaskSetPath = filepath.Join(projectRoot, filepath.FromSlash(projectManifest.TaskSet))
	}
	var mainSet taskSet
	if err = readYAML(mainTaskSetPath, &mainSet); err != nil {
		return fmt.Errorf("read main task set: %w", err)
	}
	var overlay taskSet
	parallelSetPath := filepath.Join(projectRoot, "parallel-task-set.yaml")
	if err = readYAML(parallelSetPath, &overlay); err != nil {
		return fmt.Errorf("read frozen parallel task set: %w", err)
	}
	cardByID := make(map[string]Task, len(overlay.Tasks))
	for _, card := range overlay.Tasks {
		cardByID[card.ID] = card
	}
	cardIDs := []string{"P01", "P02"}
	var taskPrompts []string
	var allowedPaths []string
	for _, id := range cardIDs {
		card, ok := cardByID[id]
		if !ok || card.Prompt == "" {
			return fmt.Errorf("frozen parallel overlay lacks task %s prompt", id)
		}
		taskPrompts = append(taskPrompts, id+": "+card.Prompt)
		allowedPaths = append(allowedPaths, allowedFor(card, arm)...)
	}
	allowedPaths = sortedUnique(allowedPaths)
	if len(allowedPaths) == 0 {
		return errors.New("parallel cards do not declare an arm-authorized integration scope")
	}
	var plan []IntegrationCommit
	if err = readYAML(filepath.Join(arena, filepath.FromSlash(state.IntegrationPlanPath)), &plan); err != nil {
		return fmt.Errorf("read frozen integration commit plan: %w", err)
	}
	remaining := make([]IntegrationCommit, 0, len(plan))
	for _, entry := range plan {
		applied, applyErr := integrationCommitApplied(state.Workspace, state.BaseRevision, entry, state.AppliedIntegrationCommits)
		if applyErr != nil {
			return fmt.Errorf("inspect integration lineage for %s commit %s: %w", entry.TaskID, entry.Commit, applyErr)
		}
		if applied {
			continue
		}
		remaining = append(remaining, entry)
	}
	manifestPath := filepath.Join(arena, "raw", runID, "08.validators.yaml")
	var manifest ValidatorManifest
	if err = readYAML(manifestPath, &manifest); err != nil {
		return fmt.Errorf("read preserved combined public validator manifest: %w", err)
	}
	workspaceDigest, err := treeDigest(state.Workspace)
	if err != nil {
		return err
	}
	feedbackHash := sha256.Sum256(savedFeedback)
	protocolHash, err := hashFile(protocolPath)
	if err != nil {
		return err
	}
	planBytes, err := os.ReadFile(filepath.Join(arena, filepath.FromSlash(state.IntegrationPlanPath)))
	if err != nil {
		return err
	}
	planHash := sha256.Sum256(planBytes)
	combined := sha256.Sum256([]byte(state.LastTurnDigest + "\x00" + workspaceDigest + "\x00" + hex.EncodeToString(feedbackHash[:]) + "\x00" + hex.EncodeToString(planHash[:]) + "\x00" + protocolHash + "\x00" + fmt.Sprint(attempt)))
	digest := hex.EncodeToString(combined[:])
	basePrompt := "Continue the same combined integration of frozen parallel tasks P01 and P02. Current task requests:\n" + strings.Join(taskPrompts, "\n")
	var planLines []string
	for _, entry := range remaining {
		planLines = append(planLines, entry.TaskID+" "+entry.Commit)
	}
	planInstruction := "\n\nRead-only frozen integration plan, in required lexical commit order: " + strings.Join(planLines, ", ") + ". Already-applied commits are omitted. Do not skip, reorder, or invent commits. If Git has a cherry-pick in progress, resolve only conflicts within the authorized scope and continue that exact cherry-pick; then apply each remaining listed commit mechanically in order using `git cherry-pick -x <commit>` so the original commit reference is retained in Git history. If another conflict occurs, preserve it and stop applying later commits. Do not make operator or product-scope changes."
	boundary := "\n\nAuthorized mutation boundary (exact union of P01/P02 for this arm): " + strings.Join(allowedPaths, ", ") + ". Do not change files outside it. Do not weaken, delete, skip, or rewrite existing tests or check assertions."
	instruction := "This is one fresh repair actor for a previously failed mechanical integration. Do not spawn, invoke, or delegate to any other agent. Read-only engineering guidance and analysis may be used; these are not public acceptance tests. Apply only the current integration correction and the frozen mechanical continuation plan. If a task explicitly requires generated-projection regeneration or reconciliation, use the corresponding apply/render operation only for that requested work and only when every write stays within the exact authorized boundary; do not perform implicit adoption, unapproved exceptions, or external/unscoped operations. Do not run builds, application tests, or configured acceptance validators outside the public-check helper. Use normal tool sandbox escalation for Git, helper, or build commands only when the authorized arena or cache permissions require it; do not change global Git trust/configuration or bypass tool rules. Commit your changes and any successfully continued frozen commits, then invoke the public-check helper exactly once. If it fails, stop; only its exact recorded output may go to the next fresh repair actor. Never inspect hidden evaluator data, future tasks, or other arms/trials."
	feedbackBlock := "\n\nExact saved prior-attempt integration/public output (unmodified):\n" + string(savedFeedback)
	helper := filepath.Join(arena, "bin", "gauntlet.exe")
	if runtime.GOOS != "windows" {
		helper = filepath.Join(arena, "bin", "gauntlet")
	}
	if _, statErr := os.Stat(helper); statErr != nil {
		helper, _ = os.Executable()
	}
	helperEnv := map[string]string{
		"PATH":    filepath.Join(arena, "bin") + string(os.PathListSeparator) + os.Getenv("PATH"),
		"GOCACHE": filepath.Join(arena, "raw", "shared-go-cache"), "PYTHONDONTWRITEBYTECODE": "1",
		"GAUNTLET_RUN_ID": runID, "GAUNTLET_TASK_ID": "08", "GAUNTLET_ACTOR_ATTEMPT": fmt.Sprint(attempt + 1),
		"GAUNTLET_HELPER_RECORD":           filepath.Join(arena, "raw", runID, fmt.Sprintf("08.attempt-%d.helper.jsonl", attempt)),
		"GAUNTLET_SNAPSHOT_ROOT":           filepath.Join(arena, "raw", runID, "snapshots", fmt.Sprintf("attempt-%d", attempt)),
		"GAUNTLET_PUBLIC_VALIDATORS":       manifestPath,
		"GAUNTLET_PUBLIC_VALIDATOR_OUTPUT": filepath.Join(arena, "raw", runID, fmt.Sprintf("08.attempt-%d.public-validator.txt", attempt)),
	}
	if cache := os.Getenv("GAUNTLET_GOCACHE"); cache != "" {
		helperEnv["GOCACHE"] = cache
	}
	var helperBlock strings.Builder
	helperBlock.WriteString("\n\nGAUNTLET shell environment (set each variable exactly):\n")
	keys := make([]string, 0, len(helperEnv))
	for key := range helperEnv {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&helperBlock, "$env:%s='%s'\n", key, helperEnv[key])
	}
	helperCommand := helper + " check --task-id 08 --manifest " + manifestPath
	helperBlock.WriteString("\nPublic check command:\n& '" + helper + "' check --task-id 08 --manifest '" + manifestPath + "'")
	env := NativeEnvelope{Version: 1, RunID: runID, Project: project, Arm: arm, Trial: trial, TaskID: "08", EvaluationTaskID: "08", Workspace: state.Workspace, Prompt: basePrompt, ActorPrompt: basePrompt + boundary + planInstruction + helperBlock.String() + "\n\n" + instruction + feedbackBlock, AllowedPaths: allowedPaths, PublicValidators: manifest.Commands, ValidatorManifest: manifestPath, PublicCheckInstruction: instruction, HelperCommand: helperCommand, HelperEnvironment: helperEnv, HelperRecord: filepath.Join(arena, "raw", runID, "08.helper.jsonl"), InputDigest: digest, Model: protocol.Agent.Model, ReasoningEffort: protocol.Agent.ReasoningEffort, DeadlineUTC: state.DeadlineUTC, IntegrationTaskIDs: cardIDs, IntegrationPlan: plan, IntegrationRemainingCommits: remaining}
	envelopePath := filepath.Join(runDir, fmt.Sprintf("08.attempt-%d.envelope.yaml", attempt))
	if err = writeEnvelopeOnce(envelopePath, env); err != nil {
		return err
	}
	state.ActorStatus = "actor_started"
	state.ActiveAttempt = attempt
	state.LastTurnDigest = digest
	state.Turns = append(state.Turns, NativeTurnRecord{Attempt: attempt, InputDigest: digest, ActorStartedUTC: time.Now().UTC().Format(time.RFC3339), Status: "actor_started"})
	if err = writeYAMLReplace(statePath, state); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func sortedUnique(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func integrationCommitApplied(workspace, baseRevision string, entry IntegrationCommit, ledger []IntegrationCommit) (bool, error) {
	for _, applied := range ledger {
		if applied.TaskID == entry.TaskID && applied.Commit == entry.Commit {
			return true, nil
		}
	}
	ancestor := exec.Command("git", "merge-base", "--is-ancestor", entry.Commit, "HEAD")
	ancestor.Dir = workspace
	if err := ancestor.Run(); err == nil {
		return true, nil
	} else if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		return false, err
	}
	log := exec.Command("git", "log", "--format=%B", baseRevision+"..HEAD")
	log.Dir = workspace
	contents, err := log.Output()
	if err != nil {
		return false, err
	}
	trailer := "(cherry picked from commit " + entry.Commit + ")"
	return strings.Contains(string(contents), trailer), nil
}

func gitRun(root string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func recordHoldoutDecision(args []string) error {
	var arena, decision, rationale, candidate string
	f, e := mustFlags("record-holdout-decision", args, func(f *flag.FlagSet) {
		f.StringVar(&arena, "arena", "", "arena root")
		f.StringVar(&decision, "decision", "", "fix or no-fix")
		f.StringVar(&rationale, "rationale", "", "recorded decision rationale")
		f.StringVar(&candidate, "candidate", "", "fixed candidate or no-fix decision identity")
	})
	if e != nil {
		return e
	}
	_ = f
	if arena == "" || rationale == "" || candidate == "" || (decision != "fix" && decision != "no-fix") {
		return errors.New("requires --arena --decision fix|no-fix --candidate ID --rationale TEXT")
	}
	record := HoldoutDecision{Decision: decision, Rationale: rationale, Candidate: candidate, RecordedUTC: time.Now().UTC().Format(time.RFC3339)}
	return writeYAMLNew(filepath.Join(arena, "decisions", "holdout.yaml"), record)
}

func writeFileNew(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return e
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = f.Write(b)
	return e
}
func verifyArenaFreeze(arena string) error {
	var r FreezeRecord
	if e := readYAML(filepath.Join(arena, "freeze.yaml"), &r); e != nil {
		return e
	}
	return verifyFreeze(arena, r)
}
func treeDigest(root string) (string, error) {
	var paths []string
	e := filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() {
			paths = append(paths, p)
		}
		return nil
	})
	if e != nil {
		return "", e
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		rel, _ := filepath.Rel(root, p)
		if excludedWorkspacePath(rel) {
			continue
		}
		sum, e := hashFile(p)
		if e != nil {
			return "", e
		}
		io.WriteString(h, filepath.ToSlash(rel)+"\x00"+sum+"\n")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func protocolDigest(p Protocol) string {
	b, _ := yaml.Marshal(p)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func resolveSeed(base, variant string) string {
	v := strings.ToLower(variant)
	letter := strings.TrimPrefix(v, "arm-")
	if strings.HasPrefix(v, "seed-") {
		letter = strings.TrimPrefix(v, "seed-")
	}
	for _, candidate := range []string{"arm-" + letter, letter, "seed-" + letter} {
		path := filepath.Join(base, candidate)
		if info, e := os.Stat(path); e == nil && info.IsDir() {
			return path
		}
	}
	return filepath.Join(base, "seed-"+letter)
}

func resolveProjectRoot(arena string, p Protocol, id string) string {
	for _, ref := range p.Projects {
		if ref.ID == id || filepath.Base(filepath.Clean(ref.Path)) == id {
			if ref.Path != "" {
				if filepath.IsAbs(ref.Path) {
					return ref.Path
				}
				return filepath.Join(arena, ref.Path)
			}
			if ref.ID != "" {
				candidate := filepath.Join(arena, "projects", ref.ID)
				if info, e := os.Stat(candidate); e == nil && info.IsDir() {
					return candidate
				}
			}
		}
	}
	candidate := filepath.Join(arena, "projects", id)
	if info, e := os.Stat(candidate); e == nil && info.IsDir() {
		return candidate
	}
	entries, _ := os.ReadDir(filepath.Join(arena, "projects"))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		for _, name := range []string{"task-set.yaml", "tasks.yaml"} {
			var set taskSet
			if readYAML(filepath.Join(arena, "projects", entry.Name(), name), &set) == nil && set.Project == id {
				return filepath.Join(arena, "projects", entry.Name())
			}
		}
	}
	return candidate
}
func resolveConfiguredSeed(arena, base string, p Protocol, id, arm string) string {
	for _, ref := range p.Projects {
		if ref.ID == id || filepath.Base(filepath.Clean(ref.Path)) == id {
			value := ref.Arms[arm].Path
			if value == "" {
				if arm == "a" {
					value = ref.A
				} else {
					value = ref.B
				}
			}
			if value != "" {
				if filepath.IsAbs(value) {
					return value
				}
				if strings.HasPrefix(filepath.ToSlash(value), "projects/") {
					return filepath.Join(arena, filepath.FromSlash(value))
				}
				return filepath.Join(base, value)
			}
		}
	}
	return resolveSeed(base, arm)
}
func projectRef(p Protocol, id string) ProjectRef {
	for _, ref := range p.Projects {
		if ref.ID == id || filepath.Base(filepath.Clean(ref.Path)) == id {
			return ref
		}
	}
	return ProjectRef{}
}
func validatorsFor(t Task, ref ProjectRef, arm string) [][]string {
	var all [][]string
	if len(t.Validator) > 0 {
		all = append(all, append([]string(nil), t.Validator...))
	}
	all = append(all, ref.Validators[arm]...)
	all = append(all, ref.Arms[arm].Validator...)
	all = append(all, ref.Arms[arm].FullAcceptance...)
	return all
}

func publicValidatorsFor(t Task, set taskSet, ref ProjectRef, arm string) [][]string {
	configured := ref.Arms[arm]
	all := validatorsFor(t, ref, arm)
	if commands, ok := set.ValidatorByArm[arm]; ok && len(commands) > 0 {
		all = append(all, append([]string(nil), commands...))
	}
	if len(set.Validator) > 0 {
		all = append(all, append([]string(nil), set.Validator...))
	}
	if len(configured.Validator) == 0 && len(configured.FullAcceptance) == 0 && len(set.ValidatorByArm) == 0 && len(set.Validator) == 0 {
		all = append(all, set.BaselineChecks...)
	}
	seen := map[string]bool{}
	unique := make([][]string, 0, len(all))
	for _, command := range all {
		key := strings.Join(command, "\x00")
		if len(command) == 0 || seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, command)
	}
	return unique
}

func allowedFor(t Task, arm string) []string {
	if scoped, ok := t.AllowedPathsByArm[arm]; ok {
		return scoped
	}
	return t.AllowedPaths
}

func run(args []string, smoke bool) error {
	var arena, project, variant, actor, protocolPath, seed string
	var trial int
	f, e := mustFlags("run", args, func(f *flag.FlagSet) {
		f.StringVar(&arena, "arena", "", "arena root")
		f.StringVar(&project, "project", "", "project name")
		f.StringVar(&variant, "variant", "", "seed variant")
		f.IntVar(&trial, "trial", 0, "trial number")
		f.StringVar(&actor, "actor", "", "actor executable (defaults to protocol)")
		f.StringVar(&protocolPath, "protocol", "", "frozen protocol YAML")
		f.StringVar(&seed, "seed", "", "explicit seed directory")
	})
	if e != nil {
		return e
	}
	_ = f
	if arena == "" || project == "" || variant == "" || protocolPath == "" {
		return errors.New("--arena, --project, --variant, --protocol required")
	}
	if trial < 1 {
		trial = 1
	}
	var p Protocol
	if e = readYAML(protocolPath, &p); e != nil {
		return e
	}
	if actor == "" {
		return errors.New("--actor required (protocol fixes the runtime, not its local executable path)")
	}
	if p.Execution.MaximumSelfRepairIterations < 0 || p.Execution.MaximumSelfRepairIterations > 2 {
		return errors.New("protocol repair limit must be 0..2")
	}
	if p.Execution.TaskTimeoutSeconds <= 0 {
		return errors.New("protocol task timeout must be positive")
	}
	base := resolveProjectRoot(arena, p, project)
	if seed == "" {
		seed = resolveConfiguredSeed(arena, base, p, project, variant)
	}
	tsPath := filepath.Join(base, "task-set.yaml")
	if _, x := os.Stat(tsPath); x != nil {
		tsPath = filepath.Join(base, "tasks.yaml")
	}
	var ts taskSet
	if e = readYAML(tsPath, &ts); e != nil {
		return e
	}
	if len(ts.Tasks) == 0 {
		return errors.New("task set has no tasks")
	}
	freezePath := filepath.Join(arena, "freeze.yaml")
	var fr FreezeRecord
	if e = readYAML(freezePath, &fr); e != nil {
		return fmt.Errorf("load freeze record: %w", e)
	}
	if e = verifyFreeze(arena, fr); e != nil {
		return e
	}
	if len(p.Projects) > 0 {
		found := false
		for _, ref := range p.Projects {
			if ref.ID == project || filepath.Base(filepath.Clean(ref.Path)) == project {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("project %q is not listed in frozen protocol", project)
		}
	}
	variant = strings.ToLower(strings.TrimPrefix(variant, "arm-"))
	if variant != "a" && variant != "b" {
		return errors.New("--variant must identify arm A or B")
	}
	runID := fmt.Sprintf("%s-%s-t%02d", project, variant, trial)
	dest := filepath.Join(arena, "snapshots", runID)
	if _, e = os.Stat(dest); e == nil {
		return fmt.Errorf("snapshot already exists: %s", dest)
	}
	if e = copyTree(seed, dest); e != nil {
		return e
	}
	record := RunRecord{Version: 1, RunID: runID, Project: project, Variant: variant, Trial: trial, Model: p.Agent.Model, ReasoningEffort: p.Agent.ReasoningEffort, FreezeDigest: fr.Digest, StartedUTC: time.Now().UTC().Format(time.RFC3339)}
	runDir := filepath.Join(arena, "runs", runID)
	if e = os.MkdirAll(runDir, 0755); e != nil {
		return e
	}
	defer func() {
		record.EndedUTC = time.Now().UTC().Format(time.RFC3339)
		_ = writeYAMLNew(filepath.Join(runDir, "run.yaml"), record)
	}()
	for _, t := range ts.Tasks {
		t.AllowedPaths = allowedFor(t, variant)
		t.PublicValidators = validatorsFor(t, projectRef(p, project), variant)
		if len(t.PublicValidators) > 0 {
			t.ValidatorManifest = filepath.Join(arena, "raw", runID, t.ID, "validators.yaml")
			if e = os.MkdirAll(filepath.Dir(t.ValidatorManifest), 0755); e != nil {
				return e
			}
			manifest, er := yaml.Marshal(t.PublicValidators)
			if er != nil {
				return er
			}
			if e = writeFileNew(t.ValidatorManifest, manifest); e != nil && !os.IsExist(e) {
				return e
			}
		} else {
			return fmt.Errorf("task %s has no public validation commands for arm %s", t.ID, variant)
		}
		res := executeTask(actor, p, dest, runDir, t, runID, smoke)
		if !smoke {
			out := filepath.Join(arena, "results", runID, "snapshots", t.ID)
			if e = copyTreeMode(dest, out, true); e != nil {
				return fmt.Errorf("save task snapshot %s: %w", t.ID, e)
			}
		}
		record.Tasks = append(record.Tasks, res)
	}
	return nil
}
func executeTask(actor string, p Protocol, root, runDir string, t Task, runID string, smoke bool) taskResult {
	start := time.Now()
	rr := taskResult{TaskID: t.ID, Status: "failed", StartedUTC: start.UTC().Format(time.RFC3339)}
	seconds := p.Execution.TaskTimeoutSeconds
	if t.TimeoutSeconds > 0 && t.TimeoutSeconds < seconds {
		seconds = t.TimeoutSeconds
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()
	lastOutput := ""
	for attempt := 0; attempt <= p.Execution.MaximumSelfRepairIterations; attempt++ {
		if attempt > 0 {
			rr.Repairs++
		}
		prompt := t.Prompt + helperInstructions(t)
		if attempt > 0 {
			prompt += "\n\nThe public validator reported the following. Fix the issue, then stop.\n" + lastOutput
		}
		helpPath := filepath.Join(filepath.Dir(filepath.Dir(runDir)), "raw", runID, t.ID, "helper.jsonl")
		callsBefore, _, _, _ := helperResults(helpPath)
		stdoutPath := filepath.Join(runDir, fmt.Sprintf("%s.attempt-%d.stdout.jsonl", t.ID, attempt+1))
		stderrPath := filepath.Join(runDir, fmt.Sprintf("%s.attempt-%d.stderr.txt", t.ID, attempt+1))
		rr.ExitCode, rr.Error = invokeActor(ctx, actor, p, root, runID, t.ID, attempt+1, t.ValidatorManifest, prompt, smoke, stdoutPath, stderrPath)
		rr.StdoutFiles = append(rr.StdoutFiles, filepath.Base(stdoutPath))
		if tokens, ok := usageFromFile(stdoutPath); ok {
			if rr.ReportedTokens == nil {
				v := tokens
				rr.ReportedTokens = &v
			} else {
				*rr.ReportedTokens += tokens
			}
		}
		if rr.ExitCode != 0 {
			rr.Status = "failed"
			break
		}
		rr.Status = "completed"
		if smoke {
			break
		}
		calls, lastCode, output, _ := helperResults(helpPath)
		rr.ValidationCalls = calls
		if calls != callsBefore+1 {
			rr.ProtocolValid = false
			rr.Status = "protocol-invalid"
			break
		}
		rr.ProtocolValid = true
		rr.BeforeRepairAvailable = true
		rr.ValidatorExit = &lastCode
		lastOutput = output
		if lastCode == 0 {
			rr.Status = "completed"
			break
		}
		rr.Status = "invalid"
		if attempt == p.Execution.MaximumSelfRepairIterations || calls >= p.Execution.InitialValidationAttempts+p.Execution.MaximumSelfRepairIterations {
			break
		}
	}
	if ctx.Err() == context.DeadlineExceeded {
		rr.Status = "timeout"
		rr.Error = ctx.Err().Error()
	}
	rr.EndedUTC = time.Now().UTC().Format(time.RFC3339)
	return rr
}

func helperInstructions(t Task) string {
	if len(t.PublicValidators) == 0 && len(t.Validator) == 0 {
		return "\nBefore any public test or validator feedback, invoke gauntlet-check for this task. The helper snapshots the current patch before validation and records the actual result. Invoke it at most once in this actor turn. If it fails, stop and return; a fresh repair turn may follow. Do not inspect or modify hidden evaluation materials."
	}
	if t.ValidatorManifest != "" {
		return "\nBefore any public tests or repair, run `gauntlet check --task-id " + t.ID + " --manifest $env:GAUNTLET_PUBLIC_VALIDATORS`. The helper snapshots the workspace before validation and records the results. Invoke once per actor turn. If it fails, stop for a fresh repair turn."
	}
	return "\nBefore any public test or validator feedback, run `gauntlet check --task-id " + t.ID + " -- " + strings.Join(t.Validator, " ") + "`. This helper snapshots the current patch before validation and records the actual result. Invoke it at most once in this actor turn. If it fails, stop and return; a fresh repair turn may follow. Do not inspect or modify hidden evaluation materials."
}

type helperEvent struct {
	AtUTC        string     `json:"at_utc"`
	TaskID       string     `json:"task_id"`
	Commands     [][]string `json:"commands"`
	ExitCode     int        `json:"exit_code"`
	Snapshot     string     `json:"snapshot"`
	Output       string     `json:"output"`
	ActorAttempt int        `json:"actor_attempt"`
}

func helperResults(path string) (int, int, string, int) {
	f, e := os.Open(path)
	if e != nil {
		return 0, -1, "", 0
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 8*1024*1024)
	count, code, attempt := 0, -1, 0
	out := ""
	for s.Scan() {
		var event helperEvent
		if json.Unmarshal(s.Bytes(), &event) == nil {
			count++
			code = event.ExitCode
			out = event.Output
			attempt = event.ActorAttempt
		}
	}
	return count, code, out, attempt
}

func publicCheck(args []string) error {
	idx := -1
	for i, a := range args {
		if a == "--" {
			idx = i
			break
		}
	}
	pre := args
	if idx >= 0 {
		pre = args[:idx]
	}
	taskID := ""
	manifestPath := ""
	f := flag.NewFlagSet("check", flag.ContinueOnError)
	f.StringVar(&taskID, "task-id", "", "task id")
	f.StringVar(&manifestPath, "manifest", "", "YAML list of public validator argv arrays")
	if e := f.Parse(pre); e != nil {
		return e
	}
	if taskID == "" {
		return errors.New("--task-id required")
	}
	var vm ValidatorManifest
	if manifestPath != "" {
		if e := readYAML(manifestPath, &vm); e != nil {
			return e
		}
	} else if idx >= 0 && idx+1 < len(args) {
		vm.Commands = append(vm.Commands, args[idx+1:])
	}
	commands := vm.Commands
	if len(commands) == 0 {
		return errors.New("check requires --manifest PATH or -- COMMAND [ARGS...]")
	}
	record := os.Getenv("GAUNTLET_HELPER_RECORD")
	runID := os.Getenv("GAUNTLET_RUN_ID")
	if record == "" || runID == "" {
		return errors.New("gauntlet check is available only inside an instrumented actor run")
	}
	count, _, _, priorAttempt := helperResults(record)
	attempt := 0
	_, _ = fmt.Sscanf(os.Getenv("GAUNTLET_ACTOR_ATTEMPT"), "%d", &attempt)
	if count >= 1 || (attempt > 0 && priorAttempt == attempt) || attempt > 3 {
		return errors.New("public validation budget exhausted (one call per actor turn, maximum three calls)")
	}
	root, e := os.Getwd()
	if e != nil {
		return e
	}
	checkOrdinal := attempt
	if checkOrdinal == 0 {
		checkOrdinal = count + 1
	}
	snapshot := filepath.Join(os.Getenv("GAUNTLET_SNAPSHOT_ROOT"), fmt.Sprintf("%s-check-%d", taskID, checkOrdinal))
	if os.Getenv("GAUNTLET_SNAPSHOT_ROOT") == "" {
		snapshot = filepath.Join(filepath.Dir(record), "snapshots", fmt.Sprintf("%s-check-%d", taskID, checkOrdinal))
	}
	if e = capturePatch(root, snapshot); e != nil {
		return e
	}
	var output strings.Builder
	code := 0
	for _, command := range commands {
		if len(command) == 0 {
			continue
		}
		fmt.Fprintf(&output, "$ %s\n", strings.Join(command, " "))
		resolved := append([]string(nil), command...)
		candidate := ""
		if strings.Contains(strings.Join(command, " "), "${CANDIDATE_REVISION}") {
			git := exec.Command("git", "rev-parse", "HEAD")
			git.Dir = root
			if bytes, ge := git.Output(); ge == nil {
				candidate = strings.TrimSpace(string(bytes))
			}
			if candidate == "" {
				code = 1
				output.WriteString("validator requires a committed candidate revision, but HEAD is unavailable\n")
				continue
			}
		}
		for i, arg := range resolved {
			arg = strings.ReplaceAll(arg, "${CANDIDATE_REVISION}", candidate)
			arg = strings.ReplaceAll(arg, "${BASE_REVISION}", vm.BaseRevision)
			resolved[i] = arg
		}
		fmt.Fprintf(&output, "resolved: %s\n", strings.Join(resolved, " "))
		cmd := exec.Command(resolved[0], resolved[1:]...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		output.Write(out)
		if err != nil {
			code = 1
		}
	}
	fmt.Print(output.String())
	if path := os.Getenv("GAUNTLET_PUBLIC_VALIDATOR_OUTPUT"); path != "" {
		if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
			return e
		}
		if e := os.WriteFile(path, []byte(output.String()), 0644); e != nil {
			return e
		}
	}
	event := helperEvent{AtUTC: time.Now().UTC().Format(time.RFC3339), TaskID: taskID, Commands: commands, ExitCode: code, Snapshot: snapshot, Output: output.String(), ActorAttempt: attempt}
	if er := os.MkdirAll(filepath.Dir(record), 0755); er != nil {
		return er
	}
	file, er := os.OpenFile(record, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if er != nil {
		return er
	}
	je := json.NewEncoder(file).Encode(event)
	ce := file.Close()
	if je != nil {
		return je
	}
	if ce != nil {
		return ce
	}
	if code != 0 {
		return errors.New("one or more public validators failed")
	}
	return nil
}

func invokeActor(ctx context.Context, actor string, p Protocol, root, runID, taskID string, actorAttempt int, validatorManifest string, prompt string, smoke bool, stdoutPath, stderrPath string) (int, string) {
	args := []string{"exec"}
	if p.Agent.IgnoreUserConfig {
		args = append(args, "--ignore-user-config")
	}
	if p.Agent.Ephemeral {
		args = append(args, "--ephemeral")
	}
	args = append(args, "--json", "--model", p.Agent.Model, "-c", "model_reasoning_effort=\""+p.Agent.ReasoningEffort+"\"")
	args = append(args, "-c", "approval_policy=\""+p.Agent.ApprovalPolicy+"\"", "-c", "web_search=\""+p.Agent.WebSearch+"\"")
	evidenceDir := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(stdoutPath))), "raw", runID, taskID)
	args = append(args, "--sandbox", p.Agent.Sandbox, "--add-dir", evidenceDir, "-C", root, "-")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" && (strings.HasSuffix(strings.ToLower(actor), ".cmd") || strings.HasSuffix(strings.ToLower(actor), ".bat")) {
		cmd = exec.CommandContext(ctx, "cmd.exe", append([]string{"/d", "/c", actor}, args...)...)
	} else {
		cmd = exec.CommandContext(ctx, actor, args...)
	}
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(prompt)
	stdout, e := os.Create(stdoutPath)
	if e != nil {
		return -1, e.Error()
	}
	defer stdout.Close()
	stderr, e := os.Create(stderrPath)
	if e != nil {
		return -1, e.Error()
	}
	defer stderr.Close()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "RUST_LOG=off", "GAUNTLET_RUN_ID="+runID, "GAUNTLET_TASK_ID="+taskID, "GAUNTLET_ACTOR_ATTEMPT="+fmt.Sprint(actorAttempt), "GAUNTLET_HELPER_RECORD="+filepath.Join(evidenceDir, "helper.jsonl"), "GAUNTLET_SNAPSHOT_ROOT="+filepath.Join(evidenceDir, "snapshots"), "GAUNTLET_PUBLIC_VALIDATORS="+validatorManifest, "GAUNTLET_PUBLIC_VALIDATOR_OUTPUT="+filepath.Join(evidenceDir, taskID+".public-validator.txt"))
	if e = cmd.Run(); e != nil {
		var ee *exec.ExitError
		if errors.As(e, &ee) {
			return ee.ExitCode(), e.Error()
		}
		return -1, e.Error()
	}
	return 0, ""
}

func runValidator(root, runDir, taskID string, validator []string) (int, error) {
	v := exec.Command(validator[0], validator[1:]...)
	v.Dir = root
	f, e := os.Create(filepath.Join(runDir, taskID+".public-validator.txt"))
	if e != nil {
		return 1, e
	}
	v.Stdout = f
	v.Stderr = f
	e = v.Run()
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return 1, e
	}
	return 0, nil
}

func capturePatch(root, dest string) error {
	manifest := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		if excludedWorkspacePath(rel) {
			return nil
		}
		sum, e := hashFile(path)
		if e != nil {
			return e
		}
		manifest[filepath.ToSlash(rel)] = sum
		return nil
	})
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dest, "tree.yaml"), b, 0644); err != nil {
		return err
	}
	// Keep the exact pre-validation workspace bytes alongside the manifest.
	return copyTreeMode(root, filepath.Join(dest, "workspace"), true)
}
func copyTree(src, dst string) error {
	return copyTreeMode(src, dst, false)
}
func copyTreeMode(src, dst string, preserveGit bool) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, e := filepath.Rel(src, p)
		if e != nil {
			return e
		}
		if rel != "." && excludedWorkspacePath(rel) && !(preserveGit && strings.Split(filepath.ToSlash(rel), "/")[0] == ".git") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink seed input unsupported: %s", p)
		}
		in, e := os.Open(p)
		if e != nil {
			return e
		}
		defer in.Close()
		if e = os.MkdirAll(filepath.Dir(target), 0755); e != nil {
			return e
		}
		out, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		_, e = io.Copy(out, in)
		ce := out.Close()
		if e != nil {
			return e
		}
		return ce
	})
}

func excludedWorkspacePath(rel string) bool {
	p := strings.ToLower(filepath.ToSlash(rel))
	if p == ".git" || strings.HasPrefix(p, ".git/") {
		return true
	}
	for _, generated := range []string{"src/commerce/bin", "src/commerce/obj", "checks/bin", "checks/obj"} {
		if p == generated || strings.HasPrefix(p, generated+"/") {
			return true
		}
	}
	return false
}

func initializeSeedRepository(root string) (string, error) {
	commands := [][]string{{"git", "init", "-b", "codex/gauntlet-seed"}, {"git", "config", "core.autocrlf", "false"}, {"git", "add", "-f", "-A"}, {"git", "-c", "user.name=Gauntlet Operator", "-c", "user.email=gauntlet@invalid", "commit", "--allow-empty", "-m", "Frozen seed"}}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("initialize isolated seed repository (%s): %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
	}
	return gitHead(root)
}
func gitHead(root string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read seed revision: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

func collect(args []string) error  { return aggregate(args, "collect") }
func evaluate(args []string) error { return aggregate(args, "evaluate") }
func aggregate(args []string, mode string) error {
	var arena, project, evaluator, taskFilter string
	var includeHoldout bool
	var f *flag.FlagSet
	var e error
	f, e = mustFlags(mode, args, func(f *flag.FlagSet) {
		f.StringVar(&arena, "arena", "", "arena root")
		f.StringVar(&project, "project", "", "project filter")
		f.StringVar(&evaluator, "evaluator", "", "sealed evaluator executable")
		f.StringVar(&taskFilter, "task", "", "evaluate only one task ID")
		f.BoolVar(&includeHoldout, "include-holdout", false, "open task 09-12 outcomes after the fix decision")
	})
	if e != nil {
		return e
	}
	_ = f
	if arena == "" {
		return errors.New("--arena required")
	}
	if includeHoldout {
		var gate HoldoutDecision
		if e = readYAML(filepath.Join(arena, "decisions", "holdout.yaml"), &gate); e != nil {
			return errors.New("cannot open holdout evaluation before the fix/no-fix decision is recorded")
		}
		if gate.Decision != "fix" && gate.Decision != "no-fix" {
			return errors.New("invalid holdout decision record")
		}
	}
	files, _ := filepath.Glob(filepath.Join(arena, "runs", "*", "run.yaml"))
	sort.Strings(files)
	count := 0
	for _, path := range files {
		var r RunRecord
		if e = readYAML(path, &r); e != nil {
			return e
		}
		if project != "" && r.Project != project {
			continue
		}
		count++
		if mode == "evaluate" && evaluator != "" {
			snapBase := filepath.Join(arena, "results", r.RunID, "snapshots")
			for _, tr := range r.Tasks {
				if taskFilter != "" && tr.TaskID != taskFilter {
					continue
				}
				if !includeHoldout && isHoldoutID(tr.TaskID) {
					continue
				}
				snap := filepath.Join(snapBase, tr.TaskID)
				if _, er := os.Stat(filepath.Join(snap, "workspace")); er == nil {
					snap = filepath.Join(snap, "workspace")
				}
				out := filepath.Join(filepath.Dir(path), tr.TaskID+".evaluation.yaml")
				if _, e = os.Stat(out); e == nil {
					continue
				}
				args := []string{"--repo", snap, "--task", tr.TaskID, "--output", out}
				c := exec.Command(evaluator, args...)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if e = c.Run(); e != nil {
					fmt.Fprintf(os.Stderr, "evaluator failed %s/%s: %v\n", r.RunID, tr.TaskID, e)
				}
			}
		}
	}
	if mode == "evaluate" || mode == "collect" {
		nativeFiles, _ := filepath.Glob(filepath.Join(arena, "runs", "*", "*.native.yaml"))
		sort.Strings(nativeFiles)
		for _, path := range nativeFiles {
			var state NativeTaskRecord
			if e = readYAML(path, &state); e != nil {
				return e
			}
			if project != "" && state.Project != project {
				continue
			}
			if mode == "collect" {
				count++
				continue
			}
			if taskFilter != "" && state.TaskID != taskFilter {
				continue
			}
			if !includeHoldout && isHoldoutID(state.TaskID) {
				continue
			}
			if state.FinalSnapshot == "" {
				continue
			}
			snap := filepath.Join(arena, state.FinalSnapshot, "workspace")
			out := filepath.Join(filepath.Dir(path), state.TaskID+".evaluation.yaml")
			if _, e = os.Stat(out); e == nil {
				continue
			}
			cmd, er := configuredEvaluator(arena, state, snap, out, evaluator)
			if er != nil {
				return er
			}
			if len(cmd) == 0 {
				continue
			}
			c := exec.Command(cmd[0], cmd[1:]...)
			c.Dir = filepath.Join(arena, "projects", state.Project)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if e = c.Run(); e != nil {
				fmt.Fprintf(os.Stderr, "evaluator failed %s/%s: %v\n", state.RunID, state.TaskID, e)
			}
		}
	}
	fmt.Printf("%s processed %d run/task records\n", mode, count)
	return nil
}

func configuredEvaluator(arena string, state NativeTaskRecord, repo, output, override string) ([]string, error) {
	if override != "" {
		taskID := state.EvaluationTaskID
		if taskID == "" {
			taskID = state.TaskID
		}
		return []string{override, "--repo", repo, "--task", taskID, "--output", output}, nil
	}
	projectRoot := filepath.Join(arena, "projects", state.Project)
	var project ProjectManifest
	_ = readYAML(filepath.Join(projectRoot, "project.yaml"), &project)
	var tasks taskSet
	taskPath := filepath.Join(projectRoot, "task-set.yaml")
	if project.Shared.TaskSet != "" {
		taskPath = filepath.Join(projectRoot, filepath.FromSlash(project.Shared.TaskSet))
	}
	if project.Controls.TaskSet != "" {
		taskPath = filepath.Join(projectRoot, filepath.FromSlash(project.Controls.TaskSet))
	}
	if project.TaskSet != "" {
		taskPath = filepath.Join(projectRoot, filepath.FromSlash(project.TaskSet))
	}
	_ = readYAML(taskPath, &tasks)
	config := project.Evaluator
	if len(config.Command) == 0 {
		config = tasks.Evaluator
	}
	if len(config.Command) == 0 {
		return nil, nil
	}
	command := append([]string(nil), config.Command...)
	taskID := state.EvaluationTaskID
	if taskID == "" {
		taskID = state.TaskID
	}
	for i, arg := range command {
		if i > 0 && strings.EqualFold(command[i-1], "-Output") && (arg == "record.yaml" || arg == "record.yml") {
			arg = output
		}
		arg = strings.ReplaceAll(arg, "<arm-repo>", repo)
		arg = strings.ReplaceAll(arg, "${REPO}", repo)
		arg = strings.ReplaceAll(arg, "<task-id>", taskID)
		arg = strings.ReplaceAll(arg, "${TASK}", taskID)
		arg = strings.ReplaceAll(arg, "<base-sha>", state.BaseRevision)
		arg = strings.ReplaceAll(arg, "${BASE}", state.BaseRevision)
		arg = strings.ReplaceAll(arg, "<prior-tasks-through>", state.PriorTasksThrough)
		arg = strings.ReplaceAll(arg, "${OUTPUT}", output)
		response := filepath.Join(arena, "runs", state.RunID, state.ResponseFile)
		arg = strings.ReplaceAll(arg, "${RESPONSE}", response)
		if (strings.HasSuffix(strings.ToLower(arg), ".ps1") || strings.HasSuffix(strings.ToLower(arg), ".py")) && !filepath.IsAbs(arg) {
			arg = filepath.Join(projectRoot, filepath.FromSlash(arg))
		}
		command[i] = arg
	}
	if len(command) == 1 && strings.HasSuffix(strings.ToLower(command[0]), ".ps1") {
		command = []string{"powershell", "-NoProfile", "-File", command[0]}
	}
	joined := strings.Join(command, " ")
	if !strings.Contains(joined, "-Repo") && !strings.Contains(joined, "--repo") {
		command = append(command, "-Repo", repo)
	}
	if !strings.Contains(joined, "-Task") && !strings.Contains(joined, "--task") {
		command = append(command, "-Task", taskID)
	}
	if !strings.Contains(joined, "-Output") && !strings.Contains(joined, "--output") {
		command = append(command, "-Output", output)
	}
	if state.BaseRevision != "" && !strings.Contains(joined, "-Base") {
		command = append(command, "-Base", state.BaseRevision)
	}
	if state.PriorTasksThrough != "" && !strings.Contains(joined, "-PriorTasksThrough") {
		command = append(command, "-PriorTasksThrough", state.PriorTasksThrough)
	}
	return command, nil
}
func isHoldoutID(id string) bool {
	for _, suffix := range []string{"09", "10", "11", "12"} {
		if id == suffix || strings.HasSuffix(id, "-"+suffix) {
			return true
		}
	}
	return false
}
func summarize(args []string) error {
	var arena, out string
	f, e := mustFlags("summarize", args, func(f *flag.FlagSet) {
		f.StringVar(&arena, "arena", "", "arena root")
		f.StringVar(&out, "out", "", "output JSON path")
	})
	if e != nil {
		return e
	}
	_ = f
	if arena == "" {
		return errors.New("--arena required")
	}
	if out == "" {
		out = filepath.Join(arena, "raw", "summary.json")
	}
	files, _ := filepath.Glob(filepath.Join(arena, "runs", "*", "run.yaml"))
	sort.Strings(files)
	var all []RunRecord
	for _, p := range files {
		var r RunRecord
		if e = readYAML(p, &r); e != nil {
			return e
		}
		all = append(all, r)
	}
	var native []NativeTaskRecord
	nativeFiles, _ := filepath.Glob(filepath.Join(arena, "runs", "*", "*.native.yaml"))
	sort.Strings(nativeFiles)
	for _, p := range nativeFiles {
		var state NativeTaskRecord
		if e = readYAML(p, &state); e != nil {
			return e
		}
		native = append(native, state)
	}
	b, e := json.MarshalIndent(struct {
		Runs        []RunRecord        `json:"runs"`
		NativeTasks []NativeTaskRecord `json:"native_tasks"`
	}{all, native}, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(out, b, 0644)
}

// parseJSONLUsage returns only explicit completed-turn usage events; absent data stays absent.
func parseJSONLUsage(r io.Reader) (int64, bool) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 4096), 4*1024*1024)
	var n int64
	ok := false
	for s.Scan() {
		var e map[string]any
		if json.Unmarshal(s.Bytes(), &e) != nil {
			continue
		}
		typ, _ := e["type"].(string)
		if typ != "turn.completed" {
			continue
		}
		u, yes := e["usage"].(map[string]any)
		if !yes {
			continue
		}
		if x, yes := u["total_tokens"].(float64); yes {
			n += int64(x)
			ok = true
		}
	}
	return n, ok
}

func usageFromFile(path string) (int64, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	return parseJSONLUsage(f)
}
