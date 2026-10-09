package projectwork

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestDocumentRendersDecisionAndIdentityChangeWithEscapedProvenance(t *testing.T) {
	root, _ := testProject(t)
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	project.Report.Decisions = []projectmodel.Decision{{
		ID:         `["project.markitect.example.org/v1alpha1","Decision","orders","cancel"]`,
		Name:       "Cancel decision",
		Subject:    `["project.markitect.example.org/v1alpha1","Statement","orders","cancel"]`,
		Actor:      `["project.markitect.example.org/v1alpha1","Manager","orders","orders"]`,
		Decision:   "Keep *cancel* [orders] <active>.",
		Reason:     "Owner's rationale | one\ntwo",
		Supersedes: `["project.markitect.example.org/v1alpha1","Decision","orders","cancel-old"]`,
		Public:     true,
		Source:     core.Source{Path: ".markitect/decisions [draft].yaml", Line: 19, Digest: "sha256:decision"},
	}}
	project.Report.IdentityChanges = []projectmodel.IdentityChange{{
		ID:        `["project.markitect.example.org/v1alpha1","IdentityChange","orders","cancel-rename"]`,
		Name:      "Rename cancellation",
		Operation: "renamed",
		Previous:  core.DefinitionIdentity{APIVersion: "project.markitect.example.org/v1alpha1", Kind: "Statement", Namespace: "orders", Name: "cancel-v1"},
		Subject:   `["project.markitect.example.org/v1alpha1","Statement","orders","cancel"]`,
		Actor:     `["project.markitect.example.org/v1alpha1","Manager","orders","orders"]`,
		Reason:    "A declared _rename_ <only>.",
		Source:    core.Source{Path: ".markitect/decisions [draft].yaml", Line: 27, Digest: "sha256:identity"},
	}}

	document, err := Document(project, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(document, "- Namespace: (root)\n") {
		t.Fatal("generated document did not label the root namespace")
	}
	for lineNumber, line := range strings.Split(document, "\n") {
		if strings.TrimRight(line, " \t") != line {
			t.Errorf("generated document line %d has trailing whitespace: %q", lineNumber+1, line)
		}
	}
	for _, want := range []string{
		"## Decisions", "## Identity changes", "Subject Statement:", "Recorded Manager actor:",
		`Keep \*cancel\* \[orders\] &lt;active&gt;.`, `Owner's rationale \| one two`,
		`Explicitly supersedes Decision:`, "- Declared operation: renamed", "name `cancel-v1`",
		`Reason: A declared \_rename\_ &lt;only&gt;.`, "../decisions%20%5Bdraft%5D.yaml",
		"line 19", "line 27", "sha256:decision", "sha256:identity",
		"not an authenticated person", "does not by itself prove that the previous definition existed",
	} {
		if !strings.Contains(document, want) {
			t.Errorf("generated document omitted %q:\n%s", want, document)
		}
	}
}
