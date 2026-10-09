// Package projectgraph maps the authoritative project model and report into a
// privacy-scoped, read-only projectknowledge index.
package projectgraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectknowledge"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

var ErrInvalidSelection = errors.New("exactly one project or Manager scope is required")

// Selection requires either a named Manager context or an explicit whole
// project view. The zero value grants no scope.
type Selection struct {
	ManagerID    string `json:"managerId,omitempty"`
	ProjectScope bool   `json:"projectScope,omitempty"`
}

// Scope is the safe adapter input boundary. DefinitionIDs are the exact Core
// identities admitted to this view; FileIDs are generated only from exact
// projectmodel.Report entries. Evidence adapters should use these identifiers
// when selecting records and must return only safe, structured facts.
type Scope struct {
	ID            string   `json:"id"`
	DefinitionIDs []string `json:"definitionIds"`
	FileIDs       []string `json:"fileIds"`
	Paths         []string `json:"paths"`
}

// ExtraFacts contains records selected by an external, typed Host adapter.
// SelectedIDs makes admission explicit; endpoints are still intersected with
// the graph view and payload fields are selected by the adapter.
type ExtraFacts struct {
	ScopeID          string                        `json:"scopeId"`
	SelectedIDs      []string                      `json:"selectedIds"`
	Facts            projectknowledge.ProjectFacts `json:"facts"`
	CaptureDigest    string                        `json:"captureDigest,omitempty"`
	Completeness     string                        `json:"completeness,omitempty"`
	EvidenceSelected bool                          `json:"evidenceSelected,omitempty"`
	SourceBindings   []RecordBinding               `json:"sourceBindings,omitempty"`
}

// RecordBinding preserves the safe source and schema digests supplied by the
// selected adapter. It intentionally omits record identifiers and bodies.
type RecordBinding struct {
	Kind          string `json:"kind"`
	Digest        string `json:"digest"`
	Schema        string `json:"schema"`
	Source        string `json:"source"`
	ModelDigest   string `json:"modelDigest,omitempty"`
	Revision      string `json:"revision,omitempty"`
	RuntimeDigest string `json:"runtimeDigest,omitempty"`
	DigestKind    string `json:"digestKind,omitempty"`
}

type Coverage struct {
	Model     string `json:"model"`
	Inventory string `json:"inventory"`
	Artifacts string `json:"artifacts"`
	Checks    string `json:"checks"`
	Records   string `json:"records"`
}

// View binds an immutable index to the project source and explicit scope used
// to build it. Index is exposed through read-only methods only.
type View struct {
	index          *projectknowledge.Index
	selection      Selection
	scope          Scope
	revision       string
	provisional    bool
	projectDigest  string
	coverage       Coverage
	captureDigest  string
	sourceBindings []RecordBinding
}

func (v *View) Index() *projectknowledge.Index {
	if v == nil {
		return nil
	}
	return v.index
}

// VisibleScope returns a defensive copy for a Host-side evidence adapter.
func (v *View) VisibleScope() Scope {
	if v == nil {
		return Scope{}
	}
	return Scope{ID: v.scope.ID, DefinitionIDs: append([]string(nil), v.scope.DefinitionIDs...), FileIDs: append([]string(nil), v.scope.FileIDs...), Paths: append([]string(nil), v.scope.Paths...)}
}

// SafeScope computes the deterministic scope key and the exact identifiers
// that a secondary evidence adapter may use for source-bound selection.
func SafeScope(project *projectwork.Project, selection Selection) (Scope, error) {
	if project == nil {
		return Scope{}, errors.New("project is required")
	}
	_, selected, err := selectContext(project, selection)
	if err != nil {
		return Scope{}, err
	}
	coreIDs := make([]string, 0, len(selected))
	for id := range selected {
		coreIDs = append(coreIDs, id)
	}
	fileIDs := make([]string, 0)
	paths := make([]string, 0)
	for _, entry := range project.Report.Files {
		if selection.ProjectScope || entry.Owner == selection.ManagerID {
			fileIDs = append(fileIDs, fileID(entry.Path))
			paths = append(paths, entry.Path)
		}
	}
	sort.Strings(coreIDs)
	sort.Strings(fileIDs)
	sort.Strings(paths)
	key := struct {
		Project                   string
		Selection                 Selection
		Definitions, Files, Paths []string
	}{project.Digest, selection, coreIDs, fileIDs, paths}
	b, _ := json.Marshal(key)
	h := sha256.Sum256(b)
	return Scope{ID: "sha256:" + hex.EncodeToString(h[:]), DefinitionIDs: coreIDs, FileIDs: fileIDs, Paths: paths}, nil
}

