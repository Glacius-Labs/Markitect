package model

import (
	"fmt"
	"math/big"
	"sort"
)

// Typed specs are private candidates. No public Definition or canonical
// content is emitted before all remaining content phases exist.
type valueResult struct {
	status   []Status
	specs    map[Identity]Value
	findings []identityFinding
}

type valueCompiler struct {
	scopes           []sourceScope
	contracts        contractResult
	result           valueResult
	propertyComplete map[*propertyContract]bool
}

type valueJob struct {
	mode     string
	input    Value
	output   *Value
	fields   []Identity
	complete bool
	property *propertyContract
	path     string
}

func checkValues(scopes []sourceScope, graph dependencyGraph, contracts contractResult) valueResult {
	c := valueCompiler{scopes: scopes, contracts: contracts, propertyComplete: map[*propertyContract]bool{}, result: valueResult{
		status: append([]Status(nil), contracts.status...), specs: map[Identity]Value{},
	}}
	for _, definition := range contracts.definitions {
		if definition.property != nil {
			c.propertyComplete[definition.property] = definition.status == Passed
		}
	}
	for model, scope := range scopes {
		for index, definition := range scope.definitions {
			contract := contracts.definitions[definition.identity]
			if contract == nil || contract.category == "" {
				c.result.status[model] = combineStatus(c.result.status[model], NotCompleted)
				continue
			}
			if contract.category != "concrete" {
				if contract.status == Passed {
					c.result.specs[definition.identity] = typedMetaSpec(contract)
				}
				continue
			}
			kind := contracts.definitions[*contract.kind]
			if kind == nil {
				c.result.status[model] = combineStatus(c.result.status[model], NotCompleted)
				continue
			}
			spec, _ := lookup(definition.envelope, "spec")
			value, status := c.spec(definitionAddress{model, index}, spec, kind)
			c.result.status[model] = combineStatus(c.result.status[model], status)
			if status == Passed && contract.status == Passed {
				c.result.specs[definition.identity] = value
			}
		}
	}
	closure := dependencyGraph{edges: graph.edges, status: c.result.status}
	propagateDependencyStatus(&closure)
	for model, scope := range scopes {
		if c.result.status[model] != Passed {
			for _, definition := range scope.definitions {
				delete(c.result.specs, definition.identity)
			}
		}
	}
	return c.result
}

func (c *valueCompiler) finding(a definitionAddress, code, path, message string, value Value) {
	definition := c.scopes[a.model].definitions[a.definition]
	c.result.findings = append(c.result.findings, identityFinding{scope: a.model, diagnostic: Diagnostic{
		Code: code, Message: path + ": " + message, Source: source(c.scopes[a.model].origin, definition.file), Position: positionPointer(value.position),
	}})
}

