// Package gitrepo acquires explicit repository sources and publishes guarded
// working-tree changes.
// Relative acquisition boundaries are relative to the selected repository;
// model paths remain exact, manifest-relative language values. No discovery,
// Model check or provider operation is performed here. Workspace creation has its
// own adapter contract; source capture remains read-only.
package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

var (
	ErrLimit           = errors.New("source acquisition limit")
	ErrChanged         = errors.New("source changed during acquisition")
	ErrUnsafe          = errors.New("unsafe source acquisition")
	ErrDirty           = errors.New("committed source differs from working repository")
	ErrUnrepresentable = errors.New("working model directory has no revision representation")
)

// Limits are acquisition budgets, not language maxima. Zero fields select
// DefaultLimits. Manifest is passed unchanged to model.InspectManifest, whose
// actual failure or incomplete result is retained in ManifestPreflightError.
// Entries bounds distinct acquired files and model directories across all
// boundaries. The same location in one Git commit or on disk counts once;
// Git and disk are separate observations. Each inventory scan is also bounded.
type Limits struct {
	Entries               int
	FileBytes, TotalBytes int64
	Timeout               time.Duration
	Manifest              model.Limits
}

func DefaultLimits() Limits {
	return Limits{Entries: 100000, FileBytes: 64 << 20, TotalBytes: 512 << 20, Timeout: 2 * time.Minute}
}

// Repository retains capture bindings and opaque publication permits in memory.
// Each call owns its handles and Git processes. Construction performs no I/O.
type Repository struct {
	limits Limits
	// A private deterministic scheduling seam for acquisition/publication races.
	observe func(event, path string)
}

func New(limits Limits) (*Repository, error) {
	defaults := DefaultLimits()
	if limits.Entries < 0 || limits.FileBytes < 0 || limits.TotalBytes < 0 || limits.Timeout < 0 {
		return nil, fmt.Errorf("%w: negative acquisition budget", ErrLimit)
	}
	if limits.Entries == 0 {
		limits.Entries = defaults.Entries
	}
	if limits.FileBytes == 0 {
		limits.FileBytes = defaults.FileBytes
	}
	if limits.TotalBytes == 0 {
		limits.TotalBytes = defaults.TotalBytes
	}
	if limits.Timeout == 0 {
		limits.Timeout = defaults.Timeout
	}
	if limits.FileBytes > limits.TotalBytes {
		return nil, fmt.Errorf("%w: file budget exceeds total", ErrLimit)
	}
	return &Repository{limits: limits}, nil
}

var _ project.Source = (*Repository)(nil)
var _ project.PreviewSource = (*Repository)(nil)

func (r *Repository) Capture(ctx context.Context, request project.CaptureRequest) (project.CapturedInputs, error) {
	a, err := r.acquire(ctx, request.Selection, "", true)
	if a == nil {
		return project.CapturedInputs{}, err
	}
	result := project.CapturedInputs{Raw: a.raw, Head: a.head}
	if err == nil {
		result.Guard = a.guard()
	}
	return result, err
}

func (r *Repository) CapturePreview(ctx context.Context, request project.PreviewRequest) (project.PreviewedInputs, error) {
	a, err := r.acquire(ctx, request.Selection, request.Revision, false)
	if a == nil {
		return project.PreviewedInputs{}, err
	}
	return project.PreviewedInputs{Raw: a.raw, Head: a.head, Revision: a.revision}, err
}
