package projectmodel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

type definitionIndex struct {
	byID   map[string]core.Definition
	byKind map[string][]core.Definition
}

func Analyze(model core.Model, inventory []File) Report {
	r := Report{APIVersion: APIVersion, ModelDigest: model.Digest, Status: "succeeded"}
	idx := definitionIndex{byID: map[string]core.Definition{}, byKind: map[string][]core.Definition{}}
	for _, d := range model.Definitions {
		id := d.Identity().Key()
		idx.byID[id] = d
		idx.byKind[d.Kind] = append(idx.byKind[d.Kind], d)
	}
	for _, d := range idx.byKind[managerKind] {
		spec := d.Spec
		parent := refID(spec["parent"])
		owns := stringsFrom(spec["owns"])
		r.Managers = append(r.Managers, Manager{ID: d.Identity().Key(), Name: d.Metadata.Name, Namespace: d.Metadata.Namespace, Purpose: d.Purpose, Parent: parent, Owns: owns, Instructions: stringValue(spec["instructions"])})
	}
	sort.Slice(r.Managers, func(i, j int) bool { return r.Managers[i].ID < r.Managers[j].ID })
	managerByID := map[string]Manager{}
	for _, m := range r.Managers {
		managerByID[m.ID] = m
	}

	for _, d := range idx.byKind[statementKind] {
		id := d.Identity().Key()
		owner := nearestManager(d.Metadata.Namespace, r.Managers)
		if owner == "" {
			addFinding(&r, "ownership.statement-unmanaged", id, "Statement has no Manager at or above its namespace.", "error")
		}
		r.Statements = append(r.Statements, Statement{ID: id, Name: d.Metadata.Name, Namespace: d.Metadata.Namespace, Owner: owner, Category: stringValue(d.Spec["category"]), Description: stringValue(d.Spec["description"]), Public: boolValue(d.Spec["public"]), Uses: refIDs(d.Spec["uses"]), Requires: refIDs(d.Spec["requires"]), Source: d.Source.Path})
	}
	sort.Slice(r.Statements, func(i, j int) bool { return r.Statements[i].ID < r.Statements[j].ID })
	statementByID := map[string]Statement{}
	for _, s := range r.Statements {
		statementByID[s.ID] = s
	}

	for _, d := range idx.byKind[artifactKind] {
		id := d.Identity().Key()
		owner := nearestManager(d.Metadata.Namespace, r.Managers)
		if owner == "" {
			addFinding(&r, "ownership.artifact-unmanaged", id, "Artifact has no Manager at or above its namespace.", "error")
		}
		r.Artifacts = append(r.Artifacts, Artifact{ID: id, Name: d.Metadata.Name, Owner: owner, Role: stringValue(d.Spec["role"]), Realizes: refIDs(d.Spec["realizes"]), Paths: stringsFrom(d.Spec["paths"]), Checks: refIDs(d.Spec["checks"]), Required: boolValue(d.Spec["required"]), Reason: stringValue(d.Spec["reason"])})
	}
	sort.Slice(r.Artifacts, func(i, j int) bool { return r.Artifacts[i].ID < r.Artifacts[j].ID })
	artifactByID := map[string]Artifact{}
	for _, a := range r.Artifacts {
		artifactByID[a.ID] = a
	}

	for _, d := range idx.byKind[checkKind] {
		id := d.Identity().Key()
		owner := nearestManager(d.Metadata.Namespace, r.Managers)
		if owner == "" {
			addFinding(&r, "ownership.check-unmanaged", id, "Check has no Manager at or above its namespace.", "error")
		}
		r.Checks = append(r.Checks, Check{ID: id, Name: d.Metadata.Name, Owner: owner, Command: orderedStrings(d.Spec["command"]), Uses: refIDs(d.Spec["uses"]), Limitation: stringValue(d.Spec["limitation"])})
	}
	sort.Slice(r.Checks, func(i, j int) bool { return r.Checks[i].ID < r.Checks[j].ID })
	checkByID := map[string]Check{}
	for _, c := range r.Checks {
		checkByID[c.ID] = c
	}
	for _, d := range idx.byKind[decisionKind] {
		id := d.Identity().Key()
		owner := nearestManager(d.Metadata.Namespace, r.Managers)
		if owner == "" {
			addFinding(&r, "ownership.decision-unmanaged", id, "Decision has no Manager at or above its namespace.", "error")
		}
		r.Decisions = append(r.Decisions, Decision{ID: id, Name: d.Metadata.Name, Namespace: d.Metadata.Namespace, Owner: owner, Subject: refID(d.Spec["subject"]), Actor: refID(d.Spec["actor"]), Decision: stringValue(d.Spec["decision"]), Reason: stringValue(d.Spec["reason"]), Supersedes: refID(d.Spec["supersedes"]), Public: boolValue(d.Spec["public"]), Source: d.Source})
	}
	sort.Slice(r.Decisions, func(i, j int) bool { return r.Decisions[i].ID < r.Decisions[j].ID })
	decisionByID := map[string]Decision{}
	for _, d := range r.Decisions {
		decisionByID[d.ID] = d
	}
	for _, d := range idx.byKind[identityChangeKind] {
		id := d.Identity().Key()
		owner := nearestManager(d.Metadata.Namespace, r.Managers)
		if owner == "" {
			addFinding(&r, "ownership.identity-change-unmanaged", id, "IdentityChange has no Manager at or above its namespace.", "error")
		}
		r.IdentityChanges = append(r.IdentityChanges, IdentityChange{ID: id, Name: d.Metadata.Name, Namespace: d.Metadata.Namespace, Owner: owner, Operation: stringValue(d.Spec["operation"]), Previous: definitionIdentity(d.Spec["previous"]), Subject: refID(d.Spec["subject"]), Actor: refID(d.Spec["actorManager"]), Reason: stringValue(d.Spec["reason"]), Public: boolValue(d.Spec["public"]), Source: d.Source})
	}
	sort.Slice(r.IdentityChanges, func(i, j int) bool { return r.IdentityChanges[i].ID < r.IdentityChanges[j].ID })

	validateManagerTree(&r, managerByID)
	validateOwnershipSelectors(&r)
	for _, s := range r.Statements {
		for _, relation := range []struct {
			name string
			ids  []string
		}{{"uses", s.Uses}, {"requires", s.Requires}} {
			for _, targetID := range relation.ids {
				target, ok := statementByID[targetID]
				if !ok {
					addFinding(&r, "reference.statement-missing", s.ID, relation.name+" refers to a missing Statement.", "error")
					continue
				}
				if target.Owner != s.Owner && !target.Public {
					addFinding(&r, "reference.private-cross-manager", s.ID, relation.name+" crosses a Manager boundary to a private Statement.", "error")
				}
			}
		}
	}
	for _, a := range r.Artifacts {
		for _, sid := range a.Realizes {
			if target, ok := statementByID[sid]; !ok {
				addFinding(&r, "reference.artifact-statement-missing", a.ID, "Artifact realizes a missing Statement.", "error")
			} else if target.Owner != a.Owner && !target.Public {
				addFinding(&r, "reference.private-cross-manager", a.ID, "Artifact crosses a Manager boundary to realize a private Statement.", "error")
			}
		}
		for _, cid := range a.Checks {
			if _, ok := checkByID[cid]; !ok {
				addFinding(&r, "reference.artifact-check-missing", a.ID, "Artifact refers to a missing Check.", "error")
			}
		}
	}
	for _, c := range r.Checks {
		for _, sid := range c.Uses {
			if target, ok := statementByID[sid]; !ok {
				addFinding(&r, "reference.check-statement-missing", c.ID, "Check uses a missing Statement.", "error")
			} else if target.Owner != c.Owner && !target.Public {
				addFinding(&r, "reference.private-cross-manager", c.ID, "Check crosses a Manager boundary to use a private Statement.", "error")
			}
		}
	}
	validateDecisions(&r, managerByID, statementByID, decisionByID)
	validateIdentityChanges(&r, managerByID, statementByID)

	fileByPath := map[string]FileEntry{}
	casePaths := map[string]string{}
	orderedInventory := append([]File(nil), inventory...)
	sort.Slice(orderedInventory, func(i, j int) bool {
		if orderedInventory[i].Path != orderedInventory[j].Path {
			return orderedInventory[i].Path < orderedInventory[j].Path
		}
		if orderedInventory[i].Mode != orderedInventory[j].Mode {
			return orderedInventory[i].Mode < orderedInventory[j].Mode
		}
		return orderedInventory[i].Digest < orderedInventory[j].Digest
	})
	for _, f := range orderedInventory {
		clean, ok := normalizeFilePath(f.Path)
		if !ok {
			addFinding(&r, "path.inventory-invalid", f.Path, "Inventory path must be a normalized repository-relative file path.", "error")
			continue
		}
		folded := strings.ToLower(clean)
		if old, found := casePaths[folded]; found && old != clean {
			addFinding(&r, "path.case-collision", clean, "Inventory contains paths that differ only by case: "+old+" and "+clean+".", "error")
		} else {
			casePaths[folded] = clean
		}
		if _, found := fileByPath[clean]; found {
			addFinding(&r, "path.duplicate-inventory", clean, "Inventory contains the same path more than once.", "error")
			continue
		}
		owner := fileOwner(clean, r.Managers)
		class := "project-file"
		if strings.HasPrefix(clean, ".markitect/") || clean == ".markitect" {
			class = "markitect-input"
		}
		if owner == "" {
			r.Unknown = append(r.Unknown, clean)
		}
		for _, manager := range r.Managers {
			for _, raw := range manager.Owns {
				selector, selectorOK := normalizeSelector(raw, true)
				if selectorOK && selectorMatches(strings.ToLower(selector), strings.ToLower(clean)) && !selectorMatches(selector, clean) {
					addFinding(&r, "path.case-mismatch", manager.ID, "Inventory path differs in case from a declared ownership selector: "+clean, "error")
				}
			}
		}
		fileByPath[clean] = FileEntry{Path: clean, Digest: f.Digest, Mode: f.Mode, Owner: owner, Class: class, Statements: []string{}, Artifacts: []string{}, Checks: []string{}, Exists: true}
	}

	for _, a := range r.Artifacts {
		if a.Required && len(a.Paths) == 0 {
			addFinding(&r, "coverage.required-artifact-unmapped", a.ID, "Required Artifact has no expected path.", "incomplete")
		}
		for _, expected := range a.Paths {
			selector, ok := normalizeSelector(expected, false)
			if !ok {
				addFinding(&r, "path.artifact-invalid", a.ID, "Artifact path must be a normalized repository-relative path or trailing-slash prefix.", "error")
				continue
			}
			if fileOwner(selector, r.Managers) == "" {
				addFinding(&r, "ownership.artifact-path-unowned", a.ID, "Expected Artifact path is outside every Manager ownership selector.", "error")
			}
			for p := range fileByPath {
				if selectorMatches(strings.ToLower(selector), strings.ToLower(p)) && !selectorMatches(selector, p) {
					addFinding(&r, "path.case-mismatch", a.ID, "Inventory path differs in case from an expected Artifact path: "+p, "error")
				}
			}
			matched := false
			for p, entry := range fileByPath {
				if selectorMatches(selector, p) {
					matched = true
					entry.Artifacts = appendUnique(entry.Artifacts, a.ID)
					for _, sid := range a.Realizes {
						entry.Statements = appendUnique(entry.Statements, sid)
					}
					for _, cid := range a.Checks {
						entry.Checks = appendUnique(entry.Checks, cid)
					}
					fileByPath[p] = entry
				}
			}
			if !matched {
				if a.Required {
					addFinding(&r, "coverage.required-artifact-missing", a.ID, "Expected Artifact path is absent from the supplied inventory: "+selector, "incomplete")
				}
				if !strings.HasSuffix(selector, "/") {
					entry := fileByPath[selector]
					owner := fileOwner(selector, r.Managers)
					entry.Path, entry.Owner, entry.Class, entry.Exists = selector, owner, "expected-artifact", false
					entry.Statements, entry.Artifacts, entry.Checks = appendUnique(entry.Statements, a.Realizes...), appendUnique(entry.Artifacts, a.ID), appendUnique(entry.Checks, a.Checks...)
					fileByPath[selector] = entry
				}
			}
		}
	}
	for p, entry := range fileByPath {
		for _, s := range r.Statements {
			if s.Source == p {
				entry.Statements = appendUnique(entry.Statements, s.ID)
			}
		}
		for _, c := range r.Checks {
			if c.Name == p {
				entry.Checks = appendUnique(entry.Checks, c.ID)
			}
		}
		entry.Statements = sortedUnique(entry.Statements)
		entry.Artifacts = sortedUnique(entry.Artifacts)
		entry.Checks = sortedUnique(entry.Checks)
		fileByPath[p] = entry
	}
	for _, entry := range fileByPath {
		r.Files = append(r.Files, entry)
	}
	sort.Slice(r.Files, func(i, j int) bool { return r.Files[i].Path < r.Files[j].Path })
	r.Unknown = sortedUnique(r.Unknown)
	for _, f := range r.Findings {
		if f.Severity == "error" {
			r.Status = "failed"
		}
		if f.Severity == "incomplete" && r.Status != "failed" {
			r.Status = "incomplete"
		}
	}
	if len(r.Unknown) > 0 && r.Status == "succeeded" {
		r.Status = "incomplete"
	}
	r.Findings = sortedFindings(r.Findings)
	r.InventoryDigest = digest(orderedInventory)
	r.Digest = digest(struct {
		API, Model, Inventory, Status string
		Managers                      []Manager
		Statements                    []Statement
		Decisions                     []Decision
		IdentityChanges               []IdentityChange
		Artifacts                     []Artifact
		Checks                        []Check
		Files                         []FileEntry
		Findings                      []Finding
		Unknown                       []string
	}{r.APIVersion, r.ModelDigest, r.InventoryDigest, r.Status, r.Managers, r.Statements, r.Decisions, r.IdentityChanges, r.Artifacts, r.Checks, r.Files, r.Findings, r.Unknown})
	return r
}

