package host

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/adoption/capture"
)

const adoptionRecordLimit = 8 << 20

type adoptionWriteOps struct {
	before func(operation, path string) error
}

func writeAdoptionWorkspace(result *AdoptionPreparation, parent, destination string, parentInfo os.FileInfo, prepared *preparedEvidence, ops adoptionWriteOps) error {
	fail := func(err error) error {
		result.Status = "failed"
		if len(result.CreatedFiles) != 0 || len(result.CreatedDirectories) != 0 {
			result.Recovery = "Preparation is partial. Inspect the listed workspace paths and handoff state; no files or directories were removed. Resolve the partial workspace deliberately, then create a new absent destination and rerun with a fresh preview."
		}
		return err
	}
	manifest, err := capture.Encode(prepared.handoff)
	if err != nil {
		return fail(fmt.Errorf("encode adoption handoff: %w", err))
	}
	if len(manifest) > adoptionRecordLimit {
		return fail(errors.New("encoded adoption handoff exceeds 8 MiB"))
	}
	var workspaceInfo os.FileInfo
	validate := func(relative string) error {
		if err := revalidateAdoptionWrite(parent, destination, parentInfo, prepared); err != nil {
			return err
		}
		if workspaceInfo == nil {
			if _, err := os.Lstat(destination); err == nil {
				return errors.New("adoption destination appeared before exclusive creation")
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("inspect adoption destination before create: %w", err)
			}
		} else {
			current, err := os.Lstat(destination)
			if err != nil || !current.IsDir() || isReparsePoint(current) || !os.SameFile(workspaceInfo, current) {
				return errors.New("adoption workspace identity changed during preparation")
			}
		}
		if ops.before != nil {
			if err := ops.before("write", relative); err != nil {
				return err
			}
		}
		return nil
	}
	if err := validate("."); err != nil {
		return fail(fmt.Errorf("revalidate before workspace create: %w", err))
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		return fail(fmt.Errorf("create adoption workspace exclusively: %w", err))
	}
	result.CreatedDirectories = append(result.CreatedDirectories, ".")
	workspaceInfo, err = os.Lstat(destination)
	if err != nil || !workspaceInfo.IsDir() || isReparsePoint(workspaceInfo) {
		return fail(errors.New("adoption workspace became unsafe after creation"))
	}

	directories := map[string]bool{"evidence": true}
	for _, repo := range prepared.handoff.Repositories {
		root := filepath.ToSlash(filepath.Join("evidence", repo.ID))
		directories[root] = true
		for _, file := range repo.Files {
			for current := filepath.ToSlash(filepath.Dir(filepath.FromSlash(file.Path))); current != "."; current = filepath.ToSlash(filepath.Dir(filepath.FromSlash(current))) {
				directories[filepath.ToSlash(filepath.Join(root, current))] = true
			}
		}
	}
	orderedDirectories := make([]string, 0, len(directories))
	for directory := range directories {
		orderedDirectories = append(orderedDirectories, directory)
	}
	sort.Slice(orderedDirectories, func(i, j int) bool {
		depthI, depthJ := strings.Count(orderedDirectories[i], "/"), strings.Count(orderedDirectories[j], "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		return orderedDirectories[i] < orderedDirectories[j]
	})
	for _, directory := range orderedDirectories {
		if err := validate(directory); err != nil {
			return fail(fmt.Errorf("revalidate before directory create %s: %w", directory, err))
		}
		path, err := safeDestination(destination, directory)
		if err != nil {
			return fail(fmt.Errorf("unsafe adoption directory %s: %w", directory, err))
		}
		if err := os.Mkdir(path, 0700); err != nil {
			return fail(fmt.Errorf("create adoption directory exclusively %s: %w", directory, err))
		}
		result.CreatedDirectories = append(result.CreatedDirectories, filepath.ToSlash(directory))
	}

	keys := make([]string, 0, len(prepared.blobs))
	for key := range prepared.blobs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		repository, relative, ok := strings.Cut(key, "/")
		if !ok || !capture.ValidID(repository) {
			return fail(fmt.Errorf("invalid internal evidence key %q", key))
		}
		workspacePath := filepath.ToSlash(filepath.Join("evidence", repository, filepath.FromSlash(relative)))
		if err := writeAdoptionFile(result, destination, workspacePath, prepared.blobs[key], validate); err != nil {
			return fail(err)
		}
	}
	// The handoff is written last, so its presence means all evidence writes
	// completed. A failure still leaves a reported, intentionally partial tree.
	if err := writeAdoptionFile(result, destination, "handoff.yaml", manifest, validate); err != nil {
		return fail(err)
	}
	if err := revalidateAdoptionWrite(parent, destination, parentInfo, prepared); err != nil {
		return fail(fmt.Errorf("final adoption workspace validation: %w", err))
	}
	return nil
}

