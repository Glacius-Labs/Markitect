package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// siblingLayout puts files beside same-prefixed model directories, in the
// local model and in a dependency. A committed Capture lists Git paths in
// byte order ("lending-dependencies.yaml" before "lending/borrow.yaml"); the
// working-tree walk publication uses visits "lending/" first.
func siblingLayout(t *testing.T) (*testkit.Repo, project.InputSelection) {
	t.Helper()
	repo, selection, _ := libraryFixture(t)
	repo.Write("inputs/local/model/lending-dependencies.yaml", "id: lending-dependencies\n")
	repo.Write("inputs/local/model/lending.yaml", "id: lending\n")
	repo.Write("inputs/vocabulary/model/software-rules.yaml", "id: software-rules\n")
	repo.Commit("Place files beside same-prefixed model directories")
	return repo, selection
}

// publicationBinding is the source guard Publisher compares before each write.
func publicationBinding(t *testing.T, r *Repository, selection project.InputSelection, baseline project.CapturedInputs) project.PublicationGuard {
	t.Helper()
	_, binding, err := r.publicationBinding(context.Background(), selection, baseline)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

// The committed Capture and publication's working re-acquisition observe the
// same unchanged source, so they must issue the same guard. Entry order is an
// acquisition detail, not source content.
func TestPublicationReacquisitionKeepsTheCommittedGuard(t *testing.T) {
	_, selection := siblingLayout(t)
	r := adapter(t)
	captured, err := r.Capture(context.Background(), project.CaptureRequest{Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	if got := publicationBinding(t, r, selection, captured); got != captured.Guard {
		t.Fatalf("unchanged source moved the publication guard: captured %s, re-acquired %s", captured.Guard, got)
	}
}

// deepNote is a Definition file beside the same-prefixed Package directory
// "a/deep/", the layout of the Makerspace pilot model ("billing/" beside
// "billing-dependencies.yaml") whose verified candidate publication refused.
func deepNote(r *testkit.Repo) {
	r.Write(".markitect/model/a/deep-note.yaml", "imports: [{package: a}]\nid: deep-note\nkind: a/Thing\nbody: A Definition beside the deep Package.\nspec: {}\n")
}

// Observed in a native Apply: Assessment, Implement and Verify passed, then
// publication refused the unchanged target before any write with "guarded
// write precondition failed: ... publication source binding drift".
func TestPublicationOfAModelFileBesideASamePrefixedPackage(t *testing.T) {
	r, prepared, git, input := runFixture(t, deepNote)
	ctx := context.Background()
	publisher := &Publisher{Repository: adapter(t), Store: git}
	if err := publisher.Preflight(ctx, prepared); err != nil {
		t.Fatal(err)
	}
	published, err := publisher.Publish(ctx, apply.PublicationRequest{Prepared: prepared, Approval: closedApproval(prepared, input)})
	if err != nil || !published.Readback || !published.NoChange {
		t.Fatal(published, err)
	}
	// The complete target Census refuses these in-repository changes before the
	// source guard compares; TestPublicationGuardStillDetectsRealSourceChanges
	// covers the guard itself.
	t.Run("changed in-repository sources are refused before any write", func(t *testing.T) {
		for _, p := range []string{".markitect/model/a/deep-note.yaml", ".markitect/dependencies/core/model.yaml", ".markitect/execution.yaml"} {
			original, err := os.ReadFile(filepath.Join(r.Dir, filepath.FromSlash(p)))
			if err != nil {
				t.Fatal(err)
			}
			changed := string(original) + "\n# changed during publication\n"
			r.Write(p, changed)
			published, err := publisher.Publish(ctx, apply.PublicationRequest{Prepared: prepared, Approval: closedApproval(prepared, input)})
			if !errors.Is(err, ErrChanged) || published.Readback || len(published.CompletedPaths) != 0 {
				t.Fatal("publication accepted a changed source at", p, published, err)
			}
			if data, err := os.ReadFile(filepath.Join(r.Dir, filepath.FromSlash(p))); err != nil || string(data) != changed {
				t.Fatal("refused publication touched the changed source", p, err)
			}
			r.Write(p, string(original))
		}
	})
}

// The guard stays as strict as before: a real change to a local model source,
// a dependency or execution still moves it, even with Census and registry
// restored as publication does.
func TestPublicationGuardStillDetectsRealSourceChanges(t *testing.T) {
	for _, tc := range []struct{ name, path string }{
		{"local model file", "inputs/local/model/lending/borrow.yaml"},
		{"local model sibling", "inputs/local/model/lending-dependencies.yaml"},
		{"new local model file", "inputs/local/model/lending/new.yaml"},
		{"dependency model file", "inputs/vocabulary/model/software/rule.yaml"},
		{"dependency manifest", "inputs/core/model.yaml"},
		{"execution", "config/execution.data"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, selection := siblingLayout(t)
			r := adapter(t)
			captured, err := r.Capture(context.Background(), project.CaptureRequest{Selection: selection})
			if err != nil {
				t.Fatal(err)
			}
			// A trailing YAML comment keeps every manifest acquirable.
			data, err := os.ReadFile(filepath.Join(repo.Dir, filepath.FromSlash(tc.path)))
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			repo.Write(tc.path, string(data)+"\n# changed during publication\n")
			if got := publicationBinding(t, r, selection, captured); got == captured.Guard {
				t.Fatal("publication guard ignored a real source change at", tc.path)
			}
		})
	}
}
