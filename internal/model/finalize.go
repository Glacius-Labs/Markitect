package model

// Finalization separates three proofs: own/closure content, the entry lock's
// external bindings, and optional diagnostic coordinates. A consumer's lock is
// never an ingredient of a dependency's own canonical bytes or content proof.
func finalizeModels(result *Result, scopes []sourceScope, sources []ModelResult) {
	global := evaluateContent(scopes, sources, result.Limits)
	result.Coordinates = global.bodies.coordinates
	for _, finding := range contentFindings(global) {
		result.Diagnostics = append(result.Diagnostics, finding.diagnostic)
	}
	var rootScopes []sourceScope
	candidates := map[string]canonicalCandidate{}
	byIndex := map[int]canonicalCandidate{}
	coreReachable := modelsReachingCore(scopes, global.graph)
	for index := range scopes {
		selected, state, selectedIndex := scopes, global, index
		fresh := false
		if global.status[index] != Passed || !coreReachable[index] {
			selectedScope, selectedSources := selectedContentScope(index, scopes, sources)
			selected, state, selectedIndex = selectedScope, evaluateContent(selectedScope, selectedSources, result.Limits), 0
			fresh = true
		}
		if index == 0 && global.status[0] != Passed {
			// Unknown supplied metadata/inventories may still supply a requested
			// identity. Do not turn the entry evaluation's conservative unknown
			// into a missing-target error by prematurely narrowing that catalog.
			selected, state, selectedIndex, fresh = scopes, global, 0, false
		}
		applyContentPhases(&result.Models[index], state, selectedIndex)
		if fresh {
			for _, finding := range contentFindings(state) {
				result.Diagnostics = append(result.Diagnostics, finding.diagnostic)
			}
		}
		if index == 0 {
			rootScopes = selected
		}
		if index > 0 {
			// A supplied model still requires a published version even when
			// independently checked as the selected scope's first element.
			if scopes[index].manifestReady && (!scopes[index].versionDeclared || scopes[index].version.kind != StringValue || scopes[index].version.text == "") {
				result.Models[index].Status = Failed
				setPhaseStatus(&result.Models[index], "dependencies", Failed)
			}
			setPhaseStatus(&result.Models[index], "lock", Passed)
		}
		status := result.Models[index].Status
		if status == Passed {
			candidate, ok := projectCanonical(selected[selectedIndex], state.contracts, state.values, state.bodies)
			if ok {
				candidates[scopes[index].modelID] = candidate
				byIndex[index] = candidate
			} else {
				status = NotCompleted
				result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: "canonical.incomplete", Message: "Checked content did not provide a complete private canonical projection.", Source: source(scopes[index].origin, scopes[index].manifest)})
			}
		}
		result.Models[index].Status = status
		setPhaseStatus(&result.Models[index], "canonical", status)
	}
	lock := checkRootLock(rootScopes, candidates)
	setPhaseStatus(&result.Models[0], "lock", lock.status)
	result.Models[0].Status = combineStatus(result.Models[0].Status, lock.status)
	setPhaseStatus(&result.Models[0], "canonical", result.Models[0].Status)
	for _, finding := range lock.findings {
		result.Diagnostics = append(result.Diagnostics, finding.diagnostic)
	}
	result.Status = Passed
	for index := range result.Models {
		model := &result.Models[index]
		if model.Status == Passed {
			candidate := byIndex[index]
			model.Packages = append([]Package(nil), scopes[index].packages...)
			model.Definitions, model.Canonical, model.Digest = candidate.definitions, candidate.bytes, candidate.digest
		}
		result.Status = combineStatus(result.Status, model.Status)
	}
	// Keep whole-supplied-set failures visible even when an unrelated model
	// has a valid independent closure. There is no partially successful Result.
	for _, status := range global.status {
		if status == Failed {
			result.Status = Failed
		}
	}
}

// A passed whole-input content proof can be reused for every model whose
// explicit closure contains the fixed Core. All names/visibility are already
// complete, so removing unrelated models cannot change a unique valid binding.
// This keeps the normal complete-input path to one content parse per source.
func modelsReachingCore(scopes []sourceScope, graph dependencyGraph) []bool {
	result := make([]bool, len(scopes))
	reverse := make([][]int, len(scopes))
	var queue []int
	for index, scope := range scopes {
		for _, definition := range scope.definitions {
			if definition.identity.Package == coreKind.Package {
				result[index] = true
				queue = append(queue, index)
				break
			}
		}
		for _, target := range graph.direct[index] {
			reverse[target] = append(reverse[target], index)
		}
	}
	for head := 0; head < len(queue); head++ {
		for _, dependent := range reverse[queue[head]] {
			if !result[dependent] {
				result[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}
	return result
}
