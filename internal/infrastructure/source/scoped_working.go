package source

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

// SelectedWorkingSnapshot contains bytes for exactly the requested working
// tree paths. MissingPaths is explicit; the snapshot is provisional and its
// digest covers only present selected files.
type SelectedWorkingSnapshot struct {
	Identity     GitIdentity
	Snapshot     *snapshot.Snapshot
	Requested    []string
	MissingPaths []string
}

// WorkingFileMetadata contains filesystem facts only. It never includes or
// fingerprints file contents.
type WorkingFileMetadata struct {
	Path string `json:"path" yaml:"path"`
	Mode string `json:"mode" yaml:"mode"`
	Size int64  `json:"size" yaml:"size"`
}

// WorkingRootInventory records paths under exact working tree prefixes.
// MetadataDigest binds prefixes, paths, modes, and sizes; it is not a content
// digest and cannot establish artifact byte equality.
type WorkingRootInventory struct {
	Identity        GitIdentity           `json:"identity" yaml:"identity"`
	Prefixes        []string              `json:"prefixes" yaml:"prefixes"`
	Entries         []WorkingFileMetadata `json:"entries" yaml:"entries"`
	MissingPrefixes []string              `json:"missingPrefixes" yaml:"missingPrefixes"`
	MetadataDigest  string                `json:"metadataDigest" yaml:"metadataDigest"`
}

// ObserveSelectedWorking reads only exact regular-file paths from the current
// working tree. Missing paths are returned explicitly instead of treated as
// empty files. It does not change source.Load or repository-wide snapshot
// digest semantics.
func ObserveSelectedWorking(root string, paths []string) (*SelectedWorkingSnapshot, error) {
	if len(paths) > DefaultMaxFiles {
		return nil, errors.New("selected working snapshot exceeds file-count limit")
	}
	clean := append([]string(nil), paths...)
	if err := validateSelectedPaths(clean); err != nil {
		return nil, fmt.Errorf("validate selected working paths: %w", err)
	}
	sort.Strings(clean)
	identity, initialStats, err := identifyGit(root, selectiveGitOutput)
	if err != nil {
		return nil, err
	}
	rootFS, err := os.OpenRoot(identity.Root)
	if err != nil {
		return nil, fmt.Errorf("open selected working tree root: %w", err)
	}
	defer rootFS.Close()

	result := &SelectedWorkingSnapshot{
		Identity:  identity,
		Snapshot:  &snapshot.Snapshot{Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}},
		Requested: append([]string(nil), clean...),
	}
	var total int64
	for _, repoPath := range clean {
		data, mode, present, err := readScopedWorkingFile(rootFS, repoPath)
		if err != nil {
			return nil, err
		}
		if !present {
			result.MissingPaths = append(result.MissingPaths, repoPath)
			continue
		}
		if int64(len(data)) > DefaultMaxFileBytes || total > DefaultMaxTotalBytes-int64(len(data)) {
			return nil, fmt.Errorf("selected working snapshot exceeds byte limit at %q", repoPath)
		}
		total += int64(len(data))
		result.Snapshot.Files[repoPath] = data
		result.Snapshot.Modes[repoPath] = mode
	}
	if err := confirmGitIdentity(identity, initialStats, selectiveGitOutput); err != nil {
		return nil, err
	}
	return result, nil
}

// InventoryWorkingRoots enumerates metadata beneath exact working-tree paths.
// It opens directories for traversal and stats files, but never opens or reads
// file contents. Overlapping roots are rejected to keep the inventory scope
// unambiguous.
func InventoryWorkingRoots(root string, exactPrefixes []string) (*WorkingRootInventory, error) {
	prefixes, err := normalizeInventoryPrefixes(exactPrefixes)
	if err != nil {
		return nil, err
	}
	identity, initialStats, err := identifyGit(root, selectiveGitOutput)
	if err != nil {
		return nil, err
	}
	rootFS, err := os.OpenRoot(identity.Root)
	if err != nil {
		return nil, fmt.Errorf("open working inventory root: %w", err)
	}
	defer rootFS.Close()

	result := &WorkingRootInventory{Identity: identity, Prefixes: prefixes, Entries: []WorkingFileMetadata{}, MissingPrefixes: []string{}}
	var total int64
	for _, prefix := range prefixes {
		info, present, err := scopedLstat(rootFS, prefix)
		if err != nil {
			return nil, err
		}
		if !present {
			result.MissingPrefixes = append(result.MissingPrefixes, prefix)
			continue
		}
		if isSymlink(info) {
			return nil, fmt.Errorf("refusing to inventory symlink or reparse point %q", prefix)
		}
		if info.Mode().IsRegular() {
			entry := metadataFor(prefix, info)
			result.Entries = append(result.Entries, entry)
			total += entry.Size
			if err := checkInventoryLimits(result.Entries, total, prefix); err != nil {
				return nil, err
			}
			continue
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("unsupported non-regular inventory root %q", prefix)
		}
		directory, err := openScopedDirectory(rootFS, prefix, info)
		if err != nil {
			return nil, err
		}
		if err := walkScopedMetadata(directory, prefix, &result.Entries, &total); err != nil {
			_ = directory.Close()
			return nil, err
		}
		if err := directory.Close(); err != nil {
			return nil, fmt.Errorf("close scoped inventory root %q: %w", prefix, err)
		}
	}
	paths := make([]string, len(result.Entries))
	for i, entry := range result.Entries {
		paths[i] = entry.Path
	}
	if err := validatePortablePaths(paths); err != nil {
		return nil, err
	}
	sort.Slice(result.Entries, func(i, j int) bool { return result.Entries[i].Path < result.Entries[j].Path })
	if err := confirmGitIdentity(identity, initialStats, selectiveGitOutput); err != nil {
		return nil, err
	}
	result.MetadataDigest = workingMetadataDigest(result.Prefixes, result.Entries, result.MissingPrefixes)
	return result, nil
}

