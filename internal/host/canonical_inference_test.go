package host

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/modules/adoption/capture"
	"github.com/Glacius-Labs/Markitect/internal/modules/adoption/review"
)

const brownfieldInferenceHelperEnv = "MARKITECT_BROWNFIELD_INFERENCE_TEST_HELPER"

func TestBrownfieldInferenceHelper(t *testing.T) {
	if os.Getenv(brownfieldInferenceHelperEnv) != "1" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		os.Exit(11)
	}
	if invocation.Request.Role != agentexec.RoleInfer || len(invocation.Request.Artifacts) != 1 ||
		invocation.Request.Artifacts[0].Path != "main/docs/selected.md" || string(invocation.Request.Artifacts[0].Content) != "selected evidence\n" {
		os.Exit(12)
	}
	if strings.Contains(string(invocation.Request.Context), "unselected.md") || strings.Contains(string(invocation.Request.Context), "GitDir") {
		os.Exit(13)
	}
	for _, instruction := range []string{"candidateJsonSchema", "protocolExample", "apiVersion", "confidenceBasis", "counterexamples", "possible violation", "exact evidence IDs", "owner review and correction"} {
		if !strings.Contains(string(invocation.Request.Context), instruction) {
			os.Exit(15)
		}
	}
	candidate := review.Candidate{
		APIVersion: review.CandidateVersion, StableID: "inferred-rule", ProposedRule: "Keep the selected representation stable",
		Scope: "the bounded selected document", Conditions: []string{}, Classification: "unclear", Support: []string{"support-1"},
		Counterexamples: []string{}, Qualifies: []string{}, Confidence: "low", ConfidenceBasis: "one owner-curated evidence item",
		Alternatives: []string{}, Uncertainty: []string{"whether the observed wording reflects a durable project decision"}, Questions: []string{},
	}
	candidateJSON, err := json.Marshal(map[string]any{
		"apiVersion": candidate.APIVersion, "stableID": candidate.StableID, "proposedRule": candidate.ProposedRule,
		"scope": candidate.Scope, "conditions": candidate.Conditions, "classification": candidate.Classification,
		"support": candidate.Support, "counterexamples": candidate.Counterexamples, "qualifies": candidate.Qualifies,
		"confidence": candidate.Confidence, "confidenceBasis": candidate.ConfidenceBasis,
		"alternatives": candidate.Alternatives, "uncertainty": candidate.Uncertainty, "questions": candidate.Questions,
	})
	if err != nil {
		os.Exit(14)
	}
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleInfer, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{"support-1"},
		VerifierObservations: []agentexec.Observation{}, CandidateJSON: candidateJSON,
		Uncertainty: []string{"the single selected representation may be incomplete"},
	}
	if os.Getenv("MARKITECT_BROWNFIELD_INFERENCE_TEST_BAD_EVIDENCE") == "1" {
		response.EvidenceRefs = []string{"unselected.md"}
	}
	_ = json.NewEncoder(os.Stdout).Encode(response)
	os.Exit(0)
}

