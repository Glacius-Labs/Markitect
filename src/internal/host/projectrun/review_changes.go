package projectrun

import (
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// reviewCandidateState binds a review to the immutable source snapshot and
// retained candidate that produced the bytes under assessment.
type reviewCandidateState struct {
	Base      *Snapshot
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
	parentID := ""
	if len(candidate.Parents) > 0 {
		parentID = candidate.Parents[0]
	} else if plan.InitialCandidateID != candidate.ID {
		parentID = plan.InitialCandidateID
	}
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

func reviewCandidateChanges(base *Snapshot, candidate candidateData, selectedPaths []string) ([]candidateChange, []agentexec.Artifact, error) {
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

func snapshotFile(base *Snapshot, path string) ([]byte, bool) {
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

func reviewChangeRefs(report projectmodel.Report, accepted projectmodel.ManagerContext, base *Snapshot, candidate candidateData, selectedPaths []string) ([]reviewFileRef, []candidateChange, []agentexec.Artifact, error) {
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
