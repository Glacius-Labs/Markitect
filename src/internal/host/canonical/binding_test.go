package canonical

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"go.yaml.in/yaml/v3"
)

const bindingFixtureRoot = "../../../../examples/canonical-projection"

func TestProjectionBindingHasRuntimeJSONAndYAMLShape(t *testing.T) {
	want := ProjectionBinding{
		Projection: core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"},
		Module:     "markitect-dotnet",
	}
	jsonBytes, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var fromJSON ProjectionBinding
	if err := json.Unmarshal(jsonBytes, &fromJSON); err != nil {
		t.Fatal(err)
	}
	if fromJSON != want {
		t.Fatalf("JSON binding round trip = %#v", fromJSON)
	}
	yamlBytes, err := yaml.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var fromYAML ProjectionBinding
	if err := yaml.Unmarshal(yamlBytes, &fromYAML); err != nil {
		t.Fatal(err)
	}
	if fromYAML != want || !strings.Contains(string(yamlBytes), "projection:") || !strings.Contains(string(yamlBytes), "module: markitect-dotnet") {
		t.Fatalf("YAML binding round trip=%#v bytes=%s", fromYAML, yamlBytes)
	}
}

func TestBindProjectionUsesExplicitScopeAndDeterministicRequestDigest(t *testing.T) {
	model, activation := loadBindingFixture(t, "application-dotnet", nil)
	identity := core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"}
	targets := map[string][]byte{"src/Orders.cs": []byte("existing target state")}
	request, err := bindFixtureProjection(model, activation, identity, targets)
	if err != nil {
		t.Fatal(err)
	}
	if request.ModulePin.Name != "markitect-dotnet" || request.TargetRepository != "." || request.TargetPrefix != "src" {
		t.Fatalf("unexpected explicit binding: module=%+v target=%+v", request.ModulePin, request)
	}
	if request.Binding != (ProjectionBinding{Projection: identity, Module: "markitect-dotnet"}) {
		t.Fatalf("runtime binding provenance = %#v", request.Binding)
	}
	if len(request.Definitions) < 2 || len(request.Policies) < 2 || len(request.Edges) != 1 || len(request.ExternalEdges) != 0 {
		t.Fatalf("request scope was not explicit: definitions=%d policies=%d edges=%d external=%d", len(request.Definitions), len(request.Policies), len(request.Edges), len(request.ExternalEdges))
	}
	if len(request.TargetFiles) != 1 || string(request.TargetFiles["src/Orders.cs"]) != "existing target state" {
		t.Fatalf("target preimage was not supplied exactly: %#v", request.TargetFiles)
	}
	if !strings.HasPrefix(request.ModelDigest, "sha256:") || !strings.HasPrefix(request.RequestDigest, "sha256:") {
		t.Fatalf("request digests are not normalized: model=%q request=%q", request.ModelDigest, request.RequestDigest)
	}

	again, err := bindFixtureProjection(model, activation, identity, targets)
	if err != nil || again.RequestDigest != request.RequestDigest {
		t.Fatalf("same canonical inputs produced different request digest: %v %q %q", err, request.RequestDigest, again.RequestDigest)
	}
	changed, err := bindFixtureProjection(model, activation, identity, map[string][]byte{"src/Orders.cs": []byte("different target state")})
	if err != nil || changed.RequestDigest == request.RequestDigest {
		t.Fatalf("request digest did not bind target bytes: %v %q", err, changed.RequestDigest)
	}
	request.TargetFiles["src/Orders.cs"][0] = 'X'
	if string(targets["src/Orders.cs"]) != "existing target state" {
		t.Fatal("request target bytes alias the caller's target map")
	}
}

func TestBindProjectionDoesNotExpandExternalReferences(t *testing.T) {
	model, activation := loadBindingFixture(t, "application-dotnet", func(projection *core.Definition) {
		projection.Spec["source"].(map[string]any)["definitions"] = []any{identityValue("commerce.example.org/v1", "UseCase", "commerce", "create-order")}
	})
	identity := core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"}
	request, err := bindFixtureProjection(model, activation, identity, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Definitions) != 1 || len(request.Edges) != 0 || len(request.ExternalEdges) != 1 {
		t.Fatalf("scope closure was inferred: definitions=%d internal=%d external=%d", len(request.Definitions), len(request.Edges), len(request.ExternalEdges))
	}
	if got := request.ExternalEdges[0].Property; got != "handledBy" {
		t.Fatalf("external dependency was not identified: %#v", request.ExternalEdges[0])
	}
}

