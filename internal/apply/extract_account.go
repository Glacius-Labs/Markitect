package apply

import (
	"cmp"
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

// extractPhase names the two phases of an Extract.
type extractPhase int

const (
	// extractStructure proposes the Packages and the layout; only Root works.
	extractStructure extractPhase = iota
	// extractModel derives the model of every Package from the artifacts and
	// registers the artifacts; the Managers of the proposed tree work.
	extractModel
)

// extractAccounting accounts the candidates of one Extract phase against that
// phase's start. It holds only immutable values: Account compiles and checks
// the tree it is given from scratch, so concurrent calls (the engine's check
// tool calls it from several Execute turns) are safe.
type extractAccounting struct {
	phase    extractPhase
	start    extractStage
	base     Candidates // the start's ownership, layout and areas
	packages []string   // the start's local Packages
}

func newExtractAccounting(phase extractPhase, start extractStage) (extractAccounting, error) {
	structural := start.structural
	base, err := newCandidates(structural, structural.files.classifications, nil, structural.registry.registry)
	if err != nil {
		return extractAccounting{}, err
	}
	return extractAccounting{phase: phase, start: start, base: base, packages: packageNames(structural.Model().compiled)}, nil
}

// OwnerOf returns the Manager that owns a path: in the structure phase Root; in
// the model phase the Package whose model directory holds the path, else the
// owner the layout and the registry shards give it.
func (a extractAccounting) OwnerOf(p snapshot.Path) string { return a.ownerOfName(p.String()) }

func (a extractAccounting) ownerOfName(name string) string {
	if a.phase == extractStructure {
		return organization.Root
	}
	if strings.HasPrefix(name, modelDirectory) {
		return modelPackageOf(a.packages, name)
	}
	p, err := snapshot.NewPath(name)
	if err != nil {
		return organization.Root
	}
	return a.base.OwnerOf(p)
}

// Account returns the blocking findings of a candidate tree of the phase. It
// compiles the candidate model, checks the layout and the Package inventory and,
// in the model phase, accounts every file against the shards; in both phases
// only the phase's own area may differ from the start.
func (a extractAccounting) Account(tree snapshot.Tree) []CandidateFinding {
	var findings []CandidateFinding
	add := func(owner, code, name, message string) {
		findings = append(findings, CandidateFinding{Owner: owner, Code: code, Path: name, Message: message})
	}
	defer func() {
		slices.SortFunc(findings, func(x, y CandidateFinding) int {
			return cmp.Or(cmp.Compare(x.Owner, y.Owner), cmp.Compare(x.Path, y.Path), cmp.Compare(x.Code, y.Code), cmp.Compare(x.Message, y.Message))
		})
		findings = slices.Compact(findings)
	}()
	// Every candidate is guarded, whatever else fails to prepare.
	guarded := a.guard(tree)
	findings = append(findings, guarded...)
	structural := rebindStructural(a.start.structural.Model(), tree, a.phase == extractStructure)
	compiled := structural.Model().compiled
	if compiled.Status != model.Passed {
		local := structural.Model().inputs.Raw.Local
		for _, d := range compiled.Diagnostics {
			name := d.Source.Path
			if d.Source.Origin == local.Source.Origin && !local.RepositoryArea.IsRoot() {
				name = local.RepositoryArea.String() + "/" + name // the compiler's paths are relative to the model's boundary
			}
			add(a.ownerOfName(name), "model."+d.Code, name, positioned(d.Message, d.Position))
		}
		if len(compiled.Diagnostics) == 0 {
			add(organization.Root, "model.not_passed", "", "The candidate model does not compile: its status is "+string(compiled.Status)+".")
		}
		return findings
	}
	for _, d := range structural.files.diagnostics {
		add(a.ownerOfName(d.Source.File), d.Code, d.Source.File, positioned(d.Message, d.Position))
	}
	if structural.execution.Status() != model.Passed {
		return findings
	}
	topology, err := organization.BuildTopology(compiled)
	if err != nil {
		add(organization.Root, "model.topology", "", err.Error())
		return findings
	}
	if a.phase == extractStructure {
		a.accountStructure(structural, topology, add)
		return findings
	}
	a.accountModel(structural, tree, guarded, &findings, add)
	return findings
}

// accountStructure checks the proposal of the structure phase: at least one
// Package, one Definition in each Package, and a layout that gives every
// existing artifact an owner.
func (a extractAccounting) accountStructure(structural CapturedPreparationAttempt, topology organization.Topology, add func(owner, code, name, message string)) {
	managers := topology.Managers()
	if len(managers) < 2 {
		add(organization.Root, "model.no_package", "", "The candidate proposes no Package. Propose the first responsibility boundaries as Packages: one directory below .markitect/model per Package, each with an overview Definition.")
	}
	for _, m := range managers {
		if m.Package != organization.Root && len(m.Definitions) == 0 {
			add(organization.Root, "model.empty_package", modelDirectory+m.Package, fmt.Sprintf("Package %q has no Definition. Every proposed Package needs its overview Definition describing the responsibility; an intermediate Package needs one too.", m.Package))
		}
	}
	layout := structural.files.layout
	byDirectory := make(map[string][]string)
	for _, p := range artifactPaths(a.start.tree) {
		if !layout.HasOwner(p) {
			byDirectory[path.Dir(p.String())] = append(byDirectory[path.Dir(p.String())], p.String())
		}
	}
	for dir, files := range byDirectory {
		shown := files[:min(len(files), 3)]
		add(organization.Root, "layout.no_owner", dir, fmt.Sprintf("%d artifact file(s) in %s have no owner under the layout (for example %s). Propose a Package whose path mirrors the directory below its artifact root, or add a layout exception for the directory to layout.exceptions of .markitect/execution.yaml with the owning Package and a reason.", len(files), dir, strings.Join(shown, ", ")))
	}
}

// accountModel accounts the model phase: the Package inventory of the start is
// kept, and every file is explained, registered in its owner's shard and linked
// to existing Definitions of the candidate model.
func (a extractAccounting) accountModel(structural CapturedPreparationAttempt, tree snapshot.Tree, guarded []CandidateFinding, findings *[]CandidateFinding, add func(owner, code, name, message string)) {
	candidate := packageNames(structural.Model().compiled)
	for _, name := range candidate {
		if !slices.Contains(a.packages, name) {
			add(modelPackageOf(a.packages, modelDirectory+name+"/x"), "model.structure_changed", modelDirectory+name, fmt.Sprintf("Package %q is new. The Manager tree is stable within an Extract round: describe the missing responsibility in your own Package and propose the new Package to Root as a finding for the next round.", name))
		}
	}
	for _, name := range a.packages {
		if !slices.Contains(candidate, name) {
			add(organization.Root, "model.structure_changed", modelDirectory+name, fmt.Sprintf("Package %q has no Definition left. Every Package of the proposed tree keeps its Definitions; restore them.", name))
		}
	}
	c, err := newCandidates(structural, structural.files.classifications, nil, structural.registry.registry)
	if err != nil {
		add(organization.Root, "model.topology", "", err.Error())
		return
	}
	// The guard's findings are in findings already; Account's compaction drops
	// the repeat, and Candidates uses their paths to suppress file.unexplained.
	c.guard = func(snapshot.Tree) []CandidateFinding { return guarded }
	*findings = append(*findings, c.Account(tree)...)
}

func positioned(message string, position *model.Position) string {
	if position == nil || position.Line == 0 {
		return message
	}
	return fmt.Sprintf("%s (line %d)", message, position.Line)
}

// guard reports every change outside the phase's writable area: artifacts,
// dependencies, manifest, lock and project configuration stay as captured. The
// structure phase may write the model and the layout exceptions of the
// execution document, the model phase the model and the registry shards.
func (a extractAccounting) guard(tree snapshot.Tree) []CandidateFinding {
	var findings []CandidateFinding
	quoted := 0
	writable := func(name string) bool {
		if strings.HasPrefix(name, modelDirectory) {
			return true
		}
		if a.phase == extractStructure {
			return name == executionFile
		}
		return strings.HasPrefix(name, registryDirectory)
	}
	report := func(name string, before snapshot.Entry, code, what string) {
		kind, rule := "The existing artifact", "Extract never changes artifacts."
		if strings.HasPrefix(name, markitectDir) {
			kind, rule = "The canonical or Markitect control input", "Managers cannot change the model's dependencies or the project controls."
		}
		message := fmt.Sprintf("%s was %s; %s Restore it, and ask the owner through Root if a change is really needed.", kind, what, rule) + originalText(before, code, &quoted)
		findings = append(findings, CandidateFinding{Owner: a.ownerOfName(name), Code: code, Path: name, Message: message})
	}
	for _, before := range a.start.tree.Entries() {
		name := before.Path().String()
		after, found := tree.Lookup(before.Path())
		switch {
		case a.phase == extractStructure && name == executionFile:
			if !found {
				report(name, before, "input.deleted", "deleted")
			} else if after != before && !equalBesidesLayoutExceptions(before.Bytes(), after.Bytes()) {
				report(name, before, "input.changed", "changed beyond layout.exceptions")
			}
		case writable(name):
		case !found:
			report(name, before, "input.deleted", "deleted")
		case after != before:
			report(name, before, "input.changed", "changed")
		}
	}
	for _, after := range tree.Entries() {
		name := after.Path().String()
		if _, found := a.start.tree.Lookup(after.Path()); !found && !writable(name) {
			report(name, snapshot.Entry{}, "input.added", "added")
		}
	}
	return findings
}

// equalBesidesLayoutExceptions reports whether two execution documents are
// equal after layout.exceptions is removed from both. A document that does not
// parse differs.
func equalBesidesLayoutExceptions(before, after []byte) bool {
	strip := func(data []byte) (map[string]any, bool) {
		var document map[string]any
		if err := yaml.Unmarshal(data, &document); err != nil || document == nil {
			return nil, false
		}
		if layout, ok := document["layout"].(map[string]any); ok {
			delete(layout, "exceptions")
		}
		return document, true
	}
	x, okX := strip(before)
	y, okY := strip(after)
	return okX && okY && reflect.DeepEqual(x, y)
}

// WritePlan returns where a Manager's Execute turns may write in the shared
// working tree. Every plan starts read-only. The structure phase's Root writes
// .markitect except the dependency models and the artifact roots, which are
// read-only mounts (the manifest, the lock and the execution document are files
// that a mount cannot cover; accounting guards them). In the model phase every
// Manager writes only its own Package's model directory and its registry shard,
// minus its descendants' directories.
func (a extractAccounting) WritePlan(manager string) run.WritePlan {
	if a.phase == extractStructure {
		return a.structurePlan()
	}
	regions := []writeRegion{{strings.TrimSuffix(registryDirectory, "/"), organization.Root}}
	for _, name := range a.packages {
		regions = append(regions, writeRegion{registryDirectory + name, name}, writeRegion{strings.TrimSuffix(modelDirectory, "/") + "/" + name, name})
	}
	return regionPlan(manager, false, regions)
}

func (a extractAccounting) structurePlan() run.WritePlan {
	plan := run.WritePlan{Enforced: true, Mounts: []run.PathMount{{Path: strings.TrimSuffix(markitectDir, "/"), Writable: true}}}
	raw := a.start.structural.Model().inputs.Raw
	var readOnly []string
	for _, dependency := range raw.Dependencies {
		if area := dependency.RepositoryArea; area.String() != "" && !area.IsRoot() {
			readOnly = append(readOnly, area.String())
		}
	}
	for _, root := range a.base.layout.Roots() {
		readOnly = append(readOnly, root.String())
	}
	slices.Sort(readOnly)
	for _, name := range slices.Compact(readOnly) {
		if p, err := snapshot.NewPath(name); err == nil {
			if _, isFile := a.start.tree.Lookup(p); !isFile {
				plan.Mounts = append(plan.Mounts, run.PathMount{Path: name})
			}
		}
	}
	return plan
}
