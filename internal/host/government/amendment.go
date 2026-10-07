package government

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
)

// AmendmentAssessment reports whether a proposed model falls within the
// authority of an already frozen order and plan. Eligibility is not acceptance:
// checks, independent review, fresh cabinet votes, and promotion remain separate.
type AmendmentAssessment struct {
	Status               string                    `json:"status" yaml:"status"`
	Digest               string                    `json:"digest" yaml:"digest"`
	PriorConstitution    string                    `json:"priorConstitution" yaml:"priorConstitution"`
	ProposedConstitution string                    `json:"proposedConstitution" yaml:"proposedConstitution"`
	Changed              []core.DefinitionIdentity `json:"changed" yaml:"changed"`
	Subjects             []core.DefinitionIdentity `json:"subjects" yaml:"subjects"`
	Findings             []AmendmentFinding        `json:"findings" yaml:"findings"`
}

// AmendmentFinding identifies a concrete violation and the next higher Area
// selected only from the active prior hierarchy, or explicitly routes to the
// trusted Owner. It never treats an agent's prose as Owner authorization.
type AmendmentFinding struct {
	Code            string                  `json:"code" yaml:"code"`
	Subject         string                  `json:"subject,omitempty" yaml:"subject,omitempty"`
	Detail          string                  `json:"detail" yaml:"detail"`
	EscalateTo      core.DefinitionIdentity `json:"escalateTo,omitempty" yaml:"escalateTo,omitempty"`
	EscalateToOwner bool                    `json:"escalateToOwner,omitempty" yaml:"escalateToOwner,omitempty"`
}

