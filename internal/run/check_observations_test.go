package run

import (
	"errors"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestFixedCheckReceiptRejectsMissingOrSubstitutedPhysicalObservations(t *testing.T) {
	var declaration project.ExecutionCheck
	declaration.ID.Value = "required"
	fixed := Commit{ID: "fixed", Tree: "git-tree"}
	var tree snapshot.Tree
	dir, _ := snapshot.NewPath("empty-package")
	dirs := []snapshot.Path{dir}
	valid := CheckObservation{ID: "required", Commit: fixed.ID, Receipt: CheckReceipt{ID: "required", ImageID: "sha256:" + strings.Repeat("a", 64), Tree: tree.Digest(), Directories: []string{"empty-package"}, Started: true, Exited: true, CopiesClosed: true}}
	if err := ValidateCheckObservations([]project.ExecutionCheck{declaration}, []CheckObservation{valid}, fixed, tree, dirs); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"id", "commit", "tree", "image", "directories", "exit", "uncertain", "cleanup", "duplicate", "missing"} {
		t.Run(kind, func(t *testing.T) {
			value := valid
			switch kind {
			case "id":
				value.Receipt.ID = "other"
			case "commit":
				value.Commit = "other"
			case "tree":
				value.Receipt.Tree = snapshot.TreeID{}
			case "image":
				value.Receipt.ImageID = ""
			case "directories":
				value.Receipt.Directories = nil
			case "exit":
				value.Receipt.ExitCode = 1
			case "uncertain":
				value.Receipt.Uncertain = true
			case "cleanup":
				value.Receipt.CopiesClosed = false
			}
			observed := []CheckObservation{value}
			if kind == "duplicate" {
				observed = append(observed, value)
			}
			if kind == "missing" {
				observed = nil
			}
			if err := ValidateCheckObservations([]project.ExecutionCheck{declaration}, observed, fixed, tree, dirs); err == nil {
				t.Fatal("accepted", kind)
			}
		})
	}
}

func TestKnownFailingCheckDoesNotHideLaterTechnicalObservationFailure(t *testing.T) {
	var a, b project.ExecutionCheck
	a.ID.Value, b.ID.Value = "a", "b"
	fixed := Commit{ID: "fixed"}
	var tree snapshot.Tree
	first := CheckObservation{ID: "a", Commit: fixed.ID, Receipt: CheckReceipt{ID: "a", ImageID: "sha256:" + strings.Repeat("a", 64), Tree: tree.Digest(), Started: true, Exited: true, ExitCode: 1, CopiesClosed: true}}
	second := first
	second.ID, second.Receipt.ID, second.Receipt.ExitCode = "b", "b", 0
	for _, kind := range []string{"missing", "duplicate", "uncertain", "wrong identity"} {
		t.Run(kind, func(t *testing.T) {
			observed := []CheckObservation{first, second}
			switch kind {
			case "missing":
				observed = observed[:1]
			case "duplicate":
				observed = append(observed, second)
			case "uncertain":
				observed[1].Receipt.Uncertain = true
			case "wrong identity":
				observed[1].Receipt.ID = "other"
			}
			err := ValidateCheckObservations([]project.ExecutionCheck{a, b}, observed, fixed, tree, nil)
			if !errors.Is(err, ErrObservationIncomplete) || errors.Is(err, project.ErrCandidateInvalid) {
				t.Fatal("known nonzero hid technical incompleteness", kind, err)
			}
		})
	}
	err := ValidateCheckObservations([]project.ExecutionCheck{a, b}, []CheckObservation{first, second}, fixed, tree, nil)
	if !errors.Is(err, project.ErrCandidateInvalid) || errors.Is(err, ErrObservationIncomplete) {
		t.Fatal("complete known check failure lost semantic classification", err)
	}
}
