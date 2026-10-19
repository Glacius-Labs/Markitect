package apply

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func cloneSelection(value InputSelection) InputSelection {
	value.Dependencies = slices.Clone(value.Dependencies)
	value.Execution = slices.Clone(value.Execution)
	value.Registry = slices.Clone(value.Registry)
	value.Ignore = slices.Clone(value.Ignore)
	value.HostProfiles = slices.Clone(value.HostProfiles)
	value.Instructions = slices.Clone(value.Instructions)
	return value
}

func cloneSource(value model.SourceModel) model.SourceModel {
	value.Entries = slices.Clone(value.Entries)
	for i := range value.Entries {
		value.Entries[i].Bytes = slices.Clone(value.Entries[i].Bytes)
	}
	return value
}

func cloneRaw(value RawInputs) RawInputs {
	value.Untracked = slices.Clone(value.Untracked)
	value.Local.Source = cloneSource(value.Local.Source)
	value.Dependencies = slices.Clone(value.Dependencies)
	for i := range value.Dependencies {
		value.Dependencies[i].Source = cloneSource(value.Dependencies[i].Source)
	}
	value.Execution = slices.Clone(value.Execution)
	value.Registry = slices.Clone(value.Registry)
	value.Ignore = slices.Clone(value.Ignore)
	value.HostProfiles = slices.Clone(value.HostProfiles)
	value.Instructions = slices.Clone(value.Instructions)
	return value
}

// Diagnostics contain optional mutable source positions.
func cloneDiagnostics(value []model.Diagnostic) []model.Diagnostic {
	value = slices.Clone(value)
	for i := range value {
		if value[i].Position != nil {
			position := *value[i].Position
			value[i].Position = &position
		}
	}
	return value
}

// model.Value and snapshot values are immutable. Compiler report containers
// and optional spans/titles remain mutable public Go values.
func cloneCompilation(value model.Result) model.Result {
	value.Diagnostics = cloneDiagnostics(value.Diagnostics)
	value.Models = slices.Clone(value.Models)
	for i := range value.Models {
		m := &value.Models[i]
		m.Phases = slices.Clone(m.Phases)
		m.Packages = slices.Clone(m.Packages)
		m.Canonical = slices.Clone(m.Canonical)
		m.Definitions = slices.Clone(m.Definitions)
		for j := range m.Definitions {
			d := &m.Definitions[j]
			if d.Title != nil {
				title := *d.Title
				d.Title = &title
			}
			d.Spec = slices.Clone(d.Spec)
			d.BodyLinks = slices.Clone(d.BodyLinks)
			for k := range d.BodyLinks {
				link := &d.BodyLinks[k]
				if link.Display != nil {
					display := *link.Display
					link.Display = &display
				}
				if link.Coordinates != nil {
					coordinates := *link.Coordinates
					link.Coordinates = &coordinates
					if coordinates.Body != nil {
						body := *coordinates.Body
						link.Coordinates.Body = &body
					}
					if coordinates.Source != nil {
						source := *coordinates.Source
						link.Coordinates.Source = &source
					}
				}
			}
		}
	}
	return value
}
