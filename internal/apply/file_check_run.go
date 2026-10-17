package apply

import (
	"cmp"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type fileChecker struct {
	result      *fileCheck
	raw         RawInputs
	inputs      map[snapshot.Path]realization.Classification
	protections map[string]bool
}

func (r *fileCheck) observe(status model.Status) {
	if status == model.Failed || r.status != model.Failed && status == model.NotCompleted {
		r.status = status
	}
}
func (c *fileChecker) issue(status model.Status, code string, path snapshot.Path, field string, position *model.Position, message string) {
	source := organization.SettingSource{File: path.String(), Field: field}
	if file, found := c.raw.Census.Lookup(path); found {
		source.ContentID = file.ContentID()
	}
	c.result.diagnostics = append(c.result.diagnostics, FileCheckDiagnostic{Code: code, Source: source, Position: position, Message: message})
	c.result.observe(status)
}

func checkFiles(parent registryPreparation, compiled model.Result, raw RawInputs, execution executionDecoding) fileCheck {
	r := fileCheck{status: parent.Status()}
	c := fileChecker{result: &r, raw: raw, inputs: make(map[snapshot.Path]realization.Classification), protections: make(map[string]bool)}
	if compiled.Status != model.Passed {
		if compiled.Status == model.Failed {
			r.observe(model.Failed)
		}
		c.issue(r.Status(), "files.model_not_passed", snapshot.Path{}, "", nil, "File checking requires the exact passed model parent.")
		return r
	}
	if parent.Status() != model.Passed {
		c.issue(parent.Status(), "files.registry_not_passed", snapshot.Path{}, "", nil, "The registry parent has not passed; no partial registry is used for accounting.")
	}
	r.observe(execution.Status())
	for _, diagnostic := range execution.diagnostics {
		r.diagnostics = append(r.diagnostics, FileCheckDiagnostic(diagnostic))
	}
	c.sourceInputs(compiled.Limits)
	c.instructions(execution.authored)
	c.ignore()
	for _, classification := range c.inputs {
		r.classifications = append(r.classifications, classification)
	}
	slices.SortFunc(r.classifications, func(a, b realization.Classification) int {
		return cmp.Or(cmp.Compare(a.Path().String(), b.Path().String()), cmp.Compare(a.Class(), b.Class()))
	})
	if parent.Status() == model.Passed {
		accounting := realization.AccountFiles(compiled, raw.Census, parent.registry, r.classifications)
		r.accounting = &accounting
		if accounting.Status() == realization.AccountingIncomplete {
			r.observe(model.NotCompleted)
		}
		if accounting.Status() == realization.AccountingInvalid {
			for _, finding := range accounting.Findings() {
				// Missing observations can still explain otherwise-unclassified files.
				// Keep the actual report, without turning that uncertainty into a
				// known policy violation. Other observed conflicts remain failures.
				if finding.Code != "file.unexplained" || r.status != model.NotCompleted {
					r.observe(model.Failed)
				}
			}
		}
	}
	slices.SortFunc(r.diagnostics, func(a, b FileCheckDiagnostic) int {
		ap, bp := model.Position{}, model.Position{}
		an, bn := 0, 0
		if a.Position != nil {
			ap, an = *a.Position, 1
		}
		if b.Position != nil {
			bp, bn = *b.Position, 1
		}
		return cmp.Or(cmp.Compare(a.Source.File, b.Source.File), cmp.Compare(a.Source.Field, b.Source.Field), cmp.Compare(a.Code, b.Code), cmp.Compare(a.Message, b.Message), cmp.Compare(an, bn), cmp.Compare(ap.Line, bp.Line), cmp.Compare(ap.Column, bp.Column))
	})
	return r
}
