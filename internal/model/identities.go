package model

import (
	"fmt"
	"sort"
)

// sourceScope is acquisition state, not validated IR. Identity checks must see
// every declaration before resolution; a map must not silently choose one of
// several definitions at the same address. Invalid/incomplete scopes are never
// exposed as compiled Definitions.
type sourceScope struct {
	origin            string
	modelID           string
	manifest          Entry
	version           Value
	versionDeclared   bool
	requirements      []modelRequirement
	manifestReady     bool
	inventoryComplete bool
	definitions       []sourceDefinition
	lock              sourceLock
}

type sourceLock struct {
	entry            Entry
	present, decoded bool
	value            Value
}

type modelRequirement struct {
	id      string
	version Value
}

type identityFinding struct {
	scope      int
	diagnostic Diagnostic
}

// checkScopeIdentities rejects colliding names and shared ownership without
// choosing a winning model. Global names include only populated Packages and
// their prefixes; empty trees still receive each model's local name checks.
func checkScopeIdentities(scopes []sourceScope) []identityFinding {
	models := map[string][]int{}
	packages := map[string][]int{}
	globalNames := map[string]map[string]map[int]bool{}
	for index, scope := range scopes {
		if scope.modelID != "" {
			key := nameKey(scope.modelID)
			models[key] = append(models[key], index)
		}
		seen := map[string]bool{}
		for _, definition := range scope.definitions {
			pkg := definition.identity.Package
			if !seen[pkg] {
				packages[pkg] = append(packages[pkg], index)
				seen[pkg] = true
				for prefix := pkg; prefix != ""; prefix = parent(prefix) {
					key := nameKey(prefix)
					if globalNames[key] == nil {
						globalNames[key] = map[string]map[int]bool{}
					}
					if globalNames[key][prefix] == nil {
						globalNames[key][prefix] = map[int]bool{}
					}
					globalNames[key][prefix][index] = true
				}
			}
		}
	}
	var findings []identityFinding
	for _, owners := range models {
		if len(owners) > 1 {
			for _, index := range owners {
				scope := scopes[index]
				findings = append(findings, identityFinding{scope: index, diagnostic: Diagnostic{
					Code: "name.model_collision", Message: fmt.Sprintf("Model ID %q collides with another supplied model ID.", scope.modelID),
					Source: source(scope.origin, scope.manifest),
				}})
			}
		}
	}
	for _, spellings := range globalNames {
		if len(spellings) < 2 {
			continue
		}
		owners := map[int]bool{}
		var paths []string
		for path, indices := range spellings {
			paths = append(paths, path)
			for index := range indices {
				owners[index] = true
			}
		}
		if len(owners) < 2 {
			continue // Already diagnosed by this model's local tree checks.
		}
		sort.Strings(paths)
		for index := range owners {
			scope := scopes[index]
			findings = append(findings, identityFinding{scope: index, diagnostic: Diagnostic{
				Code: "name.package_collision", Message: fmt.Sprintf("Package spellings %q collide across supplied models.", paths),
				Source: source(scope.origin, scope.manifest),
			}})
		}
	}
	for pkg, owners := range packages {
		if len(owners) > 1 {
			for _, index := range owners {
				scope := scopes[index]
				findings = append(findings, identityFinding{scope: index, diagnostic: Diagnostic{
					Code: "name.package_ownership", Message: fmt.Sprintf("Populated Package %q is claimed by more than one supplied model.", pkg),
					Source: source(scope.origin, scope.manifest),
				}})
			}
		}
	}
	return findings
}

type sourceDefinition struct {
	identity   Identity
	file       Entry
	envelope   Value
	kindTarget *Identity
	references referenceContext
}

func (c *sourceCompiler) collectDefinition(value Value, file Entry, root string) {
	id, present := lookup(value, "id")
	if !present || id.kind != StringValue || id.text == "" {
		return
	}
	pkg := parent(file.Path)
	if pkg == root {
		return // A root-level file has no Definition identity.
	}
	if root != "" {
		pkg = relativePackage(pkg, root)
	}
	c.scope.definitions = append(c.scope.definitions, sourceDefinition{
		identity: Identity{Package: pkg, ID: id.text}, file: file, envelope: value,
	})
}

// Keep all declarations, including colliding spellings: neither an exact
// spelling nor a presumed category may select a winning subset for resolution.
func (c *sourceCompiler) checkDefinitionIdentities() {
	declarations := append([]sourceDefinition(nil), c.scope.definitions...)
	sort.Slice(declarations, func(i, j int) bool {
		a, b := declarations[i], declarations[j]
		ak, bk := identityKey(a.identity), identityKey(b.identity)
		if ak.Package != bk.Package {
			return ak.Package < bk.Package
		}
		if ak.ID != bk.ID {
			return ak.ID < bk.ID
		}
		return a.file.Path < b.file.Path
	})
	for i, declaration := range declarations {
		if i > 0 && identityKey(declaration.identity) == identityKey(declarations[i-1].identity) {
			c.problem("name.definition_collision", "Definition identity is declared in more than one source file.", declaration.file, nil)
		}
	}
}

func identityKey(id Identity) Identity {
	return Identity{Package: nameKey(id.Package), ID: nameKey(id.ID)}
}
