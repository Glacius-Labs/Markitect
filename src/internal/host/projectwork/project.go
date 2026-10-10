package projectwork

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// Load loads only the Project manifest and exact paths it selects. A working
// tree inventory is acquired as metadata first, then its exact file list is
// captured through Infrastructure's selected-snapshot API.
func Load(root, revision string) (*Project, error) {
	// Full coverage observes the entire repository independently of the legacy
	// selected inventory roots. The fixed-revision path never consults the live
	// worktree when constructing its snapshot semantics.
	var fullUniverse *projectcoverage.Universe
	var fullOptions projectcoverage.Options
	var expectedManifest []byte
	var err error
	if revision == "" {
		// The manifest determines exact canonical model paths and optional view
		// output ownership, so read it before building the full registry.
		manifestCapture, captureErr := source.ObserveSelectedWorking(root, []string{ManifestPath})
		if captureErr != nil {
			return nil, captureErr
		}
		if len(manifestCapture.MissingPaths) != 0 {
			return nil, fmt.Errorf("project manifest %q is missing", ManifestPath)
		}
		config, decodeErr := DecodeConfig(manifestCapture.Snapshot.Files[ManifestPath])
		if decodeErr != nil {
			return nil, fmt.Errorf("%s: %w", ManifestPath, decodeErr)
		}
		fullOptions = coverageOptions(config)
		expectedManifest = append([]byte(nil), manifestCapture.Snapshot.Files[ManifestPath]...)
		if config.CoverageMode == "full" {
			fullUniverse, err = projectcoverage.ObserveWorking(root, fullOptions)
			if err != nil {
				return nil, fmt.Errorf("observe full repository coverage: %w", err)
			}
		}
	} else {
		manifestCapture, captureErr := source.LoadSelected(root, revision, []string{ManifestPath})
		if captureErr != nil {
			return nil, captureErr
		}
		config, decodeErr := DecodeConfig(manifestCapture.Snapshot.Files[ManifestPath])
		if decodeErr != nil {
			return nil, fmt.Errorf("%s: %w", ManifestPath, decodeErr)
		}
		fullOptions = coverageOptions(config)
		expectedManifest = append([]byte(nil), manifestCapture.Snapshot.Files[ManifestPath]...)
		if config.CoverageMode == "full" {
			fullUniverse, err = projectcoverage.ObserveRevision(root, manifestCapture.Snapshot.ID, fullOptions)
			if err != nil {
				return nil, fmt.Errorf("observe fixed revision coverage: %w", err)
			}
		}
	}
	var manifest *snapshot.Snapshot
	var configBytes []byte
	// err is shared with the optional coverage observation above.
	if fullUniverse != nil {
		manifest = fullUniverse.Snapshot
		if manifest == nil || !bytes.Equal(manifest.Files[ManifestPath], expectedManifest) {
			return nil, fmt.Errorf("project manifest changed while acquiring the full repository snapshot")
		}
	} else if revision == "" {
		observed, observeErr := source.ObserveSelectedWorking(root, []string{ManifestPath})
		if observeErr != nil {
			return nil, observeErr
		}
		if len(observed.MissingPaths) != 0 {
			return nil, fmt.Errorf("project manifest %q is missing", ManifestPath)
		}
		manifest = observed.Snapshot
	} else {
		selected, selectErr := source.LoadSelected(root, revision, []string{ManifestPath})
		if selectErr != nil {
			return nil, selectErr
		}
		manifest = selected.Snapshot
	}
	configBytes = manifest.Files[ManifestPath]
	config, err := DecodeConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ManifestPath, err)
	}
	paths := append([]string{ManifestPath}, config.ModelFiles...)
	if config.CoverageMode != "full" && revision != "" {
		// Runtime configuration is optional while loading the accepted model.
		// Inspect only this exact fixed-revision path to decide whether to bind
		// its bytes; runtime-dependent operations validate it separately.
		runtimeInventory, inventoryErr := source.InventoryRevisionRoots(root, manifest.ID, []string{RuntimePath})
		if inventoryErr != nil {
			return nil, fmt.Errorf("inspect optional runtime path: %w", inventoryErr)
		}
		for _, entry := range runtimeInventory.Entries {
			if entry.Path != RuntimePath {
				return nil, fmt.Errorf("%s must be a regular file", RuntimePath)
			}
			paths = append(paths, RuntimePath)
		}
	} else if config.CoverageMode != "full" {
		// Request the exact runtime path even when absent. The selected working
		// snapshot records presence or absence deterministically in its digest.
		paths = append(paths, RuntimePath)
	}
	if config.CoverageMode == "full" {
		paths = append(paths, fullSnapshotPaths(fullUniverse)...)
	}
	var inventoryMetadata *source.WorkingRootInventory
	if config.CoverageMode != "full" && revision == "" && len(config.InventoryRoots) > 0 {
		inventoryMetadata, err = source.InventoryWorkingRoots(root, config.InventoryRoots)
		if err != nil {
			return nil, fmt.Errorf("inventory selected project roots: %w", err)
		}
		for _, entry := range inventoryMetadata.Entries {
			if !excludedPath(config, entry.Path) {
				paths = append(paths, entry.Path)
			}
		}
	} else if config.CoverageMode != "full" && revision != "" && len(config.InventoryRoots) > 0 {
		fixedMetadata, inventoryErr := source.InventoryRevisionRoots(root, manifest.ID, config.InventoryRoots)
		if inventoryErr != nil {
			return nil, fmt.Errorf("inventory fixed project revision: %w", inventoryErr)
		}
		for _, entry := range fixedMetadata.Entries {
			if !excludedPath(config, entry.Path) {
				paths = append(paths, entry.Path)
			}
		}
	}
	paths = uniqueSorted(paths)
	var selected *snapshot.Snapshot
	if fullUniverse != nil {
		selected = fullUniverse.Snapshot
		if selected == nil || !bytes.Equal(selected.Files[ManifestPath], expectedManifest) {
			return nil, fmt.Errorf("project manifest changed while acquiring the full repository snapshot")
		}
	} else if revision == "" {
		observed, observeErr := source.ObserveSelectedWorking(root, paths)
		if observeErr != nil {
			return nil, observeErr
		}
		missing := map[string]bool{}
		for _, file := range observed.MissingPaths {
			missing[file] = true
		}
		for _, required := range append([]string{ManifestPath}, config.ModelFiles...) {
			if missing[required] {
				return nil, fmt.Errorf("selected project input %q is missing", required)
			}
		}
		for _, missingPath := range observed.MissingPaths {
			if missingPath != RuntimePath {
				return nil, fmt.Errorf("selected inventory file %q disappeared while acquiring the project snapshot", missingPath)
			}
		}
		if !bytes.Equal(observed.Snapshot.Files[ManifestPath], configBytes) {
			return nil, fmt.Errorf("project manifest changed while acquiring the selected snapshot")
		}
		selected = observed.Snapshot
	} else {
		var inventoryMetadata *source.RevisionRootInventory
		if config.CoverageMode != "full" && len(config.InventoryRoots) > 0 {
			inventoryMetadata, err = source.InventoryRevisionRoots(root, manifest.ID, config.InventoryRoots)
			if err != nil {
				return nil, fmt.Errorf("inventory fixed project revision: %w", err)
			}
			for _, entry := range inventoryMetadata.Entries {
				if !excludedPath(config, entry.Path) {
					paths = append(paths, entry.Path)
				}
			}
			paths = uniqueSorted(paths)
		}
		fixed, selectErr := source.LoadSelected(root, manifest.ID, paths)
		if selectErr != nil {
			return nil, selectErr
		}
		selected = fixed.Snapshot
	}
	project, err := FromSnapshot(root, selected)
	if err != nil {
		return nil, err
	}
	if fullUniverse != nil {
		coverage, coverageErr := projectcoverage.Classify(projectcoverage.Request{Universe: fullUniverse, Model: project.Report,
			Options: fullOptions, LegacyRoots: config.InventoryRoots, LegacyExclusions: legacyExclusions(config.Exclusions)})
		if coverageErr != nil {
			return nil, fmt.Errorf("classify full repository coverage: %w", coverageErr)
		}
		project.Coverage = &coverage
		project.Digest, err = digestProject(project)
		if err != nil {
			return nil, err
		}
	}
	return project, nil
}

