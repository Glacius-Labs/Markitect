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
	_, err := canonicalControllerAssuranceGraph(cfg, fixed)
	return err
}

func canonicalControllerAssuranceGraph(cfg CanonicalControllerConfig, fixed *CanonicalSource) (assurance.Input, error) {
	if len(cfg.AssuranceScopes) == 0 {
		return assurance.Input{}, nil
	}
	requests, err := projectionRequestIndex(fixed)
	if err != nil {
		return assurance.Input{}, err
	}
	graph := assurance.Input{RootIDs: cfg.AssuranceRoots}
	for _, scope := range cfg.AssuranceScopes {
		request, found := requests[scope.ProjectionID]
		if !found {
			return assurance.Input{}, fmt.Errorf("assurance scope %s selects unknown Projection %s", scope.ID, scope.ProjectionID)
		}
		node := assurance.Node{ID: scope.ID, Children: scope.Children}
		for _, def := range request.Definitions {
			node.ScopeIDs = append(node.ScopeIDs, def.Identity().Key())
		}
		for _, check := range scope.Checks {
			digest, err := digestCanonicalValue(check)
			if err != nil {
				return assurance.Input{}, err
			}
			node.RequiredChecks = append(node.RequiredChecks, records.CheckIdentity{ID: check.Name, Version: "configured/v1", Digest: digest})
		}
		graph.Nodes = append(graph.Nodes, node)
	}
	report, err := assurance.Evaluate(graph)
	if err != nil {
		return assurance.Input{}, err
	}
	if len(report.Nodes) != len(graph.Nodes) {
		return assurance.Input{}, errors.New("all configured assurance scopes must be reachable from explicit roots")
	}
	// A C11 lifecycle run reached verification before refusing scope checks
	// absent from the selected source config. Reject that Host configuration
	// before Execute can invoke an actor or materialize any candidate; this is
	// source-bound input validation, not a model or verifier conclusion.
	for _, scope := range cfg.AssuranceScopes {
		request := requests[scope.ProjectionID]
		if _, err := canonicalControllerScopedChecks(fixed, request.Projector, scope.Checks); err != nil {
			return assurance.Input{}, fmt.Errorf("assurance scope %s: %w", scope.ID, err)
		}
	}
	return graph, nil
}
