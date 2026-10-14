package guardedwrite

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
)

func TestRecordApplyUsesSharedHostLeaseAndSealedSourceCAS(t *testing.T) {
	root := testRepo(t, "feature/records")
	name := recordstore.Name{Kind: recordstore.Exploration, ID: "work", Role: "record.json"}
	rel, _ := name.Path()
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("basis"), 0644); err != nil {
		t.Fatal(err)
	}
	capture, err := CaptureFiles(root, []string{rel, "source.txt"})
	if err != nil {
		t.Fatal(err)
	}
	writer, err := OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	release, err := writer.lockWriterAt(".markitect", "write.lock")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyRecordChecked(root, capture, name, []byte("first"), nil); err == nil {
		t.Fatal("shared Host lease bypassed")
	}
	release()
	called := false
	result, err := ApplyRecordChecked(root, capture, name, []byte("first"), func() error {
		called = true
		if _, err := os.Stat(filepath.Join(root, ".markitect", "write.lock")); err != nil {
			return err
		}
		return nil
	})
	if err != nil || !called || len(result.CompletedPaths) != 1 || result.CompletedPaths[0] != rel {
		t.Fatalf("guarded result: %+v %v", result, err)
	}
	if _, err := ApplyRecordChecked(root, capture, name, []byte("stale"), nil); err == nil {
		t.Fatal("stale original capture overwrote record")
	}
	wire, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if string(wire) != "first" {
		t.Fatal("stale write changed target")
	}
	update, err := CaptureFiles(root, []string{rel, "source.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyRecordChecked(root, update, name, []byte("new"), nil); err == nil {
		t.Fatal("changed selected source accepted")
	}
	if _, err := ApplyRecordChecked(root, update, recordstore.Name{Kind: recordstore.Brownfield, ID: "work", Role: "record.json"}, nil, nil); err == nil {
		t.Fatal("wrong domain admitted")
	}
}

func TestRecordApplyDomainFailureKeepsAbsentTarget(t *testing.T) {
	root := testRepo(t, "feature/record-guard")
	name := recordstore.Name{Kind: recordstore.Exploration, ID: "work", Role: "record.json"}
	rel, _ := name.Path()
	capture, err := CaptureFiles(root, []string{rel})
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("domain CAS stale")
	result, err := ApplyRecordChecked(root, capture, name, []byte("new"), func() error { return failure })
	if !errors.Is(err, failure) || len(result.CompletedPaths) != 0 {
		t.Fatalf("domain failure: %+v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); !os.IsNotExist(err) {
		t.Fatal("domain failure created target")
	}
}

func TestRecordApplyRetainsReplacedHostLeaseAndRefusesTarget(t *testing.T) {
	root := testRepo(t, "feature/record-lease-replacement")
	name := recordstore.Name{Kind: recordstore.Exploration, ID: "work", Role: "record.json"}
	rel, _ := name.Path()
	capture, err := CaptureFiles(root, []string{rel})
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, ".markitect", "write.lock")
	result, err := ApplyRecordChecked(root, capture, name, []byte("target"), func() error {
		if err := os.Rename(lock, lock+"-original"); err != nil {
			return err
		}
		return os.WriteFile(lock, []byte("replacement"), 0600)
	})
	if err == nil || len(result.CompletedPaths) != 0 {
		t.Fatalf("replaced lease accepted: %+v %v", result, err)
	}
	wire, err := os.ReadFile(lock)
	if err != nil || string(wire) != "replacement" {
		t.Fatal("replaced Host lease removed")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); !os.IsNotExist(err) {
		t.Fatal("lease drift published target")
	}
}
