package projectmodel

import (
	"errors"
	"slices"
	"sort"
)

var ErrManagerNotFound = errors.New("manager not found")

func Context(report Report, managerID string) (ManagerContext, error) {
	var out ManagerContext
	managerByID := map[string]Manager{}
	for _, m := range report.Managers {
		managerByID[m.ID] = m
	}
	manager, ok := managerByID[managerID]
	if !ok {
		return out, ErrManagerNotFound
	}
	out.Manager = manager
	statementByID := map[string]Statement{}
	for _, s := range report.Statements {
		statementByID[s.ID] = s
		if s.Owner == managerID {
			out.Statements = append(out.Statements, s)
		}
	}
	for _, a := range report.Artifacts {
		if a.Owner == managerID {
			out.Artifacts = append(out.Artifacts, a)
		}
	}
	for _, c := range report.Checks {
		if c.Owner == managerID {
			out.Checks = append(out.Checks, c)
		}
	}
	for _, d := range report.Decisions {
		if d.Owner == managerID {
			out.Decisions = append(out.Decisions, d)
		}
	}
	// Contracts are the foreign public statements that own statements use or require,
	// own artifacts realize, own checks exercise, or own decisions decide on.
	needed := map[string]bool{}
	for _, s := range out.Statements {
		for _, id := range s.Uses {
			needed[id] = true
		}
		for _, id := range s.Requires {
			needed[id] = true
		}
	}
	for _, a := range out.Artifacts {
		for _, id := range a.Realizes {
			needed[id] = true
		}
	}
	for _, c := range out.Checks {
		for _, id := range c.Uses {
			needed[id] = true
		}
	}
	for _, d := range out.Decisions {
		needed[d.Subject] = true
	}
	for id := range needed {
		if s, found := statementByID[id]; found && s.Owner != managerID && s.Public {
			s.Uses = visibleRelations(s.Uses, statementByID)
			s.Requires = visibleRelations(s.Requires, statementByID)
			out.Contracts = append(out.Contracts, s)
		}
	}
	for _, child := range report.Managers {
		if child.Parent == managerID {
			child.Instructions = ""
			out.Children = append(out.Children, child)
		}
	}
	for _, f := range report.Findings {
		if f.Subject == managerID || findingTouchesManager(f, managerID, report) {
			out.Findings = append(out.Findings, f)
		}
	}
	sort.Slice(out.Statements, func(i, j int) bool { return out.Statements[i].ID < out.Statements[j].ID })
	sort.Slice(out.Contracts, func(i, j int) bool { return out.Contracts[i].ID < out.Contracts[j].ID })
	sort.Slice(out.Artifacts, func(i, j int) bool { return out.Artifacts[i].ID < out.Artifacts[j].ID })
	sort.Slice(out.Checks, func(i, j int) bool { return out.Checks[i].ID < out.Checks[j].ID })
	sort.Slice(out.Decisions, func(i, j int) bool { return out.Decisions[i].ID < out.Decisions[j].ID })
	sort.Slice(out.Children, func(i, j int) bool { return out.Children[i].ID < out.Children[j].ID })
	out.Findings = sortedFindings(out.Findings)
	return out, nil
}

func visibleRelations(ids []string, statements map[string]Statement) []string {
	visible := make([]string, 0, len(ids))
	for _, id := range ids {
		if target, ok := statements[id]; ok && target.Public {
			visible = append(visible, id)
		}
	}
	return sortedUnique(visible)
}

func findingTouchesManager(f Finding, managerID string, r Report) bool {
	for _, s := range r.Statements {
		if s.ID == f.Subject && s.Owner == managerID {
			return true
		}
	}
	for _, a := range r.Artifacts {
		if a.ID == f.Subject && a.Owner == managerID {
			return true
		}
	}
	for _, c := range r.Checks {
		if c.ID == f.Subject && c.Owner == managerID {
			return true
		}
	}
	for _, d := range r.Decisions {
		if d.ID == f.Subject && d.Owner == managerID {
			return true
		}
	}
	return false
}

func Impact(base, candidate Report) ChangeImpact {
	return route(base, candidate).impact
}

// routing is one impact computation together with the cause of every element
// it routed, so Explain and Impact share a single set of rules.
type routing struct {
	impact ChangeImpact
	causes causes
	// seeds are the statements a change reaches directly (DEC-023 class rule).
	seeds map[string]bool
}