// Build maps the Core model and existing ProjectModel Report into a selected
// graph. Extra evidence is optional and is always intersected with this view.
func Build(project *projectwork.Project, selection Selection, extra *ExtraFacts) (*View, error) {
	if project == nil {
		return nil, errors.New("project is required")
	}
	if project.Snapshot == nil || project.Digest == "" || project.Model.Digest == "" || project.Report.ModelDigest != project.Model.Digest {
		return nil, errors.New("project graph requires a complete, source-bound loaded project")
	}
	context, allowed, err := selectContext(project, selection)
	if err != nil {
		return nil, err
	}
	scope, err := SafeScope(project, selection)
	if err != nil {
		return nil, err
	}
	if extra != nil && extra.ScopeID != scope.ID {
		return nil, errors.New("extra facts were selected for a different project graph scope")
	}
	if extra != nil && (project.Snapshot == nil || extra.Facts.SnapshotDigest != project.Snapshot.Digest() || extra.Facts.ProjectDigest != project.Digest) {
		return nil, errors.New("extra facts are bound to a different project source")
	}
	definitions := make(map[string]core.Definition, len(project.Model.Definitions))
	for _, d := range project.Model.Definitions {
		definitions[d.Identity().Key()] = d
	}
	report := project.Report
	managerByID := map[string]projectmodel.Manager{}
	for _, m := range report.Managers {
		managerByID[m.ID] = m
	}
	statementByID := map[string]projectmodel.Statement{}
	for _, s := range report.Statements {
		statementByID[s.ID] = s
	}
	artifactByID := map[string]projectmodel.Artifact{}
	for _, a := range report.Artifacts {
		artifactByID[a.ID] = a
	}
	checkByID := map[string]projectmodel.Check{}
	for _, c := range report.Checks {
		checkByID[c.ID] = c
	}
	decisionByID := map[string]projectmodel.Decision{}
	for _, d := range report.Decisions {
		decisionByID[d.ID] = d
	}
	changeByID := map[string]projectmodel.IdentityChange{}
	for _, c := range report.IdentityChanges {
		changeByID[c.ID] = c
	}

	baseFacts := projectknowledge.ProjectFacts{SnapshotDigest: snapshotDigest(project), ProjectDigest: project.Digest, Digest: report.Digest}
	baseFacts.Facts = append(baseFacts.Facts, fileFacts(report, selection)...)
	baseFacts.Facts = append(baseFacts.Facts, historicalIdentityFacts(report, selection)...)
	selectedCore := map[string]bool{}
	coreKinds := map[string]string{}
	for _, d := range project.Model.Definitions {
		coreKinds[d.Identity().Key()] = d.Kind
	}
	for _, id := range scope.DefinitionIDs {
		selectedCore[id] = true
	}
	selectedFiles := map[string]bool{}
	for _, id := range scope.FileIDs {
		selectedFiles[id] = true
	}
	appendReportRelations(&baseFacts.Relations, report, selection, allowed, selectedFiles, definitions)
	appendReportOwnershipRelations(&baseFacts.Relations, report, allowed, definitions, selection, context)
	appendHistoryRelations(&baseFacts.Relations, report, allowed, definitions)
	extraDropped := false
	if extra != nil {
		visiblePaths := map[string]bool{}
		for _, path := range scope.Paths {
			visiblePaths[path] = true
		}
		baseFacts, extraDropped = mergeExtra(baseFacts, *extra, selectedCore, selectedFiles, coreKinds, visiblePaths, selection.ProjectScope)
	}

	selectedNodes := make([]projectknowledge.NodeSelection, 0, len(allowed)+len(scope.FileIDs)+len(baseFacts.Facts))
	for id := range allowed {
		d, ok := definitions[id]
		if !ok {
			continue
		}
		selectedNodes = append(selectedNodes, projectknowledge.NodeSelection{ID: id, Fields: nodeFields(d.Kind, id, selection, context), IncludePurpose: purposeAllowed(id, selection, context), IncludeSource: sourceAllowed(id, selection, context)})
	}
	for _, f := range baseFacts.Facts {
		fields := make([]string, 0, len(f.Properties))
		for name := range f.Properties {
			fields = append(fields, name)
		}
		sort.Strings(fields)
		selectedNodes = append(selectedNodes, projectknowledge.NodeSelection{ID: f.ID, Fields: fields})
	}
	sort.Slice(selectedNodes, func(i, j int) bool { return selectedNodes[i].ID < selectedNodes[j].ID })
	selectedID := map[string]bool{}
	for _, n := range selectedNodes {
		selectedID[n.ID] = true
	}
	edges := make([]projectknowledge.EdgeKey, 0)
	for _, e := range project.Model.Edges {
		if !selectedID[e.From] || !selectedID[e.To] || !edgeAllowed(e, selection, context, statementByID, artifactByID, checkByID, decisionByID, changeByID) {
			continue
		}
		edges = append(edges, projectknowledge.EdgeKey{From: e.From, To: e.To, Property: e.Property})
	}
	for _, relation := range baseFacts.Relations {
		if selectedID[relation.From] && selectedID[relation.To] {
			edges = append(edges, projectknowledge.EdgeKey{From: relation.From, To: relation.To, Property: relation.Property})
		}
	}
	// The index stores Core edge provenance. Foreign endpoint edges have their
	// source paths removed before indexing; selected node provenance stays opt-in.
	model := project.Model
	model.Edges = append([]core.Edge(nil), project.Model.Edges...)
	for i := range model.Edges {
		if !allowed[model.Edges[i].From] || !allowed[model.Edges[i].To] || !sourceAllowed(model.Edges[i].From, selection, context) {
			model.Edges[i].Source = core.Source{}
		}
	}
	idx, err := projectknowledge.Build(model, baseFacts, projectknowledge.Scope{ID: scope.ID, Nodes: selectedNodes, Edges: dedupeEdges(edges)})
	if err != nil {
		return nil, fmt.Errorf("build project graph: %w", err)
	}
	coverage := coverageFor(project, selection, context)
	var captureDigest string
	var bindings []RecordBinding
	if extra != nil && extra.EvidenceSelected {
		captureDigest = extra.CaptureDigest
		bindings = append([]RecordBinding(nil), extra.SourceBindings...)
		if extra.Completeness == projectknowledge.FactKnown || extra.Completeness == projectknowledge.FactPartial || extra.Completeness == projectknowledge.FactUnknown {
			coverage.Records = extra.Completeness
		}
		if extraDropped {
			coverage.Records = projectknowledge.FactPartial
		}
	}
	return &View{index: idx, selection: selection, scope: scope, revision: project.Revision, provisional: project.Provisional, projectDigest: project.Digest, coverage: coverage, captureDigest: captureDigest, sourceBindings: bindings}, nil
}

