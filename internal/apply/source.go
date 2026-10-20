// Package apply prepares fixed inputs for artifact reconciliation. This first
// model stage performs no repository I/O, execution or publication itself.
package apply

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Source captures repository-local model, execution, registry and ignore sources
// from the current committed, clean HEAD. Explicitly external dependencies and
// Host sources are current, exactly bound observations, not contents of HEAD.
// The complete actual repository census also includes Git-ignored untracked
// files, subject only to RawInputs' census exclusions. Gitignore exempts untracked
// files only from the clean-HEAD check, never from census observation.
// Before returning without error, the adapter must read exactly spelled
// manifests, locks and declared
// roots without following redirects, establish physically distinct, non-nested
// model roots, and capture every selected input and the complete census.
// Dirty tracked files and non-ignored untracked files refuse Capture. Known
// observations must be returned even on error. Model check proves none of
// these acquisition facts. One repository adapter implements both source ports
// using the same acquisition mechanism.
// If InspectManifest supplies no Root, return a ManifestPreflightError carrying
// its actual result and the known partial inputs, before capturing Definitions.
// A committed capture refuses model directories that exist only in the working
// tree and cannot be represented by the selected fixed revision.
type Source interface {
	Capture(context.Context, CaptureRequest) (CapturedInputs, error)
}

// PreviewSource observes a working-tree draft or an explicitly selected fixed
// revision. It has the same acquisition obligations, but grants no publication
// guard. A preview cannot be promoted to a Capture result by the model service.
// Working preview observes the actual tree, including empty model directories;
// historical preview observes exactly the selected Git tree, never today's files.
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
// Execution, Registry and Ignore select repository-local committed inputs
// (or their explicit preview counterparts); their observations must carry
// RepositoryPath. HostProfiles and Instructions also allow external sources.
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

// An empty Revision explicitly selects the working tree. Otherwise Revision
// must already name the explicitly resolved fixed revision; the adapter must
// return that exact identity, never substitute a symbolic selector's new value.
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
// and bytes. Root is relative to the acquisition boundary: the manifest directory
// followed by InspectManifest.Root's original declaration ('.' keeps that
// directory), without silently normalizing paths or following redirects. It is
// not a second Package inventory or proof of physical separation.
// RepositoryArea locates this acquisition boundary in Census; every Files path
// is relative to that area. Local must have an area, as must every in-repository
// dependency. Only an external dependency leaves it zero. The adapter establishes
// that membership; Apply never infers it from Selection.Boundary or Origin.
type ModelSource struct {
	Selection      ModelSelection
	Root           snapshot.DirectoryBoundary
	RepositoryArea snapshot.DirectoryBoundary
	Source         model.SourceModel
	Files          snapshot.Tree
}

// SourceFile retains the exact selected location, path, mode and original bytes.
// RepositoryPath is required for a repository-local input and absent only for
// external inputs, including safe Host sources. It supplies protection/accounting
// coordinates without interpreting an opaque boundary or provenance string.
type SourceFile struct {
	Location       SourceLocation
	File           snapshot.Entry
	RepositoryPath snapshot.Path
}

// RawInputs is shared by committed capture and read-only preview. Census is the
// complete immutable actual repository observation for Capture and Working
// preview, including Git-ignored untracked files. Fixed preview retains exactly
// its historical Git tree. Census excludes exactly .markitect/state/** and the
// technical Git administration; Gitignore supplies no other scan exclusion.
// Each repository-local ModelSource supplies its own acquisition area. Other
// input coordinates remain explicit. No classification or decoding is inferred
// from these bytes. Untracked lists, for Capture only, the Census paths that the
// HEAD tree does not hold: exactly the Git-ignored untracked files. Previews
// leave it empty. Only committed files can be drift that a Manager repairs
// (DEC-062).
type RawInputs struct {
	Census       snapshot.Tree
	Untracked    []snapshot.Path
	Local        ModelSource
	Dependencies []ModelSource
	Execution    []SourceFile
	Registry     []SourceFile
	Ignore       []SourceFile
	HostProfiles []SourceFile
	Instructions []SourceFile
}

// CapturedInputs binds repository-local committed sources to HEAD; Raw also
// retains the external source observations and actual census under the comparison
// guard. On error known fields remain useful partial observations.
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
