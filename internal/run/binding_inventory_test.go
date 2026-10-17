package run_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Captured by the unmodified v1 Bind/Propose at
// 68be968a3dae63254060ce19deedfdf90591c3c6, before any production edit.
// Never regenerate these literals with the current Bind.
func legacyBinding(t *testing.T) (run.Payload, run.Payload, run.AppendBatch) {
	t.Helper()
	read := func(name, digest string) []byte {
		data, err := os.ReadFile("testdata/" + name)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
			t.Fatal("changed pre-v2 fixture", name, err)
		}
		return data
	}
	basis, err := run.NewPayload("application/vnd.markitect.run-basis+json", "utf-8", read("basis-v1.json", "6e6437921fa0f93e547eb2fdcc33f6c2d528fbe071911a295288294ba3e69af4"))
	if err != nil {
		t.Fatal(err)
	}
	owner, err := run.NewPayload("text/plain", "utf-8", read("owner-v1.txt", "341e646c23cd9f1557a7e9d0c8c8969a8a55c426c427134120daa4544bf8a1ba"))
	if err != nil {
		t.Fatal(err)
	}
	data := read("prepared-v1.json", "b5adc4332fe9916693b6aca071596692a8fc59cec5d17af4fc00816b0cd15496")
	batch, err := run.DecodeBatch(data)
	if err != nil || !bytes.Equal(batch.Bytes(), data) {
		t.Fatal("v1 batch changed", err)
	}
	return basis, owner, batch
}

func TestLiteralV1HistoryKeepsBytesReferencesAndExactRetry(t *testing.T) {
	basis, owner, batch := legacyBinding(t)
	refs, err := batch.BindingRefs()
	if err != nil || !reflect.DeepEqual(refs, []run.BlobRef{basis.Ref()}) {
		t.Fatal(refs, err)
	}
	refs, err = batch.SessionRefs()
	if err != nil || len(refs) != 0 {
		t.Fatal(refs, err)
	}
	ref, err := run.BindingOwnerRef(basis)
	if err != nil || ref != owner.Ref() {
		t.Fatal(ref, err)
	}
	inventory, err := run.BindingInventory(basis)
	if err != nil || inventory.Basis() != basis.Ref() {
		t.Fatal(err)
	}
	if models, err := inventory.Models(); !errors.Is(err, run.ErrInventoryUnavailable) || models != nil {
		t.Fatal("v1 promoted to complete inventory", models, err)
	}
	if _, err := (run.RecordedInventory{}).Models(); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("zero inventory accepted", err)
	}
	store := journal()
	store.blobs[basis.Ref()], store.blobs[owner.Ref()] = basis, owner
	store.batches, store.head = []run.AppendBatch{batch}, batch.Revision()
	reads := &readJournal{store: store}
	c := restored(t, reads)
	expected := run.AppendReceipt{BatchID: "prepare", ContentID: batch.ContentID(), PreviousRevision: 0, Revision: 2}
	if c.Current().Basis != basis.Ref() || c.Current().State != run.Prepared || c.Budget().Consumed.Runs != 1 || run.ReceiptFor(batch) != expected {
		t.Fatal("v1 projection/receipt changed")
	}
	command := batch.Events()[len(batch.Events())-1].Body
	p, err := c.Propose(command)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Commit(context.Background(), reads, p)
	if err != nil || got != expected || reads.writes != 0 || c.Revision() != 2 {
		t.Fatal("v1 retry rewrote or changed receipt", got, err)
	}
	if !bytes.Equal(store.blobs[basis.Ref()].Bytes(), basis.Bytes()) || !bytes.Equal(store.batches[0].Bytes(), batch.Bytes()) {
		t.Fatal("v1 rewritten")
	}
}

func TestV2RecordedInventorySurvivesRestartWithOwnedViews(t *testing.T) {
	i := fixture(t)
	basis, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := run.NewCoordinator("repo")
	store := journal()
	commit(t, c, store, run.PrepareCommand("prepare", "a", "r", "", observed, basis))
	reads := &readJournal{store: store}
	reopened := restored(t, reads)
	sameProjection(t, c, reopened)
	inventory, err := run.BindingInventory(store.blobs[reopened.Current().Basis])
	if err != nil || inventory.Basis() != basis.Ref() {
		t.Fatal(err)
	}
	models, err := inventory.Models()
	if err != nil || len(models) != len(i.Compilation.Models) {
		t.Fatal(models, err)
	}
	for n, scope := range i.Compilation.Models {
		m := models[n]
		if m.ID != scope.ID || m.Digest != scope.Digest || m.Local != scope.Local || m.Manifest != scope.Manifest || !reflect.DeepEqual(m.Packages, scope.Packages) || len(m.Definitions) != len(scope.Definitions) {
			t.Fatal("lost recorded model fields", m.ID)
		}
		for d, definition := range scope.Definitions {
			if m.Definitions[d] != definition.Identity {
				t.Fatal("lost local/imported identity")
			}
		}
		if scope.ID == "extra" && (m.Definitions == nil || len(m.Definitions) != 0) {
			t.Fatal("complete empty inventory became absent")
		}
	}
	before, _ := inventory.Models()
	for n := range models {
		models[n].ID = "changed"
		if len(models[n].Packages) != 0 {
			models[n].Packages[0].Name = "changed"
		}
		if len(models[n].Definitions) != 0 {
			models[n].Definitions[0].ID = "changed"
		}
	}
	models = append(models, run.RecordedModelInventory{ID: "appended"})
	after, _ := inventory.Models()
	if !reflect.DeepEqual(before, after) || reads.writes != 0 {
		t.Fatal("inventory aliases external view or restore wrote")
	}
	// The original model Result is also no longer an owner of recorded bytes.
	i.Compilation.Models[0].Definitions[0].Identity.ID = "changed"
	again, err := run.BindingInventory(basis.Payload())
	unchanged, _ := again.Models()
	if err != nil || !reflect.DeepEqual(before, unchanged) {
		t.Fatal("inventory aliases compiler result")
	}
}