func selectContext(p *projectwork.Project, selection Selection) (projectmodel.ManagerContext, map[string]bool, error) {
	if (selection.ProjectScope && selection.ManagerID != "") || (!selection.ProjectScope && selection.ManagerID == "") {
		return projectmodel.ManagerContext{}, nil, ErrInvalidSelection
	}
	allowed := map[string]bool{}
	var ctx projectmodel.ManagerContext
	if selection.ProjectScope {
		for _, group := range [][]string{definitionIDsManagers(p.Report), definitionIDsStatements(p.Report), definitionIDsArtifacts(p.Report), definitionIDsChecks(p.Report), definitionIDsDecisions(p.Report), definitionIDsIdentityChanges(p.Report)} {
			for _, id := range group {
				allowed[id] = true
			}
		}
	} else {
		var err error
		ctx, err = projectmodel.Context(p.Report, selection.ManagerID)
		if err != nil {
			return ctx, nil, projectmodel.ErrManagerNotFound
		}
		allowed[ctx.Manager.ID] = true
		for _, x := range ctx.Statements {
			allowed[x.ID] = true
		}
		for _, x := range ctx.Contracts {
			allowed[x.ID] = true
			if x.Owner != "" {
				allowed[x.Owner] = true
			}
		}
		for _, x := range ctx.Artifacts {
			allowed[x.ID] = true
		}
		for _, x := range ctx.Checks {
			allowed[x.ID] = true
		}
		for _, x := range ctx.Decisions {
			allowed[x.ID] = true
		}
		for _, x := range ctx.IdentityChanges {
			allowed[x.ID] = true
		}
		for _, x := range ctx.Children {
			allowed[x.ID] = true
		}
	}
	return ctx, allowed, nil
}

func coreFields(kind string) []string {
	switch kind {
	case "Manager":
		return []string{"owns", "instructions"}
	case "Statement":
		return []string{"category", "description", "public"}
	case "Artifact":
		return []string{"role", "paths", "required", "reason"}
	case "Check":
		return []string{"command", "limitation"}
	case "Decision":
		return []string{"decision", "reason", "public", "supersedes"}
	case "IdentityChange":
		return []string{"operation", "previous", "reason", "public"}
	default:
		return nil
	}
}

func nodeFields(kind, id string, s Selection, ctx projectmodel.ManagerContext) []string {
	if kind == "Manager" && identityOnlyManager(id, s, ctx) {
		return nil
	}
	if kind == "Manager" && !s.ProjectScope && id != ctx.Manager.ID {
		return []string{"owns"}
	}
	return coreFields(kind)
}

func identityOnlyManager(id string, s Selection, ctx projectmodel.ManagerContext) bool {
	if s.ProjectScope || id == ctx.Manager.ID {
		return false
	}
	for _, child := range ctx.Children {
		if child.ID == id {
			return false
		}
	}
	for _, contract := range ctx.Contracts {
		if contract.Owner == id {
			return true
		}
	}
	return false
}

func fileID(path string) string {
	b, _ := json.Marshal([2]string{"project-file", path})
	return string(b)
}
func snapshotDigest(p *projectwork.Project) string {
	if p.Snapshot == nil {
		return "unavailable"
	}
	return p.Snapshot.Digest()
}

