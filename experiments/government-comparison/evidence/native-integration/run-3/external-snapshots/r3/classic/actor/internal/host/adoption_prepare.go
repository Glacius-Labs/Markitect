package host

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/adoption/capture"
)

// AdoptionPreparation is a read-only preview or the result of creating one
// separately owned adoption handoff workspace.
type AdoptionPreparation struct {
	Status             string          `yaml:"status"`
	Handoff            capture.Handoff `yaml:"handoff"`
	Destination        string          `yaml:"destination"`
	CreatedFiles       []string        `yaml:"createdFiles,omitempty"`
	CreatedDirectories []string        `yaml:"createdDirectories,omitempty"`
	Recovery           string          `yaml:"recovery,omitempty"`
}

type preparedEvidence struct {
	handoff capture.Handoff
	blobs   map[string][]byte
	ids     map[string]source.GitIdentity
	stats   map[string]sourceDirectoryStats
}

type sourceDirectoryStats struct {
	root   os.FileInfo
	git    os.FileInfo
	common os.FileInfo
}

// PrepareAdoption captures only the explicitly selected files in scope. A
// preview does not write; a write must present the exact digest from that
// preview and creates an absent external workspace exclusively.
func PrepareAdoption(scope capture.Scope, destination, expectedDigest string, write bool) (*AdoptionPreparation, error) {
	if err := capture.ValidateScope(scope); err != nil {
		return nil, fmt.Errorf("validate adoption scope: %w", err)
	}
	if destination == "" || !filepath.IsAbs(destination) {
		return nil, errors.New("adoption destination must be an explicit absolute path")
	}
	identities, identityRepos, err := identifyAdoptionScope(scope)
	if err != nil {
		return nil, err
	}
	parent, dest, parentInfo, err := validateAdoptionDestination(destination, identityRepos)
	if err != nil {
		return nil, err
	}
	prepared, err := captureAdoptionScope(scope, identities)
	if err != nil {
		return nil, err
	}
	result := &AdoptionPreparation{Status: "planned", Handoff: prepared.handoff, Destination: dest}
	if !write {
		return result, nil
	}
	if expectedDigest == "" || expectedDigest != prepared.handoff.Digest {
		return result, fmt.Errorf("stale adoption preview: --expect must equal handoff digest %s", prepared.handoff.Digest)
	}
	if err := writeAdoptionWorkspace(result, parent, dest, parentInfo, prepared, adoptionWriteOps{}); err != nil {
		return result, err
	}
	result.Status = "written"
	return result, nil
}

func identifyAdoptionScope(scope capture.Scope) (map[string]source.GitIdentity, []capture.Repository, error) {
	// Identify every repository before any selected blob is opened. This avoids
	// acquiring evidence from an earlier repository if a later root is invalid.
	identities := make(map[string]source.GitIdentity, len(scope.Repositories))
	repositories := make([]capture.Repository, 0, len(scope.Repositories))
	for _, repo := range scope.Repositories {
		if !filepath.IsAbs(repo.Root) {
			return nil, nil, fmt.Errorf("source root for %s must be an explicit absolute path", repo.ID)
		}
		identity, err := source.IdentifyGit(repo.Root)
		if err != nil {
			return nil, nil, fmt.Errorf("identify source repository %s: %w", repo.ID, err)
		}
		inputSpelling, err := canonicalUserPath(filepath.Clean(repo.Root))
		identitySpelling, identityErr := canonicalUserPath(identity.Root)
		if err != nil || identityErr != nil || filepath.Clean(repo.Root) != inputSpelling || identity.Root != identitySpelling || identity.Root != filepath.Clean(repo.Root) {
			return nil, nil, fmt.Errorf("source root for %s must use its canonical absolute path spelling", repo.ID)
		}
		identities[repo.ID] = identity
		repositoryIdentity := capture.RepositoryIdentity{Root: identity.Root, GitDir: identity.GitDir, CommonDir: identity.CommonDir, ObjectFormat: identity.ObjectFormat}
		repositoryIdentity.Digest = capture.ValueDigest(repositoryIdentity)
		repositories = append(repositories, capture.Repository{ID: repo.ID, Identity: repositoryIdentity, Commit: repo.Commit})
	}
	return identities, repositories, nil
}

