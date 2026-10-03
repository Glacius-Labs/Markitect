package core

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Registry is the deterministic catalogue used to validate and compile
// versioned resource vocabularies. It contains no executable plugin code.
type Registry struct {
	domains   map[string]DomainDefinition
	kinds     map[string]KindDefinition
	active    map[string]string
	relations map[string]map[string]RelationDefinition
}

func NewRegistry() *Registry {
	r := &Registry{domains: map[string]DomainDefinition{}, kinds: map[string]KindDefinition{}, active: map[string]string{}, relations: map[string]map[string]RelationDefinition{}}
	for _, kind := range []string{"Project", "Package", "Domain"} {
		r.kinds[kindKey(APIVersion, kind)] = KindDefinition{Properties: map[string]PropertyDefinition{}}
	}
	// These are the explicit relations of the built-in AI domain. The legacy
	// lowering view remains available to existing render and application code.
	ai := DomainDefinition{APIVersion: APIVersion, Kinds: map[string]KindDefinition{}, Relations: map[string]RelationDefinition{
		"rules":      {Field: "rules", SourceKinds: []string{"Rule", "Text", "Contract", "Workflow", "Skill", "Agent"}, TargetKinds: []string{"Rule"}, Context: true, Invalidate: true},
		"uses":       {Field: "uses", SourceKinds: []string{"Workflow", "Skill", "Agent"}, TargetKinds: []string{"*"}, Context: true, Invalidate: true},
		"needs":      {Field: "needs", SourceKinds: []string{"Workflow", "Skill", "Agent"}, TargetKinds: []string{"Contract"}, Context: true, Invalidate: true},
		"implements": {Field: "implements", SourceKinds: []string{"Workflow", "Skill", "Agent"}, TargetKinds: []string{"Contract"}, Context: true, Invalidate: true},
	}}
	ref := PropertyDefinition{Type: "array", Items: &PropertyDefinition{Type: "ref", RefKind: "*"}}
	for _, kind := range []string{"Text", "Rule", "Contract", "Workflow", "Skill", "Agent"} {
		props := map[string]PropertyDefinition{}
		for _, field := range []string{"rules", "uses", "needs", "implements"} {
			props[field] = ref
		}
		ai.Kinds[kind] = KindDefinition{Properties: props}
	}
	if err := r.AddDomain(ai); err != nil {
		panic("invalid built-in AI domain: " + err.Error())
	}
	return r
}

func kindKey(apiVersion, kind string) string { return apiVersion + "\x00" + kind }

