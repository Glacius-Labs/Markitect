package projectrun

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	projectconfig "github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// planBasis contains the immutable project inputs and runtime policy that one
// PlanExpecting call has checked before it derives executable Manager work.
type planBasis struct {
	Root             string
	Request          PlanRequest
	Operation        string
	Project          *Project
	Working          *Project
	ChangeBase       *Project
	ChangeImpact     *projectmodel.ChangeImpact
	BriefingView     *projectbriefing.Store
	Runtime          projectconfig.Runtime
	TargetHead       string
	TargetBranch     string
	RepositoryDigest string
}

func loadPlanBasis(host Host, root, revision string, request PlanRequest) (planBasis, error) {
	basis := planBasis{}
	operation, err := NormalizeOperation(request.Operation)
	if err != nil {
		return basis, err
	}
	request.Operation = operation
	if operation != OperationApply && (len(request.Managers) > 0 || strings.TrimSpace(request.SinceRevision) != "" || request.ModelEdit != nil) {
		return basis, fmt.Errorf("%s does not accept manager, since-revision, or model-edit narrowing; prepare and accept any model changes separately", operation)
	}
	if host.Load == nil || host.FromSnapshot == nil || host.PlanEdit == nil {
		return basis, fmt.Errorf("project Host frontend is incomplete")
	}
	if strings.TrimSpace(request.Goal) == "" {
		return basis, fmt.Errorf("a bounded project goal is required")
	}
	if request.BaseRevision != "" {
		revision = request.BaseRevision
	}
	if strings.TrimSpace(revision) == "" {
		revision, err = resolveGitHead(root)
		if err != nil {
			return basis, fmt.Errorf("resolve fixed default base revision: %w", err)
		}
	}
	targetHead, err := resolveGitHead(root)
	if err != nil {
		return basis, fmt.Errorf("capture current target HEAD: %w", err)
	}
	targetBranch, err := resolveGitBranch(root)
	if err != nil {
		return basis, fmt.Errorf("capture current target branch: %w", err)
	}
	project, err := host.Load(root, revision)
	if err != nil {
		return basis, fmt.Errorf("load selected project revision: %w", err)
	}
	if project == nil || project.Snapshot == nil {
		return basis, fmt.Errorf("project frontend returned no fixed snapshot")
	}
	var changeBase *Project
	var changeImpact *projectmodel.ChangeImpact
	if strings.TrimSpace(request.SinceRevision) != "" {
		changeBase, err = host.Load(root, request.SinceRevision)
		if err != nil {
			return basis, fmt.Errorf("load project change baseline %q: %w", request.SinceRevision, err)
		}
		if changeBase == nil || changeBase.Snapshot == nil {
			return basis, fmt.Errorf("project frontend returned no change-baseline snapshot")
		}
		impact := projectmodel.Impact(changeBase.Report, project.Report)
		if impact.Digest == "" || impact.BaseDigest != changeBase.Report.Digest || impact.CandidateDigest != project.Report.Digest {
			return basis, fmt.Errorf("project change impact did not bind both selected reports")
		}
		changeImpact = &impact
	}
	if project.Snapshot.Provisional {
		return basis, fmt.Errorf("project run requires a non-provisional snapshot")
	}
	// A preview reads accepted history in memory; only an authorized plan
	// persists it. ExplorationBinding passes the view it already computed.
	briefingView := request.acceptedBriefings
	if request.ExecuteAuthorized {
		briefingView = nil
	}
	if briefingView == nil && (project.Config.WorkflowMode == "guided" || request.ExplorationID != "") && request.ModelEdit == nil {
		if briefingView, err = captureAcceptedHistory(root, project.Revision, request.ExecuteAuthorized); err != nil {
			return basis, fmt.Errorf("capture accepted model history: %w", err)
		}
	}
	working, err := host.Load(root, "")
	if err != nil {
		return basis, fmt.Errorf("capture current project working inputs: %w", err)
	}
	if working == nil || working.Snapshot == nil {
		return basis, fmt.Errorf("project runtime requires a selected working-input snapshot")
	}
	if err := requireCleanSelectedBasisAtRevision(root, project.Revision, project.Snapshot, working.Snapshot); err != nil {
		return basis, err
	}
	repository, err := source.IdentifyGit(root)
	if err != nil {
		return basis, fmt.Errorf("identify project repository: %w", err)
	}
	currentHead, headErr := resolveGitHead(root)
	currentBranch, branchErr := resolveGitBranch(root)
	if headErr != nil || branchErr != nil || currentHead != targetHead || currentBranch != targetBranch {
		return basis, ErrStale
	}
	if !startableReport(project.Report) {
		return basis, fmt.Errorf("project model analysis did not succeed: %s", project.Report.Status)
	}
	if hasErrorFinding(project.Report.Findings) {
		return basis, fmt.Errorf("project model has structural error findings")
	}
	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		return basis, err
	}
	basis = planBasis{
		Root: root, Request: request, Operation: operation, Project: project, Working: working,
		ChangeBase: changeBase, ChangeImpact: changeImpact, BriefingView: briefingView,
		Runtime: runtime, TargetHead: targetHead, TargetBranch: targetBranch,
		RepositoryDigest: repository.Digest,
	}
	return basis, nil
}
