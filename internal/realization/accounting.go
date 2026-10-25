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
// and required realization remain separate: Artifacts(id) may still be empty,
// and Stale may still list links to Definitions the model no longer has.
// Ignored/input/state files never count as artifact links.
type Accounting struct {
	status   AccountingStatus
	tree     snapshot.TreeID
	files    []File
	findings []Finding
	stale    []Finding
	links    map[model.Identity][]snapshot.Path
}

func (a Accounting) Status() AccountingStatus { return a.status }
func (a Accounting) TreeID() snapshot.TreeID  { return a.tree }
func (a Accounting) Files() []File            { return slices.Clone(a.files) }
func (a Accounting) Findings() []Finding      { return slices.Clone(a.findings) }

// Stale returns realization still to do, not invalid accounting: links to
// Definitions absent from the checked model set (registry.stale_link, for the
// registrant) and registrations by a Package the model does not have
// (registry.orphaned, for the Manager owning the artifact's area, else Root).
// Each names in Owner the Manager that must act. Whether a state may keep
// them is the caller's rule.
func (a Accounting) Stale() []Finding { return slices.Clone(a.stale) }

// Artifacts returns only present, unconflicted registrations whose owner was
// valid; a stale link realizes nothing. An empty result does not waive any requirement
// for realization; in particular, ignoring files cannot supply that realization.
func (a Accounting) Artifacts(id model.Identity) []snapshot.Path { return slices.Clone(a.links[id]) }

