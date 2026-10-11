package projectrun

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
)

// BriefingContext contains structured changes and public contracts, never
// another Manager's private transcript. Dismissals do not alter this context.
type BriefingContext struct {
	Digest    string                     `json:"digest"`
	Briefings []projectbriefing.Briefing `json:"briefings"`
	Events    []projectbriefing.Event    `json:"events"`
}

// captureAcceptedHistory reconciles accepted model history at revision. A
// write path persists it and its briefings are then read from the store (nil
// view). A read preview writes nothing and returns the history computed in
// memory, so it reports the same briefings the write path would persist.
func captureAcceptedHistory(root, revision string, persist bool) (*projectbriefing.Store, error) {
	if persist {
		_, err := projectbriefing.EnsureAcceptedHistory(root, revision)
		return nil, err
	}
	history, err := projectbriefing.ReadAcceptedHistory(root, revision)
	if err != nil {
		return nil, err
	}
	return &history.State, nil
}

func managerBriefing(root, modelDigest, manager string, revision ...string) (BriefingContext, error) {
	return managerBriefingIn(root, nil, modelDigest, manager, revision...)
}

// managerBriefingIn reads a Manager's briefing context from view, the
// accepted history a read preview computed, or from the persisted store when
// view is nil.
func managerBriefingIn(root string, view *projectbriefing.Store, modelDigest, manager string, revision ...string) (BriefingContext, error) {
	var state projectbriefing.Store
	if view != nil {
		state = *view
	} else {
		var err error
		if state, _, err = projectbriefing.Read(root); err != nil {
			return BriefingContext{}, err
		}
	}
	if len(state.Briefings) == 0 {
		return BriefingContext{Briefings: []projectbriefing.Briefing{}, Events: []projectbriefing.Event{}}, nil
	}
	briefings, events, binding, err := projectbriefing.LoadForManagerFrom(root, state, modelDigest, manager, revision...)
	if err != nil {
		return BriefingContext{}, fmt.Errorf("record accepted model briefing before work on this model: %w", err)
	}
	return BriefingContext{Digest: binding, Briefings: briefings, Events: events}, nil
}

func briefingBindings(root string, project *Project) (map[string]string, error) {
	return briefingBindingsIn(root, nil, project)
}

// briefingBindingsIn is briefingBindings over view, as in managerBriefingIn.
func briefingBindingsIn(root string, view *projectbriefing.Store, project *Project) (map[string]string, error) {
	result := map[string]string{}
	for _, manager := range project.Report.Managers {
		briefing, err := managerBriefingIn(root, view, project.Report.ModelDigest, manager.ID, project.Revision)
		if err != nil {
			return nil, err
		}
		result[manager.ID] = briefing.Digest
	}
	return result, nil
}
