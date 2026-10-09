package projectworkspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitServiceParentCandidateIsReadBaselineWithoutNewWriteAuthority(t *testing.T) {
	fixture := newGitFixture(t)
	service, request := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "child", []string{"src"}, []string{"docs"})
	overlay := []Change{
		{Kind: ChangeModify, Path: "docs/guide.md", Mode: "100644", Content: []byte("parent doc\n")},
		{Kind: ChangeDelete, Path: "assets/original.bin"},
		{Kind: ChangeAdd, Path: "assets/parent.bin", Mode: "100644", Content: []byte{0, 255, 1}},
		{Kind: ChangeAdd, Path: "src/parent.sh", Mode: "100755", Content: []byte("#!/bin/sh\n")},
	}
	digest, err := CandidateOverlayDigest(overlay)
	if err != nil {
		t.Fatal(err)
	}
	h, err := service.PrepareCandidate(context.Background(), request, overlay, digest)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close(context.Background(), h)
	binding, err := InspectRepository(context.Background(), fixture.root, fixture.base)
	if err != nil {
		t.Fatal(err)
	}
	if h.BaseDigest == binding.InventoryDigest {
		t.Fatal("candidate baseline omitted overlay")
	}
	delta, err := service.Harvest(context.Background(), h)
	if err != nil || len(delta.Changes) != 0 {
		t.Fatalf("parent overlay treated as child writes: %+v %v", delta, err)
	}
	writeFixtureFile(t, h.CWD, "src/child.go", []byte("package app\n"), 0644)
	delta, err = service.Harvest(context.Background(), h)
	if err != nil || len(delta.Changes) != 1 || delta.Changes[0].Path != "src/child.go" {
		t.Fatalf("child write wrong: %+v %v", delta, err)
	}
	writeFixtureFile(t, h.CWD, "docs/guide.md", []byte("unauthorized child doc\n"), 0644)
	if _, err := service.Harvest(context.Background(), h); err == nil {
		t.Fatal("parent context granted foreign child writes")
	}
}

func TestGitServiceOverlayExistenceAndDigestChecks(t *testing.T) {
	fixture := newGitFixture(t)
	service, r := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "overlay", []string{"src"}, nil)
	for _, changes := range [][]Change{
		{{Kind: ChangeAdd, Path: "src/app.go", Mode: "100644", Content: []byte("existing")}},
		{{Kind: ChangeModify, Path: "src/missing.go", Mode: "100644", Content: []byte("missing")}},
		{{Kind: ChangeDelete, Path: "src/missing.go"}},
		{{Kind: ChangeRename, OldPath: "src/missing.go", Path: "src/new.go", Mode: "100644", Content: []byte("missing")}},
	} {
		digest, err := CandidateOverlayDigest(changes)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.PrepareCandidate(context.Background(), r, changes, digest); err == nil {
			t.Fatalf("accepted impossible overlay %+v", changes)
		}
	}
	changes := []Change{{Kind: ChangeDelete, Path: "src/app.go"}}
	if _, err := service.PrepareCandidate(context.Background(), r, changes, "sha256:"+strings.Repeat("0", 64)); err == nil {
		t.Fatal("accepted stale overlay digest")
	}
	if _, err := CandidateOverlayDigest([]Change{{Kind: ChangeDelete, Path: ".markitect/model.yaml"}}); err == nil {
		t.Fatal("accepted control overlay")
	}
}

func TestGitServiceCapturesIgnoredWIPAndImmutableScope(t *testing.T) {
	fixture := newGitFixture(t)
	writeFixtureFile(t, fixture.root, ".gitignore", []byte("generated.bin\n"), 0644)
	writeFixtureFile(t, fixture.root, "generated.bin", []byte{255, 0}, 0644)
	service, r := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "immutable", []string{"src"}, nil)
	h, err := service.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close(context.Background(), h)
	data, err := os.ReadFile(filepath.Join(h.CWD, "generated.bin"))
	if err != nil || len(data) != 2 {
		t.Fatalf("ignored WIP lost: %v %v", data, err)
	}
	r.AllowedPaths[0] = "docs"
	writeFixtureFile(t, h.CWD, "docs/guide.md", []byte("outside\n"), 0644)
	if _, err := service.Harvest(context.Background(), h); err == nil {
		t.Fatal("caller mutated retained write scope")
	}
}

func TestGitServiceScopeChecksBothRenamePaths(t *testing.T) {
	fixture := newGitFixture(t)
	service, r := newGitServiceRequest(t, fixture, filepath.Join(t.TempDir(), "storage"), "rename", []string{"src"}, nil)
	h, err := service.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close(context.Background(), h)
	if err := os.Rename(filepath.Join(h.CWD, "docs/guide.md"), filepath.Join(h.CWD, "src/guide.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Harvest(context.Background(), h); err == nil {
		t.Fatal("foreign rename source accepted")
	}
}

func TestGitServiceFailureLeavesNoOwnCandidateAndStorageCannotBeInput(t *testing.T) {
	fixture := newGitFixture(t)
	storage := filepath.Join(fixture.root, "workspaces")
	service, r := newGitServiceRequest(t, fixture, storage, "nested", []string{"src"}, nil)
	if _, err := service.Prepare(context.Background(), r); err == nil {
		t.Fatal("storage admitted within source")
	}
	entries, err := os.ReadDir(storage)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed prepare leaked candidate: %v %v", entries, err)
	}
}