func addFinding(r *Report, code, subject, message, severity string) {
	r.Findings = append(r.Findings, Finding{Code: code, Subject: subject, Message: message, Severity: severity})
}

func validateDecisions(r *Report, managers map[string]Manager, statements map[string]Statement, decisions map[string]Decision) {
	supersededBy := map[string]string{}
	for _, d := range r.Decisions {
		if _, ok := statements[d.Subject]; !ok {
			addFinding(r, "reference.decision-subject-missing", d.ID, "Decision refers to a missing Statement.", "error")
		} else if target := statements[d.Subject]; target.Owner != d.Owner && !target.Public {
			addFinding(r, "reference.private-cross-manager", d.ID, "Decision crosses a Manager boundary to a private Statement.", "error")
		}
		if _, ok := managers[d.Actor]; !ok {
			addFinding(r, "reference.decision-actor-missing", d.ID, "Decision actor is not a declared Manager.", "error")
		} else if d.Owner != "" && !managerInScope(d.Actor, d.Owner, managers) {
			addFinding(r, "decision.actor-out-of-scope", d.ID, "Recorded Decision actor Manager is outside the responsible Manager's scope.", "error")
		}
		if d.Supersedes == "" {
			continue
		}
		old, ok := decisions[d.Supersedes]
		if !ok {
			addFinding(r, "reference.decision-supersedes-missing", d.ID, "Decision supersedes a missing Decision.", "error")
			continue
		}
		if old.Subject != d.Subject {
			addFinding(r, "decision.supersedes-subject-mismatch", d.ID, "A Decision may supersede only a Decision about the same Statement.", "error")
		}
		if old.Owner != d.Owner && !old.Public {
			addFinding(r, "reference.private-cross-manager", d.ID, "Decision refers across a Manager boundary to a private superseded Decision.", "error")
		}
		if prior, exists := supersededBy[old.ID]; exists && prior != d.ID {
			addFinding(r, "decision.supersedes-ambiguous", d.ID, "A Decision is superseded by more than one Decision.", "error")
		} else {
			supersededBy[old.ID] = d.ID
		}
	}
	state := map[string]uint8{}
	var visit func(string, []string)
	visit = func(id string, stack []string) {
		if state[id] == 2 {
			return
		}
		if state[id] == 1 {
			addFinding(r, "decision.supersedes-cycle", id, "Decision supersession contains a cycle.", "error")
			return
		}
		state[id] = 1
		if d, ok := decisions[id]; ok && d.Supersedes != "" {
			visit(d.Supersedes, append(stack, id))
		}
		state[id] = 2
	}
	for id := range decisions {
		visit(id, nil)
	}
}

