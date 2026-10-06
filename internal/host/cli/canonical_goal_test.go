package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
)

const canonicalGoalHelperEnv = "MARKITECT_CANONICAL_GOAL_CLI_HELPER"

func TestCanonicalGoalCLIHelperProcess(t *testing.T) {
	if os.Getenv(canonicalGoalHelperEnv) != "1" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		fmt.Fprintln(os.Stderr, "invalid invocation")
		os.Exit(2)
	}
	var contextValue struct {
		Phase string `json:"phase"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextValue); err != nil {
		fmt.Fprintln(os.Stderr, "invalid context")
		os.Exit(2)
	}
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest,
		Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{},
		EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		Uncertainty: []string{"Protocol tests do not assess provider quality."},
	}
	switch contextValue.Phase {
	case "recommend-modules":
		id := invocation.Request.ScopeIDs[0]
		response.CandidateJSON = json.RawMessage(fmt.Sprintf(`{"recommendations":[{"id":%q,"basis":"The supplied schema may express the goal.","uncertainty":["Owner review is required."]}],"uncertainty":["No provider-quality claim is made."]}`, id))
	case "model-from-selected-modules":
		property := `"name":"Orders"`
		if os.Getenv("MARKITECT_CANONICAL_GOAL_INVALID") == "1" {
			property = `"undeclared":"Orders"`
		}
		response.CandidateJSON = json.RawMessage(`{"definitions":[{"apiVersion":"goals.example.org/v1","kind":"Module","metadata":{"name":"orders","namespace":"commerce"},"purpose":"Own order placement intent.","spec":{` + property + `}}],"summary":"A bounded order module proposal.","uncertainty":["The owner must decide whether payment belongs here."]}`)
	default:
		fmt.Fprintln(os.Stderr, "unexpected phase")
		os.Exit(2)
	}
	_ = json.NewEncoder(os.Stdout).Encode(response)
	os.Exit(0)
}

func TestCanonicalGoalCLIRecommendThenProposeReturnsFlatJSONCandidate(t *testing.T) {
	t.Setenv(canonicalGoalHelperEnv, "1")
	t.Setenv("MARKITECT_CANONICAL_GOAL_INVALID", "")
	t.Setenv("MARKITECT_AGENT_PRIVATE_LOG", "")
	files := writeCanonicalGoalCLIInputs(t)
	var recommendationOutput bytes.Buffer
	if code := runCanonicalGoal("goal-recommend", files.goal, "", "", files.runtime, &recommendationOutput); code != 0 {
		t.Fatalf("goal-recommend exit %d: %s", code, recommendationOutput.String())
	}
	var recommendation map[string]any
	if err := json.Unmarshal(recommendationOutput.Bytes(), &recommendation); err != nil {
		t.Fatal(err)
	}
	if recommendation["status"] != "proposed" || recommendation["adopted"] != false || recommendation["accepted"] != false {
		t.Fatalf("recommendation result is not a noncanonical proposal: %v", recommendation)
	}
	if err := os.WriteFile(files.recommendations, recommendationOutput.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	items, ok := recommendation["recommendations"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("recommendation list missing: %v", recommendation)
	}
	first, _ := items[0].(map[string]any)
	selectedID, _ := first["id"].(string)
	if selectedID == "" {
		t.Fatalf("recommendation has no exact supplied ID: %v", first)
	}
	selection, _ := json.Marshal(map[string][]string{"selectedRecommendationIds": {selectedID}})
	if err := os.WriteFile(files.selection, selection, 0600); err != nil {
		t.Fatal(err)
	}
	var proposalOutput bytes.Buffer
	if code := runCanonicalGoal("goal-propose", files.goal, files.recommendations, files.selection, files.runtime, &proposalOutput); code != 0 {
		t.Fatalf("goal-propose exit %d: %s", code, proposalOutput.String())
	}
	var proposal map[string]any
	if err := json.Unmarshal(proposalOutput.Bytes(), &proposal); err != nil {
		t.Fatal(err)
	}
	if proposal["status"] != "proposed" || proposal["adopted"] != false || proposal["accepted"] != false || proposal["candidateDigest"] == "" {
		t.Fatalf("proposal result is not explicitly noncanonical and bound: %v", proposal)
	}
	if candidate, ok := proposal["candidateJson"].(string); !ok || !strings.Contains(candidate, `"kind":"Module"`) {
		t.Fatalf("exact candidate JSON bytes were not returned: %v", proposal["candidateJson"])
	}
	if _, ok := proposal["receipt"].(map[string]any); !ok {
		t.Fatalf("flat proposal result omitted runner receipt: %v", proposal)
	}
}

func TestCanonicalGoalCLIReportsPartialReceiptForInvalidDefinition(t *testing.T) {
	t.Setenv(canonicalGoalHelperEnv, "1")
	t.Setenv("MARKITECT_CANONICAL_GOAL_INVALID", "1")
	t.Setenv("MARKITECT_AGENT_PRIVATE_LOG", "")
	files := writeCanonicalGoalCLIInputs(t)
	var recommendationOutput bytes.Buffer
	if code := runCanonicalGoal("goal-recommend", files.goal, "", "", files.runtime, &recommendationOutput); code != 0 {
		t.Fatalf("goal-recommend exit %d: %s", code, recommendationOutput.String())
	}
	if err := os.WriteFile(files.recommendations, recommendationOutput.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	var recommendation map[string]any
	if err := json.Unmarshal(recommendationOutput.Bytes(), &recommendation); err != nil {
		t.Fatal(err)
	}
	items := recommendation["recommendations"].([]any)
	selectedID := items[0].(map[string]any)["id"].(string)
	selection, _ := json.Marshal(map[string][]string{"selectedRecommendationIds": {selectedID}})
	if err := os.WriteFile(files.selection, selection, 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if code := runCanonicalGoal("goal-propose", files.goal, files.recommendations, files.selection, files.runtime, &output); code != 1 {
		t.Fatalf("invalid candidate exit = %d; output: %s", code, output.String())
	}
	var result map[string]any
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "invalid" || result["receipt"] == nil || result["error"] == nil || result["adopted"] != false {
		t.Fatalf("partial invalid result omitted receipts or boundary flags: %v", result)
	}
}

type canonicalGoalCLIPaths struct {
	goal, recommendations, selection, runtime string
}

func writeCanonicalGoalCLIInputs(t *testing.T) canonicalGoalCLIPaths {
	t.Helper()
	root := t.TempDir()
	paths := canonicalGoalCLIPaths{
		goal: filepath.Join(root, "goal.json"), recommendations: filepath.Join(root, "recommendations.json"),
		selection: filepath.Join(root, "selection.json"), runtime: filepath.Join(root, "runtime.json"),
	}
	goal := map[string]any{
		"goal":                   "Plan a small order management application with clear module ownership.",
		"decisionReferenceClaim": "owner-review/T-104",
		"packages": []any{map[string]any{
			"manifest": `apiVersion: markitect.example.org/module/v1alpha1
name: goal-cli-schema
version: 1.0.0
type: schema
purpose: Provides a test Schema.
requires:
  core: "1"
  modules: []
provides:
  schemas:
    - schema.yaml
  projectors: []
`,
			"files": map[string]string{"schema.yaml": `apiVersion: goals.example.org/v1
purpose: Describes a bounded application module.
kinds:
  Module:
    purpose: One independently named module.
    properties:
      name:
        purpose: Display name.
        type: string
        minCount: 1
        maxCount: 1
`},
		}},
	}
	if err := writeCanonicalGoalCLIJSON(paths.goal, goal); err != nil {
		t.Fatal(err)
	}
	runtime := brownfieldInferenceRuntime{
		APIVersion: brownfieldInferenceRuntimeAPIVersion,
		Agent: host.CanonicalRunnerConfig{
			Command: os.Args[0], Args: []string{"-test.run=^TestCanonicalGoalCLIHelperProcess$"},
			Model: "test-helper-model", ModelOptions: json.RawMessage(`{"test":true}`),
			ProviderVersion: "canonical-goal-test-helper/1", TimeoutSeconds: 3,
			MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
		},
		TempParent: t.TempDir(), PrivateLogs: filepath.Join(t.TempDir(), "private-logs"),
	}
	if err := writeCanonicalGoalCLIJSON(paths.runtime, runtime); err != nil {
		t.Fatal(err)
	}
	return paths
}

func writeCanonicalGoalCLIJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func TestCanonicalGoalMalformedInputDoesNotExitSuccessfully(t *testing.T) {
	var out bytes.Buffer
	if code := runCanonicalGoal("goal-recommend", "missing-goal.json", "", "", "missing-runtime.json", &out); code != 2 {
		t.Fatalf("malformed input returned %d: %s", code, out.String())
	}
}

func TestCanonicalGoalCommandHasNoImplicitRepositoryInputs(t *testing.T) {
	for _, extra := range [][]string{nil, {"--repo", "."}, {"--revision", strings.Repeat("a", 40)}, {"--config", "canonical.yaml"}, {"--write"}} {
		args := append([]string{"canonical", "--action", "goal-recommend", "--goal-input", "goal.json", "--runtime", "runtime.json"}, extra...)
		flags, _ := commandFlags("canonical")
		var out, errs bytes.Buffer
		_, _, done := parseOptions("canonical", args, flags, &out, &errs)
		if done != (len(extra) > 0) {
			t.Fatalf("unexpected goal option acceptance %v: %s", extra, errs.String())
		}
	}
}

func TestCanonicalGoalEntryDelegatesWithoutRepositoryConfiguration(t *testing.T) {
	t.Setenv(canonicalGoalHelperEnv, "1")
	t.Setenv("MARKITECT_CANONICAL_GOAL_INVALID", "")
	files := writeCanonicalGoalCLIInputs(t)
	var out, errs bytes.Buffer
	if code := Run([]string{"canonical", "--action", "goal-recommend", "--goal-input", files.goal, "--runtime", files.runtime}, &out, &errs); code != 0 {
		t.Fatalf("entry exit %d: %s %s", code, out.String(), errs.String())
	}
	var report map[string]any
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report["status"] != "proposed" || report["adopted"] != false {
		t.Fatalf("entry did not produce a noncanonical goal proposal: %v", report)
	}
}
