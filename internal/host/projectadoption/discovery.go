package projectadoption

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const (
	maxSelectedFiles = 2048
	// Keep serialized Discovery records comfortably below the closed-record
	// decoder's bound even when JSON must escape control characters.
	maxEvidenceBytes = 4 << 20
)

// Discover acquires only caller-selected exact file blobs from a full Git
// commit. Scope roots, exclusions, and unselected paths are declarations, not
// recursive read permissions.
func Discover(root string, request DiscoveryRequest) (Discovery, error) {
	if err := validateDiscoveryRequest(request); err != nil {
		return Discovery{}, err
	}
	paths := make([]string, 0, len(request.Selected))
	for _, selected := range request.Selected {
		paths = append(paths, selected.Path)
	}
	loaded, err := source.LoadSelected(root, request.Commit, paths)
	if err != nil {
		return Discovery{}, fmt.Errorf("load exact selected discovery blobs: %w", err)
	}
	selected := append([]SelectedPath(nil), request.Selected...)
	sort.Slice(selected, func(i, j int) bool { return selected[i].Path < selected[j].Path })
	exclusions := canonicalPathReasons(request.Exclusions)
	unselected := canonicalPathReasons(request.Unselected)
	roots := append([]string(nil), request.ScopeRoots...)
	sort.Strings(roots)
	identity := RepositoryIdentity{
		Root: loaded.Identity.Root, GitDir: loaded.Identity.GitDir,
		CommonDir: loaded.Identity.CommonDir, ObjectFormat: loaded.Identity.ObjectFormat,
	}
	identity.Digest = digestValue(identity)
	discovery := Discovery{
		APIVersion: DiscoveryVersion, ID: request.ID, Purpose: request.Purpose, Review: request.Review,
		Identity: identity, Commit: request.Commit, ScopeRoots: roots, Selected: selected,
		Exclusions: exclusions, Unselected: unselected,
		Evidence: make([]Evidence, 0, len(selected)),
	}
	var total int64
	for _, item := range selected {
		content := loaded.Snapshot.Files[item.Path]
		if !utf8.Valid(content) {
			return Discovery{}, fmt.Errorf("selected evidence %q must be UTF-8 text", item.Path)
		}
		total += int64(len(content))
		if total > maxEvidenceBytes {
			return Discovery{}, fmt.Errorf("selected discovery evidence exceeds %d bytes", maxEvidenceBytes)
		}
		discovery.Evidence = append(discovery.Evidence, Evidence{
			ID: item.ID, Path: item.Path, Basis: item.Basis, Mode: loaded.Snapshot.Modes[item.Path],
			Digest: digestBytes(content), Content: string(content),
		})
	}
	SealDiscovery(&discovery)
	if err := ValidateDiscovery(discovery); err != nil {
		return Discovery{}, fmt.Errorf("validate captured discovery: %w", err)
	}
	return discovery, nil
}

// RefreshDiscovery re-reads the same selected blobs at the same full commit
// and rejects repository substitution or any changed recorded field.
func RefreshDiscovery(root string, discovery Discovery) (Discovery, error) {
	if err := ValidateDiscovery(discovery); err != nil {
		return Discovery{}, err
	}
	request := DiscoveryRequest{
		APIVersion: DiscoveryVersion, ID: discovery.ID, Purpose: discovery.Purpose,
		Review: discovery.Review, Commit: discovery.Commit,
		ScopeRoots: append([]string(nil), discovery.ScopeRoots...),
		Selected:   append([]SelectedPath(nil), discovery.Selected...),
		Exclusions: canonicalPathReasons(discovery.Exclusions),
		Unselected: canonicalPathReasons(discovery.Unselected),
	}
	refreshed, err := Discover(root, request)
	if err != nil {
		return Discovery{}, err
	}
	if refreshed.Digest != discovery.Digest {
		return Discovery{}, errors.New("discovery source or repository identity changed; create a new discovery")
	}
	return refreshed, nil
}