func route(base, candidate Report) routing {
	out := ChangeImpact{APIVersion: APIVersion, BaseDigest: base.Digest, CandidateDigest: candidate.Digest}
	changed := map[string]bool{}
	addChanged := func(id string) {
		if id != "" {
			changed[id] = true
		}
	}
	compareManagers(base.Managers, candidate.Managers, addChanged)
	compareStatements(base.Statements, candidate.Statements, addChanged)
	compareArtifacts(base.Artifacts, candidate.Artifacts, addChanged)
	compareChecks(base.Checks, candidate.Checks, addChanged)
	compareDecisions(base.Decisions, candidate.Decisions, addChanged)
	// A changed purpose is a change of its definition, routed like any other (DEC-021).
	for _, id := range purposeChanges(base, candidate) {
		addChanged(id)
	}
	for id := range changed {
		out.ChangedDefinitions = append(out.ChangedDefinitions, id)
	}
	out.ChangedDefinitions = sortedUnique(out.ChangedDefinitions)

	managerByID := map[string]Manager{}
	for _, m := range base.Managers {
		managerByID[m.ID] = m
	}
	for _, m := range candidate.Managers {
		managerByID[m.ID] = m
	}
	statementByID := map[string]Statement{}
	for _, s := range base.Statements {
		statementByID[s.ID] = s
	}
	for _, s := range candidate.Statements {
		statementByID[s.ID] = s
	}
	artifactByID := map[string]Artifact{}
	allArtifacts := append(append([]Artifact(nil), base.Artifacts...), candidate.Artifacts...)
	for _, a := range base.Artifacts {
		artifactByID[a.ID] = a
	}
	for _, a := range candidate.Artifacts {
		artifactByID[a.ID] = a
	}
	rec := causes{}
	seed := map[string]bool{}
	seedFrom := func(statementID string, from element, relation string) {
		if statementID != "" {
			seed[statementID] = true
			rec.add(element{"statement", statementID}, cause{"seeded", from, relation})
		}
	}
	ownedBy := func(managerID string, from element) {
		rec.add(element{"manager", managerID}, cause{"owner", from, "owned by"})
	}
	addAncestorsOf := func(managerID string, managers map[string]Manager) {
		seen := map[string]bool{}
		for id := managerID; id != "" && !seen[id]; {
			seen[id] = true
			m, ok := managers[id]
			if !ok || m.Parent == "" {
				return
			}
			rec.add(element{"manager", m.Parent}, cause{"ancestor", element{"manager", id}, "parent"})
			id = m.Parent
		}
	}
	artifactPaths := func(a Artifact, from element, reason string) {
		for _, p := range a.Paths {
			rec.add(element{"file", p}, cause{reason, from, "expects"})
		}
	}
	for id := range changed {
		for _, report := range []Report{base, candidate} {
			for _, s := range report.Statements {
				if s.ID == id {
					self := element{"statement", id}
					seed[id] = true
					rec.add(self, cause{reason: "changed"})
					ownedBy(s.Owner, self)
				}
			}
			for _, a := range report.Artifacts {
				if a.ID == id {
					self := element{"artifact", id}
					rec.add(self, cause{reason: "changed"})
					ownedBy(a.Owner, self)
					for _, sid := range a.Realizes {
						seedFrom(sid, self, "realizes")
					}
					artifactPaths(a, self, "realization")
					for _, cid := range a.Checks {
						rec.add(element{"check", cid}, cause{"artifact check", self, "checked by"})
					}
				}
			}
			for _, c := range report.Checks {
				if c.ID == id {
					self := element{"check", id}
					rec.add(self, cause{reason: "changed"})
					ownedBy(c.Owner, self)
					for _, sid := range c.Uses {
						seedFrom(sid, self, "exercises")
					}
				}
			}
			// A changed Decision is a change of its subject, routed with the Decision's owner.
			for _, d := range report.Decisions {
				if d.ID == id {
					self := element{"decision", id}
					rec.add(self, cause{reason: "changed"})
					ownedBy(d.Owner, self)
					seedFrom(d.Subject, self, "decides on")
					if s, ok := statementByID[d.Subject]; ok {
						ownedBy(s.Owner, element{"statement", d.Subject})
					}
				}
			}
			for _, m := range report.Managers {
				if m.ID == id {
					rec.add(element{"manager", id}, cause{reason: "changed"})
					addAncestorsOf(id, managersForReport(report))
				}
			}
			if managerByID[id].ID != "" {
				self := element{"manager", id}
				for _, statement := range report.Statements {
					if statement.Owner == id {
						seedFrom(statement.ID, self, "owns")
					}
				}
				for _, artifact := range report.Artifacts {
					if artifact.Owner == id {
						owned := element{"artifact", artifact.ID}
						rec.add(owned, cause{"owned by changed manager", self, "owns"})
						for _, sid := range artifact.Realizes {
							seedFrom(sid, owned, "realizes")
						}
						artifactPaths(artifact, owned, "realization")
						for _, cid := range artifact.Checks {
							rec.add(element{"check", cid}, cause{"artifact check", owned, "checked by"})
						}
					}
				}
				for _, check := range report.Checks {
					if check.Owner == id {
						owned := element{"check", check.ID}
						rec.add(owned, cause{"owned by changed manager", self, "owns"})
						for _, sid := range check.Uses {
							seedFrom(sid, owned, "exercises")
						}
					}
				}
				for _, entry := range report.Files {
					if entry.Owner == id {
						rec.add(element{"file", entry.Path}, cause{"owned by changed manager", self, "owns"})
					}
				}
			}
		}
	}
	baseFiles, candidateFiles := entryMap(base.Files), entryMap(candidate.Files)
	allPaths := map[string]bool{}
	for p := range baseFiles {
		allPaths[p] = true
	}
	for p := range candidateFiles {
		allPaths[p] = true
	}
	for p := range allPaths {
		a, aok := baseFiles[p]
		b, bok := candidateFiles[p]
		if aok && bok && equal(a, b) {
			continue
		}
		self := element{"file", p}
		rec.add(self, cause{reason: "file changed"})
		for _, entry := range []FileEntry{a, b} {
			if entry.Owner != "" {
				ownedBy(entry.Owner, self)
			}
			for _, sid := range entry.Statements {
				seedFrom(sid, self, "maps to")
			}
			for _, aid := range entry.Artifacts {
				if art, ok := artifactByID[aid]; ok {
					mapped := element{"artifact", aid}
					rec.add(mapped, cause{"mapped by changed file", self, "maps to"})
					ownedBy(art.Owner, mapped)
					for _, sid := range art.Realizes {
						seedFrom(sid, mapped, "realizes")
					}
				}
			}
			for _, cid := range entry.Checks {
				rec.add(element{"check", cid}, cause{"mapped by changed file", self, "maps to"})
			}
		}
	}

	// The graph is evaluated across both revisions so removed edges and obligations remain visible.
	uses := map[string][]string{}
	requires := map[string][]string{}
	for _, r := range []Report{base, candidate} {
		for _, s := range r.Statements {
			uses[s.ID] = appendUnique(uses[s.ID], s.Uses...)
			requires[s.ID] = appendUnique(requires[s.ID], s.Requires...)
		}
	}
	// An edit that changes only how a definition is written routes through its model
	// file alone: the file's owner and the definitions in that file (DEC-021).
	rewritten := writingChanges(base, candidate)
	rewrittenStatements := map[string]bool{}
	for _, id := range rewritten {
		self := definitionElement(id, base, candidate)
		rec.add(self, cause{reason: "rewritten"})
		inFile := []element{self}
		if path := base.sources[id]; path != "" {
			file := element{"file", path}
			rec.add(file, cause{"model file", self, "written in"})
			for _, r := range []Report{base, candidate} {
				ownedBy(fileOwner(path, r.Managers), file)
				for other, source := range r.sources {
					if source == path && other != id {
						declared := definitionElement(other, base, candidate)
						rec.add(declared, cause{"in model file", file, "declares"})
						inFile = append(inFile, declared)
					}
				}
			}
		}
		for _, e := range inFile {
			if s, ok := statementByID[e.id]; ok {
				rewrittenStatements[e.id] = true
				ownedBy(s.Owner, e)
			}
			for _, r := range []Report{base, candidate} {
				for _, a := range r.Artifacts {
					if a.ID == e.id {
						ownedBy(a.Owner, e)
					}
				}
				for _, c := range r.Checks {
					if c.ID == e.id {
						ownedBy(c.Owner, e)
					}
				}
				for _, d := range r.Decisions {
					if d.ID == e.id {
						ownedBy(d.Owner, e)
					}
				}
			}
		}
	}
	// A model change that nothing above names, or reports not built by Analyze, still widen.
	if base.ModelDigest != candidate.ModelDigest && (len(changed) == 0 && len(rewritten) == 0 || !traced(base) || !traced(candidate)) {
		out.Unknown = append(out.Unknown, "model digest changed without a definition change the report can name; the declared project needs review")
	}

	// Directly changed statements and reverse dependents need their own realizing artifacts
	// and every check that exercises them.
	allChecks := append(append([]Check(nil), base.Checks...), candidate.Checks...)
	addCoverage := func(statementID string) {
		from := element{"statement", statementID}
		for _, artifact := range allArtifacts {
			if contains(artifact.Realizes, statementID) {
				realizing := element{"artifact", artifact.ID}
				rec.add(realizing, cause{"realization", from, "realized by"})
				ownedBy(artifact.Owner, realizing)
				artifactPaths(artifact, realizing, "realization")
				for _, checkID := range artifact.Checks {
					rec.add(element{"check", checkID}, cause{"artifact check", realizing, "checked by"})
				}
			}
		}
		for _, check := range allChecks {
			if contains(check.Uses, statementID) {
				exercising := element{"check", check.ID}
				rec.add(exercising, cause{"exercises", from, "exercised by"})
				ownedBy(check.Owner, exercising)
			}
		}
	}
	consumers := map[string][]string{}
	statementOwners := map[string][]string{}
	for _, r := range []Report{base, candidate} {
		for _, s := range r.Statements {
			statementOwners[s.ID] = appendUnique(statementOwners[s.ID], s.Owner)
			for _, dep := range append(append([]string(nil), s.Uses...), s.Requires...) {
				consumers[dep] = appendUnique(consumers[dep], s.ID)
			}
		}
	}
	// The closure is every statement reachable from a seed over uses and requires in either
	// direction. Coverage below depends only on that set, never on the order of the walk, so
	// adding a change can only add to the impact.
	closure := map[string]bool{}
	pending := make([]string, 0, len(seed))
	for id := range seed {
		pending = append(pending, id)
		addCoverage(id)
	}
	for len(pending) > 0 {
		id := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if closure[id] {
			continue
		}
		closure[id] = true
		pending = append(append(append(pending, uses[id]...), requires[id]...), consumers[id]...)
	}
	for id := range closure {
		from := element{"statement", id}
		// uses adds context and ownership routing. It does not imply implementation coverage.
		for _, dep := range uses[id] {
			used := element{"statement", dep}
			rec.add(used, cause{"used", from, "uses"})
			if s, ok := statementByID[dep]; ok {
				ownedBy(s.Owner, used)
			}
		}
		// requires adds the target contract and its declared artifact/check coverage.
		for _, dep := range requires[id] {
			required := element{"statement", dep}
			rec.add(required, cause{"required", from, "requires"})
			addCoverage(dep)
			if s, ok := statementByID[dep]; ok {
				ownedBy(s.Owner, required)
			}
		}
		// An affected statement routes every direct consumer; each consumer's own realization is affected too.
		for _, consumer := range consumers[id] {
			consuming := element{"statement", consumer}
			relation := "required by"
			if contains(uses[consumer], id) {
				relation = "used by"
			}
			rec.add(consuming, cause{"consumer", from, relation})
			addCoverage(consumer)
			for _, owner := range statementOwners[consumer] {
				ownedBy(owner, consuming)
			}
		}
	}
	for id := range closure {
		out.AffectedStatements = append(out.AffectedStatements, id)
	}
	for id := range rewrittenStatements {
		out.AffectedStatements = append(out.AffectedStatements, id)
	}
	for _, id := range rec.ids("manager") {
		addAncestorsOf(id, managerByID)
	}
	out.Unknown = append(out.Unknown, base.Unknown...)
	out.Unknown = append(out.Unknown, candidate.Unknown...)
	if base.Status != "succeeded" {
		out.Unknown = append(out.Unknown, "base report status is "+base.Status)
	}
	if candidate.Status != "succeeded" {
		out.Unknown = append(out.Unknown, "candidate report status is "+candidate.Status)
	}
	out.Unknown = sortedUnique(out.Unknown)
	if len(out.Unknown) > 0 {
		// Unknown scope is never treated as a no-op: route every declared Manager and inventory item.
		widened := cause{reason: "widened"}
		for _, report := range []Report{base, candidate} {
			for _, manager := range report.Managers {
				rec.add(element{"manager", manager.ID}, widened)
				addAncestorsOf(manager.ID, managersForReport(report))
			}
			for _, statement := range report.Statements {
				rec.add(element{"statement", statement.ID}, widened)
				out.AffectedStatements = append(out.AffectedStatements, statement.ID)
			}
			for _, entry := range report.Files {
				rec.add(element{"file", entry.Path}, widened)
			}
			for _, artifact := range report.Artifacts {
				artifactPaths(artifact, element{"artifact", artifact.ID}, "widened")
				rec.add(element{"artifact", artifact.ID}, widened)
				for _, checkID := range artifact.Checks {
					rec.add(element{"check", checkID}, widened)
				}
			}
			for _, check := range report.Checks {
				rec.add(element{"check", check.ID}, widened)
			}
		}
		out.Findings = append(out.Findings, Finding{Code: "impact.unknown-scope", Message: "Some project or inventory scope is unresolved and remains in the impact.", Severity: "incomplete"})
	}
	out.Managers = rec.ids("manager")
	out.Files = rec.ids("file")
	out.Checks = rec.ids("check")
	for _, m := range out.Managers {
		out.Findings = append(out.Findings, Finding{Code: "impact.manager-routing", Subject: m, Message: "Manager is included through changed ownership or a relevant dependency.", Severity: "info"})
	}
	for id := range changed {
		if _, ok := statementByID[id]; ok {
			out.Findings = append(out.Findings, Finding{Code: "impact.statement-change", Subject: id, Message: "Statement definition changed.", Severity: "info"})
		}
	}
	for _, r := range []Report{base, candidate} {
		for _, d := range r.Decisions {
			if changed[d.ID] {
				out.Findings = append(out.Findings, Finding{Code: "impact.decision-change", Subject: d.ID, Message: "Decision changed; it is routed through its subject.", Severity: "info"})
			}
		}
	}
	out.Findings = uniqueFindings(sortedFindings(out.Findings))
	out.ChangedDefinitions = sortedUnique(out.ChangedDefinitions)
	out.AffectedStatements = sortedUnique(out.AffectedStatements)
	out.Managers = sortedUnique(out.Managers)
	out.Files = sortedUnique(out.Files)
	out.Checks = sortedUnique(out.Checks)
	out.Digest = digest(out)
	return routing{impact: out, causes: rec, seeds: seed}
}

