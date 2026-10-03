package examples

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// These are deliberately negative language-pressure probes. A passing parse
// demonstrates that the current finite model does not enforce the named
// invariant; it is not an endorsement of the mutated architecture.
func loadSoftwareArchitecturePressureBase(t *testing.T) *app.Project {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate pressure test source")
	}
	project, err := app.Load(filepath.Join(filepath.Dir(sourceFile), "software-architecture"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("software-architecture baseline has diagnostics: %#v", project.Diagnostics)
	}
	return project
}

func parseSoftwareArchitecturePressureResourceMutation(t *testing.T, base *app.Project, path string, mutate func(*core.Resource)) *app.Project {
	t.Helper()
	private := privateSoftwareArchitectureSnapshot(base)
	data, exists := private.Files[path]
	if !exists {
		t.Fatalf("pressure fixture source %q is absent", path)
	}
	resource, err := format.ParseWithRegistry(path, data, base.Graph.Registry)
	if err != nil {
		t.Fatalf("parse pressure resource %s: %v", path, err)
	}
	mutate(resource)
	encoded, err := format.Encode(resource)
	if err != nil {
		t.Fatalf("encode mutated pressure resource %s: %v", path, err)
	}
	private.Files[path] = encoded
	project, err := app.Parse(private)
	if err != nil {
		t.Fatalf("parse mutated private snapshot: %v", err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("current language unexpectedly rejected pressure mutation in %s: %#v", path, project.Diagnostics)
	}
	return project
}

func privateSoftwareArchitectureSnapshot(base *app.Project) *snapshot.Snapshot {
	files := make(map[string][]byte, len(base.Snapshot.Files))
	for name, data := range base.Snapshot.Files {
		files[name] = append([]byte(nil), data...)
	}
	modes := make(map[string]string, len(base.Snapshot.Modes))
	for name, mode := range base.Snapshot.Modes {
		modes[name] = mode
	}
	return &snapshot.Snapshot{ID: base.Snapshot.ID, Provisional: base.Snapshot.Provisional, Files: files, Modes: modes}
}

func assertPressureRelationship(t *testing.T, project *app.Project, from, relation, to string) {
	t.Helper()
	for _, edge := range project.Graph.Relationships {
		if edge.From == from && edge.Relation == relation && edge.To == to {
			return
		}
	}
	t.Fatalf("mutated graph lost expected %s edge %s -> %s; relationships: %#v", relation, from, to, project.Graph.Relationships)
}

func setPressureReferenceName(t *testing.T, resource *core.Resource, field, name string) {
	t.Helper()
	reference, ok := resource.Data[field].(map[string]any)
	if !ok {
		t.Fatalf("resource %s field %s is not a reference map: %#v", resource.GraphKey(), field, resource.Data[field])
	}
	reference["name"] = name
}

func setPressureArrayReferenceName(t *testing.T, resource *core.Resource, field string, index int, name string) {
	t.Helper()
	references, ok := resource.Data[field].([]any)
	if !ok || index < 0 || index >= len(references) {
		t.Fatalf("resource %s field %s does not contain reference index %d: %#v", resource.GraphKey(), field, index, resource.Data[field])
	}
	reference, ok := references[index].(map[string]any)
	if !ok {
		t.Fatalf("resource %s field %s item %d is not a reference map: %#v", resource.GraphKey(), field, index, references[index])
	}
	reference["name"] = name
}

func TestSoftwareArchitecturePressureUseCaseFeatureOwnerMismatchPasses(t *testing.T) {
	root := t.TempDir()
	copySoftwareArchitecture(t, root)
	installSoftwareArchitectureV1(t, root)
	base := loadSoftwareArchitecture(t, root, "")
	if len(base.Diagnostics) != 0 {
		t.Fatalf("historical v1.0.0 baseline should not enforce Feature owner equality: %#v", base.Diagnostics)
	}
	project := parseSoftwareArchitecturePressureResourceMutation(t, base, "resources/usecase-create-order.yaml", func(resource *core.Resource) {
		setPressureReferenceName(t, resource, "feature", "invoicing")
	})
	useCase := "engineering/architecture.markitect.org/v1alpha1/UseCase/create-order"
	assertPressureRelationship(t, project, useCase, "belongsToModule", "engineering/architecture.markitect.org/v1alpha1/Module/orders")
	assertPressureRelationship(t, project, useCase, "realizesFeature", "engineering/architecture.markitect.org/v1alpha1/Feature/invoicing")
	assertPressureRelationship(t, project, "engineering/architecture.markitect.org/v1alpha1/Feature/invoicing", "belongsToModule", "engineering/architecture.markitect.org/v1alpha1/Module/billing")
}

func TestSoftwareArchitecturePressureUseCaseForeignAggregateOwnerPasses(t *testing.T) {
	base := loadSoftwareArchitecturePressureBase(t)
	project := parseSoftwareArchitecturePressureResourceMutation(t, base, "resources/usecase-get-order.yaml", func(resource *core.Resource) {
		setPressureArrayReferenceName(t, resource, "aggregates", 0, "invoice")
	})
	assertPressureRelationship(t, project, "engineering/architecture.markitect.org/v1alpha1/UseCase/get-order", "touchesAggregate", "engineering/architecture.markitect.org/v1alpha1/Aggregate/invoice")
	assertPressureRelationship(t, project, "engineering/architecture.markitect.org/v1alpha1/Aggregate/invoice", "belongsToModule", "engineering/architecture.markitect.org/v1alpha1/Module/billing")
}

func TestSoftwareArchitecturePressureTwoUseCasesShareHandlerPasses(t *testing.T) {
	base := loadSoftwareArchitecturePressureBase(t)
	project := parseSoftwareArchitecturePressureResourceMutation(t, base, "resources/usecase-get-order.yaml", func(resource *core.Resource) {
		setPressureReferenceName(t, resource, "handler", "create-order-handler")
	})
	target := "engineering/architecture.markitect.org/v1alpha1/Handler/create-order-handler"
	incoming := 0
	for _, edge := range project.Graph.Relationships {
		if edge.Relation == "handledBy" && edge.To == target {
			incoming++
		}
	}
	if incoming != 2 {
		t.Fatalf("expected two distinct UseCases to share one Handler; found %d incoming handledBy edges", incoming)
	}
}

func TestSoftwareArchitecturePressureInterfaceProviderIdentityMismatchPasses(t *testing.T) {
	base := loadSoftwareArchitecturePressureBase(t)
	project := parseSoftwareArchitecturePressureResourceMutation(t, base, "resources/interface-order-accepted.yaml", func(resource *core.Resource) {
		setPressureReferenceName(t, resource, "provider", "inventory")
	})
	assertPressureRelationship(t, project, "engineering/architecture.markitect.org/v1alpha1/Module/billing", "usesInterfaces", "engineering/architecture.markitect.org/v1alpha1/Interface/order-accepted")
	assertPressureRelationship(t, project, "engineering/architecture.markitect.org/v1alpha1/Interface/order-accepted", "providedBy", "engineering/architecture.markitect.org/v1alpha1/Module/inventory")
}

func TestSoftwareArchitecturePressureMixedRelationCyclePasses(t *testing.T) {
	base := loadSoftwareArchitecturePressureBase(t)
	project := parseSoftwareArchitecturePressureResourceMutation(t, base, "resources/interface-order-accepted.yaml", func(resource *core.Resource) {
		setPressureReferenceName(t, resource, "provider", "billing")
	})
	module := "engineering/architecture.markitect.org/v1alpha1/Module/billing"
	contract := "engineering/architecture.markitect.org/v1alpha1/Interface/order-accepted"
	domain, ok := project.Graph.Registry.Domain("architecture.markitect.org/v1alpha1")
	if !ok || !domain.Relations["usesInterfaces"].Acyclic {
		t.Fatalf("fixture must declare usesInterfaces acyclic: %#v", domain.Relations)
	}
	// The fixture currently leaves providedBy non-acyclic. In a private
	// in-memory Domain copy, turn it on too, then rebuild the graph to isolate
	// the current per-relation cycle semantics without changing fixture files.
	providedBy := domain.Relations["providedBy"]
	providedBy.Acyclic = true
	domain.Relations["providedBy"] = providedBy
	if !domain.Relations["providedBy"].Acyclic {
		t.Fatal("private pressure Domain failed to mark providedBy acyclic")
	}
	registry := core.NewRegistry()
	if err := registry.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	graph := core.BuildWithRegistry(project.Resources, registry)
	assertPressureRelationship(t, &app.Project{Graph: graph}, module, "usesInterfaces", contract)
	assertPressureRelationship(t, &app.Project{Graph: graph}, contract, "providedBy", module)
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Code == "relation.cycle" {
			t.Fatalf("cycle across usesInterfaces (Module -> Interface) and providedBy (Interface -> Module) should remain undetected when each relation is acyclic independently; found %#v", diagnostic)
		}
	}
}