func TestBindProjectionSupportsManyToManyWithoutArtifactAssumptions(t *testing.T) {
	model, activation := loadBindingFixture(t, "application-dotnet", nil)
	identity := core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"}
	request, err := bindFixtureProjection(model, activation, identity, map[string][]byte{"src/one-combined-artifact.cs": []byte("one artifact")})
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Definitions) < 2 || len(request.TargetFiles) != 1 {
		t.Fatalf("binder assumed one artifact per Definition: definitions=%d files=%d", len(request.Definitions), len(request.TargetFiles))
	}
}

func TestBindProjectionRejectsMissingOrAmbiguousModuleSelection(t *testing.T) {
	model, activation := loadBindingFixture(t, "application-dotnet", nil)
	identity := core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"}

	if _, err := BindProjection(model, activation, nil, identity, nil); err == nil || !strings.Contains(err.Error(), "no runtime Module binding") {
		t.Fatalf("missing runtime binding was accepted: %v", err)
	}
	if _, err := BindProjection(model, activation, []ProjectionBinding{{Projection: identity, Module: "markitect-dotnet"}, {Projection: identity, Module: "markitect-dotnet"}}, identity, nil); err == nil || !strings.Contains(err.Error(), "duplicate runtime Module bindings") {
		t.Fatalf("duplicate runtime bindings were accepted: %v", err)
	}
	if _, err := bindFixtureProjection(model, activation, core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "not-selected"}, nil); err == nil {
		t.Fatal("Module registration alone produced a Projection request")
	}
	withoutProjectors := activation
	withoutProjectors.Projectors = nil
	if _, err := bindFixtureProjection(model, withoutProjectors, identity, nil); err == nil || !strings.Contains(err.Error(), "unavailable Projection Module") {
		t.Fatalf("missing Projection Module was accepted: %v", err)
	}
	wrongType := activation
	wrongType.ModuleTypes = make(map[Pin]string, len(activation.ModuleTypes))
	for pin, moduleType := range activation.ModuleTypes {
		wrongType.ModuleTypes[pin] = moduleType
	}
	for pin := range wrongType.ModuleTypes {
		if pin.Name == "markitect-dotnet" {
			wrongType.ModuleTypes[pin] = ModuleTypeSchema
		}
	}
	if _, err := bindFixtureProjection(model, wrongType, identity, nil); err == nil || !strings.Contains(err.Error(), "not registered as a Projection Module") {
		t.Fatalf("Schema Module was selected as a Projection Module: %v", err)
	}
	ambiguous := activation
	for _, registered := range activation.Projectors {
		if registered.Module.Name == "markitect-dotnet" {
			ambiguous.Projectors = append(append([]RegisteredProjector(nil), activation.Projectors...), registered)
			break
		}
	}
	if _, err := bindFixtureProjection(model, ambiguous, identity, nil); err == nil || !strings.Contains(err.Error(), "multiple registered entrypoints") {
		t.Fatalf("ambiguous internal entrypoint was accepted: %v", err)
	}
	targetMismatch := activation
	targetMismatch.Projectors = append([]RegisteredProjector(nil), activation.Projectors...)
	for i := range targetMismatch.Projectors {
		if targetMismatch.Projectors[i].Module.Name == "markitect-dotnet" {
			targetMismatch.Projectors[i].Registration.Target = "markdown"
		}
	}
	if _, err := bindFixtureProjection(model, targetMismatch, identity, nil); err == nil || !strings.Contains(err.Error(), "does not match selected Projection Module target") {
		t.Fatalf("registration target mismatch was accepted: %v", err)
	}
}