func fileFacts(report projectmodel.Report, selection Selection) []projectknowledge.Fact {
	var facts []projectknowledge.Fact
	for _, entry := range report.Files {
		if !selection.ProjectScope && entry.Owner != selection.ManagerID {
			continue
		}
		props := map[string]json.RawMessage{}
		put := func(k string, v any) { b, _ := json.Marshal(v); props[k] = b }
		put("path", entry.Path)
		put("digest", entry.Digest)
		put("mode", entry.Mode)
		put("present", entry.Exists)
		put("class", entry.Class)
		facts = append(facts, projectknowledge.Fact{ID: fileID(entry.Path), Kind: "ProjectFile", State: projectknowledge.FactKnown, Properties: props})
	}
	return facts
}

func historicalIdentityID(changeID string) string { return "history/previous/" + changeID }

func historicalIdentityFacts(report projectmodel.Report, selection Selection) []projectknowledge.Fact {
	var out []projectknowledge.Fact
	for _, change := range report.IdentityChanges {
		if !selection.ProjectScope && change.Owner != selection.ManagerID && !change.Public {
			continue
		}
		properties := map[string]json.RawMessage{}
		for key, value := range map[string]string{"apiVersion": change.Previous.APIVersion, "kind": change.Previous.Kind, "namespace": change.Previous.Namespace, "name": change.Previous.Name} {
			b, _ := json.Marshal(value)
			properties[key] = b
		}
		out = append(out, projectknowledge.Fact{ID: historicalIdentityID(change.ID), Kind: "HistoricalStatementIdentity", State: projectknowledge.FactKnown, Properties: properties})
	}
	return out
}

func appendReportRelations(dst *[]projectknowledge.Relation, r projectmodel.Report, selection Selection, nodes map[string]bool, files map[string]bool, definitions map[string]core.Definition) {
	for _, entry := range r.Files {
		fid := fileID(entry.Path)
		if !files[fid] || (!selection.ProjectScope && entry.Owner != selection.ManagerID) {
			continue
		}
		add := func(from, property, basis string) {
			if nodes[from] {
				d, ok := definitions[from]
				var source core.Source
				if ok {
					source = d.Source
				}
				*dst = append(*dst, projectknowledge.Relation{From: from, To: fid, Property: property, Source: source, Basis: basis})
			}
		}
		add(entry.Owner, "owns-file", "ProjectModel Report file ownership")
		for _, aid := range entry.Artifacts {
			if a, ok := findArtifact(r.Artifacts, aid); ok && a.Owner == entry.Owner {
				add(aid, "realizes-file", "ProjectModel Report exact artifact path match")
			}
		}
		for _, sid := range entry.Statements {
			if s, ok := findStatement(r.Statements, sid); ok && s.Owner == entry.Owner {
				if nodes[sid] {
					d := definitions[sid]
					*dst = append(*dst, projectknowledge.Relation{From: fid, To: sid, Property: "documents-statement", Source: d.Source, Basis: "ProjectModel Report exact source-file match"})
				}
			}
		}
		for _, cid := range entry.Checks {
			if c, ok := findCheck(r.Checks, cid); ok && c.Owner == entry.Owner {
				add(cid, "checks-file", "ProjectModel Report exact check-file match")
			}
		}
	}
}

func appendReportOwnershipRelations(dst *[]projectknowledge.Relation, r projectmodel.Report, nodes map[string]bool, definitions map[string]core.Definition, selection Selection, ctx projectmodel.ManagerContext) {
	add := func(id, owner string) {
		if id == "" || owner == "" || !nodes[id] || !nodes[owner] {
			return
		}
		d, ok := definitions[id]
		var source core.Source
		if ok && sourceAllowed(id, selection, ctx) {
			source = d.Source
		}
		*dst = append(*dst, projectknowledge.Relation{From: id, To: owner, Property: "owned-by", Source: source, Basis: "ProjectModel Report nearest Manager ownership"})
	}
	for _, x := range r.Statements {
		add(x.ID, x.Owner)
	}
	for _, x := range r.Artifacts {
		add(x.ID, x.Owner)
	}
	for _, x := range r.Checks {
		add(x.ID, x.Owner)
	}
	for _, x := range r.Decisions {
		add(x.ID, x.Owner)
	}
	for _, x := range r.IdentityChanges {
		add(x.ID, x.Owner)
	}
}

func appendHistoryRelations(dst *[]projectknowledge.Relation, r projectmodel.Report, nodes map[string]bool, definitions map[string]core.Definition) {
	for _, change := range r.IdentityChanges {
		id := historicalIdentityID(change.ID)
		if !nodes[change.ID] {
			continue
		}
		d := definitions[change.ID]
		*dst = append(*dst, projectknowledge.Relation{From: change.ID, To: id, Property: "declares-previous-identity", Source: d.Source, Basis: "declared structured historical identity"})
	}
}