// AssessAmendment compares a candidate against the immutable prior model. The
// only mutable organizational relations in this bounded contract are
// Responsibility and Realization. All domain identities, schemas, artifacts,
// authority, Area/Mandate/Ressort/Capability definitions, and Constitution
// content remain fixed. Existing domain Definitions may change only when the
// exact prior amend-model mandate covers their subject.
func AssessAmendment(prior Model, proposed Source, order Order, plan Plan) AmendmentAssessment {
	candidate := Compile(proposed)
	a := AmendmentAssessment{
		Status:               "eligible-for-fresh-review",
		PriorConstitution:    prior.Digest,
		ProposedConstitution: candidate.Digest,
		Changed:              []core.DefinitionIdentity{},
		Subjects:             []core.DefinitionIdentity{},
		Findings:             []AmendmentFinding{},
	}
	add := func(code, subject, detail string, target core.DefinitionIdentity) {
		owner := code == "amendment.schema" || code == "amendment.constitution-identity" || code == "amendment.constitution" || code == "amendment.organization" || code == "amendment.domain-identities" || code == "amendment.definition-identities" || code == "amendment.protected-subject"
		if owner {
			target = core.DefinitionIdentity{}
		}
		a.Findings = append(a.Findings, AmendmentFinding{Code: code, Subject: subject, Detail: detail, EscalateTo: target, EscalateToOwner: owner})
	}
	root := prior.Root

	if len(prior.Findings) != 0 {
		add("amendment.prior-invalid", prior.Constitution.Key(), "active prior model has structural findings", root)
	}
	if len(candidate.Findings) != 0 {
		add("amendment.candidate-invalid", proposed.Constitution.Key(), "proposed model does not compile cleanly", root)
	}
	if proposed.Constitution.Key() != prior.Constitution.Key() {
		add("amendment.constitution-identity", proposed.Constitution.Key(), "the selected active Constitution identity is fixed for this decision", root)
	}
	if !sameSchemas(prior.Canonical.Schemas, candidate.Canonical.Schemas) {
		add("amendment.schema", prior.Constitution.Key(), "schema changes require a separately authorized owner decision", root)
	}
	if order.Action != "amend-model" || order.ActiveConstitution != prior.Digest || order.APIVersion != OrderVersion || order.Kind != "Order" {
		add("amendment.order", prior.Constitution.Key(), "the frozen order must be a valid amend-model request bound to the exact prior digest", root)
	}
	if plan.ActiveConstitution != prior.Digest || plan.OrderDigest != Digest(order) || plan.Status != "planned-within-declared-boundary" && plan.Status != "planned-scoped" || len(plan.Findings) != 0 {
		add("amendment.plan", prior.Constitution.Key(), "the frozen plan must be clean and bound to this prior Constitution and exact order", root)
	}
	planForDigest := plan
	planForDigest.Findings = append([]Finding(nil), plan.Findings...)
	if finishPlan(planForDigest).Digest != plan.Digest {
		add("amendment.plan-digest", prior.Constitution.Key(), "frozen plan fields do not match its digest", root)
	}
	if plan.Action != "amend-model" || !validAmendmentModelPath(prior, plan) {
		add("amendment.model-path", plan.ModelPath, "amend-model requires the exact safe prior canonical model path, writer assignment, and observed bytes", root)
	}
	priorObservation := inventory.Options{Roots: plan.Survey.Observation.Roots, Exclusions: plan.Survey.Observation.Exclusions, AdmitIgnored: plan.Survey.Observation.AdmitIgnored}
	if !reflect.DeepEqual(normalizeInventoryOptions(proposed.Observation), normalizeInventoryOptions(priorObservation)) {
		add("amendment.observation", prior.Constitution.Key(), "repository acquisition roots and boundaries are frozen by the prior plan", root)
	}
	if !sameIdentities(plan.Cabinet, prior.Cabinet) {
		add("amendment.cabinet", prior.Constitution.Key(), "plan electorate differs from the frozen prior cabinet", root)
	}
	if order.UnknownScope {
		add("amendment.unknown-scope", prior.Constitution.Key(), "unknown scope cannot authorize a model amendment", root)
	}

	priorDefs := definitionsByIdentity(prior.Canonical.Definitions)
	candidateDefs := definitionsByIdentity(candidate.Canonical.Definitions)
	priorDomain := domainIdentities(priorDefs)
	candidateDomain := domainIdentities(candidateDefs)
	if !sameIdentityKeys(priorDomain, candidateDomain) {
		add("amendment.domain-identities", prior.Constitution.Key(), "G4 permits changes to existing domain Definitions only; additions and removals require owner escalation", root)
	}
	if !sameIdentityKeys(identityKeys(priorDefs), identityKeys(candidateDefs)) {
		add("amendment.definition-identities", prior.Constitution.Key(), "Definition identity additions and removals are outside this amendment boundary", root)
	}

	protected := protectedSubjects(priorDefs[prior.Constitution.Key()])
	changedSubjects := map[string]core.DefinitionIdentity{}
	for key, old := range priorDefs {
		candidateDefinition, ok := candidateDefs[key]
		if !ok {
			continue
		}
		if sameDefinition(old, candidateDefinition) {
			continue
		}
		id := old.Identity()
		a.Changed = append(a.Changed, id)
		if old.APIVersion != APIVersion {
			changedSubjects[key] = id
			continue
		}
		switch old.Kind {
		case "Responsibility", "Realization":
			for _, subject := range []core.DefinitionIdentity{identity(old.Spec["subject"]), identity(candidateDefinition.Spec["subject"])} {
				if subject.Name != "" {
					changedSubjects[subject.Key()] = subject
				}
			}
		case "Constitution":
			add("amendment.constitution", key, "Constitution identity, root, protected goals, cabinet, and approval rule are immutable during this decision", root)
		case "Area", "Mandate", "Ressort", "Capability", "Artifact":
			add("amendment.organization", key, "organizational authority, capabilities, and artifact declarations are frozen for this decision", root)
		default:
			add("amendment.organization", key, "only Responsibility and Realization relations may change under this G4 contract", root)
		}
	}
	a.Changed = uniqueSortedIdentities(a.Changed)
	for key, subject := range changedSubjects {
		a.Subjects = append(a.Subjects, subject)
		if _, exists := priorDefs[key]; !exists || priorDefs[key].APIVersion == APIVersion {
			add("amendment.subject-identity", key, "changed relation must name an existing exact domain identity", root)
			continue
		}
		if !amendmentContainsIdentity(plan.Affected, subject) {
			add("amendment.outside-plan", key, "changed subject is outside the frozen prior affected closure", escalationTarget(prior, subject))
		}
		if protected[key] {
			add("amendment.protected-subject", key, "the active Constitution marks this root goal as protected", root)
		}
		if !priorPlanAuthorizesAmendment(prior, plan, subject) {
			add("amendment.authority", key, "no prior amend-model mandate in the frozen plan covers this subject", escalationTarget(prior, subject))
		}
	}
	a.Subjects = uniqueSortedIdentities(a.Subjects)

	if len(candidate.Findings) == 0 && len(prior.Findings) == 0 {
		for _, affected := range changedSubjects {
			if !candidateOwnershipCovered(prior, candidateDefs, plan, affected) {
				add("amendment.ownership-coverage", affected.Key(), "proposed accountable Area must exist in prior authority and lie within frozen work or review coverage", escalationTarget(prior, affected))
			}
		}
		for _, subject := range candidateRealizationSubjects(candidateDefs) {
			if !amendmentContainsIdentity(plan.Affected, subject) && changedSubjects[subject.Key()].Name != "" {
				add("amendment.realization-scope", subject.Key(), "proposed realization expands beyond the frozen prior affected closure", escalationTarget(prior, subject))
			}
		}
		for _, subject := range changedSubjects {
			for _, outside := range amendmentClosure(priorDefs, prior.Canonical.Edges, subject) {
				if !amendmentContainsIdentity(plan.Affected, outside) {
					add("amendment.prior-closure", outside.Key(), "prior dependency closure is not contained in the frozen affected plan", escalationTarget(prior, outside))
				}
			}
			for _, outside := range amendmentClosure(candidateDefs, candidate.Canonical.Edges, subject) {
				if !amendmentContainsIdentity(plan.Affected, outside) {
					add("amendment.proposed-closure", outside.Key(), "proposed graph or realization closure widens the frozen affected plan", escalationTarget(prior, outside))
				}
			}
		}
		for _, subject := range changedSubjects {
			if !validateChangedSubjectRealizations(prior, priorDefs, candidateDefs, plan, subject) {
				add("amendment.realization-path", subject.Key(), "every changed realization must target a prior-selected managed path, writer, and observed byte input; unchanged foreign inputs may remain contextual", escalationTarget(prior, subject))
			}
			if countManagedSelectedRealizations(candidateDefs, plan, subject) == 0 {
				add("amendment.realization-required", subject.Key(), "each changed subject must retain a managed non-foreign realization within frozen selected and observed paths", escalationTarget(prior, subject))
			}
		}
	}

	sort.Slice(a.Findings, func(i, j int) bool {
		if a.Findings[i].Code != a.Findings[j].Code {
			return a.Findings[i].Code < a.Findings[j].Code
		}
		if a.Findings[i].Subject != a.Findings[j].Subject {
			return a.Findings[i].Subject < a.Findings[j].Subject
		}
		if a.Findings[i].Detail != a.Findings[j].Detail {
			return a.Findings[i].Detail < a.Findings[j].Detail
		}
		return a.Findings[i].EscalateTo.Key() < a.Findings[j].EscalateTo.Key()
	})
	if len(a.Findings) > 0 {
		a.Status = "blocked-escalation-required"
	}
	a.Digest = Digest(struct {
		Prior, Proposed, Order, Plan string
		Status                       string
		Changed                      []core.DefinitionIdentity
		Subjects                     []core.DefinitionIdentity
		Findings                     []AmendmentFinding
	}{prior.Digest, candidate.Digest, Digest(order), plan.Digest, a.Status, a.Changed, a.Subjects, a.Findings})
	return a
}

