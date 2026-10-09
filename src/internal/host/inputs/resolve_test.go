package inputs

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

func fixtureResource(kind, namespace, name, file string) *authoring.Resource {
	return &authoring.Resource{Core: core.Resource{Kind: kind, Metadata: core.Metadata{Name: name, Namespace: namespace}, Path: file}}
}

func diagnosticExists(diagnostics []core.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestResolveUsesExplicitFilesAndEnforcesAreaImports(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Areas = []authoring.Area{
		{Name: "general", Path: "docs/general", Imports: []string{"client-a"}},
		{Name: "client-a", Path: "docs/customers/a"},
		{Name: "client-b", Path: "docs/customers/b"},
	}
	workflow := fixtureResource("Workflow", "general", "review", "docs/general/review.yaml")
	workflow.Spec.Files = []string{"docs/general/schema.json", "docs/customers/a/contract.md", "docs/general/missing.md", "docs/customers/b/private.md"}
	client := fixtureResource("Contract", "client-a", "api", "docs/customers/a/api.yaml")
	client.Spec.Files = []string{"docs/customers/b/private.md"}
	g := &authoring.Graph{Project: project, Resources: map[string]*authoring.Resource{project.Key(): project, workflow.Key(): workflow, client.Key(): client}}
	files := map[string][]byte{
		"docs/general/schema.json":     {},
		"docs/customers/a/contract.md": {},
		"docs/customers/b/private.md":  {},
	}
	resolved, diagnostics := Resolve(g, files, ProjectionScope{})
	if got := resolved[workflow.Key()]; len(got) != 2 || got[0] != "docs/general/schema.json" || got[1] != "docs/customers/a/contract.md" {
		t.Fatalf("resolved paths = %#v", got)
	}
	for _, code := range []string{"input.missing", "input.scope"} {
		if !diagnosticExists(diagnostics, code) {
			t.Errorf("expected %s diagnostic, got %#v", code, diagnostics)
		}
	}
	if len(resolved[client.Key()]) != 0 {
		t.Errorf("cross-client file was resolved: %#v", resolved[client.Key()])
	}
}

func TestResolveRejectsUnsafeDirectoryConfigAndTypedCompanionPaths(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Areas = []authoring.Area{{Name: "general", Path: "docs/general"}}
	workflow := fixtureResource("Workflow", "general", "review", "docs/general/review.yaml")
	workflow.Spec.Files = []string{
		"../private.md",
		"docs/general/*.md",
		"docs/general/directory",
		"markitect.yaml",
		"docs/general/review.md",
	}
	g := &authoring.Graph{Project: project, Resources: map[string]*authoring.Resource{project.Key(): project, workflow.Key(): workflow}}
	files := map[string][]byte{"docs/general/directory/child.md": {}, "docs/general/review.md": {}}
	resolved, diagnostics := Resolve(g, files, ProjectionScope{})
	if len(resolved[workflow.Key()]) != 1 || resolved[workflow.Key()][0] != "docs/general/review.md" {
		t.Fatalf("disabled Markdown view path should remain an ordinary file input: %#v", resolved[workflow.Key()])
	}
	for _, code := range []string{"input.path", "input.directory", "input.project-config"} {
		if !diagnosticExists(diagnostics, code) {
			t.Errorf("expected %s diagnostic, got %#v", code, diagnostics)
		}
	}
}

func TestResolveRejectsCaseAliasOfSelectedMarkdownView(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Areas = []authoring.Area{
		{Name: "general", Path: "docs/general", Imports: []string{"view-inputs"}},
		{Name: "view-inputs", Path: "DOCS/MARKITECT"},
	}
	project.Spec.Targets = []string{"markdown"}
	workflow := fixtureResource("Workflow", "general", "review", "docs/general/review.yaml")
	workflow.Spec.Files = []string{"DOCS/MARKITECT/GENERAL/REVIEW.WORKFLOW.MD"}
	g := &authoring.Graph{Project: project, Resources: map[string]*authoring.Resource{project.Key(): project, workflow.Key(): workflow}, ResourceAreas: map[string]authoring.Area{workflow.GraphKey(): {Name: "general", Path: "docs/general"}}}
	scope := ProjectionScope{TypedViews: map[string]string{"docs/markitect/general/review.workflow.md": "general/Workflow/review"}}
	_, diagnostics := Resolve(g, map[string][]byte{workflow.Spec.Files[0]: []byte("unmarked aliased Markdown view")}, scope)
	if len(diagnostics) != 1 || diagnostics[0].Code != "input.typed-companion" {
		t.Fatalf("case alias of selected Markdown view was not rejected as a typed view: %#v", diagnostics)
	}
	project.Spec.Targets = nil
	resolved, diagnostics := Resolve(g, map[string][]byte{workflow.Spec.Files[0]: []byte("unmarked aliased Markdown view")}, ProjectionScope{})
	if len(diagnostics) != 0 || len(resolved[workflow.GraphKey()]) != 1 {
		t.Fatalf("same scope-valid file must be ordinary input with Markdown disabled: %#v, %#v", resolved, diagnostics)
	}
}

func TestResolveRejectsUnmarkedCaseAliasOfSelectedNativeOutput(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Areas = []authoring.Area{
		{Name: "general", Path: "docs/general", Imports: []string{"native-inputs"}},
		{Name: "native-inputs", Path: ".Agents"},
	}
	project.Spec.Targets = []string{"codex"}
	skill := fixtureResource("Skill", "general", "reviewer", "docs/general/reviewer.yaml")
	skill.Spec.Files = []string{".Agents/skills/reviewer/SKILL.md"}
	g := &authoring.Graph{Project: project, Resources: map[string]*authoring.Resource{project.Key(): project, skill.Key(): skill}}
	scope := ProjectionScope{OutputPaths: map[string]bool{".agents/skills/reviewer/SKILL.md": true}}
	_, diagnostics := Resolve(g, map[string][]byte{skill.Spec.Files[0]: []byte("unmarked native output path")}, scope)
	if len(diagnostics) != 1 || diagnostics[0].Code != "input.generated-output" {
		t.Fatalf("unmarked case alias of selected native output was not rejected: %#v", diagnostics)
	}
	project.Spec.Targets = nil
	resolved, diagnostics := Resolve(g, map[string][]byte{skill.Spec.Files[0]: []byte("unmarked native output path")}, ProjectionScope{})
	if len(diagnostics) != 0 || len(resolved[skill.GraphKey()]) != 1 {
		t.Fatalf("same scope-valid file must be ordinary input with Codex disabled: %#v, %#v", resolved, diagnostics)
	}
}

func TestResolveRejectsMarkedGeneratedOrdinaryFileInput(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Areas = []authoring.Area{{Name: "general", Path: "docs/general"}}
	workflow := fixtureResource("Workflow", "general", "review", "docs/general/review.yaml")
	workflow.Spec.Files = []string{"docs/general/generated.md"}
	g := &authoring.Graph{Project: project, Resources: map[string]*authoring.Resource{project.Key(): project, workflow.Key(): workflow}}
	_, diagnostics := Resolve(g, map[string][]byte{workflow.Spec.Files[0]: []byte("<!-- Generated by Markitect; source: docs/general/review.yaml -->\n")}, ProjectionScope{})
	if len(diagnostics) != 1 || diagnostics[0].Code != "input.generated-output" {
		t.Fatalf("marker-bearing generated file was accepted as an input: %#v", diagnostics)
	}
}

func TestPackageOrdinaryInputMayShareLocalSelectedOutputPath(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Targets = []string{"markdown"}
	project.Spec.Areas = []authoring.Area{{Name: "local", Path: "docs/local"}}
	local := fixtureResource("Workflow", "local", "review", "docs/local/review.yaml")
	manifest := fixtureResource("Package", "", "shared", "markitect-package.yaml")
	manifest.Package = "shared"
	manifest.Spec.Areas = []authoring.Area{{Name: "shared", Path: "docs/markitect/local"}}
	imported := fixtureResource("Text", "shared", "policy", "docs/markitect/local/review.workflow.md")
	imported.Package = "shared"
	imported.Spec.Files = []string{"docs/markitect/local/review.workflow.md"}
	g := &authoring.Graph{Project: project, Packages: map[string]*authoring.Resource{"shared": manifest}, Resources: map[string]*authoring.Resource{
		project.Key(): project, local.Key(): local, manifest.GraphKey(): manifest, imported.GraphKey(): imported,
	}, ResourceAreas: map[string]authoring.Area{local.GraphKey(): {Name: "local", Path: "docs/local"}}}
	resolved, diagnostics := ResolveWithPackages(g, map[string][]byte{}, map[string]map[string][]byte{"shared": {
		"docs/markitect/local/review.workflow.md": []byte("package owned evidence"),
	}}, ProjectionScope{})
	if len(diagnostics) != 0 {
		t.Fatalf("package-origin file was classified as local generated output: %#v", diagnostics)
	}
	if got := resolved[imported.GraphKey()]; len(got) != 1 || got[0] != imported.Spec.Files[0] {
		t.Fatalf("resolved package input = %#v", got)
	}
}

func TestResolveRequiresGraphProjectAndRejectsProjectFiles(t *testing.T) {
	if _, diagnostics := Resolve(nil, nil, ProjectionScope{}); !diagnosticExists(diagnostics, "input.graph") {
		t.Fatalf("expected nil graph diagnostic: %#v", diagnostics)
	}
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Files = []string{"docs/general/notes.md"}
	g := &authoring.Graph{Project: project, Resources: map[string]*authoring.Resource{project.Key(): project}}
	if _, diagnostics := Resolve(g, nil, ProjectionScope{}); !diagnosticExists(diagnostics, "input.project-files") {
		t.Fatalf("expected Project files diagnostic: %#v", diagnostics)
	}
}

func TestResolveRejectsBinaryOrInvalidUTF8InputContent(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Areas = []authoring.Area{{Name: "general", Path: "docs/general"}}
	contract := fixtureResource("Contract", "general", "api", "docs/general/api.yaml")
	contract.Spec.Files = []string{"docs/general/invalid.md", "docs/general/binary.dat"}
	g := &authoring.Graph{Project: project, Resources: map[string]*authoring.Resource{project.Key(): project, contract.Key(): contract}}
	files := map[string][]byte{
		"docs/general/invalid.md": []byte{0xff, 0xfe},
		"docs/general/binary.dat": []byte("content\x00payload"),
	}
	resolved, diagnostics := Resolve(g, files, ProjectionScope{})
	if len(resolved[contract.Key()]) != 0 {
		t.Fatalf("binary paths were resolved: %#v", resolved[contract.Key()])
	}
	if !diagnosticExists(diagnostics, "input.binary") {
		t.Fatalf("expected binary diagnostic, got %#v", diagnostics)
	}
}

func TestResolveWithPackagesSeparatesOriginsAndCannotReadConsumerFiles(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Areas = []authoring.Area{{Name: "local", Path: "docs/local"}}
	manifest := fixtureResource("Package", "", "policy-set", "markitect-package.yaml")
	manifest.Package = "policy-set"
	manifest.Spec.Areas = []authoring.Area{{Name: "content", Path: "content"}}
	local := fixtureResource("Workflow", "local", "build", "docs/local/build.yaml")
	local.Spec.Files = []string{"docs/local/evidence.md"}
	imported := fixtureResource("Rule", "content", "policy", "content/rules/policy.yaml")
	imported.Package = "policy-set"
	imported.Spec.Files = []string{"content/data.md", "content/evidence.md", "../escape.md"}
	g := &authoring.Graph{
		Project:  project,
		Packages: map[string]*authoring.Resource{"policy-set": manifest},
		Resources: map[string]*authoring.Resource{
			project.Key():       project,
			local.Key():         local,
			imported.GraphKey(): imported,
			manifest.GraphKey(): manifest,
		},
	}
	consumerFiles := map[string][]byte{"docs/local/evidence.md": []byte("consumer-only"), "content/evidence.md": []byte("consumer-only")}
	archiveFiles := map[string]map[string][]byte{"policy-set": {
		"content/data.md": []byte("package-only"),
	}}
	resolved, diagnostics := ResolveWithPackages(g, consumerFiles, archiveFiles, ProjectionScope{})
	if got := resolved[local.GraphKey()]; len(got) != 1 || got[0] != "docs/local/evidence.md" {
		t.Fatalf("local inputs = %#v", got)
	}
	if got := resolved[imported.GraphKey()]; len(got) != 1 || got[0] != "content/data.md" {
		t.Fatalf("package inputs = %#v", got)
	}
	for _, code := range []string{"input.missing", "input.path"} {
		if !diagnosticExists(diagnostics, code) {
			t.Errorf("expected %s for package-origin isolation, got %#v", code, diagnostics)
		}
	}
}
