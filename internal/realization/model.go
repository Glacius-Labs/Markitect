package realization

import (
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

// accountingModel consumes the compiler's authoritative inventory. It does not
// parse source, revalidate v1 names, or infer Packages from Definition addresses.
// Basic contradictions in the public supplied DTOs prevent a coverage claim.
func accountingModel(result model.Result) (map[string]bool, map[model.Identity]bool, bool, []Finding) {
	packages := map[string]bool{}
	definitions := map[model.Identity]bool{}
	unavailable := []Finding{{Code: "model.unavailable", Message: "A complete Passed compiled model set with one local model is required for owner and Definition resolution."}}
	if result.Status != model.Passed {
		return packages, definitions, false, unavailable
	}
	var local *model.ModelResult
	ids := map[string]bool{}
	var findings []Finding
	problem := func(message string, id model.Identity) {
		findings = append(findings, Finding{Code: "model.inconsistent", Definition: id, Message: message})
	}
	for i := range result.Models {
		scope := &result.Models[i]
		if scope.Status != model.Passed {
			return packages, definitions, false, unavailable
		}
		if scope.ID == "" || ids[scope.ID] {
			problem("Compiled model identities are empty or repeated.", model.Identity{})
		}
		ids[scope.ID] = true
		if scope.Local {
			if local != nil {
				problem("More than one supplied model is marked local.", model.Identity{})
			}
			local = scope
		}
		for _, definition := range scope.Definitions {
			id := definition.Identity
			if id.Package == "" || id.ID == "" || definitions[id] || definition.ModelID != scope.ID {
				problem("Compiled Definition identities or owning model facts are inconsistent.", id)
			}
			definitions[id] = true
		}
	}
	if local == nil {
		return packages, definitions, false, unavailable
	}
	for _, pkg := range local.Packages {
		if pkg.Name == "" || packages[pkg.Name] {
			problem("The authoritative local Package inventory has empty or repeated names.", model.Identity{})
		}
		packages[pkg.Name] = true
	}
	for _, definition := range local.Definitions {
		if !packages[definition.Identity.Package] {
			problem("A local Definition's Package is absent from the supplied authoritative inventory.", definition.Identity)
		}
	}
	return packages, definitions, len(findings) == 0, findings
}

func hasModelInconsistency(findings []Finding) bool {
	for _, finding := range findings {
		if finding.Code == "model.inconsistent" {
			return true
		}
	}
	return false
}

// modelInventory is the private structural projection shared by live compiler
// consumers and historical records. It carries no compiler verdict of its own.
type modelInventory struct {
	models         []claimModelBasis
	owners         map[string]bool
	definitions    map[model.Identity]bool
	local          map[model.Identity]bool
	ready, invalid bool
	findings       []Finding
}

func inventoryFromResult(result model.Result) modelInventory {
	owners, definitions, ready, findings := accountingModel(result)
	inventory := modelInventory{owners: owners, definitions: definitions, local: map[model.Identity]bool{}, ready: ready, invalid: result.Status == model.Failed || hasModelInconsistency(findings), findings: findings}
	if !ready {
		return inventory
	}
	for _, scope := range result.Models {
		binding := claimModelBasis{id: scope.ID, digest: scope.Digest, local: scope.Local}
		for _, pkg := range scope.Packages {
			binding.packages = append(binding.packages, pkg.Name)
		}
		for _, definition := range scope.Definitions {
			binding.definitions = append(binding.definitions, inventoryDefinition{definition.Identity, definition.ModelID})
			if scope.Local {
				inventory.local[definition.Identity] = true
			}
		}
		slices.Sort(binding.packages)
		slices.SortFunc(binding.definitions, func(a, b inventoryDefinition) int { return compareIdentity(a.identity, b.identity) })
		inventory.models = append(inventory.models, binding)
	}
	slices.SortFunc(inventory.models, func(a, b claimModelBasis) int { return strings.Compare(a.id, b.id) })
	return inventory
}

type inventoryDefinition struct {
	identity model.Identity
	modelID  string
}
