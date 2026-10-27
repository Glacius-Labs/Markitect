package runner

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

const maxRuntimeProfileBytes = 1 << 20

// TranslateRuntimeProfile returns the owner's runtime profile as the runner
// reads it: every engineExecutable, which must name hostEngine (the run
// configuration's engine, as Apply already requires), names the runner's own
// engine CLI instead. Nothing else changes. Instruction boundaries must be
// repository-relative, because the runner mounts no further host directory;
// anchors and aliases are refused so that one edit cannot change another value.
func TranslateRuntimeProfile(data []byte, hostEngine string) ([]byte, error) {
	if len(data) == 0 || len(data) > maxRuntimeProfileBytes {
		return nil, fmt.Errorf("runtime profile is empty or exceeds %d bytes", maxRuntimeProfileBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("runtime profile: %w", err)
	}
	var next yaml.Node
	if err := decoder.Decode(&next); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("runtime profile must be exactly one YAML document")
	}
	engines := 0
	if err := translateNode(&document, hostEngine, &engines); err != nil {
		return nil, err
	}
	if engines == 0 {
		return nil, fmt.Errorf("runtime profile names no engineExecutable")
	}
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func translateNode(n *yaml.Node, hostEngine string, engines *int) error {
	if n.Kind == yaml.AliasNode || n.Anchor != "" {
		return fmt.Errorf("runtime profile anchors and aliases are not supported by the runner")
	}
	if n.Kind != yaml.MappingNode {
		for _, child := range n.Content {
			if err := translateNode(child, hostEngine, engines); err != nil {
				return err
			}
		}
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		if key.Kind == yaml.ScalarNode && key.Value == "engineExecutable" {
			if value.Kind != yaml.ScalarNode || value.Anchor != "" || value.Value != hostEngine {
				return fmt.Errorf("every runtime profile engineExecutable must name the run configuration's engineExecutable")
			}
			value.Value, value.Tag, value.Style = EngineBinary, "!!str", 0
			*engines++
			continue
		}
		if key.Kind == yaml.ScalarNode && key.Value == "instructions" && value.Kind == yaml.SequenceNode {
			if err := checkInstructionBoundaries(value); err != nil {
				return err
			}
		}
		if err := translateNode(key, hostEngine, engines); err != nil {
			return err
		}
		if err := translateNode(value, hostEngine, engines); err != nil {
			return err
		}
	}
	return nil
}

func checkInstructionBoundaries(list *yaml.Node) error {
	for _, item := range list.Content {
		if item.Kind != yaml.MappingNode {
			continue // Apply's own profile validation reports it.
		}
		for i := 0; i+1 < len(item.Content); i += 2 {
			if item.Content[i].Value == "boundary" && !RepositoryRelative(item.Content[i+1].Value, true) {
				return fmt.Errorf("runtime instruction boundary %q is not repository-relative; the runner mounts no external instruction directory", item.Content[i+1].Value)
			}
		}
	}
	return nil
}
