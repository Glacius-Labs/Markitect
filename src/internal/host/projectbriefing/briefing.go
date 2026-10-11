// Package projectbriefing records deterministic briefings for accepted project
// model changes. It compares compiled declarations only; it does not infer
// meaning from prose. Accepted history is the first-parent line of the
// checked-out HEAD: briefings count when recorded on it, resolutions when their
// delivered result is in HEAD's tree, and dismissals on the branch that made
// them. Other entries stay provisional. Event identity is the model change's
// content, so a change keeps its identity when a topic is merged or squashed.
package projectbriefing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

const APIVersion = "markitect.example.org/project-model-briefing/v1alpha2"

var (
	ErrUncommittedModel    = errors.New("briefings require explicit committed model revisions")
	ErrMissingProvenance   = errors.New("accepted model change requires explicit decision provenance")
	ErrRevisionNotAncestor = errors.New("since revision is not an ancestor of the accepted revision")
	ErrInvalidBundle       = errors.New("briefing bundle is invalid")
)

// Provenance binds the source decision reference, declared actor, and authority
// policy. Automatically collected Git metadata is source attribution only and
// does not authenticate a human decision.
type Provenance struct {
	DecisionReference string `json:"decisionReference"`
	Actor             string `json:"actor"`
	Authority         string `json:"authority"`
}

// Event is one changed definition. Its ID and Digest come from the change's
// content and the previous event for the same definition (see eventDigest):
// the same change on the same history keeps one identity wherever it is
// briefed, and a change re-applied after a revert gets a new one. Provenance
// and affected references stay per briefing.
type Event struct {
	ID                string                  `json:"id"`
	Digest            string                  `json:"digest"`
	DefinitionID      core.DefinitionIdentity `json:"definitionId"`
	Change            string                  `json:"change"`
	Before            *core.Definition        `json:"before,omitempty"`
	After             *core.Definition        `json:"after,omitempty"`
	AffectedManagers  []string                `json:"affectedManagers"`
	AffectedArtifacts []string                `json:"affectedArtifacts"`
	Category          string                  `json:"category"`
	Severity          string                  `json:"severity"`
	Provenance        Provenance              `json:"provenance"`
	// Predecessor is the ID of the previous event for the same definition on
	// the first-parent line it was briefed on, empty for the first.
	Predecessor string `json:"predecessor,omitempty"`
}

type Briefing struct {
	ID          string                   `json:"id"`
	ManagerID   string                   `json:"managerId,omitempty"`
	Revision    string                   `json:"revision"`
	ModelDigest string                   `json:"modelDigest"`
	EventIDs    []string                 `json:"eventIds"`
	Contracts   []projectmodel.Statement `json:"contracts,omitempty"`
	Summary     string                   `json:"summary"`
}

type Bundle struct {
	APIVersion        string                    `json:"apiVersion"`
	Digest            string                    `json:"digest"`
	SinceRevision     string                    `json:"sinceRevision"`
	Revision          string                    `json:"revision"`
	SinceModelDigest  string                    `json:"sinceModelDigest"`
	ModelDigest       string                    `json:"modelDigest"`
	SinceReportDigest string                    `json:"sinceReportDigest"`
	ReportDigest      string                    `json:"reportDigest"`
	Provenance        Provenance                `json:"provenance"`
	Impact            projectmodel.ChangeImpact `json:"impact"`
	Events            []Event                   `json:"events"`
	Global            Briefing                  `json:"global"`
	Managers          []Briefing                `json:"managers"`
}

