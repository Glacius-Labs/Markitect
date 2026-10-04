package examples

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
)

const deliveryTopologyAPI = "delivery.example.org/v1alpha1"

const deliveryEqualityConstraint = "deployment-service-and-environment-share-product"

func loadDeliveryTargetEquality(t *testing.T) *host.Project {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate Delivery target equality fixture")
	}
	project, err := host.Load(filepath.Join(filepath.Dir(source), "delivery-target-equality"), "")
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func deliveryTargetEqualityRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate Delivery target equality fixture")
	}
	return filepath.Join(filepath.Dir(source), "delivery-target-equality")
}

func relationshipTarget(t *testing.T, graph *authoring.Graph, from, relation string) string {
	t.Helper()
	var found []string
	for _, edge := range graph.Relationships {
		if edge.From == from && edge.Relation == relation {
			found = append(found, edge.To)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s via %s resolved to %d targets, want exactly one", from, relation, len(found))
	}
	return found[0]
}

func deploymentProductPaths(t *testing.T, graph *authoring.Graph, deployment string) (string, string) {
	t.Helper()
	service := relationshipTarget(t, graph, deployment, "deploysService")
	environment := relationshipTarget(t, graph, deployment, "deploysTo")
	return relationshipTarget(t, graph, service, "belongsToProduct"),
		relationshipTarget(t, graph, environment, "belongsToProduct")
}

func TestDeliveryTargetEqualityCurrentLanguageAcceptsMatchingAndMismatchingDeployments(t *testing.T) {
	project := loadDeliveryTargetEquality(t)
	if len(project.Diagnostics) != 0 {
		t.Fatalf("both fixtures should be structurally valid in the current language: %#v", project.Diagnostics)
	}
	domain, ok := project.Graph.Registry.Domain(deliveryTopologyAPI)
	if !ok {
		t.Fatalf("Delivery Domain %q was not activated", deliveryTopologyAPI)
	}
	if len(domain.Constraints) != 1 || domain.Constraints[0].Name != deliveryEqualityConstraint {
		t.Fatalf("fixture should declare only the bounded target-equality assertion: %#v", domain.Constraints)
	}
	for _, name := range []string{"belongsToProduct", "deploysService", "deploysTo"} {
		if _, ok := domain.Relations[name]; !ok {
			t.Fatalf("required relation %q is missing", name)
		}
	}

	matching := "engineering/delivery.example.org/v1alpha1/Deployment/orders-production"
	left, right := deploymentProductPaths(t, project.Graph, matching)
	wantCommerce := "engineering/delivery.example.org/v1alpha1/Product/commerce"
	if left != wantCommerce || right != wantCommerce {
		t.Fatalf("positive Deployment should resolve both paths to %s, got %s and %s", wantCommerce, left, right)
	}

	for _, subject := range []string{matching, "engineering/delivery.example.org/v1alpha1/Deployment/telemetry-production"} {
		result, ok := deliveryPolicyResult(project.Graph.Core.PolicyResults, subject)
		if !ok || result.Status != core.PolicyPassed || result.Comparison == nil {
			t.Fatalf("matching Deployment should have a traceable passing equality result for %s: %+v", subject, result)
		}
	}
	if len(project.Graph.Core.PolicyResults) != 2 {
		t.Fatalf("each Deployment should have one equality result, got %#v", project.Graph.Core.PolicyResults)
	}
}

func TestDeliveryTargetEqualityPreOperatorKernelAcceptsTypedMismatch(t *testing.T) {
	base := loadDeliveryTargetEquality(t)
	private := privateDeliveryTargetSnapshot(base)
	definitionPath := "domains/delivery-topology.yaml"
	domain, err := authoring.ParseDomain(definitionPath, private.Files[definitionPath])
	if err != nil {
		t.Fatal(err)
	}
	filtered := domain.Constraints[:0]
	for _, constraint := range domain.Constraints {
		if constraint.Name != deliveryEqualityConstraint {
			filtered = append(filtered, constraint)
		}
	}
	domain.Constraints = filtered
	encodedDomain, err := authoring.EncodeDomain(domain)
	if err != nil {
		t.Fatal(err)
	}
	private.Files[definitionPath] = encodedDomain
	mutateDeliveryReferenceInSnapshot(t, private, base.Graph.Registry, "resources/deployment-orders-production.yaml", "environment", "support-production")
	project, err := host.Parse(private)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("current finite language previously accepts the fully typed mismatch: %#v", project.Diagnostics)
	}
	left, right := deploymentProductPaths(t, project.Graph, "engineering/delivery.example.org/v1alpha1/Deployment/orders-production")
	wantCommerce := "engineering/delivery.example.org/v1alpha1/Product/commerce"
	wantSupport := "engineering/delivery.example.org/v1alpha1/Product/support-tools"
	if left != wantCommerce || right != wantSupport {
		t.Fatalf("baseline mismatch did not retain distinct resolved paths: %s / %s", left, right)
	}
	if len(project.Graph.Core.PolicyResults) != 0 {
		t.Fatalf("without the new assertion, the current language should emit no equality result: %#v", project.Graph.Core.PolicyResults)
	}
}