func validateDiscoveryRequest(request DiscoveryRequest) error {
	if request.APIVersion != DiscoveryVersion || !validID(request.ID) || strings.TrimSpace(request.Purpose) == "" || strings.TrimSpace(request.Review) == "" {
		return errors.New("discovery requires supported apiVersion, stable ID, purpose, and review reference")
	}
	if len(request.Commit) != 40 && len(request.Commit) != 64 || request.Commit != strings.ToLower(request.Commit) {
		return errors.New("discovery requires a full lowercase Git commit ID")
	}
	if len(request.Selected) == 0 || len(request.Selected) > maxSelectedFiles {
		return fmt.Errorf("discovery requires 1..%d exact selected files", maxSelectedFiles)
	}
	if request.Exclusions == nil || request.Unselected == nil {
		return errors.New("discovery must explicitly provide exclusions and unselected scope lists, using empty lists when none apply")
	}
	if len(request.ScopeRoots) == 0 {
		return errors.New("discovery requires at least one explicitly declared scope root")
	}
	roots := append([]string(nil), request.ScopeRoots...)
	for _, root := range roots {
		if root != "." {
			if err := validateRepoPath(root); err != nil {
				return fmt.Errorf("invalid scope root %q: %w", root, err)
			}
		}
	}
	for i, root := range roots {
		for _, other := range roots[i+1:] {
			if root == "." || other == "." || samePath(root, other) || pathWithin(root, other) || pathWithin(other, root) {
				return fmt.Errorf("duplicate or overlapping scope roots %q and %q", root, other)
			}
		}
	}
	seenIDs := map[string]bool{}
	selectedPaths := make([]string, 0, len(request.Selected))
	for _, item := range request.Selected {
		if !validID(item.ID) || seenIDs[item.ID] || strings.TrimSpace(item.Reason) == "" || !validEvidenceBasis(item.Basis) {
			return fmt.Errorf("selected path requires unique stable ID, reason, and explicit source basis: %q", item.Path)
		}
		seenIDs[item.ID] = true
		if err := validateRepoPath(item.Path); err != nil {
			return err
		}
		if !withinAnyRoot(item.Path, roots) {
			return fmt.Errorf("selected path %q is outside every declared scope root", item.Path)
		}
		selectedPaths = append(selectedPaths, item.Path)
	}
	if err := validateDistinctPaths(selectedPaths); err != nil {
		return err
	}
	if err := validatePathReasons("exclusion", request.Exclusions, roots); err != nil {
		return err
	}
	if err := validatePathReasons("unselected scope", request.Unselected, roots); err != nil {
		return err
	}
	for _, selected := range selectedPaths {
		for _, excluded := range request.Exclusions {
			if pathOverlap(selected, excluded.Path) {
				return fmt.Errorf("selected path %q overlaps excluded path %q", selected, excluded.Path)
			}
		}
		for _, omitted := range request.Unselected {
			if pathOverlap(selected, omitted.Path) {
				return fmt.Errorf("selected path %q overlaps explicitly unselected scope %q", selected, omitted.Path)
			}
		}
	}
	for _, excluded := range request.Exclusions {
		for _, omitted := range request.Unselected {
			if pathOverlap(excluded.Path, omitted.Path) {
				return fmt.Errorf("excluded path %q overlaps unselected scope %q", excluded.Path, omitted.Path)
			}
		}
	}
	return nil
}

func ValidateDiscovery(discovery Discovery) error {
	if discovery.APIVersion != DiscoveryVersion || !validID(discovery.ID) || strings.TrimSpace(discovery.Purpose) == "" || strings.TrimSpace(discovery.Review) == "" {
		return errors.New("invalid discovery identity or required metadata")
	}
	request := DiscoveryRequest{
		APIVersion: DiscoveryVersion, ID: discovery.ID, Purpose: discovery.Purpose,
		Review: discovery.Review, Commit: discovery.Commit,
		ScopeRoots: discovery.ScopeRoots, Selected: discovery.Selected,
		Exclusions: discovery.Exclusions, Unselected: discovery.Unselected,
	}
	if err := validateDiscoveryRequest(request); err != nil {
		return err
	}
	identity := discovery.Identity
	identity.Digest = ""
	if identity.Root == "" || identity.GitDir == "" || identity.CommonDir == "" || (identity.ObjectFormat != "sha1" && identity.ObjectFormat != "sha256") || digestValue(identity) != discovery.Identity.Digest {
		return errors.New("invalid discovery repository identity binding")
	}
	if identity.ObjectFormat == "sha1" && len(discovery.Commit) != 40 || identity.ObjectFormat == "sha256" && len(discovery.Commit) != 64 {
		return errors.New("discovery commit length does not match repository object format")
	}
	if len(discovery.Evidence) != len(discovery.Selected) {
		return errors.New("discovery must contain exactly one evidence item per selected path")
	}
	byID := make(map[string]Evidence, len(discovery.Evidence))
	var total int64
	for _, evidence := range discovery.Evidence {
		if _, duplicate := byID[evidence.ID]; duplicate {
			return fmt.Errorf("duplicate evidence ID %q", evidence.ID)
		}
		if !utf8.ValidString(evidence.Content) || evidence.Digest != digestBytes([]byte(evidence.Content)) || (evidence.Mode != "100644" && evidence.Mode != "100755") || !validEvidenceBasis(evidence.Basis) {
			return fmt.Errorf("invalid selected evidence bytes or mode for %q", evidence.Path)
		}
		total += int64(len(evidence.Content))
		if total > maxEvidenceBytes {
			return fmt.Errorf("selected discovery evidence exceeds %d bytes", maxEvidenceBytes)
		}
		byID[evidence.ID] = evidence
	}
	for _, selected := range discovery.Selected {
		evidence, ok := byID[selected.ID]
		if !ok || evidence.Path != selected.Path || evidence.Basis != selected.Basis {
			return fmt.Errorf("selected path %q is missing matching evidence", selected.Path)
		}
	}
	copy := discovery
	copy.Digest = ""
	if digestValue(copy) != discovery.Digest {
		return errors.New("discovery digest mismatch")
	}
	return nil
}

