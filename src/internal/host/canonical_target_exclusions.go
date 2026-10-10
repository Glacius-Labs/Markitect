package host

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/src/internal/host/records"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// CanonicalTargetExclusion is an owner-supplied exact file path removed from
// the byte scope sent to a Projection Module. It makes no claim about that
// file's content or about the semantics of the remainder of the target root.
type CanonicalTargetExclusion struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// CanonicalExcludedArtifact makes an explicit exclusion visible alongside the
// complete metadata inventory. Missing paths remain visible as missing.
type CanonicalExcludedArtifact struct {
	Path     string                      `json:"path"`
	Reason   string                      `json:"reason"`
	State    string                      `json:"state"`
	Metadata *source.WorkingFileMetadata `json:"metadata,omitempty"`
}

func validateCanonicalTargetExclusionConfig(exclusions []CanonicalTargetExclusion) error {
	if len(exclusions) > 128 {
		return errors.New("target exclusion bound exceeded")
	}
	seen := map[string]string{}
	for _, exclusion := range exclusions {
		if len(exclusion.Path) > 512 {
			return errors.New("target exclusion path exceeds 512 bytes")
		}
		if err := source.ValidateIncludedPaths([]string{exclusion.Path}); err != nil {
			return fmt.Errorf("invalid target exclusion path %q: %w", exclusion.Path, err)
		}
		if !utf8.ValidString(exclusion.Reason) || len(exclusion.Reason) == 0 || len(exclusion.Reason) > 1024 || strings.TrimSpace(exclusion.Reason) == "" || strings.ContainsRune(exclusion.Reason, 0) {
			return errors.New("target exclusion reason must be nonempty UTF-8 text of at most 1024 bytes")
		}
		for _, path := range seen {
			if strings.EqualFold(path, exclusion.Path) {
				return fmt.Errorf("duplicate or aliased target exclusion paths %q and %q", path, exclusion.Path)
			}
		}
		seen[strings.ToLower(exclusion.Path)] = exclusion.Path
	}
	return nil
}

func validateCanonicalTargetExclusionScope(exclusions []CanonicalTargetExclusion, requestIDs []string, requests map[string]canonical.ProjectionRequest, active map[string]records.ProjectionRecord, canonicalPaths, checkPaths []string) error {
	for _, exclusion := range exclusions {
		matches := 0
		for _, id := range requestIDs {
			prefix := requests[id].TargetPrefix
			if exclusion.Path == prefix {
				return fmt.Errorf("target exclusion %q must be a file strictly beneath its declared target root", exclusion.Path)
			}
			if stringsHasPathPrefix(exclusion.Path, prefix) {
				matches++
			}
		}
		if matches != 1 {
			return fmt.Errorf("target exclusion %q must be an exact file beneath one declared target root", exclusion.Path)
		}
		for _, name := range canonicalPaths {
			if canonicalPortablePathsOverlap(exclusion.Path, name) {
				return fmt.Errorf("target exclusion %q overlaps canonical input %q", exclusion.Path, name)
			}
		}
		for _, name := range checkPaths {
			if canonicalPortablePathsOverlap(exclusion.Path, name) {
				return fmt.Errorf("target exclusion %q overlaps declared check input %q", exclusion.Path, name)
			}
		}
		for _, record := range active {
			for _, artifact := range record.Artifacts {
				if canonicalPortablePathsOverlap(exclusion.Path, artifact.Path) {
					return fmt.Errorf("target exclusion %q overlaps active artifact ownership %q", exclusion.Path, artifact.Path)
				}
			}
		}
	}
	return nil
}

