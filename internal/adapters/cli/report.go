package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
)

// These values are CLI projections of actual bound producer reports. They are
// presentation records, not canonical Definitions or an execution configuration.
type preparationReport struct {
	FormatVersion     int             `json:"formatVersion"`
	Operation         string          `json:"operation"`
	RequestedStage    string          `json:"requestedStage"`
	RequestedRevision string          `json:"requestedRevision"`
	Status            model.Status    `json:"status"`
	Source            any             `json:"source"`
	Stages            map[string]any  `json:"stages"`
	Contexts          []contextReport `json:"contexts,omitempty"`
	Attempts          []any           `json:"attempts"`
	Error             string          `json:"error,omitempty"`
}

type contextReport struct {
	Package              string `json:"package"`
	ModelID              string `json:"modelID"`
	CanonicalModelDigest string `json:"canonicalModelDigest"`
	Definitions          []any  `json:"definitions"`
}

func (c *Command) prepare(ctx context.Context, request project.PreviewRequest, operation, packagePath string, metadata bool) (preparationReport, error) {
	r := preparationReport{FormatVersion: 1, Operation: operation, RequestedStage: "structural", Status: model.NotCompleted, Stages: map[string]any{"metadata": map[string]any{"requested": false}, "settings": map[string]any{"requested": false}}}
	var final project.PreviewPreparationAttempt
	var err error
	result, e := c.preparation.Preview(ctx, request, model.Limits{})
	err = e
	r.Status = result.Status()
	for _, attempt := range result.Attempts() {
		r.Attempts = append(r.Attempts, attemptReport(attempt))
		final = attempt
	}
	if metadata {
		r.RequestedStage = "metadata"
		observed, metadataErr := runtimeconfig.Prepare(ctx, final.Model().Inputs().Raw, c.agents, c.checks)
		err = errors.Join(err, metadataErr)
		var profiles []any
		for _, p := range observed.Profiles {
			id, complete := p.Identity()
			profiles = append(profiles, map[string]any{"namespace": p.Input().Namespace(), "name": p.Input().Name(), "observation": p.Observation(), "comparisonIdentity": id, "identityAvailable": complete})
		}
		r.Stages["metadata"] = map[string]any{"requested": true, "status": observed.Status, "profiles": profiles, "requiredInstructions": observed.RequiredInstructions, "diagnostics": observed.Diagnostics}
		if observed.Status == model.Failed || r.Status != model.Failed && observed.Status != model.Passed {
			r.Status = observed.Status
		}
	}

	m := final.Model()
	inputs := m.Inputs()
	mode := "working_preview"
	if request.Revision != "" {
		mode = "fixed_preview"
	}
	r.Source = map[string]any{"observation": mode, "head": inputs.Head, "revision": inputs.Revision, "request": m.Request(), "censusContentDigest": inputs.Raw.Census.Digest().String(), "censusFiles": len(inputs.Raw.Census.Entries())}
	r.Stages["structural"] = attemptReport(final)
	selections, complete := final.Contexts().Selections()
	if complete {
		for _, selected := range selections {
			if operation == "context" && selected.Package() != packagePath {
				continue
			}
			view := contextReport{Package: selected.Package(), ModelID: selected.ModelID(), CanonicalModelDigest: selected.ModelDigest()}
			for _, d := range selected.Definitions() {
				view.Definitions = append(view.Definitions, map[string]any{"definition": definitionReport(d.Definition), "canonicalModelDigest": d.ModelDigest, "reasons": d.Reasons})
			}
			r.Contexts = append(r.Contexts, view)
		}
		if operation == "context" && len(r.Contexts) == 0 {
			r.Status = model.Failed
			err = fmt.Errorf("unknown or imported local Package %q", packagePath)
		}
	}
	return r, err
}

func attemptReport(a project.PreviewPreparationAttempt) map[string]any {
	checked := a.Model().Checked()
	var models []any
	for _, m := range checked.Models {
		models = append(models, map[string]any{"id": m.ID, "local": m.Local, "status": m.Status, "source": m.Manifest, "canonicalModelDigest": m.Digest, "packages": m.Packages, "definitions": len(m.Definitions), "phases": m.Phases})
	}
	files := a.Files()
	accounting := map[string]any{"available": false}
	if report, ok := files.Accounting(); ok {
		var entries []any
		for _, f := range report.Files() {
			registered, present := f.Registration()
			entry := map[string]any{"path": f.Entry().Path().String(), "class": f.Class(), "why": f.Why(), "contentDigest": f.Entry().ContentID().String(), "mode": f.Entry().Mode(), "registered": present}
			if present {
				entry["registration"] = registered
			}
			entries = append(entries, entry)
		}
		// Stale links are realization still to do, not failures of this state.
		accounting = map[string]any{"available": true, "status": report.Status(), "treeDigest": report.TreeID().String(), "files": entries, "findings": report.Findings(), "stale": report.Stale()}
	}
	r := map[string]any{"model": map[string]any{"status": checked.Status, "coordinates": checked.Coordinates, "models": models, "diagnostics": checked.Diagnostics}, "execution": map[string]any{"status": a.Execution().Status(), "declarations": a.Execution().Partial(), "diagnostics": a.Execution().Diagnostics()}, "registry": map[string]any{"status": a.Registry().Status(), "discovered": a.Registry().Discovered(), "records": a.Registry().Records(), "diagnostics": a.Registry().Diagnostics()}, "files": map[string]any{"status": files.Status(), "accounting": accounting, "diagnostics": files.Diagnostics()}, "selection": map[string]any{"status": a.Contexts().Status(), "diagnostics": a.Contexts().Diagnostics()}}
	if a.Error() != nil {
		r["error"] = a.Error().Error()
	}
	return r
}

func definitionReport(d model.Definition) map[string]any {
	spec := map[string]any{}
	for _, field := range d.Spec {
		spec[field.Name] = valueReport(field.Value)
	}
	return map[string]any{"identity": d.Identity, "kind": d.Kind, "title": d.Title, "body": d.Body, "spec": spec, "bodyLinks": d.BodyLinks, "source": d.Source}
}

func valueReport(v model.Value) any {
	switch v.Kind() {
	case model.StringValue, model.EnumValue:
		x, _ := v.Text()
		return x
	case model.BooleanValue:
		x, _ := v.Boolean()
		return x
	case model.IntegerValue:
		x, _ := v.Integer()
		return map[string]string{"integer": x}
	case model.NumberValue:
		x, _ := v.Number()
		return x
	case model.ReferenceValue:
		x, _ := v.Reference()
		return x
	case model.ObjectValue:
		x, _ := v.Fields()
		r := map[string]any{}
		for _, f := range x {
			r[f.Name] = valueReport(f.Value)
		}
		return r
	case model.ListValue:
		x, _ := v.Items()
		r := make([]any, len(x))
		for i, item := range x {
			r[i] = valueReport(item)
		}
		return r
	}
	return nil
}
