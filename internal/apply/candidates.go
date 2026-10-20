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

// ErrUnsupportedOwnership reports an ownership shape the Runtime cannot
// enforce: execution is refused rather than degraded to trust.
var ErrUnsupportedOwnership = errors.New("the declared ownership cannot be enforced")

// CandidateFinding is one blocking structural observation on a stand. Owner is
// the Manager that must fix it: the owner of the path, the Manager of a bad
// registry shard, or Root when nothing narrower applies. Path may be empty for
// a model-level observation.
type CandidateFinding struct {
	Owner   string
	Code    string
	Path    string
	Message string
}

// Candidates holds the deterministic gates and ownership facts of one run over
// its captured start: the File check (every file is a protected input, a
// registered artifact or an explained ignore; every registry entry lies in its
// registrant's area; the local registries are valid; no canonical or control
// input changed), the Model check of a stand, the owner of every path, the write
// plans that enforce ownership and the derived registry view. It starts no
// work, writes nothing and publishes nothing.
type Candidates struct {
	start           CapturedModel
	checked         model.Result
	tree            snapshot.Tree
	topology        organization.Topology
	registry        realization.Registry
	classifications []realization.Classification
	protections     []realization.Protection
	layout          realization.Layout
	// guard, when set, replaces the protected-input check of Account: Extract
	// guards a different set of inputs than Apply does.
	guard func(snapshot.Tree) []CandidateFinding
}

// NewCandidates binds the gates to one passed preparation: its checked model,
// captured start tree and registry, classifications, protected inputs and
// declared ownership. Nothing is read from the host.
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
	return newCandidates(structural, classifications, protections, registry)
}

// newCandidates binds the gates to a structural preparation whose complete sets
// the caller has already obtained.
func newCandidates(structural CapturedPreparationAttempt, classifications []realization.Classification, protections []realization.Protection, registry realization.Registry) (Candidates, error) {
	stage := structural.Model()
	topology, err := organization.BuildTopology(stage.checked)
	if err != nil {
		return Candidates{}, fmt.Errorf("candidate accounting: %w", err)
	}
	c := Candidates{start: stage, checked: stage.checked, tree: stage.inputs.Raw.Census, topology: topology, registry: registry, protections: protections}
	c.layout = structural.files.layout
	// The start shards are replaced by the candidate's own: a Manager may add,
	// change or remove its shard.
	for _, classification := range classifications {
		if !strings.HasPrefix(classification.Path().String(), registryDirectory) {
			c.classifications = append(c.classifications, classification)
		}
	}
	return c, nil
}

// OwnerOf returns the Manager that owns a path, Root as ".", or "" when
// nothing owns it. A registry shard belongs to its Manager. Under declared
// ownership the most specific area decides; without it the registrant of the
// most specific entry in the captured start registry.
func (c Candidates) OwnerOf(path snapshot.Path) string {
	if owner, err := registryOwner(path); err == nil {
		return ownerName(owner)
	}
	if c.layout.Declared() {
		owner, ok := c.layout.OwnerOf(path)
		if !ok {
			return ""
		}
		return ownerName(owner)
	}
	if covering := c.registry.Covering(path); len(covering) > 0 {
		return ownerName(covering[0].Owner())
	}
	return ""
}

// Shard returns the repository path of the local registry a Manager maintains.
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
	if !c.layout.Declared() {
		return []string{"the files it registers in " + c.Shard(manager)}
	}
	return c.layout.Describe(ownerOf(manager))
}

func ownerOf(manager string) realization.Owner {
	if manager == organization.Root {
		return realization.RootOwner()
	}
	owner, _ := realization.PackageOwner(manager)
	return owner
}

// region is a directory or exact file of the shared tree and the Manager that
// holds it; an empty owner holds it for no Manager.
type region struct {
	path  string
	owner string
	file  bool
}

// WritePlan returns the enforced write plan of a Manager's turns: its own
// areas and its local registry are writable; protected inputs, .markitect and
// every other Manager's areas, including deeper ones inside its own, are
// read-only. The mounts overlay only where writability changes, parents before
// descendants. An exact-file area that needs its own mount must exist in the
// start stand, because a file that does not exist cannot be mounted; such a
// shape is refused instead of left to trust. Declared ownership is required.
func (c Candidates) WritePlan(manager string) (run.WritePlan, error) {
	if !c.layout.Declared() {
		return run.WritePlan{}, fmt.Errorf("%w: .markitect/execution.yaml declares no ownership, so no write plan can enforce who writes what", ErrUnsupportedOwnership)
	}
	regions := []region{{path: ".markitect"}, {path: strings.TrimSuffix(registryDirectory, "/"), owner: organization.Root}}
	for _, member := range c.topology.Managers() {
		if member.Package != organization.Root {
			regions = append(regions, region{path: registryDirectory + member.Package, owner: member.Package})
		}
	}
	base := ""
	for _, area := range c.layout.Areas() {
		if area.Path == "" {
			base = ownerName(area.Owner)
			continue
		}
		regions = append(regions, region{path: area.Path, owner: ownerName(area.Owner), file: !area.Directory})
	}
	for _, p := range c.protections {
		if file, ok := p.File(); ok && !strings.HasPrefix(file.String(), ".markitect/") {
			regions = append(regions, region{path: file.String(), file: true})
		}
		if dir, ok := p.Directory(); ok && !dir.IsRoot() && dir.String() != ".markitect" && !strings.HasPrefix(dir.String(), ".markitect/") && dir.String() != ".git" {
			regions = append(regions, region{path: dir.String()})
		}
	}
	return planOf(manager, base == manager, regions, c.tree)
}

