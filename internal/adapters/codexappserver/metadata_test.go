package codexappserver

import (
	"context"
	"crypto/sha256"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestM1SourceProfileReadsActualPrivateFilesAndCompleteInventories(t *testing.T) {
	f := fixture(t)
	input := profileInput(t, f.inline(), nil)
	out, err := prepareMetadata(context.Background(), input, f.pin)
	if err != nil || out.Status != model.Passed {
		t.Fatalf("static metadata: %v %+v", err, out)
	}
	if out.InputBinding != input.Binding() || out.Contract != (apply.ProfileContract{Name: "markitect.codex-app-server.offline-metadata", Version: 1}) || len(out.Files) != 1 || len(out.Inventories) != 2 {
		t.Fatalf("binding/shape: %+v", out)
	}
	if out.Files[0].ContentID != snapshot.ContentID(sha256.Sum256([]byte("fixture executable\r\n"))) || out.Files[0].Size != 20 {
		t.Fatal("actual executable bytes missing")
	}
	for _, inventory := range out.Inventories {
		if len(inventory.Files) != 2 || inventory.Files[1].Location.Path.String() != "unused.json" || inventory.Files[1].Location.Boundary != inventory.Boundary {
			t.Fatal("complete inventory/actual boundary lost")
		}
	}
	if historicalPin().executable == f.pin.executable {
		t.Fatal("fixture escaped into production pin")
	}
	production, err := NewMetadataPreparer().PrepareAgentProfile(context.Background(), input)
	if production.Status != model.Failed {
		t.Fatalf("production accepted fixture pin: %v %+v", err, production)
	}
}
