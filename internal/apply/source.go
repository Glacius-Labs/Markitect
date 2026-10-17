// Package apply prepares fixed inputs for artifact reconciliation. This first
// model stage performs no repository I/O, execution or publication itself.
package apply

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Source captures the current committed, clean HEAD. Before returning without
// error, the adapter must read exactly spelled manifests, locks and declared
// roots without following redirects, establish physically distinct, non-nested
// model roots, and capture every selected input and the complete census.
// Dirty tracked files and non-ignored untracked files refuse Capture. Known
// observations must be returned even on error. Compilation proves none of
// these acquisition facts. One repository adapter implements both source ports
// using the same acquisition mechanism.
type Source interface {
	Capture(context.Context, CaptureRequest) (CapturedInputs, error)
}

// PreviewSource observes a working-tree draft or an explicitly selected fixed
// revision. It has the same acquisition obligations, but grants no publication
// guard. A preview cannot be promoted to a Capture result by the model service.
type PreviewSource interface {
	CapturePreview(context.Context, PreviewRequest) (PreviewedInputs, error)
}

// ModelSelection names an explicit acquisition boundary and its relative
// manifest. Boundary is an adapter location, never inferred from Origin.
type ModelSelection struct {
	Boundary     string
	ManifestPath snapshot.Path
}

// SourceLocation names an explicit input within an acquisition boundary.
// Host profiles and instructions may be outside the repository census.
type SourceLocation struct {
	Boundary string
	Path     snapshot.Path
}

// InputSelection retains caller order and duplicates for adapter validation.
// It selects sources, not decoded settings or provider wire configuration.
type InputSelection struct {
	Repository   string
	Local        ModelSelection
	Dependencies []ModelSelection
	Execution    []SourceLocation
	Registry     []SourceLocation
	Ignore       []SourceLocation
	HostProfiles []SourceLocation
	Instructions []SourceLocation
}

type CaptureRequest struct{ Selection InputSelection }

// An empty Revision explicitly selects the working tree. Otherwise the adapter
// resolves the selected revision once and records its fixed identity.
type PreviewRequest struct {
	Selection InputSelection
	Revision  string
}

// PublicationGuard is an opaque comparison token issued by Source.Capture.
// Its spelling has no application meaning. It grants neither write permission
// nor ownership; later publication needs its own authority and source recheck.
type PublicationGuard string

// ModelSource retains the actual complete model acquisition, including empty
// directories and duplicate entry paths. Files retains original file modes
// and bytes. Root is the declared lexical root within this boundary, not an
// independently authored Package inventory or proof of physical separation.
type ModelSource struct {
	Selection ModelSelection
	Root      snapshot.DirectoryBoundary
	Source    model.SourceModel
	Files     snapshot.Tree
}

// SourceFile retains the exact selected location, path, mode and original bytes.
type SourceFile struct {
	Location SourceLocation
	File     snapshot.Entry
}

// RawInputs is shared by committed capture and read-only preview. Census is the
// complete immutable repository tree, with the contract's state exclusion.
// LocalModelArea locates the local model acquisition boundary in that tree;
// model entry paths are relative to that area. Other input coordinates remain
// explicit. No classification or decoding is inferred from these bytes.
type RawInputs struct {
	Census         snapshot.Tree
	LocalModelArea snapshot.DirectoryBoundary
	Local          ModelSource
	Dependencies   []ModelSource
	Execution      []SourceFile
	Registry       []SourceFile
	Ignore         []SourceFile
	HostProfiles   []SourceFile
	Instructions   []SourceFile
}

// CapturedInputs binds Raw to the current committed HEAD and its comparison
// guard. On error any known fields remain useful partial observations.
type CapturedInputs struct {
	Raw   RawInputs
	Head  string
	Guard PublicationGuard
}

// PreviewedInputs binds a draft to its observed HEAD, or a fixed preview to its
// resolved Revision. Revision is empty only for a working-tree preview. Neither
// identity authenticates acquisition, owner approval or permission to write.
type PreviewedInputs struct {
	Raw      RawInputs
	Head     string
	Revision string
}
