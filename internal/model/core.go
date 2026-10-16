package model

// fixedCore specifies effective content, not source bytes or reference spelling.
// It is pinned to language-contract §3.1, including the decoded folded bodies.
type fixedCoreDefinition struct {
	body, valueType, min, max, target string
	fields, values                    []string
}

var fixedCore = map[string]fixedCoreDefinition{
	"Kind":       {body: "Deklariert einen Typ. properties benennt die erlaubten Felder der spec seiner Definitionen. Der body einer Typdefinition beschreibt deren gemeinsame Bedeutung und geltende Festlegungen.", fields: []string{"properties"}},
	"Property":   {body: "Deklariert einen Feldvertrag mit Werttyp und Kardinalität. Der body beschreibt die Bedeutung seiner Werte oder Beziehungen. Bei type object benennt properties die erlaubten Felder des Objektwerts.", fields: []string{"name", "type", "minCount", "maxCount", "values", "target", "properties"}},
	"name":       {body: "Optionaler ausdrücklich abweichender nichtleerer Feldname. Ohne Angabe gilt die vollständige lokale ID der Property als Feldname. Der effektive Feldname muss innerhalb jedes verwendenden Vertrags eindeutig sein.", valueType: "string", min: "0", max: "1"},
	"type":       {body: "Der vom Core verstandene Werttyp dieser Property.", valueType: "enum", min: "1", max: "1", values: []string{"string", "boolean", "integer", "number", "enum", "object", "reference"}},
	"minCount":   {body: "Nichtnegative Mindestanzahl vorhandener Werte.", valueType: "integer", min: "1", max: "1"},
	"maxCount":   {body: "Höchstanzahl vorhandener Werte. Der Wert -1 erlaubt beliebig viele Werte innerhalb der Werkzeuggrenzen. Andere Werte müssen mindestens minCount entsprechen. Genau 1 bezeichnet einen Einzelwert; andere Höchstwerte eine Liste.", valueType: "integer", min: "1", max: "1"},
	"values":     {body: "Erlaubte Stringwerte einer enum-Property. Bei type enum ist eine nichtleere Liste eindeutiger Werte erforderlich. Bei anderen Werttypen ist diese Angabe nicht zulässig.", valueType: "string", min: "0", max: "-1"},
	"target":     {body: "Der verlangte Kind der referenzierten Definitionen. Bei type reference ist diese Angabe erforderlich. Bei anderen Werttypen ist sie nicht zulässig.", valueType: "reference", min: "0", max: "1", target: "Kind"},
	"properties": {body: "Referenziert die erlaubten Feldverträge des hier beschriebenen Objekts. Die effektiven Feldnamen müssen innerhalb dieses Vertrags eindeutig sein.", valueType: "reference", min: "0", max: "-1", target: "Property"},
}

func (c *contractCompiler) checkCore() Status {
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
				c.finding(a, "core.definition_set", "The fixed Core Package must contain exactly its nine prescribed Definitions without replacement or duplication.", Value{})
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
				matches = contract.category == "kind" && sameCoreFields(contract.fields, expected.fields)
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
