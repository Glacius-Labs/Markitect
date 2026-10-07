package government

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
)

// Order is a request against a previously active Constitution. This input is
// selected by the caller; a matching digest authenticates neither owner nor agent.
type Order struct {
	APIVersion         string                    `json:"apiVersion" yaml:"apiVersion"`
	Kind               string                    `json:"kind" yaml:"kind"`
	Purpose            string                    `json:"purpose" yaml:"purpose"`
	ActiveConstitution string                    `json:"activeConstitution" yaml:"activeConstitution"`
	Action             string                    `json:"action" yaml:"action"`
	Subjects           []core.DefinitionIdentity `json:"subjects" yaml:"subjects"`
	Paths              []string                  `json:"paths,omitempty" yaml:"paths,omitempty"`
	UnknownScope       bool                      `json:"unknownScope,omitempty" yaml:"unknownScope,omitempty"`
}

type ArtifactObservation struct {
	Path     string                    `json:"path" yaml:"path"`
	Class    string                    `json:"class" yaml:"class"`
	Purpose  string                    `json:"purpose" yaml:"purpose"`
	Writer   *core.DefinitionIdentity  `json:"writer,omitempty" yaml:"writer,omitempty"`
	Subjects []core.DefinitionIdentity `json:"subjects" yaml:"subjects"`
	Observed *inventory.Entry          `json:"observed,omitempty" yaml:"observed,omitempty"`
}

type Survey struct {
	Observation        inventory.Report          `json:"observation" yaml:"observation"`
	Artifacts          []ArtifactObservation     `json:"artifacts" yaml:"artifacts"`
	Unknown            []string                  `json:"unknown" yaml:"unknown"`
	Missing            []string                  `json:"missing" yaml:"missing"`
	Unavailable        []string                  `json:"unavailable" yaml:"unavailable"`
	UnassignedSubjects []core.DefinitionIdentity `json:"unassignedSubjects" yaml:"unassignedSubjects"`
	UnrealizedSubjects []core.DefinitionIdentity `json:"unrealizedSubjects" yaml:"unrealizedSubjects"`
	Coverage           string                    `json:"coverage" yaml:"coverage"`
}

// SurveyRepository joins normative roles with observed facts in both directions.
// It never promotes the observed bytes to accepted intent or semantic evidence.
func SurveyRepository(m Model, report inventory.Report) Survey {
	s := Survey{Observation: report, Coverage: "accounted-within-declared-boundary"}
	declared := map[string]core.Definition{}
	subjects := map[string][]core.DefinitionIdentity{}
	assigned := map[string]bool{}
	realized := map[string]bool{}
	for _, d := range m.Canonical.Definitions {
		if d.APIVersion != APIVersion {
			continue
		}
		if d.Kind == "Artifact" {
			declared[fmt.Sprint(d.Spec["path"])] = d
		}
		if d.Kind == "Responsibility" {
			assigned[identity(d.Spec["subject"]).Key()] = true
		}
		if d.Kind == "Realization" {
			a := m.byID[identity(d.Spec["artifact"]).Key()]
			p := fmt.Sprint(a.Spec["path"])
			subjects[p] = append(subjects[p], identity(d.Spec["subject"]))
			if a.Spec["class"] != "foreign" {
				realized[identity(d.Spec["subject"]).Key()] = true
			}
		}
	}
	for _, d := range m.Canonical.Definitions {
		if d.APIVersion == APIVersion {
			continue
		}
		if !assigned[d.Identity().Key()] {
			s.UnassignedSubjects = append(s.UnassignedSubjects, d.Identity())
		}
		if !realized[d.Identity().Key()] {
			s.UnrealizedSubjects = append(s.UnrealizedSubjects, d.Identity())
		}
	}
	sortIDs(s.UnassignedSubjects)
	sortIDs(s.UnrealizedSubjects)
	seen := map[string]bool{}
	for _, entry := range report.Entries {
		e := entry
		seen[e.Path] = true
		a := ArtifactObservation{Path: e.Path, Class: "unknown", Purpose: "unknown", Observed: &e, Subjects: subjects[e.Path]}
		if d, ok := declared[e.Path]; ok {
			a.Class = fmt.Sprint(d.Spec["class"])
			a.Purpose = d.Purpose
			if d.Spec["writer"] != nil {
				id := identity(d.Spec["writer"])
				a.Writer = &id
			}
		}
		gitMetadata := e.Reason == ".git is Git metadata"
		if e.Status == "excluded" || gitMetadata {
			a.Class = "excluded"
			a.Purpose = e.Reason
		}
		// Directories carry boundary metadata rather than invented obligations.
		if e.Type == "directory" && e.Status == "observed" {
			continue
		}
		if a.Class == "unknown" || e.Status == "unreadable" || (e.Status == "boundary" && !gitMetadata) {
			s.Unknown = append(s.Unknown, e.Path)
		}
		sortIDs(a.Subjects)
		s.Artifacts = append(s.Artifacts, a)
	}
	for p, d := range declared {
		if seen[p] {
			continue
		}
		a := ArtifactObservation{Path: p, Class: fmt.Sprint(d.Spec["class"]), Purpose: d.Purpose, Subjects: subjects[p]}
		if d.Spec["writer"] != nil {
			id := identity(d.Spec["writer"])
			a.Writer = &id
		}
		sortIDs(a.Subjects)
		// An excluded or linked ancestor is not evidence that its descendants
		// are absent. Preserve the actual observed boundary and block the input.
		for i := range report.Entries {
			e := report.Entries[i]
			if e.Status != "observed" && (e.Path == "." || strings.HasPrefix(p, e.Path+"/")) {
				a.Observed = &e
				break
			}
		}
		if a.Observed != nil || !withinRoots(p, report.Roots) {
			s.Unavailable = append(s.Unavailable, p)
		} else {
			s.Missing = append(s.Missing, p)
		}
		s.Artifacts = append(s.Artifacts, a)
	}
	if !report.Complete || len(s.Unknown) > 0 || len(s.Missing) > 0 || len(s.Unavailable) > 0 || len(s.UnassignedSubjects) > 0 || len(s.UnrealizedSubjects) > 0 || len(m.Findings) > 0 {
		s.Coverage = "incomplete"
	}
	sort.Strings(s.Unknown)
	sort.Strings(s.Missing)
	sort.Strings(s.Unavailable)
	sort.Slice(s.Artifacts, func(i, j int) bool { return s.Artifacts[i].Path < s.Artifacts[j].Path })
	return s
}

