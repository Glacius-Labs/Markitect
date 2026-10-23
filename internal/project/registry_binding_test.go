package project_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestRegistryResolvesNestedAndImportedFullAddresses(t *testing.T) {
	for _, target := range []string{"lending/reminders/assess-overdue", "markitect/core/Kind"} {
		selection, raw := singleRegistry(t, ".markitect/registry/lending/reminders/registry.yaml", strings.Replace(childRegistry, "lending/borrow", target, 1))
		result := registryModel(t, selection, raw).PrepareRegistry()
		registry, ok := result.Registry()
		if result.Status() != model.Passed || !ok {
			t.Fatalf("valid complete address refused: %+v", result.Diagnostics())
		}
		registration := registry.Entries()[0]
		owner, _ := registration.Owner().Package()
		basis := registration.Artifact().Basis()[0]
		if owner != "lending/reminders" || basis.Package+"/"+basis.ID != target {
			t.Fatal("nested Package or imported basis was rewritten")
		}
	}
}

func TestRegistryDuplicateOwnershipAcrossShardsHasNoWinner(t *testing.T) {
	selection, raw := libraryInputs(t)
	censusFile(t, &raw, "realization/library.md", snapshot.Regular, []byte("shared artifact"))
	registrySource(t, &selection, &raw, ".markitect/registry/registry.yaml", rootRegistry, true)
	registrySource(t, &selection, &raw, ".markitect/registry/lending/registry.yaml", strings.Replace(childRegistry, "realization/lending.go", "realization/library.md", 1), true)
	first := registryModel(t, selection, raw).PrepareRegistry()
	slices.Reverse(selection.Registry)
	slices.Reverse(raw.Registry)
	second := registryModel(t, selection, raw).PrepareRegistry()
	if first.Status() != model.Failed || second.Status() != model.Failed || !registryDiagnostic(first.Diagnostics(), "registry.records_invalid") {
		t.Fatal("ownership winner selected")
	}
	if _, ok := first.Registry(); ok {
		t.Fatal("partial conflicting registry exposed")
	}
	if !reflect.DeepEqual(first.Diagnostics(), second.Diagnostics()) || !reflect.DeepEqual(first.Records(), second.Records()) || !reflect.DeepEqual(first.Discovered(), second.Discovered()) {
		t.Fatal("source enumeration picked a winner or changed diagnostics")
	}
}

func TestRegistryPartialRecordAndCopiesRemainBoundToOriginalStage(t *testing.T) {
	content := strings.Replace(childRegistry, "form: code", "form: unsupported", 1)
	selection, raw := libraryInputs(t)
	censusFile(t, &raw, "realization/library.md", snapshot.Regular, []byte("usage"))
	censusFile(t, &raw, "realization/lending.go", snapshot.Regular, []byte("implementation"))
	registrySource(t, &selection, &raw, ".markitect/registry/registry.yaml", rootRegistry, true)
	registrySource(t, &selection, &raw, ".markitect/registry/lending/registry.yaml", content, true)
	stage := registryModel(t, selection, raw)
	result := stage.PrepareRegistry()
	if _, ok := result.Registry(); ok || result.Status() != model.Failed || len(result.Records()) != 2 {
		t.Fatalf("known partial records lost or accepted: %+v", result.Diagnostics())
	}
	var partial *project.RegistryRecord
	records := result.Records()
	for i := range records {
		if records[i].Form == "unsupported" {
			partial = &records[i]
		}
	}
	if partial == nil || partial.Path != "realization/lending.go" || len(partial.Realizes) != 1 || partial.Why != "Lending operation." || partial.Position.Line <= 0 {
		t.Fatal("known invalid record's facts discarded")
	}
	beforeRecords, beforeDiagnostics, beforeDiscovered, beforeModel := result.Records(), result.Diagnostics(), result.Discovered(), result.Model().Inputs()
	records[0].Realizes[0] = "mutation/target"
	records[0].Why = "mutation"
	diagnostics := result.Diagnostics()
	positions := 0
	for i := range diagnostics {
		diagnostics[i].Message = "mutation"
		if diagnostics[i].Position != nil {
			diagnostics[i].Position.Line = -1
			positions++
		}
	}
	discovered := result.Discovered()
	discovered[0] = path(t, "mutation")
	returned := result.Model().Inputs()
	returned.Raw.Registry[0].Location.Boundary = "mutation"
	returned.Raw.Local.Source.Entries[0].Path = "mutation"
	bytes := returned.Raw.Registry[0].File.Bytes()
	bytes[0] = '!'
	// Mutating the original controlled Source's slice must also leave the
	// already captured stage and its registry preparation fixed.
	raw.Registry[0].Location.Boundary = "caller mutation"
	selection.Registry[0].Boundary = "caller mutation"
	if positions == 0 || !reflect.DeepEqual(beforeRecords, result.Records()) || !reflect.DeepEqual(beforeDiagnostics, result.Diagnostics()) || !reflect.DeepEqual(beforeDiscovered, result.Discovered()) || !reflect.DeepEqual(beforeModel, result.Model().Inputs()) {
		t.Fatal("record, diagnostic or parent model alias escaped")
	}
}

func TestMalformedRecordRetainsKnownFieldsAndOriginalBytes(t *testing.T) {
	content := strings.Replace(childRegistry, "realizes: [lending/borrow]", "realizes: [lending/borrow, 123]", 1)
	selection, raw := singleRegistry(t, ".markitect/registry/lending/registry.yaml", content)
	result := registryModel(t, selection, raw).PrepareRegistry()
	if result.Status() != model.Failed || len(result.Records()) != 1 || result.Records()[0].Path != "realization/lending.go" || len(result.Records()[0].Realizes) != 1 || string(result.Model().Inputs().Raw.Registry[0].File.Bytes()) != content {
		t.Fatal("known or original malformed facts disappeared")
	}
}
