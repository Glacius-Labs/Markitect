package apply

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ErrCandidatesUnavailable reports a preparation that cannot ground candidate
// accounting because it did not pass.
var ErrCandidatesUnavailable = errors.New("candidate accounting requires a passed settings preparation")

// CandidateFinding is one blocking structural observation on a candidate tree.
// Owner is the Manager that must fix it: the owner of the path, the Manager of
// a bad registry shard, or Root when nothing narrower applies. Path may be empty
// for a model-level observation.
type CandidateFinding struct {
	Owner   string
	Code    string
	Path    string
	Message string
}

// Candidates accounts candidate trees of one Apply against its captured start.
// It decides ownership for the run's team and checks, as the closing structural
// gate before any Verify, that the candidate is completely accounted: every file
// is a protected input, a registered artifact or an explained ignore; every
// registration sits in the shard of the Manager that owns its path; the shards
// themselves are valid; and no canonical or control input changed. It starts no
// work, writes nothing and publishes nothing.
type Candidates struct {
	compiled        model.Result
	start           snapshot.Tree
	topology        organization.Topology
	registry        realization.Registry
	classifications []realization.Classification
	protections     []realization.Protection
	layout          realization.Layout
}

// NewCandidates binds candidate accounting to one passed preparation: its
// compiled model, captured start tree and registry, classifications, protected
// inputs and declared layout. Nothing is read from the host.
func NewCandidates(prepared CapturedSettingsAttempt) (Candidates, error) {
	if prepared.Settings().Status() != model.Passed {
		return Candidates{}, ErrCandidatesUnavailable
	}
	structural := prepared.Structural()
	files := structural.Files()
	classifications, classified := files.Classifications()
	protections, protected := files.Protections()
	registry, registered := structural.Registry().Registry()
	if !classified || !protected || !registered {
		return Candidates{}, ErrCandidatesUnavailable
	}
	stage := structural.Model()
	topology, err := organization.BuildTopology(stage.compiled)
	if err != nil {
		return Candidates{}, fmt.Errorf("candidate accounting: %w", err)
	}
	c := Candidates{compiled: stage.compiled, start: stage.inputs.Raw.Census, topology: topology, registry: registry, protections: protections}
	c.layout, _ = files.Layout()
	// The start shards are replaced by the candidate's own: a Manager may add,
	// change or remove its shard.
	for _, classification := range classifications {
		if !strings.HasPrefix(classification.Path().String(), registryDirectory) {
			c.classifications = append(c.classifications, classification)
		}
	}
	return c, nil
}

// OwnerOf returns the Manager that owns a path: Root is ".". A registry shard
// belongs to its Manager. Under a declared layout the layout decides; without
// one the path's registration in the captured start registry decides, and a new
// or unregistered path belongs to Root. A file in an unmapped subdirectory of an
// artifact root has no owner under the layout: OwnerOf returns Root, which must
// resolve it, and accounting reports it as layout.no_owner.
func (c Candidates) OwnerOf(path snapshot.Path) string {
	if owner, err := registryOwner(path); err == nil {
		return ownerName(owner)
	}
	if c.layout.Declared() {
		return ownerName(c.layout.OwnerOf(path))
	}
	if registration, found := c.registry.Lookup(path); found {
		return ownerName(registration.Owner())
	}
	return organization.Root
}

// Shard returns the repository path of the registry shard a Manager maintains.
func (c Candidates) Shard(manager string) string {
	if manager == organization.Root {
		return registryDirectory + "registry.yaml"
	}
	return registryDirectory + manager + "/registry.yaml"
}

// Areas describes in short human-readable words the areas a Manager owns, for
// its prompts. It is empty for an unknown Manager.
func (c Candidates) Areas(manager string) []string {
	if _, found := c.topology.Manager(manager); !found {
		return nil
	}
	owner := realization.RootOwner()
	if manager != organization.Root {
		owner, _ = realization.PackageOwner(manager)
	}
	if c.layout.Declared() {
		return c.layout.Describe(owner, c.start)
	}
	if manager == organization.Root {
		return []string{"every file that no other Manager registered in its shard"}
	}
	return []string{"the files it registers in " + c.Shard(manager)}
}

