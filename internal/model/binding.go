package model

import (
	"fmt"
	"strings"
)

type definitionAddress struct{ model, definition int }

type packageBinding struct {
	name     string
	owner    int
	members  []definitionAddress
	complete bool
}

type importAlias struct {
	name string
	pkg  *packageBinding
}

// referenceContext is file-local. It grants no privileges based on a target's
// category, title, physical location, or the spelling of an expected Kind.
type referenceContext struct {
	packages []*packageBinding
	aliases  []importAlias
	declared []string
	complete bool
}

type authorBindings struct {
	status   []Status
	findings []identityFinding
}

func bindAuthorReferences(scopes []sourceScope, graph dependencyGraph) authorBindings {
	result := authorBindings{status: make([]Status, len(scopes))}
	packages := map[string][]*packageBinding{}
	catalogComplete := true
	for model, scope := range scopes {
		catalogComplete = catalogComplete && scope.inventoryComplete
		local := map[string]*packageBinding{}
		for index, definition := range scope.definitions {
			name := definition.identity.Package
			pkg := local[name]
			if pkg == nil {
				pkg = &packageBinding{name: name, owner: model, complete: scope.inventoryComplete}
				local[name] = pkg
				packages[nameKey(name)] = append(packages[nameKey(name)], pkg)
			}
			pkg.members = append(pkg.members, definitionAddress{model: model, definition: index})
		}
	}
	for model := range scopes {
		status := graph.status[model]
		if !scopes[model].inventoryComplete || !catalogComplete {
			status = combineStatus(status, NotCompleted)
		}
		for index := range scopes[model].definitions {
			definition := &scopes[model].definitions[index]
			definition.kindTarget = nil
			context := referenceContext{complete: catalogComplete && scopes[model].inventoryComplete}
			// The authored own-Package/import declarations are known even when an
			// unrelated inventory is incomplete. Keep them separate from candidates.
			context.declared = append(context.declared, definition.identity.Package)
			finding := func(code, message string, position *Position) {
				status = Failed
				result.findings = append(result.findings, identityFinding{scope: model, diagnostic: Diagnostic{
					Code: code, Message: message, Source: source(scopes[model].origin, definition.file), Position: position,
				}})
			}
			own := packages[nameKey(definition.identity.Package)]
			if len(own) != 1 {
				context.complete = false
				if len(own) > 1 {
					finding("binding.package_ambiguous", "The Definition's own Package has more than one supplied owner or spelling.", nil)
				}
			} else {
				context.packages = append(context.packages, own[0])
			}
			imports, present := lookup(definition.envelope, "imports")
			if present && imports.kind != ListValue {
				context.complete = false
				status = combineStatus(status, NotCompleted)
			}
			for _, item := range imports.items {
				written, hasPackage := lookup(item, "package")
				alias, hasAlias := lookup(item, "as")
				var bound *packageBinding
				if item.kind != ObjectValue || !hasPackage || written.kind != StringValue || !validPackageName(written.text) {
					context.complete = false
					status = combineStatus(status, NotCompleted)
				} else {
					context.declared = append(context.declared, written.text)
					candidates := packages[nameKey(written.text)]
					switch len(candidates) {
					case 0:
						context.complete = false
						if catalogComplete {
							finding("binding.package_missing", fmt.Sprintf("Imported Package %q has no supplied Definitions.", written.text), positionPointer(written.position))
						} else {
							status = combineStatus(status, NotCompleted)
						}
					case 1:
						candidate := candidates[0]
						if candidate.name != written.text {
							context.complete = false
							finding("binding.package_spelling", fmt.Sprintf("Package %q must use declared spelling %q.", written.text, candidate.name), positionPointer(written.position))
						} else if candidate.owner != model && !containsModel(graph.direct[model], candidate.owner) {
							context.complete = false
							owner := scopes[candidate.owner]
							if !scopes[model].manifestReady || owner.modelID == "" || declaresModel(scopes[model], owner.modelID) {
								status = combineStatus(status, NotCompleted)
							} else {
								finding("binding.model_visibility", fmt.Sprintf("Package %q requires an exact direct model dependency.", written.text), positionPointer(written.position))
							}
						} else {
							bound = candidate
							context.packages = append(context.packages, candidate)
						}
					default:
						context.complete = false
						finding("binding.package_ambiguous", fmt.Sprintf("Imported Package %q has colliding supplied owners or spellings.", written.text), positionPointer(written.position))
					}
				}
				if hasAlias && alias.kind == StringValue {
					context.aliases = append(context.aliases, importAlias{name: alias.text, pkg: bound})
				}
			}
			kind, present := lookup(definition.envelope, "kind")
			if !present || kind.kind != StringValue {
				status = combineStatus(status, NotCompleted)
				continue
			}
			target, resolved, code, message := resolveAuthorReference(context, kind.text, scopes)
			status = combineStatus(status, resolved)
			if code != "" {
				finding(code, message, positionPointer(kind.position))
			}
			if resolved == Passed && context.complete {
				definition.kindTarget = &target
			}
		}
		result.status[model] = status
	}
	return result
}

