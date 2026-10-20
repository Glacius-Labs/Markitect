package cli

import (
	"context"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
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

func (c *Command) prepare(ctx context.Context, request apply.PreviewRequest, operation, packagePath string, metadata bool) (preparationReport, error) {
	r := preparationReport{FormatVersion: 1, Operation: operation, RequestedStage: "structural", Status: model.NotCompleted, Stages: map[string]any{"metadata": map[string]any{"requested": false}, "settings": map[string]any{"requested": false}}}
	var final apply.PreviewPreparationAttempt
	var err error
	if metadata {
		r.RequestedStage = "metadata"
		result, e := c.preparation.PreviewSettings(ctx, request, model.Limits{}, c.agents, c.checks)
		err = e
		r.Status = result.Status()
		for _, attempt := range result.Attempts() {
			s := attempt.Settings()
			managers := []organization.ManagerSettings(nil)
			if resolved, ok := s.Settings(); ok {
				managers = resolved.Managers()
			}
			var profiles, checks []any
			for _, p := range s.ProfileObservations() {
				identity, complete := p.Identity()
				input := p.Input()
				profiles = append(profiles, map[string]any{"namespace": input.Namespace(), "name": input.Name(), "source": p.Source(), "observation": p.Observation(), "comparisonIdentity": identity, "identityAvailable": complete})
			}
			if selected, ok := s.Checks(); ok {
				for _, check := range selected {
					identity, _ := check.Profile().Identity()
					checks = append(checks, map[string]any{"declaration": check.Declaration(), "comparisonIdentity": identity})
				}
			}
			declarations, declared := attempt.Defaults().Declarations()
			entry := map[string]any{"structural": attemptReport(attempt.Structural()), "defaults": map[string]any{"status": attempt.Defaults().Status(), "productSource": attempt.Defaults().ProductSource(), "declarationsAvailable": declared, "declarations": declarations, "diagnostics": attempt.Defaults().Diagnostics()}, "metadata": map[string]any{"requested": true, "status": s.Status(), "profiles": profiles, "requiredInstructions": s.RequiredInstructions(), "diagnostics": s.Diagnostics()}, "settings": map[string]any{"requested": true, "status": s.Status(), "managers": managers, "checks": checks}}
			if attempt.Error() != nil {
				entry["error"] = attempt.Error().Error()
			}
			r.Attempts = append(r.Attempts, entry)
			r.Stages["metadata"], r.Stages["settings"], r.Stages["defaults"] = entry["metadata"], entry["settings"], entry["defaults"]
			final = attempt.Structural()
		}
	} else {
		result, e := c.preparation.Preview(ctx, request, model.Limits{})
		err = e
		r.Status = result.Status()
		for _, attempt := range result.Attempts() {
			r.Attempts = append(r.Attempts, attemptReport(attempt))
			final = attempt
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

func attemptReport(a apply.PreviewPreparationAttempt) map[string]any {
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
		accounting = map[string]any{"available": true, "status": report.Status(), "treeDigest": report.TreeID().String(), "files": entries, "findings": report.Findings()}
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
