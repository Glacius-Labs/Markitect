package projectapp

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

type InitOperation struct {
	Root  string `json:"root"`
	Name  string `json:"name"`
	Write bool   `json:"write,omitempty"`
	// ExpectedDigest binds a write to the reviewed preview's digest.
	ExpectedDigest string `json:"expectedDigest,omitempty"`
}

type CheckSource struct {
	Name          string `json:"name"`
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
	Selection      Selection `json:"selection"`
	Write          bool      `json:"write,omitempty"`
	ExpectedDigest string    `json:"expectedDigest,omitempty"`
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

// ExplainOperation explains an impact (DEC-023): for the whole project, or
// with ManagerID only as that Manager sees it. The impact stays project scope.
type ExplainOperation struct {
	ImpactOperation
	ManagerID string `json:"managerId,omitempty"`
}

// ExplainedImpact is an impact with the explanation of its elements.
type ExplainedImpact struct {
	Impact      projectmodel.ChangeImpact      `json:"impact"`
	Explanation projectmodel.ImpactExplanation `json:"explanation"`
}

// TraceOperation walks one Manager's knowledge graph, which is built from its
// Context alone.
type TraceOperation struct {
	ContextOperation
	Request projectmodel.TraceRequest `json:"request"`
}

// TracedContext is a Manager's Context with a trace over its graph.
type TracedContext struct {
	Context projectmodel.ManagerContext `json:"context"`
	Trace   projectmodel.TraceResult    `json:"trace"`
}

func (o Operations) Init(operation InitOperation) (projectwork.InitPlan, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectwork.InitPlan{}, err
	}
	if operation.Write && operation.ExpectedDigest != "" {
		preview, err := projectwork.Init(operation.Root, operation.Name, false)
		if err != nil {
			return projectwork.InitPlan{}, err
		}
		if preview.Digest != operation.ExpectedDigest {
			return projectwork.InitPlan{}, staleError("the init preview changed; review the new preview and its digest")
		}
	}
	return projectwork.Init(operation.Root, operation.Name, operation.Write)
}

func (o Operations) Check(selection Selection) (CheckResult, error) {
	project, err := loadSelectedProject(selection)
	if err != nil {
		return CheckResult{}, err
	}
	return CheckResult{
		Source: CheckSource{Name: project.Config.Name, ProjectDigest: project.Digest, Revision: project.Revision, Provisional: project.Provisional, CoverageMode: project.Config.CoverageMode},
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
	managerID, err := resolveManagerName(project.Report, operation.ManagerID)
	if err != nil {
		return projectmodel.ManagerContext{}, err
	}
	return projectmodel.Context(project.Report, managerID)
}

// ManagerNameError is a short Manager name that names no Manager or more than
// one in the selected project.
type ManagerNameError struct{ Message string }

func (e ManagerNameError) Error() string { return e.Message }

// resolveManagerName accepts a Manager's full ID, or its short name when that
// name is unique in the same loaded project.
func resolveManagerName(report projectmodel.Report, manager string) (string, error) {
	if strings.HasPrefix(manager, "[") {
		return manager, nil
	}
	matches := []string{}
	for _, candidate := range report.Managers {
		if candidate.Name == manager {
			matches = append(matches, candidate.ID)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", ManagerNameError{fmt.Sprintf("no Manager is named %q; `markitect model` lists each Manager's ID and name", manager)}
	}
	return "", ManagerNameError{fmt.Sprintf("Manager name %q is ambiguous; use one of the full IDs: %s", manager, strings.Join(matches, ", "))}
}

// DocumentResult is the readable model document with its destination and
// digest, so a write can be bound to the reviewed preview.
type DocumentResult struct {
	Revision string `json:"revision,omitempty"`
	Path     string `json:"path"`
	Digest   string `json:"digest"`
	Content  string `json:"content"`
	Written  bool   `json:"written"`
}

func (o Operations) Document(operation DocumentOperation) (DocumentResult, error) {
	project, err := loadSelectedProject(operation.Selection)
	if err != nil {
		return DocumentResult{}, err
	}
	content, err := projectwork.Document(project, false)
	if err != nil {
		return DocumentResult{}, err
	}
	result := DocumentResult{Revision: project.Revision, Path: projectwork.DocumentPath(project.Config), Digest: contentDigest(content), Content: content}
	if !operation.Write {
		return result, nil
	}
	if operation.ExpectedDigest != "" && operation.ExpectedDigest != result.Digest {
		return DocumentResult{}, staleError("the document preview changed; review the new preview and its digest")
	}
	written, err := projectwork.Document(project, true)
	if err != nil {
		return DocumentResult{}, err
	}
	result.Content, result.Digest, result.Written = written, contentDigest(written), true
	return result, nil
}

// staleError reports a write whose reviewed preview no longer matches; it
// matches projectrun.ErrStale for adapters that classify stale writes.
type staleError string

func (e staleError) Error() string        { return string(e) }
func (e staleError) Is(target error) bool { return target == projectrun.ErrStale }

func contentDigest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
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
	base, candidate, err := loadImpactReports(operation)
	if err != nil {
		return projectmodel.ChangeImpact{}, err
	}
	return projectmodel.Impact(base, candidate), nil
}

func (o Operations) Explain(operation ExplainOperation) (ExplainedImpact, error) {
	base, candidate, err := loadImpactReports(operation.ImpactOperation)
	if err != nil {
		return ExplainedImpact{}, err
	}
	out := ExplainedImpact{Impact: projectmodel.Impact(base, candidate)}
	if operation.ManagerID == "" {
		out.Explanation = projectmodel.Explain(base, candidate)
		return out, nil
	}
	out.Explanation, err = projectmodel.ExplainForManager(base, candidate, operation.ManagerID)
	return out, err
}

func loadImpactReports(operation ImpactOperation) (projectmodel.Report, projectmodel.Report, error) {
	if err := requireRoot(operation.Root); err != nil {
		return projectmodel.Report{}, projectmodel.Report{}, err
	}
	base, err := projectwork.Load(operation.Root, operation.BaseRevision)
	if err != nil {
		return projectmodel.Report{}, projectmodel.Report{}, fmt.Errorf("load base project: %w", err)
	}
	candidate, err := projectwork.Load(operation.Root, operation.Revision)
	if err != nil {
		return projectmodel.Report{}, projectmodel.Report{}, fmt.Errorf("load candidate project: %w", err)
	}
	return base.Report, candidate.Report, nil
}

func (o Operations) Trace(operation TraceOperation) (TracedContext, error) {
	context, err := o.Context(operation.ContextOperation)
	if err != nil {
		return TracedContext{}, err
	}
	trace, err := projectmodel.ManagerGraph(context).Trace(operation.Request)
	return TracedContext{Context: context, Trace: trace}, err
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
