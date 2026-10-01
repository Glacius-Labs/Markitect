package format

import (
	"strings"
	"testing"
)

const documentationProject = `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: sample}
spec:
  documentation:
    roots: [docs, reference]
`

func TestDocumentationRootsParseAndRejectUnsafeConfiguration(t *testing.T) {
	project, err := Parse("markitect.yaml", []byte(documentationProject))
	if err != nil || project.Spec.Documentation == nil || len(project.Spec.Documentation.Roots) != 2 {
		t.Fatalf("valid documentation roots: %#v, %v", project, err)
	}
	for _, roots := range []string{
		"[]", "[docs, docs]", "[docs, docs/sub]", "[Docs, docs]",
		"[../docs]", "[/docs]", "[docs/../other]", "[docs\\sub]",
	} {
		t.Run(roots, func(t *testing.T) {
			input := strings.Replace(documentationProject, "[docs, reference]", roots, 1)
			if _, err := Parse("markitect.yaml", []byte(input)); err == nil {
				t.Fatalf("accepted invalid roots %s", roots)
			}
		})
	}
	for _, field := range []string{"extra: true", "roots: docs"} {
		t.Run(field, func(t *testing.T) {
			input := strings.Replace(documentationProject, "roots: [docs, reference]", field, 1)
			if _, err := Parse("markitect.yaml", []byte(input)); err == nil {
				t.Fatalf("accepted invalid documentation field %s", field)
			}
		})
	}
}