func findArtifact(values []projectmodel.Artifact, id string) (projectmodel.Artifact, bool) {
	for _, v := range values {
		if v.ID == id {
			return v, true
		}
	}
	return projectmodel.Artifact{}, false
}
func findStatement(values []projectmodel.Statement, id string) (projectmodel.Statement, bool) {
	for _, v := range values {
		if v.ID == id {
			return v, true
		}
	}
	return projectmodel.Statement{}, false
}
func findCheck(values []projectmodel.Check, id string) (projectmodel.Check, bool) {
	for _, v := range values {
		if v.ID == id {
			return v, true
		}
	}
	return projectmodel.Check{}, false
}

func mergeExtra(dst projectknowledge.ProjectFacts, extra ExtraFacts, coreIDs, fileIDs map[string]bool, coreKinds map[string]string, visiblePaths map[string]bool, projectScope bool) (projectknowledge.ProjectFacts, bool) {
	visibleKinds := map[string]string{}
	for id := range coreIDs {
		if kind := coreKinds[id]; kind != "" {
			visibleKinds[id] = kind
		}
	}
	selected := map[string]bool{}
	for _, id := range extra.SelectedIDs {
		selected[id] = true
	}
	factIDs := map[string]bool{}
	dropped := false
	for _, f := range extra.Facts.Facts {
		if selected[f.ID] && allowedEvidenceKinds[f.Kind] {
			f.Source = core.Source{}
			var omitted bool
			f.Properties, omitted = safeExtraProperties(f, visibleKinds, visiblePaths, selected, projectScope)
			if omitted {
				f.State = projectknowledge.FactPartial
				dropped = true
			}
			dst.Facts = append(dst.Facts, f)
			factIDs[f.ID] = true
		} else if selected[f.ID] {
			dropped = true
		}
	}
	for _, r := range extra.Facts.Relations {
		if allowedEvidenceRelations[r.Property] && (coreIDs[r.From] || fileIDs[r.From] || factIDs[r.From]) && (coreIDs[r.To] || fileIDs[r.To] || factIDs[r.To]) {
			r.Source = core.Source{}
			dst.Relations = append(dst.Relations, r)
		} else if selected[r.From] || selected[r.To] {
			dropped = true
		}
	}
	// Bind the exact external facts as part of this index input.
	b, _ := json.Marshal(struct{ Base, Extra string }{dst.Digest, extra.Facts.Digest})
	sum := sha256.Sum256(b)
	dst.Digest = "sha256:" + hex.EncodeToString(sum[:])
	return dst, dropped
}

func safeExtraProperties(f projectknowledge.Fact, coreKinds map[string]string, visiblePaths, selectedFacts map[string]bool, projectScope bool) (map[string]json.RawMessage, bool) {
	allowed := allowedEvidenceProperties[f.Kind]
	out := map[string]json.RawMessage{}
	dropped := false
	for k, v := range f.Properties {
		if !allowed[k] {
			dropped = true
			continue
		}
		var value any
		dec := json.NewDecoder(bytes.NewReader(v))
		dec.UseNumber()
		if dec.Decode(&value) != nil {
			dropped = true
			continue
		}
		filtered, keep, changed := filterEvidenceValue(f.Kind, k, value, coreKinds, visiblePaths, selectedFacts, projectScope)
		if changed {
			dropped = true
		}
		if keep {
			encoded, err := json.Marshal(filtered)
			if err == nil {
				out[k] = encoded
			} else {
				dropped = true
			}
		}
	}
	return out, dropped
}

func filterEvidenceValue(kind, key string, value any, coreKinds map[string]string, visiblePaths, selectedFacts map[string]bool, projectScope bool) (any, bool, bool) {
	keepID := func(value any, kind string) bool {
		s, ok := value.(string)
		return ok && (kind == "" && coreKinds[s] != "" || coreKinds[s] == kind)
	}
	keepPath := func(value any) bool { s, ok := value.(string); return ok && visiblePaths[s] }
	switch key {
	case "id":
		if kind == "CheckExecution" {
			return value, keepID(value, "Check"), !keepID(value, "Check")
		}
		return value, true, false
	case "path":
		return value, keepPath(value), !keepPath(value)
	case "managerId", "owner":
		return value, keepID(value, "Manager"), !keepID(value, "Manager")
	case "managerIds":
		return filterIDArray(value, "Manager", coreKinds)
	case "expectedArtifacts":
		return filterIDArray(value, "Artifact", coreKinds)
	case "expectedChecks":
		return filterIDArray(value, "Check", coreKinds)
	case "expectedStatements":
		return filterIDArray(value, "Statement", coreKinds)
	case "definitionId":
		return value, keepID(value, ""), !keepID(value, "")
	case "visiblePaths":
		return filterPathArray(value, visiblePaths)
	case "writtenPaths":
		return filterPathArray(value, visiblePaths)
	case "eventIds", "claimIds":
		return filterSelectedLeaves(value, selectedFacts)
	case "coveredManagerIds":
		return filterIDArray(value, "Manager", coreKinds)
	case "runId", "planId":
		return value, selectedLeaf(fmt.Sprint(value), selectedFacts), !selectedLeaf(fmt.Sprint(value), selectedFacts)
	case "decisionReference":
		return value, keepID(value, "") || selectedLeaf(fmt.Sprint(value), selectedFacts), !(keepID(value, "") || selectedLeaf(fmt.Sprint(value), selectedFacts))
	case "candidateId", "verificationId":
		if projectScope {
			return value, true, false
		}
		return nil, false, true
	case "actor":
		return nil, false, true
	}
	return value, true, false
}

