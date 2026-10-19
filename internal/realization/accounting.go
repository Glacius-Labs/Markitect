package realization

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// FileClass records the observed accounting of an exact supplied file.
type FileClass string

const (
	CanonicalInput     FileClass = "canonical_input"
	MarkitectInput     FileClass = "markitect_input"
	IgnoredFile        FileClass = "ignored"
	RegisteredArtifact FileClass = "artifact"
	OperationalState   FileClass = "operational_state"
	UnexplainedFile    FileClass = "unexplained"
	ConflictingFile    FileClass = "conflicting"
)

var ErrInvalidClassification = errors.New("invalid realization file classification")

// Classification is a fixed, explicitly supplied input or ignore explanation.
// It is not a path pattern, mutable ignore policy, or Agent-issued grant.
type Classification struct {
	path  snapshot.Path
	class FileClass
	why   string
}

func NewClassification(path snapshot.Path, class FileClass, why string) (Classification, error) {
	if path.String() == "" || statePath(path) || strings.TrimSpace(why) == "" ||
		(class != CanonicalInput && class != MarkitectInput && class != IgnoredFile) {
		return Classification{}, fmt.Errorf("%w: %q", ErrInvalidClassification, path)
	}
	return Classification{path: path, class: class, why: why}, nil
}

func (c Classification) Path() snapshot.Path { return c.path }
func (c Classification) Class() FileClass    { return c.class }
func (c Classification) Why() string         { return c.why }

// AccountingStatus describes only file/registry validity, never run completion,
// review approval or semantic realization of every Definition.
type AccountingStatus uint8

const (
	AccountingIncomplete AccountingStatus = iota
	AccountingInvalid
	Accounted
)

// Finding is a blocking structural accounting observation. Empty Path denotes a
// model-level observation; zero Owner/Definition mean those fields do not apply.
type Finding struct {
	Code       string
	Path       snapshot.Path
	Owner      Owner
	Definition model.Identity
	Message    string
}

// File records an actual supplied file, its class and the declared registration
// when present. Registration presence is not approval or write authorization.
type File struct {
	entry        snapshot.Entry
	class        FileClass
	why          string
	registered   bool
	registration Registration
}

func (f File) Entry() snapshot.Entry              { return f.entry }
func (f File) Class() FileClass                   { return f.class }
func (f File) Why() string                        { return f.why }
func (f File) Registration() (Registration, bool) { return f.registration, f.registered }

// Accounting is a fixed report over explicit values. Accounted means all files
// and registry records passed these structural checks. Definition assessments
// and required realization remain separate: Artifacts(id) may still be empty.
// Ignored/input/state files never count as artifact links.
type Accounting struct {
	status   AccountingStatus
	tree     snapshot.TreeID
	files    []File
	findings []Finding
	links    map[model.Identity][]snapshot.Path
}

func (a Accounting) Status() AccountingStatus { return a.status }
func (a Accounting) TreeID() snapshot.TreeID  { return a.tree }
func (a Accounting) Files() []File            { return slices.Clone(a.files) }
func (a Accounting) Findings() []Finding      { return slices.Clone(a.findings) }

// Artifacts returns only present, unconflicted registrations whose owner and
// entire model basis were valid. An empty result does not waive any requirement
// for realization; in particular, ignoring files cannot supply that realization.
func (a Accounting) Artifacts(id model.Identity) []snapshot.Path { return slices.Clone(a.links[id]) }

// AccountFiles checks explicit facts without discovering policy or assigning
// owners. Classifications must cover exact canonical/Markitect inputs and ignores
// from the caller's fixed capture. Source provenance is not a physical path map.
// The complete compiled Result must pass before Package/Definition absence is
// authoritative. No package is inferred from Definitions or imported inventories.
//
// A declared Layout additionally requires every registration to sit in the
// registry shard of the owner the layout assigns to its path
// (registry.outside_area) and reports every file without an owner, in an
// unmapped subdirectory of an artifact root, as layout.no_owner addressed to
// Root. The zero Layout adds no ownership rule.
func AccountFiles(compiled model.Result, tree snapshot.Tree, registry Registry, classifications []Classification, layout Layout) Accounting {
	return accountFiles(inventoryFromResult(compiled), tree, registry, classifications, layout)
}

