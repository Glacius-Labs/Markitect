package journal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestLiteralV1SQLiteHistoryKeepsOriginalBytesAndReceipt(t *testing.T) {
	// Producer captured these bytes at 68be968a before changing Bind to v2.
	// Read the same immutable fixtures; never regenerate them with current Bind.
	read := func(name, digest string) []byte {
		data, err := os.ReadFile(filepath.Join("..", "..", "run", "testdata", name))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
			t.Fatal("changed pre-v2 producer fixture", name, err)
		}
		return data
	}
	basisData := read("basis-v1.json", "6e6437921fa0f93e547eb2fdcc33f6c2d528fbe071911a295288294ba3e69af4")
	ownerData := read("owner-v1.txt", "341e646c23cd9f1557a7e9d0c8c8969a8a55c426c427134120daa4544bf8a1ba")
	batchData := read("prepared-v1.json", "b5adc4332fe9916693b6aca071596692a8fc59cec5d17af4fc00816b0cd15496")
	basis, err := run.NewPayload("application/vnd.markitect.run-basis+json", "utf-8", basisData)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := run.NewPayload("text/plain", "utf-8", ownerData)
	if err != nil {
		t.Fatal(err)
	}
	b, err := run.DecodeBatch(batchData)
	if err != nil || !bytes.Equal(b.Bytes(), batchData) {
		t.Fatal(err)
	}
	refs, err := b.BindingRefs()
	if err != nil || !reflect.DeepEqual(refs, []run.BlobRef{basis.Ref()}) {
		t.Fatal(refs, err)
	}
	ownerRef, err := run.BindingOwnerRef(basis)
	if err != nil || ownerRef != owner.Ref() {
		t.Fatal(ownerRef, err)
	}
	refs, err = b.SessionRefs()
	if err != nil || len(refs) != 0 {
		t.Fatal(refs, err)
	}
	r := request(t)
	r.RepositoryIdentity = "repo"
	s := acquire(t, r)
	ctx := context.Background()
	if _, err := s.Put(ctx, basis); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Append(ctx, b); !errors.Is(err, ErrCorrupt) {
		t.Fatal("v1 owner closure waived", err)
	}
	if _, err := s.Put(ctx, owner); err != nil {
		t.Fatal(err)
	}
	expected := run.AppendReceipt{BatchID: "prepare", ContentID: b.ContentID(), PreviousRevision: 0, Revision: 2}
	if receipt, err := s.Append(ctx, b); err != nil || receipt != expected {
		t.Fatal(receipt, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = acquire(t, r)
	for _, payload := range []run.Payload{basis, owner} {
		got, err := s.ReadBlob(ctx, payload.Ref())
		if err != nil || got.Ref() != payload.Ref() || !bytes.Equal(got.Bytes(), payload.Bytes()) {
			t.Fatal("v1 payload rewritten", err)
		}
	}
	page, err := s.Read(ctx, run.EventQuery{Limit: 1})
	if err != nil || page.Revision != 2 || len(page.Batches) != 1 || !bytes.Equal(page.Batches[0].Bytes(), batchData) {
		t.Fatal("v1 batch rewritten", err)
	}
	c, err := run.Restore(ctx, s, "repo", run.RestoreLimits{PageSize: 1, Batches: 8, Events: 16, Bytes: 16 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if c.Current().State != run.Prepared || c.Current().Basis != basis.Ref() || c.Revision() != 2 || c.Budget().Consumed.Runs != 1 {
		t.Fatal("original v1 projection or charge changed")
	}
	inventory, err := run.BindingInventory(basis)
	if err != nil || inventory.Basis() != basis.Ref() {
		t.Fatal(err)
	}
	if models, err := inventory.Models(); !errors.Is(err, run.ErrInventoryUnavailable) || models != nil {
		t.Fatal("v1 acquired invented complete inventory", err)
	}
	p, err := c.Propose(b.Events()[len(b.Events())-1].Body)
	if err != nil {
		t.Fatal(err)
	}
	if receipt, err := c.Commit(ctx, s, p); err != nil || receipt != expected || c.Revision() != 2 || c.Budget().Consumed.Runs != 1 {
		t.Fatal("v1 original retry changed receipt or budget", receipt, err)
	}
	if receipt, err := s.Append(ctx, b); err != nil || receipt != expected {
		t.Fatal("physical v1 identity lookup did not precede CAS", receipt, err)
	}
}

func TestV2SQLiteBasisReconstructsRecordedInventory(t *testing.T) {
	f := newIntegrationFixture(t)
	c := f.open(t)
	batch := integrationCommit(t, c, run.PrepareCommand("prepare", "assignment", "run", "", integrationAt, c.Basis()))
	basisRef := c.Basis().Ref()
	prepared, _ := f.prepared.Prepared()
	original := prepared.Structural().Model().Compilation()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	c = f.open(t)
	if c.Coordinator().Current().Basis != basisRef || c.Coordinator().Revision() != batch.Revision() || c.Coordinator().Budget().Consumed.Runs != 1 {
		t.Fatal("v2 restore lost original projection or charge")
	}
	payload, err := c.Journal().ReadBlob(context.Background(), basisRef)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := run.BindingInventory(payload)
	if err != nil || inventory.Basis() != basisRef {
		t.Fatal(err)
	}
	models, err := inventory.Models()
	if err != nil || len(models) != len(original.Models) {
		t.Fatal("v2 complete inventory unavailable", err)
	}
	for n, scope := range original.Models {
		m := models[n]
		if m.ID != scope.ID || m.Digest != scope.Digest || m.Local != scope.Local || m.Manifest != scope.Manifest || !reflect.DeepEqual(m.Packages, scope.Packages) || len(m.Definitions) != len(scope.Definitions) {
			t.Fatal("v2 recorded scope changed", scope.ID)
		}
		for d, definition := range scope.Definitions {
			if m.Definitions[d] != definition.Identity {
				t.Fatal("v2 recorded identity changed")
			}
		}
	}
	page, err := c.Journal().Read(context.Background(), run.EventQuery{Limit: 1})
	if err != nil || len(page.Batches) != 1 || !bytes.Equal(page.Batches[0].Bytes(), batch.Bytes()) {
		t.Fatal("v2 batch changed", err)
	}
}
