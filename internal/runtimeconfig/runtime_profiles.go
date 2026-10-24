package runtimeconfig

import (
	"context"
	"encoding/json"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

// AgentProfilePreparer validates one closed, versioned adapter section and
// observes static metadata. Image observation never starts the image or provider.
// Supported CLI versions and metadata parsing belong to the adapter. Preparation
// must not authenticate, negotiate tools, or read uncaptured instructions.
type AgentProfilePreparer interface {
	PrepareAgentProfile(context.Context, ProfileInput) (ProfileObservation, error)
}

// CheckProfilePreparer validates one closed, versioned process-check section.
// Preparation observes metadata only; it executes no check or command.
type CheckProfilePreparer interface {
	PrepareCheckProfile(context.Context, ProfileInput) (ProfileObservation, error)
}

// ProfileInput can only be produced from an acquired Runtime profile document by Apply.
// Section names the selected adapter section (codexAppServer for an Agent or
// container for a check); Inline returns that section, never a whole catalog
// to parse.
type ProfileInput struct {
	document                        project.SourceFile
	namespace, name, section, field string
	inline                          *yaml.Node
	instructions                    []project.SourceFile
}

// ProfileBinding is an opaque comparison value for the exact supplied request.
// It prevents accidental reuse of another request's observations, but does not
// replace the port's actual metadata observations or their validation.
type ProfileBinding struct{ content snapshot.ContentID }

func (p ProfileInput) Document() project.SourceFile       { return p.document }
func (p ProfileInput) Location() project.SourceLocation   { return p.document.Location }
func (p ProfileInput) Namespace() string                  { return p.namespace }
func (p ProfileInput) Name() string                       { return p.name }
func (p ProfileInput) Section() string                    { return p.section }
func (p ProfileInput) Field() string                      { return p.field }
func (p ProfileInput) Inline() *yaml.Node                 { return cloneRuntimeNode(p.inline) }
func (p ProfileInput) Instructions() []project.SourceFile { return slices.Clone(p.instructions) }

type RuntimeProfileDiagnostic struct {
	Code     string
	Source   organization.SettingSource
	Position *model.Position
	Message  string
}

// ProfileContract identifies the concrete validation contract with an explicit
// positive version. Apply checks this shape and binds both literal values;
// supported versions and their semantics belong to the concrete adapter.
type ProfileContract struct {
	Name    string
	Version uint32
}

type AgentImageProvider string

const AgentImageCodexAppServer AgentImageProvider = "codex-app-server"

// AgentImageMetadataFile identifies actual whole bytes read from a regular file
// inside ImageID. Path is an exact canonical absolute POSIX image path, not a
// Runtime location or an instruction for Apply to read. It is independent of a
// schema digest merely recorded in those bytes. Size is bounded to 1 MiB.
type AgentImageMetadataFile struct {
	Path      string
	ContentID snapshot.ContentID
	Size      int64
}

// AgentImageObservation is an adapter's static validation assertion, bound to an
// immutable engine image ID (sha256:<64 lowercase hex digits>). CLIVersion and
// AgentdVersion retain literal recorded values, not observed process handshakes.
// RecordedCodexSchemaDigest is required; it is a build-recorded sha256 digest,
// never a complete observed schema inventory.
// Metadata binds the entire actual file, including other recorded values.
// This evidence grants no process, credential, Session or execution authority.
type AgentImageObservation struct {
	Provider                  AgentImageProvider
	ImageID                   string
	Metadata                  AgentImageMetadataFile
	CLIVersion                string
	AgentdVersion             string
	RecordedCodexSchemaDigest string
}

// CheckImageObservation binds a container check to its immutable engine image
// ID and to the executable it runs: Executable identifies the actual whole
// bytes of that regular file inside the image, by its canonical absolute POSIX
// path. It is observed from the image, never from a Runtime file, and grants no
// process or execution authority.
type CheckImageObservation struct {
	ImageID    string
	Executable AgentImageMetadataFile
}

// ProfileObservation is static evidence supplied by a port, not a runtime
// capability or a provider DTO. InputBinding echoes ProfileInput.Binding();
// Passed without that exact echo is rejected. Contract identifies the concrete
// validation contract and its version. Passed requires exactly one evidence
// mode: AgentImage for an Agent or CheckImage for a container check.
// Instructions select only captured inputs;
// RequiredInstructions requests a NEW complete acquisition. Apply computes the
// identity itself; callers cannot supply an arbitrary profile identity.
type ProfileObservation struct {
	InputBinding         ProfileBinding
	Status               model.Status
	Contract             ProfileContract
	AgentImage           *AgentImageObservation `json:",omitempty"`
	CheckImage           *CheckImageObservation `json:",omitempty"`
	Instructions         []project.SourceLocation
	RequiredInstructions []project.SourceLocation
	Diagnostics          []RuntimeProfileDiagnostic
}

// PreparedProfile retains partial port observations, even on error. Identity is
// available only for complete static evidence; it establishes no model access,
// negotiated tools, Sessions support or execution authority.
type PreparedProfile struct {
	input       ProfileInput
	observation ProfileObservation
	identity    string
}

func (p PreparedProfile) Input() ProfileInput {
	r := p.input
	r.inline = cloneRuntimeNode(r.inline)
	r.instructions = slices.Clone(r.instructions)
	return r
}
func (p PreparedProfile) Observation() ProfileObservation {
	return cloneProfileObservation(p.observation)
}
func (p PreparedProfile) Identity() (string, bool) { return p.identity, p.identity != "" }
func (p PreparedProfile) Source() organization.SettingSource {
	return runtimeSource(p.input.document, p.input.field)
}

func runtimeSource(file project.SourceFile, field string) organization.SettingSource {
	if file.Location.Boundary == "" || file.Location.Path.String() == "" {
		return organization.SettingSource{Field: field}
	}
	// A JSON tuple is injective across boundary/path spellings. This opaque
	// locator has no resolver and is never interpreted as a read instruction.
	locator, _ := json.Marshal([2]string{file.Location.Boundary, file.Location.Path.String()})
	return organization.SettingSource{File: "source:" + string(locator), ContentID: file.File.ContentID(), Field: field}
}

func cloneRuntimeNode(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	r := *n
	r.Alias = nil // Only restricted, alias-free nodes may enter ProfileInput.
	r.Content = make([]*yaml.Node, len(n.Content))
	for i, child := range n.Content {
		r.Content[i] = cloneRuntimeNode(child)
	}
	return &r
}

func cloneRuntimeDiagnostics(in []RuntimeProfileDiagnostic) []RuntimeProfileDiagnostic {
	r := slices.Clone(in)
	for i := range r {
		if r[i].Position != nil {
			p := *r[i].Position
			r[i].Position = &p
		}
	}
	return r
}

func cloneProfileObservation(o ProfileObservation) ProfileObservation {
	if o.AgentImage != nil {
		image := *o.AgentImage
		o.AgentImage = &image
	}
	if o.CheckImage != nil {
		image := *o.CheckImage
		o.CheckImage = &image
	}
	o.Instructions = slices.Clone(o.Instructions)
	o.RequiredInstructions = slices.Clone(o.RequiredInstructions)
	o.Diagnostics = cloneRuntimeDiagnostics(o.Diagnostics)
	return o
}
