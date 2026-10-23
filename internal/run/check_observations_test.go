package run

import (
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