func definitionsByIdentity(definitions []core.Definition) map[string]core.Definition {
	out := make(map[string]core.Definition, len(definitions))
	for _, d := range definitions {
		out[d.Identity().Key()] = d
	}
	return out
}

func domainIdentities(definitions map[string]core.Definition) []string {
	var out []string
	for key, d := range definitions {
		if d.APIVersion != APIVersion {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func identityKeys(definitions map[string]core.Definition) []string {
	out := make([]string, 0, len(definitions))
	for key := range definitions {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func sameIdentityKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameDefinition(a, b core.Definition) bool {
	a.Source = core.Source{}
	b.Source = core.Source{}
	return reflect.DeepEqual(a, b)
}

func sameSchemas(a, b []core.Schema) bool {
	a = append([]core.Schema(nil), a...)
	b = append([]core.Schema(nil), b...)
	for i := range a {
		a[i].Source = core.Source{}
	}
	for i := range b {
		b[i].Source = core.Source{}
	}
	return reflect.DeepEqual(a, b)
}

func normalizeInventoryOptions(options inventory.Options) inventory.Options {
	out := inventory.Options{
		Roots:        append([]string(nil), options.Roots...),
		Exclusions:   append([]inventory.Boundary(nil), options.Exclusions...),
		AdmitIgnored: append([]inventory.Boundary(nil), options.AdmitIgnored...),
	}
	sort.Strings(out.Roots)
	less := func(a, b inventory.Boundary) bool {
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Reason < b.Reason
	}
	sort.Slice(out.Exclusions, func(i, j int) bool { return less(out.Exclusions[i], out.Exclusions[j]) })
	sort.Slice(out.AdmitIgnored, func(i, j int) bool { return less(out.AdmitIgnored[i], out.AdmitIgnored[j]) })
	return out
}

func protectedSubjects(constitution core.Definition) map[string]bool {
	out := map[string]bool{}
	for _, id := range identities(constitution.Spec["protectedSubjects"]) {
		out[id.Key()] = true
	}
	return out
}

func sameIdentities(a, b []core.DefinitionIdentity) bool {
	return sameIdentityKeys(identityKeySlice(a), identityKeySlice(b))
}

func identityKeySlice(ids []core.DefinitionIdentity) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.Key())
	}
	sort.Strings(out)
	return out
}

func uniqueSortedIdentities(ids []core.DefinitionIdentity) []core.DefinitionIdentity {
	byKey := map[string]core.DefinitionIdentity{}
	for _, id := range ids {
		byKey[id.Key()] = id
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]core.DefinitionIdentity, 0, len(keys))
	for _, key := range keys {
		out = append(out, byKey[key])
	}
	return out
}

func amendmentContainsIdentity(ids []core.DefinitionIdentity, target core.DefinitionIdentity) bool {
	for _, id := range ids {
		if id.Key() == target.Key() {
			return true
		}
	}
	return false
}

func priorPlanAuthorizesAmendment(prior Model, plan Plan, subject core.DefinitionIdentity) bool {
	owner := priorOwner(prior, subject)
	if owner.Name == "" {
		return false
	}
	mandateIDs := map[string]bool{}
	for _, work := range plan.Work {
		if work.Area.Key() == owner.Key() {
			for _, id := range work.Mandates {
				mandateIDs[id.Key()] = true
			}
		}
	}
	for _, d := range prior.Canonical.Definitions {
		if d.APIVersion != APIVersion || d.Kind != "Mandate" || identity(d.Spec["area"]).Key() != owner.Key() || !contains(stringsList(d.Spec["actions"]), "amend-model") || !subsetIDs([]core.DefinitionIdentity{subject}, identities(d.Spec["scope"])) {
			continue
		}
		if mandateIDs[d.Identity().Key()] {
			return true
		}
	}
	return false
}

func priorMandateAllows(prior Model, area, subject core.DefinitionIdentity, action string) bool {
	return mandateAllows(prior.Canonical.Definitions, area, subject, action)
}

func mandateAllows(definitions []core.Definition, area, subject core.DefinitionIdentity, action string) bool {
	for _, d := range definitions {
		if d.APIVersion == APIVersion && d.Kind == "Mandate" && identity(d.Spec["area"]).Key() == area.Key() && contains(stringsList(d.Spec["actions"]), action) && subsetIDs([]core.DefinitionIdentity{subject}, identities(d.Spec["scope"])) {
			return true
		}
	}
	return false
}

func priorOwner(prior Model, subject core.DefinitionIdentity) core.DefinitionIdentity {
	for _, d := range prior.Canonical.Definitions {
		if d.APIVersion == APIVersion && d.Kind == "Responsibility" && identity(d.Spec["subject"]).Key() == subject.Key() {
			return identity(d.Spec["area"])
		}
	}
	return core.DefinitionIdentity{}
}

func escalationTarget(prior Model, subject core.DefinitionIdentity) core.DefinitionIdentity {
	owner := priorOwner(prior, subject)
	if owner.Name == "" {
		return prior.Root
	}
	definitions := definitionsByIdentity(prior.Canonical.Definitions)
	area := definitions[owner.Key()]
	parent := optionalIdentity(area.Spec["parent"])
	for parent != "" {
		id := definitions[parent].Identity()
		if priorMandateAllows(prior, id, subject, "amend-model") {
			return id
		}
		parent = optionalIdentity(definitions[parent].Spec["parent"])
	}
	return prior.Root
}

func candidateOwnershipCovered(prior Model, candidate map[string]core.Definition, plan Plan, subject core.DefinitionIdentity) bool {
	responsibility, ok := responsibilityFor(candidate, subject)
	if !ok {
		return false
	}
	area := identity(responsibility.Spec["area"])
	if _, ok := definitionsByIdentity(prior.Canonical.Definitions)[area.Key()]; !ok || !areaCovered(plan, area) {
		return false
	}
	return priorMandateAllows(prior, area, subject, "amend-model")
}

func responsibilityFor(definitions map[string]core.Definition, subject core.DefinitionIdentity) (core.Definition, bool) {
	for _, d := range definitions {
		if d.APIVersion == APIVersion && d.Kind == "Responsibility" && identity(d.Spec["subject"]).Key() == subject.Key() {
			return d, true
		}
	}
	return core.Definition{}, false
}

func areaCovered(plan Plan, area core.DefinitionIdentity) bool {
	for _, work := range plan.Work {
		if work.Area.Key() == area.Key() {
			return true
		}
	}
	return amendmentContainsIdentity(plan.IntegrationReviews, area)
}

func validAmendmentModelPath(prior Model, plan Plan) bool {
	if !SafePath(plan.ModelPath) {
		return false
	}
	var artifact core.Definition
	for _, d := range prior.Canonical.Definitions {
		if d.APIVersion == APIVersion && d.Kind == "Artifact" && fmt.Sprint(d.Spec["path"]) == plan.ModelPath {
			if artifact.Identity().Name != "" {
				return false
			}
			artifact = d
		}
	}
	if artifact.Identity().Name == "" || artifact.Spec["class"] != "canonical" {
		return false
	}
	writer := identity(artifact.Spec["writer"])
	if writer.Name == "" || !pathInWork(plan, writer, plan.ModelPath) || !observedFile(plan, plan.ModelPath) {
		return false
	}
	linked := false
	for _, d := range prior.Canonical.Definitions {
		if d.APIVersion == APIVersion && d.Kind == "Realization" && identity(d.Spec["artifact"]).Key() == artifact.Identity().Key() && amendmentContainsIdentity(plan.Affected, identity(d.Spec["subject"])) {
			linked = true
			if !selectedWorkMandateAllows(prior, plan, writer, identity(d.Spec["subject"]), "amend-model", plan.ModelPath) {
				return false
			}
		}
	}
	return linked
}

func pathInWork(plan Plan, writer core.DefinitionIdentity, target string) bool {
	for _, work := range plan.Work {
		if work.Area.Key() != writer.Key() {
			continue
		}
		for _, path := range work.Paths {
			if path == target {
				return true
			}
		}
	}
	return false
}

func selectedWorkMandateAllows(prior Model, plan Plan, area, subject core.DefinitionIdentity, action, path string) bool {
	for _, work := range plan.Work {
		if work.Area.Key() != area.Key() || !contains(work.Paths, path) {
			continue
		}
		for _, mandateID := range work.Mandates {
			mandate, exists := prior.byID[mandateID.Key()]
			if exists && mandate.APIVersion == APIVersion && mandate.Kind == "Mandate" && identity(mandate.Spec["area"]).Key() == area.Key() && contains(stringsList(mandate.Spec["actions"]), action) && subsetIDs([]core.DefinitionIdentity{subject}, identities(mandate.Spec["scope"])) {
				return true
			}
		}
	}
	return false
}

func observedFile(plan Plan, target string) bool {
	for _, entry := range plan.Survey.Observation.Entries {
		if entry.Path == target {
			return entry.Type == "file" && entry.Status == "observed" && entry.Digest != ""
		}
	}
	return false
}

func validRealizationPath(prior Model, priorDefs, candidateDefs map[string]core.Definition, plan Plan, realization core.Definition, subject core.DefinitionIdentity) bool {
	artifactID := identity(realization.Spec["artifact"])
	priorArtifact, priorOK := priorDefs[artifactID.Key()]
	candidateArtifact, candidateOK := candidateDefs[artifactID.Key()]
	if !priorOK || !candidateOK || priorArtifact.APIVersion != APIVersion || priorArtifact.Kind != "Artifact" || !sameDefinition(priorArtifact, candidateArtifact) {
		return false
	}
	if priorArtifact.Spec["class"] == "foreign" {
		return false
	}
	path := fmt.Sprint(priorArtifact.Spec["path"])
	writer := identity(priorArtifact.Spec["writer"])
	if !SafePath(path) || writer.Name == "" || !pathInWork(plan, writer, path) || !observedFile(plan, path) {
		return false
	}
	action := "implement"
	if path == plan.ModelPath {
		action = "amend-model"
	}
	return selectedWorkMandateAllows(prior, plan, writer, subject, action, path)
}

func validateChangedSubjectRealizations(prior Model, priorDefs, candidateDefs map[string]core.Definition, plan Plan, subject core.DefinitionIdentity) bool {
	for key, old := range priorDefs {
		if old.APIVersion != APIVersion || old.Kind != "Realization" || identity(old.Spec["subject"]).Key() != subject.Key() {
			continue
		}
		oldArtifact := priorDefs[identity(old.Spec["artifact"]).Key()]
		if oldArtifact.Spec["class"] == "foreign" {
			if candidate, exists := candidateDefs[key]; exists && sameDefinition(old, candidate) {
				continue
			}
			// A removed or rewired old foreign edge is handled through its
			// proposed target below; it is not a managed verification input.
			continue
		}
		if !validRealizationPath(prior, priorDefs, candidateDefs, plan, old, subject) {
			return false
		}
	}
	for key, candidate := range candidateDefs {
		if candidate.APIVersion != APIVersion || candidate.Kind != "Realization" || identity(candidate.Spec["subject"]).Key() != subject.Key() {
			continue
		}
		old, exists := priorDefs[key]
		unchanged := exists && sameDefinition(old, candidate)
		artifact := candidateDefs[identity(candidate.Spec["artifact"]).Key()]
		if artifact.Spec["class"] == "foreign" {
			if unchanged {
				continue
			}
			return false
		}
		if !validRealizationPath(prior, priorDefs, candidateDefs, plan, candidate, subject) {
			return false
		}
	}
	return true
}

func countManagedSelectedRealizations(candidateDefs map[string]core.Definition, plan Plan, subject core.DefinitionIdentity) int {
	count := 0
	for _, realization := range candidateDefs {
		if realization.APIVersion != APIVersion || realization.Kind != "Realization" || identity(realization.Spec["subject"]).Key() != subject.Key() {
			continue
		}
		artifactID := identity(realization.Spec["artifact"])
		artifact, ok := candidateDefs[artifactID.Key()]
		if !ok || artifact.APIVersion != APIVersion || artifact.Kind != "Artifact" || artifact.Spec["class"] == "foreign" {
			continue
		}
		path := fmt.Sprint(artifact.Spec["path"])
		writer := identity(artifact.Spec["writer"])
		if SafePath(path) && writer.Name != "" && pathInWork(plan, writer, path) && observedFile(plan, path) {
			count++
		}
	}
	return count
}

func candidateRealizationSubjects(definitions map[string]core.Definition) []core.DefinitionIdentity {
	var out []core.DefinitionIdentity
	for _, d := range definitions {
		if d.APIVersion == APIVersion && d.Kind == "Realization" {
			out = append(out, identity(d.Spec["subject"]))
		}
	}
	return uniqueSortedIdentities(out)
}

func amendmentClosure(definitions map[string]core.Definition, edges []core.Edge, start core.DefinitionIdentity) []core.DefinitionIdentity {
	seen := map[string]core.DefinitionIdentity{start.Key(): start}
	queue := []string{start.Key()}
	artifactSubjects := map[string][]core.DefinitionIdentity{}
	for _, d := range definitions {
		if d.APIVersion == APIVersion && d.Kind == "Realization" {
			artifactSubjects[identity(d.Spec["artifact"]).Key()] = append(artifactSubjects[identity(d.Spec["artifact"]).Key()], identity(d.Spec["subject"]))
		}
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		add := func(next string) {
			d, ok := definitions[next]
			if !ok || d.APIVersion == APIVersion || seen[next].Name != "" {
				return
			}
			seen[next] = d.Identity()
			queue = append(queue, next)
		}
		for _, edge := range edges {
			if edge.From == key {
				add(edge.To)
			}
			if edge.To == key {
				add(edge.From)
			}
		}
		for _, d := range definitions {
			if d.APIVersion != APIVersion || d.Kind != "Realization" || identity(d.Spec["subject"]).Key() != key {
				continue
			}
			artifact := identity(d.Spec["artifact"]).Key()
			for _, subject := range artifactSubjects[artifact] {
				add(subject.Key())
			}
		}
	}
	out := make([]core.DefinitionIdentity, 0, len(seen))
	for _, id := range seen {
		out = append(out, id)
	}
	return uniqueSortedIdentities(out)
}
