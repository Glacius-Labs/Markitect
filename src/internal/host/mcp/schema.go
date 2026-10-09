package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
)

// Schemas derive from the DTOs actually decoded, avoiding a second operation contract.
func schema(t reflect.Type) map[string]any {
	// RawMessage emits its underlying JSON value, unlike ordinary []byte's
	// base64 string. The shared service owns validation of this embedded value.
	if t == reflect.TypeFor[json.RawMessage]() {
		return map[string]any{}
	}
	if t == reflect.TypeFor[projectrun.Duration]() || t == reflect.TypeFor[[]byte]() {
		return map[string]any{"type": "string"}
	}
	if t.Kind() == reflect.Pointer {
		return map[string]any{"anyOf": []any{schema(t.Elem()), map[string]any{"type": "null"}}}
	}
	if t == reflect.TypeFor[time.Time]() {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.Struct:
		p := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")
			name := tag[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			p[name] = schema(f.Type)
			if !strings.Contains(f.Tag.Get("json"), ",omitempty") {
				required = append(required, name)
			}
		}
		return map[string]any{"type": "object", "properties": p, "required": required, "additionalProperties": false}
	case reflect.Map:
		return map[string]any{"type": []string{"object", "null"}, "additionalProperties": schema(t.Elem())}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": []string{"array", "null"}, "items": schema(t.Elem())}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	default:
		return map[string]any{}
	}
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
	if err = validate(v, s); err != nil {
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
func validate(v any, s map[string]any) error {
	if variants, ok := s["anyOf"].([]any); ok {
		for _, x := range variants {
			if validate(v, x.(map[string]any)) == nil {
				return nil
			}
		}
		return errors.New("invalid nullable value")
	}
	typ, _ := s["type"].(string)
	if types, ok := s["type"].([]string); ok {
		if v == nil {
			return nil
		}
		typ = types[0]
	}
	bad := errors.New("arguments do not match closed tool schema")
	switch typ {
	case "object":
		m, ok := v.(map[string]any)
		if !ok {
			return bad
		}
		p, _ := s["properties"].(map[string]any)
		for _, r := range toStrings(s["required"]) {
			if _, ok := m[r]; !ok {
				return bad
			}
		}
		for k, x := range m {
			if child, ok := p[k]; ok {
				if err := validate(x, child.(map[string]any)); err != nil {
					return err
				}
			} else if child, ok := s["additionalProperties"].(map[string]any); ok {
				if err := validate(x, child); err != nil {
					return err
				}
			} else {
				return bad
			}
		}
	case "array":
		a, ok := v.([]any)
		if !ok {
			return bad
		}
		for _, x := range a {
			if err := validate(x, s["items"].(map[string]any)); err != nil {
				return err
			}
		}
	case "string":
		if _, ok := v.(string); !ok {
			return bad
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return bad
		}
	case "integer":
		n, ok := v.(json.Number)
		if !ok {
			return bad
		}
		if _, err := n.Int64(); err != nil {
			if _, err := strconv.ParseUint(string(n), 10, 64); err != nil {
				return bad
			}
		}
	case "number":
		if _, ok := v.(json.Number); !ok {
			return bad
		}
	case "null":
		if v != nil {
			return bad
		}
	}
	return nil
}
func toStrings(v any) []string { r, _ := v.([]string); return r }
