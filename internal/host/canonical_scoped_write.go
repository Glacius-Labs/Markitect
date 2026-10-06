package host

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// canonicalScopedWriteCapture binds exact input bytes and separately declared
// target-root metadata. It is an operational observation, never a full snapshot.
type canonicalScopedWriteCapture struct {
	Revision       string
	CanonicalPaths []string
	Observed       *snapshot.Snapshot
	Inventory      *source.WorkingRootInventory
}

// writeCanonicalScopedOutputs is called only after the Host validates an exact
// aggregate reviewed candidate and explicit write intent. It does not authorize
// generation, adoption, deletion or overwrite of unknown ownership.
func writeCanonicalScopedOutputs(root string, captured canonicalScopedWriteCapture, outputs map[string][]byte, outputModes map[string]string) ([]string, error) {
	if captured.Observed == nil || !captured.Observed.Provisional || captured.Inventory == nil || !canonicalRevisionPattern.MatchString(captured.Revision) {
		return nil, errors.New("scoped apply requires fixed source and explicit provisional byte/inventory capture")
	}
	if err := validateProjectionWriteModes(outputs, outputModes); err != nil {
		return nil, err
	}
	branch, err := writeBranchName(root)
	if err != nil {
		return nil, err
	}
	protected := map[string]bool{}
	for _, name := range captured.CanonicalPaths {
		if _, found := captured.Observed.Files[name]; !found {
			return nil, fmt.Errorf("canonical input was not captured: %s", name)
		}
		protected[name] = true
	}
	names := make([]string, 0, len(outputs))
	selected := map[string]bool{}
	for name := range captured.Observed.Files {
		selected[name] = true
	}
	for name := range outputs {
		if protected[name] {
			return nil, fmt.Errorf("candidate targets canonical input: %s", name)
		}
		allowed := false
		for _, prefix := range captured.Inventory.Prefixes {
			if stringsHasPathPrefix(name, prefix) {
				allowed = true
			}
		}
		if !allowed {
			return nil, fmt.Errorf("candidate is outside inventoried target roots: %s", name)
		}
		if _, err := safeDestination(root, name); err != nil {
			return nil, err
		}
		selected[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	paths := make([]string, 0, len(selected))
	for name := range selected {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	checkHead := func() error {
		if err := ensureWriteBranch(root, branch); err != nil {
			return err
		}
		head, err := source.GitOutput(root, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(head)) != captured.Revision {
			return errors.New("source HEAD changed since reviewed scoped plan")
		}
		return nil
	}
	if err := checkHead(); err != nil {
		return nil, err
	}
	fresh, err := source.ObserveSelectedWorking(root, paths)
	if err != nil {
		return nil, err
	}
	if !equalCanonicalValue(fresh.Identity, captured.Inventory.Identity) {
		return nil, errors.New("repository identity changed since reviewed scoped capture")
	}
	if fresh.Snapshot.Digest() != captured.Observed.Digest() {
		return nil, errors.New("selected source/target bytes or modes changed since reviewed plan")
	}
	inventory, err := source.InventoryWorkingRoots(root, captured.Inventory.Prefixes)
	if err != nil {
		return nil, err
	}
	if !equalCanonicalValue(inventory.Identity, captured.Inventory.Identity) {
		return nil, errors.New("inventory repository identity changed since reviewed scope")
	}
	if inventory.MetadataDigest != captured.Inventory.MetadataDigest {
		return nil, errors.New("declared target inventory changed since reviewed plan")
	}
	unlock, err := lockWriter(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	// Revalidate under the shared writer lock before the first mutation.
	if err := checkHead(); err != nil {
		return nil, err
	}
	fresh, err = source.ObserveSelectedWorking(root, paths)
	if err != nil {
		return nil, err
	}
	if !equalCanonicalValue(fresh.Identity, captured.Inventory.Identity) {
		return nil, errors.New("repository identity changed since reviewed scoped capture")
	}
	if fresh.Snapshot.Digest() != captured.Observed.Digest() {
		return nil, errors.New("selected inputs changed while acquiring write lock")
	}
	inventory, err = source.InventoryWorkingRoots(root, captured.Inventory.Prefixes)
	if err != nil {
		return nil, err
	}
	if !equalCanonicalValue(inventory.Identity, captured.Inventory.Identity) {
		return nil, errors.New("inventory repository identity changed since reviewed scope")
	}
	if inventory.MetadataDigest != captured.Inventory.MetadataDigest {
		return nil, errors.New("target inventory changed while acquiring write lock")
	}
	written := []string{}
	for _, name := range names {
		if err := checkHead(); err != nil {
			return written, err
		}
		destination, err := safeDestination(root, name)
		if err != nil {
			return written, err
		}
		old, exists := captured.Observed.Files[name]
		current, readErr := os.ReadFile(destination)
		if exists && (readErr != nil || !bytes.Equal(current, old)) {
			return written, fmt.Errorf("target changed during scoped apply: %s", name)
		}
		if !exists && !os.IsNotExist(readErr) {
			return written, fmt.Errorf("target appeared during scoped apply: %s", name)
		}
		mode := projectionOutputMode(outputModes, name)
		if exists && bytes.Equal(current, outputs[name]) && captured.Observed.Modes[name] == mode {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return written, err
		}
		if _, err := safeDestination(root, name); err != nil {
			return written, err
		}
		if err := atomicWrite(destination, outputs[name]); err != nil {
			return written, err
		}
		written = append(written, name)
		if mode == snapshot.ExecutableMode {
			if err := os.Chmod(destination, 0755); err != nil {
				return written, fmt.Errorf("set executable artifact mode for %s: %w", name, err)
			}
		}
	}
	final, err := source.ObserveSelectedWorking(root, paths)
	if err != nil {
		return written, err
	}
	if !equalCanonicalValue(final.Identity, captured.Inventory.Identity) {
		return written, errors.New("repository identity changed during scoped Apply")
	}
	for _, name := range names {
		if !bytes.Equal(final.Snapshot.Files[name], outputs[name]) || final.Snapshot.Modes[name] != projectionOutputMode(outputModes, name) {
			return written, fmt.Errorf("materialized target changed during apply: %s", name)
		}
		delete(final.Snapshot.Files, name)
		delete(final.Snapshot.Modes, name)
	}
	unchanged := &snapshot.Snapshot{Files: cloneByteMap(captured.Observed.Files), Modes: map[string]string{}}
	for name, mode := range captured.Observed.Modes {
		unchanged.Modes[name] = mode
	}
	for _, name := range names {
		delete(unchanged.Files, name)
		delete(unchanged.Modes, name)
	}
	if final.Snapshot.Digest() != unchanged.Digest() {
		return written, errors.New("non-target selected inputs changed during apply; outputs remain provisional")
	}
	finalInventory, err := source.InventoryWorkingRoots(root, captured.Inventory.Prefixes)
	if err != nil {
		return written, err
	}
	if !equalCanonicalValue(finalInventory.Identity, captured.Inventory.Identity) {
		return written, errors.New("inventory identity changed during scoped Apply")
	}
	if !sameNonTargetInventory(captured.Inventory, finalInventory, names) {
		return written, errors.New("non-target declared inventory changed during apply; outputs remain provisional")
	}
	if err := checkHead(); err != nil {
		return written, err
	}
	return written, nil
}

func sameNonTargetInventory(before, after *source.WorkingRootInventory, targets []string) bool {
	ignored := map[string]bool{}
	for _, name := range targets {
		ignored[name] = true
	}
	a, b := map[string]string{}, map[string]string{}
	for _, entry := range before.Entries {
		if !ignored[entry.Path] {
			a[entry.Path] = fmt.Sprintf("%s:%d", entry.Mode, entry.Size)
		}
	}
	for _, entry := range after.Entries {
		if !ignored[entry.Path] {
			b[entry.Path] = fmt.Sprintf("%s:%d", entry.Mode, entry.Size)
		}
	}
	if len(a) != len(b) {
		return false
	}
	for name, value := range a {
		if b[name] != value {
			return false
		}
	}
	missing := func(values []string) []string {
		out := []string{}
		for _, prefix := range values {
			expectedCreation := false
			for _, name := range targets {
				if stringsHasPathPrefix(name, prefix) {
					expectedCreation = true
				}
			}
			if !expectedCreation {
				out = append(out, prefix)
			}
		}
		sort.Strings(out)
		return out
	}
	return equalStringSets(missing(before.MissingPrefixes), missing(after.MissingPrefixes))
}