func filterIDArray(value any, kind string, coreKinds map[string]string) (any, bool, bool) {
	values, ok := value.([]any)
	if !ok {
		return nil, false, true
	}
	out := make([]any, 0, len(values))
	changed := false
	for _, v := range values {
		s, ok := v.(string)
		if ok && coreKinds[s] == kind {
			out = append(out, v)
		} else {
			changed = true
		}
	}
	return out, true, changed
}
func filterPathArray(value any, visiblePaths map[string]bool) (any, bool, bool) {
	values, ok := value.([]any)
	if !ok {
		return nil, false, true
	}
	out := make([]any, 0, len(values))
	changed := false
	for _, v := range values {
		s, ok := v.(string)
		if ok && visiblePaths[s] {
			out = append(out, v)
		} else {
			changed = true
		}
	}
	return out, true, changed
}
func filterSelectedLeaves(value any, selected map[string]bool) (any, bool, bool) {
	values, ok := value.([]any)
	if !ok {
		return nil, false, true
	}
	out := make([]any, 0, len(values))
	changed := false
	for _, v := range values {
		s, ok := v.(string)
		if ok && selectedLeaf(s, selected) {
			out = append(out, v)
		} else {
			changed = true
		}
	}
	return out, true, changed
}
func selectedLeaf(value string, selected map[string]bool) bool {
	if selected[value] {
		return true
	}
	for id := range selected {
		if strings.HasSuffix(id, "/"+value) {
			return true
		}
	}
	return false
}

var allowedEvidenceKinds = map[string]bool{
	"ExecutionPlan": true, "ExecutionRun": true, "ManagerTask": true, "CandidatePath": true, "Candidate": true, "CheckExecution": true,
	"CandidateReview": true, "CandidateVerification": true,
	"Exploration": true, "WorkScope": true, "WorkDecision": true, "StructureAcknowledgement": true, "ApplyReceipt": true,
	"BrownfieldSession": true, "BrownfieldScope": true, "BrownfieldIteration": true, "BrownfieldClaim": true, "BrownfieldEvidenceRef": true,
	"BrownfieldContradiction": true, "BrownfieldQuestion": true, "BrownfieldResolution": true, "Briefing": true, "ModelHistoryEvent": true, "BriefingResolution": true,
}

var allowedEvidenceRelations = map[string]bool{"usesPlan": true, "hasInitialCandidate": true, "expectsRun": true, "plansTask": true, "hasTask": true, "wrotePath": true, "producedCandidate": true, "containsPath": true, "hasCheckExecution": true, "expectsCheck": true, "checkedCandidate": true, "hasReview": true, "reviewedBy": true, "reviewedCandidate": true, "hasVerification": true, "containsScope": true, "hasDecision": true, "hasAcknowledgement": true, "completedBy": true, "hasApplyReceipt": true, "appliedBy": true, "hasIteration": true, "proposedClaim": true, "citesEvidence": true, "recordsContradiction": true, "recordsQuestion": true, "hasResolution": true, "referencesRun": true, "changedDefinition": true, "containsEvent": true}

