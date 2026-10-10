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
	seed := map[string]bool{}
	managers := map[string]bool{}
	files := map[string]bool{}
	checks := map[string]bool{}
	for id := range changed {
		for _, report := range []Report{base, candidate} {
			for _, s := range report.Statements {
				if s.ID == id {
					seed[id] = true
					managers[s.Owner] = true
				}
			}
			for _, a := range report.Artifacts {
				if a.ID == id {
					managers[a.Owner] = true
					for _, sid := range a.Realizes {
						seed[sid] = true
					}
					addArtifactPaths(files, a)
					for _, cid := range a.Checks {
						checks[cid] = true
					}
				}
			}
			for _, c := range report.Checks {
				if c.ID == id {
					managers[c.Owner] = true
					for _, sid := range c.Uses {
						seed[sid] = true
					}
					checks[id] = true
				}
			}
			// A changed Decision is a change of its subject, routed with the Decision's owner.
			for _, d := range report.Decisions {
				if d.ID == id {
					managers[d.Owner] = true
					seed[d.Subject] = true
					if s, ok := statementByID[d.Subject]; ok {
						managers[s.Owner] = true
					}
				}
			}
			for _, m := range report.Managers {
				if m.ID == id {
					managers[id] = true
					addAncestors(managers, id, managersForReport(report))
				}
			}
			if managerByID[id].ID != "" {
				for _, statement := range report.Statements {
					if statement.Owner == id {
						seed[statement.ID] = true
					}
				}
				for _, artifact := range report.Artifacts {
					if artifact.Owner == id {
						for _, sid := range artifact.Realizes {
							seed[sid] = true
						}
						addArtifactPaths(files, artifact)
						for _, cid := range artifact.Checks {
							checks[cid] = true
						}
					}
				}
				for _, check := range report.Checks {
					if check.Owner == id {
						checks[check.ID] = true
						for _, sid := range check.Uses {
							seed[sid] = true
						}
					}
				}
				for _, entry := range report.Files {
					if entry.Owner == id {
						files[entry.Path] = true
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
		files[p] = true
		for _, entry := range []FileEntry{a, b} {
			if entry.Owner != "" {
				managers[entry.Owner] = true
			}
			for _, sid := range entry.Statements {
				seed[sid] = true
			}
			for _, aid := range entry.Artifacts {
				if art, ok := artifactByID[aid]; ok {
					managers[art.Owner] = true
					for _, sid := range art.Realizes {
						seed[sid] = true
					}
				}
			}
			for _, cid := range entry.Checks {
				checks[cid] = true
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
	modelUnprojected := base.ModelDigest != candidate.ModelDigest && (len(changed) == 0 || unprojectedChange(base, candidate, changed) || writingChange(base, candidate))
	if modelUnprojected {
		out.Unknown = append(out.Unknown, "model digest changed beyond the projected definition delta; decision or unprojected definition changes may require review")
	}

	// Directly changed statements and reverse dependents need their own realizing artifacts
	// and every check that exercises them.
	allChecks := append(append([]Check(nil), base.Checks...), candidate.Checks...)
	addCoverage := func(statementID string) {
		for _, artifact := range allArtifacts {
			if contains(artifact.Realizes, statementID) {
				managers[artifact.Owner] = true
				addArtifactPaths(files, artifact)
				for _, checkID := range artifact.Checks {
					checks[checkID] = true
				}
			}
		}
		for _, check := range allChecks {
			if contains(check.Uses, statementID) {
				managers[check.Owner] = true
				checks[check.ID] = true
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
		// uses adds context and ownership routing. It does not imply implementation coverage.
		for _, dep := range uses[id] {
			if s, ok := statementByID[dep]; ok {
				managers[s.Owner] = true
			}
		}
		// requires adds the target contract and its declared artifact/check coverage.
		for _, dep := range requires[id] {
			addCoverage(dep)
			if s, ok := statementByID[dep]; ok {
				managers[s.Owner] = true
			}
		}
		// An affected statement routes every direct consumer; each consumer's own realization is affected too.
		for _, consumer := range consumers[id] {
			addCoverage(consumer)
			for _, owner := range statementOwners[consumer] {
				managers[owner] = true
			}
		}
	}
	for id := range closure {
		out.AffectedStatements = append(out.AffectedStatements, id)
	}
	for id := range changed {
		if _, ok := managerByID[id]; ok {
			addAncestors(managers, id, managerByID)
		}
	}
	initialManagers := mapKeys(managers)
	for _, id := range initialManagers {
		if id != "" {
			addAncestors(managers, id, managerByID)
			out.Managers = append(out.Managers, id)
		}
	}
	for id := range managers {
		if id != "" {
			out.Managers = append(out.Managers, id)
		}
	}
	out.Files = mapKeys(files)
	out.Checks = mapKeys(checks)
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
		for _, report := range []Report{base, candidate} {
			for _, manager := range report.Managers {
				managers[manager.ID] = true
				addAncestors(managers, manager.ID, managersForReport(report))
			}
			for _, statement := range report.Statements {
				out.AffectedStatements = append(out.AffectedStatements, statement.ID)
			}
			for _, entry := range report.Files {
				files[entry.Path] = true
			}
			for _, artifact := range report.Artifacts {
				addArtifactPaths(files, artifact)
				for _, checkID := range artifact.Checks {
					checks[checkID] = true
				}
			}
			for _, check := range report.Checks {
				checks[check.ID] = true
			}
		}
		out.Managers = mapKeys(managers)
		out.Files = mapKeys(files)
		out.Checks = mapKeys(checks)
		out.Findings = append(out.Findings, Finding{Code: "impact.unknown-scope", Message: "Some project or inventory scope is unresolved and remains in the impact.", Severity: "incomplete"})
	}
	for _, m := range out.Managers {
		out.Findings = append(out.Findings, Finding{Code: "impact.manager-routing", Subject: m, Message: "Manager is included through changed ownership or a relevant dependency.", Severity: "info"})
	}
	for id := range changed {
		if s, ok := statementByID[id]; ok {
			_ = s
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
	out.Findings = sortedFindings(out.Findings)
	out.ChangedDefinitions = sortedUnique(out.ChangedDefinitions)
	out.AffectedStatements = sortedUnique(out.AffectedStatements)
	out.Managers = sortedUnique(out.Managers)
	out.Files = sortedUnique(out.Files)
	out.Checks = sortedUnique(out.Checks)
	out.Digest = digest(out)
	return out
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

// unprojectedChange reports a change the report collections do not show: an
// unprojected part, such as a purpose, of a definition in both revisions, even
// when its projection changed too, or a Decision added or removed. Reports not
// built by Analyze carry no such digests, so the change cannot be ruled out.
func unprojectedChange(base, candidate Report, changed map[string]bool) bool {
	if base.unprojected == nil || candidate.unprojected == nil {
		return true
	}
	for id, d := range base.unprojected {
		if next, ok := candidate.unprojected[id]; ok && next != d || !ok && !changed[id] {
			return true
		}
	}
	for id := range candidate.unprojected {
		if _, ok := base.unprojected[id]; !ok && !changed[id] {
			return true
		}
	}
	return false
}

// writingChange reports an edit that changes only how a property is written,
// such as list order, a repeated entry or an explicit default. Such an edit
// widens on its own, so it widens next to other changes too. In a set-like
// list, the entries kept in both revisions must keep their order and count.
func writingChange(base, candidate Report) bool {
	if base.written == nil || candidate.written == nil {
		return true
	}
	for id, properties := range base.written {
		for name, before := range properties {
			after, ok := candidate.written[id][name]
			if !ok {
				continue
			}
			if before.raw != after.raw && before.value == after.value || !slices.Equal(keptElements(before.elements, after.elements), keptElements(after.elements, before.elements)) {
				return true
			}
		}
	}
	return false
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