// uniqueFindings drops repeats from sorted findings, such as one finding
// raised for both revisions of the same definition.
func uniqueFindings(sorted []Finding) []Finding {
	out := sorted[:0]
	for i, f := range sorted {
		if i == 0 || f != sorted[i-1] {
			out = append(out, f)
		}
	}
	return out
}

// definitionElement names a definition by its kind as either report knows it.
func definitionElement(id string, reports ...Report) element {
	for _, r := range reports {
		for _, s := range r.Statements {
			if s.ID == id {
				return element{"statement", id}
			}
		}
		for _, m := range r.Managers {
			if m.ID == id {
				return element{"manager", id}
			}
		}
		for _, a := range r.Artifacts {
			if a.ID == id {
				return element{"artifact", id}
			}
		}
		for _, c := range r.Checks {
			if c.ID == id {
				return element{"check", id}
			}
		}
		for _, d := range r.Decisions {
			if d.ID == id {
				return element{"decision", id}
			}
		}
	}
	return element{"definition", id}
}

func compareManagers(a, b []Manager, add func(string)) {
	am := map[string]Manager{}
	bm := map[string]Manager{}
	for _, v := range a {
		am[v.ID] = v
	}
	for _, v := range b {
		bm[v.ID] = v
	}
	for id, v := range am {
		w, ok := bm[id]
		if !ok || !equal(v, w) {
			add(id)
		}
	}
	for id := range bm {
		if _, ok := am[id]; !ok {
			add(id)
		}
	}
}
func compareStatements(a, b []Statement, add func(string)) {
	am := map[string]Statement{}
	bm := map[string]Statement{}
	for _, v := range a {
		am[v.ID] = v
	}
	for _, v := range b {
		bm[v.ID] = v
	}
	for id, v := range am {
		w, ok := bm[id]
		if !ok || !equal(v, w) {
			add(id)
		}
	}
	for id := range bm {
		if _, ok := am[id]; !ok {
			add(id)
		}
	}
}
func compareArtifacts(a, b []Artifact, add func(string)) {
	am := map[string]Artifact{}
	bm := map[string]Artifact{}
	for _, v := range a {
		am[v.ID] = v
	}
	for _, v := range b {
		bm[v.ID] = v
	}
	for id, v := range am {
		w, ok := bm[id]
		if !ok || !equal(v, w) {
			add(id)
		}
	}
	for id := range bm {
		if _, ok := am[id]; !ok {
			add(id)
		}
	}
}
func compareDecisions(a, b []Decision, add func(string)) {
	am := map[string]Decision{}
	bm := map[string]Decision{}
	for _, v := range a {
		am[v.ID] = v
	}
	for _, v := range b {
		bm[v.ID] = v
	}
	for id, v := range am {
		w, ok := bm[id]
		if !ok || !equal(v, w) {
			add(id)
		}
	}
	for id := range bm {
		if _, ok := am[id]; !ok {
			add(id)
		}
	}
}
func compareChecks(a, b []Check, add func(string)) {
	am := map[string]Check{}
	bm := map[string]Check{}
	for _, v := range a {
		am[v.ID] = v
	}
	for _, v := range b {
		bm[v.ID] = v
	}
	for id, v := range am {
		w, ok := bm[id]
		if !ok || !equal(v, w) {
			add(id)
		}
	}
	for id := range bm {
		if _, ok := am[id]; !ok {
			add(id)
		}
	}
}