func validateIdentityChanges(r *Report, managers map[string]Manager, statements map[string]Statement) {
	previousOwner := map[string]string{}
	subjectOwner := map[string]string{}
	transition := map[string]string{}
	for _, c := range r.IdentityChanges {
		prev := c.Previous
		if !prev.Valid() || prev.Kind != statementKind {
			addFinding(r, "identity-change.previous-invalid", c.ID, "Previous must be a valid full historical Statement identity.", "error")
		}
		if prev.Valid() && prev.Kind == statementKind {
			if prior, exists := previousOwner[prev.Key()]; exists && prior != c.ID {
				addFinding(r, "identity-change.previous-ambiguous", c.ID, "A historical Statement identity may be declared by only one IdentityChange.", "error")
			} else {
				previousOwner[prev.Key()] = c.ID
			}
			if _, active := statements[prev.Key()]; active && (c.Operation == "renamed" || c.Operation == "retired") {
				addFinding(r, "identity-change.previous-still-active", c.ID, "A renamed or retired historical identity is still an active Statement.", "error")
			}
		}
		if _, ok := managers[c.Actor]; !ok {
			addFinding(r, "reference.identity-change-actor-missing", c.ID, "IdentityChange actorManager is not a declared Manager.", "error")
		} else if c.Owner != "" && !managerInScope(c.Actor, c.Owner, managers) {
			addFinding(r, "identity-change.actor-out-of-scope", c.ID, "Recorded IdentityChange actorManager is outside the responsible Manager's scope.", "error")
		}
		if c.Operation == "retired" {
			if c.Subject != "" {
				addFinding(r, "identity-change.retired-subject", c.ID, "A retired identity must not declare a current Statement subject.", "error")
			}
			continue
		}
		if c.Operation != "renamed" && c.Operation != "replaced" {
			continue
		}
		if c.Subject == "" {
			addFinding(r, "identity-change.subject-required", c.ID, "Renamed and replaced identities require an explicit current Statement subject.", "error")
			continue
		}
		target, ok := statements[c.Subject]
		if !ok {
			addFinding(r, "reference.identity-change-subject-missing", c.ID, "IdentityChange refers to a missing current Statement.", "error")
		} else if target.Owner != c.Owner && !target.Public {
			addFinding(r, "reference.private-cross-manager", c.ID, "IdentityChange crosses a Manager boundary to a private Statement.", "error")
		}
		if prev.Valid() && prev.Key() == c.Subject {
			addFinding(r, "identity-change.self-change", c.ID, "Previous historical identity cannot equal the current Statement subject.", "error")
		}
		if prior, exists := subjectOwner[c.Subject]; exists && prior != c.ID {
			addFinding(r, "identity-change.subject-ambiguous", c.ID, "A current Statement may be the subject of only one identity transition.", "error")
		} else {
			subjectOwner[c.Subject] = c.ID
		}
		if prev.Valid() && prev.Kind == statementKind {
			transition[prev.Key()] = c.Subject
		}
	}
	state := map[string]uint8{}
	var visit func(string)
	visit = func(id string) {
		if state[id] == 2 {
			return
		}
		if state[id] == 1 {
			if changeID := previousOwner[id]; changeID != "" {
				addFinding(r, "identity-change.cycle", changeID, "IdentityChange transitions contain a cycle.", "error")
			}
			return
		}
		state[id] = 1
		if next := transition[id]; next != "" {
			visit(next)
		}
		state[id] = 2
	}
	for id := range transition {
		visit(id)
	}
}

