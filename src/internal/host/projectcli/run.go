package projectcli

import (
	"encoding/json"
	"io"

	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

func projectRunHost() projectrun.Host {
	return projectrun.Host{
		Load:         projectwork.Load,
		FromSnapshot: projectwork.FromSnapshot,
		PlanEdit:     projectwork.PlanEdit,
		ApplyEdit:    projectwork.ApplyEdit,
		Workspaces:   projectrun.NewLocalWorkspaceService(),
	}
}

func projectRunInvoker() projectrun.Invoker {
	return projectrun.NewTransportInvoker(codexappserver.Options{})
}

func projectOperations() projectapp.Operations {
	return projectapp.Operations{Host: projectRunHost(), Invoker: projectRunInvoker()}
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
