package host

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"go.yaml.in/yaml/v3"
)

const CanonicalSourceAPIVersion = "markitect.canonical/v1alpha1"

const maxCanonicalSourceConfigBytes = 4 << 20

// CanonicalSourceConfig names every input to the new structural compiler.
// Paths are repository-relative and are resolved only in the selected source
// snapshot. It deliberately does not scan directories for Definitions or
// Module files.
type CanonicalSourceConfig struct {
	APIVersion         string                        `yaml:"apiVersion"`
	Kind               string                        `yaml:"kind"`
	Definitions        []string                      `yaml:"definitions"`
	Modules            []CanonicalModuleSource       `yaml:"modules"`
	ProjectionBindings []canonical.ProjectionBinding `yaml:"projectionBindings,omitempty"`
	Checks             []authoring.Check             `yaml:"checks,omitempty"`
}

// CanonicalModuleSource supplies one exact local Module manifest and every
// exact Schema file included in its package. Pin is required for compilation,
// but optional during the read-only digest-preview action.
type CanonicalModuleSource struct {
	Manifest string         `yaml:"manifest"`
	Files    []string       `yaml:"files"`
	Pin      *canonical.Pin `yaml:"pin,omitempty"`
}

type CanonicalModulePreview struct {
	Pin        canonical.Pin `json:"pin" yaml:"pin"`
	Type       string        `json:"type" yaml:"type"`
	Digest     string        `json:"digest" yaml:"digest"`
	Manifest   string        `json:"manifest" yaml:"manifest"`
	Files      []string      `json:"files" yaml:"files"`
	Schemas    []string      `json:"schemas" yaml:"schemas"`
	Projectors []string      `json:"projectors" yaml:"projectors"`
}

// CanonicalSource is a single fixed-source input set, before or after
// structural compilation. Snapshot is retained as evidence; callers must not
// read files outside it after this boundary.
type CanonicalSource struct {
	Snapshot *snapshot.Snapshot
	// AcquisitionScope is non-nil only when Snapshot contains the exact
	// canonical source inputs selected from a fixed Git revision. A sparse
	// snapshot must never be presented as a repository-wide snapshot.
	AcquisitionScope *SelectedInputScope
	ConfigPath       string
	Config           CanonicalSourceConfig
	Packages         []canonical.ModulePackage
	Pins             []canonical.Pin
	Previews         []CanonicalModulePreview
	Activation       canonical.Activation
	Model            core.Model
	Diagnostics      []core.Diagnostic
}

// LoadCanonicalSource reads a closed source config and its exact listed inputs
// from one normal Markitect source snapshot. When requirePins is true, every
// supplied Module must have an exact Name/Version/Digest pin and Resolve must
// accept that exact package set before any Schema is compiled.
func LoadCanonicalSource(root, revision, configPath string, requirePins bool) (*CanonicalSource, error) {
	configPath, err := canonicalRepositoryPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("canonical --config: %w", err)
	}
	s, err := source.Load(root, revision)
	if err != nil {
		return nil, err
	}
	return loadCanonicalSourceSnapshot(s, configPath, requirePins)
}