func TestRunBrownfieldInferenceProducesBoundNonAuthoritativeCandidate(t *testing.T) {
	input, root := brownfieldInferenceFixture(t)
	options := brownfieldInferenceRuntime(t, root)
	t.Setenv(brownfieldInferenceHelperEnv, "1")
	t.Setenv("MARKITECT_BROWNFIELD_INFERENCE_TEST_BAD_EVIDENCE", "")
	result, err := RunBrownfieldInference(context.Background(), input, brownfieldInferenceConfig(), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "proposed" || result.Adopted || result.Candidate == nil || result.Report == nil {
		t.Fatalf("inference did not return a review-only proposal: %#v", result)
	}
	if result.HandoffByteDigest != capture.Hash(input.HandoffBytes) || result.QueueByteDigest != capture.Hash(input.QueueBytes) ||
		result.ConfigFingerprint == "" || result.Receipt.ConfigDigest != result.ConfigFingerprint || result.Receipt.InputDigest == "" {
		t.Fatalf("result is missing exact input/configuration/receipt bindings: %#v", result)
	}
	if result.Candidate.StableID != "inferred-rule" || result.CandidateByteDigest == "" || result.Report.Adopted || result.Report.Decision != nil {
		t.Fatalf("candidate crossed the explicit owner review boundary: %#v", result)
	}
}

func TestRunBrownfieldInferenceRejectsUnboundAndOverbroadInputs(t *testing.T) {
	input, root := brownfieldInferenceFixture(t)
	options := brownfieldInferenceRuntime(t, root)
	config := brownfieldInferenceConfig()
	t.Setenv(brownfieldInferenceHelperEnv, "1")
	t.Setenv("MARKITECT_BROWNFIELD_INFERENCE_TEST_BAD_EVIDENCE", "1")
	if _, err := RunBrownfieldInference(context.Background(), input, config, options); err == nil || !strings.Contains(err.Error(), "evidence reference") {
		t.Fatalf("unselected evidence reference accepted: %v", err)
	}
	t.Setenv("MARKITECT_BROWNFIELD_INFERENCE_TEST_BAD_EVIDENCE", "")
	changed := BrownfieldInferenceInput{HandoffBytes: input.HandoffBytes, Blobs: map[string][]byte{}, QueueBytes: input.QueueBytes}
	for key, value := range input.Blobs {
		changed.Blobs[key] = value
	}
	changed.Blobs[capture.BlobKey("main", "notes/unselected.md")] = []byte("not selected\n")
	if _, err := RunBrownfieldInference(context.Background(), changed, config, options); err == nil || !strings.Contains(err.Error(), "unselected evidence") {
		t.Fatalf("unselected bytes accepted as inference context: %v", err)
	}
}

func TestBrownfieldInferenceRuntimeLocationsCannotOverlapAdopterRoots(t *testing.T) {
	input, root := brownfieldInferenceFixture(t)
	var handoff capture.Handoff
	if err := capture.Decode(input.HandoffBytes, &handoff); err != nil {
		t.Fatal(err)
	}
	logs := filepath.Join(t.TempDir(), "private-logs")
	if err := os.Mkdir(logs, 0700); err != nil {
		t.Fatal(err)
	}
	if err := validateInferenceRuntimeDirectories(BrownfieldInferenceOptions{TempParent: root, PrivateLogDirectory: logs}, handoff); err == nil {
		t.Fatal("temporary directory within adopter root accepted")
	}
}

func TestBrownfieldInferenceRuntimeGuardDoesNotRequireOriginalRepositoryToExist(t *testing.T) {
	input, originalRoot := brownfieldInferenceFixture(t)
	var handoff capture.Handoff
	if err := capture.Decode(input.HandoffBytes, &handoff); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(originalRoot); err != nil {
		t.Fatal(err)
	}
	options := brownfieldInferenceRuntime(t, originalRoot)
	t.Setenv(brownfieldInferenceHelperEnv, "1")
	result, err := RunBrownfieldInference(context.Background(), input, brownfieldInferenceConfig(), options)
	if err != nil {
		t.Fatalf("inference with copied handoff required the original repository: %v", err)
	}
	if result.Status != "proposed" || result.Adopted {
		t.Fatalf("copied handoff did not produce a review-only proposal: %#v", result)
	}

	missingParent := t.TempDir()
	missingRoot := filepath.Join(missingParent, "repository-no-longer-mounted")
	if _, err := os.Stat(missingRoot); !os.IsNotExist(err) {
		t.Fatalf("test repository boundary unexpectedly exists: %v", err)
	}
	handoff.Repositories[0].Identity.Root = missingRoot
	handoff.Repositories[0].Identity.GitDir = filepath.Join(missingRoot, ".git")
	handoff.Repositories[0].Identity.CommonDir = filepath.Join(missingRoot, ".git")
	missingOptions := brownfieldInferenceRuntime(t, missingRoot)
	if err := validateInferenceRuntimeDirectories(missingOptions, handoff); err != nil {
		t.Fatalf("copied handoff required the source repository to remain mounted: %v", err)
	}
	logs := t.TempDir()
	if err := validateInferenceRuntimeDirectories(BrownfieldInferenceOptions{TempParent: missingParent, PrivateLogDirectory: logs}, handoff); err == nil {
		t.Fatal("runtime directory lexically overlapping the unavailable repository root was accepted")
	}
}

func brownfieldInferenceFixture(t *testing.T) (BrownfieldInferenceInput, string) {
	t.Helper()
	base := adoptionTestTempDir(t)
	root, commit := adoptionTestRepository(t, filepath.Join(base, "source"))
	scope := adoptionTestScope(root, commit)
	destination := filepath.Join(base, "captured")
	preview, err := PrepareAdoption(scope, destination, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareAdoption(scope, destination, preview.Handoff.Digest, true); err != nil {
		t.Fatal(err)
	}
	handoffBytes, blobs, err := ReadAdoptionWorkspace(destination)
	if err != nil {
		t.Fatal(err)
	}
	var handoff capture.Handoff
	if err := capture.Decode(handoffBytes, &handoff); err != nil {
		t.Fatal(err)
	}
	selected := blobs[capture.BlobKey("main", "docs/selected.md")]
	queue, err := capture.Encode(review.Queue{
		APIVersion: review.QueueVersion,
		Evidence:   []review.Evidence{{ID: "support-1", Repository: "main", Path: "docs/selected.md", SourceDigest: capture.Hash(selected), Stance: "supports", Observation: "owner-curated evidence for bounded inference"}},
		Candidates: []review.CandidateReference{}, Coverage: append([]capture.Coverage{}, handoff.Coverage...), Requests: []review.EvidenceRequest{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return BrownfieldInferenceInput{HandoffBytes: handoffBytes, Blobs: blobs, QueueBytes: queue}, root
}

func brownfieldInferenceRuntime(t *testing.T, adopterRoot string) BrownfieldInferenceOptions {
	t.Helper()
	parent := t.TempDir()
	if strings.HasPrefix(strings.ToLower(parent), strings.ToLower(adopterRoot+string(filepath.Separator))) {
		t.Fatal("test runtime unexpectedly falls within adopter root")
	}
	logs := t.TempDir()
	return BrownfieldInferenceOptions{TempParent: parent, PrivateLogDirectory: logs}
}

func brownfieldInferenceConfig() agentexec.Config {
	return agentexec.Config{
		Command: os.Args[0], Args: []string{"-test.run=^TestBrownfieldInferenceHelper$"},
		Model: "fake-protocol-actor", ModelOptions: json.RawMessage(`{"purpose":"protocol-only"}`),
		ProviderVersion: "host-test/1", Timeout: 5 * time.Second, MaxStdoutBytes: 64 << 10, MaxStderrBytes: 64 << 10,
	}
}
