package gitrepo

import (
	"bytes"
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
)

func TestReturnedSourceContainersAreIndependentAndConcurrentCallsShareNoState(t *testing.T) {
	_, selection, _ := libraryFixture(t)
	r := adapter(t)
	first, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), first.Raw.Local.Source.Entries[0].Bytes...)
	first.Raw.Local.Source.Entries[0].Bytes[0] ^= 1
	first.Raw.Dependencies[0].Source.Entries[0].Bytes[0] ^= 1
	first.Raw.Execution[0].Location.Boundary = "changed returned container"
	var wg sync.WaitGroup
	wg.Add(2)
	results := make([]apply.CapturedInputs, 2)
	errs := make([]error, 2)
	for i := range results {
		go func() {
			defer wg.Done()
			results[i], errs[i] = r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
		}()
	}
	wg.Wait()
	for i, result := range results {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		if !bytes.Equal(result.Raw.Local.Source.Entries[0].Bytes, original) || result.Guard != first.Guard {
			t.Fatal("output mutation or concurrent acquisition changed fixed basis")
		}
	}
	if !reflect.DeepEqual(results[0], results[1]) {
		t.Fatal("concurrent captures mixed observations")
	}
}