func TestCompatibleModuleSwitchLeavesCanonicalModelDigestUnchanged(t *testing.T) {
	model, activation := loadBindingFixture(t, "application-dotnet", nil)
	identity := core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"}
	firstBinding := []ProjectionBinding{{Projection: identity, Module: "markitect-dotnet"}}
	first, err := BindProjection(model, activation, firstBinding, identity, nil)
	if err != nil {
		t.Fatal(err)
	}

	alternateRegistration := strings.Replace(projectorYAML("dotnet-source-alt", "true"), "target: markdown", "target: dotnet", 1)
	alternateRegistration = strings.Replace(alternateRegistration, "docs/generated", "src/", 1)
	alternate := ModulePackage{ManifestBytes: makeManifest("markitect-dotnet-alt", "1.0.0", "", alternateRegistration, "", ""), Files: map[string][]byte{}}
	alternatePin := exactPin(t, alternate)
	alternateManifest, err := DecodeManifest(alternate.ManifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	changedActivation := activation
	changedActivation.Modules = append(append([]Pin(nil), activation.Modules...), alternatePin)
	changedActivation.ModuleTypes = make(map[Pin]string, len(activation.ModuleTypes)+1)
	for pin, moduleType := range activation.ModuleTypes {
		changedActivation.ModuleTypes[pin] = moduleType
	}
	changedActivation.ModuleTypes[alternatePin] = ModuleTypeProjection
	changedActivation.Projectors = append(append([]RegisteredProjector(nil), activation.Projectors...), RegisteredProjector{Module: alternatePin, Registration: alternateManifest.Provides.Projectors[0]})
	secondBinding := []ProjectionBinding{{Projection: identity, Module: "markitect-dotnet-alt"}}
	second, err := BindProjection(model, changedActivation, secondBinding, identity, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.ModelDigest != second.ModelDigest || model.Digest != first.ModelDigest {
		t.Fatalf("runtime Module switch changed canonical model digest: before=%s after=%s canonical=%s", first.ModelDigest, second.ModelDigest, model.Digest)
	}
	if first.RequestDigest == second.RequestDigest || first.Binding.Module == second.Binding.Module || first.ModulePin == second.ModulePin {
		t.Fatalf("runtime Module switch did not change request/tool provenance: first=%#v second=%#v", first, second)
	}
	if first.Projection.Spec["representation"] != second.Projection.Spec["representation"] || first.Binding.Projection != second.Binding.Projection {
		t.Fatal("compatible runtime switch changed canonical Projection intent")
	}
}

func TestBindProjectionRejectsInvalidScopePoliciesAndTargets(t *testing.T) {
	identity := core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"}
	if _, err := projectionDefinitionIdentities([]any{}); err == nil || !strings.Contains(err.Error(), "select at least one") {
		t.Fatalf("empty Projection selection was accepted by Host binding: %v", err)
	}

	model, activation := loadBindingFixture(t, "application-dotnet", func(projection *core.Definition) {
		projection.Spec["source"].(map[string]any)["definitions"] = []any{identityValue("commerce.example.org/v1", "UseCase", "commerce", "missing")}
	})
	if _, err := bindFixtureProjection(model, activation, identity, nil); err == nil || !strings.Contains(err.Error(), "unresolved Definition") {
		t.Fatalf("unresolved source identity was accepted: %v", err)
	}

	model, activation = loadBindingFixture(t, "application-dotnet", func(projection *core.Definition) {
		items := projection.Spec["source"].(map[string]any)["definitions"].([]any)
		projection.Spec["source"].(map[string]any)["definitions"] = append(items, items[0])
	})
	if _, err := bindFixtureProjection(model, activation, identity, nil); err == nil || !strings.Contains(err.Error(), "repeats selected Definition") {
		t.Fatalf("duplicate source identity was accepted: %v", err)
	}

	model, activation = loadBindingFixture(t, "application-dotnet", func(_ *core.Definition) {})
	for i := range model.Definitions {
		if model.Definitions[i].Kind == projectionPolicyKind && model.Definitions[i].Metadata.Name == "use-case-to-dotnet" {
			model.Definitions[i].Spec["targetTechnology"] = "java"
		}
	}
	model, diagnostics := core.Compile(activation.Schemas, model.Definitions, model.Revision)
	if len(diagnostics) != 0 {
		t.Fatalf("wrong-target policy fixture did not compile: %#v", diagnostics)
	}
	if _, err := bindFixtureProjection(model, activation, identity, nil); err == nil || !strings.Contains(err.Error(), "targets technology") {
		t.Fatalf("wrong-target policy was accepted: %v", err)
	}

	model, activation = loadBindingFixture(t, "application-dotnet", nil)
	for _, targetCase := range []struct {
		repository string
		path       string
		files      map[string][]byte
	}{
		{repository: "../other", path: "src/"},
		{repository: ".", path: "../src/"},
		{repository: ".", path: "src/", files: map[string][]byte{"outside/file.cs": []byte("x")}},
		{repository: ".", path: "outside/"},
	} {
		changed := cloneProjectionModel(t, model, identity, func(projection *core.Definition) {
			target := projection.Spec["target"].(map[string]any)
			target["repository"], target["path"] = targetCase.repository, targetCase.path
		})
		if _, err := bindFixtureProjection(changed, activation, identity, targetCase.files); err == nil {
			t.Fatalf("unsafe or out-of-bounds target case was accepted: %+v", targetCase)
		}
	}
}

func TestBindProjectionDoesNotClaimPolicyGuidanceSufficiency(t *testing.T) {
	model, activation := loadBindingFixture(t, "application-dotnet", func(projection *core.Definition) {
		projection.Spec["policies"] = []any{}
	})
	identity := core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"}
	request, err := bindFixtureProjection(model, activation, identity, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Policies) != 0 {
		t.Fatalf("Host inferred policy applicability instead of passing the explicit selection: policies=%d", len(request.Policies))
	}
}

func TestBindProjectionRejectsAmbiguousPolicyMappingAndTamperedModels(t *testing.T) {
	model, activation := loadBindingFixture(t, "application-dotnet", nil)
	identity := core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"}
	definitions := append([]core.Definition(nil), model.Definitions...)
	for i := range definitions {
		if definitions[i].Kind == projectionPolicyKind && definitions[i].Metadata.Name == "handler-to-dotnet" {
			definitions[i].Spec["sourceKind"] = map[string]any{"apiVersion": "commerce.example.org/v1", "kind": "UseCase"}
			definitions[i].Spec["guidance"] = "A conflicting UseCase representation decision."
		}
	}
	conflicting, diagnostics := core.Compile(model.Schemas, definitions, model.Revision)
	if len(diagnostics) != 0 {
		t.Fatalf("compile conflicting policy fixture: %#v", diagnostics)
	}
	if _, err := bindFixtureProjection(conflicting, activation, identity, nil); err == nil || !strings.Contains(err.Error(), "one unambiguous policy mapping") {
		t.Fatalf("multiple selected policies for one Kind and target were hidden: %v", err)
	}

	tampered := model
	tampered.Definitions = append([]core.Definition(nil), model.Definitions...)
	for i := range tampered.Definitions {
		if tampered.Definitions[i].Identity().Key() == identity.Key() {
			tampered.Definitions[i].Purpose = "changed without changing the Core digest"
		}
	}
	if _, err := bindFixtureProjection(tampered, activation, identity, nil); err == nil || !strings.Contains(err.Error(), "model digest") {
		t.Fatalf("changed untrusted model was accepted: %v", err)
	}
}

func loadBindingFixture(t *testing.T, projectionName string, mutate func(*core.Definition)) (core.Model, Activation) {
	t.Helper()
	root, err := filepath.Abs(bindingFixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	moduleDirs := []string{"foundation", "commerce", "dotnet", "markdown"}
	packages := make([]ModulePackage, 0, len(moduleDirs))
	pins := make([]Pin, 0, len(moduleDirs))
	for _, dir := range moduleDirs {
		moduleRoot := filepath.Join(root, "modules", dir)
		manifestBytes, err := os.ReadFile(filepath.Join(moduleRoot, "module.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		manifest, err := DecodeManifest(manifestBytes)
		if err != nil {
			t.Fatal(err)
		}
		pkg := ModulePackage{ManifestBytes: manifestBytes, Files: map[string][]byte{}}
		for _, schemaPath := range manifest.Provides.Schemas {
			content, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(schemaPath)))
			if err != nil {
				t.Fatal(err)
			}
			pkg.Files[schemaPath] = content
		}
		digest, err := DigestPackage(pkg)
		if err != nil {
			t.Fatal(err)
		}
		packages = append(packages, pkg)
		pins = append(pins, Pin{Name: manifest.Name, Version: manifest.Version, Digest: digest})
	}
	activation, err := Resolve(packages, pins)
	if err != nil {
		t.Fatal(err)
	}

	definitionPaths := []string{
		"commerce.projection.yaml", "commerce.markdown-projection.yaml",
		"create-order.projection-policy.yaml", "handler.projection-policy.yaml", "effect-axis.projection-policy.yaml",
		"create-order.use-case.yaml", "create-order.handler.yaml", "create-order.effect-axis.yaml",
	}
	definitions := make([]core.Definition, 0, len(definitionPaths))
	for _, name := range definitionPaths {
		content, err := os.ReadFile(filepath.Join(root, "definitions", name))
		if err != nil {
			t.Fatal(err)
		}
		definition, err := DecodeDefinition(filepath.Join("examples", "canonical-projection", "definitions", name), content)
		if err != nil {
			t.Fatal(err)
		}
		if definition.Kind == projectionKind && definition.Metadata.Name == projectionName && mutate != nil {
			mutate(&definition)
		}
		definitions = append(definitions, definition)
	}
	model, diagnostics := core.Compile(activation.Schemas, definitions, "fixture-revision")
	if len(diagnostics) != 0 {
		t.Fatalf("compile fixture: %#v", diagnostics)
	}
	return model, activation
}

func bindFixtureProjection(model core.Model, activation Activation, identity core.DefinitionIdentity, targets map[string][]byte) (ProjectionRequest, error) {
	module := "markitect-dotnet"
	if identity.Name == "application-markdown" {
		module = "markitect-markdown"
	}
	bindings := []ProjectionBinding{{Projection: identity, Module: module}}
	return BindProjection(model, activation, bindings, identity, targets)
}

func cloneProjectionModel(t *testing.T, model core.Model, identity core.DefinitionIdentity, mutate func(*core.Definition)) core.Model {
	t.Helper()
	definitions := append([]core.Definition(nil), model.Definitions...)
	for i := range definitions {
		if definitions[i].Identity().Key() == identity.Key() {
			mutate(&definitions[i])
		}
	}
	compiled, diagnostics := core.Compile(model.Schemas, definitions, model.Revision)
	if len(diagnostics) != 0 {
		t.Fatalf("compile changed fixture: %#v", diagnostics)
	}
	return compiled
}

func identityValue(apiVersion, kind, namespace, name string) map[string]any {
	return map[string]any{"apiVersion": apiVersion, "kind": kind, "namespace": namespace, "name": name}
}

// Binding validates the installed target contract without keeping a second
// provider allowlist. Host execution remains an explicit static composition.
func TestBindProjectionAcceptsExplicitModuleTargetWithoutProviderAllowlist(t *testing.T) {
	model, activation := loadBindingFixture(t, "application-dotnet", nil)
	identity := core.DefinitionIdentity{APIVersion: foundationAPIVersion, Kind: projectionKind, Namespace: "commerce", Name: "application-dotnet"}
	const target = "another-target"
	encoded, err := json.Marshal(model.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	var schemas []core.Schema
	if err := json.Unmarshal(encoded, &schemas); err != nil {
		t.Fatal(err)
	}
	for i := range schemas {
		if schemas[i].APIVersion == foundationAPIVersion {
			kind := schemas[i].Kinds[projectionKind]
			property := kind.Properties["representation"]
			property.Values = append(property.Values, target)
			kind.Properties["representation"] = property
			schemas[i].Kinds[projectionKind] = kind
		}
	}
	encoded, err = json.Marshal(model.Definitions)
	if err != nil {
		t.Fatal(err)
	}
	var definitions []core.Definition
	if err := json.Unmarshal(encoded, &definitions); err != nil {
		t.Fatal(err)
	}
	for i := range definitions {
		if definitions[i].Identity() == identity {
			definitions[i].Spec["representation"] = target
		}
		if definitions[i].Kind == projectionPolicyKind {
			definitions[i].Spec["targetTechnology"] = target
		}
	}
	compiled, diagnostics := core.Compile(schemas, definitions, model.Revision)
	if len(diagnostics) != 0 {
		t.Fatalf("explicit target Schema: %v", diagnostics)
	}
	activation.Projectors = append([]RegisteredProjector(nil), activation.Projectors...)
	for i := range activation.Projectors {
		if activation.Projectors[i].Module.Name == "markitect-dotnet" {
			activation.Projectors[i].Registration.Target = target
		}
	}
	bound, err := bindFixtureProjection(compiled, activation, identity, nil)
	if err != nil || bound.Projector.Target != target {
		t.Fatalf("explicit installed target: %v %+v", err, bound)
	}
	for i := range activation.Projectors {
		if activation.Projectors[i].Module.Name == "markitect-dotnet" {
			activation.Projectors[i].Registration.Target = "dotnet"
		}
	}
	if _, err := bindFixtureProjection(compiled, activation, identity, nil); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("target mismatch accepted: %v", err)
	}
}
