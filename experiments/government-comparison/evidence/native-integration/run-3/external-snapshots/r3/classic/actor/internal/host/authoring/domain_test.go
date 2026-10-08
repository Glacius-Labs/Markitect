package authoring

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
)

const testDomainYAML = `apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata:
  name: software
spec:
  apiVersion: software.markitect.org/v1alpha1
  kinds:
    Module:
      required: [intent]
      inputsField: files
      properties:
        intent: {type: string}
        files: {type: array, items: {type: string}}
        dependsOn: {type: array, items: {type: ref, refKind: Module}}
  relations:
    dependsOn: {field: dependsOn, sourceKinds: [Module], targetKinds: [Module], context: true, invalidate: true, acyclic: true}
  constraints:
    - name: module-intent
      select: {kind: Module}
      assert: {op: present, field: intent}
`

func TestParseDomainRoundTripsAndCompilesGenericResource(t *testing.T) {
	domain, err := ParseDomain("domains/software.yaml", []byte(testDomainYAML))
	if err != nil {
		t.Fatal(err)
	}
	if domain.Name != "software" || domain.Path != "domains/software.yaml" || domain.Line != 1 {
		t.Fatalf("domain source metadata = %#v", domain)
	}
	encoded, err := EncodeDomain(domain)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte("apiVersion: software.markitect.org/v1alpha1")) {
		t.Fatalf("encoded Domain omitted its domain API version:\n%s", encoded)
	}
	registry := core.NewRegistry()
	if err := registry.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	resourceYAML := []byte(`apiVersion: software.markitect.org/v1alpha1
kind: Module
metadata:
  name: orders
  namespace: engineering
spec:
  intent: Owns the order workflow.
  files: [src/Orders/Orders.csproj]
`)
	resource, err := ParseWithRegistry("resources/orders.yaml", resourceYAML, registry)
	if err != nil {
		t.Fatal(err)
	}
	if resource.Data["intent"] != "Owns the order workflow." {
		t.Fatalf("normalized spec = %#v", resource.Data)
	}
	if !reflect.DeepEqual(resource.Spec.Files, []string{"src/Orders/Orders.csproj"}) {
		t.Fatalf("declared opaque input files = %#v", resource.Spec.Files)
	}
	serialized, err := Encode(*resource)
	if err != nil {
		t.Fatal(err)
	}
	reparsed, err := ParseWithRegistry("resources/orders.yaml", serialized, registry)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resource.Data, reparsed.Data) {
		t.Fatalf("generic Data roundtrip changed: %#v -> %#v", resource.Data, reparsed.Data)
	}
}

func TestParseWithRegistryRejectsUnknownKindsAndFields(t *testing.T) {
	domain, err := ParseDomain("domains/software.yaml", []byte(testDomainYAML))
	if err != nil {
		t.Fatal(err)
	}
	registry := core.NewRegistry()
	if err := registry.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		source string
	}{
		{name: "unknown kind", source: `apiVersion: software.markitect.org/v1alpha1
kind: Component
metadata: {name: x}
spec: {intent: x}`},
		{name: "unknown field", source: `apiVersion: software.markitect.org/v1alpha1
kind: Module
metadata: {name: x}
spec: {intent: x, unknown: true}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseWithRegistry("resource.yaml", []byte(test.source), registry); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestParseWithRegistryRejectsDomainDefinitionsAsOrdinaryResources(t *testing.T) {
	if _, err := ParseWithRegistry("domains/software.yaml", []byte(testDomainYAML), core.NewRegistry()); err == nil || !strings.Contains(err.Error(), "select this file through project.spec.domains or package.spec.domains") {
		t.Fatalf("ordinary resource scan should require explicit Domain selection, got %v", err)
	}
}

func TestSchemasWithRegistryExposeDomainAndRegisteredKinds(t *testing.T) {
	domain, err := ParseDomain("domains/software.yaml", []byte(testDomainYAML))
	if err != nil {
		t.Fatal(err)
	}
	registry := core.NewRegistry()
	if err := registry.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	schemas, err := SchemasWithRegistry(registry)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := schemas["schema/domains/software.markitect.org-v1alpha1-Module.yaml"]; !ok {
		t.Fatalf("domain kind schema missing: %#v", schemas)
	}
	if _, ok := schemas["schema/Domain.yaml"]; !ok {
		t.Fatal("Domain descriptor schema missing")
	}
}
