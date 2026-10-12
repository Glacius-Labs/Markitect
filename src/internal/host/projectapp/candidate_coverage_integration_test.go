package projectapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

func TestFinalCandidateCoverageUsesRepositoryCensus(t *testing.T) {
	t.Run("exact ignore", func(t *testing.T) {
		root := deliveryFixture(t)
		deliveryWrite(t, root, "scratch.txt", "ignored notes\n")
		deliveryWrite(t, root, projectcoverage.IgnorePath, "apiVersion: "+projectcoverage.IgnoreAPIVersion+"\nkind: RepositoryIgnore\nentries:\n  - path: scratch.txt\n    reason: Local notes outside the model\n")
		commitDeliveryModel(t, root, "commit exact scratch ignore", "scratch.txt")
		base := loadDeliveryProject(t, root)
		if base.Coverage == nil || !base.Coverage.Conforming {
			t.Fatalf("precondition: census not conforming: %+v", base.Coverage)
		}
		candidate := classifyDeliveryCandidate(t, root, base, nil, nil)
		if candidate.Coverage == nil || !candidate.Coverage.Conforming {
			t.Fatalf("census-satisfied exact ignore entry failed candidate classification: %+v", candidate.Coverage)
		}
	})
	t.Run("unclassified candidate path", func(t *testing.T) {
		root := deliveryFixture(t)
		base := loadDeliveryProject(t, root)
		candidate, err := classifyDeliveryCandidateResult(t, root, base, map[string][]byte{"unexpected/unowned.txt": []byte("unclassified candidate bytes\n")}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if candidate.Coverage == nil || candidate.Coverage.Conforming {
			t.Fatalf("unclassified candidate path remained conforming: %+v", candidate.Coverage)
		}
	})
	t.Run("transitional", func(t *testing.T) {
		root := deliveryFixture(t)
		markDeliveryTransitional(t, root)
		base := loadDeliveryProject(t, root)
		if base.Coverage == nil || !base.Coverage.Accounted || base.Coverage.Conforming {
			t.Fatalf("precondition: census must be accounted but nonconforming: %+v", base.Coverage)
		}
		candidate := classifyDeliveryCandidate(t, root, base, nil, nil)
		if candidate.Coverage == nil || candidate.Coverage.Conforming {
			t.Fatalf("transitional path did not remain nonconforming in candidate classification: %+v", candidate.Coverage)
		}
	})
	t.Run("rename modelled file", func(t *testing.T) {
		root := deliveryFixture(t)
		base := loadDeliveryProject(t, root)
		const renamed = "src/orders/renamed.txt"
		artifactPath := ".markitect/model/orders/artifact.yaml"
		artifact, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifactPath)))
		if err != nil {
			t.Fatal(err)
		}
		updatedArtifact := strings.Replace(string(artifact), "src/orders/implementation.txt", renamed, 1)
		if updatedArtifact == string(artifact) {
			t.Fatal("orders Artifact path was not found")
		}
		candidate := classifyDeliveryCandidate(t, root, base, map[string][]byte{renamed: base.Snapshot.Files["src/orders/implementation.txt"], artifactPath: []byte(updatedArtifact)}, []string{"src/orders/implementation.txt"})
		if candidate.Coverage == nil || !candidate.Coverage.Conforming {
			t.Fatalf("rename with its Artifact path moved failed candidate classification: %+v", candidate.Coverage)
		}
	})
	t.Run("delete modelled file", func(t *testing.T) {
		root := deliveryFixture(t)
		const notes = "src/orders/notes.txt"
		artifactPath := ".markitect/model/orders/artifact.yaml"
		artifact, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifactPath)))
		if err != nil {
			t.Fatal(err)
		}
		updatedArtifact := strings.Replace(string(artifact), "src/orders/implementation.txt]", "src/orders/implementation.txt, "+notes+"]", 1)
		if updatedArtifact == string(artifact) {
			t.Fatal("orders Artifact paths were not found")
		}
		deliveryWrite(t, root, artifactPath, updatedArtifact)
		deliveryWrite(t, root, notes, "orders notes\n")
		commitDeliveryModel(t, root, "model orders notes", artifactPath, notes)
		base := loadDeliveryProject(t, root)
		if _, exists := base.Snapshot.Files[notes]; !exists {
			t.Fatal("fixture notes path was not part of the acquired base snapshot")
		}
		candidate := classifyDeliveryCandidate(t, root, base, map[string][]byte{artifactPath: []byte(artifact)}, []string{notes})
		if candidate.Coverage == nil || !candidate.Coverage.Conforming {
			t.Fatalf("modelled notes deletion failed candidate classification: %+v", candidate.Coverage)
		}
	})
	t.Run("transitional file deleted", func(t *testing.T) {
		root := deliveryFixture(t)
		markDeliveryTransitional(t, root)
		base := loadDeliveryProject(t, root)
		if _, included := base.Snapshot.Files["legacy/old.txt"]; included {
			t.Fatal("transitional path unexpectedly appeared in the reviewed base snapshot")
		}
		_, err := classifyDeliveryCandidateResult(t, root, base, nil, []string{"legacy/old.txt"})
		if err == nil || !strings.Contains(err.Error(), "outside the reviewed snapshot") {
			t.Fatalf("deleting a census-only transitional file was not rejected: %v", err)
		}
	})
	t.Run("transitional file modelled in place", func(t *testing.T) {
		root := deliveryFixture(t)
		markDeliveryTransitional(t, root)
		base := loadDeliveryProject(t, root)
		const artifactPath = ".markitect/model/orders/artifact.yaml"
		artifact, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifactPath)))
		if err != nil {
			t.Fatal(err)
		}
		updatedArtifact := strings.Replace(string(artifact), "src/orders/implementation.txt]", "src/orders/implementation.txt, legacy/old.txt]", 1)
		manifestPath := filepath.Join(root, filepath.FromSlash(projectvalue.ManifestPath))
		manifest, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		updatedManifest := strings.Replace(string(manifest), "transitionalExclusions:\n  - path: legacy/\n    reason: Existing file awaits explicit modeling\n", "", 1)
		if updatedArtifact == string(artifact) || updatedManifest == string(manifest) {
			t.Fatal("fixture transitional/model Artifact declarations were not found")
		}
		candidate := classifyDeliveryCandidate(t, root, base, map[string][]byte{artifactPath: []byte(updatedArtifact), projectvalue.ManifestPath: []byte(updatedManifest)}, nil)
		if candidate.Coverage == nil || !candidate.Coverage.Conforming {
			t.Fatalf("modelled transitional file was not admitted from the base census: %+v", candidate.Coverage)
		}
		document, err := projectwork.Document(candidate, false)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(document, "legacy/old.txt") {
			t.Fatalf("generated document omitted the modeled transitional path: %s", document)
		}
	})
}

