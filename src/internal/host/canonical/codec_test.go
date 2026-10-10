package canonical

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

const schemaYAML = `apiVersion: architecture.example.org/v1
purpose: Describes bounded application architecture intent.
kinds:
  Module:
    purpose: One independently understandable application unit.
    properties:
      name:
        purpose: The stable human-readable name.
        type: string
        minCount: 1
        maxCount: 1
      responsibilities:
        purpose: Explicit responsibilities owned by this unit.
        type: string
        minCount: 0
        maxCount: unbounded
`

func TestDecodeSchemaCompilesAndBindsRawProvenance(t *testing.T) {
	got, err := DecodeSchema("schemas/architecture.yaml", []byte(schemaYAML))
	if err != nil {
		t.Fatal(err)
	}
	if got.Source.Path != "schemas/architecture.yaml" || got.Source.Digest != digestBytes([]byte(schemaYAML)) || got.Source.Line != 1 {
		t.Fatalf("source provenance = %#v", got.Source)
	}
	if got.Kinds["Module"].Properties["responsibilities"].MaxCount != core.Unbounded {
		t.Fatalf("unbounded maxCount = %d", got.Kinds["Module"].Properties["responsibilities"].MaxCount)
	}
	model, diagnostics := core.Compile([]core.Schema{got}, nil, "")
	if len(diagnostics) != 0 {
		t.Fatalf("Core rejected decoded schema: %#v", diagnostics)
	}
	if len(model.Schemas) != 1 || model.Schemas[0].APIVersion != got.APIVersion {
		t.Fatalf("compiled schema = %#v", model.Schemas)
	}
}

func TestDecodeDefinitionPreservesNativeScalarTypesAndExplicitNamespace(t *testing.T) {
	data := []byte(`apiVersion: architecture.example.org/v1
kind: Module
metadata:
  namespace: ""
  name: checkout
purpose: Owns checkout behavior.
spec:
  name: Checkout
  active: false
  count: 3
`)
	got, err := DecodeDefinition("definitions/checkout.yaml", data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Namespace != "" || got.Metadata.Name != "checkout" || got.Source.Digest != digestBytes(data) {
		t.Fatalf("decoded definition = %#v", got)
	}
	if value, ok := got.Spec["active"].(bool); !ok || value {
		t.Fatalf("active should be native false bool, got %#v", got.Spec["active"])
	}
	if value, ok := got.Spec["count"].(int); !ok || value != 3 {
		t.Fatalf("count should be native integer, got %#v", got.Spec["count"])
	}
}

func TestStrictSourceCodecsRejectUnsafeRepresentations(t *testing.T) {
	tests := []struct {
		name, body string
		decode     func([]byte) error
	}{
		{"unknown schema field", strings.Replace(schemaYAML, "purpose: Describes", "extra: false\npurpose: Describes", 1), func(b []byte) error { _, e := DecodeSchema("schema.yaml", b); return e }},
		{"duplicate key", strings.Replace(schemaYAML, "purpose: Describes", "purpose: first\npurpose: second", 1), func(b []byte) error { _, e := DecodeSchema("schema.yaml", b); return e }},
		{"alias", strings.Replace(schemaYAML, "purpose: Describes", "purpose: &p Describes", 1) + "copy: *p\n", func(b []byte) error { _, e := DecodeSchema("schema.yaml", b); return e }},
		{"merge key", strings.Replace(schemaYAML, "kinds:\n", "kinds:\n  <<: {Other: {purpose: bad, properties: {}}}\n", 1), func(b []byte) error { _, e := DecodeSchema("schema.yaml", b); return e }},
		{"tag", strings.Replace(schemaYAML, "purpose: Describes", "purpose: !!str Describes", 1), func(b []byte) error { _, e := DecodeSchema("schema.yaml", b); return e }},
		{"extra document", schemaYAML + "---\nother: true\n", func(b []byte) error { _, e := DecodeSchema("schema.yaml", b); return e }},
		{"missing namespace", "apiVersion: a.example.org/v1\nkind: K\nmetadata: {name: n}\npurpose: p\nspec: {}\n", func(b []byte) error { _, e := DecodeDefinition("d.yaml", b); return e }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.decode([]byte(tc.body)); err == nil {
				t.Fatal("expected strict codec error")
			}
		})
	}
}

func TestSchemaCodecRejectsUntypedOrMissingCounts(t *testing.T) {
	for name, body := range map[string]string{
		"quoted numeric max": strings.Replace(schemaYAML, "maxCount: 1", "maxCount: \"1\"", 1),
		"missing minimum":    strings.Replace(schemaYAML, "minCount: 1\n", "", 1),
		"negative maximum":   strings.Replace(schemaYAML, "maxCount: 1", "maxCount: -1", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeSchema("schema.yaml", []byte(body)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