func validateCanonicalTargetExclusionInventory(root string, exclusions []CanonicalTargetExclusion, inventory *source.WorkingRootInventory) error {
	paths := make([]string, 0, len(inventory.Entries)+len(exclusions))
	byPath := make(map[string]source.WorkingFileMetadata, len(inventory.Entries))
	for _, entry := range inventory.Entries {
		paths = append(paths, entry.Path)
		byPath[entry.Path] = entry
	}
	for _, exclusion := range exclusions {
		found := false
		for _, entry := range inventory.Entries {
			if strings.EqualFold(exclusion.Path, entry.Path) && exclusion.Path != entry.Path {
				return fmt.Errorf("target exclusion %q is a path alias for inventoried artifact %q", exclusion.Path, entry.Path)
			}
			if exclusion.Path == entry.Path {
				found = true
			}
		}
		if !found {
			probe, err := source.InventoryWorkingRoots(root, []string{exclusion.Path})
			if err != nil {
				return fmt.Errorf("inspect target exclusion metadata %q: %w", exclusion.Path, err)
			}
			if !equalCanonicalValue(probe.Identity, inventory.Identity) {
				return errors.New("repository identity changed while validating target exclusion metadata")
			}
			if len(probe.MissingPrefixes) == 0 {
				return fmt.Errorf("target exclusion %q resolves to a directory or non-file path", exclusion.Path)
			}
			for _, entry := range probe.Entries {
				if prior, exists := byPath[entry.Path]; !exists || prior != entry {
					return fmt.Errorf("target inventory changed while validating exclusion %q", exclusion.Path)
				}
			}
		}
		paths = append(paths, exclusion.Path)
	}
	if err := source.ValidateIncludedPaths(paths); err != nil {
		return fmt.Errorf("target exclusion conflicts with portable target inventory: %w", err)
	}
	return nil
}

func classifyCanonicalTargetExclusions(exclusions []CanonicalTargetExclusion, inventory map[string]source.WorkingFileMetadata) []CanonicalExcludedArtifact {
	result := make([]CanonicalExcludedArtifact, 0, len(exclusions))
	for _, exclusion := range exclusions {
		artifact := CanonicalExcludedArtifact{Path: exclusion.Path, Reason: exclusion.Reason, State: "missing"}
		if metadata, found := inventory[exclusion.Path]; found {
			artifact.State = "present"
			copy := metadata
			artifact.Metadata = &copy
		}
		result = append(result, artifact)
	}
	return result
}

func canonicalTargetExclusionSet(exclusions []CanonicalTargetExclusion) map[string]struct{} {
	result := make(map[string]struct{}, len(exclusions))
	for _, exclusion := range exclusions {
		result[exclusion.Path] = struct{}{}
	}
	return result
}

func canonicalPortablePathsOverlap(left, right string) bool {
	leftParts, rightParts := strings.Split(left, "/"), strings.Split(right, "/")
	count := len(leftParts)
	if len(rightParts) < count {
		count = len(rightParts)
	}
	for i := 0; i < count; i++ {
		if !strings.EqualFold(leftParts[i], rightParts[i]) {
			return false
		}
	}
	return true
}

// ValidateCanonicalControllerExclusionOutputs refuses any candidate output
// that would replace an excluded file or create a file beneath its path. The
// controller calls this before execution/apply when candidates can write.
func ValidateCanonicalControllerExclusionOutputs(outputs map[string][]byte, excludedPaths []string) error {
	exclusions := make([]CanonicalTargetExclusion, 0, len(excludedPaths))
	for _, path := range excludedPaths {
		exclusions = append(exclusions, CanonicalTargetExclusion{Path: path, Reason: "output boundary validation"})
	}
	if err := validateCanonicalTargetExclusionConfig(exclusions); err != nil {
		return err
	}
	for output := range outputs {
		if err := source.ValidateIncludedPaths([]string{output}); err != nil {
			return fmt.Errorf("invalid controller candidate output path %q: %w", output, err)
		}
		for _, excluded := range excludedPaths {
			if canonicalPortablePathsOverlap(output, excluded) {
				return fmt.Errorf("controller candidate output %q overlaps explicitly excluded target path %q", output, excluded)
			}
		}
	}
	return nil
}
