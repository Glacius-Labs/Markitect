package dotnet

import (
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

const CandidateUnverified = "candidate-unverified"

// Input carries a bounded semantic scope and candidate bytes supplied by the
// Host or its agent. Guidance is keyed by core.KindIdentity.Key().
type Input struct {
	Definitions       []core.Definition
	Schemas           []core.Schema
	Guidance          map[string]string
	AllowedPaths      []string
	CandidateFiles    map[string][]byte
	Policies          []core.Definition
	TargetPrefix      string
	AllowedRoots      []string
	RequestDigest     string
	CanonicalAffected bool
	InventoryComplete bool
	Previous          *PriorProjection
	ObservedArtifacts []ArtifactObservation
	Verification      *VerificationBinding
	Repair            *RepairEvidence
}

// Escalation identifies a representation decision that needs explicit intent.
type Escalation struct {
	Code     string
	Identity string
	Message  string
}

// Result reports candidate bytes pending Host checks. It never asserts that a
// candidate is a valid or accepted semantic representation.
type Result struct {
	Status      string
	Files       map[string][]byte
	Escalations []Escalation
	Diagnostics []core.Diagnostic
}

// Evaluate accepts only supplied candidate content within the exact allowed
// target set. It does not generate C# or infer conventions from Kind names.
func Evaluate(input Input) Result {
	result := Result{Files: map[string][]byte{}}
	if len(input.Definitions) == 0 {
		result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Code: "dotnet.scope.empty", Message: "the selected projection scope contains no Definitions"})
		return result
	}

	kinds := make(map[string]core.Kind)
	for _, schema := range input.Schemas {
		for name, kind := range schema.Kinds {
			kinds[(core.KindIdentity{APIVersion: schema.APIVersion, Kind: name}).Key()] = kind
		}
	}
	missingGuidance := false
	for _, definition := range input.Definitions {
		kindID := core.KindIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind}
		kindKey := kindID.Key()
		if _, ok := kinds[kindKey]; !ok {
			result.Escalations = append(result.Escalations, Escalation{
				Code: "dotnet.kind-missing", Identity: kindKey,
				Message: "the selected Definition Kind has no explicitly supplied Schema contract",
			})
			missingGuidance = true
		}
		if strings.TrimSpace(input.Guidance[kindKey]) == "" {
			result.Escalations = append(result.Escalations, Escalation{
				Code: "dotnet.guidance-missing", Identity: kindKey,
				Message: "project-owned representation guidance is required for this selected Kind",
			})
			missingGuidance = true
		}
	}
	if missingGuidance {
		return result
	}
	if len(input.CandidateFiles) == 0 {
		result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Code: "dotnet.candidate.empty", Message: "no candidate artifacts were supplied; projection is incomplete"})
		return result
	}

	allowed := make(map[string]bool, len(input.AllowedPaths))
	for _, target := range input.AllowedPaths {
		if validTargetPath(target) {
			allowed[target] = true
		}
	}
	candidatePaths := make([]string, 0, len(input.CandidateFiles))
	for target := range input.CandidateFiles {
		candidatePaths = append(candidatePaths, target)
	}
	sortStrings(candidatePaths)

	valid := true
	for _, target := range candidatePaths {
		content := input.CandidateFiles[target]
		if !allowed[target] {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Code: "dotnet.target.unowned", Message: "candidate path is not in the exact allowed target set: " + target})
			valid = false
			continue
		}
		ext := filepath.Ext(target)
		if ext != ".cs" && ext != ".csproj" {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Code: "dotnet.target.unsupported", Message: "only .cs and .csproj candidate paths are supported: " + target})
			valid = false
		}
		if len(content) == 0 || !utf8.Valid(content) || strings.IndexByte(string(content), 0) >= 0 {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Code: "dotnet.content.invalid", Message: "candidate must contain nonempty UTF-8 text without NUL bytes: " + target})
			valid = false
		}
	}
	if !valid {
		return result
	}
	for _, target := range candidatePaths {
		result.Files[target] = append([]byte(nil), input.CandidateFiles[target]...)
	}
	result.Status = CandidateUnverified
	return result
}

func validTargetPath(target string) bool {
	if target == "" || strings.Contains(target, "\\") || path.IsAbs(target) || filepath.IsAbs(target) || filepath.VolumeName(target) != "" || path.Clean(target) != target || target == "." {
		return false
	}
	for _, segment := range strings.Split(target, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func sortStrings(values []string) {
	// Small local helper keeps this package independent of other project packages.
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
