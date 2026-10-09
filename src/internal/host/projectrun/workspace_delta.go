package projectrun

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// applyWorkspaceDelta applies Host-observed bytes to an isolated candidate.
// The caller must first validate the delta's canonical digest and bind its
// repository, base SHA, overlay, task ID, and base digest to the exact
// workspace request and handle. This function rechecks product write policy;
// it does not treat model proposals as a source of candidate bytes.
func applyWorkspaceDelta(base candidateData, delta projectworkspace.Delta, proposals []agentexec.CandidateFile, config projectwork.Config, report projectmodel.Report, task ManagerTask, phase string, conflictPaths []string, limits Limits, input *Snapshot) (candidateData, error) {
	ignored, err := ignoredWritePaths(config, input)
	if err != nil {
		return candidateData{}, err
	}

	files := make(map[string]File, len(base.Files)+len(delta.Changes)*2)
	for path, file := range base.Files {
		file.Content = append([]byte(nil), file.Content...)
		files[path] = file
	}
	conflictSet := map[string]bool{}
	if phase == "integrate" {
		for _, path := range conflictPaths {
			conflictSet[path] = true
		}
	}

	usedPaths := make(map[string]string, len(delta.Changes)*2)
	observedWrites := make(map[string]File, len(delta.Changes))
	var totalBytes int64
	for _, change := range delta.Changes {
		if !safeRepoPath(change.Path) {
			return candidateData{}, fmt.Errorf("workspace delta path is unsafe: %q", change.Path)
		}
		if err := reserveObservedPath(usedPaths, change.Path); err != nil {
			return candidateData{}, err
		}
		if err := validateWorkspaceWritePath(change.Path, config, report, task, phase, conflictSet, ignored); err != nil {
			return candidateData{}, err
		}

		switch change.Kind {
		case projectworkspace.ChangeAdd, projectworkspace.ChangeModify:
			if change.OldPath != "" || change.Content == nil {
				return candidateData{}, fmt.Errorf("workspace %s requires content and no old path", change.Kind)
			}
			mode := workspaceSnapshotMode(change.Mode)
			if mode == "" {
				return candidateData{}, fmt.Errorf("workspace change %s has unsupported mode %q", change.Path, change.Mode)
			}
			if err := addWorkspaceBytes(&totalBytes, int64(len(change.Content)), change.Path, limits); err != nil {
				return candidateData{}, err
			}
			file := File{Path: change.Path, Mode: mode, Content: append([]byte(nil), change.Content...)}
			files[change.Path] = file
			observedWrites[change.Path] = file
		case projectworkspace.ChangeDelete:
			if change.OldPath != "" || change.Mode != "" || change.Content != nil {
				return candidateData{}, fmt.Errorf("workspace delete %s cannot carry old path, mode, or content", change.Path)
			}
			files[change.Path] = File{Path: change.Path, Delete: true}
		case projectworkspace.ChangeRename:
			if !safeRepoPath(change.OldPath) || change.OldPath == change.Path || change.Content == nil {
				return candidateData{}, fmt.Errorf("workspace rename %s has an invalid old path or content", change.Path)
			}
			if err := reserveObservedPath(usedPaths, change.OldPath); err != nil {
				return candidateData{}, err
			}
			if err := validateWorkspaceWritePath(change.OldPath, config, report, task, phase, conflictSet, ignored); err != nil {
				return candidateData{}, err
			}
			mode := workspaceSnapshotMode(change.Mode)
			if mode == "" {
				return candidateData{}, fmt.Errorf("workspace rename %s has unsupported mode %q", change.Path, change.Mode)
			}
			if err := addWorkspaceBytes(&totalBytes, int64(len(change.Content)), change.Path, limits); err != nil {
				return candidateData{}, err
			}
			file := File{Path: change.Path, Mode: mode, Content: append([]byte(nil), change.Content...)}
			files[change.OldPath] = File{Path: change.OldPath, Delete: true}
			files[change.Path] = file
			observedWrites[change.Path] = file
		default:
			return candidateData{}, fmt.Errorf("workspace delta has unsupported change kind %q", change.Kind)
		}
	}

	seenProposals := map[string]string{}
	for _, proposal := range proposals {
		if !safeRepoPath(proposal.Path) {
			return candidateData{}, fmt.Errorf("proposal path is unsafe: %q", proposal.Path)
		}
		key := strings.ToLower(proposal.Path)
		if prior, exists := seenProposals[key]; exists {
			if prior != proposal.Path {
				return candidateData{}, fmt.Errorf("case-alias proposal paths %q and %q", prior, proposal.Path)
			}
			return candidateData{}, fmt.Errorf("duplicate proposal path %s", proposal.Path)
		}
		seenProposals[key] = proposal.Path
		observed, exists := observedWrites[proposal.Path]
		if !exists {
			return candidateData{}, fmt.Errorf("model proposal %s was not observed in the workspace delta", proposal.Path)
		}
		mode := snapshotMode(proposal.Mode)
		if mode == "" || mode != observed.Mode || !bytesEqual([]byte(proposal.Content), observed.Content) {
			return candidateData{}, fmt.Errorf("model proposal %s contradicts observed workspace bytes or mode", proposal.Path)
		}
	}

	base.Files = files
	return base, nil
}

func validateWorkspaceWritePath(path string, config projectwork.Config, report projectmodel.Report, task ManagerTask, phase string, conflictSet map[string]bool, ignored []string) error {
	if !safeRepoPath(path) || forbiddenRuntimePath(path) || !projectPathAllowed(config, path) {
		return fmt.Errorf("workspace delta path %s is outside selected inventory or enters Markitect control-plane state", path)
	}
	if pathIgnored(ignored, path) {
		return fmt.Errorf("workspace delta path %s is explicitly ignored", path)
	}
	owner, known := ownerForPath(report, path)
	if !known || (owner != task.ManagerID && !conflictSet[path]) {
		return fmt.Errorf("manager %s observed path owned by %q (phase %s): %s", task.ManagerID, owner, phase, path)
	}
	return nil
}

func addWorkspaceBytes(total *int64, size int64, path string, limits Limits) error {
	if limits.MaxCandidateFileBytes <= 0 || size > limits.MaxCandidateFileBytes {
		return fmt.Errorf("workspace file %s exceeds size limit", path)
	}
	if limits.MaxCandidateBytes <= 0 || size > limits.MaxCandidateBytes-*total {
		return fmt.Errorf("workspace delta exceeds candidate byte limit")
	}
	*total += size
	return nil
}

func reserveObservedPath(used map[string]string, path string) error {
	key := strings.ToLower(path)
	if prior, exists := used[key]; exists {
		if prior != path {
			return fmt.Errorf("workspace delta contains case-alias paths %q and %q", prior, path)
		}
		return fmt.Errorf("workspace delta uses path more than once: %s", path)
	}
	used[key] = path
	return nil
}

func workspaceSnapshotMode(mode string) string {
	switch mode {
	case "100644":
		return snapshotMode("0644")
	case "100755":
		return snapshotMode("0755")
	default:
		return ""
	}
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// applyAgentCandidate selects the explicit byte source returned by the Host.
func applyAgentCandidate(base candidateData, result agentexec.RunResult, config projectwork.Config, report projectmodel.Report, task ManagerTask, phase string, conflictPaths []string, limits Limits, input *Snapshot) (candidateData, error) {
	if result.Delta != nil {
		return applyWorkspaceDelta(base, *result.Delta, result.Response.CandidateFiles, config, report, task, phase, conflictPaths, limits, input)
	}
	return applyProposal(base, result.Response.CandidateFiles, config, report, task, phase, conflictPaths, limits, input)
}
