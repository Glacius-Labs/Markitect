package model

import (
	"fmt"
	"math/big"
)

var coreKind = Identity{Package: "markitect/core", ID: "Kind"}
var coreProperty = Identity{Package: "markitect/core", ID: "Property"}

// These contracts are private checked acquisition state. Concrete values and
// the public Definition IR are not produced by this phase.
type propertyContract struct {
	name, valueType string
	min, max        *big.Int
	values          []string
	target          *Identity
	fields          []Identity
}

type definitionContract struct {
	category string
	kind     *Identity
	property *propertyContract
	// fields are the Kind's own declared Properties, which are its whole
	// field contract; interfaces its declared implements targets.
	fields     []Identity
	interfaces []Identity
	status     Status
}

// contractFields is the closed field contract that values of a Kind or an
// object Property must satisfy. Interfaces add nothing to it.
func (d *definitionContract) contractFields() []Identity {
	return d.fields
}

type contractResult struct {
	status, names []Status
	definitions   map[Identity]*definitionContract
	findings      []identityFinding
}

type contractChecker struct {
	scopes []sourceScope
	index  map[Identity][]definitionAddress
	kinds  map[definitionAddress]*Identity
	result contractResult
}

func checkContracts(scopes []sourceScope, graph dependencyGraph, bindings authorBindings) contractResult {
	c := contractChecker{scopes: scopes, index: map[Identity][]definitionAddress{}, kinds: map[definitionAddress]*Identity{}, result: contractResult{
		status: append([]Status(nil), bindings.status...), names: make([]Status, len(scopes)),
		definitions: map[Identity]*definitionContract{},
	}}
	for model, scope := range scopes {
		c.result.names[model] = Passed
		if !scope.inventoryComplete {
			c.result.names[model] = NotCompleted
		}
		for index, definition := range scope.definitions {
			key := identityKey(definition.identity)
			c.index[key] = append(c.index[key], definitionAddress{model, index})
		}
	}
	for model, scope := range scopes {
		for index, definition := range scope.definitions {
			address := definitionAddress{model, index}
			c.kinds[address] = definition.kindTarget
			if definition.kindTarget == nil {
				// Complete local declarations still support independent error
				// proofs beside another incomplete inventory. These candidates
				// do not complete the binding phase or become published IR.
				written, _ := lookup(definition.envelope, "kind")
				context := definition.references
				context.complete = context.declarationsBound
				if written.kind == StringValue {
					id, status, _, _ := resolveAuthorReference(context, written.text, scopes)
					if status == Passed {
						c.kinds[address] = &id
					}
				}
			}
		}
	}
	// Retain all candidates in the index. A unique raw spelling must never
	// select one of several declarations before category checking.
	for model, scope := range scopes {
		for index, definition := range scope.definitions {
			address := definitionAddress{model, index}
			contract := c.classify(address)
			if len(c.index[identityKey(definition.identity)]) == 1 {
				c.result.definitions[definition.identity] = contract
			}
		}
	}
	for model, scope := range scopes {
		for index, definition := range scope.definitions {
			contract := c.result.definitions[definition.identity]
			if contract == nil {
				c.result.status[model] = combineStatus(c.result.status[model], NotCompleted)
				continue
			}
			address := definitionAddress{model, index}
			spec, _ := lookup(definition.envelope, "spec")
			if spec.kind != ObjectValue {
				contract.status = combineStatus(contract.status, NotCompleted)
			} else {
				switch contract.category {
				case "kind":
					if !c.closed(address, spec, "properties", "implements") {
						contract.status = Failed
					}
					contract.fields = c.propertyList(address, spec, &contract.status)
					contract.interfaces = c.interfaceList(address, spec, &contract.status)
				case "property":
					contract.property = c.property(address, spec, &contract.status)
				}
			}
			c.result.status[model] = combineStatus(c.result.status[model], contract.status)
		}
	}
	c.checkFieldNames()
	c.checkObjectCycles()
	c.checkInterfaces()
	coreStatus := c.checkCore()
	for model := range scopes {
		c.result.status[model] = combineStatus(c.result.status[model], coreStatus)
		if coreStatus != Passed {
			c.result.names[model] = combineStatus(c.result.names[model], NotCompleted)
		}
	}
	// A failed or incomplete contract in a dependency prevents the requesting
	// model's contract closure from claiming completion.
	closure := dependencyGraph{edges: graph.edges, status: c.result.status}
	propagateDependencyStatus(&closure)
	return c.result
}

