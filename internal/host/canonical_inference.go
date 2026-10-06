package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/modules/adoption/capture"
	"github.com/Glacius-Labs/Markitect/internal/modules/adoption/review"
)

const brownfieldInferenceModulePin = "markitect.host/brownfield-inference@v1alpha1"

// BrownfieldInferenceInput is a closed set of already captured inputs. It has
// no repository path or Git acquisition field: the handoff and its selected
// byte map are the complete source boundary for this operation.
type BrownfieldInferenceInput struct {
	HandoffBytes []byte
	Blobs        map[string][]byte
	QueueBytes   []byte
}

// BrownfieldInferenceOptions names only private runtime locations. They must
// be existing directories outside every repository named by the handoff.
type BrownfieldInferenceOptions struct {
	TempParent          string
	PrivateLogDirectory string
}

// BrownfieldInferenceResult is a noncanonical proposal with its runner and
// input bindings. A successful result never records a decision or adoption.
type BrownfieldInferenceResult struct {
	Status              string            `json:"status"`
	HandoffID           string            `json:"handoffId"`
	HandoffIdentity     string            `json:"handoffIdentity"`
	HandoffByteDigest   string            `json:"handoffByteDigest"`
	QueueByteDigest     string            `json:"queueByteDigest"`
	ConfigFingerprint   string            `json:"configFingerprint"`
	Receipt             agentexec.Receipt `json:"receipt"`
	Candidate           *review.Candidate `json:"candidate,omitempty"`
	CandidateByteDigest string            `json:"candidateByteDigest,omitempty"`
	Report              *review.Report    `json:"validation,omitempty"`
	Adopted             bool              `json:"adopted"`
}

// RunBrownfieldInference invokes one fresh RoleInfer process over only the
// immutable selected handoff bytes, and validates any proposal with Copy Me's
// existing evidence/candidate validator. The owner must still review and
// deliberately submit canonical intent; this function cannot adopt anything.
func RunBrownfieldInference(ctx context.Context, input BrownfieldInferenceInput, config agentexec.Config, options BrownfieldInferenceOptions) (BrownfieldInferenceResult, error) {
	var handoff capture.Handoff
	if err := capture.Decode(input.HandoffBytes, &handoff); err != nil {
		return BrownfieldInferenceResult{}, fmt.Errorf("decode brownfield handoff: %w", err)
	}
	if err := capture.ValidateHandoff(handoff, input.Blobs); err != nil {
		return BrownfieldInferenceResult{}, fmt.Errorf("validate brownfield handoff: %w", err)
	}
	if err := validateInferenceRuntimeDirectories(options, handoff); err != nil {
		return BrownfieldInferenceResult{}, err
	}
	var queue review.Queue
	if err := capture.Decode(input.QueueBytes, &queue); err != nil {
		return BrownfieldInferenceResult{}, fmt.Errorf("decode inference evidence queue: %w", err)
	}
	if len(queue.Candidates) != 0 {
		return BrownfieldInferenceResult{}, errors.New("brownfield inference requires an evidence queue with no preexisting candidates")
	}
	// Validate the owner-supplied evidence ledger before it is shown to a model.
	// This also checks queue coverage and every evidence-to-byte binding.
	if _, err := review.Validate(input.HandoffBytes, input.Blobs, input.QueueBytes, map[string][]byte{}, nil); err != nil {
		return BrownfieldInferenceResult{}, fmt.Errorf("validate inference evidence queue: %w", err)
	}
	if len(queue.Evidence) > 128 {
		return BrownfieldInferenceResult{}, errors.New("inference evidence IDs exceed the agent protocol's 128-entry bound")
	}

	request, err := brownfieldInferenceRequest(handoff, input.HandoffBytes, input.Blobs, queue)
	if err != nil {
		return BrownfieldInferenceResult{}, err
	}
	fingerprint, err := agentexec.Fingerprint(config)
	if err != nil {
		return BrownfieldInferenceResult{}, fmt.Errorf("fingerprint configured inference runner: %w", err)
	}
	run, err := agentexec.Run(ctx, config, request, agentexec.RunOptions{
		TempParent: options.TempParent, PrivateLogDirectory: options.PrivateLogDirectory,
	})
	result := BrownfieldInferenceResult{
		Status:            "incomplete",
		HandoffID:         handoff.ID,
		HandoffIdentity:   handoff.Digest,
		HandoffByteDigest: capture.Hash(input.HandoffBytes),
		QueueByteDigest:   capture.Hash(input.QueueBytes),
		ConfigFingerprint: fingerprint,
		Adopted:           false,
	}
	if err != nil {
		result.Receipt = run.Receipt
		return result, err
	}
	result.Receipt = run.Receipt
	if run.Receipt.ConfigDigest != fingerprint {
		return result, errors.New("inference receipt configuration differs from the prepared runner fingerprint")
	}
	result.Status = run.Response.Outcome
	if run.Response.Outcome != agentexec.OutcomeProposed {
		return result, nil
	}
	if len(run.Response.Uncertainty) == 0 {
		return result, errors.New("inference proposal must include explicit response uncertainty")
	}
	var candidate review.Candidate
	if err := capture.Decode(run.Response.CandidateJSON, &candidate); err != nil {
		return result, fmt.Errorf("decode proposed Copy Me candidate: %w", err)
	}
	if len(candidate.Uncertainty) == 0 {
		return result, errors.New("inference candidate must state at least one uncertainty")
	}
	if err := sameInferenceEvidenceReferences(run.Response.EvidenceRefs, candidate); err != nil {
		return result, err
	}
	candidateBytes, err := capture.Encode(candidate)
	if err != nil {
		return result, fmt.Errorf("encode inference candidate: %w", err)
	}
	queue.Candidates = []review.CandidateReference{{StableID: candidate.StableID, Path: "candidates/" + candidate.StableID + ".yaml", Digest: capture.Hash(candidateBytes)}}
	proposalQueue, err := capture.Encode(queue)
	if err != nil {
		return result, fmt.Errorf("encode proposal validation queue: %w", err)
	}
	report, err := review.Validate(input.HandoffBytes, input.Blobs, proposalQueue, map[string][]byte{queue.Candidates[0].Path: candidateBytes}, nil)
	if err != nil {
		return result, fmt.Errorf("validate inferred candidate: %w", err)
	}
	if report.Adopted || report.Decision != nil || !report.UnauthenticatedReviewer {
		return result, errors.New("Copy Me candidate validation crossed the non-authoritative review boundary")
	}
	result.Candidate = &candidate
	result.CandidateByteDigest = capture.Hash(candidateBytes)
	result.Report = &report
	result.Status = "proposed"
	return result, nil
}