func managerInScope(actor, owner string, managers map[string]Manager) bool {
	seen := map[string]bool{}
	for owner != "" && !seen[owner] {
		if owner == actor {
			return true
		}
		seen[owner] = true
		owner = managers[owner].Parent
	}
	return false
}

func stringValue(v any) string { s, _ := v.(string); return s }
func boolValue(v any) bool     { b, _ := v.(bool); return b }
func stringsFrom(v any) []string {
	if s, ok := v.(string); ok {
		return []string{s}
	}
	values, _ := v.([]any)
	out := make([]string, 0, len(values))
	for _, x := range values {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return sortedUnique(out)
}
func orderedStrings(v any) []string {
	if value, ok := v.(string); ok {
		return []string{value}
	}
	if values, ok := v.([]string); ok {
		return append([]string(nil), values...)
	}
	values, _ := v.([]any)
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			out = append(out, text)
		}
	}
	return out
}
func refID(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	api, _ := m["apiVersion"].(string)
	kind, _ := m["kind"].(string)
	ns, _ := m["namespace"].(string)
	name, _ := m["name"].(string)
	return (core.DefinitionIdentity{APIVersion: api, Kind: kind, Namespace: ns, Name: name}).Key()
}

func definitionIdentity(v any) core.DefinitionIdentity {
	m, ok := v.(map[string]any)
	if !ok {
		return core.DefinitionIdentity{}
	}
	api, _ := m["apiVersion"].(string)
	kind, _ := m["kind"].(string)
	ns, _ := m["namespace"].(string)
	name, _ := m["name"].(string)
	return core.DefinitionIdentity{APIVersion: api, Kind: kind, Namespace: ns, Name: name}
}
func refIDs(v any) []string {
	if one := refID(v); one != "" {
		return []string{one}
	}
	values, _ := v.([]any)
	out := make([]string, 0, len(values))
	for _, x := range values {
		if id := refID(x); id != "" {
			out = append(out, id)
		}
	}
	return sortedUnique(out)
}
func nearestManager(namespace string, managers []Manager) string {
	return nearestManagerAncestor(namespace, managers, "")
}
func nearestManagerAncestor(namespace string, managers []Manager, exclude string) string {
	best := ""
	bestLen := -1
	for _, m := range managers {
		if m.ID != exclude && (namespace == m.Namespace || strings.HasPrefix(namespace, m.Namespace+".") || m.Namespace == "") {
			if len(m.Namespace) > bestLen {
				best, bestLen = m.ID, len(m.Namespace)
			}
		}
	}
	return best
}
func validateManagerTree(r *Report, managers map[string]Manager) {
	roots := 0
	for _, m := range r.Managers {
		if m.Parent == "" {
			roots++
			if m.Namespace != "" {
				addFinding(r, "manager.root-namespace", m.ID, "Root Manager namespace must be empty.", "error")
			}
			continue
		}
		parent, ok := managers[m.Parent]
		if !ok {
			addFinding(r, "manager.parent-missing", m.ID, "Manager parent does not exist or is not a Manager.", "error")
			continue
		}
		if !namespaceDescendant(parent.Namespace, m.Namespace) {
			addFinding(r, "manager.namespace-parent-mismatch", m.ID, "Manager namespace must be a strict dot-delimited descendant of its declared parent.", "error")
		} else if nearest := nearestManagerAncestor(m.Namespace, r.Managers, m.ID); nearest != m.Parent {
			addFinding(r, "manager.namespace-parent-mismatch", m.ID, "Manager parent must be the nearest declared Manager ancestor in namespace order.", "error")
		}
		seen := map[string]bool{m.ID: true}
		current := parent
		for current.ID != "" {
			if seen[current.ID] {
				addFinding(r, "manager.parent-cycle", m.ID, "Manager parent relation contains a cycle.", "error")
				break
			}
			seen[current.ID] = true
			current = managers[current.Parent]
		}
	}
	if roots != 1 {
		addFinding(r, "manager.root-count", "", fmt.Sprintf("Project model must declare exactly one root Manager; found %d.", roots), "error")
	}
}
func namespaceDescendant(parent, child string) bool {
	if child == "" {
		return false
	}
	if parent == "" {
		return true
	}
	return strings.HasPrefix(child, parent+".")
}
func validateOwnershipSelectors(r *Report) {
	type item struct{ manager, selector string }
	var all []item
	for _, m := range r.Managers {
		for _, raw := range m.Owns {
			s, ok := normalizeSelector(raw, true)
			if !ok {
				addFinding(r, "path.ownership-invalid", m.ID, "Manager ownership must be a normalized repository-relative exact path or trailing-slash prefix; '.' means the whole repository.", "error")
				continue
			}
			all = append(all, item{m.ID, s})
		}
		if m.Parent != "" {
			parent, ok := func() (Manager, bool) {
				for _, candidate := range r.Managers {
					if candidate.ID == m.Parent {
						return candidate, true
					}
				}
				return Manager{}, false
			}()
			if ok {
				for _, raw := range m.Owns {
					childSelector, childOK := normalizeSelector(raw, true)
					if !childOK {
						continue
					}
					delegated := false
					for _, parentRaw := range parent.Owns {
						parentSelector, parentOK := normalizeSelector(parentRaw, true)
						if parentOK && selectorContains(parentSelector, childSelector) {
							delegated = true
							break
						}
					}
					if !delegated {
						addFinding(r, "ownership.undelegated", m.ID, "Child Manager owns a path outside its parent's declared ownership.", "error")
					}
				}
			}
		}
	}
	for i := range all {
		for j := i + 1; j < len(all); j++ {
			a, b := all[i], all[j]
			if selectorsOverlap(strings.ToLower(a.selector), strings.ToLower(b.selector)) && !selectorsOverlap(a.selector, b.selector) {
				addFinding(r, "path.selector-case-collision", a.manager, "Ownership selectors differ only by case.", "error")
			}
			if selectorsOverlap(a.selector, b.selector) && !ancestorManager(a.manager, b.manager, r.Managers) && !ancestorManager(b.manager, a.manager, r.Managers) {
				addFinding(r, "ownership.sibling-overlap", a.manager, "Sibling Managers claim overlapping repository paths.", "error")
			}
		}
	}
}
func ancestorManager(ancestor, child string, managers []Manager) bool {
	byID := map[string]Manager{}
	for _, m := range managers {
		byID[m.ID] = m
	}
	seen := map[string]bool{}
	for child != "" && !seen[child] {
		if child == ancestor {
			return true
		}
		seen[child] = true
		child = byID[child].Parent
	}
	return false
}
func normalizeFilePath(raw string) (string, bool) {
	s, ok := normalizeSelector(raw, false)
	return s, ok && !strings.HasSuffix(s, "/") && s != "."
}
func normalizeSelector(raw string, allowRoot bool) (string, bool) {
	if raw == "." {
		return raw, allowRoot
	}
	if raw == "" || strings.Contains(raw, "\\") || strings.Contains(raw, ":") || strings.HasPrefix(raw, "/") {
		return "", false
	}
	prefix := strings.HasSuffix(raw, "/")
	body := strings.TrimSuffix(raw, "/")
	if body == "" || path.Clean(body) != body || body == ".." || strings.HasPrefix(body, "../") {
		return "", false
	}
	if prefix {
		body += "/"
	}
	return body, true
}
func selectorMatches(selector, file string) bool {
	if selector == "." {
		return true
	}
	if strings.HasSuffix(selector, "/") {
		return strings.HasPrefix(file, selector)
	}
	return selector == file
}
func selectorsOverlap(a, b string) bool {
	if a == "." || b == "." {
		return true
	}
	ap, bp := strings.HasSuffix(a, "/"), strings.HasSuffix(b, "/")
	if !ap && !bp {
		return a == b
	}
	if ap && bp {
		return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
	}
	if ap {
		return strings.HasPrefix(b, a)
	}
	return strings.HasPrefix(a, b)
}
func selectorContains(parent, child string) bool {
	if parent == "." {
		return true
	}
	if parent == child {
		return true
	}
	if strings.HasSuffix(parent, "/") {
		if strings.HasSuffix(child, "/") {
			return strings.HasPrefix(child, parent)
		}
		return strings.HasPrefix(child, parent)
	}
	return false
}
func fileOwner(file string, managers []Manager) string {
	best := ""
	bestLen := -1
	bestDepth := -1
	for _, m := range managers {
		for _, raw := range m.Owns {
			s, ok := normalizeSelector(raw, true)
			if ok && selectorMatches(s, file) {
				depth := strings.Count(m.Namespace, ".") + 1
				if len(s) > bestLen || len(s) == bestLen && depth > bestDepth {
					best, bestLen, bestDepth = m.ID, len(s), depth
				}
			}
		}
	}
	return best
}
func appendUnique(dst []string, src ...string) []string {
	seen := map[string]bool{}
	for _, v := range dst {
		seen[v] = true
	}
	for _, v := range src {
		if v != "" && !seen[v] {
			dst = append(dst, v)
			seen[v] = true
		}
	}
	return dst
}
func sortedUnique(values []string) []string {
	values = append([]string(nil), values...)
	sort.Strings(values)
	out := values[:0]
	for _, v := range values {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}
func sortedFindings(values []Finding) []Finding {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i], values[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		if a.Severity != b.Severity {
			return a.Severity < b.Severity
		}
		return a.Message < b.Message
	})
	return values
}
func digest(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
