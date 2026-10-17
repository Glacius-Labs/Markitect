package apply

import (
	"context"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

var (
	ErrSourceUnavailable = errors.New("model preparation source unavailable")
	ErrIncompleteInputs  = errors.New("model preparation inputs incomplete or mismatched")
)

// ModelService acquires and compiles model bytes only. Passed means language
// validation on the supplied bytes; it is not a complete Check, Context or Apply.
type ModelService struct {
	source  Source
	preview PreviewSource
}

func NewModelService(source Source, preview PreviewSource) *ModelService {
	return &ModelService{source: source, preview: preview}
}

// CapturedModel retains a fixed committed acquisition and compiler result.
// Accessors return copies. The zero value has no successful compilation.
type CapturedModel struct {
	request  CaptureRequest
	inputs   CapturedInputs
	compiled model.Result
	limits   model.Limits
}

func (m CapturedModel) Request() CaptureRequest {
	return CaptureRequest{cloneSelection(m.request.Selection)}
}
func (m CapturedModel) Inputs() CapturedInputs {
	result := m.inputs
	result.Raw = cloneRaw(result.Raw)
	return result
}
func (m CapturedModel) Compilation() model.Result     { return cloneCompilation(m.compiled) }
func (m CapturedModel) RequestedLimits() model.Limits { return m.limits }

// PreviewedModel intentionally exposes no publication guard or Capture result.
type PreviewedModel struct {
	request  PreviewRequest
	inputs   PreviewedInputs
	compiled model.Result
	limits   model.Limits
}

func (m PreviewedModel) Request() PreviewRequest {
	result := m.request
	result.Selection = cloneSelection(result.Selection)
	return result
}
func (m PreviewedModel) Inputs() PreviewedInputs {
	result := m.inputs
	result.Raw = cloneRaw(result.Raw)
	return result
}
func (m PreviewedModel) Compilation() model.Result     { return cloneCompilation(m.compiled) }
func (m PreviewedModel) RequestedLimits() model.Limits { return m.limits }

// CaptureModel returns acquisition errors separately from compiler diagnostics.
// An errored or incomplete acquisition is never compiled into a passing model;
// its actual partial observations and selected request remain available.
func (s *ModelService) CaptureModel(ctx context.Context, request CaptureRequest, limits model.Limits) (CapturedModel, error) {
	result := CapturedModel{request: CaptureRequest{cloneSelection(request.Selection)}, compiled: model.Result{Status: model.NotCompleted}, limits: limits}
	if s == nil || s.source == nil {
		return result, ErrSourceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	observed, err := s.source.Capture(ctx, CaptureRequest{cloneSelection(result.request.Selection)})
	observed.Raw = cloneRaw(observed.Raw)
	result.inputs = observed
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if observed.Head == "" || observed.Guard == "" {
		return result, fmt.Errorf("%w: committed HEAD and guard required", ErrIncompleteInputs)
	}
	if err := validateSelection(result.request.Selection, observed.Raw); err != nil {
		return result, err
	}
	result.compiled = compileModel(observed.Raw, limits)
	return result, nil
}

func (s *ModelService) PreviewModel(ctx context.Context, request PreviewRequest, limits model.Limits) (PreviewedModel, error) {
	request.Selection = cloneSelection(request.Selection)
	result := PreviewedModel{request: request, compiled: model.Result{Status: model.NotCompleted}, limits: limits}
	if s == nil || s.preview == nil {
		return result, ErrSourceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	portRequest := request
	portRequest.Selection = cloneSelection(request.Selection)
	observed, err := s.preview.CapturePreview(ctx, portRequest)
	observed.Raw = cloneRaw(observed.Raw)
	result.inputs = observed
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if observed.Head == "" || request.Revision != observed.Revision {
		return result, fmt.Errorf("%w: preview HEAD and revision selection required", ErrIncompleteInputs)
	}
	if err := validateSelection(result.request.Selection, observed.Raw); err != nil {
		return result, err
	}
	result.compiled = compileModel(observed.Raw, limits)
	return result, nil
}

func compileModel(raw RawInputs, limits model.Limits) model.Result {
	input := model.Input{Root: raw.Local.Source, Limits: limits}
	for _, dependency := range raw.Dependencies {
		input.Dependencies = append(input.Dependencies, dependency.Source)
	}
	return model.Compile(input)
}

// These are completeness/binding checks, not physical acquisition proof.
func validateSelection(selection InputSelection, raw RawInputs) error {
	invalid := func(detail string) error { return fmt.Errorf("%w: %s", ErrIncompleteInputs, detail) }
	if selection.Repository == "" || raw.LocalModelArea.String() == "" {
		return invalid("repository and local acquisition area required")
	}
	checkModel := func(want ModelSelection, actual ModelSource) bool {
		return want.Boundary != "" && want.ManifestPath.String() != "" && want == actual.Selection &&
			actual.Source.ManifestPath == want.ManifestPath.String() && actual.Root.String() != ""
	}
	if !checkModel(selection.Local, raw.Local) || len(selection.Dependencies) != len(raw.Dependencies) {
		return invalid("selected local/dependency model observations required")
	}
	if err := validateModelFiles(raw.Local); err != nil {
		return err
	}
	for _, file := range raw.Local.Files.Entries() {
		p := file.Path().String()
		if !raw.LocalModelArea.IsRoot() {
			p = raw.LocalModelArea.String() + "/" + p
		}
		if !sameCensusFile(raw, p, file) {
			return invalid("local model files differ from repository census")
		}
	}
	for i, dependency := range selection.Dependencies {
		if !checkModel(dependency, raw.Dependencies[i]) {
			return invalid("dependency acquisition coordinates differ from selection")
		}
		if err := validateModelFiles(raw.Dependencies[i]); err != nil {
			return err
		}
	}
	for _, group := range []struct {
		selected        []SourceLocation
		actual          []SourceFile
		repositoryLocal bool
	}{
		{selection.Execution, raw.Execution, true}, {selection.Registry, raw.Registry, true}, {selection.Ignore, raw.Ignore, true},
		{selection.HostProfiles, raw.HostProfiles, false}, {selection.Instructions, raw.Instructions, false},
	} {
		if len(group.selected) != len(group.actual) {
			return invalid("selected input observations missing")
		}
		for i, location := range group.selected {
			if location.Boundary == "" || location.Path.String() == "" || group.actual[i].Location != location || group.actual[i].File.Path() != location.Path {
				return invalid("input acquisition coordinates differ from selection")
			}
			if group.repositoryLocal && group.actual[i].RepositoryPath.String() == "" {
				return invalid("execution, registry and ignore inputs require repository coordinates")
			}
			if file := group.actual[i]; file.RepositoryPath.String() != "" && !sameCensusFile(raw, file.RepositoryPath.String(), file.File) {
				return invalid("local input differs from repository census")
			}
		}
	}
	return nil
}
