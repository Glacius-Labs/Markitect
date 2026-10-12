package projectapp

import (
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

// NewOperations is the product composition root shared by CLI and MCP.
// Constructing it does not start a provider or create a workspace.
func NewOperations() Operations {
	host := projectrun.Host{
		Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot,
		PlanEdit: projectwork.PlanEdit, ApplyEdit: projectwork.ApplyEdit,
		ClassifyCandidate: projectwork.ClassifyCandidate,
		RenderDocument:    func(project *projectvalue.Project) (string, error) { return projectwork.Document(project, false) },
		Workspaces:        projectrun.NewLocalWorkspaceService(),
	}
	host.PrepareBriefings = prepareBriefings
	host.BriefingForManager = briefingForManager
	host.ResolveVerifiedBriefing = resolveVerifiedBriefing
	host.LoadDeliveryScope = loadDeliveryScope
	host.LoadExplorationReadiness = func(root, explorationID, scopeID string) (projectrun.DeliveryScope, projectvalue.Binding, projectrun.ExplorationReadiness, error) {
		return loadExplorationReadiness(host, root, explorationID, scopeID)
	}
	host.BindExploration = bindExploration
	host.ValidateExploration = validateExploration
	host.ExplorationCapturePaths = explorationCapturePaths
	host.CompleteExploration = completeExploration
	return Operations{Host: host, Invoker: projectexecutor.New(codexappserver.Options{}, projectrun.NativeHelperPolicy{})}
}
