package organization

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type ContextView string

const (
	ManagerView          ContextView = "manager"
	ReviewerExpectations ContextView = "reviewer-expectations"
	ReviewerCandidate    ContextView = "reviewer-candidate"
)

// HostSource is a caller-supplied record/field binding, not a runtime ID type
// or proof of acquisition, validation, completeness or authority. Empty Field
// denotes the whole recorded input. Organization never reads the record.
type HostSource struct {
	RecordID  string
	ContentID snapshot.ContentID
	Field     string
}

type HostText struct {
	Text   string
	Source HostSource
}

type FindingFact struct {
	ID       string
	Grounds  HostText
	Evidence []HostSource // References are permitted even in expectation review.
}

// OpenFindings is the Host's complete projection for this responsibility and
// dependencies. IDs and Items must match exactly, including the empty set.
// Organization checks that projection, not the external journal's completeness.
type OpenFindings struct {
	Source HostSource
	IDs    []string
	Items  []FindingFact
}

type TaskFacts struct {
	ID      string
	Package string
	// Outcome must equal ApplyOperation's exact text. Source identifies the
	// Host's operation record; it does not authenticate receipt or authority.
	Outcome HostText
	// Constraints and Requirements project already bound obligations/checks.
	// They cannot add intent, narrow responsibility or waive the fixed process.
	Constraints  []HostText
	Requirements []HostText
	Findings     OpenFindings
}

// ManagerPhase is an instruction/input projection, not a run state, evidence
// of completed Explore, admission decision or artifact write grant.
type ManagerPhase string

const (
	ManagerExplore   ManagerPhase = "explore"
	ManagerWork      ManagerPhase = "work"
	ManagerIntegrate ManagerPhase = "integrate"
	ManagerReply     ManagerPhase = "reply"
)

type ManagerProcess struct {
	Phase  ManagerPhase
	Source HostSource
}

// ParentHint is separate advice. Package identifies its asserted sender, which
// must be the exact direct parent. Topology/record shape is not authentication.
// Advice cannot author scope, replace the operation or change model authority.
type ParentHint struct {
	Package string
	Advice  HostText
}

// MessageFact is a question or reply that reaches the Manager along the tree:
// addressed to it, answering its question, or passing through it. From and To
// are Manager paths or "owner"; About names the question a reply answers.
// A message carries no authority and cannot change the operation or scope.
type MessageFact struct {
	ID, Kind, From, To, About string
	Body                      HostText
}

type PackageFact struct {
	Package string
	Purpose *HostText // Explicit mapping; absent never means an inferred purpose.
}

// ChildFact supplies direct-child state/report projections only. It has no file
// or transcript payload channel. A report is a Host-asserted validated summary,
// not a validation event performed by this renderer.
type ChildFact struct {
	Package string
	State   *HostText
	Reports []HostText
}

type ArtifactContent string

const (
	ArtifactText      ArtifactContent = "utf8-text"
	ArtifactBinaryRef ArtifactContent = "binary-reference"
	// ArtifactWorkspaceRef names a file that the step's workspace holds but
	// whose content the context does not copy.
	ArtifactWorkspaceRef ArtifactContent = "workspace-reference"
)

// ArtifactFact displays an explicit Host ownership/content assertion, never
// grants access. Text content is exact and must match ContentID. Binary content
// has no Text payload; only its metadata/reference is displayed.
type ArtifactFact struct {
	Package   string
	Path      snapshot.Path
	ContentID snapshot.ContentID
	Content   ArtifactContent
	Text      *string
	Source    HostSource
}

// ReviewFacts identifies the exact Host review subject, candidate and recorded
// expectations. No phase-completed boolean substitutes for these references.
// The Host must validate their actual bindings and enforce this view's phase
// boundary on later read/check tools as well as on the prepared context.
type ReviewFacts struct {
	Package      string
	Subject      HostSource
	Candidate    HostSource
	Expectations HostSource
	Files        []ArtifactFact
	Evidence     []HostText
}

// RenderInput is an explicit frozen projection, not a run, registry, budget
// ledger or tool grant. InputBinding references the Host's captured basis; its
// presence does not verify all run inputs. RemainingLimits are display facts.
// OwnArtifacts belong to this Manager, while candidate files are explicit
// phase-2 review inputs. No implementer conversation field is supported.
type RenderInput struct {
	View            ContextView
	Process         *ManagerProcess // Required only for ManagerView.
	ParentHints     []ParentHint    // ManagerView only; the root receives none.
	Messages        []MessageFact   // ManagerView only.
	InputBinding    HostSource
	Task            TaskFacts
	OwnerInput      *HostText
	Packages        []PackageFact
	Children        []ChildFact
	OwnArtifacts    []ArtifactFact
	RemainingLimits []HostText
	Review          *ReviewFacts
	MaxContextBytes Setting[int64]
}

type ProductText struct {
	ID        string
	Version   string
	ContentID snapshot.ContentID
}

type RenderSourceKind string

const (
	ModelRenderSource   RenderSourceKind = "model"
	ConfigRenderSource  RenderSourceKind = "config"
	HostRenderSource    RenderSourceKind = "host"
	ProductRenderSource RenderSourceKind = "product"
)

// RenderSource keeps source classes distinct. Kind selects exactly one variant.
// Model refers to original file provenance of the decoded model content, not
// reconstructed YAML bytes. Config/Host sources remain supplied assertions.
type RenderSource struct {
	Kind        RenderSourceKind
	Definition  model.Identity
	ModelID     string
	ModelDigest string
	Model       model.Source
	Config      SettingSource
	Host        HostSource
	Product     ProductText
}

// RenderSection uses zero-based UTF-8 byte offsets in Bytes(). Start/End include
// framing; TextStart/TextEnd identify precisely the unmodified text payload.
// Payload sources apply to that range; framing is bound by Format().
type RenderSection struct {
	Name               string
	Start, End         int64
	TextStart, TextEnd int64
	Sources            []RenderSource
}

type SectionSize struct {
	Name  string
	Bytes int64
}

// RenderRefusal exposes diagnostics but never a partial deliverable. Too-large
// reports count the complete payload, including frames, templates/instructions.
type RenderRefusal struct {
	Code          string
	Detail        string
	RequiredBytes int64
	LimitBytes    int64
	LimitSource   SettingSource
	Sections      []SectionSize
}

func (r *RenderRefusal) Error() string { return r.Code + ": " + r.Detail }

// ContextPackage is prepared text, never a sent/received invocation receipt.
// All accessors return fixed values or owned copies. Zero is not a package.
type ContextPackage struct {
	view        ContextView
	packagePath string
	template    ProductText
	format      ProductText
	bytes       []byte
	sections    []RenderSection
}

func (c ContextPackage) View() ContextView     { return c.view }
func (c ContextPackage) Package() string       { return c.packagePath }
func (c ContextPackage) Template() ProductText { return c.template }
func (c ContextPackage) Format() ProductText   { return c.format }
func (c ContextPackage) Bytes() []byte         { return slices.Clone(c.bytes) }
func (c ContextPackage) ContentID() snapshot.ContentID {
	if c.bytes == nil {
		return snapshot.ContentID{}
	}
	return renderContentID(c.bytes)
}
func (c ContextPackage) Sections() []RenderSection {
	result := slices.Clone(c.sections)
	for i := range result {
		result[i].Sources = slices.Clone(result[i].Sources)
	}
	return result
}
