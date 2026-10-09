package projectapp

import (
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/host/knowledgeevidence"
	"github.com/Glacius-Labs/Markitect/internal/host/projectgraph"
)

// KnowledgeOperation selects one source snapshot, one explicit visibility scope
// and optional operational records. Record visibility is derived by the facade;
// callers cannot supply a list of purportedly visible definitions or files.
type KnowledgeOperation struct {
	Selection Selection                   `json:"selection"`
	Scope     projectgraph.Selection      `json:"scope"`
	Query     projectgraph.Request        `json:"query"`
	Records   knowledgeevidence.Selection `json:"records"`
}

type EvidenceSummary struct {
	CaptureDigest  string                            `json:"captureDigest"`
	Completeness   string                            `json:"completeness"`
	Unknown        []knowledgeevidence.UnknownRecord `json:"unknown,omitempty"`
	SourceBindings []knowledgeevidence.SourceBinding `json:"sourceBindings,omitempty"`
}

type KnowledgeResult struct {
	Graph    projectgraph.Result `json:"graph"`
	Evidence EvidenceSummary     `json:"evidence"`
}

// Knowledge is the common read-only application boundary for CLI and MCP.
// It uses no Invoker and persists no graph or evidence ledger.
func (o Operations) Knowledge(operation KnowledgeOperation) (KnowledgeResult, error) {
	if err := requireRoot(operation.Selection.Root); err != nil {
		return KnowledgeResult{}, err
	}
	if o.Host.Load == nil {
		return KnowledgeResult{}, errors.New("project loading port is required")
	}
	project, err := o.Host.Load(operation.Selection.Root, operation.Selection.Revision)
	if err != nil {
		return KnowledgeResult{}, err
	}
	view, err := projectgraph.Build(project, operation.Scope, nil)
	if err != nil {
		return KnowledgeResult{}, err
	}
	scope := view.VisibleScope()
	capture, err := knowledgeevidence.Read(project, knowledgeevidence.Scope{
		ManagerID: operation.Scope.ManagerID, WholeProject: operation.Scope.ProjectScope,
		VisibleDefinitionIDs: scope.DefinitionIDs, VisiblePaths: scope.Paths,
	}, operation.Records)
	if err != nil {
		return KnowledgeResult{}, err
	}
	selected := make([]string, 0, len(capture.Facts.Facts))
	for _, fact := range capture.Facts.Facts {
		selected = append(selected, fact.ID)
	}
	bindings := make([]projectgraph.RecordBinding, 0, len(capture.SourceBindings))
	for _, binding := range capture.SourceBindings {
		bindings = append(bindings, projectgraph.RecordBinding{
			Kind: binding.Kind, Digest: binding.Digest, Schema: binding.Schema, Source: binding.Source,
			ModelDigest: binding.ModelDigest, Revision: binding.Revision, RuntimeDigest: binding.RuntimeDigest,
		})
	}
	evidenceSelected := len(operation.Records.RunIDs)+len(operation.Records.ExplorationIDs)+len(operation.Records.BrownfieldSessionIDs) > 0 || operation.Records.IncludeBriefingHistory
	view, err = projectgraph.Build(project, operation.Scope, &projectgraph.ExtraFacts{
		ScopeID: scope.ID, SelectedIDs: selected, Facts: capture.Facts,
		CaptureDigest: capture.CaptureDigest, Completeness: capture.Completeness,
		EvidenceSelected: evidenceSelected, SourceBindings: bindings,
	})
	if err != nil {
		return KnowledgeResult{}, err
	}
	result, err := projectgraph.Query(view, operation.Query)
	if err != nil {
		return KnowledgeResult{}, err
	}
	return KnowledgeResult{Graph: result, Evidence: EvidenceSummary{
		CaptureDigest: capture.CaptureDigest, Completeness: capture.Completeness,
		Unknown: capture.Unknown, SourceBindings: capture.SourceBindings,
	}}, nil
}
