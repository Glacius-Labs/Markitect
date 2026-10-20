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
// applies. Like the capture's second pass, the instruction and ignore files
// the proposed tree newly references are read from it and prepared once more.
func CheckProposal(start CapturedPreparationAttempt, tree snapshot.Tree) ProposalCheck {
	base := start.Model()
	a := rebindStructural(base, tree, extractModel)
	if raw, missing := proposalClosure(a, tree); missing {
		base.inputs.Raw = raw
		a = rebindStructural(base, tree, extractModel)
	}
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

// proposalClosure adds the instruction and ignore files the prepared stand
// references but did not read, when the proposed tree holds them.
func proposalClosure(a CapturedPreparationAttempt, tree snapshot.Tree) (RawInputs, bool) {
	raw := cloneRaw(a.stage.inputs.Raw)
	closure := observedPreparationClosure(raw, a.registry.Discovered(), a.execution.Partial())
	repository := a.stage.request.Selection.Repository
	missing := false
	add := func(files []SourceFile, required []snapshot.Path) []SourceFile {
		seen := make(map[snapshot.Path]bool)
		for _, f := range files {
			seen[f.RepositoryPath] = true
		}
		for _, path := range required {
			entry, found := tree.Lookup(path)
			if seen[path] || !found {
				continue
			}
			seen[path], missing = true, true
			files = append(files, SourceFile{Location: SourceLocation{Boundary: repository, Path: path}, File: entry, RepositoryPath: path})
		}
		return files
	}
	raw.Instructions = add(raw.Instructions, closure.instructions)
	raw.Ignore = add(raw.Ignore, closure.ignore)
	return raw, missing
}
