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
	origin      string
	modelID     string
	manifest    Entry
	definitions []sourceDefinition
}

type identityFinding struct {
	scope      int
	diagnostic Diagnostic
}

// checkExactScopeIdentities rejects shared ownership without choosing a
// winning input model. Intermediate folders may be shared; only Packages with
// actual declarations claim ownership. Unicode-equivalent but differently
// spelled paths are intentionally left to the version-bound name phase.
func checkExactScopeIdentities(scopes []sourceScope) []identityFinding {
	models := map[string][]int{}
	packages := map[string][]int{}
	for index, scope := range scopes {
		if scope.modelID != "" {
			models[scope.modelID] = append(models[scope.modelID], index)
		}
		seen := map[string]bool{}
		for _, definition := range scope.definitions {
			pkg := definition.identity.Package
			if !seen[pkg] {
				packages[pkg] = append(packages[pkg], index)
				seen[pkg] = true
			}
		}
	}
	var findings []identityFinding
	for id, owners := range models {
		if len(owners) > 1 {
			for _, index := range owners {
				scope := scopes[index]
				findings = append(findings, identityFinding{scope: index, diagnostic: Diagnostic{
					Code: "name.model_collision", Message: fmt.Sprintf("Model ID %q is supplied more than once.", id),
					Source: source(scope.origin, scope.manifest),
				}})
			}
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
	identity Identity
	file     Entry
	envelope Value
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
		pkg = pkg[len(root)+1:]
	}
	c.scope.definitions = append(c.scope.definitions, sourceDefinition{
		identity: Identity{Package: pkg, ID: id.text}, file: file, envelope: value,
	})
}

// checkExactIdentities establishes the first, data-independent identity rule.
// Unicode-equivalent spellings and category-dependent names are checked by the
// separate version-bound name phase; this exact check never claims that phase
// complete. Keeping all declarations also preserves the later ambiguity facts.
func (c *sourceCompiler) checkExactIdentities() {
	declarations := append([]sourceDefinition(nil), c.scope.definitions...)
	sort.Slice(declarations, func(i, j int) bool {
		a, b := declarations[i], declarations[j]
		if a.identity.Package != b.identity.Package {
			return a.identity.Package < b.identity.Package
		}
		if a.identity.ID != b.identity.ID {
			return a.identity.ID < b.identity.ID
		}
		return a.file.Path < b.file.Path
	})
	for i, declaration := range declarations {
		if i > 0 && declaration.identity == declarations[i-1].identity {
			c.problem("name.definition_collision", "Definition identity is declared in more than one source file.", declaration.file, nil)
		}
	}
}
