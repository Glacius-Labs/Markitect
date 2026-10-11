package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
)

// Schemas derive from the DTOs actually decoded, avoiding a second operation
// contract. Input schemas are written inline, because clients build arguments
// from them; result schemas use $defs (outputSchema).
func schema(t reflect.Type) map[string]any { return schemaOf(t, map[reflect.Type]bool{}) }

// schemaOf describes a recursive struct type's nested occurrence as any JSON
// value instead of expanding it forever.
func schemaOf(t reflect.Type, open map[reflect.Type]bool) map[string]any {
	if special, ok := specialSchema(t); ok {
		return special
	}
	of := func(t reflect.Type) map[string]any { return schemaOf(t, open) }
	switch t.Kind() {
	case reflect.Pointer:
		return map[string]any{"anyOf": []any{of(t.Elem()), map[string]any{"type": "null"}}}
	case reflect.Struct:
		if open[t] {
			return map[string]any{}
		}
		open[t] = true
		defer delete(open, t)
		return structSchemaWith(t, of)
	case reflect.Map:
		return map[string]any{"type": []string{"object", "null"}, "additionalProperties": of(t.Elem())}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": []string{"array", "null"}, "items": of(t.Elem())}
	}
	return scalarSchema(t)
}

// specialSchema covers types whose JSON form differs from their Go kind.
func specialSchema(t reflect.Type) (map[string]any, bool) {
	switch t {
	case reflect.TypeFor[json.RawMessage]():
		// RawMessage emits its underlying JSON value, unlike ordinary []byte's
		// base64 string. The shared service owns validation of this embedded value.
		return map[string]any{}, true
	case reflect.TypeFor[projectrun.Duration](), reflect.TypeFor[[]byte]():
		return map[string]any{"type": "string"}, true
	case reflect.TypeFor[codexappserver.WindowsSandboxBackend]():
		return map[string]any{"type": "string", "enum": []string{string(codexappserver.WindowsSandboxBackendMXC)}}, true
	case reflect.TypeFor[projectrun.AppServerEnvironmentMode]():
		return map[string]any{"type": "string", "enum": []string{string(projectrun.AppServerEnvironmentModeInherit)}}, true
	case reflect.TypeFor[time.Time]():
		return map[string]any{"type": "string", "format": "date-time"}, true
	}
	return nil, false
}

// structSchemaWith describes a struct's JSON object; of describes each field.
func structSchemaWith(t reflect.Type, of func(reflect.Type) map[string]any) map[string]any {
	p := map[string]any{}
	required := []string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("json"), ",")
		name := tag[0]
		if name == "-" {
			continue
		}
		// encoding/json promotes the fields of an untagged embedded struct.
		if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct {
			embedded := structSchemaWith(f.Type, of)
			for key, value := range embedded["properties"].(map[string]any) {
				p[key] = value
			}
			required = append(required, toStrings(embedded["required"])...)
			continue
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		p[name] = of(f.Type)
		if !strings.Contains(f.Tag.Get("json"), ",omitempty") {
			required = append(required, name)
		}
	}
	return map[string]any{"type": "object", "properties": p, "required": required, "additionalProperties": false}
}

func scalarSchema(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	}
	return map[string]any{}
}
func decodeTyped(raw []byte, s map[string]any, out any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	v, err := uniqueValue(d)
	if err != nil {
		return errors.New("invalid JSON arguments")
	}
	if _, err = d.Token(); err != io.EOF {
		return errors.New("trailing JSON arguments")
	}
	if err = validate(v, s, ""); err != nil {
		return err
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err = d.Decode(out); err != nil {
		return errors.New("arguments do not match tool schema")
	}
	return nil
}

// Reject duplicate keys at every depth rather than silently selecting authority inputs.
func uniqueValue(d *json.Decoder) (any, error) {
	v, err := d.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := v.(json.Delim)
	if !ok {
		return v, nil
	}
	switch delim {
	case '{':
		m := map[string]any{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return nil, err
			}
			key := k.(string)
			if _, ok := m[key]; ok {
				return nil, errors.New("duplicate JSON key")
			}
			x, err := uniqueValue(d)
			if err != nil {
				return nil, err
			}
			m[key] = x
		}
		_, err = d.Token()
		return m, err
	case '[':
		a := []any{}
		for d.More() {
			x, err := uniqueValue(d)
			if err != nil {
				return nil, err
			}
			a = append(a, x)
		}
		_, err = d.Token()
		return a, err
	}
	return nil, errors.New("invalid JSON delimiter")
}

