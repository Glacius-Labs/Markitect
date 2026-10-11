// Package projectmodel derives project-owned management, artifact and impact
// views from the structural Core model. It performs no I/O or provider calls.
package projectmodel

const APIVersion = "project.markitect.example.org/v1alpha1"

type File struct {
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Digest string `json:"digest"`
}

type Finding struct {
	Code     string `json:"code"`
	Subject  string `json:"subject,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type Manager struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace"`
	Purpose      string   `json:"purpose"`
	Parent       string   `json:"parent,omitempty"`
	Owns         []string `json:"owns"`
	Instructions string   `json:"instructions,omitempty"`
}

type Statement struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Owner       string   `json:"owner"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Public      bool     `json:"public"`
	Uses        []string `json:"uses"`
	Requires    []string `json:"requires"`
	Source      string   `json:"source"`
}

type Artifact struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Owner    string   `json:"owner"`
	Role     string   `json:"role"`
	Realizes []string `json:"realizes"`
	Paths    []string `json:"paths"`
	Checks   []string `json:"checks"`
	Required bool     `json:"required"`
	Reason   string   `json:"reason,omitempty"`
}

type Check struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Owner      string   `json:"owner"`
	Command    []string `json:"command"`
	Uses       []string `json:"uses"`
	Limitation string   `json:"limitation"`
}

// Decision records a Manager's decision about a Statement. Its Owner is the
// nearest Manager at or above its namespace; Actor is provenance, not authority.
type Decision struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Owner     string `json:"owner"`
	Subject   string `json:"subject"`
	Actor     string `json:"actor"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
}

type FileEntry struct {
	Path       string   `json:"path"`
	Digest     string   `json:"digest,omitempty"`
	Mode       string   `json:"mode,omitempty"`
	Owner      string   `json:"owner,omitempty"`
	Class      string   `json:"class"`
	Statements []string `json:"statements"`
	Artifacts  []string `json:"artifacts"`
	Checks     []string `json:"checks"`
	Exists     bool     `json:"exists"`
}

type Report struct {
	APIVersion      string      `json:"apiVersion"`
	ModelDigest     string      `json:"modelDigest"`
	InventoryDigest string      `json:"inventoryDigest"`
	Digest          string      `json:"digest"`
	Status          string      `json:"status"`
	Managers        []Manager   `json:"managers"`
	Statements      []Statement `json:"statements"`
	Artifacts       []Artifact  `json:"artifacts"`
	Checks          []Check     `json:"checks"`
	Decisions       []Decision  `json:"decisions,omitempty"`
	Files           []FileEntry `json:"files"`
	Findings        []Finding   `json:"findings"`
	Unknown         []string    `json:"unknown"`

	// unprojected maps each definition to a digest of the parts the collections
	// above do not carry, so Impact can see changes to them. It is set only by
	// Analyze and is not part of the JSON contract or Digest.
	unprojected map[string]string
	// written maps each definition and property to how it is written, so Impact
	// can see edits that change only the writing. Set only by Analyze.
	written map[string]map[string]writtenProperty
	// sources maps each definition to its model file. Set only by Analyze.
	sources map[string]string
}

// writtenProperty compares the digest of a property as written with the
// digest of the value projected from it. For a set-like list, elements holds
// the projected keys in written order, repeats included.
type writtenProperty struct {
	raw, value string
	elements   []string
}

type ChangeImpact struct {
	APIVersion         string    `json:"apiVersion"`
	BaseDigest         string    `json:"baseDigest"`
	CandidateDigest    string    `json:"candidateDigest"`
	Digest             string    `json:"digest"`
	ChangedDefinitions []string  `json:"changedDefinitions"`
	AffectedStatements []string  `json:"affectedStatements"`
	Managers           []string  `json:"managers"`
	Files              []string  `json:"files"`
	Checks             []string  `json:"checks"`
	Unknown            []string  `json:"unknown"`
	Findings           []Finding `json:"findings"`
}

// ImpactExplanation explains each element of one ChangeImpact (DEC-023).
type ImpactExplanation struct {
	APIVersion   string             `json:"apiVersion"`
	ImpactDigest string             `json:"impactDigest"`
	Digest       string             `json:"digest"`
	Elements     []ExplainedElement `json:"elements"`
}

// ExplainedElement says why one impact element is included and whether it
// must change ("change") or is only context to read ("context"). The class is
// a priority signal; every element stays in the impact.
type ExplainedElement struct {
	Kind    string        `json:"kind"`
	ID      string        `json:"id"`
	Class   string        `json:"class"`
	Reason  string        `json:"reason"`
	Witness []WitnessStep `json:"witness"`
	// Partial marks a witness cut where it leaves what the Manager can see.
	Partial bool `json:"partial,omitempty"`
}

// KnowledgeGraph is a read-only view of model relations, either for the whole
// project or for one Manager, built from its Context only.
type KnowledgeGraph struct {
	Scope  string      `json:"scope"`
	Digest string      `json:"digest"`
	Nodes  []GraphNode `json:"nodes"`
	Edges  []GraphEdge `json:"edges"`
}

// GraphNode is a definition, a file or an expected path. A reference node is
// known only as the end of a visible edge and carries no content.
type GraphNode struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Reference bool   `json:"reference,omitempty"`
}

// GraphEdge is one declared relation, in its declared direction.
type GraphEdge struct {
	From     string `json:"from"`
	Relation string `json:"relation"`
	To       string `json:"to"`
}

// TraceRequest selects a start node, a direction ("out", "in" or "both") and
// limits; zero limits select defaults.
type TraceRequest struct {
	From      string `json:"from"`
	Direction string `json:"direction"`
	MaxDepth  int    `json:"maxDepth"`
	MaxNodes  int    `json:"maxNodes"`
}

// TraceResult lists the nodes a trace reached. Complete is false when a limit
// stopped it before every reachable node was found.
type TraceResult struct {
	Scope    string       `json:"scope"`
	From     string       `json:"from"`
	Complete bool         `json:"complete"`
	Reached  []TracedNode `json:"reached"`
}

// TracedNode is one reached node with a shortest witness path from the start.
type TracedNode struct {
	Node    GraphNode   `json:"node"`
	Depth   int         `json:"depth"`
	Witness []GraphEdge `json:"witness"`
}

// WitnessStep is one model relation on the path from a change to an element.
type WitnessStep struct {
	FromKind string `json:"fromKind"`
	From     string `json:"from"`
	Relation string `json:"relation"`
	ToKind   string `json:"toKind"`
	To       string `json:"to"`
}

type ManagerContext struct {
	Manager    Manager     `json:"manager"`
	Statements []Statement `json:"statements"`
	Contracts  []Statement `json:"contracts"`
	Artifacts  []Artifact  `json:"artifacts"`
	Checks     []Check     `json:"checks"`
	Decisions  []Decision  `json:"decisions,omitempty"`
	// ForeignChecks are Checks other Managers own that this Manager's own
	// Artifacts declare or that use its own Statements; Checks are its own.
	ForeignChecks []Check   `json:"foreignChecks,omitempty"`
	Children      []Manager `json:"children"`
	Findings      []Finding `json:"findings"`
}
