package examples

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/render"
)

func canonicalEngineeringRoot(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate example test source")
	}
	return filepath.Join(filepath.Dir(sourceFile), "canonical-engineering")
}

func copyCanonicalExample(t *testing.T, destination string) {
	t.Helper()
	source := canonicalEngineeringRoot(t)
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
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalEngineeringDomainsCompilePolicyContextAndViews(t *testing.T) {
	root := canonicalEngineeringRoot(t)
	project, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("example has diagnostics: %#v", project.Diagnostics)
	}

	compiled, err := app.CompileContext(project, "engineering/Skill/architecture-review", "example-test")
	if err != nil {
		t.Fatal(err)
	}
	resources, domains := map[string]bool{}, map[string]string{}
	for _, input := range compiled.Inputs {
		if input.Resource != nil {
			resources[input.Resource.GraphKey()] = true
		}
		if input.Role == "domain" {
			domains[input.Path] = input.Text
		}
	}
	for _, key := range []string{"engineering/Skill/architecture-review", "engineering/software.markitect.org/v1alpha1/Module/orders", "engineering/software.markitect.org/v1alpha1/Core/platform-core", "engineering/delivery.markitect.org/v1alpha1/Release/0-10-0-candidate", "engineering/delivery.markitect.org/v1alpha1/ReleasePolicy/candidate", "engineering/delivery.markitect.org/v1alpha1/ReleaseEvidence/unit-tests-8d7f3c1", "engineering/delivery.markitect.org/v1alpha1/ReleaseEvidence/policy-check-8d7f3c1"} {
		if !resources[key] {
			t.Errorf("compiled Skill context omitted %s", key)
		}
	}
	for _, key := range []string{"engineering/software.markitect.org/v1alpha1/UseCase/place-order", "engineering/software.markitect.org/v1alpha1/Interface/order-store"} {
		if resources[key] {
			t.Errorf("non-context owns relation pulled %s into context", key)
		}
	}
	software := domains["domains/software.yaml"]
	if !strings.Contains(software, "application-modules-use-core") || !strings.Contains(software, "allowed-targets") || !strings.Contains(software, "Core") {
		t.Fatalf("compiled context omitted the structured software policy: %q", software)
	}

	outputs, err := render.Generate(project.Graph, project.Snapshot.Files)
	if err != nil {
		t.Fatal(err)
	}
	var moduleView string
	for name, content := range outputs {
		if strings.Contains(string(content), "application-modules-use-core") && strings.HasSuffix(name, ".md") {
			moduleView = string(content)
		}
	}
	for _, value := range []string{"application-modules-use-core", "relationships may target only kinds", "Core", "dependsOn"} {
		if !strings.Contains(moduleView, value) {
			t.Errorf("generated Module view omitted structured policy value %q: %s", value, moduleView)
		}
	}
	if findings := app.CheckOutputs(project); len(findings) != 0 {
		t.Fatalf("checked-in generated views must match the canonical model: %#v", findings)
	}
}

func TestCanonicalEngineeringPolicyConflictAndStalePlan(t *testing.T) {
	root := canonicalEngineeringRoot(t)
	project, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	toolDigest := app.Hash([]byte("canonical-example-tool"))
	plan, err := app.PlanProjection(project, "0.10.0", toolDigest)
	if err != nil {
		t.Fatal(err)
	}

	modulePath := filepath.Join(root, "resources", "module.yaml")
	original, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.WriteFile(modulePath, original, 0644)
	conflict := strings.Replace(string(original), "kind: Core", "kind: Module", 1)
	if conflict == string(original) {
		t.Fatal("test could not locate the dependency target")
	}
	if err := os.WriteFile(modulePath, []byte(conflict), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(changed.Diagnostics) == 0 {
		t.Fatal("policy-conflicting relation unexpectedly passed")
	}
	foundPolicy := false
	for _, diagnostic := range changed.Diagnostics {
		if strings.Contains(diagnostic.Code, "constraint") && strings.Contains(diagnostic.Message, "application-modules-use-core") {
			foundPolicy = true
		}
	}
	if !foundPolicy {
		t.Fatalf("expected a policy diagnostic tied to the structured constraint, got %#v", changed.Diagnostics)
	}
	if err := app.ValidateProjectionPlan(changed, plan, "0.10.0", toolDigest); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("source edit should invalidate the captured local projection plan, got %v", err)
	}
}