func (r *Registry) AddDomain(domain DomainDefinition) error {
	if r == nil {
		return fmt.Errorf("registry is nil")
	}
	if !validAPIVersion(domain.APIVersion) {
		return fmt.Errorf("domain apiVersion %q must be a group/version", domain.APIVersion)
	}
	if len(domain.Kinds) == 0 {
		return fmt.Errorf("domain %q must define at least one kind", domain.APIVersion)
	}
	if _, exists := r.domains[domain.APIVersion]; exists {
		return fmt.Errorf("domain apiVersion %q is already registered", domain.APIVersion)
	}
	group := apiGroup(domain.APIVersion)
	if prior := r.active[group]; prior != "" && prior != domain.APIVersion {
		return fmt.Errorf("domain group %q has conflicting active versions %q and %q", group, prior, domain.APIVersion)
	}
	for _, name := range sortedMapKeys(domain.Kinds) {
		definition := domain.Kinds[name]
		if !validIdentifier(name) {
			return fmt.Errorf("domain kind %q is not a valid identifier", name)
		}
		if domain.APIVersion != APIVersion && isBuiltinKind(name) {
			return fmt.Errorf("kind %q is reserved by the built-in Markitect domains", name)
		}
		if _, exists := r.kinds[kindKey(domain.APIVersion, name)]; exists {
			return fmt.Errorf("kind %s/%s is already registered", domain.APIVersion, name)
		}
		if old := r.activeKindGroup(group, name); old != "" {
			return fmt.Errorf("kind identity %s/%s is already registered as %s", group, name, old)
		}
		if err := validateKindDefinition(name, definition); err != nil {
			return err
		}
	}
	for _, name := range sortedMapKeys(domain.Relations) {
		relation := domain.Relations[name]
		if !validIdentifier(name) {
			return fmt.Errorf("relation %q is not a valid identifier", name)
		}
		if relation.Field == "" || len(relation.SourceKinds) == 0 || len(relation.TargetKinds) == 0 {
			return fmt.Errorf("relation %q requires field, sourceKinds, and targetKinds", name)
		}
		if err := uniqueNames(relation.SourceKinds); err != nil {
			return fmt.Errorf("relation %q sourceKinds: %w", name, err)
		}
		if err := uniqueRelationKinds(relation.TargetKinds); err != nil {
			return fmt.Errorf("relation %q targetKinds: %w", name, err)
		}
		if relation.MinTargets != nil && *relation.MinTargets < 0 || relation.MaxTargets != nil && *relation.MaxTargets < 0 || relation.MinTargets != nil && relation.MaxTargets != nil && *relation.MinTargets > *relation.MaxTargets {
			return fmt.Errorf("relation %q has invalid min/max target bounds", name)
		}
		for _, k := range append(append([]string{}, relation.SourceKinds...), relation.TargetKinds...) {
			if k == "*" {
				continue
			}
			if _, ok := domain.Kinds[k]; !ok {
				return fmt.Errorf("relation %q refers to undefined kind %q", name, k)
			}
		}
		for _, k := range relation.SourceKinds {
			prop, ok := domain.Kinds[k].Properties[relation.Field]
			if !ok {
				return fmt.Errorf("relation %q field %q is absent from source kind %q", name, relation.Field, k)
			}
			if !propertyContainsRefs(prop) {
				return fmt.Errorf("relation %q field %q must contain typed references", name, relation.Field)
			}
			if err := validateRelationRefKinds(prop, relation.TargetKinds); err != nil {
				return fmt.Errorf("relation %q field %q: %w", name, relation.Field, err)
			}
		}
	}
	for _, kindName := range sortedMapKeys(domain.Kinds) {
		kind := domain.Kinds[kindName]
		for _, field := range sortedMapKeys(kind.Properties) {
			covered := false
			for _, relation := range domain.Relations {
				if relation.Field == field && contains(relation.SourceKinds, kindName) {
					covered = true
					break
				}
			}
			if err := validateTypedRefProperties(kind.Properties[field], covered); err != nil {
				return fmt.Errorf("kind %q property %q: %w", kindName, field, err)
			}
		}
	}
	constraintNames := map[string]bool{}
	for _, c := range domain.Constraints {
		if err := validateConstraint(c, domain); err != nil {
			return err
		}
		if constraintNames[c.Name] {
			return fmt.Errorf("duplicate constraint name %q", c.Name)
		}
		constraintNames[c.Name] = true
	}
	copyDomain := cloneDomain(domain)
	r.domains[domain.APIVersion] = copyDomain
	r.active[group] = domain.APIVersion
	for kind, definition := range copyDomain.Kinds {
		r.kinds[kindKey(domain.APIVersion, kind)] = definition
	}
	r.relations[domain.APIVersion] = copyDomain.Relations
	return nil
}

func isBuiltinKind(kind string) bool {
	switch kind {
	case "Text", "Rule", "Contract", "Workflow", "Skill", "Agent", "Project", "Package", "Domain":
		return true
	}
	return false
}

func (r *Registry) activeKindGroup(group, kind string) string {
	for api, d := range r.domains {
		if apiGroup(api) == group {
			if _, ok := d.Kinds[kind]; ok {
				return api
			}
		}
	}
	return ""
}

