package host

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/render"
)

type Impact struct {
	Base                     string               `yaml:"base"`
	Candidate                string               `yaml:"candidate"`
	Changed                  []string             `yaml:"changed"`
	Affected                 []string             `yaml:"affected"`
	Reason                   string               `yaml:"reason"`
	Causes                   []ImpactCause        `yaml:"causes,omitempty"`
	Analysis                 *AnalysisEvidence    `yaml:"analysis,omitempty"`
	PolicyChanges            []PolicyResultChange `yaml:"policyChanges,omitempty"`
	DirectPolicySubjects     []string             `yaml:"directPolicySubjects,omitempty"`
	DirectPolicySubjectCount *int                 `yaml:"directPolicySubjectCount,omitempty"`
	AffectedCount            *int                 `yaml:"affectedCount,omitempty"`
}

// ImpactCause records a changed input or an invalidation relationship in the
// affected closure. A global cause has an empty Resource and applies to all.
type ImpactCause struct {
	Kind             string `yaml:"kind"`
	Path             string `yaml:"path,omitempty"`
	Resource         string `yaml:"resource,omitempty"`
	From             string `yaml:"from,omitempty"`
	To               string `yaml:"to,omitempty"`
	Relation         string `yaml:"relation,omitempty"`
	Constraint       string `yaml:"constraint,omitempty"`
	DomainAPIVersion string `yaml:"domainApiVersion,omitempty"`
	Line             int    `yaml:"line,omitempty"`
	Snapshot         string `yaml:"snapshot,omitempty"`
}

