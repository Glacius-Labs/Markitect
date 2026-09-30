package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/render"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

// WriteMigration is an explicit one-time ownership transfer. Existing Markdown
// is replaced only after confirming the exact inventoried input set, while all
// YAML paths must be new. Ordinary render never overwrites unmanaged content.
func WriteMigration(root string, snap *source.Snapshot, canonical map[string][]byte) ([]string, error) {
	if !snap.Provisional {
		return nil, fmt.Errorf("migration writes require the isolated working tree")
	}
	if _, exists := snap.Files["markitect.yaml"]; exists {
		return nil, fmt.Errorf("migration is a one-time import; markitect.yaml already exists")
	}
	unlock, err := lockWriter(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	branch, err := source.GitOutput(absolute, "branch", "--show-current")
	name := strings.TrimSpace(string(branch))
	if err != nil || name == "" || name == "master" || name == "main" {
		return nil, fmt.Errorf("migration requires an isolated non-protected Git branch")
	}
	proposed := &source.Snapshot{Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}}
	for name, data := range snap.Files {
		proposed.Files[name] = data
		proposed.Modes[name] = snap.Modes[name]
	}
	for name, data := range canonical {
		if _, exists := snap.Files[name]; exists {
			return nil, fmt.Errorf("migration would overwrite existing canonical path %s", name)
		}
		proposed.Files[name] = data
		proposed.Modes[name] = "100644"
	}
	project, err := Parse(proposed)
	if err != nil {
		return nil, err
	}
	if len(project.Diagnostics) > 0 {
		return nil, fmt.Errorf("migration graph has unresolved diagnostics: %+v", project.Diagnostics)
	}
	outputs, err := render.Generate(project.Graph)
	if err != nil {
		return nil, err
	}
	approved := map[string]bool{}
	for _, legacy := range LegacyInventory(snap) {
		if _, ok := canonical[strings.TrimSuffix(legacy.Path, ".md")+".yaml"]; ok {
			approved[legacy.Path] = true
		}
	}
	for name := range outputs {
		if _, exists := snap.Files[name]; exists && !approved[name] {
			return nil, fmt.Errorf("migration refuses unmanaged output outside the inventoried legacy definitions: %s", name)
		}
	}
	for name, data := range canonical {
		if _, ok := outputs[name]; ok {
			return nil, fmt.Errorf("migration output collision %s", name)
		}
		outputs[name] = data
	}
	for name := range outputs {
		if _, err = safeDestination(root, name); err != nil {
			return nil, err
		}
	}
	current, err := source.Load(root, "")
	if err != nil {
		return nil, err
	}
	if current.Digest() != snap.Digest() {
		return nil, fmt.Errorf("source changed since migration inventory; reload")
	}
	names := sortedFiles(outputs)
	for _, name := range names {
		dest, err := safeDestination(root, name)
		if err != nil {
			return nil, err
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return nil, err
		}
		old, existed := snap.Files[name]
		now, readErr := os.ReadFile(dest)
		if existed && (readErr != nil || !bytes.Equal(old, now)) {
			return nil, fmt.Errorf("migration target changed: %s", name)
		}
		if !existed && !os.IsNotExist(readErr) {
			return nil, fmt.Errorf("migration target appeared: %s", name)
		}
		if err = atomicWrite(dest, outputs[name]); err != nil {
			return nil, err
		}
	}
	after, err := source.Load(root, "")
	if err != nil {
		return nil, err
	}
	for name, data := range outputs {
		current.Files[name] = data
		current.Modes[name] = after.Modes[name]
	}
	if after.Digest() != current.Digest() {
		return nil, fmt.Errorf("source changed during migration; branch is provisional and must be rechecked")
	}
	// A failure or interruption leaves a visible, reviewable branch diff. It
	// cannot produce a successful check until the complete plan matches.
	return names, nil
}