type recordedDefinition struct {
	Identity model.Identity
	ModelID  string
}

func inventoryRows(t *testing.T, w *closureRecord, index int) []recordedDefinition {
	t.Helper()
	var rows []recordedDefinition
	if w.Compilation[index].Definitions == nil {
		t.Fatal("missing v2 fixture inventory")
	}
	if err := json.Unmarshal(*w.Compilation[index].Definitions, &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func setInventoryRows(t *testing.T, w *closureRecord, index int, rows []recordedDefinition) {
	t.Helper()
	data, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(data)
	w.Compilation[index].Definitions = &raw
}

func rejectInventoryRecord(t *testing.T, basis, owner run.Payload, batch run.AppendBatch, change func(*closureRecord)) {
	t.Helper()
	var w closureRecord
	if err := json.Unmarshal(basis.Bytes(), &w); err != nil {
		t.Fatal(err)
	}
	change(&w)
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := run.NewPayload(basis.Ref().MediaType, basis.Ref().Encoding, data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.BindingOwnerRef(payload); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("invalid record accepted by owner helper", err)
	}
	if _, err := run.BindingInventory(payload); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("invalid record exposed inventory", err)
	}
	store := journal()
	store.blobs[payload.Ref()], store.blobs[owner.Ref()] = payload, owner
	batch = rewrittenPreparation(t, batch, payload.Ref())
	store.batches, store.head = []run.AppendBatch{batch}, batch.Revision()
	reads := &readJournal{store: store}
	if c, err := run.Restore(context.Background(), reads, "repo", restoreLimits); !errors.Is(err, run.ErrInvalid) || c != nil || reads.writes != 0 {
		t.Fatal("invalid record restored", err)
	}
}

func TestV3RejectsChangedSettingSourceDocumentWithoutRecompiling(t *testing.T) {
	i := fixture(t)
	basis, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := run.NewCoordinator("repo")
	p, err := c.Propose(run.PrepareCommand("prepare", "a", "r", "", observed, basis))
	if err != nil {
		t.Fatal(err)
	}
	type document struct {
		Source               organization.SettingSource
		Path, RepositoryPath string
		Mode                 snapshot.Mode
		Bytes                []byte
	}
	for _, probe := range []string{"bytes", "partial source"} {
		t.Run(probe, func(t *testing.T) {
			rejectInventoryRecord(t, basis.Payload(), i.OwnerRequest, p.Batch(), func(w *closureRecord) {
				var documents []document
				if err := json.Unmarshal(w.Documents, &documents); err != nil {
					t.Fatal(err)
				}
				found := false
				for n := range documents {
					if documents[n].Source.File == i.MaxContextBytes.Source.File {
						found = true
						if probe == "bytes" {
							documents[n].Bytes = append(documents[n].Bytes, '\n')
						} else {
							documents[n].Source.Field = "/limits"
						}
					}
				}
				if !found {
					t.Fatal("fixture lacks retained context-limit source")
				}
				w.Documents, err = json.Marshal(documents)
				if err != nil {
					t.Fatal(err)
				}
			})
		})
	}
}

func TestV2RejectsInconsistentInventoryInsteadOfNarrowingScope(t *testing.T) {
	i := fixture(t)
	basis, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := run.NewCoordinator("repo")
	p, err := c.Propose(run.PrepareCommand("prepare", "a", "r", "", observed, basis))
	if err != nil {
		t.Fatal(err)
	}
	foreign := -1
	for n, scope := range i.Compilation.Models {
		if !scope.Local && len(scope.Definitions) > 0 {
			foreign = n
		}
	}
	if foreign < 0 {
		t.Fatal("fixture lacks imported Definitions")
	}
	for name, change := range map[string]func(*closureRecord){
		"missing imported inventory": func(w *closureRecord) { w.Compilation[foreign].Definitions = nil },
		"null imported inventory":    func(w *closureRecord) { setInventoryRows(t, w, foreign, nil) },
		"duplicate imported identity": func(w *closureRecord) {
			rows := inventoryRows(t, w, foreign)
			setInventoryRows(t, w, foreign, append(rows, rows[0]))
		},
		"wrong owning model": func(w *closureRecord) {
			rows := inventoryRows(t, w, foreign)
			rows[0].ModelID = "library"
			setInventoryRows(t, w, foreign, rows)
		},
		"out of Package inventory": func(w *closureRecord) {
			rows := inventoryRows(t, w, foreign)
			rows[0].Identity.Package = "not-present"
			setInventoryRows(t, w, foreign, rows)
		},
		"empty identity": func(w *closureRecord) {
			rows := inventoryRows(t, w, foreign)
			rows[0].Identity.ID = ""
			setInventoryRows(t, w, foreign, rows)
		},
		"malformed identity": func(w *closureRecord) {
			rows := inventoryRows(t, w, foreign)
			rows[0].Identity.ID = "../foreign"
			setInventoryRows(t, w, foreign, rows)
		},
		"unordered identity": func(w *closureRecord) {
			rows := inventoryRows(t, w, foreign)
			rows[0], rows[1] = rows[1], rows[0]
			setInventoryRows(t, w, foreign, rows)
		},
		"local responsibility omitted": func(w *closureRecord) { rows := inventoryRows(t, w, 0); setInventoryRows(t, w, 0, rows[:1]) },
		"duplicate model":              func(w *closureRecord) { w.Compilation[foreign].ID = w.Compilation[0].ID },
		"missing parent Package":       func(w *closureRecord) { w.Compilation[0].Packages = w.Compilation[0].Packages[2:] },
		"duplicate Package": func(w *closureRecord) {
			w.Compilation[foreign].Packages = append(w.Compilation[foreign].Packages, w.Compilation[foreign].Packages[0])
		},
		"foreign Package source": func(w *closureRecord) { w.Compilation[foreign].Packages[0].Source.Origin = "elsewhere" },
		"root raw-source mismatch": func(w *closureRecord) {
			w.ModelInput.Root, w.ModelInput.Dependencies[0] = w.ModelInput.Dependencies[0], w.ModelInput.Root
		},
	} {
		t.Run(name, func(t *testing.T) { rejectInventoryRecord(t, basis.Payload(), i.OwnerRequest, p.Batch(), change) })
	}
}

func TestLiteralV1MalformedHistoryStillRefuses(t *testing.T) {
	basis, owner, batch := legacyBinding(t)
	for name, change := range map[string]func(*closureRecord){
		"v1 cannot assert v2 completeness": func(w *closureRecord) { w.Version = 2 },
		"unsupported version":              func(w *closureRecord) { w.Version = 3 },
		"missing source":                   func(w *closureRecord) { w.ModelInput.Root = model.SourceModel{} },
		"wrong local selection":            func(w *closureRecord) { w.Compilation[0].Local = false },
		"bad digest":                       func(w *closureRecord) { w.Compilation[0].Digest = "sha256:wrong" },
		"broken parent inventory":          func(w *closureRecord) { w.Compilation[0].Packages = w.Compilation[0].Packages[2:] },
		"invalid policy":                   func(w *closureRecord) { w.Policy = json.RawMessage(`{}`) },
		"unbound manager": func(w *closureRecord) {
			w.Managers = bytes.Replace(w.Managers, []byte(`"Parent":"lending"`), []byte(`"Parent":"empty"`), 1)
		},
		"v1 cannot smuggle new inventory": func(w *closureRecord) { setInventoryRows(t, w, 0, []recordedDefinition{}) },
	} {
		t.Run(name, func(t *testing.T) { rejectInventoryRecord(t, basis, owner, batch, change) })
	}
}

func TestV2InventoryRetainsExistingPayloadAndRestoreLimits(t *testing.T) {
	b := binding(t)
	if _, err := run.NewPayload(b.Ref().MediaType, b.Ref().Encoding, make([]byte, (64<<20)+1)); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("basis payload limit removed", err)
	}
	c, _ := run.NewCoordinator("repo")
	store := journal()
	commit(t, c, store, run.PrepareCommand("prepare", "a", "r", "", observed, b))
	var total int64
	for _, batch := range store.batches {
		total += int64(len(batch.Bytes()))
	}
	for ref := range store.blobs {
		total += ref.Size
	}
	limits := restoreLimits
	limits.Bytes = total - 1
	if c, err := run.Restore(context.Background(), store, "repo", limits); !errors.Is(err, run.ErrRestoreLimit) || c != nil {
		t.Fatal("inventory escaped read budget", err)
	}
	limits.Bytes = total
	if c, err := run.Restore(context.Background(), store, "repo", limits); err != nil || c == nil {
		t.Fatal("exact inventory read budget refused", err)
	}
}