// planOf turns regions into the write plan of one Manager: the base is
// writable or not, and the mounts overlay only where writability changes,
// parents before their descendants. A region that needs a mount must exist in
// the tree as the kind it is declared: a missing file cannot be mounted.
func planOf(manager string, base bool, regions []region, tree snapshot.Tree) (run.WritePlan, error) {
	slices.SortFunc(regions, func(a, b region) int { return cmp.Compare(a.path, b.path) })
	regions = slices.Compact(regions)
	plan := run.WritePlan{Base: base}
	for i, r := range regions {
		inherited := plan.Base
		for j := i - 1; j >= 0; j-- {
			if !regions[j].file && strings.HasPrefix(r.path, regions[j].path+"/") {
				inherited = regions[j].owner == manager
				break
			}
		}
		writable := r.owner == manager
		if writable == inherited {
			continue
		}
		_, isFile := tree.Lookup(mustPath(r.path))
		switch {
		case r.file && !isFile:
			holder := "a read-only overlay"
			if writable {
				holder = "a writable overlay for its owner " + manager
			}
			return run.WritePlan{}, fmt.Errorf("%w: the exact file %s needs %s, but it does not exist in the start stand and a missing file cannot be mounted; commit the file first or assign its directory instead", ErrUnsupportedOwnership, r.path, holder)
		case !r.file && isFile:
			return run.WritePlan{}, fmt.Errorf("%w: the directory area %s/ is a file in the start stand", ErrUnsupportedOwnership, r.path)
		}
		plan.Mounts = append(plan.Mounts, run.PathMount{Path: r.path, Writable: writable, File: r.file})
	}
	return plan, nil
}

func mustPath(name string) snapshot.Path {
	path, _ := snapshot.NewPath(name)
	return path
}

// Account returns the blocking findings of the File check of a stand; an
// accounted stand returns an empty slice. Findings are ordered by Owner, Path,
// Code and Message.
func (c Candidates) Account(tree snapshot.Tree) []CandidateFinding {
	findings, _ := c.account(tree)
	return findings
}

func (c Candidates) account(tree snapshot.Tree) ([]CandidateFinding, *realization.Accounting) {
	var findings []CandidateFinding
	add := func(owner, code, path, message string) {
		findings = append(findings, CandidateFinding{Owner: owner, Code: code, Path: path, Message: message})
	}
	registry, classifications := c.shards(tree, add)
	changed := make(map[string]bool)
	protected := c.protectedInputs
	if c.guard != nil {
		protected = c.guard
	}
	for _, finding := range protected(tree) {
		changed[finding.Path] = true
		findings = append(findings, finding)
	}
	var report *realization.Accounting
	if registry != nil {
		accounting := realization.AccountFiles(c.checked, tree, *registry, append(slices.Clone(c.classifications), classifications...), c.layout)
		report = &accounting
		for _, finding := range accounting.Findings() {
			path := finding.Path.String()
			if changed[path] && (finding.Code == "file.unexplained" || finding.Code == "layout.no_owner" || finding.Code == "classification.missing_file") {
				continue
			}
			add(c.findingOwner(finding), finding.Code, path, c.findingMessage(finding))
		}
	}
	slices.SortFunc(findings, func(a, b CandidateFinding) int {
		return cmp.Or(cmp.Compare(a.Owner, b.Owner), cmp.Compare(a.Path, b.Path), cmp.Compare(a.Code, b.Code), cmp.Compare(a.Message, b.Message))
	})
	return slices.Compact(findings), report
}

// RegistryView assembles the derived read-only view of a stand's local
// registries: the version of every shard it includes and the effective
// registration of every registered artifact.
func (c Candidates) RegistryView(tree snapshot.Tree) run.RegistryView {
	view := run.RegistryView{Shards: map[string]string{}}
	for _, entry := range tree.Entries() {
		if strings.HasPrefix(entry.Path().String(), registryDirectory) {
			view.Shards[entry.Path().String()] = entry.ContentID().String()
		}
	}
	_, report := c.account(tree)
	if report == nil {
		return view
	}
	for _, file := range report.Files() {
		registration, ok := file.Registration()
		if !ok || file.Class() != realization.RegisteredArtifact {
			continue
		}
		var realizes []string
		for _, id := range registration.Artifact().Basis() {
			realizes = append(realizes, identityText(id))
		}
		view.Registrations = append(view.Registrations, run.Registration{Path: file.Entry().Path().String(), Owner: ownerName(registration.Owner()), Form: string(registration.Artifact().Form()), Realizes: realizes, Entry: registration.Artifact().Selector()})
	}
	return view
}