func writeAdoptionFile(result *AdoptionPreparation, root, relative string, data []byte, validate func(string) error) error {
	if err := validate(relative); err != nil {
		return fmt.Errorf("revalidate before file create %s: %w", relative, err)
	}
	destination, err := safeDestination(root, relative)
	if err != nil {
		return fmt.Errorf("unsafe adoption file %s: %w", relative, err)
	}
	f, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create adoption file exclusively %s: %w", relative, err)
	}
	result.CreatedFiles = append(result.CreatedFiles, filepath.ToSlash(relative))
	if err := validate(relative); err != nil {
		_ = f.Close()
		return fmt.Errorf("revalidate before file write %s: %w", relative, err)
	}
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return fmt.Errorf("write adoption file %s: %w", relative, errors.Join(writeErr, syncErr, closeErr))
	}
	return nil
}

func revalidateAdoptionWrite(parent, destination string, parentInfo os.FileInfo, prepared *preparedEvidence) error {
	parentNow, err := os.Lstat(parent)
	if err != nil || !parentNow.IsDir() || isReparsePoint(parentNow) || !os.SameFile(parentInfo, parentNow) {
		return errors.New("adoption destination parent changed during preparation")
	}
	if err := rejectReparseAncestors(parent); err != nil {
		return fmt.Errorf("unsafe destination parent: %w", err)
	}
	for _, id := range sortedRepoIDs(prepared.ids) {
		before := prepared.ids[id]
		current, err := source.IdentifyGit(before.Root)
		if err != nil || !sameSourceIdentity(before, current) {
			return fmt.Errorf("source repository identity changed during preparation: %s", id)
		}
		stats, err := statSourceIdentity(current)
		if err != nil {
			return fmt.Errorf("source repository paths changed during preparation: %s", id)
		}
		original := prepared.stats[id]
		if !os.SameFile(original.root, stats.root) || !os.SameFile(original.git, stats.git) || !os.SameFile(original.common, stats.common) {
			return fmt.Errorf("source repository directory identity changed during preparation: %s", id)
		}
	}
	if _, err := os.Lstat(destination); err == nil {
		// Once created, the workspace itself is expected to exist; the caller
		// confirms its identity through path traversal checks. Before creation,
		// existence is rejected by the exclusive mkdir.
		if err := rejectReparseAncestors(destination); err != nil {
			return fmt.Errorf("unsafe adoption workspace: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect adoption workspace: %w", err)
	}
	return nil
}

// ReadAdoptionWorkspace validates the handoff before opening only its listed
// evidence. It does not query Git, load a Project, or discover extra files.
func ReadAdoptionWorkspace(directory string) ([]byte, map[string][]byte, error) {
	root, err := realDirectory(directory)
	if err != nil {
		return nil, nil, err
	}
	manifestPath, err := safeDestination(root, "handoff.yaml")
	if err != nil {
		return nil, nil, err
	}
	manifestBytes, err := readBoundedRegular(manifestPath, adoptionRecordLimit, true)
	if err != nil {
		return nil, nil, fmt.Errorf("read adoption handoff: %w", err)
	}
	var handoff capture.Handoff
	if err := capture.Decode(manifestBytes, &handoff); err != nil {
		return nil, nil, fmt.Errorf("decode adoption handoff: %w", err)
	}
	if err := capture.ValidateManifest(handoff); err != nil {
		return nil, nil, fmt.Errorf("validate adoption handoff manifest: %w", err)
	}
	expected := map[string]bool{"handoff.yaml": true}
	expectedDirectories := map[string]bool{"evidence": true}
	for _, repo := range handoff.Repositories {
		expectedDirectories[filepath.ToSlash(filepath.Join("evidence", repo.ID))] = true
		for _, file := range repo.Files {
			relative := filepath.ToSlash(filepath.Join("evidence", repo.ID, filepath.FromSlash(file.Path)))
			expected[relative] = true
			for current := filepath.ToSlash(filepath.Dir(filepath.FromSlash(relative))); current != "."; current = filepath.ToSlash(filepath.Dir(filepath.FromSlash(current))) {
				expectedDirectories[current] = true
			}
		}
	}
	if err := validateWorkspaceTree(root, expected, expectedDirectories); err != nil {
		return nil, nil, err
	}
	blobs := make(map[string][]byte, len(expected)-1)
	var total int64
	for _, repo := range handoff.Repositories {
		for _, file := range repo.Files {
			relative := filepath.ToSlash(filepath.Join("evidence", repo.ID, filepath.FromSlash(file.Path)))
			path, err := safeDestination(root, relative)
			if err != nil {
				return nil, nil, err
			}
			remaining := source.DefaultMaxTotalBytes - total
			limit := int64(source.DefaultMaxFileBytes)
			if remaining < limit {
				limit = remaining
			}
			data, err := readBoundedRegular(path, limit, true)
			if err != nil {
				return nil, nil, fmt.Errorf("read selected evidence %s/%s: %w", repo.ID, file.Path, err)
			}
			total += int64(len(data))
			if total > source.DefaultMaxTotalBytes {
				return nil, nil, errors.New("adoption workspace evidence exceeds total byte limit")
			}
			blobs[capture.BlobKey(repo.ID, file.Path)] = data
		}
	}
	if err := capture.ValidateHandoff(handoff, blobs); err != nil {
		return nil, nil, fmt.Errorf("validate adoption workspace evidence: %w", err)
	}
	return manifestBytes, blobs, nil
}

// ReadAdoptionRecord reads one explicitly named small UTF-8 record without
// following symlink/reparse ancestors. It is for owner-supplied scope, queue,
// or decision files, not evidence selection.
func ReadAdoptionRecord(path string) ([]byte, error) {
	return readExplicitRecord(path, adoptionRecordLimit)
}

// ReadAdoptionCandidate reads one exact queue-relative candidate path. It
// cannot escape the queue directory or follow a link/reparse point.
func ReadAdoptionCandidate(queueDirectory, exactRelativePath string) ([]byte, error) {
	if err := capture.ExactPath(exactRelativePath); err != nil {
		return nil, err
	}
	root, err := realDirectory(queueDirectory)
	if err != nil {
		return nil, err
	}
	path, err := safeDestination(root, exactRelativePath)
	if err != nil {
		return nil, err
	}
	if err := rejectCaseAliases(filepath.Dir(path)); err != nil {
		return nil, err
	}
	if err := checkDiskComponentAlias(filepath.Dir(path), filepath.Base(path), exactRelativePath); err != nil {
		return nil, err
	}
	return readBoundedRegular(path, adoptionRecordLimit, true)
}

func readExplicitRecord(path string, max int64) ([]byte, error) {
	if path == "" {
		return nil, errors.New("record path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	abs = filepath.Clean(abs)
	if err := rejectReparseAncestors(filepath.Dir(abs)); err != nil {
		return nil, fmt.Errorf("unsafe record path: %w", err)
	}
	canonical, err := canonicalUserPath(abs)
	if err != nil || !samePathSpelling(abs, canonical) {
		return nil, fmt.Errorf("record path must use canonical spelling: %s", abs)
	}
	if err := rejectCaseAliases(filepath.Dir(abs)); err != nil {
		return nil, err
	}
	if err := checkDiskComponentAlias(filepath.Dir(abs), filepath.Base(abs), filepath.Base(abs)); err != nil {
		return nil, err
	}
	return readBoundedRegular(abs, max, true)
}

func realDirectory(directory string) (string, error) {
	if directory == "" {
		return "", errors.New("directory is required")
	}
	abs, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if err := rejectReparseAncestors(abs); err != nil {
		return "", fmt.Errorf("unsafe directory: %w", err)
	}
	info, err := os.Lstat(abs)
	if err != nil || !info.IsDir() || isReparsePoint(info) {
		return "", fmt.Errorf("expected a real directory: %s", abs)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalUserPath(resolved)
	if err != nil || !samePathSpelling(abs, canonical) {
		return "", fmt.Errorf("directory must use canonical path spelling: %s", abs)
	}
	if err := rejectCaseAliases(abs); err != nil {
		return "", err
	}
	return abs, nil
}

func readBoundedRegular(path string, limit int64, requireUTF8 bool) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || isReparsePoint(before) || before.Size() < 0 || before.Size() > limit {
		return nil, fmt.Errorf("expected regular file no larger than %d bytes: %s", limit, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("record changed while opening: %s", path)
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit || int64(len(data)) != opened.Size() {
		return nil, fmt.Errorf("file changed or exceeds %d-byte limit: %s", limit, path)
	}
	if requireUTF8 && !utf8.Valid(data) {
		return nil, fmt.Errorf("file is not UTF-8: %s", path)
	}
	return data, nil
}

func validateWorkspaceTree(root string, expected, expectedDirectories map[string]bool) error {
	actual := map[string]bool{}
	var walk func(string, string) error
	walk = func(directory, relative string) error {
		entries, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, entry := range entries {
			name := entry.Name()
			for previous := range seen {
				if strings.EqualFold(previous, name) && previous != name {
					return fmt.Errorf("case-colliding workspace entries %q and %q", previous, name)
				}
			}
			seen[name] = true
			rel := filepath.ToSlash(filepath.Join(relative, name))
			path := filepath.Join(directory, name)
			info, err := os.Lstat(path)
			if err != nil || isReparsePoint(info) {
				return fmt.Errorf("workspace contains unsafe entry %s", rel)
			}
			if info.IsDir() {
				if !expectedDirectories[rel] {
					return fmt.Errorf("unexpected workspace directory %s", rel)
				}
				if err := walk(path, rel); err != nil {
					return err
				}
				continue
			}
			if !info.Mode().IsRegular() || !expected[rel] {
				return fmt.Errorf("unexpected workspace file %s", rel)
			}
			actual[rel] = true
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return err
	}
	for path := range expected {
		if !actual[path] {
			return fmt.Errorf("adoption workspace is missing %s", path)
		}
	}
	return nil
}