func accountFiles(inventory modelInventory, tree snapshot.Tree, registry Registry, classifications []Classification, layout Layout) Accounting {
	result := Accounting{status: Accounted, tree: tree.Digest(), links: map[model.Identity][]snapshot.Path{}}
	add := func(code string, path snapshot.Path, owner Owner, definition model.Identity, message string) {
		result.findings = append(result.findings, Finding{code, path, owner, definition, message})
		result.status = AccountingInvalid
	}
	packages, definitions, ready, modelFindings := inventory.owners, inventory.definitions, inventory.ready, inventory.findings
	result.findings = append(result.findings, modelFindings...)
	if !ready {
		result.status = AccountingIncomplete
		if inventory.invalid {
			result.status = AccountingInvalid
		}
	}
	classes := map[snapshot.Path][]Classification{}
	for _, classification := range classifications {
		if classification.path.String() == "" {
			add("classification.invalid", snapshot.Path{}, Owner{}, model.Identity{}, "A zero classification is not an explicit input or ignore.")
			continue
		}
		classes[classification.path] = append(classes[classification.path], classification)
	}
	for path, entries := range classes {
		if len(entries) > 1 {
			add("classification.duplicate", path, Owner{}, model.Identity{}, "An exact file has more than one supplied classification.")
		}
		if _, found := tree.Lookup(path); !found {
			add("classification.missing_file", path, Owner{}, model.Identity{}, "The classified file is absent from the supplied tree.")
		}
	}
	valid := map[snapshot.Path]bool{}
	for _, entry := range registry.entries {
		artifact := entry.artifact
		path := artifact.path
		ok := ready
		if _, found := tree.Lookup(path); !found {
			add("registry.missing_file", path, entry.owner, model.Identity{}, "The registered artifact is absent from the supplied tree.")
			ok = false
		}
		if len(classes[path]) > 0 {
			add("file.conflicting_classification", path, entry.owner, model.Identity{}, "A registered artifact is also a protected input or ignored file.")
			ok = false
		}
		if ready {
			if name, packageOwner := entry.owner.Package(); packageOwner && !packages[name] {
				add("registry.unknown_owner", path, entry.owner, model.Identity{}, "The owner is not in the selected local model's own Package inventory.")
				ok = false
			}
			for _, basis := range artifact.basis {
				if !definitions[basis] {
					add("registry.dangling_basis", path, entry.owner, basis, "The Definition basis does not resolve in the supplied compiled model set.")
					ok = false
				}
			}
		}
		if layout.declared {
			ok = ok && layout.HasOwner(path)
			if expected := layout.OwnerOf(path); expected != entry.owner {
				add("registry.outside_area", path, entry.owner, model.Identity{}, fmt.Sprintf("The path lies in the area of %s, not in the area of the registering %s; it belongs in that owner's registry shard.", expected.label(), entry.owner.label()))
				ok = false
			}
		}
		valid[path] = ok
	}
	for _, entry := range tree.Entries() {
		path := entry.Path()
		file := File{entry: entry, class: UnexplainedFile}
		registration, registered := registry.Lookup(path)
		file.registration, file.registered = registration, registered
		classified := classes[path]
		// A file without an owner is reported once as such; "unexplained" would only
		// tell Root to register a file that no Manager may own.
		unowned := layout.declared && !layout.HasOwner(path) && !statePath(path) && len(classified) == 0
		if unowned {
			add("layout.no_owner", path, RootOwner(), model.Identity{}, "The file lies in a subdirectory of an artifact root that no Package directory or persistent layout exception covers, so no Manager owns it; the subdirectory needs an owning Package directory or a persistent layout exception.")
		}
		switch {
		case statePath(path):
			file.class, file.why = OperationalState, "Markitect operational state is outside realization."
		case len(classified) > 1 || registered && len(classified) > 0:
			file.class = ConflictingFile
		case registered:
			file.class, file.why = RegisteredArtifact, registration.artifact.why
			if valid[path] {
				for _, id := range registration.artifact.basis {
					result.links[id] = append(result.links[id], path)
				}
			}
		case len(classified) == 1:
			file.class, file.why = classified[0].class, classified[0].why
		case !unowned:
			add("file.unexplained", path, Owner{}, model.Identity{}, "The file is neither a registered artifact, a protected input nor explicitly ignored.")
		}
		result.files = append(result.files, file)
	}
	slices.SortFunc(result.findings, compareFinding)
	return result
}

func compareFinding(a, b Finding) int {
	for _, pair := range [][2]string{{a.Path.String(), b.Path.String()}, {a.Code, b.Code}, {a.Owner.packageName, b.Owner.packageName}} {
		if c := strings.Compare(pair[0], pair[1]); c != 0 {
			return c
		}
	}
	if a.Owner.root != b.Owner.root {
		if a.Owner.root {
			return -1
		}
		return 1
	}
	if c := compareIdentity(a.Definition, b.Definition); c != 0 {
		return c
	}
	return strings.Compare(a.Message, b.Message)
}
