// Package projectapp provides a small application facade over the existing
// projectrun services. It keeps root selection and the configured execution
// ports explicit without adding a workflow engine or transport layer.
package projectapp

import (
	"context"
	"errors"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
)

// Operations composes the project application services with the selected
// project Host and provider Invoker.
type Operations struct {
	Host    projectrun.Host
	Invoker projectrun.Invoker
}

// Selection identifies an explicit project root and optional source revision.
// Revision is passed through unchanged to the selected project Host.
type Selection struct {
	Root     string `json:"root"`
	Revision string `json:"revision"`
}

type PlanOperation struct {
	Selection Selection              `json:"selection"`
	Request   projectrun.PlanRequest `json:"request"`
	// ExpectedPreviewDigest binds persisting to the reviewed preview's
	// projectrun.PlanPreviewDigest.
	ExpectedPreviewDigest string `json:"expectedPreviewDigest,omitempty"`
}

type RunOperation struct {
	Root  string `json:"root"`
	RunID string `json:"runId"`
}

type ApplyOperation struct {
	Root    string                  `json:"root"`
	Request projectrun.ApplyRequest `json:"request"`
}

type PreflightOperation struct {
	Root        string `json:"root"`
	RunID       string `json:"runId"`
	CandidateID string `json:"candidateId"`
}

type FullVerifyOperation struct {
	Root    string                       `json:"root"`
	Request projectrun.FullVerifyRequest `json:"request"`
}

type DeliverOperation struct {
	Root    string                    `json:"root"`
	Request projectrun.DeliverRequest `json:"request"`
}

func (o Operations) Plan(operation PlanOperation) (projectrun.PlanRecord, error) {
	if err := requireRoot(operation.Selection.Root); err != nil {
		return projectrun.PlanRecord{}, err
	}
	if operation.Selection.Revision != "" && operation.Request.BaseRevision != "" &&
		operation.Selection.Revision != operation.Request.BaseRevision {
		return projectrun.PlanRecord{}, errors.New("selected revision conflicts with plan request baseRevision")
	}
	// projectrun.Plan uses Request.BaseRevision when set and otherwise falls
	// back to Selection.Revision. The conflict check above prevents that
	// existing precedence from silently hiding a second explicit selection.
	return projectrun.PlanExpecting(o.Host, operation.Selection.Root, operation.Selection.Revision, operation.Request, operation.ExpectedPreviewDigest)
}

func (o Operations) Run(ctx context.Context, operation RunOperation) (projectrun.RunReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.RunReport{}, err
	}
	return projectrun.Run(ctx, o.Host, o.Invoker, operation.Root, operation.RunID)
}

func (o Operations) Resume(ctx context.Context, operation RunOperation) (projectrun.RunReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.RunReport{}, err
	}
	return projectrun.Resume(ctx, o.Host, o.Invoker, operation.Root, operation.RunID)
}

func (o Operations) Repair(ctx context.Context, operation RunOperation) (projectrun.RunReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.RunReport{}, err
	}
	return projectrun.Repair(ctx, o.Host, o.Invoker, operation.Root, operation.RunID)
}

func (o Operations) Status(operation RunOperation) (projectrun.StatusReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.StatusReport{}, err
	}
	return projectrun.Status(operation.Root, operation.RunID)
}

func (o Operations) Verify(ctx context.Context, operation RunOperation) (projectrun.VerifyReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.VerifyReport{}, err
	}
	return projectrun.Verify(ctx, o.Host, o.Invoker, operation.Root, operation.RunID)
}

func (o Operations) FullVerify(ctx context.Context, operation FullVerifyOperation) (projectrun.FullVerifyReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.FullVerifyReport{}, err
	}
	return projectrun.FullVerify(ctx, o.Host, o.Invoker, operation.Root, operation.Request)
}

func (o Operations) PreflightApply(operation PreflightOperation) (projectrun.ApplyPreflight, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.ApplyPreflight{}, err
	}
	return projectrun.PreflightApply(o.Host, operation.Root, operation.RunID, operation.CandidateID)
}

func (o Operations) Apply(operation ApplyOperation) (projectrun.ApplyReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.ApplyReport{}, err
	}
	return projectrun.Apply(o.Host, o.Invoker, operation.Root, operation.Request)
}

func (o Operations) Deliver(ctx context.Context, operation DeliverOperation) (projectrun.DeliverReport, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectrun.DeliverReport{}, err
	}
	return projectrun.Deliver(ctx, o.Host, o.Invoker, operation.Root, operation.Request)
}

func requireRoot(root string) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("selected project root is required")
	}
	return nil
}
