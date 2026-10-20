package apply

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ErrNotExtractable reports a preparation that is not the start of an Extract.
var ErrNotExtractable = errors.New("the prepared repository is not the start of an Extract")

const (
	modelDirectory = ".markitect/model/"
	executionFile  = ".markitect/execution.yaml"
	markitectDir   = ".markitect/"
)

// extractStage is everything one phase of an Extract is built from: the
// structural preparation of the phase's start tree, its resolved settings and
// the authored budget. The first phase starts from the captured repository; the
// second from the verified candidate of the first.
type extractStage struct {
	structural    CapturedPreparationAttempt
	settings      organization.Settings
	facts         organization.SettingsFacts
	budget        ExecutionBudget
	checkProfiles []checkProfileOffer
	tree          snapshot.Tree
}

// checkProfileOffer is a check profile the Runtime offers: its name and what it
// runs. A proposed check can name only these profiles.
type checkProfileOffer struct{ Name, Runs string }

// offeredCheckProfiles lists the check profiles of the Runtime that the start
// prepared, in name order.
func offeredCheckProfiles(prepared CapturedSettingsPreparation) []checkProfileOffer {
	var offers []checkProfileOffer
	for _, profile := range prepared.ProfileObservations() {
		if _, ok := profile.Identity(); !ok || profile.Input().Namespace() != "checkProfiles" {
			continue
		}
		observation := profile.Observation()
		runs := ""
		switch {
		case observation.CheckImage != nil:
			runs = observation.CheckImage.Executable.Path
		case len(observation.Files) > 0:
			runs = observation.Files[0].Location.Path.String()
		}
		offers = append(offers, checkProfileOffer{Name: profile.Input().Name(), Runs: runs})
	}
	slices.SortFunc(offers, func(a, b checkProfileOffer) int { return strings.Compare(a.Name, b.Name) })
	return offers
}

// newExtractStart validates a preparation made by CaptureExtract as the start of
// an Extract. The artifacts are unregistered and the local model has no
// Definitions yet; a model that already describes the project is Apply's input.
// The project declares the ownership of its artifacts.
func newExtractStart(prepared CapturedSettingsAttempt) (extractStage, error) {
	if prepared.Settings().Status() != model.Passed {
		return extractStage{}, fmt.Errorf("%w: the preparation did not pass", ErrNotExtractable)
	}
	structural := prepared.Structural()
	settings, ok := prepared.Settings().Settings()
	if !ok {
		return extractStage{}, errors.New("resolved settings are required")
	}
	facts, ok := prepared.Settings().SettingsFacts()
	if !ok {
		return extractStage{}, errors.New("the settings facts are required")
	}
	budget, ok := prepared.Settings().Budget()
	if !ok {
		return extractStage{}, errors.New("the execution budget is required")
	}
	if !structural.Model().extract {
		return extractStage{}, fmt.Errorf("%w: it was not prepared for Extract", ErrNotExtractable)
	}
	if err := requireEmptyModel(structural.Model().checked); err != nil {
		return extractStage{}, err
	}
	if !structural.files.layout.Declared() || len(structural.files.layout.Areas()) == 0 {
		return extractStage{}, fmt.Errorf("%w: .markitect/execution.yaml declares no ownership areas for the artifacts", ErrNotExtractable)
	}
	return extractStage{structural: structural, settings: settings, facts: facts, budget: budget, checkProfiles: offeredCheckProfiles(prepared.Settings()), tree: structural.Model().inputs.Raw.Census}, nil
}

// requireEmptyModel refuses a local model that already holds Definitions.
func requireEmptyModel(checked model.Result) error {
	for _, owner := range checked.Models {
		if owner.Local && len(owner.Definitions) > 0 {
			return fmt.Errorf("%w: the local model already has %d Definitions; use Apply", ErrNotExtractable, len(owner.Definitions))
		}
	}
	return nil
}

// next prepares the second phase from the verified candidate of the first: the
// same capture with the candidate's model, layout and registry, and the same
// settings resolved for the candidate's Manager tree.
func (s extractStage) next(candidate snapshot.Tree) (extractStage, error) {
	structural := rebindStructural(s.structural.Model(), candidate, extractStructure)
	for name, status := range map[string]model.Status{"model": structural.Model().checked.Status, "registry": structural.registry.Status(), "files": structural.files.Status(), "execution": structural.execution.Status(), "contexts": structural.contexts.Status()} {
		if status != model.Passed {
			return extractStage{}, fmt.Errorf("the structure candidate does not prepare (%s %s)", name, status)
		}
	}
	topology, err := organization.BuildTopology(structural.Model().checked)
	if err != nil {
		return extractStage{}, err
	}
	// The profiles and instructions were observed once at the start; only the
	// Manager tree changed, so the same facts resolve for it.
	settings, err := organization.ResolveSettings(topology, s.facts)
	if err != nil {
		return extractStage{}, err
	}
	return extractStage{structural: structural, settings: settings, facts: s.facts, budget: s.budget, checkProfiles: s.checkProfiles, tree: candidate}, nil
}

