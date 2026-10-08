package projectmodel

import (
	"errors"
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
	needed := map[string]bool{}
	for _, s := range out.Statements {
		for _, id := range s.Uses {
			needed[id] = true
		}
		for _, id := range s.Requires {
			needed[id] = true
		}
	}
	for id := range needed {
		if s, found := statementByID[id]; found && s.Owner != managerID && s.Public {
			out.Contracts = append(out.Contracts, s)
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
	for _, child := range report.Managers {
		if child.Parent == managerID {
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
	sort.Slice(out.Children, func(i, j int) bool { return out.Children[i].ID < out.Children[j].ID })
	out.Findings = sortedFindings(out.Findings)
	return out, nil
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
	for _, a := range base.Artifacts {
		artifactByID[a.ID] = a
	}
	for _, a := range candidate.Artifacts {
		artifactByID[a.ID] = a
	}
	checkByID := map[string]Check{}
	for _, c := range base.Checks {
		checkByID[c.ID] = c
	}
	for _, c := range candidate.Checks {
		checkByID[c.ID] = c
	}
	seed := map[string]bool{}
	managers := map[string]bool{}
	files := map[string]bool{}
	checks := map[string]bool{}
	for id := range changed {
		if s, ok := statementByID[id]; ok {
			seed[id] = true
			managers[s.Owner] = true
		}
		if a, ok := artifactByID[id]; ok {
			managers[a.Owner] = true
			for _, sid := range a.Realizes {
				seed[sid] = true
			}
			addArtifactPaths(files, a)
			for _, cid := range a.Checks {
				checks[cid] = true
			}
		}
		if c, ok := checkByID[id]; ok {
			managers[c.Owner] = true
			for _, sid := range c.Uses {
				seed[sid] = true
			}
			checks[id] = true
		}
		if _, ok := managerByID[id]; ok {
			managers[id] = true
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
	closure := map[string]bool{}
	queue := make([]string, 0, len(seed))
	for id := range seed {
		closure[id] = true
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		// uses adds context and ownership routing. It does not imply implementation coverage.
		for _, dep := range uses[id] {
			if !closure[dep] {
				closure[dep] = true
				queue = append(queue, dep)
			}
			if s, ok := statementByID[dep]; ok {
				managers[s.Owner] = true
			}
		}
		// requires adds the target contract and its declared artifact/check coverage.
		for _, dep := range requires[id] {
			if !closure[dep] {
				closure[dep] = true
				queue = append(queue, dep)
			}
			if s, ok := statementByID[dep]; ok {
				managers[s.Owner] = true
			}
			for _, a := range artifactByID {
				if contains(a.Realizes, dep) {
					managers[a.Owner] = true
					addArtifactPaths(files, a)
					for _, cid := range a.Checks {
						checks[cid] = true
					}
				}
			}
		}
	}
	for id := range closure {
		out.AffectedStatements = append(out.AffectedStatements, id)
	}
	// A changed public contract also routes its direct consumers, including removed consumers from base.
	for id := range closure {
		for _, r := range []Report{base, candidate} {
			for _, s := range r.Statements {
				if contains(s.Uses, id) {
					managers[s.Owner] = true
				}
				if contains(s.Requires, id) {
					managers[s.Owner] = true
					for _, a := range r.Artifacts {
						if contains(a.Realizes, id) {
							addArtifactPaths(files, a)
							managers[a.Owner] = true
							for _, cid := range a.Checks {
								checks[cid] = true
							}
						}
					}
				}
			}
		}
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
	if base.ModelDigest != candidate.ModelDigest && len(changed) == 0 {
		out.Unknown = append(out.Unknown, "model digest changed without a projected definition delta; decision or unprojected definition changes may require review")
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