// purposeChanges returns the definitions in both revisions whose purpose, the
// part the report collections do not carry, changed.
func purposeChanges(base, candidate Report) []string {
	var ids []string
	for id, d := range base.unprojected {
		if next, ok := candidate.unprojected[id]; ok && next != d {
			ids = append(ids, id)
		}
	}
	return ids
}

// writingChanges returns the definitions in both revisions with an edit that
// changes only how a property is written, such as list order, a repeated entry
// or an explicit default. In a set-like list, the entries kept in both
// revisions must also keep their order and count.
func writingChanges(base, candidate Report) []string {
	var ids []string
	for id, properties := range base.written {
		for name, before := range properties {
			after, ok := candidate.written[id][name]
			if !ok {
				continue
			}
			if before.raw != after.raw && before.value == after.value || !slices.Equal(keptElements(before.elements, after.elements), keptElements(after.elements, before.elements)) {
				ids = append(ids, id)
				break
			}
		}
	}
	return ids
}

// traced reports whether Analyze built the report, so its purpose and writing
// digests can name every model change.
func traced(r Report) bool {
	return r.unprojected != nil && r.written != nil && r.sources != nil
}

// keptElements returns the elements also present in other, in written order.
func keptElements(elements, other []string) []string {
	var kept []string
	for _, e := range elements {
		if slices.Contains(other, e) {
			kept = append(kept, e)
		}
	}
	return kept
}
func equal(a, b any) bool { return digest(a) == digest(b) }
func entryMap(values []FileEntry) map[string]FileEntry {
	out := map[string]FileEntry{}
	for _, v := range values {
		out[v.Path] = v
	}
	return out
}
func addArtifactPaths(files map[string]bool, a Artifact) {
	for _, p := range a.Paths {
		files[p] = true
	}
}
func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
func mapKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for k := range values {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
func addAncestors(set map[string]bool, id string, managers map[string]Manager) {
	seen := map[string]bool{}
	for id != "" && !seen[id] {
		seen[id] = true
		m, ok := managers[id]
		if !ok || m.Parent == "" {
			return
		}
		set[m.Parent] = true
		id = m.Parent
	}
}

func managersForReport(report Report) map[string]Manager {
	out := make(map[string]Manager, len(report.Managers))
	for _, manager := range report.Managers {
		out[manager.ID] = manager
	}
	return out
}
