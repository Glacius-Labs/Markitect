package projectwork

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"sort"
	"strings"

	coverageacquisition "github.com/Glacius-Labs/Markitect/src/internal/application/projectcoverage"
	modelcoverage "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectcoverage"
	
	"github.com/Glacius-Labs/Markitect/internal/tooling/source"
)

// Coverage observes complete repository classification for the requested
// source. It is available even when the project still uses legacy selected
// coverage, so callers can inspect migration gaps without changing config.
func Coverage(root, revision string) (modelcoverage.Report, error) {
	var config projectvalue.Config
	var manifestBytes []byte
	var err error
	if revision == "" {
		observed, observeErr := source.ObserveSelectedWorking(root, []string{projectvalue.ManifestPath})
		if observeErr != nil {
			return modelcoverage.Report{}, observeErr
		}
		if len(observed.MissingPaths) != 0 {
			return modelcoverage.Report{}, fmt.Errorf("project manifest %q is missing", projectvalue.ManifestPath)
		}
		manifestBytes = append([]byte(nil), observed.Snapshot.Files[projectvalue.ManifestPath]...)
		config, err = DecodeConfig(manifestBytes)
	} else {
		selected, selectErr := source.LoadSelected(root, revision, []string{projectvalue.ManifestPath})
		if selectErr != nil {
			return modelcoverage.Report{}, selectErr
		}
		manifestBytes = append([]byte(nil), selected.Snapshot.Files[projectvalue.ManifestPath]...)
		config, err = DecodeConfig(manifestBytes)
		revision = selected.Snapshot.ID
	}
	if err != nil {
		return modelcoverage.Report{}, fmt.Errorf("%s: %w", projectvalue.ManifestPath, err)
	}
	options := coverageOptions(config)
	if config.CoverageMode == "full" {
		project, loadErr := Load(root, revision)
		if loadErr != nil {
			return modelcoverage.Report{}, loadErr
		}
		if project.Coverage == nil {
			return modelcoverage.Report{}, fmt.Errorf("full project load did not produce repository coverage")
		}
		return *project.Coverage, nil
	}
	var universe *modelcoverage.Universe
	if revision == "" {
		universe, err = coverageacquisition.ObserveWorking(root, options)
	} else {
		universe, err = coverageacquisition.ObserveRevision(root, revision, options)
	}
	if err != nil {
		return modelcoverage.Report{}, err
	}
	if universe.Snapshot == nil || string(universe.Snapshot.Files[projectvalue.ManifestPath]) != string(manifestBytes) {
		return modelcoverage.Report{}, fmt.Errorf("project manifest changed while acquiring repository coverage snapshot")
	}
	project, err := FromSnapshot(root, universe.Snapshot)
	if err != nil {
		return modelcoverage.Report{}, err
	}
	return modelcoverage.Classify(modelcoverage.Request{Universe: universe, Model: project.Report, Options: options,
		LegacyRoots: config.InventoryRoots, LegacyExclusions: legacyExclusions(config.Exclusions)})
}

