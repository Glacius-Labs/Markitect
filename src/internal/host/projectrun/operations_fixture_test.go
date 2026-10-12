package projectrun

import (
	"encoding/json"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

const operationsVerifyHelperEnv = "MARKITECT_OPERATIONS_VERIFY_HELPER"
const operationsVerifyLogEnv = "MARKITECT_OPERATIONS_VERIFY_LOG"
const operationsTaskHelperEnv = "MARKITECT_OPERATIONS_TASK_HELPER"
const operationsSourceChangeEnv = "MARKITECT_OPERATIONS_SOURCE_CHANGE"

// This subprocess implements the bounded typed audit protocol without invoking
// a model. It records every Manager reached and deliberately fails Inventory
// when its supplied snapshot contains the drift marker.
func TestOperationsFullVerifyProcess(t *testing.T) {
	if path := os.Getenv(operationsVerifyLogEnv); path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err == nil {
			_ = json.NewEncoder(file).Encode(map[string]string{"helperStarted": os.Getenv(operationsVerifyHelperEnv)})
			_ = file.Close()
		}
	}
	if os.Getenv(operationsVerifyHelperEnv) != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		t.Fatal(err)
	}
	var discriminator struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &discriminator); err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv(operationsVerifyLogEnv); path != "" {
		file, openErr := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if openErr != nil {
			t.Fatal(openErr)
		}
		_ = json.NewEncoder(file).Encode(map[string]string{"kind": discriminator.Kind})
		_ = file.Close()
	}
	if discriminator.Kind == "projectrun-review/v1" {
		report, err := json.Marshal(reviewResponse{Status: "pass", Summary: "no-op is supported by the complete accepted scope", Findings: []reviewFindingResponse{}})
		if err != nil {
			t.Fatal(err)
		}
		usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
		encoded, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
			Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
			CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
			ReportJSON: report, Uncertainty: []string{}, Usage: usage})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = os.Stdout.Write(append(encoded, '\n'))
		os.Exit(0)
	}
	var request struct {
		Kind    string `json:"kind"`
		Manager struct {
			Manager struct {
				ID string `json:"id"`
			} `json:"manager"`
		} `json:"manager"`
		Subjects   []string                        `json:"requiredSubjects"`
		Strictness projectconfig.StrictnessProfile `json:"strictness"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &request); err != nil || request.Kind != "projectrun-full-verify/v1" {
		t.Fatalf("unexpected full-verify request: %s (%v)", invocation.Request.Context, err)
	}
	if path := os.Getenv(operationsVerifyLogEnv); path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(file).Encode(map[string]string{"managerId": request.Manager.Manager.ID})
		_ = file.Close()
	}
	status := "pass"
	assessments := make([]FullAssessment, 0, len(request.Subjects))
	findings := []string{}
	if request.Manager.Manager.ID == e2eManagerID("inventory", "inventory") {
		for _, artifact := range invocation.Request.Artifacts {
			if strings.Contains(string(artifact.Content), "UNIMPACTED_DRIFT") {
				status = "fail"
				findings = append(findings, "unimpacted inventory artifact contains drift")
				break
			}
		}
	}
	for _, subject := range request.Subjects {
		outcome := "pass"
		if status == "fail" && len(assessments) == 0 {
			outcome = "fail"
		}
		assessments = append(assessments, FullAssessment{Subject: subject, Outcome: outcome, Detail: "bounded snapshot audit"})
	}
	counterexamples := make([]FullCounterexample, 0, request.Strictness.Counterexamples)
	for i := 0; i < request.Strictness.Counterexamples; i++ {
		ref := "manager:" + request.Manager.Manager.ID
		if len(request.Subjects) > 0 {
			ref = request.Subjects[i%len(request.Subjects)]
		}
		counterexamples = append(counterexamples, FullCounterexample{Expected: "accepted obligation " + string(rune('a'+i)), Observed: "fixture inspection evidence " + string(rune('a'+i)), EvidenceRefs: []string{ref}})
	}
	report, err := json.Marshal(fullAuditResponse{Status: status, Summary: "bounded fixture audit", Assessments: assessments, Findings: findings, Counterexamples: counterexamples})
	if err != nil {
		t.Fatal(err)
	}
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
	response, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: report, Uncertainty: []string{}, Usage: usage})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(response, '\n'))
	os.Exit(0)
}

// This helper returns a justified no-op from each leaf Manager while the root
// still delegates and performs an explicit integration phase.
func TestOperationsNoOpTaskProcess(t *testing.T) {
	if os.Getenv(operationsTaskHelperEnv) != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		t.Fatal(err)
	}
	var request struct {
		Kind           string   `json:"kind"`
		Phase          string   `json:"phase"`
		DirectChildren []string `json:"directChildren"`
		Manager        struct {
			Manager struct {
				ID string `json:"id"`
			} `json:"manager"`
		} `json:"manager"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &request); err != nil || request.Kind != "projectrun-task/v1" {
		t.Fatalf("unexpected task request: %s (%v)", invocation.Request.Context, err)
	}
	response := TaskResponse{Status: "no-op", Summary: "The complete owned realization was reviewed; no warranted change was found.",
		Delegations: []Delegation{}, ReworkRequests: []ReworkRequest{}, Integrated: request.Phase == "integrate",
		Questions: []string{}, Risks: []string{}, ResolvedQuestions: []string{}, ResolvedRisks: []string{}}
	if request.Phase == "work" && request.Manager.Manager.ID == e2eManagerID("", "project-owner") {
		response.Status = "complete"
		for _, child := range request.DirectChildren {
			response.Delegations = append(response.Delegations, Delegation{ManagerID: child, Goal: "Review your complete owned responsibility and report whether a change is warranted."})
		}
	}
	if request.Phase == "integrate" {
		response.Status = "complete"
		response.Summary = "Reviewed child reports and integrated the no-op findings."
	}
	candidateFiles := []agentexec.CandidateFile{}
	if os.Getenv(operationsSourceChangeEnv) == "1" && request.Phase == "work" && request.Manager.Manager.ID == e2eManagerID("orders", "orders") {
		response.Status = "complete"
		response.Summary = "Improved the Orders implementation while preserving its accepted contract."
		candidateFiles = []agentexec.CandidateFile{{Path: "src/orders/implementation.txt", Mode: "0644", Content: "orders implementation v2\n"}}
	}
	report, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
	encoded, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: candidateFiles, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: report, Uncertainty: []string{}, Usage: usage})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(encoded, '\n'))
	os.Exit(0)
}

