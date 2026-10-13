package projectapp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

func TestRecordTransportClosedRolesAndReadOnlyCutover(t *testing.T) {
	id := "0123456789abcdef0123456789abcdef"
	for _, rel := range []string{".markitect/records/run/" + id + "/plan.json", ".markitect/records/run/" + id + "/external/responses/" + id + ".json", ".markitect/records/brownfield/session/record.json", ".markitect/runs/" + id + "/external-response-" + id + ".json"} {
		if got, err := RecordPath(rel); err != nil || got != rel {
			t.Fatalf("admitted role: %s %v", rel, err)
		}
	}
	for _, rel := range []string{".markitect/records/briefing/accepted/record.json", ".markitect/records/exploration/work/record.json", ".markitect/records/run/private/log.json", ".markitect/records/run/" + id + "/.pending-test.json", ".markitect/records/brownfield/session/session.lock"} {
		if _, err := RecordPath(rel); err == nil {
			t.Fatalf("unbounded transport role: %s", rel)
		}
	}
	root := t.TempDir()
	if runs, err := projectrun.ListRuns(root); err != nil || len(runs) != 0 {
		t.Fatalf("empty list: %v %v", runs, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect")); !os.IsNotExist(err) {
		t.Fatal("read created metadata")
	}
	old := filepath.Join(root, ".markitect", "runs", id, "plan.json")
	if err := os.MkdirAll(filepath.Dir(old), 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := []byte("preserved old authority")
	if err := os.WriteFile(old, sentinel, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := projectrun.ListRuns(root); err == nil {
		t.Fatal("old authority hidden by empty new list")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(recordstore.RootPath))); !os.IsNotExist(err) {
		t.Fatal("cutover refusal created records")
	}
	wire, _ := os.ReadFile(old)
	if !bytes.Equal(wire, sentinel) {
		t.Fatal("old authority changed")
	}
}
