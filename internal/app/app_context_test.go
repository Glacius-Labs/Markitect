package app

import (
	"reflect"
	"testing"
)

func TestLoadParseAndCompileContext(t *testing.T) {
	root := tempRoot(t)
	writeFixture(t, root, fixtureFiles(t, "", "Keep the owner source.", projectNS).Files)
	p, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Snapshot == nil || !p.Snapshot.Provisional || len(p.Resources) != 3 || len(p.Inventory) != 3 {
		t.Fatalf("unexpected loaded project: snapshot=%#v resources=%d inventory=%d", p.Snapshot, len(p.Resources), len(p.Inventory))
	}

	parsed, err := Parse(fixtureFiles(t, "fixture-commit", "Keep the owner source.", projectNS))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Snapshot.Revision != "fixture-commit" || len(parsed.Diagnostics) != 0 {
		t.Fatalf("unexpected parsed project: %#v", parsed)
	}

	ctx, err := CompileContext(parsed, projectNS+"/Skill/entry", "test-tool")
	if err != nil {
		t.Fatal(err)
	}
	wantKeys := []string{"/Project/sample-project", projectNS + "/Rule/policy", projectNS + "/Skill/entry"}
	gotKeys := make([]string, 0, len(ctx.Inputs))
	for _, input := range ctx.Inputs {
		gotKeys = append(gotKeys, input.Key)
		if input.Hash != Hash(parsed.Snapshot.Files[input.Path]) {
			t.Fatalf("context hash for %s does not match captured bytes", input.Key)
		}
	}
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Fatalf("context inputs = %v, want %v", gotKeys, wantKeys)
	}
}
