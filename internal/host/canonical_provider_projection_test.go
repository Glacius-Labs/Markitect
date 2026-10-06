package host

import (
	"bytes"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalWorkflowProviderProjectionsShareOnlyCanonicalInputs(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	revision := scopedTestGit(t, root, "rev-parse", "HEAD")
	fixed, err := LoadSelectedCanonicalSource(root, revision, "examples/canonical-workflow/canonical.yaml", true)
	if err != nil || len(fixed.Diagnostics) != 0 {
		t.Fatalf("canonical workflow: %v %v", err, fixed.Diagnostics)
	}
	observed := &snapshot.Snapshot{ID: "working", Provisional: true, Files: cloneByteMap(fixed.Snapshot.Files), Modes: fixed.Snapshot.Modes}
	outputs := map[string][]byte{}
	for _, provider := range []string{"codex", "claude", "markdown"} {
		id := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "example", Name: "workflow-" + provider}
		prepared, err := PrepareCanonicalProjection(fixed, observed, id, "provider-test/1", sha256Prefix(sha256Hex([]byte("provider-test"))), nil)
		if err != nil || prepared.Plan == nil || len(prepared.Escalations) != 0 {
			t.Fatalf("%s: %v %v", provider, err, prepared.Escalations)
		}
		if len(prepared.Request.Definitions) != 4 {
			t.Fatal("lost exact shared canonical scope")
		}
		for name, content := range prepared.Outputs {
			if provider == "codex" && filepath.Base(name) != "AGENTS.md" {
				t.Fatal("module did not own Codex filename")
			}
			if provider == "claude" && filepath.Base(name) != "CLAUDE.md" {
				t.Fatal("module did not own Claude filename")
			}
			for _, d := range prepared.Request.Definitions {
				if !bytes.Contains(content, []byte(d.Metadata.Name)) {
					t.Fatalf("%s omitted canonical subject %s", provider, d.Metadata.Name)
				}
			}
			outputs[name] = content
		}
	}
	if len(outputs) != 3 {
		t.Fatalf("expected two provider files and one Module-owned combined Markdown file, got %d", len(outputs))
	}
	// Neither target representation was supplied as input to either provider.
	for name := range observed.Files {
		if strings.HasSuffix(name, "/AGENTS.md") || strings.HasSuffix(name, "/CLAUDE.md") {
			t.Fatal("provider depended on sibling target bytes")
		}
	}
}