func brownfieldInferenceRequest(handoff capture.Handoff, handoffBytes []byte, blobs map[string][]byte, queue review.Queue) (agentexec.Request, error) {
	contextValue := struct {
		APIVersion      string                `json:"apiVersion"`
		HandoffID       string                `json:"handoffId"`
		HandoffIdentity string                `json:"handoffIdentity"`
		HandoffDigest   string                `json:"handoffByteDigest"`
		SelectionDigest string                `json:"selectionDigest"`
		CaptureDigest   string                `json:"captureDigest"`
		Purpose         string                `json:"purpose"`
		Review          string                `json:"review"`
		Privacy         capture.Privacy       `json:"privacy"`
		Retention       string                `json:"retention"`
		Repositories    []inferenceRepository `json:"repositories"`
		Coverage        []capture.Coverage    `json:"coverage"`
		Evidence        []review.Evidence     `json:"evidence"`
	}{
		APIVersion: "markitect.example.org/brownfield-inference-context/v1alpha1", HandoffID: handoff.ID,
		HandoffIdentity: handoff.Digest, HandoffDigest: capture.Hash(handoffBytes), SelectionDigest: handoff.SelectionDigest,
		CaptureDigest: handoff.CaptureDigest, Purpose: handoff.Purpose, Review: handoff.Review, Privacy: handoff.Privacy,
		Retention: handoff.Retention, Coverage: append([]capture.Coverage{}, handoff.Coverage...), Evidence: append([]review.Evidence{}, queue.Evidence...),
	}
	artifacts := make([]agentexec.Artifact, 0)
	scopeIDs := []string{"handoff/" + handoff.ID}
	for _, repo := range handoff.Repositories {
		contextValue.Repositories = append(contextValue.Repositories, inferenceRepository{ID: repo.ID, Commit: repo.Commit, SnapshotDigest: repo.SnapshotDigest})
		for _, file := range repo.Files {
			key := capture.BlobKey(repo.ID, file.Path)
			data, ok := blobs[key]
			if !ok {
				return agentexec.Request{}, fmt.Errorf("selected handoff bytes missing for %s", key)
			}
			artifactPath := repo.ID + "/" + file.Path
			mode := "0644"
			if file.Mode == "100755" {
				mode = "0755"
			}
			artifacts = append(artifacts, agentexec.Artifact{Path: artifactPath, Mode: mode, Digest: "sha256:" + capture.Hash(data), Content: append([]byte(nil), data...)})
			scopeIDs = append(scopeIDs, artifactPath)
		}
	}
	for _, evidence := range queue.Evidence {
		scopeIDs = append(scopeIDs, evidence.ID)
	}
	contextBytes, err := json.Marshal(contextValue)
	if err != nil {
		return agentexec.Request{}, fmt.Errorf("encode bounded inference context: %w", err)
	}
	return agentexec.Request{
		Role: agentexec.RoleInfer, SourceRevision: "handoff/" + handoff.ID + "/" + handoff.Digest,
		ModelDigest: "sha256:" + capture.Hash(handoffBytes), ModulePin: brownfieldInferenceModulePin,
		ProjectionID: "brownfield-inference/" + handoff.ID, ScopeIDs: scopeIDs, PolicyIDs: []string{},
		Context: contextBytes, Artifacts: artifacts,
	}, nil
}

