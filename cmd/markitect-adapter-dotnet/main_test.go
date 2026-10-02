package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"go.yaml.in/yaml/v3"
)

const (
	ordersModule = "engineering/software.markitect.org/v1alpha1/Module/orders"
	coreModule   = "engineering/software.markitect.org/v1alpha1/Core/platform-core"
	adapterID    = "dotnet-architecture"
)

func TestObserveAcceptsCanonicalProjectReferences(t *testing.T) {
	root := fixture(t, `<Project Sdk="Microsoft.NET.Sdk"><ItemGroup><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`)
	defer os.Chdir(mustGetwd(t))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	req := validRequest("observe")
	// Unmapped resources remain outside this explicit adapter's scope.
	req.Model.Resources = append(req.Model.Resources, resource{Identity: resourceIdentity{Kind: "Module", Key: "engineering/software.markitect.org/v1alpha1/Module/payments"}})
	got := run(req)
	if got.Status != "complete" || len(got.Findings) != 0 {
		t.Fatalf("status/findings = %s/%v", got.Status, got.Findings)
	}
	want := map[string][]string{ordersModule: {coreModule}, coreModule: {}}
	if got.Observed.Evidence != "literal-unconditional-project-reference-xml" || !reflect.DeepEqual(got.Observed.Scope, []string{coreModule, ordersModule}) || !reflect.DeepEqual(got.Observed.Dependencies, want) {
		t.Fatalf("observed = %#v, want scoped literal-reference evidence %#v", got.Observed, want)
	}
}

func TestMappingMustUseAnExistingExactResourceIdentity(t *testing.T) {
	root := fixture(t, `<Project />`)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	req := validRequest("observe")
	projects := req.Adapter.Parameters["projectMappings"].([]projectMapping)
	projects[0].Resource = "engineering/Module/orders" // Not the qualified semantic-model key.
	req.Adapter.Parameters["projectMappings"] = projects
	got := run(req)
	if got.Status != "incomplete" || !hasCode(got.Findings, "resource-mapping-invalid") {
		t.Fatalf("status/findings = %s/%v", got.Status, got.Findings)
	}
}

func TestUnknownAdapterParameterFailsClosed(t *testing.T) {
	root := fixture(t, `<Project />`)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	req := validRequest("observe")
	req.Adapter.Parameters["inferMappings"] = true
	got := run(req)
	if got.Status != "failed" || !hasCode(got.Findings, "adapter-parameters-invalid") {
		t.Fatalf("status/findings = %s/%v", got.Status, got.Findings)
	}
}

