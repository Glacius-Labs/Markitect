package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

// CanonicalImpactCause is a declared input or resolved reference, never an
// inferred source-code dependency. Via is present only for an explicit edge.
type CanonicalImpactCause struct {
	Code    string     `json:"code"`
	Subject string     `json:"subject"`
	Via     *core.Edge `json:"via,omitempty"`
}

type CanonicalProjectionImpact struct {
	ProjectionID      string                 `json:"projectionId"`
	Causes            []CanonicalImpactCause `json:"causes"`
	RecordedArtifacts []string               `json:"recordedArtifacts"`
}

type CanonicalImpact struct {
	BaseRevision                       string                          `json:"baseRevision"`
	CandidateRevision                  string                          `json:"candidateRevision"`
	ChangedDefinitions                 []string                        `json:"changedDefinitions"`
	AffectedDefinitions                []string                        `json:"affectedDefinitions"`
	DefinitionCauses                   map[string]CanonicalImpactCause `json:"definitionCauses"`
	ScopeAffectedProjections           []CanonicalProjectionImpact     `json:"scopeAffectedProjections"`
	ConservativeInvalidatedProjections []string                        `json:"conservativeInvalidatedProjections"`
	ConservativeCause                  string                          `json:"conservativeCause,omitempty"`
}

// AnalyzeCanonicalImpact compares structurally compiled input sets. Host gives
// resolved references reverse-invalidation meaning here; this never changes
// agent context or adds a Core relation/query primitive. Whole-model/revision
// request bindings remain conservative and are reported separately.
func AnalyzeCanonicalImpact(base, candidate *CanonicalSource, activeRecords []records.ProjectionRecord) (CanonicalImpact, error) {
	report := CanonicalImpact{DefinitionCauses: map[string]CanonicalImpactCause{}}
	if base == nil || candidate == nil || len(base.Diagnostics) != 0 || len(candidate.Diagnostics) != 0 {
		return report, errors.New("canonical impact requires two structurally valid compiled sources")
	}
	for _, source := range []*CanonicalSource{base, candidate} {
		compiled, diagnostics := core.Compile(source.Model.Schemas, source.Model.Definitions, source.Model.Revision)
		if len(diagnostics) != 0 || compiled.Digest != source.Model.Digest || !reflect.DeepEqual(compiled.Edges, source.Model.Edges) {
			return report, errors.New("canonical impact model does not match its structural compilation")
		}
	}
	report.BaseRevision, report.CandidateRevision = base.Model.Revision, candidate.Model.Revision
	oldDefs, newDefs := definitionIndex(base.Model), definitionIndex(candidate.Model)
	oldSchemas, newSchemas := schemaIndex(base.Model), schemaIndex(candidate.Model)
	all := map[string]core.Definition{}
	for key, d := range oldDefs {
		all[key] = d
	}
	for key, d := range newDefs {
		all[key] = d
	}
	keys := make([]string, 0, len(all))
	for key := range all {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	queue := []string{}
	for _, key := range keys {
		old, oldOK := oldDefs[key]
		next, nextOK := newDefs[key]
		code := ""
		switch {
		case !oldOK:
			code = "definition.added"
		case !nextOK:
			code = "definition.removed"
		case !equalCanonicalValue(old, next):
			code = "definition.bytes-or-meaning-changed"
		}
		if code != "" {
			report.ChangedDefinitions = append(report.ChangedDefinitions, key)
		}
		if code == "" && !equalCanonicalValue(oldSchemas[all[key].APIVersion], newSchemas[all[key].APIVersion]) {
			code = "schema-contract-or-source-changed"
		}
		if code != "" {
			report.DefinitionCauses[key] = CanonicalImpactCause{Code: code, Subject: key}
			queue = append(queue, key)
		}
	}
	edges := append(append([]core.Edge(nil), base.Model.Edges...), candidate.Model.Edges...)
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.To != b.To {
			return a.To < b.To
		}
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Property != b.Property {
			return a.Property < b.Property
		}
		if a.Source.Path != b.Source.Path {
			return a.Source.Path < b.Source.Path
		}
		if a.Source.Line != b.Source.Line {
			return a.Source.Line < b.Source.Line
		}
		return a.Source.Digest < b.Source.Digest
	})
	incoming := map[string][]core.Edge{}
	for _, edge := range edges {
		incoming[edge.To] = append(incoming[edge.To], edge)
	}
	for index := 0; index < len(queue); index++ {
		for _, edge := range incoming[queue[index]] {
			if _, seen := report.DefinitionCauses[edge.From]; seen {
				continue
			}
			edgeCopy := edge
			report.DefinitionCauses[edge.From] = CanonicalImpactCause{Code: "referenced-definition-changed", Subject: edge.To, Via: &edgeCopy}
			queue = append(queue, edge.From)
		}
	}
	for key := range report.DefinitionCauses {
		report.AffectedDefinitions = append(report.AffectedDefinitions, key)
	}
	sort.Strings(report.AffectedDefinitions)
	oldRequests, err := projectionRequestIndex(base)
	if err != nil {
		return report, err
	}
	newRequests, err := projectionRequestIndex(candidate)
	if err != nil {
		return report, err
	}
	projectionKeys := map[string]bool{}
	for key := range oldRequests {
		projectionKeys[key] = true
	}
	for key := range newRequests {
		projectionKeys[key] = true
	}
	sortedProjections := []string{}
	for key := range projectionKeys {
		sortedProjections = append(sortedProjections, key)
	}
	sort.Strings(sortedProjections)
	pathsByProjection := map[string][]string{}
	for _, record := range activeRecords {
		if err := records.ValidateProjectionRecord(record); err != nil {
			return report, fmt.Errorf("invalid supplied active ProjectionRecord: %w", err)
		}
		for _, artifact := range record.Artifacts {
			pathsByProjection[record.ProjectionID] = append(pathsByProjection[record.ProjectionID], artifact.Path)
		}
	}
	for _, key := range sortedProjections {
		causes := []CanonicalImpactCause{}
		seen := map[string]bool{}
		add := func(cause CanonicalImpactCause) {
			token := cause.Code + "\x00" + cause.Subject
			if !seen[token] {
				seen[token] = true
				causes = append(causes, cause)
			}
		}
		if cause, ok := report.DefinitionCauses[key]; ok {
			add(cause)
		}
		for _, request := range []canonical.ProjectionRequest{oldRequests[key], newRequests[key]} {
			for _, d := range append(append([]core.Definition(nil), request.Definitions...), request.Policies...) {
				if cause, ok := report.DefinitionCauses[d.Identity().Key()]; ok {
					add(cause)
				}
			}
		}
		old, oldOK := oldRequests[key]
		next, nextOK := newRequests[key]
		if oldOK && nextOK && !equalCanonicalValue(old.ModulePin, next.ModulePin) {
			add(CanonicalImpactCause{Code: "projection-module-pin-changed", Subject: key})
		}
		if len(causes) == 0 {
			continue
		}
		sort.Slice(causes, func(i, j int) bool {
			if causes[i].Subject != causes[j].Subject {
				return causes[i].Subject < causes[j].Subject
			}
			return causes[i].Code < causes[j].Code
		})
		paths := pathsByProjection[key]
		report.ScopeAffectedProjections = append(report.ScopeAffectedProjections, CanonicalProjectionImpact{ProjectionID: key, Causes: causes, RecordedArtifacts: sortedUniquePaths(paths)})
	}
	if base.Model.Digest != candidate.Model.Digest || base.Model.Revision != candidate.Model.Revision {
		report.ConservativeInvalidatedProjections = sortedProjections
		report.ConservativeCause = "all requests bind the complete model digest and fixed revision; a local semantic review set does not make old plans or records fresh"
	}
	return report, nil
}
func definitionIndex(model core.Model) map[string]core.Definition {
	out := map[string]core.Definition{}
	for _, d := range model.Definitions {
		out[d.Identity().Key()] = d
	}
	return out
}
func schemaIndex(model core.Model) map[string]core.Schema {
	out := map[string]core.Schema{}
	for _, s := range model.Schemas {
		out[s.APIVersion] = s
	}
	return out
}
func equalCanonicalValue(a, b any) bool {
	left, e1 := json.Marshal(a)
	right, e2 := json.Marshal(b)
	return e1 == nil && e2 == nil && string(left) == string(right)
}
func projectionRequestIndex(source *CanonicalSource) (map[string]canonical.ProjectionRequest, error) {
	out := map[string]canonical.ProjectionRequest{}
	for _, d := range source.Model.Definitions {
		if d.APIVersion != "markitect.foundation/v1" || d.Kind != "Projection" {
			continue
		}
		request, err := canonical.BindProjection(source.Model, source.Activation, source.Config.ProjectionBindings, d.Identity(), nil)
		if err != nil {
			return nil, err
		}
		out[d.Identity().Key()] = request
	}
	return out, nil
}