type inferenceRepository struct {
	ID             string `json:"id"`
	Commit         string `json:"commit"`
	SnapshotDigest string `json:"snapshotDigest"`
}

func sameInferenceEvidenceReferences(references []string, candidate review.Candidate) error {
	want := make(map[string]bool, len(candidate.Support)+len(candidate.Counterexamples)+len(candidate.Qualifies))
	for _, values := range [][]string{candidate.Support, candidate.Counterexamples, candidate.Qualifies} {
		for _, id := range values {
			want[id] = true
		}
	}
	got := make(map[string]bool, len(references))
	for _, id := range references {
		got[id] = true
	}
	if len(got) != len(references) || len(got) != len(want) {
		return errors.New("inference evidenceRefs must list every exact candidate evidence ID once")
	}
	for id := range want {
		if !got[id] {
			return fmt.Errorf("inference evidenceRefs omit candidate evidence ID %q", id)
		}
	}
	return nil
}

func validateInferenceRuntimeDirectories(options BrownfieldInferenceOptions, handoff capture.Handoff) error {
	if options.TempParent == "" || options.PrivateLogDirectory == "" {
		return errors.New("inference requires explicit temporary and private log directories")
	}
	resolve := func(value string) (string, error) {
		absolute, err := filepath.Abs(value)
		if err != nil {
			return "", err
		}
		resolved, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("inference runtime path must be an existing directory: %s", value)
		}
		return filepath.Clean(resolved), nil
	}
	temp, err := resolve(options.TempParent)
	if err != nil {
		return err
	}
	logs, err := resolve(options.PrivateLogDirectory)
	if err != nil {
		return err
	}
	if inferencePathsOverlap(temp, logs) {
		return errors.New("inference temporary and private log directories must be disjoint")
	}
	for _, repo := range handoff.Repositories {
		for _, guarded := range []string{repo.Identity.Root, repo.Identity.GitDir, repo.Identity.CommonDir} {
			resolved, err := resolve(guarded)
			if err != nil {
				return fmt.Errorf("resolve handoff repository boundary: %w", err)
			}
			if inferencePathsOverlap(temp, resolved) || inferencePathsOverlap(logs, resolved) {
				return fmt.Errorf("inference temporary and private log directories must be outside handoff repository boundary %s", guarded)
			}
		}
	}
	return nil
}

func inferencePathsOverlap(left, right string) bool {
	within := func(path, root string) bool {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return false
		}
		return relative == "." || (relative != ".." && !strings.HasPrefix(strings.ToLower(relative), strings.ToLower(".."+string(filepath.Separator))))
	}
	return within(left, right) || within(right, left)
}