// Generate loads the two explicitly named committed revisions and generates
// stable global and manager-scoped briefing records.
func Generate(root, sinceRevision, revision string, provenance Provenance) (Bundle, error) {
	if strings.TrimSpace(sinceRevision) == "" || strings.TrimSpace(revision) == "" || sinceRevision == revision {
		return Bundle{}, ErrUncommittedModel
	}
	before, err := projectwork.Load(root, sinceRevision)
	if err != nil {
		return Bundle{}, fmt.Errorf("load since revision: %w", err)
	}
	after, err := projectwork.Load(root, revision)
	if err != nil {
		return Bundle{}, fmt.Errorf("load revision: %w", err)
	}
	if before.Provisional || after.Provisional || before.Revision == after.Revision {
		return Bundle{}, ErrUncommittedModel
	}
	if err := requireAncestor(root, before.Revision, after.Revision); err != nil {
		return Bundle{}, err
	}
	state, _, err := readStore(root)
	if err != nil {
		return Bundle{}, err
	}
	predecessors, err := lastEvents(root, state, before.Revision)
	if err != nil {
		return Bundle{}, err
	}
	return build(before, after, provenance, predecessors)
}

// lastEvents returns, per definition key, the ID of the newest event that
// state briefs for that definition on the first-parent line ending at
// revision. Generate chains new events to it.
func lastEvents(root string, state Store, revision string) (map[string]string, error) {
	revisions, err := firstParentRevisions(root, revision)
	if err != nil {
		return nil, fmt.Errorf("enumerate briefed history before %s: %w", revision, err)
	}
	positions := make(map[string]int, len(revisions))
	for i, item := range revisions {
		positions[item] = i
	}
	last, at := map[string]string{}, map[string]int{}
	for _, bundle := range state.Briefings {
		i, onLine := positions[bundle.Revision]
		if !onLine {
			continue
		}
		for _, event := range bundle.Events {
			key := event.DefinitionID.Key()
			if prior, seen := at[key]; !seen || i > prior {
				last[key], at[key] = event.ID, i
			}
		}
	}
	return last, nil
}

// Build compares already loaded, fixed project revisions. Both projects must
// come from committed snapshots and carry distinct full Git object IDs. Its
// events start their definitions' histories; Generate continues the briefed
// history instead.
func Build(before, after *projectwork.Project, provenance Provenance) (Bundle, error) {
	return build(before, after, provenance, nil)
}

