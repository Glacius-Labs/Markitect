package dotnet

import (
	"reflect"
	"testing"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

const (
	ordersModule = "engineering/software.markitect.org/v1alpha1/Module/orders"
	coreResource = "engineering/software.markitect.org/v1alpha1/Core/platform-core"
)

func TestRunObservesOnlyMappedProjectReferences(t *testing.T) {
	req := validRequest("observe")
	req.Model.Resources = append(req.Model.Resources, core.ModelResource{Identity: core.ModelIdentity{Kind: "Module", Key: "unmapped/resource"}})
	got := Run(req)
	want := map[string][]string{ordersModule: {coreResource}, coreResource: {}}
	if got.Status != "complete" || len(got.Findings) != 0 || got.Observed == nil || !reflect.DeepEqual(got.Observed.Dependencies, want) {
		t.Fatalf("observe = %#v", got)
	}
}

func TestRunRejectsInvalidMappingsAndMissingBytes(t *testing.T) {
	req := validRequest("observe")
	req.Config.ProjectMappings[0].Resource = "engineering/Module/orders"
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "resource-mapping-invalid") {
		t.Fatalf("unknown mapping = %#v", got)
	}
	req = validRequest("observe")
	req.Captures = nil
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "project-input-missing") {
		t.Fatalf("missing captured bytes = %#v", got)
	}
	req = validRequest("observe")
	req.Config.ProjectMappings[0].ProjectFile = "../outside.csproj"
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "project-path-invalid") {
		t.Fatalf("escaping path = %#v", got)
	}
	req = validRequest("observe")
	req.Config.ProjectMappings = append(req.Config.ProjectMappings, projectMapping{Resource: ordersModule, ProjectFile: "src/Other/Other.csproj"})
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "resource-mapping-ambiguous") {
		t.Fatalf("duplicate resource mapping = %#v", got)
	}
	req = validRequest("observe")
	req.Model.Relationships = append(req.Model.Relationships, core.ModelRelationship{From: ordersModule, To: "missing/resource", Type: "dependsOn"})
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "dependency-target-invalid") {
		t.Fatalf("missing dependency target = %#v", got)
	}
}

func TestRunReportsMissingAndForbiddenEdges(t *testing.T) {
	req := validRequest("observe")
	other := "engineering/software.markitect.org/v1alpha1/Module/other"
	req.Model.Resources = append(req.Model.Resources, core.ModelResource{Identity: core.ModelIdentity{Kind: "Module", Key: other}})
	req.Model.Relationships = append(req.Model.Relationships, core.ModelRelationship{From: other, To: coreResource, Type: "dependsOn"})
	req.Config.ProjectMappings = append(req.Config.ProjectMappings, projectMapping{Resource: other, ProjectFile: "src/Other/Other.csproj"})
	req.Captures["src/Other/Other.csproj"] = []byte(`<Project><ItemGroup><ProjectReference Include="../Orders/Orders.csproj"/></ItemGroup></Project>`)
	got := Run(req)
	if got.Status != "complete" || !hasFinding(got, "dependency-forbidden") || !hasFinding(got, "dependency-missing") {
		t.Fatalf("dependency findings = %#v", got)
	}
}

func TestRunRequiresAndBindsObservationAndPlan(t *testing.T) {
	req := validRequest("plan")
	if got := Run(req); got.Status != "incomplete" || !hasFinding(got, "observation-missing") {
		t.Fatalf("missing observation = %#v", got)
	}
	observeReq := validRequest("observe")
	observed := Run(observeReq)
	req.Observation = &observed
	planned := Run(req)
	if planned.Status != "complete" {
		t.Fatalf("plan = %#v", planned)
	}
	verifyReq := validRequest("verify")
	verifyReq.Plan = &planned
	if got := Run(verifyReq); got.Status != "complete" {
		t.Fatalf("verify = %#v", got)
	}
	verifyReq.Captures["src/Orders/Orders.csproj"] = []byte(`<Project />`)
	if got := Run(verifyReq); got.Status != "complete" || !hasFinding(got, "verification-drift") {
		t.Fatalf("changed bytes must produce drift = %#v", got)
	}
}

