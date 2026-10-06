package host

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/assurance"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

// The configured assurance graph, not filename order, controls preparation.
func canonicalControllerOrderedProposals(cfg CanonicalControllerConfig, proposal CanonicalControllerProposal) ([]CanonicalScopedProposal, error) {
	if len(cfg.AssuranceScopes) == 0 {
		return append([]CanonicalScopedProposal(nil), proposal.Plan.Proposals...), nil
	}
	graph, err := canonicalControllerAssuranceGraph(cfg, proposal.fixed)
	if err != nil {
		return nil, err
	}
	order, err := assurance.ExecutionOrder(graph)
	if err != nil {
		return nil, err
	}
	scopes := map[string]CanonicalAssuranceScope{}
	for _, scope := range cfg.AssuranceScopes {
		scopes[scope.ID] = scope
	}
	byProjection := map[string]CanonicalScopedProposal{}
	for _, p := range proposal.Plan.Proposals {
		if _, duplicate := byProjection[p.ProjectionID]; duplicate {
			return nil, errors.New("duplicate prepared Projection")
		}
		byProjection[p.ProjectionID] = p
	}
	result := []CanonicalScopedProposal{}
	for _, id := range order {
		key := scopes[id].ProjectionID
		if p, exists := byProjection[key]; exists {
			result = append(result, p)
			delete(byProjection, key)
		}
	}
	independent := []string{}
	for id := range byProjection {
		independent = append(independent, id)
	}
	sort.Strings(independent)
	for _, id := range independent {
		result = append(result, byProjection[id])
	}
	return result, nil
}

func canonicalControllerDescendantProjections(cfg CanonicalControllerConfig, projectionID string) ([]string, error) {
	scopes := map[string]CanonicalAssuranceScope{}
	rootID := ""
	for _, scope := range cfg.AssuranceScopes {
		scopes[scope.ID] = scope
		if scope.ProjectionID == projectionID {
			rootID = scope.ID
		}
	}
	if rootID == "" {
		return nil, nil
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	selected := map[string]bool{}
	var walk func(string, int) error
	walk = func(id string, depth int) error {
		if depth > 8 || len(visited) > 128 {
			return errors.New("candidate dependency graph exceeds supported bound")
		}
		if visiting[id] {
			return errors.New("candidate dependency graph contains a cycle")
		}
		if visited[id] {
			return nil
		}
		scope, exists := scopes[id]
		if !exists {
			return fmt.Errorf("candidate dependency references missing scope %s", id)
		}
		visiting[id] = true
		for _, child := range scope.Children {
			if err := walk(child, depth+1); err != nil {
				return err
			}
			selected[scopes[child].ProjectionID] = true
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	if err := walk(rootID, 0); err != nil {
		return nil, err
	}
	result := []string{}
	for id := range selected {
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}

func cloneCanonicalCandidateSnapshot(input *snapshot.Snapshot) *snapshot.Snapshot {
	copied := *input
	copied.Files, copied.Modes = map[string][]byte{}, map[string]string{}
	for path, data := range input.Files {
		copied.Files[path] = append([]byte(nil), data...)
	}
	for path, mode := range input.Modes {
		copied.Modes[path] = mode
	}
	return &copied
}

func stageCanonicalCandidate(input *snapshot.Snapshot, outputs map[string][]byte) {
	for path, data := range outputs {
		input.Files[path] = append([]byte(nil), data...)
		input.Modes[path] = snapshot.RegularMode
	}
}

func canonicalControllerExecutorArtifacts(cfg CanonicalControllerConfig, p CanonicalScopedProposal, active []records.ProjectionRecord, stage *snapshot.Snapshot, staged map[string]map[string][]byte, scheduled map[string]bool) ([]agentexec.Artifact, []string, error) {
	paths := map[string]bool{}
	for path := range p.Request.TargetFiles {
		paths[path] = true
	}
	dependencyPaths := []string{}
	descendants, err := canonicalControllerDescendantProjections(cfg, p.ProjectionID)
	if err != nil {
		return nil, nil, err
	}
	for _, id := range descendants {
		if outputs, exists := staged[id]; exists {
			for path := range outputs {
				paths[path] = true
				dependencyPaths = append(dependencyPaths, path)
			}
			continue
		}
		if scheduled[id] {
			return nil, nil, fmt.Errorf("required child Projection did not produce an applicable candidate: %s", id)
		}
		found := false
		for _, record := range active {
			if record.ProjectionID != id {
				continue
			}
			found = true
			for _, artifact := range record.Artifacts {
				data, exists := stage.Files[artifact.Path]
				if !exists || sha256Prefix(sha256Hex(data)) != artifact.Digest || stage.Modes[artifact.Path] != artifact.Mode {
					return nil, nil, fmt.Errorf("retained dependency artifact is missing or drifted: %s", artifact.Path)
				}
				paths[artifact.Path] = true
				dependencyPaths = append(dependencyPaths, artifact.Path)
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("required child Projection has neither a staged candidate nor an active record: %s", id)
		}
	}
	names := []string{}
	for path := range paths {
		names = append(names, path)
	}
	sort.Strings(names)
	artifacts := []agentexec.Artifact{}
	for _, name := range names {
		data, exists := stage.Files[name]
		if !exists || stage.Modes[name] != snapshot.RegularMode {
			return nil, nil, fmt.Errorf("Executor evidence contains missing or unsupported text artifact: %s", name)
		}
		artifacts = append(artifacts, agentexec.Artifact{Path: name, Content: append([]byte(nil), data...), Mode: "0644", Digest: sha256Prefix(sha256Hex(data))})
	}
	return artifacts, sortedUniquePaths(dependencyPaths), nil
}

// Paths are needed as evidence for parent implementation, not child work.
func canonicalControllerDependencyEvidencePaths(cfg CanonicalControllerConfig, plan CanonicalScopedReconcilePlan, active []records.ProjectionRecord) ([]string, error) {
	work := map[string]bool{}
	for _, p := range plan.Proposals {
		if p.Decision == "work" {
			work[p.ProjectionID] = true
		}
	}
	paths := []string{}
	for _, p := range plan.Proposals {
		if p.Decision != "work" || p.Task == nil {
			continue
		}
		descendants, err := canonicalControllerDescendantProjections(cfg, p.ProjectionID)
		if err != nil {
			return nil, err
		}
		for _, id := range descendants {
			if work[id] {
				continue
			} // Exact candidate bytes will be supplied by the preceding child execution.
			found := false
			for _, record := range active {
				if record.ProjectionID != id {
					continue
				}
				found = true
				for _, artifact := range record.Artifacts {
					paths = append(paths, artifact.Path)
				}
			}
			if !found {
				return nil, fmt.Errorf("required unchanged child Projection has no active record: %s", id)
			}
		}
	}
	return sortedUniquePaths(paths), nil
}