func TestDeliveryTargetEqualityReportsDifferentPathResults(t *testing.T) {
	for _, test := range []struct {
		name, path, field, targetName, wantLeft, wantRight string
	}{
		{
			name: "service product path changes",
			path: "resources/service-orders.yaml", field: "product", targetName: "support-tools",
			wantLeft:  "engineering/delivery.example.org/v1alpha1/Product/support-tools",
			wantRight: "engineering/delivery.example.org/v1alpha1/Product/commerce",
		},
		{
			name: "environment product path changes",
			path: "resources/environment-commerce-production.yaml", field: "product", targetName: "support-tools",
			wantLeft:  "engineering/delivery.example.org/v1alpha1/Product/commerce",
			wantRight: "engineering/delivery.example.org/v1alpha1/Product/support-tools",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := loadDeliveryTargetEquality(t)
			private := privateDeliveryTargetSnapshot(base)
			mutateDeliveryReferenceInSnapshot(t, private, base.Graph.Registry, test.path, test.field, test.targetName)
			changed, err := host.Parse(private)
			if err != nil {
				t.Fatal(err)
			}
			if len(changed.Diagnostics) != 1 || changed.Diagnostics[0].Code != "constraint."+deliveryEqualityConstraint {
				t.Fatalf("cross-Product wiring should produce only its subject policy diagnostic: %#v", changed.Diagnostics)
			}
			deployment := "engineering/delivery.example.org/v1alpha1/Deployment/orders-production"
			result, ok := deliveryPolicyResult(changed.Graph.Core.PolicyResults, deployment)
			if !ok || result.Status != core.PolicyFailed || result.Comparison == nil {
				t.Fatalf("mismatch should retain its failed per-Deployment result and trace: %+v", result)
			}
			if result.Comparison.Left.Target != test.wantLeft || result.Comparison.Right.Target != test.wantRight {
				t.Fatalf("comparison trace lost independent endpoints: left=%s right=%s", result.Comparison.Left.Target, result.Comparison.Right.Target)
			}
			if len(result.Comparison.Left.Steps) != 2 || len(result.Comparison.Right.Steps) != 2 || result.Comparison.Left.Relations[0] != "deploysService" || result.Comparison.Right.Relations[0] != "deploysTo" {
				t.Fatalf("comparison trace should show both fixed two-edge paths: %+v", result.Comparison)
			}
			telemetry := "engineering/delivery.example.org/v1alpha1/Deployment/telemetry-production"
			other, ok := deliveryPolicyResult(changed.Graph.Core.PolicyResults, telemetry)
			if !ok || other.Status != core.PolicyPassed {
				t.Fatalf("unrelated Deployment should remain a passing subject: %+v", other)
			}
		})
	}
}

