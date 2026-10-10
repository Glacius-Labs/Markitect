package projectcli

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestSchemaNeedsNoRepositoryAndWritesNoFiles(t *testing.T) {
	isolated := t.TempDir()
	t.Chdir(isolated)

	code, out, errout := runCLI(t, "schema")
	if code != 0 {
		t.Fatalf("schema exit=%d stderr=%s", code, errout)
	}
	var got any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode schema output: %v", err)
	}
	wantBytes, err := json.Marshal(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	var want any
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("schema output differs from the active project-model schema")
	}
	entries, err := os.ReadDir(isolated)
	if err != nil || len(entries) != 0 {
		t.Fatalf("read-only schema verb created files: entries=%v err=%v", entries, err)
	}
	if code, _, errout := runCLI(t, "schema", "--repo", "."); code != 2 || !strings.Contains(errout, "flag provided but not defined: -repo") {
		t.Fatalf("schema unexpectedly accepted a repository flag: exit=%d stderr=%s", code, errout)
	}
	if _, help, _ := runCLI(t, "--help"); !strings.Contains(help, "  schema ") {
		t.Fatalf("verb list omitted schema: %s", help)
	}

	// schema is also an MCP tool with the same output.
	server, err := newMCPServer(env{root: isolated, ops: projectOperations()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.Call(context.Background(), "schema", json.RawMessage(`{}`))
	if err != nil || result.IsError {
		t.Fatalf("MCP schema: %+v %v", result, err)
	}
	if data := result.StructuredContent.(map[string]any)["data"]; !reflect.DeepEqual(normalizeJSON(t, data), want) {
		t.Fatal("MCP schema output differs from the active project-model schema")
	}
}

func normalizeJSON(t *testing.T, value any) any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
