// Package apply composes checked capture, closed Runs and durable publication.
package apply

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/workflow"
)

// ErrUnavailable is retained only for the bootstrap transition in this slice.
var ErrUnavailable = errors.New("Apply unavailable until concrete native composition is configured")

type Candidates interface {
	Capture(context.Context, string, project.Prepared) (project.CandidateRef, error)
}
type Records interface {
	Read(context.Context, string, string) (run.Result, error)
}
type PublicationRequest struct {
	Prepared project.Prepared
	Approval run.Result
}
type Publication struct {
	Candidate      project.CandidateRef   `json:"candidate"`
	NoChange       bool                   `json:"noChange"`
	CompletedPaths []string               `json:"completedPaths"`
	Checks         []run.CheckObservation `json:"checks"`
	Readback       bool                   `json:"readback"`
}
type Publisher interface {
	Publish(context.Context, PublicationRequest) (Publication, error)
}
type Command struct {
	Executor   workflow.Executor
	Candidates Candidates
	Records    Records
	Publisher  Publisher
}
type Report struct {
	Status      model.Status    `json:"status"`
	Workflow    workflow.Result `json:"workflow"`
	Publication *Publication    `json:"publication,omitempty"`
}

func ValidateApproval(p project.Prepared, value run.Result) error {
	if value.Mandate != run.Verify || value.Decision != run.Approve {
		return fmt.Errorf("publication requires final Verify approval")
	}
	return workflow.ValidateClosed(value, run.Request{ID: value.ID, Project: value.Project, Prepared: p, Input: value.Input, Mandate: run.Verify})
}

func (c *Command) Execute(ctx context.Context, request run.Request) (out Report, err error) {
	out.Status = model.NotCompleted
	if c == nil || c.Candidates == nil || c.Executor == nil || c.Records == nil || c.Publisher == nil || !request.Prepared.Valid() {
		return out, fmt.Errorf("Apply requires checked preparation and concrete ports")
	}
	request.Input, err = c.Candidates.Capture(ctx, request.Project, request.Prepared)
	if err != nil {
		return out, err
	}
	out.Workflow, err = workflow.Apply(ctx, c.Executor, request)
	if err != nil {
		return out, err
	}
	final := out.Workflow.Final
	if final.Decision != run.Approve || final.Mandate != run.Verify {
		out.Status = model.Failed
		return out, nil
	}
	// All Execute calls have returned after closure. Reopen acceptance through
	// persistent storage; neither a live handle nor an Implement judgment suffices.
	stored, err := c.Records.Read(ctx, request.Project, final.ID)
	if err != nil {
		return out, fmt.Errorf("stored final Verify acceptance unavailable: %w", err)
	}
	if !reflect.DeepEqual(stored, final) {
		return out, fmt.Errorf("stored final Verify acceptance changed")
	}
	if err = ValidateApproval(request.Prepared, stored); err != nil {
		return out, err
	}
	published, err := c.Publisher.Publish(ctx, PublicationRequest{request.Prepared, stored})
	out.Publication = &published
	if err == nil && published.Readback {
		out.Status = model.Passed
	} else if err == nil {
		err = fmt.Errorf("publication readback not confirmed")
	}
	return out, err
}