func SealDiscovery(discovery *Discovery) {
	if discovery == nil {
		return
	}
	copy := *discovery
	copy.Digest = ""
	discovery.Digest = digestValue(copy)
}

func validatePathReasons(kind string, items []PathReason, roots []string) error {
	paths := make([]string, 0, len(items))
	for _, item := range items {
		if err := validateRepoPath(item.Path); err != nil {
			return err
		}
		if strings.TrimSpace(item.Reason) == "" {
			return fmt.Errorf("%s %q requires a reason", kind, item.Path)
		}
		if !withinAnyRoot(item.Path, roots) {
			return fmt.Errorf("%s %q is outside every declared scope root", kind, item.Path)
		}
		paths = append(paths, item.Path)
	}
	return validateDistinctPaths(paths)
}

func validateRepoPath(value string) error {
	if value == "" || value == "." || strings.ContainsAny(value, `\:*?[]<>"|`+"\x00\r\n\t") || path.Clean(value) != value || strings.HasPrefix(value, "/") {
		return fmt.Errorf("unsafe exact repository path %q", value)
	}
	for _, component := range strings.Split(value, "/") {
		for _, r := range component {
			if r < 32 {
				return fmt.Errorf("unsafe exact repository path %q", value)
			}
		}
		upper := strings.ToUpper(strings.SplitN(component, ".", 2)[0])
		reserved := upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" || reservedWindowsDevice(upper)
		if component == "" || component == "." || component == ".." || strings.EqualFold(component, ".git") || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") || reserved {
			return fmt.Errorf("unsafe exact repository path %q", value)
		}
	}
	if strings.ContainsAny(value, "*?[]") {
		return fmt.Errorf("repository path %q must be literal; globs are not supported", value)
	}
	return nil
}

func validateDistinctPaths(paths []string) error {
	for i, path := range paths {
		for _, other := range paths[i+1:] {
			if samePath(path, other) || pathWithin(path, other) || pathWithin(other, path) {
				return fmt.Errorf("duplicate, case-aliased, or overlapping paths %q and %q", path, other)
			}
		}
	}
	return nil
}

func samePath(a, b string) bool { return strings.EqualFold(a, b) }
func pathWithin(pathValue, root string) bool {
	if root == "." || len(pathValue) <= len(root) || pathValue[len(root)] != '/' {
		return false
	}
	return strings.EqualFold(pathValue[:len(root)], root)
}
func pathOverlap(a, b string) bool { return samePath(a, b) || pathWithin(a, b) || pathWithin(b, a) }
func withinAnyRoot(pathValue string, roots []string) bool {
	for _, root := range roots {
		if root == "." || samePath(pathValue, root) || pathWithin(pathValue, root) {
			return true
		}
	}
	return false
}

func canonicalPathReasons(items []PathReason) []PathReason {
	result := append([]PathReason{}, items...)
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func digestBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func digestValue(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return digestBytes(data)
}

func reservedWindowsDevice(value string) bool {
	if len(value) != 4 || (value[:3] != "COM" && value[:3] != "LPT") {
		return false
	}
	return value[3] >= '1' && value[3] <= '9'
}

func validEvidenceBasis(value string) bool {
	switch value {
	case "code", "documentation", "configuration", "test", "runtime-record", "other":
		return true
	default:
		return false
	}
}