// FromSnapshot compiles only the exact model files selected by the closed
// manifest and derives a report from the inventory bytes present in s and
// admitted by configured roots. Other snapshot entries are ignored.
func FromSnapshot(root string, s *snapshot.Snapshot) (*Project, error) {
	if s == nil {
		return nil, fmt.Errorf("project snapshot is required")
	}
	rootAbs, err := absoluteRoot(root)
	if err != nil {
		return nil, err
	}
	manifest, ok := s.Files[ManifestPath]
	if !ok {
		return nil, fmt.Errorf("selected project snapshot is missing %q", ManifestPath)
	}
	config, err := DecodeConfig(manifest)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ManifestPath, err)
	}
	files := map[string][]byte{ManifestPath: append([]byte(nil), manifest...)}
	modes := map[string]string{ManifestPath: s.Modes[ManifestPath]}
	if modes[ManifestPath] == "" {
		return nil, fmt.Errorf("selected project manifest has no source mode")
	}
	if runtimeBytes, present := s.Files[RuntimePath]; present {
		files[RuntimePath] = append([]byte(nil), runtimeBytes...)
		modes[RuntimePath] = s.Modes[RuntimePath]
		if modes[RuntimePath] == "" {
			return nil, fmt.Errorf("selected runtime config has no source mode")
		}
	}
	definitions := make([]core.Definition, 0, len(config.ModelFiles))
	for _, file := range config.ModelFiles {
		data, exists := s.Files[file]
		if !exists {
			return nil, fmt.Errorf("selected model file %q is missing from the source snapshot", file)
		}
		mode := s.Modes[file]
		if mode == "" {
			return nil, fmt.Errorf("selected model file %q has no source mode", file)
		}
		files[file] = append([]byte(nil), data...)
		modes[file] = mode
		definition, decodeErr := canonical.DecodeDefinition(file, data)
		if decodeErr != nil {
			return nil, decodeErr
		}
		namespace, namespaceErr := namespaceForModelPath(file)
		if namespaceErr != nil {
			return nil, namespaceErr
		}
		if definition.Metadata.Namespace != namespace {
			return nil, fmt.Errorf("Definition %s declares namespace %q, but its selected model path requires %q", file, definition.Metadata.Namespace, namespace)
		}
		definitions = append(definitions, definition)
	}
	var inventory []projectmodel.File
	for file, data := range s.Files {
		selectedInventory := config.CoverageMode != "full" && inInventoryRoots(config, file) && !excludedPath(config, file) && !strings.HasPrefix(file, ".markitect/") && file != ManifestPath && file != RuntimePath
		fullInventory := config.CoverageMode == "full" && !coverageToolOwned(config, file) && !coverageCanonicalModel(config, file)
		fullToolInput := config.CoverageMode == "full" && coverageToolOwned(config, file)
		if !(selectedInventory || fullInventory || fullToolInput) {
			continue
		}
		mode := s.Modes[file]
		if mode == "" {
			return nil, fmt.Errorf("selected inventory file %q has no source mode", file)
		}
		files[file] = append([]byte(nil), data...)
		modes[file] = mode
		if selectedInventory || fullInventory {
			digest := sha256.Sum256(data)
			inventory = append(inventory, projectmodel.File{Path: file, Mode: mode, Digest: hex.EncodeToString(digest[:])})
		}
	}
	sort.Slice(inventory, func(i, j int) bool { return inventory[i].Path < inventory[j].Path })
	model, diagnostics := core.Compile([]core.Schema{projectmodel.Schema()}, definitions, s.ID)
	if len(diagnostics) > 0 {
		return nil, fmt.Errorf("compile selected project model: %s", diagnostics[0].Message)
	}
	report := projectmodel.Analyze(model, inventory)
	selected := &snapshot.Snapshot{ID: s.ID, Provisional: s.Provisional, Files: files, Modes: modes}
	project := &Project{Root: rootAbs, Revision: s.ID, Provisional: s.Provisional, Config: config, Model: model, Report: report, Snapshot: selected}
	if config.CoverageMode == "full" {
		options := coverageOptions(config)
		universe, universeErr := projectcoverage.FromSnapshot(s, options)
		if universeErr != nil {
			return nil, fmt.Errorf("build candidate coverage universe: %w", universeErr)
		}
		coverage, coverageErr := projectcoverage.Classify(projectcoverage.Request{Universe: universe, Model: report, Options: options,
			LegacyRoots: config.InventoryRoots, LegacyExclusions: legacyExclusions(config.Exclusions)})
		if coverageErr != nil {
			return nil, fmt.Errorf("classify candidate repository snapshot: %w", coverageErr)
		}
		project.Coverage = &coverage
	}
	project.Digest, err = digestProject(project)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func digestProject(project *Project) (string, error) {
	h := sha256.New()
	write := func(label string, data []byte) {
		_, _ = fmt.Fprintf(h, "%d:", len(label))
		_, _ = h.Write([]byte(label))
		_, _ = fmt.Fprintf(h, "%d:", len(data))
		_, _ = h.Write(data)
	}
	if project.Snapshot != nil {
		write("snapshot", []byte(project.Snapshot.Digest()))
	}
	write("revision", []byte(project.Revision))
	write("provisional", []byte(fmt.Sprint(project.Provisional)))
	configBytes, _ := json.Marshal(project.Config)
	write("config", configBytes)
	schemaBytes, _ := json.Marshal(projectmodel.Schema())
	write("schema", schemaBytes)
	modelBytes, _ := json.Marshal(project.Model)
	write("model", modelBytes)
	reportBytes, _ := json.Marshal(project.Report)
	write("report", reportBytes)
	if project.Coverage != nil {
		write("coverage", []byte(project.Coverage.Digest))
	}
	toolBuildDigest, err := ToolBuildDigest()
	if err != nil {
		return "", fmt.Errorf("identify running Markitect build: %w", err)
	}
	write("tool-build", []byte(toolBuildDigest))
	if info, ok := debug.ReadBuildInfo(); ok && info != nil {
		write("build-info", []byte(info.String()))
	} else {
		write("build-info", []byte("unavailable"))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func inInventoryRoots(config Config, file string) bool {
	for _, root := range config.InventoryRoots {
		if file == root || strings.HasPrefix(file, strings.TrimSuffix(root, "/")+"/") {
			return true
		}
	}
	return false
}

func excludedPath(config Config, file string) bool {
	for _, exclusion := range config.Exclusions {
		if pathWithin(file, exclusion.Path) {
			return true
		}
	}
	return false
}

func uniqueSorted(paths []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(paths))
	for _, value := range paths {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func absoluteRoot(root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("project root is empty")
	}
	return filepath.Abs(root)
}
