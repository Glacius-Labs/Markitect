package host

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/canonical"
)

const goalModelingHelperEnv = "MARKITECT_GOAL_MODELING_TEST_HELPER"

func TestGoalModelingHelperProcess(t *testing.T) {
	if os.Getenv(goalModelingHelperEnv) != "1" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		fmt.Fprintln(os.Stderr, "invalid invocation")
		os.Exit(2)
	}
	var contextValue struct {
		Phase string `json:"phase"`
		Task  string `json:"task"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextValue); err != nil {
		fmt.Fprintln(os.Stderr, "invalid request context")
		os.Exit(2)
	}
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest,
		Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{},
		EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		Uncertainty: []string{"helper does not assess provider quality"},
	}
	switch contextValue.Phase {
	case "recommend-modules":
		id := invocation.Request.ScopeIDs[0]
		if os.Getenv("MARKITECT_GOAL_MODELING_TEST_UNKNOWN_MODULE") == "1" {
			id = "not-supplied@1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000000"
		}
		response.CandidateJSON = json.RawMessage(fmt.Sprintf(`{"recommendations":[{"id":%q,"basis":"The supplied schema can describe this goal.","uncertainty":["The goal needs owner review."]}],"uncertainty":["No provider-quality claim is made."]}`, id))
	case "model-from-selected-modules":
		for _, fragment := range []string{
			`Property typed reference`,
			`required string fields namespace and name`,
			`Example: {"namespace":"commerce","name":"orders"}`,
			`never wrap a Definition reference in metadata`,
			`Property typed kindReference`,
			`containing exactly apiVersion and kind`,
			`this identifies a Kind, not a Definition`,
		} {
			if !strings.Contains(contextValue.Task, fragment) {
				fmt.Fprintf(os.Stderr, "proposal task omits Core reference contract %q", fragment)
				os.Exit(2)
			}
		}
		definition := `{"apiVersion":"goals.example.org/v1","kind":"Module","metadata":{"name":"orders","namespace":"commerce"},"purpose":"Own order placement intent.","spec":{"name":"Orders"}}`
		if os.Getenv("MARKITECT_GOAL_MODELING_TEST_INVALID_DEFINITION") == "1" {
			definition = strings.Replace(definition, `"name":"Orders"`, `"undeclared":"Orders"`, 1)
		}
		response.CandidateJSON = json.RawMessage(`{"definitions":[` + definition + `],"summary":"A bounded order module proposal.","uncertainty":["The owner must decide whether payment belongs in this module."]}`)
	default:
		fmt.Fprintln(os.Stderr, "unexpected inference phase")
		os.Exit(2)
	}
	_ = json.NewEncoder(os.Stdout).Encode(response)
	os.Exit(0)
}

func TestGoalModelingUsesOnlyExplicitCatalogAndSelection(t *testing.T) {
	input := goalModelingTestInput()
	config := goalModelingTestConfig()
	options := goalModelingTestOptions(t)
	t.Setenv(goalModelingHelperEnv, "1")
	t.Setenv("MARKITECT_GOAL_MODELING_TEST_UNKNOWN_MODULE", "")
	t.Setenv("MARKITECT_GOAL_MODELING_TEST_INVALID_DEFINITION", "")
	t.Setenv("MARKITECT_AGENT_PRIVATE_LOG", "")

	recommendations, err := RecommendGoalModules(context.Background(), input, config, options)
	if err != nil {
		t.Fatal(err)
	}
	if recommendations.Status != "proposed" || recommendations.Adopted || recommendations.Accepted || len(recommendations.Recommendations) != 1 {
		t.Fatalf("unexpected recommendation result: %#v", recommendations)
	}
	recommendation := recommendations.Recommendations[0]
	if recommendation.ID != goalTestModuleID(t, input.Packages[0]) || recommendation.Pin.Digest == "" || recommendation.Basis == "" || len(recommendation.Uncertainty) == 0 {
		t.Fatalf("recommendation was not bound to its supplied exact package: %#v", recommendation)
	}

	proposal, err := ProposeGoalModel(context.Background(), input, recommendations, []string{recommendation.ID}, config, goalModelingTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status != "proposed" || proposal.Adopted || proposal.Accepted || proposal.CompiledCandidate == nil || len(proposal.CompiledCandidate.Definitions) != 1 {
		t.Fatalf("unexpected compiled proposal: %#v", proposal)
	}
	if proposal.Receipt.RunID == recommendations.Receipt.RunID || proposal.InputDigest == "" || proposal.CandidateDigest == "" {
		t.Fatalf("stage two was not a separately bound fresh run: %#v", proposal)
	}
	if len(proposal.SelectedPins) != 1 || proposal.SelectedPins[0] != recommendation.Pin {
		t.Fatalf("selected pins differ from the explicit recommendation: %#v", proposal.SelectedPins)
	}
}

func TestGoalRecommendationCannotInventModuleIDs(t *testing.T) {
	input := goalModelingTestInput()
	t.Setenv(goalModelingHelperEnv, "1")
	t.Setenv("MARKITECT_GOAL_MODELING_TEST_UNKNOWN_MODULE", "1")
	t.Setenv("MARKITECT_AGENT_PRIVATE_LOG", "")
	_, err := RecommendGoalModules(context.Background(), input, goalModelingTestConfig(), goalModelingTestOptions(t))
	if err == nil || !strings.Contains(err.Error(), "unknown or repeated supplied Module ID") {
		t.Fatalf("invented module ID was accepted: %v", err)
	}
}

func TestGoalModelingCompilesWithSelectedCoreSchemas(t *testing.T) {
	input := goalModelingTestInput()
	t.Setenv(goalModelingHelperEnv, "1")
	t.Setenv("MARKITECT_GOAL_MODELING_TEST_UNKNOWN_MODULE", "")
	t.Setenv("MARKITECT_GOAL_MODELING_TEST_INVALID_DEFINITION", "1")
	t.Setenv("MARKITECT_AGENT_PRIVATE_LOG", "")
	recommendations, err := RecommendGoalModules(context.Background(), input, goalModelingTestConfig(), goalModelingTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ProposeGoalModel(context.Background(), input, recommendations, []string{recommendations.Recommendations[0].ID}, goalModelingTestConfig(), goalModelingTestOptions(t))
	if err == nil || proposal.Status != "invalid" || len(proposal.Diagnostics) == 0 || proposal.CompiledCandidate != nil {
		t.Fatalf("Definition outside the selected closed schema was accepted: result=%#v err=%v", proposal, err)
	}
}

func TestGoalModelingRequiresUntamperedPriorCatalogBinding(t *testing.T) {
	input := goalModelingTestInput()
	t.Setenv(goalModelingHelperEnv, "1")
	t.Setenv("MARKITECT_GOAL_MODELING_TEST_UNKNOWN_MODULE", "")
	t.Setenv("MARKITECT_GOAL_MODELING_TEST_INVALID_DEFINITION", "")
	t.Setenv("MARKITECT_AGENT_PRIVATE_LOG", "")
	recommendations, err := RecommendGoalModules(context.Background(), input, goalModelingTestConfig(), goalModelingTestOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	input.Goal += " changed"
	_, err = ProposeGoalModel(context.Background(), input, recommendations, []string{recommendations.Recommendations[0].ID}, goalModelingTestConfig(), goalModelingTestOptions(t))
	if err == nil || !strings.Contains(err.Error(), "unchanged") {
		t.Fatalf("changed goal was accepted: %v", err)
	}
}

func goalModelingTestInput() GoalModelingInput {
	return GoalModelingInput{
		Goal: "Plan a small order management application with clear module ownership.",
		Packages: []canonical.ModulePackage{{
			ManifestBytes: []byte(`apiVersion: markitect.example.org/module/v1alpha1
name: goal-test-schema
version: 1.0.0
type: schema
purpose: Provides one test Schema for goal-led modeling.
requires:
  core: "1"
  modules: []
provides:
  schemas:
    - schema.yaml
  projectors: []
`),
			Files: map[string][]byte{"schema.yaml": []byte(`apiVersion: goals.example.org/v1
purpose: Describes a bounded application module.
kinds:
  Module:
    purpose: One independently named application module.
    properties:
      name:
        purpose: The module's display name.
        type: string
        minCount: 1
        maxCount: 1
`)},
		}},
	}
}

func goalModelingTestConfig() agentexec.Config {
	return agentexec.Config{
		Command: os.Args[0], Args: []string{"-test.run=^TestGoalModelingHelperProcess$"},
		Model: "test-helper-model", ModelOptions: json.RawMessage(`{"test":true}`),
		ProviderVersion: "goal-modeling-test-helper/1", Timeout: 3 * time.Second,
		MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
	}
}

func goalModelingTestOptions(t *testing.T) GoalModelingOptions {
	t.Helper()
	return GoalModelingOptions{TempParent: t.TempDir(), PrivateLogDirectory: filepath.Join(t.TempDir(), "private-logs")}
}

func goalTestModuleID(t *testing.T, pkg canonical.ModulePackage) string {
	t.Helper()
	manifest, err := canonical.DecodeManifest(pkg.ManifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := canonical.DigestPackage(pkg)
	if err != nil {
		t.Fatal(err)
	}
	return goalModuleID(canonical.Pin{Name: manifest.Name, Version: manifest.Version, Digest: digest})
}
