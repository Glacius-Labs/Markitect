package apply

import (
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ProposalCheck is a proposed stand of a captured repository prepared the way
// the structural check prepares the repository itself: Model check, registry,
// File check, execution settings and context selection of the proposed tree.
// Status is the structural status; Findings lists every diagnostic of the
// stages with its stage, code, location and message.
type ProposalCheck struct {
	Status   model.Status
	Findings []ProposalFinding
	attempt  CapturedPreparationAttempt
}

// ProposalFinding is one diagnostic of a proposed stand.
type ProposalFinding struct {
	Stage   string `json:"stage"`
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// Attempt is the structural preparation of the proposed stand.
func (c ProposalCheck) Attempt() CapturedPreparationAttempt { return c.attempt }

// CheckProposal prepares a proposed tree against a captured start. The inputs
// outside the repository, such as dependency models, stay as captured. Extract
// rebinds its model-phase candidates exactly so; nothing else of Extract
// applies.
func CheckProposal(start CapturedPreparationAttempt, tree snapshot.Tree) ProposalCheck {
	a := rebindStructural(start.Model(), tree, extractModel)
	c := ProposalCheck{attempt: a, Status: structuralPreparationStatus(a.stage.checked.Status, a.registry.Status(), a.files.Status(), a.execution.Status(), a.contexts.Status())}
	// A later stage can repeat an earlier stage's diagnostic; it is reported
	// once, under the stage that found it first.
	seen := map[[3]string]bool{}
	add := func(stage, code, path, message string) {
		key := [3]string{code, path, message}
		if seen[key] {
			return
		}
		seen[key] = true
		c.Findings = append(c.Findings, ProposalFinding{Stage: stage, Code: code, Path: path, Message: message})
	}
	local := a.stage.inputs.Raw.Local
	for _, d := range a.stage.checked.Diagnostics {
		path := d.Source.Path
		if d.Source.Origin == local.Source.Origin && path != "" && !local.RepositoryArea.IsRoot() {
			path = local.RepositoryArea.String() + "/" + path
		}
		add("model", d.Code, path, positioned(d.Message, d.Position))
	}
	for _, d := range a.registry.Diagnostics() {
		add("registry", d.Code, d.Source.String(), positioned(d.Message, d.Position))
	}
	for _, d := range a.execution.Diagnostics() {
		add("execution", d.Code, d.Source.File, positioned(d.Message, d.Position))
	}
	for _, d := range a.files.Diagnostics() {
		add("files", d.Code, d.Source.File, positioned(d.Message, d.Position))
	}
	if accounting, ok := a.files.Accounting(); ok {
		for _, f := range accounting.Findings() {
			add("files", f.Code, f.Path.String(), f.Message)
		}
	}
	for _, d := range a.contexts.Diagnostics() {
		add("context", d.Code, d.Package, d.Message)
	}
	return c
}