func (r *Registry) Lookup(apiVersion, kind string) (KindDefinition, bool) {
	if r == nil {
		return KindDefinition{}, false
	}
	d, ok := r.kinds[kindKey(apiVersion, kind)]
	if !ok {
		return KindDefinition{}, false
	}
	return cloneKind(d), true
}
func (r *Registry) IsKnownKind(apiVersion, kind string) bool {
	if r == nil {
		return false
	}
	_, ok := r.kinds[kindKey(apiVersion, kind)]
	return ok
}
func (r *Registry) IsAPIVersionRegistered(apiVersion string) bool {
	if r == nil {
		return false
	}
	if _, ok := r.domains[apiVersion]; ok {
		return true
	}
	return apiVersion == APIVersion
}
func (r *Registry) Domain(apiVersion string) (DomainDefinition, bool) {
	if r == nil {
		return DomainDefinition{}, false
	}
	d, ok := r.domains[apiVersion]
	return cloneDomain(d), ok
}
func (r *Registry) Domains() []DomainDefinition {
	out := make([]DomainDefinition, 0, len(r.domains))
	for _, d := range r.domains {
		out = append(out, cloneDomain(d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].APIVersion < out[j].APIVersion })
	return out
}
func (r *Registry) Relations(apiVersion string) map[string]RelationDefinition {
	d := map[string]RelationDefinition{}
	if r == nil {
		return d
	}
	for k, v := range r.relations[apiVersion] {
		v.SourceKinds = append([]string(nil), v.SourceKinds...)
		v.TargetKinds = append([]string(nil), v.TargetKinds...)
		if v.MinTargets != nil {
			x := *v.MinTargets
			v.MinTargets = &x
		}
		if v.MaxTargets != nil {
			x := *v.MaxTargets
			v.MaxTargets = &x
		}
		d[k] = v
	}
	return d
}

func (r *Registry) ValidateData(apiVersion, kind string, data map[string]any) error {
	definition, ok := r.Lookup(apiVersion, kind)
	if !ok {
		return fmt.Errorf("unsupported resource kind %q for apiVersion %q", kind, apiVersion)
	}
	return validateObject(data, definition.Properties, definition.Required, "spec")
}