var allowedEvidenceProperties = map[string]map[string]bool{
	"ExecutionPlan":            set("digest", "schema", "modelDigest", "sourceSnapshot", "baseRevision", "status", "modelBindingStatus", "runtimeBindingStatus"),
	"ExecutionRun":             set("digest", "schema", "planId", "status", "candidateDigest", "bindingStatus", "modelBindingStatus", "runtimeDigest", "runtimeBindingStatus", "candidateBindingPresent", "candidateId", "captureState"),
	"ManagerTask":              set("managerId", "state", "reportStatus", "present", "expectedArtifacts", "expectedChecks", "expectedStatements"),
	"CandidatePath":            set("path", "digest"),
	"Candidate":                set("digest", "schema", "integrated", "visiblePaths", "id", "bindingStatus"),
	"CheckExecution":           set("id", "outcome", "exitCode", "expected", "checked", "checkPassed", "verificationStatus", "candidateId", "required", "owner"),
	"CandidateReview":          set("managerId", "phase", "round", "outcome", "candidateDigest", "scopeDigest", "inputDigest", "candidateBindingStatus", "candidateId"),
	"CandidateVerification":    set("verificationStatus", "candidateDigest", "schema", "digest", "verified"),
	"Exploration":              set("digest", "status", "createdAgainstBindingDigest", "bindingStatus"),
	"WorkScope":                set("id", "operation", "managerIds"),
	"WorkDecision":             set("status", "blocking", "answer", "reason", "callerAccepted", "authenticated"),
	"StructureAcknowledgement": set("bindingDigest", "structureDigest", "actor", "authority", "provenance", "authenticated", "callerAccepted"),
	"ApplyReceipt":             set("schema", "contentDigest", "digestKind", "recordedStatus", "planId", "planDigest", "candidateDigest", "verificationDigest", "applyId", "applyDigest", "appliedAt", "applied", "appliedStatus", "authenticated", "candidateId", "writtenPaths"),
	"BrownfieldSession":        set("digest", "schema", "targetRevision", "targetProjectDigest", "targetModelDigest", "targetBindingStatus", "sourceBindingStatus", "sourceCommit", "sourceDigest"),
	"BrownfieldScope":          set("status", "reason"),
	"BrownfieldIteration":      set("managerId", "purposeRecorded", "reviewRecorded", "targetContextDigest", "proposalPresent", "integrationPresent", "resolutionPresent"),
	"BrownfieldClaim":          set("kind", "method", "statement", "uncertainty", "scopeId"),
	"BrownfieldEvidenceRef":    set("evidenceId", "startLine", "endLine", "path"),
	"BrownfieldContradiction":  set("scopeId", "description", "claimIds", "questionId"),
	"BrownfieldQuestion":       set("scopeId", "blocking", "alternativeCount", "claimIds"),
	"BrownfieldResolution":     set("digest", "actor", "authorityClaim", "callerAccepted", "authenticated", "decisionReferencePresent"),
	"Briefing":                 set("revision", "modelDigest", "eventIds", "summary"),
	"ModelHistoryEvent":        set("digest", "change", "category", "severity", "definitionId", "decisionReference", "actor", "authority", "authenticated", "resolutionStatus"),
	"BriefingResolution":       set("status", "digest", "modelRevision", "modelDigest", "runId", "planDigest", "candidateDigest", "verificationDigest", "applyDigest", "recordedFullVerifyPassed", "authenticated", "candidateId", "coveredManagerIds"),
}

func set(fields ...string) map[string]bool {
	out := map[string]bool{}
	for _, f := range fields {
		out[f] = true
	}
	return out
}

func purposeAllowed(id string, s Selection, ctx projectmodel.ManagerContext) bool {
	if s.ProjectScope {
		return true
	}
	if id == ctx.Manager.ID {
		return true
	}
	for _, v := range ctx.Statements {
		if v.ID == id {
			return true
		}
	}
	for _, v := range ctx.Artifacts {
		if v.ID == id {
			return true
		}
	}
	for _, v := range ctx.Checks {
		if v.ID == id {
			return true
		}
	}
	for _, v := range ctx.Decisions {
		if v.ID == id {
			return true
		}
	}
	for _, v := range ctx.IdentityChanges {
		if v.ID == id {
			return true
		}
	}
	for _, v := range ctx.Contracts {
		if v.ID == id {
			return true
		}
	}
	for _, v := range ctx.Children {
		if v.ID == id {
			return true
		}
	}
	return false
}
func sourceAllowed(id string, s Selection, ctx projectmodel.ManagerContext) bool {
	if s.ProjectScope || id == ctx.Manager.ID {
		return true
	}
	for _, values := range [][]string{
		definitionIDsFromStatements(ctx.Statements),
		definitionIDsFromArtifacts(ctx.Artifacts),
		definitionIDsFromChecks(ctx.Checks),
		definitionIDsFromDecisions(ctx.Decisions),
		definitionIDsFromIdentityChanges(ctx.IdentityChanges),
	} {
		for _, value := range values {
			if value == id {
				return true
			}
		}
	}
	return false
}

