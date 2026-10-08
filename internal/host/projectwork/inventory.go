package projectwork

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// loadConfiguredInventory reads only regular paths enumerated beneath the
// manifest's explicit inventory roots, after applying its exclusions.
func loadConfiguredInventory(root, revision string, provisional bool, config Config) (*snapshot.Snapshot, error) {
	selected := &snapshot.Snapshot{ID: revision, Provisional: provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	if len(config.InventoryRoots) == 0 {
		return selected, nil
	}
	var paths []string
	if provisional {
		metadata, err := source.InventoryWorkingRoots(root, config.InventoryRoots)
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
		observed, err := source.ObserveSelectedWorking(root, paths)
		if err != nil {
			return nil, fmt.Errorf("read working inventory selection: %w", err)
		}
		if len(observed.MissingPaths) != 0 {
			return nil, fmt.Errorf("selected inventory file %q disappeared while acquiring it", observed.MissingPaths[0])
		}
		return observed.Snapshot, nil
	}
	metadata, err := source.InventoryRevisionRoots(root, revision, config.InventoryRoots)
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
	fixed, err := source.LoadSelected(root, revision, paths)
	if err != nil {
		return nil, fmt.Errorf("read fixed inventory selection: %w", err)
	}
	return fixed.Snapshot, nil
}
