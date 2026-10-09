package canonical

import (
	"fmt"
	"strings"
	"testing"
)

func makeManifest(name, version, schemas, projectors, dependencies, discovery string) []byte {
	moduleType := "projection"
	if strings.TrimSpace(schemas) != "" {
		moduleType = "schema"
	}
	if schemas == "" {
		schemas = "  schemas: []\n"
	} else {
		schemas = "  schemas:\n" + schemas
	}
	if projectors == "" {
		projectors = "  projectors: []\n"
	} else {
		projectors = "  projectors:\n" + projectors
	}
	if dependencies == "" {
		dependencies = "  core: \"1\"\n"
	} else {
		dependencies = "  core: \"1\"\n  modules:\n" + dependencies
	}
	if discovery != "" {
		discovery = "discovery:\n" + discovery
	}
	return []byte(fmt.Sprintf(`apiVersion: %s
name: %s
version: %s
type: %s
purpose: Module capability fixture.
requires:
%sprovides:
%s%s`, ModuleManifestAPIVersion, name, version, moduleType, dependencies, schemas+projectors, discovery))
}

func schemaPackage(name, version, schemaPath string) ModulePackage {
	paths := "    - " + schemaPath + "\n"
	return ModulePackage{ManifestBytes: makeManifest(name, version, paths, "", "", ""), Files: map[string][]byte{schemaPath: []byte(schemaYAML)}}
}

func projectorYAML(id, allKinds string) string {
	return fmt.Sprintf(`    - id: %s
      version: 1.0.0
      target: markdown
      allKinds: %s
      allowedRoots:
        - docs/generated
`, id, allKinds)
}

func exactPin(t *testing.T, pkg ModulePackage) Pin {
	t.Helper()
	manifest, err := DecodeManifest(pkg.ManifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := DigestPackage(pkg)
	if err != nil {
		t.Fatal(err)
	}
	return Pin{Name: manifest.Name, Version: manifest.Version, Digest: digest}
}

func TestDecodeManifestDefinesClosedVersionedModuleProtocol(t *testing.T) {
	body := makeManifest("markitect-docs", "1.2.0", "", projectorYAML("markdown", "true"), "", "  goals:\n    - generate markdown\n  paths:\n    - docs/architecture.md\n")
	got, err := DecodeManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.APIVersion != ModuleManifestAPIVersion || got.Requires.Core != "1" || len(got.Provides.Projectors) != 1 || !got.Provides.Projectors[0].AllKinds {
		t.Fatalf("manifest decode = %#v", got)
	}
	if got.Type != ModuleTypeProjection || got.Provides.Projectors[0].ID != "markdown" {
		t.Fatalf("projector = %#v", got.Provides.Projectors[0])
	}
}

func TestManifestRejectsUnknownFieldsRangesAndExecutableRegistration(t *testing.T) {
	valid := string(makeManifest("markitect-docs", "1.0.0", "", projectorYAML("markdown", "true"), "", ""))
	for name, body := range map[string]string{
		"unknown executable":  strings.Replace(valid, "  projectors:", "  executable: ./run\n  projectors:", 1),
		"core range":          strings.Replace(valid, "core: \"1\"", "core: \">=1\"", 1),
		"unknown public mode": strings.Replace(valid, "      target: markdown", "      target: markdown\n      mode: deterministic", 1),
		"missing capability":  strings.Replace(valid, "  projectors:\n    -", "  projectors: []\n  schemas: []\n  ignored:\n    -", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeManifest([]byte(body)); err == nil {
				t.Fatal("expected closed protocol rejection")
			}
		})
	}
	noAuthority := string(makeManifest("markitect-docs", "1.0.0", "", projectorYAML("markdown", "false"), "", ""))
	if _, err := DecodeManifest([]byte(noAuthority)); err == nil || !strings.Contains(err.Error(), "supportedKinds") {
		t.Fatalf("implicit all-Kinds registration error = %v", err)
	}
}