func TestCanonicalRenderPlanApplyVerifyRoundTrip(t *testing.T) {
	root := filepath.Join(t.TempDir(), "canonical-engineering")
	copyCanonicalExample(t, root)
	project, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("fixture should load before planning: %#v", project.Diagnostics)
	}
	toolDigest := app.Hash([]byte("canonical-example-tool"))
	plan, err := app.PlanProjection(project, "0.10.0", toolDigest)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) == 0 {
		outputs, renderErr := render.Generate(project.Graph, project.Snapshot.Files)
		if renderErr != nil {
			t.Fatal(renderErr)
		}
		for name := range outputs {
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(name))); err != nil {
				t.Fatal(err)
			}
			break
		}
		project, err = app.Load(root, "")
		if err != nil {
			t.Fatal(err)
		}
		plan, err = app.PlanProjection(project, "0.10.0", toolDigest)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(plan.Operations) == 0 {
		t.Fatal("fixture should produce a local projection operation after introducing output drift")
	}
	planPath := filepath.Join(root, ".artifacts", "markitect", "reconcile", "render-plan.yaml")
	if err := os.MkdirAll(filepath.Dir(planPath), 0755); err != nil {
		t.Fatal(err)
	}
	planBytes, err := app.YAML(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, planBytes, 0644); err != nil {
		t.Fatal(err)
	}
	savedPlan, err := app.ReadPlan(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.ValidateProjectionPlan(project, savedPlan, "0.10.0", toolDigest); err != nil {
		t.Fatalf("saved plan should match its captured project: %v", err)
	}

	tampered := savedPlan
	tampered.Operations = append([]app.ReconcileOperation(nil), savedPlan.Operations...)
	tampered.Operations[0].Content += "tampered"
	tamperedBytes, err := app.YAML(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ParseReconcilePlan(tamperedBytes); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("tampered saved plan should fail its content hash: %v", err)
	}

	if _, err := app.ApplyProjection(root, project, savedPlan, "0.10.0", toolDigest); err != nil {
		t.Fatalf("apply saved local plan: %v", err)
	}
	afterApply, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.VerifyProjectionPlan(afterApply, savedPlan, "0.10.0", toolDigest); err != nil {
		t.Fatalf("verify applied plan: %v", err)
	}

	changedInput := filepath.Join(root, "resources", "module.yaml")
	data, err := os.ReadFile(changedInput)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changedInput, append(data, []byte("\n# changed after verification\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	stale, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.VerifyProjectionPlan(stale, savedPlan, "0.10.0", toolDigest); err == nil || !strings.Contains(err.Error(), "source inputs changed") {
		t.Fatalf("source change should invalidate saved reconcile evidence: %v", err)
	}
}

func TestCanonicalRelationEffectsSeparateContextFromInvalidation(t *testing.T) {
	root := filepath.Join(t.TempDir(), "canonical-engineering")
	copyCanonicalExample(t, root)
	before, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	owned := filepath.Join(root, "resources", "order-store.yaml")
	content, err := os.ReadFile(owned)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(content), "Persist and retrieve order state.", "Persist and retrieve order state with optimistic concurrency.", 1)
	if changed == string(content) {
		t.Fatal("could not locate owned Interface description")
	}
	if err := os.WriteFile(owned, []byte(changed), 0644); err != nil {
		t.Fatal(err)
	}
	after, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	impact := app.Changes(before, after)
	affected := map[string]bool{}
	for _, key := range impact.Affected {
		affected[key] = true
	}
	for _, key := range []string{"engineering/software.markitect.org/v1alpha1/Module/orders", "engineering/Skill/architecture-review"} {
		if !affected[key] {
			t.Errorf("owns invalidation should affect %s: %#v", key, impact.Affected)
		}
	}
	context, err := app.CompileContext(before, "engineering/Skill/architecture-review", "example-test")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range context.Inputs {
		if input.Resource != nil && input.Resource.GraphKey() == "engineering/software.markitect.org/v1alpha1/Interface/order-store" {
			t.Fatal("owns relation unexpectedly added its target to Skill context")
		}
	}
}

func TestCanonicalDotNetAdapterObservesAndVerifiesProjectDependencies(t *testing.T) {
	root := canonicalEngineeringRoot(t)
	repo := filepath.Dir(filepath.Dir(root))
	buildDir := t.TempDir()
	adapterName := "markitect-adapter-dotnet"
	if os.PathSeparator == '\\' {
		adapterName += ".exe"
	}
	adapterPath := filepath.Join(buildDir, adapterName)
	t.Setenv("GOCACHE", filepath.Join(t.TempDir(), "go-build-cache"))
	command := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", adapterPath, "./cmd/markitect-adapter-dotnet")
	command.Dir = repo
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build the reference adapter: %v\n%s", err, output)
	}
	t.Setenv("PATH", buildDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	project, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	observation, err := app.ObserveCommandAdapter(project, "dotnet-dependencies")
	if err != nil {
		t.Fatal(err)
	}
	if observation.Status != "complete" || len(observation.Findings) != 0 {
		t.Fatalf("literal ProjectReference evidence does not match canonical dependsOn: %#v", observation)
	}
	toolDigest := app.Hash([]byte("test-markitect-tool"))
	plan, err := app.PlanCommandAdapter(project, "dotnet-dependencies", "0.10.0", toolDigest)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Result.Status != "complete" {
		t.Fatalf("adapter plan is incomplete: %#v", plan.Result)
	}
	verified, err := app.VerifyCommandAdapter(project, "dotnet-dependencies", "0.10.0", toolDigest, plan)
	if err != nil {
		t.Fatalf("verify unchanged project evidence: %v", err)
	}
	if verified.Status != "complete" {
		t.Fatalf("verification is incomplete: %#v", verified)
	}
}

