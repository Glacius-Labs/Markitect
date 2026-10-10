package projectwork

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// Coverage observes complete repository classification for the requested
// source. It is available even when the project still uses legacy selected
// coverage, so callers can inspect migration gaps without changing config.
func Coverage(root, revision string) (projectcoverage.Report, error) {
	var config Config
	var manifestBytes []byte
	var err error
	if revision == "" {
		observed, observeErr := source.ObserveSelectedWorking(root, []string{ManifestPath})
		if observeErr != nil {
			return projectcoverage.Report{}, observeErr
		}
		if len(observed.MissingPaths) != 0 {
			return projectcoverage.Report{}, fmt.Errorf("project manifest %q is missing", ManifestPath)
		}
		manifestBytes = append([]byte(nil), observed.Snapshot.Files[ManifestPath]...)
		config, err = DecodeConfig(manifestBytes)
	} else {
		selected, selectErr := source.LoadSelected(root, revision, []string{ManifestPath})
		if selectErr != nil {
			return projectcoverage.Report{}, selectErr
		}
		manifestBytes = append([]byte(nil), selected.Snapshot.Files[ManifestPath]...)
		config, err = DecodeConfig(manifestBytes)
		revision = selected.Snapshot.ID
	}
	if err != nil {
		return projectcoverage.Report{}, fmt.Errorf("%s: %w", ManifestPath, err)
	}
	options := coverageOptions(config)
	if config.CoverageMode == "full" {
		project, loadErr := Load(root, revision)
		if loadErr != nil {
			return projectcoverage.Report{}, loadErr
		}
		if project.Coverage == nil {
			return projectcoverage.Report{}, fmt.Errorf("full project load did not produce repository coverage")
		}
		return *project.Coverage, nil
	}
	var universe *projectcoverage.Universe
	if revision == "" {
		universe, err = projectcoverage.ObserveWorking(root, options)
	} else {
		universe, err = projectcoverage.ObserveRevision(root, revision, options)
	}
	if err != nil {
		return projectcoverage.Report{}, err
	}
	if universe.Snapshot == nil || string(universe.Snapshot.Files[ManifestPath]) != string(manifestBytes) {
		return projectcoverage.Report{}, fmt.Errorf("project manifest changed while acquiring repository coverage snapshot")
	}
	project, err := FromSnapshot(root, universe.Snapshot)
	if err != nil {
		return projectcoverage.Report{}, err
	}
	return projectcoverage.Classify(projectcoverage.Request{Universe: universe, Model: project.Report, Options: options,
		LegacyRoots: config.InventoryRoots, LegacyExclusions: legacyExclusions(config.Exclusions)})
}