func readScopedWorkingFile(root *os.Root, repoPath string) ([]byte, string, bool, error) {
	parent, name := path.Split(repoPath)
	parent = strings.TrimSuffix(parent, "/")
	parentRoot := root
	ownedParent := false
	if parent != "" {
		info, present, err := scopedLstat(root, parent)
		if err != nil {
			return nil, "", false, err
		}
		if !present {
			return nil, "", false, nil
		}
		if isSymlink(info) || !info.IsDir() {
			return nil, "", false, fmt.Errorf("selected working path parent %q is not a real directory", parent)
		}
		parentRoot, err = openScopedDirectory(root, parent, info)
		if err != nil {
			return nil, "", false, err
		}
		ownedParent = true
	}
	if ownedParent {
		defer parentRoot.Close()
	}
	info, err := parentRoot.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, fmt.Errorf("stat selected working file %q: %w", repoPath, err)
	}
	if isSymlink(info) {
		return nil, "", false, fmt.Errorf("refusing to read symlink or reparse point %q", repoPath)
	}
	if !info.Mode().IsRegular() {
		return nil, "", false, fmt.Errorf("unsupported non-regular selected working file %q", repoPath)
	}
	if info.Size() > DefaultMaxFileBytes {
		return nil, "", false, fmt.Errorf("source file %q exceeds per-file limit", repoPath)
	}
	file, err := parentRoot.Open(name)
	if err != nil {
		return nil, "", false, fmt.Errorf("open selected working file %q: %w", repoPath, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, "", false, fmt.Errorf("selected working file %q changed during acquisition", repoPath)
	}
	data, err := io.ReadAll(io.LimitReader(file, DefaultMaxFileBytes+1))
	if err != nil {
		return nil, "", false, fmt.Errorf("read selected working file %q: %w", repoPath, err)
	}
	if int64(len(data)) > DefaultMaxFileBytes {
		return nil, "", false, fmt.Errorf("source file %q exceeds per-file limit", repoPath)
	}
	after, err := file.Stat()
	pathAfter, pathErr := parentRoot.Lstat(name)
	if err != nil || pathErr != nil || isSymlink(pathAfter) || !os.SameFile(opened, after) || !os.SameFile(after, pathAfter) || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) || int64(len(data)) != after.Size() {
		return nil, "", false, fmt.Errorf("selected working file %q changed during acquisition", repoPath)
	}
	mode := snapshot.RegularMode
	if info.Mode().Perm()&0111 != 0 {
		mode = snapshot.ExecutableMode
	}
	return data, mode, true, nil
}

func normalizeInventoryPrefixes(input []string) ([]string, error) {
	if len(input) == 0 || len(input) > DefaultMaxFiles {
		return nil, errors.New("working inventory requires a bounded nonempty prefix list")
	}
	prefixes := append([]string(nil), input...)
	for i, prefix := range prefixes {
		prefix = strings.TrimSuffix(prefix, "/")
		if err := validateRepoPath(prefix); err != nil {
			return nil, fmt.Errorf("invalid inventory prefix %q: %w", input[i], err)
		}
		for _, component := range strings.Split(prefix, "/") {
			if component == ".git" || strings.ContainsAny(component, "*?[]") {
				return nil, fmt.Errorf("inventory prefix %q contains a reserved or non-exact component", prefix)
			}
		}
		prefixes[i] = prefix
	}
	sort.Strings(prefixes)
	if err := validatePortablePaths(prefixes); err != nil {
		return nil, err
	}
	for i := range prefixes {
		for j := i + 1; j < len(prefixes); j++ {
			if scopedPrefixesOverlap(prefixes[i], prefixes[j]) {
				return nil, fmt.Errorf("overlapping working inventory prefixes %q and %q", prefixes[i], prefixes[j])
			}
		}
	}
	return prefixes, nil
}

func scopedPrefixesOverlap(left, right string) bool {
	leftParts, rightParts := strings.Split(left, "/"), strings.Split(right, "/")
	if len(leftParts) > len(rightParts) {
		leftParts, rightParts = rightParts, leftParts
	}
	for i := range leftParts {
		if !strings.EqualFold(leftParts[i], rightParts[i]) {
			return false
		}
	}
	return true
}