func (c *contractChecker) definition(a definitionAddress) sourceDefinition {
	return c.scopes[a.model].definitions[a.definition]
}

func (c *contractChecker) finding(a definitionAddress, code, message string, at Value) {
	definition := c.definition(a)
	if contract := c.result.definitions[definition.identity]; contract != nil {
		contract.status = Failed
	}
	c.result.status[a.model] = Failed
	c.result.findings = append(c.result.findings, identityFinding{scope: a.model, diagnostic: Diagnostic{
		Code: code, Message: message, Source: source(c.scopes[a.model].origin, definition.file), Position: positionPointer(at.position),
	}})
}

func (c *contractChecker) classify(a definitionAddress) *definitionContract {
	d := c.definition(a)
	contract := &definitionContract{status: Passed}
	kindID := c.kinds[a]
	contract.kind = kindID
	if kindID == nil {
		contract.status = NotCompleted
		c.result.names[a.model] = combineStatus(c.result.names[a.model], NotCompleted)
		return contract
	}
	kindTarget := c.kindOf(*kindID)
	if kindTarget == nil {
		contract.status = NotCompleted
		c.result.names[a.model] = combineStatus(c.result.names[a.model], NotCompleted)
		return contract
	}
	if *kindTarget != coreKind {
		written, _ := lookup(d.envelope, "kind")
		c.finding(a, "contract.kind_target", "kind must resolve to a Definition of the fixed Core Kind.", written)
		contract.status = Failed
		c.result.names[a.model] = combineStatus(c.result.names[a.model], NotCompleted)
		return contract
	}
	validName := false
	switch *kindID {
	case coreKind:
		contract.category = "kind"
		validName = nameWord(d.identity.ID, "pascal", false)
	case coreProperty:
		contract.category = "property"
		validName = validFieldName(d.identity.ID)
	default:
		contract.category = "concrete"
		validName = validFilename(d.identity.ID + ".yaml")
	}
	if !validName {
		id, _ := lookup(d.envelope, "id")
		c.finding(a, "name.category", fmt.Sprintf("ID %q does not follow its resolved %s naming convention.", d.identity.ID, contract.category), id)
		c.result.names[a.model] = Failed
		contract.status = Failed
	}
	return contract
}

func (c *contractChecker) kindOf(id Identity) *Identity {
	candidates := c.index[identityKey(id)]
	if len(candidates) != 1 || c.definition(candidates[0]).identity != id {
		return nil
	}
	return c.kinds[candidates[0]]
}

func (c *contractChecker) closed(a definitionAddress, value Value, allowed ...string) bool {
	valid := true
	for _, field := range value.fields {
		found := false
		for _, name := range allowed {
			found = found || field.Name == name
		}
		if !found {
			valid = false
			c.finding(a, "contract.unknown_field", fmt.Sprintf("Field %q is not permitted in this meta-contract.", field.Name), field.Value)
		}
	}
	return valid
}

func (c *contractChecker) reference(a definitionAddress, value Value, expected Identity, status *Status) *Identity {
	if value.kind != StringValue || !validAuthorReference(value.text) {
		c.finding(a, "contract.reference", "A contract reference must be a complete valid author reference string.", value)
		*status = Failed
		return nil
	}
	d := c.definition(a)
	context := d.references
	context.complete = context.declarationsBound
	id, bound, code, message := resolveAuthorReference(context, value.text, c.scopes)
	*status = combineStatus(*status, bound)
	if code != "" {
		c.finding(a, code, message, value)
	}
	if bound != Passed {
		return nil
	}
	kind := c.kindOf(id)
	if kind == nil {
		*status = combineStatus(*status, NotCompleted)
		return nil
	}
	if *kind != expected {
		c.finding(a, "contract.reference_target", fmt.Sprintf("Reference %q must target a Definition of %s/%s.", value.text, expected.Package, expected.ID), value)
		*status = Failed
		return nil
	}
	return &id
}

func (c *contractChecker) propertyList(a definitionAddress, spec Value, status *Status) []Identity {
	value, present := lookup(spec, "properties")
	if !present {
		return nil
	}
	if value.kind != ListValue {
		c.finding(a, "contract.properties_list", "properties must be a list of Property references.", value)
		*status = Failed
		return nil
	}
	seen := map[Identity]bool{}
	var fields []Identity
	for _, item := range value.items {
		id := c.reference(a, item, coreProperty, status)
		if id == nil {
			continue
		}
		if seen[*id] {
			c.finding(a, "contract.property_duplicate", "A field contract must not declare the same Property twice.", item)
			*status = Failed
		}
		seen[*id] = true
		fields = append(fields, *id)
	}
	return fields
}

