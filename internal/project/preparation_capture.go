package project

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func (s *PreparationService) capturePreparation(ctx context.Context, request CaptureRequest, limits model.Limits) (CapturedPreparation, error) {
	r := CapturedPreparation{request: CaptureRequest{cloneSelection(request.Selection)}}
	selection, err := initialPreparationSelection(request.Selection)
	if err != nil {
		r.status = model.Failed
		return r, err
	}
	var first preparationClosure
	for index := 0; index < 2; index++ {
		var models *ModelService
		if s != nil {
			models = s.models
		}
		stage, err := models.CaptureModel(ctx, CaptureRequest{selection}, limits)
		registry, execution := stage.PrepareRegistry(), stage.DecodeExecution()
		structural := CapturedPreparationAttempt{stage: stage, registry: registry, files: registry.CheckFiles(), execution: execution, contexts: execution.SelectContexts(), err: err}
		attempt := structural
		r.status = structuralPreparationStatus(stage.checked.Status, registry.Status(), structural.files.Status(), execution.Status(), structural.contexts.Status())
		if err != nil {
			r.attempts = append(r.attempts, attempt)
			r.status = acquisitionPreparationStatus(err)
			return r, err
		}
		closure := observedPreparationClosure(stage.inputs.Raw, registry.Discovered(), execution.Partial())
		r.attempts = append(r.attempts, attempt)
		if err != nil {
			r.status = incompletePreparationStatus(r.status)
			if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return r, err
			}
		}
		if index == 1 && !closure.equal(first) {
			r.status = incompletePreparationStatus(r.status)
			return r, preparationClosureError(err)
		}
		next, missing := closure.selectMissing(selection, stage.inputs.Raw)
		if !missing {
			return r, err
		}
		if index == 1 {
			r.status = incompletePreparationStatus(r.status)
			return r, preparationClosureError(err)
		}
		first, selection = closure, next
	}
	return r, ErrPreparationClosure
}

func (s *PreparationService) previewPreparation(ctx context.Context, request PreviewRequest, limits model.Limits) (PreviewPreparation, error) {
	request.Selection = cloneSelection(request.Selection)
	r := PreviewPreparation{request: request}
	selection, err := initialPreparationSelection(request.Selection)
	if err != nil {
		r.status = model.Failed
		return r, err
	}
	var first preparationClosure
	for index := 0; index < 2; index++ {
		var models *ModelService
		if s != nil {
			models = s.models
		}
		stage, err := models.PreviewModel(ctx, PreviewRequest{Selection: selection, Revision: request.Revision}, limits)
		registry, execution := stage.PrepareRegistry(), stage.DecodeExecution()
		structural := PreviewPreparationAttempt{stage: stage, registry: registry, files: registry.CheckFiles(), execution: execution, contexts: execution.SelectContexts(), err: err}
		attempt := structural
		r.status = structuralPreparationStatus(stage.checked.Status, registry.Status(), structural.files.Status(), execution.Status(), structural.contexts.Status())
		if err != nil {
			r.attempts = append(r.attempts, attempt)
			r.status = acquisitionPreparationStatus(err)
			return r, err
		}
		closure := observedPreparationClosure(stage.inputs.Raw, registry.Discovered(), execution.Partial())
		r.attempts = append(r.attempts, attempt)
		if err != nil {
			r.status = incompletePreparationStatus(r.status)
			if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return r, err
			}
		}
		if index == 1 && !closure.equal(first) {
			r.status = incompletePreparationStatus(r.status)
			return r, preparationClosureError(err)
		}
		next, missing := closure.selectMissing(selection, stage.inputs.Raw)
		if !missing {
			return r, err
		}
		if index == 1 {
			r.status = incompletePreparationStatus(r.status)
			return r, preparationClosureError(err)
		}
		first, selection = closure, next
	}
	return r, ErrPreparationClosure
}

func incompletePreparationStatus(status model.Status) model.Status {
	if status == model.Failed {
		return status
	}
	return model.NotCompleted
}

func preparationClosureError(metadata error) error {
	if metadata == nil {
		return ErrPreparationClosure
	}
	return errors.Join(ErrPreparationClosure, metadata)
}
