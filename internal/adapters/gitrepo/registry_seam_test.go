package gitrepo

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const (
	seamRootShard  = ".markitect/registry/registry.yaml"
	seamChildShard = ".markitect/registry/lending/registry.yaml"
	seamRootYAML   = "apiVersion: markitect.registry/v1\r\nartifacts:\r\n  - path: realization/library.md\r\n    form: doc\r\n    realizes: [library/library]\r\n    why: Library usage fixture.\r\n"
	seamChildYAML  = "apiVersion: markitect.registry/v1\r\nartifacts:\r\n  - path: realization/lending.go\r\n    form: code\r\n    realizes: [lending/borrow]\r\n    why: Lending operation fixture.\r\n"
)

func registrySeamFixture(t *testing.T) (*testkit.Repo, project.InputSelection, string) {
	t.Helper()
	// Keep the actual M1, Vocabulary and Core bytes supplied by libraryFixture.
	// These artifact bytes test structural registration, not semantic realization.
	repo, selection, _ := libraryFixture(t)
	repo.Write("realization/library.md", "\xef\xbb\xbfLibrary usage fixture.\r\n")
	repo.Write("realization/lending.go", "package lending\r\n")
	repo.Write(seamRootShard, seamRootYAML)
	repo.Write(seamChildShard, seamChildYAML)
	selection.Registry = []project.SourceLocation{
		{Boundary: filepath.Join(repo.Dir, ".markitect", "registry"), Path: fixturePath(t, "registry.yaml")},
		{Boundary: filepath.Join(repo.Dir, ".markitect", "registry", "lending"), Path: fixturePath(t, "registry.yaml")},
	}
	return repo, selection, repo.Commit("Actual root and lending registry shards with artifact files")
}

func TestCaptureRegistrySeamPreparesActualCommittedRootAndLendingShards(t *testing.T) {
	_, selection, head := registrySeamFixture(t)
	r := adapter(t)
	stage, err := project.NewModelService(r, r).CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || stage.Checked().Status != model.Passed || len(stage.Checked().Models) != 3 {
		t.Fatalf("actual M1 model capture failed: %v %+v", err, stage.Checked())
	}
	prepared := stage.PrepareRegistry()
	registry, available := prepared.Registry()
	if prepared.Status() != model.Passed || !available || len(registry.Entries()) != 2 || len(prepared.Records()) != 2 || len(prepared.Diagnostics()) != 0 {
		t.Fatalf("actual registry preparation failed: %s %+v", prepared.Status(), prepared.Diagnostics())
	}
	inputs := prepared.Model().Inputs()
	if inputs.Head != head || inputs.Guard == "" || inputs.Guard != stage.Inputs().Guard || !inputs.Raw.Census.Equal(stage.Inputs().Raw.Census) {
		t.Fatal("registry preparation changed its opaque captured parent binding")
	}
	if !reflect.DeepEqual(prepared.Discovered(), []snapshot.Path{fixturePath(t, seamChildShard), fixturePath(t, seamRootShard)}) {
		t.Fatal("actual root and child shards were not discovered exactly")
	}
	for i, expected := range []struct{ shard, yaml string }{{seamRootShard, seamRootYAML}, {seamChildShard, seamChildYAML}} {
		source := inputs.Raw.Registry[i]
		census, found := inputs.Raw.Census.Lookup(fixturePath(t, expected.shard))
		if !found || source.RepositoryPath.String() != expected.shard || source.File.Path().String() != "registry.yaml" || !bytes.Equal(source.File.Bytes(), []byte(expected.yaml)) || !bytes.Equal(census.Bytes(), source.File.Bytes()) || census.Mode() != source.File.Mode() {
			t.Fatal("registry bytes, modes or physical shard coordinates changed")
		}
	}
	root, found := registry.Lookup(fixturePath(t, "realization/library.md"))
	if !found || !root.Owner().IsRoot() || root.Artifact().Form() != realization.Doc || !reflect.DeepEqual(root.Artifact().Basis(), []model.Identity{{Package: "library", ID: "library"}}) {
		t.Fatal("root registration lost its actual owner or Definition basis")
	}
	child, found := registry.Lookup(fixturePath(t, "realization/lending.go"))
	owner, packageOwner := child.Owner().Package()
	if !found || !packageOwner || owner != "lending" || child.Artifact().Form() != realization.Code || !reflect.DeepEqual(child.Artifact().Basis(), []model.Identity{{Package: "lending", ID: "borrow"}}) {
		t.Fatal("lending registration lost its actual owner or Definition basis")
	}
	for path, original := range map[string]string{"realization/library.md": "\xef\xbb\xbfLibrary usage fixture.\r\n", "realization/lending.go": "package lending\r\n"} {
		file, found := inputs.Raw.Census.Lookup(fixturePath(t, path))
		if !found || !bytes.Equal(file.Bytes(), []byte(original)) {
			t.Fatal("registered artifact was not bound to its original Census bytes")
		}
	}
}