func Changes(before, after *Project) *Impact {
	result := &Impact{Base: before.Snapshot.ID, Candidate: after.Snapshot.ID, Reason: "Union of old and new invalidation closures; configuration, Domain definitions, inventory, collection policies, uncertain context effects or unowned inputs conservatively affect all resources. See causes for this comparison."}
	result.Changed = snapshot.Compare(before.Snapshot, after.Snapshot).Paths()
	changed := map[string]bool{}
	for _, name := range result.Changed {
		changed[name] = true
	}
	all := false
	seeds := map[string]bool{}
	changedInputs := map[string]bool{}
	policySubjects := map[string]bool{}
	ownedInputs := map[string]map[string]bool{}
	causes := map[string]ImpactCause{}
	addCause := func(cause ImpactCause) {
		key := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%d\x00%s", cause.Kind, cause.Path, cause.Resource, cause.From, cause.To, cause.Relation, cause.Constraint, cause.DomainAPIVersion, cause.Line, cause.Snapshot)
		causes[key] = cause
	}
	for _, p := range []*Project{before, after} {
		for _, r := range p.Resources {
			if r.Package == "" && changed[r.Path] {
				seeds[r.GraphKey()] = true
				changedInputs[r.GraphKey()] = true
				addCause(ImpactCause{Kind: "resource-change", Path: r.Path, Resource: r.GraphKey()})
				if r.Kind == "Project" {
					all = true
					addCause(ImpactCause{Kind: "configuration", Path: r.Path})
				} else if constraintReadsCollection(p, r) {
					all = true
					addCause(ImpactCause{Kind: "constraint-selection", Path: r.Path})
				} else if constraintReadsLegacyResource(p, r) && hasContextWithoutInvalidation(before, after) {
					all = true
					addCause(ImpactCause{Kind: "context-policy-effect", Path: r.Path})
				}
			}
		}
		fileOwners, err := impactFileOwners(p)
		if err != nil {
			all = true
			addCause(ImpactCause{Kind: "ownership-analysis"})
			continue
		}
		for file, owners := range fileOwners {
			for owner := range owners {
				if ownedInputs[file] == nil {
					ownedInputs[file] = map[string]bool{}
				}
				ownedInputs[file][owner] = true
			}
		}
	}
	// New/deleted symbols may affect global constraints and scope-level discovery.
	if len(before.Graph.Resources) != len(after.Graph.Resources) {
		all = true
		addCause(ImpactCause{Kind: "inventory"})
	}
	for k := range before.Graph.Resources {
		if _, ok := after.Graph.Resources[k]; !ok {
			all = true
			path := before.Graph.Resources[k].Path
			addCause(ImpactCause{Kind: "inventory", Path: path})
		}
	}
	for k := range after.Graph.Resources {
		if _, ok := before.Graph.Resources[k]; !ok {
			all = true
			path := after.Graph.Resources[k].Path
			addCause(ImpactCause{Kind: "inventory", Path: path})
		}
	}
	// Unmodelled inputs may contain normative contracts, router membership or
	// checker code. Until their independence is declared, never reuse evidence.
	modelled := map[string]bool{"markitect.yaml": true}
	for _, p := range []*Project{before, after} {
		for _, r := range p.Resources {
			if r.Package == "" {
				modelled[r.Path] = true
			}
		}
	}
	for name := range changed {
		if selectedLocalDomainInput(before, name) || selectedLocalDomainInput(after, name) {
			all = true
			addCause(ImpactCause{Kind: "domain-definition", Path: name})
		} else if owners := ownedInputs[name]; len(owners) > 0 {
			for owner := range owners {
				seeds[owner] = true
				kind := "generated-output"
				if declaredInputOwned(before, owner, name) || declaredInputOwned(after, owner, name) {
					kind = "declared-input"
					changedInputs[owner] = true
				}
				addCause(ImpactCause{Kind: kind, Path: name, Resource: owner})
			}
		} else if !modelled[name] {
			all = true
			addCause(ImpactCause{Kind: "unowned-input", Path: name})
		}
	}
	// Same-target policies read the subject and every resolved path resource.
	// Union old and new dependencies so rewires and newly incomplete paths keep
	// their valid-prefix evidence. Only actual changed canonical inputs trigger
	// the policy subject; ordinary affected closure membership is not a read.
	if !all {
		for _, candidate := range []struct {
			name string
			p    *Project
		}{{"base", before}, {"candidate", after}} {
			p := candidate.p
			if p == nil || p.Graph == nil {
				continue
			}
			for _, dependency := range p.Graph.PolicyDependencies {
				if !changedInputs[dependency.Input] {
					continue
				}
				policySubjects[dependency.Subject] = true
				seeds[dependency.Subject] = true
				addCause(ImpactCause{Kind: "policy-dependency", Path: dependency.Path, Resource: dependency.Subject, To: dependency.Input, Relation: dependency.Relation, Constraint: dependency.Constraint, DomainAPIVersion: dependency.APIVersion, Line: dependency.Line, Snapshot: candidate.name})
			}
		}
	}
	// Compute context consumers independently per snapshot. Mixing edges from
	// the base and candidate graphs could otherwise invent a path that existed
	// in neither revision.
	if !all && len(policySubjects) > 0 {
		for _, candidate := range []struct {
			name string
			p    *Project
		}{{"base", before}, {"candidate", after}} {
			closure := map[string]bool{}
			for subject := range policySubjects {
				closure[subject] = true
			}
			for again := true; again; {
				again = false
				for _, relationship := range candidate.p.Graph.Relationships {
					if !relationship.Context || !closure[relationship.To] {
						continue
					}
					if !closure[relationship.From] {
						closure[relationship.From] = true
						again = true
					}
					seeds[relationship.From] = true
					addCause(ImpactCause{Kind: "policy-context", Path: relationship.Path, Resource: relationship.From, From: relationship.From, To: relationship.To, Relation: relationship.Relation, DomainAPIVersion: relationship.DomainAPIVersion, Line: relationship.Line, Snapshot: candidate.name})
				}
			}
		}
	}
	if all {
		if len(causes) == 0 {
			addCause(ImpactCause{Kind: "global"})
		}
		for _, p := range []*Project{before, after} {
			for k := range p.Graph.Resources {
				seeds[k] = true
			}
		}
	}
	for again := !all && len(seeds) > 0; again; {
		again = false
		for _, candidate := range []struct {
			name string
			p    *Project
		}{{"base", before}, {"candidate", after}} {
			fromKeys := make([]string, 0, len(candidate.p.Graph.InvalidationEdges))
			for from := range candidate.p.Graph.InvalidationEdges {
				fromKeys = append(fromKeys, from)
			}
			sort.Strings(fromKeys)
			for _, from := range fromKeys {
				edges := append([]string(nil), candidate.p.Graph.InvalidationEdges[from]...)
				sort.Strings(edges)
				for _, to := range edges {
					if !seeds[to] {
						continue
					}
					if !seeds[from] {
						seeds[from] = true
						again = true
					}
					for _, relationship := range candidate.p.Graph.Relationships {
						if relationship.From != from || relationship.To != to || !relationship.Invalidate {
							continue
						}
						addCause(ImpactCause{Kind: "invalidation", Path: relationship.Path, From: from, To: to, Resource: from, Relation: relationship.Relation, DomainAPIVersion: relationship.DomainAPIVersion, Line: relationship.Line, Snapshot: candidate.name})
					}
				}
			}
		}
	}
	for k := range seeds {
		result.Affected = append(result.Affected, k)
	}
	sort.Strings(result.Affected)
	for _, cause := range causes {
		result.Causes = append(result.Causes, cause)
	}
	sort.Slice(result.Causes, func(i, j int) bool { return impactCauseLess(result.Causes[i], result.Causes[j]) })
	return result
}

// constraintReadsResource reports whether the resource is a subject of any
// policy in this snapshot. Collection-wide invalidation is classified
// separately; this broader check guards per-resource outcomes whose consumers
// may be context-visible without an invalidation edge.
func constraintReadsResource(p *Project, resource *core.Resource) bool {
	if p == nil || p.Graph == nil || p.Graph.Registry == nil || resource == nil {
		return false
	}
	for _, domain := range p.Graph.Registry.Domains() {
		if resource.APIVersion != domain.APIVersion {
			continue
		}
		for _, constraint := range domain.Constraints {
			selector := constraint.Select
			if selector.Kind != "" && selector.Kind != resource.Kind {
				continue
			}
			matches := true
			for name, value := range selector.Labels {
				actual, exists := resource.Metadata.Labels[name]
				if !exists || actual != value {
					matches = false
					break
				}
			}
			if matches {
				return true
			}
		}
	}
	return false
}