func TestDeliveryTargetEqualityOwnershipIsNonContextAndInvalidatesPathDependents(t *testing.T) {
	project := loadDeliveryTargetEquality(t)
	if len(project.Diagnostics) != 0 {
		t.Fatalf("baseline should be structurally valid: %#v", project.Diagnostics)
	}
	definition, ok := project.Graph.Registry.Domain(deliveryTopologyAPI)
	if !ok {
		t.Fatal("Delivery Domain is unavailable")
	}
	ownership := definition.Relations["belongsToProduct"]
	if ownership.Context || ownership.Invalidate {
		t.Fatalf("Product ownership should be neither context nor ordinary invalidation: %+v", ownership)
	}
	ordersService := "engineering/delivery.example.org/v1alpha1/Service/orders-api"
	commerceProduct := "engineering/delivery.example.org/v1alpha1/Product/commerce"
	foundRelation := false
	for _, edge := range project.Graph.Relationships {
		if edge.From == ordersService && edge.To == commerceProduct && edge.Relation == "belongsToProduct" {
			foundRelation = true
			if edge.Context || edge.Invalidate {
				t.Fatalf("resolved Product relation did not retain its declared effects: %+v", edge)
			}
		}
	}
	if !foundRelation {
		t.Fatal("orders Service Product ownership relation is absent")
	}

	context, err := host.CompileContext(project, "engineering/Skill/deployment-review", "test")
	if err != nil {
		t.Fatal(err)
	}
	contextResources := map[string]bool{}
	for _, input := range context.Inputs {
		if input.Resource != nil {
			contextResources[input.Resource.GraphKey()] = true
		}
	}
	for _, expected := range []string{
		"engineering/Skill/deployment-review",
		"engineering/delivery.example.org/v1alpha1/Deployment/orders-production",
		ordersService,
		"engineering/delivery.example.org/v1alpha1/Environment/commerce-production",
	} {
		if !contextResources[expected] {
			t.Errorf("deployment review context omitted expected input %s", expected)
		}
	}
	for _, excluded := range []string{
		commerceProduct,
		"engineering/delivery.example.org/v1alpha1/Product/telemetry",
		"engineering/delivery.example.org/v1alpha1/Service/telemetry-api",
		"engineering/delivery.example.org/v1alpha1/Environment/telemetry-production",
		"engineering/delivery.example.org/v1alpha1/Deployment/telemetry-production",
	} {
		if contextResources[excluded] {
			t.Errorf("context unexpectedly included non-context or unrelated resource %s", excluded)
		}
	}

	for _, test := range []struct {
		name, file, resourceKey, field, newTarget string
		wantAffected                              []string
	}{
		{
			name: "service ownership path",
			file: "resources/service-orders.yaml", resourceKey: ordersService, field: "product",
			newTarget: "support-tools",
			wantAffected: []string{
				ordersService,
				"engineering/delivery.example.org/v1alpha1/Deployment/orders-production",
			},
		},
		{
			name:        "environment ownership path",
			file:        "resources/environment-commerce-production.yaml",
			resourceKey: "engineering/delivery.example.org/v1alpha1/Environment/commerce-production",
			field:       "product", newTarget: "support-tools",
			wantAffected: []string{
				"engineering/delivery.example.org/v1alpha1/Environment/commerce-production",
				"engineering/delivery.example.org/v1alpha1/Deployment/orders-production",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			copyDeliveryTargetFixture(t, root)
			base := loadDeliveryTargetEqualityAt(t, root)
			mutateDeliveryProductOwner(t, root, base, test.resourceKey, test.file, test.field, test.newTarget)
			after := loadDeliveryTargetEqualityAt(t, root)
			if len(after.Diagnostics) != 1 || after.Diagnostics[0].Code != "constraint."+deliveryEqualityConstraint {
				t.Fatalf("changing a valid Product should produce only the subject's equality diagnostic: %#v", after.Diagnostics)
			}
			if result, ok := deliveryPolicyResult(after.Graph.Core.PolicyResults, "engineering/delivery.example.org/v1alpha1/Deployment/orders-production"); !ok || result.Status != core.PolicyFailed {
				t.Fatalf("changed ownership path should fail the selected Deployment: %+v", result)
			}
			impact := host.Changes(base, after)
			for _, want := range test.wantAffected {
				if !deliveryContains(impact.Affected, want) {
					t.Errorf("ownership-path edit omitted dependent %s from impact: %#v", want, impact.Affected)
				}
			}
			for _, unrelated := range []string{
				"engineering/delivery.example.org/v1alpha1/Product/telemetry",
				"engineering/delivery.example.org/v1alpha1/Service/telemetry-api",
				"engineering/delivery.example.org/v1alpha1/Environment/telemetry-production",
				"engineering/delivery.example.org/v1alpha1/Deployment/telemetry-production",
			} {
				if deliveryContains(impact.Affected, unrelated) {
					t.Errorf("ownership-path edit unexpectedly affected unrelated cohort member %s: %#v", unrelated, impact.Affected)
				}
			}
			foundPolicyCause := false
			for _, cause := range impact.Causes {
				if cause.Kind == "policy-dependency" && cause.Resource == "engineering/delivery.example.org/v1alpha1/Deployment/orders-production" && cause.Constraint == deliveryEqualityConstraint {
					foundPolicyCause = true
				}
			}
			if !foundPolicyCause {
				t.Errorf("impact omitted explicit same-target policy dependency trace: %#v", impact.Causes)
			}
		})
	}
}

