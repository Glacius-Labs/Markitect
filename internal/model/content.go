package model

// Each published model is checked in its own dependency closure. The entry
// model additionally retains whole-input acquisition/name diagnostics; a bad
// consumer must not prevent an independently valid dependency being hashed.
type contentState struct {
	graph      dependencyGraph
	bindings   authorBindings
	contracts  contractResult
	values     valueResult
	bodies     bodyResult
	identities []identityFinding
	status     []Status
}

func evaluateContent(scopes []sourceScope, sources []ModelResult, limits Limits) contentState {
	state := contentState{identities: checkScopeIdentities(scopes)}
	state.graph = bindDependencies(scopes)
	state.bindings = bindAuthorReferences(scopes, state.graph)
	state.contracts = checkContracts(scopes, state.graph, state.bindings)
	state.values = checkValues(scopes, state.graph, state.contracts)
	state.bodies = checkBodyLinks(scopes, state.graph, state.values, limits)
	state.status = append([]Status(nil), state.bodies.status...)
	for index, source := range sources {
		if source.Status == Failed {
			state.status[index] = Failed
		}
	}
	for _, finding := range state.identities {
		state.status[finding.scope] = Failed
	}
	closure := dependencyGraph{edges: state.graph.edges, status: state.status}
	propagateDependencyStatus(&closure)
	return state
}

func applyContentPhases(result *ModelResult, state contentState, index int) {
	setPhaseStatus(result, "dependencies", state.graph.status[index])
	setPhaseStatus(result, "imports_and_kinds", state.bindings.status[index])
	setPhaseStatus(result, "core_and_contracts", state.contracts.status[index])
	setPhaseStatus(result, "values_and_references", state.values.status[index])
	setPhaseStatus(result, "body_links", state.bodies.status[index])
	if result.Phases[2].Status != Failed {
		setPhaseStatus(result, "names_and_unicode", state.contracts.names[index])
	}
	for _, finding := range state.identities {
		if finding.scope == index {
			setPhaseStatus(result, "names_and_unicode", Failed)
		}
	}
	result.Status = state.status[index]
}

func contentFindings(state contentState) []identityFinding {
	var findings []identityFinding
	for _, group := range [][]identityFinding{state.identities, state.graph.findings, state.bindings.findings, state.contracts.findings, state.values.findings, state.bodies.findings} {
		findings = append(findings, group...)
	}
	return findings
}

// Include every colliding requirement candidate, never a preferred exact one.
// Supplied models that are unrelated to this selected model do not enter its
// independent content proof. DFS order is irrelevant to canonical ordering.
func selectedContentScope(selected int, scopes []sourceScope, sources []ModelResult) ([]sourceScope, []ModelResult) {
	byKey := map[string][]int{}
	var unknownIdentities []int
	for index, scope := range scopes {
		if scope.modelID != "" {
			byKey[nameKey(scope.modelID)] = append(byKey[nameKey(scope.modelID)], index)
		} else {
			unknownIdentities = append(unknownIdentities, index)
		}
	}
	var chosenScopes []sourceScope
	var chosenSources []ModelResult
	seen := map[int]bool{}
	pending := append([]int(nil), byKey[nameKey(scopes[selected].modelID)]...)
	pending = append(pending, selected)
	for len(pending) > 0 {
		index := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[index] {
			continue
		}
		seen[index] = true
		scope := scopes[index]
		scope.definitions = append([]sourceDefinition(nil), scope.definitions...)
		chosenScopes = append(chosenScopes, scope)
		result := sources[index]
		result.Phases = append([]Phase(nil), result.Phases...)
		chosenSources = append(chosenSources, result)
		for _, requirement := range scope.requirements {
			matches := byKey[nameKey(requirement.id)]
			pending = append(pending, matches...)
			if len(matches) == 0 {
				pending = append(pending, unknownIdentities...)
			}
		}
	}
	return chosenScopes, chosenSources
}