func TestCaptureRegistrySeamRetainsUnselectedCensusShardAsNotCompleted(t *testing.T) {
	_, selection, head := registrySeamFixture(t)
	selection.Registry = selection.Registry[:1]
	r := adapter(t)
	stage, err := project.NewModelService(r, r).CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || stage.Checked().Status != model.Passed {
		t.Fatalf("actual M1 model capture failed: %v", err)
	}
	prepared := stage.PrepareRegistry()
	registry, available := prepared.Registry()
	if prepared.Status() != model.NotCompleted || available || len(registry.Entries()) != 0 || len(prepared.Discovered()) != 2 || len(prepared.Records()) != 1 {
		t.Fatalf("unselected Census shard became a partial registry: %s %+v", prepared.Status(), prepared.Diagnostics())
	}
	diagnostics := prepared.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Code != "registry.source_uncaptured" || diagnostics[0].Source.String() != seamChildShard {
		t.Fatalf("missing physical unselected-shard diagnostic: %+v", diagnostics)
	}
	inputs := prepared.Model().Inputs()
	child, found := inputs.Raw.Census.Lookup(fixturePath(t, seamChildShard))
	if !found || !bytes.Equal(child.Bytes(), []byte(seamChildYAML)) || len(inputs.Raw.Registry) != 1 || inputs.Head != head || inputs.Guard != stage.Inputs().Guard || !inputs.Raw.Census.Equal(stage.Inputs().Raw.Census) {
		t.Fatal("incomplete registry discarded its actual Census or parent binding")
	}
}

func TestCaptureRegistrySeamIgnoredUntrackedShardRemainsPreviewOnly(t *testing.T) {
	repo, selection, _ := registrySeamFixture(t)
	ignore, err := os.ReadFile(filepath.Join(repo.Dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	repo.Git("rm", "--", seamChildShard)
	repo.Write(".gitignore", string(ignore)+seamChildShard+"\n")
	head := repo.Commit("Committed registry contains only the root shard")
	repo.Write(seamChildShard, seamChildYAML)
	if repo.Git("check-ignore", "--", seamChildShard) != seamChildShard || repo.Git("ls-files", "--", seamChildShard) != "" {
		t.Fatal("child shard is not an ignored untracked working file")
	}
	r := adapter(t)
	service := project.NewModelService(r, r)
	t.Run("committed-selected", func(t *testing.T) {
		stage, err := service.CaptureModel(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
		inputs := stage.Inputs()
		if err == nil || stage.Checked().Status != model.NotCompleted || inputs.Head != head || inputs.Guard != "" {
			t.Fatalf("today's untracked shard became committed input: %v", err)
		}
		if len(inputs.Raw.Registry) != 2 || inputs.Raw.Registry[1].RepositoryPath.String() != seamChildShard || len(inputs.Raw.Registry[1].File.Bytes()) != 0 || len(inputs.Raw.Local.Files.Entries()) == 0 {
			t.Fatal("refusal lost known inputs or read the uncommitted shard bytes")
		}
		prepared := stage.PrepareRegistry()
		registry, available := prepared.Registry()
		if prepared.Status() != model.NotCompleted || available || len(registry.Entries()) != 0 {
			t.Fatal("refused capture became a usable registry")
		}
	})
	t.Run("working", func(t *testing.T) {
		stage, err := service.PreviewModel(context.Background(), project.PreviewRequest{Selection: selection}, model.Limits{})
		if err != nil || stage.Checked().Status != model.Passed {
			t.Fatalf("working model preview failed: %v", err)
		}
		prepared := stage.PrepareRegistry()
		registry, available := prepared.Registry()
		inputs := prepared.Model().Inputs()
		child, found := inputs.Raw.Census.Lookup(fixturePath(t, seamChildShard))
		if prepared.Status() != model.Passed || !available || len(registry.Entries()) != 2 || !found || !bytes.Equal(child.Bytes(), []byte(seamChildYAML)) {
			t.Fatalf("actual working shard was not previewed: %s %+v", prepared.Status(), prepared.Diagnostics())
		}
		if inputs.Revision != "" || prepared.Model().Request().Revision != "" || inputs.Head != head || !bytes.Equal(inputs.Raw.Registry[1].File.Bytes(), child.Bytes()) {
			t.Fatal("working preview acquired a historical binding or changed shard bytes")
		}
	})
	t.Run("fixed", func(t *testing.T) {
		historical := selection
		historical.Registry = historical.Registry[:1]
		stage, err := service.PreviewModel(context.Background(), project.PreviewRequest{Selection: historical, Revision: head}, model.Limits{})
		if err != nil || stage.Checked().Status != model.Passed {
			t.Fatalf("fixed model preview failed: %v", err)
		}
		prepared := stage.PrepareRegistry()
		registry, available := prepared.Registry()
		inputs := prepared.Model().Inputs()
		_, hasTodayShard := inputs.Raw.Census.Lookup(fixturePath(t, seamChildShard))
		if prepared.Status() != model.Passed || !available || len(registry.Entries()) != 1 || len(prepared.Discovered()) != 1 || len(inputs.Raw.Registry) != 1 || hasTodayShard || inputs.Revision != head || prepared.Model().Request().Revision != head {
			t.Fatal("fixed preview imported today's ignored shard or lost its exact revision")
		}
	})
}