// WritePlan returns where a Manager's Execute turns may write in the shared
// working tree. Without a declared layout nothing is enforced and ownership is
// trusted. With one, Root writes the whole tree except the protected inputs
// under .markitect, the areas of every other Manager and their registry shards;
// a Package writes only its own areas and its registry shard, minus the deeper
// areas of its descendants. The mounts name directories, parents before their
// descendants, and list only where the writability changes. An exception that
// names a file of the start stand cannot be mounted: it stays with trust and
// accounting, like the files directly in an artifact root.
func (c Candidates) WritePlan(manager string) run.WritePlan {
	if !c.layout.Declared() {
		return run.WritePlan{}
	}
	type region struct{ path, owner string }
	regions := []region{{".markitect", ""}, {strings.TrimSuffix(registryDirectory, "/"), organization.Root}}
	for _, member := range c.topology.Managers() {
		if member.Package != organization.Root {
			regions = append(regions, region{registryDirectory + member.Package, member.Package})
		}
	}
	for _, area := range c.layout.Areas(c.start) {
		regions = append(regions, region{area.Path.String(), ownerName(area.Owner)})
	}
	slices.SortFunc(regions, func(a, b region) int { return cmp.Compare(a.path, b.path) })
	plan := run.WritePlan{Enforced: true, Base: manager == organization.Root}
	for i, r := range regions {
		inherited := plan.Base
		for j := i - 1; j >= 0; j-- {
			if strings.HasPrefix(r.path, regions[j].path+"/") {
				inherited = regions[j].owner == manager
				break
			}
		}
		if writable := r.owner == manager; writable != inherited {
			plan.Mounts = append(plan.Mounts, run.PathMount{Path: r.path, Writable: writable})
		}
	}
	return plan
}

// Account returns the blocking findings of a candidate tree; an accounted
// candidate returns an empty slice. Findings are ordered by Owner, Path, Code and
// Message.
func (c Candidates) Account(tree snapshot.Tree) []CandidateFinding {
	var findings []CandidateFinding
	add := func(owner, code, path, message string) {
		findings = append(findings, CandidateFinding{Owner: owner, Code: code, Path: path, Message: message})
	}
	registry, classifications := c.shards(tree, add)
	changed := make(map[string]bool)
	for _, finding := range c.protectedInputs(tree) {
		changed[finding.Path] = true
		findings = append(findings, finding)
	}
	if registry != nil {
		accounting := realization.AccountFiles(c.compiled, tree, *registry, append(slices.Clone(c.classifications), classifications...), c.layout)
		for _, finding := range accounting.Findings() {
			path := finding.Path.String()
			if changed[path] && (finding.Code == "file.unexplained" || finding.Code == "classification.missing_file") {
				continue
			}
			add(c.findingOwner(finding), finding.Code, path, c.findingMessage(finding))
		}
	}
	slices.SortFunc(findings, func(a, b CandidateFinding) int {
		return cmp.Or(cmp.Compare(a.Owner, b.Owner), cmp.Compare(a.Path, b.Path), cmp.Compare(a.Code, b.Code), cmp.Compare(a.Message, b.Message))
	})
	return slices.Compact(findings)
}

// shards decodes the registry shards found in the candidate tree. Their invalid
// records, repeated registrations and layout faults are findings of the owning
// Manager. A nil registry means accounting cannot continue.
func (c Candidates) shards(tree snapshot.Tree, add func(owner, code, path, message string)) (*realization.Registry, []realization.Classification) {
	decoded := registryPreparation{status: model.Passed}
	type registration struct {
		owner string
		value realization.Registration
	}
	var registrations []registration
	var classifications []realization.Classification
	for _, entry := range tree.Entries() {
		path := entry.Path()
		if !strings.HasPrefix(path.String(), registryDirectory) {
			continue
		}
		if classification, err := realization.NewClassification(path, realization.MarkitectInput, "Registry shard maintained by its Manager."); err == nil {
			classifications = append(classifications, classification)
		}
		owner, err := registryOwner(path)
		if err != nil {
			add(organization.Root, "registry.layout", path.String(), err.Error())
			continue
		}
		name := ownerName(owner)
		if _, found := c.topology.Manager(name); !found {
			add(organization.Root, "registry.unknown_owner", path.String(), "Registry shard belongs to a Package outside the current local inventory.")
			continue
		}
		for _, value := range decoded.decodeRegistry(SourceFile{File: entry, RepositoryPath: path}, owner, c.compiled, tree) {
			registrations = append(registrations, registration{name, value})
		}
	}
	for _, diagnostic := range decoded.diagnostics {
		owner := organization.Root
		if shard, err := registryOwner(diagnostic.Source); err == nil {
			owner = ownerName(shard)
		}
		message := diagnostic.Message
		if diagnostic.Position != nil {
			message = fmt.Sprintf("%s (line %d)", message, diagnostic.Position.Line)
		}
		add(owner, diagnostic.Code, diagnostic.Source.String(), message)
	}
	// A path registered more than once is ambiguous. Every registrant must fix it;
	// accounting continues with the first registration.
	slices.SortStableFunc(registrations, func(a, b registration) int {
		return cmp.Or(cmp.Compare(a.value.Artifact().Path().String(), b.value.Artifact().Path().String()), cmp.Compare(a.owner, b.owner))
	})
	var unique []realization.Registration
	for i, entry := range registrations {
		path := entry.value.Artifact().Path()
		if i > 0 && registrations[i-1].value.Artifact().Path() == path {
			add(entry.owner, "registry.duplicate", path.String(), fmt.Sprintf("The artifact is also registered in the shard of %q; one path has exactly one owner.", registrations[i-1].owner))
			add(registrations[i-1].owner, "registry.duplicate", path.String(), fmt.Sprintf("The artifact is also registered in the shard of %q; one path has exactly one owner.", entry.owner))
			continue
		}
		unique = append(unique, entry.value)
	}
	registry, err := realization.NewRegistry(unique)
	if err != nil {
		add(organization.Root, "registry.records_invalid", "", err.Error())
		return nil, nil
	}
	return &registry, classifications
}

