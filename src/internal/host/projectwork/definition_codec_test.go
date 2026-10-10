package projectwork

import (
	"strings"
	"testing"
)

const definitionYAML = `apiVersion: architecture.example.org/v1
kind: Module
metadata:
  namespace: ""
  name: checkout
purpose: Owns checkout behavior.
spec:
  name: Checkout
  active: false
  count: 3
`

func TestDecodeDefinitionPreservesNativeScalarTypesAndExplicitNamespace(t *testing.T) {
	data := []byte(definitionYAML)
	got, err := decodeDefinition("definitions/checkout.yaml", data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Namespace != "" || got.Metadata.Name != "checkout" || got.Source.Path != "definitions/checkout.yaml" || got.Source.Digest != digestBytes(data) || got.Source.Line != 1 {
		t.Fatalf("decoded definition = %#v", got)
	}
	if value, ok := got.Spec["active"].(bool); !ok || value {
		t.Fatalf("active should be native false bool, got %#v", got.Spec["active"])
	}
	if value, ok := got.Spec["count"].(int); !ok || value != 3 {
		t.Fatalf("count should be native integer, got %#v", got.Spec["count"])
	}
}

func TestStrictDefinitionCodecRejectsUnsafeRepresentations(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"empty document", ""},
		{"unknown field", strings.Replace(definitionYAML, "purpose: Owns", "extra: false\npurpose: Owns", 1)},
		{"duplicate key", strings.Replace(definitionYAML, "purpose: Owns", "purpose: first\npurpose: Owns", 1)},
		{"alias", strings.Replace(definitionYAML, "name: Checkout", "name: &n Checkout\n  copy: *n", 1)},
		{"merge key", strings.Replace(definitionYAML, "spec:\n", "spec:\n  <<: {other: true}\n", 1)},
		{"tag", strings.Replace(definitionYAML, "purpose: Owns", "purpose: !!str Owns", 1)},
		{"extra document", definitionYAML + "---\nother: true\n"},
		{"missing namespace", "apiVersion: a.example.org/v1\nkind: K\nmetadata: {name: n}\npurpose: p\nspec: {}\n"},
		{"non-mapping spec", strings.Replace(definitionYAML, "spec:\n  name: Checkout\n  active: false\n  count: 3\n", "spec: []\n", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decodeDefinition("d.yaml", []byte(tc.body)); err == nil {
				t.Fatal("expected strict codec error")
			}
		})
	}
}