func definitionIDsFromStatements(values []projectmodel.Statement) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}
func definitionIDsFromArtifacts(values []projectmodel.Artifact) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}
func definitionIDsFromChecks(values []projectmodel.Check) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}
func definitionIDsFromDecisions(values []projectmodel.Decision) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}
func definitionIDsFromIdentityChanges(values []projectmodel.IdentityChange) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids
}
func coverageFor(p *projectwork.Project, s Selection, ctx projectmodel.ManagerContext) Coverage {
	c := Coverage{Model: projectknowledge.FactKnown, Inventory: projectknowledge.FactKnown, Artifacts: projectknowledge.FactKnown, Checks: projectknowledge.FactKnown, Records: projectknowledge.FactUnknown}
	if s.ProjectScope && len(p.Report.Unknown) > 0 {
		c.Inventory = projectknowledge.FactPartial
	}
	findings := p.Report.Findings
	if !s.ProjectScope {
		findings = ctx.Findings
	}
	if p.Model.Digest == "" || p.Report.ModelDigest == "" {
		c.Model = projectknowledge.FactPartial
	} else {
		for _, f := range findings {
			if f.Severity == "error" || f.Severity == "incomplete" {
				c.Model = projectknowledge.FactPartial
				break
			}
		}
	}
	artifacts := p.Report.Artifacts
	checks := p.Report.Checks
	if !s.ProjectScope {
		artifacts = ctx.Artifacts
		checks = ctx.Checks
	}
	for _, a := range artifacts {
		if a.Required && len(a.Paths) == 0 {
			c.Artifacts = projectknowledge.FactPartial
		}
		for _, f := range findings {
			if f.Subject == a.ID && f.Code == "coverage.required-artifact-missing" {
				c.Artifacts = projectknowledge.FactPartial
			}
		}
	}
	if len(checks) == 0 {
		c.Checks = projectknowledge.FactUnknown
	}
	return c
}

func dedupeEdges(in []projectknowledge.EdgeKey) []projectknowledge.EdgeKey {
	seen := map[projectknowledge.EdgeKey]bool{}
	out := make([]projectknowledge.EdgeKey, 0, len(in))
	for _, e := range in {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].Property != out[j].Property {
			return out[i].Property < out[j].Property
		}
		return out[i].To < out[j].To
	})
	return out
}

func edgeAllowed(e core.Edge, selection Selection, ctx projectmodel.ManagerContext, statements map[string]projectmodel.Statement, artifacts map[string]projectmodel.Artifact, checks map[string]projectmodel.Check, decisions map[string]projectmodel.Decision, changes map[string]projectmodel.IdentityChange) bool {
	if selection.ProjectScope {
		return true
	}
	switch e.Property {
	case "parent":
		return e.To == ctx.Manager.ID
	case "uses", "requires":
		s, own := statements[e.From]
		if !own || s.Owner != selection.ManagerID {
			return false
		}
		_, allowed := statements[e.To]
		return allowed
	case "realizes":
		a, own := artifacts[e.From]
		if !own || a.Owner != selection.ManagerID {
			return false
		}
		_, allowed := statements[e.To]
		return allowed
	case "checks":
		a, own := artifacts[e.From]
		if !own || a.Owner != selection.ManagerID {
			return false
		}
		_, allowed := checks[e.To]
		return allowed
	case "subject":
		if d, own := decisions[e.From]; own {
			return (d.Owner == selection.ManagerID || d.Public) && statements[e.To].ID != ""
		}
		if c, own := changes[e.From]; own {
			return (c.Owner == selection.ManagerID || c.Public) && statements[e.To].ID != ""
		}
		return false
	case "actor", "actorManager":
		d, own := decisions[e.From]
		if own {
			return d.Owner == selection.ManagerID && e.To == selection.ManagerID
		}
		c, own := changes[e.From]
		return own && c.Owner == selection.ManagerID && e.To == selection.ManagerID
	case "supersedes":
		d, own := decisions[e.From]
		if !own || d.Owner != selection.ManagerID {
			return false
		}
		for _, x := range ctx.Decisions {
			if x.ID == e.To {
				return true
			}
		}
		return false
	default:
		_, d := decisions[e.From]
		_, c := changes[e.From]
		return d || c
	}
}

func definitionIDsManagers(r projectmodel.Report) []string {
	out := make([]string, 0, len(r.Managers))
	for _, x := range r.Managers {
		out = append(out, x.ID)
	}
	return out
}
func definitionIDsStatements(r projectmodel.Report) []string {
	out := make([]string, 0, len(r.Statements))
	for _, x := range r.Statements {
		out = append(out, x.ID)
	}
	return out
}
func definitionIDsArtifacts(r projectmodel.Report) []string {
	out := make([]string, 0, len(r.Artifacts))
	for _, x := range r.Artifacts {
		out = append(out, x.ID)
	}
	return out
}
func definitionIDsChecks(r projectmodel.Report) []string {
	out := make([]string, 0, len(r.Checks))
	for _, x := range r.Checks {
		out = append(out, x.ID)
	}
	return out
}
func definitionIDsDecisions(r projectmodel.Report) []string {
	out := make([]string, 0, len(r.Decisions))
	for _, x := range r.Decisions {
		out = append(out, x.ID)
	}
	return out
}
func definitionIDsIdentityChanges(r projectmodel.Report) []string {
	out := make([]string, 0, len(r.IdentityChanges))
	for _, x := range r.IdentityChanges {
		out = append(out, x.ID)
	}
	return out
}