// ClassifyCandidate returns candidate, compiled from a snapshot derived from
// base, with full coverage classified against base's repository census plus
// the files the candidate adds, replaces or deletes. Ignored, transitional and
// operational files keep their census state, so an unchanged candidate is
// classified exactly like its base. Those files are in no snapshot that review,
// checks or Apply see, so a candidate that names one in deleted is refused.
// Files the candidate's policy no longer excludes, such as a transitional file
// it now models, are read from the base source and compiled into the candidate.
func ClassifyCandidate(base, candidate *projectvalue.Project, deleted ...string) (*projectvalue.Project, error) {
	if candidate == nil || candidate.Config.CoverageMode != "full" {
		return candidate, nil
	}
	if base == nil || base.Census == nil || base.Snapshot == nil || candidate.Snapshot == nil {
		return nil, fmt.Errorf("candidate coverage requires the repository census of its base")
	}
	if err := refuseCensusOnlyDeletes(base, deleted); err != nil {
		return nil, err
	}
	options := coverageOptions(candidate.Config)
	delta := candidateDelta(base.Snapshot, candidate.Snapshot)
	overlaid, err := modelcoverage.Overlay(base.Census, delta, options)
	if err != nil {
		return nil, fmt.Errorf("classify candidate against the repository census: %w", err)
	}
	unbound, err := modelcoverage.UnboundPaths(overlaid, options)
	if err != nil {
		return nil, err
	}
	if len(unbound) > 0 {
		read, readErr := readCensusFiles(base, unbound)
		if readErr != nil {
			return nil, readErr
		}
		extended := &source.Input{ID: candidate.Snapshot.ID, Provisional: candidate.Snapshot.Provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
		for _, input := range []*source.Input{candidate.Snapshot, read} {
			for file, data := range input.Files {
				extended.Files[file], extended.Modes[file] = data, input.Modes[file]
			}
		}
		if candidate, err = FromSnapshot(base.Root, extended); err != nil {
			return nil, err
		}
		delta = candidateDelta(base.Snapshot, candidate.Snapshot)
	}
	coverage, err := modelcoverage.ValidateCandidate(base.Census, delta, candidate.Report,
		options, candidate.Config.InventoryRoots, legacyExclusions(candidate.Config.Exclusions))
	if err != nil {
		return nil, fmt.Errorf("classify candidate against the repository census: %w", err)
	}
	classified := *candidate
	classified.Coverage = &coverage
	classified.Digest, err = digestProject(&classified)
	if err != nil {
		return nil, err
	}
	return &classified, nil
}

// refuseCensusOnlyDeletes rejects deleting a current census file that base's
// snapshot omits. Apply writes only snapshot changes, so accepting the delete
// would close a candidate whose coverage Apply never produces.
func refuseCensusOnlyDeletes(base *projectvalue.Project, deleted []string) error {
	names := make(map[string]bool, len(deleted))
	for _, file := range deleted {
		names[file] = true
	}
	for _, state := range base.Census.Paths {
		present := state.Worktree.Present
		if base.Census.FixedRevision {
			present = state.Head.Present
		}
		if _, inSnapshot := base.Snapshot.Files[state.Path]; names[state.Path] && present && !inSnapshot {
			return fmt.Errorf("candidate deletes %s, an ignored, transitional or operational file outside the reviewed snapshot; Apply cannot delete it, so remove it outside the run", state.Path)
		}
	}
	return nil
}

func candidateDelta(base, candidate *source.Input) []modelcoverage.Delta {
	var delta []modelcoverage.Delta
	for file, data := range candidate.Files {
		mode := candidate.Modes[file]
		if old, exists := base.Files[file]; exists && bytes.Equal(old, data) && base.Modes[file] == mode {
			continue
		}
		digest := sha256.Sum256(data)
		change := modelcoverage.Delta{Path: file, Mode: mode, Digest: hex.EncodeToString(digest[:])}
		if file == modelcoverage.IgnorePath {
			change.Content = append([]byte(nil), data...)
		}
		delta = append(delta, change)
	}
	for file := range base.Files {
		if _, exists := candidate.Files[file]; !exists {
			delta = append(delta, modelcoverage.Delta{Path: file, Delete: true})
		}
	}
	sort.Slice(delta, func(i, j int) bool { return delta[i].Path < delta[j].Path })
	return delta
}

// readCensusFiles reads files the census skipped from the same source it
// observed: the fixed revision, or the working tree with an unchanged identity.
func readCensusFiles(base *projectvalue.Project, paths []string) (*source.Input, error) {
	if base.Census.FixedRevision {
		selected, err := source.LoadSelected(base.Root, base.Census.Revision, paths)
		if err != nil {
			return nil, fmt.Errorf("read census files the candidate now models: %w", err)
		}
		return selected.Snapshot, nil
	}
	observed, err := source.ObserveSelectedWorking(base.Root, paths)
	if err != nil {
		return nil, fmt.Errorf("read census files the candidate now models: %w", err)
	}
	if observed.Identity.Digest != base.Census.IdentityDigest || len(observed.MissingPaths) != 0 {
		return nil, fmt.Errorf("repository changed since its census; reload the project")
	}
	return observed.Snapshot, nil
}

func coverageOptions(config projectvalue.Config) modelcoverage.Options {
	transitional := make([]modelcoverage.TransitionalExclusion, 0, len(config.TransitionalExclusions))
	for _, exclusion := range config.TransitionalExclusions {
		transitional = append(transitional, modelcoverage.TransitionalExclusion{Path: exclusion.Path, Reason: exclusion.Reason})
	}
	return modelcoverage.Options{CanonicalModelPaths: append([]string(nil), config.ModelFiles...), ToolPaths: projectvalue.ToolPaths(config), Transitional: transitional}
}

// IsToolPath reports whether an exact repository path is registered as a
// Markitect-owned input/output for this project configuration.
func fullSnapshotPaths(universe *modelcoverage.Universe) []string {
	if universe == nil {
		return nil
	}
	paths := []string{}
	for _, state := range universe.Paths {
		present := state.Worktree.Present
		if universe.FixedRevision {
			present = state.Head.Present
		}
		if state.Path == modelcoverage.IgnorePath && present {
			paths = append(paths, state.Path)
			break
		}
	}
	for _, state := range universe.Paths {
		if state.OpaqueBoundary {
			continue
		}
		present := state.Worktree.Present
		if universe.FixedRevision {
			present = state.Head.Present
		}
		if !present || strings.HasSuffix(state.Path, "/") {
			continue
		}
		if state.Path == modelcoverage.IgnorePath {
			continue
		}
		if coverageOperational(state.Path) || coverageIgnored(universe, state.Path) {
			continue
		}
		paths = append(paths, state.Path)
	}
	return uniqueSorted(paths)
}

func loadFullCoverageSnapshot(root, revision string, provisional bool, config projectvalue.Config) (*source.Input, error) {
	options := coverageOptions(config)
	var universe *modelcoverage.Universe
	var err error
	if provisional {
		universe, err = coverageacquisition.ObserveWorking(root, options)
	} else {
		universe, err = coverageacquisition.ObserveRevision(root, revision, options)
	}
	if err != nil {
		return nil, err
	}
	if universe.Snapshot == nil {
		return nil, fmt.Errorf("repository census did not return its exact selected snapshot")
	}
	return externalInventorySnapshot(universe.Snapshot), nil
}

func externalInventorySnapshot(input *source.Input) *source.Input {
	result := &source.Input{ID: input.ID, Provisional: input.Provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	for file, data := range input.Files {
		if strings.HasPrefix(file, ".markitect/") {
			continue
		}
		result.Files[file] = append([]byte(nil), data...)
		result.Modes[file] = input.Modes[file]
	}
	return result
}

func coverageOperational(file string) bool {
	for _, tool := range projectvalue.ToolPaths(projectvalue.Config{}) {
		if tool.Operational && pathMatches(tool.Selector, file) {
			return true
		}
	}
	return false
}

func coverageIgnored(universe *modelcoverage.Universe, file string) bool {
	ignore, err := modelcoverage.DecodeIgnore(universe.IgnoreBytes)
	if err != nil {
		return false
	}
	for _, entry := range ignore.Entries {
		if pathMatches(entry.Path, file) {
			return true
		}
	}
	return false
}

func pathMatches(selector, file string) bool {
	selector = strings.TrimSuffix(selector, "/")
	return file == selector || strings.HasPrefix(file, selector+"/")
}

func coverageToolOwned(config projectvalue.Config, file string) bool {
	return projectvalue.IsToolPath(config, file)
}

func coverageCanonicalModel(config projectvalue.Config, file string) bool {
	return projectvalue.IsCanonicalModelPath(config, file)
}

func legacyExclusions(exclusions []projectvalue.Exclusion) []modelcoverage.TransitionalExclusion {
	result := make([]modelcoverage.TransitionalExclusion, 0, len(exclusions))
	for _, exclusion := range exclusions {
		result = append(result, modelcoverage.TransitionalExclusion{Path: exclusion.Path, Reason: exclusion.Reason})
	}
	return result
}
