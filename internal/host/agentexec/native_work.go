package agentexec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

const nativeWorkspaceAPIVersion = "markitect.example.org/native-workspace/v1"

// validateNativeWorkspaceRequest keeps the native filesystem contract inside
// the projectrun Manager executor flow. WorkspaceMode is configuration, while
// nativeWorkspace is invocation context; both must agree before a process is
// allowed to run.
func validateNativeWorkspaceRequest(workspaceMode string, req Request) error {
	if workspaceMode != "" && workspaceMode != "scoped" {
		return errors.New("workspaceMode must be empty or scoped")
	}
	if workspaceMode != "scoped" {
		return nil
	}
	if req.Role != RoleExecutor {
		return errors.New("scoped native workspace is supported only for projectrun Manager executor tasks")
	}
	var context struct {
		Kind            string `json:"kind"`
		Phase           string `json:"phase"`
		NativeWorkspace *struct {
			APIVersion   string          `json:"apiVersion"`
			Instructions json.RawMessage `json:"instructions"`
		} `json:"nativeWorkspace"`
	}
	if err := json.Unmarshal(req.Context, &context); err != nil {
		return errors.New("scoped native workspace context is malformed")
	}
	if context.Kind != "projectrun-task/v1" || (context.Phase != "work" && context.Phase != "integrate" && context.Phase != "repair") || context.NativeWorkspace == nil ||
		context.NativeWorkspace.APIVersion != nativeWorkspaceAPIVersion {
		return errors.New("scoped native workspace requires projectrun-task/v1 and nativeWorkspace v1 context")
	}
	var instructions []json.RawMessage
	if len(context.NativeWorkspace.Instructions) == 0 || json.Unmarshal(context.NativeWorkspace.Instructions, &instructions) != nil || len(instructions) == 0 || len(instructions) > 128 {
		return errors.New("scoped native workspace requires a bounded nonempty instruction list")
	}
	return nil
}

func validateNativeWork(work NativeWork) error {
	if !validDigest(work.WorkspaceBaseDigest) || !validDigest(work.WorkspaceFinalDigest) || !validDigest(work.DeltaDigest) {
		return errors.New("nativeWork digests must be lowercase sha256 values")
	}
	if work.ChangedPaths == nil || len(work.ChangedPaths) > maxArtifactCount {
		return errors.New("nativeWork changedPaths must be a bounded array")
	}
	seen := make(map[string]struct{}, len(work.ChangedPaths))
	previous := ""
	for index, path := range work.ChangedPaths {
		clean, key, err := portablePath(path)
		if err != nil || clean != path {
			return fmt.Errorf("nativeWork changed path is unsafe or non-canonical")
		}
		if index > 0 && path <= previous {
			return errors.New("nativeWork changedPaths must be sorted and unique")
		}
		if _, exists := seen[key]; exists {
			return errors.New("nativeWork changedPaths contain a duplicate or alias")
		}
		seen[key] = struct{}{}
		previous = path
	}
	if work.ToolCalls < 0 {
		return errors.New("nativeWork toolCalls cannot be negative")
	}
	if work.HelperStarts != 0 || work.HelperAccounting != "disabled" {
		return errors.New("nativeWork helper accounting must declare zero starts and disabled accounting")
	}
	return nil
}

func validateNativeWorkProposal(work NativeWork, files []CandidateFile) error {
	paths := make([]string, len(files))
	for index, file := range files {
		paths[index] = file.Path
	}
	sort.Strings(paths)
	if len(paths) != len(work.ChangedPaths) {
		return errors.New("nativeWork changedPaths do not match proposed candidateFiles")
	}
	for index := range paths {
		if paths[index] != work.ChangedPaths[index] {
			return errors.New("nativeWork changedPaths do not match proposed candidateFiles")
		}
	}
	deltaDigest, err := nativeDeltaDigest(files)
	if err != nil {
		return errors.New("nativeWork proposed delta could not be canonically encoded")
	}
	if work.DeltaDigest != deltaDigest {
		return errors.New("nativeWork deltaDigest does not match proposed candidateFiles")
	}
	return nil
}

// nativeDeltaDigest matches codexrunner/native_work.py: sorted changed file
// objects with sorted keys, compact UTF-8 JSON, and no trailing newline.
func nativeDeltaDigest(files []CandidateFile) (string, error) {
	ordered := append([]CandidateFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	type deltaFile struct {
		Digest string `json:"digest"`
		Mode   string `json:"mode"`
		Path   string `json:"path"`
	}
	delta := make([]deltaFile, 0, len(ordered))
	for _, file := range ordered {
		contentDigest := sha256.Sum256([]byte(file.Content))
		delta = append(delta, deltaFile{
			Digest: "sha256:" + hex.EncodeToString(contentDigest[:]),
			Mode:   file.Mode,
			Path:   file.Path,
		})
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(delta); err != nil {
		return "", err
	}
	canonical := bytes.TrimSuffix(encoded.Bytes(), []byte("\n"))
	canonical = bytes.ReplaceAll(canonical, []byte(`\u2028`), []byte("\u2028"))
	canonical = bytes.ReplaceAll(canonical, []byte(`\u2029`), []byte("\u2029"))
	return digest(canonical), nil
}

func cloneNativeWork(work *NativeWork) *NativeWork {
	if work == nil {
		return nil
	}
	copy := *work
	copy.ChangedPaths = append([]string(nil), work.ChangedPaths...)
	if work.ChangedPaths != nil && copy.ChangedPaths == nil {
		copy.ChangedPaths = []string{}
	}
	return &copy
}
