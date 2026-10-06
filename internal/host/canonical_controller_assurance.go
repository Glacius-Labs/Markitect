package host

import (
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/host/assurance"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

// Validate the declared composition graph before executing any actor. This is
// structural graph validation, not fabricated verification evidence.
func validateCanonicalControllerAssurance(cfg CanonicalControllerConfig, fixed *CanonicalSource) error {
	if len(cfg.AssuranceScopes) == 0 {
		return nil
	}
	requests, err := projectionRequestIndex(fixed)
	if err != nil {
		return err
	}
	graph := assurance.Input{RootIDs: cfg.AssuranceRoots}
	for _, scope := range cfg.AssuranceScopes {
		request, found := requests[scope.ProjectionID]
		if !found {
			return fmt.Errorf("assurance scope %s selects unknown Projection %s", scope.ID, scope.ProjectionID)
		}
		node := assurance.Node{ID: scope.ID, Children: scope.Children}
		for _, def := range request.Definitions {
			node.ScopeIDs = append(node.ScopeIDs, def.Identity().Key())
		}
		for _, check := range scope.Checks {
			digest, err := digestCanonicalValue(check)
			if err != nil {
				return err
			}
			node.RequiredChecks = append(node.RequiredChecks, records.CheckIdentity{ID: check.Name, Version: "configured/v1", Digest: digest})
		}
		graph.Nodes = append(graph.Nodes, node)
	}
	report, err := assurance.Evaluate(graph)
	if err != nil {
		return err
	}
	if len(report.Nodes) != len(graph.Nodes) {
		return errors.New("all configured assurance scopes must be reachable from explicit roots")
	}
	return nil
}
