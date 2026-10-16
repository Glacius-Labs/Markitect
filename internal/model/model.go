// Package model compiles explicitly supplied canonical model sources.
// It performs no filesystem, repository, process or provider operations.
//
// Compile validates canonical v1 content, the entry lock, and model-digest/1
// bytes. Exact body-link coordinate coverage is a separate tool profile.
package model

// Identity is a canonical Definition address. Model IDs and source filenames
// are provenance, not additional components of a Definition's identity.
type Identity struct {
	Package string
	ID      string
}

// EntryKind describes an acquired source entry. Acquisition must report
// redirects rather than follow them. Implicit parent directories are allowed;
// explicit Directory entries retain otherwise empty directories.
type EntryKind string

const (
	Regular   EntryKind = "regular"
	Directory EntryKind = "directory"
	Redirect  EntryKind = "redirect"
)

// Entry is relative to the supplied model source's acquisition boundary.
// Path uses '/' without normalization; Bytes is meaningful only for Regular.
type Entry struct {
	Path  string
	Kind  EntryKind
	Bytes []byte
}

// SourceModel supplies a complete acquired tree, not a prefiltered list of
// Definition files. Origin is an opaque provenance label. ManifestPath names
// the explicitly selected model.yaml within Entries. Entries remain a slice so
// duplicate paths cannot disappear before validation.
// Each SourceModel is a separate acquired boundary. Before Definition capture,
// the caller must establish physically distinct, non-nested model roots and
// complete, exactly spelled directories/files. Forbidden redirects in the
// relative root path, model tree, manifest and lock must be reported without
// following them. Compile checks the supplied relative paths/entries;
// opaque origins are not physical-root evidence and are never interpreted as
// filesystem paths.
type SourceModel struct {
	Origin       string
	ManifestPath string
	Entries      []Entry
}

// Input supplies the local model and every available dependency as bytes.
// Supplying a dependency does not grant import visibility or select a version.
type Input struct {
	Root         SourceModel
	Dependencies []SourceModel
	Limits       Limits
}

// Limits are tool budgets, not language maxima. Zero selects the documented
// default. Negative values are invalid configuration. A limit refusal is
// not_completed, never evidence that a model violates the language.
type Limits struct {
	SourceBytes     int64
	Entries         int
	YAMLDepth       int
	Models          int
	BodyBytes       int
	MarkdownNesting int
}

// DefaultLimits returns the finite budgets used for zero-valued limit fields.
// SourceBytes and Entries cover the entire supplied input, including metadata
// and files outside a declared model root. Models includes the local model.
// YAMLDepth counts the root as one. BodyBytes bounds each decoded original
// UTF-8 body before Markdown preprocessing. MarkdownNesting bounds actual block
// depth (Document excluded), pending Goldmark bracket opener calls per inline
// block, and constructed AST depth. These are tool refusals, not language maxima.
func DefaultLimits() Limits {
	return Limits{SourceBytes: 64 << 20, Entries: 100000, YAMLDepth: 128, Models: 1024, BodyBytes: 2_000_000, MarkdownNesting: 128}
}

type Status string

const (
	Passed       Status = "passed"
	Failed       Status = "failed"
	NotCompleted Status = "not_completed"
)

// Source binds diagnostics and IR to an exact acquired path and opaque Origin.
// For regular files, Digest hashes the original bytes; it is not a canonical
// model content digest. Directory provenance has an empty Digest. A directory
// path may be implicit in a supplied descendant; it is not physical-root proof.
type Source struct {
	Origin string
	Path   string
	Digest string
}

// Position counts one-based lines and Unicode-codepoint columns. A diagnostic
// may identify a token start; it does not promise an exact body-link YAML span.
type Position struct {
	Line   int
	Column int
}

type Diagnostic struct {
	Code     string
	Message  string
	Source   Source
	Position *Position
}

type Phase struct {
	Name   string
	Status Status
}

// Result separates content validity from exact body-link coordinate coverage.
// A zero Result does not claim success. Diagnostics are deterministically
// ordered by origin, path, position, code and message.
type Result struct {
	Status      Status
	Models      []ModelResult
	Diagnostics []Diagnostic
	Limits      Limits
	Coordinates Status
}

// ModelResult validates a supplied model in its complete requirement closure.
// Packages, Definitions, Canonical and Digest are absent unless this scope fully
// passed. A passed scope's Packages is its complete own directory inventory;
// an empty list is authoritative only alongside Status == Passed.
// Only Input.Root has an entry-lock obligation. For a supplied dependency,
// lock=passed means no separate sidecar is required in this Compile; its own
// content may pass independently beside an invalid consumer or entry lock.
// Provided sidecars never replace the entry lock; their known loader errors
// remain diagnostics. Result.Status aggregates the complete supplied input.
type ModelResult struct {
	Origin      string
	Manifest    Source
	Local       bool
	ID          string
	Status      Status
	Phases      []Phase
	Packages    []Package
	Definitions []Definition
	Canonical   []byte
	Digest      string
}

// Package is an exact, nonempty directory name relative to the selected model
// root. Source.Path is relative to the SourceModel acquisition boundary, with
// its opaque Origin and no directory Digest. Implicit parents, intermediate
// Packages and empty leaves are included; the model root itself is excluded.
// This inventory does not make empty Packages importable or include them in
// canonical bytes. Each ModelResult contains only its own Packages.
type Package struct {
	Name   string
	Source Source
}

// Definition is the compiled, resolved language value, independent of paths,
// author reference spelling and source ordering. Body remains unchanged.
type Definition struct {
	Identity  Identity
	ModelID   string
	Kind      Identity
	Body      string
	Title     *string
	Spec      []Field
	BodyLinks []BodyLink
	Source    Source
}

// Field uses the effective contract field name, not the Property's address.
// Compiler output orders fields by their exact UTF-8 names.
type Field struct {
	Name  string
	Value Value
}

// BodyLink preserves occurrence order and repetition. Display distinguishes
// omitted display text from an explicitly supplied literal display string.
type BodyLink struct {
	Target      Identity
	Display     *string
	Coordinates *LinkCoordinates
}

// LinkCoordinates separates independently provable body and YAML spans under
// diagnostic-contract.md. An unavailable span is nil, never a guessed zero.
type LinkCoordinates struct {
	Body   *BodySpan
	Source *SourceSpan
}

// BodySpan addresses the decoded original body in zero-based UTF-8 bytes.
type BodySpan struct {
	StartByte int
	EndByte   int // Exclusive.
}

// SourceSpan addresses the original YAML with one-based Unicode-codepoint
// positions. End is exclusive; CRLF counts as one line break.
type SourceSpan struct {
	Start Position
	End   Position
}