// ModelCheck runs the Model check on the model inputs of a stand: they must
// check as passed and be the exact model the run is bound to.
func (c Candidates) ModelCheck(tree snapshot.Tree) []CandidateFinding {
	raw := cloneRaw(c.start.inputs.Raw)
	raw.Census = tree
	raw.Local = rebindModelSource(raw.Local, tree)
	checked := checkModel(raw, c.start.limits)
	switch {
	case checked.Status != model.Passed:
		var findings []CandidateFinding
		for _, d := range checked.Diagnostics {
			findings = append(findings, CandidateFinding{Owner: organization.Root, Code: "model." + d.Code, Path: d.Source.Path, Message: d.Message})
		}
		if len(findings) == 0 {
			findings = append(findings, CandidateFinding{Owner: organization.Root, Code: "model.not_passed", Message: fmt.Sprintf("The Model check of the stand is %s.", checked.Status)})
		}
		return findings
	case modelDigest(checked) != modelDigest(c.checked):
		return []CandidateFinding{{Owner: organization.Root, Code: "model.changed", Message: "The stand's model is not the model the run is bound to."}}
	}
	return nil
}

// modelDigest identifies the checked local model.
func modelDigest(checked model.Result) string {
	for _, m := range checked.Models {
		if m.Local {
			return m.ID + "@" + m.Digest
		}
	}
	return ""
}

// shards decodes the registry shards found in the stand. Their invalid
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
		if classification, err := realization.NewClassification(path, realization.MarkitectInput, "Local registry maintained by its Manager."); err == nil {
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
		for _, value := range decoded.decodeRegistry(SourceFile{File: entry, RepositoryPath: path}, owner, c.checked, tree) {
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
	// One registry entry path registered more than once is ambiguous. Every
	// registrant must fix it; accounting continues with the first one.
	slices.SortStableFunc(registrations, func(a, b registration) int {
		return cmp.Or(cmp.Compare(a.value.Artifact().Path().String(), b.value.Artifact().Path().String()), cmp.Compare(a.owner, b.owner))
	})
	var unique []realization.Registration
	for i, entry := range registrations {
		path := entry.value.Artifact().Path()
		if i > 0 && registrations[i-1].value.Artifact().Path() == path {
			add(entry.owner, "registry.duplicate", path.String(), fmt.Sprintf("The path is also registered in the shard of %q; one path has exactly one registration.", registrations[i-1].owner))
			add(registrations[i-1].owner, "registry.duplicate", path.String(), fmt.Sprintf("The path is also registered in the shard of %q; one path has exactly one registration.", entry.owner))
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
// stand changed, deleted or newly added against the captured start. Registry
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
		owner := c.OwnerOf(path)
		if owner == "" {
			owner = organization.Root
		}
		findings = append(findings, CandidateFinding{Owner: owner, Code: code, Path: path.String(), Message: "The canonical or Markitect control input was " + what + "; Managers cannot change the model or project controls. Restore it, and escalate through Root if the intent must change." + originalText(before, code, &quoted)})
	}
	for _, before := range c.tree.Entries() {
		after, found := tree.Lookup(before.Path())
		check(before.Path(), before, after, true, found)
	}
	for _, after := range tree.Entries() {
		if _, found := c.tree.Lookup(after.Path()); !found {
			check(after.Path(), snapshot.Entry{}, after, false, true)
		}
	}
	return findings
}

// maxOriginalText is the largest original protected input a finding quotes;
// maxQuotedTotal bounds all quotes of one stand, so a large deletion cannot
// flood the owner's prompt.
const (
	maxOriginalText = 8 << 10
	maxQuotedTotal  = 32 << 10
)

// originalText states the start content of a changed or deleted protected input,
// so the owner can restore the exact bytes without access to the start stand: the
// text itself when it is valid UTF-8 of at most maxOriginalText bytes and the
// stand's quotes stay within maxQuotedTotal, else its SHA-256 and size.
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
// registering Manager when there is one, else the owner of the path, else Root.
func (c Candidates) findingOwner(finding realization.Finding) string {
	if finding.Owner.IsRoot() {
		return organization.Root
	}
	if name, ok := finding.Owner.Package(); ok {
		return name
	}
	if finding.Path.String() != "" {
		if owner := c.OwnerOf(finding.Path); owner != "" {
			return owner
		}
	}
	return organization.Root
}

func (c Candidates) findingMessage(finding realization.Finding) string {
	if finding.Code != "file.unexplained" {
		return finding.Message
	}
	owner := c.OwnerOf(finding.Path)
	if owner == "" {
		return finding.Message
	}
	return fmt.Sprintf("%s Register it, or a directory holding it, in the local registry of its owner %q: %s.", finding.Message, owner, c.Shard(owner))
}

func ownerName(owner realization.Owner) string {
	if name, ok := owner.Package(); ok {
		return name
	}
	return organization.Root
}
