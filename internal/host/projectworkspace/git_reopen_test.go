package projectworkspace

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGitServiceReopenPreservesCompletedBinaryDeleteAndParentBaseline(t *testing.T) {
	fixture := newGitFixture(t)
	storage := filepath.Join(t.TempDir(), "storage")
	original, r := newGitServiceRequest(t, fixture, storage, "recover", []string{"src", "assets"}, nil)
	overlay := []Change{{Kind: ChangeModify, Path: "docs/guide.md", Mode: "100644", Content: []byte("parent context\n")}}
	digest, err := CandidateOverlayDigest(overlay)
	if err != nil {
		t.Fatal(err)
	}
	h, err := original.PrepareCandidate(context.Background(), r, overlay, digest)
	if err != nil {
		t.Fatal(err)
	}
	binary := []byte{0, 255, 42, 128}
	writeFixtureFile(t, h.CWD, "assets/completed.bin", binary, 0644)
	if err := os.Remove(filepath.Join(h.CWD, "src/app.go")); err != nil {
		t.Fatal(err)
	}
	expected, err := original.Harvest(context.Background(), h)
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := NewGitService(storage, gitServiceTestLimits)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.ReopenCandidate(context.Background(), r, h, overlay, digest, true); err != nil {
		t.Fatal(err)
	}
	actual, err := restarted.Harvest(context.Background(), h)
	if err != nil {
		t.Fatal(err)
	}
	if expected.Digest != actual.Digest {
		t.Fatalf("completed output/baseline changed on reopen: %+v vs %+v", expected, actual)
	}
	content, err := os.ReadFile(filepath.Join(h.CWD, "assets/completed.bin"))
	if err != nil || !bytes.Equal(content, binary) {
		t.Fatalf("output overwritten: %v %v", content, err)
	}
	if err := restarted.Close(context.Background(), h); err != nil {
		t.Fatal(err)
	}
}

func TestGitServiceReopenRejectsUnknownTerminalStaleAndForgedBindings(t *testing.T) {
	for _, scenario := range []string{"active-or-unknown", "scope", "handle", "overlay", "source", "marker", "missing-marker", "git-identity"} {
		t.Run(scenario, func(t *testing.T) {
			fixture := newGitFixture(t)
			storage := filepath.Join(t.TempDir(), "storage")
			original, r := newGitServiceRequest(t, fixture, storage, "recover-negative", []string{"src"}, nil)
			overlay := []Change{{Kind: ChangeModify, Path: "docs/guide.md", Mode: "100644", Content: []byte("parent\n")}}
			digest, err := CandidateOverlayDigest(overlay)
			if err != nil {
				t.Fatal(err)
			}
			h, err := original.PrepareCandidate(context.Background(), r, overlay, digest)
			if err != nil {
				t.Fatal(err)
			}
			originalHandle := h
			defer original.Close(context.Background(), originalHandle)
			writeFixtureFile(t, h.CWD, "src/completed.go", []byte("package app\n"), 0644)
			terminal := true
			switch scenario {
			case "active-or-unknown":
				terminal = false
			case "scope":
				r.AllowedPaths = []string{"src", "docs"}
			case "handle":
				h.BaseDigest = r.OverlayDigest
			case "overlay":
				overlay[0].Content = []byte("different")
			case "source":
				writeFixtureFile(t, fixture.root, "src/app.go", []byte("changed source\n"), 0644)
			case "marker":
				if err := os.WriteFile(filepath.Join(filepath.Dir(h.CWD), ownershipRecordName), []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing-marker":
				if err := os.Remove(filepath.Join(filepath.Dir(h.CWD), ownershipRecordName)); err != nil {
					t.Fatal(err)
				}
			case "git-identity":
				testGit(t, h.CWD, "config", "core.worktree", fixture.root)
			}
			restarted, err := NewGitService(storage, gitServiceTestLimits)
			if err != nil {
				t.Fatal(err)
			}
			if err := restarted.ReopenCandidate(context.Background(), r, h, overlay, digest, terminal); err == nil {
				t.Fatalf("unsafe reopen accepted: %s", scenario)
			}
			if _, err := restarted.Harvest(context.Background(), originalHandle); err == nil {
				t.Fatal("rejected reopen registered state")
			}
			if err := restarted.Close(context.Background(), originalHandle); err == nil {
				t.Fatal("rejected reopen authorized cleanup")
			}
			if _, err := os.Stat(filepath.Join(originalHandle.CWD, "src/completed.go")); err != nil {
				t.Fatalf("rejected reopen destroyed preserved work: %v", err)
			}
		})
	}
}

func TestGitServiceReopenCannotAdoptForeignCandidateInStorage(t *testing.T) {
	fixture := newGitFixture(t)
	storage := filepath.Join(t.TempDir(), "storage")
	original, r := newGitServiceRequest(t, fixture, storage, "foreign", []string{"src"}, nil)
	h, err := original.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer original.Close(context.Background(), h)
	foreign := filepath.Join(storage, "workspace-"+h.ID+"-foreign", "repo")
	if err := os.MkdirAll(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	testGit(t, storage, "clone", "--quiet", fixture.root, foreign)
	copied, err := os.ReadFile(filepath.Join(filepath.Dir(h.CWD), ownershipRecordName))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(foreign), ownershipRecordName), copied, 0600); err != nil {
		t.Fatal(err)
	}
	forged := h
	forged.CWD = foreign
	restarted, err := NewGitService(storage, gitServiceTestLimits)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.ReopenCandidate(context.Background(), r, forged, nil, "", true); err == nil {
		t.Fatal("copied marker authorized foreign directory")
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatalf("failed adoption destroyed foreign directory: %v", err)
	}
}