func captureAdoptionScope(scope capture.Scope, identities map[string]source.GitIdentity) (*preparedEvidence, error) {
	h := capture.Handoff{
		APIVersion: capture.HandoffVersion,
		ID:         scope.ID,
		Purpose:    scope.Purpose,
		Review:     scope.Review,
		Privacy:    scope.Privacy,
		Retention:  scope.Retention,
		Coverage:   append([]capture.Coverage{}, scope.Coverage...),
	}
	blobs := map[string][]byte{}
	stats := make(map[string]sourceDirectoryStats, len(scope.Repositories))
	var totalBytes int64
	for _, repo := range scope.Repositories {
		paths := make([]string, 0, len(repo.Paths))
		pathReasons := make(map[string]string, len(repo.Paths))
		for _, selected := range repo.Paths {
			paths = append(paths, selected.Path)
			pathReasons[selected.Path] = selected.Reason
		}
		selectedSnapshot, err := source.LoadSelected(repo.Root, repo.Commit, paths)
		if err != nil {
			return nil, fmt.Errorf("capture selected files from %s: %w", repo.ID, err)
		}
		if !sameSourceIdentity(identities[repo.ID], selectedSnapshot.Identity) {
			return nil, fmt.Errorf("repository identity changed while capturing %s", repo.ID)
		}
		stats[repo.ID], err = statSourceIdentity(selectedSnapshot.Identity)
		if err != nil {
			return nil, fmt.Errorf("record source directory identity %s: %w", repo.ID, err)
		}
		identity := capture.RepositoryIdentity{
			Root:         selectedSnapshot.Identity.Root,
			GitDir:       selectedSnapshot.Identity.GitDir,
			CommonDir:    selectedSnapshot.Identity.CommonDir,
			ObjectFormat: selectedSnapshot.Identity.ObjectFormat,
		}
		identity.Digest = capture.ValueDigest(identity)
		files := make([]capture.File, 0, len(selectedSnapshot.Snapshot.Files))
		selectedPaths := make([]string, 0, len(selectedSnapshot.Snapshot.Files))
		for path := range selectedSnapshot.Snapshot.Files {
			selectedPaths = append(selectedPaths, path)
		}
		sort.Strings(selectedPaths)
		for _, path := range selectedPaths {
			data := selectedSnapshot.Snapshot.Files[path]
			if totalBytes > source.DefaultMaxTotalBytes-int64(len(data)) {
				return nil, fmt.Errorf("selected adoption evidence exceeds total byte limit at %s/%s", repo.ID, path)
			}
			totalBytes += int64(len(data))
			if !utf8.Valid(data) {
				return nil, fmt.Errorf("selected evidence %s/%s is not UTF-8", repo.ID, path)
			}
			files = append(files, capture.File{Path: path, Reason: pathReasons[path], Mode: selectedSnapshot.Snapshot.Modes[path], Digest: capture.Hash(data)})
			blobs[capture.BlobKey(repo.ID, path)] = append([]byte(nil), data...)
		}
		h.Repositories = append(h.Repositories, capture.Repository{
			ID:             repo.ID,
			Identity:       identity,
			Commit:         repo.Commit,
			SnapshotDigest: selectedSnapshot.Snapshot.Digest(),
			Files:          files,
			Exclusions:     append([]capture.Exclusion(nil), repo.Exclusions...),
			Markitect:      repo.Markitect,
		})
	}
	capture.Seal(&h)
	if err := capture.ValidateHandoff(h, blobs); err != nil {
		return nil, fmt.Errorf("validate captured adoption handoff: %w", err)
	}
	return &preparedEvidence{handoff: h, blobs: blobs, ids: identities, stats: stats}, nil
}

func statSourceIdentity(identity source.GitIdentity) (sourceDirectoryStats, error) {
	var result sourceDirectoryStats
	for _, item := range []struct {
		path string
		dst  *os.FileInfo
	}{{identity.Root, &result.root}, {identity.GitDir, &result.git}, {identity.CommonDir, &result.common}} {
		info, err := os.Lstat(item.path)
		if err != nil || !info.IsDir() || isReparsePoint(info) {
			return sourceDirectoryStats{}, fmt.Errorf("source identity path is no longer a real directory: %s", item.path)
		}
		*item.dst = info
	}
	return result, nil
}

func sameSourceIdentity(a, b source.GitIdentity) bool {
	return a.Root == b.Root && a.GitDir == b.GitDir && a.CommonDir == b.CommonDir && a.ObjectFormat == b.ObjectFormat
}