// interfaceList checks the declared implements targets of a Kind: ordinary
// Kinds only, each named once (language contract §3.3).
func (c *contractChecker) interfaceList(a definitionAddress, spec Value, status *Status) []Identity {
	value, present := lookup(spec, "implements")
	if !present {
		return nil
	}
	if value.kind != ListValue {
		c.finding(a, "contract.implements_list", "implements must be a list of Kind references.", value)
		*status = Failed
		return nil
	}
	seen := map[Identity]bool{}
	var interfaces []Identity
	for _, item := range value.items {
		id := c.reference(a, item, coreKind, status)
		if id == nil {
			continue
		}
		if *id == coreKind || *id == coreProperty {
			c.finding(a, "contract.implements_meta", "The meta-Kinds Kind and Property cannot be implemented.", item)
			*status = Failed
			continue
		}
		if seen[*id] {
			c.finding(a, "contract.implements_duplicate", "A Kind must not name the same interface twice.", item)
			*status = Failed
			continue
		}
		seen[*id] = true
		interfaces = append(interfaces, *id)
	}
	return interfaces
}

type interfaceFrame struct {
	id   Identity
	next int
}

// checkInterfaces rejects implements cycles and checks that every Kind
// declares each Property of every interface it names. The walk is iterative
// and visits each Kind once.
func (c *contractChecker) checkInterfaces() {
	const (
		unvisited = iota
		active
		done
	)
	state := map[Identity]int{}
	for _, scope := range c.scopes {
		for _, d := range scope.definitions {
			root := c.result.definitions[d.identity]
			if root == nil || root.category != "kind" || state[d.identity] != unvisited {
				continue
			}
			stack := []interfaceFrame{{id: d.identity}}
			state[d.identity] = active
			for len(stack) > 0 {
				top := &stack[len(stack)-1]
				contract := c.result.definitions[top.id]
				if top.next < len(contract.interfaces) {
					target := contract.interfaces[top.next]
					top.next++
					next := c.result.definitions[target]
					switch {
					case next == nil || next.category != "kind":
						c.incomplete(top.id)
					case state[target] == active:
						c.interfaceCycle(stack, target)
					case state[target] == unvisited:
						state[target] = active
						stack = append(stack, interfaceFrame{id: target})
					}
					continue
				}
				c.conformance(top.id)
				state[top.id] = done
				stack = stack[:len(stack)-1]
			}
		}
	}
}

func (c *contractChecker) interfaceCycle(stack []interfaceFrame, target Identity) {
	start := len(stack) - 1
	for start > 0 && stack[start].id != target {
		start--
	}
	for _, f := range stack[start:] {
		if address, ok := c.address(f.id); ok {
			implements, _ := lookup(c.definitionSpec(address), "implements")
			c.finding(address, "contract.implements_cycle", "The implements graph of Kinds must be acyclic.", implements)
		}
	}
}

// conformance requires every Property of each interface in the Kind's own
// properties. An incomplete interface leaves conformance not completed.
func (c *contractChecker) conformance(id Identity) {
	contract := c.result.definitions[id]
	if len(contract.interfaces) == 0 {
		return
	}
	if contract.status != Passed {
		// An implementer whose own properties did not fully resolve cannot be
		// judged; an earlier finding or incompleteness already holds it.
		c.incomplete(id)
		return
	}
	own := map[Identity]bool{}
	for _, field := range contract.fields {
		own[field] = true
	}
	address, _ := c.address(id)
	implements, _ := lookup(c.definitionSpec(address), "implements")
	for _, target := range contract.interfaces {
		iface := c.result.definitions[target]
		if iface == nil || iface.category != "kind" || iface.status != Passed {
			c.incomplete(id)
			continue
		}
		for _, field := range iface.fields {
			if !own[field] {
				c.finding(address, "contract.implements_missing", fmt.Sprintf("Kind %s/%s implements %s/%s but does not declare its Property %s/%s.", id.Package, id.ID, target.Package, target.ID, field.Package, field.ID), implements)
			}
		}
	}
}

// incomplete marks a Kind's contract and its model scope as not completed.
func (c *contractChecker) incomplete(id Identity) {
	contract := c.result.definitions[id]
	contract.status = combineStatus(contract.status, NotCompleted)
	if address, ok := c.address(id); ok {
		c.result.status[address.model] = combineStatus(c.result.status[address.model], NotCompleted)
	}
}

