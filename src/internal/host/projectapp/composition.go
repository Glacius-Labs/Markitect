package projectapp

import (
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

// NewOperations is the product composition root shared by CLI and MCP.
// Constructing it does not start a provider or create a workspace.
func NewOperations() Operations {
	return Operations{Host: projectrun.Host{
		Load: projectwork.Load, FromSnapshot: projectwork.FromSnapshot,
		PlanEdit: projectwork.PlanEdit, ApplyEdit: projectwork.ApplyEdit,
		Workspaces: projectrun.NewLocalWorkspaceService(),
	}, Invoker: projectexecutor.New(codexappserver.Options{}, projectrun.NativeHelperPolicy{})}
}