// validate reports the failing field path; adapters decide whether the text
// is shown, and it never contains argument values.
func validate(v any, s map[string]any, path string) error {
	defs, _ := s["$defs"].(map[string]any)
	return validateIn(v, s, path, defs)
}

func validateIn(v any, s map[string]any, path string, defs map[string]any) error {
	if ref, ok := s["$ref"].(string); ok {
		def, ok := defs[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any)
		if !ok {
			return schemaError(path, "refers to an unknown definition")
		}
		return validateIn(v, def, path, defs)
	}
	if variants, ok := s["anyOf"].([]any); ok {
		for _, x := range variants {
			if validateIn(v, x.(map[string]any), path, defs) == nil {
				return nil
			}
		}
		return schemaError(path, "has an invalid value")
	}
	typ, _ := s["type"].(string)
	if types, ok := s["type"].([]string); ok {
		if v == nil {
			return nil
		}
		typ = types[0]
	}
	if values, ok := s["enum"].([]string); ok {
		text, ok := v.(string)
		if !ok {
			return schemaError(path, "must be one of "+strings.Join(values, ", "))
		}
		for _, value := range values {
			if text == value {
				return nil
			}
		}
		return schemaError(path, "must be one of "+strings.Join(values, ", "))
	}
	switch typ {
	case "object":
		m, ok := v.(map[string]any)
		if !ok {
			return schemaError(path, "must be an object")
		}
		p, _ := s["properties"].(map[string]any)
		for _, r := range toStrings(s["required"]) {
			if _, ok := m[r]; !ok {
				return schemaError(join(path, r), "is required")
			}
		}
		for k, x := range m {
			if child, ok := p[k]; ok {
				if err := validateIn(x, child.(map[string]any), join(path, k), defs); err != nil {
					return err
				}
			} else if child, ok := s["additionalProperties"].(map[string]any); ok {
				if err := validateIn(x, child, join(path, k), defs); err != nil {
					return err
				}
			} else if closed, ok := s["additionalProperties"].(bool); ok && !closed {
				return schemaError(join(path, k), "is not a known field")
			}
		}
	case "array":
		a, ok := v.([]any)
		if !ok {
			return schemaError(path, "must be an array")
		}
		for i, x := range a {
			if err := validateIn(x, s["items"].(map[string]any), path+"["+strconv.Itoa(i)+"]", defs); err != nil {
				return err
			}
		}
	case "string":
		if _, ok := v.(string); !ok {
			return schemaError(path, "must be a string")
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return schemaError(path, "must be a boolean")
		}
	case "integer":
		n, ok := v.(json.Number)
		if !ok {
			return schemaError(path, "must be an integer")
		}
		if _, err := n.Int64(); err != nil {
			if _, err := strconv.ParseUint(string(n), 10, 64); err != nil {
				return schemaError(path, "must be an integer")
			}
		}
	case "number":
		if _, ok := v.(json.Number); !ok {
			return schemaError(path, "must be a number")
		}
	case "null":
		if v != nil {
			return schemaError(path, "must be null")
		}
	}
	return nil
}

func join(path, field string) string {
	if path == "" {
		return field
	}
	return path + "." + field
}

func schemaError(path, problem string) error {
	if path == "" {
		path = "arguments"
	}
	return fmt.Errorf("arguments do not match closed tool schema: %s %s", path, problem)
}

func toStrings(v any) []string { r, _ := v.([]string); return r }
