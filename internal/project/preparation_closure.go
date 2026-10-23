package project

import (
	"fmt"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Validate explicit requests without repairing duplicates, paths or boundaries.
// Physical locations and fixed revisions remain the acquisition adapter's job.
func initialPreparationSelection(original InputSelection) (InputSelection, error) {
	selection := cloneSelection(original)
	invalid := func(detail string) (InputSelection, error) {
		return selection, fmt.Errorf("%w: %s", ErrPreparationSelection, detail)
	}
	if selection.Repository == "" {
		return invalid("repository is required")
	}
	seenModels := make(map[ModelSelection]bool)
	for _, selected := range append([]ModelSelection{selection.Local}, selection.Dependencies...) {
		if selected.Boundary == "" || selected.ManifestPath.String() == "" || seenModels[selected] {
			return invalid("invalid or repeated model selection")
		}
		seenModels[selected] = true
	}
	for _, group := range [][]SourceLocation{selection.Execution, selection.Registry, selection.Ignore, selection.RuntimeProfiles, selection.Instructions} {
		seen := make(map[SourceLocation]bool)
		for _, selected := range group {
			if selected.Boundary == "" || selected.Path.String() == "" || seen[selected] {
				return invalid("invalid or repeated source location")
			}
			seen[selected] = true
		}
	}
	if len(selection.Execution) == 0 {
		selection.Execution = []SourceLocation{{Boundary: selection.Repository, Path: mustExecutionPath()}}
	}
	return selection, nil
}

type preparationClosure struct {
	registry, ignore, instructions []snapshot.Path
	runtime                        []SourceLocation
}

func observedPreparationClosure(raw RawInputs, registry []snapshot.Path, execution AuthoredExecution) preparationClosure {
	closure := preparationClosure{registry: registry}
	ignore, _ := snapshot.NewPath(ignorePath)
	if _, found := raw.Census.Lookup(ignore); found {
		closure.ignore = []snapshot.Path{ignore}
	}
	lists := instructionLists(execution)
	seen := make(map[snapshot.Path]bool)
	for _, list := range lists {
		if list == nil {
			continue
		}
		for _, ref := range list.Value {
			path, err := snapshot.NewPath(ref)
			// Existing FileCheck retains invalid and physically missing references
			// as known failures. Never request invented bytes for them.
			if err != nil || seen[path] {
				continue
			}
			seen[path] = true
			closure.instructions = append(closure.instructions, path)
		}
	}
	return closure
}

// Derive repository-root locations only from actual Census coordinates. An
// existing opaque locator counts only through its observed RepositoryPath.
// Authored instruction order/repetitions stay untouched in Execution; only
// acquisition of distinct required files is deduplicated.
func (c preparationClosure) selectMissing(selection InputSelection, raw RawInputs) (InputSelection, bool) {
	selection = cloneSelection(selection)
	missing := false
	groups := []struct {
		required []snapshot.Path
		observed []SourceFile
		selected *[]SourceLocation
	}{
		{c.registry, raw.Registry, &selection.Registry},
		{c.ignore, raw.Ignore, &selection.Ignore},
		{c.instructions, raw.Instructions, &selection.Instructions},
	}
	for _, group := range groups {
		seen := make(map[snapshot.Path]bool)
		for _, source := range group.observed {
			seen[source.RepositoryPath] = true
		}
		for _, path := range group.required {
			if seen[path] {
				continue
			}
			if _, present := raw.Census.Lookup(path); !present {
				continue
			}
			location := SourceLocation{Boundary: selection.Repository, Path: path}
			if !slices.Contains(*group.selected, location) {
				*group.selected = append(*group.selected, location)
			}
			seen[path], missing = true, true
		}
	}
	seen := make(map[SourceLocation]bool)
	for _, location := range selection.Instructions {
		seen[location] = true
	}
	for _, location := range c.runtime {
		if !seen[location] {
			selection.Instructions = append(selection.Instructions, location)
			seen[location], missing = true, true
		}
	}
	return selection, missing
}

func (c preparationClosure) equal(other preparationClosure) bool {
	return samePreparationPaths(c.registry, other.registry) && samePreparationPaths(c.ignore, other.ignore) && samePreparationPaths(c.instructions, other.instructions) && samePreparationLocations(c.runtime, other.runtime)
}

func samePreparationLocations(left, right []SourceLocation) bool {
	if len(left) != len(right) {
		return false
	}
	for _, location := range left {
		if !slices.Contains(right, location) {
			return false
		}
	}
	return true
}

func samePreparationPaths(left, right []snapshot.Path) bool {
	if len(left) != len(right) {
		return false
	}
	for _, path := range left {
		if !slices.Contains(right, path) {
			return false
		}
	}
	return true
}