func TestCanonicalPolicyValueDrivesCheckContextAndHumanView(t *testing.T) {
	root := canonicalEngineeringRoot(t)
	modulePath := filepath.Join(root, "resources", "module.yaml")
	domainPath := filepath.Join(root, "domains", "software.yaml")
	moduleOriginal, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	domainOriginal, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.WriteFile(modulePath, moduleOriginal, 0644)
	defer os.WriteFile(domainPath, domainOriginal, 0644)

	moduleForModuleDependency := strings.Replace(string(moduleOriginal), "kind: Core\n          name: platform-core", "kind: Module\n          name: payments", 1)
	if moduleForModuleDependency == string(moduleOriginal) {
		t.Fatal("test could not create its non-cyclic Module dependency fixture")
	}
	if err := os.WriteFile(modulePath, []byte(moduleForModuleDependency), 0644); err != nil {
		t.Fatal(err)
	}
	baseline, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(baseline.Diagnostics) == 0 {
		t.Fatal("Module dependency unexpectedly passed the Core-only policy")
	}

	domain, err := format.ParseDomain("domains/software.yaml", domainOriginal)
	if err != nil {
		t.Fatal(err)
	}
	policyFound := false
	for i := range domain.Constraints {
		if domain.Constraints[i].Name == "application-modules-use-core" {
			domain.Constraints[i].Assert.Values = []any{"Module"}
			policyFound = true
		}
	}
	if !policyFound {
		t.Fatal("test could not locate the canonical allowed-targets constraint")
	}
	changedPolicy, err := format.EncodeDomain(domain)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(domainPath, changedPolicy, 0644); err != nil {
		t.Fatal(err)
	}
	updated, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Diagnostics) != 0 {
		t.Fatalf("the same Module dependency should pass after the canonical policy value changes: %#v", updated.Diagnostics)
	}

	compiled, err := app.CompileContext(updated, "engineering/Skill/architecture-review", "example-test")
	if err != nil {
		t.Fatal(err)
	}
	var selectedPolicy string
	for _, input := range compiled.Inputs {
		if input.Role == "domain" && input.Path == "domains/software.yaml" {
			selectedPolicy = input.Text
		}
	}
	definition, err := format.ParseDomain("domains/software.yaml", []byte(selectedPolicy))
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.Constraints) != 1 || len(definition.Constraints[0].Assert.Values) != 1 || definition.Constraints[0].Assert.Values[0] != "Module" {
		t.Fatalf("Skill context does not contain the revised canonical assertion value: %#v", definition.Constraints)
	}
	outputs, err := render.Generate(updated.Graph, updated.Snapshot.Files)
	if err != nil {
		t.Fatal(err)
	}
	var moduleView string
	for name, content := range outputs {
		if strings.Contains(string(content), "application-modules-use-core") && strings.HasSuffix(name, ".md") {
			moduleView = string(content)
		}
	}
	if !strings.Contains(moduleView, "relationships may target only kinds `Module`") || strings.Contains(moduleView, "target only kinds `Core`") {
		t.Fatalf("human Module view did not derive the same revised policy value: %s", moduleView)
	}
}

func TestCanonicalEngineeringRejectsBrokenRelations(t *testing.T) {
	root := canonicalEngineeringRoot(t)
	modulePath := filepath.Join(root, "resources", "module.yaml")
	original, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.WriteFile(modulePath, original, 0644)

	cases := []struct {
		name, replacement, value, want string
	}{
		{"missing reference", "name: platform-core", "name: missing-core", "reference"},
		{"wrong target kind", "kind: Core\n          name: platform-core", "kind: Interface\n          name: order-store", "relation"},
		{"dependency cycle", "kind: Core\n          name: platform-core", "kind: Module\n          name: orders", "cycle"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Replace(string(original), tc.replacement, tc.value, 1)
			if text == string(original) {
				t.Fatalf("test could not locate %q", tc.replacement)
			}
			if err := os.WriteFile(modulePath, []byte(text), 0644); err != nil {
				t.Fatal(err)
			}
			project, err := app.Load(root, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(project.Diagnostics) == 0 {
				t.Fatalf("%s unexpectedly passed", tc.name)
			}
			found := false
			for _, diagnostic := range project.Diagnostics {
				if strings.Contains(strings.ToLower(diagnostic.Code), tc.want) || strings.Contains(strings.ToLower(diagnostic.Message), tc.want) {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %q diagnostic, got %#v", tc.want, project.Diagnostics)
			}
		})
	}
}