func validateKindDefinition(name string, d KindDefinition) error {
	if len(d.Properties) == 0 {
		return fmt.Errorf("kind %q must define properties", name)
	}
	for _, field := range d.Required {
		if _, ok := d.Properties[field]; !ok {
			return fmt.Errorf("kind %q requires undefined property %q", name, field)
		}
	}
	for _, field := range sortedMapKeys(d.Properties) {
		p := d.Properties[field]
		if !validIdentifier(field) {
			return fmt.Errorf("kind %q property %q is not a valid identifier", name, field)
		}
		if err := validateProperty(p); err != nil {
			return fmt.Errorf("kind %q property %q: %w", name, field, err)
		}
	}
	if d.InputsField != "" {
		p, ok := d.Properties[d.InputsField]
		if !ok || p.Type != "array" || p.Items == nil || p.Items.Type != "string" {
			return fmt.Errorf("kind %q inputsField must name an array-of-string property", name)
		}
	}
	return nil
}
func validateProperty(p PropertyDefinition) error {
	switch p.Type {
	case "string", "boolean", "integer", "number":
		if p.Items != nil || len(p.Properties) > 0 || p.RefKind != "" {
			return fmt.Errorf("scalar type cannot declare items, properties, or refKind")
		}
	case "ref":
		if p.Items != nil || len(p.Properties) > 0 || len(p.Required) > 0 || len(p.Enum) > 0 {
			return fmt.Errorf("ref cannot declare items, properties, required, or enum")
		}
		if p.RefKind != "" && p.RefKind != "*" && !validIdentifier(p.RefKind) {
			return fmt.Errorf("refKind must be a valid kind name when provided")
		}
	case "array":
		if len(p.Properties) > 0 || len(p.Required) > 0 || len(p.Enum) > 0 || p.RefKind != "" {
			return fmt.Errorf("array cannot declare properties, required, enum, or refKind")
		}
		if p.Items == nil {
			return fmt.Errorf("array requires items")
		}
		if err := validateProperty(*p.Items); err != nil {
			return err
		}
	case "object":
		if p.Items != nil || len(p.Enum) > 0 || p.RefKind != "" {
			return fmt.Errorf("object cannot declare items, enum, or refKind")
		}
		if len(p.Properties) == 0 {
			return fmt.Errorf("object requires closed properties")
		}
		for _, n := range sortedMapKeys(p.Properties) {
			v := p.Properties[n]
			if !validIdentifier(n) {
				return fmt.Errorf("invalid object property %q", n)
			}
			if err := validateProperty(v); err != nil {
				return fmt.Errorf("property %q: %w", n, err)
			}
		}
		for _, n := range p.Required {
			if _, ok := p.Properties[n]; !ok {
				return fmt.Errorf("required property %q is undefined", n)
			}
		}
	default:
		return fmt.Errorf("unsupported type %q", p.Type)
	}
	if len(p.Enum) > 0 {
		if p.Type != "string" && p.Type != "boolean" && p.Type != "integer" && p.Type != "number" {
			return fmt.Errorf("enum is only supported for scalar types")
		}
		for _, v := range p.Enum {
			if !scalar(v) {
				return fmt.Errorf("enum values must be scalar")
			}
			if err := validateValue(v, PropertyDefinition{Type: p.Type}, "enum"); err != nil {
				return fmt.Errorf("enum value type mismatch: %w", err)
			}
		}
	}
	return nil
}
func validateObject(v map[string]any, properties map[string]PropertyDefinition, required []string, where string) error {
	for _, name := range required {
		if _, ok := v[name]; !ok {
			return fmt.Errorf("%s requires field %q", where, name)
		}
	}
	for _, name := range sortedMapKeys(v) {
		value := v[name]
		p, ok := properties[name]
		if !ok {
			return fmt.Errorf("%s has unknown field %q", where, name)
		}
		if err := validateValue(value, p, where+"."+name); err != nil {
			return err
		}
	}
	return nil
}
func validateValue(v any, p PropertyDefinition, where string) error {
	if p.Type == "ref" {
		m, ok := asMap(v)
		if !ok {
			return fmt.Errorf("%s must be a reference object", where)
		}
		for key, value := range m {
			if key != "apiVersion" && key != "kind" && key != "name" && key != "namespace" && key != "package" {
				return fmt.Errorf("%s reference has unknown field %q", where, key)
			}
			if _, ok := value.(string); !ok {
				return fmt.Errorf("%s reference field %s must be string", where, key)
			}
		}
		if name, ok := m["name"].(string); !ok || name == "" {
			return fmt.Errorf("%s reference requires name", where)
		}
		if kind, ok := m["kind"].(string); ok && p.RefKind != "" && p.RefKind != "*" && kind != p.RefKind {
			return fmt.Errorf("%s reference must target kind %s", where, p.RefKind)
		}
		return nil
	}
	switch p.Type {
	case "string":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("%s must be string", where)
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("%s must be boolean", where)
		}
	case "integer":
		if !isInteger(v) {
			return fmt.Errorf("%s must be integer", where)
		}
	case "number":
		if !isNumber(v) {
			return fmt.Errorf("%s must be number", where)
		}
	case "array":
		a, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%s must be array", where)
		}
		for i, x := range a {
			if err := validateValue(x, *p.Items, fmt.Sprintf("%s[%d]", where, i)); err != nil {
				return err
			}
		}
	case "object":
		m, ok := asMap(v)
		if !ok {
			return fmt.Errorf("%s must be object", where)
		}
		if err := validateObject(m, p.Properties, p.Required, where); err != nil {
			return err
		}
	}
	if len(p.Enum) > 0 {
		found := false
		for _, x := range p.Enum {
			if reflect.DeepEqual(x, v) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s value is outside its declared enum", where)
		}
	}
	return nil
}

