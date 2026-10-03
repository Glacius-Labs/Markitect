package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestParallelWaveMarkdownProjectionIgnoresHumanDocumentBody(t *testing.T) {
	g, files := proseGraph("Read the [handbook](../tools/README.md) before proceeding.\n")
	const humanDocument = "docs/general/tools/README.md"
	files[humanDocument] = []byte("# Human-owned handbook\nThe canonical guidance lives here.\n")

	want, owners, err := GenerateWithOwners(g, files)
	if err != nil {
		t.Fatal(err)
	}
	const view = "docs/markitect/general/skills/setup.skill.md"
	if got := owners[view]; len(got) != 1 || got[0] != "general/Skill/setup" {
		t.Fatalf("generated view owners = %v, want only its canonical Skill", got)
	}
	if len(g.Edges) != 0 || len(g.Relationships) != 0 {
		t.Fatal("a prose link to a human document created a typed relationship")
	}

	files[humanDocument] = []byte("# Revised human-owned handbook\nPRIVATE BODY THAT MUST NOT BE COPIED\n")
	for i := 0; i < 32; i++ {
		got, gotOwners, err := GenerateWithOwners(g, files)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got[view], want[view]) {
			t.Fatalf("human document bytes or map iteration changed generated Markdown:\n%s", got[view])
		}
		if strings.Contains(string(got[view]), "PRIVATE BODY THAT MUST NOT BE COPIED") {
			t.Fatal("human-owned Markdown body was copied into generated output")
		}
		if _, emitted := got[humanDocument]; emitted {
			t.Fatal("renderer emitted a generated file at the human-owned input path")
		}
		if strings.Join(gotOwners[view], ",") != strings.Join(owners[view], ",") {
			t.Fatalf("generated owner changed across runs: %v", gotOwners[view])
		}
	}
}