// ClassifyCandidate returns candidate, compiled from a snapshot derived from
// base, with full coverage classified against base's repository census plus
// the files the candidate adds, replaces or deletes. Ignored, transitional and
// operational paths absent from both snapshots keep their census state, so an
// unchanged candidate is classified exactly like its base.
func ClassifyCandidate(base, candidate *Project) (*Project, error) {
	if candidate == nil || candidate.Config.CoverageMode != "full" {
		return candidate, nil
	}
	if base == nil || base.census == nil || base.Snapshot == nil || candidate.Snapshot == nil {
		return nil, fmt.Errorf("candidate coverage requires the repository census of its base")
	}
	coverage, err := projectcoverage.ValidateCandidate(base.census, candidateDelta(base.Snapshot, candidate.Snapshot), candidate.Report,
		coverageOptions(candidate.Config), candidate.Config.InventoryRoots, legacyExclusions(candidate.Config.Exclusions))
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

func candidateDelta(base, candidate *snapshot.Snapshot) []projectcoverage.Delta {
	var delta []projectcoverage.Delta
	for file, data := range candidate.Files {
		mode := candidate.Modes[file]
		if old, exists := base.Files[file]; exists && bytes.Equal(old, data) && base.Modes[file] == mode {
			continue
		}
		digest := sha256.Sum256(data)
		change := projectcoverage.Delta{Path: file, Mode: mode, Digest: hex.EncodeToString(digest[:])}
		if file == projectcoverage.IgnorePath {
			change.Content = append([]byte(nil), data...)
		}
		delta = append(delta, change)
	}
	for file := range base.Files {
		if _, exists := candidate.Files[file]; !exists {
			delta = append(delta, projectcoverage.Delta{Path: file, Delete: true})
		}
	}
	sort.Slice(delta, func(i, j int) bool { return delta[i].Path < delta[j].Path })
	return delta
}

func ToolPaths(config Config) []projectcoverage.ToolPath {
	tools := []projectcoverage.ToolPath{
		{Selector: ManifestPath, Owner: "projectwork"},
		{Selector: RuntimePath, Owner: "projectwork"},
		{Selector: ".markitect/.gitignore", Owner: "projectwork"},
		{Selector: ".markitect/runs/", Owner: "projectrun", Operational: true},
		{Selector: ".markitect/write.lock", Owner: "projectwork", Operational: true},
		{Selector: ".markitect/cache/", Owner: "projectwork", Operational: true},
		{Selector: ".markitect/views/", Owner: "projectwork"},
		{Selector: ".markitect/drafts/", Owner: "projectwork", Operational: true},
		{Selector: ".markitect/state/briefings.json", Owner: "projectbriefing", Operational: true},
		{Selector: ".markitect/state/briefings.json.lock", Owner: "projectbriefing", Operational: true},
		{Selector: ".markitect/state/briefings.json.tmp", Owner: "projectbriefing", Operational: true},
		{Selector: ".markitect/state/briefings/", Owner: "projectbriefing", Operational: true},
		{Selector: ".markitect/state/explorations/", Owner: "projectexplore", Operational: true},
		{Selector: ".markitect/workflows/model-first.md", Owner: "projectwork"},
		{Selector: "AGENTS.md", Owner: "project-onboarding"},
		{Selector: "CLAUDE.md", Owner: "project-onboarding"},
	}
	for _, provider := range []string{"codex", "claude"} {
		for _, path := range NativeSkillPaths(provider) {
			if !toolRegistered(tools, path) {
				tools = append(tools, projectcoverage.ToolPath{Selector: path, Owner: "project-onboarding"})
			}
		}
	}
	if config.DocumentPath != "" && !toolRegistered(tools, config.DocumentPath) {
		tools = append(tools, projectcoverage.ToolPath{Selector: config.DocumentPath, Owner: "projectwork"})
	}
	return tools
}

// NativeSkillNames declares the exact operation skill identities owned by
// Markitect onboarding. Provider renderers supply their canonical content.
func NativeSkillNames() []string {
	return []string{"markitect-init", "markitect-extract", "markitect-design", "markitect-implement", "markitect-cleanup", "markitect-verify", "markitect-apply", "markitect-check", "markitect-suggest", "markitect-configure"}
}

// NativeSkillPaths returns exact generated files, never a directory prefix.
// Unrelated repository skills therefore remain owned by the adopting project.
func NativeSkillPaths(provider string) []string {
	base := ""
	switch provider {
	case "codex":
		base = ".agents/skills/"
	case "claude":
		base = ".claude/skills/"
	default:
		return nil
	}
	paths := []string{base + "markitect-implement/references/recovery.md"}
	for _, name := range NativeSkillNames() {
		paths = append(paths, base+name+"/SKILL.md", base+name+"/references/operating-guide.md")
	}
	return paths
}

func coverageOptions(config Config) projectcoverage.Options {
	transitional := make([]projectcoverage.TransitionalExclusion, 0, len(config.TransitionalExclusions))
	for _, exclusion := range config.TransitionalExclusions {
		transitional = append(transitional, projectcoverage.TransitionalExclusion{Path: exclusion.Path, Reason: exclusion.Reason})
	}
	return projectcoverage.Options{CanonicalModelPaths: append([]string(nil), config.ModelFiles...), ToolPaths: ToolPaths(config), Transitional: transitional}
}

// IsToolPath reports whether an exact repository path is registered as a
// Markitect-owned input/output for this project configuration.
func IsToolPath(config Config, file string) bool {
	return file == projectcoverage.IgnorePath || toolRegistered(ToolPaths(config), file)
}

// IsCanonicalModelPath reports whether file is one of the exact selected
// Definition sources for this project.
func IsCanonicalModelPath(config Config, file string) bool {
	for _, candidate := range config.ModelFiles {
		if candidate == file {
			return true
		}
	}
	return false
}

func fullSnapshotPaths(universe *projectcoverage.Universe) []string {
	if universe == nil {
		return nil
	}
	paths := []string{}
	for _, state := range universe.Paths {
		present := state.Worktree.Present
		if universe.FixedRevision {
			present = state.Head.Present
		}
		if state.Path == projectcoverage.IgnorePath && present {
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
		if state.Path == projectcoverage.IgnorePath {
			continue
		}
		if coverageOperational(state.Path) || coverageIgnored(universe, state.Path) {
			continue
		}
		paths = append(paths, state.Path)
	}
	return uniqueSorted(paths)
}

func loadFullCoverageSnapshot(root, revision string, provisional bool, config Config) (*snapshot.Snapshot, error) {
	options := coverageOptions(config)
	var universe *projectcoverage.Universe
	var err error
	if provisional {
		universe, err = projectcoverage.ObserveWorking(root, options)
	} else {
		universe, err = projectcoverage.ObserveRevision(root, revision, options)
	}
	if err != nil {
		return nil, err
	}
	if universe.Snapshot == nil {
		return nil, fmt.Errorf("repository census did not return its exact selected snapshot")
	}
	return externalInventorySnapshot(universe.Snapshot), nil
}

func externalInventorySnapshot(input *snapshot.Snapshot) *snapshot.Snapshot {
	result := &snapshot.Snapshot{ID: input.ID, Provisional: input.Provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
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
	for _, tool := range ToolPaths(Config{}) {
		if tool.Operational && pathMatches(tool.Selector, file) {
			return true
		}
	}
	return false
}

func toolRegistered(tools []projectcoverage.ToolPath, file string) bool {
	for _, tool := range tools {
		if pathMatches(tool.Selector, file) {
			return true
		}
	}
	return false
}

func coverageIgnored(universe *projectcoverage.Universe, file string) bool {
	ignore, err := projectcoverage.DecodeIgnore(universe.IgnoreBytes)
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

func coverageToolOwned(config Config, file string) bool {
	return IsToolPath(config, file)
}

func coverageCanonicalModel(config Config, file string) bool {
	return IsCanonicalModelPath(config, file)
}

func legacyExclusions(exclusions []Exclusion) []projectcoverage.TransitionalExclusion {
	result := make([]projectcoverage.TransitionalExclusion, 0, len(exclusions))
	for _, exclusion := range exclusions {
		result = append(result, projectcoverage.TransitionalExclusion{Path: exclusion.Path, Reason: exclusion.Reason})
	}
	return result
}
