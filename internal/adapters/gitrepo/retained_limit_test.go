package gitrepo

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

func TestRetainedInputSerializedBoundaryIsReadableOrRefusedBeforePublication(t *testing.T) {
	_, previous, g, _ := runFixture(t)
	external := t.TempDir()
	data := []byte(strings.Repeat("captured guidance\n", 4096))
	if err := os.WriteFile(filepath.Join(external, "instruction.md"), data, 0600); err != nil {
		t.Fatal(err)
	}
	selection := previous.Selection()
	selection.Instructions = []project.SourceLocation{{Boundary: external, Path: fixturePath(t, "instruction.md")}}
	repo := adapter(t)
	captured, err := project.NewPreparationService(project.NewModelService(repo, repo)).Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
	a, ok := captured.Prepared()
	if err != nil || !ok {
		t.Fatal(captured.Status(), err)
	}
	p, err := project.Prepare(a)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(retainedFrom(p))
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) <= len(data) {
		t.Fatal("fixture does not exercise serialized expansion")
	}
	g.MaxInputBytes = int64(len(encoded))
	ref, err := g.Capture(context.Background(), "boundary", p)
	if err != nil {
		t.Fatal("exact serialized boundary refused", err)
	}
	if _, err = g.retained(context.Background(), ref, p); err != nil {
		t.Fatal("published inputs could not be read", err)
	}
	g.MaxInputBytes--
	ref, err = g.Capture(context.Background(), "boundary", p)
	if !errors.Is(err, ErrLimit) {
		t.Fatal("oversized serialized input was published", err)
	}
	if path, pathErr := g.candidatePath(ref); pathErr == nil {
		if _, err = os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("over-limit reference became visible", err)
		}
	}
}
