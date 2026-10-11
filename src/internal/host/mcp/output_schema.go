package mcp

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// outputSchema describes a result type like schema, but writes each named
// struct type that occurs more than once, or refers to itself, once under
// $defs and points to it with $ref. Results stay fully typed; repeated types
// are no longer repeated in tools/list.
func outputSchema(t reflect.Type) map[string]any {
	b := &defsBuilder{defs: map[string]map[string]any{}, names: map[reflect.Type]string{}, building: map[reflect.Type]bool{}}
	root := b.structSchema(t)
	b.inlineSingleUses(root)
	// The server produces results, so their schemas describe types only:
	// required lists and closed objects guard inputs, not outputs.
	open := func(node any) {
		walkSchema(node, func(m map[string]any) {
			// A field named "required" is a schema map, not a []string list.
			if _, ok := m["required"].([]string); ok {
				delete(m, "required")
			}
			if closed, ok := m["additionalProperties"].(bool); ok && !closed {
				delete(m, "additionalProperties")
			}
		})
	}
	open(root)
	for _, def := range b.defs {
		open(def)
	}
	if len(b.defs) != 0 {
		defs := map[string]any{}
		for name, def := range b.defs {
			defs[name] = def
		}
		root["$defs"] = defs
	}
	return root
}

type defsBuilder struct {
	defs     map[string]map[string]any
	names    map[reflect.Type]string
	building map[reflect.Type]bool
}

var defNameUnsafe = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

func (b *defsBuilder) name(t reflect.Type) string {
	if name, ok := b.names[t]; ok {
		return name
	}
	name := defNameUnsafe.ReplaceAllString(t.String(), "_")
	for taken := true; taken; {
		taken = false
		for _, other := range b.names {
			if other == name {
				name += "_"
				taken = true
				break
			}
		}
	}
	b.names[t] = name
	return name
}

// of returns the schema of t, with named struct types as references.
func (b *defsBuilder) of(t reflect.Type) map[string]any {
	if special, ok := specialSchema(t); ok {
		return special
	}
	switch t.Kind() {
	case reflect.Pointer:
		return map[string]any{"anyOf": []any{b.of(t.Elem()), map[string]any{"type": "null"}}}
	case reflect.Struct:
		if t.Name() == "" {
			return b.structSchema(t)
		}
		name := b.name(t)
		if _, done := b.defs[name]; !done && !b.building[t] {
			b.building[t] = true
			b.defs[name] = b.structSchema(t)
			delete(b.building, t)
		}
		return map[string]any{"$ref": "#/$defs/" + name}
	case reflect.Map:
		return map[string]any{"type": []string{"object", "null"}, "additionalProperties": b.of(t.Elem())}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": []string{"array", "null"}, "items": b.of(t.Elem())}
	}
	return scalarSchema(t)
}

func (b *defsBuilder) structSchema(t reflect.Type) map[string]any {
	return structSchemaWith(t, b.of)
}

// inlineSingleUses writes a definition used exactly once back in place, so
// $defs only holds types that save space, and recursive types.
func (b *defsBuilder) inlineSingleUses(root map[string]any) {
	for changed := true; changed; {
		changed = false
		uses := map[string][]map[string]any{}
		collect := func(node any) {
			walkSchema(node, func(m map[string]any) {
				if ref, ok := m["$ref"].(string); ok {
					name := strings.TrimPrefix(ref, "#/$defs/")
					uses[name] = append(uses[name], m)
				}
			})
		}
		collect(root)
		names := make([]string, 0, len(b.defs))
		for name, def := range b.defs {
			collect(def)
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			def := b.defs[name]
			if len(uses[name]) != 1 || refersTo(def, name) {
				continue
			}
			site := uses[name][0]
			delete(site, "$ref")
			for key, value := range def {
				site[key] = value
			}
			delete(b.defs, name)
			changed = true
			break
		}
	}
}

func refersTo(node any, name string) bool {
	found := false
	walkSchema(node, func(m map[string]any) {
		if m["$ref"] == "#/$defs/"+name {
			found = true
		}
	})
	return found
}

func walkSchema(node any, visit func(map[string]any)) {
	switch value := node.(type) {
	case map[string]any:
		visit(value)
		for _, child := range value {
			walkSchema(child, visit)
		}
	case []any:
		for _, child := range value {
			walkSchema(child, visit)
		}
	}
}
