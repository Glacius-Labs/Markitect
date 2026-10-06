package host

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// SelectedInputScope makes the boundary of a sparse canonical source snapshot
// explicit. FullCommit identifies the immutable repository revision;
// Digest fingerprints only Paths and their bytes/modes.
type SelectedInputScope struct {
	Repository source.GitIdentity `json:"repository" yaml:"repository"`
	FullCommit string             `json:"fullCommit" yaml:"fullCommit"`
	Paths      []string           `json:"paths" yaml:"paths"`
	Digest     string             `json:"digest" yaml:"digest"`
}

// LoadSelectedCanonicalSource loads only the exact canonical source paths
// named by configPath and that config's Definitions and local Module files.
// Snapshot.ID remains the full Git commit ID. Snapshot.Digest fingerprints
// only the selected bytes; AcquisitionScope records that explicit boundary.
func LoadSelectedCanonicalSource(root, fullCommit, configPath string, requirePins bool) (*CanonicalSource, error) {
	return loadSelectedCanonicalSourceWith(root, fullCommit, configPath, requirePins, source.LoadSelected)
}

type selectedSnapshotLoader func(root, fullCommit string, paths []string) (*source.SelectedSnapshot, error)

func loadSelectedCanonicalSourceWith(root, fullCommit, configPath string, requirePins bool, load selectedSnapshotLoader) (*CanonicalSource, error) {
	cleanConfigPath, err := canonicalRepositoryPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("canonical --config: %w", err)
	}
	if fullCommit == "" {
		return nil, errors.New("selected canonical source requires a full immutable commit ID")
	}
	if load == nil {
		return nil, errors.New("selected canonical source loader is required")
	}
	configOnly, err := load(root, fullCommit, []string{cleanConfigPath})
	if err != nil {
		return nil, fmt.Errorf("load selected canonical config: %w", err)
	}
	if err := validateSelectedAcquisition(configOnly, fullCommit); err != nil {
		return nil, err
	}
	configBytes, ok := configOnly.Snapshot.Files[cleanConfigPath]
	if !ok {
		return nil, fmt.Errorf("canonical source config %q is not present in its selected snapshot", cleanConfigPath)
	}
	config, err := DecodeCanonicalSourceConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", cleanConfigPath, err)
	}
	paths := canonicalSelectedInputPaths(cleanConfigPath, config)
	selected, err := load(root, fullCommit, paths)
	if err != nil {
		return nil, fmt.Errorf("load selected canonical inputs: %w", err)
	}
	if err := validateSelectedAcquisition(selected, fullCommit); err != nil {
		return nil, err
	}
	if configOnly.Identity != selected.Identity {
		return nil, errors.New("Git repository identity changed between canonical config and input acquisition")
	}
	selectedConfig, ok := selected.Snapshot.Files[cleanConfigPath]
	if !ok || !bytes.Equal(configBytes, selectedConfig) {
		return nil, errors.New("canonical source config changed between selected input acquisitions")
	}
	loaded, err := loadCanonicalSourceSnapshot(selected.Snapshot, cleanConfigPath, requirePins)
	if err != nil {
		return nil, err
	}
	loaded.AcquisitionScope = &SelectedInputScope{
		Repository: selected.Identity,
		FullCommit: fullCommit,
		Paths:      append([]string(nil), paths...),
		Digest:     selected.Snapshot.Digest(),
	}
	return loaded, nil
}

func validateSelectedAcquisition(selected *source.SelectedSnapshot, fullCommit string) error {
	if selected == nil || selected.Snapshot == nil {
		return errors.New("selected canonical source loader returned no snapshot")
	}
	if selected.Snapshot.Provisional || selected.Snapshot.ID != fullCommit {
		return errors.New("selected canonical source does not match the full immutable commit ID")
	}
	if selected.Identity.Root == "" || selected.Identity.GitDir == "" || selected.Identity.CommonDir == "" || selected.Identity.Digest == "" {
		return errors.New("selected canonical source has incomplete Git repository identity")
	}
	return nil
}

func canonicalSelectedInputPaths(configPath string, config CanonicalSourceConfig) []string {
	seen := map[string]bool{configPath: true}
	paths := []string{configPath}
	add := func(path string) {
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	for _, definition := range config.Definitions {
		add(definition)
	}
	for _, module := range config.Modules {
		add(module.Manifest)
		for _, file := range module.Files {
			add(file)
		}
	}
	sort.Strings(paths)
	return paths
}

// SelectedInputSnapshotDigest returns the digest named by an explicit source
// acquisition scope. It rejects ordinary full snapshots to prevent callers
// from accidentally treating the new scoped path as legacy whole-source load.
func SelectedInputSnapshotDigest(selected *CanonicalSource) (string, error) {
	if selected == nil || selected.Snapshot == nil || selected.AcquisitionScope == nil {
		return "", errors.New("canonical source has no explicit selected-input scope")
	}
	if selected.AcquisitionScope.FullCommit != selected.Snapshot.ID || selected.AcquisitionScope.Digest != selected.Snapshot.Digest() {
		return "", errors.New("selected canonical source scope does not match its snapshot")
	}
	return selected.AcquisitionScope.Digest, nil
}
