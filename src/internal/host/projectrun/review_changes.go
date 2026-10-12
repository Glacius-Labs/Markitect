package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
import "github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// reviewCandidateState binds a review to the immutable source snapshot and
// retained candidate that produced the bytes under assessment.
type reviewCandidateState struct {
	Base      *snapshot.Snapshot
	Candidate candidateData
}

// candidateChange makes file operations explicit in reviewer context. For a
// deletion, PriorDigest and PriorMode bind the supplied prior-byte artifact;
// candidate fields are intentionally absent.
type candidateChange struct {
	Path            string `json:"path"`
	Operation       string `json:"operation"`
	PriorMode       string `json:"priorMode,omitempty"`
	PriorDigest     string `json:"priorDigest,omitempty"`
	CandidateMode   string `json:"candidateMode,omitempty"`
	CandidateDigest string `json:"candidateDigest,omitempty"`
}

func reviewCandidateStateFor(host Host, root string, plan PlanRecord, run RunReport, candidate candidateData) (reviewCandidateState, error) {
	fixed, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		return reviewCandidateState{}, fmt.Errorf("load fixed review base: %w", err)
	}
	if fixed == nil || fixed.Snapshot == nil {
		return reviewCandidateState{}, fmt.Errorf("fixed review base is unavailable")
	}
	base := fixed.Snapshot
	if run.ID == "" {
		return reviewCandidateState{Base: base, Candidate: candidate}, nil
	}
	parentID := reviewParentCandidateID(plan, candidate)
	if parentID == "" {
		return reviewCandidateState{Base: base, Candidate: candidate}, nil
	}
	store, err := newRunStore(root)
	if err != nil {
		return reviewCandidateState{}, err
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		return reviewCandidateState{}, err
	}
	parent, err := store.readCandidate(dir, parentID)
	if err != nil {
		return reviewCandidateState{}, fmt.Errorf("load retained review parent candidate: %w", err)
	}
	base, err = snapshotWithCandidate(base, parent)
	if err != nil {
		return reviewCandidateState{}, err
	}
	return reviewCandidateState{Base: base, Candidate: candidate}, nil
}

func reviewParentCandidateID(plan PlanRecord, candidate candidateData) string {
	if len(candidate.Parents) > 0 {
		return candidate.Parents[0]
	}
	if plan.InitialCandidateID != candidate.ID {
		return plan.InitialCandidateID
	}
	return ""
}

// sameReviewDeletePreimages prevents a current candidate from reusing a prior
// pass after its retained deletion evidence changed. The operation scope
// digest binds the source preimage; this exact candidate-row comparison also
// catches tampering in a deletion row when the source path is already absent
// from the stable review baseline.
func sameReviewDeletePreimages(reviewed, current candidateData, paths []string) bool {
	for _, path := range paths {
		prior, hadPrior := reviewed.Files[path]
		now, hasNow := current.Files[path]
		priorDelete := hadPrior && prior.Delete
		currentDelete := hasNow && now.Delete
		if priorDelete != currentDelete {
			return false
		}
		if priorDelete && (prior.Mode != now.Mode || !bytes.Equal(prior.Content, now.Content)) {
			return false
		}
	}
	return true
}

// reviewCandidateDeletesMatchBaseline rejects a retained delete row whose
// preimage disagrees with the source snapshot used for that review. If the
// immediate baseline already contains a delete, reviewCandidateChanges binds
// its retained prior bytes into the scope digest instead.
func reviewCandidateDeletesMatchBaseline(state reviewCandidateState, parent *candidateData, paths []string) bool {
	for _, path := range paths {
		file, exists := state.Candidate.Files[path]
		if !exists || !file.Delete {
			continue
		}
		prior, exists := snapshotFile(state.Base, path)
		mode := ""
		if exists {
			mode = protocolMode(state.Base.Modes[path])
		} else if parent != nil {
			parentFile, present := parent.Files[path]
			if !present || !parentFile.Delete {
				return false
			}
			prior = append([]byte(nil), parentFile.Content...)
			mode = reviewProtocolMode(parentFile.Mode)
		}
		if mode == "" || !bytes.Equal(file.Content, prior) || reviewProtocolMode(file.Mode) != mode {
			return false
		}
	}
	return true
}

