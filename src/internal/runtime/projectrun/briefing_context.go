package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectbriefing"
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
func captureAcceptedHistory(host Host, root, revision string, persist bool) (*projectbriefing.Store, error) {
	if host.PrepareBriefings == nil {
		return nil, fmt.Errorf("accepted history acquisition is not bound")
	}
	return host.PrepareBriefings(root, revision, persist)
}

func managerBriefing(host Host, root, modelDigest, manager string, revision ...string) (BriefingContext, error) {
	return managerBriefingIn(host, root, nil, modelDigest, manager, revision...)
}

// managerBriefingIn reads a Manager's briefing context from view, the
// accepted history a read preview computed, or from the persisted store when
// view is nil.
func managerBriefingIn(host Host, root string, view *projectbriefing.Store, modelDigest, manager string, revision ...string) (BriefingContext, error) {
	if host.BriefingForManager == nil {
		return BriefingContext{}, fmt.Errorf("manager briefing acquisition is not bound")
	}
	selectedRevision := ""
	if len(revision) != 0 {
		selectedRevision = revision[0]
	}
	return host.BriefingForManager(root, selectedRevision, modelDigest, manager, view)
}

func briefingBindings(host Host, root string, project *projectvalue.Project) (map[string]string, error) {
	return briefingBindingsIn(host, root, nil, project)
}

// briefingBindingsIn is briefingBindings over view, as in managerBriefingIn.
func briefingBindingsIn(host Host, root string, view *projectbriefing.Store, project *projectvalue.Project) (map[string]string, error) {
	result := map[string]string{}
	for _, manager := range project.Report.Managers {
		briefing, err := managerBriefingIn(host, root, view, project.Report.ModelDigest, manager.ID, project.Revision)
		if err != nil {
			return nil, err
		}
		result[manager.ID] = briefing.Digest
	}
	return result, nil
}
