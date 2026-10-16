// Package model compiles explicitly supplied canonical model sources.
// It performs no filesystem, repository, process or provider operations.
//
// Compile currently implements the input and YAML acquisition slice only. Its
// result remains not_completed until every normative v1 content phase exists.
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
	SourceBytes int64
	Entries     int
	YAMLDepth   int
	Models      int
}

// DefaultLimits returns the finite budgets used for zero-valued limit fields.
// SourceBytes and Entries cover the entire supplied input, including metadata
// and files outside a declared model root. Models includes the local model.
// YAMLDepth counts the root as one.
func DefaultLimits() Limits {
	return Limits{SourceBytes: 64 << 20, Entries: 100000, YAMLDepth: 128, Models: 1024}
}

type Status string

const (
	Passed       Status = "passed"
	Failed       Status = "failed"
	NotCompleted Status = "not_completed"
)

// Source binds diagnostics and IR to an exact supplied file. Digest hashes the
// original bytes; it is not a canonical model content digest.
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

// ModelResult scopes validation to a supplied model. Canonical and Digest are
// absent unless all content checks for the model and its dependencies passed.
// Definitions likewise contains only fully resolved and checked IR.
type ModelResult struct {
	Origin      string
	Manifest    Source
	Local       bool
	ID          string
	Status      Status
	Phases      []Phase
	Definitions []Definition
	Canonical   []byte
	Digest      string
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
