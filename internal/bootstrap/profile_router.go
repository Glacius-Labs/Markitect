package bootstrap

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

type agentMetadataRouter struct {
	codex, claude apply.AgentProfilePreparer
}

func (r agentMetadataRouter) PrepareAgentProfile(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	var selected apply.AgentProfilePreparer
	switch input.Section() {
	case "codexAppServer":
		selected = r.codex
	case "claudeCode":
		selected = r.claude
	}
	if selected == nil {
		return apply.ProfileObservation{InputBinding: input.Binding(), Status: model.NotCompleted}, errors.New("selected Agent metadata adapter is unavailable")
	}
	return selected.PrepareAgentProfile(ctx, input)
}