// loadCanonicalSourceSnapshot is the shared decode, activation, and compiler
// path for complete and explicitly selected source snapshots. The supplied
// snapshot's ID remains the full source revision; its file map may be sparse
// only when the caller also exposes a SelectedInputScope.
func loadCanonicalSourceSnapshot(s *snapshot.Snapshot, configPath string, requirePins bool) (*CanonicalSource, error) {
	if s == nil {
		return nil, errors.New("canonical source snapshot is required")
	}
	configBytes, ok := s.Files[configPath]
	if !ok {
		return nil, fmt.Errorf("canonical source config %q is not present in the selected source snapshot", configPath)
	}
	config, err := DecodeCanonicalSourceConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", configPath, err)
	}
	loaded := &CanonicalSource{Snapshot: s, ConfigPath: configPath, Config: config}
	seenManifest := map[string]bool{}
	moduleTypes := map[string]string{}
	for _, entry := range config.Modules {
		preview, pkg, err := loadCanonicalModule(s, entry)
		if err != nil {
			return nil, err
		}
		if seenManifest[entry.Manifest] {
			return nil, fmt.Errorf("canonical source config repeats Module manifest %q", entry.Manifest)
		}
		seenManifest[entry.Manifest] = true
		loaded.Previews = append(loaded.Previews, preview)
		loaded.Packages = append(loaded.Packages, pkg)
		moduleTypes[preview.Pin.Name] = preview.Type
		if entry.Pin == nil {
			if requirePins {
				return nil, fmt.Errorf("Module %q has no exact pin; run canonical --action modules and add the returned Name, Version and Digest", entry.Manifest)
			}
			continue
		}
		if entry.Pin.Name != preview.Pin.Name || entry.Pin.Version != preview.Pin.Version {
			return nil, fmt.Errorf("Module pin for %q identifies %s@%s but supplied manifest is %s@%s", entry.Manifest, entry.Pin.Name, entry.Pin.Version, preview.Pin.Name, preview.Pin.Version)
		}
		if entry.Pin.Digest != preview.Digest {
			return nil, fmt.Errorf("Module %s@%s digest mismatch: configured %s, supplied package is %s", preview.Pin.Name, preview.Pin.Version, entry.Pin.Digest, preview.Digest)
		}
		loaded.Pins = append(loaded.Pins, *entry.Pin)
	}
	if err := validateCanonicalProjectionBindings(config.ProjectionBindings, moduleTypes); err != nil {
		return nil, err
	}
	if requirePins {
		loaded.Activation, err = canonical.Resolve(loaded.Packages, loaded.Pins)
		if err != nil {
			return nil, fmt.Errorf("activate exact Modules: %w", err)
		}
		definitions := make([]core.Definition, 0, len(config.Definitions))
		for _, file := range config.Definitions {
			data, ok := s.Files[file]
			if !ok {
				return nil, fmt.Errorf("Definition %q is not present in the selected source snapshot", file)
			}
			definition, err := canonical.DecodeDefinition(file, data)
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		loaded.Model, loaded.Diagnostics = core.Compile(loaded.Activation.Schemas, definitions, s.ID)
	}
	return loaded, nil
}

