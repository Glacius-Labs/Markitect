package gitrepo

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestActualLibraryCaptureAndBothPreviews(t *testing.T) {
	repo, selection, head := libraryFixture(t)
	repo.Write("ignored/raw.bin", "\x00\xffignored census bytes")
	repo.Write("node_modules/payload.bin", "also part of census")
	repo.Write(".markitect/state/trace", "excluded control observation")
	r := adapter(t)
	service := apply.NewModelService(r, r)
	captured, err := service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	compiled := captured.Compilation()
	inputs := captured.Inputs()
	if compiled.Status != model.Passed || len(compiled.Models) != 3 || compiled.Models[0].ID != "small-library" || inputs.Head != head || inputs.Guard == "" {
		t.Fatalf("actual capture/compilation failed: %+v %+v", inputs, compiled)
	}
	if inputs.Raw.Local.RepositoryArea.String() != "inputs/local" || inputs.Raw.Dependencies[0].RepositoryArea.String() != "inputs/vocabulary" || inputs.Raw.Dependencies[1].RepositoryArea.String() != "inputs/core" {
		t.Fatal("missing explicit model census coordinates")
	}
	for _, p := range []string{"ignored/raw.bin", "node_modules/payload.bin"} {
		if _, found := inputs.Raw.Census.Lookup(fixturePath(t, p)); !found {
			t.Fatalf("census excluded %s", p)
		}
	}
	if _, found := inputs.Raw.Census.Lookup(fixturePath(t, ".markitect/state/trace")); found {
		t.Fatal("state entered census")
	}
	file, found := inputs.Raw.Census.Lookup(fixturePath(t, "real/library.go"))
	if !found || file.Mode() != snapshot.Executable || !bytes.Equal(file.Bytes(), []byte("\xef\xbb\xbfpackage library\r\n")) {
		t.Fatal("original bytes or Git mode changed")
	}
	if !bytes.Equal(inputs.Raw.Execution[0].File.Bytes(), []byte("\xef\xbb\xbforiginal\x00\r\nconfig/execution.data")) {
		t.Fatal("selected source bytes changed")
	}
	working, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{})
	if err != nil || working.Compilation().Status != model.Passed {
		t.Fatalf("working: %v %+v", err, working.Compilation())
	}
	fixed, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: head}, model.Limits{})
	if err != nil || fixed.Compilation().Status != model.Passed {
		t.Fatalf("fixed: %v %+v", err, fixed.Compilation())
	}
	if fixed.Inputs().Revision != head || working.Inputs().Revision != "" {
		t.Fatal("preview revision substitution")
	}
	if _, found := fixed.Inputs().Raw.Census.Lookup(fixturePath(t, "ignored/raw.bin")); found {
		t.Fatal("historical capture included today's ignored file")
	}
	if !reflect.DeepEqual(inputs.Raw.Local.Source, working.Inputs().Raw.Local.Source) || !reflect.DeepEqual(inputs.Raw.Local.Source, fixed.Inputs().Raw.Local.Source) {
		t.Fatal("shared acquisition changed original model entries")
	}
	before := inputs.Raw.Local.Source.Entries[0].Bytes[0]
	inputs.Raw.Local.Source.Entries[0].Bytes[0] ^= 1
	if captured.Inputs().Raw.Local.Source.Entries[0].Bytes[0] != before {
		t.Fatal("application's acquired observation aliases returned bytes")
	}
}
