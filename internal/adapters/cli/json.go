package cli

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ReportValue projects a report into plain JSON values, exactly as the command
// prints it, so retained copies of a report match the printed one.
func ReportValue(value any) any { return reportValue(value) }

// Producer value types deliberately keep their internals private. Translate
// those exact types at the presentation boundary instead of encoding them as
// empty objects or adding serialization contracts to the pure producers.
func reportValue(value any) any {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case ApplyReport:
		// Outcome is a projection of the command status, never separately
		// mutable state enriched only after the evidence file was written.
		type fields ApplyReport
		out := reportValue(fields(v)).(map[string]any)
		out["outcome"] = RunOutcome(v.Status)
		return out
	case snapshot.Path:
		return v.String()
	case snapshot.DirectoryBoundary:
		return v.String()
	case snapshot.ContentID:
		return v.String()
	case snapshot.TreeID:
		return v.String()
	case model.Value:
		return valueReport(v)
	case realization.Owner:
		if v.IsRoot() {
			return "."
		}
		if name, ok := v.Package(); ok {
			return name
		}
		return nil
	case realization.Registration:
		artifact := v.Artifact()
		return map[string]any{"owner": reportValue(v.Owner()), "artifact": map[string]any{"path": artifact.Path().String(), "form": artifact.Form(), "realizes": reportValue(artifact.Basis()), "why": artifact.Why()}}
	case time.Duration:
		return v.String()
	case time.Time:
		return v.Format(time.RFC3339Nano)
	case []byte:
		return value
	case json.RawMessage:
		// An embedded report another Markitect process already projected,
		// such as the runner's Apply report, passes through unchanged.
		if len(v) == 0 {
			return nil
		}
		return v
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return reportValue(v.Elem().Interface())
	case reflect.Struct:
		out := map[string]any{}
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			name := field.Name
			tag := strings.Split(field.Tag.Get("json"), ",")
			if tag[0] == "-" {
				continue
			}
			if tag[0] != "" {
				name = tag[0]
			}
			omit := false
			for _, option := range tag[1:] {
				omit = omit || option == "omitempty"
			}
			if omit && v.Field(i).IsZero() {
				continue
			}
			out[name] = reportValue(v.Field(i).Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return nil
		}
		out := make([]any, v.Len())
		for i := range out {
			out[i] = reportValue(v.Index(i).Interface())
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		out := map[string]any{}
		iter := v.MapRange()
		for iter.Next() {
			if iter.Key().Kind() == reflect.String {
				out[iter.Key().String()] = reportValue(iter.Value().Interface())
			}
		}
		return out
	}
	return value
}
