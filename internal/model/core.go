package model

// fixedCore specifies effective content, not source bytes or reference spelling.
// It is pinned to language-contract §3.1, including the decoded folded bodies.
type fixedCoreDefinition struct {
	body, valueType, min, max, target string
	fields, values                    []string
}

var fixedCore = map[string]fixedCoreDefinition{
	"Kind":       {body: "Declares a type. properties names the permitted fields of the spec of its definitions. The body of a type definition describes their shared meaning and applicable stipulations. implements names interface Kinds; the type declares every Property of each in properties.", fields: []string{"properties", "implements"}},
	"Property":   {body: "Declares a field contract with value type and cardinality. The body describes the meaning of its values or relationships. For value type object, properties names the permitted fields of the object value.", fields: []string{"name", "valueType", "minCount", "maxCount", "values", "target", "properties"}},
	"name":       {body: "Optional explicitly deviating non-empty field name. If omitted, the complete local ID of the Property applies as the field name. The effective field name must be unique within every contract that uses it.", valueType: "string", min: "0", max: "1"},
	"valueType":  {body: "The value type of this Property, understood by the Core.", valueType: "enum", min: "1", max: "1", values: []string{"string", "boolean", "integer", "number", "enum", "object", "reference"}},
	"minCount":   {body: "Non-negative minimum number of present values.", valueType: "integer", min: "1", max: "1"},
	"maxCount":   {body: "Maximum number of present values. The value -1 permits any number of values within the tool limits. Other values must be at least minCount. Exactly 1 denotes a single value; other maximum values denote a list.", valueType: "integer", min: "1", max: "1"},
	"values":     {body: "Permitted string values of an enum Property. For value type enum, a non-empty list of distinct values is required. For other value types, this entry is not permitted.", valueType: "string", min: "0", max: "-1"},
	"target":     {body: "The Kind required of the referenced definitions. Definitions of a Kind that implements the required Kind, directly or through other interfaces, also satisfy it. For value type reference, this entry is required. For other value types, it is not permitted.", valueType: "reference", min: "0", max: "1", target: "Kind"},
	"properties": {body: "References the permitted field contracts of the object described here. The effective field names must be unique within this contract.", valueType: "reference", min: "0", max: "-1", target: "Property"},
	"implements": {body: "References the interface Kinds a Kind implements. The Kind must declare every Property of each interface in its own properties, and its definitions satisfy references that require the interface, directly or indirectly. The implements graph must be acyclic.", valueType: "reference", min: "0", max: "-1", target: "Kind"},
}

func (c *contractChecker) checkCore() Status {
	status := Passed
	complete := true
	counts := map[string]int{}
	for model, scope := range c.scopes {
		complete = complete && scope.inventoryComplete
		for index, definition := range scope.definitions {
			if nameKey(definition.identity.Package) != nameKey(coreKind.Package) {
				continue
			}
			expected, present := fixedCore[definition.identity.ID]
			counts[definition.identity.ID]++
			a := definitionAddress{model, index}
			if !present || definition.identity.Package != coreKind.Package || len(c.index[identityKey(definition.identity)]) != 1 {
				c.finding(a, "core.definition_set", "The fixed Core Package must contain exactly its ten prescribed Definitions without replacement or duplication.", Value{})
				status = Failed
				continue
			}
			body, _ := lookup(definition.envelope, "body")
			_, title := lookup(definition.envelope, "title")
			if body.kind != StringValue || body.text != expected.body || title {
				c.finding(a, "core.content", "The fixed Core body and all other content must match the prescribed effective self-description.", body)
				status = Failed
			}
			contract := c.result.definitions[definition.identity]
			if contract == nil || contract.status != Passed {
				if contract != nil {
					status = combineStatus(status, contract.status)
				} else {
					status = combineStatus(status, NotCompleted)
				}
				continue
			}
			matches := false
			if expected.valueType == "" {
				matches = contract.category == "kind" && sameCoreFields(contract.fields, expected.fields) && len(contract.interfaces) == 0
			} else if p := contract.property; p != nil {
				matches = contract.category == "property" && p.name == definition.identity.ID && p.valueType == expected.valueType &&
					p.min != nil && p.min.String() == expected.min && p.max != nil && p.max.String() == expected.max &&
					sameStringSet(p.values, expected.values) && len(p.fields) == 0
				if expected.target != "" {
					matches = matches && p.target != nil && *p.target == (Identity{coreKind.Package, expected.target})
				} else {
					matches = matches && p.target == nil
				}
			}
			if !matches {
				c.finding(a, "core.content", "The fixed Core contract must match the prescribed effective self-description.", Value{})
				status = Failed
			}
		}
	}
	missing := false
	for id := range fixedCore {
		missing = missing || counts[id] == 0
	}
	if missing && complete {
		status = Failed
		for model, scope := range c.scopes {
			c.result.findings = append(c.result.findings, identityFinding{scope: model, diagnostic: Diagnostic{
				Code: "core.missing", Message: "The supplied model set is missing prescribed Definitions from markitect/core.", Source: source(scope.origin, scope.manifest),
			}})
		}
	}
	if !complete {
		status = combineStatus(status, NotCompleted)
	}
	return status
}

func sameCoreFields(actual []Identity, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	wanted := make(map[Identity]bool, len(expected))
	for _, id := range expected {
		wanted[Identity{coreKind.Package, id}] = true
	}
	for _, id := range actual {
		if !wanted[id] {
			return false
		}
		delete(wanted, id)
	}
	return len(wanted) == 0
}

func sameStringSet(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	wanted := make(map[string]bool, len(expected))
	for _, value := range expected {
		wanted[value] = true
	}
	for _, value := range actual {
		if !wanted[value] {
			return false
		}
		delete(wanted, value)
	}
	return len(wanted) == 0
}