func scopedLstat(root *os.Root, repoPath string) (os.FileInfo, bool, error) {
	parts := strings.Split(repoPath, "/")
	for i := range parts {
		current := strings.Join(parts[:i+1], "/")
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, fmt.Errorf("stat scoped working path %q: %w", current, err)
		}
		if isSymlink(info) {
			return nil, false, fmt.Errorf("refusing to traverse symlink or reparse point %q", current)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, false, fmt.Errorf("scoped working path parent %q is not a directory", current)
		}
		if i == len(parts)-1 {
			return info, true, nil
		}
	}
	return nil, false, errors.New("empty scoped working path")
}

func openScopedDirectory(parent *os.Root, repoPath string, expected os.FileInfo) (*os.Root, error) {
	current := parent
	owned := false
	for _, component := range strings.Split(repoPath, "/") {
		info, err := current.Lstat(component)
		if err != nil || isSymlink(info) || !info.IsDir() {
			if owned {
				_ = current.Close()
			}
			return nil, fmt.Errorf("scoped directory %q changed or contains a symlink during acquisition", repoPath)
		}
		next, err := current.OpenRoot(filepath.FromSlash(component))
		if err != nil {
			if owned {
				_ = current.Close()
			}
			return nil, fmt.Errorf("open scoped directory %q: %w", repoPath, err)
		}
		opened, statErr := next.Stat(".")
		if statErr != nil || !opened.IsDir() || !os.SameFile(info, opened) {
			_ = next.Close()
			if owned {
				_ = current.Close()
			}
			return nil, fmt.Errorf("scoped directory %q changed during acquisition", repoPath)
		}
		if owned {
			_ = current.Close()
		}
		current, owned = next, true
	}
	opened, err := current.Stat(".")
	if err != nil || !os.SameFile(expected, opened) {
		if owned {
			_ = current.Close()
		}
		return nil, fmt.Errorf("scoped directory %q changed during acquisition", repoPath)
	}
	return current, nil
}

func walkScopedMetadata(directory *os.Root, globalPrefix string, entries *[]WorkingFileMetadata, total *int64) error {
	listing, err := directory.Open(".")
	if err != nil {
		return fmt.Errorf("read scoped directory %q: %w", globalPrefix, err)
	}
	children, err := listing.ReadDir(-1)
	closeErr := listing.Close()
	if err != nil {
		return fmt.Errorf("read scoped directory %q: %w", globalPrefix, err)
	}
	if closeErr != nil {
		return fmt.Errorf("close scoped directory %q: %w", globalPrefix, closeErr)
	}
	sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
	for _, child := range children {
		name := child.Name()
		repoPath := globalPrefix + "/" + name
		if err := validateRepoPath(repoPath); err != nil {
			return err
		}
		if name == ".git" {
			return fmt.Errorf("working inventory path %q enters Git metadata", repoPath)
		}
		info, err := directory.Lstat(name)
		if err != nil {
			return fmt.Errorf("stat working inventory path %q: %w", repoPath, err)
		}
		if isSymlink(info) {
			return fmt.Errorf("refusing to traverse symlink or reparse point %q", repoPath)
		}
		if info.IsDir() {
			nested, err := openScopedDirectory(directory, name, info)
			if err != nil {
				return err
			}
			walkErr := walkScopedMetadata(nested, repoPath, entries, total)
			closeErr := nested.Close()
			if walkErr != nil {
				return walkErr
			}
			if closeErr != nil {
				return fmt.Errorf("close scoped directory %q: %w", repoPath, closeErr)
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported non-regular inventory file %q", repoPath)
		}
		entry := metadataFor(repoPath, info)
		*entries = append(*entries, entry)
		*total += entry.Size
		if err := checkInventoryLimits(*entries, *total, repoPath); err != nil {
			return err
		}
	}
	return nil
}

func metadataFor(repoPath string, info os.FileInfo) WorkingFileMetadata {
	mode := snapshot.RegularMode
	if info.Mode().Perm()&0111 != 0 {
		mode = snapshot.ExecutableMode
	}
	return WorkingFileMetadata{Path: repoPath, Mode: mode, Size: info.Size()}
}

func checkInventoryLimits(entries []WorkingFileMetadata, total int64, repoPath string) error {
	if len(entries) > DefaultMaxFiles {
		return errors.New("working inventory exceeds file-count limit")
	}
	if total > DefaultMaxTotalBytes {
		return fmt.Errorf("working inventory exceeds byte-size limit at %q", repoPath)
	}
	return nil
}

func workingMetadataDigest(prefixes []string, entries []WorkingFileMetadata, missing []string) string {
	h := sha256.New()
	write := func(value string) {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = h.Write(size[:])
		_, _ = io.WriteString(h, value)
	}
	for _, prefix := range prefixes {
		write("prefix")
		write(prefix)
	}
	for _, entry := range entries {
		write("entry")
		write(entry.Path)
		write(entry.Mode)
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(entry.Size))
		_, _ = h.Write(size[:])
	}
	for _, prefix := range missing {
		write("missing")
		write(prefix)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