// build is Build with each event chained to predecessors[key].
func build(before, after *projectwork.Project, provenance Provenance, predecessors map[string]string) (Bundle, error) {
	if before == nil || after == nil || before.Provisional || after.Provisional || !fullObjectID(before.Revision) || !fullObjectID(after.Revision) || before.Revision == after.Revision {
		return Bundle{}, ErrUncommittedModel
	}
	if strings.TrimSpace(provenance.DecisionReference) == "" || strings.TrimSpace(provenance.Actor) == "" || strings.TrimSpace(provenance.Authority) == "" {
		return Bundle{}, ErrMissingProvenance
	}
	impact := projectmodel.Impact(before.Report, after.Report)
	oldDefs, newDefs := definitionMap(before.Model.Definitions), definitionMap(after.Model.Definitions)
	ids := make([]string, 0, len(oldDefs)+len(newDefs))
	seen := map[string]bool{}
	for id := range oldDefs {
		ids = append(ids, id)
		seen[id] = true
	}
	for id := range newDefs {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	changed := make([]string, 0)
	if before.Model.Digest != after.Model.Digest {
		for _, id := range ids {
			if !sameDefinition(oldDefs[id], newDefs[id]) {
				changed = append(changed, id)
			}
		}
	}
	bundle := Bundle{APIVersion: APIVersion, SinceRevision: before.Revision, Revision: after.Revision, SinceModelDigest: before.Model.Digest, ModelDigest: after.Model.Digest, SinceReportDigest: before.Report.Digest, ReportDigest: after.Report.Digest, Provenance: provenance, Impact: impact, Events: []Event{}, Managers: []Briefing{}}
	for _, key := range changed {
		var oldValue, newValue *core.Definition
		if value, ok := oldDefs[key]; ok {
			copied := value
			oldValue = &copied
		}
		if value, ok := newDefs[key]; ok {
			copied := value
			newValue = &copied
		}
		identity := oldDefs[key].Identity()
		if newValue != nil {
			identity = newValue.Identity()
		}
		change := "modified"
		if oldValue == nil {
			change = "added"
		} else if newValue == nil {
			change = "removed"
		}
		managers := managersForDefinition(before.Report, after.Report, key)
		artifacts := artifactsForDefinition(before.Report, after.Report, key)
		digest := eventDigest(predecessors[key], key, change, oldValue, newValue)
		bundle.Events = append(bundle.Events, Event{ID: "model-change-" + digest[:24], Digest: digest, Predecessor: predecessors[key], DefinitionID: identity, Change: change, Before: oldValue, After: newValue, AffectedManagers: managers, AffectedArtifacts: artifacts, Category: "information", Severity: "info", Provenance: provenance})
	}
	eventIDs := make([]string, 0, len(bundle.Events))
	managerSet := map[string]bool{}
	for _, event := range bundle.Events {
		eventIDs = append(eventIDs, event.ID)
		for _, manager := range event.AffectedManagers {
			managerSet[manager] = true
		}
	}
	bundle.Global = makeBriefing("", after, eventIDs, nil, fmt.Sprintf("Accepted model revision %s changes %d declared definitions.", after.Revision, len(eventIDs)))
	for _, managerID := range sortedKeys(managerSet) {
		contextProject := after
		if _, err := projectmodel.Context(after.Report, managerID); err != nil {
			contextProject = before
		}
		ctx, err := projectmodel.Context(contextProject.Report, managerID)
		if err != nil {
			continue
		}
		contracts := relevantContracts(ctx.Contracts, changed, oldDefs, newDefs)
		managerEvents := make([]string, 0)
		for _, event := range bundle.Events {
			if contains(event.AffectedManagers, managerID) {
				managerEvents = append(managerEvents, event.ID)
			}
		}
		bundle.Managers = append(bundle.Managers, makeBriefing(managerID, after, managerEvents, contracts, fmt.Sprintf("Accepted model revision %s changes %d declared definitions relevant to manager %s.", after.Revision, len(managerEvents), managerID)))
	}
	bundle.Digest = FullBundleDigest(bundle)
	return bundle, nil
}

// FullBundleDigest seals all stable bundle content. It detects accidental or
// unsophisticated record mutation; it does not authenticate the decision actor.
func FullBundleDigest(bundle Bundle) string {
	bundle.Digest = ""
	return hash(bundle)
}

// eventDigest identifies a model change by the previous event for the same
// definition and the change's content: the definition key, the kind of change
// and the compiled definition before and after, source file digest included.
// The revisions it was briefed at are not part of it, so a merge commit or
// squash that brings the same change onto a line with the same predecessor
// keeps its identity.
func eventDigest(predecessor, key, change string, before, after *core.Definition) string {
	return hash(struct {
		Predecessor, Key, Change string
		Before, After            *core.Definition
	}{predecessor, key, change, before, after})
}

func validEventID(id string) bool {
	const prefix = "model-change-"
	if !strings.HasPrefix(id, prefix) || len(id) != len(prefix)+24 {
		return false
	}
	_, err := hex.DecodeString(id[len(prefix):])
	return err == nil
}

func makeBriefing(manager string, p *projectwork.Project, ids []string, contracts []projectmodel.Statement, summary string) Briefing {
	return Briefing{ID: briefingID(manager, p.Revision, p.Model.Digest, ids), ManagerID: manager, Revision: p.Revision, ModelDigest: p.Model.Digest, EventIDs: ids, Contracts: contracts, Summary: summary}
}

func definitionMap(defs []core.Definition) map[string]core.Definition {
	out := make(map[string]core.Definition, len(defs))
	for _, d := range defs {
		out[d.Identity().Key()] = d
	}
	return out
}
func sameDefinition(a, b core.Definition) bool { return hash(a) == hash(b) }
func hash(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func fullObjectID(id string) bool {
	if len(id) != 40 && len(id) != 64 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func requireAncestor(root, sinceRevision, revision string) error {
	if sinceRevision == revision {
		return ErrUncommittedModel
	}
	if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", sinceRevision, revision); err != nil {
		return fmt.Errorf("%w: %s..%s: %v", ErrRevisionNotAncestor, sinceRevision, revision, err)
	}
	return nil
}
func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func managersForDefinition(a, b projectmodel.Report, key string) []string {
	set := map[string]bool{}
	for _, report := range []projectmodel.Report{a, b} {
		for _, s := range report.Statements {
			if definitionKey(projectmodel.APIVersion, "Statement", s.Namespace, s.Name) == key {
				set[s.Owner] = true
				addAncestors(set, s.Owner, report)
			}
		}
		for _, x := range report.Artifacts {
			if definitionKey(projectmodel.APIVersion, "Artifact", ownerNamespace(report, x.Owner), x.Name) == key {
				set[x.Owner] = true
				addAncestors(set, x.Owner, report)
			}
		}
		for _, x := range report.Managers {
			if definitionKey(projectmodel.APIVersion, "Manager", x.Namespace, x.Name) == key {
				set[x.ID] = true
				addAncestors(set, x.ID, report)
			}
		}
		for _, x := range report.Checks {
			if definitionKey(projectmodel.APIVersion, "Check", ownerNamespace(report, x.Owner), x.Name) == key {
				set[x.Owner] = true
				addAncestors(set, x.Owner, report)
			}
		}
	}
	return sortedKeys(set)
}
func artifactsForDefinition(a, b projectmodel.Report, key string) []string {
	set := map[string]bool{}
	for _, r := range []projectmodel.Report{a, b} {
		for _, x := range r.Artifacts {
			if definitionKey(projectmodel.APIVersion, "Artifact", ownerNamespace(r, x.Owner), x.Name) == key {
				set[x.ID] = true
			}
			for _, sid := range x.Realizes {
				if reportStatementKey(r, sid) == key {
					set[x.ID] = true
				}
			}
		}
		for _, c := range r.Checks {
			for _, sid := range c.Uses {
				if reportStatementKey(r, sid) == key {
					for _, x := range r.Artifacts {
						if contains(x.Checks, c.ID) {
							set[x.ID] = true
						}
					}
				}
			}
		}
	}
	return sortedKeys(set)
}
func definitionKey(api, kind, ns, name string) string {
	return core.DefinitionIdentity{APIVersion: api, Kind: kind, Namespace: ns, Name: name}.Key()
}
func reportStatementKey(r projectmodel.Report, id string) string {
	for _, s := range r.Statements {
		if s.ID == id {
			return definitionKey(projectmodel.APIVersion, "Statement", s.Namespace, s.Name)
		}
	}
	return ""
}
func ownerNamespace(r projectmodel.Report, owner string) string {
	for _, m := range r.Managers {
		if m.ID == owner {
			return m.Namespace
		}
	}
	return ""
}
func addAncestors(set map[string]bool, id string, r projectmodel.Report) {
	parents := map[string]string{}
	for _, m := range r.Managers {
		parents[m.ID] = m.Parent
	}
	seen := map[string]bool{}
	for id != "" && !seen[id] {
		seen[id] = true
		id = parents[id]
		if id != "" {
			set[id] = true
		}
	}
}
func relevantContracts(contracts []projectmodel.Statement, changed []string, oldDefs, newDefs map[string]core.Definition) []projectmodel.Statement {
	out := make([]projectmodel.Statement, 0)
	for _, s := range contracts {
		key := definitionKey(projectmodel.APIVersion, "Statement", s.Namespace, s.Name)
		relevant := false
		for _, id := range changed {
			if oldDefs[id].Kind == "Statement" || newDefs[id].Kind == "Statement" {
				if id == key {
					relevant = true
				}
				if contains(s.Uses, id) || contains(s.Requires, id) {
					relevant = true
				}
			}
		}
		if relevant {
			out = append(out, s)
		}
	}
	return out
}