func TestSoftwareArchitecturePressureUniqueReferenceConstraintRejected(t *testing.T) {
	project := loadSoftwareArchitecturePressureBase(t)
	domain, ok := project.Graph.Registry.Domain("architecture.markitect.org/v1alpha1")
	if !ok {
		t.Fatal("fixture did not activate the architecture Domain")
	}
	domain.Constraints = append(domain.Constraints, core.ConstraintDefinition{
		Name:   "module-product-unique",
		Select: core.ResourceSelector{Kind: "Module"},
		Assert: core.ConstraintAssertion{Op: "unique", Field: "product"},
	})
	if err := core.NewRegistry().AddDomain(domain); err == nil || !strings.Contains(err.Error(), "unique requires a scalar field") {
		t.Fatalf("reference uniqueness should be rejected by the finite constraint language, got %v", err)
	}
}

func TestSoftwareArchitecturePressureArbitrarySpecFieldSelectorRejected(t *testing.T) {
	unsupported := []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: pressure}
spec:
  apiVersion: pressure.example.org/v1
  kinds:
    UseCase:
      properties:
        intent: {type: string}
  constraints:
    - name: command-selector
      select:
        kind: UseCase
        fields: {intent: Command}
      assert: {op: equal, field: intent, value: Command}
`)
	if _, err := format.ParseDomain("domains/pressure.yaml", unsupported); err == nil || !strings.Contains(err.Error(), "field fields not found") {
		t.Fatalf("arbitrary spec-field selector should be rejected, got %v", err)
	}
}

func TestSoftwareArchitecturePressureEmptyOptInCohortHasNoSubjectResultsV11V2AndV21(t *testing.T) {
	for _, version := range []string{"v1.1", "v2", "v2.1"} {
		t.Run(version, func(t *testing.T) {
			root := t.TempDir()
			copySoftwareArchitecture(t, root)
			switch version {
			case "v1.1":
				installSoftwareArchitectureV1_1(t, root)
			case "v2":
				installSoftwareArchitectureV2(t, root)
			case "v2.1":
				installSoftwareArchitectureV2_1(t, root)
			}
			base, err := app.Load(root, "")
			if err != nil {
				t.Fatalf("%s baseline failed to load: %v", version, err)
			}
			if version == "v1.1" && len(base.Diagnostics) != 0 {
				t.Fatalf("v1.1 baseline should have no policy findings: %#v", base.Diagnostics)
			}
			if version == "v2" || version == "v2.1" {
				failures := 0
				for _, diagnostic := range base.Diagnostics {
					if diagnostic.Code == "constraint.selected-commands-have-validators" {
						failures++
					}
				}
				if failures != 2 || len(base.Diagnostics) != failures {
					t.Fatalf("%s baseline should expose exactly the two opted-in validator failures before labels are removed: %#v", version, base.Diagnostics)
				}
			}
			private := privateSoftwareArchitectureSnapshot(base)
			for _, path := range []string{"resources/usecase-create-order.yaml", "resources/usecase-issue-invoice.yaml"} {
				data, exists := private.Files[path]
				if !exists {
					t.Fatalf("%s fixture is missing opt-in cohort member %s", version, path)
				}
				resource, err := format.ParseWithRegistry(path, data, base.Graph.Registry)
				if err != nil {
					t.Fatalf("parse %s cohort member %s: %v", version, path, err)
				}
				if resource.Metadata.Labels["validation"] != "required" || (version != "v2" && resource.Metadata.Labels["feature-ownership"] != "required") {
					t.Fatalf("%s cohort member %s does not carry the opt-in labels: %#v", version, path, resource.Metadata.Labels)
				}
				labels := make(map[string]string, len(resource.Metadata.Labels))
				for key, value := range resource.Metadata.Labels {
					if key != "validation" && key != "feature-ownership" {
						labels[key] = value
					}
				}
				resource.Metadata.Labels = labels
				encoded, err := format.Encode(resource)
				if err != nil {
					t.Fatalf("encode %s cohort member %s: %v", version, path, err)
				}
				private.Files[path] = encoded
			}
			project, err := app.Parse(private)
			if err != nil || len(project.Diagnostics) != 0 {
				t.Fatalf("empty %s opt-in cohort failed to parse: %v; %#v", version, err, project.Diagnostics)
			}
			for _, result := range project.Graph.PolicyResults {
				if result.Constraint == "selected-command-labels-identify-commands" || result.Constraint == featureOwnershipRule || (version == "v2" || version == "v2.1") && result.Constraint == "selected-commands-have-validators" {
					t.Fatalf("empty %s opt-in cohort unexpectedly emitted a per-subject result: %+v", version, result)
				}
			}
		})
	}
}
