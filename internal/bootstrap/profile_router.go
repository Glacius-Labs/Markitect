package bootstrap

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
)

type agentMetadataRouter struct {
	codex, claude runtimeconfig.AgentProfilePreparer
}

func (r agentMetadataRouter) PrepareAgentProfile(ctx context.Context, input runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
	var selected runtimeconfig.AgentProfilePreparer
	switch input.Section() {
	case "codexAppServer":
		selected = r.codex
	case "claudeCode":
		selected = r.claude
	}
	if selected == nil {
		return runtimeconfig.ProfileObservation{InputBinding: input.Binding(), Status: model.NotCompleted}, errors.New("selected Agent metadata adapter is unavailable")
	}
	return selected.PrepareAgentProfile(ctx, input)
}