// rebindStructural prepares the structure of a tree the way the capture
// prepared the original repository: the model, the execution document, the
// registry shards and the other repository-local inputs are read from the tree,
// everything that lies outside the repository stays as captured. It is a pure
// function of its arguments and shares no mutable state, so concurrent callers
// are safe. The phase says which phase's candidate the tree is: while the
// structure is proposed, artifacts without an owner or registration are still
// expected; in the model phase they block.
func rebindStructural(start CapturedModel, tree snapshot.Tree, phase extractPhase) CapturedPreparationAttempt {
	raw := cloneRaw(start.inputs.Raw)
	raw.Census = tree
	raw.Local = rebindModelSource(raw.Local, tree)
	raw.Execution = rebindFiles(raw.Execution, tree)
	raw.Ignore = rebindFiles(raw.Ignore, tree)
	raw.Instructions = rebindFiles(raw.Instructions, tree)
	raw.Registry = nil
	repository := start.request.Selection.Repository
	for _, entry := range tree.Entries() {
		if name := entry.Path().String(); strings.HasPrefix(name, registryDirectory) {
			raw.Registry = append(raw.Registry, SourceFile{Location: SourceLocation{Boundary: repository, Path: entry.Path()}, File: entry, RepositoryPath: entry.Path()})
		}
	}
	stage := CapturedModel{request: start.request, inputs: CapturedInputs{Raw: raw, Head: start.inputs.Head, Guard: start.inputs.Guard}, limits: start.limits, extract: phase == extractStructure}
	stage.checked = checkModel(raw, start.limits)
	registry, execution := stage.PrepareRegistry(), stage.DecodeExecution()
	return CapturedPreparationAttempt{stage: stage, registry: registry, files: registry.CheckFiles(), execution: execution, contexts: execution.SelectContexts()}
}

// rebindModelSource reads the manifest, the lock and the model tree of a
// repository-local model from the tree. An external dependency is unchanged.
func rebindModelSource(source ModelSource, tree snapshot.Tree) ModelSource {
	area := source.RepositoryArea
	if area.String() == "" {
		return source
	}
	known := make(map[string]bool)
	for _, entry := range source.Files.Entries() {
		known[entry.Path().String()] = true
	}
	var files []snapshot.Entry
	var entries []model.Entry
	for _, entry := range tree.Entries() {
		if !area.Contains(entry.Path()) {
			continue
		}
		relative := entry.Path().String()
		if !area.IsRoot() {
			relative = strings.TrimPrefix(relative, area.String()+"/")
		}
		relativePath, err := snapshot.NewPath(relative)
		if err != nil || !known[relative] && !source.Root.Contains(relativePath) {
			continue
		}
		file, err := snapshot.NewEntry(relativePath, entry.Mode(), entry.Bytes())
		if err != nil {
			continue
		}
		files = append(files, file)
		entries = append(entries, model.Entry{Path: relative, Kind: model.Regular, Bytes: entry.Bytes()})
	}
	source.Files, _ = snapshot.NewTree(files)
	source.Source.Entries = entries
	return source
}

// rebindFiles reads repository-local inputs from the tree; a file the tree no
// longer holds is absent, and an input outside the repository stays.
func rebindFiles(files []SourceFile, tree snapshot.Tree) []SourceFile {
	var result []SourceFile
	for _, source := range files {
		if source.RepositoryPath.String() == "" {
			result = append(result, source)
			continue
		}
		entry, found := tree.Lookup(source.RepositoryPath)
		if !found {
			continue
		}
		if file, err := snapshot.NewEntry(source.File.Path(), entry.Mode(), entry.Bytes()); err == nil {
			source.File = file
			result = append(result, source)
		}
	}
	return result
}

// artifactPaths returns the existing artifacts of a tree: every file outside
// .markitect.
func artifactPaths(tree snapshot.Tree) []snapshot.Path {
	var paths []snapshot.Path
	for _, entry := range tree.Entries() {
		if !strings.HasPrefix(entry.Path().String(), markitectDir) {
			paths = append(paths, entry.Path())
		}
	}
	return paths
}

// packageNames lists the local Packages of a checked model.
func packageNames(checked model.Result) []string {
	var names []string
	for _, owner := range checked.Models {
		if owner.Local {
			for _, p := range owner.Packages {
				names = append(names, p.Name)
			}
		}
	}
	slices.Sort(names)
	return names
}

// modelPackageOf returns the deepest of the Packages whose model directory
// contains the repository path, or Root.
func modelPackageOf(packages []string, name string) string {
	relative, ok := strings.CutPrefix(name, modelDirectory)
	if !ok {
		return organization.Root
	}
	for dir := path.Dir(relative); dir != "." && dir != "/"; dir = path.Dir(dir) {
		if slices.Contains(packages, dir) {
			return dir
		}
	}
	return organization.Root
}