func TestPackageDigestBindsExactManifestSchemaBytesAndMode(t *testing.T) {
	pkg := schemaPackage("markitect-foundation", "1.0.0", "schemas/foundation.yaml")
	one, err := DigestPackage(pkg)
	if err != nil {
		t.Fatal(err)
	}
	pkg.Modes = map[string]string{"module.yaml": "100644", "schemas/foundation.yaml": "100644"}
	two, err := DigestPackage(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if one != two {
		t.Fatalf("implicit/explicit regular mode digest differ: %s != %s", one, two)
	}
	pkg.Modes["schemas/foundation.yaml"] = "100755"
	if _, err := DigestPackage(pkg); err == nil {
		t.Fatal("executable Schema data mode should be rejected")
	}
	pkg.Modes["schemas/foundation.yaml"] = "100644"
	pkg.Modes["module.yaml"] = "100755"
	if _, err := DigestPackage(pkg); err == nil {
		t.Fatal("executable manifest mode should be rejected")
	}
	pkg.Modes = nil
	pkg.Files["schemas/extra.yaml"] = []byte(schemaYAML)
	if _, err := DigestPackage(pkg); err == nil {
		t.Fatal("undeclared content should be rejected")
	}
	delete(pkg.Files, "schemas/extra.yaml")
	delete(pkg.Files, "schemas/foundation.yaml")
	if _, err := DigestPackage(pkg); err == nil {
		t.Fatal("missing declared Schema should be rejected")
	}
}

func TestResolveUsesOnlyExactSelectedPinsAndPreservesProjectorOrigin(t *testing.T) {
	base := schemaPackage("markitect-foundation", "1.0.0", "schemas/foundation.yaml")
	basePin := exactPin(t, base)
	dependency := fmt.Sprintf("    - name: %s\n      version: %s\n      digest: %s\n", basePin.Name, basePin.Version, basePin.Digest)
	projectors := projectorYAML("generic-markdown", "true")
	docs := ModulePackage{ManifestBytes: makeManifest("markitect-markdown", "1.0.0", "", projectors, dependency, ""), Files: map[string][]byte{}}
	docsPin := exactPin(t, docs)
	unselected := schemaPackage("unselected", "9.0.0", "schemas/unselected.yaml")

	activation, err := Resolve([]ModulePackage{docs, unselected, base}, []Pin{docsPin, basePin})
	if err != nil {
		t.Fatal(err)
	}
	if len(activation.Modules) != 2 || activation.Modules[0].Name != basePin.Name || activation.Modules[1] != docsPin {
		t.Fatalf("activation modules = %#v", activation.Modules)
	}
	if len(activation.Schemas) != 1 || activation.Schemas[0].APIVersion != "architecture.example.org/v1" {
		t.Fatalf("activated schemas = %#v", activation.Schemas)
	}
	if activation.ModuleTypes[basePin] != ModuleTypeSchema || activation.ModuleTypes[docsPin] != ModuleTypeProjection {
		t.Fatalf("activation module classes = %#v", activation.ModuleTypes)
	}
	if len(activation.Projectors) != 1 {
		t.Fatalf("registered projectors = %#v", activation.Projectors)
	}
	registered := activation.Projectors[0]
	if registered.Module != docsPin || registered.Registration.ID != "generic-markdown" || !registered.Registration.AllKinds {
		t.Fatalf("registered capability lost exact origin: %#v", registered)
	}
	// Activation contains only schema and capability registrations. It has no
	// scope, target binding, Plan, or execution result to invoke implicitly.
}

func TestResolveRejectsUnselectedDependenciesAndUnresolvedRegisteredKinds(t *testing.T) {
	base := schemaPackage("markitect-foundation", "1.0.0", "schemas/foundation.yaml")
	basePin := exactPin(t, base)
	dependency := fmt.Sprintf("    - name: %s\n      version: %s\n      digest: %s\n", basePin.Name, basePin.Version, basePin.Digest)
	dependent := ModulePackage{ManifestBytes: makeManifest("markitect-dependent", "1.0.0", "", projectorYAML("markdown", "true"), dependency, ""), Files: map[string][]byte{}}
	dependentPin := exactPin(t, dependent)
	if _, err := Resolve([]ModulePackage{base, dependent}, []Pin{dependentPin}); err == nil || !strings.Contains(err.Error(), "explicitly selected") {
		t.Fatalf("missing selected dependency = %v", err)
	}
	if _, err := Resolve([]ModulePackage{base, dependent}, []Pin{dependentPin, {Name: basePin.Name, Version: basePin.Version, Digest: "sha256:" + strings.Repeat("0", 64)}}); err == nil {
		t.Fatal("wrong exact dependency digest should fail")
	}

	registration := `    - id: only-architecture
      version: 1.0.0
      target: source
      supportedKinds:
        - apiVersion: missing.example.org/v1
          kind: Missing
      allowedRoots:
        - src
`
	projector := ModulePackage{ManifestBytes: makeManifest("markitect-source", "1.0.0", "", registration, "", ""), Files: map[string][]byte{}}
	if _, err := Resolve([]ModulePackage{projector}, []Pin{exactPin(t, projector)}); err == nil || !strings.Contains(err.Error(), "unresolved Kind") {
		t.Fatalf("unresolved registered Kind = %v", err)
	}
}

func TestManifestRequiresOneOfTwoDisjointModuleTypes(t *testing.T) {
	schema := string(makeManifest("schema-only", "1.0.0", "    - schemas/core.yaml\n", "", "", ""))
	decoded, err := DecodeManifest([]byte(schema))
	if err != nil || decoded.Type != ModuleTypeSchema {
		t.Fatalf("schema Module manifest type=%q err=%v", decoded.Type, err)
	}
	projection := string(makeManifest("projection-only", "1.0.0", "", projectorYAML("render", "true"), "", ""))
	decoded, err = DecodeManifest([]byte(projection))
	if err != nil || decoded.Type != ModuleTypeProjection {
		t.Fatalf("projection Module manifest type=%q err=%v", decoded.Type, err)
	}
	for name, body := range map[string]string{
		"unknown type":                 strings.Replace(schema, "type: schema", "type: bundle", 1),
		"mixed type":                   strings.Replace(schema, "projectors: []", strings.TrimSuffix(projectorYAML("render", "true"), "\n"), 1),
		"schema without schema":        strings.Replace(schema, "    - schemas/core.yaml\n", "", 1),
		"projection without projector": strings.Replace(projection, strings.TrimSuffix(projectorYAML("render", "true"), "\n"), "", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeManifest([]byte(body)); err == nil {
				t.Fatal("expected invalid Module type/capability combination")
			}
		})
	}
}