// protectedInputs reports every canonical or Markitect control input that the
// candidate changed, deleted or newly added against the captured start. Registry
// shards are the exception: the Managers maintain them.
func (c Candidates) protectedInputs(tree snapshot.Tree) []CandidateFinding {
	var findings []CandidateFinding
	quoted := 0
	check := func(path snapshot.Path, before, after snapshot.Entry, hadBefore, hasAfter bool) {
		if name := path.String(); strings.HasPrefix(name, registryDirectory) || strings.HasPrefix(name, ".markitect/state/") || strings.HasPrefix(name, ".git/") {
			return
		}
		if !slices.ContainsFunc(c.protections, func(p realization.Protection) bool { return p.Covers(path) }) {
			return
		}
		var code, what string
		switch {
		case hadBefore && !hasAfter:
			code, what = "input.deleted", "deleted"
		case !hadBefore && hasAfter:
			code, what = "input.added", "added"
		case before != after:
			code, what = "input.changed", "changed"
		default:
			return
		}
		findings = append(findings, CandidateFinding{Owner: c.OwnerOf(path), Code: code, Path: path.String(), Message: "The canonical or Markitect control input was " + what + "; Managers cannot change the model or project controls. Restore it, and ask the owner through Root if the intent must change." + originalText(before, code, &quoted)})
	}
	for _, before := range c.start.Entries() {
		after, found := tree.Lookup(before.Path())
		check(before.Path(), before, after, true, found)
	}
	for _, after := range tree.Entries() {
		if _, found := c.start.Lookup(after.Path()); !found {
			check(after.Path(), snapshot.Entry{}, after, false, true)
		}
	}
	return findings
}

// maxOriginalText is the largest original protected input a finding quotes;
// maxQuotedTotal bounds all quotes of one candidate, so a large deletion cannot
// flood the owner's prompt.
const (
	maxOriginalText = 8 << 10
	maxQuotedTotal  = 32 << 10
)

// originalText states the start content of a changed or deleted protected input,
// so the owner can restore the exact bytes without access to the start stand: the
// text itself when it is valid UTF-8 of at most maxOriginalText bytes and the
// candidate's quotes stay within maxQuotedTotal, else its SHA-256 and size.
func originalText(before snapshot.Entry, code string, quoted *int) string {
	if code == "input.added" {
		return ""
	}
	content := before.Bytes()
	if len(content) <= maxOriginalText && *quoted+len(content) <= maxQuotedTotal && utf8.Valid(content) {
		*quoted += len(content)
		return fmt.Sprintf(" Original content, exactly %d bytes, between the BEGIN and END lines (the line breaks around it are not part of it):\nBEGIN\n%s\nEND", len(content), content)
	}
	return fmt.Sprintf(" The original content is %d bytes with SHA-256 %s.", len(content), before.ContentID())
}

// findingOwner names the Manager that must fix an accounting finding: the
// registering Manager when there is one, else the owner of the path.
func (c Candidates) findingOwner(finding realization.Finding) string {
	if finding.Owner.IsRoot() {
		return organization.Root
	}
	if name, ok := finding.Owner.Package(); ok {
		return name
	}
	if finding.Path.String() != "" {
		return c.OwnerOf(finding.Path)
	}
	return organization.Root
}

func (c Candidates) findingMessage(finding realization.Finding) string {
	if finding.Code == "layout.no_owner" {
		// Root cannot fix this by registering, and the layout lives in the protected
		// execution.yaml that no Manager may change.
		return "The file has no owner under the persistent layout: remove it, or ask the owner through Root for a layout exception."
	}
	if finding.Code != "file.unexplained" {
		return finding.Message
	}
	owner := c.OwnerOf(finding.Path)
	if !c.layout.Declared() && owner == organization.Root {
		// No declared layout: nothing says who made the file, and Root owns only what
		// it registers itself. Root coordinates and finds the creator through its children.
		return finding.Message + " The Manager that created this file registers it in its own registry shard (" + registryDirectory + "<package>/registry.yaml, or " + c.Shard(organization.Root) + " for a shared file); Root asks its children who created it and does not take the file over."
	}
	return fmt.Sprintf("%s Register it in the shard of its owner %q: %s.", finding.Message, owner, c.Shard(owner))
}

func ownerName(owner realization.Owner) string {
	if name, ok := owner.Package(); ok {
		return name
	}
	return organization.Root
}
