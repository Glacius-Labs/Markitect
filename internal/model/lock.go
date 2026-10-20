package model

import (
	"fmt"
	"strings"
)

func (c *sourceChecker) acquireLock(entries map[string]Entry) {
	path := joinPath(parent(c.model.ManifestPath), "model-lock.yaml")
	c.scope.lock = sourceLock{entry: Entry{Path: path}, status: NotCompleted}
	entry, present := entries[path]
	if !present {
		return
	}
	c.scope.lock.entry, c.scope.lock.present = entry, true
	if entry.Kind != Regular {
		c.problem("lock.not_regular", "The selected lock must be a regular file.", entry, nil)
		c.scope.lock.status = Failed
		return
	}
	start := len(c.diagnostics)
	c.scope.lock.value, c.scope.lock.decoded = c.decode(entry)
	c.scope.lock.status = Passed
	for _, diagnostic := range c.diagnostics[start:] {
		if strings.HasPrefix(diagnostic.Code, "limit.") {
			c.scope.lock.status = combineStatus(c.scope.lock.status, NotCompleted)
		} else {
			c.scope.lock.status = Failed
		}
	}
}

type lockResult struct {
	status   Status
	findings []identityFinding
}
type lockedModel struct {
	id              string
	version, digest Value
}

// Only the entry model's sidecar is an authoritative binding source. Dependency
// sidecars retain their loader diagnostics but cannot replace or extend it.
func checkRootLock(scopes []sourceScope, candidates map[string]canonicalCandidate) lockResult {
	result := lockResult{status: Passed}
	root := scopes[0]
	add := func(code, message string, value Value) {
		result.status = Failed
		result.findings = append(result.findings, identityFinding{scope: 0, diagnostic: Diagnostic{Code: code, Message: message, Source: source(root.origin, root.lock.entry), Position: positionPointer(value.position)}})
	}
	if !root.lock.present {
		if root.manifestReady {
			add("lock.missing", "The entry manifest requires an explicit sibling model-lock.yaml, including an empty models table when no external models are reachable.", Value{})
		} else {
			result.status = NotCompleted
		}
		return result
	}
	result.status = root.lock.status
	if !root.lock.decoded {
		return result
	}
	closed := func(value Value, names []string, path string) bool {
		if value.kind != ObjectValue {
			add("lock.object", path+" must be a mapping.", value)
			return false
		}
		allowed := map[string]bool{}
		for _, name := range names {
			allowed[name] = true
		}
		present := map[string]bool{}
		for _, field := range value.fields {
			if !allowed[field.Name] {
				add("lock.field", fmt.Sprintf("%s has unknown or wrongly spelled field %q.", path, field.Name), field.Value)
			}
			present[field.Name] = true
		}
		for _, name := range names {
			if !present[name] {
				add("lock.required", fmt.Sprintf("%s requires field %q.", path, name), value)
			}
		}
		return true
	}
	if !closed(root.lock.value, []string{"formatVersion", "models"}, "lock") {
		return result
	}
	format, _ := lookup(root.lock.value, "formatVersion")
	if format.kind != IntegerValue || format.text != "1" {
		add("lock.format", "Lock formatVersion must be the integer 1.", format)
	}
	models, _ := lookup(root.lock.value, "models")
	if models.kind != ObjectValue {
		add("lock.models", "Lock models must be a mapping.", models)
		return result
	}
	var bindings []lockedModel
	seen := map[string]bool{}
	for _, field := range models.fields {
		if !validModelLabel(field.Name, false) {
			add("lock.model_id", fmt.Sprintf("Locked model ID %q must use the exact NFC lowercase label grammar.", field.Name), field.Value)
		}
		key := nameKey(field.Name)
		if seen[key] {
			add("lock.model_collision", fmt.Sprintf("Locked model ID %q collides with another lock entry.", field.Name), field.Value)
		}
		seen[key] = true
		if !closed(field.Value, []string{"version", "digest"}, "locked model "+field.Name) {
			continue
		}
		version, _ := lookup(field.Value, "version")
		digest, _ := lookup(field.Value, "digest")
		if version.kind != StringValue || !validModelLabel(version.text, true) {
			add("lock.version", "Locked version must be an exact nonempty NFC version label.", version)
		}
		if digest.kind != StringValue || !validContentDigest(digest.text) {
			add("lock.digest", "Locked digest must be sha256: followed by exactly 64 lowercase hexadecimal digits.", digest)
		}
		bindings = append(bindings, lockedModel{id: field.Name, version: version, digest: digest})
	}
	// Manifest closure is a different proof from definition acquisition. A
	// refused Definition cannot erase a known version or membership mismatch.
	metadata := append([]sourceScope(nil), scopes...)
	allIdentitiesKnown := true
	for index := range metadata {
		metadata[index].inventoryComplete = metadata[index].manifestReady
		allIdentitiesKnown = allIdentitiesKnown && metadata[index].modelID != ""
	}
	graph := bindDependencies(metadata)
	complete := graph.status[0] == Passed && allIdentitiesKnown
	expected := map[string]bool{}
	versions := map[string]map[string]bool{}
	visited := map[int]bool{}
	pending := []int{0}
	for len(pending) > 0 {
		index := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if visited[index] {
			continue
		}
		visited[index] = true
		if index != 0 {
			expected[scopes[index].modelID] = true
		}
		if scopes[index].manifestReady {
			for _, requirement := range scopes[index].requirements {
				if nameKey(requirement.id) == nameKey(root.modelID) {
					continue
				}
				// Every known direct requirement needs a lock entry even if its
				// supplied target manifest is still unavailable.
				expected[requirement.id] = true
				if versions[requirement.id] == nil {
					versions[requirement.id] = map[string]bool{}
				}
				versions[requirement.id][requirement.version.text] = true
			}
		}
		pending = append(pending, graph.direct[index]...)
	}
	bound := map[string]bool{}
	expectedByKey := map[string][]string{}
	for id := range expected {
		expectedByKey[nameKey(id)] = append(expectedByKey[nameKey(id)], id)
	}
	provided := map[string][]sourceScope{}
	for _, scope := range scopes {
		if scope.manifestReady {
			provided[nameKey(scope.modelID)] = append(provided[nameKey(scope.modelID)], scope)
		}
	}
	for _, field := range models.fields {
		bound[field.Name] = true
		if root.modelID != "" && nameKey(field.Name) == nameKey(root.modelID) {
			add("lock.extra", "The local entry model must not be repeated as an external lock entry.", field.Value)
		} else if complete && !expected[field.Name] {
			add("lock.extra", fmt.Sprintf("Lock entry %q is not an exactly written member of the reachable external model closure.", field.Name), field.Value)
		}
	}
	for _, binding := range bindings {
		if nameKey(binding.id) == nameKey(root.modelID) {
			continue
		}
		known := expectedByKey[nameKey(binding.id)]
		if len(known) == 0 {
			// A supplied unrelated model is not evidence that this row is
			// reachable. An unknown transitive closure cannot prove an extra
			// row, nor a semantic mismatch against that unrelated candidate.
			continue
		}
		for _, id := range known {
			if binding.id != id {
				add("lock.model_spelling", fmt.Sprintf("Locked model %q must use the known reachable spelling %q.", binding.id, id), binding.version)
			}
		}
		if binding.version.kind == StringValue {
			for version := range versions[binding.id] {
				if binding.version.text != version {
					add("lock.version_mismatch", fmt.Sprintf("Locked model %q must match the known exact manifest requirement %q.", binding.id, version), binding.version)
				}
			}
		}
		for _, scope := range provided[nameKey(binding.id)] {
			if !expected[scope.modelID] {
				continue
			}
			if scope.modelID != binding.id {
				add("lock.model_spelling", fmt.Sprintf("Locked model %q must use declared spelling %q.", binding.id, scope.modelID), binding.version)
			}
			if scope.modelID == binding.id && scope.versionDeclared && binding.version.kind == StringValue && binding.version.text != scope.version.text {
				add("lock.version_mismatch", fmt.Sprintf("Locked model %q must use the supplied exact version %q.", binding.id, scope.version.text), binding.version)
			}
		}
		candidate, contentComplete := candidates[binding.id]
		if !contentComplete {
			result.status = combineStatus(result.status, NotCompleted)
			continue
		}
		if binding.digest.kind == StringValue && validContentDigest(binding.digest.text) && binding.digest.text != candidate.digest {
			add("lock.digest_mismatch", fmt.Sprintf("Locked model %q does not match its computed canonical content digest.", binding.id), binding.digest)
		}
	}
	for id := range expected {
		if !bound[id] {
			add("lock.missing_model", fmt.Sprintf("The lock is missing known reachable external model %q.", id), models)
		}
	}
	if !complete {
		result.status = combineStatus(result.status, NotCompleted)
	}

	return result
}

func validContentDigest(text string) bool {
	if len(text) != 71 || !strings.HasPrefix(text, "sha256:") {
		return false
	}
	for _, char := range text[7:] {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}
