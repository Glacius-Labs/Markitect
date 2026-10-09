package projectapp

import (
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

type InitOperation struct {
	Root  string `json:"root"`
	Name  string `json:"name"`
	Write bool   `json:"write,omitempty"`
}

type CheckSource struct {
	ProjectDigest string `json:"projectDigest"`
	Revision      string `json:"revision"`
	Provisional   bool   `json:"provisional"`
	CoverageMode  string `json:"coverageMode"`
}

// CheckResult gives transports the selected source identity and the complete
// structural report while retaining direct findings and coverage for callers.
type CheckResult struct {
	Source   CheckSource             `json:"source"`
	Report   projectmodel.Report     `json:"report"`
	Findings []projectmodel.Finding  `json:"findings"`
	Unknown  []string                `json:"unknown"`
	Coverage *projectcoverage.Report `json:"coverage,omitempty"`
}

type ContextOperation struct {
	Selection Selection `json:"selection"`
	ManagerID string    `json:"managerId"`
}

type DocumentOperation struct {
	Selection Selection `json:"selection"`
	Write     bool      `json:"write,omitempty"`
}

type EditOperation struct {
	Selection      Selection            `json:"selection"`
	Mutation       projectwork.Mutation `json:"mutation"`
	Write          bool                 `json:"write,omitempty"`
	ExpectedDigest string               `json:"expectedDigest,omitempty"`
}

type ImpactOperation struct {
	Root         string `json:"root"`
	BaseRevision string `json:"baseRevision"`
	Revision     string `json:"revision"`
}

func (o Operations) Init(operation InitOperation) (projectwork.InitPlan, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectwork.InitPlan{}, err
	}
	return projectwork.Init(operation.Root, operation.Name, operation.Write)
}

func (o Operations) Check(selection Selection) (CheckResult, error) {
	project, err := loadSelectedProject(selection)
	if err != nil {
		return CheckResult{}, err
	}
	return CheckResult{
		Source: CheckSource{ProjectDigest: project.Digest, Revision: project.Revision, Provisional: project.Provisional, CoverageMode: project.Config.CoverageMode},
		Report: project.Report, Findings: project.Report.Findings, Unknown: project.Report.Unknown, Coverage: project.Coverage,
	}, nil
}

func (o Operations) Index(selection Selection) (projectmodel.Report, error) {
	project, err := loadSelectedProject(selection)
	if err != nil {
		return projectmodel.Report{}, err
	}
	return project.Report, nil
}

func (o Operations) Context(operation ContextOperation) (projectmodel.ManagerContext, error) {
	project, err := loadSelectedProject(operation.Selection)
	if err != nil {
		return projectmodel.ManagerContext{}, err
	}
	if project.Config.WorkflowMode == projectwork.WorkflowModeGuided && !project.Provisional && project.Revision != "" {
		if _, err := projectbriefing.EnsureAcceptedHistory(operation.Selection.Root, project.Revision); err != nil {
			return projectmodel.ManagerContext{}, err
		}
	}
	return projectmodel.Context(project.Report, operation.ManagerID)
}

func (o Operations) Document(operation DocumentOperation) (string, error) {
	project, err := loadSelectedProject(operation.Selection)
	if err != nil {
		return "", err
	}
	return projectwork.Document(project, operation.Write)
}

func (o Operations) Edit(operation EditOperation) (projectwork.EditPlan, error) {
	project, err := loadSelectedProject(operation.Selection)
	if err != nil {
		return projectwork.EditPlan{}, err
	}
	plan, err := projectwork.PlanEdit(project, operation.Mutation)
	if err != nil {
		return projectwork.EditPlan{}, err
	}
	if !operation.Write {
		return plan, nil
	}
	if operation.ExpectedDigest != plan.Digest {
		return projectwork.EditPlan{}, fmt.Errorf("--expect does not match the exact edit plan digest %s", plan.Digest)
	}
	return projectwork.ApplyEdit(operation.Selection.Root, plan, plan.BaseDigest)
}

func (o Operations) Impact(operation ImpactOperation) (projectmodel.ChangeImpact, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectmodel.ChangeImpact{}, err
	}
	base, err := projectwork.Load(operation.Root, operation.BaseRevision)
	if err != nil {
		return projectmodel.ChangeImpact{}, fmt.Errorf("load base project: %w", err)
	}
	candidate, err := projectwork.Load(operation.Root, operation.Revision)
	if err != nil {
		return projectmodel.ChangeImpact{}, fmt.Errorf("load candidate project: %w", err)
	}
	return projectmodel.Impact(base.Report, candidate.Report), nil
}

func (o Operations) Coverage(selection Selection) (projectcoverage.Report, error) {
	if err := requireRoot(selection.Root); err != nil {
		return projectcoverage.Report{}, err
	}
	return projectwork.Coverage(selection.Root, selection.Revision)
}

func loadSelectedProject(selection Selection) (*projectwork.Project, error) {
	if err := requireRoot(selection.Root); err != nil {
		return nil, err
	}
	project, err := projectwork.Load(selection.Root, selection.Revision)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("selected project load returned no project")
	}
	return project, nil
}