// constraintReadsLegacyResource scopes the conservative context-without-
// invalidation fallback to operators whose dependency behavior is not yet
// represented by explicit graph evidence. same-target has PolicyDependencies
// and receives an exact context closure.
func constraintReadsLegacyResource(p *Project, resource *core.Resource) bool {
	if p == nil || p.Graph == nil || p.Graph.Registry == nil || resource == nil {
		return false
	}
	for _, domain := range p.Graph.Registry.Domains() {
		if resource.APIVersion != domain.APIVersion {
			continue
		}
		for _, constraint := range domain.Constraints {
			if constraint.Assert.Op == "same-target" {
				continue
			}
			selector := constraint.Select
			if selector.Kind != "" && selector.Kind != resource.Kind {
				continue
			}
			matches := true
			for name, value := range selector.Labels {
				actual, exists := resource.Metadata.Labels[name]
				if !exists || actual != value {
					matches = false
					break
				}
			}
			if matches {
				return true
			}
		}
	}
	return false
}

// constraintReadsCollection identifies assertions whose outcome depends on
// the selected set as a whole. Per-resource assertions are bounded by their
// subject and its declared invalidation dependents.
func constraintReadsCollection(p *Project, resource *core.Resource) bool {
	if p == nil || p.Graph == nil || p.Graph.Registry == nil || resource == nil {
		return false
	}
	for _, domain := range p.Graph.Registry.Domains() {
		if resource.APIVersion != domain.APIVersion {
			continue
		}
		for _, constraint := range domain.Constraints {
			a := constraint.Assert
			global := a.Op == "unique" || a.Op == "count" && a.Scope != "resource"
			if !global {
				continue
			}
			selector := constraint.Select
			if selector.Kind != "" && selector.Kind != resource.Kind {
				continue
			}
			if a.Op == "count" && a.Relation != "" && !containsString(domain.Relations[a.Relation].SourceKinds, resource.Kind) {
				continue
			}
			matches := true
			for name, value := range selector.Labels {
				actual, exists := resource.Metadata.Labels[name]
				if !exists || actual != value {
					matches = false
					break
				}
			}
			if matches {
				return true
			}
		}
	}
	return false
}

func selectedLocalDomainInput(p *Project, name string) bool {
	if p == nil {
		return false
	}
	for _, input := range p.DomainInputs {
		if input.Package == "" && input.Path == name {
			return true
		}
	}
	return false
}

func declaredInputOwned(p *Project, owner, name string) bool {
	if p == nil || p.Graph == nil {
		return false
	}
	resource := p.Graph.Resources[owner]
	if resource == nil {
		return false
	}
	for _, input := range resource.Spec.Files {
		if input == name {
			return true
		}
	}
	for _, input := range p.InputFiles[owner] {
		if input == name {
			return true
		}
	}
	return false
}

func hasContextWithoutInvalidation(projects ...*Project) bool {
	for _, p := range projects {
		if p == nil || p.Graph == nil {
			continue
		}
		for _, relationship := range p.Graph.Relationships {
			if relationship.Context && !relationship.Invalidate {
				return true
			}
		}
	}
	return false
}

func impactCauseLess(a, b ImpactCause) bool {
	left := []string{a.Kind, a.Path, a.Resource, a.From, a.To, a.DomainAPIVersion, a.Relation, a.Constraint, fmt.Sprint(a.Line), a.Snapshot}
	right := []string{b.Kind, b.Path, b.Resource, b.From, b.To, b.DomainAPIVersion, b.Relation, b.Constraint, fmt.Sprint(b.Line), b.Snapshot}
	return strings.Join(left, "\x00") < strings.Join(right, "\x00")
}

// impactFileOwners maps only paths whose content is already represented by a
// typed resource or by an explicit Spec.Files declaration. Unknown generated
// files remain broad-impact inputs until their ownership is modeled.
func impactFileOwners(p *Project) (map[string]map[string]bool, error) {
	owners := map[string]map[string]bool{}
	add := func(file, key string) {
		if file == "" || key == "" {
			return
		}
		if owners[file] == nil {
			owners[file] = map[string]bool{}
		}
		owners[file][key] = true
	}
	if p == nil {
		return owners, nil
	}
	for _, r := range p.Resources {
		if r == nil || r.Kind == "Project" || r.Package != "" {
			continue
		}
		key := r.GraphKey()
		for _, file := range r.Spec.Files {
			add(file, key)
		}
		for _, file := range p.InputFiles[key] {
			add(file, key)
		}
	}
	_, generatedOwners, err := render.GenerateWithOwners(p.Graph, p.Snapshot.Files)
	if err != nil {
		return nil, err
	}
	for file, resources := range generatedOwners {
		for _, resource := range resources {
			add(file, resource)
		}
	}
	return owners, nil
}
