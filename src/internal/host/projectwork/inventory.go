package projectwork

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// loadConfiguredInventory reads only regular paths enumerated beneath the
// manifest's explicit inventory roots, after applying its exclusions.
func loadConfiguredInventory(root, revision string, provisional bool, config Config) (*snapshot.Snapshot, error) {
	if config.CoverageMode == "full" {
		return loadFullCoverageSnapshot(root, revision, provisional, config)
	}
	selected := &snapshot.Snapshot{ID: revision, Provisional: provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	if len(config.InventoryRoots) == 0 {
		return selected, nil
	}
	tree := "fixed"
	if provisional {
		tree = "working"
	}
	// One repository identity check covers both the listing and the read.
	acquisition, err := source.BeginAcquisition(root)
	if err != nil {
		return nil, fmt.Errorf("list %s inventory roots: %w", tree, err)
	}
	inventory, err := readConfiguredInventory(acquisition, revision, provisional, config, selected)
	if err != nil {
		return nil, err
	}
	if err := acquisition.Confirm(); err != nil {
		return nil, fmt.Errorf("read %s inventory selection: %w", tree, err)
	}
	return inventory, nil
}

func readConfiguredInventory(acquisition *source.Acquisition, revision string, provisional bool, config Config, selected *snapshot.Snapshot) (*snapshot.Snapshot, error) {
	var paths []string
	if provisional {
		metadata, err := acquisition.InventoryWorkingRoots(config.InventoryRoots)
		if err != nil {
			return nil, fmt.Errorf("list working inventory roots: %w", err)
		}
		for _, entry := range metadata.Entries {
			if !excludedPath(config, entry.Path) {
				paths = append(paths, entry.Path)
			}
		}
		if len(paths) == 0 {
			return selected, nil
		}
		observed, err := acquisition.ObserveSelectedWorking(paths)
		if err != nil {
			return nil, fmt.Errorf("read working inventory selection: %w", err)
		}
		if len(observed.MissingPaths) != 0 {
			return nil, fmt.Errorf("selected inventory file %q disappeared while acquiring it", observed.MissingPaths[0])
		}
		return observed.Snapshot, nil
	}
	metadata, err := acquisition.InventoryRevisionRoots(revision, config.InventoryRoots)
	if err != nil {
		return nil, fmt.Errorf("list fixed inventory roots: %w", err)
	}
	for _, entry := range metadata.Entries {
		if !excludedPath(config, entry.Path) {
			paths = append(paths, entry.Path)
		}
	}
	if len(paths) == 0 {
		return selected, nil
	}
	fixed, err := acquisition.LoadSelected(revision, paths)
	if err != nil {
		return nil, fmt.Errorf("read fixed inventory selection: %w", err)
	}
	return fixed.Snapshot, nil
}
