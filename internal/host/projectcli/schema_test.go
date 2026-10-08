package projectcli

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestProjectSchemaNeedsNoRepositoryAndWritesNoFiles(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	isolated := t.TempDir()
	if err := os.Chdir(isolated); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	var out, errout bytes.Buffer
	if code := Run([]string{"schema"}, &out, &errout); code != 0 {
		t.Fatalf("project schema exit=%d stderr=%s", code, errout.String())
	}
	var got any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
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
		t.Fatal("project schema output differs from the active project-model schema")
	}
	entries, err := os.ReadDir(isolated)
	if err != nil || len(entries) != 0 {
		t.Fatalf("read-only schema action created files: entries=%v err=%v", entries, err)
	}
	if _, _, err := parse([]string{"schema", "--repo", "."}, nil); err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("schema unexpectedly accepted a repository flag: %v", err)
	}
	var help bytes.Buffer
	if code := Run([]string{"--help"}, &help, new(bytes.Buffer)); code != 0 || !strings.Contains(help.String(), "schema") {
		t.Fatalf("project action help omitted schema: code=%d output=%s", code, help.String())
	}
}