// resolveAuthorReference first gathers every matching source declaration. An
// exact-spelled subset or an expected target type never removes ambiguity.
// The returned address is private acquisition state, not a validated type/IR.
func resolveAuthorReference(context referenceContext, written string, scopes []sourceScope) (Identity, Status, string, string) {
	if !isNFC(written) || written == "" {
		return Identity{}, Failed, "binding.reference_syntax", "References must be nonempty and already NFC."
	}
	selected := context.packages
	id := written
	qualifierExact := true
	complete := context.complete
	if strings.Contains(written, "/") {
		separator := strings.LastIndexByte(written, '/')
		pkg := written[:separator]
		id = written[separator+1:]
		selected = nil
		for _, candidate := range context.packages {
			if nameKey(candidate.name) == nameKey(pkg) {
				selected = append(selected, candidate)
				qualifierExact = qualifierExact && candidate.name == pkg
			}
		}
		if len(selected) == 0 {
			for _, declared := range context.declared {
				if nameKey(declared) == nameKey(pkg) {
					return Identity{}, NotCompleted, "", ""
				}
			}
			return Identity{}, Failed, "binding.package_not_visible", fmt.Sprintf("Package %q is not the own Package or an explicit file-local import.", pkg)
		}
		complete = true
	} else if strings.Contains(written, ".") {
		alias, local, _ := strings.Cut(written, ".")
		id = local
		selected = nil
		var aliases []importAlias
		for _, candidate := range context.aliases {
			if nameKey(candidate.name) == nameKey(alias) {
				aliases = append(aliases, candidate)
			}
		}
		if len(aliases) > 1 {
			return Identity{}, Failed, "binding.alias_ambiguous", fmt.Sprintf("Alias %q matches multiple import declarations.", alias)
		}
		if len(aliases) == 0 {
			return Identity{}, Failed, "binding.alias_missing", fmt.Sprintf("Alias %q is not declared in this file.", alias)
		}
		qualifierExact = aliases[0].name == alias
		if !qualifierExact {
			return Identity{}, Failed, "binding.reference_spelling", fmt.Sprintf("Alias %q must use declared spelling %q.", alias, aliases[0].name)
		}
		if aliases[0].pkg == nil {
			return Identity{}, NotCompleted, "", ""
		}
		selected = append(selected, aliases[0].pkg)
		complete = true
	}
	if !validLocalName(id) {
		return Identity{}, Failed, "binding.reference_syntax", "A reference must contain one complete local ID, without a field path or extra qualifier."
	}
	seen := map[definitionAddress]bool{}
	var candidates []definitionAddress
	for _, pkg := range selected {
		complete = complete && pkg.complete
		for _, member := range pkg.members {
			if nameKey(scopes[member.model].definitions[member.definition].identity.ID) == nameKey(id) && !seen[member] {
				candidates = append(candidates, member)
				seen[member] = true
			}
		}
	}
	if len(candidates) > 1 {
		return Identity{}, Failed, "binding.reference_ambiguous", fmt.Sprintf("Reference %q matches multiple visible declarations.", written)
	}
	if !complete {
		return Identity{}, NotCompleted, "", ""
	}
	if len(candidates) == 0 {
		return Identity{}, Failed, "binding.reference_missing", fmt.Sprintf("Reference %q has no target in the file-local visible Packages.", written)
	}
	member := candidates[0]
	target := scopes[member.model].definitions[member.definition].identity
	if !qualifierExact || target.ID != id {
		return Identity{}, Failed, "binding.reference_spelling", fmt.Sprintf("Reference %q does not use the exact declared Package/alias/ID spelling.", written)
	}
	return target, Passed, "", ""
}

func containsModel(models []int, target int) bool {
	for _, model := range models {
		if model == target {
			return true
		}
	}
	return false
}

// A declared but unbound edge is not proof of an omitted dependency. The graph
// phase owns spelling/version errors; incomplete target metadata stays unknown.
func declaresModel(scope sourceScope, id string) bool {
	for _, requirement := range scope.requirements {
		if nameKey(requirement.id) == nameKey(id) {
			return true
		}
	}
	return false
}

func combineStatus(a, b Status) Status {
	if a == Failed || b == Failed {
		return Failed
	}
	if a != Passed || b != Passed {
		return NotCompleted
	}
	return Passed
}
