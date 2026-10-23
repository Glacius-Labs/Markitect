package source

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// RevisionRootInventory lists metadata under explicitly selected paths at a
// fixed commit. It does not read blob contents or authorize their acquisition.
type RevisionRootInventory struct {
	Identity        GitIdentity           `json:"identity"`
	Revision        string                `json:"revision"`
	Prefixes        []string              `json:"prefixes"`
	Entries         []WorkingFileMetadata `json:"entries"`
	MissingPrefixes []string              `json:"missingPrefixes"`
	MetadataDigest  string                `json:"metadataDigest"`
}

// InventoryRevisionRoots is the fixed-commit counterpart of InventoryWorkingRoots.
// Paths are literal exact files or directory prefixes, never Git expressions.
func InventoryRevisionRoots(root, fullCommit string, exactPrefixes []string) (*RevisionRootInventory, error) {
	return inventoryRevisionRoots(root, fullCommit, exactPrefixes, selectiveGitOutput)
}

// InventoryRevisionRoots is the package-level InventoryRevisionRoots bound to
// the acquisition's repository identity.
func (a *Acquisition) InventoryRevisionRoots(fullCommit string, exactPrefixes []string) (*RevisionRootInventory, error) {
	prefixes, err := normalizeInventoryPrefixes(exactPrefixes)
	if err != nil {
		return nil, err
	}
	return a.inventoryRevisionPrefixes(fullCommit, prefixes)
}

func inventoryRevisionRoots(root, fullCommit string, exactPrefixes []string, run gitOutputFunc) (*RevisionRootInventory, error) {
	prefixes, err := normalizeInventoryPrefixes(exactPrefixes)
	if err != nil {
		return nil, err
	}
	return acquireOnce(root, run, func(a *Acquisition) (*RevisionRootInventory, error) {
		return a.inventoryRevisionPrefixes(fullCommit, prefixes)
	})
}

func (a *Acquisition) inventoryRevisionPrefixes(fullCommit string, prefixes []string) (*RevisionRootInventory, error) {
	identity, run := a.identity, a.run
	wantLength := 40
	if identity.ObjectFormat == "sha256" {
		wantLength = 64
	}
	if len(fullCommit) != wantLength || fullCommit != strings.ToLower(fullCommit) {
		return nil, errors.New("revision inventory requires a full lowercase commit ID matching the object format")
	}
	if _, err := hex.DecodeString(fullCommit); err != nil {
		return nil, errors.New("revision inventory requires a hexadecimal commit ID")
	}
	resolved, err := a.verifyCommit(fullCommit)
	if err != nil {
		return nil, fmt.Errorf("verify inventory commit: %w", err)
	}
	if resolved != fullCommit {
		return nil, errors.New("inventory commit resolved to another ID")
	}
	result := &RevisionRootInventory{Identity: identity, Revision: fullCommit, Prefixes: prefixes, Entries: []WorkingFileMetadata{}, MissingPrefixes: []string{}}
	batches, err := selectedTreePathBatches(identity.Root, fullCommit, prefixes)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	found := map[string]bool{}
	var total int64
	for _, batch := range batches {
		args := []string{"--literal-pathspecs", "ls-tree", "-r", "-z", "--full-tree", "--long", fullCommit, "--"}
		out, err := run(identity.Root, append(args, batch...)...)
		if err != nil {
			return nil, fmt.Errorf("inspect inventory paths: %w", err)
		}
		entries, err := parseSelectedTreeEntries(out)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			allowed := false
			for _, prefix := range batch {
				if entry.path == prefix || strings.HasPrefix(entry.path, prefix+"/") {
					allowed = true
					found[prefix] = true
				}
			}
			if !allowed {
				return nil, fmt.Errorf("Git returned an out-of-scope inventory entry %q", entry.path)
			}
			if seen[entry.path] {
				return nil, fmt.Errorf("duplicate inventory entry %q", entry.path)
			}
			seen[entry.path] = true
			if err := validateSelectedPaths([]string{entry.path}); err != nil {
				return nil, err
			}
			if entry.kind != "blob" || (entry.mode != "100644" && entry.mode != "100755") {
				return nil, fmt.Errorf("inventory entry %q is not a regular file", entry.path)
			}
			result.Entries = append(result.Entries, WorkingFileMetadata{Path: entry.path, Mode: entry.mode, Size: entry.size})
			if entry.size > DefaultMaxTotalBytes-total {
				return nil, errors.New("revision inventory exceeds byte-size limit")
			}
			total += entry.size
			if err := checkInventoryLimits(result.Entries, total, entry.path); err != nil {
				return nil, err
			}
		}
	}
	paths := make([]string, 0, len(result.Entries))
	for _, entry := range result.Entries {
		paths = append(paths, entry.Path)
	}
	if err := validatePortablePaths(paths); err != nil {
		return nil, err
	}
	for _, prefix := range prefixes {
		if !found[prefix] {
			result.MissingPrefixes = append(result.MissingPrefixes, prefix)
		}
	}
	sort.Slice(result.Entries, func(i, j int) bool { return result.Entries[i].Path < result.Entries[j].Path })
	result.MetadataDigest = workingMetadataDigest(result.Prefixes, result.Entries, result.MissingPrefixes)
	return result, nil
}