func TestDeliveryTargetEqualitySourcesUseCanonicalFormat(t *testing.T) {
	root := deliveryTargetEqualityRoot(t)
	projectBytes, err := os.ReadFile(filepath.Join(root, "markitect.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	project, err := authoring.Parse(filepath.Join(root, "markitect.yaml"), projectBytes)
	if err != nil {
		t.Fatal(err)
	}
	canonicalProject, err := authoring.Encode(*project)
	if err != nil {
		t.Fatal(err)
	}
	if string(projectBytes) != string(canonicalProject) {
		t.Fatal("Project source is not in canonical Markitect format")
	}
	domainBytes, err := os.ReadFile(filepath.Join(root, "domains", "delivery-topology.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	domain, err := authoring.ParseDomain("domains/delivery-topology.yaml", domainBytes)
	if err != nil {
		t.Fatal(err)
	}
	canonicalDomain, err := authoring.EncodeDomain(domain)
	if err != nil {
		t.Fatal(err)
	}
	if string(domainBytes) != string(canonicalDomain) {
		t.Fatal("Domain source is not in canonical Markitect format")
	}
	projectModel := loadDeliveryTargetEquality(t)
	for path, data := range projectModel.Snapshot.Files {
		if !strings.HasPrefix(path, "resources/") || !strings.HasSuffix(path, ".yaml") {
			continue
		}
		resource, err := authoring.ParseWithRegistry(path, data, projectModel.Graph.Registry)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		canonical, err := authoring.Encode(*resource)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(canonical) {
			t.Errorf("resource source %s is not in canonical Markitect format", path)
		}
	}
}

func loadDeliveryTargetEqualityAt(t *testing.T, root string) *host.Project {
	t.Helper()
	project, err := host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func privateDeliveryTargetSnapshot(project *host.Project) *snapshot.Snapshot {
	files := make(map[string][]byte, len(project.Snapshot.Files))
	for name, data := range project.Snapshot.Files {
		files[name] = append([]byte(nil), data...)
	}
	modes := make(map[string]string, len(project.Snapshot.Modes))
	for name, mode := range project.Snapshot.Modes {
		modes[name] = mode
	}
	return &snapshot.Snapshot{ID: project.Snapshot.ID, Provisional: project.Snapshot.Provisional, Files: files, Modes: modes}
}

func mutateDeliveryReferenceInSnapshot(t *testing.T, private *snapshot.Snapshot, registry *core.Registry, path, field, targetName string) {
	t.Helper()
	resource, err := authoring.ParseWithRegistry(path, private.Files[path], registry)
	if err != nil {
		t.Fatal(err)
	}
	ref, ok := resource.Data[field].(map[string]any)
	if !ok {
		t.Fatalf("%s.%s is not a reference map: %#v", path, field, resource.Data[field])
	}
	updated := make(map[string]any, len(ref))
	for name, value := range ref {
		updated[name] = value
	}
	updated["name"] = targetName
	resource.Data[field] = updated
	encoded, err := authoring.Encode(*resource)
	if err != nil {
		t.Fatal(err)
	}
	private.Files[path] = encoded
}

func copyDeliveryTargetFixture(t *testing.T, destination string) {
	t.Helper()
	source := deliveryTargetEqualityRoot(t)
	if err := filepath.WalkDir(source, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, name)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		content, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0644)
	}); err != nil {
		t.Fatal(err)
	}
}

func mutateDeliveryProductOwner(t *testing.T, root string, project *host.Project, resourceKey, relativePath, field, targetName string) {
	t.Helper()
	resource := project.Graph.Resources[resourceKey]
	if resource == nil {
		t.Fatalf("resource %s is missing", resourceKey)
	}
	updated := *resource
	updated.Data = make(map[string]any, len(resource.Data))
	for name, value := range resource.Data {
		updated.Data[name] = value
	}
	ref, ok := resource.Data[field].(map[string]any)
	if !ok {
		t.Fatalf("%s.%s is not a reference map: %#v", resourceKey, field, resource.Data[field])
	}
	refCopy := make(map[string]any, len(ref))
	for name, value := range ref {
		refCopy[name] = value
	}
	refCopy["name"] = targetName
	updated.Data[field] = refCopy
	encoded, err := authoring.Encode(updated)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relativePath)), encoded, 0644); err != nil {
		t.Fatal(err)
	}
}

func deliveryContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func deliveryPolicyResult(results []core.PolicyResult, subject string) (core.PolicyResult, bool) {
	for _, result := range results {
		if result.APIVersion == deliveryTopologyAPI && result.Constraint == deliveryEqualityConstraint && result.Subject == subject {
			return result, true
		}
	}
	return core.PolicyResult{}, false
}
