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

type ManagerContext struct {
	Manager    Manager     `json:"manager"`
	Statements []Statement `json:"statements"`
	Contracts  []Statement `json:"contracts"`
	Artifacts  []Artifact  `json:"artifacts"`
	Checks     []Check     `json:"checks"`
	Decisions  []Decision  `json:"decisions,omitempty"`
	Children   []Manager   `json:"children"`
	Findings   []Finding   `json:"findings"`
}
