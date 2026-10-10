package capture

import (
	"bytes"
	"fmt"
	"go.yaml.in/yaml/v3"
	"io"
	"unicode/utf8"
)

// Decode accepts one closed YAML record without aliases, merges or extra docs.
func Decode(data []byte, out any) error {
	if len(data) == 0 || len(data) > 8<<20 || !utf8.Valid(data) {
		return fmt.Errorf("record must be nonempty UTF-8, at most 8 MiB")
	}
	d := yaml.NewDecoder(bytes.NewReader(data))
	var n yaml.Node
	if err := d.Decode(&n); err != nil {
		return err
	}
	var inspect func(*yaml.Node) error
	inspect = func(n *yaml.Node) error {
		if n.Kind == yaml.AliasNode || n.Anchor != "" || n.Tag == "!!merge" {
			return fmt.Errorf("aliases/anchors/merges are not allowed")
		}
		switch n.Tag {
		case "", "!!map", "!!seq", "!!str", "!!int", "!!bool", "!!null", "!!float", "!!timestamp":
		default:
			return fmt.Errorf("unsupported YAML tag %q", n.Tag)
		}
		for _, c := range n.Content {
			if err := inspect(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := inspect(&n); err != nil {
		return err
	}
	var extra yaml.Node
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("record must contain exactly one YAML document")
	}
	strict := yaml.NewDecoder(bytes.NewReader(data))
	strict.KnownFields(true)
	return strict.Decode(out)
}

func Encode(v any) ([]byte, error) {
	var b bytes.Buffer
	e := yaml.NewEncoder(&b)
	e.SetIndent(2)
	if err := e.Encode(v); err != nil {
		return nil, err
	}
	if err := e.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
