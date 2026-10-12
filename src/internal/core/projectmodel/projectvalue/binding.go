package projectvalue

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type structureShape struct {
	ScopeID               string   `json:"scopeId"`
	ScopeName             string   `json:"scopeName"`
	Goal                  string   `json:"goal"`
	Operation             string   `json:"operation"`
	ManagerIDs            []string `json:"managerIds"`
	ResponsibleManagerIDs []string `json:"responsibleManagerIds"`
	RequiredArtifacts     []string `json:"requiredArtifacts"`
	Files                 []string `json:"files"`
	Checks                []string `json:"checks"`
}

// BindingDigest returns a stable digest of the complete current basis and
// named scope. No random run or plan ID participates in this identity.
func BindingDigest(binding Binding) (string, error) {
	binding, err := CanonicalBinding(binding)
	if err != nil {
		return "", err
	}
	return digest(binding)
}

// StructureDigest binds the exact named-scope file structure and its declared
// Managers, artifacts, operation, and checks, independent of run identifiers.
func StructureDigest(binding Binding) (string, error) {
	binding, err := CanonicalBinding(binding)
	if err != nil {
		return "", err
	}
	return digest(structureShape{
		ScopeID: binding.ScopeID, ScopeName: binding.ScopeName, Goal: binding.Goal, Operation: binding.Operation,
		ManagerIDs: binding.ManagerIDs, ResponsibleManagerIDs: binding.ResponsibleManagerIDs, RequiredArtifacts: binding.RequiredArtifacts, Files: binding.FileStructure, Checks: binding.Checks,
	})
}

// CanonicalBinding validates and returns the canonical ordering used in
// persisted readiness bindings and their stable digests.
func CanonicalBinding(binding Binding) (Binding, error) {
	if strings.TrimSpace(binding.RepositoryRoot) == "" || strings.TrimSpace(binding.Branch) == "" || strings.TrimSpace(binding.Head) == "" && binding.ModelAccepted {
		return Binding{}, fmt.Errorf("exploration binding requires a repository root, branch, and accepted HEAD identity")
	}
	if binding.ModelAccepted && (!IsFullObjectID(binding.Head) || binding.ModelRevision != binding.Head || strings.TrimSpace(binding.AcceptancePolicy) == "") {
		return Binding{}, fmt.Errorf("accepted readiness binding requires a fixed committed model at HEAD and an explicit acceptance policy")
	}
	for _, field := range []struct{ name, value string }{{"projectDigest", binding.ProjectDigest}, {"modelDigest", binding.ModelDigest}, {"snapshotDigest", binding.SnapshotDigest}, {"selectionDigest", binding.SelectionDigest}} {
		if !IsValidDigest(field.value) {
			return Binding{}, fmt.Errorf("binding %s is not a sha256 digest", field.name)
		}
	}
	if binding.ScopeID == "" || strings.TrimSpace(binding.ScopeName) == "" || strings.TrimSpace(binding.Goal) == "" || binding.Operation == "" || binding.ManagerIDs == nil || !uniqueValues(binding.ManagerIDs, false) || binding.RequiredArtifacts == nil || binding.FileStructure == nil || binding.Checks == nil || binding.BasisFiles == nil {
		return Binding{}, fmt.Errorf("binding requires a named scope, goal, operation, Managers, and explicit artifact/file/check/basis arrays")
	}
	if !uniqueValues(binding.ResponsibleManagerIDs, false) {
		return Binding{}, fmt.Errorf("binding responsible Manager IDs must be unique")
	}
	if !uniqueValues(binding.RequiredArtifacts, true) || !uniqueValues(binding.FileStructure, false) || !uniqueValues(binding.Checks, true) {
		return Binding{}, fmt.Errorf("binding artifact, file, and check lists must be unique and nonempty where applicable")
	}
	for _, value := range binding.FileStructure {
		if strings.TrimSpace(value) == "" || len(value) > 1024 || strings.ContainsRune(value, 0) {
			return Binding{}, fmt.Errorf("binding file structure entry %q is invalid", value)
		}
	}
	for _, basis := range binding.BasisFiles {
		if !IsRepositoryPath(basis.Path) || !IsValidDigest(basis.Digest) {
			return Binding{}, fmt.Errorf("binding basis file %q is invalid", basis.Path)
		}
	}
	seenBasis := map[string]bool{}
	for _, basis := range binding.BasisFiles {
		if seenBasis[basis.Path] {
			return Binding{}, fmt.Errorf("binding repeats basis file %q", basis.Path)
		}
		seenBasis[basis.Path] = true
	}
	binding.ManagerIDs = SortedCopy(binding.ManagerIDs)
	binding.ResponsibleManagerIDs = SortedCopy(binding.ResponsibleManagerIDs)
	binding.RequiredArtifacts = SortedCopy(binding.RequiredArtifacts)
	binding.FileStructure = SortedCopy(binding.FileStructure)
	binding.Checks = SortedCopy(binding.Checks)
	binding.BasisFiles = append([]BasisFile{}, binding.BasisFiles...)
	sort.Slice(binding.BasisFiles, func(i, j int) bool { return binding.BasisFiles[i].Path < binding.BasisFiles[j].Path })
	return binding, nil
}

// IsValidDigest reports whether value is a sha256 digest using the persisted
// "sha256:<hex>" framing used by project and exploration records.
func IsValidDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

// IsFullObjectID reports whether value is a full 40- or 64-digit Git object ID.
func IsFullObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// SortedCopy returns a sorted, non-nil copy while preserving the distinction
// between nil and explicitly empty input through the caller's validation.
func SortedCopy(values []string) []string {
	copyValues := append([]string{}, values...)
	sort.Strings(copyValues)
	return copyValues
}

func uniqueValues(values []string, nonempty bool) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if nonempty && strings.TrimSpace(value) == "" || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func digest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
