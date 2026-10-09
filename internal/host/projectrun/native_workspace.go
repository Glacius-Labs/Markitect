package projectrun

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const (
	nativeWorkspaceAPIVersion = "markitect.example.org/native-workspace/v1"
	maxNativeInstructionFiles = 128
	maxNativeInstructionBytes = 4 << 20
)

type nativeWorkspaceContext struct {
	APIVersion   string               `json:"apiVersion"`
	Instructions []agentexec.Artifact `json:"instructions"`
}

// buildNativeWorkspace binds the explicitly selected, read-only project
// instructions for one native Manager invocation to the fixed accepted
// revision and the current working tree. It does not materialize a workspace;
// the provider adapter receives these exact artifacts and owns materialization.
func buildNativeWorkspace(root, revision string, project *Project, agent Agent) (*nativeWorkspaceContext, error) {
	if agent.WorkspaceMode == "" {
		return nil, nil
	}
	if agent.WorkspaceMode != "scoped" && agent.WorkspaceMode != "git" {
		return nil, fmt.Errorf("unsupported native workspace mode %q", agent.WorkspaceMode)
	}
	if project == nil || project.Snapshot == nil {
		return nil, fmt.Errorf("native workspace requires a selected project snapshot")
	}
	if revision == "" || project.Revision != revision || project.Snapshot.ID != revision {
		return nil, fmt.Errorf("native workspace revision differs from the fixed project snapshot")
	}
	paths, err := normalizeNativeInstructionPaths(agent.InstructionPaths, project.Config)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("scoped native workspace requires explicit instructionPaths")
	}
	if len(paths) > maxNativeInstructionFiles {
		return nil, fmt.Errorf("native workspace instruction count exceeds %d", maxNativeInstructionFiles)
	}

	root, err = filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve native workspace project root: %w", err)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("native workspace project root must be a real directory")
	}
	fixed, err := source.LoadSelected(root, revision, paths)
	if err != nil {
		return nil, fmt.Errorf("load fixed native instruction snapshot: %w", err)
	}
	working, err := source.ObserveSelectedWorking(root, paths)
	if err != nil {
		return nil, fmt.Errorf("observe working native instructions: %w", err)
	}
	if working.Snapshot == nil || len(working.MissingPaths) != 0 {
		return nil, fmt.Errorf("native workspace instruction is missing from the working tree")
	}

	artifacts := make([]agentexec.Artifact, 0, len(paths))
	var total int
	for _, path := range paths {
		if err := rejectReparsePath(root, path); err != nil {
			return nil, fmt.Errorf("native instruction %s: %w", path, err)
		}
		fixedBytes, exists := fixed.Snapshot.Files[path]
		if !exists {
			return nil, fmt.Errorf("fixed native instruction %s is missing", path)
		}
		fixedMode := fixed.Snapshot.Modes[path]
		if fixedMode != "100644" && fixedMode != "100755" {
			return nil, fmt.Errorf("native instruction %s has unsupported fixed mode %q", path, fixedMode)
		}
		workingBytes, exists := working.Snapshot.Files[path]
		if !exists || string(workingBytes) != string(fixedBytes) || working.Snapshot.Modes[path] != fixedMode {
			return nil, fmt.Errorf("native instruction %s differs from the fixed accepted revision", path)
		}
		if projectBytes, included := project.Snapshot.Files[path]; included && (string(projectBytes) != string(fixedBytes) || project.Snapshot.Modes[path] != fixedMode) {
			return nil, fmt.Errorf("native instruction %s differs from the selected project snapshot", path)
		}
		if !utf8.Valid(fixedBytes) {
			return nil, fmt.Errorf("native instruction %s is not valid UTF-8", path)
		}
		if len(fixedBytes) > maxNativeInstructionBytes-total {
			return nil, fmt.Errorf("native instructions exceed %d bytes", maxNativeInstructionBytes)
		}
		total += len(fixedBytes)
		absolute := filepath.Clean(filepath.Join(root, filepath.FromSlash(path)))
		if !pathUnder(root, absolute) {
			return nil, fmt.Errorf("native instruction %s escaped the project root", path)
		}
		runtimeMode := protocolMode(fixedMode)
		if runtimeMode == "" || !hasNativeRuntimePin(agent.RuntimeFiles, absolute, runtimeMode, "sha256:"+digestBytes(fixedBytes)) {
			return nil, fmt.Errorf("native instruction %s is missing its exact absolute RuntimeFiles pin", path)
		}
		artifacts = append(artifacts, agentexec.Artifact{Path: path, Mode: fixedMode, Digest: digestBytes(fixedBytes), Content: append([]byte(nil), fixedBytes...)})
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	return &nativeWorkspaceContext{APIVersion: nativeWorkspaceAPIVersion, Instructions: artifacts}, nil
}

func normalizeNativeInstructionPaths(paths []string, config projectwork.Config) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	copyPaths := append([]string(nil), paths...)
	if err := validatePortablePaths(copyPaths); err != nil {
		return nil, fmt.Errorf("native instruction paths: %w", err)
	}
	seen := make(map[string]string, len(copyPaths))
	for _, path := range copyPaths {
		key := strings.ToLower(path)
		if prior, ok := seen[key]; ok {
			return nil, fmt.Errorf("native instruction paths %q and %q alias", prior, path)
		}
		seen[key] = path
		if !declaredNativeInstructionPath(path, config) {
			return nil, fmt.Errorf("native instruction path %q is outside the exact declared project instruction paths", path)
		}
	}
	sort.Strings(copyPaths)
	return copyPaths, nil
}

func declaredNativeInstructionPath(path string, config projectwork.Config) bool {
	if !safeRepoPath(path) || strings.HasPrefix(strings.ToLower(path), ".markitect/") || !strings.EqualFold(filepath.Ext(path), ".md") {
		return false
	}
	for _, tool := range projectwork.ToolPaths(config) {
		if tool.Selector == path && !tool.Operational && !strings.HasSuffix(tool.Selector, "/") {
			return true
		}
	}
	return false
}

func hasNativeRuntimePin(files []agentexec.RuntimeFile, absolutePath, mode, digest string) bool {
	for _, file := range files {
		candidate, err := filepath.Abs(file.Path)
		if err != nil || filepath.Clean(candidate) != absolutePath {
			continue
		}
		if file.Mode == mode && file.Digest == digest {
			return true
		}
	}
	return false
}

func digestBytes(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}