type Work struct {
	Area     core.DefinitionIdentity   `json:"area" yaml:"area"`
	Subjects []core.DefinitionIdentity `json:"subjects" yaml:"subjects"`
	Paths    []string                  `json:"paths" yaml:"paths"`
	Mandates []core.DefinitionIdentity `json:"mandates" yaml:"mandates"`
}

type Plan struct {
	APIVersion         string                    `json:"apiVersion" yaml:"apiVersion"`
	Action             string                    `json:"action,omitempty" yaml:"action,omitempty"`
	ModelPath          string                    `json:"modelPath,omitempty" yaml:"modelPath,omitempty"`
	Status             string                    `json:"status" yaml:"status"`
	Digest             string                    `json:"digest" yaml:"digest"`
	ActiveConstitution string                    `json:"activeConstitution" yaml:"activeConstitution"`
	OrderDigest        string                    `json:"orderDigest" yaml:"orderDigest"`
	InventoryDigest    string                    `json:"inventoryDigest" yaml:"inventoryDigest"`
	Provisional        bool                      `json:"provisional" yaml:"provisional"`
	Affected           []core.DefinitionIdentity `json:"affected" yaml:"affected"`
	Work               []Work                    `json:"work" yaml:"work"`
	IntegrationReviews []core.DefinitionIdentity `json:"integrationReviews" yaml:"integrationReviews"`
	Cabinet            []core.DefinitionIdentity `json:"cabinet" yaml:"cabinet"`
	Survey             Survey                    `json:"survey" yaml:"survey"`
	Findings           []Finding                 `json:"findings" yaml:"findings"`
	Limits             []string                  `json:"limits" yaml:"limits"`
}