func validateAdoptionDestination(destination string, repositories []capture.Repository) (string, string, os.FileInfo, error) {
	abs, err := filepath.Abs(destination)
	if err != nil {
		return "", "", nil, fmt.Errorf("resolve adoption destination: %w", err)
	}
	abs = filepath.Clean(abs)
	parent := filepath.Dir(abs)
	parentInfo, err := os.Lstat(parent)
	if err != nil || !parentInfo.IsDir() || isReparsePoint(parentInfo) {
		return "", "", nil, fmt.Errorf("adoption destination parent must be an existing real directory: %s", parent)
	}
	if err := rejectReparseAncestors(parent); err != nil {
		return "", "", nil, fmt.Errorf("unsafe adoption destination parent: %w", err)
	}
	canonicalParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", "", nil, fmt.Errorf("canonicalize adoption destination parent: %w", err)
	}
	canonicalParent, err = canonicalPathSpelling(canonicalParent)
	if err != nil {
		return "", "", nil, fmt.Errorf("canonicalize adoption destination spelling: %w", err)
	}
	canonicalParent = stripWindowsLongPathPrefix(canonicalParent)
	if !samePathSpelling(filepath.Clean(parent), filepath.Clean(canonicalParent)) {
		return "", "", nil, fmt.Errorf("adoption destination parent must use its canonical path spelling: %s", parent)
	}
	if err := rejectCaseAliases(parent); err != nil {
		return "", "", nil, err
	}
	name := filepath.Base(abs)
	if name == "" || name == "." || name == ".." {
		return "", "", nil, errors.New("adoption destination must name a new directory")
	}
	if err := capture.ExactPath(name); err != nil {
		return "", "", nil, fmt.Errorf("unsafe adoption destination name: %w", err)
	}
	if err := checkDiskComponentAlias(parent, name, name); err != nil {
		return "", "", nil, err
	}
	if _, err := os.Lstat(abs); err == nil {
		return "", "", nil, fmt.Errorf("adoption destination already exists: %s", abs)
	} else if !os.IsNotExist(err) {
		return "", "", nil, fmt.Errorf("inspect adoption destination: %w", err)
	}
	// The leaf is required absent, so Windows cannot resolve it through an
	// existing 8.3 alias. Build it from the already canonicalized parent.
	canonicalDest := filepath.Join(canonicalParent, name)
	if !samePathSpelling(abs, canonicalDest) {
		return "", "", nil, fmt.Errorf("adoption destination must use its canonical path spelling: %s", abs)
	}
	for _, repo := range repositories {
		for _, guarded := range []string{repo.Identity.Root, repo.Identity.GitDir, repo.Identity.CommonDir} {
			if pathsOverlap(canonicalDest, guarded) {
				return "", "", nil, fmt.Errorf("adoption destination overlaps source or Git metadata path %s", guarded)
			}
		}
	}
	return filepath.Dir(canonicalDest), canonicalDest, parentInfo, nil
}

func canonicalUserPath(path string) (string, error) {
	canonical, err := canonicalPathSpelling(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(stripWindowsLongPathPrefix(canonical)), nil
}

func stripWindowsLongPathPrefix(path string) string {
	if strings.HasPrefix(path, `\\?\UNC\`) {
		return `\\` + strings.TrimPrefix(path, `\\?\UNC\`)
	}
	if strings.HasPrefix(path, `\\?\`) {
		return strings.TrimPrefix(path, `\\?\`)
	}
	return path
}

func pathsOverlap(a, b string) bool {
	cleanA, cleanB := filepath.Clean(a), filepath.Clean(b)
	contains := func(root, candidate string) bool {
		rel, err := filepath.Rel(root, candidate)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return false
		}
		return true
	}
	if filepath.VolumeName(cleanA) != "" || filepath.VolumeName(cleanB) != "" {
		if strings.EqualFold(cleanA, cleanB) {
			return true
		}
		// Compare path components case-insensitively for Windows roots, including
		// candidates that differ only by spelling.
		fold := func(s string) string { return strings.ToLower(filepath.Clean(s)) }
		return contains(fold(cleanA), fold(cleanB)) || contains(fold(cleanB), fold(cleanA))
	}
	return contains(cleanA, cleanB) || contains(cleanB, cleanA)
}

func rejectCaseAliases(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	volume := filepath.VolumeName(abs)
	rest := strings.TrimPrefix(abs, volume)
	current := volume + string(filepath.Separator)
	for _, part := range strings.Split(strings.Trim(rest, string(filepath.Separator)), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return fmt.Errorf("inspect destination path component %s: %w", current, err)
		}
		found := false
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), part) {
				if entry.Name() != part {
					return fmt.Errorf("adoption destination uses a case alias for %s", filepath.Join(current, entry.Name()))
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("adoption destination parent changed during validation: %s", current)
		}
		current = filepath.Join(current, part)
	}
	return nil
}

func sortedRepoIDs(ids map[string]source.GitIdentity) []string {
	result := make([]string, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