// AccountFiles checks explicit facts without discovering policy or assigning
// owners. Classifications must cover exact canonical/Markitect inputs and ignores
// from the caller's fixed capture. Source provenance is not a physical path map.
// The complete checked Result must pass before Package/Definition absence is
// authoritative. No package is inferred from Definitions or imported inventories.
//
// Every observed file is classified exactly once: a protected input, an
// explicitly ignored file or a registered artifact. A directory entry covers
// the eligible files below it, those that are neither inputs nor ignored, and
// the most specific covering entry registers a file. With a declared Layout an
// entry registers only files its registrant owns: an exact entry for another
// owner's file and a directory entry at a directory its registrant does not own
// are registry.outside_area, another owner's files below a directory entry are
// excluded from it, and a file no area covers is layout.no_owner addressed to
// Root. A directory entry that registers no file is registry.missing_file. The
// zero Layout adds no ownership rule: the most specific entry decides.
func AccountFiles(checked model.Result, tree snapshot.Tree, registry Registry, classifications []Classification, layout Layout) Accounting {
	return accountFiles(inventoryFromResult(checked), tree, registry, classifications, layout)
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
	// orphaned reports a registrant that is no Package of the checked model,
	// such as one the owner removed or merged into another.
	orphaned := func(o Owner) bool {
		name, local := o.Package()
		return ready && local && !packages[name]
	}
	valid := map[string]bool{}
	for _, entry := range registry.entries {
		artifact := entry.artifact
		path := artifact.path
		ok := ready
		if orphaned(entry.owner) {
			// It realizes nothing; the files it still explains are stale below.
			valid[artifact.Selector()] = false
			continue
		}
		if !artifact.directory {
			if _, found := tree.Lookup(path); !found {
				add("registry.missing_file", path, entry.owner, model.Identity{}, "The registered artifact is absent from the supplied tree.")
				ok = false
			}
			if len(classes[path]) > 0 {
				add("file.conflicting_classification", path, entry.owner, model.Identity{}, "A registered artifact is also a protected input or ignored file.")
				ok = false
			}
		}
		if ready {
			for _, basis := range artifact.basis {
				if !definitions[basis] {
					result.stale = append(result.stale, Finding{"registry.stale_link", path, entry.owner, basis, fmt.Sprintf("The registered artifact links %s/%s, which the checked model does not have: remove the artifact and its registration, or link it to an existing Definition.", basis.Package, basis.ID)})
				}
			}
		}
		if layout.declared {
			var expected Owner
			var owned bool
			if artifact.directory {
				expected, owned = layout.DirectoryOwner(path.String())
			} else {
				expected, owned = layout.OwnerOf(path)
			}
			switch {
			case !owned:
				add("registry.outside_area", path, entry.owner, model.Identity{}, fmt.Sprintf("No declared area owns %s, so no Manager may register it.", artifact.Selector()))
				ok = false
			case expected != entry.owner:
				add("registry.outside_area", path, entry.owner, model.Identity{}, fmt.Sprintf("%s lies in the area of %s, not of the registering %s; it belongs in that owner's registry shard.", artifact.Selector(), expected.label(), entry.owner.label()))
				ok = false
			}
		}
		valid[artifact.Selector()] = ok
	}
	registered := map[string]int{}
	for _, entry := range tree.Entries() {
		path := entry.Path()
		file := File{entry: entry, class: UnexplainedFile}
		classified := classes[path]
		state := statePath(path)
		owner, owned := layout.OwnerOf(path)
		exactEntry, exact := registry.Lookup(path)
		exact = exact && !orphaned(exactEntry.owner)
		if !state && len(classified) == 0 {
			var orphan *Registration
			for _, candidate := range registry.Covering(path) {
				if orphaned(candidate.owner) {
					if orphan == nil {
						orphan = &candidate
					}
					continue
				}
				if layout.declared && (!owned || candidate.owner != owner) {
					// Another owner's entry neither explains nor takes the file.
					continue
				}
				file.registration, file.registered = candidate, true
				break
			}
			if !file.registered && orphan != nil {
				// Until the Manager owning the file's area, or Root where no
				// declared area owns it, registers it, the orphaned entry
				// explains the file as stale realization.
				responsible := RootOwner()
				if layout.declared && owned {
					responsible = owner
				}
				name, _ := orphan.owner.Package()
				file.registration, file.registered = *orphan, true
				result.stale = append(result.stale, Finding{"registry.orphaned", path, responsible, model.Identity{}, fmt.Sprintf("The file is registered only by Package %q, which the checked model does not have: %s registers it in its own shard with existing Definitions, or removes it.", name, responsible.label())})
			}
		}
		switch {
		case state:
			file.class, file.why = OperationalState, "Markitect operational state is outside realization."
		case len(classified) > 1 || exact && len(classified) > 0:
			file.class = ConflictingFile
		case file.registered:
			artifact := file.registration.artifact
			file.class, file.why = RegisteredArtifact, artifact.why
			registered[artifact.Selector()]++
			if valid[artifact.Selector()] {
				for _, id := range artifact.basis {
					if definitions[id] {
						result.links[id] = append(result.links[id], path)
					}
				}
			}
		case len(classified) == 1:
			file.class, file.why = classified[0].class, classified[0].why
		case layout.declared && !owned && within(markitectDirectory, path.String()):
			// Nobody can own a path under .markitect: it holds only Markitect's
			// inputs and the local registries.
			add("file.unexplained", path, RootOwner(), model.Identity{}, "The file lies in .markitect, which holds only Markitect's inputs and the local registries and is no ownership area: remove it, or declare it in .markitect/ignore.yaml with a reason.")
		case layout.declared && !owned:
			add("layout.no_owner", path, RootOwner(), model.Identity{}, "No declared area owns the file, so no Manager may keep it: the ownership in .markitect/execution.yaml needs an area for it, or the file must go.")
		default:
			add("file.unexplained", path, Owner{}, model.Identity{}, "The file is neither a registered artifact, a protected input nor explicitly ignored.")
		}
		result.files = append(result.files, file)
	}
	for _, entry := range registry.entries {
		if entry.artifact.directory && registered[entry.artifact.Selector()] == 0 && valid[entry.artifact.Selector()] {
			add("registry.missing_file", entry.artifact.path, entry.owner, model.Identity{}, "The registered directory holds no artifact of its registrant.")
		}
	}
	slices.SortFunc(result.findings, compareFinding)
	slices.SortFunc(result.stale, compareFinding)
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
