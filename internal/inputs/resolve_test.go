package inputs

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func fixtureResource(kind, namespace, name, file string) *core.Resource {
	return &core.Resource{Kind: kind, Metadata: core.Metadata{Name: name, Namespace: namespace}, Path: file}
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
	project.Spec.Areas = []core.Area{
		{Name: "general", Path: "docs/general", Imports: []string{"client-a"}},
		{Name: "client-a", Path: "docs/customers/a"},
		{Name: "client-b", Path: "docs/customers/b"},
	}
	workflow := fixtureResource("Workflow", "general", "review", "docs/general/review.yaml")
	workflow.Spec.Files = []string{"docs/general/schema.json", "docs/customers/a/contract.md", "docs/general/missing.md", "docs/customers/b/private.md"}
	client := fixtureResource("Contract", "client-a", "api", "docs/customers/a/api.yaml")
	client.Spec.Files = []string{"docs/customers/b/private.md"}
	g := &core.Graph{Project: project, Resources: map[string]*core.Resource{project.Key(): project, workflow.Key(): workflow, client.Key(): client}}
	files := map[string][]byte{
		"docs/general/schema.json":     {},
		"docs/customers/a/contract.md": {},
		"docs/customers/b/private.md":  {},
	}
	resolved, diagnostics := Resolve(g, files)
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
	project.Spec.Areas = []core.Area{{Name: "general", Path: "docs/general"}}
	workflow := fixtureResource("Workflow", "general", "review", "docs/general/review.yaml")
	workflow.Spec.Files = []string{
		"../private.md",
		"docs/general/*.md",
		"docs/general/directory",
		"markitect.yaml",
		"docs/general/review.md",
	}
	g := &core.Graph{Project: project, Resources: map[string]*core.Resource{project.Key(): project, workflow.Key(): workflow}}
	files := map[string][]byte{"docs/general/directory/child.md": {}, "docs/general/review.md": {}}
	resolved, diagnostics := Resolve(g, files)
	if len(resolved[workflow.Key()]) != 0 {
		t.Fatalf("invalid paths were resolved: %#v", resolved[workflow.Key()])
	}
	for _, code := range []string{"input.path", "input.directory", "input.project-config", "input.typed-companion"} {
		if !diagnosticExists(diagnostics, code) {
			t.Errorf("expected %s diagnostic, got %#v", code, diagnostics)
		}
	}
}

func TestResolveRequiresGraphProjectAndRejectsProjectFiles(t *testing.T) {
	if _, diagnostics := Resolve(nil, nil); !diagnosticExists(diagnostics, "input.graph") {
		t.Fatalf("expected nil graph diagnostic: %#v", diagnostics)
	}
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Files = []string{"docs/general/notes.md"}
	g := &core.Graph{Project: project, Resources: map[string]*core.Resource{project.Key(): project}}
	if _, diagnostics := Resolve(g, nil); !diagnosticExists(diagnostics, "input.project-files") {
		t.Fatalf("expected Project files diagnostic: %#v", diagnostics)
	}
}

func TestResolveRejectsBinaryOrInvalidUTF8InputContent(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Areas = []core.Area{{Name: "general", Path: "docs/general"}}
	contract := fixtureResource("Contract", "general", "api", "docs/general/api.yaml")
	contract.Spec.Files = []string{"docs/general/invalid.md", "docs/general/binary.dat"}
	g := &core.Graph{Project: project, Resources: map[string]*core.Resource{project.Key(): project, contract.Key(): contract}}
	files := map[string][]byte{
		"docs/general/invalid.md": []byte{0xff, 0xfe},
		"docs/general/binary.dat": []byte("content\x00payload"),
	}
	resolved, diagnostics := Resolve(g, files)
	if len(resolved[contract.Key()]) != 0 {
		t.Fatalf("binary paths were resolved: %#v", resolved[contract.Key()])
	}
	if !diagnosticExists(diagnostics, "input.binary") {
		t.Fatalf("expected binary diagnostic, got %#v", diagnostics)
	}
}

func TestResolveWithPackagesSeparatesOriginsAndCannotReadConsumerFiles(t *testing.T) {
	project := fixtureResource("Project", "", "sample", "markitect.yaml")
	project.Spec.Areas = []core.Area{{Name: "local", Path: "docs/local"}}
	manifest := fixtureResource("Package", "", "policy-set", "markitect-package.yaml")
	manifest.Package = "policy-set"
	manifest.Spec.Areas = []core.Area{{Name: "content", Path: "content"}}
	local := fixtureResource("Workflow", "local", "build", "docs/local/build.yaml")
	local.Spec.Files = []string{"docs/local/evidence.md"}
	imported := fixtureResource("Rule", "content", "policy", "content/rules/policy.yaml")
	imported.Package = "policy-set"
	imported.Spec.Files = []string{"content/data.md", "content/evidence.md", "../escape.md"}
	g := &core.Graph{
		Project:  project,
		Packages: map[string]*core.Resource{"policy-set": manifest},
		Resources: map[string]*core.Resource{
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
	resolved, diagnostics := ResolveWithPackages(g, consumerFiles, archiveFiles)
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
