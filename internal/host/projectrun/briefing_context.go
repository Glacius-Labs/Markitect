package projectrun

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/host/projectbriefing"
)

// BriefingContext contains structured changes and public contracts, never
// another Manager's private transcript. Dismissals do not alter this context.
type BriefingContext struct {
	Digest    string                     `json:"digest"`
	Briefings []projectbriefing.Briefing `json:"briefings"`
	Events    []projectbriefing.Event    `json:"events"`
}

func managerBriefing(root, modelDigest, manager string, revision ...string) (BriefingContext, error) {
	state, _, err := projectbriefing.Read(root)
	if err != nil {
		return BriefingContext{}, err
	}
	if len(state.Briefings) == 0 {
		return BriefingContext{Briefings: []projectbriefing.Briefing{}, Events: []projectbriefing.Event{}}, nil
	}
	briefings, events, binding, err := projectbriefing.LoadForManager(root, modelDigest, manager, revision...)
	if err != nil {
		return BriefingContext{}, fmt.Errorf("record accepted model briefing before work on this model: %w", err)
	}
	return BriefingContext{Digest: binding, Briefings: briefings, Events: events}, nil
}

func briefingBindings(root string, project *Project) (map[string]string, error) {
	result := map[string]string{}
	for _, manager := range project.Report.Managers {
		briefing, err := managerBriefing(root, project.Report.ModelDigest, manager.ID, project.Revision)
		if err != nil {
			return nil, err
		}
		result[manager.ID] = briefing.Digest
	}
	return result, nil
}