func TestObserveReportsForbiddenAndMissingDependencies(t *testing.T) {
	root := fixture(t, `<Project><ItemGroup><ProjectReference Include="../Core/Core.csproj"/><ProjectReference Include="../Other/Other.csproj"/></ItemGroup></Project>`)
	defer os.Chdir(mustGetwd(t))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	req := validRequest("observe")
	otherModule := "engineering/software.markitect.org/v1alpha1/Module/other"
	projects := req.Adapter.Parameters["projectMappings"].([]projectMapping)
	req.Adapter.Parameters["projectMappings"] = append(projects, projectMapping{Resource: otherModule, ProjectFile: "src/Other/Other.csproj"})
	req.Model.Resources = append(req.Model.Resources, resource{Identity: resourceIdentity{Kind: "Module", Key: otherModule}})
	req.Model.Relationships = append(req.Model.Relationships, relationship{From: otherModule, To: coreModule, Type: "dependsOn"})
	if err := os.MkdirAll(filepath.Join(root, "src", "Other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "Other", "Other.csproj"), []byte(`<Project />`), 0o600); err != nil {
		t.Fatal(err)
	}

	got := run(req)
	if got.Status != "complete" {
		t.Fatalf("status = %s, findings: %v", got.Status, got.Findings)
	}
	codes := map[string]bool{}
	for _, item := range got.Findings {
		codes[item.Code] = true
	}
	if !codes["dependency-forbidden"] || !codes["dependency-missing"] {
		t.Fatalf("findings = %#v, want forbidden and missing dependencies", got.Findings)
	}
}

func TestMissingMappingsAndCapturedInputsAreIncomplete(t *testing.T) {
	root := fixture(t, `<Project />`)
	defer os.Chdir(mustGetwd(t))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	req := validRequest("observe")
	projects := req.Adapter.Parameters["projectMappings"].([]projectMapping)
	req.Adapter.Parameters["projectMappings"] = projects[:1]
	got := run(req)
	if got.Status != "incomplete" || !hasCode(got.Findings, "dependency-target-unmapped") {
		t.Fatalf("missing map status/findings = %s/%v", got.Status, got.Findings)
	}

	req = validRequest("observe")
	projects = req.Adapter.Parameters["projectMappings"].([]projectMapping)
	projects[0].ProjectFile = "src/Missing/Missing.csproj"
	req.Adapter.Parameters["projectMappings"] = projects
	got = run(req)
	if got.Status != "incomplete" || !hasCode(got.Findings, "project-input-missing") {
		t.Fatalf("missing input status/findings = %s/%v", got.Status, got.Findings)
	}
}

func TestUnsupportedActionFailsClosed(t *testing.T) {
	got := run(validRequest("apply"))
	if got.Status != "failed" || !hasCode(got.Findings, "unsupported-action") {
		t.Fatalf("status/findings = %s/%v", got.Status, got.Findings)
	}
}

func TestUnvalidatedSemanticModelFailsClosed(t *testing.T) {
	req := validRequest("observe")
	req.Model.ValidationStatus = "failed"
	got := run(req)
	if got.Status != "failed" || !hasCode(got.Findings, "model-not-validated") {
		t.Fatalf("status/findings = %s/%v", got.Status, got.Findings)
	}
}

func TestVerifyDetectsChangedProjectSnapshot(t *testing.T) {
	root := fixture(t, `<Project />`)
	defer os.Chdir(mustGetwd(t))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	req := validRequest("verify")
	req.Plan = &result{Adapter: adapterID, Action: "plan", Status: "complete", ModelDigest: "model-digest", Observed: &observation{Evidence: "literal-unconditional-project-reference-xml", Scope: []string{coreModule, ordersModule}, Dependencies: map[string][]string{ordersModule: {coreModule}, coreModule: {}}}}
	got := run(req)
	if got.Status != "complete" || !hasCode(got.Findings, "verification-drift") {
		t.Fatalf("status/findings = %s/%v", got.Status, got.Findings)
	}
}

func TestProjectReferenceExpressionsAndPathEscapeAreIncomplete(t *testing.T) {
	root := fixture(t, `<Project><ItemGroup><ProjectReference Include="$(SharedProject)" /></ItemGroup></Project>`)
	defer os.Chdir(mustGetwd(t))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	got := run(validRequest("observe"))
	if got.Status != "incomplete" || !hasCode(got.Findings, "project-reference-unsupported") {
		t.Fatalf("expression status/findings = %s/%v", got.Status, got.Findings)
	}
	if _, err := resolveProjectReference("src/Orders/Orders.csproj", "../../../outside.csproj"); err == nil {
		t.Fatal("path escaping staged inputs was accepted")
	}
}

func TestUnsupportedMSBuildSemanticsAreIncomplete(t *testing.T) {
	for _, project := range []string{
		`<Project><ItemGroup Condition="'$(Configuration)' == 'Debug'"><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`,
		`<Project><Import Project="shared.props" /><ItemGroup><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`,
		`<Project><ItemGroup><ProjectReference Include="../Core/$(Target).csproj" /></ItemGroup></Project>`,
		`<Project><ItemGroup><ProjectReference Include="../Core/*.csproj" /></ItemGroup></Project>`,
	} {
		t.Run(project, func(t *testing.T) {
			root := fixture(t, project)
			old, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(root); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(old)
			got := run(validRequest("observe"))
			if got.Status != "incomplete" || (!hasCode(got.Findings, "project-semantics-unsupported") && !hasCode(got.Findings, "project-reference-unsupported")) {
				t.Fatalf("status/findings = %s/%v", got.Status, got.Findings)
			}
		})
	}
}

func TestUnmappedObservedProjectIsIncomplete(t *testing.T) {
	root := fixture(t, `<Project><ItemGroup><ProjectReference Include="../Other/Other.csproj" /></ItemGroup></Project>`)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	got := run(validRequest("observe"))
	if got.Status != "incomplete" || !hasCode(got.Findings, "project-reference-unmapped") {
		t.Fatalf("status/findings = %s/%v", got.Status, got.Findings)
	}
}

func TestDuplicateMappingIsIncomplete(t *testing.T) {
	root := fixture(t, `<Project />`)
	defer os.Chdir(mustGetwd(t))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	req := validRequest("observe")
	projects := req.Adapter.Parameters["projectMappings"].([]projectMapping)
	req.Adapter.Parameters["projectMappings"] = append(projects, projects[0])
	got := run(req)
	if got.Status != "incomplete" || !hasCode(got.Findings, "resource-mapping-ambiguous") {
		t.Fatalf("status/findings = %s/%v", got.Status, got.Findings)
	}
}

func TestPlanRequiresAndChecksObservation(t *testing.T) {
	root := fixture(t, `<Project Sdk="Microsoft.NET.Sdk"><ItemGroup><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	req := validRequest("plan")
	got := run(req)
	if got.Status != "incomplete" || !hasCode(got.Findings, "observation-missing") {
		t.Fatalf("missing observation status/findings = %s/%v", got.Status, got.Findings)
	}
	req.Observation = &result{Adapter: adapterID, Action: "observe", Status: "complete", ModelDigest: req.Model.ModelDigest, Observed: &observation{Evidence: "literal-unconditional-project-reference-xml", Scope: []string{coreModule, ordersModule}, Dependencies: map[string][]string{ordersModule: {coreModule}, coreModule: {}}}}
	got = run(req)
	if got.Status != "complete" || got.Action != "plan" {
		t.Fatalf("plan status/action/findings = %s/%s/%v", got.Status, got.Action, got.Findings)
	}
}

func TestBuiltExecutableAcceptsNormalizedYAMLProtocol(t *testing.T) {
	root := fixture(t, `<Project Sdk="Microsoft.NET.Sdk"><ItemGroup><ProjectReference Include="../Core/Core.csproj" /></ItemGroup></Project>`)
	packageDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(t.TempDir(), "go-cache")
	t.Setenv("GOCACHE", cache)
	binary := filepath.Join(t.TempDir(), "markitect-adapter-dotnet")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = packageDirectory
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build adapter: %v\n%s", err, output)
	}
	input, err := yaml.Marshal(validRequest("observe"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary)
	command.Dir = root
	command.Stdin = bytes.NewReader(input)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("invoke adapter: %v", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(output))
	var response result
	if err := decoder.Decode(&response); err != nil {
		t.Fatalf("decode response: %v\n%s", err, output)
	}
	if response.APIVersion != resultAPIVersion || response.Adapter != adapterID || response.Action != "observe" || response.Status != "complete" || response.Observed == nil || response.Observed.Evidence != "literal-unconditional-project-reference-xml" {
		t.Fatalf("response identity/status = %#v", response)
	}
}

func validRequest(action string) request {
	return request{
		APIVersion: requestAPIVersion,
		Action:     action,
		Adapter: adapterRequest{Name: adapterID, Type: "command", Version: "markitect-dotnet/v0.1.0", Parameters: map[string]any{"projectMappings": []projectMapping{
			{Resource: ordersModule, ProjectFile: "src/Orders/Orders.csproj"},
			{Resource: coreModule, ProjectFile: "src/Core/Core.csproj"},
		}}},
		Model: semanticModel{
			APIVersion:       "markitect.example.org/semantic-model/v1alpha1",
			ModelDigest:      "model-digest",
			ValidationStatus: "passed",
			Resources: []resource{
				{Identity: resourceIdentity{Kind: "Module", Key: ordersModule}},
				{Identity: resourceIdentity{Kind: "Module", Key: coreModule}},
			},
			Relationships: []relationship{{From: ordersModule, To: coreModule, Type: "dependsOn"}},
		},
	}
}

func fixture(t *testing.T, ordersXML string) string {
	t.Helper()
	root := t.TempDir()
	for path, body := range map[string]string{
		"src/Orders/Orders.csproj": ordersXML,
		"src/Core/Core.csproj":     `<Project />`,
	} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

func hasCode(items []finding, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}