func validateConstraint(c ConstraintDefinition, d DomainDefinition) error {
	if !validIdentifier(c.Name) {
		return fmt.Errorf("constraint name %q is invalid", c.Name)
	}
	if c.Assert.Op != "count" && c.Assert.Scope != "" {
		return fmt.Errorf("constraint %q scope is only supported for count", c.Name)
	}
	if c.Assert.Op != "same-target" && (len(c.Assert.Left) > 0 || len(c.Assert.Right) > 0) {
		return fmt.Errorf("constraint %q left and right are only supported for same-target", c.Name)
	}
	if c.Select.Kind != "" {
		if _, ok := d.Kinds[c.Select.Kind]; !ok {
			return fmt.Errorf("constraint %q selects undefined kind %q", c.Name, c.Select.Kind)
		}
	}
	a := c.Assert
	switch a.Op {
	case "same-target":
		if c.Select.Kind == "" {
			return fmt.Errorf("constraint %q same-target requires select.kind", c.Name)
		}
		if a.Scope != "" || a.Field != "" || a.Relation != "" || a.Value != nil || len(a.Values) > 0 || a.Min != nil || a.Max != nil {
			return fmt.Errorf("constraint %q same-target does not accept scope, field, relation, value, values, min, or max", c.Name)
		}
		if err := validateSameTargetPath(c.Name, "left", a.Left, c.Select.Kind, d); err != nil {
			return err
		}
		if err := validateSameTargetPath(c.Name, "right", a.Right, c.Select.Kind, d); err != nil {
			return err
		}
		leftTargets := terminalKinds(a.Left, d)
		rightTargets := terminalKinds(a.Right, d)
		compatible := false
		for _, kind := range leftTargets {
			if contains(rightTargets, kind) {
				compatible = true
				break
			}
		}
		if !compatible {
			return fmt.Errorf("constraint %q same-target paths have no compatible terminal kinds", c.Name)
		}
	case "present", "equal", "allowed", "unique":
		if a.Field == "" {
			return fmt.Errorf("constraint %q op %s requires field", c.Name, a.Op)
		}
		kinds := []string{}
		if c.Select.Kind != "" {
			kinds = []string{c.Select.Kind}
		} else {
			kinds = sortedMapKeys(d.Kinds)
		}
		for _, kind := range kinds {
			property, ok := d.Kinds[kind].Properties[a.Field]
			if !ok {
				return fmt.Errorf("constraint %q refers to undefined field %q on selected kind %q", c.Name, a.Field, kind)
			}
			if a.Op == "unique" && (property.Type == "array" || property.Type == "object" || property.Type == "ref") {
				return fmt.Errorf("constraint %q unique requires a scalar field", c.Name)
			}
			if a.Op == "equal" {
				if !scalar(a.Value) {
					return fmt.Errorf("constraint %q equal requires scalar value", c.Name)
				}
				if err := validateValue(a.Value, property, c.Name+".assert.value"); err != nil {
					return err
				}
			}
			if a.Op == "allowed" {
				if len(a.Values) == 0 {
					return fmt.Errorf("constraint %q allowed requires nonempty values", c.Name)
				}
				for _, value := range a.Values {
					if !scalar(value) {
						return fmt.Errorf("constraint %q allowed values must be scalar", c.Name)
					}
					if err := validateValue(value, property, c.Name+".assert.values"); err != nil {
						return err
					}
				}
			}
		}
	case "allowed-targets":
		if a.Relation == "" {
			return fmt.Errorf("constraint %q allowed-targets requires relation", c.Name)
		}
		relation, ok := d.Relations[a.Relation]
		if !ok {
			return fmt.Errorf("constraint %q refers to undefined relation %q", c.Name, a.Relation)
		}
		if c.Select.Kind != "" && !contains(relation.SourceKinds, c.Select.Kind) {
			return fmt.Errorf("constraint %q selects kind %q, which is not a source of relation %q", c.Name, c.Select.Kind, a.Relation)
		}
		if len(a.Values) == 0 {
			return fmt.Errorf("constraint %q allowed-targets requires nonempty values", c.Name)
		}
		for _, value := range a.Values {
			kind, ok := value.(string)
			if !ok {
				return fmt.Errorf("constraint %q allowed-targets values must be kind names", c.Name)
			}
			found := false
			for _, target := range relation.TargetKinds {
				if target == "*" || target == kind {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("constraint %q allowed-targets kind %q is outside relation targetKinds", c.Name, kind)
			}
		}
	case "count":
		if a.Scope != "" && a.Scope != "resource" && a.Scope != "selection" {
			return fmt.Errorf("constraint %q count scope must be resource or selection", c.Name)
		}
		if a.Scope == "resource" && a.Relation == "" {
			return fmt.Errorf("constraint %q resource count requires relation", c.Name)
		}
		if (a.Min == nil && a.Max == nil) || a.Min != nil && *a.Min < 0 || a.Max != nil && *a.Max < 0 || a.Min != nil && a.Max != nil && *a.Min > *a.Max {
			return fmt.Errorf("constraint %q count requires valid min/max bounds", c.Name)
		}
		if a.Relation != "" {
			relation, ok := d.Relations[a.Relation]
			if !ok {
				return fmt.Errorf("constraint %q refers to undefined relation %q", c.Name, a.Relation)
			}
			if c.Select.Kind != "" && !contains(relation.SourceKinds, c.Select.Kind) {
				return fmt.Errorf("constraint %q selects kind %q, which is not a source of relation %q", c.Name, c.Select.Kind, a.Relation)
			}
		}
	default:
		return fmt.Errorf("constraint %q uses unsupported op %q", c.Name, a.Op)
	}
	return nil
}

func propertyContainsRefs(p PropertyDefinition) bool {
	if p.Type == "ref" {
		return true
	}
	if p.Type == "array" && p.Items != nil {
		return propertyContainsRefs(*p.Items)
	}
	return false
}
func validateTypedRefProperties(p PropertyDefinition, relationField bool) error {
	switch p.Type {
	case "ref":
		if p.RefKind == "" && !relationField {
			return fmt.Errorf("typed reference needs refKind or a relation descriptor")
		}
	case "array":
		if p.Items != nil {
			return validateTypedRefProperties(*p.Items, relationField)
		}
	case "object":
		for _, name := range sortedMapKeys(p.Properties) {
			if err := validateTypedRefProperties(p.Properties[name], relationField); err != nil {
				return err
			}
		}
	}
	return nil
}
func validateRelationRefKinds(p PropertyDefinition, targetKinds []string) error {
	if p.Type == "ref" {
		if p.RefKind != "" && p.RefKind != "*" && !contains(targetKinds, p.RefKind) {
			return fmt.Errorf("refKind %q is outside targetKinds", p.RefKind)
		}
		return nil
	}
	if p.Type == "array" && p.Items != nil {
		return validateRelationRefKinds(*p.Items, targetKinds)
	}
	return nil
}
func asMap(v any) (map[string]any, bool) { m, ok := v.(map[string]any); return m, ok }
func scalar(v any) bool {
	switch v.(type) {
	case string, bool, int, int64, float64:
		return true
	}
	return false
}
func isInteger(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	}
	return false
}
func isNumber(v any) bool {
	return isInteger(v) || func() bool {
		switch v.(type) {
		case float32, float64:
			return true
		}
		return false
	}()
}
func validIdentifier(v string) bool {
	if v == "" || strings.TrimSpace(v) != v {
		return false
	}
	for i, c := range v {
		if !(c == '_' || c == '-' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
func validAPIVersion(v string) bool {
	p := strings.Split(v, "/")
	return len(p) == 2 && validDNS(p[0]) && validIdentifier(p[1])
}
func validDNS(v string) bool {
	if v == "" || strings.Contains(v, " ") {
		return false
	}
	for _, s := range strings.Split(v, ".") {
		if s == "" {
			return false
		}
		for i, c := range s {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' && i > 0 && i < len(s)-1) {
				return false
			}
		}
	}
	return true
}
func apiGroup(v string) string {
	p := strings.Split(v, "/")
	if len(p) == 2 {
		return p[0]
	}
	return v
}
func uniqueNames(names []string) error {
	seen := map[string]bool{}
	for _, n := range names {
		if !validIdentifier(n) {
			return fmt.Errorf("invalid name %q", n)
		}
		if seen[n] {
			return fmt.Errorf("duplicate name %q", n)
		}
		seen[n] = true
	}
	return nil
}

func uniqueRelationKinds(names []string) error {
	seen := map[string]bool{}
	for _, n := range names {
		if n != "*" && !validIdentifier(n) {
			return fmt.Errorf("invalid kind %q", n)
		}
		if seen[n] {
			return fmt.Errorf("duplicate kind %q", n)
		}
		seen[n] = true
	}
	return nil
}
func cloneDomain(d DomainDefinition) DomainDefinition {
	out := d
	out.Kinds = map[string]KindDefinition{}
	for k, v := range d.Kinds {
		out.Kinds[k] = cloneKind(v)
	}
	out.Relations = map[string]RelationDefinition{}
	for k, v := range d.Relations {
		v.SourceKinds = append([]string(nil), v.SourceKinds...)
		v.TargetKinds = append([]string(nil), v.TargetKinds...)
		if v.MinTargets != nil {
			x := *v.MinTargets
			v.MinTargets = &x
		}
		if v.MaxTargets != nil {
			x := *v.MaxTargets
			v.MaxTargets = &x
		}
		out.Relations[k] = v
	}
	out.Constraints = append([]ConstraintDefinition(nil), d.Constraints...)
	for i := range out.Constraints {
		c := &out.Constraints[i]
		if c.Assert.Op == "count" && c.Assert.Scope == "" {
			c.Assert.Scope = "selection"
		}
		if c.Select.Labels != nil {
			m := map[string]string{}
			for k, v := range c.Select.Labels {
				m[k] = v
			}
			c.Select.Labels = m
		}
		c.Assert.Value = cloneAny(c.Assert.Value)
		c.Assert.Values = append([]any(nil), c.Assert.Values...)
		c.Assert.Left = append([]string(nil), c.Assert.Left...)
		c.Assert.Right = append([]string(nil), c.Assert.Right...)
		for j := range c.Assert.Values {
			c.Assert.Values[j] = cloneAny(c.Assert.Values[j])
		}
		if c.Assert.Min != nil {
			x := *c.Assert.Min
			c.Assert.Min = &x
		}
		if c.Assert.Max != nil {
			x := *c.Assert.Max
			c.Assert.Max = &x
		}
	}
	return out
}

func cloneKind(v KindDefinition) KindDefinition {
	v.Required = append([]string(nil), v.Required...)
	props := map[string]PropertyDefinition{}
	for f, p := range v.Properties {
		props[f] = cloneProperty(p)
	}
	v.Properties = props
	return v
}
func cloneProperty(p PropertyDefinition) PropertyDefinition {
	p.Required = append([]string(nil), p.Required...)
	p.Enum = append([]any(nil), p.Enum...)
	for i := range p.Enum {
		p.Enum[i] = cloneAny(p.Enum[i])
	}
	if p.Items != nil {
		x := cloneProperty(*p.Items)
		p.Items = &x
	}
	if p.Properties != nil {
		m := map[string]PropertyDefinition{}
		for k, v := range p.Properties {
			m[k] = cloneProperty(v)
		}
		p.Properties = m
	}
	return p
}
func cloneAny(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, x := range value {
			out[k] = cloneAny(x)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, x := range value {
			out[i] = cloneAny(x)
		}
		return out
	default:
		return v
	}
}
func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
