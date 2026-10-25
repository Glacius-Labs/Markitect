package project

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// RegistryDiagnostic is a source-bound observation, not a run Finding or ID.
// An empty Source denotes a stage-level problem; Position is optional.
type RegistryDiagnostic struct {
	Code     string
	Source   snapshot.Path
	Position *model.Position
	Message  string
}

// RegistryRecord retains decoded observations even beside invalid records.
// Original malformed values remain in the bound model stage's Raw.Registry.
// These records are not a prepared registry or artifact authority.
type RegistryRecord struct {
	Source   snapshot.Path
	Position model.Position
	Owner    realization.Owner
	Path     string
	Form     string
	Realizes []string
	Why      string
}

type registryPreparation struct {
	status      model.Status
	registry    realization.Registry
	discovered  []snapshot.Path
	orphaned    []snapshot.Path
	records     []RegistryRecord
	diagnostics []RegistryDiagnostic
}

// Passed describes registry selection, decoding and structural records only,
// not complete file accounting, semantic realization or readiness to Apply.
// Missing observations or a non-passed model give NotCompleted; known invalid
// registry data give Failed, including beside missing observations. Zero is
// NotCompleted. Registry never returns a usable partial value on refusal.
func (r registryPreparation) Status() model.Status {
	if r.status == "" {
		return model.NotCompleted
	}
	return r.status
}

func (r registryPreparation) Registry() (realization.Registry, bool) {
	if r.Status() != model.Passed {
		return realization.Registry{}, false
	}
	return r.registry, true
}

func (r registryPreparation) Discovered() []snapshot.Path { return slices.Clone(r.discovered) }

// Orphaned lists the shards of Packages the checked model does not have, such
// as one the owner removed or merged into another. They are realization still
// to do: accounting hands their registrations to the area owners, and Root
// deletes the shard (StaleLinks).
func (r registryPreparation) Orphaned() []snapshot.Path { return slices.Clone(r.orphaned) }
func (r registryPreparation) Records() []RegistryRecord {
	result := slices.Clone(r.records)
	for i := range result {
		result[i].Realizes = slices.Clone(result[i].Realizes)
	}
	return result
}
func (r registryPreparation) Diagnostics() []RegistryDiagnostic {
	result := slices.Clone(r.diagnostics)
	for i := range result {
		if result[i].Position != nil {
			position := *result[i].Position
			result[i].Position = &position
		}
	}
	return result
}

// CapturedRegistryPreparation retains its exact committed model-stage binding.
type CapturedRegistryPreparation struct {
	registryPreparation
	stage CapturedModel
}

func (r CapturedRegistryPreparation) Model() CapturedModel { return r.stage }

// PreviewRegistryPreparation retains its draft/fixed preview and has no guard.
type PreviewRegistryPreparation struct {
	registryPreparation
	stage PreviewedModel
}

func (r PreviewRegistryPreparation) Model() PreviewedModel { return r.stage }

func (m CapturedModel) PrepareRegistry() CapturedRegistryPreparation {
	return CapturedRegistryPreparation{registryPreparation: prepareRegistry(m.checked, m.inputs.Raw), stage: m}
}
func (m PreviewedModel) PrepareRegistry() PreviewRegistryPreparation {
	return PreviewRegistryPreparation{registryPreparation: prepareRegistry(m.checked, m.inputs.Raw), stage: m}
}

func (r *registryPreparation) add(status model.Status, code string, source snapshot.Path, position *model.Position, message string) {
	r.diagnostics = append(r.diagnostics, RegistryDiagnostic{Code: code, Source: source, Position: position, Message: message})
	if status == model.Failed || r.status != model.Failed {
		r.status = status
	}
}
