package app

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/format"
)

func projectWithDeclaredFiles(t *testing.T, files []string, extra map[string][]byte) *Project {
	t.Helper()
	snapshot := fixtureFiles(t, "", "Policy text.", projectNS)
	skill, err := format.Parse(skillPath, snapshot.Files[skillPath])
	if err != nil {
		t.Fatal(err)
	}
	skill.Spec.Files = append([]string(nil), files...)
	snapshot.Files[skillPath] = encodeResource(t, *skill)
	for name, data := range extra {
		snapshot.Files[name] = data
		snapshot.Modes[name] = "100644"
	}
	project, err := Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func hasDiagnostic(project *Project, code, path string) bool {
	for _, diagnostic := range project.Diagnostics {
		if diagnostic.Code == code && (path == "" || diagnostic.Path == path) {
			return true
		}
	}
	return false
}

func TestParseAllowsExplicitOrdinaryYAMLInputs(t *testing.T) {
	files := map[string][]byte{
		"docs/general/config.yaml": []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: local-settings\ndata:\n  mode: review\n"),
		"docs/general/cluster.yml": []byte("---\napiVersion: v1\nkind: Namespace\nmetadata:\n  name: example\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: options\ndata:\n  enabled: 'true'\n"),
	}
	project := projectWithDeclaredFiles(t, []string{"docs/general/config.yaml", "docs/general/cluster.yml"}, files)
	if len(project.Diagnostics) != 0 {
		t.Fatalf("ordinary declared YAML produced diagnostics: %#v", project.Diagnostics)
	}
	got := project.InputFiles[projectNS+"/Skill/entry"]
	if len(got) != 2 || got[0] != "docs/general/config.yaml" || got[1] != "docs/general/cluster.yml" {
		t.Fatalf("resolved inputs = %v, want both declared YAML paths in declaration order", got)
	}
}

func TestParseKeepsRecognizableMarkitectParseErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "unknown Markitect version",
			data: "apiVersion: markitect.example.org/v99\nkind: Text\nmetadata: {name: policy, namespace: cockpit-general}\nspec: {text: body}\n",
		},
		{
			name: "wrong kind in Markitect envelope",
			data: "apiVersion: markitect.example.org/v1alpha1\nkind: Unknown\nmetadata: {name: policy, namespace: cockpit-general}\nspec: {text: body}\n",
		},
		{
			name: "missing API version",
			data: "kind: Text\nmetadata: {name: policy, namespace: cockpit-general}\nspec: {text: body}\n",
		},
		{
			name: "duplicate Markitect discriminator",
			data: "apiVersion: markitect.example.org/v1alpha1\napiVersion: v1\nkind: Text\nmetadata: {name: policy, namespace: cockpit-general}\nspec: {text: body}\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := "docs/general/input.yaml"
			project := projectWithDeclaredFiles(t, []string{path}, map[string][]byte{path: []byte(test.data)})
			if !hasDiagnostic(project, "parse", path) {
				t.Fatalf("parse diagnostic for recognizable malformed Markitect input missing: %#v", project.Diagnostics)
			}
		})
	}
}

func TestParseRejectsTypedResourceListedAsOrdinaryInput(t *testing.T) {
	project := projectWithDeclaredFiles(t, []string{rulePath}, nil)
	if !hasDiagnostic(project, "input.typed-source", skillPath) {
		t.Fatalf("typed source file should be referenced through the graph: %#v", project.Diagnostics)
	}
	if hasDiagnostic(project, "parse", rulePath) {
		t.Fatalf("valid typed resource was hidden from the graph: %#v", project.Diagnostics)
	}
}

func TestParseKeepsUnclaimedMalformedYAMLDiagnostic(t *testing.T) {
	path := "docs/general/unclaimed.yaml"
	project := projectWithDeclaredFiles(t, nil, map[string][]byte{path: []byte("broken: [\n")})
	if !hasDiagnostic(project, "parse", path) {
		t.Fatalf("unclaimed malformed YAML should remain a parse error: %#v", project.Diagnostics)
	}
}

func TestParseKeepsMissingAndOutOfAreaYAMLInputErrors(t *testing.T) {
	tests := []struct {
		name string
		file string
		code string
	}{
		{name: "missing", file: "docs/general/missing.yaml", code: "input.missing"},
		{name: "outside area", file: "docs/other/config.yaml", code: "input.area"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project := projectWithDeclaredFiles(t, []string{test.file}, nil)
			if !hasDiagnostic(project, test.code, skillPath) {
				t.Fatalf("diagnostic %s for %s missing: %#v", test.code, test.file, project.Diagnostics)
			}
		})
	}
}

func TestMarkitectEnvelopeDoesNotClaimOrdinaryKubernetesShape(t *testing.T) {
	data := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata: {name: app}\nspec: {replicas: 2}\n")
	if markitectResourceEnvelope(data) {
		t.Fatal("ordinary Kubernetes API version was classified as a Markitect envelope")
	}
}