// Jobs traverse actual finite values, never unfold an optional schema or follow
// a reference's content. Even a caller-configured large depth uses no Go recursion.
func (c *valueCompiler) spec(a definitionAddress, input Value, kind *definitionContract) (Value, Status) {
	var output Value
	status := kind.status
	jobs := []valueJob{{mode: "object", input: input, output: &output, fields: kind.fields, complete: kind.status == Passed, path: "spec"}}
	problem := func(job valueJob, code, message string) {
		status = Failed
		c.finding(a, code, job.path, message, job.input)
	}
	for len(jobs) > 0 {
		last := len(jobs) - 1
		job := jobs[last]
		jobs = jobs[:last]
		switch job.mode {
		case "sort":
			sort.Slice(job.output.fields, func(i, j int) bool { return job.output.fields[i].Name < job.output.fields[j].Name })
		case "object":
			if job.input.kind != ObjectValue {
				problem(job, "value.type", "expected a closed object.")
				continue
			}
			schema := map[string][]*propertyContract{}
			complete := job.complete
			for _, id := range job.fields {
				field := c.contracts.definitions[id]
				if field == nil || field.property == nil || field.status == Failed {
					complete = false
					status = combineStatus(status, NotCompleted)
					continue
				}
				status = combineStatus(status, field.status)
				complete = complete && field.status == Passed
				schema[nameKey(field.property.name)] = append(schema[nameKey(field.property.name)], field.property)
			}
			*job.output = Value{kind: ObjectValue, fields: make([]Field, 0, len(job.input.fields)+len(job.fields)), position: job.input.position}
			jobs = append(jobs, valueJob{mode: "sort", output: job.output})
			present := map[string]bool{}
			for _, field := range job.input.fields {
				key := nameKey(field.Name)
				candidates := schema[key]
				fieldJob := valueJob{input: field.Value, path: job.path + "." + field.Name}
				if len(candidates) == 0 {
					if complete {
						problem(fieldJob, "value.unknown_field", "field is not declared by this contract.")
					} else {
						status = combineStatus(status, NotCompleted)
					}
					continue
				}
				if len(candidates) > 1 {
					problem(fieldJob, "value.ambiguous_field", "more than one declared Property supplies this field name.")
					continue
				}
				p := candidates[0]
				present[key] = true
				if field.Name != p.name {
					problem(fieldJob, "value.field_spelling", fmt.Sprintf("use the exact declared field spelling %q.", p.name))
				}
				job.output.fields = append(job.output.fields, Field{Name: p.name})
				fieldJob.mode, fieldJob.property, fieldJob.output = "field", p, &job.output.fields[len(job.output.fields)-1].Value
				jobs = append(jobs, fieldJob)
			}
			for _, id := range job.fields {
				field := c.contracts.definitions[id]
				if field == nil || field.property == nil || field.status == Failed {
					continue
				}
				p := field.property
				key := nameKey(p.name)
				if present[key] || len(schema[key]) != 1 {
					continue
				}
				present[key] = true
				if p.min == nil || p.max == nil {
					status = combineStatus(status, NotCompleted)
					continue
				}
				if p.min.Sign() > 0 {
					problem(valueJob{input: job.input, path: job.path + "." + p.name}, "value.cardinality", "a required field is absent.")
				} else if field.status == Passed && p.max.Cmp(big.NewInt(1)) != 0 {
					// Default only a permitted optional list inside an existing
					// object. Never instantiate an absent optional object scalar.
					job.output.fields = append(job.output.fields, Field{Name: p.name, Value: Value{kind: ListValue}})
				}
			}
		case "field":
			p := job.property
			if p.min == nil || p.max == nil {
				status = combineStatus(status, NotCompleted)
				continue
			}
			if p.max.Cmp(big.NewInt(1)) == 0 {
				if job.input.kind == ListValue {
					problem(job, "value.shape", "maxCount 1 requires one scalar or object, not a list.")
					continue
				}
				job.mode = "item"
				jobs = append(jobs, job)
				continue
			}
			if job.input.kind != ListValue {
				problem(job, "value.shape", "this cardinality requires a list.")
				continue
			}
			count := new(big.Int).SetUint64(uint64(len(job.input.items)))
			if count.Cmp(p.min) < 0 || (p.max.Sign() >= 0 && count.Cmp(p.max) > 0) {
				problem(job, "value.cardinality", "list length is outside the declared minCount/maxCount.")
			}
			*job.output = Value{kind: ListValue, items: make([]Value, len(job.input.items)), position: job.input.position}
			for index, item := range job.input.items {
				jobs = append(jobs, valueJob{mode: "item", input: item, output: &job.output.items[index], property: p, path: fmt.Sprintf("%s[%d]", job.path, index)})
			}
		case "item":
			p := job.property
			valid := false
			switch p.valueType {
			case "string":
				valid = job.input.kind == StringValue
				*job.output = job.input
			case "boolean":
				valid = job.input.kind == BooleanValue
				*job.output = job.input
			case "integer":
				valid = job.input.kind == IntegerValue
				*job.output = job.input
			case "number":
				valid = job.input.kind == IntegerValue || job.input.kind == NumberValue
				*job.output = job.input
				if job.input.kind == IntegerValue {
					*job.output = Value{kind: NumberValue, decimal: canonicalDecimal(job.input.text), position: job.input.position}
				}
			case "enum":
				if job.input.kind == StringValue {
					for _, allowed := range p.values {
						valid = valid || allowed == job.input.text
					}
				}
				*job.output = Value{kind: EnumValue, text: job.input.text, position: job.input.position}
			case "object":
				job.mode, job.fields = "object", p.fields
				job.complete = c.propertyComplete[p]
				// A partially resolved object declaration is still closed only
				// over its known fields; unknown names cannot yet be rejected.
				jobs = append(jobs, job)
				continue
			case "reference":
				value, bound := c.reference(a, job, p.target)
				status = combineStatus(status, bound)
				*job.output = value
				continue
			}
			if !valid {
				problem(job, "value.type", "value does not satisfy type "+p.valueType+" without coercion.")
			}
		}
	}
	return output, status
}

