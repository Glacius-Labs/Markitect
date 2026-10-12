package projectbriefing

import (
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
)

type ManagerView struct {
	context    projectmodel.ManagerContext
	statements map[string]projectmodel.Statement
	artifacts  map[string]bool
	checks     map[string]projectmodel.Check
	decisions  map[string]bool
	managers   map[string]bool
}

func NewManagerView(report projectmodel.Report, managerID string) (ManagerView, error) {
	context, err := projectmodel.Context(report, managerID)
	if err != nil {
		return ManagerView{}, err
	}
	view := ManagerView{
		context:    context,
		statements: map[string]projectmodel.Statement{},
		artifacts:  map[string]bool{},
		checks:     map[string]projectmodel.Check{},
		decisions:  map[string]bool{},
		managers:   map[string]bool{context.Manager.ID: true},
	}
	for _, statement := range context.Statements {
		view.statements[statement.ID] = statement
	}
	for _, statement := range context.Contracts {
		view.statements[statement.ID] = statement
	}
	for _, artifact := range context.Artifacts {
		view.artifacts[artifact.ID] = true
	}
	for _, check := range context.Checks {
		view.checks[check.ID] = check
	}
	for _, check := range context.ForeignChecks {
		view.checks[check.ID] = check
	}
	for _, decision := range context.Decisions {
		view.decisions[decision.ID] = true
	}
	for _, child := range context.Children {
		view.managers[child.ID] = true
	}
	return view, nil
}

func (v ManagerView) projectDefinition(definition *core.Definition) (*core.Definition, bool) {
	if definition == nil {
		return nil, false
	}
	projected := *definition
	projected.Spec = cloneSpec(definition.Spec)
	key := definition.Identity().Key()
	switch definition.Kind {
	case "Statement":
		statement, ok := v.statements[key]
		if !ok {
			return nil, false
		}
		projected.Purpose = ""
		projected.Spec["uses"] = append([]string(nil), statement.Uses...)
		projected.Spec["requires"] = append([]string(nil), statement.Requires...)
	case "Artifact":
		if !v.artifacts[key] {
			return nil, false
		}
		projected.Purpose = ""
	case "Check":
		check, ok := v.checks[key]
		if !ok {
			return nil, false
		}
		projected.Purpose = ""
		projected.Spec["uses"] = append([]string(nil), check.Uses...)
	case "Decision":
		if !v.decisions[key] {
			return nil, false
		}
		projected.Purpose = ""
	case "Manager":
		if v.context.Manager.ID == key {
			projected.Spec["instructions"] = v.context.Manager.Instructions
			break
		}
		for _, child := range v.context.Children {
			if child.ID == key {
				projected.Spec["instructions"] = child.Instructions
				return &projected, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
	return &projected, true
}

func cloneSpec(spec map[string]any) map[string]any {
	cloned := make(map[string]any, len(spec))
	for key, value := range spec {
		cloned[key] = value
	}
	return cloned
}

func ProjectEvent(event Event, before, after ManagerView) (Event, bool) {
	projectedBefore, beforeVisible := before.projectDefinition(event.Before)
	projectedAfter, afterVisible := after.projectDefinition(event.After)
	if !beforeVisible && !afterVisible {
		return Event{}, false
	}
	event.Before = projectedBefore
	event.After = projectedAfter
	if beforeVisible {
		event.DefinitionID = event.Before.Identity()
	} else {
		event.DefinitionID = event.After.Identity()
	}
	managers := map[string]bool{}
	for _, managerID := range event.AffectedManagers {
		if before.managers[managerID] || after.managers[managerID] {
			managers[managerID] = true
		}
	}
	event.AffectedManagers = make([]string, 0, len(managers))
	for managerID := range managers {
		event.AffectedManagers = append(event.AffectedManagers, managerID)
	}
	if len(event.AffectedManagers) == 0 {
		return Event{}, false
	}
	sort.Strings(event.AffectedManagers)
	artifacts := map[string]bool{}
	for _, artifactID := range event.AffectedArtifacts {
		if before.artifacts[artifactID] || after.artifacts[artifactID] {
			artifacts[artifactID] = true
		}
	}
	event.AffectedArtifacts = make([]string, 0, len(artifacts))
	for artifactID := range artifacts {
		event.AffectedArtifacts = append(event.AffectedArtifacts, artifactID)
	}
	sort.Strings(event.AffectedArtifacts)
	return event, true
}

func EmptyManagerView() ManagerView {
	return ManagerView{
		statements: map[string]projectmodel.Statement{},
		artifacts:  map[string]bool{},
		checks:     map[string]projectmodel.Check{},
		decisions:  map[string]bool{},
		managers:   map[string]bool{},
	}
}