func setupOperationsFullVerifyChildren(t *testing.T) {
	t.Helper()
	setRuntimeFixtureEnv(t, fullVerifyHelperEnv, "1")
	setRuntimeFixtureEnv(t, fullVerifyCheckEnv, "1")
	setRuntimeFixtureEnv(t, fullVerifyLogEnv, filepath.Join(testkit.TempDir(t), "full-verify.jsonl"))
}

func configureOperationsNoOpRuntime(t *testing.T, root string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	setRuntimeFixtureEnv(t, operationsTaskHelperEnv, "1")
	setRuntimeFixtureEnv(t, operationsVerifyHelperEnv, "1")
	setRuntimeFixtureEnv(t, operationsVerifyLogEnv, filepath.Join(testkit.TempDir(t), "operations-review.jsonl"))
	setupOperationsFullVerifyChildren(t)
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
		for id, agent := range runtime.Agents {
			agent.Command = executable
			agent.Args = []string{"-test.run=^TestOperationsNoOpTaskProcess$"}
			agent.Environment = append(agent.Environment, operationsTaskHelperEnv, operationsSourceChangeEnv)
			runtime.Agents[id] = agent
		}
		reviewers := make(map[string]projectconfig.Agent, len(runtime.Agents))
		for id, agent := range runtime.Agents {
			agent.Args = []string{"-test.run=^TestOperationsFullVerifyProcess$"}
			agent.Environment = append(agent.Environment, operationsVerifyHelperEnv, operationsVerifyLogEnv)
			reviewers[id] = agent
		}
		runtime.Review = &projectconfig.ReviewConfig{Agents: reviewers, MaxRounds: 1, MaxManagerRounds: 1}
		runtime.Strictness = &projectconfig.StrictnessConfig{Default: projectconfig.StrictnessProfile{Evidence: []string{"project checks reviewed"}, Counterexamples: 1},
			Managers: map[string]projectconfig.StrictnessProfile{e2eManagerID("orders", "orders"): {Evidence: []string{"orders contract review"}, Counterexamples: 1}}}
	})
}

func configureOperationsFullVerify(t *testing.T, root string) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(testkit.TempDir(t), "operations-full-verify.jsonl")
	setRuntimeFixtureEnv(t, operationsVerifyHelperEnv, "1")
	setRuntimeFixtureEnv(t, operationsVerifyLogEnv, logPath)
	setupOperationsFullVerifyChildren(t)
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
		reviewers := make(map[string]projectconfig.Agent, len(runtime.Agents))
		for id, agent := range runtime.Agents {
			agent.Command = executable
			agent.Args = []string{"-test.run=^TestOperationsFullVerifyProcess$"}
			agent.Environment = append(agent.Environment, operationsVerifyHelperEnv, operationsVerifyLogEnv)
			reviewers[id] = agent
		}
		runtime.Review = &projectconfig.ReviewConfig{Agents: reviewers, MaxRounds: 1, MaxManagerRounds: 1}
	})
	return logPath
}