// BuildPlan is conservative and read-only. Edges in either direction and shared
// realization files broaden the work; unknown scope widens to every subject and
// root review. No absence of evidence turns into a successful empty plan.
func BuildPlan(m Model, order Order, report inventory.Report, modelPaths ...string) Plan {
	p := Plan{APIVersion: "markitect.government-plan/v1alpha1", Status: "planned-scoped", ActiveConstitution: m.Digest, OrderDigest: Digest(order), InventoryDigest: report.Digest, Provisional: report.Provisional, Cabinet: m.Cabinet, Survey: SurveyRepository(m, report), Findings: append([]Finding(nil), m.Findings...), Limits: []string{
		"Read-only proposal; no execution, independent agent review, votes, acceptance or promotion occurred.",
		"Native observation is provisional, not an atomic Git revision; G2 must rebind immutable inputs and revalidate prior authority.",
		"Declared assignments and purposes are structural claims, not proof of implementation semantics or sufficient cabinet selection.",
		"Cooperative process shares caller OS rights; worktree and path checks are not a sandbox.",
	}}
	add := func(code, subject, detail string) { p.Findings = append(p.Findings, Finding{code, subject, detail}) }
	if order.Action == "amend-model" {
		p.Action = order.Action
		if len(modelPaths) != 1 || !SafePath(modelPaths[0]) {
			add("amendment.source-path", "", "amend-model requires one exact caller-selected Government source path")
		} else {
			p.ModelPath = modelPaths[0]
		}
	}
	if order.APIVersion != OrderVersion || order.Kind != "Order" || strings.TrimSpace(order.Purpose) == "" {
		add("order.invalid", "", "version, kind Order and purpose are required")
	}
	if !validDigest(order.ActiveConstitution) || order.ActiveConstitution != m.Digest {
		add("order.stale", "", "order must bind the exact selected prior Constitution digest")
	}
	if order.Action != "implement" && order.Action != "amend-model" {
		add("order.action", "", "expected implement or amend-model")
	}
	if len(m.Findings) > 0 {
		p.Status = "blocked"
		return finishPlan(p)
	}
	affected := map[string]bool{}
	for _, id := range order.Subjects {
		if _, ok := m.byID[id.Key()]; !ok {
			add("order.subject", id.Key(), "unknown exact identity")
		} else {
			affected[id.Key()] = true
		}
	}
	for _, path := range order.Paths {
		if !SafePath(path) {
			add("order.path", path, "unsafe path")
			continue
		}
		found := false
		for _, a := range p.Survey.Artifacts {
			if a.Path == path {
				for _, id := range a.Subjects {
					affected[id.Key()] = true
				}
				found = len(a.Subjects) > 0
			}
		}
		if !found {
			add("order.unmapped-path", path, "no modeled realization; root must resolve scope")
		}
	}
	if order.UnknownScope {
		for key, d := range m.byID {
			if d.APIVersion != APIVersion {
				affected[key] = true
			}
		}
		add("scope.unknown", m.Root.Key(), "scope widened to all modeled subjects and root integration; unresolved scope requires clarification")
	}
	if len(affected) == 0 {
		add("order.empty", "", "no resolved work; cannot report no-op success")
	}
	changed := true
	for changed {
		changed = false
		include := func(key string) {
			if !affected[key] {
				affected[key] = true
				changed = true
			}
		}
		for _, e := range m.Canonical.Edges {
			if m.byID[e.From].APIVersion == APIVersion || m.byID[e.To].APIVersion == APIVersion {
				continue
			}
			if affected[e.From] || affected[e.To] {
				include(e.From)
				include(e.To)
			}
		}
		for _, a := range p.Survey.Artifacts {
			shared := false
			for _, id := range a.Subjects {
				if affected[id.Key()] {
					shared = true
				}
			}
			if shared {
				for _, id := range a.Subjects {
					include(id.Key())
				}
			}
		}
	}
	work := map[string]*Work{}
	modelSelected := false
	reviews := map[string]bool{}
	if order.UnknownScope {
		reviews[m.Root.Key()] = true
	}
	realized := map[string]bool{}
	getWork := func(area core.DefinitionIdentity) *Work {
		key := area.Key()
		if work[key] == nil {
			work[key] = &Work{Area: area}
		}
		return work[key]
	}
	for key := range affected {
		d := m.byID[key]
		p.Affected = append(p.Affected, d.Identity())
		if d.APIVersion == APIVersion {
			add("authority.protected", key, "organizational authority, root goals and approval changes require trusted owner escalation")
			continue
		}
		owner := core.DefinitionIdentity{}
		for _, r := range m.Canonical.Definitions {
			if r.APIVersion == APIVersion && r.Kind == "Responsibility" && identity(r.Spec["subject"]).Key() == key {
				owner = identity(r.Spec["area"])
			}
		}
		if owner.Name == "" {
			add("responsibility.unknown", key, "no accountable Area; root review required")
			reviews[m.Root.Key()] = true
			continue
		}
		w := getWork(owner)
		w.Subjects = append(w.Subjects, d.Identity())
		for current := owner.Key(); current != ""; current = optionalIdentity(m.byID[current].Spec["parent"]) {
			reviews[current] = true
		}
		if !authorize(m, owner, d.Identity(), order.Action, w) {
			add("authority.missing", key, "prior mandate for responsible Area does not allow "+order.Action)
		}
	}
	for _, a := range p.Survey.Artifacts {
		selected := false
		for _, id := range a.Subjects {
			if affected[id.Key()] {
				selected = true
			}
		}
		if !selected {
			continue
		}
		if a.Class == "foreign" {
			continue
		}
		for _, id := range a.Subjects {
			if affected[id.Key()] {
				realized[id.Key()] = true
			}
		}
		if a.Writer == nil {
			add("writer.unknown", a.Path, "selected path has no unique writer")
			continue
		}
		w := getWork(*a.Writer)
		writerAction := "implement"
		if p.ModelPath != "" && a.Path == p.ModelPath {
			modelSelected = true
			writerAction = "amend-model"
			if a.Class != "canonical" {
				add("amendment.source-class", a.Path, "Government source must be a prior-declared canonical Artifact")
			}
		}
		w.Paths = append(w.Paths, a.Path)
		for _, id := range a.Subjects {
			if !subsetIDs([]core.DefinitionIdentity{id}, w.Subjects) {
				w.Subjects = append(w.Subjects, id)
			}
		}
		for current := a.Writer.Key(); current != ""; current = optionalIdentity(m.byID[current].Spec["parent"]) {
			reviews[current] = true
		}
		for _, id := range a.Subjects {
			if !authorize(m, *a.Writer, id, writerAction, w) {
				add("writer.unauthorized", a.Path, "writer lacks prior "+writerAction+" mandate for "+id.Key())
			}
		}
		if a.Observed != nil && (a.Observed.Status != "observed" || a.Observed.Digest == "") {
			add("input.unavailable", a.Path, "selected bytes are excluded, boundary, ignored or unreadable")
		}
		if a.Observed == nil && !withinRoots(a.Path, report.Roots) {
			add("input.outside-boundary", a.Path, "target not observed and outside declared acquisition roots")
		}
	}
	if order.Action == "amend-model" && !modelSelected {
		add("amendment.source-unmapped", p.ModelPath, "selected Government source must have an affected prior realization and exactly one canonical writer")
	}
	for key := range affected {
		if !realized[key] && m.byID[key].APIVersion != APIVersion {
			add("realization.missing", key, "no managed intended artifact relation; foreign inputs alone cannot fulfill implementation work")
		}
	}
	for key := range reviews {
		p.IntegrationReviews = append(p.IntegrationReviews, m.byID[key].Identity())
	}
	for _, w := range work {
		sortIDs(w.Subjects)
		sortIDs(w.Mandates)
		sort.Strings(w.Paths)
		p.Work = append(p.Work, *w)
	}
	sort.Slice(p.Work, func(i, j int) bool { return p.Work[i].Area.Key() < p.Work[j].Area.Key() })
	sortIDs(p.Affected)
	sortIDs(p.IntegrationReviews)
	if len(p.Findings) > 0 {
		p.Status = "blocked"
	} else if p.Survey.Coverage != "incomplete" {
		p.Status = "planned-within-declared-boundary"
	}
	return finishPlan(p)
}

func authorize(m Model, area, subject core.DefinitionIdentity, action string, w *Work) bool {
	allowed := false
	for _, d := range m.Canonical.Definitions {
		if d.APIVersion == APIVersion && d.Kind == "Mandate" && identity(d.Spec["area"]).Key() == area.Key() && contains(stringsList(d.Spec["actions"]), action) && subsetIDs([]core.DefinitionIdentity{subject}, identities(d.Spec["scope"])) {
			allowed = true
			seen := false
			for _, id := range w.Mandates {
				if id.Key() == d.Identity().Key() {
					seen = true
				}
			}
			if !seen {
				w.Mandates = append(w.Mandates, d.Identity())
			}
		}
	}
	return allowed
}
func withinRoots(p string, roots []string) bool {
	for _, root := range roots {
		if root == "." || root == p || len(p) > len(root) && p[:len(root)+1] == root+"/" {
			return true
		}
	}
	return false
}
func finishPlan(p Plan) Plan { sortFindings(p.Findings); p.Digest = ""; p.Digest = Digest(p); return p }