// DecodeCanonicalSourceConfig strictly decodes the v1alpha1 source manifest.
func DecodeCanonicalSourceConfig(data []byte) (CanonicalSourceConfig, error) {
	if len(data) == 0 || len(data) > maxCanonicalSourceConfigBytes {
		return CanonicalSourceConfig{}, fmt.Errorf("canonical source config must be between 1 and %d bytes", maxCanonicalSourceConfigBytes)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return CanonicalSourceConfig{}, fmt.Errorf("invalid canonical source config: %w", err)
	}
	if err := rejectConfigAliases(&document); err != nil {
		return CanonicalSourceConfig{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var config CanonicalSourceConfig
	if err := decoder.Decode(&config); err != nil {
		return CanonicalSourceConfig{}, fmt.Errorf("invalid canonical source config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return CanonicalSourceConfig{}, fmt.Errorf("canonical source config must contain exactly one YAML document")
	} else if !errors.Is(err, io.EOF) {
		return CanonicalSourceConfig{}, fmt.Errorf("invalid canonical source config: %w", err)
	}
	if config.APIVersion != CanonicalSourceAPIVersion || config.Kind != "Source" {
		return CanonicalSourceConfig{}, fmt.Errorf("canonical source config must use apiVersion %q and kind Source", CanonicalSourceAPIVersion)
	}
	if config.Definitions == nil || config.Modules == nil {
		return CanonicalSourceConfig{}, fmt.Errorf("canonical source config must explicitly provide definitions and modules sequences")
	}
	if len(config.Definitions) > core.MaxDefinitions || len(config.Modules) > core.MaxSchemas || len(config.ProjectionBindings) > core.MaxDefinitions || len(config.Checks) > 128 {
		return CanonicalSourceConfig{}, fmt.Errorf("canonical source config exceeds Definition or Module input limits")
	}
	seen := map[string]bool{}
	for _, file := range config.Definitions {
		clean, err := canonicalRepositoryPath(file)
		if err != nil || clean != file {
			return CanonicalSourceConfig{}, fmt.Errorf("Definition path %q must be a clean repository-relative path", file)
		}
		if seen[file] {
			return CanonicalSourceConfig{}, fmt.Errorf("Definition path %q is duplicated", file)
		}
		seen[file] = true
	}
	seen = map[string]bool{}
	for i, module := range config.Modules {
		manifest, err := canonicalRepositoryPath(module.Manifest)
		if err != nil || manifest != module.Manifest {
			return CanonicalSourceConfig{}, fmt.Errorf("Module manifest path %q must be a clean repository-relative path", module.Manifest)
		}
		if path.Base(manifest) != canonical.ModuleManifestPath {
			return CanonicalSourceConfig{}, fmt.Errorf("Module manifest path %q must end in %s", manifest, canonical.ModuleManifestPath)
		}
		if seen[manifest] {
			return CanonicalSourceConfig{}, fmt.Errorf("Module manifest %q is duplicated", manifest)
		}
		seen[manifest] = true
		if module.Files == nil {
			return CanonicalSourceConfig{}, fmt.Errorf("Module %q must explicitly list files, including an empty list when it provides no Schemas", manifest)
		}
		if len(module.Files) > core.MaxSchemas {
			return CanonicalSourceConfig{}, fmt.Errorf("Module %q exceeds the exact Schema file input limit", manifest)
		}
		fileSeen := map[string]bool{}
		for _, file := range module.Files {
			clean, err := canonicalRepositoryPath(file)
			if err != nil || clean != file {
				return CanonicalSourceConfig{}, fmt.Errorf("Module file path %q must be a clean repository-relative path", file)
			}
			if fileSeen[file] {
				return CanonicalSourceConfig{}, fmt.Errorf("Module %q repeats file %q", manifest, file)
			}
			fileSeen[file] = true
		}
		if module.Pin != nil {
			if module.Pin.Name == "" || module.Pin.Version == "" || !strings.HasPrefix(module.Pin.Digest, "sha256:") || len(module.Pin.Digest) != len("sha256:")+64 {
				return CanonicalSourceConfig{}, fmt.Errorf("Module %q pin must contain exact name, version and sha256 digest", manifest)
			}
			for _, digit := range strings.TrimPrefix(module.Pin.Digest, "sha256:") {
				if !(digit >= '0' && digit <= '9' || digit >= 'a' && digit <= 'f') {
					return CanonicalSourceConfig{}, fmt.Errorf("Module %q pin digest must use 64 lowercase hexadecimal digits", manifest)
				}
			}
		}
		config.Modules[i] = module
	}
	if err := validateCanonicalProjectionBindings(config.ProjectionBindings, nil); err != nil {
		return CanonicalSourceConfig{}, err
	}
	return config, nil
}

func validateCanonicalProjectionBindings(bindings []canonical.ProjectionBinding, moduleTypes map[string]string) error {
	seen := map[string]bool{}
	for _, binding := range bindings {
		identity := binding.Projection
		if strings.TrimSpace(identity.APIVersion) != identity.APIVersion || identity.APIVersion == "" || strings.TrimSpace(identity.Kind) != identity.Kind || identity.Kind == "" || strings.TrimSpace(identity.Name) != identity.Name || identity.Name == "" {
			return fmt.Errorf("projection binding must contain an exact apiVersion, kind and name identity")
		}
		if identity.APIVersion != "markitect.foundation/v1" || identity.Kind != "Projection" {
			return fmt.Errorf("projection binding %q must identify a markitect.foundation/v1 Projection", identity.Key())
		}
		key := identity.Key()
		if seen[key] {
			return fmt.Errorf("projection binding for %q is duplicated", key)
		}
		seen[key] = true
		if strings.TrimSpace(binding.Module) != binding.Module || binding.Module == "" {
			return fmt.Errorf("projection binding %q must name one exact Projection Module", key)
		}
		if moduleTypes != nil {
			typeName, ok := moduleTypes[binding.Module]
			if !ok {
				return fmt.Errorf("projection binding %q names Module %q that is not listed in canonical source config", key, binding.Module)
			}
			if typeName != "projection" {
				return fmt.Errorf("projection binding %q names Module %q, which is not a Projection Module", key, binding.Module)
			}
		}
	}
	return nil
}

func rejectConfigAliases(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode {
		return fmt.Errorf("canonical source config does not allow YAML aliases")
	}
	for _, child := range node.Content {
		if err := rejectConfigAliases(child); err != nil {
			return err
		}
	}
	return nil
}

func loadCanonicalModule(s *snapshot.Snapshot, entry CanonicalModuleSource) (CanonicalModulePreview, canonical.ModulePackage, error) {
	manifestBytes, ok := s.Files[entry.Manifest]
	if !ok {
		return CanonicalModulePreview{}, canonical.ModulePackage{}, fmt.Errorf("Module manifest %q is not present in the selected source snapshot", entry.Manifest)
	}
	manifest, err := canonical.DecodeManifest(manifestBytes)
	if err != nil {
		return CanonicalModulePreview{}, canonical.ModulePackage{}, fmt.Errorf("%s: %w", entry.Manifest, err)
	}
	root := path.Dir(entry.Manifest)
	if root == "." {
		root = ""
	}
	pkg := canonical.ModulePackage{ManifestBytes: manifestBytes, Files: map[string][]byte{}, Modes: map[string]string{}}
	if mode := s.Modes[entry.Manifest]; mode != "" {
		pkg.Modes[canonical.ModuleManifestPath] = mode
	}
	files := append([]string(nil), entry.Files...)
	sort.Strings(files)
	preview := CanonicalModulePreview{Manifest: entry.Manifest, Type: manifest.Type, Files: append([]string(nil), files...), Schemas: append([]string(nil), manifest.Provides.Schemas...)}
	for _, registration := range manifest.Provides.Projectors {
		preview.Projectors = append(preview.Projectors, registration.ID)
	}
	listed := map[string]bool{}
	for _, repoPath := range files {
		if listed[repoPath] {
			return CanonicalModulePreview{}, canonical.ModulePackage{}, fmt.Errorf("Module %q repeats Schema file %q", entry.Manifest, repoPath)
		}
		listed[repoPath] = true
		relative := repoPath
		if root != "" {
			prefix := root + "/"
			if !strings.HasPrefix(repoPath, prefix) {
				return CanonicalModulePreview{}, canonical.ModulePackage{}, fmt.Errorf("Module file %q must remain under manifest directory %q", repoPath, root)
			}
			relative = strings.TrimPrefix(repoPath, prefix)
		}
		if relative == "" || relative == ".." || strings.HasPrefix(relative, "../") || path.IsAbs(relative) {
			return CanonicalModulePreview{}, canonical.ModulePackage{}, fmt.Errorf("Module file %q must remain under manifest directory %q", repoPath, root)
		}
		data, ok := s.Files[repoPath]
		if !ok {
			return CanonicalModulePreview{}, canonical.ModulePackage{}, fmt.Errorf("Module file %q is not present in the selected source snapshot", repoPath)
		}
		pkg.Files[relative] = data
		if mode := s.Modes[repoPath]; mode != "" {
			pkg.Modes[relative] = mode
		}
	}
	for _, schemaPath := range manifest.Provides.Schemas {
		if !listed[path.Join(root, schemaPath)] {
			return CanonicalModulePreview{}, canonical.ModulePackage{}, fmt.Errorf("Module %q provides Schema %q but that exact file is not listed in canonical config", manifest.Name, schemaPath)
		}
	}
	digest, err := canonical.DigestPackage(pkg)
	if err != nil {
		return CanonicalModulePreview{}, canonical.ModulePackage{}, fmt.Errorf("Module %q package: %w", manifest.Name, err)
	}
	preview.Digest = digest
	preview.Pin = canonical.Pin{Name: manifest.Name, Version: manifest.Version, Digest: digest}
	return preview, pkg, nil
}

func canonicalRepositoryPath(value string) (string, error) {
	if value == "" || strings.Contains(value, "\\") || path.IsAbs(value) {
		return "", fmt.Errorf("path must be nonempty and repository-relative")
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != value {
		return "", fmt.Errorf("path must be a clean repository-relative path")
	}
	return clean, nil
}