func (c *valueCompiler) reference(a definitionAddress, job valueJob, expected *Identity) (Value, Status) {
	if job.input.kind != StringValue || !validAuthorReference(job.input.text) {
		c.finding(a, "value.reference", job.path, "a reference must be a valid author reference String.", job.input)
		return Value{}, Failed
	}
	context := c.scopes[a.model].definitions[a.definition].references
	context.complete = context.declarationsBound
	id, status, code, message := resolveAuthorReference(context, job.input.text, c.scopes)
	if code != "" {
		c.finding(a, code, job.path, message, job.input)
	}
	if status != Passed {
		return Value{}, status
	}
	target := c.contracts.definitions[id]
	if target == nil || target.kind == nil || expected == nil {
		return Value{}, NotCompleted
	}
	if *target.kind != *expected {
		c.finding(a, "value.reference_kind", job.path, fmt.Sprintf("target %s/%s does not have the required Kind %s/%s.", id.Package, id.ID, expected.Package, expected.ID), job.input)
		return Value{}, Failed
	}
	return Value{kind: ReferenceValue, reference: id, position: job.input.position}, Passed
}

// Meta defaults are conditional on actual resolved category/type. Ordinary
// fields called properties or values never enter this normalization path.
func typedMetaSpec(contract *definitionContract) Value {
	object := Value{kind: ObjectValue}
	if contract.category == "kind" {
		object.fields = []Field{{Name: "properties", Value: typedDeclarationList(contract.fields)}}
		return object
	}
	p := contract.property
	object.fields = []Field{
		{Name: "name", Value: Value{kind: StringValue, text: p.name}},
		{Name: "type", Value: Value{kind: EnumValue, text: p.valueType}},
		{Name: "minCount", Value: Value{kind: IntegerValue, text: p.min.String()}},
		{Name: "maxCount", Value: Value{kind: IntegerValue, text: p.max.String()}},
	}
	switch p.valueType {
	case "object":
		object.fields = append(object.fields, Field{Name: "properties", Value: typedDeclarationList(p.fields)})
	case "reference":
		object.fields = append(object.fields, Field{Name: "target", Value: Value{kind: ReferenceValue, reference: *p.target}})
	case "enum":
		values := append([]string(nil), p.values...)
		sort.Strings(values)
		list := Value{kind: ListValue}
		for _, value := range values {
			list.items = append(list.items, Value{kind: StringValue, text: value})
		}
		object.fields = append(object.fields, Field{Name: "values", Value: list})
	}
	sort.Slice(object.fields, func(i, j int) bool { return object.fields[i].Name < object.fields[j].Name })
	return object
}

func typedDeclarationList(fields []Identity) Value {
	// A stable private order represents the unordered declaration. The later
	// canonical projection must apply its own normative JSON-byte ordering.
	ids := append([]Identity(nil), fields...)
	sort.Slice(ids, func(i, j int) bool {
		if ids[i].Package != ids[j].Package {
			return ids[i].Package < ids[j].Package
		}
		return ids[i].ID < ids[j].ID
	})
	list := Value{kind: ListValue}
	for _, id := range ids {
		list.items = append(list.items, Value{kind: ReferenceValue, reference: id})
	}
	return list
}