// freshPassedReviewForScope finds the newest retained passed review whose
// exact candidate-relative operation baseline still produces the current
// Manager scope. Sibling integration can change the root candidate's parent
// ordering, so freshness must be recomputed from each reviewed candidate's
// own parent, with only the candidate bytes replaced by the current final
// candidate.
func freshPassedReviewForScope(host Host, root string, store *runStore, dir string, plan PlanRecord, run RunReport, project *projectvalue.Project, currentCandidate candidateData, task ManagerTask, phase string) (ReviewRecord, reviewCandidateState, bool, error) {
	for index := len(run.Reviews) - 1; index >= 0; index-- {
		review := run.Reviews[index]
		if review.ManagerID != task.ManagerID || review.Phase != phase || review.Outcome != "pass" {
			continue
		}
		retained, err := store.readCandidate(dir, review.CandidateID)
		if err != nil || retained.ID != review.CandidateID || retained.Digest != review.CandidateDigest {
			return ReviewRecord{}, reviewCandidateState{}, false, fmt.Errorf("review for Manager %s is not bound to a retained exact candidate", task.ManagerID)
		}
		state, err := reviewCandidateStateFor(host, root, plan, run, retained)
		if err != nil {
			return ReviewRecord{}, reviewCandidateState{}, false, fmt.Errorf("load retained review scope for Manager %s: %w", task.ManagerID, err)
		}
		paths := uniqueSorted(append(reviewScopePaths(reviewCandidateFiles(project, task, run.Tasks, phase)), reviewChangedPaths(task, run.Tasks, phase)...))
		var parent *candidateData
		if parentID := reviewParentCandidateID(plan, retained); parentID != "" {
			parentCandidate, err := store.readCandidate(dir, parentID)
			if err != nil {
				return ReviewRecord{}, reviewCandidateState{}, false, fmt.Errorf("load retained deletion source candidate for Manager %s: %w", task.ManagerID, err)
			}
			parent = &parentCandidate
		}
		if !reviewCandidateDeletesMatchBaseline(state, parent, paths) {
			return ReviewRecord{}, reviewCandidateState{}, false, fmt.Errorf("review for Manager %s has deletion evidence that does not match its exact source baseline", task.ManagerID)
		}
		if !sameReviewDeletePreimages(retained, currentCandidate, paths) {
			continue
		}
		state.Candidate = currentCandidate
		scopeDigest, err := reviewScopeDigest(plan, project, task, phase, run, state)
		if err != nil {
			return ReviewRecord{}, reviewCandidateState{}, false, err
		}
		if scopeDigest == review.ScopeDigest {
			return review, state, true, nil
		}
	}
	return ReviewRecord{}, reviewCandidateState{}, false, nil
}

func reviewCandidateChanges(base *snapshot.Snapshot, candidate candidateData, selectedPaths []string) ([]candidateChange, []agentexec.Artifact, error) {
	selected := map[string]bool{}
	for _, path := range selectedPaths {
		selected[path] = true
	}
	changes := []candidateChange{}
	deleted := []agentexec.Artifact{}
	paths := make([]string, 0, len(candidate.Files))
	for path := range candidate.Files {
		if len(selected) == 0 || selected[path] {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		file := candidate.Files[path]
		priorBytes, existed := snapshotFile(base, path)
		priorMode := ""
		if existed {
			priorMode = protocolMode(base.Modes[path])
		}
		if file.Delete {
			if !existed {
				// Retained delete records carry their exact preimage so later
				// review and verification do not need a mutable workspace read.
				priorBytes = append([]byte(nil), file.Content...)
				priorMode = reviewProtocolMode(file.Mode)
				existed = priorMode != ""
			}
			if !existed || priorMode == "" {
				return nil, nil, fmt.Errorf("deleted candidate path %s has no grounded prior file", path)
			}
			changes = append(changes, candidateChange{Path: path, Operation: "delete", PriorMode: priorMode, PriorDigest: rawContentDigest(priorBytes)})
			deleted = append(deleted, agentexec.Artifact{Path: path, Mode: priorMode, Digest: rawContentDigest(priorBytes), Content: append([]byte(nil), priorBytes...)})
			continue
		}
		operation := "add"
		if existed {
			candidateMode := reviewProtocolMode(file.Mode)
			if string(priorBytes) == string(file.Content) && priorMode == candidateMode {
				continue
			}
			operation = "modify"
		}
		changes = append(changes, candidateChange{Path: path, Operation: operation, PriorMode: priorMode,
			PriorDigest: digestIfPresent(priorBytes, existed), CandidateMode: reviewProtocolMode(file.Mode), CandidateDigest: rawContentDigest(file.Content)})
	}
	return changes, deleted, nil
}

func reviewProtocolMode(mode string) string {
	if mode == "0644" || mode == "0755" {
		return mode
	}
	return protocolMode(mode)
}

func snapshotFile(base *snapshot.Snapshot, path string) ([]byte, bool) {
	if base == nil {
		return nil, false
	}
	content, ok := base.Files[path]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), content...), true
}

func digestIfPresent(content []byte, present bool) string {
	if !present {
		return ""
	}
	return rawContentDigest(content)
}

func reviewChangeRefs(report projectmodel.Report, accepted projectmodel.ManagerContext, base *snapshot.Snapshot, candidate candidateData, selectedPaths []string) ([]reviewFileRef, []candidateChange, []agentexec.Artifact, error) {
	changes, deleted, err := reviewCandidateChanges(base, candidate, selectedPaths)
	if err != nil {
		return nil, nil, nil, err
	}
	refs := reviewFileReferences(report, accepted, deleted)
	for i := range refs {
		refs[i].Operation = "delete"
		refs[i].Grounding = uniqueSorted(append(refs[i].Grounding, "operation:delete", "prior-file-bytes", "prior-file-mode"))
	}
	return refs, changes, deleted, nil
}