func TestRunRejectsUnsupportedActionsAndInvalidConfig(t *testing.T) {
	req := validRequest("apply")
	if got := Run(req); got.Status != "failed" || !hasFinding(got, "unsupported-action") {
		t.Fatalf("apply = %#v", got)
	}
	if _, err := DecodeConfig(map[string]any{"inferMappings": true}); err == nil {
		t.Fatal("unknown config accepted")
	}
	req = validRequest("observe")
	req.Model.ValidationStatus = "failed"
	if got := Run(req); got.Status != "failed" || !hasFinding(got, "model-not-validated") {
		t.Fatalf("invalid model = %#v", got)
	}
}

func TestProjectReferenceExpressionsAndEscapesAreUnsupported(t *testing.T) {
	for _, include := range []string{"$(ProjectDir)Core.csproj", "../../outside.csproj", "../Core/*.csproj"} {
		req := validRequest("observe")
		req.Captures["src/Orders/Orders.csproj"] = []byte(`<Project><ItemGroup><ProjectReference Include="` + include + `"/></ItemGroup></Project>`)
		if got := Run(req); got.Status != "incomplete" {
			t.Fatalf("Include %q result = %#v", include, got)
		}
	}
}

func TestProjectReferencesRespectMSBuildNamespaces(t *testing.T) {
	for _, xml := range []string{
		`<Project><ItemGroup><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`,
		`<Project xmlns="http://schemas.microsoft.com/developer/msbuild/2003"><ItemGroup><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`,
	} {
		req := validRequest("observe")
		req.Captures["src/Orders/Orders.csproj"] = []byte(xml)
		got := Run(req)
		if got.Status != "complete" {
			t.Fatalf("supported namespace result = %#v", got)
		}
	}
	for _, xml := range []string{
		`<Project xmlns="urn:foreign"><ItemGroup><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`,
		`<Project><x:ItemGroup xmlns:x="urn:foreign"><ProjectReference Include="../Core/Core.csproj" /></x:ItemGroup></Project>`,
		`<Project xmlns:x="urn:foreign"><ItemGroup x:Condition="true"><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`,
	} {
		req := validRequest("observe")
		req.Captures["src/Orders/Orders.csproj"] = []byte(xml)
		got := Run(req)
		if got.Status != "incomplete" || !hasFinding(got, "project-semantics-unsupported") {
			t.Fatalf("foreign namespace must be incomplete: %#v", got)
		}
	}
}

func validRequest(action string) Request {
	return Request{
		APIVersion: requestAPIVersion,
		Action:     action,
		Adapter:    Identity{Name: "dotnet-architecture", Type: "command", Version: "v0.1.0"},
		Model: core.SemanticModel{
			APIVersion: semanticModelAPI, ModelDigest: "sha256:model", ValidationStatus: "passed",
			Resources: []core.ModelResource{
				{Identity: core.ModelIdentity{Kind: "Module", Key: ordersModule}},
				{Identity: core.ModelIdentity{Kind: "Core", Key: coreResource}},
			},
			Relationships: []core.ModelRelationship{{From: ordersModule, To: coreResource, Type: "dependsOn"}},
		},
		Config: Config{ProjectMappings: []projectMapping{
			{Resource: ordersModule, ProjectFile: "src/Orders/Orders.csproj"},
			{Resource: coreResource, ProjectFile: "src/Core/Core.csproj"},
		}},
		Captures: map[string][]byte{
			"src/Orders/Orders.csproj": []byte(`<Project><ItemGroup><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`),
			"src/Core/Core.csproj":     []byte(`<Project />`),
		},
		CaptureErrors: map[string]string{},
	}
}

func hasFinding(result Result, code string) bool {
	for _, finding := range result.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func TestCapturePathsOnlyReturnsMappedSafePaths(t *testing.T) {
	config := Config{ProjectMappings: []projectMapping{{ProjectFile: "a.csproj"}, {ProjectFile: "../escape.csproj"}, {ProjectFile: "readme.md"}, {ProjectFile: "a\\b.csproj"}, {ProjectFile: "a.csproj"}}}
	if got := CapturePaths(config); !reflect.DeepEqual(got, []string{"a.csproj", "a/b.csproj"}) {
		t.Fatalf("capture paths = %v", got)
	}
}