func (c *contractChecker) address(id Identity) (definitionAddress, bool) {
	candidates := c.index[identityKey(id)]
	if len(candidates) != 1 || c.definition(candidates[0]).identity != id {
		return definitionAddress{}, false
	}
	return candidates[0], true
}

func (c *contractChecker) definitionSpec(a definitionAddress) Value {
	spec, _ := lookup(c.definition(a).envelope, "spec")
	return spec
}

func (c *contractChecker) property(a definitionAddress, spec Value, status *Status) *propertyContract {
	if !c.closed(a, spec, "name", "valueType", "minCount", "maxCount", "values", "target", "properties") {
		*status = Failed
	}
	d := c.definition(a)
	p := &propertyContract{name: d.identity.ID}
	if name, present := lookup(spec, "name"); present {
		if name.kind != StringValue || !validFieldName(name.text) {
			c.finding(a, "contract.field_name", "An explicit name must be a nonempty NFC camelCase field name.", name)
			*status = Failed
		} else {
			p.name = name.text
		}
	}
	typeValue, _ := lookup(spec, "valueType")
	if typeValue.kind == StringValue {
		p.valueType = typeValue.text
	}
	switch p.valueType {
	case "string", "boolean", "integer", "number", "enum", "object", "reference":
	default:
		c.finding(a, "contract.value_type", "valueType must select one of the seven fixed Core value types.", typeValue)
		*status = Failed
	}
	min, _ := lookup(spec, "minCount")
	max, _ := lookup(spec, "maxCount")
	if min.kind == IntegerValue {
		p.min, _ = new(big.Int).SetString(min.text, 10)
	}
	if max.kind == IntegerValue {
		p.max, _ = new(big.Int).SetString(max.text, 10)
	}
	if p.min == nil || p.min.Sign() < 0 || p.max == nil || (p.max.Cmp(big.NewInt(-1)) != 0 && (p.max.Sign() < 0 || (p.min != nil && p.max.Cmp(p.min) < 0))) {
		c.finding(a, "contract.cardinality", "minCount must be an integer >= 0; maxCount must be -1 or an integer >= minCount.", min)
		*status = Failed
	}
	for _, conditional := range []struct{ field, kind string }{{"values", "enum"}, {"target", "reference"}, {"properties", "object"}} {
		value, present := lookup(spec, conditional.field)
		if present && p.valueType != conditional.kind {
			c.finding(a, "contract.forbidden_field", fmt.Sprintf("%s is permitted only for type %s.", conditional.field, conditional.kind), value)
			*status = Failed
		}
	}
	switch p.valueType {
	case "enum":
		values, _ := lookup(spec, "values")
		if values.kind != ListValue || len(values.items) == 0 {
			c.finding(a, "contract.enum_values", "An enum requires a nonempty list of exact String values.", values)
			*status = Failed
		}
		seen := map[string]bool{}
		for _, item := range values.items {
			if item.kind != StringValue || seen[item.text] {
				c.finding(a, "contract.enum_values", "Enum values must be distinct exact Strings.", item)
				*status = Failed
			}
			seen[item.text] = true
			p.values = append(p.values, item.text)
		}
	case "reference":
		target, _ := lookup(spec, "target")
		p.target = c.reference(a, target, coreKind, status)
	case "object":
		p.fields = c.propertyList(a, spec, status)
	}
	return p
}

func (c *contractChecker) checkFieldNames() {
	for model, scope := range c.scopes {
		for index, d := range scope.definitions {
			contract := c.result.definitions[d.identity]
			if contract == nil {
				continue
			}
			fields := contract.contractFields()
			if contract.property != nil && contract.property.valueType == "object" {
				fields = contract.property.fields
			}
			seen := map[string]Identity{}
			for _, field := range fields {
				target := c.result.definitions[field]
				if target == nil || target.property == nil {
					c.result.status[model] = combineStatus(c.result.status[model], NotCompleted)
					continue
				}
				key := nameKey(target.property.name)
				if previous, exists := seen[key]; exists && previous != field {
					c.finding(definitionAddress{model, index}, "contract.field_collision", fmt.Sprintf("Properties %s/%s and %s/%s have colliding effective field names.", previous.Package, previous.ID, field.Package, field.ID), Value{})
				}
				seen[key] = field
			}
		}
	}
}
