package apply

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// ContextSelectionDiagnostic reports a static selection boundary. Package is
// empty for a parent-stage problem. Organization errors do not supply source
// coordinates; original source and decode diagnostics remain in Execution().
type ContextSelectionDiagnostic struct {
	Code    string
	Package string
	Message string
}

type contextSelections struct {
	status      model.Status
	selections  []organization.ContextSelection
	diagnostics []ContextSelectionDiagnostic
}

// Passed means every local Manager's static model selection succeeded. It
// establishes no rendered context, size-cap check, resolved settings or runtime
// readiness. The zero result is NotCompleted; refusals expose no partial set.
func (r contextSelections) Status() model.Status {
	if r.status == "" {
		return model.NotCompleted
	}
	return r.status
}

// Selections returns the complete local set, Root first then exact Package
// order. ContextSelection has immutable internals and copying accessors.
func (r contextSelections) Selections() ([]organization.ContextSelection, bool) {
	if r.Status() != model.Passed {
		return nil, false
	}
	return slices.Clone(r.selections), true
}

func (r contextSelections) Diagnostics() []ContextSelectionDiagnostic {
	return slices.Clone(r.diagnostics)
}

// CapturedContextSelections retains its exact decoded committed-source parent,
// including the acquisition guard. It cannot be built from free model values.
type CapturedContextSelections struct {
	contextSelections
	stage CapturedExecutionDecoding
}

func (r CapturedContextSelections) Execution() CapturedExecutionDecoding { return r.stage }

// PreviewContextSelections retains a working or fixed preview, without a guard.
type PreviewContextSelections struct {
	contextSelections
	stage PreviewExecutionDecoding
}

func (r PreviewContextSelections) Execution() PreviewExecutionDecoding { return r.stage }

func (e CapturedExecutionDecoding) SelectContexts() CapturedContextSelections {
	return CapturedContextSelections{contextSelections: selectContexts(e.executionDecoding, e.stage.compiled), stage: e}
}

func (e PreviewExecutionDecoding) SelectContexts() PreviewContextSelections {
	return PreviewContextSelections{contextSelections: selectContexts(e.executionDecoding, e.stage.compiled), stage: e}
}

func selectContexts(execution executionDecoding, compiled model.Result) contextSelections {
	r := contextSelections{status: execution.Status()}
	if r.status != model.Passed {
		r.diagnostics = []ContextSelectionDiagnostic{{Code: "context.execution_not_passed", Message: "Static context selection requires a passed bound execution decode."}}
		return r
	}
	topology, err := organization.BuildTopology(compiled)
	if err != nil {
		r.status = model.Failed
		r.diagnostics = []ContextSelectionDiagnostic{{Code: "context.topology", Message: err.Error()}}
		return r
	}
	var scope []organization.Setting[organization.ScopeRule]
	if execution.authored.Scope != nil {
		scope = execution.authored.Scope.Value
	}
	var selections []organization.ContextSelection
	for _, manager := range topology.Managers() {
		selection, err := organization.SelectContext(compiled, manager.Package, scope)
		if err != nil {
			r.status = model.Failed
			r.diagnostics = append(r.diagnostics, ContextSelectionDiagnostic{Code: "context.selection", Package: manager.Package, Message: err.Error()})
			continue
		}
		selections = append(selections, selection)
	}
	if r.status == model.Passed {
		r.selections = selections
	}
	return r
}
