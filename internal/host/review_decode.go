package host

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// DecodeReviewConfig strictly decodes one bounded YAML review configuration.
func DecodeReviewConfig(data []byte) (ReviewConfig, error) {
	var config ReviewConfig
	if err := strictReviewDecode(data, maxReviewConfigSize, &config, "question", "promptVersion", "model", "effort", "allowReuse"); err != nil {
		return config, err
	}
	if err := validateReviewConfig(config); err != nil {
		return config, err
	}
	return config, nil
}

// DecodeReviewRecord strictly decodes one bounded YAML review record and
// validates its stored report digest and advisory marker.
func DecodeReviewRecord(data []byte) (*ReviewRecord, error) {
	var record ReviewRecord
	if err := strictReviewDecode(data, maxReviewRecordSize, &record,
		"schemaVersion", "entry", "revision", "snapshotDigest", "contextDigest", "toolDigest", "version", "config", "report", "reportDigest", "recordedAt", "advisory", "trustNotice"); err != nil {
		return nil, err
	}
	record.RecordedAt = record.RecordedAt.UTC()
	if err := requireNestedReviewKeys(data, "config", "question", "promptVersion", "model", "effort", "allowReuse"); err != nil {
		return nil, err
	}
	if err := validateReviewRecord(&record); err != nil {
		return nil, err
	}
	return &record, nil
}

func strictReviewDecode(data []byte, max int, out any, required ...string) error {
	if len(data) == 0 || len(data) > max || !utf8.Valid(data) {
		return fmt.Errorf("review YAML must be valid UTF-8, nonempty, and no larger than %d bytes", max)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := decoder.Decode(&doc); err != nil {
		return fmt.Errorf("invalid review YAML: %w", err)
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return errors.New("review YAML root must be one mapping")
	}
	if err := inspectReviewNode(&doc); err != nil {
		return err
	}
	if err := requireReviewKeys(doc.Content[0], required...); err != nil {
		return err
	}
	if err := validateReviewScalarTypes(doc.Content[0], out); err != nil {
		return err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple review YAML documents are not allowed")
		}
		return fmt.Errorf("invalid review YAML: %w", err)
	}
	strict := yaml.NewDecoder(bytes.NewReader(data))
	strict.KnownFields(true)
	if err := strict.Decode(out); err != nil {
		return fmt.Errorf("invalid review YAML fields: %w", err)
	}
	return nil
}

func validateReviewScalarTypes(root *yaml.Node, out any) error {
	fields := map[string]string{}
	if _, ok := out.(*ReviewConfig); ok {
		fields = map[string]string{"question": "!!str", "promptVersion": "!!str", "model": "!!str", "effort": "!!str", "allowReuse": "!!bool"}
	} else {
		fields = map[string]string{
			"schemaVersion": "!!int", "entry": "!!str", "revision": "!!str", "snapshotDigest": "!!str",
			"contextDigest": "!!str", "toolDigest": "!!str", "version": "!!str", "report": "!!str",
			"reportDigest": "!!str", "advisory": "!!bool", "trustNotice": "!!str",
		}
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value == "recordedAt" {
				n := root.Content[i+1]
				if n.Kind != yaml.ScalarNode || (n.Tag != "!!timestamp" && n.Tag != "!!str") {
					return errors.New("review YAML field \"recordedAt\" must be a timestamp or string")
				}
				if strings.ContainsRune(n.Value, 0) {
					return errors.New("review YAML field \"recordedAt\" contains NUL")
				}
			}
			if root.Content[i].Value == "config" {
				if err := validateReviewConfigNode(root.Content[i+1]); err != nil {
					return err
				}
			}
		}
	}
	for field, tag := range fields {
		var value *yaml.Node
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value == field {
				value = root.Content[i+1]
				break
			}
		}
		if value == nil || value.Kind != yaml.ScalarNode || value.Tag != tag {
			return fmt.Errorf("review YAML field %q must be a %s scalar", field, strings.TrimPrefix(tag, "!!"))
		}
		if strings.ContainsRune(value.Value, 0) {
			return fmt.Errorf("review YAML field %q contains NUL", field)
		}
	}
	return nil
}

func validateReviewConfigNode(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.New("review YAML field \"config\" must be a mapping")
	}
	if err := requireReviewKeys(node, "question", "promptVersion", "model", "effort", "allowReuse"); err != nil {
		return err
	}
	fields := map[string]string{"question": "!!str", "promptVersion": "!!str", "model": "!!str", "effort": "!!str", "allowReuse": "!!bool"}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		value := node.Content[i+1]
		if value.Kind != yaml.ScalarNode || value.Tag != fields[key] {
			return fmt.Errorf("review YAML config field %q has the wrong scalar type", key)
		}
		if strings.ContainsRune(value.Value, 0) {
			return fmt.Errorf("review YAML config field %q contains NUL", key)
		}
	}
	return nil
}

func requireNestedReviewKeys(data []byte, parent string, keys ...string) error {
	var doc yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		return fmt.Errorf("invalid review YAML: %w", err)
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return errors.New("review YAML root must be one mapping")
	}
	root := doc.Content[0]
	var nested *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == parent {
			nested = root.Content[i+1]
			break
		}
	}
	if nested == nil || nested.Kind != yaml.MappingNode {
		return fmt.Errorf("review YAML field %q must be a mapping", parent)
	}
	return requireReviewKeys(nested, keys...)
}

func requireReviewKeys(node *yaml.Node, keys ...string) error {
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		seen[node.Content[i].Value] = true
	}
	for _, key := range keys {
		if !seen[key] {
			return fmt.Errorf("review YAML is missing required field %q", key)
		}
	}
	return nil
}

func inspectReviewNode(node *yaml.Node) error {
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return errors.New("review YAML anchors and aliases are not allowed")
	}
	if node.Tag != "" && node.Tag != "!!map" && node.Tag != "!!seq" && node.Tag != "!!str" && node.Tag != "!!int" && node.Tag != "!!float" && node.Tag != "!!bool" && node.Tag != "!!null" && node.Tag != "!!timestamp" && node.Tag != "!!binary" {
		return errors.New("custom YAML tags are not allowed in review YAML")
	}
	if node.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return errors.New("review YAML mapping keys must be strings")
			}
			if seen[key.Value] {
				return fmt.Errorf("duplicate review YAML key %q", key.Value)
			}
			seen[key.Value] = true
		}
	}
	for _, child := range node.Content {
		if err := inspectReviewNode(child); err != nil {
			return err
		}
	}
	return nil
}