func TestSchemaModuleCannotDependOnProjectionModule(t *testing.T) {
	projection := ModulePackage{ManifestBytes: makeManifest("renderer", "1.0.0", "", projectorYAML("render", "true"), "", ""), Files: map[string][]byte{}}
	projectionPin := exactPin(t, projection)
	dep := fmt.Sprintf("    - name: %s\n      version: %s\n      digest: %s\n", projectionPin.Name, projectionPin.Version, projectionPin.Digest)
	schema := ModulePackage{ManifestBytes: makeManifest("vocabulary", "1.0.0", "    - schemas/vocabulary.yaml\n", "", dep, ""), Files: map[string][]byte{"schemas/vocabulary.yaml": []byte(schemaYAML)}}
	schemaPin := exactPin(t, schema)
	if _, err := Resolve([]ModulePackage{projection, schema}, []Pin{projectionPin, schemaPin}); err == nil || !strings.Contains(err.Error(), "cannot depend on projection") {
		t.Fatalf("schema to projection dependency = %v", err)
	}
}

func TestResolveAllowsSameProjectorIDFromDistinctPinnedModules(t *testing.T) {
	first := ModulePackage{ManifestBytes: makeManifest("markitect-markdown", "1.0.0", "", projectorYAML("render", "true"), "", ""), Files: map[string][]byte{}}
	second := ModulePackage{ManifestBytes: makeManifest("markitect-markdown-alt", "1.0.0", "", projectorYAML("render", "true"), "", ""), Files: map[string][]byte{}}
	pins := []Pin{exactPin(t, first), exactPin(t, second)}
	activation, err := Resolve([]ModulePackage{second, first}, pins)
	if err != nil {
		t.Fatal(err)
	}
	if len(activation.Projectors) != 2 || activation.Projectors[0].Registration.ID != "render" || activation.Projectors[1].Registration.ID != "render" {
		t.Fatalf("projector identities = %#v", activation.Projectors)
	}
	if activation.Projectors[0].Module.Name == activation.Projectors[1].Module.Name {
		t.Fatal("projector registration origins were flattened")
	}
}

func TestRecommendUsesOnlyExplicitGoalsAndSuppliedPathInventory(t *testing.T) {
	pkg := ModulePackage{ManifestBytes: makeManifest("markitect-dotnet", "1.0.0", "    - schemas/not-read.yaml\n", "", "", "  goals:\n    - dotnet\n    - csharp\n  paths:\n    - src/App.sln\n"), Files: map[string][]byte{"schemas/not-read.yaml": []byte("not valid Schema and never decoded")}}
	got, err := Recommend([]ModulePackage{pkg}, []string{"Please use dotnet for this project"}, []string{"src/App.sln"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "markitect-dotnet" || len(got[0].MatchedGoals) != 1 || len(got[0].MatchedPaths) != 1 {
		t.Fatalf("recommendations = %#v", got)
	}
	if _, err := Recommend([]ModulePackage{pkg}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Recommend([]ModulePackage{pkg}, nil, []string{"../secret.csproj"}); err == nil {
		t.Fatal("unsafe inventory path should be rejected")
	}
}