func TestApplicationCandidateChangeKeepsModelDigestAndChangesProjectDigest(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	base := loadDeliveryProject(t, root)
	path := "src/orders/implementation.txt"
	content := append([]byte(nil), base.Snapshot.Files[path]...)
	content = append(content, []byte("\nverified candidate implementation update\n")...)
	candidate, err := classifyDeliveryCandidateResult(t, root, base, map[string][]byte{path: content}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Report.ModelDigest != base.Report.ModelDigest || candidate.Report.Digest == base.Report.Digest {
		t.Fatalf("implementation edit did not isolate model-stable project digest change: model=%t report=%t", candidate.Report.ModelDigest == base.Report.ModelDigest, candidate.Report.Digest != base.Report.Digest)
	}
}

func TestApplicationCheckDetectsUnclassifiedWorktreePathAddedAfterPlanning(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	operations := NewOperations()
	planned, err := operations.Host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if planned.Coverage == nil || !planned.Coverage.Conforming {
		t.Fatalf("fixture worktree is not conforming: %+v", planned.Coverage)
	}
	deliveryWrite(t, root, "unexpected/unowned.txt", "appeared after planning\n")
	current, err := operations.Host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if current.Snapshot.Digest() == planned.Snapshot.Digest() || current.Digest == planned.Digest {
		t.Fatal("worktree change did not invalidate the planned project and snapshot digests")
	}
	if current.Coverage == nil || current.Coverage.Conforming {
		t.Fatalf("unclassified post-plan worktree path remained conforming: %+v", current.Coverage)
	}
}

func markDeliveryTransitional(t *testing.T, root string) {
	t.Helper()
	manifestPath := filepath.Join(root, filepath.FromSlash(projectvalue.ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "exclusions: []\n", "exclusions: []\ntransitionalExclusions:\n  - path: legacy/\n    reason: Existing file awaits explicit modeling\n", 1)
	if updated == string(manifest) {
		t.Fatal("could not add a transitional exclusion")
	}
	deliveryWrite(t, root, projectvalue.ManifestPath, updated)
	deliveryWrite(t, root, "legacy/old.txt", "legacy bytes\n")
	commitDeliveryModel(t, root, "mark legacy path transitional", projectvalue.ManifestPath, "legacy/old.txt")
}

func loadDeliveryProject(t *testing.T, root string) *projectvalue.Project {
	t.Helper()
	head := git(t, root, "rev-parse", "HEAD")
	project, err := projectwork.Load(root, head)
	if err != nil {
		t.Fatalf("load exact delivery fixture source: %v", err)
	}
	return project
}

func classifyDeliveryCandidate(t *testing.T, root string, base *projectvalue.Project, changes map[string][]byte, deleted []string) *projectvalue.Project {
	t.Helper()
	classified, err := classifyDeliveryCandidateResult(t, root, base, changes, deleted)
	if err != nil {
		t.Fatal(err)
	}
	return classified
}

func classifyDeliveryCandidateResult(t *testing.T, root string, base *projectvalue.Project, changes map[string][]byte, deleted []string) (*projectvalue.Project, error) {
	t.Helper()
	selected := &snapshot.Snapshot{ID: base.Snapshot.ID, Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, data := range base.Snapshot.Files {
		selected.Files[path] = append([]byte(nil), data...)
		selected.Modes[path] = base.Snapshot.Modes[path]
	}
	for _, path := range deleted {
		delete(selected.Files, path)
		delete(selected.Modes, path)
	}
	for path, data := range changes {
		selected.Files[path] = append([]byte(nil), data...)
		selected.Modes[path] = snapshot.RegularMode
	}
	compiled, err := projectwork.FromSnapshot(root, selected)
	if err != nil {
		return nil, err
	}
	document, err := projectwork.Document(compiled, false)
	if err != nil {
		return nil, err
	}
	documentPath := projectvalue.DocumentPath(compiled.Config)
	selected.Files[documentPath] = []byte(document)
	selected.Modes[documentPath] = snapshot.RegularMode
	compiled, err = projectwork.FromSnapshot(root, selected)
	if err != nil {
		return nil, err
	}
	return projectwork.ClassifyCandidate(base, compiled, deleted...)
}

func commitDeliveryModel(t *testing.T, root, message string, paths ...string) {
	t.Helper()
	operations := NewOperations()
	document, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}})
	if err != nil {
		t.Fatalf("render fixture source document through Application: %v", err)
	}
	deliveryWrite(t, root, document.Path, document.Content)
	all := append(append([]string(nil), paths...), document.Path)
	args := append([]string{"add", "--"}, all...)
	git(t, root, args...)
	git(t, root, "commit", "-m", message)
}
