package projectworkspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGitServiceFileDirectoryReplacementHasTrueEndInventory(t *testing.T) {
	for _, overlayBasis := range []bool{false, true} {
		t.Run(map[bool]string{false: "task", true: "parent-overlay"}[overlayBasis], func(t *testing.T) {
			fixture := newGitFixture(t)
			service, r := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "replace", []string{"src"}, nil)
			changes := []Change{{Kind: ChangeDelete, Path: "src/app.go"}, {Kind: ChangeAdd, Path: "src/app.go/child.go", Mode: "100644", Content: []byte("package child\n")}}
			var h Handle
			var err error
			if overlayBasis {
				digest, e := CandidateOverlayDigest(changes)
				if e != nil {
					t.Fatal(e)
				}
				h, err = service.PrepareCandidate(context.Background(), r, changes, digest)
			} else {
				h, err = service.Prepare(context.Background(), r)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer service.Close(context.Background(), h)
			if !overlayBasis {
				if err := os.Remove(filepath.Join(h.CWD, "src/app.go")); err != nil {
					t.Fatal(err)
				}
				writeFixtureFile(t, h.CWD, "src/app.go/child.go", []byte("package child\n"), 0644)
			}
			d, err := service.Harvest(context.Background(), h)
			if err != nil {
				t.Fatal(err)
			}
			expected := 2
			if overlayBasis {
				expected = 0
			}
			if len(d.Changes) != expected {
				t.Fatalf("bad observed replacement delta: %+v", d)
			}
			if _, err := os.Stat(filepath.Join(h.CWD, "src/app.go/child.go")); err != nil {
				t.Fatalf("not true final inventory: %v", err)
			}
		})
	}
}

func TestGitServiceRetryAfterCanceledCloseAndUnknownHandlePreservesEvidence(t *testing.T) {
	fixture := newGitFixture(t)
	storage := filepath.Join(t.TempDir(), "storage")
	service, r := newGitServiceRequest(t, fixture, storage, "close-retry", []string{"src"}, nil)
	h, err := service.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.Close(canceled, h); err == nil {
		t.Fatal("canceled close succeeded")
	}
	writeFixtureFile(t, h.CWD, "src/completed.go", []byte("package app\n"), 0644)
	recovered, err := NewGitService(storage, gitServiceTestLimits)
	if err != nil {
		t.Fatal(err)
	}
	if err := recovered.Close(context.Background(), h); err == nil {
		t.Fatal("new process assumed ownership of unknown evidence")
	}
	if _, err := os.Stat(filepath.Join(h.CWD, "src/completed.go")); err != nil {
		t.Fatalf("rejected close lost evidence: %v", err)
	}
	if _, err := service.Harvest(context.Background(), h); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(context.Background(), h); err != nil {
		t.Fatal(err)
	}
}
